package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// hooks.json is somebody else's file. Cursor owns its format, the user owns its
// contents, and this package is a guest in it: it adds one entry per event and
// takes the same one back out. Everything below is about being a good guest —
// nothing of the user's is lost, running twice is the same as running once, and
// a file this build cannot parse is left alone rather than replaced.
//
// The *shape* of the file is documentary. That Cursor reads
// ~/.cursor/hooks.json, that the table is keyed by event name, and that an
// entry is an object with a `command` string, all come from Cursor's published
// hooks documentation; none of it has been observed against a running Cursor.
// The tests below hold this package to a merge that preserves whatever is
// there, which is the property that survives being wrong about the format.

// testHookCommand is what a registration looks like. It names a path under no
// real directory on purpose: nothing in this package's tests may resemble a
// command a real hooks.json could hold.
const testHookCommand = "/nonexistent/test/profiler ingest || true"

func hooksJSONPath(home string) string { return filepath.Join(home, ".cursor", "hooks.json") }

func TestInstall_RegistersEveryDocumentedEventOnce(t *testing.T) {
	home := sandboxHome(t)

	res, err := InstallHooks(home, testHookCommand)
	if err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}

	if res.HooksJSON != hooksJSONPath(home) {
		t.Errorf("HooksJSON = %q, want %q", res.HooksJSON, hooksJSONPath(home))
	}
	if res.EventsRegistered != len(CursorHookEvents) {
		t.Errorf("EventsRegistered = %d, want %d", res.EventsRegistered, len(CursorHookEvents))
	}
	if res.Backup != "" {
		t.Errorf("Backup = %q; there was no file to back up", res.Backup)
	}

	hooks := hooksTableOnDisk(t, home)
	if len(hooks) != len(CursorHookEvents) {
		t.Errorf("the file holds %d events, want %d", len(hooks), len(CursorHookEvents))
	}
	for _, event := range CursorHookEvents {
		entries, ok := hooks[event].([]any)
		if !ok {
			t.Errorf("event %q is missing or is not a list: %#v", event, hooks[event])
			continue
		}
		if len(entries) != 1 {
			t.Errorf("event %q has %d entries, want 1", event, len(entries))
		}
	}

	t.Run("the file is owner-only", func(t *testing.T) {
		info, err := os.Stat(res.HooksJSON)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("mode = %04o, want 0600", perm)
		}
	})
}

// TestInstall_IsIdempotent is one of the two properties the slice was asked to
// prove. Registering a hook is something a user reruns — after an upgrade, or
// because they are not sure it took — and the second run must be the same as
// the first rather than a second copy of every entry.
func TestInstall_IsIdempotent(t *testing.T) {
	home := sandboxHome(t)

	first, err := InstallHooks(home, testHookCommand)
	if err != nil {
		t.Fatalf("first InstallHooks: %v", err)
	}
	firstBytes, err := os.ReadFile(first.HooksJSON)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	second, err := InstallHooks(home, testHookCommand)
	if err != nil {
		t.Fatalf("second InstallHooks: %v", err)
	}

	t.Run("one entry per event, not two", func(t *testing.T) {
		for event, raw := range hooksTableOnDisk(t, home) {
			entries, _ := raw.([]any)
			if len(entries) != 1 {
				t.Errorf("event %q has %d entries after two installs, want 1", event, len(entries))
			}
		}
	})

	t.Run("the second run says it registered nothing", func(t *testing.T) {
		if second.EventsRegistered != 0 {
			t.Errorf("EventsRegistered = %d on the second run, want 0", second.EventsRegistered)
		}
	})

	t.Run("the second run did not rewrite the file", func(t *testing.T) {
		secondBytes, err := os.ReadFile(second.HooksJSON)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(firstBytes) != string(secondBytes) {
			t.Errorf("the file changed on a run that registered nothing:\n%s\n---\n%s", firstBytes, secondBytes)
		}
	})

	// The draft took its backup before deciding whether it had anything to
	// write, so every rerun dropped another hooks.json.bak-<timestamp> beside
	// the file. A backup of a write that did not happen is noise in the user's
	// config directory and a claim in the result that is not true.
	t.Run("a run that changes nothing leaves no backup behind", func(t *testing.T) {
		if second.Backup != "" {
			t.Errorf("Backup = %q on a run that registered nothing", second.Backup)
		}
		if n := len(backupFiles(t, home)); n != 0 {
			t.Errorf("%d backup files exist after two installs, want 0 — neither run replaced anything", n)
		}
	})
}

