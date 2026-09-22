# Warp `skill-doctor` — what it is, how good it is, what we take

Target: `github.com/warpdotdev/common-skills`, `.agents/skills/skill-doctor/` @ `main` (== `b811c24` for
`collect_sessions.py`); MIT, © 2026 Denver Technologies (`LICENSE:1-3`) **[O]**. Paths below are relative to
that directory. `cs` = `scripts/collect_sessions.py`, `rr` = `scripts/render_report.py`, `wd` =
`scripts/warp_decoder.py`. **[O]** observed · **[I]** inferred.

## 1. Five-line summary

1. **Not a skill linter.** It never reads a SKILL.md for quality — it grades *the user's recent agent
   conversations* and blames the skills (`SKILL.md:7`) **[O]**.
2. The only deterministic code is a **six-harness session harvester** (`cs`, 1669 lines) that condenses
   transcripts to markdown; **all judging and all arithmetic is LLM work**, from two rubric files and prose
   formulas (`SKILL.md:101-109`) **[O]**.
3. Output: a curved letter grade (A+…F) on one self-contained HTML page, plus LLM-drafted `diff`s against
   real skills, ending in a Warp Factories CTA (`SKILL.md:152,169`) **[O]**.
4. Two demonstrable measurement defects: `skill_coverage` measures **the sampler, not the user** (§3.1:
   true 0.95 → reported 0.75; true 0.05 → 0.20) **[O]**, and code-quality is judged on tool payloads
   truncated to **500 chars** (`cs:30-31,397`) **[O]**.
5. Borrow the evidence gate, `insufficient_evidence` as an *excluded* state, the prompt-invariant tests and
   the parsers; reject the curve, the weights, coverage-as-metric, and LLM-executed arithmetic.

## 2. What it is, precisely

### 2.1 File inventory **[O]**

| File | Lines | Role | Nature |
|---|---|---|---|
| `SKILL.md` | 171 | 7-step orchestration prompt; holds every formula | LLM prompt |
| `scorers/efficiency.md` | 34 | 4-label rubric + 7 dimensions | LLM-judged |
| `scorers/code-quality.md` | 37 | 3-label rubric + 11 dimensions | LLM-judged |
| `references/supported-harnesses.md` | 47 | harness gate, collector IDs, skill dirs | LLM prompt |
| `references/skill-improvements.md` | 32 | when a suggestion may be filed | LLM prompt |
| `scripts/collect_sessions.py` | 1669 | harvest + condense → `inventory.json`, `transcripts/*.md` | deterministic |
| `scripts/warp_decoder.py` | 388 | protobuf-free decode of Warp `Task` blobs | deterministic |
| `scripts/render_report.py` | 584 | `report.json` → self-contained `report.html` + PNG card | deterministic |
| `scripts/test_collect_sessions.py` | 593 | **14** tests in **5** classes (`:44,245,374,435,484`) | deterministic |
| `scripts/test_render_report.py` | 252 | **11** tests; **3** read `SKILL.md` as golden prose (`:18,190,210`) | deterministic |
| `assets/pierre-diffs.js` | 1531 | minified diff renderer; 1,156,623 B, **0** copyright/license strings | asset |
| `assets/warp-pixel-icon.svg` | 5 | branding | asset |

### 2.2 Every check — **35 deterministic + 17 LLM-judged** **[O]**

IDs match the independent re-enumeration in `.scuba/teams/warp-skill-doctor/arena-verdict.md` §2.

