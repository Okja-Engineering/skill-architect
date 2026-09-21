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

	"github.com/Okja-Engineering/skill-architect/profiler/internal/homesafe"
)

// fixedNow is the capture time every case here is stamped with, so the spool
// file a case lands in is a property of the test and not of the clock.
var fixedNow = time.Date(2026, 9, 20, 14, 30, 0, 0, time.UTC)

// --- The spool envelope ---
//
// What a spool line says about itself is the whole of what this layer claims.
// Every assertion below is about the envelope and the payload: none of them is
// a claim that a running Cursor produces payloads of this shape, because
// nothing here has seen one. The fixtures are the documented shapes, and they
// stand for "a payload with these fields", not for "the payload Cursor sends".

// TestSpoolSchema_NamesThisProduct pins the rename.
//
// The draft's constant was spelled for the sibling research repository rather
// than for this product, and it is embedded in every line written. Renaming it
// costs nothing until the first file is written with it and a migration after
// that, so it is pinned here rather than left to a reviewer's eye. The old
// spelling is deliberately not repeated anywhere this check reads — it searches
// these files for it.
func TestSpoolSchema_NamesThisProduct(t *testing.T) {
	const want = "skill-architect/spool/v1"
	if SpoolSchemaVersion != want {
		t.Errorf("SpoolSchemaVersion = %q, want %q", SpoolSchemaVersion, want)
	}

	// The constant is not the only place a namespace can hide: the spool
	// directory and the environment variable carried the same borrowed name.
	// Every file that names the spool is asked, so a fourth one cannot arrive
	// quietly.
	//
	// The needles are assembled rather than written out, because this file is
	// one of the files searched — spelling them here would make the check fail
	// on itself, which is a check that can only be satisfied by deleting it.
	const sibling = "cursor" // the borrowed half; the product half is appended below
	needles := []string{sibling + "-profiler", sibling + "_profiler", strings.ToUpper(sibling + "_profiler")}

	for _, name := range filesNamingTheSpool(t) {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, borrowed := range needles {
			if strings.Contains(string(body), borrowed) {
				t.Errorf("%s still carries %q, a namespace from a different repository", name, borrowed)
			}
		}
	}
}

// filesNamingTheSpool is everything the check above reads: this package's Go
// files and everything under queries/.
//
// The queries are the half that was missing, and they are the half that would
// have slipped past. A check scoped to `*.go` reported clean over a `queries/`
// directory whose every SQL file spelled the borrowed namespace into a default
// path, and the DuckDB queries are the surface a user is most likely to copy a
// path out of. Both groups are required to be non-empty: reading nothing is how
// a search for an absent string passes without running.
func filesNamingTheSpool(t *testing.T) []string {
	t.Helper()
	goFiles := filesUnder(t, ".", func(name string) bool { return strings.HasSuffix(name, ".go") })
	if len(goFiles) == 0 {
		t.Fatal("no Go file was found in this package, so this check reads nothing")
	}
	queries := filesUnder(t, "queries", func(string) bool { return true })
	if len(queries) == 0 {
		t.Fatal("no file was found under queries/, so the half of this check that covers the SQL reads nothing")
	}
	return append(goFiles, queries...)
}

// filesUnder is the files directly in dir whose names keep.
func filesUnder(t *testing.T, dir string, keep func(name string) bool) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && keep(e.Name()) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

