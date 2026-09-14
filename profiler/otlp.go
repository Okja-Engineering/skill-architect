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

// otlpBatch holds the envelope fields as pointers, because "the export carried
// no value for this field" and "it carried an empty list under it" are
// different answers. The test is the value, not the key: ProtoJSON reads a JSON
// null as the field's default, so {"resourceMetrics":null} names the key and
// still said nothing about metrics, and the pointer is what tells it apart from
// {"resourceMetrics":[]}. The first is a file that is not an OTLP export; the
// second is an OTLP export of a session that emitted nothing yet, and telling
// its owner their file is not OTLP/JSON sends them to fix their exporter
// protocol when nothing is wrong with it.
type otlpBatch struct {
	ResourceMetrics *[]otlpResourceMetrics `json:"resourceMetrics"`
	ResourceLogs    *[]otlpResourceLogs    `json:"resourceLogs"`
}

type otlpResourceMetrics struct {
	ScopeMetrics []otlpScopeMetrics `json:"scopeMetrics"`
}

type otlpScopeMetrics struct {
	Metrics []otlpMetric `json:"metrics"`
}

type otlpResourceLogs struct {
	ScopeLogs []otlpScopeLogs `json:"scopeLogs"`
}

type otlpScopeLogs struct {
	LogRecords []otlpLogRecord `json:"logRecords"`
}

// entries is the slice a pointer envelope field holds, or none. Absent and
// empty walk the same way; only hasEnvelope cares which one arrived.
func entries[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
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
//
// The three composite kinds are decoded but never interpreted. They are here
// because a series is defined by its whole attribute set: an attribute this
// layer has no use for still tells two series apart, and a kind it does not
// decode at all is a kind every value of which looks identical.
type otlpAttrValue struct {
	StringValue *string         `json:"stringValue"`
	BoolValue   *bool           `json:"boolValue"`
	IntValue    json.RawMessage `json:"intValue"`
	DoubleValue json.RawMessage `json:"doubleValue"`
	ArrayValue  json.RawMessage `json:"arrayValue"`
	KvlistValue json.RawMessage `json:"kvlistValue"`
	BytesValue  json.RawMessage `json:"bytesValue"`
}

// otlpExport is every batch in one capture file, in file order. A capture is a
// stream of Export*ServiceRequest objects — one per POST body, or one per
// collector flush — so a single object, NDJSON, and objects concatenated with
// no separator at all are the same input to this reader.
type otlpExport []otlpBatch

// hasEnvelope reports whether any batch carried an OTLP envelope — a
// resourceMetrics or resourceLogs value, not entries under it. A JSON null is
// not a value: ProtoJSON reads null as the field default, so a batch written as
// {"resourceMetrics":null} said nothing about metrics at all. A file that
// parses as JSON but carries neither value is not a broken export; it is not an
// export. One that carries either — an empty list included — is an export with
// no telemetry in it, and each signal reports that for itself.
func (e otlpExport) hasEnvelope() bool {
	for _, b := range e {
		if b.ResourceMetrics != nil || b.ResourceLogs != nil {
			return true
		}
	}
	return false
}

// --- Reading the file ---

// readOTLP decodes every Export*ServiceRequest object in r. Every byte of the
// capture is accounted for: read as a batch, or reported as a failure. Nothing
// between the last batch that decoded and the end of the file is passed over —
// a capture read only as far as its first unreadable byte is part of a file
// reported as a measurement of the file, which a profile has no way to show.
//
// The returned error's message is the reason all three signals will carry, so
// it is written for the person holding the file: it names the shape that could
// not be read, the batch it was in, and where. It never quotes a Go type at
// them.
func readOTLP(r io.Reader) (otlpExport, error) {
	// The count is taken under the buffered reader, so it is bytes pulled from
	// the capture itself. When the file ends mid-object there is no offending
	// byte to name and the length is what there is; see decodeFailure.
	counted := &countingReader{r: r}
	br := bufio.NewReader(counted)
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
	// Decoding is what ends the walk, because only decoding can tell the end of
	// the file from a byte that cannot start a batch. Asking the decoder
	// whether another element follows cannot: it answers no for a stray `}` or
	// `]` as readily as for the end of the file, so a walk driven by that
	// question stops at the first such byte, reports the batches before it, and
	// calls a partial read a complete measurement. io.EOF is the file ending
	// between batches — whitespace after the last batch included, which is how
	// text files end. Every other error is the file failing.
	//
	// Batch indices are 1-based: batch 1 is the first object in the file, which
	// is what someone counting lines in their capture will call it.
	for batchIdx := 1; ; batchIdx++ {
		var b otlpBatch
		err := dec.Decode(&b)
		if errors.Is(err, io.EOF) {
			return export, nil
		}
		if err != nil {
			return nil, decodeFailure(err, batchIdx, prelude, counted.n)
		}
		export = append(export, b)
	}
}

// countingReader counts the bytes pulled from the capture file. It is the only
// thing that knows how long the file was, which is the one place a failure with
// no offset of its own can honestly point at.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
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
			return n, fileReadError(err)
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
//
// One convention for every byte offset it reports: the 0-based offset, in the
// capture file, of the first byte the decoder could not accept. The reader has
// that file open in an editor, so an offset into anything else — the stream
// after the prelude was consumed, the batch's own start — is an offset they
// cannot use.
func decodeFailure(err error, batchIdx int, prelude, fileLen int64) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntaxErr):
		// SyntaxError names characters, not types. Its Offset is the number of
		// bytes read when the error was found, so the offending byte is the one
		// before it, and prelude puts that back in the file's frame.
		return malformedJSON(max(prelude+syntaxErr.Offset-1, 0), batchIdx, syntaxErr.Error())
	case errors.As(err, &typeErr):
		// Field is the JSON path of the value that did not fit, built from the
		// struct tags above. typeErr.Error() would print the Go type it did not
		// fit into, which tells the reader nothing they can act on.
		if typeErr.Field != "" {
			return fmt.Errorf("OTel export does not fit the OTLP schema: value at %s in batch %d is not the expected type",
				typeErr.Field, batchIdx)
		}
		return fmt.Errorf("OTel export does not fit the OTLP schema: a value in batch %d is not the expected type", batchIdx)
	case errors.Is(err, io.ErrUnexpectedEOF):
		// The object began and the file ended. There is no offending byte in
		// the file — the byte the decoder wanted is the one past the end — so
		// the file's length is the honest place to point at. The batch's own
		// start would send the reader to a batch that is fine as far as it
		// goes. A file that ended *between* batches is not a failure at all and
		// never reaches here: that is io.EOF, and it is where the walk ends.
		return malformedJSON(fileLen, batchIdx, "unexpected end of JSON input")
	}
	// Anything left is the file failing under us, not its contents.
	return fileReadError(err)
}