// TestInstall_PreservesWhatIsAlreadyThere is the other property the slice was
// asked to prove, and it is the reason the merge decodes into map[string]any
// rather than into a struct of the fields this build knows: a foreign hook, a
// foreign event, and a top-level field nobody here has heard of all survive.
func TestInstall_PreservesWhatIsAlreadyThere(t *testing.T) {
	home := sandboxHome(t)
	existing := map[string]any{
		"version":           2,
		"someFutureSetting": map[string]any{"deep": []any{"a", "b"}},
		"hooks": map[string]any{
			"sessionStart":   []any{map[string]any{"command": "/opt/someone-else/hook"}},
			"aFutureEvent":   []any{map[string]any{"command": "/opt/someone-else/other"}},
			"beforeReadFile": []any{},
		},
	}
	writeHooksFile(t, home, existing)

	res, err := InstallHooks(home, testHookCommand)
	if err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}

	root := hooksRootOnDisk(t, home)
	hooks, _ := root["hooks"].(map[string]any)

	t.Run("a foreign hook on an event we also register survives beside ours", func(t *testing.T) {
		commands := commandsFor(t, hooks, "sessionStart")
		want := []string{"/opt/someone-else/hook", testHookCommand}
		sort.Strings(want)
		if !reflect.DeepEqual(commands, want) {
			t.Errorf("sessionStart = %v, want %v", commands, want)
		}
	})

	t.Run("an event this build does not register is untouched", func(t *testing.T) {
		if got := commandsFor(t, hooks, "aFutureEvent"); !reflect.DeepEqual(got, []string{"/opt/someone-else/other"}) {
			t.Errorf("aFutureEvent = %v, want the foreign entry alone", got)
		}
	})

	t.Run("top-level fields nobody here understands survive", func(t *testing.T) {
		if got := root["version"]; got == nil {
			t.Error("the top-level `version` field was dropped")
		}
		if got, ok := root["someFutureSetting"].(map[string]any); !ok || got["deep"] == nil {
			t.Errorf("someFutureSetting = %#v, want it carried through whole", root["someFutureSetting"])
		}
	})

	t.Run("the file it replaced was backed up first", func(t *testing.T) {
		if res.Backup == "" {
			t.Fatal("an existing file was replaced with no backup taken")
		}
		var backedUp map[string]any
		readJSONFile(t, res.Backup, &backedUp)
		if !reflect.DeepEqual(backedUp, normalizedJSON(t, existing)) {
			t.Errorf("the backup is not the file as it was:\n%#v", backedUp)
		}
	})
}

