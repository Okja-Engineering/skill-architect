package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reading the spool back is what makes writing it worth doing, and it is the
// half that proves the capture claim: a line that cannot be read back is a line
// that was lost.
//
// Nothing here turns a spool into a measurement. The projection from hook
// events to a Profile — pairing tool calls, inferring skill activations,
// attributing one to the other — is not in this slice: every one of those is a
// claim about payloads nobody here has observed, and a Profile saying
// `tool_calls: present, source: hooks` is that claim made in the data rather
// than in the prose. See the note at the top of spool_profile.go.

func TestLoadSessionEvents_MatchesOnEveryIdentifierASessionIsNamedBy(t *testing.T) {
	dir := sandboxSpoolDir(t)
	writeSpool(t, dir, "2026-09-20.jsonl",
		storedLine(t, `{"hook_event_name":"sessionStart","conversation_id":"conv-1"}`, fixedNow),
		storedLine(t, `{"hook_event_name":"preToolUse","session_id":"conv-1"}`, fixedNow.Add(time.Minute)),
		storedLine(t, `{"hook_event_name":"postToolUse","generation_id":"conv-1"}`, fixedNow.Add(2*time.Minute)),
		storedLine(t, `{"hook_event_name":"stop","conversation_id":"someone-elses-session"}`, fixedNow.Add(3*time.Minute)),
	)

	events, err := LoadSessionEvents(dir, "conv-1")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}

	if got := eventNames(events); strings.Join(got, ",") != "sessionStart,preToolUse,postToolUse" {
		t.Errorf("events = %v, want the three carrying the session and not the fourth", got)
	}
}

// TestLoadSessionEvents_RefusesAnEmptySessionID is the provenance rule this
// release exists to close, applied one layer down.
//
// An empty id is not a session that matched nothing. Matched loosely it selects
// every event whose conversation_id happens to be absent, which is a whole
// machine's spool returned as one session — the session-scoping defect four
// releases were spent closing, and the one the three unlanded adapters each
// reproduced. The read refuses instead.
func TestLoadSessionEvents_RefusesAnEmptySessionID(t *testing.T) {
	dir := sandboxSpoolDir(t)
	writeSpool(t, dir, "2026-09-20.jsonl",
		storedLine(t, `{"hook_event_name":"sessionStart"}`, fixedNow),
		storedLine(t, `{"hook_event_name":"stop","conversation_id":"conv-1"}`, fixedNow.Add(time.Minute)),
	)

	events, err := LoadSessionEvents(dir, "")
	if err == nil {
		t.Fatalf("an empty session id returned %d events instead of a refusal", len(events))
	}
	if events != nil {
		t.Errorf("the refusal still returned %d events", len(events))
	}
	if !strings.Contains(err.Error(), "session") {
		t.Errorf("error = %q, want it to name what is missing", err)
	}
}

// TestSessionMatches_AnEventThatNamesNoSessionBelongsToNone is the invariant
// underneath the refusal above, at the layer that does the comparing.
//
// Every comparison in sessionMatches is an equality against a field that reads
// as "" when it is absent. So an event carrying no identifier at all must match
// no named session — and the reason an *empty* id is refused one layer up is
// that it would match exactly these events, which is a machine's whole spool
// handed back as one session.
func TestSessionMatches_AnEventThatNamesNoSessionBelongsToNone(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{"an object with none of the three ids", `{"hook_event_name":"stop"}`},
		{"an object whose ids are empty", `{"hook_event_name":"stop","conversation_id":"","session_id":""}`},
		{"a payload that was never JSON, stored as a string", `this was never JSON`},
		{"an object whose id is not a string", `{"hook_event_name":"stop","conversation_id":42}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := NormalizeHookPayload([]byte(tc.payload), fixedNow)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if sessionMatches(ev, "conv-1") {
				t.Errorf("an event naming no session was claimed by conv-1: %s", ev.Raw)
			}
		})
	}
}

// TestLoadSessionEvents_ReadsTheWholeSpoolInCaptureOrder. Events for one
// session are spread over as many daily files as the session spans, and a
// reader has to see them as one ordered sequence.
func TestLoadSessionEvents_ReadsTheWholeSpoolInCaptureOrder(t *testing.T) {
	dir := sandboxSpoolDir(t)

	// Written out of order on purpose: the later day is appended first, and
	// within one file the two lines are stored newest-first.
	writeSpool(t, dir, "2026-09-21.jsonl",
		spoolLine(t, "afterAgentResponse", "conv-1", fixedNow.Add(26*time.Hour)),
		spoolLine(t, "sessionEnd", "conv-1", fixedNow.Add(25*time.Hour)),
	)
	writeSpool(t, dir, "2026-09-20.jsonl",
		spoolLine(t, "preToolUse", "conv-1", fixedNow.Add(time.Minute)),
		spoolLine(t, "sessionStart", "conv-1", fixedNow),
	)

	events, err := LoadSessionEvents(dir, "conv-1")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}

	want := "sessionStart,preToolUse,sessionEnd,afterAgentResponse"
	if got := eventNames(events); strings.Join(got, ",") != want {
		t.Errorf("events = %v, want %s — ordered by capture time across files", got, want)
	}
}

// TestLoadSessionEvents_SurvivesALineItCannotRead. A hook that was killed
// mid-write leaves a truncated last line. Losing the rest of the day because of
// it would make the spool's one promise — that what was captured can be read
// back — false exactly when it matters.
func TestLoadSessionEvents_SurvivesALineItCannotRead(t *testing.T) {
	dir := sandboxSpoolDir(t)
	writeSpool(t, dir, "2026-09-20.jsonl",
		spoolLine(t, "sessionStart", "conv-1", fixedNow),
		`{"ts":"2026-09-20T14:31:00Z","event":"preToolUse","raw":{"conversation`, // killed mid-write
		"",
		spoolLine(t, "stop", "conv-1", fixedNow.Add(2*time.Minute)),
	)

	events, err := LoadSessionEvents(dir, "conv-1")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}
	if got := eventNames(events); strings.Join(got, ",") != "sessionStart,stop" {
		t.Errorf("events = %v, want the two whole lines around the truncated one", got)
	}
}

// TestLoadSessionEvents_ReadsOnlyTheSpoolFiles. The spool directory is under
// the user's home and other things end up there: an editor's swap file, a
// backup, a subdirectory. Reading them as spool lines would be a reader
// inventing events.
func TestLoadSessionEvents_ReadsOnlyTheSpoolFiles(t *testing.T) {
	dir := sandboxSpoolDir(t)
	writeSpool(t, dir, "2026-09-20.jsonl", spoolLine(t, "sessionStart", "conv-1", fixedNow))
	writeSpool(t, dir, "2026-09-20.jsonl.bak", spoolLine(t, "notAnEvent", "conv-1", fixedNow))
	writeSpool(t, dir, "notes.txt", "conv-1 was the interesting one")
	if err := os.MkdirAll(filepath.Join(dir, "archive.jsonl"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	events, err := LoadSessionEvents(dir, "conv-1")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}
	if got := eventNames(events); strings.Join(got, ",") != "sessionStart" {
		t.Errorf("events = %v, want only the one line in the one spool file", got)
	}
}

// TestLoadSessionEvents_ALineWhoseRawIsNotAnObject. A payload that was not JSON
// is stored as a JSON string, so the session ids cannot be looked up inside it.
// Such a line belongs to a session only if the envelope says so, and asking
// must not take the read down.
func TestLoadSessionEvents_ALineWhoseRawIsNotAnObject(t *testing.T) {
	dir := sandboxSpoolDir(t)
	writeSpool(t, dir, "2026-09-20.jsonl",
		storedLine(t, `this was never JSON`, fixedNow),
		storedLine(t, `{"hook_event_name":"stop","conversation_id":"conv-1"}`, fixedNow.Add(time.Minute)),
	)

	events, err := LoadSessionEvents(dir, "conv-1")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}
	if got := eventNames(events); strings.Join(got, ",") != "stop" {
		t.Errorf("events = %v, want only the line whose envelope names the session", got)
	}
}

func TestLoadSessionEvents_AnEmptySpoolIsNotAnError(t *testing.T) {
	dir := sandboxSpoolDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	events, err := LoadSessionEvents(dir, "conv-1")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("got %d events from an empty spool", len(events))
	}
}

