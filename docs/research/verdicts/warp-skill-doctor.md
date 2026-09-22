# Arena verdict — Warp `skill-doctor` finding (A / B / C)

**Judge:** fresh hunter, wrote none of the candidates · **Date:** 2026-09-11
**Bar:** `arena.md` B1–B6 (B1/B2/B3 must-pass) · **Mandate:** `mandate.md`

**Coverage.** I re-derived the target independently before reading any candidate: all 10 non-binary
files under `.agents/skills/skill-doctor/` fetched from `raw.githubusercontent.com/.../main`
(SKILL.md 171 L, scorers/efficiency.md 34, scorers/code-quality.md 37,
references/supported-harnesses.md 47, references/skill-improvements.md 32,
scripts/collect_sessions.py 1669, scripts/render_report.py 584, scripts/warp_decoder.py 388,
scripts/test_collect_sessions.py 593, scripts/test_render_report.py 252) plus both assets by size.
`main` == `b811c24` for `collect_sessions.py` (`gh api .../commits?path=...` — no drift between the
candidates' commit and mine, so every line-number disagreement below is a candidate error, not
version skew). Built my own enumeration (§2: 35 deterministic + 17 LLM-judged), then diffed all three
candidate enumerations against it item by item. Spot-checked 14 B3 cells against
`skill-audit/SKILL.md`, `skill-audit/references/evaluation-matrix.md`,
`skill-audit/scripts/*.sh`, `docs/research/skill-profiling-state-of-the-art.md:73-104`, and
`.scuba/teams/skill-audit-tool/L1-static-report.md:126,128,142,163-167,194`. Re-ran candidate B's
`skill_coverage` spike against the unmodified published collector. Second sweep added items 4, 5 and
12 of §6.

---

## 1. Per-candidate scorecard

### Candidate A — `candidates/A/finding.md`

| # | Criterion | Score | Evidence |
|---|---|---|---|
| **B1** | enumeration complete, nothing invented | **PASS (precision defect)** | Most complete taxonomy in the arena: 26 deterministic (D1–D26) + 14 LLM (J1–J14, with J2a–g / J3a–k sub-dimensions). **Only candidate** with the collector's three hard-exit validation gates (arg conflict → exit 2; per-harness source-missing → exit 1; no-source → exit 1). Nothing invented. **Defect: line numbers drift ±2 to ±10 across both scripts** — D6 `:1354` (actual 1357), D16 `:396-399` (actual 400-402), D17 `:417-420` (actual 421-424), D19 `:1566-1583` (actual 1576-1593), D14 `:1553-1560` (actual 1562-1570), D23 `:1600-1640` (actual 1620-1650), D7 Warp `:724-731` (actual 726-730). `SKILL.md` and `scorers/*` cites are exact. **Count errors: "15 collector tests + 10 renderer tests" — actual 14 and 11.** Missed: `SKILL.md:88` zero-session stop; `Skill` tool-arg branch `:393-396`; `render_report.py:563-565`; 32 MB Warp cap `:29`. |
| **B2** | det vs LLM classified | **PASS — cleanest of the three** | Correct on every contested item, and the **only** candidate to catch that the code-quality applicability gate (`SKILL.md:97`) is LLM-judged *even though* `has_code_edits` already decides it deterministically at `:421-424` and prints it in the transcript header at `:1209` (A: J4 + R3). Correctly labels the aggregation arithmetic as LLM-executed (J5). |
| **B3** | head-to-head evidenced | **PASS (weakest cite density by row-count, best cell accuracy)** | 43/133 cells carry evidence (32%). Most accurate contested cell in the arena: "skill-audit token cost = **P** via `skill-validator` token counts (`L/SKILL.md:56`)" — B and C both score this ◐ off the *judged* "token discipline" dimension and miss that `skill-validator` emits real `o200k_base` BPE counts (verified `skill-audit/SKILL.md:56`, `L1:126`). Unique row C13 (remediation cost / debt ratio = N across all six columns, tied to `L1:142`) is the row that carries the strategic point. **Defect: SOTA cites off by 1–3 throughout** — `:76-78`→77-78, `:93-95`→92-94, `:80`→83, `:101`→97, `:79` for "Claude Code only"→80; and `L1:5` is a heading, the claim lives at `L1:163-167`. |
| **B4** | verdict specific | **STRONGEST** | 7 borrow / 8 reject / 7 compete, each with a location and a why. Unique and **verified REAL**: `assets/pierre-diffs.js` is 1,156,623 bytes with **0** copyright or license strings (I grepped it) — "do not vendor, upstream terms unverified" is a live legal caution no other candidate raises. |
| **B5** | ≤4 pages, readable | **WEAKEST — over the bar** | 4,101 words / 140 table rows ≈ 6 pages. §2b+§2c alone is 40 rows. Tables-over-prose ✓, but it fails the stated length bar. |
| **B6** | candor | **STRONGEST on format** | Dedicated §6d, 4 items, each **quoting the ambiguous line** exactly as the mandate's quality bar demands (STE-100, batch determinism, curve rationale, unanchored efficiency counterfactual); reachable-range table explicitly marked **[I]**. |