| ID | Check / rule | Location | D/L | Effect |
|---|---|---|---|---|
| L1 | Harness self-identification gate; stop before creating REPORT_DIR or reading history if unsupported | `SKILL.md:19`; `supported-harnesses.md:16-20` | **L** — model introspects its runtime, explicitly *not* from disk | hard stop |
| L2 | Conversation-scope interview (3 options with repo, 2 without) | `SKILL.md:31-42` | L | run config |
| L3 | Skill-scope interview (project+global / project only) | `SKILL.md:44-51` | L | run config |
| D1 | `--all-conversations` + `--repo` conflict → **exit 2** | `cs:1302-1307` | D | hard fail |
| D2 | Explicit-harness source missing → **exit 1** (claude/codex/warp/pi/grok/zcode) | `cs:1368-1373,1402-1404,1445-1447,1476-1478,1507-1509,1538-1543` | D | hard fail |
| D3 | No supported source found at all → **exit 1** | `cs:1545-1550` | D | hard fail |
| D4 | Skill discovery: 3 project + 6 global roots, glob `*/SKILL.md` | `cs:132-152,158` | D | inventory |
| D5 | First-name-wins collision dedupe | `cs:160-161` | D | inventory |
| D6 | `description:` regex, stripped, capped at 300 chars; + `bytes`, `modified_at` | `cs:166-176` | D | **the only static read of a skill** |
| D7 | `--days` cutoff (default 45) on mtime | `cs:94,1327` | D | corpus |
| D8 | Session discovery per source: codex `:180-194`, claude `:197-216`, warp `:547-591,610-655`, pi `:852-867`, grok `:992-1007`, zcode `:1099-1114` | `cs` as listed | D | corpus |
| D9 | Warp cross-channel dedup: newest row per `conversation_id` | `cs:610-655` | D | dedup |
| D10 | Warp conversation size cap **32 MiB** — silently drops larger conversations | `cs:29` | D | hard limit |
| D11 | Warp protobuf-free decode: varint reader `wd:122-171`; field decoders `wd:285-374`; `decode_task` `wd:376` | `wd` as listed | D | parsing |
| D12 | Warp skill-name-from-reference resolution | `cs:705-718` | D | attribution |
| D13 | Sidechain/child exclusion unless `--include-subagents` (claude `isSidechain`; warp `parent_*_id`) | `cs:324-327,726-730` | D | filter |
| D14 | `looks_injected` filter: leading `<` + one of 9 tag names | `cs:428-437` | D | denoise |
| D15 | Truncation `MAX_MSG_CHARS` 1500 / `MAX_TOOL_CHARS` 500 | `cs:30-31,219-223`; applied `:376,397,403` | D | condense |
| D16 | `TranscriptBuffer`: >160 entries → head 100 + omitted-note + tail 40 | `cs:32-34,256-280`; re-clipped `:1214-1217` | D | condense |
| D17 | Stats: `user_turns` `:405-406`; `assistant_turns` deduped by message id `:360-364`; `tool_calls` `:380`; `repeated_tool_calls` = SHA-1(name+args) count>1 `:384-387`; `error_outputs` = `is_error` OR substring `error`/`failed`/`traceback` in first 2000 chars lowercased `:398-402` | `cs` as listed | D | **signals only, no thresholds** |
| D18 | `has_code_edits` = tool name ∈ edit-tool sets OR arg matches one of 6 `CODE_EDIT_HINTS` | `cs:36-38,421-424`; generic `:1193-1197` | D | header string only — **gates nothing** (cf. L9) |
| D19 | `detect_skill_candidates` regex on tool args: `(^\|/)skills/+([^/]+)/+`, `"skill"\|"name"\|"bundled_skill_id"` | `cs:283-291` | D | attribution |
| D20 | Direct capture of the `Skill` tool's `skill` argument | `cs:393-396` | D | attribution |
| D21 | `detect_skills_from_entries`: 5 literal markers per installed name | `cs:1280-1297` | D | attribution |
| D22 | Detections intersected with installed set (unknown names dropped) | `cs:1562-1570` | D | attribution |
| D23 | Repo attribution: cwd inside repo **OR** basename equality **OR** repo name in path parts | `cs:1224-1244`; over-match caveat `:1232-1234` | D | filter |
| D24 | `--all-conversations`: infer repos via `git rev-parse --show-toplevel`, re-discover skills | `cs:1251-1277,1551-1561` | D | scoping |
| D25 | Scoreability filter: drop `assistant_turns < 1` **or** `tool_calls < 1` | `cs:1357,1391,1433,1465,1496,1527` | D | filter |
| D26 | Newest-first sort, stable session key | `cs:1572-1574` | D | selection |
| D27 | Sampling: ≤`--per-skill` 3/skill, then ≤`--no-skill` 4 skill-free, cap `--max-sessions` 12 | `cs:95-97,1576-1593` | D | **biases L10's coverage** |
| D28 | Transcript rendering incl. the deterministic stats header line | `cs:1201-1221`; stats `:1207-1209` | D | judge context |
| D29 | `inventory.json` schema + console summary | `cs:1620-1650,1652-1665` | D | audit trail |
| L4 | Zero-result gate: `sessions_sampled == 0` → tell user and **stop**; `skills_found == 0` → **continue**, coverage 0 | `SKILL.md:88` | L | control flow |
| L5 | Batching: ≤50 transcripts one batch; >50 → parallel batches of 20; local child agents only | `SKILL.md:92` | L | privacy/scale |
| L6 | **Efficiency scorer** — `highly_efficient` 1 / `mostly_efficient` .8 / `mostly_inefficient` .4 / `highly_inefficient` .2; 7 dimensions: rework, cost-to-human, information gathering, routine-step overhead, batching, flailing, verification timing; "not an exhaustive checklist" | `efficiency.md:4-16`, `:20`, `:24-30`, `:32-34` | **L** | **50% of grade** |
| L7 | **Code-quality scorer** — `approve` 1 / `block` .2 / `insufficient_evidence` .5; binary, one defect ⇒ block; 11 dimensions: design, correctness, complexity, repo conventions, smells, tests, naming, comments, diff hygiene, docs, mid-run corrections; efficiency + instruction-following out of scope | `code-quality.md:4-13`, `:17`, `:21-31`, `:33`, `:35-37` | **L** | **35% of grade** |
| L8 | Per-scorer record: label + numeric score + 1–3 sentence reason citing transcript specifics | `SKILL.md:97` | L | output |
| L9 | **Code-quality applicability gate** — apply "only where the transcript shows code changes", else `insufficient_evidence` and exclude | `SKILL.md:97` | **L** — and **duplicative of D18**, which already answers it in code and prints it at `cs:1209` | gates L7 |
| L10 | Aggregation: `raw_efficiency` mean; `raw_code_quality` mean excl. `insufficient_evidence` (0.5 if none); `curve(s)=0.5+0.5*s`; `skill_coverage`; `overall = .5·E + .35·Q + .15·C` | `SKILL.md:101-107` | **L executing a formula** — no script computes or validates it | the grade |
| L11 | `failed_conversations` = any applicable **raw** score < 0.5; `insufficient_evidence` never fails | `SKILL.md:109` | L | gates L13/L14 |
| L12 | `top_findings` = 3 most impactful patterns, "following the STE-100 standard" | `SKILL.md:113` | L | output (**STE-100 undefined**) |
| L13 | `suggestions` cite failed session + scorer + moment; never-fired installed skill ⇒ "usually a description problem" | `SKILL.md:114` | L | output |
| L14 | Edit drafting: read skill from `inventory.json`, write improved file to `proposed/<skill>/SKILL.md`, `diff -u` into `diff`; never modify real skill files | `SKILL.md:116-126` | L (+ `diff` D) | output |
| L15 | Method: cluster by root cause → prioritize frequency × severity → verify against the live repo, drop what fails → state behavioral rule + owning surface first → prefer **replace** over **append** | `skill-improvements.md:5-9` | L | precision |
| L16 | Admissibility: 4 must-all-be-true file-when conditions; 4 do-not-file conditions; "when nothing clears this bar, open no change and say why — that is a success" | `skill-improvements.md:15-20,22-27,29` | **L** | suppresses noise |
| D30 | `GRADES` thresholds A+ .97 / A .93 / A− .90 / B+ .87 / B .83 / B− .80 / C+ .77 / C .73 / C− .70 / D .60 / else F | `rr:23-28` (entries `:24-27`); `grade_for` `:38-42` | D | presentation |
| D31 | `pct()` = `round(score*100)` | `rr:45-46` | D | presentation |
| D32 | Grade defaulted from `overall`; **`overall` is never recomputed or validated** | `rr:231,567` (in `main()` `:560-571`) | D | no validation layer |
| D33 | Missing `report.json` → **exit 1** | `rr:563-565` | D | hard fail |
| D34 | HTML render: diff clamp 320 px, collapsed long diffs, 1200×675 client-side PNG, browser launch | `rr:35,83-110,228-314,315-540,71-77` | D | presentation |
| D35 | Self-tests: **14** collector tests / 5 classes; **11** render tests; **3** read `SKILL.md` — `:18` startup contract, `:190` output labels, `:210` curve + weights + failed-conversations rule (19 assertions over `skill_text`) | `scripts/test_*.py` | D | prompt-drift guard |
| L17 | Step-6 output contract: speak grade + 3 findings, then two exact closing lines incl. the Warp Factories CTA | `SKILL.md:162-171`; `cta_url` `:152` | L | output |