func TestLoadSessionEvents_SaysSoWhenTheSpoolIsNotThere(t *testing.T) {
	dir := sandboxSpoolDir(t)

	_, err := LoadSessionEvents(filepath.Join(dir, "never-created"), "conv-1")
	if err == nil {
		t.Fatal("a spool directory that does not exist read as an empty one; a caller cannot tell a session with no events from a spool that was never written")
	}
}

// TestLoadSessionEvents_RoundTripsWhatIngestWrote joins the two halves: what
// the capture layer put on disk is what the read layer hands back. This is the
// capture claim, proved rather than asserted.
func TestLoadSessionEvents_RoundTripsWhatIngestWrote(t *testing.T) {
	dir := sandboxSpoolDir(t)
	payloads := [][]byte{
		[]byte(`{"hook_event_name":"sessionStart","conversation_id":"conv-7","cursor_version":"1.7.3","cwd":"/work"}`),
		[]byte(`{"hook_event_name":"preToolUse","conversation_id":"conv-7","tool_use_id":"call-1","tool_name":"Read","start_time_unix_nano":1758378600123456789}`),
		[]byte(`{"hook_event_name":"aFutureEvent","conversation_id":"conv-7","somethingNobodyHereReads":{"a":[1,2,3]}}`),
	}

	src := NewSliceSource(payloads)
	for i := range payloads {
		if _, err := Ingest(src, dir, fixedNow.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("Ingest %d: %v", i, err)
		}
	}

	events, err := LoadSessionEvents(dir, "conv-7")
	if err != nil {
		t.Fatalf("LoadSessionEvents: %v", err)
	}
	if len(events) != len(payloads) {
		t.Fatalf("read back %d events, wrote %d", len(events), len(payloads))
	}

	if got := eventNames(events); strings.Join(got, ",") != "sessionStart,preToolUse,aFutureEvent" {
		t.Errorf("events = %v, want them in the order they were captured", got)
	}
	if events[0].CursorVersion != "1.7.3" || events[0].Cwd != "/work" {
		t.Errorf("envelope = %+v, want the promoted fields intact", events[0])
	}
	for _, want := range []string{"call-1", "Read", "1758378600123456789", "somethingNobodyHereReads"} {
		if !strings.Contains(string(events[1].Raw)+string(events[2].Raw), want) {
			t.Errorf("%q did not survive the round trip through the spool", want)
		}
	}
}

// --- helpers ---

func writeSpool(t *testing.T, dir, name string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// storedLine builds one stored line out of a hook payload, the way Ingest
// would, so a fixture cannot drift from the writer it stands for. A fixture
// written as a payload where the file holds envelopes reads as an empty spool,
// which is a test that passes by matching nothing.
func storedLine(t *testing.T, payload string, at time.Time) string {
	t.Helper()
	ev, err := NormalizeHookPayload([]byte(payload), at)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	body, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(body)
}

func spoolLine(t *testing.T, event, conversationID string, at time.Time) string {
	t.Helper()
	return storedLine(t, `{"hook_event_name":"`+event+`","conversation_id":"`+conversationID+`"}`, at)
}

func eventNames(events []SpoolEvent) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Event)
	}
	return out
}
