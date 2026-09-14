package profiler

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Tests for the OTLP/JSON format layer that carry no fixture file: a byte-order
// mark is invisible in a reviewed text file, and a bare top-level scalar is not
// worth a file each. They go through the adapter, because the adapter's reasons
// are what a user reads when one of these files lands on them.

// writeExport puts raw bytes where the adapter will read them, for inputs that
// cannot be reviewed as a fixture.
func writeExport(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "otel.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const metricsBatch = `{"resourceMetrics":[{"scopeMetrics":[{"metrics":[{"name":"claude_code.token.usage",` +
	`"sum":{"aggregationTemporality":1,"dataPoints":[{"attributes":[{"key":"type","value":{"stringValue":"input"}}],` +
	`"timeUnixNano":"1789332596272000000","asDouble":%VALUE%}]}}]}]}]}`

func metricsBatchWith(value string) string { return strings.ReplaceAll(metricsBatch, "%VALUE%", value) }

// An editor, a shell redirect, or a Windows tool can leave a UTF-8 byte-order
// mark at the head of a capture file. It is not JSON, and it must not cost the
// user the whole export.
func TestOTLP_LeadingByteOrderMarkIsStripped(t *testing.T) {
	path := writeExport(t, "\xef\xbb\xbf"+metricsBatchWith("1523"))
	adapter := ClaudeCodeAdapter{OtelExportFile: path}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.State != MetricPresent {
		t.Fatalf("tokens state = %q (reason %q), want present — a BOM is not a parse failure",
			profile.Tokens.State, profile.Tokens.Reason)
	}
	assertTokenJSON(t, profile.Tokens.Value, `{"input":1523}`)
}

// One object per line is the framing both documented capture routes produce,
// but nothing in OTLP requires the newline: a receiver that appends bodies
// without one produces a legal stream of concatenated objects.
func TestOTLP_ConcatenatedObjectsNeedNoNewline(t *testing.T) {
	path := writeExport(t, metricsBatchWith("100")+metricsBatchWith("200"))
	adapter := ClaudeCodeAdapter{OtelExportFile: path}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.Value == nil {
		t.Fatalf("tokens value is nil (state %q, reason %q)", profile.Tokens.State, profile.Tokens.Reason)
	}
	// Both objects are batches of the same capture.
	assertTokenJSON(t, profile.Tokens.Value, `{"input":300}`)
}

// --- Where the file ends ---
//
// The reader's contract over a stream of batches is that every byte of the
// capture is accounted for: read as a batch, or reported as a failure. Nothing
// between the last batch that decoded and the end of the file may be passed
// over. It is a contract easy to lose by accident, because there are two ways
// to ask a JSON decoder whether the stream is done and they disagree about a
// stray closing delimiter: "is there another element" answers no for a lone `}`
// or `]` exactly as it does for the end of the file, so a walk driven by that
// question stops at the first byte it cannot begin a value with, reports
// whatever it had read up to there, and calls a partial read a measurement —
// the one failure a profile has no way to show its reader.
//
// These pin the contract rather than the walk: however the walk is driven, a
// tail that is not a batch fails the whole file, and whitespace is not a tail.

// A file whose last batch is followed by anything the reader cannot decode is a
// file whose end was never reached. Reporting the batches before it would report
// part of a capture as a measurement of the capture.
func TestOTLP_ATailThatIsNotABatchFailsTheWholeFile(t *testing.T) {
	for _, tc := range []struct {
		name string
		tail string
	}{
		{"a stray closing brace", "}"},
		{"a stray closing brace on its own line", "}\n"},
		{"a stray closing bracket", "]"},
		{"a stray closing bracket on its own line", "]\n"},
		{"a stray comma, as unwrapping an array by hand leaves", ","},
		{"text that is not JSON", "not json\n"},
		{"a batch that ends mid-object", `{"resourceMetrics":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := metricsBatchWith("100") + "\n" + tc.tail

			// At the reader: the file has no readable end, so it has no value.
			if export, err := readOTLP(strings.NewReader(content)); err == nil {
				t.Fatalf("a file ending in %q read as %d batches and no failure; its tail was neither read nor reported",
					tc.tail, len(export))
			}

			// At the adapter: what the person holding the file is told.
			adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, content)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			got := profile.Tokens.RawMetricResult
			if got.State != MetricError {
				t.Errorf("tokens state = %q carrying %v, want error: the first batch is not a measurement of this file",
					got.State, profile.Tokens.Value)
			}
			if profile.Tokens.Value != nil {
				t.Errorf("tokens value = %v; a file that could not be read to its end carries no count", profile.Tokens.Value)
			}
			// The tail is in the second batch position, and the reason says
			// what became of the batch that did decode.
			if !strings.Contains(got.Reason, "in batch 2") {
				t.Errorf("reason = %q, want it to name batch 2 — the position the unreadable tail is in", got.Reason)
			}
			if !strings.Contains(got.Reason, "No data from earlier batches was used") {
				t.Errorf("reason = %q, want it to say the earlier batch was not used", got.Reason)
			}
			if strings.Contains(got.Reason, "profiler.") {
				t.Errorf("reason = %q names a Go type; it is read by users, not by the compiler", got.Reason)
			}
		})
	}
}

// The tail that costs the most is the one with a real batch behind it: a
// collector flush lands after a stray byte, and a reader that stops at the
// stray byte drops a batch it never mentions. Neither number may be reported —
// not the batch before the tail, which would be part of a capture reported as
// the whole of it, and not the batch after it, which would be a parse failure
// swallowed in order to keep reading.
func TestOTLP_ABatchAfterAnUnreadableTailIsNotReadPast(t *testing.T) {
	const dropped = "424242" // distinctive, so its absence is provable by search
	content := metricsBatchWith("100") + "\n}\n" + metricsBatchWith(dropped) + "\n"

	if export, err := readOTLP(strings.NewReader(content)); err == nil {
		t.Fatalf("read %d batches and no failure across a stray `}`; the batch behind it was dropped in silence", len(export))
	}

	adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, content)}
	profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Tokens.State != MetricError {
		t.Errorf("tokens state = %q carrying %v, want error", profile.Tokens.State, profile.Tokens.Value)
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), dropped) {
		t.Errorf("the profile carries %s, from the batch behind the stray `}`: a parse failure was read past rather than reported\n%s",
			dropped, data)
	}
}

// Whitespace after the last batch is how every text file ends. It is the end of
// the file, not a tail, and a reader strict about its end must still reach it.
func TestOTLP_AFileEndsAtItsLastBatchOrTheWhitespaceAfterIt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		batches int
		want    string
	}{
		{"one batch, no trailing byte", metricsBatchWith("100"), 1, `{"input":100}`},
		{"a final newline", metricsBatchWith("100") + "\n", 1, `{"input":100}`},
		{"blank lines and indentation", metricsBatchWith("100") + "\n\n  \t\r\n", 1, `{"input":100}`},
		{"two batches", metricsBatchWith("100") + "\n" + metricsBatchWith("500"), 2, `{"input":600}`},
		{"two batches and a final newline", metricsBatchWith("100") + "\n" + metricsBatchWith("500") + "\n", 2, `{"input":600}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			export, err := readOTLP(strings.NewReader(tc.content))
			if err != nil {
				t.Fatalf("a file that ends cleanly was refused: %v", err)
			}
			if len(export) != tc.batches {
				t.Fatalf("read %d batches, want %d", len(export), tc.batches)
			}
			adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, tc.content)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			if profile.Tokens.State != MetricPresent {
				t.Fatalf("tokens state = %q (reason %q), want present", profile.Tokens.State, profile.Tokens.Reason)
			}
			assertTokenJSON(t, profile.Tokens.Value, tc.want)
		})
	}
}