**Nothing in D1–D35 evaluates skill content.** The only static read is D6. No frontmatter validation, token
counting, link resolution, security scan, duplication check, rule IDs, severities, SARIF, or exit-code gating
on quality **[O]**. Warp's static authoring guidance lives in a *different* skill,
`.agents/skills/update-skill/references/best-practices.md` **[O]**.

### 2.3 Inputs, outputs, invocation **[O]**

| Aspect | Detail |
|---|---|
| Invocation | Natural-language trigger via frontmatter `description` (`SKILL.md:3`); no slash command, no CLI entrypoint |
| Inputs | Local session stores (Claude `~/.claude/projects` JSONL, Codex rollouts, Warp SQLite, `~/.pi/agent/sessions`, `~/.grok/sessions`, `~/.zcode/cli/rollout`) + skill dirs `.agents\|.claude\|.codex/skills` (`supported-harnesses.md:7-14,39-47`) |
| Runtime | `python3` stdlib only (`sqlite3`, `hashlib`, `json`); no third-party deps |
| Intermediate | `$REPORT_DIR/inventory.json`, `transcripts/*.md`, `proposed/<skill>/SKILL.md` |
| Final | `report.json` (schema `SKILL.md:131-154`) → `report.html`; 1200×675 PNG share card in-browser (`SKILL.md:160`) |
| Privacy & scratch hygiene | "Never upload transcripts… anywhere" (`SKILL.md:11`); `mktemp -d` scratch dir, never write into the user's repo (`SKILL.md:53-57`). Both are **prompt-enforced, not code-enforced** — no script checks either **[I]** |
| Undocumented | `--per-skill` / `--no-skill` exist (`cs:96-97`) but are absent from SKILL.md's flag list (`SKILL.md:77-86`) — the agent cannot tune the sampler that drives 15% of the grade **[O]** |
| Ambiguous | `SKILL.md:113` demands findings "following the STE-100 standard"; **STE-100 is defined nowhere in the repo** (`grep -rn "STE-100"` → one hit, the reference itself) **[O]** |