### Candidate B — `candidates/B/finding.md`

| # | Criterion | Score | Evidence |
|---|---|---|---|
| **B1** | enumeration complete, nothing invented | **PASS — highest citation fidelity** | 24 checks (C1–C24). I spot-checked 12 script cites and **could not break one**: `:1357-1358`, `:1576-1593`, `:400-402`, `:421-424`, `:1562-1570`, `:324-327`, `:428-437`, `:1224-1244` (+ over-match docstring `:1232-1234`), `:30-31,376,397,403`, `:384-387`, `:1207-1209`, `render_report.py:560-571`, `:23-28,38-42` — all exact. File-inventory line counts all verified correct (incl. `pierre-diffs.js` 1531, `warp-pixel-icon.svg` 5). **Unique verified finding: `--per-skill` / `--no-skill` exist at `:96-97` but are absent from SKILL.md's flag list `:77-86`** — the agent cannot tune the sampler that drives 15% of the grade. **Missed: all three hard-exit gates** (`:1302-1307` exit 2; per-harness exit 1; no-source exit 1), `SKILL.md:88`, `render_report.py:563-565`. Loose: "11 prompt-invariant assertions" — 3 tests read SKILL.md (`:18`, `:190`, `:210`), 19 assertions run against `skill_text`. |
| **B2** | det vs LLM classified | **PASS (one real defect)** | Sharpest phrasing in the arena on the boundary: C19 `skill_coverage` = "**LLM arithmetic over deterministic data**"; C23 correctly notes `render_report.py` never recomputes. **Defect: C12 labels `has_code_edits` as the thing that "gates C17"** — it does not. `SKILL.md:97` makes the *model* decide "where the transcript shows code changes"; nothing in the scripts or in `scorers/code-quality.md` consumes `has_code_edits` as a gate (it appears only in the header line `:1209`). B has no row for `SKILL.md:97`. Minor: C3 labels `mktemp -d` isolation "deterministic" while §2.3 correctly says privacy is "enforced by prompt … **not** by code". |
| **B3** | head-to-head evidenced | **PASS — highest cite density** | 42/114 cells (37%). **All SOTA cites exact**: `[D:77-78]`, `[D:79]`, `[D:80-82]`, `[D:83]`, `[D:92-94]`, `[D:96-103]`, `[D:102-103]`. Local cites exact and deeper than the others' — `scripts/check-frontmatter.sh:24` is genuinely the `skill-validator validate structure` call; `scripts/check-quality.sh:20` is genuinely the `skillscore --json` call; `SKILL.md:87` is the exit-code + PL/PT rule-ID line. |
| **B4** | verdict specific | **STRONG** | 7 borrow / 7 reject / 6 compete, each located and reasoned. B4 (six parsers + `warp_decoder.py` as "the executable spec for each format, MIT") is the sharpest strategic borrow. Licensing note is the most operationally useful: separates code-vendoring from prose-copying, and flags the conflict with the repo's self-contained-Go rule (`L1:194` — verified, that line is exactly the runtime-dependency objection). |
| **B5** | ≤4 pages, readable | **MID** | 3,497 words / 118 table rows ≈ 5 pages. Slightly over. §2.2 as one flat 24-row table with a D/L column is the most scannable structure in the arena. |
| **B6** | candor | **STRONG** | 3 named unclears; explicitly states "I did not run the LLM stage" where §3.2 would otherwise overclaim; explicitly flags that §3.1's corpus is synthetic and that its spike is re-runnable in ~2 minutes. |
| — | **reproduction** | **VERIFIED** | I rebuilt synthetic Claude-Code histories and ran the **unmodified published collector**. B's §3.1 table reproduces **exactly, all four rows**: 19/20-1-skill → sampled 4, coverage 0.75; 14/20-2-skills → sampled 10, 0.60; 20/20-5-skills → sampled 12, 1.00; 1/20-1-skill → sampled 5, 0.20. The only empirically proven finding in the arena. |

