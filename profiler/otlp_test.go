package profiler

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
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
	if want := (TokenCounts{Input: 1523}); *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v", *profile.Tokens.Value, want)
	}
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
	if want := (TokenCounts{Input: 300}); *profile.Tokens.Value != want {
		t.Errorf("tokens = %+v, want %+v — both objects are batches of the same capture", *profile.Tokens.Value, want)
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

// The accumulator's totals are int64 and the profile's counts are int. Nothing
// a real session costs comes near either limit, but a malformed export can, and
// a wrapped total serialises as a negative number of tokens — an answer that is
// wrong in a direction no reader would question. There is no fixture for this:
// a 2^63 token count in a reviewed file teaches nobody anything.
func TestTokenAccumulator_TotalsSaturateRatherThanWrap(t *testing.T) {
	acc := tokenAccumulator{}
	// Two series of the same token type, summed at reduce.
	acc.add("model=a", tokenTypeInput, math.MaxInt64, 1, true, false)
	acc.add("model=b", tokenTypeInput, 1000, 1, true, false)
	// One series accumulating deltas past the limit.
	acc.add("model=c", tokenTypeOutput, math.MaxInt64, 1, true, false)
	acc.add("model=c", tokenTypeOutput, math.MaxInt64, 2, true, false)

	totals := acc.reduce()
	if got := totals[tokenTypeInput]; got != math.MaxInt64 {
		t.Errorf("input total = %d, want %d — summing series must saturate", got, int64(math.MaxInt64))
	}
	if got := totals[tokenTypeOutput]; got != math.MaxInt64 {
		t.Errorf("output total = %d, want %d — accumulating deltas must saturate", got, int64(math.MaxInt64))
	}
	if got := clampToInt(math.MaxInt64); got != math.MaxInt {
		t.Errorf("clampToInt(MaxInt64) = %d, want %d", got, math.MaxInt)
	}
	if got := clampToInt(math.MinInt64); got != math.MinInt {
		t.Errorf("clampToInt(MinInt64) = %d, want %d", got, math.MinInt)
	}
	if got := clampToInt(1523); got != 1523 {
		t.Errorf("clampToInt(1523) = %d, want 1523 — a count in range is carried unchanged", got)
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

// OTel defines a time series by its full attribute set, so the series key must
// be injective over that set: two attribute sets that differ anywhere are two
// series and must not merge, and two that differ only in how the exporter
// ordered or spelled the same values are one series and must not split.
//
// The key is internal and never shown to anyone, so these cases pin identity
// rather than any particular encoding.
func TestOTLP_SeriesKeyIsTheAttributeSetNotItsText(t *testing.T) {
	distinct := []struct {
		name string
		a, b string
	}{
		{"a value that was not read is not the empty string",
			`[{"key":"m","value":{}}]`,
			`[{"key":"m","value":{"stringValue":""}}]`},
		{"two different array values",
			`[{"key":"tool.names","value":{"arrayValue":{"values":[{"stringValue":"Read"}]}}}]`,
			`[{"key":"tool.names","value":{"arrayValue":{"values":[{"stringValue":"Bash"}]}}}]`},
		{"two different kvlist values",
			`[{"key":"m","value":{"kvlistValue":{"values":[{"key":"a","value":{"stringValue":"1"}}]}}}]`,
			`[{"key":"m","value":{"kvlistValue":{"values":[{"key":"a","value":{"stringValue":"2"}}]}}}]`},
		{"two different byte values",
			`[{"key":"m","value":{"bytesValue":"YQ=="}}]`,
			`[{"key":"m","value":{"bytesValue":"Yg=="}}]`},
		{"an array value and a kvlist value",
			`[{"key":"m","value":{"arrayValue":{"values":[]}}}]`,
			`[{"key":"m","value":{"kvlistValue":{"values":[]}}}]`},
		{"a string and the integer that prints the same",
			`[{"key":"m","value":{"stringValue":"5"}}]`,
			`[{"key":"m","value":{"intValue":5}}]`},
		{"a string and the boolean that prints the same",
			`[{"key":"m","value":{"stringValue":"true"}}]`,
			`[{"key":"m","value":{"boolValue":true}}]`},
		// An attribute key or value is arbitrary text and may carry whatever
		// byte the key encoding separates fields with. A key those bytes can
		// forge is a key two unrelated exports share.
		{"a value carrying the key/value separator, against the split it forges",
			`[{"key":"m","value":{"stringValue":"a\u0001b"}}]`,
			`[{"key":"m\u0001a","value":{"stringValue":"b"}}]`},
		{"a value carrying the pair separator, against the pair it forges",
			`[{"key":"m","value":{"stringValue":"a\u0000z\u0001v"}}]`,
			`[{"key":"m","value":{"stringValue":"a"}},{"key":"z","value":{"stringValue":"v"}}]`},
	}
	for _, tc := range distinct {
		t.Run("distinct/"+tc.name, func(t *testing.T) {
			if a, b := attrs(t, tc.a).seriesKey(), attrs(t, tc.b).seriesKey(); a == b {
				t.Errorf("%s and %s share the series key %q — two series would merge into one",
					tc.a, tc.b, a)
			}
		})
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
			if a, b := attrs(t, tc.a).seriesKey(), attrs(t, tc.b).seriesKey(); a != b {
				t.Errorf("%s keys as %q and %s as %q — one series would split in two",
					tc.a, a, tc.b, b)
			}
		})
	}
}
