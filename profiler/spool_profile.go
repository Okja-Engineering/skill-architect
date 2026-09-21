// Reading the spool back: the half that makes writing it worth doing.
//
// # Why there is no Profile in this file
//
// The draft this is ported from went further: it paired preToolUse with
// postToolUse on tool_use_id, inferred a skill activation from a read of a
// SKILL.md, attributed each tool call to the most recent activation before it,
// and returned a Profile with `tool_calls: present, source: hooks`.
//
// None of that lands here, and not because it is unfinished. Every one of those
// steps is a claim about the shape of a payload nobody in this repository has
// observed — Cursor was not installed on the machine where this was written —
// and a Profile reporting `present` is that claim made in the data rather than
// in the prose. A capability a capture cannot support is the defect that kept
// three adapters out of this release; a profile built from guessed field names
// would be the same defect one layer down, and it would travel: a stored
// profile is compared, subtracted and reported months later by callers who
// never see this file.
//
// What is here instead is the part that is true regardless of whether the
// guesses are right: the lines that were written can be read back, whole, for
// one named session. If the promoted fields turn out to be wrong, the payloads
// are still on disk and a better reader works from the same files. That
// property is the reason the capture layer ships at all, so it is the property
// this file provides and the only one it claims.
//
// The projection to a Profile belongs to the slice that can pass the adapter
// contract — which means the slice where somebody has run Cursor and looked at
// what it actually sends.
package profiler

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// spoolFileSuffix is the extension AppendSpool writes. The spool directory
// lives under the user's home and collects other things — a backup, an editor's
// swap file, a subdirectory somebody made — and reading those as spool lines
// would be a reader inventing events.
const spoolFileSuffix = ".jsonl"

// ErrSessionIDRequired is the refusal for a read with no session named.
//
// An empty id is not a session that matched nothing. Matched loosely it selects
// every event that happens to carry no conversation id, which hands back a
// whole machine's spool as if it were one session — the session-scoping defect
// this release spent four versions closing, and the one each unlanded adapter
// reproduced. The same rule as SessionIDRequiredError, stated for a read rather
// than for a Capture, because there is no adapter here to name.
var ErrSessionIDRequired = errors.New("a session id is required: a read returns the events carrying one session, so an empty id is no assertion rather than a session that matched nothing")

// LoadSessionEvents returns the spool events belonging to one session, ordered
// by capture time across every daily file in dir.
//
// A session is named by its conversation id, its session id or its generation
// id; which of the three a given hook payload carries is not something this
// build has observed, so all three are matched rather than one being chosen.
//
// A line that cannot be decoded is skipped and the rest of the file is read: a
// hook killed mid-write leaves a truncated last line, and losing the rest of
// the day over it would break the one promise the spool makes.
func LoadSessionEvents(dir, sessionID string) ([]SpoolEvent, error) {
	if sessionID == "" {
		return nil, ErrSessionIDRequired
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		// A spool that was never written is not a session with no events, and
		// a caller who cannot tell them apart will report "nothing happened"
		// for a capture that never ran.
		return nil, err
	}

	// os.ReadDir returns its entries sorted by filename, and a spool file is
	// named for its UTC day — so the files are already read oldest-first. A
	// second sort here was in the draft and cannot change anything; the
	// dependency is written down instead, because it is the thing that would
	// break if the directory listing were ever replaced.
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), spoolFileSuffix) {
			names = append(names, e.Name())
		}
	}

	var out []SpoolEvent
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev SpoolEvent
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			if sessionMatches(ev, sessionID) {
				out = append(out, ev)
			}
		}
	}

	// Stable, so two events captured in the same second keep the order they
	// were written in — which within one file is the order they happened.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ts < out[j].Ts })
	return out, nil
}

// sessionMatches reports whether one event belongs to the named session. The
// envelope's promoted conversation id is checked first because it needs no
// decode; the payload's own ids are checked after, because which of the three a
// hook carries is a documentary assumption and not a measured fact.
//
// sessionID is never empty here: LoadSessionEvents is the only caller and
// refuses one. That matters, because every comparison below is an equality
// against a field that is "" when absent — so an empty id would match every
// event that names no session, and the refusal is what stands between this
// function and that. A failure to decode Raw leaves the map nil, whose fields
// read as "" and therefore match nothing; the draft's explicit guard for it
// could not change an outcome and is gone.
func sessionMatches(ev SpoolEvent, sessionID string) bool {
	if ev.ConversationID == sessionID {
		return true
	}
	var raw map[string]any
	_ = json.Unmarshal(ev.Raw, &raw)
	return getString(raw, "session_id") == sessionID || getString(raw, "generation_id") == sessionID
}
