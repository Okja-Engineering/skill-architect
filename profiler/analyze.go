// Spool analysis — the "parse later" read pass over the append-only capture
// corpus. Pure Go, zero-dep: DuckDB remains the ad-hoc engine for questions
// that outgrow this summary, and reads the same files.
package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SpoolStats is the aggregate summary produced by AnalyzeSpool.
type SpoolStats struct {
	SpoolFiles    int            `json:"spool_files"`
	Events        int            `json:"events"`
	EventsByDay   map[string]int `json:"events_by_day"`
	EventsByName  map[string]int `json:"events_by_name"`
	Conversations int            `json:"conversations"`

	ToolCalls        map[string]int `json:"tool_calls"`        // preToolUse tool_name
	MCPServers       map[string]int `json:"mcp_servers"`       // beforeMCPExecution mcp_server_name
	SubagentRuns     map[string]int `json:"subagent_runs"`     // subagentStart subagent_type
	Models           map[string]int `json:"models"`            // model field on any event
	SkillActivations map[string]int `json:"skill_activations"` // inferred from SKILL.md reads

	// Honest token picture: EstimatedTokens is chars/4 over raw payload bytes —
	// a relative signal only, never a billed count. LastContextTokens is the
	// Cursor-reported context occupancy observed at the most recent preCompact.
	EstimatedTokens   int64 `json:"estimated_tokens"`
	Compactions       int   `json:"compactions"`
	LastContextTokens int64 `json:"last_context_tokens,omitempty"`
}

// skillPathRe matches a SKILL.md path under any skills/ directory and captures
// the skill name: /repo/.cursor/skills/commit/SKILL.md → "commit". Reading the
// SKILL.md file itself is the activation signal; other files under a skill dir
// are resource reads, not activations.
var skillPathRe = regexp.MustCompile(`(?:^|[/\\])skills[/\\]([^/\\]+)[/\\]SKILL\.md$`)

// AnalyzeSpool reads every *.jsonl file under dir and aggregates. Tolerant by
// contract: malformed lines are skipped, non-object raw payloads contribute to
// event counts but nothing else.
func AnalyzeSpool(dir string) (SpoolStats, error) {
	stats := SpoolStats{
		EventsByDay:      map[string]int{},
		EventsByName:     map[string]int{},
		ToolCalls:        map[string]int{},
		MCPServers:       map[string]int{},
		SubagentRuns:     map[string]int{},
		Models:           map[string]int{},
		SkillActivations: map[string]int{},
	}
	conversations := map[string]bool{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return stats, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		stats.SpoolFiles++
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return stats, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			var ev SpoolEvent
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			stats.accumulate(ev, conversations)
		}
	}
	stats.Conversations = len(conversations)
	return stats, nil
}

func (s *SpoolStats) accumulate(ev SpoolEvent, conversations map[string]bool) {
	s.Events++
	s.EventsByName[ev.Event]++
	if len(ev.Ts) >= 10 {
		s.EventsByDay[ev.Ts[:10]]++
	}
	if ev.ConversationID != "" {
		conversations[ev.ConversationID] = true
	}
	s.EstimatedTokens += int64(len(ev.Raw)) / 4

	var raw map[string]any
	if json.Unmarshal(ev.Raw, &raw) != nil {
		return // non-object raw (e.g. non-JSON stdin captured as a string)
	}
	if m := getString(raw, "model"); m != "" {
		s.Models[m]++
	}

	switch ev.Event {
	case "preToolUse":
		if name := getString(raw, "tool_name"); name != "" {
			s.ToolCalls[name]++
		}
	case "beforeMCPExecution":
		if name := getString(raw, "mcp_server_name"); name != "" {
			s.MCPServers[name]++
		}
	case "subagentStart":
		if name := getString(raw, "subagent_type"); name != "" {
			s.SubagentRuns[name]++
		}
	case "beforeReadFile", "beforeTabFileRead":
		if m := skillPathRe.FindStringSubmatch(getString(raw, "file_path")); m != nil {
			s.SkillActivations[m[1]]++
		}
	case "preCompact":
		s.Compactions++
		s.LastContextTokens = int64(toInt(raw["context_tokens"]))
	}
}
