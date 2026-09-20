package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Spool analysis — the "parse later" read pass. Consumes an append-only spool
// directory and produces the stats the user actually asks for: what ran, how
// often, which models, and the honest token picture.

func seedSpool(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(day string, payloads ...string) {
		f, err := os.OpenFile(filepath.Join(dir, day+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		for _, p := range payloads {
			ts := day + "T12:00:00Z"
			ev, err := NormalizeHookPayload([]byte(p), mustTime(t, ts))
			if err != nil {
				t.Fatal(err)
			}
			line, _ := json.Marshal(ev)
			f.Write(append(line, '\n'))
		}
	}
	write("2026-09-10",
		`{"hook_event_name":"sessionStart","conversation_id":"c1","model":"gpt-5"}`,
		`{"hook_event_name":"beforeSubmitPrompt","conversation_id":"c1","prompt":"fix the flaky test"}`,
		`{"hook_event_name":"preToolUse","conversation_id":"c1","tool_name":"Bash","tool_input":{"command":"go test ./..."}}`,
		`{"hook_event_name":"postToolUse","conversation_id":"c1","tool_name":"Bash","duration":1200}`,
		`{"hook_event_name":"beforeReadFile","conversation_id":"c1","file_path":"/repo/.cursor/skills/commit/SKILL.md"}`,
		`{"hook_event_name":"sessionEnd","conversation_id":"c1","duration_ms":300000,"final_status":"completed"}`,
	)
	write("2026-09-11",
		`{"hook_event_name":"sessionStart","conversation_id":"c2","model":"claude-4"}`,
		`{"hook_event_name":"beforeMCPExecution","conversation_id":"c2","tool_name":"query","mcp_server_name":"linear"}`,
		`{"hook_event_name":"subagentStart","conversation_id":"c2","subagent_type":"researcher"}`,
		`{"hook_event_name":"preCompact","conversation_id":"c2","context_tokens":48200,"context_window_size":200000}`,
		`{"hook_event_name":"beforeReadFile","conversation_id":"c2","file_path":"/repo/src/main.go"}`,
	)
	return dir
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestAnalyzeSpool_BasicCounts(t *testing.T) {
	stats, err := AnalyzeSpool(seedSpool(t))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Events != 11 {
		t.Errorf("events = %d, want 11", stats.Events)
	}
	if stats.SpoolFiles != 2 {
		t.Errorf("spool_files = %d, want 2", stats.SpoolFiles)
	}
	if stats.Conversations != 2 {
		t.Errorf("conversations = %d, want 2", stats.Conversations)
	}
	if stats.EventsByDay["2026-09-10"] != 6 || stats.EventsByDay["2026-09-11"] != 5 {
		t.Errorf("events_by_day = %v", stats.EventsByDay)
	}
}

func TestAnalyzeSpool_ToolAndModelMix(t *testing.T) {
	stats, err := AnalyzeSpool(seedSpool(t))
	if err != nil {
		t.Fatal(err)
	}
	if stats.ToolCalls["Bash"] != 1 {
		t.Errorf("tool_calls[Bash] = %d, want 1", stats.ToolCalls["Bash"])
	}
	if stats.Models["gpt-5"] != 1 || stats.Models["claude-4"] != 1 {
		t.Errorf("models = %v", stats.Models)
	}
	if stats.MCPServers["linear"] != 1 {
		t.Errorf("mcp_servers[linear] = %d, want 1", stats.MCPServers["linear"])
	}
	if stats.SubagentRuns["researcher"] != 1 {
		t.Errorf("subagent_runs[researcher] = %d, want 1", stats.SubagentRuns["researcher"])
	}
}

func TestAnalyzeSpool_SkillActivationInference(t *testing.T) {
	stats, err := AnalyzeSpool(seedSpool(t))
	if err != nil {
		t.Fatal(err)
	}
	// SKILL.md under a skills/ dir → inferred activation of "commit".
	if stats.SkillActivations["commit"] != 1 {
		t.Errorf("skill_activations[commit] = %d, want 1", stats.SkillActivations["commit"])
	}
	// A plain source file read must not count.
	if stats.SkillActivations["main.go"] != 0 {
		t.Errorf("main.go counted as skill: %v", stats.SkillActivations)
	}
}

func TestAnalyzeSpool_TokenHonesty(t *testing.T) {
	stats, err := AnalyzeSpool(seedSpool(t))
	if err != nil {
		t.Fatal(err)
	}
	// chars/4 over raw payloads is an estimate, labelled as such.
	if stats.EstimatedTokens <= 0 {
		t.Errorf("estimated_tokens = %d, want > 0", stats.EstimatedTokens)
	}
	// preCompact context_tokens is the only Cursor-reported number.
	if stats.LastContextTokens != 48200 {
		t.Errorf("last_context_tokens = %d, want 48200", stats.LastContextTokens)
	}
	if stats.Compactions != 1 {
		t.Errorf("compactions = %d, want 1", stats.Compactions)
	}
}

func TestAnalyzeSpool_EmptyDir(t *testing.T) {
	stats, err := AnalyzeSpool(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Events != 0 {
		t.Errorf("events = %d, want 0", stats.Events)
	}
}
