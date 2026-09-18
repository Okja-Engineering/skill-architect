package profiler

import "strings"

// Provenance: which records in a capture file belong to the run being profiled.
//
// A capture file is not a session. Both documented capture routes append to one
// file by design — route (a) is an OTLP receiver on a loopback port, which
// anything on the machine may post to, and route (b) is a collector's file
// exporter, which fans in whatever it is configured to collect. So one export
// legitimately carries several sessions and more than one product's telemetry,
// while a profile names exactly one session and is read as a measurement of it.
//
// This file is the bridge between the two layers either side of that fact:
// otlp.go knows what a scope and an attribute are but nothing about sessions,
// and claude_code.go knows the harness's names but cannot see the envelope a
// record arrived in. The projection lives here, once, and the extractors are
// handed its result rather than the file — so scoping is not a step any of them
// can forget, and a second OTLP-speaking adapter gets it by supplying its own
// names.

// provenance is the test a record must pass to be read into a profile.
//
// Two independent signals, because either alone leaves a hole. The session
// attribute is the strong one: Claude Code puts the run's identity on every
// metric data point and every log record, so a record either says which run it
// is from or cannot be attributed to one. The instrumentation scope is the
// second: a record that names another product's scope is that product's,
// whatever identity it carries.
type provenance struct {
	// namespace is the harness's own component in a reverse-DNS
	// instrumentation scope name. A scope is *foreign* when it names
	// components and none of them is this one.
	namespace string

	// sessionAttr is the attribute a record carries its run identity in.
	sessionAttr string

	// sessionID is the identity the profile asserts. A record contributes only
	// if it carries this value under sessionAttr — carrying a different one and
	// carrying none at all are the same answer, because neither says the record
	// is this run's.
	sessionID string

	// anySession drops the session test, and is what probe asks: probe is asked
	// what an export can yield without being told a session, so it answers
	// about the export. It is a field of its own rather than an empty
	// sessionID because a filter that matches everything has to be something a
	// reader can grep for, not something an empty string quietly becomes.
	// Capture never sets it; it refuses an empty session id instead.
	anySession bool
}

// ownsScope reports whether s recorded records this provenance may read.
//
// A scope that names nothing is not a foreign scope. An instrumentation scope
// is optional in OTLP and a receiver or collector in the path may not carry one
// through, so refusing an unnamed scope would trade a wrong number for no
// number on every pipeline that drops it — and the session test still stands
// over those records. The check is therefore an exclusion of a scope that
// positively names another product, not an allowlist of an exact scope string:
// the observed names are com.anthropic.claude_code and com.anthropic.
// claude_code.events, but the subagent meter and later releases vary the tail,
// and an allowlist would silently zero a real capture the day one of them
// changed. Matching the harness's own namespace component admits all of those
// and refuses some.other.product.
func (p provenance) ownsScope(s otlpScope) bool {
	if s.Name == "" {
		return true
	}
	for _, part := range strings.Split(s.Name, ".") {
		if part == p.namespace {
			return true
		}
	}
	return false
}

// ownsAttrs reports whether an attribute set carries the asserted run identity.
func (p provenance) ownsAttrs(a otlpAttrs) bool {
	if p.anySession {
		return true
	}
	id, ok := a.String(p.sessionAttr)
	return ok && id == p.sessionID
}

// exclusions is what the projection removed, counted in the units a reader can
// look for in their own export. It is the only thing the "not read" clause of a
// reason is built from, so a reason cannot state a number the projection did
// not count.
type exclusions struct {
	foreignScopeMetricPoints int
	otherSessionMetricPoints int
	foreignScopeLogRecords   int
	otherSessionLogRecords   int
}

// scopedExport is one export projected onto one provenance: the records a
// profile of that provenance may read, beside a count of what the projection
// removed.
//
// The extractors take this rather than an otlpExport. That is the point of the
// type: an extractor cannot be handed the whole file by accident, so no
// extractor has to remember to scope anything, and adding a fourth signal
// cannot reintroduce the defect this repair exists for.
type scopedExport struct {
	otlpExport
	excluded exclusions
	prov     provenance
}