### Candidate C — `candidates/C/finding.md`

| # | Criterion | Score | Evidence |
|---|---|---|---|
| **B1** | enumeration complete, nothing invented | **PASS (borderline — highest cite-error rate)** | Largest raw count (31 items) and **best `warp_decoder.py` precision** — `decode_task` at `:376`, field decoders `:285-374`, hand-rolled varint `:122-171`: all three exact, and no other candidate opened that file. Nothing invented. **Four cites land on the wrong construct:** "first-name-wins dedupe `:139-141`" (that range is `roots.extend`; the dedupe is `:160-161`); "over-match self-documented at `:1240-1243`" (that is the `try/resolve` block; the docstring caveat is `:1232-1234`); "Basename repo matching `:1246`" (actual `:1244`); "defaults `:92-95`" (actual `:94-97`). Plus drift: `:161-164`→166-169, `:396-400`→400-402, `:414-418`→421-424, `:1567-1574`→1562-1570, `:1462,1494,1526`→1465,1496,1527. **Two count errors, the worst in the arena: "17 parser tests" (actual 14, in 5 classes) and "12 render tests, 4 of which pin SKILL.md prose" (actual 11 tests, 3 of which read SKILL.md).** Missed: all three hard-exit gates, `SKILL.md:88`, `Skill` tool-arg branch `:393-396`, `render_report.py:563-565`. |
| **B2** | det vs LLM classified | **PASS (same defect as B)** | Item 23 "**L executing a formula**" and item 27 "L (+ `diff` D)" are the two most precise labels written by anyone. **Defect: item 11 says `has_code_edits` "gates the code-quality scorer"** — same misattribution as B; the gate is `SKILL.md:97` and is LLM-judged. C does cite `:97` in §3 and in borrow B4, so it read the line but mis-assigned the gating. Minor: `mktemp -d` labeled D. |
| **B3** | head-to-head evidenced | **PASS — lowest cite density** | 30/114 cells (26%); the most bare ✗ cells. SOTA and L1 cites are exact where present (`SOTA:77`, `:83`, `:92-95`, `:96`, `:99`; `L1:126,128,142`). **Two cells beat the field:** the unique "Repeatable eval suite" row correctly gives `claude plugin eval` a ✅ on `evals/evals.json` + isolated subagents (`SOTA:96-97` ✓), and "Rule IDs + severity: skill-audit ◐" is more accurate than A's **H** — rule IDs cover only house-policy findings (`skill-audit/SKILL.md:87`), not the spec/quality sources. |
| **B4** | verdict specific | **STRONG — best single item** | 6 borrow / 9 reject / 6 compete, each located. **Borrow B4 is the best item produced by any candidate:** `insufficient_evidence` as a first-class *excluded* state (`code-quality.md:11-13`) set against our own `skill-audit/SKILL.md:155` "when in doubt, mark 1 … " — verified verbatim, and it is a real upward bias in every aggregate our tool emits. It is the only item in the arena that converts the research into a named defect in *our* product. |
| **B5** | ≤4 pages, readable | **BEST** | 3,017 words / 102 table rows ≈ 4.3 pages — closest to the bar, tightest prose, one flat check table. |
| **B6** | candor | **BEST single item** | 4 unclears, including the arena's sharpest line: the `insufficient_evidence: 0.5` (`code-quality.md:13`) vs "exclude it" (`SKILL.md:97,102`) tension is "a genuine ambiguity in the source, **not a reading error on my part**" — I confirmed it; the 0.5 survives only as the all-empty fallback at `SKILL.md:102`. Also flags its `claude plugin eval` row as second-hand via SOTA, which itself says no canonical CLI reference was reached. |

