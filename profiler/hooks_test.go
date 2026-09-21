package profiler

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
// spelling is deliberately not repeated anywhere in this package — the check
// below searches these files for it.
func TestSpoolSchema_NamesThisProduct(t *testing.T) {
	const want = "skill-architect/spool/v1"
	if SpoolSchemaVersion != want {
		t.Errorf("SpoolSchemaVersion = %q, want %q", SpoolSchemaVersion, want)
	}

	// The constant is not the only place a namespace can hide: the spool
	// directory and the environment variable carried the same borrowed name.
	// The whole package is asked, so a fourth one cannot arrive quietly.
	//
	// The needles are assembled rather than written out, because this file is
	// one of the files searched — spelling them here would make the check fail
	// on itself, which is a check that can only be satisfied by deleting it.
	const sibling = "cursor" // the borrowed half; the product half is appended below
	needles := []string{sibling + "-profiler", sibling + "_profiler", strings.ToUpper(sibling + "_profiler")}

	for _, name := range goFilesIn(t, ".") {
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

func goFilesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	if len(out) == 0 {
		t.Fatalf("no Go file was found under %s, so this check reads nothing", dir)
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
	sealed := filepath.Join(sandboxHome(t), "sealed")
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
	dir := filepath.Join(sandboxHome(t), ".skill-architect", "spool")
	mustBeOutsideRealHome(t, dir)
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
// Everything in this file and its two siblings writes into a directory that
// stands for a home directory. The one thing none of them may do is write into
// the real one: `hooks install` creates ~/.cursor/hooks.json and `ingest`
// creates the spool, so a test that reached the real home would rewrite the
// machine's Cursor configuration.
//
// This repo has already shipped exactly that bug. A suite here once had a guard
// that *reported* a violation and then ran the install anyway, because the
// assertion helper had no abort path; a reviewer reproduced it deleting files
// in a decoy home. So the barrier has two halves, and each is proved on its own:
//
//   - the decision is made after both paths are resolved, not from the shape of
//     the strings (TestRealHomeContains_DecidesAfterResolution), and
//   - the abort stops the statement after it from running
//     (TestHomeBarrier_StopsTheWriteThatWouldFollow).

// TestPathContains_DecidesAfterResolution pins the decision to where a path
// lands rather than to how it is spelled.
//
// The two cases a spelling test gets wrong are the point of the table. A
// symlink pointing into the protected directory is *outside* by every string
// test and inside by every filesystem test; a sibling whose name begins with
// the same letters is *inside* by strings.HasPrefix and outside in fact. Both
// are reachable by accident — macOS hands t.TempDir() a symlinked path of its
// own — so both are asserted.
//
// Every path here is one this test created, so no case depends on what happens
// to exist beside the real home and no case is skipped. A skipped case would be
// the barrier's own proof passing by not running.
func TestPathContains_DecidesAfterResolution(t *testing.T) {
	root := t.TempDir()
	stands := filepath.Join(root, "alice") // stands for the home
	inside := filepath.Join(stands, ".cursor")
	sibling := filepath.Join(root, "alice-backup") // shares its name as a prefix
	for _, d := range []string{stands, inside, sibling} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	link := filepath.Join(root, "looks-harmless")
	if err := os.Symlink(inside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	for _, tc := range []struct {
		name  string
		child string
		want  bool
	}{
		{"the directory itself", stands, true},
		{"something inside it", inside, true},
		{"a symlink from elsewhere pointing inside it", link, true},
		{"something inside it that does not exist yet", filepath.Join(inside, "hooks.json"), true},
		{"a path climbing back into it", filepath.Join(root, "alice-backup", "..", "alice", ".cursor"), true},
		{"a sibling whose name starts with the same letters", sibling, false},
		{"the parent of both", root, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pathContains(stands, tc.child)
			if err != nil {
				t.Fatalf("pathContains(%q, %q): %v", stands, tc.child, err)
			}
			if got != tc.want {
				t.Errorf("pathContains(%q, %q) = %v, want %v", stands, tc.child, got, tc.want)
			}
		})
	}
}

// TestRealHomeContains_KnowsTheRealHome joins the decision to the directory it
// protects, and gives the reason that a refusal has to print.
//
// Only paths that already exist are asked about, and nothing is created: these
// are the only assertions in the package that look at the real home, and they
// look only.
func TestRealHomeContains_KnowsTheRealHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	for _, tc := range []struct {
		name string
		dir  string
		want bool
	}{
		{"the real home itself", home, true},
		{"the Cursor config inside it", filepath.Join(home, ".cursor"), true},
		{"the spool inside it", filepath.Join(home, ".skill-architect", "spool"), true},
		{"a temp dir", t.TempDir(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, why, err := realHomeContains(tc.dir)
			if err != nil {
				t.Fatalf("realHomeContains(%q): %v", tc.dir, err)
			}
			if got != tc.want {
				t.Errorf("realHomeContains(%q) = %v, want %v (reason %q)", tc.dir, got, tc.want, why)
			}
			if got && why == "" {
				t.Error("a path was called contained with no reason given, so a failure would not say what it found")
			}
			if !got && why != "" {
				t.Errorf("a path outside the home carries a reason %q, which reads as a refusal", why)
			}
		})
	}
}

// TestRealHomeContains_RefusesRatherThanGuessesWhenItCannotResolve pins the
// answer on the side of safety. A path the barrier cannot resolve is not a path
// it has shown to be outside the home, and "I could not tell" must not read as
// "go ahead".
func TestRealHomeContains_RefusesRatherThanGuessesWhenItCannotResolve(t *testing.T) {
	tmp := t.TempDir()

	// A directory the process cannot traverse: resolution of anything beneath
	// it fails with EACCES rather than ENOENT, which is the case that must not
	// be read as "does not exist, therefore fine".
	sealed := filepath.Join(tmp, "sealed")
	if err := os.Mkdir(sealed, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o700) })

	unresolvable := filepath.Join(sealed, "inner", ".cursor")

	// Whether this case is reachable at all is asked of the filesystem, not of
	// the function under test. Skipping on what realHomeContains returned would
	// make the test pass by not running the moment the answer went wrong —
	// which is what happened: a mutation that made an unresolvable path read as
	// "outside the home" turned this test green by skipping it.
	if _, err := filepath.EvalSymlinks(unresolvable); err == nil || os.IsNotExist(err) {
		t.Skipf("this filesystem resolves %s without a permission error, so the case is not reachable here", unresolvable)
	}

	contained, why, err := realHomeContains(unresolvable)
	if err == nil {
		t.Error("realHomeContains reported no error for a path the filesystem refused to resolve")
	}
	if !contained {
		t.Error("a path that could not be resolved was reported as outside the real home; \"I could not tell\" must not read as \"go ahead\"")
	}
	if why == "" {
		t.Error("no reason was given for the refusal")
	}
}

