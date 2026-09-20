package profiler

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// S1′ capture+spool acceptance tests — the capture layer is dumb and lossless:
// one JSONL line per hook invocation, secrets redacted before disk, content kept.

var fixedNow = time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)

// --- Normalization ---

func TestNormalizeHookPayload_KnownEvent(t *testing.T) {
	payload := `{
		"hook_event_name": "beforeReadFile",
		"conversation_id": "conv-123",
		"generation_id": "gen-456",
		"cursor_version": "1.7.0",
		"cwd": "/work/repo",
		"file_path": "/work/repo/.cursor/skills/commit/SKILL.md",
		"model": "gpt-5"
	}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Event != "beforeReadFile" {
		t.Errorf("event = %q, want beforeReadFile", ev.Event)
	}
	if ev.ConversationID != "conv-123" {
		t.Errorf("conversation_id = %q, want conv-123", ev.ConversationID)
	}
	if ev.CursorVersion != "1.7.0" {
		t.Errorf("cursor_version = %q, want 1.7.0", ev.CursorVersion)
	}
	if ev.Cwd != "/work/repo" {
		t.Errorf("cwd = %q, want /work/repo", ev.Cwd)
	}
	if ev.SchemaVersion != SpoolSchemaVersion {
		t.Errorf("schema_version = %q, want %q", ev.SchemaVersion, SpoolSchemaVersion)
	}
	if ev.Ts != "2026-09-11T15:30:00Z" {
		t.Errorf("ts = %q, want 2026-09-11T15:30:00Z", ev.Ts)
	}
	// Raw keeps every field, including ones not promoted to the envelope.
	var raw map[string]any
	if err := json.Unmarshal(ev.Raw, &raw); err != nil {
		t.Fatalf("raw is not valid JSON: %v", err)
	}
	if raw["generation_id"] != "gen-456" || raw["file_path"] != "/work/repo/.cursor/skills/commit/SKILL.md" {
		t.Errorf("raw lost fields: %s", string(ev.Raw))
	}
}

func TestNormalizeHookPayload_UnknownEventNameCaptured(t *testing.T) {
	// An event name not in the 21-event table is captured verbatim, not rejected.
	payload := `{"hook_event_name": "someFutureEvent", "conversation_id": "c1"}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Event != "someFutureEvent" {
		t.Errorf("event = %q, want someFutureEvent", ev.Event)
	}
}

func TestNormalizeHookPayload_MissingEventName(t *testing.T) {
	payload := `{"conversation_id": "c1"}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Event != "unknown" {
		t.Errorf("event = %q, want unknown", ev.Event)
	}
}

func TestNormalizeHookPayload_NonJSONCaptured(t *testing.T) {
	// Lossless: garbage stdin is still spooled, wrapped as a JSON string.
	ev, err := NormalizeHookPayload([]byte("not json at all"), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Event != "unknown" {
		t.Errorf("event = %q, want unknown", ev.Event)
	}
	var raw string
	if err := json.Unmarshal(ev.Raw, &raw); err != nil {
		t.Fatalf("raw should be a JSON string for non-JSON input: %v", err)
	}
	if raw != "not json at all" {
		t.Errorf("raw = %q, want original bytes preserved", raw)
	}
}

func TestNormalizeHookPayload_ExtraFieldsPreserved(t *testing.T) {
	// Fields nobody anticipated must survive — that is what "lossless" means.
	payload := `{"hook_event_name": "preToolUse", "brand_new_field": {"nested": [1,2,3]}}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(ev.Raw, &raw)
	nested, ok := raw["brand_new_field"].(map[string]any)
	if !ok || nested["nested"] == nil {
		t.Errorf("raw dropped unanticipated field: %s", string(ev.Raw))
	}
}

// --- Redaction ---

func TestRedactSecrets_SensitiveKeys(t *testing.T) {
	payload := `{
		"hook_event_name": "beforeSubmitPrompt",
		"prompt": "refactor the auth module please",
		"api_key": "sk-abcdef0123456789abcdef",
		"nested": {"access_token": "tok_live_999", "note": "keep me"},
		"list": [{"password": "hunter2"}]
	}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	s := string(ev.Raw)
	for _, secret := range []string{"sk-abcdef0123456789abcdef", "tok_live_999", "hunter2"} {
		if strings.Contains(s, secret) {
			t.Errorf("secret %q survived redaction in %s", secret, s)
		}
	}
	// Content is kept by default: prompt text is the learning data.
	if !strings.Contains(s, "refactor the auth module") {
		t.Errorf("prompt text was stripped: %s", s)
	}
}

func TestRedactSecrets_BearerAndKeyPatternsInValues(t *testing.T) {
	payload := `{
		"hook_event_name": "postToolUse",
		"tool_output": "curl -H 'Authorization: Bearer eyJhbGciOiJ9abc' https://api.example.com"
	}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ev.Raw), "eyJhbGciOiJ9abc") {
		t.Errorf("bearer token survived in value: %s", string(ev.Raw))
	}
}