**Must-pass summary: no candidate fails B1, B2 or B3.** None invented a check; none missed a *scorer*; every positive or contested head-to-head cell carries evidence in all three. The gaps recorded above are omissions and cite errors, listed item-by-item in §6 for the merge.

---

## 2. Authoritative enumeration — the reference the merge must use

Paths relative to `.agents/skills/skill-doctor/`. Lines verified against `main` (== `b811c24` for
`collect_sessions.py`). `cs` = `scripts/collect_sessions.py`, `rr` = `scripts/render_report.py`,
`wd` = `scripts/warp_decoder.py`.

### 2a. LLM-judged (17)

| # | Item | Location |
|---|---|---|
| L1 | Harness self-identification gate; stop before creating REPORT_DIR or reading history if unsupported | `SKILL.md:19`; `references/supported-harnesses.md:16-20` |
| L2 | Conversation-scope interview (3 options with repo, 2 without) | `SKILL.md:31-42` |
| L3 | Skill-scope interview (project+global / project only) | `SKILL.md:44-51` |
| L4 | Zero-result gate: `sessions_sampled == 0` → tell user and **stop**; `skills_found == 0` → **continue**, coverage = 0 | `SKILL.md:88` |
| L5 | Batching policy: ≤50 transcripts one batch; >50 → parallel batches of 20; local child agents only | `SKILL.md:92` |
| L6 | **Efficiency scorer** — 4 labels `highly_efficient` 1 / `mostly_efficient` 0.8 / `mostly_inefficient` 0.4 / `highly_inefficient` 0.2; 7 dimensions (rework, cost-to-human, information gathering, routine-step overhead, batching, flailing, verification timing); "not an exhaustive checklist" | `scorers/efficiency.md:4-16` (labels), `:20` (open-ended), `:24-30` (dimensions), `:32-34` (reason format) |
| L7 | **Code-quality scorer** — 3 labels `approve` 1 / `block` 0.2 / `insufficient_evidence` 0.5; binary verdict, one defect ⇒ block; 11 dimensions (design, correctness, complexity, repo conventions, code smells, tests, naming, comments, diff hygiene, documentation, mid-run corrections); efficiency + instruction-following explicitly out of scope | `scorers/code-quality.md:4-13` (labels), `:17` (binary), `:21-31` (dimensions), `:33` (out of scope), `:35-37` (reason format) |
| L8 | Per-scorer record contract: label + numeric score + 1–3 sentence reason citing transcript specifics | `SKILL.md:97` |
| L9 | **Code-quality applicability gate** — apply "only where the transcript shows code changes", else `insufficient_evidence` and exclude | `SKILL.md:97` — **LLM-judged**, and duplicative of deterministic D18 |
| L10 | Aggregation arithmetic: `raw_efficiency` mean; `raw_code_quality` mean excl. `insufficient_evidence`, 0.5 if none; `curve(s)=0.5+0.5*s`; `skill_coverage`; `overall = 0.5·E + 0.35·Q + 0.15·C` | `SKILL.md:101-107` — executed by the model; **no script computes or validates it** |
| L11 | `failed_conversations` = any applicable **raw** score < 0.5; `insufficient_evidence` never fails | `SKILL.md:109` |
| L12 | `top_findings` = 3 most impactful patterns, "following the STE-100 standard" | `SKILL.md:113` (STE-100 undefined repo-wide) |
| L13 | `suggestions` must cite failed session + scorer + moment; never-fired installed skill ⇒ "usually a description problem" | `SKILL.md:114` |
| L14 | Edit drafting: read current skill from `inventory.json`, write full improved file to `proposed/<skill>/SKILL.md`, `diff -u` into the `diff` field; never modify real skill files | `SKILL.md:116-126` |
| L15 | Improvement method: cluster by root cause → prioritize frequency × severity → verify against the live repo, drop what does not verify → state the behavioral rule + owning surface first → prefer **replace** over **append** | `references/skill-improvements.md:5-9` |
| L16 | Suggestion admissibility: 4 must-all-be-true file-when conditions; 4 do-not-file conditions; "when nothing clears this bar, open no change and say why — that is a success" | `references/skill-improvements.md:15-20`, `:22-27`, `:29` |
| L17 | Step-6 output contract: speak the grade + 3 findings, then two exact closing lines incl. the Warp Factories CTA | `SKILL.md:162-171`; `cta_url` at `:152` |