// TestHomeBarrier_StopsTheWriteThatWouldFollow is the reproduction of the
// defect this barrier exists for: not "does the guard notice", but "does the
// guard stop the next statement".
//
// It runs one test in a child `go test` whose HOME is a temp directory, so the
// home the child refuses to touch is a fake one and the real home is never in
// play. The child calls the barrier on a path inside that fake home and then
// writes a marker file. The assertions are that the child failed, that it said
// why, and that **the marker does not exist** — the last is the one that would
// have caught the shipped bug, where the guard reported and returned.
func TestHomeBarrier_StopsTheWriteThatWouldFollow(t *testing.T) {
	if os.Getenv(barrierChildEnv) != "" {
		t.Skip("running as the child of TestHomeBarrier_StopsTheWriteThatWouldFollow")
	}

	fakeHome := t.TempDir()
	marker := filepath.Join(t.TempDir(), "the-install-ran")

	cmd := exec.Command("go", "test", "-count=1", "-run", "^"+barrierChildTest+"$", "-v", ".")
	cmd.Env = append(envWithout(os.Environ(), "HOME", barrierChildEnv),
		"HOME="+fakeHome,
		barrierChildEnv+"="+filepath.Join(fakeHome, ".cursor"),
		barrierChildMarkerEnv+"="+marker,
	)
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Errorf("the child test passed; the barrier let a path inside its own home through\n%s", out)
	}
	if !strings.Contains(string(out), "refusing") {
		t.Errorf("the child did not say it was refusing:\n%s", out)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the statement after the barrier ran: the guard reported and returned instead of aborting, which is the defect this test exists for")
	}
}

const (
	barrierChildEnv       = "SKILL_ARCHITECT_HOME_BARRIER_DIR"
	barrierChildMarkerEnv = "SKILL_ARCHITECT_HOME_BARRIER_MARKER"
	barrierChildTest      = "TestHomeBarrierChild"
)

