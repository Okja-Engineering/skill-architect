package profiler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OTLP/JSON format layer: how a capture file decodes, and nothing about what
// any harness means by what it carries. claude_code.go holds the semantics and
// speaks only otlpMetric and otlpLogRecord.
//
// The split is the repair this file exists for. When decoding and meaning were
// one struct, a JSON shape nothing ever emitted read as a model of the harness,
// and no test could tell the difference. A second OTLP-speaking adapter
// (Cursor, Codex) would read this layer as it stands; it is deliberately
// harness-agnostic, and it stays in package profiler until there is a second
// consumer to justify a package boundary.
//
// Two rules run through it:
//
//   - Every leaf the OTLP spec lets vary decodes as json.RawMessage and is read
//     by a helper returning (value, ok). A leaf that cannot be read degrades one
//     data point or one log record, never the file. The walk itself is fully
//     typed, and a document that does not fit it is a whole-file failure with a
//     reason built from the failure's type.
//   - Field names are lowerCamelCase per the OTLP JSON encoding, and unknown
//     fields, metric names and event names are ignored: telemetry gains fields
//     between releases, and a capture from a newer Claude Code must still read.

// --- Envelope ---
//
// One Export*ServiceRequest per JSON object. Both message types are decoded
// from the same struct because a capture may carry either, and route (b) may
// put both in one file; which top-level field is present says which it is.

type otlpBatch struct {
	ResourceMetrics []struct {
		ScopeMetrics []struct {
			Metrics []otlpMetric `json:"metrics"`
		} `json:"scopeMetrics"`
	} `json:"resourceMetrics"`
	ResourceLogs []struct {
		ScopeLogs []struct {
			LogRecords []otlpLogRecord `json:"logRecords"`
		} `json:"scopeLogs"`
	} `json:"resourceLogs"`
}

// otlpMetric is one metric. Sum is nil for a gauge or a histogram — a shape a
// future release could switch to — which is counted and skipped, never a panic.
type otlpMetric struct {
	Name string   `json:"name"`
	Sum  *otlpSum `json:"sum"`
}

type otlpSum struct {
	AggregationTemporality json.RawMessage `json:"aggregationTemporality"`
	DataPoints             []otlpDataPoint `json:"dataPoints"`
}

type otlpDataPoint struct {
	Attributes   otlpAttrs       `json:"attributes"`
	TimeUnixNano json.RawMessage `json:"timeUnixNano"`
	AsDouble     json.RawMessage `json:"asDouble"`
	AsInt        json.RawMessage `json:"asInt"`
}

type otlpLogRecord struct {
	TimeUnixNano json.RawMessage `json:"timeUnixNano"`
	Body         json.RawMessage `json:"body"`
	Attributes   otlpAttrs       `json:"attributes"`
}

type otlpAttrs []otlpAttr

type otlpAttr struct {
	Key   string        `json:"key"`
	Value otlpAttrValue `json:"value"`
}

// otlpAttrValue is the AnyValue an attribute carries. Pointers and raw bytes,
// because "absent" and "the zero value" are different answers, and because a
// semantically numeric attribute arrives as any of the three.
type otlpAttrValue struct {
	StringValue *string         `json:"stringValue"`
	BoolValue   *bool           `json:"boolValue"`
	IntValue    json.RawMessage `json:"intValue"`
	DoubleValue json.RawMessage `json:"doubleValue"`
}

// otlpExport is every batch in one capture file, in file order. A capture is a
// stream of Export*ServiceRequest objects — one per POST body, or one per
// collector flush — so a single object, NDJSON, and objects concatenated with
// no separator at all are the same input to this reader.
type otlpExport []otlpBatch

// hasEnvelope reports whether any batch carried an OTLP envelope. A file that
// parses as JSON but has neither resourceMetrics nor resourceLogs is not a
// broken export; it is not an export.
func (e otlpExport) hasEnvelope() bool {
	for _, b := range e {
		if len(b.ResourceMetrics) > 0 || len(b.ResourceLogs) > 0 {
			return true
		}
	}
	return false
}

// --- Reading the file ---

