// Cursor lifecycle-hook capture and spool.
//
// The capture layer is deliberately dumb: every hook invocation becomes one
// JSONL line carrying the payload it was handed, so a parser written later
// works from the same files instead of needing the sessions captured again.
// Nothing here interprets a payload into a measurement — no Profile is built,
// no capability is claimed. What is written is what was received, after secret
// redaction, and that is the whole of the claim.
//
// # What is assumed, and what has been observed
//
// Every payload shape this file reads is taken from Cursor's published hooks
// documentation. None of it has been observed against a running Cursor: Cursor
// was not installed on the machine where this was written, and no spool line in
// this repository was produced by Cursor. Concretely, these are assumptions and
// not measurements:
//
//   - that a hook is invoked with one JSON document on stdin;
//   - that the document is an object carrying `hook_event_name`,
//     `cursor_version`, `cwd` and `conversation_id`;
//   - that the event names in CursorHookEvents are the names Cursor uses.
//
// The design is arranged so that being wrong about any of them costs a re-read
// rather than the data: the promoted envelope fields are a convenience, the
// payload keeps its own copy of everything they were taken from, and a payload
// that does not parse at all is still written down. A wrong guess here is
// recoverable from the files it wrote. That is the reason this lands while an
// adapter reporting a *measurement* from these payloads does not.
package profiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SpoolSchemaVersion is embedded in every spool line so a later parser can tell
// which envelope it is reading.
const SpoolSchemaVersion = "skill-architect/spool/v1"

// unknownEventName is what a payload that named no event is recorded as. The
// absence is written down rather than guessed at, for the same reason a metric
// with no source is `unknown` and not zero.
const unknownEventName = "unknown"

// SpoolEvent is the envelope written as one JSONL line per hook invocation.
//
// The envelope fields are promoted out of the payload so a reader can filter a
// large spool without decoding every line. They are a convenience and not the
// record: Raw keeps the payload's own copy of each of them.
type SpoolEvent struct {
	Ts             string `json:"ts"` // capture time, RFC3339 UTC
	Event          string `json:"event"`
	SchemaVersion  string `json:"schema_version"`
	CursorVersion  string `json:"cursor_version,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`

	// Strict marks a line whose content fields were replaced by their sizes.
	// Without it a stripped line is indistinguishable from one that simply
	// carried no prompt, and a reader pooling a spool cannot tell what it has.
	// Absent means not stripped.
	Strict bool `json:"strict,omitempty"`

	Raw json.RawMessage `json:"raw"`
}

// HookSource is where one hook invocation's payload comes from: stdin in
// production, a replayed corpus in tests and offline re-ingestion.
type HookSource interface {
	Read() (json.RawMessage, error)
}

// StdinSource reads one hook invocation's payload from a stream.
type StdinSource struct {
	In io.Reader
}

func (s StdinSource) Read() (json.RawMessage, error) {
	return io.ReadAll(s.In)
}

// SliceSource replays a captured corpus in order, reporting io.EOF when drained.
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

// NormalizeHookPayload wraps one raw hook payload in a SpoolEvent.
//
// Nothing is dropped: an event name this build has never heard of is kept
// verbatim, a field nobody reads is carried through, and input that is not JSON
// at all is stored as a JSON string rather than refused.
func NormalizeHookPayload(payload []byte, now time.Time) (SpoolEvent, error) {
	return normalize(payload, now, false)
}

// NormalizeHookPayloadStrict is NormalizeHookPayload with content fields —
// prompts, tool I/O, commands, message text — replaced by their sizes. It is
// the metadata-only mode for a shared machine, and the line it produces says
// so.
func NormalizeHookPayloadStrict(payload []byte, now time.Time) (SpoolEvent, error) {
	return normalize(payload, now, true)
}