// TestNormalize_PromotesTheEnvelopeFieldsItDocuments covers the indexing
// fields. They are promoted out of the payload for cheap filtering; the payload
// keeps its own copy, so a wrong promotion costs a re-read and not the data.
func TestNormalize_PromotesTheEnvelopeFieldsItDocuments(t *testing.T) {
	ev, err := NormalizeHookPayload([]byte(`{
		"hook_event_name": "beforeSubmitPrompt",
		"cursor_version": "1.7.3",
		"cwd": "/work/repo",
		"conversation_id": "conv-1",
		"prompt": "write the test first"
	}`), fixedNow)
	if err != nil {
		t.Fatalf("NormalizeHookPayload: %v", err)
	}

	for _, c := range []struct{ field, got, want string }{
		{"ts", ev.Ts, "2026-09-20T14:30:00Z"},
		{"event", ev.Event, "beforeSubmitPrompt"},
		{"schema_version", ev.SchemaVersion, SpoolSchemaVersion},
		{"cursor_version", ev.CursorVersion, "1.7.3"},
		{"cwd", ev.Cwd, "/work/repo"},
		{"conversation_id", ev.ConversationID, "conv-1"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
	if ev.Strict {
		t.Error("the line says it was stripped, and it was not")
	}
	if got := rawField(t, ev, "prompt"); got != "write the test first" {
		t.Errorf("raw.prompt = %v, want the payload's own copy", got)
	}
}

// TestNormalize_KeepsWhatItDoesNotUnderstand is the property the whole capture
// layer rests on. An event name nobody here has heard of, and a field nobody
// here reads, reach the spool unchanged — so a parser written later works from
// the same files rather than needing the sessions captured again.
func TestNormalize_KeepsWhatItDoesNotUnderstand(t *testing.T) {
	t.Run("an event name this build does not know", func(t *testing.T) {
		ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"someEventInventedLater"}`), fixedNow)
		if err != nil {
			t.Fatalf("NormalizeHookPayload: %v", err)
		}
		if ev.Event != "someEventInventedLater" {
			t.Errorf("event = %q, want it kept verbatim rather than mapped to a known name", ev.Event)
		}
	})

	t.Run("fields nobody here reads", func(t *testing.T) {
		ev, err := NormalizeHookPayload([]byte(`{
			"hook_event_name":"postToolUse",
			"a_field_from_a_later_cursor": {"nested": [1, "two", true, null]}
		}`), fixedNow)
		if err != nil {
			t.Fatalf("NormalizeHookPayload: %v", err)
		}
		if got := rawField(t, ev, "a_field_from_a_later_cursor"); got == nil {
			t.Errorf("the unrecognised field was dropped; raw = %s", ev.Raw)
		}
	})

	t.Run("no event name at all", func(t *testing.T) {
		ev, err := NormalizeHookPayload([]byte(`{"cwd":"/work"}`), fixedNow)
		if err != nil {
			t.Fatalf("NormalizeHookPayload: %v", err)
		}
		if ev.Event != "unknown" {
			t.Errorf("event = %q, want %q — the absence is recorded, not guessed at", ev.Event, "unknown")
		}
	})
}

// TestNormalize_KeepsLargeIntegersExactly is the defect that would have made
// "lossless" false.
//
// The draft decoded into map[string]any and re-encoded. Every number goes
// through float64 that way, so an integer above 2^53 comes back changed — and
// a nanosecond epoch timestamp is about 1.7e18, which is exactly the kind of
// field a hook payload carries and exactly the kind this repo already
// pairs runs on. A capture that silently rounds the number it captured is not
// a capture a later parser can recover from.
func TestNormalize_KeepsLargeIntegersExactly(t *testing.T) {
	const nanos = "1758378600123456789"
	ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionEnd","start_time_unix_nano":`+nanos+`}`), fixedNow)
	if err != nil {
		t.Fatalf("NormalizeHookPayload: %v", err)
	}
	if !strings.Contains(string(ev.Raw), nanos) {
		t.Errorf("raw = %s\nwant it to carry %s unchanged", ev.Raw, nanos)
	}
}

// TestNormalize_CapturesAPayloadThatIsNotJSON keeps a malformed or unexpected
// stdin out of the "drop it" path. Cursor sending something this build cannot
// parse is the case the spool exists for; refusing it would lose the one
// recording of the shape nobody anticipated.
func TestNormalize_CapturesAPayloadThatIsNotJSON(t *testing.T) {
	for _, payload := range []string{"not json at all", "", `{"unterminated":`, `{"a":1} {"b":2}`} {
		ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
		if err != nil {
			t.Fatalf("NormalizeHookPayload(%q): %v", payload, err)
		}
		if ev.Event != "unknown" {
			t.Errorf("event = %q for %q, want %q", ev.Event, payload, "unknown")
		}
		var back string
		if err := json.Unmarshal(ev.Raw, &back); err != nil {
			t.Fatalf("raw for %q is not a JSON string: %v (%s)", payload, err, ev.Raw)
		}
		if back != payload {
			t.Errorf("raw for %q read back as %q; the bytes were not kept", payload, back)
		}
	}
}

// TestNormalize_RedactsAPayloadThatIsNotAnObject closes a hole in the draft.
//
// The draft decoded into map[string]any, so anything that was not an object —
// a JSON array, which is a shape a hook could legitimately send — failed to
// decode and fell down the "not JSON" path, where it was written to disk with
// no redaction at all. A credential inside an array reached the spool in the
// clear.
func TestNormalize_RedactsAPayloadThatIsNotAnObject(t *testing.T) {
	ev, err := NormalizeHookPayload([]byte(`[{"api_key":"sk-thisisaverylongsecretvalue"},"bearer abc123def456"]`), fixedNow)
	if err != nil {
		t.Fatalf("NormalizeHookPayload: %v", err)
	}
	if strings.Contains(string(ev.Raw), "sk-thisisaverylongsecretvalue") {
		t.Errorf("a credential in a JSON array reached the spool in the clear: %s", ev.Raw)
	}
	if strings.Contains(string(ev.Raw), "abc123def456") {
		t.Errorf("a bearer token in a JSON array reached the spool in the clear: %s", ev.Raw)
	}
	if ev.Event != "unknown" {
		t.Errorf("event = %q, want %q — a payload with no object has no event name to read", ev.Event, "unknown")
	}
}

// --- Redaction ---
//
// The spool can carry prompt text and file paths, and on this tool those are
// the data. Credentials are not: a spool file is a plain file on disk that
// outlives the session.

func TestRedact_CredentialFieldsAndCredentialShapedValues(t *testing.T) {
	ev, err := NormalizeHookPayload([]byte(`{
		"hook_event_name": "beforeShellExecution",
		"api_key":     "sk-live-0000000000000000",
		"apiKey":      "sk-live-1111111111111111",
		"api-key":     "sk-live-2222222222222222",
		"authorization": "Bearer abcdefghijklmnop",
		"session_key":  "zzz",
		"credentials":  {"user": "me"},
		"nested":       {"access_token": "aaa"},
		"list":         [{"password": "bbb"}],
		"command":      "curl -H 'Authorization: Bearer abcdefghijklmnop' https://x",
		"note":         "the key is sk-abcdefghijklmnopqrstuvwxyz and ghp_aaaaaaaaaaaaaaaaaaaaaa",
		"aws":          "AKIAIOSFODNN7EXAMPLE",
		"pem":          "-----BEGIN RSA PRIVATE KEY-----\nMIIE",
		"context_tokens": 4096,
		"tokenUsage":   {"input": 10},
		"file_path":    "/Users/me/work/skills/my-skill/SKILL.md",
		"tool_name":    "Read"
	}`), fixedNow)
	if err != nil {
		t.Fatalf("NormalizeHookPayload: %v", err)
	}
	raw := string(ev.Raw)

	t.Run("credentials do not reach the spool", func(t *testing.T) {
		for _, secret := range []string{
			"sk-live-0000000000000000", "sk-live-1111111111111111", "sk-live-2222222222222222",
			"abcdefghijklmnop", "zzz", "aaa", "bbb",
			"sk-abcdefghijklmnopqrstuvwxyz", "ghp_aaaaaaaaaaaaaaaaaaaaaa",
			"AKIAIOSFODNN7EXAMPLE", "BEGIN RSA PRIVATE KEY",
		} {
			if strings.Contains(raw, secret) {
				t.Errorf("%q reached the spool\n%s", secret, raw)
			}
		}
	})

	t.Run("measurements and paths survive, because they are the data", func(t *testing.T) {
		for _, kept := range []string{"4096", `"input"`, "/Users/me/work/skills/my-skill/SKILL.md", "Read"} {
			if !strings.Contains(raw, kept) {
				t.Errorf("%q was redacted; it is a measurement or a path, not a credential\n%s", kept, raw)
			}
		}
	})

	t.Run("a credential inside a string is replaced, not the whole string", func(t *testing.T) {
		if !strings.Contains(raw, "curl -H") || !strings.Contains(raw, "https://x") {
			t.Errorf("the command around the credential was thrown away too\n%s", raw)
		}
	})
}

// --- Strict mode ---
//
// The metadata-only mode for a shared machine. The draft shipped it with no
// test at all: `NormalizeHookPayloadStrict` and `stripContent` were both at 0%
// coverage, which for a privacy control is the same defect as a capability
// claim nothing implements.

func TestNormalizeStrict_ReplacesContentWithItsSizeAndKeepsMetadata(t *testing.T) {
	payload := []byte(`{
		"hook_event_name": "preToolUse",
		"conversation_id": "conv-9",
		"tool_name":  "Edit",
		"file_path":  "/work/repo/main.go",
		"tool_input": {"old": "alpha", "new": "beta"},
		"prompt":     "a sentence the machine's owner would rather not store",
		"steps":      [{"text": "also content"}]
	}`)

	ev, err := NormalizeHookPayloadStrict(payload, fixedNow)
	if err != nil {
		t.Fatalf("NormalizeHookPayloadStrict: %v", err)
	}
	raw := string(ev.Raw)

	t.Run("the line says it was stripped", func(t *testing.T) {
		if !ev.Strict {
			t.Error("a stripped line is indistinguishable from a whole one; a reader pooling a spool cannot tell what it has")
		}
	})

	t.Run("content is gone and its size is kept", func(t *testing.T) {
		for _, content := range []string{"a sentence the machine's owner", "alpha", "also content"} {
			if strings.Contains(raw, content) {
				t.Errorf("%q survived strict mode\n%s", content, raw)
			}
		}
		if !strings.Contains(raw, "_stripped_bytes") {
			t.Errorf("nothing records that something was there\n%s", raw)
		}
	})

	t.Run("metadata survives, or the mode records nothing worth having", func(t *testing.T) {
		for _, kept := range []string{"preToolUse", "conv-9", "Edit", "/work/repo/main.go"} {
			if !strings.Contains(raw, kept) {
				t.Errorf("%q was stripped; it is metadata\n%s", kept, raw)
			}
		}
	})

	t.Run("the envelope is unaffected", func(t *testing.T) {
		if ev.Event != "preToolUse" || ev.ConversationID != "conv-9" {
			t.Errorf("envelope = %+v, want the promoted fields intact", ev)
		}
	})
}

func TestNormalize_NonStrictKeepsContent(t *testing.T) {
	ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"preToolUse","prompt":"kept"}`), fixedNow)
	if err != nil {
		t.Fatalf("NormalizeHookPayload: %v", err)
	}
	if !strings.Contains(string(ev.Raw), "kept") {
		t.Errorf("the default mode stripped content\n%s", ev.Raw)
	}
	if ev.Strict {
		t.Error("the default line claims to be stripped")
	}
}

// --- The spool file ---

func TestAppendSpool_OneDailyFileAppendedTo(t *testing.T) {
	dir := sandboxSpoolDir(t)

	first, err := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionStart"}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	second, err := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionEnd"}`), fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	nextDay, err := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionStart"}`), fixedNow.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	for _, ev := range []SpoolEvent{first, second, nextDay} {
		if err := AppendSpool(dir, ev); err != nil {
			t.Fatalf("AppendSpool: %v", err)
		}
	}

	t.Run("one file per UTC day", func(t *testing.T) {
		names := spoolFileNames(t, dir)
		want := []string{"2026-09-20.jsonl", "2026-09-21.jsonl"}
		if strings.Join(names, ",") != strings.Join(want, ",") {
			t.Errorf("spool files = %v, want %v", names, want)
		}
	})

	t.Run("the second line did not replace the first", func(t *testing.T) {
		lines := spoolLines(t, filepath.Join(dir, "2026-09-20.jsonl"))
		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2 — an append truncated", len(lines))
		}
		if lines[0].Event != "sessionStart" || lines[1].Event != "sessionEnd" {
			t.Errorf("lines = %q, %q, want them in the order they were captured", lines[0].Event, lines[1].Event)
		}
	})

	t.Run("owner-only, because a spool line can carry a prompt", func(t *testing.T) {
		if info, err := os.Stat(dir); err != nil {
			t.Fatalf("stat dir: %v", err)
		} else if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("spool directory mode = %04o, want 0700", perm)
		}
		if info, err := os.Stat(filepath.Join(dir, "2026-09-20.jsonl")); err != nil {
			t.Fatalf("stat file: %v", err)
		} else if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("spool file mode = %04o, want 0600", perm)
		}
	})
}