// TestInstall_RefusesAHooksFileItCannotParse keeps the destructive case out.
// A hooks.json this build cannot read is a file whose contents it cannot
// preserve, and overwriting it would throw away the user's configuration to
// install ours.
func TestInstall_RefusesAHooksFileItCannotParse(t *testing.T) {
	home := sandboxHome(t)
	path := hooksJSONPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const garbage = "{ this is not JSON, it may be a comment-laden config }"
	if err := os.WriteFile(path, []byte(garbage), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, tc := range []struct {
		name string
		run  func() (HookInstallResult, error)
	}{
		{"install", func() (HookInstallResult, error) { return InstallHooks(home, testHookCommand) }},
		{"uninstall", func() (HookInstallResult, error) { return UninstallHooks(home, testHookCommand) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.run()
			if err == nil {
				t.Fatal("a file that could not be parsed was accepted")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error = %q, want it to name the file the user has to fix", err)
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read: %v", readErr)
			}
			if string(body) != garbage {
				t.Errorf("the file was modified after the refusal:\n%s", body)
			}
		})
	}
}

func TestUninstall_RemovesOnlyOurEntries(t *testing.T) {
	home := sandboxHome(t)
	writeHooksFile(t, home, map[string]any{
		"version": 2,
		"hooks": map[string]any{
			"sessionStart": []any{
				map[string]any{"command": "/opt/someone-else/hook"},
				map[string]any{"command": testHookCommand},
			},
			"aFutureEvent": []any{map[string]any{"command": "/opt/someone-else/other"}},
		},
	})

	res, err := UninstallHooks(home, testHookCommand)
	if err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}
	if res.EventsRemoved != 1 {
		t.Errorf("EventsRemoved = %d, want 1", res.EventsRemoved)
	}

	root := hooksRootOnDisk(t, home)
	hooks, _ := root["hooks"].(map[string]any)

	if got := commandsFor(t, hooks, "sessionStart"); !reflect.DeepEqual(got, []string{"/opt/someone-else/hook"}) {
		t.Errorf("sessionStart = %v, want the foreign entry left alone", got)
	}
	if got := commandsFor(t, hooks, "aFutureEvent"); !reflect.DeepEqual(got, []string{"/opt/someone-else/other"}) {
		t.Errorf("aFutureEvent = %v, want it untouched", got)
	}
	if root["version"] == nil {
		t.Error("the top-level `version` field was dropped by the uninstall")
	}

	// Removing entries from somebody's configuration is the more destructive of
	// the two operations, and the draft was the one that took no backup for it.
	t.Run("the file it rewrote was backed up first", func(t *testing.T) {
		if res.Backup == "" {
			t.Fatal("entries were removed from the user's file with no backup taken")
		}
		var backedUp map[string]any
		readJSONFile(t, res.Backup, &backedUp)
		hooksBefore, _ := backedUp["hooks"].(map[string]any)
		if got := commandsFor(t, hooksBefore, "sessionStart"); len(got) != 2 {
			t.Errorf("the backup holds %v, want the file as it was before the removal", got)
		}
	})
}

// TestUninstall_LeavesNoEmptyEventBehind. An event key whose list is empty is a
// registration that is not one; leaving it would make our uninstall visible in
// a file we are supposed to have left as we found it.
func TestUninstall_LeavesNoEmptyEventBehind(t *testing.T) {
	home := sandboxHome(t)
	if _, err := InstallHooks(home, testHookCommand); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}
	if _, err := UninstallHooks(home, testHookCommand); err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}

	hooks := hooksTableOnDisk(t, home)
	if len(hooks) != 0 {
		t.Errorf("the hooks table still holds %d events after an uninstall: %#v", len(hooks), hooks)
	}
}

func TestUninstall_IsANoopWhenNothingIsRegistered(t *testing.T) {
	t.Run("no file at all", func(t *testing.T) {
		home := sandboxHome(t)
		res, err := UninstallHooks(home, testHookCommand)
		if err != nil {
			t.Fatalf("UninstallHooks: %v", err)
		}
		if res.EventsRemoved != 0 || res.Backup != "" {
			t.Errorf("result = %+v, want nothing removed and no backup", res)
		}
		if _, err := os.Stat(res.HooksJSON); err == nil {
			t.Error("the uninstall created a hooks.json that was not there")
		}
	})

	t.Run("a file holding only somebody else's hooks", func(t *testing.T) {
		home := sandboxHome(t)
		writeHooksFile(t, home, map[string]any{
			"hooks": map[string]any{"sessionStart": []any{map[string]any{"command": "/opt/someone-else/hook"}}},
		})
		before, err := os.ReadFile(hooksJSONPath(home))
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		res, err := UninstallHooks(home, testHookCommand)
		if err != nil {
			t.Fatalf("UninstallHooks: %v", err)
		}
		if res.EventsRemoved != 0 || res.Backup != "" {
			t.Errorf("result = %+v, want nothing removed and no backup", res)
		}
		after, err := os.ReadFile(hooksJSONPath(home))
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("a file with none of our entries was rewritten:\n%s\n---\n%s", before, after)
		}
		if n := len(backupFiles(t, home)); n != 0 {
			t.Errorf("%d backups were left by a run that changed nothing", n)
		}
	})
}