### 2b. Deterministic (35)

| # | Item | Location |
|---|---|---|
| D1 | Arg conflict `--all-conversations` + `--repo` → **exit 2** | `cs:1302-1307` |
| D2 | Explicit-harness source-missing → **exit 1** (claude / codex / warp / pi / grok / zcode) | `cs:1368-1373`, `:1402-1404`, `:1445-1447`, `:1476-1478`, `:1507-1509`, `:1538-1543` |
| D3 | No supported source found at all → **exit 1** | `cs:1545-1550` |
| D4 | Skill discovery: 3 project roots + 6 global roots, glob `*/SKILL.md` | `cs:132-152`, `:158` |
| D5 | First-name-wins collision dedupe | `cs:160-161` |
| D6 | `description:` regex, stripped, truncated to 300 chars; + `bytes`, `modified_at` | `cs:166-169`, `:170-176` |
| D7 | `--days` cutoff (default 45) applied to mtime | `cs:94`, `:1327` |
| D8 | Per-source session discovery: codex `:180-194`, claude `:197-216`, warp `:547-591` + `:610-655`, pi `:852-867`, grok `:992-1007`, zcode `:1099-1114` | `cs` as listed |
| D9 | Warp cross-channel dedup: newest row per `conversation_id` | `cs:610-655` |
| D10 | Warp conversation size cap 32 MiB | `cs:29` |
| D11 | Warp protobuf-free decode: varint reader + field parser `wd:122-171`; per-message decoders `wd:285-374`; `decode_task` `wd:376` | `wd` as listed |
| D12 | Warp skill-name-from-reference resolution | `cs:705-718` |
| D13 | Sidechain/child exclusion unless `--include-subagents` (claude `isSidechain`; warp `parent_agent_id`/`parent_conversation_id`) | `cs:324-327`, `:726-730` |
| D14 | `looks_injected` filter — leading `<` + one of 9 tag names | `cs:428-437` |
| D15 | Truncation: `MAX_MSG_CHARS` 1500, `MAX_TOOL_CHARS` 500 | `cs:30-31`, `:219-223`; applied `:376`, `:397`, `:403` |
| D16 | `TranscriptBuffer` head/tail bounding: >160 entries → head 100 + omitted-note + tail 40 | `cs:32-34`, `:256-280`; re-clipped at render `:1214-1217` |
| D17 | Per-session stats: `user_turns` `:405-406`; `assistant_turns` deduped by message id `:360-364`; `tool_calls` `:380`; `repeated_tool_calls` = SHA-1(name+args) count>1 `:384-387`; `error_outputs` = `is_error` OR substring `error`/`failed`/`traceback` in first 2000 chars lowercased `:398-402` | `cs` as listed |
| D18 | `has_code_edits` = tool name ∈ `CLAUDE_CODE_EDIT_TOOLS` / `GENERIC_EDIT_TOOLS`, OR arg contains one of 6 `CODE_EDIT_HINTS` | `cs:36-38`, `:421-424`; generic path `:1193-1197` |
| D19 | `detect_skill_candidates` regex on tool args: `(^\|/)skills/+([^/]+)/+` and `"skill"\|"name"\|"bundled_skill_id"` | `cs:283-291` |
| D20 | Direct capture of the `Skill` tool's `skill` argument | `cs:393-396` |
| D21 | `detect_skills_from_entries` — 5 literal markers per installed name over `skill`/`tool:*` entries | `cs:1280-1297` |
| D22 | Detections intersected with the installed set (unknown names dropped) | `cs:1562-1570` |
| D23 | Repo attribution: cwd inside repo **OR** basename equality **OR** repo name in path parts; over-match caveat self-documented | `cs:1224-1244`; caveat `:1232-1234` |
| D24 | `--all-conversations`: infer repos via `git rev-parse --show-toplevel` per cwd, then re-discover skills | `cs:1251-1277`, `:1551-1561` |
| D25 | Scoreability filter: drop sessions with `assistant_turns < 1` **or** `tool_calls < 1` | `cs:1357`, `:1391`, `:1433`, `:1465`, `:1496`, `:1527` |
| D26 | Newest-first sort and stable session key | `cs:1572-1574` |
| D27 | Sampling: ≤`--per-skill` 3 per skill, then ≤`--no-skill` 4 skill-free, global cap `--max-sessions` 12 | `cs:95-97`, `:1576-1593` |
| D28 | Transcript rendering with the deterministic stats header line | `cs:1201-1221` (stats line `:1207-1209`) |
| D29 | `inventory.json` schema + console summary | `cs:1620-1650`, `:1652-1665` |
| D30 | `GRADES` letter thresholds: A+ .97 / A .93 / A− .90 / B+ .87 / B .83 / B− .80 / C+ .77 / C .73 / C− .70 / D .60 / else F | `rr:24-27`, `grade_for` `:38-42` |
| D31 | `pct()` = `round(score*100)` | `rr:45-46` |
| D32 | Grade defaulted from `overall`; **`overall` itself is never recomputed or validated** | `rr:231`, `:567` |
| D33 | Missing `report.json` → **exit 1** | `rr:563-565` |
| D34 | HTML render: diff clamp 320 px, collapsed long diffs, 1200×675 client-side share PNG, `open_report` browser launch | `rr:35`, `:83-110`, `:228-314`, `:315-540`, `:71-77` |
| D35 | Self-tests: **14** collector tests in **5** classes (`test_collect_sessions.py:44,245,374,435,484`); **11** render tests in 1 class; **3** of those read `SKILL.md` as golden prose — `:18` startup contract, `:190` output labels, `:210` curve + weights + failed-conversations rule | `scripts/test_*.py` |