## 3. The scoring model

```
curve(s) = 0.5 + 0.5*s                                                      (SKILL.md:103)
efficiency   = curve(mean raw efficiency)                                   (SKILL.md:101,104)
code_quality = curve(mean raw CQ, excl. insufficient_evidence; 0.5 if none) (SKILL.md:102,105)
skill_coverage = sampled sessions with >=1 skill / sampled sessions         (SKILL.md:106) — NOT curved
overall = 0.50*efficiency + 0.35*code_quality + 0.15*skill_coverage         (SKILL.md:107)
```

`render_report.py` **never recomputes or validates** these — `main()` reads `r["scores"]["overall"]` and only
derives the letter (`rr:231,567`) **[O]**. An LLM arithmetic slip is undetectable.

**The curve, at all three scopes** (arithmetic on the published constants) **[I from [O] constants]**:

| Quantity | Value | Derivation |
|---|---|---|
| Mathematical floor of `curve()` | **0.50** | `curve(0)` — **unreachable**: no rubric label scores 0 |
| Attainable floor of `efficiency` / `code_quality` | **0.60** | min label 0.2 (`efficiency.md:16`, `code-quality.md:9`) → `curve(0.2)` |
| Attainable floor of `overall` | **0.51** | coverage is uncurved and can be 0 → `.5(.6)+.35(.6)+.15(0)` → **F** (`rr:27`) |
| Range at coverage 0 | **[0.51, 0.85]** | F, D, C **and B all reachable**; B (0.85) caps a perfect run with no skills |
| Range at coverage 1.0 | **[0.66, 1.00]** | worst-possible rubrics still grade **D** |
| A+ | needs coverage ≥ 0.80 even with perfect rubrics | `0.5+0.35+0.15c ≥ 0.97` |

So **0.60 is the per-component floor, 0.51 the overall floor, and 0.50 a floor the function has but the
rubrics cannot reach.** F/D/C/B are all reachable at low coverage; it is the **A band that coverage gates** —
and the report's CTA sells more skills (`SKILL.md:152,169`) **[O]**. Coverage, the only uncurved term, carries
most of the *variance* despite the smallest weight.

