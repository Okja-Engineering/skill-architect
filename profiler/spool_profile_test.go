package profiler

import (
	"testing"
	"time"
)

// Spool → profile/v1 — hooks as a first-class capture source for CursorAdapter.
// This is the first surface where skill_activation and attribution can be
// "present" for Cursor.

func seedProfileSpool(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	day := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	payloads := []string{
		`{"hook_event_name":"sessionStart","conversation_id":"c1","session_id":"s-1","model":"gpt-5"}`,
		`{"hook_event_name":"beforeReadFile","conversation_id":"c1","file_path":"/repo/.cursor/skills/commit/SKILL.md"}`,
		`{"hook_event_name":"preToolUse","conversation_id":"c1","tool_name":"Bash","tool_use_id":"tu-1","tool_input":{"command":"git status"}}`,
		`{"hook_event_name":"postToolUse","conversation_id":"c1","tool_name":"Bash","tool_use_id":"tu-1","duration":800}`,
		`{"hook_event_name":"preToolUse","conversation_id":"c1","tool_name":"Read","tool_use_id":"tu-2"}`,
		`{"hook_event_name":"postToolUseFailure","conversation_id":"c1","tool_name":"Read","tool_use_id":"tu-2","failure_type":"timeout"}`,
		`{"hook_event_name":"sessionEnd","conversation_id":"c1","session_id":"s-1","duration_ms":300000,"final_status":"completed"}`,
	}
	for i, p := range payloads {
		ev, err := NormalizeHookPayload([]byte(p), day.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if err := AppendSpool(dir, ev); err != nil {
			t.Fatal(err)
		}
	}
	// A second conversation that must not leak into c1's profile.
	other, _ := NormalizeHookPayload(
		[]byte(`{"hook_event_name":"preToolUse","conversation_id":"c2","tool_name":"Write","tool_use_id":"tu-9"}`),
		day.Add(time.Minute))
	if err := AppendSpool(dir, other); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCaptureFromSpool_PresentMetrics(t *testing.T) {
	dir := seedProfileSpool(t)
	adapter := CursorAdapter{SpoolDir: dir}
	profile, err := adapter.Capture("c1", CaptureOpts{SnapshotHash: "sha", SkillDir: "/s"})
	if err != nil {
		t.Fatal(err)
	}

	if profile.ToolCalls.State != MetricPresent || profile.ToolCalls.Source != "hooks" {
		t.Fatalf("tool_calls = %q/%q, want present/hooks", profile.ToolCalls.State, profile.ToolCalls.Source)
	}
	if len(profile.ToolCalls.Value) != 2 {
		t.Fatalf("tool_calls = %d, want 2 (c2's call must not leak)", len(profile.ToolCalls.Value))
	}
	// Pairing by tool_use_id: tu-1 succeeded, tu-2 failed with failure_type.
	var bash, read *ToolCallEntry
	for i := range profile.ToolCalls.Value {
		e := &profile.ToolCalls.Value[i]
		if e.Name == "Bash" {
			bash = e
		}
		if e.Name == "Read" {
			read = e
		}
	}
	if bash == nil || !bash.Success {
		t.Errorf("Bash entry = %+v, want success", bash)
	}
	if read == nil || read.Success || read.ErrorType != "timeout" {
		t.Errorf("Read entry = %+v, want failure/error_type=timeout", read)
	}

	if profile.SkillActivation.State != MetricPresent || len(profile.SkillActivation.Value) != 1 {
		t.Fatalf("skill_activation = %+v, want 1 present entry", profile.SkillActivation)
	}
	if profile.SkillActivation.Value[0].SkillName != "commit" {
		t.Errorf("skill = %q, want commit", profile.SkillActivation.Value[0].SkillName)
	}
	if profile.SkillActivation.Value[0].Trigger != "inferred" {
		t.Errorf("trigger = %q, want inferred (file-read heuristic)", profile.SkillActivation.Value[0].Trigger)
	}

	if profile.Timing.State != MetricPresent || profile.Timing.Value.TotalMs != 300000 {
		t.Errorf("timing = %+v, want present/300000ms", profile.Timing)
	}
}

func TestCaptureFromSpool_TokenHonesty(t *testing.T) {
	dir := seedProfileSpool(t)
	profile, err := CursorAdapter{SpoolDir: dir}.Capture("c1", CaptureOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// Hooks never carry billed tokens — unknown with a truthful reason.
	if profile.Tokens.State != MetricUnknown {
		t.Errorf("tokens = %q, want unknown", profile.Tokens.State)
	}
	// The estimate lives in its own labelled field, present, sourced as estimated.
	if profile.EstimatedContextTokens.State != MetricPresent ||
		profile.EstimatedContextTokens.Source != string(SourceHooksEstimated) {
		t.Errorf("estimated_context_tokens = %q/%q, want present/hooks_estimated",
			profile.EstimatedContextTokens.State, profile.EstimatedContextTokens.Source)
	}
	if profile.EstimatedContextTokens.Value == nil || profile.EstimatedContextTokens.Value.Total <= 0 {
		t.Errorf("estimated total = %+v, want > 0", profile.EstimatedContextTokens.Value)
	}
}

func TestCaptureFromSpool_Attribution(t *testing.T) {
	dir := seedProfileSpool(t)
	profile, err := CursorAdapter{SpoolDir: dir}.Capture("c1", CaptureOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Attribution.State != MetricPresent || profile.Attribution.Source != "hooks" {
		t.Fatalf("attribution = %q/%q, want present/hooks", profile.Attribution.State, profile.Attribution.Source)
	}
	// Both tool calls follow the commit activation → attributed to it.
	found := 0
	for _, a := range profile.Attribution.Value.Attributions {
		if a.SkillName == "commit" && (a.Target == "tu-1" || a.Target == "tu-2") {
			found++
			if a.Confidence != "inferred" {
				t.Errorf("confidence = %q, want inferred", a.Confidence)
			}
		}
	}
	if found != 2 {
		t.Errorf("attributed calls = %d, want 2: %+v", found, profile.Attribution.Value.Attributions)
	}
}

func TestCaptureFromSpool_SessionIDPreference(t *testing.T) {
	dir := seedProfileSpool(t)
	profile, err := CursorAdapter{SpoolDir: dir}.Capture("c1", CaptureOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// Hierarchy: session_id > conversation_id (spec decision, report risk S-8).
	if profile.SessionID != "s-1" {
		t.Errorf("session_id = %q, want s-1", profile.SessionID)
	}
}

func TestCaptureFromSpool_NoSessionMatch(t *testing.T) {
	dir := seedProfileSpool(t)
	profile, err := CursorAdapter{SpoolDir: dir}.Capture("nobody", CaptureOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if profile.ToolCalls.State != MetricUnknown {
		t.Errorf("tool_calls = %q, want unknown for unmatched session", profile.ToolCalls.State)
	}
}

func TestProbe_SpoolDirDeclaresHooks(t *testing.T) {
	dir := seedProfileSpool(t)
	cap := CursorAdapter{SpoolDir: dir}.Probe()
	if cap.Capabilities[MetricToolCalls] != SourceHooks {
		t.Errorf("tool_calls = %q, want hooks", cap.Capabilities[MetricToolCalls])
	}
	if cap.Capabilities[MetricSkillActivation] != SourceHooks {
		t.Errorf("skill_activation = %q, want hooks", cap.Capabilities[MetricSkillActivation])
	}
}