**Nothing in D1–D35 evaluates skill content.** The only static read of a skill is D6 (name, path,
description regex, bytes, mtime). No frontmatter validation, no token counting, no link resolution,
no security scan, no duplication, no rule IDs, no severities, no SARIF, no exit-code gating on
quality.

### 2c. Curve verdict — authoritative

All three candidates state **both** numbers and all three are right; they describe different
quantities, and the merged text must say which is which or it reads as a contradiction.

| Quantity | Value | Derivation |
|---|---|---|
| Mathematical floor of `curve()` | **0.50** | `curve(0) = 0.5 + 0.5·0` (`SKILL.md:103`) — **unreachable**: no rubric label scores 0 |
| Attainable floor of `efficiency` and `code_quality` | **0.60** | min label = 0.2 (`efficiency.md:16`, `code-quality.md:9`) → `curve(0.2) = 0.60` |
| Attainable floor of `overall` | **0.51** | `skill_coverage` is **not** curved (`SKILL.md:106`) and can be 0 → `0.5(0.6)+0.35(0.6)+0.15(0)` = 0.51 → **F** (< 0.60, `rr:27`) |
| Range at coverage 0 | **[0.51, 0.85]** | F, D, C and B all reachable; **B (0.85) is the cap for a perfect run with no skills** |
| Range at coverage 1.0 | **[0.66, 1.00]** | worst-possible rubrics still grade **D** |
| A+ | requires coverage ≥ 0.80 even with perfect rubrics | `0.5+0.35+0.15c ≥ 0.97` |