// scopedTo projects e onto p.
//
// The envelope is rebuilt rather than filtered in place, and every level is
// kept even when nothing under it survives: an export of a session that emitted
// nothing is still an export, and collapsing it would hand the signals the
// "this is not OTLP/JSON" reason for a file that plainly is one.
//
// One rule at the metric level: a metric that carried data points and kept none
// is dropped. Such a metric is not this run's — it is the name matching under
// somebody else's identity — and keeping it as an empty sum would tell the
// reader their export "carried no sum data points" when it carries several.
// A metric that arrived with no data points at all is kept, because that is the
// export saying so and the reason for it is true.
func (e otlpExport) scopedTo(p provenance) scopedExport {
	var ex exclusions
	out := make(otlpExport, 0, len(e))

	for _, b := range e {
		var kept otlpBatch

		if b.ResourceMetrics != nil {
			rms := make([]otlpResourceMetrics, 0, len(*b.ResourceMetrics))
			for _, rm := range *b.ResourceMetrics {
				sms := make([]otlpScopeMetrics, 0, len(rm.ScopeMetrics))
				for _, sm := range rm.ScopeMetrics {
					if !p.ownsScope(sm.Scope) {
						ex.foreignScopeMetricPoints += dataPointCount(sm.Metrics)
						continue
					}
					sm.Metrics = scopedMetrics(sm.Metrics, p, &ex)
					sms = append(sms, sm)
				}
				rm.ScopeMetrics = sms
				rms = append(rms, rm)
			}
			kept.ResourceMetrics = &rms
		}

		if b.ResourceLogs != nil {
			rls := make([]otlpResourceLogs, 0, len(*b.ResourceLogs))
			for _, rl := range *b.ResourceLogs {
				sls := make([]otlpScopeLogs, 0, len(rl.ScopeLogs))
				for _, sl := range rl.ScopeLogs {
					if !p.ownsScope(sl.Scope) {
						ex.foreignScopeLogRecords += len(sl.LogRecords)
						continue
					}
					records := make([]otlpLogRecord, 0, len(sl.LogRecords))
					for _, r := range sl.LogRecords {
						if !p.ownsAttrs(r.Attributes) {
							ex.otherSessionLogRecords++
							continue
						}
						records = append(records, r)
					}
					sl.LogRecords = records
					sls = append(sls, sl)
				}
				rl.ScopeLogs = sls
				rls = append(rls, rl)
			}
			kept.ResourceLogs = &rls
		}

		out = append(out, kept)
	}

	return scopedExport{otlpExport: out, excluded: ex, prov: p}
}

// scopedMetrics keeps the data points p owns, and drops a metric that had
// points and kept none. See scopedTo for why those two cases differ.
func scopedMetrics(metrics []otlpMetric, p provenance, ex *exclusions) []otlpMetric {
	out := make([]otlpMetric, 0, len(metrics))
	for _, m := range metrics {
		if m.Sum == nil {
			out = append(out, m)
			continue
		}
		points := make([]otlpDataPoint, 0, len(m.Sum.DataPoints))
		for _, dp := range m.Sum.DataPoints {
			if !p.ownsAttrs(dp.Attributes) {
				ex.otherSessionMetricPoints++
				continue
			}
			points = append(points, dp)
		}
		if len(points) == 0 && len(m.Sum.DataPoints) > 0 {
			continue
		}
		sum := *m.Sum
		sum.DataPoints = points
		m.Sum = &sum
		out = append(out, m)
	}
	return out
}

func dataPointCount(metrics []otlpMetric) int {
	n := 0
	for _, m := range metrics {
		if m.Sum != nil {
			n += len(m.Sum.DataPoints)
		}
	}
	return n
}

// metricsNotRead and logsNotRead are the clause a signal's unknown reason
// carries when the projection removed records the signal reads from. Without it
// a reason saying a signal was "not found in OTel export" would be read as
// saying the export carries nothing of the kind, which is the same shape of
// false statement this repair exists to remove — only inverted.
//
// Each names only the records its own signal could have read, so the tool-call
// reason does not report metric data points. Both are empty when the projection
// removed nothing, which is every capture of one session with nothing else on
// the port; that is why an export of a single session reads exactly as it did
// before this filter existed.
func (s scopedExport) metricsNotRead() string {
	return s.notReadClause(s.excluded.otherSessionMetricPoints, s.excluded.foreignScopeMetricPoints, "data point")
}

func (s scopedExport) logsNotRead() string {
	return s.notReadClause(s.excluded.otherSessionLogRecords, s.excluded.foreignScopeLogRecords, "log record")
}

func (s scopedExport) notReadClause(otherSession, foreignScope int, noun string) string {
	// Both clauses are participial, so each reads the same for one record and
	// for five — the same rule every reason in claude_code.go is built by.
	clauses := make([]string, 0, 2)
	if otherSession > 0 {
		clauses = append(clauses, quantity(otherSession, noun)+" not carrying "+
			s.prov.sessionAttr+" "+s.prov.sessionID)
	}
	if foreignScope > 0 {
		clauses = append(clauses, quantity(foreignScope, noun)+
			" recorded by an instrumentation scope that is not "+s.prov.namespace+"'s")
	}
	if len(clauses) == 0 {
		return ""
	}
	return " — not read: " + strings.Join(clauses, "; ")
}
