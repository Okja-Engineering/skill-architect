// Reading the spool back: the half that makes writing it worth doing.
//
// Two readers of the written corpus live here — the read of one named session,
// and the walk both it and AnalyzeSpool share — and that is the whole of what
// this file is. It was called spool_profile.go for two slices and produced no
// profile in either of them; the name is now what the file does, because a file
// named for a type it does not build is the same defect as a profile claiming a
// capability nobody captured, one layer down in the tooling.
//
// # Why there is no Profile here
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
// the day over it would break the one promise the spool makes. A caller who
// needs to know how many of those there were asks AnalyzeSpool, which counts
// them; this read is scoped to one session and a line it cannot decode is not
// one it can tell belongs.
func LoadSessionEvents(dir, sessionID string) ([]SpoolEvent, error) {
	if sessionID == "" {
		return nil, ErrSessionIDRequired
	}

	var out []SpoolEvent
	if _, err := forEachSpoolLine(dir, func(line []byte) {
		var ev SpoolEvent
		if json.Unmarshal(line, &ev) != nil {
			return
		}
		if sessionMatches(ev, sessionID) {
			out = append(out, ev)
		}
	}); err != nil {
		return nil, err
	}

	// Stable, so two events captured in the same second keep the order they
	// were written in — which within one file is the order they happened.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ts < out[j].Ts })
	return out, nil
}

// forEachSpoolLine calls visit once for every non-blank line of every spool
// file in dir, oldest file first, and reports how many spool files it read.
//
// It is the one place that decides what a spool line is, because there are now
// two readers of the same corpus — this file's session read and AnalyzeSpool's
// summary — and a second copy of this loop is two answers to "which files
// count". The draft had exactly that: one reader skipped subdirectories and the
// other did not, so the same directory held a different number of events
// depending on which function was asked.
//
// Three properties, and each of them is a decision:
//
//   - **A directory that is not there is an error**, not an empty spool. A
//     caller who cannot tell them apart reports "nothing happened" for a
//     capture that never ran.
//   - **Only `*.jsonl` files, and no subdirectories.** The spool sits under the
//     user's home and collects other things — a backup, an editor's swap file,
//     a directory somebody made — and reading those as spool lines would be a
//     reader inventing events.
//   - **Oldest file first**, which os.ReadDir's filename ordering already gives
//     because a spool file is named for its UTC day. The dependency is written
//     down rather than re-sorted, because it is what would break if the
//     directory listing were ever replaced.
//
// A line that cannot be decoded is visit's problem, not this function's: it
// hands over every line it read and lets each reader say what it does with one
// it cannot parse. Silence is what the summary exists to avoid.
func forEachSpoolLine(dir string, visit func(line []byte)) (files int, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), spoolFileSuffix) {
			names = append(names, e.Name())
		}
	}

	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return files, err
		}
		files++
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			visit([]byte(line))
		}
	}
	return files, nil
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
