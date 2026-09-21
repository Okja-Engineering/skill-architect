package profiler

// Reading the spool in aggregate, and the two readers that have to agree about
// what a spool line is.
//
// `AnalyzeSpool` is the pure-Go reader; `queries/*.sql` is the DuckDB one for
// questions that outgrow a summary. They read the same files, so the assertions
// that keep them describing the same envelope live together here rather than in
// two places that could come to disagree about which columns exist.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// --- The queries and the envelope they read -----------------------------------

// TestSpoolQueries_ReadTheEnvelopeThatIsWritten derives the column list every
// query declares from SpoolEvent itself.
//
// The queries name their columns explicitly — `columns = {ts: 'TIMESTAMPTZ', …}`
// — which is what makes them readable and also what makes them able to fall
// behind. A field added to the envelope is simply absent from a query that
// never mentioned it, and DuckDB reports nothing: the query returns rows with
// the column missing rather than an error. The draft had already fallen behind
// by one — `strict`, the field that says a line's content was replaced by its
// sizes, was in no query at all, so a reader pooling a spool could not tell a
// stripped line from one that carried no prompt.
//
// So the set is read off the struct's json tags, and every query is required to
// name all of it. A ninth envelope field turns this red instead of quietly
// shipping eight queries that cannot see it.
func TestSpoolQueries_ReadTheEnvelopeThatIsWritten(t *testing.T) {
	fields := envelopeJSONFields(t)
	if len(fields) < 2 {
		t.Fatalf("the envelope was derived as %v, which is too few for this check to mean anything", fields)
	}

	queries := sqlFilesIn(t, "queries")
	if len(queries) == 0 {
		t.Fatal("no .sql file was found under queries/, so this check reads nothing")
	}

	for _, path := range queries {
		t.Run(filepath.Base(path), func(t *testing.T) {
			body := readFile(t, path)
			for _, field := range fields {
				if !strings.Contains(body, field+":") {
					t.Errorf("declares no %q column, so a reader of this query cannot see the envelope field of that name", field)
				}
			}
		})
	}
}

// TestSpoolQueries_DefaultToTheSpoolThisProductWrites pins the path in the
// queries to the one the writer uses, derived rather than restated.
//
// Every query falls back to a default spool directory when SPOOL_DIR is unset,
// and that default is a path spelled into eight files. DefaultSpoolDir is the
// one that decides where lines are actually written, so the queries are asked
// to agree with it: a rename there turns these red rather than leaving eight
// copies of a directory nothing writes to.
func TestSpoolQueries_DefaultToTheSpoolThisProductWrites(t *testing.T) {
	want := spoolDirRelativeToHome(t)

	for _, path := range sqlFilesIn(t, "queries") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if body := readFile(t, path); !strings.Contains(body, want) {
				t.Errorf("does not fall back to %q, the spool DefaultSpoolDir writes to", want)
			}
		})
	}
}

// TestSpoolQueries_ClaimNoMeasurementOfWhatTheHarnessDid is the wording rule,
// asserted over the files rather than left to a reviewer.
//
// A query is a document a user runs and quotes. One named for a token count,
// computing chars/4 over payload bytes, reports an estimate as a measurement —
// and one extracting a skill name out of a file path asserts an activation
// nobody observed. Both were in the draft. Neither is in a position to carry a
// hedge: a column header travels without its README.
//
// The forbidden spellings are the vocabulary of a measurement this release
// cannot make from a spool. They are assembled at runtime because this file is
// not one of the files searched, but the queries' own README is.
func TestSpoolQueries_ClaimNoMeasurementOfWhatTheHarnessDid(t *testing.T) {
	forbidden := []string{
		"est_tokens", "estimated_tokens", "context_tokens",
		"skill_activation", "SKILL.md", "inferred",
	}

	files := append(sqlFilesIn(t, "queries"), filepath.Join("queries", "README.md"))
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			body := readFile(t, path)
			for _, word := range forbidden {
				if strings.Contains(body, word) {
					t.Errorf("carries %q: a query may describe what the spool contains, and a token count or a "+
						"skill activation is a claim about what the harness did — which no adapter in this release reads", word)
				}
			}
		})
	}
}