So: **0.60 = per-component floor, 0.51 = overall floor, 0.50 = a floor the function has but the
rubrics cannot reach.** B's "the F/D/C bands are nearly unreachable" is the one overstatement —
F/D/C/B are all reachable at low coverage; it is the **A** band that coverage gates.

---

## 3. Ranking

| Rank | Candidate | One-line basis |
|---|---|---|
| **1** | **B** | Highest citation fidelity in the arena — I could not break a single one of 12 spot-checked script cites — plus the only **reproduced** finding (I re-ran its sampler-bias spike on the unmodified collector and got all four rows exactly) and a unique verified defect (undocumented `--per-skill`/`--no-skill`). Its two gaps are narrow and cheap to patch. |
| **2** | **A** | The most complete taxonomy (40 enumerated items; the only one with the three hard-exit gates and the only one to catch the `SKILL.md:97` LLM/deterministic redundancy), the strongest verdict section, and a unique real licensing finding — but the loosest line numbers across two scripts *and* the SOTA doc, and ~6 pages against a 4-page bar. |
| **3** | **C** | Tightest and most readable, best `warp_decoder` precision, the single best borrow item and the single best candor line — but the highest cite-error rate (four cites point at the wrong construct) and two wrong test counts. |

The gap 1→2 is narrow and is entirely about *verifiability*, not insight; A has more insight per page
and less trust per citation. The gap 2→3 is likewise narrow.

---

## 4. Recommended base: **B**, by soundest shape

- **§2.2 is one flat 24-row table with a `Deterministic or LLM-judged` column per row.** That single
  column is what makes B2 auditable at a glance; A achieves the same separation only by splitting
  into two tables (§2b + §2c, 40 rows) and pays for it with the length overrun.
- **Its cites survive verification unchanged**, so the merge inherits a spine it does not have to
  re-derive. A's and C's script cites must be re-walked line by line before they can be trusted.
- **§3 is the cleanest scoring-model section**: formula block → simulated grade table → two
  *empirical* subsections. §3.1 is reproducible and I reproduced it.
- **§2.3 already carries the inputs/outputs/invocation contract plus an `Undocumented` and an
  `Ambiguous` row**, which is where the merged §6 candor items can land without new structure.

Fold A's completeness and C's best two items into B's frame; do not restructure.

---

## 5. Single strongest idea to fold in, per runner-up