// TestInstallUninstall_RoundTripLeavesTheFileAsItWas is the guest property
// stated end to end: after installing and uninstalling, what the user had is
// what the user has.
func TestInstallUninstall_RoundTripLeavesTheFileAsItWas(t *testing.T) {
	home := sandboxHome(t)
	existing := map[string]any{
		"version": 2,
		"hooks": map[string]any{
			"sessionStart": []any{map[string]any{"command": "/opt/someone-else/hook"}},
			"aFutureEvent": []any{map[string]any{"command": "/opt/someone-else/other"}},
		},
	}
	writeHooksFile(t, home, existing)

	if _, err := InstallHooks(home, testHookCommand); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}
	if _, err := UninstallHooks(home, testHookCommand); err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}

	if got := hooksRootOnDisk(t, home); !reflect.DeepEqual(got, normalizedJSON(t, existing)) {
		t.Errorf("after install then uninstall the file reads\n%#v\nwant\n%#v", got, normalizedJSON(t, existing))
	}
}

// TestHooks_LeaveAloneAnEntryShapedUnlikeOurs. An entry that is not an object
// with a `command` string is one this build cannot interpret — a shorthand a
// later Cursor allows, or something the user wrote by hand. It is not ours, so
// install must not treat it as already-installed and uninstall must not remove
// it.
func TestHooks_LeaveAloneAnEntryShapedUnlikeOurs(t *testing.T) {
	home := sandboxHome(t)
	writeHooksFile(t, home, map[string]any{
		"hooks": map[string]any{
			"sessionStart": []any{"a bare string entry", map[string]any{"notCommand": 1}},
		},
	})

	if _, err := InstallHooks(home, testHookCommand); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}
	entries, _ := hooksTableOnDisk(t, home)["sessionStart"].([]any)
	if len(entries) != 3 {
		t.Errorf("sessionStart holds %d entries after install, want the two foreign ones plus ours: %#v", len(entries), entries)
	}

	if _, err := UninstallHooks(home, testHookCommand); err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}
	entries, _ = hooksTableOnDisk(t, home)["sessionStart"].([]any)
	if len(entries) != 2 {
		t.Errorf("sessionStart holds %d entries after uninstall, want the two foreign ones: %#v", len(entries), entries)
	}
}

// TestHooks_MatchTheCommandByTypeAndNotByItsRendering. The command a user names
// is a string, and an entry whose `command` is a number that reads the same is
// not the same entry. Comparing a rendering rather than a value would let this
// build remove somebody else's registration because the two look alike printed.
// Reachable because --command takes whatever the caller gives it.
//
// (M33 survived until this existed: the mutation replaced the type assertion
// with fmt.Sprint and nothing noticed.)
func TestHooks_MatchTheCommandByTypeAndNotByItsRendering(t *testing.T) {
	home := sandboxHome(t)
	const numericLooking = "42"
	writeHooksFile(t, home, map[string]any{
		"hooks": map[string]any{
			"sessionStart": []any{map[string]any{"command": 42}},
		},
	})

	// Install must not read the number as ours already being there.
	res, err := InstallHooks(home, numericLooking)
	if err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}
	if res.EventsRegistered != len(CursorHookEvents) {
		t.Errorf("EventsRegistered = %d, want %d — the numeric entry was read as ours", res.EventsRegistered, len(CursorHookEvents))
	}

	// Uninstall must take ours back out and leave the number alone.
	if _, err := UninstallHooks(home, numericLooking); err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}
	entries, _ := hooksTableOnDisk(t, home)["sessionStart"].([]any)
	if len(entries) != 1 {
		t.Fatalf("sessionStart holds %d entries, want the foreign numeric one: %#v", len(entries), entries)
	}
	em, _ := entries[0].(map[string]any)
	if _, isString := em["command"].(string); isString {
		t.Errorf("the surviving entry is %#v; the numeric command was rewritten", em)
	}
}