// envelopeJSONFields is the spool line's own field names, read off the type
// that writes them.
func envelopeJSONFields(t *testing.T) []string {
	t.Helper()
	rt := reflect.TypeOf(SpoolEvent{})
	var names []string
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			t.Fatalf("SpoolEvent.%s has no json tag, so the queries cannot be held to it", rt.Field(i).Name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// spoolDirRelativeToHome is DefaultSpoolDir with the real home taken off the
// front, which is the form a query's `getenv('HOME') || …` fallback spells.
//
// The real home is read and nothing else: no directory is created, and the
// value is used only to subtract a prefix.
func spoolDirRelativeToHome(t *testing.T) string {
	t.Helper()
	dir, err := DefaultSpoolDir()
	if err != nil {
		t.Fatalf("DefaultSpoolDir: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}
	rel := strings.TrimPrefix(dir, home)
	if rel == dir {
		t.Fatalf("DefaultSpoolDir %q is not under the home %q, so there is no relative form to look for", dir, home)
	}
	return rel
}

func sqlFilesIn(t *testing.T, dir string) []string {
	t.Helper()
	return filesUnder(t, dir, func(name string) bool { return strings.HasSuffix(name, ".sql") })
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

// --- AnalyzeSpool -------------------------------------------------------------

// seedSpool writes a spool the way `ingest` writes one: through the normalizer,
// so what is read back is what the writer would have produced and not a
// hand-rolled line that happens to parse.
//
// The payload shapes are the documented ones. They stand for "a payload
// carrying these fields" and not for "the payload Cursor sends" — nothing here
// has seen one, which is the whole reason the summary below describes the files
// rather than the session.
func seedSpool(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(day string, payloads ...string) {
		for _, p := range payloads {
			ev, err := NormalizeHookPayload([]byte(p), mustTime(t, day+"T12:00:00Z"))
			if err != nil {
				t.Fatalf("normalize %s: %v", p, err)
			}
			if err := AppendSpool(dir, ev); err != nil {
				t.Fatalf("append: %v", err)
			}
		}
	}
	write("2026-09-10",
		`{"hook_event_name":"sessionStart","conversation_id":"c1","model":"gpt-5"}`,
		`{"hook_event_name":"beforeSubmitPrompt","conversation_id":"c1","prompt":"fix the flaky test"}`,
		`{"hook_event_name":"preToolUse","conversation_id":"c1","tool_name":"Bash","tool_input":{"command":"go test ./..."}}`,
		`{"hook_event_name":"postToolUse","conversation_id":"c1","tool_name":"Bash","duration":1200}`,
		`{"hook_event_name":"beforeReadFile","conversation_id":"c1","file_path":"/repo/.cursor/skills/commit/SKILL.md"}`,
		`{"hook_event_name":"sessionEnd","conversation_id":"c1","duration_ms":300000,"final_status":"completed"}`,
	)
	write("2026-09-11",
		`{"hook_event_name":"sessionStart","conversation_id":"c2","model":"claude-4"}`,
		`{"hook_event_name":"beforeMCPExecution","conversation_id":"c2","tool_name":"query","mcp_server_name":"linear"}`,
		`{"hook_event_name":"subagentStart","conversation_id":"c2","subagent_type":"researcher"}`,
		`{"hook_event_name":"preCompact","conversation_id":"c2","context_tokens":48200,"context_window_size":200000}`,
		`{"hook_event_name":"beforeReadFile","conversation_id":"c2","file_path":"/repo/src/main.go"}`,
	)
	return dir
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return tm
}

// TestAnalyzeSpool_CountsTheFilesAndLinesItRead is the summary's foundation: a
// user asking what is in the spool gets the size of the corpus first.
func TestAnalyzeSpool_CountsTheFilesAndLinesItRead(t *testing.T) {
	dir := seedSpool(t)
	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	if got.Files != 2 {
		t.Errorf("files = %d, want 2", got.Files)
	}
	if got.Lines != 11 {
		t.Errorf("lines = %d, want 11", got.Lines)
	}
	if got.Envelopes != 11 {
		t.Errorf("envelopes = %d, want 11", got.Envelopes)
	}
	if got.UnreadableLines != 0 {
		t.Errorf("unreadable_lines = %d, want 0", got.UnreadableLines)
	}
	if got.ConversationIDs != 2 {
		t.Errorf("conversation_ids = %d, want 2", got.ConversationIDs)
	}
	if got.CaptureDays["2026-09-10"] != 6 || got.CaptureDays["2026-09-11"] != 5 {
		t.Errorf("capture_days = %v, want 6 and 5", got.CaptureDays)
	}
	if got.EventNames["sessionStart"] != 2 {
		t.Errorf("event_names[sessionStart] = %d, want 2", got.EventNames["sessionStart"])
	}
	if got.SchemaVersions[SpoolSchemaVersion] != 11 {
		t.Errorf("schema_versions = %v, want 11 lines at %q", got.SchemaVersions, SpoolSchemaVersion)
	}
	// The exact total, summed from the same files by reading them back.
	//
	// "> 0" was the first form of this assertion and it could not tell bytes
	// from bytes over four — which is exactly the difference between a size and
	// the draft's estimated token count, so it is the one number here that has
	// to be pinned rather than sanity-checked.
	if want := payloadBytesIn(t, dir); got.PayloadBytes != want {
		t.Errorf("payload_bytes = %d, want %d — the size of the payloads as they are stored", got.PayloadBytes, want)
	}
}

// payloadBytesIn sums the stored size of every payload in a spool, read back
// out of the files rather than restated as a literal.
func payloadBytesIn(t *testing.T, dir string) int64 {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var total int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		body := readFile(t, filepath.Join(dir, e.Name()))
		for _, line := range strings.Split(body, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev SpoolEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				t.Fatalf("decode %s: %v", line, err)
			}
			total += int64(len(ev.Raw))
		}
	}
	if total == 0 {
		t.Fatal("the spool being compared against holds no payload bytes, so this assertion reads nothing")
	}
	return total
}

// TestAnalyzeSpool_ReportsTheMostRecentCaptureTime is what answers "is the
// capture still running": a spool with lines in it whose newest line is three
// weeks old is a different machine from one whose newest is a minute old, and
// the day a file is named for is not precise enough to tell them apart.
//
// It is the maximum across every file rather than the last line of the last
// file. Those are the same thing only while the files are read in order and
// every line is appended in order, and a reader that assumed both would report
// a stale time for a spool synced from another machine.
func TestAnalyzeSpool_ReportsTheMostRecentCaptureTime(t *testing.T) {
	dir := t.TempDir()
	for _, ts := range []string{
		"2026-09-11T02:00:00Z", // written first, and the newest
		"2026-09-10T01:00:00Z",
		"2026-09-10T23:59:59Z",
	} {
		ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"stop"}`), mustTime(t, ts))
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if err := AppendSpool(dir, ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}
	if got.LastCaptureAt != "2026-09-11T02:00:00Z" {
		t.Errorf("last_capture_at = %q, want the newest capture time in the spool", got.LastCaptureAt)
	}

	empty, err := AnalyzeSpool(t.TempDir())
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}
	if empty.LastCaptureAt != "" {
		t.Errorf("last_capture_at = %q for a spool with no lines, want empty", empty.LastCaptureAt)
	}
}

// TestAnalyzeSpool_ReportsWhatThePayloadsCarryRatherThanWhatTheHarnessDid is
// the census that replaced four hand-guessed accumulators.
//
// The draft read `tool_name` off preToolUse into `tool_calls`, `model` into
// `models`, `mcp_server_name` into `mcp_servers` and `subagent_type` into
// `subagent_runs` — four guesses at field names nobody has observed, each
// reported under the name of a thing the harness does. If Cursor calls the
// field something else, all four report zero and the summary says a session
// made no tool calls.
//
// One census of the keys that are actually there cannot be wrong that way, and
// it answers the question the release needs answered — what does a payload
// carry — instead of asserting the answer.
func TestAnalyzeSpool_ReportsWhatThePayloadsCarryRatherThanWhatTheHarnessDid(t *testing.T) {
	got, err := AnalyzeSpool(seedSpool(t))
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	for _, c := range []struct {
		key  string
		want int
	}{
		{"hook_event_name", 11},
		{"conversation_id", 11},
		{"tool_name", 3},
		{"model", 2},
		{"mcp_server_name", 1},
		{"subagent_type", 1},
		{"context_tokens", 1},
	} {
		if got.PayloadKeys[c.key] != c.want {
			t.Errorf("payload_keys[%q] = %d, want %d", c.key, got.PayloadKeys[c.key], c.want)
		}
	}
	if _, present := got.PayloadKeys["no_such_field"]; present {
		t.Error("a key no payload carried is in the census")
	}

	// The values, for the keys whose values are short strings. This is where a
	// reader sees which tools and which models, without anything having decided
	// that `tool_name` means a tool call.
	if got.PayloadKeyValues["tool_name"]["Bash"] != 2 {
		t.Errorf("payload_key_values[tool_name][Bash] = %d, want 2", got.PayloadKeyValues["tool_name"]["Bash"])
	}
	if got.PayloadKeyValues["model"]["gpt-5"] != 1 || got.PayloadKeyValues["model"]["claude-4"] != 1 {
		t.Errorf("payload_key_values[model] = %v, want one each", got.PayloadKeyValues["model"])
	}
	if got.PayloadKeyValues["subagent_type"]["researcher"] != 1 {
		t.Errorf("payload_key_values[subagent_type] = %v", got.PayloadKeyValues["subagent_type"])
	}

	// A number is counted as a key and not censused as a value: a census of
	// every numeric value in a spool is a histogram of measurements nobody
	// made, and the whole point of the key census is that it asserts nothing
	// about what the numbers mean.
	if vals, ok := got.PayloadKeyValues["context_tokens"]; ok {
		t.Errorf("payload_key_values[context_tokens] = %v, and a number has no value census", vals)
	}
}

// TestAnalyzeSpool_DoesNotQuoteALongValue is the other half of the census
// bound, and it is not about privacy — the content keys cover that. It is that
// a summary quoting an arbitrarily long string stops being a summary: one
// pasted `cwd` of a thousand characters, or a field carrying a whole document,
// and the document a user is meant to read is a copy of the spool.
// The two lengths are written out rather than derived from
// maxCensusedValueLen. Deriving them makes the test's own inputs scale with the
// bound, so raising the bound raises the "too long" value with it and the case
// can never fail — the guard comparing a count against the set it came from,
// which this release has now found in five different shapes. So the constant is
// asserted to be what these literals were chosen for, and a change to it turns
// this red instead of quietly rescaling.
func TestAnalyzeSpool_DoesNotQuoteALongValue(t *testing.T) {
	if maxCensusedValueLen != 64 {
		t.Fatalf("maxCensusedValueLen = %d; the lengths below were written for 64 and have to be rewritten with it", maxCensusedValueLen)
	}

	dir := t.TempDir()
	short := strings.Repeat("s", 64)
	long := strings.Repeat("L", 65)
	ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"workspaceOpen","cwd":"`+short+`","note":"`+long+`"}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := AppendSpool(dir, ev); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	if got.PayloadKeyValues["cwd"][short] != 1 {
		t.Errorf("a value of exactly %d characters was not quoted: %v", len(short), got.PayloadKeyValues["cwd"])
	}
	if got.PayloadKeys["note"] != 1 {
		t.Errorf("payload_keys[note] = %d, want 1 — the key is counted whatever its value's length", got.PayloadKeys["note"])
	}
	if vals, ok := got.PayloadKeyValues["note"]; ok {
		t.Errorf("payload_key_values[note] has %d entries; a value of %d characters is counted, not quoted", len(vals), len(long))
	}
}

// TestAnalyzeSpool_CountsAnEnvelopeCarryingNoPayload covers the line this
// package cannot write and a later writer might: an envelope with no `raw`
// member at all.
//
// It decodes as an envelope, so it is not an unreadable line — but there is no
// payload to walk, and a summary reporting neither an object payload nor
// anything else would leave the two counts not adding up to the envelopes with
// nothing saying why.
func TestAnalyzeSpool_CountsAnEnvelopeCarryingNoPayload(t *testing.T) {
	dir := t.TempDir()
	line := `{"ts":"2026-09-20T14:30:00Z","event":"stop","schema_version":"` + SpoolSchemaVersion + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "2026-09-20.jsonl"), []byte(line), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	if got.Envelopes != 1 || got.UnreadableLines != 0 {
		t.Errorf("envelopes = %d, unreadable_lines = %d, want 1 and 0 — the envelope parsed", got.Envelopes, got.UnreadableLines)
	}
	if got.ObjectPayloads != 0 || got.OtherPayloads != 1 {
		t.Errorf("object_payloads = %d, other_payloads = %d, want 0 and 1", got.ObjectPayloads, got.OtherPayloads)
	}
	if got.ObjectPayloads+got.OtherPayloads != got.Envelopes {
		t.Errorf("%d object and %d other payloads do not account for %d envelopes",
			got.ObjectPayloads, got.OtherPayloads, got.Envelopes)
	}
}

// TestAnalyzeSpool_KeepsPromptTextOutOfTheSummary is the privacy half.
//
// The summary is a document a user pastes into an issue. A census of every
// short string value would put a short prompt in it, so the keys the capture
// layer already knows to be content are counted and never valued — the same set
// `--strict` replaces with sizes, asked of the one place it is declared.
func TestAnalyzeSpool_KeepsPromptTextOutOfTheSummary(t *testing.T) {
	dir := t.TempDir()
	ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"beforeSubmitPrompt","prompt":"ship it","summary":"tiny"}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := AppendSpool(dir, ev); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	if got.PayloadKeys["prompt"] != 1 {
		t.Errorf("payload_keys[prompt] = %d, want 1 — that a prompt was carried is metadata", got.PayloadKeys["prompt"])
	}
	for _, key := range []string{"prompt", "summary"} {
		if vals, ok := got.PayloadKeyValues[key]; ok {
			t.Errorf("payload_key_values[%q] = %v: content is counted, never quoted", key, vals)
		}
	}

	// And nothing else in the document carries the text either.
	doc, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(doc), "ship it") {
		t.Errorf("the summary quotes a prompt:\n%s", doc)
	}
}

// TestAnalyzeSpool_SaysHowManyLinesItCouldNotRead is the count the draft did
// not keep.
//
// A corrupt line is skipped, because a hook killed mid-write leaves a truncated
// last one and losing the rest of the day over it would break the spool's one
// promise. Skipping it silently is a different thing: the summary then reports a
// smaller corpus than the files hold and nothing says so, which is the "the
// capture was empty" conclusion drawn from a reader's own failure.
func TestAnalyzeSpool_SaysHowManyLinesItCouldNotRead(t *testing.T) {
	dir := t.TempDir()
	ev, err := NormalizeHookPayload([]byte(`{"hook_event_name":"sessionStart"}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := AppendSpool(dir, ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	// A truncated final line, and a blank one, appended to the same file.
	f, err := os.OpenFile(filepath.Join(dir, fixedNow.Format("2006-01-02")+".jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := f.WriteString("{\"ts\":\"2026-09-20T14:30:00Z\",\"eve\n\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	if got.Lines != 2 {
		t.Errorf("lines = %d, want 2 — a blank line is not a line that was read", got.Lines)
	}
	if got.Envelopes != 1 {
		t.Errorf("envelopes = %d, want 1", got.Envelopes)
	}
	if got.UnreadableLines != 1 {
		t.Errorf("unreadable_lines = %d, want 1 — a line skipped in silence is a corpus reported smaller than it is", got.UnreadableLines)
	}
}

// TestAnalyzeSpool_CountsAPayloadThatIsNotAnObject covers the shape the capture
// layer explicitly keeps: input that was not JSON at all is stored as a JSON
// string, and a summary that ignored it would report fewer payloads than lines
// for no stated reason.
func TestAnalyzeSpool_CountsAPayloadThatIsNotAnObject(t *testing.T) {
	dir := t.TempDir()
	for _, payload := range []string{`{"hook_event_name":"stop"}`, `not json at all`, `[1,2,3]`} {
		ev, err := NormalizeHookPayload([]byte(payload), fixedNow)
		if err != nil {
			t.Fatalf("normalize %q: %v", payload, err)
		}
		if err := AppendSpool(dir, ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}

	if got.ObjectPayloads != 1 {
		t.Errorf("object_payloads = %d, want 1", got.ObjectPayloads)
	}
	if got.OtherPayloads != 2 {
		t.Errorf("other_payloads = %d, want 2 — a payload the census cannot walk is still a payload that arrived", got.OtherPayloads)
	}
	if got.EventNames["unknown"] != 2 {
		t.Errorf("event_names[unknown] = %d, want 2", got.EventNames["unknown"])
	}
}

// TestAnalyzeSpool_CountsStrippedLinesSeparately keeps the one deliberate loss
// visible. A summary over a strict spool describes payloads whose content was
// replaced by its size, and a reader who cannot tell has a corpus that looks
// like it carried no prompts.
func TestAnalyzeSpool_CountsStrippedLinesSeparately(t *testing.T) {
	dir := t.TempDir()
	plain, err := NormalizeHookPayload([]byte(`{"hook_event_name":"beforeSubmitPrompt","prompt":"one"}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	stripped, err := NormalizeHookPayloadStrict([]byte(`{"hook_event_name":"beforeSubmitPrompt","prompt":"two"}`), fixedNow)
	if err != nil {
		t.Fatalf("normalize strict: %v", err)
	}
	for _, ev := range []SpoolEvent{plain, stripped} {
		if err := AppendSpool(dir, ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}
	if got.StrippedEnvelopes != 1 {
		t.Errorf("stripped_envelopes = %d, want 1", got.StrippedEnvelopes)
	}
	if got.Envelopes != 2 {
		t.Errorf("envelopes = %d, want 2", got.Envelopes)
	}
}

// TestAnalyzeSpool_ReadsOnlyTheSpoolFiles pins what counts as a spool line. The
// spool sits under the user's home and collects other things — a backup, an
// editor's swap file, a directory somebody made — and reading those as spool
// lines would be a reader inventing events. It is the same rule
// LoadSessionEvents keeps, because both now walk the same listing.
func TestAnalyzeSpool_ReadsOnlyTheSpoolFiles(t *testing.T) {
	dir := seedSpool(t)
	if err := os.WriteFile(filepath.Join(dir, "2026-09-10.jsonl.bak"), []byte(`{"ts":"2026-09-10T00:00:00Z","event":"nope","raw":{}}`+"\n"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "2026-09-12.jsonl"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := AnalyzeSpool(dir)
	if err != nil {
		t.Fatalf("AnalyzeSpool: %v", err)
	}
	if got.Files != 2 {
		t.Errorf("files = %d, want 2 — a backup beside the spool is not a spool file and a directory named like one is not either", got.Files)
	}
	if got.EventNames["nope"] != 0 {
		t.Error("an event was read out of a file that is not a spool file")
	}
}

// TestAnalyzeSpool_DistinguishesAnEmptySpoolFromOneThatIsNotThere is the
// refusal LoadSessionEvents already makes, kept the same here: a caller who
// cannot tell them apart reports "nothing happened" for a capture that never
// ran.
func TestAnalyzeSpool_DistinguishesAnEmptySpoolFromOneThatIsNotThere(t *testing.T) {
	empty, err := AnalyzeSpool(t.TempDir())
	if err != nil {
		t.Fatalf("an existing but empty spool is not an error: %v", err)
	}
	if empty.Lines != 0 || empty.Files != 0 {
		t.Errorf("an empty spool summarised as %d files and %d lines", empty.Files, empty.Lines)
	}

	missing := filepath.Join(t.TempDir(), "never-captured")
	if _, err := AnalyzeSpool(missing); err == nil {
		t.Error("a spool directory that is not there was summarised as an empty one")
	}
}

// TestSpoolSummary_MakesNoSignalClaim is the rule in the type rather than in
// the prose.
//
// The projection this slice does not ship asserted `tool_calls: present,
// source: hooks` on the strength of guessed field names — a claim made in the
// data, which travels where a prose limit cannot follow it. A summary of the
// files is allowed; a signal result is not, and the way to keep it that way is
// to refuse the types that carry one rather than to remember not to use them.
func TestSpoolSummary_MakesNoSignalClaim(t *testing.T) {
	forbidden := map[string]bool{}
	for _, v := range []any{
		RawMetricResult{}, TokenResult{}, ToolCallResult{}, ActivationResult{},
		TimingResult{}, AttributionResult{}, EstimatedTokensResult{},
		MetricState(""), MetricSource(""), CapabilityReport{}, Profile{},
	} {
		forbidden[reflect.TypeOf(v).String()] = true
	}
	if len(forbidden) < 11 {
		t.Fatalf("the forbidden set collapsed to %d entries, so this check reads almost nothing", len(forbidden))
	}

	rt := reflect.TypeOf(SpoolSummary{})
	if rt.NumField() == 0 {
		t.Fatal("SpoolSummary has no fields, so this check reads nothing")
	}
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		for typ := field.Type; ; {
			if forbidden[typ.String()] {
				t.Errorf("SpoolSummary.%s is a %s: a summary of the spool may not carry a signal result, "+
					"because a state of \"present\" is a claim about what the harness did and it travels with the document",
					field.Name, typ)
			}
			if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
				typ = typ.Elem()
				continue
			}
			break
		}
	}
}