func TestRedactSecrets_MetricFieldsSurvive(t *testing.T) {
	// "token" inside a measurement name is a metric, not a credential.
	payload := `{"hook_event_name": "preCompact", "context_tokens": 48200, "token_usage": {"input_tokens": 100}}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	s := string(ev.Raw)
	if !strings.Contains(s, "48200") || !strings.Contains(s, "input_tokens") {
		t.Errorf("metric fields were redacted: %s", s)
	}
}

func TestRedactSecrets_FilePathsKept(t *testing.T) {
	payload := `{"hook_event_name": "beforeReadFile", "file_path": "/home/u/secret-project/main.go"}`
	ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ev.Raw), "/home/u/secret-project/main.go") {
		t.Errorf("file_path was stripped: %s", string(ev.Raw))
	}
}

// --- Spool writer ---

func TestAppendSpool_DailyFile(t *testing.T) {
	dir := t.TempDir()
	day1 := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)

	ev1, _ := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionStart","conversation_id":"c1"}`), day1)
	ev2, _ := NormalizeHookPayload([]byte(`{"hook_event_name":"stop","conversation_id":"c1"}`), day1)
	ev3, _ := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionStart","conversation_id":"c2"}`), day2)

	for _, ev := range []SpoolEvent{ev1, ev2, ev3} {
		if err := AppendSpool(dir, ev); err != nil {
			t.Fatal(err)
		}
	}

	f1 := filepath.Join(dir, "2026-09-11.jsonl")
	f2 := filepath.Join(dir, "2026-09-12.jsonl")

	b1, err := os.ReadFile(f1)
	if err != nil {
		t.Fatalf("expected daily file %s: %v", f1, err)
	}
	if lines := strings.Count(strings.TrimSpace(string(b1)), "\n") + 1; lines != 2 {
		t.Errorf("day-1 file has %d lines, want 2", lines)
	}
	b2, err := os.ReadFile(f2)
	if err != nil {
		t.Fatalf("expected daily file %s: %v", f2, err)
	}
	if lines := strings.Count(strings.TrimSpace(string(b2)), "\n") + 1; lines != 1 {
		t.Errorf("day-2 file has %d lines, want 1", lines)
	}
}

func TestAppendSpool_LineRoundTrips(t *testing.T) {
	dir := t.TempDir()
	ev, _ := NormalizeHookPayload([]byte(`{"hook_event_name":"stop","conversation_id":"c9","prompt":"what did I ship today"}`), fixedNow)
	if err := AppendSpool(dir, ev); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "2026-09-11.jsonl"))
	var back SpoolEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &back); err != nil {
		t.Fatal(err)
	}
	if back.ConversationID != "c9" || back.Event != "stop" {
		t.Errorf("round-trip lost envelope fields: %+v", back)
	}
}

// --- HookSource + Ingest ---

type errSource struct{}

func (errSource) Read() (json.RawMessage, error) { return nil, errors.New("boom") }

func TestIngest_FixtureReplay(t *testing.T) {
	dir := t.TempDir()
	src := NewSliceSource([][]byte{
		[]byte(`{"hook_event_name":"sessionStart","conversation_id":"c1"}`),
		[]byte(`{"hook_event_name":"beforeSubmitPrompt","conversation_id":"c1","prompt":"hi"}`),
	})

	n := 0
	for {
		ev, err := Ingest(src, dir, fixedNow)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		n++
		_ = ev
	}
	if n != 2 {
		t.Fatalf("ingested %d events, want 2", n)
	}
	data, err := os.ReadFile(filepath.Join(dir, "2026-09-11.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(data)), "\n") + 1; lines != 2 {
		t.Errorf("spool has %d lines, want 2", lines)
	}
}

func TestIngest_PropagatesReadError(t *testing.T) {
	_, err := Ingest(errSource{}, t.TempDir(), fixedNow)
	if err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestStdinSource_ReadsAll(t *testing.T) {
	src := StdinSource{In: strings.NewReader(`{"hook_event_name":"stop"}`)}
	raw, err := src.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "stop") {
		t.Errorf("raw = %s", string(raw))
	}
}