// A top-level value that is not an object is a JSON file, not an OTLP export.
// The reason says which kind arrived, so the user can see what they pointed at,
// and never quotes a Go type name at them.
func TestOTLP_TopLevelValueThatIsNotAnObject(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantIn  string
	}{
		{"a string", `"claude_code"`, "top-level JSON value is a string"},
		{"a number", `1523`, "top-level JSON value is a number"},
		{"a boolean", `true`, "top-level JSON value is a boolean"},
		{"null", `null`, "top-level JSON value is null"},
		{"whitespace only", "  \n\t\n", "OTel export file is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, tc.content)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			got := profile.Tokens.RawMetricResult
			if got.State != MetricError {
				t.Errorf("state = %q (reason %q), want error", got.State, got.Reason)
			}
			if !strings.Contains(got.Reason, tc.wantIn) {
				t.Errorf("reason = %q, want it to contain %q", got.Reason, tc.wantIn)
			}
			if strings.Contains(got.Reason, "profiler.") {
				t.Errorf("reason = %q names a Go type; it is read by users, not by the compiler", got.Reason)
			}
		})
	}
}

// An envelope key written as null carries no envelope. ProtoJSON reads null as
// the field's default, so an exporter that wrote {"resourceMetrics":null} said
// nothing about metrics, and a file that says nothing about either signal is
// not an OTLP export — which is a different thing to tell its owner than "your
// export is empty". The key written as an empty list *is* an export, of a
// session that emitted nothing; testdata/otlp/empty_envelope.json is that case
// as a reviewed fixture. The null spelling gets no fixture of its own and is
// pinned inline here beside it, because a second file differing from the first
// by one word on one line is one a reviewer reads straight past as a duplicate.
func TestOTLP_ANullEnvelopeIsNoEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantIn  string
		wantOut string
	}{
		{"null", `{"resourceMetrics":null,"resourceLogs":null}`,
			"OTel export is not OTLP/JSON", "no claude_code.token.usage metric found"},
		{"an empty list", `{"resourceMetrics":[]}`,
			"no claude_code.token.usage metric found", "not OTLP/JSON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, tc.content)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			got := profile.Tokens.RawMetricResult
			if got.State != MetricUnknown {
				t.Fatalf("state = %q (reason %q), want unknown — the file parsed", got.State, got.Reason)
			}
			if !strings.Contains(got.Reason, tc.wantIn) {
				t.Errorf("reason = %q, want it to contain %q", got.Reason, tc.wantIn)
			}
			if strings.Contains(got.Reason, tc.wantOut) {
				t.Errorf("reason = %q, want it not to contain %q", got.Reason, tc.wantOut)
			}
		})
	}
}

// A value that does not fit the OTLP schema names the field path it was found
// at — when the decoder has one to give. It does not for a batch whose own top
// level is the wrong shape: there is no path *inside* the batch to name, so the
// reason names the batch and stops. Both wordings are what a user reads, and
// the docs describe both, so both are pinned here. Only the first batch of a
// file gets the "top-level JSON value is an array" reading; every batch after
// it reaches the decoder, which is why this takes two.
func TestOTLP_ASchemaMismatchNamesAFieldPathOnlyWhenThereIsOne(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantIn  []string
		wantOut []string
	}{
		{
			name:    "a wrong type inside a batch names the path to it",
			content: `{"resourceMetrics":[{"scopeMetrics":"nope"}]}`,
			wantIn:  []string{"does not fit the OTLP schema", "value at resourceMetrics", "in batch 1"},
		},
		{
			name:    "a batch that is not an object has no path to name",
			content: "{\"resourceMetrics\":[]}\n[1,2]\n",
			wantIn:  []string{"does not fit the OTLP schema", "a value in batch 2"},
			wantOut: []string{"value at"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, tc.content)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			got := profile.Tokens.RawMetricResult
			if got.State != MetricError {
				t.Fatalf("state = %q (reason %q), want error", got.State, got.Reason)
			}
			for _, want := range tc.wantIn {
				if !strings.Contains(got.Reason, want) {
					t.Errorf("reason = %q, want it to contain %q", got.Reason, want)
				}
			}
			for _, unwanted := range tc.wantOut {
				if strings.Contains(got.Reason, unwanted) {
					t.Errorf("reason = %q, want it not to contain %q", got.Reason, unwanted)
				}
			}
			if strings.Contains(got.Reason, "profiler.") {
				t.Errorf("reason = %q names a Go type; it is read by users, not by the compiler", got.Reason)
			}
		})
	}
}

