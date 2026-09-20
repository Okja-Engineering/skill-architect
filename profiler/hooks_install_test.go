package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hooks install/uninstall — merge our ingest command into ~/.cursor/hooks.json
// without clobbering foreign entries, with backup and clean removal.

const testHookCmd = "/usr/local/bin/profiler ingest || true"

func readHooksFile(t *testing.T, home string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("hooks.json is not valid JSON: %v", err)
	}
	return m
}

func hookCommands(t *testing.T, m map[string]any, event string) []string {
	t.Helper()
	hooks, _ := m["hooks"].(map[string]any)
	entries, _ := hooks[event].([]any)
	var cmds []string
	for _, e := range entries {
		if em, ok := e.(map[string]any); ok {
			if c, ok := em["command"].(string); ok {
				cmds = append(cmds, c)
			}
		}
	}
	return cmds
}

func countOurEntries(t *testing.T, m map[string]any) int {
	t.Helper()
	n := 0
	hooks, _ := m["hooks"].(map[string]any)
	for _, entries := range hooks {
		for _, e := range entries.([]any) {
			if em, ok := e.(map[string]any); ok {
				if c, _ := em["command"].(string); strings.Contains(c, "profiler ingest") {
					n++
				}
			}
		}
	}
	return n
}

func TestInstall_CreatesHooksJSON(t *testing.T) {
	home := t.TempDir()
	res, err := InstallHooks(home, testHookCmd)
	if err != nil {
		t.Fatal(err)
	}
	m := readHooksFile(t, home)
	if got := hookCommands(t, m, "sessionStart"); len(got) != 1 || got[0] != testHookCmd {
		t.Errorf("sessionStart commands = %v, want [%q]", got, testHookCmd)
	}
	if res.EventsRegistered != len(CursorHookEvents) {
		t.Errorf("events_registered = %d, want %d", res.EventsRegistered, len(CursorHookEvents))
	}
}

func TestInstall_RegistersAllDocumentedEvents(t *testing.T) {
	home := t.TempDir()
	if _, err := InstallHooks(home, testHookCmd); err != nil {
		t.Fatal(err)
	}
	m := readHooksFile(t, home)
	hooks, _ := m["hooks"].(map[string]any)
	for _, event := range CursorHookEvents {
		cmds := hookCommands(t, m, event)
		found := false
		for _, c := range cmds {
			if c == testHookCmd {
				found = true
			}
		}
		if !found {
			t.Errorf("event %q missing our command", event)
		}
	}
	// No invented events beyond the documented set.
	for event := range hooks {
		known := false
		for _, e := range CursorHookEvents {
			if event == e {
				known = true
			}
		}
		if !known {
			t.Errorf("invented event %q in hooks.json", event)
		}
	}
}

func TestInstall_PreservesForeignHooks(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".cursor"), 0755)
	existing := `{"hooks": {"stop": [{"command": "my-formatter --fix"}], "sessionStart": [{"command": "other-tool"}]}, "customField": 42}`
	os.WriteFile(filepath.Join(home, ".cursor", "hooks.json"), []byte(existing), 0644)

	if _, err := InstallHooks(home, testHookCmd); err != nil {
		t.Fatal(err)
	}
	m := readHooksFile(t, home)

	stopCmds := hookCommands(t, m, "stop")
	found := false
	for _, c := range stopCmds {
		if c == "my-formatter --fix" {
			found = true
		}
	}
	if !found {
		t.Errorf("foreign 'stop' hook lost: %v", stopCmds)
	}
	if m["customField"] != float64(42) {
		t.Errorf("top-level custom field lost: %v", m["customField"])
	}
}

func TestInstall_Idempotent(t *testing.T) {
	home := t.TempDir()
	if _, err := InstallHooks(home, testHookCmd); err != nil {
		t.Fatal(err)
	}
	res, err := InstallHooks(home, testHookCmd)
	if err != nil {
		t.Fatal(err)
	}
	m := readHooksFile(t, home)
	if got := countOurEntries(t, m); got != len(CursorHookEvents) {
		t.Errorf("our entries = %d after double install, want %d", got, len(CursorHookEvents))
	}
	if res.EventsRegistered != 0 {
		t.Errorf("second install registered %d events, want 0 (already present)", res.EventsRegistered)
	}
}

func TestInstall_BacksUp(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".cursor"), 0755)
	original := `{"hooks": {"stop": [{"command": "mine"}]}}`
	path := filepath.Join(home, ".cursor", "hooks.json")
	os.WriteFile(path, []byte(original), 0644)

	res, err := InstallHooks(home, testHookCmd)
	if err != nil {
		t.Fatal(err)
	}
	if res.Backup == "" {
		t.Fatal("no backup path reported")
	}
	b, err := os.ReadFile(res.Backup)
	if err != nil {
		t.Fatalf("backup unreadable: %v", err)
	}
	if string(b) != original {
		t.Errorf("backup content = %q, want original %q", string(b), original)
	}
}

func TestUninstall_RemovesOnlyOurs(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".cursor"), 0755)
	existing := `{"hooks": {"stop": [{"command": "my-formatter"}, {"command": "` + testHookCmd + `"}], "sessionStart": [{"command": "` + testHookCmd + `"}]}}`
	os.WriteFile(filepath.Join(home, ".cursor", "hooks.json"), []byte(existing), 0644)

	res, err := UninstallHooks(home, testHookCmd)
	if err != nil {
		t.Fatal(err)
	}
	m := readHooksFile(t, home)

	if got := countOurEntries(t, m); got != 0 {
		t.Errorf("our entries remaining = %d, want 0", got)
	}
	stopCmds := hookCommands(t, m, "stop")
	if len(stopCmds) != 1 || stopCmds[0] != "my-formatter" {
		t.Errorf("foreign stop hook = %v, want [my-formatter]", stopCmds)
	}
	if res.EventsRemoved != 2 {
		t.Errorf("events_removed = %d, want 2", res.EventsRemoved)
	}
}

func TestUninstall_MissingFileIsNoop(t *testing.T) {
	res, err := UninstallHooks(t.TempDir(), testHookCmd)
	if err != nil {
		t.Fatalf("uninstall on missing file should not error: %v", err)
	}
	if res.EventsRemoved != 0 {
		t.Errorf("events_removed = %d, want 0", res.EventsRemoved)
	}
}