| raw eff | raw CQ | coverage | eff | CQ | overall | grade |
|---|---|---|---|---|---|---|
| 0.2 | 0.2 (block) | 0.0 | 0.60 | 0.60 | **0.510** | F |
| 0.2 | 0.2 | 1.0 | 0.60 | 0.60 | 0.660 | D |
| 1.0 | 1.0 | 0.0 | 1.00 | 1.00 | **0.850** | B |
| 0.8 | 1.0 | 0.5 | 0.90 | 1.00 | 0.875 | B+ |
| 1.0 | 1.0 | 1.0 | 1.00 | 1.00 | 1.000 | A+ |

**Comparability.** Across runs: poor — corpus is "newest 12 sessions in a 45-day window". Across skills:
**undefined** — the grade is per *user/repo*; no skill is ever scored. Across users: only with the same model
and harness — free-form LLM labels, no calibration set, no seed, no inter-rater check **[O]**.

### 3.1 Empirical defect: `skill_coverage` measures the sampler

Synthetic Claude-Code histories (20 sessions each), published collector run unmodified **[O]**. The arena
hunter rebuilt the fixtures independently and reproduced all four rows exactly **[O]**.

| Scenario | True sessions-with-skill | Sampled | Reported coverage | Error |
|---|---|---|---|---|
| 19/20 sessions use 1 skill | 0.95 | 4 | **0.75** | −0.20 |
| 14/20 use 2 skills | 0.70 | 10 | **0.60** | −0.10 |
| 20/20 use 5 skills | 1.00 | 12 | **1.00** | 0 |
| 1/20 uses 1 skill | 0.05 | 5 | **0.20** | +0.15 |

Cause: the `≤3 per skill` + `≤4 no-skill` caps (D27) make the sample's skill fraction a function of **how many
distinct skills are installed**, not how often they fire. `inventory.json` already carries the unbiased
counters (`skill_usage`, `sessions_considered`, D29), so the fix is a one-line definition change — but
SKILL.md specifies the biased one. Row 1's entire grade rested on **4 transcripts**, with no confidence
interval reported anywhere **[O]**.

### 3.2 Empirical defect: the code-quality scorer cannot see the code

A 5.4 KB `Write` payload reaches the judge as 500 chars plus `…[truncated 4939 chars]`, while the header still
asserts `code edits: True` (D15; D28 `cs:1207-1209`) **[O]** — yet the rubric demands senior-reviewer judgment
on tests, edge cases, concurrency and diff hygiene (`code-quality.md:21-31`) **[O]**. Either the judge returns
`insufficient_evidence` constantly (CQ then falls back to 0.5 → curved 0.75, `SKILL.md:102`) or it grades
blind. Unclear which; I did not run the LLM stage.

## 4. Static vs dynamic

| Class | Present? | Evidence |
|---|---|---|
| Static analysis of skill files | **No** — metadata only | D6 **[O]** |
| Dynamic: observe agent behavior | **Yes, retrospective** — parses recorded sessions | D8 `cs:294-1199` **[O]** |
| Dynamic: execute the skill / spawn a trial | **No** | no agent subprocess anywhere in `scripts/` **[O]** |
| Paired with/without-skill comparison | **No** | `overall` has no counterfactual term, L10 **[O]** |
| Before/after re-measurement of an edit | **No** — drafts diffs, stops | L14 `SKILL.md:126`; closing line only *offers* to apply, `:171` **[O]** |
| Eval harness for skills | **No**; **yes** for its own code | 25 unit tests, 3 pinning SKILL.md prose — D35 **[O]** |

A **retrospective observational instrument**, not an experiment: it proposes edits it cannot verify. Same
limitation our research pinned on Anthropic's `/skill-doctor` — measures *whether*, not *whether it helped*
(`docs/research/skill-profiling-state-of-the-art.md:92-94`) **[O]** — but one step further, since it scores
session *outcome quality*, not just invocation counts.

## 5. Head-to-head

