package profiler

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// DevinAdapter captures runtime signals from Devin via ATIF export or server API.
//
// Devin is cloud-based and does not expose OTel or hooks. The adapter recognises an
// ATIF export file but does not yet parse the schema; it reports the capability as
// `session_data` and honestly returns `unknown` until the ATIF structure is mapped.
// The server API path is a placeholder for future enterprise support.
type DevinAdapter struct {
	// ExportFile is the path to an ATIF export or transcript file.
	ExportFile string
}

// Name returns the harness identifier.
func (a DevinAdapter) Name() string { return "devin" }

// Probe inspects the environment and returns what metrics this adapter can produce.
func (a DevinAdapter) Probe() CapabilityReport {
	caps := map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}

	if a.ExportFile != "" {
		if _, err := os.Stat(a.ExportFile); err == nil {
			if hasATIFSignals(a.ExportFile) {
				caps[MetricTokens] = SourceSessionData
				caps[MetricToolCalls] = SourceSessionData
				caps[MetricTiming] = SourceSessionData
			}
		}
	}

	return CapabilityReport{
		Harness:      a.Name(),
		AdapterVer:   AdapterVersion,
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
		Capabilities: caps,
	}
}

// Capture reads telemetry for a specific Devin session and produces a Profile.
func (a DevinAdapter) Capture(sessionID string, opts CaptureOpts) (Profile, error) {
	cap := a.Probe()
	now := time.Now().UTC().Format(time.RFC3339)

	profile := Profile{
		Schema:       ProfileSchema,
		ProfiledAt:   now,
		Harness:      a.Name(),
		SessionID:    sessionID,
		SnapshotHash: opts.SnapshotHash,
		SkillDir:     opts.SkillDir,
		Capability:   cap,
	}

	// Devin has no skill-level activation or attribution events in this adapter.
	profile.SkillActivation = UnknownActivationResult("Devin has no skill-level activation events in this adapter")
	profile.Attribution = UnknownAttributionResult("Devin does not attribute outputs to skills in this adapter")

	if cap.Capabilities[MetricTokens] == SourceNone &&
		cap.Capabilities[MetricToolCalls] == SourceNone &&
		cap.Capabilities[MetricTiming] == SourceNone {
		noSourceReason := "No Devin telemetry source detected. Provide an ATIF export file via --export-file."
		profile.Tokens = UnknownTokenResult(noSourceReason)
		profile.ToolCalls = UnknownToolCallResult(noSourceReason)
		profile.Timing = UnknownTimingResult(noSourceReason)
		return profile, nil
	}

	exportFile := a.ExportFile
	if exportFile == "" {
		exportFile = opts.ExportFile
	}

	if exportFile == "" {
		errReason := "session_data enabled but no ATIF export file path provided."
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	// The ATIF parser is a placeholder. We verify the file is readable JSON.
	data, err := os.ReadFile(exportFile)
	if err != nil {
		errReason := fmt.Sprintf("failed to read ATIF export file: %v", err)
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	var atifData map[string]any
	if err := json.Unmarshal(data, &atifData); err != nil {
		errReason := fmt.Sprintf("failed to parse ATIF export JSON: %v", err)
		profile.Tokens = ErrorTokenResult(errReason)
		profile.ToolCalls = UnknownToolCallResult(errReason)
		profile.Timing = UnknownTimingResult(errReason)
		return profile, nil
	}

	reason := "Devin ATIF schema not yet verified; token, tool, and timing fields are unmapped"
	profile.Tokens = UnknownTokenResult(reason)
	profile.ToolCalls = UnknownToolCallResult(reason)
	profile.Timing = UnknownTimingResult(reason)
	return profile, nil
}

func hasATIFSignals(file string) bool {
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()

	var header struct {
		Steps    any `json:"steps"`
		Messages any `json:"messages"`
		Transcript any `json:"transcript"`
	}
	if err := json.NewDecoder(f).Decode(&header); err != nil {
		return false
	}
	return header.Steps != nil || header.Messages != nil || header.Transcript != nil
}
