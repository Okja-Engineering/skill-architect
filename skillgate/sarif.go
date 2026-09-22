package skillgate

import "encoding/json"

// SARIF 2.1.0 emitter — thin projection over report/v1 for GitHub Code
// Scanning and IDE viewers. report/v1 remains the contract; SARIF is never
// the sole output (it carries no effort field and no ledger).

type sarifReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string         `json:"id"`
	ShortDescription sarifText      `json:"shortDescription"`
	Properties       sarifRuleProps `json:"properties"`
}

type sarifRuleProps struct {
	SecuritySeverity string `json:"security-severity,omitempty"`
	EffortMinutes    int    `json:"effortMinutes,omitempty"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // error | warning | note
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func sarifLevel(f Finding) string {
	if f.Suppressed || f.Advisory {
		if f.Severity == SeverityBlocker || f.Severity == SeverityHigh {
			return "warning"
		}
		return "note"
	}
	switch f.Severity {
	case SeverityBlocker, SeverityHigh:
		return "error"
	case SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}

// MarshalSARIF renders a report as SARIF 2.1.0. Suppressed findings are
// included (they stay in machine output) at note level with a suppression
// marker in the message.
func MarshalSARIF(rep *Report) ([]byte, error) {
	rules := map[string]bool{}
	var driver []sarifRule
	var results []sarifResult
	for _, f := range rep.Findings {
		if !rules[f.RuleID] {
			rules[f.RuleID] = true
			driver = append(driver, sarifRule{
				ID:               f.RuleID,
				ShortDescription: sarifText{Text: f.Message},
				Properties:       sarifRuleProps{EffortMinutes: f.EffortMinutes},
			})
		}
		msg := f.Message
		if f.Suppressed {
			msg += " (suppressed: " + f.SuppressReason + ")"
		}
		r := sarifResult{RuleID: f.RuleID, Level: sarifLevel(f), Message: sarifText{Text: msg}}
		if f.File != "" {
			loc := sarifLocation{PhysicalLocation: sarifPhysical{ArtifactLocation: sarifArtifact{URI: f.File}}}
			if f.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{StartLine: f.Line}
			}
			r.Locations = []sarifLocation{loc}
		}
		results = append(results, r)
	}
	return json.MarshalIndent(sarifReport{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool:    sarifTool{Driver: sarifDriver{Name: "skillgate", Version: rep.Tool.Version, Rules: driver}},
			Results: results,
		}},
	}, "", "  ")
}