**From A (rank 2):** **the `SKILL.md:97` redundancy finding (A's J4 + R3).** The code-quality
applicability gate — "apply the code-quality scorer only where the transcript shows code changes" —
is LLM-judged, *even though* `has_code_edits` already computes exactly that deterministically at
`cs:421-424` and is printed into the header the judge reads at `cs:1207-1209`. This is the one place
where the target throws away a deterministic answer and re-asks a model for it. It simultaneously
(a) fixes B's only B2 error, (b) is the cleanest single illustration of the det-vs-LLM boundary the
mandate asks us to draw, and (c) is a design rule our own tool should adopt: never re-ask a judge a
question a script has already answered.

**From C (rank 3):** **borrow item B4 — `insufficient_evidence` as a first-class *excluded* state**
(`scorers/code-quality.md:11-13`, excluded at `SKILL.md:97,102`) set directly against our own
`skill-audit/SKILL.md:155` "When in doubt, mark the dimension as 1 (present but weak)", which forces
a number into every aggregate and biases it upward. Verified verbatim on both sides. It is the only
item in the arena that converts this research into a named, fixable defect in *our* product rather
than an observation about Warp's.

---

## 6. Defects the merged finding must fix regardless of base

| # | Defect | Fix invariant |
|---|---|---|
| 1 | **Three hard-exit validation gates missing from B and C** — `cs:1302-1307` (`--all-conversations` + `--repo` → exit 2), per-harness source-missing → exit 1 (`cs:1368-1373`, `:1402-1404`, `:1445-1447`, `:1476-1478`, `:1507-1509`, `:1538-1543`), no-source → exit 1 (`cs:1545-1550`) | Every nonzero-exit path in the target is a check and must appear in the enumeration. |
| 2 | **`SKILL.md:88` zero-result gate missing from all three** — `sessions_sampled == 0` → stop; `skills_found == 0` → continue with coverage 0 | The enumeration must cover the control-flow gates in the prompt, not only the scorers. |
| 3 | **`Skill` tool-arg direct capture (`cs:393-396`) present only in B** | Skill attribution has three mechanisms (regex `:283-291`, direct arg `:393-396`, marker match `:1280-1297`), not two. |
| 4 | **`render_report.py:563-565` (missing `report.json` → exit 1) missing from all three** | Same invariant as #1. |
| 5 | **`MAX_WARP_CONVERSATION_BYTES` 32 MiB cap (`cs:29`) missing from all three** | Every hard limit that can silently drop input is a check. |
| 6 | **`has_code_edits` misattributed as the code-quality gate (B's C12, C's item 11)** | The gate is `SKILL.md:97` and is LLM-judged; `has_code_edits` reaches the judge only as a header string at `cs:1209`. |
| 7 | **Test counts wrong in all three** — A "15 + 10", B "11 prompt-invariant assertions", C "17 + 12, 4 pinning prose" | Truth: **14** collector tests in **5** classes; **11** render tests; **3** read `SKILL.md` (`:18`, `:190`, `:210`); 19 assertions run against `skill_text`. |
| 8 | **Line-number drift** — every `collect_sessions.py`, `render_report.py` and SOTA cite in A, and C's four wrong-construct cites (`:139-141` dedupe, `:1240-1243` over-match caveat, `:1246` basename match, `:92-95` defaults) | Re-cite against §2 above; `main` == `b811c24`, so there is no version excuse for drift. |
| 9 | **63–74% of head-to-head cells carry no citation in every candidate** (A 32% cited, B 37%, C 26%) — almost all of them negative cells | A "no" is a claim. Either cite the absence (a grep, a doc section that would have said so) or mark the cell UNKNOWN. |
| 10 | **The curve must be stated with its scope** | Publish §2c's table: 0.50 = unreachable function floor, 0.60 = per-component floor, 0.51 = overall floor at zero coverage, [0.51,0.85] at coverage 0, [0.66,1.00] at coverage 1.0, A+ requires coverage ≥ 0.80. Drop "F/D/C nearly unreachable". |
| 11 | **`insufficient_evidence` internal inconsistency carried only by C** — `code-quality.md:13` assigns it `score: 0.5` while `SKILL.md:97,102` excludes it; the 0.5 survives only as the all-empty fallback | Keep C's framing verbatim: source ambiguity, flagged not resolved. |
| 12 | **A's unique licensing caution must survive the merge** — `assets/pierre-diffs.js`, 1,156,623 bytes, **0** copyright/license strings (verified by grep); repo LICENSE is MIT © 2026 Denver Technologies, Inc. | MIT covers the repo, not a vendored minified third-party bundle with unverified upstream terms. Do not vendor that file. |
| 13 | **B's unique undocumented-flag finding must survive** — `--per-skill` / `--no-skill` at `cs:96-97` absent from `SKILL.md:77-86` | The agent cannot tune the sampler that produces 15% of the grade. |
| 14 | **A's `L1:5`, `SOTA:79`, `SOTA:80`, `SOTA:101` mis-cites** | The standalone/generic-issue-JSON recommendation is `L1:163-167`; the Stats-tab line is `SOTA:83`; the HTML viewer is `SOTA:97`; "Claude Code only" is `SOTA:80`. |