// The accumulator's totals are int64 and the profile's counts are int. Nothing
// a real session costs comes near either limit, but a malformed export can, and
// a wrapped total serialises as a negative number of tokens — an answer that is
// wrong in a direction no reader would question. There is no fixture for this:
// a 2^63 token count in a reviewed file teaches nobody anything.
func TestCounterAccumulator_TotalsSaturateRatherThanWrap(t *testing.T) {
	acc := counterAccumulator{}
	// Two series under the same label, summed at reduce.
	acc.add("model=a", delta(math.MaxInt64, at(1)))
	acc.add("model=b", delta(1000, at(1)))
	// One series accumulating deltas past the limit.
	acc.add("model=c", delta(math.MaxInt64, at(1)).as(tokenTypeOutput))
	acc.add("model=c", delta(math.MaxInt64, at(2)).as(tokenTypeOutput))
	// One cumulative series whose runs sum past the limit: a reset adds the
	// run before it, and that addition has the same job to do.
	acc.add("model=d", cumulative(math.MaxInt64, at(1)).from(at(1)).as(tokenTypeCacheCreation))
	acc.add("model=d", cumulative(math.MaxInt64, at(3)).from(at(2)).as(tokenTypeCacheCreation))

	totals := acc.reduce()
	if got := totals[tokenTypeInput]; got != math.MaxInt64 {
		t.Errorf("input total = %d, want %d — summing series must saturate", got, int64(math.MaxInt64))
	}
	if got := totals[tokenTypeOutput]; got != math.MaxInt64 {
		t.Errorf("output total = %d, want %d — accumulating deltas must saturate", got, int64(math.MaxInt64))
	}
	if got := totals[tokenTypeCacheCreation]; got != math.MaxInt64 {
		t.Errorf("cache_creation total = %d, want %d — summing a series' runs must saturate", got, int64(math.MaxInt64))
	}
	// A label no series carried is absent, not zero: that is how the caller
	// tells "nothing was read for this" from "this came to nothing".
	if _, ok := totals[tokenTypeCacheRead]; ok {
		t.Errorf("totals carry %q, which no series was added under", tokenTypeCacheRead)
	}
}

// The clamp that carries an int64 total into the profile's int fields only ever
// does anything where int is 32 bits. On the machine this is developed on, and
// on CI, MinInt..MaxInt is the whole range of an int64 — so a test that calls
// clampToInt exercises no clamping at all and would pass just as well with the
// clamping deleted. The bounds are a parameter so the behaviour can be reached.
func TestClampToRange_SaturatesAtTheBoundsItIsGiven(t *testing.T) {
	const min32, max32 = math.MinInt32, math.MaxInt32
	for _, tc := range []struct {
		v    int64
		want int64
	}{
		{math.MaxInt64, max32},
		{max32 + 1, max32},
		{max32, max32},
		{1523, 1523},
		{0, 0},
		{min32, min32},
		{min32 - 1, min32},
		{math.MinInt64, min32},
	} {
		if got := clampToRange(tc.v, min32, max32); got != tc.want {
			t.Errorf("clampToRange(%d, %d, %d) = %d, want %d", tc.v, int64(min32), int64(max32), got, tc.want)
		}
	}
	// And the bounds clampToInt itself passes carry a count through unchanged.
	if got := clampToInt(1523); got != 1523 {
		t.Errorf("clampToInt(1523) = %d, want 1523 — a count in range is carried unchanged", got)
	}
	if got := clampToInt(math.MaxInt64); got != math.MaxInt {
		t.Errorf("clampToInt(MaxInt64) = %d, want %d", got, math.MaxInt)
	}
}

// --- Leaf readers: decoded is not the same as representable ---
//
// Every reader returns (value, ok), and that second return is the adapter's
// assimilated-value rule. It is only true if "ok" also means "the value fits
// the domain it is being read into": a leaf that decodes to a float64 the
// profile cannot carry as a count is not a value that was read.

// attrs decodes an attribute list from the wire JSON, so these tests speak the
// shape an exporter writes rather than the struct the decoder happens to have.
func attrs(t *testing.T, wire string) otlpAttrs {
	t.Helper()
	var a otlpAttrs
	if err := json.Unmarshal([]byte(wire), &a); err != nil {
		t.Fatalf("attribute list does not decode: %v", err)
	}
	return a
}

