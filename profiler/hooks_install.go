// Registration of cursor-profiler's ingest command in ~/.cursor/hooks.json.
// The merge is additive and idempotent: foreign hooks and unknown top-level
// fields are preserved, an existing file is backed up first, and uninstall
// removes only our entries.
package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CursorHookEvents is the documented hook surface — 21 events per Cursor's
// hooks spec. Unknown future events still reach the spool if registered
// manually; this list is only what `hooks install` writes.
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

// HookInstallResult reports what an install/uninstall pass changed.
type HookInstallResult struct {
	HooksJSON        string `json:"hooks_json"`
	Backup           string `json:"backup,omitempty"`
	EventsRegistered int    `json:"events_registered"`
	EventsRemoved    int    `json:"events_removed"`
}

// InstallHooks merges command into every documented event in
// ~/.cursor/hooks.json. A pre-existing file is backed up to
// hooks.json.bak-<timestamp> before any write. Re-running adds nothing.
func InstallHooks(home, command string) (HookInstallResult, error) {
	path := filepath.Join(home, ".cursor", "hooks.json")
	res := HookInstallResult{HooksJSON: path}

	doc, err := readHooksDoc(path)
	if err != nil {
		return res, err
	}
	if doc.exists {
		res.Backup = path + ".bak-" + time.Now().UTC().Format("20060102T150405Z")
		if err := os.WriteFile(res.Backup, doc.raw, 0600); err != nil {
			return res, fmt.Errorf("backup: %w", err)
		}
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
		return res, nil // already installed; no write, no churn
	}
	return res, writeHooksDoc(path, doc.root)
}

// UninstallHooks removes entries whose command matches ours. Foreign entries
// and unknown fields are left untouched; a missing file is a no-op.
func UninstallHooks(home, command string) (HookInstallResult, error) {
	path := filepath.Join(home, ".cursor", "hooks.json")
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
		kept := entries[:0]
		for _, e := range entries {
			if em, ok := e.(map[string]any); ok && em["command"] == command {
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
	return res, writeHooksDoc(path, doc.root)
}

// hooksDoc is a parsed hooks.json preserving everything we don't understand.
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
		return hooksDoc{}, fmt.Errorf("%s is not valid JSON — refusing to touch it (fix or remove it first): %w", path, err)
	}
	return hooksDoc{root: root, raw: raw, exists: true}, nil
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

func hasCommand(entries []any, command string) bool {
	for _, e := range entries {
		if em, ok := e.(map[string]any); ok && em["command"] == command {
			return true
		}
	}
	return false
}

func writeHooksDoc(path string, root map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0600)
}
