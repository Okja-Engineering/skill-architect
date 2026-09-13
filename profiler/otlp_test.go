package profiler

import (
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