// A float leaf reads a number, and JSON has no literal for NaN or infinity.
// strconv.ParseFloat accepts all of them from a quoted string, and none is a
// number any arithmetic downstream survives: the leaf must refuse them rather
// than hand its caller a float that is not one.
func TestOTLP_FloatLeafReadsOnlyRealNumbers(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want float64
		ok   bool
	}{
		{raw: `1522.7`, want: 1522.7, ok: true},
		{raw: `"1522.7"`, want: 1522.7, ok: true},
		{raw: `1e300`, want: 1e300, ok: true},
		{raw: `0`, want: 0, ok: true},
		{raw: `"NaN"`},
		{raw: `"nan"`},
		{raw: `"Inf"`},
		{raw: `"+Inf"`},
		{raw: `"-Inf"`},
		{raw: `"Infinity"`},
		{raw: `"-Infinity"`},
		{raw: `1e400`},
		{raw: `"tokens"`},
		{raw: `true`},
		{raw: ``},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, ok := jsonFloat64([]byte(tc.raw))
			if ok != tc.ok {
				t.Fatalf("jsonFloat64(%s) ok = %v, want %v — a leaf that is not a real number must read as unreadable",
					tc.raw, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("jsonFloat64(%s) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

// An integer leaf reads an integer. A fraction is not one, and rounding it
// would move a nanosecond timestamp to a different instant.
func TestOTLP_IntegerLeafRefusesWhatIsNotADecimalInteger(t *testing.T) {
	for _, raw := range []string{`1522.7`, `"1522.7"`, `1e3`, `"0x10"`, `"NaN"`, `""`, `"12 "`, `true`} {
		if got, ok := jsonInt64([]byte(raw)); ok {
			t.Errorf("jsonInt64(%s) = %d, ok — only a decimal integer is one", raw, got)
		}
	}
	for _, tc := range []struct {
		raw  string
		want int64
	}{
		{`1523`, 1523},
		{`"1523"`, 1523},
		{`-1`, -1},
		{`"9223372036854775807"`, math.MaxInt64},
	} {
		got, ok := jsonInt64([]byte(tc.raw))
		if !ok || got != tc.want {
			t.Errorf("jsonInt64(%s) = %d, %v; want %d, true", tc.raw, got, ok, tc.want)
		}
	}
}

// kindName is for the failure messages below: valueKind is an unnamed int, and
// "read as 1" tells the reader nothing about what went wrong.
func kindName(k valueKind) string {
	switch k {
	case valueUnreadable:
		return "valueUnreadable"
	case valueNotACount:
		return "valueNotACount"
	case valueRead:
		return "valueRead"
	}
	return "valueKind(" + strconv.Itoa(int(k)) + ")"
}

// count is where a decoded number becomes a count, and it is the one leaf
// reader whose boundary a float64 cannot land on. There is no float64 for
// MaxInt64: the literal 9223372036854775807 parses to exactly 2^63 — the first
// value an int64 cannot hold — and so does every literal down to
// 9223372036854775296, the midpoint between 2^63 and the largest float64 below
// it, which the table below pins. Not "within 512 of MaxInt64": MaxInt64−512 is
// 9223372036854775295, one below that midpoint, and it rounds down. The
// accepted window has to close *below* the bound rather than at it, and no
// fixture can show the difference: 9.3e+18 is refused either way, while a
// capture carrying MaxInt64 would be assimilated as MaxInt64 on arm64 and
// MinInt64 on amd64, the same export yielding two different profiles.
//
// The invariant, stated where it can fail: a value becomes a count if and only
// if it is finite and 0 <= round(v) < 2^63, and the count it becomes is the
// same number on every architecture. The three-way answer is part of it — a
// number a count cannot be is a different defect, with a different thing to go
// and look at in the capture, from a leaf that carried no number at all.
func TestOTLP_CountIsExactlyTheValuesAnInt64Holds(t *testing.T) {
	const (
		// The largest float64 below 2^63: the spacing between neighbours up
		// there is 1024, so this is the largest count an asDouble can carry.
		maxCountAsDouble = 9223372036854774784
		// Halfway between that neighbour and 2^63, which ties-to-even resolves
		// *up* to 2^63. It is under MaxInt64 as written and still not a count.
		roundsUpToTheBound = "9223372036854775296"
	)
	for _, tc := range []struct {
		name  string
		wire  string
		want  valueKind
		count int64
	}{
		// The boundary itself, in the spellings an exporter writes.
		{"MaxInt64 as a bare double", `{"asDouble":9223372036854775807}`, valueNotACount, 0},
		{"MaxInt64 as a quoted double", `{"asDouble":"9223372036854775807"}`, valueNotACount, 0},
		{"2^63 itself", `{"asDouble":9223372036854775808}`, valueNotACount, 0},
		{"a double that rounds up to 2^63", `{"asDouble":` + roundsUpToTheBound + `}`, valueNotACount, 0},
		{"the largest double below 2^63", `{"asDouble":9223372036854774784}`, valueRead, maxCountAsDouble},
		{"far past the bound", `{"asDouble":1e300}`, valueNotACount, 0},

		// The rest of the accepted window.
		{"a count in range", `{"asDouble":1523}`, valueRead, 1523},
		{"a fractional count rounds, never truncates", `{"asDouble":1522.7}`, valueRead, 1523},
		{"zero", `{"asDouble":0}`, valueRead, 0},
		// Float drift on a true zero lands a hair under it. Rounding carries
		// it back to the zero it was, and nothing below -0.5 is absorbed.
		{"drift below zero rounds back to zero", `{"asDouble":-0.4}`, valueRead, 0},
		{"a half count below zero rounds away from it", `{"asDouble":-0.5}`, valueNotACount, 0},
		{"a negative count", `{"asDouble":-500}`, valueNotACount, 0},

		// Not a number at all: refused by the float leaf, so the point carried
		// no readable value rather than an unusable one.
		{"NaN", `{"asDouble":"NaN"}`, valueUnreadable, 0},
		{"infinity", `{"asDouble":"Infinity"}`, valueUnreadable, 0},
		{"no value at all", `{}`, valueUnreadable, 0},

		// asInt is read as an integer, so it holds exactly the value the float
		// path cannot, and overflows out of the reader instead of past it.
		{"MaxInt64 as an int", `{"asInt":"9223372036854775807"}`, valueRead, math.MaxInt64},
		{"an int past MaxInt64", `{"asInt":"9223372036854775808"}`, valueUnreadable, 0},
		{"a negative int", `{"asInt":"-1"}`, valueNotACount, 0},

		// asDouble first, because Claude Code's own exporter writes every
		// counter as one; asInt is the fallback a collector in the path leaves.
		{"asDouble is read before asInt", `{"asDouble":1523,"asInt":"9"}`, valueRead, 1523},
		{"asInt is read when asDouble is not a number", `{"asDouble":"tokens","asInt":"9"}`, valueRead, 9},
		// NaN is refused by the float leaf rather than answered by it, so it
		// does not shadow asInt the way an out-of-range finite double does.
		// This is the case docs/profiler-spec.md had classified as a refusal.
		{"asInt is read when asDouble is NaN", `{"asDouble":"NaN","asInt":"9"}`, valueRead, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p otlpDataPoint
			if err := json.Unmarshal([]byte(tc.wire), &p); err != nil {
				t.Fatalf("data point %s does not decode: %v", tc.wire, err)
			}
			got, kind := p.count()
			if kind != tc.want {
				t.Fatalf("count(%s) read as %s, want %s", tc.wire, kindName(kind), kindName(tc.want))
			}
			if got != tc.count {
				t.Errorf("count(%s) = %d, want %d", tc.wire, got, tc.count)
			}
		})
	}
}

// A data point's attributes are the part of a series' identity the point
// carries itself, so the encoding must be injective over them: two attribute
// sets that differ anywhere are two series and must not merge, and two that
// differ only in how the exporter ordered or spelled the same values are one
// series and must not split.
//
// The encoding is internal and never shown to anyone, so these cases pin
// identity rather than any particular spelling of it.
func TestOTLP_AttributeIdentityIsTheSetNotItsText(t *testing.T) {
	// Every entry is a different attribute set, so every key must be
	// different. Asserting injectivity over a corpus rather than pair by pair
	// is what makes the test independent of how the key is built: the corpus
	// carries values that spell the separator bytes a joining encoding would
	// use, the length prefixes a length-prefixed one would use, and the kind
	// tags either would need — so a key any of them can forge is a collision
	// this test finds without knowing which one is in use.
	corpus := []string{
		`[]`,
		// Absent, empty, and present: three answers, not one.
		`[{"key":"m","value":{}}]`,
		`[{"key":"m","value":{"stringValue":""}}]`,
		`[{"key":"m","value":{"stringValue":"a"}}]`,
		`[{"key":"m","value":{"stringValue":"b"}}]`,

		// A scalar and the string that prints the same way.
		`[{"key":"m","value":{"stringValue":"5"}}]`,
		`[{"key":"m","value":{"intValue":5}}]`,
		`[{"key":"m","value":{"intValue":6}}]`,
		`[{"key":"m","value":{"doubleValue":5}}]`,
		`[{"key":"m","value":{"boolValue":true}}]`,
		`[{"key":"m","value":{"stringValue":"true"}}]`,

		// Strings that spell what a tagged encoding writes for another kind.
		`[{"key":"m","value":{"stringValue":"i5"}}]`,
		`[{"key":"m","value":{"stringValue":"s5"}}]`,
		`[{"key":"m","value":{"stringValue":"btrue"}}]`,

		// The composite kinds, which this layer never interprets and still has
		// to tell apart.
		`[{"key":"m","value":{"arrayValue":{"values":[{"stringValue":"Read"}]}}}]`,
		`[{"key":"m","value":{"arrayValue":{"values":[{"stringValue":"Bash"}]}}}]`,
		`[{"key":"m","value":{"arrayValue":{"values":[]}}}]`,
		`[{"key":"m","value":{"kvlistValue":{"values":[]}}}]`,
		`[{"key":"m","value":{"kvlistValue":{"values":[{"key":"a","value":{"stringValue":"1"}}]}}}]`,
		`[{"key":"m","value":{"kvlistValue":{"values":[{"key":"a","value":{"stringValue":"2"}}]}}}]`,
		`[{"key":"m","value":{"bytesValue":"YQ=="}}]`,
		`[{"key":"m","value":{"bytesValue":"Yg=="}}]`,

		// Forgeries: a key or value carrying the bytes an encoding joins with,
		// beside the attribute set it would otherwise be indistinguishable
		// from. A tool name, a model id and a prompt are all arbitrary text.
		`[{"key":"m","value":{"stringValue":"a\u0001b"}}]`,
		`[{"key":"m\u0001a","value":{"stringValue":"b"}}]`,
		`[{"key":"m","value":{"stringValue":"a\u0001sb"}}]`,
		`[{"key":"m\u0001sa","value":{"stringValue":"b"}}]`,
		`[{"key":"m","value":{"stringValue":"a\u0000z\u0001v"}}]`,
		`[{"key":"m","value":{"stringValue":"a\u0000z\u0001sv"}}]`,
		`[{"key":"m","value":{"stringValue":"a"}},{"key":"z","value":{"stringValue":"v"}}]`,
		`[{"key":"m","value":{"stringValue":"1:a"}}]`,
		`[{"key":"1:m","value":{"stringValue":"a"}}]`,
		`[{"key":"m","value":{"stringValue":"a"}},{"key":"1:z","value":{"stringValue":"v"}}]`,
	}
	keyed := make(map[string]string, len(corpus))
	for _, wire := range corpus {
		key := attrs(t, wire).identity()
		if before, collides := keyed[key]; collides {
			t.Errorf("two different attribute sets share the identity %q, so they would merge into one series:\n  %s\n  %s",
				key, before, wire)
			continue
		}
		keyed[key] = wire
	}

	same := []struct {
		name string
		a, b string
	}{
		{"the OTLP spec's two spellings of a 64-bit integer",
			`[{"key":"m","value":{"intValue":5}}]`,
			`[{"key":"m","value":{"intValue":"5"}}]`},
		{"the OTLP spec's two spellings of a double",
			`[{"key":"m","value":{"doubleValue":1.5}}]`,
			`[{"key":"m","value":{"doubleValue":"1.5"}}]`},
		{"the same attributes in either order",
			`[{"key":"type","value":{"stringValue":"input"}},{"key":"model","value":{"stringValue":"opus"}}]`,
			`[{"key":"model","value":{"stringValue":"opus"}},{"key":"type","value":{"stringValue":"input"}}]`},
	}
	for _, tc := range same {
		t.Run("same/"+tc.name, func(t *testing.T) {
			if a, b := attrs(t, tc.a).identity(), attrs(t, tc.b).identity(); a != b {
				t.Errorf("%s keys as %q and %s as %q — one series would split in two",
					tc.a, a, tc.b, b)
			}
		})
	}
}

// The attribute set is one of four parts. OTel identifies a series by the
// resource it came from, the scope that recorded it, the metric's name and
// those attributes, and the whole of that has to be injective too: a capture
// that aggregates two service instances carries two series whose data points
// are identical, and merging them reports one instance's running total as both.
//
// The corpus differs in exactly one part per entry, and carries the entries
// where one part's text would be another part's if the parts were run together.
func TestOTLP_SeriesIdentityIsAllFourPartsOfIt(t *testing.T) {
	corpus := []struct {
		name    string
		metric  string
		batches []string
	}{
		{name: "nothing but the metric and one point", metric: "m", batches: []string{batch("", "", `[]`)}},
		{name: "a resource attribute", metric: "m", batches: []string{batch(`"resource":{"attributes":[{"key":"a","value":{"stringValue":"b"}}]},`, "", `[]`)}},
		{name: "a different resource attribute value", metric: "m", batches: []string{batch(`"resource":{"attributes":[{"key":"a","value":{"stringValue":"c"}}]},`, "", `[]`)}},
		{name: "a resource attribute of another AnyValue kind", metric: "m", batches: []string{batch(`"resource":{"attributes":[{"key":"a","value":{"intValue":5}}]},`, "", `[]`)}},
		{name: "the string that kind prints as", metric: "m", batches: []string{batch(`"resource":{"attributes":[{"key":"a","value":{"stringValue":"5"}}]},`, "", `[]`)}},
		{name: "a scope name", metric: "m", batches: []string{batch("", `"scope":{"name":"a"},`, `[]`)}},
		{name: "the same text as a scope version", metric: "m", batches: []string{batch("", `"scope":{"version":"a"},`, `[]`)}},
		{name: "a scope name and version", metric: "m", batches: []string{batch("", `"scope":{"name":"a","version":"b"},`, `[]`)}},
		{name: "the two run together as a name", metric: "m", batches: []string{batch("", `"scope":{"name":"ab"},`, `[]`)}},
		// The same attribute, on each of the three things that can carry one.
		{name: "a scope attribute", metric: "m", batches: []string{batch("", `"scope":{"attributes":[{"key":"a","value":{"stringValue":"b"}}]},`, `[]`)}},
		{name: "a data-point attribute", metric: "m", batches: []string{batch("", "", `[{"key":"a","value":{"stringValue":"b"}}]`)}},
		{name: "another metric's name", metric: "n", batches: []string{batch("", "", `[]`)}},
	}

	seen := make(map[seriesID]string, len(corpus))
	for _, tc := range corpus {
		for _, id := range seriesIDs(t, tc.metric, tc.batches...) {
			if before, collides := seen[id]; collides {
				t.Errorf("%q and %q share a series identity, so two series would merge into one", before, tc.name)
				continue
			}
			seen[id] = tc.name
		}
	}

	// The other direction: one series written across two batches, as every
	// capture of more than one flush is, must stay one series.
	t.Run("same/one series across two batches", func(t *testing.T) {
		b := batch(`"resource":{"attributes":[{"key":"a","value":{"stringValue":"b"}}]},`,
			`"scope":{"name":"s","version":"1"},`, `[{"key":"type","value":{"stringValue":"input"}}]`)
		ids := seriesIDs(t, "m", b, b)
		if len(ids) != 2 {
			t.Fatalf("walked %d data points across two batches, want 2", len(ids))
		}
		if ids[0] != ids[1] {
			t.Errorf("the same series in two batches has two identities, so its points would not merge")
		}
	})
}

// batch is one metrics export object: a metric named m and a metric named n,
// each with one data point, under the resource and scope given. The resource
// and scope arguments are the JSON member and its trailing comma, or empty for
// an entry that carries none — absent and empty are the same identity, and an
// entry that says nothing about its scope is the common case on the wire.
func batch(resource, scope, attributes string) string {
	const metric = `{"name":"%s","sum":{"aggregationTemporality":2,"dataPoints":[{"attributes":%s,` +
		`"startTimeUnixNano":"1","timeUnixNano":"2","asDouble":1}]}}`
	return `{"resourceMetrics":[{` + resource + `"scopeMetrics":[{` + scope + `"metrics":[` +
		fmt.Sprintf(metric, "m", attributes) + "," + fmt.Sprintf(metric, "n", attributes) + `]}]}]}`
}

// seriesIDs is the identity of every data point the named metric carries,
// across the batches given, in walk order.
func seriesIDs(t *testing.T, name string, batches ...string) []seriesID {
	t.Helper()
	export, err := readOTLP(strings.NewReader(strings.Join(batches, "\n")))
	if err != nil {
		t.Fatalf("fixture does not read: %v", err)
	}
	var ids []seriesID
	for m := range export.metrics(name) {
		for _, dp := range m.Sum.DataPoints {
			ids = append(ids, m.series(dp))
		}
	}
	if len(ids) == 0 {
		t.Fatalf("no %q data points in the batches given", name)
	}
	return ids
}

// --- Byte offsets ---
//
// A failure reason points at a byte so the person holding the capture can open
// the file and look at it. That only works if every reason counts bytes the
// same way, in the file they have rather than in whatever the decoder happened
// to be given. The convention: the 0-based offset of the first byte the decoder
// could not accept, and for a file that simply ended, the file's length.

// reasonOffset is the byte offset a malformed-JSON reason names.
func reasonOffset(t *testing.T, reason string) int64 {
	t.Helper()
	m := regexp.MustCompile(`malformed JSON at byte (\d+) in batch (\d+)`).FindStringSubmatch(reason)
	if m == nil {
		t.Fatalf("reason %q names no byte offset", reason)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// The offset is an offset into the file, not into the stream the decoder was
// handed: a byte-order mark and leading whitespace are bytes the reader's
// editor will count, so the reason has to count them too.
func TestOTLP_AnOffsetNamesTheOffendingByteInTheFile(t *testing.T) {
	// The only character in each of these that cannot appear where it does is
	// the `]`, so the offset the reason names must land exactly on it.
	for _, tc := range []struct {
		name    string
		content string
	}{
		{"no prelude", `{"resourceMetrics":]}`},
		{"a byte-order mark", "\xef\xbb\xbf" + `{"resourceMetrics":]}`},
		{"a byte-order mark and leading whitespace", "\xef\xbb\xbf\n  " + `{"resourceMetrics":]}`},
		{"leading whitespace alone", "\n\n\t" + `{"resourceMetrics":]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter := ClaudeCodeAdapter{OtelExportFile: writeExport(t, tc.content)}
			profile, err := adapter.Capture("session-001", CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"})
			if err != nil {
				t.Fatal(err)
			}
			at := reasonOffset(t, profile.Tokens.Reason)
			if at < 0 || at >= int64(len(tc.content)) {
				t.Fatalf("reason names byte %d, outside a file of %d bytes: %q",
					at, len(tc.content), profile.Tokens.Reason)
			}
			if got := tc.content[at]; got != ']' {
				t.Errorf("reason names byte %d, which is %q; the byte the decoder could not accept is the %q at %d",
					at, string(got), "]", strings.IndexByte(tc.content, ']'))
			}
		})
	}
}

// When the file simply ended, there is no offending byte in it: the byte the
// decoder wanted is the one past the end, so the length is what there is to
// name. Naming the batch's first byte instead sends the reader to the start of
// a batch that is fine as far as it goes.
func TestOTLP_AnUnexpectedEndOfFileNamesTheFileLength(t *testing.T) {
	for _, name := range []string{"malformed.json", "truncated_final_line.ndjson"} {
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(fixture(name))
			if err != nil {
				t.Fatal(err)
			}
			profile := capturedProfile(t, name)
			if profile.Tokens.State != MetricError {
				t.Fatalf("tokens state = %q, want error", profile.Tokens.State)
			}
			if at := reasonOffset(t, profile.Tokens.Reason); at != int64(len(content)) {
				t.Errorf("reason names byte %d for a %d-byte file that ends mid-object: %q",
					at, len(content), profile.Tokens.Reason)
			}
		})
	}
}

// --- Merging counter data points ---
//
// Cumulative points are running totals, so the accumulator keeps the latest and
// discards the rest. "Latest" has to be decided for every pair of points it can
// be handed, including the pairs a well-behaved exporter never produces: the
// wrong answer here does not fail, it reports a number.

func TestCounterAccumulator_CumulativeKeepsTheLatestPoint(t *testing.T) {
	const series = "model=a"
	for _, tc := range []struct {
		name string
		// points are folded in in file order, and all belong to one run.
		points []counterPoint
		want   int64
		reason string
	}{
		{
			name:   "a later timestamp supersedes an earlier one",
			points: []counterPoint{cumulative(500, at(1)), cumulative(900, at(2))},
			want:   900,
			reason: "the running total at the later instant is the total",
		},
		{
			name:   "an earlier timestamp does not supersede a later one",
			points: []counterPoint{cumulative(900, at(2)), cumulative(500, at(1))},
			want:   900,
			reason: "file order is not time order; an exporter may write batches out of order",
		},
		{
			name:   "at the same instant, the greater running total wins",
			points: []counterPoint{cumulative(500, at(1)), cumulative(900, at(1))},
			want:   900,
			reason: "a running total only goes up, so the greater one is the later",
		},
		{
			name:   "at the same instant, the greater running total wins wherever it arrived",
			points: []counterPoint{cumulative(900, at(1)), cumulative(500, at(1))},
			want:   900,
			reason: "the same instant orders nothing, so file order must not decide it either — this is " +
				"the order that under-counts if an equal instant is read as a later one",
		},
		{
			name:   "a timed point beats an untimed one",
			points: []counterPoint{cumulative(900, never), cumulative(500, at(1))},
			want:   500,
			reason: "a point that says when it was is better evidence than one that does not",
		},
		{
			name:   "an untimed point does not beat a timed one",
			points: []counterPoint{cumulative(500, at(1)), cumulative(900, never)},
			want:   500,
			reason: "a point that says when it was is better evidence than one that does not",
		},
		{
			name:   "with no times at all, the greater running total wins",
			points: []counterPoint{cumulative(500, never), cumulative(900, never), cumulative(700, never)},
			want:   900,
			reason: "nothing else orders them, and a running total only goes up",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := counterAccumulator{}
			for _, p := range tc.points {
				acc.add(series, p.from(at(1)))
			}
			if got := acc.reduce()[tokenTypeInput]; got != tc.want {
				t.Errorf("total = %d, want %d — %s", got, tc.want, tc.reason)
			}
		})
	}
}

// A cumulative counter that restarts reports a running total from zero again,
// and startTimeUnixNano is what says which run a point is from. The run before
// a restart is tokens the session really spent: it stays, beside the new run
// rather than under it. Nothing here depends on the runs arriving in order — a
// capture is a stream of batches an exporter may have written in any order.
func TestCounterAccumulator_ARunIsIdentifiedByItsStartTime(t *testing.T) {
	const series = "model=a"
	for _, tc := range []struct {
		name   string
		points []counterPoint
		want   int64
		reason string
	}{
		{
			name:   "a reset opens a run beside the one before it",
			points: []counterPoint{cumulative(100, at(10)).from(at(1)), cumulative(20, at(20)).from(at(11))},
			want:   120,
			reason: "the counter restarted, and the 100 it had reached was still spent",
		},
		{
			name: "each run keeps its own latest point",
			points: []counterPoint{
				cumulative(60, at(5)).from(at(1)),
				cumulative(100, at(10)).from(at(1)),
				cumulative(12, at(15)).from(at(11)),
				cumulative(20, at(20)).from(at(11)),
			},
			want:   120,
			reason: "two runs reaching 100 and 20: the points inside a run supersede, the runs add",
		},
		{
			name:   "runs out of file order still add",
			points: []counterPoint{cumulative(20, at(20)).from(at(11)), cumulative(100, at(10)).from(at(1))},
			want:   120,
			reason: "a run is identified by its start time, not by where in the file it arrived",
		},
		{
			name:   "one run does not add to itself",
			points: []counterPoint{cumulative(100, at(10)).from(at(1)), cumulative(120, at(20)).from(at(1))},
			want:   120,
			reason: "the same start is the same run, and its points are running totals of each other",
		},
		{
			name:   "a delta series is one run whatever its points' start times say",
			points: []counterPoint{delta(100, at(10)).from(at(1)), delta(200, at(20)).from(at(11))},
			want:   300,
			reason: "a delta point's start time is the start of its own interval, not of a run: read as " +
				"a run boundary it would leave every batch a run of its own and total the last of each",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := counterAccumulator{}
			for _, p := range tc.points {
				acc.add(series, p)
			}
			if got := acc.reduce()[tokenTypeInput]; got != tc.want {
				t.Errorf("total = %d, want %d — %s", got, tc.want, tc.reason)
			}
		})
	}
}

// A cumulative point whose startTimeUnixNano is absent, zero or unreadable
// cannot say which run it came from. It is evidence that the series reached
// that running total, and it is not evidence of anything else — least of all of
// a counter that restarted, which is the one reading that adds tokens the
// export does not carry.
//
// The invariant every case below pins: an unplaceable point may raise a series
// to the greatest running total observed on it, and may never raise it past
// that. Where it can be absorbed it is absorbed; it never becomes an addend.
func TestCounterAccumulator_AnUnplaceablePointIsAFloorAndNeverAnAddend(t *testing.T) {
	const series = "model=a"
	for _, tc := range []struct {
		name   string
		points []counterPoint
		want   int64
		reason string
	}{
		{
			name:   "an unplaced point before a run does not add to it",
			points: []counterPoint{cumulative(100, at(10)), cumulative(120, at(20)).from(at(11))},
			want:   120,
			reason: "the capture holds a series that reached 120; the unplaceable 100 is a running " +
				"total of it, not 100 more tokens",
		},
		{
			name:   "an unplaced point after a run does not add to it",
			points: []counterPoint{cumulative(100, at(10)).from(at(1)), cumulative(120, at(20))},
			want:   120,
			reason: "the same series seen twice, one flush of which omitted its start time",
		},
		{
			name: "a final flush that omitted its start time is not a second session",
			points: []counterPoint{
				cumulative(800000, at(10)).from(at(1)),
				cumulative(1200000, at(20)).from(at(1)),
				cumulative(1200050, at(30)),
			},
			want: 1200050,
			reason: "one session, one run, and a last flush with no start: 2400050 is a session " +
				"reported at twice its size",
		},
		{
			name: "an unplaced total above the placed runs raises the floor to it",
			points: []counterPoint{
				cumulative(100, at(10)).from(at(1)),
				cumulative(20, at(20)).from(at(11)),
				cumulative(500, at(30)),
			},
			want: 500,
			reason: "some run of this series reached 500, so the series holds at least 500 — and 620 " +
				"is mass no point in the export ever reported",
		},
		{
			name: "an unplaced total below the placed runs is absorbed",
			points: []counterPoint{
				cumulative(100, at(10)).from(at(1)),
				cumulative(20, at(20)).from(at(11)),
				cumulative(50, at(30)),
			},
			want:   120,
			reason: "the two runs already hold more than the unplaced point observed",
		},
		{
			name:   "with nothing placed, the greatest running total observed stands",
			points: []counterPoint{cumulative(100, at(10)), cumulative(120, at(20))},
			want:   120,
			reason: "no point says when its run began, so there is no restart to act on and the " +
				"capture merges as it did before runs existed",
		},
		{
			name:   "an unplaced series does not lose the total it reached",
			points: []counterPoint{cumulative(100, at(10)), cumulative(120, at(20)), cumulative(90, at(30))},
			want:   120,
			reason: "the series was seen at 120; a later point reporting less is either a restart " +
				"nothing declared or an exporter contradicting itself, and neither unsees the 120",
		},
		{
			name:   "one unplaced point is the total it reports",
			points: []counterPoint{cumulative(100, at(10))},
			want:   100,
			reason: "a running total is a total, whether or not it can name its run",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := counterAccumulator{}
			for _, p := range tc.points {
				acc.add(series, p)
			}
			got := acc.reduce()[tokenTypeInput]
			if got != tc.want {
				t.Errorf("total = %d, want %d — %s", got, tc.want, tc.reason)
			}
			// The invariant itself, asserted over the same points rather than
			// left implied by the number above: whatever the rule, no series
			// may exceed the greatest running total any of its points reported.
			var observed int64
			for _, p := range tc.points {
				if p.value > observed {
					observed = p.value
				}
			}
			if got > observed {
				t.Errorf("total = %d, above the greatest running total observed (%d): a point that "+
					"cannot place itself opened a run that was summed", got, observed)
			}
		})
	}
}

// Delta and cumulative are opposite instructions — add, or supersede — and a
// series carrying both is malformed input. No total derived from it is in the
// export: adding the increments to a running total double-counts them, and
// dropping them under-counts. The series is refused and counted, exactly as a
// temporality that reads as neither 1 nor 2 already is, rather than being made
// to produce a number nobody could trust.
func TestCounterAccumulator_AMixedTemporalitySeriesIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order []bool // cumulative flag per point, in file order
	}{
		{"cumulative then delta", []bool{true, false}},
		{"delta then cumulative", []bool{false, true}},
		{"cumulative between deltas", []bool{false, true, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := counterAccumulator{}
			// Values that discriminate: every rule anyone might pick — keep the
			// cumulative, sum the deltas, sum everything — gives a different
			// number, so this case cannot pass by arithmetic coincidence.
			for i, isCumulative := range tc.order {
				p := delta(int64(100*(i+1)), at(int64(i+1)))
				if isCumulative {
					p = cumulative(50, at(int64(i+1)))
				}
				acc.add("model=a", p.from(at(1)))
			}
			// A second series, well-formed, on the same label: refusing one
			// series must not cost the export the rest of its counts.
			acc.add("model=b", delta(7, at(1)).from(at(1)))

			totals := acc.reduce()
			if got := totals[tokenTypeInput]; got != 7 {
				t.Errorf("total = %d, want 7 — the mixed series contributes nothing and the "+
					"well-formed one still counts", got)
			}
		})
	}
}

// cumulative and delta build the points these cases fold in, carrying the two
// instructions a sum declares. from names the run a point belongs to and as
// changes the label it totals under, so each case states only what it is about.
func cumulative(value int64, when optionalNanos) counterPoint {
	return counterPoint{label: tokenTypeInput, value: value, time: when, cumulative: true}
}

func delta(value int64, when optionalNanos) counterPoint {
	return counterPoint{label: tokenTypeInput, value: value, time: when}
}

func (p counterPoint) from(start optionalNanos) counterPoint {
	p.start = start
	return p
}

func (p counterPoint) as(label string) counterPoint {
	p.label = label
	return p
}

// at and never are the two answers a timestamp leaf gives.
func at(nanos int64) optionalNanos { return optionalNanos{nanos: nanos, ok: true} }

var never optionalNanos

// startTimeUnixNano and timeUnixNano are proto3 fixed64 fields, and OTLP
// mandates the proto3 JSON mapping, whose documented deviations do not touch
// default values. An absent field and an explicit 0 are therefore two spellings
// of one message — protojson emits "0" with EmitUnpopulated on and omits the
// field with it off — so no reader may tell them apart. Reading them apart is
// what gave one series two runs and reported a session at twice its size.
func TestOTLP_AZeroNanosecondTimestampIsTheProto3Default(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"the field is absent", ``},
		{"the field is JSON null", `null`},
		{"a bare zero", `0`},
		{"a quoted zero", `"0"`},
		{"a quoted zero with leading zeroes", `"000"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readNanos(json.RawMessage(tc.raw)); got.ok {
				t.Errorf("readNanos(%q) read %d, want no instant — 0 is the field's default, "+
					"which is the same message as the field being absent", tc.raw, got.nanos)
			}
			if _, ok := nanoTime(json.RawMessage(tc.raw)); ok {
				t.Errorf("nanoTime(%q) read an instant, want none — one encoding rule, "+
					"both of this layer's timestamp readers", tc.raw)
			}
		})
	}
	// A timestamp that is not the default still reads, or this test would pass
	// against a reader that refused everything.
	if got := readNanos(json.RawMessage(`"1789332595000000000"`)); !got.ok || got.nanos != 1789332595000000000 {
		t.Errorf("readNanos of a real instant = %+v, want 1789332595000000000", got)
	}
	if _, ok := nanoTime(json.RawMessage(`1789332595000000000`)); !ok {
		t.Error("nanoTime of a real instant read nothing")
	}
}
