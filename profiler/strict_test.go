package profiler

import (
	"encoding/json"
	"strings"
	"testing"
)

// Strict mode (Q10 knob): default keeps content for self-analysis; strict
// strips prompt/tool text to size stubs while keeping structure and metadata.

func TestNormalizeStrict_StripsContentKeepsMetadata(t *testing.T) {
	payload := `{
		"hook_event_name": "preToolUse",
		"conversation_id": "c1",
		"tool_name": "Bash",
		"tool_input": {"command": "cat ~/.ssh/id_rsa"},
		"file_path": "/repo/.cursor/skills/commit/SKILL.md"
	}`
	ev, err := NormalizeHookPayloadStrict([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	s := string(ev.Raw)
	if strings.Contains(s, "id_rsa") || strings.Contains(s, "cat ~/.ssh") {
		t.Errorf("strict mode left tool_input content: %s", s)
	}
	// Size stub preserves that something was there and how big it was.
	if !strings.Contains(s, "_stripped_bytes") {
		t.Errorf("strict mode should record stripped size: %s", s)
	}
	// Metadata survives — file_path is the skill signal, not content.
	if !strings.Contains(s, "SKILL.md") || !strings.Contains(s, `"tool_name":"Bash"`) {
		t.Errorf("strict mode stripped metadata: %s", s)
	}
}

func TestNormalizeStrict_StillRedactsSecrets(t *testing.T) {
	payload := `{"hook_event_name":"postToolUse","api_key":"sk-abc123456789012345","tool_output":"some result"}`
	ev, err := NormalizeHookPayloadStrict([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ev.Raw), "sk-abc") {
		t.Errorf("strict mode leaked secret: %s", string(ev.Raw))
	}
}

func TestNormalizeStrict_PromptsStripped(t *testing.T) {
	payload := `{"hook_event_name":"beforeSubmitPrompt","prompt":"my private thoughts"}`
	ev, err := NormalizeHookPayloadStrict([]byte(payload), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ev.Raw), "private thoughts") {
		t.Errorf("prompt survived strict mode: %s", string(ev.Raw))
	}
	var raw map[string]any
	json.Unmarshal(ev.Raw, &raw)
	stub, ok := raw["prompt"].(map[string]any)
	if !ok || stub["_stripped_bytes"] == nil {
		t.Errorf("prompt should be a size stub: %s", string(ev.Raw))
	}
}
