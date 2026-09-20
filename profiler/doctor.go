// Environment detection — the "Modernizr" layer. DetectEnvironment inspects
// the machine and reports which telemetry surfaces are reachable; the selected
// Tier is the strategy the rest of the tool should use. Surfaces are strictly
// additive: hooks are the baseline, Admin API and OTel upgrade specific metrics.
package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnvironmentTier names the strongest reachable telemetry surface.
type EnvironmentTier string

const (
	TierNone       EnvironmentTier = "none"       // nothing detected
	TierHooks      EnvironmentTier = "hooks"      // hook capture available/active
	TierServerAPI  EnvironmentTier = "server_api" // hooks + Admin API key configured
	TierEnterprise EnvironmentTier = "enterprise" // OTel export configured
)

// EnvironmentReport is the doctor output: what this machine can provide.
type EnvironmentReport struct {
	DetectedAt      string          `json:"detected_at"`
	Tier            EnvironmentTier `json:"tier"`
	CursorInstalled bool            `json:"cursor_installed"`
	HooksRegistered bool            `json:"hooks_registered"`
	SpoolDir        string          `json:"spool_dir"`
	SpoolFiles      int             `json:"spool_files"`
	SpoolBytes      int64           `json:"spool_bytes"`
	LastEvent       string          `json:"last_event,omitempty"`
	AdminAPIKey     bool            `json:"admin_api_key"`   // configured, not validated
	OTelConfigured  bool            `json:"otel_configured"` // configured, not validated
}

// DetectEnvironment inspects home (the user's home directory) and getenv (for
// env-var configuration, e.g. os.Getenv) and reports the reachable surfaces.
// It never fails — detection is a report, not a gate.
func DetectEnvironment(home string, getenv func(string) string) EnvironmentReport {
	rep := EnvironmentReport{
		DetectedAt: time.Now().UTC().Format(time.RFC3339),
		Tier:       TierNone,
		SpoolDir:   filepath.Join(home, ".cursor-profiler", "spool"),
	}

	// Cursor installed: its user-data dir exists (macOS path; the hook corpus
	// machine is a Mac).
	if _, err := os.Stat(filepath.Join(home, "Library", "Application Support", "Cursor")); err == nil {
		rep.CursorInstalled = true
	}

	// Our hook registered: hooks.json mentions our ingest command. A string
	// match on the file is deliberate — a foreign hooks.json must not count,
	// and a partial/broken JSON file still reveals the registration.
	if data, err := os.ReadFile(filepath.Join(home, ".cursor", "hooks.json")); err == nil {
		s := string(data)
		if strings.Contains(s, "profiler ingest") || strings.Contains(s, "cursor-profiler ingest") {
			rep.HooksRegistered = true
		}
	}

	// Spool state: files, bytes, most recent event timestamp.
	if entries, err := os.ReadDir(rep.SpoolDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			rep.SpoolFiles++
			rep.SpoolBytes += info.Size()
			if last := lastSpoolTs(filepath.Join(rep.SpoolDir, e.Name())); last > rep.LastEvent {
				rep.LastEvent = last
			}
		}
	}

	rep.AdminAPIKey = getenv("CURSOR_ADMIN_API_KEY") != ""
	rep.OTelConfigured = getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""

	// Strongest reachable surface wins.
	switch {
	case rep.OTelConfigured:
		rep.Tier = TierEnterprise
	case rep.AdminAPIKey:
		rep.Tier = TierServerAPI
	case rep.HooksRegistered || rep.SpoolFiles > 0:
		rep.Tier = TierHooks
	}
	return rep
}

// lastSpoolTs returns the largest "ts" value in a spool file. Lines are
// append-ordered by capture time, so scanning is sufficient and cheap at
// expected file sizes.
func lastSpoolTs(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	last := ""
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		var ev struct {
			Ts string `json:"ts"`
		}
		if json.Unmarshal([]byte(line), &ev) == nil && ev.Ts > last {
			last = ev.Ts
		}
	}
	return last
}
