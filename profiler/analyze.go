// Summarising the spool: what is in these files, and nothing about what the
// harness did with them.
//
// # The line this file does not cross
//
// The spool is captured by a layer that has never seen a payload from a running
// Cursor. Every field name in it comes from Cursor's published documentation, so
// a reader that turns `tool_name` into a tool call and `context_tokens` into a
// token count is asserting the documentation is right — in the data, where a
// hedge in a doc comment cannot follow it. A summary travels: it is pasted into
// issues, diffed between weeks and quoted months later by people who never read
// this file.
//
// So the summary describes the files. It counts lines, files, envelope field
// values, payload bytes and which payload keys are present, and it stops there.
// The draft this replaces went four steps further — `tool_calls` from
// `tool_name`, `models` from `model`, `mcp_servers` from `mcp_server_name`,
// `subagent_runs` from `subagent_type`, plus `skill_activations` inferred from
// a read of a SKILL.md and `estimated_tokens` as payload bytes over four. Each
// was a guess at a field name reported under the name of a thing the harness
// does, and if any guess is wrong the summary reports that a session made no
// tool calls rather than that this build could not find the field.
//
// One census of the keys that are actually there cannot be wrong that way, and
// it answers the question the release needs answered — what does a hook payload
// carry — rather than assuming the answer. When somebody has run Cursor, the
// census output is what the adapter should be written against.
package profiler

import (
	"encoding/json"
	"strings"
)

// SpoolSummary is what a spool directory contains.
//
// Every field is a count of something in the files. None is a measurement of a
// session: there is no state, no source and no signal result anywhere in it,
// and TestSpoolSummary_MakesNoSignalClaim keeps it that way by refusing the
// types that carry one.
type SpoolSummary struct {
	// Dir is the directory this summary was read from, so a stored summary
	// says which spool it describes.
	Dir string `json:"dir"`

	// Files is the number of *.jsonl files read; Lines is every non-blank line
	// in them. Envelopes and UnreadableLines partition Lines: a line that could
	// not be decoded is counted rather than skipped in silence, because a
	// corpus reported smaller than it is reads as a capture that did not
	// happen.
	Files           int `json:"files"`
	Lines           int `json:"lines"`
	Envelopes       int `json:"envelopes"`
	UnreadableLines int `json:"unreadable_lines"`

	// StrippedEnvelopes is the lines written in metadata-only mode, whose
	// content fields were replaced by their sizes before the line was written.
	// Reported apart because their payload bytes are not comparable with an
	// unstripped line's, and because a reader who cannot tell has a corpus that
	// looks like it carried no prompts.
	StrippedEnvelopes int `json:"stripped_envelopes"`

	// The envelope's own fields, counted by value exactly as written. An
	// envelope whose event name or schema version is absent counts under the
	// empty string: "unknown" is the capture layer's word for a payload that
	// named no event, and reporting it for a field that was never there would
	// put the writer's vocabulary on a line the writer did not write.
	SchemaVersions map[string]int `json:"schema_versions"`
	EventNames     map[string]int `json:"event_names"`

	// CaptureDays is lines per UTC day of the capture timestamp — the time the
	// hook ran on this machine, not a time the harness reported.
	CaptureDays map[string]int `json:"capture_days"`

	// LastCaptureAt is the newest capture timestamp in the spool, and empty
	// when there are no envelopes.
	//
	// The maximum across every file, not the last line of the last one. Those
	// coincide only while the files are read in order and every line was
	// appended in order, and a spool synced from another machine satisfies
	// neither — so the assumption is not made. The timestamps are RFC3339 UTC
	// and fixed-width, which is what makes a string comparison the right one.
	LastCaptureAt string `json:"last_capture_at,omitempty"`

	// ConversationIDs is how many distinct non-empty conversation ids the
	// envelopes carry. Named for the field, because whether the harness starts
	// a new one per session is its business and not a claim made here.
	ConversationIDs int `json:"conversation_ids"`

	// PayloadBytes is the total size of the payloads as they are stored.
	//
	// Bytes, deliberately. The draft divided this by four and called it
	// estimated tokens, which is a claim about what a model consumed made out
	// of the size of a file. A conversation whose payloads are ten times
	// another's is a real signal about the corpus without pretending to be a
	// number anyone was billed for.
	PayloadBytes int64 `json:"payload_bytes"`

	// ObjectPayloads and OtherPayloads partition the envelopes whose payload
	// was read. Input that was not JSON at all is stored as a JSON string, on
	// purpose, and the key census cannot walk it — but it is still a payload
	// that arrived, and a summary that ignored it would report fewer payloads
	// than lines for no stated reason.
	ObjectPayloads int `json:"object_payloads"`
	OtherPayloads  int `json:"other_payloads"`

	// PayloadKeys is how many payloads carried each top-level key. This is the
	// census: it says what is in the payloads without deciding what any of it
	// means.
	PayloadKeys map[string]int `json:"payload_keys"`

	// PayloadKeyValues is the values of those keys, for the keys whose values
	// are short strings and are not content.
	//
	// Two bounds, both deliberate. Content keys — the set `--strict` replaces
	// with sizes — are counted above and never valued here, because a summary
	// is a document a user pastes into an issue and a short prompt is still a
	// prompt. Numbers are not valued either: a histogram of every number in a
	// spool is a table of measurements nobody made, which is the thing this
	// file exists not to produce.
	PayloadKeyValues map[string]map[string]int `json:"payload_key_values"`
}