// TestAppendSpool_RefusesAnEventItCannotFile keeps a line out of a file named
// for no day. The day comes from the capture timestamp, so an event without one
// has nowhere to go — and writing it to `.jsonl` would hide it from every
// reader that lists the spool by date.
func TestAppendSpool_RefusesAnEventItCannotFile(t *testing.T) {
	dir := sandboxSpoolDir(t)

	err := AppendSpool(dir, SpoolEvent{Event: "sessionStart", SchemaVersion: SpoolSchemaVersion})
	if err == nil {
		t.Fatal("an event with no capture timestamp was filed anyway")
	}
	if !strings.Contains(err.Error(), "timestamp") {
		t.Errorf("error = %q, want it to name the missing timestamp", err)
	}
	if names := spoolFileNames(t, dir); len(names) != 0 {
		t.Errorf("the refusal still wrote %v", names)
	}
}

// TestAppendSpool_LineRoundTrips is the losslessness claim, stated as narrowly
// as it is true: every field and every value survives the trip to disk and back.
// Key order does not, and is not claimed to.
func TestAppendSpool_LineRoundTrips(t *testing.T) {
	dir := sandboxSpoolDir(t)

	ev, err := NormalizeHookPayload([]byte(`{
		"hook_event_name": "postToolUse",
		"cursor_version": "1.7.3",
		"cwd": "/work",
		"conversation_id": "conv-2",
		"tool_use_id": "call-1",
		"nested": {"n": 1758378600123456789, "s": "text", "b": true, "null": null, "arr": [1,2,3]}
	}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := AppendSpool(dir, ev); err != nil {
		t.Fatalf("AppendSpool: %v", err)
	}

	back := spoolLines(t, filepath.Join(dir, "2026-09-20.jsonl"))[0]

	if back.Ts != ev.Ts || back.Event != ev.Event || back.SchemaVersion != ev.SchemaVersion ||
		back.CursorVersion != ev.CursorVersion || back.Cwd != ev.Cwd || back.ConversationID != ev.ConversationID {
		t.Errorf("envelope read back as %+v, want %+v", back, ev)
	}
	if !json.Valid(back.Raw) {
		t.Fatalf("raw read back invalid: %s", back.Raw)
	}
	var wantRaw, gotRaw any
	if err := json.Unmarshal(ev.Raw, &wantRaw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := json.Unmarshal(back.Raw, &gotRaw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !jsonEqual(wantRaw, gotRaw) {
		t.Errorf("raw read back as %s, want %s", back.Raw, ev.Raw)
	}
	if !strings.Contains(string(back.Raw), "1758378600123456789") {
		t.Errorf("the large integer did not survive the round trip: %s", back.Raw)
	}
}

// --- Ingest ---

func TestIngest_ReplaysACorpusInOrderAndStopsAtTheEnd(t *testing.T) {
	dir := sandboxSpoolDir(t)
	src := NewSliceSource([][]byte{
		[]byte(`{"hook_event_name":"sessionStart","conversation_id":"c1"}`),
		[]byte(`{"hook_event_name":"preToolUse","conversation_id":"c1"}`),
		[]byte(`not json`),
	})

	// Bounded, not `for {}`. Draining a source is the caller's loop and io.EOF
	// is what ends it, so a regression that stops reporting EOF turns an
	// unbounded loop into a test that never finishes — a hang rather than a
	// failure, which is a verdict nobody gets. The bound makes the same
	// regression a named failure. (Found by mutation M15, which did exactly
	// that.)
	const readsAllowed = 8
	var seen []string
	drained := false
	for i := 0; i < readsAllowed; i++ {
		ev, err := Ingest(src, dir, fixedNow)
		if errors.Is(err, io.EOF) {
			drained = true
			break
		}
		if err != nil {
			t.Fatalf("Ingest: %v", err)
		}
		seen = append(seen, ev.Event)
	}
	if !drained {
		t.Fatalf("a drained source never reported io.EOF: %d reads returned an event, and a caller's loop over it would not end", readsAllowed)
	}

	if want := "sessionStart,preToolUse,unknown"; strings.Join(seen, ",") != want {
		t.Errorf("events = %v, want %s", seen, want)
	}
	if got := len(spoolLines(t, filepath.Join(dir, "2026-09-20.jsonl"))); got != 3 {
		t.Errorf("%d lines on disk, want 3 — the payload nobody understood is the one worth keeping", got)
	}
}

func TestIngest_AReadThatFailedWritesNothing(t *testing.T) {
	dir := sandboxSpoolDir(t)

	_, err := Ingest(errSource{}, dir, fixedNow)
	if !errors.Is(err, errSourceFailed) {
		t.Fatalf("err = %v, want the source's own error", err)
	}
	if names := spoolFileNames(t, dir); len(names) != 0 {
		t.Errorf("a failed read left %v behind; a spool line would stand for a hook invocation that was never read", names)
	}
}

// TestIngest_SaysSoWhenTheSpoolCannotBeWritten keeps a failed capture from
// looking like a captured nothing. The hook is registered as `… ingest || true`
// so that a failure cannot break the Cursor session, which means the exit
// status is discarded — all the more reason the error must exist and travel, so
// the caller that does look (the CLI, a replay) is told.
func TestIngest_SaysSoWhenTheSpoolCannotBeWritten(t *testing.T) {
	sealed := filepath.Join(homesafe.SandboxHome(t), "sealed")
	if err := os.Mkdir(sealed, 0o500); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o700) })
	dir := filepath.Join(sealed, "spool")

	if _, err := os.Stat(dir); err == nil {
		t.Skip("this filesystem created the directory under an unwritable parent; the case is not reachable here")
	}

	t.Run("AppendSpool", func(t *testing.T) {
		ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"stop"}`), fixedNow)
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if err := AppendSpool(dir, ev); err == nil {
			t.Error("AppendSpool reported success against a directory it could not create")
		}
	})

	t.Run("Ingest", func(t *testing.T) {
		src := NewSliceSource([][]byte{[]byte(`{"hook_event_name":"stop"}`)})
		if _, err := Ingest(src, dir, fixedNow); err == nil {
			t.Error("Ingest reported success against a spool it could not write")
		}
	})
}

