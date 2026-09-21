// Registration of the ingest command in Cursor's hooks.json.
//
// hooks.json is somebody else's file: Cursor owns the format and the user owns
// the contents. This package is a guest in it. The merge is additive and
// idempotent, foreign entries and unrecognised fields are carried through
// untouched, every write is preceded by a backup, and a file that cannot be
// parsed is refused rather than replaced.
//
// # What is assumed, and what has been observed
//
// That Cursor reads ~/.cursor/hooks.json, that its `hooks` member is keyed by
// event name, that an entry is an object with a `command` string, and that the
// names in CursorHookEvents are the events Cursor invokes — all of it is taken
// from Cursor's published hooks documentation and none of it has been observed
// against a running Cursor, which was not installed on the machine where this
// was written. If the format is wrong, `hooks install` writes a file Cursor
// ignores and nothing is captured; the merge is written so that being wrong
// still cannot cost the user what was already in the file.
//
// The home directory is a parameter, never read here. A caller that has one
// passes it; only DefaultSpoolDir consults the real user's home, and only to
// compute a path.
package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CursorHookEvents is the hook surface `hooks install` registers for: the 21
// events Cursor's documentation lists. An event invented later still reaches
// the spool if it is registered by hand — this list is only what this command
// writes, not what the spool can hold.
var CursorHookEvents = []string{
	"sessionStart", "sessionEnd",
	"beforeSubmitPrompt",
	"preToolUse", "postToolUse", "postToolUseFailure",
	"beforeShellExecution", "afterShellExecution",
	"beforeMCPExecution", "afterMCPExecution",
	"beforeReadFile", "afterFileEdit",
	"beforeTabFileRead", "afterTabFileEdit",
	"subagentStart", "subagentStop",
	"afterAgentResponse", "afterAgentThought",
	"stop", "preCompact", "workspaceOpen",
}

// HookInstallResult reports what one pass changed. Backup is set only when a
// file was actually replaced: a backup named for a write that did not happen is
// a claim the result cannot support, and a directory full of them is what a
// user who reran the install would find.
type HookInstallResult struct {
	HooksJSON        string `json:"hooks_json"`
	Backup           string `json:"backup,omitempty"`
	EventsRegistered int    `json:"events_registered"`
	EventsRemoved    int    `json:"events_removed"`
}

// InstallHooks merges command into every event in CursorHookEvents in
// <home>/.cursor/hooks.json. Re-running adds nothing and writes nothing.
func InstallHooks(home, command string) (HookInstallResult, error) {
	path := hooksJSONIn(home)
	res := HookInstallResult{HooksJSON: path}

	doc, err := readHooksDoc(path)
	if err != nil {
		return res, err
	}

	hooks := hooksTable(doc.root)
	for _, event := range CursorHookEvents {
		entries, _ := hooks[event].([]any)
		if hasCommand(entries, command) {
			continue
		}
		hooks[event] = append(entries, map[string]any{"command": command})
		res.EventsRegistered++
	}
	if res.EventsRegistered == 0 {
		return res, nil
	}

	res.Backup, err = saveHooksDoc(doc, path)
	return res, err
}

// UninstallHooks removes the entries whose command is ours and leaves
// everything else — foreign entries, foreign events, unrecognised top-level
// fields. An event left with no entries is removed with them, because an empty
// registration is a trace of us in a file we are meant to have left as we found
// it. A missing file is a no-op.
func UninstallHooks(home, command string) (HookInstallResult, error) {
	path := hooksJSONIn(home)
	res := HookInstallResult{HooksJSON: path}

	doc, err := readHooksDoc(path)
	if err != nil {
		return res, err
	}
	if !doc.exists {
		return res, nil
	}

	hooks := hooksTable(doc.root)
	for event, raw := range hooks {
		entries, _ := raw.([]any)
		kept := make([]any, 0, len(entries))
		for _, e := range entries {
			if entryCommand(e) == command {
				res.EventsRemoved++
				continue
			}
			kept = append(kept, e)
		}
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}
	if res.EventsRemoved == 0 {
		return res, nil
	}

	res.Backup, err = saveHooksDoc(doc, path)
	return res, err
}

func hooksJSONIn(home string) string { return filepath.Join(home, ".cursor", "hooks.json") }

// hooksDoc is a parsed hooks.json that has kept everything this build does not
// understand. Decoding into map[string]any rather than into a struct of the
// known fields is what makes the merge safe to run against a file written by a
// newer Cursor than this one knows about.
type hooksDoc struct {
	root   map[string]any
	raw    []byte
	exists bool
}

func readHooksDoc(path string) (hooksDoc, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return hooksDoc{root: map[string]any{}}, nil
	}
	if err != nil {
		return hooksDoc{}, err
	}
	root := map[string]any{}
	if err := json.Unmarshal(raw, &root); err != nil {
		// Refused rather than replaced. A file this build cannot read is a file
		// whose contents it cannot preserve, and overwriting it would throw the
		// user's configuration away in order to install ours.
		return hooksDoc{}, fmt.Errorf("%s is not valid JSON — refusing to touch it, fix or remove it first: %w", path, err)
	}
	return hooksDoc{root: root, raw: raw, exists: true}, nil
}

// saveHooksDoc backs the file up and then writes the merged document, returning
// the backup path or "" when there was no file to preserve.
//
// Both install and uninstall go through it. The draft backed up only before an
// install, which left the more destructive of the two — removing entries from a
// user's configuration — as the one with nothing to go back to. One path, so
// the two cannot come to disagree about when a backup is taken.
func saveHooksDoc(doc hooksDoc, path string) (backup string, err error) {
	if doc.exists {
		backup = path + ".bak-" + time.Now().UTC().Format("20060102T150405Z")
		if err := os.WriteFile(backup, doc.raw, 0o600); err != nil {
			return "", fmt.Errorf("backup %s: %w", path, err)
		}
	}
	out, err := json.MarshalIndent(doc.root, "", "  ")
	if err != nil {
		return backup, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return backup, err
	}
	// Owner-only: the file names a command that runs on this machine.
	return backup, os.WriteFile(path, append(out, '\n'), 0o600)
}

// hooksTable returns doc["hooks"] as a mutable map, creating it if absent.
func hooksTable(root map[string]any) map[string]any {
	if h, ok := root["hooks"].(map[string]any); ok {
		return h
	}
	h := map[string]any{}
	root["hooks"] = h
	return h
}

// entryCommand reads the command out of one registration, answering "" for any
// entry shaped differently from the one this build writes — which is how a
// foreign entry this build cannot interpret is left alone rather than removed.
func entryCommand(entry any) string {
	em, ok := entry.(map[string]any)
	if !ok {
		return ""
	}
	cmd, _ := em["command"].(string)
	return cmd
}

func hasCommand(entries []any, command string) bool {
	for _, e := range entries {
		if entryCommand(e) == command {
			return true
		}
	}
	return false
}