● has · ◐ partial · ○ no · **UNK** not established by any available source. [A] =
`/Users/matthewvandusen/Development/Auraprix/skill-architect/skills/skill-audit`; [D] =
`/Users/matthewvandusen/Development/Auraprix/cursor-profiler/docs/research/skill-profiling-state-of-the-art.md`;
[L] = `/Users/matthewvandusen/Development/Auraprix/cursor-profiler/.scuba/teams/skill-audit-tool/L1-static-report.md`.
**Negative cells:** Warp ○ = full read of all 10 non-binary files + `grep` for `token`/`sarif`/`severity`/`rule`,
no relevant hits **[O]**; agnix/SkillSpector ○ = absent from the capability enumerations [L:126], [L:128] **[O]**;
Anthropic ○ = absent from [D:73-104] **[O]**. `claude plugin eval` cells are second-hand via [D:96-103], which
itself notes no canonical CLI reference was reached **[I]**.

| Capability | Warp skill-doctor | `skill-audit` | Anthropic `/skill-doctor` | `claude plugin eval` | agnix | SkillSpector |
|---|---|---|---|---|---|---|
| Reads SKILL.md for spec conformance | ○ metadata only, D6 | ● `skill-validator` via `check-frontmatter.sh:24` | ○ [D:77-78] | ○ [D:96-103] | ● 455 rules [L:126] | ◐ security lens [L:128] |
| Rule IDs + severity | ○ | ◐ PL001-005/PT001-002 cover **house policy only** (`A/SKILL.md:87`) | ○ | ○ | ● `CC-*`,`CUR-*`+`explain` [L:126] | ● 71 patterns/17 cats [L:128] |
| Per-skill context/token cost | ○ | ◐ real `o200k_base` counts from `skill-validator` (`A/SKILL.md:56`) + judged dim (`A/evaluation-matrix.md:19`) | ● cost + invocation count [D:77-78] | ○ | ◐ [L:126] | ○ |
| Trigger/description quality | ◐ only inferred from a never-fired skill, L13 | ● `skillscore` routing dim via `check-quality.sh:20` | ○ | ○ | ● [L:126] | ◐ trigger abuse [L:128] |
| Security scanning | ○ | ○ | ○ | ○ | ◐ config/hook rules [L:126] | ● SARIF, 0-100 risk [L:128] |
| Duplication / cross-skill overlap | ○ | ◐ Tier-2 prose, unimplemented (`A/evaluation-matrix.md:29`) | ○ | ○ | ○ | ○ unclaimed [L:142] |
| **Real session history ("did it fire")** | ● 6 harnesses, D19-D22 | ○ | ● current session only [D:80] | ○ synthetic cases [D:96-97] | ○ | ○ |
| Judges outcome quality (efficiency/code) | ● L6, L7 | ○ | ○ cost only [D:92-94] | ◐ output assertions [D:102-103] | ○ | ○ |
| Repeatable eval suite | ○ | ○ | ○ | ● `evals/evals.json`, isolated subagents [D:96-97] | ● CI action [L:126] | ● baseline re-scan [L:128] |
| Paired with/without eval | ○ | ◐ named Tier 3, unimplemented (`A/evaluation-matrix.md:30`) | ○ [D:92-94] | ○ assertion ≠ paired [D:102-103] | ○ | ○ |
| Proposes concrete fixes (diffs) | ● L14 | ○ reports only (`A/SKILL.md:24,155-159`) | ◐ "where to turn them off" [D:77-78] | ○ | ● auto-fix tiers [L:126] | ○ |
| Evidence gate on when **not** to file | ● L11 + L16 | ◐ fixes "tied to failed criteria" (`A/SKILL.md:116`), no do-not-file list | UNK | UNK | n/a (rules) | n/a |
| Aggregate score / grade | ● A+…F, D30 | ◐ 0-2 × 10 dims, no aggregate (`A/evaluation-matrix.md:11-22`) | ○ | ◐ pass/fail | ◐ `--strict` [L:126] | ● 0-100 [L:128] |
| Comparable across runs | ○ corpus drifts (§3) | ● same input → same dims | ● | ● fixed cases | ● | ● + baseline suppression [L:128] |
| Machine-readable output | ◐ bespoke `report.json`, LLM-written (`SKILL.md:131-154`) | ◐ bespoke JSON `scripts/audit-report.sh` | ◐ text under `-p` [D:83] | ◐ HTML viewer [D:97] | ● `--format github` [L:126] | ● JSON/MD/SARIF [L:128] |
| CI gate / exit codes | ○ interactive only — D1/D3/D33 gate *inputs*, never scores | ● 0/1/2/3 (`A/SKILL.md:87`) | ○ | ◐ | ● [L:126] | ● [L:128] |
| Shareable report artifact | ● self-contained HTML + PNG, D34 | ○ markdown | ◐ Stats tab [D:83] | ◐ HTML viewer [D:97] | ○ | ○ |
| Remediation cost (min) / debt ratio | ○ | ○ | ○ | ○ | ○ | ○ — **unclaimed by every tool** [L:142] |
| Multi-harness | ● 6 | ◐ file-format agnostic | ○ Claude Code only [D:80] | ○ | ● 8 ecosystems [L:126] | ● [L:128] |

