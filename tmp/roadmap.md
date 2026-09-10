# Roadmap

Last reconciled: 2026-09-10 · SHA 541af3e · evidence: dogfood run 2026-09-10, F03-dogfood-decisions.md

## Decisions needed
- **F03 attribution / session boundary** — user — whether F04's with-skill/without-skill comparison absorbs per-skill cost, or F03 builds a Claude Code session-boundary/attribution mechanism (finding 1; see [F03-dogfood-decisions.md](teams/architect/F03-dogfood-decisions.md)).
- **F03 Claude Code telemetry source** — user — whether to add a `session_data` (JSONL transcript) adapter for Claude Code in addition to the OTel file (finding 4; see memo).
- **F03 OTel probe validation** — user — whether `Probe` should parse and validate the OTel export file before declaring `otel` capability (finding 3; see memo).
- **F03 tool-call semantics / duration field** — user — what `ToolCallEntry.Success` means (permission vs execution) and whether to compute, drop, or keep `Duration` (findings 5, 6; see memo).
- **skill-rewrite patch approval** — user — whether to turn the `skill-rewrite-draft.md` into a story and apply the proposed fixes (findings 10, 11, 13; see draft in `tmp/profiler-run-2026-09-10/`).

## Active
| Thread | Stage | Current artifact | Last verified | Next action | Owner | Blocker |
|---|---|---|---|---|---|---|
| F03 profiler contract | intake | [F03-slices.md](teams/architect/F03-slices.md) · [F03.status.md](teams/architect/F03.status.md) · [F03-dogfood-decisions.md](teams/architect/F03-dogfood-decisions.md) | 2026-09-10 | Human approves the decision memo, then implement F03 Slice 1.1 and 1.2 as approved | unassigned | F03 dogfood decisions |
| F03 Slice 1.1: OTel probe validation & tool-call semantics | design | [F03-slices.md](teams/architect/F03-slices.md) | 2026-09-10 | Start after F03 decisions approved | unassigned | probe / tool semantics decision |
| F03 Slice 1.2: Claude Code transcript (session_data) adapter | design | [F03-slices.md](teams/architect/F03-slices.md) | 2026-09-10 | Start after Slice 1.1 | unassigned | telemetry-source decision |
| skill-rewrite patch (dogfood draft) | plan | `tmp/profiler-run-2026-09-10/skill-rewrite-draft.md` | 2026-09-10 | Human approves draft, then implement the `draft-rewrite.sh` fixes | unassigned | human approval |

## Parked
| Thread | Reason | Resume condition | Context |
|---|---|---|---|
| F04 paired comparisons | F03 intake is producing new scope; Slice 1.1/1.2 must land first | A real skill comparison is requested and F03 hardening is merged | [F04.status](teams/architect/F04.status.md) |
| F05 evidence-linked candidates | F04 must produce a working comparison | F04 delivers a verified comparison | [F05.status](teams/architect/F05.status.md) |
| F06 regression coverage | F04 must produce a verified failure to convert | F04 delivers a verified failure | [F06.status](teams/architect/F06.status.md) |

## Completed
| Thread | Outcome | Evidence | Completed at |
|---|---|---|---|
| F01 format/policy split | Format validation separated from house policy; YAML parser; path resolution; `--json` mode on all checkers | test_f01.sh 32 pass · SHA 3e2efd8 | 2026-09-10 |
| F02 audit.json (static scope) | Unified audit report composing skill-validator + skillscore + house policy; self-audit passes; runtime-signal fields left draft | test_f02.sh 58 pass · SHA 3e2efd8 | 2026-09-10 |
| F03 Slice 1 (adapter interface + Claude Code) | MetricResult/CapabilityReport/ProfilerAdapter types, Claude Code OTel adapter, profile serialization, CLI entrypoint | profiler_test.go 11 pass · [profiler-spec.md](../docs/profiler-spec.md) | 2026-09-10 |
| Profiler design space | Surveyed 4 harnesses' telemetry surfaces; 3/4 support OTel, Devin is cloud-only; skill activation is Cursor-only; Option C recommended | [profiler-design-space.md](teams/architect/profiler-design-space.md) | 2026-09-10 |

## State conflicts
- None.