// TestHomeBarrierChild is the body the test above runs in a child process. It
// is skipped in every ordinary run; only the parent sets the two variables, and
// the home it is pointed at is the parent's temp directory.
func TestHomeBarrierChild(t *testing.T) {
	dir := os.Getenv(barrierChildEnv)
	if dir == "" {
		t.Skip("child of TestHomeBarrier_StopsTheWriteThatWouldFollow; not run on its own")
	}

	mustBeOutsideRealHome(t, dir)

	// Unreachable when the barrier does its job. This stands for `InstallHooks`
	// — the line that, in the bug this reproduces, ran after the guard had
	// already reported the violation.
	if err := os.WriteFile(os.Getenv(barrierChildMarkerEnv), []byte("ran"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}
}

// envWithout returns env with the named variables removed, so the caller's
// assignment of them is the only one the child sees.
func envWithout(env []string, names ...string) []string {
	var out []string
	for _, kv := range env {
		drop := false
		for _, name := range names {
			if strings.HasPrefix(kv, name+"=") {
				drop = true
			}
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}

// fatalTB is the part of *testing.T the barrier is allowed to use.
//
// It is this narrow on purpose. The bug this barrier replaces was a guard that
// called Errorf, which marks the test failed and then *returns* — so the
// install on the next line ran anyway. An interface carrying only Fatalf makes
// writing that bug a compile error rather than something a reviewer has to
// catch again.
type fatalTB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// mustBeOutsideRealHome aborts unless dir is somewhere other than the real
// user's home directory. Every test that hands a home or a spool directory to
// code that writes goes through it.
//
// The explicit return after Fatalf is not redundant. With a *testing.T, Fatalf
// does not come back; the return says the barrier does not depend on that, so
// the guarantee is the function's own rather than borrowed from testing.
func mustBeOutsideRealHome(tb fatalTB, dir string) {
	tb.Helper()
	contained, why, err := realHomeContains(dir)
	if err != nil {
		tb.Fatalf("refusing to run: %s: %v", why, err)
		return
	}
	if contained {
		tb.Fatalf("refusing to run: %s", why)
		return
	}
}

// sandboxHome returns a directory that stands in for a home directory, proved
// to be outside the real one before it is handed back.
//
// This is the only way a test in this package names a home. A test that built
// one itself would be the test the barrier could not see.
func sandboxHome(tb interface {
	fatalTB
	TempDir() string
}) string {
	tb.Helper()
	dir := tb.TempDir()
	mustBeOutsideRealHome(tb, dir)
	return dir
}

// realHomeContains reports whether dir is the real user's home directory or
// something inside it, and says why when it is.
//
// An error is reported with contained=true. A path that could not be resolved
// is not a path shown to be outside the home, and "I could not tell" must not
// read as "go ahead".
func realHomeContains(dir string) (contained bool, why string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return true, "the real home directory could not be named, so nothing can be shown to be outside it", err
	}
	contained, err = pathContains(home, dir)
	if err != nil {
		return true, dir + " could not be placed relative to the real home " + home + ", so where it would be written is unknown", err
	}
	if !contained {
		return false, "", nil
	}
	return true, dir + " is inside the real home " + home, nil
}

// pathContains reports whether child is parent or something inside it, deciding
// after both are resolved.
//
// Resolution is the point. The question is which directory will be written, not
// how the path was spelled: on macOS t.TempDir() hands back /var/folders/…,
// which is a symlink to /private/var/folders/…, $HOME can itself be a symlink,
// and `..` inside a path says nothing about where it lands. A comparison of the
// unresolved strings answers a different question from the one being asked.
//
// Containment is filepath.Rel rather than a prefix test, because a parent's
// name is a prefix of every sibling that starts with the same letters —
// strings.HasPrefix calls /Users/alice-backup a part of /Users/alice.
func pathContains(parent, child string) (bool, error) {
	resolvedParent, err := resolveForBarrier(parent)
	if err != nil {
		return false, err
	}
	resolvedChild, err := resolveForBarrier(child)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(resolvedParent, resolvedChild)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

// resolveForBarrier makes a path absolute and follows every symlink in it,
// including when the leaf does not exist yet.
//
// A directory a test is about to create does not exist at the moment the
// barrier is asked about it, and refusing to answer for it would push every
// caller into checking a path only after the thing that creates it has run —
// which is after the write. So the deepest existing ancestor is resolved and
// the remainder appended: the place the path *would* be created is what the
// barrier is about.
func resolveForBarrier(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	current, rest := filepath.Clean(abs), ""
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, rest), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		rest = filepath.Join(filepath.Base(current), rest)
		current = parent
	}
}