// maxCensusedValueLen bounds a value the census will quote. A field name is
// short, an identifier is short, and a body of text is not: the cut is what
// keeps a summary a summary rather than a copy of the spool.
const maxCensusedValueLen = 64

// AnalyzeSpool summarises every spool file in dir.
//
// A directory that is not there is an error, not an empty spool — the same
// distinction LoadSessionEvents makes, and for the same reason: a caller who
// cannot tell them apart reports "nothing happened" for a capture that never
// ran. An existing directory with no spool files in it is an empty summary,
// because that is what it is.
func AnalyzeSpool(dir string) (SpoolSummary, error) {
	summary := SpoolSummary{
		Dir:              dir,
		SchemaVersions:   map[string]int{},
		EventNames:       map[string]int{},
		CaptureDays:      map[string]int{},
		PayloadKeys:      map[string]int{},
		PayloadKeyValues: map[string]map[string]int{},
	}
	conversationIDs := map[string]bool{}

	files, err := forEachSpoolLine(dir, func(line []byte) {
		summary.Lines++
		var ev SpoolEvent
		if json.Unmarshal(line, &ev) != nil {
			summary.UnreadableLines++
			return
		}
		summary.accumulate(ev, conversationIDs)
	})
	summary.Files = files
	summary.ConversationIDs = len(conversationIDs)
	if err != nil {
		return summary, err
	}
	return summary, nil
}

// accumulate folds one envelope into the summary.
func (s *SpoolSummary) accumulate(ev SpoolEvent, conversationIDs map[string]bool) {
	s.Envelopes++
	if ev.Strict {
		s.StrippedEnvelopes++
	}
	s.SchemaVersions[ev.SchemaVersion]++
	s.EventNames[ev.Event]++

	const dayLen = len("2006-01-02")
	if len(ev.Ts) >= dayLen {
		s.CaptureDays[ev.Ts[:dayLen]]++
	}
	if ev.Ts > s.LastCaptureAt {
		s.LastCaptureAt = ev.Ts
	}
	if ev.ConversationID != "" {
		conversationIDs[ev.ConversationID] = true
	}

	s.PayloadBytes += int64(len(ev.Raw))
	s.censusPayload(ev.Raw)
}

// censusPayload counts the top-level keys of one payload, and the values of the
// keys it is allowed to quote.
//
// The payload is decoded with the same decoder that wrote it, so a number
// arrives as the literal it was captured as rather than through a float64. That
// matters even here, where no number is reported: decoding into `any` without
// it rounds an integer above 2^53, and a census that re-encoded a payload would
// hand back a value that is not the one on disk. The capture layer went to
// trouble to avoid exactly that.
func (s *SpoolSummary) censusPayload(raw json.RawMessage) {
	decoded, isJSON := decodeOneJSONValue(raw)
	if !isJSON {
		// Unreachable for a line this package wrote — Raw is marshalled from a
		// decoded value — and counted rather than dropped for a line it did
		// not. Silence is the failure mode this summary exists to remove.
		s.OtherPayloads++
		return
	}
	obj, ok := decoded.(map[string]any)
	if !ok {
		s.OtherPayloads++
		return
	}
	s.ObjectPayloads++

	for key, value := range obj {
		s.PayloadKeys[key]++
		if !s.censusable(key, value) {
			continue
		}
		text, _ := value.(string)
		if s.PayloadKeyValues[key] == nil {
			s.PayloadKeyValues[key] = map[string]int{}
		}
		s.PayloadKeyValues[key][text]++
	}
}

// censusable reports whether one key's value may be quoted in the summary.
//
// contentKeys is asked rather than restated: it is the set the capture layer's
// strict mode replaces with sizes, so "this field carries text a user wrote" is
// decided in one place and both the writer and this reader answer to it. A
// second list here would be the copy that forgot the field somebody added.
func (s *SpoolSummary) censusable(key string, value any) bool {
	if contentKeys[strings.ToLower(key)] {
		return false
	}
	text, isString := value.(string)
	return isString && len(text) <= maxCensusedValueLen
}
