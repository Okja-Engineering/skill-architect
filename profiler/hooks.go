// Cursor lifecycle-hook capture and spool. The capture layer is deliberately
// dumb and lossless: every hook invocation becomes one JSONL line carrying the
// untouched (post-secret-redaction) stdin payload, so analysis passes can be
// written later without re-capturing.
package profiler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SpoolSchemaVersion is embedded in every spool line so future parsers can
// evolve the envelope without breaking old files.
const SpoolSchemaVersion = "cursor-profiler/spool/v1"

// SpoolEvent is the normalized envelope written as one JSONL line per hook
// invocation. Envelope fields are promoted for cheap indexing; Raw carries the
// full payload so nothing is lost.
type SpoolEvent struct {
	Ts             string          `json:"ts"` // capture time, RFC3339 UTC
	Event          string          `json:"event"`
	SchemaVersion  string          `json:"schema_version"`
	CursorVersion  string          `json:"cursor_version,omitempty"`
	Cwd            string          `json:"cwd,omitempty"`
	ConversationID string          `json:"conversation_id,omitempty"`
	Raw            json.RawMessage `json:"raw"`
}

// HookSource abstracts where a hook payload comes from: real stdin in
// production, a replayed fixture slice in tests and offline analysis.
type HookSource interface {
	Read() (json.RawMessage, error)
}

// StdinSource reads one hook invocation's payload from stdin.
type StdinSource struct {
	In io.Reader
}

func (s StdinSource) Read() (json.RawMessage, error) {
	return io.ReadAll(s.In)
}

// SliceSource replays a captured corpus in order; returns io.EOF when drained.
type SliceSource struct {
	Payloads [][]byte
	next     int
}

func NewSliceSource(payloads [][]byte) *SliceSource {
	return &SliceSource{Payloads: payloads}
}

func (s *SliceSource) Read() (json.RawMessage, error) {
	if s.next >= len(s.Payloads) {
		return nil, io.EOF
	}
	p := s.Payloads[s.next]
	s.next++
	return p, nil
}

// NormalizeHookPayload wraps one raw hook payload in a SpoolEvent. Capture is
// lossless: unanticipated event names and fields are kept, and non-JSON input
// is preserved as a JSON string rather than rejected.
func NormalizeHookPayload(payload []byte, now time.Time) (SpoolEvent, error) {
	return normalize(payload, now, false)
}

// NormalizeHookPayloadStrict additionally strips content fields (prompts, tool
// I/O, commands, message text) to size stubs — the metadata-only mode for
// shared machines (decision Q10).
func NormalizeHookPayloadStrict(payload []byte, now time.Time) (SpoolEvent, error) {
	return normalize(payload, now, true)
}

func normalize(payload []byte, now time.Time, strict bool) (SpoolEvent, error) {
	ev := SpoolEvent{
		Ts:            now.UTC().Format(time.RFC3339),
		Event:         "unknown",
		SchemaVersion: SpoolSchemaVersion,
	}

	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		// Not JSON — still capture, verbatim, as a JSON string.
		raw, _ := json.Marshal(string(payload))
		ev.Raw = raw
		return ev, nil
	}

	if name, ok := obj["hook_event_name"].(string); ok && name != "" {
		ev.Event = name
	}
	ev.CursorVersion = getString(obj, "cursor_version")
	ev.Cwd = getString(obj, "cwd")
	ev.ConversationID = getString(obj, "conversation_id")

	clean := redactSecrets(obj)
	if strict {
		clean = stripContent(clean).(map[string]any)
	}
	redacted, err := json.Marshal(clean)
	if err != nil {
		return ev, fmt.Errorf("re-marshal after redaction: %w", err)
	}
	ev.Raw = redacted
	return ev, nil
}

// contentKeys are payload fields that carry user/agent text rather than
// metadata. In strict mode their values become {"_stripped_bytes": N} so
// downstream analysis still sees that something existed and how big it was.
// file_path, cwd, ids, names, counts, and statuses are metadata and survive.
var contentKeys = map[string]bool{
	"prompt": true, "text": true, "content": true, "tool_input": true,
	"tool_output": true, "output": true, "command": true, "edits": true,
	"result_json": true, "description": true, "summary": true,
	"agent_message": true, "task": true, "attachments": true,
}

func stripContent(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if contentKeys[strings.ToLower(k)] {
				n, _ := json.Marshal(val)
				t[k] = map[string]any{"_stripped_bytes": len(n)}
			} else {
				t[k] = stripContent(val)
			}
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = stripContent(val)
		}
		return t
	default:
		return v
	}
}

// Ingest reads one payload from src, normalizes it, and appends it to the
// daily spool file under spoolDir. Returns io.EOF when a replaying source is
// drained; other read errors propagate.
func Ingest(src HookSource, spoolDir string, now time.Time) (SpoolEvent, error) {
	return IngestMode(src, spoolDir, now, false)
}

// IngestMode is Ingest with the strict content-stripping knob.
func IngestMode(src HookSource, spoolDir string, now time.Time, strict bool) (SpoolEvent, error) {
	payload, err := src.Read()
	if err != nil {
		return SpoolEvent{}, err
	}
	ev, err := normalize(payload, now, strict)
	if err != nil {
		return SpoolEvent{}, err
	}
	return ev, AppendSpool(spoolDir, ev)
}

// AppendSpool appends one event to <spoolDir>/<YYYY-MM-DD>.jsonl, creating the
// directory and file as needed. Spool files are owner-only: they can carry
// prompt text and paths.
func AppendSpool(spoolDir string, ev SpoolEvent) error {
	if err := os.MkdirAll(spoolDir, 0700); err != nil {
		return err
	}
	day := ev.Ts
	if len(day) >= 10 {
		day = day[:10]
	}
	f, err := os.OpenFile(filepath.Join(spoolDir, day+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// DefaultSpoolDir is ~/.cursor-profiler/spool.
func DefaultSpoolDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor-profiler", "spool"), nil
}

// --- secret redaction ---

// sensitiveKey reports whether a JSON field name looks like it holds a
// credential. Normalized (lowercased, separators stripped) then suffix-matched
// so "api_key", "apiKey", and "api-key" all hit — while metric names like
// "context_tokens" or "tokenUsage" (measurements, not credentials) survive.
func sensitiveKey(key string) bool {
	n := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(key))
	switch n {
	case "auth", "authorization", "credential", "credentials":
		return true
	}
	for _, suf := range []string{"token", "secret", "password", "passwd", "apikey", "privatekey", "accesskey", "sessionkey"} {
		if strings.HasSuffix(n, suf) {
			return true
		}
	}
	return false
}

var secretValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-]+`),
	regexp.MustCompile(`sk-[a-zA-Z0-9_\-]{16,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{20,}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

const redacted = "[REDACTED]"

// redactSecrets walks a decoded JSON value, replacing credential-looking
// fields and secret-shaped strings. Prompt text, file paths, and all other
// content pass through — on this personal tool they are the learning data.
func redactSecrets(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if sensitiveKey(k) {
				t[k] = redacted
			} else {
				t[k] = redactSecrets(val)
			}
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = redactSecrets(val)
		}
		return t
	case string:
		for _, re := range secretValuePatterns {
			t = re.ReplaceAllString(t, redacted)
		}
		return t
	default:
		return v
	}
}