func TestIngestMode_StrictReachesTheSpool(t *testing.T) {
	dir := sandboxSpoolDir(t)
	payload := []byte(`{"hook_event_name":"beforeSubmitPrompt","prompt":"secret plan","cwd":"/work"}`)

	if _, err := IngestMode(NewSliceSource([][]byte{payload}), dir, fixedNow, true); err != nil {
		t.Fatalf("IngestMode: %v", err)
	}

	line := spoolLines(t, filepath.Join(dir, "2026-09-20.jsonl"))[0]
	if !line.Strict {
		t.Error("the stored line does not say it was stripped")
	}
	if strings.Contains(string(line.Raw), "secret plan") {
		t.Errorf("strict mode did not reach the file that was written: %s", line.Raw)
	}
	if !strings.Contains(string(line.Raw), "/work") {
		t.Errorf("metadata was stripped along with the content: %s", line.Raw)
	}
}

func TestStdinSource_ReadsTheWholePayload(t *testing.T) {
	payload, err := StdinSource{In: strings.NewReader(`{"hook_event_name":"stop"}`)}.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(payload) != `{"hook_event_name":"stop"}` {
		t.Errorf("read %q, want the whole payload", payload)
	}
}

// --- The default spool directory ---

// TestDefaultSpoolDir_IsUnderTheHomeAndNamedForThisProduct is the one function
// in the package that reads the real home, and it only computes a path: nothing
// here creates it. The draft had no test for it at all, and it named the
// sibling repository's directory.
func TestDefaultSpoolDir_IsUnderTheHomeAndNamedForThisProduct(t *testing.T) {
	dir, err := DefaultSpoolDir()
	if err != nil {
		t.Fatalf("DefaultSpoolDir: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}
	if want := filepath.Join(home, ".skill-architect", "spool"); dir != want {
		t.Errorf("DefaultSpoolDir() = %q, want %q", dir, want)
	}

	// Computing the path must not have created it. This is the only assertion
	// in the package that looks at the real home, and it looks only.
	if _, err := os.Stat(dir); err == nil {
		t.Logf("note: %s already exists on this machine; this test did not create it", dir)
	}
}

// --- helpers ---

// sandboxSpoolDir returns a spool directory proved to be outside the real home
// before anything is written into it.
func sandboxSpoolDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(homesafe.SandboxHome(t), ".skill-architect", "spool")
	homesafe.MustBeOutside(t, dir)
	return dir
}

func spoolFileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read spool dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func spoolLines(t *testing.T, path string) []SpoolEvent {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spool file: %v", err)
	}
	var out []SpoolEvent
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var ev SpoolEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("spool line is not a SpoolEvent: %v\n%s", err, line)
		}
		out = append(out, ev)
	}
	if len(out) == 0 {
		t.Fatalf("no line was read out of %s, so this check reads nothing", path)
	}
	return out
}

func rawField(t *testing.T, ev SpoolEvent, key string) any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(ev.Raw, &m); err != nil {
		t.Fatalf("raw is not an object: %v\n%s", err, ev.Raw)
	}
	return m[key]
}

func jsonEqual(a, b any) bool {
	x, errA := json.Marshal(a)
	y, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(x) == string(y)
}

var errSourceFailed = errors.New("the source could not be read")

type errSource struct{}

func (errSource) Read() (json.RawMessage, error) { return nil, errSourceFailed }

// --- The home-directory barrier ---
//
// Everything in this file and its siblings writes into a directory that stands
// for a home directory. The one thing none of them may do is write into the
// real one: `hooks install` creates ~/.cursor/hooks.json, `ingest` creates the
// spool, and `doctor`'s tests seed both inside the home they then ask about, so
// a test that reached the real home would rewrite the machine's Cursor
// configuration. This repo has already shipped exactly that bug.
//
// The barrier itself now lives in internal/homesafe, with its own tests: it was
// duplicated here and in package cmd, `doctor` made a third caller, and three
// copies of a guard are three chances for one of them to be the copy that
// degraded. Every test in this package names a home through
// homesafe.SandboxHome and hands no other directory to code that writes.