## 6. Verdict

### Borrow (MIT, attribution-only — `LICENSE:1-3`) **[O]**

| # | Item | Location | Why |
|---|---|---|---|
| B1 | **`failed_conversations` gate** — only a sub-0.5 raw score may motivate a recommendation | L11 | Kills the generic-best-practice noise that makes audit tools ignorable; cheapest quality lever in the target |
| B2 | **Suggestion-admissibility contract** — 4 must / 4 must-not, plus "when nothing clears this bar, open no change and say why — that is a success" | L16 | A precision discipline our `skill-audit` lacks entirely; transplantable into our finding-emission rules as-is |
| B3 | **`insufficient_evidence` as a first-class *excluded* state**, not a middling score | L7/L9 (`code-quality.md:11-13`; excluded at `SKILL.md:97,102`) | **Names a defect in our own product:** `skill-audit/SKILL.md:155` says "When in doubt, mark the dimension as 1 (present but weak)" **[O]** — forcing a number into every aggregate and biasing every score we emit upward. Not-applicable must be excludable |
| B4 | **Prompt-invariant unit tests** asserting SKILL.md literally contains its own formulas and contracts | D35 | Prompt text silently drifts from the code it describes; we have no equivalent |
| B5 | **Six-harness parsers + `warp_decoder.py`** | D8, D11 | Weeks of undocumented-format work, MIT. Even if we write Go, this is the executable spec per format |
| B6 | **Deterministic per-session signals in the transcript header** | D17, D18, D28 | Right pattern — compute cheap facts in code, let the judge reason over them; but *threshold* them (R5) |
| B7 | **Self-contained HTML + PNG card**; **"propose to `proposed/`, never touch real files"** | D34; L14 | The mechanic behind 570★; and a reviewable-diff flow matching our report-don't-rewrite rule (`A/SKILL.md:24,155`) |

### Reject

| # | Item | Why |
|---|---|---|
| R1 | **`curve(s)=0.5+0.5*s`** (L10) | Lifts the component floor to 0.60 and the overall floor to 0.51, and makes the **A band a function of coverage, not quality** (§3). We must be able to say "this is bad" on the merits |
| R2 | **`skill_coverage` as a quality term** (`SKILL.md:106`) | Empirically measures the sampler (§3.1); rewarding "more skills installed" is a vendor incentive, paired with a sales CTA |
| R3 | **LLM-executed aggregation** (L10; no recompute, D32) | Unverifiable, unreproducible. Weighted sums belong in code |
| R4 | **Re-asking a judge what a script already answered** — L9 is LLM-judged although D18 computes it and prints it at `cs:1209` | The clearest det-vs-LLM anti-pattern in the target; costs determinism for nothing. Adopt the inverse as a house rule |
| R5 | **Unthresholded substring heuristics** — `error_outputs` fires on "no errors", "error handling", `stderr` (D17) | Fine as prompt garnish, unusable as a metric. If we adopt B6, tighten it |
| R6 | **Binary approve/block** on 500-char-truncated payloads (L7; D15) | The instrument cannot see its subject (§3.2); one nit costs the same as a concurrency bug — incompatible with a debt model |
| R7 | **Basename repo matching** (D23) and **head/tail clipping** (D16) | Over-match is self-documented (`cs:1232-1234`); clipping drops the middle of long sessions — where rework loops live — so it flatters long sessions on efficiency |
| R8 | **"STE-100"** (L12) and the **Warp Factories CTA** (L17) | A dangling reference we cannot resolve; and marketing we must not carry |

### Compete