// readOTLP decodes every Export*ServiceRequest object in r.
//
// The returned error's message is the reason all three signals will carry, so
// it is written for the person holding the file: it names the shape that could
// not be read, the batch it was in, and where. It never quotes a Go type at
// them.
func readOTLP(r io.Reader) (otlpExport, error) {
	br := bufio.NewReader(r)
	prelude, err := skipPrelude(br)
	if err != nil {
		return nil, err
	}
	// Classify the top level once, before the decoder, so a JSON file that is
	// not an export is diagnosed as what it is rather than as a schema
	// mismatch. A head that starts none of the JSON value kinds is left to the
	// decoder, whose syntax error names the offending byte better than a guess.
	head, _ := br.Peek(5)
	if kind := nonObjectKind(head); kind != "" {
		return nil, fmt.Errorf("top-level JSON value is %s; an OTLP export is a JSON object "+
			"(one ExportMetricsServiceRequest or ExportLogsServiceRequest per object)", kind)
	}

	dec := json.NewDecoder(br)
	var export otlpExport
	// Batch indices are 1-based: batch 1 is the first object in the file, which
	// is what someone counting lines in their capture will call it.
	for batchIdx := 1; dec.More(); batchIdx++ {
		start := prelude + dec.InputOffset()
		var b otlpBatch
		if err := dec.Decode(&b); err != nil {
			return nil, decodeFailure(err, batchIdx, prelude, start)
		}
		export = append(export, b)
	}
	return export, nil
}

// skipPrelude consumes a UTF-8 byte-order mark and any leading whitespace, and
// reports how many bytes that was — so the offsets in failure reasons stay true
// of the file rather than of what the decoder happened to see. An editor or a
// Windows tool leaves a BOM behind; it is not JSON, and it must not cost the
// user the whole capture.
func skipPrelude(br *bufio.Reader) (int64, error) {
	var n int64
	if bom, err := br.Peek(3); err == nil && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		br.Discard(3)
		n += 3
	}
	for {
		c, err := br.ReadByte()
		switch {
		case err == io.EOF:
			return n, errors.New("OTel export file is empty")
		case err != nil:
			return n, fmt.Errorf("failed to read OTel export file: %v", err)
		}
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			br.UnreadByte()
			return n, nil
		}
		n++
	}
}

// nonObjectKind names the kind of JSON value head starts, or "" when it starts
// an object — or something no JSON value starts, which the decoder reports
// against the byte itself.
func nonObjectKind(head []byte) string {
	if len(head) == 0 {
		return ""
	}
	switch c := head[0]; {
	case c == '{':
		return ""
	case c == '[':
		return "an array"
	case c == '"':
		return "a string"
	case c == '-' || (c >= '0' && c <= '9'):
		return "a number"
	case bytes.HasPrefix(head, []byte("true")) || bytes.HasPrefix(head, []byte("false")):
		return "a boolean"
	case bytes.HasPrefix(head, []byte("null")):
		return "null"
	}
	return ""
}

// decodeFailure turns a decoder error into the reason for it. Each arm reads
// the error's type, never its message, where that message names Go types.
func decodeFailure(err error, batchIdx int, prelude, start int64) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntaxErr):
		// SyntaxError names characters, not types, and carries an absolute
		// offset into the stream the decoder read.
		return malformedJSON(prelude+syntaxErr.Offset, batchIdx, syntaxErr.Error())
	case errors.As(err, &typeErr):
		// Field is the JSON path of the value that did not fit, built from the
		// struct tags above. typeErr.Error() would print the Go type it did not
		// fit into, which tells the reader nothing they can act on.
		if typeErr.Field != "" {
			return fmt.Errorf("OTel export does not fit the OTLP schema: value at %s in batch %d is not the expected type",
				typeErr.Field, batchIdx)
		}
		return fmt.Errorf("OTel export does not fit the OTLP schema: a value in batch %d is not the expected type", batchIdx)
	case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, io.EOF):
		// The object began and the file ended. The decoder has no offset for
		// it, so the batch's own start is what there is to name.
		return malformedJSON(start, batchIdx, "unexpected end of JSON input")
	}
	// Anything left is the file failing under us, not its contents.
	return fmt.Errorf("failed to read OTel export file: %v", err)
}

// malformedJSON reports a parse failure at a byte. Nothing from earlier batches
// is used: a partly-read export is how the adapter used to report numbers it
// had not read. Past batch 1 the likeliest cause is a collector stopped
// mid-write, so the reason says what to do about it.
func malformedJSON(offset int64, batchIdx int, detail string) error {
	msg := fmt.Sprintf("malformed JSON at byte %d in batch %d: %s", offset, batchIdx, detail)
	if batchIdx > 1 {
		msg += ". No data from earlier batches was used; if the collector was stopped mid-write, " +
			"delete the final partial line and retry."
	}
	return errors.New(msg)
}

// --- Walking the envelope ---
//
// Multiple resource and scope entries per object are legal and do occur. Both
// walks iterate every level: indexing [0] silently drops the rest of the file.

// metrics yields every metric with the given name, across every batch.
func (e otlpExport) metrics(name string) iter.Seq[otlpMetric] {
	return func(yield func(otlpMetric) bool) {
		for _, b := range e {
			for _, rm := range b.ResourceMetrics {
				for _, sm := range rm.ScopeMetrics {
					for _, m := range sm.Metrics {
						if m.Name != name {
							continue
						}
						if !yield(m) {
							return
						}
					}
				}
			}
		}
	}
}