func normalize(payload []byte, now time.Time, strict bool) (SpoolEvent, error) {
	ev := SpoolEvent{
		Ts:            now.UTC().Format(time.RFC3339),
		Event:         unknownEventName,
		SchemaVersion: SpoolSchemaVersion,
		Strict:        strict,
	}

	decoded, isJSON := decodeOneJSONValue(payload)
	if !isJSON {
		// Not one JSON document. Still captured, and captured whole: the bytes
		// go to disk as a JSON string, so whatever was sent can be read back
		// even though nothing here understood it. This is the case the spool
		// exists for, so it is the last case that may drop anything.
		raw, err := json.Marshal(string(payload))
		if err != nil {
			return ev, fmt.Errorf("capture a payload that is not JSON: %w", err)
		}
		ev.Raw = raw
		return ev, nil
	}

	// Only an object has the fields the envelope promotes. A payload of some
	// other shape is captured whole and described as unknown, which is true of
	// it rather than a guess about it.
	if obj, ok := decoded.(map[string]any); ok {
		if name := getString(obj, "hook_event_name"); name != "" {
			ev.Event = name
		}
		ev.CursorVersion = getString(obj, "cursor_version")
		ev.Cwd = getString(obj, "cwd")
		ev.ConversationID = getString(obj, "conversation_id")
	}

	// Redaction runs over every decoded shape, not only objects. Restricting it
	// to objects is how a credential inside a JSON array reaches the spool in
	// the clear.
	clean := redactSecrets(decoded)
	if strict {
		clean = stripContent(clean)
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return ev, fmt.Errorf("re-marshal after redaction: %w", err)
	}
	ev.Raw = raw
	return ev, nil
}

// decodeOneJSONValue decodes exactly one JSON document, keeping every numeric
// literal as it arrived.
//
// UseNumber is the whole point. Decoding into `any` without it sends every
// number through float64, so an integer above 2^53 comes back changed — and a
// nanosecond epoch timestamp is about 1.7e18. A capture that silently rounds
// the number it captured is not one a later parser can recover from, which
// would make the reason this file ships untrue.
//
// Trailing content makes it not-one-document, and the whole input is then kept
// verbatim rather than the first half of it being kept and the rest lost.
func decodeOneJSONValue(payload []byte) (any, bool) {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, false
	}
	if dec.More() {
		return nil, false
	}
	return v, true
}

// contentKeys are the payload fields that carry user or agent text rather than
// metadata. In strict mode their values become {"_stripped_bytes": N}, so a
// later reader still sees that something was there and how large it was.
// Paths, ids, names, counts and statuses are metadata and survive.
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

// Ingest reads one payload from src, normalizes it, and appends it to the daily
// spool file under spoolDir. It reports io.EOF when a replaying source is
// drained; other read errors propagate and write nothing.
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

// errNoCaptureTime is the refusal for an event that cannot be filed. The day
// comes from the capture timestamp, so an event without one has nowhere to go,
// and a file named `.jsonl` is hidden from every reader listing the spool by
// date.
var errNoCaptureTime = errors.New("spool event has no capture timestamp, so there is no daily file to append it to")

// AppendSpool appends one event to <spoolDir>/<YYYY-MM-DD>.jsonl, creating the
// directory and file as needed. Both are owner-only: a spool line can carry
// prompt text and paths, and the file outlives the session.
func AppendSpool(spoolDir string, ev SpoolEvent) error {
	const dayLen = len("2006-01-02")
	if len(ev.Ts) < dayLen {
		return errNoCaptureTime
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(spoolDir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(spoolDir, ev.Ts[:dayLen]+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// DefaultSpoolDir is ~/.skill-architect/spool.
//
// This is the only function in the package that reads the real home directory,
// and it only computes a path — nothing here creates it. Every function that
// writes takes the directory as a parameter, so a caller that has one never
// goes near the home and a test never can.
func DefaultSpoolDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skill-architect", "spool"), nil
}

// getString reads a string field out of a decoded JSON object, answering ""
// for absent and for any other type. An envelope field that was not a string
// is not a field this build can promote, and guessing at it would put a
// rendering of some other value where a payload's own text belongs.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// --- secret redaction ---
//
// A spool file is a plain file on disk that outlives the session. Prompt text
// and file paths stay, because on this tool they are the data; credentials do
// not.

// sensitiveKey reports whether a JSON field name looks like it holds a
// credential. The name is normalized (lowercased, separators dropped) and then
// suffix-matched, so "api_key", "apiKey" and "api-key" all hit — while metric
// names like "context_tokens" and "tokenUsage", which are measurements, survive.
func sensitiveKey(key string) bool {
	n := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(key))
	switch n {
	case "auth", "authorization", "credential", "credentials":
		return true
	}
	for _, suffix := range []string{"token", "secret", "password", "passwd", "apikey", "privatekey", "accesskey", "sessionkey"} {
		if strings.HasSuffix(n, suffix) {
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

// redactSecrets walks a decoded JSON value, replacing credential-looking fields
// wholesale and credential-shaped substrings in place. The text around a
// substring survives: a command with a bearer token in it is still the command
// that ran, and throwing all of it away would lose the measurement to protect
// the credential.
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