// TestInstall_SaysSoWhenItCannotBackUpAndWritesNothing. The backup is the only
// thing standing between a merge and a user's configuration, so a backup that
// could not be taken has to stop the write rather than be skipped.
func TestInstall_SaysSoWhenItCannotBackUpAndWritesNothing(t *testing.T) {
	home := sandboxHome(t)
	writeHooksFile(t, home, map[string]any{
		"hooks": map[string]any{"sessionStart": []any{map[string]any{"command": "/opt/someone-else/hook"}}},
	})
	before, err := os.ReadFile(hooksJSONPath(home))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	cursorDir := filepath.Dir(hooksJSONPath(home))
	if err := os.Chmod(cursorDir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(cursorDir, 0o700) })
	if probe := filepath.Join(cursorDir, "probe"); os.WriteFile(probe, nil, 0o600) == nil {
		_ = os.Remove(probe)
		t.Skip("this filesystem allowed a write into a read-only directory; the case is not reachable here")
	}

	if _, err := InstallHooks(home, testHookCommand); err == nil {
		t.Error("InstallHooks reported success when it could not take a backup")
	}

	after, err := os.ReadFile(hooksJSONPath(home))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("the file was modified even though the backup failed:\n%s\n---\n%s", before, after)
	}
}

// TestHooks_SurfaceAReadErrorThatIsNotAMissingFile. A missing hooks.json means
// "nothing is registered"; every other read failure means "this build does not
// know what is in the file", and the two must not be answered the same way.
func TestHooks_SurfaceAReadErrorThatIsNotAMissingFile(t *testing.T) {
	home := sandboxHome(t)
	// A directory where the file belongs: readable as an entry, unreadable as
	// a file, and not os.IsNotExist.
	if err := os.MkdirAll(hooksJSONPath(home), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := InstallHooks(home, testHookCommand); err == nil {
		t.Error("InstallHooks read a directory as an empty hooks.json")
	}
	if _, err := UninstallHooks(home, testHookCommand); err == nil {
		t.Error("UninstallHooks read a directory as a missing hooks.json")
	}
}

// TestCursorHookEvents_AreDistinct. The list is what `hooks install` writes; a
// duplicate in it would register two identical entries for one event on the
// first run and make the idempotence check above pass for the wrong reason.
func TestCursorHookEvents_AreDistinct(t *testing.T) {
	if len(CursorHookEvents) == 0 {
		t.Fatal("the event list is empty, so every check over it reads nothing")
	}
	seen := map[string]bool{}
	for _, e := range CursorHookEvents {
		if e == "" {
			t.Error("the event list holds an empty name")
		}
		if seen[e] {
			t.Errorf("%q appears twice in the event list", e)
		}
		seen[e] = true
	}
}

// --- helpers ---

func writeHooksFile(t *testing.T, home string, doc map[string]any) {
	t.Helper()
	path := hooksJSONPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func hooksRootOnDisk(t *testing.T, home string) map[string]any {
	t.Helper()
	var root map[string]any
	readJSONFile(t, hooksJSONPath(home), &root)
	return root
}

func hooksTableOnDisk(t *testing.T, home string) map[string]any {
	t.Helper()
	hooks, ok := hooksRootOnDisk(t, home)["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("the file on disk has no `hooks` object")
	}
	return hooks
}

func readJSONFile(t *testing.T, path string, into any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("%s is not JSON: %v\n%s", path, err, body)
	}
}

// commandsFor returns the commands registered for one event, sorted, so a case
// asserts the set that is there rather than the order a map iteration produced.
func commandsFor(t *testing.T, hooks map[string]any, event string) []string {
	t.Helper()
	entries, ok := hooks[event].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, e := range entries {
		em, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("entry under %q is not an object: %#v", event, e)
		}
		cmd, _ := em["command"].(string)
		out = append(out, cmd)
	}
	sort.Strings(out)
	return out
}

func backupFiles(t *testing.T, home string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, ".cursor"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var out []string
	for _, e := range entries {
		if strings.Contains(e.Name(), ".bak-") {
			out = append(out, e.Name())
		}
	}
	return out
}

// normalizedJSON returns v as it reads back off disk, so a comparison is
// between two decoded documents rather than between a Go literal and a decoded
// one — where every number would differ by being an int on one side and a
// float64 on the other.
func normalizedJSON(t *testing.T, v any) map[string]any {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
