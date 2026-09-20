// Spool → profile/v1: hooks as a first-class capture source. Tool calls are
// paired by tool_use_id (pre/post/failure), skill activation is inferred from
// SKILL.md reads, and attribution links each tool call to the most recent
// inferred activation — all labelled confidence:"inferred".
package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LoadSessionEvents returns spool events belonging to a session, matched on
// conversation_id, session_id, or generation_id (session and generation ids
// live inside raw). Sorted by capture time.
func LoadSessionEvents(dir, sessionID string) ([]SpoolEvent, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var out []SpoolEvent
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			var ev SpoolEvent
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			if sessionMatches(ev, sessionID) {
				out = append(out, ev)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ts < out[j].Ts })
	return out, nil
}

func sessionMatches(ev SpoolEvent, id string) bool {
	if ev.ConversationID == id {
		return true
	}
	var raw map[string]any
	if json.Unmarshal(ev.Raw, &raw) != nil {
		return false
	}
	return getString(raw, "session_id") == id || getString(raw, "generation_id") == id
}

// profileFromSpool builds a Profile from the session's hook events. Metrics
// are present only when the data exists; tokens stay unknown because hooks
// carry no billed counts.
func profileFromSpool(events []SpoolEvent, sessionID string, opts CaptureOpts, cap CapabilityReport) Profile {
	now := time.Now().UTC().Format(time.RFC3339)
	p := Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   now,
		Harness:      "cursor",
		SessionID:    resolveSessionID(events, sessionID),
		SnapshotHash: opts.SnapshotHash,
		SkillDir:     opts.SkillDir,
		Capability:   cap,
	}

	calls := pairToolCalls(events)
	if len(calls) > 0 {
		p.ToolCalls = PresentToolCallResult(calls, string(SourceHooks))
	} else {
		p.ToolCalls = UnknownToolCallResult("no tool events for session in spool")
	}

	activations := inferActivations(events)
	if len(activations) > 0 {
		p.SkillActivation = PresentActivationResult(activations, string(SourceHooks))
	} else {
		p.SkillActivation = UnknownActivationResult("no skill activations inferred for session")
	}

	if td, ok := sessionTiming(events); ok {
		p.Timing = PresentTimingResult(td, string(SourceHooks))
	} else {
		p.Timing = UnknownTimingResult("no sessionStart/sessionEnd for session in spool")
	}

	attributions := attributeCalls(calls, activations)
	if len(attributions) > 0 {
		p.Attribution = PresentAttributionResult(AttributionData{Attributions: attributions}, string(SourceHooks))
	} else {
		p.Attribution = UnknownAttributionResult("no attributions inferred for session")
	}

	p.Tokens = UnknownTokenResult("Cursor hooks do not expose billed token counts")
	var est int64
	for _, ev := range events {
		est += int64(len(ev.Raw)) / 4
	}
	p.EstimatedContextTokens = PresentEstimatedTokensResult(EstimatedTokens{Total: est}, string(SourceHooksEstimated))
	return p
}

// resolveSessionID prefers the session_id carried on sessionStart/sessionEnd,
// then falls back to the id the caller asked for (conversation or generation).
// Precedence decided per spec (report risk S-8): session_id > caller id.
func resolveSessionID(events []SpoolEvent, callerID string) string {
	for _, ev := range events {
		var raw map[string]any
		if json.Unmarshal(ev.Raw, &raw) != nil {
			continue
		}
		if s := getString(raw, "session_id"); s != "" {
			return s
		}
	}
	return callerID
}

// pairToolCalls joins preToolUse / postToolUse / postToolUseFailure on
// tool_use_id into one ToolCallEntry each.
func pairToolCalls(events []SpoolEvent) []ToolCallEntry {
	type acc struct{ e ToolCallEntry }
	byID := map[string]*acc{}
	var order []string
	rawOf := func(ev SpoolEvent) map[string]any {
		var m map[string]any
		json.Unmarshal(ev.Raw, &m)
		return m
	}
	for _, ev := range events {
		raw := rawOf(ev)
		if raw == nil {
			continue
		}
		id := getString(raw, "tool_use_id")
		if id == "" {
			continue
		}
		a, ok := byID[id]
		if !ok {
			a = &acc{}
			a.e.ID = id
			byID[id] = a
			order = append(order, id)
		}
		a.e.Name = firstNonEmpty(a.e.Name, getString(raw, "tool_name"))
		if a.e.Timestamp == "" {
			a.e.Timestamp = ev.Ts
		}
		switch ev.Event {
		case "postToolUse":
			a.e.Success = true
		case "postToolUseFailure":
			a.e.Success = false
			a.e.ErrorType = firstNonEmpty(getString(raw, "failure_type"), "failure")
		}
	}
	var calls []ToolCallEntry
	for _, id := range order {
		calls = append(calls, byID[id].e)
	}
	return calls
}

// inferActivations maps SKILL.md file reads to ActivationEntries. Reading the
// SKILL.md itself is the signal; trigger is "inferred" because Cursor exposes
// no first-party activation hook on this tier.
func inferActivations(events []SpoolEvent) []ActivationEntry {
	var out []ActivationEntry
	for _, ev := range events {
		if ev.Event != "beforeReadFile" && ev.Event != "beforeTabFileRead" {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(ev.Raw, &raw) != nil {
			continue
		}
		if m := skillPathRe.FindStringSubmatch(getString(raw, "file_path")); m != nil {
			out = append(out, ActivationEntry{
				SkillName: m[1],
				Timestamp: ev.Ts,
				Trigger:   "inferred",
			})
		}
	}
	return out
}

// sessionTiming derives start/end from event timestamps and sessionEnd's
// duration_ms when present.
func sessionTiming(events []SpoolEvent) (TimingData, bool) {
	var start, end string
	var totalMs int64
	for _, ev := range events {
		if start == "" || ev.Ts < start {
			start = ev.Ts
		}
		if ev.Ts > end {
			end = ev.Ts
		}
		if ev.Event == "sessionEnd" {
			var raw map[string]any
			if json.Unmarshal(ev.Raw, &raw) == nil {
				totalMs = int64(toInt(raw["duration_ms"]))
			}
		}
	}
	if start == "" {
		return TimingData{}, false
	}
	td := TimingData{StartTime: start, EndTime: end, TotalMs: totalMs}
	if td.TotalMs == 0 {
		if t1, t2 := parseTime(start), parseTime(end); !t1.IsZero() && !t2.IsZero() {
			td.TotalMs = t2.Sub(t1).Milliseconds()
		}
	}
	return td, true
}

// attributeCalls links each tool call to the most recent activation that
// precedes it within the session. All links are confidence:"inferred" — the
// heuristic is temporal, not causal.
func attributeCalls(calls []ToolCallEntry, acts []ActivationEntry) []Attribution {
	if len(acts) == 0 {
		return nil
	}
	var out []Attribution
	for _, c := range calls {
		skill := ""
		for _, a := range acts {
			if a.Timestamp <= c.Timestamp {
				skill = a.SkillName
			}
		}
		if skill == "" {
			continue
		}
		out = append(out, Attribution{
			Target:        toolCallTarget(c),
			SkillName:     skill,
			Category:      "skill",
			Detail:        "file",
			OperationName: "execute_tool",
			Confidence:    "inferred",
		})
	}
	return out
}

// toolCallTarget prefers the tool_use_id, falling back to name@timestamp.
func toolCallTarget(c ToolCallEntry) string {
	if c.ID != "" {
		return c.ID
	}
	return c.Name + "@" + c.Timestamp
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