| # | Where we win | Basis |
|---|---|---|
| W1 | **Per-skill verdicts.** It grades *the user*; nobody grades *a skill* across the sessions that used it. `inventory.json` already carries `skill_usage` (D29) — the join is unbuilt | [O] + [L:142] |
| W2 | **Deterministic, reproducible scores** — weighted sum in code, pinned corpus, confidence interval. A 4-transcript grade with no CI (§3.1) is the bar to beat | [O] |
| W3 | **Counterfactual / paired eval** — unbuilt in every column of §5; our matrix already names it Tier 3 (`A/evaluation-matrix.md:30`) | [O] |
| W4 | **Static × dynamic fusion.** "Lints clean but fired in 0 of 14 eligible sessions" is a finding no tool emits today | [I] |
| W5 | **Remediation cost + debt ratio + baseline/trend + interop** — unclaimed everywhere; L1 already picks generic-issue JSON primary, SARIF secondary | [O] + [L:142,163-167] |
| W6 | **Closing the propose→re-score loop.** They draft edits and never re-measure (§4); re-running our static lens on `proposed/` and showing the debt delta in minutes beats their headline feature | [I] |
| W7 | **Untruncated evidence** — read the real diff from git, not a 500-char transcript echo | [I] |

**Licensing.** MIT permits verbatim vendoring of `cs` / `wd` / rubric prose with the copyright notice retained
**[O]**. Three cautions: (a) rubric text is *prose* — copying it whole imports Warp's opinions with their bugs;
re-derive the dimension lists; (b) vendoring Python contradicts the repo's self-contained-Go rule (`L:194`)
**[O]**; (c) **do not vendor `assets/pierre-diffs.js`** — 1,156,623 bytes of minified third-party code with
**0** copyright or license strings **[O]**. The repo LICENSE covers Warp's code, not an unattributed bundle.

## 7. Decisions needed above my level

1. **Unit of grading.** Warp grades the user's setup; our SonarQube model grades an artifact. Per-user is
   shareable and viral; per-skill is CI-gateable. Both doubles the surface.
2. **Do we ingest session history at all?** It is the target's entire moat and every dynamic signal it has,
   but it drags in 6 undocumented formats, a privacy posture, and Python-or-reimplement (`L:194`).
3. **Vendor vs re-derive.** B5 is the highest-value MIT asset; vendoring Python conflicts with the
   self-contained-binary rule.
4. **Shareable-artifact scope.** B7 is the growth mechanic and also a whole frontend. In or out of v1?
5. **Fix `skill-audit/SKILL.md:155` now?** B3 is a live upward bias in every aggregate our current tool emits;
   replacing "when in doubt, mark 1" with an excludable not-applicable state changes historical scores.

## Handoff to synthesis

- §2.2 is the canonical inventory: **52 checks — 35 deterministic, 17 LLM-judged** — of which exactly **two**
  (L6, L7) are quality rubrics, and they carry **85% of the grade**.
- §3.1 and §3.2 are reproducible: the spike ran the published collector unmodified on synthetic Claude-Code
  histories, and the arena hunter reproduced all four rows. Re-runnable in ~2 minutes.
- Unclear, stated as such: (a) how often the code-quality judge returns `insufficient_evidence` in the field —
  the LLM stage was not run; (b) what "STE-100" is; (c) whether L1's harness gate works, since it asks a model
  to introspect its own runtime; (d) `insufficient_evidence` carries `score: 0.5` (`code-quality.md:13`) while
  `SKILL.md:97,102` excludes it — the 0.5 survives only as the all-empty fallback. That is a genuine ambiguity
  **in the source**, flagged not resolved; (e) `claude plugin eval` cells are second-hand via [D:96-103].

---

**Provenance.** Three independent researcher candidates (A, B, C) wrote to one mandate with no
cross-visibility; a fresh hunter judged them against a stated bar (`.scuba/teams/warp-skill-doctor/arena.md`,
B1–B6) after independently re-enumerating the target from `main`. **Base: candidate B** (highest citation
fidelity; its §3.1 spike was reproduced on the unmodified collector). **Fold-ins — from A:** the `SKILL.md:97`
LLM-vs-`has_code_edits` redundancy (L9 / R4) and the `pierre-diffs.js` licensing caution; **from C:**
`insufficient_evidence` as an excluded state set against `skill-audit`'s mark-1 rule (B3). All enumeration,
curve and count corrections follow the hunter's authoritative §2. **Date: 2026-09-11.**