// logRecords yields every log record, across every batch, in file order.
func (e otlpExport) logRecords() iter.Seq[otlpLogRecord] {
	return func(yield func(otlpLogRecord) bool) {
		for _, b := range e {
			for _, rl := range b.ResourceLogs {
				for _, sl := range rl.ScopeLogs {
					for _, r := range sl.LogRecords {
						if !yield(r) {
							return
						}
					}
				}
			}
		}
	}
}

// --- Reading leaves ---
//
// Every reader returns (value, ok). That second return is the adapter's
// assimilated-value rule made mechanical: absent and zero are different
// answers, and only a value that was read may reach a profile.

// jsonString reads a JSON string leaf.
func jsonString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// jsonInt64 reads a 64-bit integer from a JSON number or a decimal string. The
// OTLP spec accepts both for every int64 field — "64-bit integer numbers ...
// are encoded as decimal strings, and either numbers or strings are accepted
// when decoding" — and the two producers in the documented capture routes
// genuinely differ. The digits reach ParseInt exactly as they arrived, so a
// nanosecond timestamp cannot lose its last digits to a float64. A fractional
// value is not an integer and is refused.
func jsonInt64(raw json.RawMessage) (int64, bool) {
	s, ok := numericText(raw)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

// jsonFloat64 reads a float from a JSON number or a numeric string. Float
// arithmetic touches asDouble and doubleValue and nothing else.
func jsonFloat64(raw json.RawMessage) (float64, bool) {
	s, ok := numericText(raw)
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// numericText is the digits a numeric leaf carries, quoted or not.
func numericText(raw json.RawMessage) (string, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return "", false
	}
	if s[0] == '"' {
		return jsonString(raw)
	}
	return s, true
}

// nanoTime reads a record's timeUnixNano: nanoseconds since the epoch, as a
// decimal string or a bare number. The RFC 3339 event.timestamp attribute
// beside it is deliberately not a fallback — it duplicates this field, and a
// second time source is a branch that breeds.
func nanoTime(raw json.RawMessage) (time.Time, bool) {
	n, ok := jsonInt64(raw)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(0, n).UTC(), true
}

// bodyName reads a log record's body as a name. The body is an AnyValue, and
// three shapes all occur in practice: an object carrying stringValue, a bare
// JSON string, and no body at all. None of them is a reason to fail the record.
func bodyName(raw json.RawMessage) (string, bool) {
	if s, ok := jsonString(raw); ok {
		return s, true
	}
	var body struct {
		StringValue *string `json:"stringValue"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || body.StringValue == nil {
		return "", false
	}
	return *body.StringValue, true
}

// String reads a stringValue attribute. Lookup is a linear scan: a record
// carries at most a dozen attributes and is read two to four times, so a map
// per record would be an allocation and an intermediate representation buying
// nothing.
func (a otlpAttrs) String(key string) (string, bool) {
	for _, at := range a {
		if at.Key != key {
			continue
		}
		if at.Value.StringValue == nil {
			return "", false
		}
		return *at.Value.StringValue, true
	}
	return "", false
}

// Bool reads a boolean attribute. Claude Code documents success as the string
// "true"/"false"; a collector in the path, or a later version, may send a real
// JSON boolean. Both are the same fact, so both read.
func (a otlpAttrs) Bool(key string) (bool, bool) {
	for _, at := range a {
		if at.Key != key {
			continue
		}
		if at.Value.BoolValue != nil {
			return *at.Value.BoolValue, true
		}
		if at.Value.StringValue != nil {
			b, err := strconv.ParseBool(*at.Value.StringValue)
			return b, err == nil
		}
		return false, false
	}
	return false, false
}

// seriesKey identifies the time series a data point belongs to. OTel defines a
// series by its full attribute set, so two token counts that differ only by
// model are two series and must not be merged into one. Sorted by key and
// joined with the value as it arrived, so the key is stable however the
// exporter ordered the attributes.
func (a otlpAttrs) seriesKey() string {
	parts := make([]string, 0, len(a))
	for _, at := range a {
		parts = append(parts, at.Key+"\x01"+at.Value.text())
	}
	sort.Strings(parts)
	return strings.Join(parts, "\x00")
}

// text is the attribute's value as it arrived on the wire, for identity only.
func (v otlpAttrValue) text() string {
	switch {
	case v.StringValue != nil:
		return *v.StringValue
	case v.BoolValue != nil:
		return strconv.FormatBool(*v.BoolValue)
	case len(v.IntValue) > 0:
		return string(v.IntValue)
	case len(v.DoubleValue) > 0:
		return string(v.DoubleValue)
	}
	return ""
}

// value reads a data point's number.
//
// asDouble first: Claude Code creates every counter through one helper that
// never sets a value type, and the OTel JS SDK defaults to DOUBLE, so even an
// integral token count arrives as asDouble. A collector in the path may
// re-serialise the same number as asInt, as digits or as a string.
//
// Round, never truncate: a float64 standing in for an integral count can land a
// hair low, and 1522.7 tokens were 1523 tokens.
func (p otlpDataPoint) value() (int64, bool) {
	if f, ok := jsonFloat64(p.AsDouble); ok {
		return int64(math.Round(f)), true
	}
	return jsonInt64(p.AsInt)
}

// Aggregation temporality, from the OTLP proto enum.
const (
	temporalityDelta      = 1
	temporalityCumulative = 2
)

// temporality reads the sum's aggregation temporality. The OTLP JSON encoding
// mandates the integer enum, but a producer going through the standard protobuf
// JSON mapping emits the name instead, and a collector may quote the digit. All
// three read; anything else does not, and the caller decides what an unreadable
// temporality costs rather than this layer guessing.
func (s otlpSum) temporality() (int64, bool) {
	if n, ok := jsonInt64(s.AggregationTemporality); ok {
		return n, true
	}
	name, ok := jsonString(s.AggregationTemporality)
	if !ok {
		return 0, false
	}
	switch name {
	case "AGGREGATION_TEMPORALITY_UNSPECIFIED":
		return 0, true
	case "AGGREGATION_TEMPORALITY_DELTA":
		return temporalityDelta, true
	case "AGGREGATION_TEMPORALITY_CUMULATIVE":
		return temporalityCumulative, true
	}
	return 0, false
}

// --- Accumulating counter data points ---

// tokenSeries is one time series' contribution to a counter total.
type tokenSeries struct {
	tokenType  string
	value      int64
	timeNanos  int64
	hasTime    bool
	cumulative bool
}

// tokenAccumulator merges counter data points across every batch in a file,
// keyed by series.
//
// Temporality decides how, and the two are opposite instructions: delta points
// are the increments since the last export and add up; cumulative points are
// running totals, so the latest supersedes the rest. Merging per series rather
// than per sum is what lets a capture whose batches were written minutes apart
// still add up, and what keeps two models' token counts from collapsing into
// one.
type tokenAccumulator map[string]*tokenSeries

// add folds one data point into its series.
func (acc tokenAccumulator) add(key, tokenType string, value, timeNanos int64, hasTime, cumulative bool) {
	cur, seen := acc[key]
	if !seen {
		acc[key] = &tokenSeries{
			tokenType:  tokenType,
			value:      value,
			timeNanos:  timeNanos,
			hasTime:    hasTime,
			cumulative: cumulative,
		}
		return
	}

	// A series that ever declares itself cumulative stays cumulative for the
	// rest of the file. No producer mixes the two, and of the two ways to be
	// wrong about one that does, this is the one that cannot double-count.
	cur.cumulative = cur.cumulative || cumulative

	if !cur.cumulative {
		cur.value = addSaturating(cur.value, value)
		if hasTime && (!cur.hasTime || timeNanos > cur.timeNanos) {
			cur.timeNanos, cur.hasTime = timeNanos, true
		}
		return
	}

	// Cumulative: keep the latest point. A timed point beats an untimed one,
	// and with no time to compare — or the same time twice — the greater value
	// wins, which for a running total is the later one by another route.
	switch {
	case hasTime && cur.hasTime:
		if timeNanos > cur.timeNanos || (timeNanos == cur.timeNanos && value > cur.value) {
			cur.value, cur.timeNanos = value, timeNanos
		}
	case hasTime && !cur.hasTime:
		cur.value, cur.timeNanos, cur.hasTime = value, timeNanos, true
	case !hasTime && !cur.hasTime:
		if value > cur.value {
			cur.value = value
		}
	}
}

// reduce totals each token type across every series it was seen on. The merge
// above already turned each series into one number, so this adds series, never
// points.
func (acc tokenAccumulator) reduce() map[string]int64 {
	totals := make(map[string]int64, len(acc))
	for _, s := range acc {
		totals[s.tokenType] = addSaturating(totals[s.tokenType], s.value)
	}
	return totals
}

// addSaturating adds two counts without wrapping. Nothing a session really
// costs overflows an int64; a malformed export can, and a wrapped total would
// serialise as a negative number of tokens — a worse answer than a clamped one.
func addSaturating(a, b int64) int64 {
	sum := a + b
	switch {
	case b > 0 && sum < a:
		return math.MaxInt64
	case b < 0 && sum > a:
		return math.MinInt64
	}
	return sum
}

// clampToInt carries an int64 total into the profile's int fields, saturating
// rather than wrapping where int is 32 bits.
func clampToInt(v int64) int {
	switch {
	case v > math.MaxInt:
		return math.MaxInt
	case v < math.MinInt:
		return math.MinInt
	}
	return int(v)
}
