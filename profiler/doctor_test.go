package profiler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Environment detection — the "Modernizr" layer. Detect() inspects the machine
// and reports which telemetry surfaces are reachable; the selected tier is the
// strategy the rest of the tool should use.

func seedHome(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestDetect_BareMachine(t *testing.T) {
	home := seedHome(t)
	rep := DetectEnvironment(home, func(string) string { return "" })

	if rep.CursorInstalled {
		t.Error("cursor_installed = true, want false")
	}
	if rep.HooksRegistered {
		t.Error("hooks_registered = true, want false")
	}
	if rep.Tier != TierNone {
		t.Errorf("tier = %q, want %q", rep.Tier, TierNone)
	}
}

func TestDetect_CursorInstalled(t *testing.T) {
	home := seedHome(t)
	// Cursor's user-data dir existing means Cursor has run here.
	os.MkdirAll(filepath.Join(home, "Library", "Application Support", "Cursor"), 0755)

	rep := DetectEnvironment(home, func(string) string { return "" })
	if !rep.CursorInstalled {
		t.Error("cursor_installed = false, want true")
	}
}

func TestDetect_HooksRegistered(t *testing.T) {
	home := seedHome(t)
	cursorDir := filepath.Join(home, ".cursor")
	os.MkdirAll(cursorDir, 0755)
	hooksJSON := `{"hooks": {"beforeSubmitPrompt": [{"command": "profiler ingest || true"}]}}`
	os.WriteFile(filepath.Join(cursorDir, "hooks.json"), []byte(hooksJSON), 0644)

	rep := DetectEnvironment(home, func(string) string { return "" })
	if !rep.HooksRegistered {
		t.Error("hooks_registered = false, want true")
	}
	if rep.Tier != TierHooks {
		t.Errorf("tier = %q, want %q", rep.Tier, TierHooks)
	}
}

func TestDetect_HooksJSONWithoutUs(t *testing.T) {
	home := seedHome(t)
	cursorDir := filepath.Join(home, ".cursor")
	os.MkdirAll(cursorDir, 0755)
	// Someone else's hook — must not count as ours.
	os.WriteFile(filepath.Join(cursorDir, "hooks.json"),
		[]byte(`{"hooks": {"stop": [{"command": "my-own-thing"}]}}`), 0644)

	rep := DetectEnvironment(home, func(string) string { return "" })
	if rep.HooksRegistered {
		t.Error("hooks_registered = true for foreign hooks.json, want false")
	}
}

func TestDetect_SpoolStats(t *testing.T) {
	home := seedHome(t)
	spool := filepath.Join(home, ".cursor-profiler", "spool")
	os.MkdirAll(spool, 0755)
	old := `{"ts":"2026-09-10T01:00:00Z","event":"sessionStart","raw":{}}` + "\n"
	newer := `{"ts":"2026-09-11T02:00:00Z","event":"stop","raw":{}}` + "\n"
	os.WriteFile(filepath.Join(spool, "2026-09-10.jsonl"), []byte(old), 0600)
	os.WriteFile(filepath.Join(spool, "2026-09-11.jsonl"), []byte(newer), 0600)

	rep := DetectEnvironment(home, func(string) string { return "" })
	if rep.SpoolFiles != 2 {
		t.Errorf("spool_files = %d, want 2", rep.SpoolFiles)
	}
	if rep.SpoolBytes != int64(len(old)+len(newer)) {
		t.Errorf("spool_bytes = %d, want %d", rep.SpoolBytes, len(old)+len(newer))
	}
	if rep.LastEvent != "2026-09-11T02:00:00Z" {
		t.Errorf("last_event = %q, want 2026-09-11T02:00:00Z", rep.LastEvent)
	}
	// Spool data implies capture is (or was) running → hooks tier even without
	// a registered hooks.json (e.g. spool synced from another machine).
	if rep.Tier != TierHooks {
		t.Errorf("tier = %q, want %q", rep.Tier, TierHooks)
	}
}

func TestDetect_AdminKeyUpgradesTier(t *testing.T) {
	home := seedHome(t)
	getenv := func(k string) string {
		if k == "CURSOR_ADMIN_API_KEY" {
			return "key-123"
		}
		return ""
	}
	rep := DetectEnvironment(home, getenv)
	if !rep.AdminAPIKey {
		t.Error("admin_api_key = false, want true")
	}
	if rep.Tier != TierServerAPI {
		t.Errorf("tier = %q, want %q", rep.Tier, TierServerAPI)
	}
}

func TestDetect_OTelUpgradesTier(t *testing.T) {
	home := seedHome(t)
	getenv := func(k string) string {
		if k == "OTEL_EXPORTER_OTLP_ENDPOINT" {
			return "http://localhost:4318"
		}
		return ""
	}
	rep := DetectEnvironment(home, getenv)
	if !rep.OTelConfigured {
		t.Error("otel_configured = false, want true")
	}
	if rep.Tier != TierEnterprise {
		t.Errorf("tier = %q, want %q", rep.Tier, TierEnterprise)
	}
}

func TestDetect_StrongestTierWins(t *testing.T) {
	home := seedHome(t)
	cursorDir := filepath.Join(home, ".cursor")
	os.MkdirAll(cursorDir, 0755)
	os.WriteFile(filepath.Join(cursorDir, "hooks.json"),
		[]byte(`{"hooks": {"stop": [{"command": "profiler ingest || true"}]}}`), 0644)
	getenv := func(k string) string {
		switch k {
		case "CURSOR_ADMIN_API_KEY":
			return "key-123"
		case "OTEL_EXPORTER_OTLP_ENDPOINT":
			return "http://x:4318"
		}
		return ""
	}
	rep := DetectEnvironment(home, getenv)
	if rep.Tier != TierEnterprise {
		t.Errorf("tier = %q, want %q (strongest surface wins)", rep.Tier, TierEnterprise)
	}
}

// Doctor output stays honest: detection timestamps itself so a stale report is
// visibly stale.
func TestDetect_Timestamps(t *testing.T) {
	rep := DetectEnvironment(seedHome(t), func(string) string { return "" })
	if rep.DetectedAt == "" {
		t.Error("detected_at empty")
	}
	if _, err := time.Parse(time.RFC3339, rep.DetectedAt); err != nil {
		t.Errorf("detected_at %q not RFC3339: %v", rep.DetectedAt, err)
	}
}