// fileReadError is the reason for a capture file that could not be read as
// bytes — it would not open, or it stopped mid-read. It is the one owner of
// that sentence: the adapter, the prelude and the decoder all reach the same
// failure from different places, and three copies of a string a test asserts on
// are three chances for two of them to drift.
func fileReadError(err error) error {
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
			for _, rm := range entries(b.ResourceMetrics) {
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
			for _, rl := range entries(b.ResourceLogs) {
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
//
// Only a real number reads. JSON has no literal for NaN or infinity, but
// strconv.ParseFloat accepts "NaN", "Inf" and "Infinity" from a quoted string,
// and none of them is a number any arithmetic downstream survives — so none of
// them is a value that was read. An overflowing literal is already refused:
// ParseFloat reports a range error for it rather than returning silently.
func jsonFloat64(raw json.RawMessage) (float64, bool) {
	s, ok := numericText(raw)
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
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
// three shapes are handled: an object carrying stringValue, which is the one
// observed on the wire from Claude Code v2.1.221; a bare JSON string, which the
// OTLP JSON encoding permits and a collector may re-serialise to; and no body
// at all, which attribute cardinality limits can produce. Only the first has
// been seen; the other two are read because they cost one branch each and the
// alternative is losing a record over a formatting difference. None of them is
// a reason to fail the record.
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
// model are two series and must not be merged into one.
//
// Each key and each value goes in with its byte length in front of it, so the
// concatenation is unambiguous and no attribute can spell a key that belongs to
// a different attribute set. A separator character is a character a value can
// carry — a tool name, a prompt, a model id — and therefore one an export can
// forge a foreign series key with. Sorted, so the key is stable however the
// exporter ordered the attributes.
func (a otlpAttrs) seriesKey() string {
	parts := make([]string, 0, len(a))
	for _, at := range a {
		parts = append(parts, lengthPrefixed(at.Key)+lengthPrefixed(at.Value.identity()))
	}
	sort.Strings(parts)
	return strings.Join(parts, "")
}

// lengthPrefixed writes s behind its byte length, which is what lets the parts
// above be concatenated without escaping anything.
func lengthPrefixed(s string) string {
	return strconv.Itoa(len(s)) + ":" + s
}

// identity is the attribute's value as a canonical string, tagged with the kind
// it arrived as. It exists for series identity and for nothing else.
//
// The tag is what makes it injective over AnyValue. Without one, the string
// "5" and the integer 5 are the same value, an absent value and an empty
// string are the same value, and every kind this layer does not read is the
// same empty string — so two series differing only in an arrayValue merged into
// one, and a session's tokens were reported as a fraction of themselves.
//
// The scalar kinds are canonicalised rather than kept verbatim, because the
// OTLP JSON encoding lets one 64-bit integer arrive as 5 or as "5" and those
// are one series, not two. A composite is compacted but not otherwise
// normalised: the order of a composite's members is part of its value.
func (v otlpAttrValue) identity() string {
	switch {
	case v.StringValue != nil:
		return "s" + *v.StringValue
	case v.BoolValue != nil:
		return "b" + strconv.FormatBool(*v.BoolValue)
	case len(v.IntValue) > 0:
		if n, ok := jsonInt64(v.IntValue); ok {
			return "i" + strconv.FormatInt(n, 10)
		}
		return "i?" + compactJSON(v.IntValue)
	case len(v.DoubleValue) > 0:
		if f, ok := jsonFloat64(v.DoubleValue); ok {
			return "d" + strconv.FormatFloat(f, 'g', -1, 64)
		}
		return "d?" + compactJSON(v.DoubleValue)
	case len(v.ArrayValue) > 0:
		return "a" + compactJSON(v.ArrayValue)
	case len(v.KvlistValue) > 0:
		return "k" + compactJSON(v.KvlistValue)
	case len(v.BytesValue) > 0:
		return "y" + compactJSON(v.BytesValue)
	}
	// No kind was set at all, which is not the same answer as any of them.
	return "-"
}

// compactJSON is raw with its insignificant whitespace removed, so a
// pretty-printed capture and a compact one identify the same series.
func compactJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return string(raw)
	}
	return buf.String()
}

// valueKind says how a data point's value read: as a count, as a number that
// cannot be one, or not as a number at all.
//
// The middle answer exists because it is a different defect with a different
// cause, and the caller names it in its own clause. Folding it into "read" is
// what let a number outside the range of a count reach a profile as one;
// folding it into "unreadable" would send the reader looking for a missing
// value they do have.
type valueKind int

const (
	valueUnreadable valueKind = iota // no asDouble or asInt that reads as a number
	valueNotACount                   // a number, but not one a count can be
	valueRead
)

// maxCountPlusOne is 2^63, the first value an int64 cannot hold. float64 cannot
// represent MaxInt64 itself — it rounds up to exactly this — so this is the
// only bound a rounded float can be compared against exactly.
const maxCountPlusOne = float64(1 << 63)

// count reads a data point's value as a count of things, which is what a
// monotonic sum carries and the only kind of sum this layer is asked for.
//
// asDouble first: Claude Code creates every counter through one helper that
// never sets a value type, and the OTel JS SDK defaults to DOUBLE, so even an
// integral token count arrives as asDouble. A collector in the path may
// re-serialise the same number as asInt, as digits or as a string.
//
// Round, never truncate: a float64 standing in for an integral count can land a
// hair low, and 1522.7 tokens were 1523 tokens. The rule below is applied to
// the rounded value, which is what makes a true zero that drifted to -0.4 the
// zero it was, while -0.5 is a number the exporter meant to be negative.
//
// A count is the rounded value when that is at least 0 and below 2^63 — the
// bound maxCountPlusOne names. Anything else decoded, but it is not a count,
// and Go leaves the conversion of an out-of-range float to an integer type up
// to the architecture: assimilating one made the same export read as MaxInt64
// on arm64 and MinInt64 on amd64. It is refused here, where the caller can
// count it and say so, rather than converted into whichever answer the machine
// gives.
func (p otlpDataPoint) count() (int64, valueKind) {
	if f, ok := jsonFloat64(p.AsDouble); ok {
		r := math.Round(f)
		if r < 0 || r >= maxCountPlusOne {
			return 0, valueNotACount
		}
		return int64(r), valueRead
	}
	if n, ok := jsonInt64(p.AsInt); ok {
		if n < 0 {
			return 0, valueNotACount
		}
		return n, valueRead
	}
	return 0, valueUnreadable
}

// Aggregation temporality, from the OTLP proto enum.
const (
	temporalityDelta      = 1
	temporalityCumulative = 2
)

// temporalityState says how a sum's aggregationTemporality read. Absent and
// unreadable are separate answers because they are separate things to go and
// look at in a capture, and because a sum that declared nothing did not declare
// a wrong temporality.
type temporalityState int

const (
	temporalityAbsent     temporalityState = iota // no aggregationTemporality field at all
	temporalityUnreadable                         // present, but neither an enum value nor an enum name
	temporalityDeclared                           // a value was read; what it means is the caller's business
)

// temporality reads the sum's aggregation temporality. The OTLP JSON encoding
// mandates the integer enum, but a producer going through the standard protobuf
// JSON mapping emits the name instead, and a collector may quote the digit. All
// three read; anything else does not, and the caller decides what each answer
// costs rather than this layer guessing.
func (s otlpSum) temporality() (int64, temporalityState) {
	if len(s.AggregationTemporality) == 0 {
		return 0, temporalityAbsent
	}
	if n, ok := jsonInt64(s.AggregationTemporality); ok {
		return n, temporalityDeclared
	}
	name, ok := jsonString(s.AggregationTemporality)
	if !ok {
		return 0, temporalityUnreadable
	}
	switch name {
	case "AGGREGATION_TEMPORALITY_UNSPECIFIED":
		return 0, temporalityDeclared
	case "AGGREGATION_TEMPORALITY_DELTA":
		return temporalityDelta, temporalityDeclared
	case "AGGREGATION_TEMPORALITY_CUMULATIVE":
		return temporalityCumulative, temporalityDeclared
	}
	return 0, temporalityUnreadable
}

// --- Accumulating counter data points ---

// counterSeries is one time series' contribution to a counter total, under the
// label its caller totals by.
type counterSeries struct {
	label      string
	value      int64
	timeNanos  int64
	hasTime    bool
	cumulative bool
}

// counterAccumulator merges counter data points across every batch in a file,
// keyed by series.
//
// Temporality decides how, and the two are opposite instructions: delta points
// are the increments since the last export and add up; cumulative points are
// running totals, so the latest supersedes the rest. Merging per series rather
// than per sum is what lets a capture whose batches were written minutes apart
// still add up, and what keeps two models' counts from collapsing into one.
//
// The label is whatever the caller totals by — the token type, for this file's
// only caller today. Nothing here knows what a token is: this is the format
// layer, and a second OTLP-speaking adapter counting something else would use
// it unchanged.
type counterAccumulator map[string]*counterSeries

// add folds one data point into its series.
func (acc counterAccumulator) add(key, label string, value, timeNanos int64, hasTime, cumulative bool) {
	cur, seen := acc[key]
	if !seen {
		acc[key] = &counterSeries{
			label:      label,
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

// reduce totals each label across every series it was seen on. The merge above
// already turned each series into one number, so this adds series, never
// points. A label no series carried is absent from the map, which is how the
// caller tells "nothing was read for this" from "this came to zero".
func (acc counterAccumulator) reduce() map[string]int64 {
	totals := make(map[string]int64, len(acc))
	for _, s := range acc {
		totals[s.label] = addSaturating(totals[s.label], s.value)
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
func clampToInt(v int64) int { return int(clampToRange(v, math.MinInt, math.MaxInt)) }

// clampToRange is clampToInt with its bounds made arguments. On a 64-bit build
// the bounds clampToInt passes are the whole range of an int64, so there is no
// input that exercises the saturation at all — and a unit test that cannot
// reach the behaviour it names is a test that would go green if the behaviour
// were deleted. The bounds are a parameter so the 32-bit case can be reached on
// the machine this is developed on.
func clampToRange(v, min, max int64) int64 {
	switch {
	case v > max:
		return max
	case v < min:
		return min
	}
	return v
}
