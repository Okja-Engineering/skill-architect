# Arena verdict — SkillSpector learning + skill-architect evaluation

**Judge:** fresh hunter, wrote no candidate. **Date:** 2026-09-12.
**Reference clone:** `github.com/NVIDIA/SkillSpector` @ `8421a2eb4e7bb98af2557bd4b54325d6ad90a3da`
(2026-09-11), read-only at
`/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-cursor-profiler/d8adb395-4eb1-43d1-98af-41b354d56449/scratchpad/skillspector-judge`.
Marked `SS:`. `SA:` = `/Users/matthewvandusen/Development/Auraprix/skill-architect`.
`CP:` = `/Users/matthewvandusen/Development/Auraprix/cursor-profiler`.

**Coverage.** 27/27 registered analyzer modules enumerated from source; 113/113 rule IDs
re-derived independently (scripted sweep, not read from any candidate); 3/3 `requires_api_key`
modules verified; 2/2 contested behavioural claims tested against source + the repo's own tests;
**every** stale/missing cell in all three section-6 tables (A 19, B 17, C 20 cells) checked against
the actual `SA:` line and the named contradicting source; 37/37 improvement rows checked for the B3
triple (Claude Code value / Cursor value / effort); repo metadata and 6 cited issues verified via
`gh api`. Two full sweeps; the second added items 6, 9 and 13 in section 6 below.

---

## 1. Per-candidate scorecard

| # | Criterion | A | B | C |
|---|---|---|---|---|
| **B1** | Detector catalogue complete / path:line / static-vs-LLM correct | **FAIL** | **FAIL** | **PASS** |
| **B2** | Every stale/missing verdict dual-evidenced | **FAIL** | **FAIL** | **PASS** (notes) |
| **B3** | Improvements specific / CC + Cursor value separately / effort | PASS | PASS | PASS |
| **B4** | Calls reasoned / licensing stated / no reimplementation | Strong | Strong | Strong |
| **B5** | One sitting / <= 5 pages / tables | Weakest (~9.4 pp) | Best (~7.9 pp) | Mid (~8.3 pp) |
| **B6** | Corrects/confirms L1 explicitly | Strongest form, wrong figure | Adequate | Good |
| **B7** | Candor on the unverifiable | Strong, best denominator | Strong | Strongest (actionable) |

### Candidate A — must-pass failures

**B1 FAIL, six items.**

1. Count wrong: **111**. True count **113** (section 2). `A/finding.md:77`.
2. **Asserts a rule that exists does not exist.** `A/finding.md:66`: "No AE1/AE7 in v2.11.2".
   AE1 is emitted at `SS:src/skillspector/nodes/finalize_inspection_ledger.py:77`
   (`rule_id="AE1"`, HIGH, confidence 1.0).
3. Module count wrong: **25** (`A/finding.md:9`, `:38`). True **27** — `grep '^ANALYZER_ID'`
   over `SS:src/skillspector/nodes/analyzers/*.py` returns 27 files, all exporting `node()`.
   A lists `osv_client.py:814` and `whitespace_padding.py` as analyzer rows (`A/finding.md:44,50`);
   **neither exports `ANALYZER_ID`**, so neither is registered by
   `SS:src/skillspector/nodes/analyzers/__init__.py:50-56`.
4. Internal arithmetic: A's table 3a sums to **101** rule IDs, not the 100 it states
   (`A/finding.md:38,77`).
5. **Static/LLM behaviour misclassified — the fatal one.** `A/finding.md:9` "a **meta-analyzer
   that can delete static findings**"; `:26` "**drops unconfirmed static findings**"; `:160`
   builds an *Avoid* on it. This is false. `SS:.../meta_analyzer.py:386` — *"Enrich deterministic
   findings **without letting LLM output suppress them**"*; `:395-397` — *"Every deterministic
   finding remains in primary output. Unconfirmed findings receive an annotation tag"*; the
   `else` branch at `:447-480` **appends** the finding with an `llm-unconfirmed` tag and
   `continue`s. Pinned by the repo's own test
   `SS:tests/nodes/test_meta_analyzer.py:171 test_rejected_finding_is_retained_as_unconfirmed`
   — docstring: *"LLM disagreement may annotate but cannot erase deterministic evidence."*
   A's cited `:811` is the *batch-miss* path, which also preserves (`:809-811`).
6. TP4 placed in the static column (`A/finding.md:63`); it is `_TP4Analyzer`, LLM-gated at
   `SS:.../mcp_tool_poisoning.py:1712`.

**B2 FAIL.** `A/finding.md:124` "Missing — **No before/after proof loop**: the draft is produced
but nothing re-runs the audit on the draft and shows the delta", marked **[I]** with no doc line.
Contradicted three times in the very file A evaluated:
`SA:skills/skill-rewrite/SKILL.md:95` "Re-run `skill-audit` and confirm no dimension scored lower
than before."; `:132` "Re-audit scores are >= the original on every dimension."; `:140` "Every
rewrite must be followed by a re-audit." A then makes this false gap improvement #7
(`A/finding.md:155`) and one of its two "sharpest differentiators" (`:182`).
Secondary B2 defects: `A/finding.md:121` cites `SA:README.md:36` for Cursor "Planned" — line 36 is
**Codex**; Cursor is `:35`. `A/finding.md:120` calls `SA:PRINCIPLES.md:26` stale — that line is a
normative sequencing belief ("Static checks come first"), not a status claim, and `compare.go`
does not contradict it.

**What A got right and no one else did** (keep these): the `SA:CHANGELOG.md:12` +
`SA:RELEASE_NOTES.md:8` "Claude Code has no skill-level events" vs `SA:docs/profiler-spec.md:200,209`
contradiction — verified exact on both sides, the single best-evidenced stale verdict in the arena;
the README AST severity drift (`SS:README.md:466-478` says AST1 CRITICAL / AST3 HIGH / AST4 HIGH /
AST7 MEDIUM, `SS:.../behavioral_ast.py:148-158` emits HIGH / MEDIUM / MEDIUM / LOW, and the README
heading reads "Behavioral AST (**9** patterns)", omitting AST10) — verified; the install-cost
reality (section 6 item 10); and the only correct open-issue denominator (section 6 item 5).

### Candidate B — must-pass failures

**B1 FAIL, one item — but the same fatal one.** `B/finding.md:35` "LLM per-file **filter** +
enrich"; `:45` "on the success path a model verdict of `is_vulnerability: false` **removes the
finding**", citing `meta_analyzer.py:379-390`. Those exact lines are the `apply_filter` docstring
that says the opposite (see A-5 above). B then ships improvement #11 "**Do not let an LLM delete
deterministic findings**" (`B/finding.md:157`) as an *Avoid* — inverting a mechanism that is in
fact the strongest thing to **borrow**.
Everything else in B's section 3 is sound: 27 modules OK, 24 static / 3 LLM OK, 113 total OK
(though B does not separate the 112 finding-emitting from the 1 informational SSR-1, which it
itself notes emits no finding at `:83`), "41 undocumented" OK (113 - 72; I independently counted
**72** unique IDs in the README tables across 17 category headings — so the README's own "71" is
off by one against its own tables).

**B2 FAIL, one item.** `B/finding.md:136` "Missing — **No before/after evidence loop: the draft is
never re-scored** or profiled". False on "re-scored" per `SA:skills/skill-rewrite/SKILL.md:95,132,140`.
True only on "profiled". B cites `SA:profiler/compare.go` as the contradicting source, but the doc
line it would contradict does not exist.
Otherwise B's section 6 is the most rigorously dual-cited of the three — all verified exact:
`SA:README.md:39-49` documents only `probe`/`capture` while `SA:profiler/cmd/main.go:23-41`
dispatches nine commands (`probe capture compare ingest doctor hooks analyze experiment version`) OK;
`SA:PRINCIPLES.md:79` cites NVIDIA SkillEvaluator but not SkillSpector OK; `SA:CHANGELOG.md:5`
`## Unreleased` empty against five merged commits OK; `SA:skills/skill-audit/SKILL.md:46-47`
hard `exit 1` OK.

**What B got right and no one else did** (keep these): **per-rule diminishing returns** —
`SS:.../report.py:447-450` "the first occurrence of a rule_id contributes full points, the second
half, the third a quarter; occurrences beyond the third are ignored", plus confidence scaling
(`:452-454`) and the 1.3x applying **only** to executable scripts so documentation is not punished
(`:459-462`). A and C both cite `SS:README.md:521-536` here and miss all of it. And the **dead rule
families**, verified two ways: `SS:.../mcp_rug_pull.py:422` reads `manifest.get("version")`, but
`SS:.../build_context.py:1445-1503` only ever sets `name`, `description`, `triggers`,
`permissions`, `allowed-tools`, `parameters` — **`version` is never populated, so RP3 is
unreachable**; TR1-3 gate on `manifest["triggers"]`
(`SS:.../static_patterns_supply_chain.py:1706-1711`), populated from a `triggers:` frontmatter key
that the Agent Skills spec does not define. Confirmed by maintainer-open issues #458 and #472
(fetched; titles match B's description verbatim).

### Candidate C — passes both must-pass criteria

**B1 PASS.** `C/finding.md:10,70`: "**112 finding-emitting rule IDs + 1 informational (SSR-1)**" —
exactly matches my independent derivation (section 2). "27 registered nodes ... 24 static nodes,
3 LLM nodes, 1 hybrid (TP4), 1 LLM join node" OK. **C is the only candidate that read the LLM
invariant correctly**: `:26` "an unconfirmed finding is **tagged `llm-unconfirmed`, not dropped**"
(`meta_analyzer.py:448-470`) and `:32` "the LLM can raise confidence and add findings but cannot
remove a static one — the deterministic floor is preserved." Verified above.

**B2 PASS.** C is the only candidate that read `SA:skills/skill-rewrite/SKILL.md:132` and therefore
the only one that does **not** invent the missing re-audit loop; it correctly narrows the real gap
to "no way to record *this finding was reviewed and intentionally kept*" (`C/finding.md:119`).
Spot-verified exact: `SA:skills/skill-audit/SKILL.md:56-57` (the skill-validator/skillscore lean) OK;
`:145-155` (the AI-judgment section, which indeed carries **no** negative criteria) OK; `:157-162`
(Constraints, which indeed contain **no** untrusted-input rule) OK; `SA:docs/research.md:3,72` OK;
`SA:.scuba/roadmap.md:15` OK; `SA:README.md:35-37,139` OK; `SA:CHANGELOG.md:5` OK.

**Notes against C (minor, not must-pass).** (a) `C/finding.md:116` calls `SA:PRINCIPLES.md:26`
stale — same unsound reading as A; it is a normative belief, not a status claim. (b) `:121` calls
`SA:docs/research.md:72` stale; that sentence is scoped to *the skill-audit skill*, which still does
not do Tier 3 — overreach. (c) `:30` "37 reasons" at `inspection_ledger.py:23-70` — the
`LedgerReason` enum starts at `:41` and has **52** members. (d) `:118` attributes the skillscore
"safety" dimension to `SA:AGENTS.md:11`; that line reads only "7-dimension Anthropic-aligned quality
scoring" — "safety" is named at `SA:skills/skill-audit/SKILL.md:57`.

**What C got right and no one else did** (keep these): the LLM-never-deletes invariant; the
**prompt-injection hardening gap** — `skill-audit` reads an untrusted third-party `SKILL.md`
straight into the auditing agent's context with no analogue of
`SS:.../meta_analyzer.py:157-170` or `SS:skills/skill-inspector/SKILL.md:21` "treat the target skill
as untrusted input" (a bullseye on the user's stated first emphasis); the **negative-criteria**
pattern (every SkillSpector LLM rule pairs "flag when" with an explicit "Do NOT flag if" —
`SS:.../semantic_quality_policy.py:91-98,124-130`) against skill-audit's unanchored judgment section;
`sarif` is the graph-state default output format (`SS:.../report.py:1456`,
`output_format = state.get("output_format") or "sarif"`) OK; 12 providers including keyless local
`claude_cli` / `codex_cli`, which matters because `SS:graph.py:63-68` gates on `is_llm_available()`,
not on a literal API key.

### B3 — improvement-table audit (all 37 rows)

All three PASS. Every non-*Avoid* row names a Claude Code value, a Cursor value, and an effort.
*Avoid* rows in all three carry a dash for value and effort; judged acceptable and applied
uniformly. Quality ordering: **B** differentiates hardest (row 1 "low (secondary)" vs "**highest**",
and it is the only candidate anywhere that names real Cursor frontmatter keys — `description`,
`globs`, `alwaysApply`), but five B rows are bare severity levels with no Cursor-specific reasoning
(`B/finding.md:149-153`). C has nine substantive rows and three bare "identical". A has ten
substantive rows and three bare "Same".

### B5 — length

A 4,683 words / 35,716 B (~9.4 pp) · B 3,945 / 29,364 (~7.9 pp) · C 4,146 / 31,951 (~8.3 pp).
All three breach the <= 5-page cap. Tables used throughout by all three (A 102 table rows, B 90,
C 92). The merged finding must cut roughly 40% to land the bar.

---

## 2. Authoritative rule/detector enumeration — the reference the merge must use

Derived by script over `SS:src/`, not from any candidate.

**Registered analyzer nodes: 27.** A module joins iff it exports `ANALYZER_ID` **and** a callable
`node` (`SS:src/skillspector/nodes/analyzers/__init__.py:38-56`, `pkgutil.iter_modules`).
`grep -c '^ANALYZER_ID'` over that directory = 27. Split **24 static / 3 LLM**. The three LLM
modules are exactly those exporting `requires_api_key = True`:
`semantic_developer_intent.py:45`, `semantic_quality_policy.py:45`, `semantic_security_discovery.py:47`.
`osv_client.py`, `whitespace_padding.py`, `common.py`, `pattern_defaults.py`, `static_runner.py`
are **helpers, not nodes** — none exports `ANALYZER_ID`. `osv_client.py:814` is the only
`is_available()` in the package.

**True rule-ID count: 113 distinct IDs — 112 finding-emitting + 1 informational.**

| Layer | IDs | Count | Source |
|---|---|---|---|
| `RULE_ID_TO_CATEGORY` (18-member `PatternCategory` enum) | P1-P9, E1-E5, PE1-PE3, SC1-SC9, EA1-EA5, OH1-OH3, MP1-MP3, TM1-TM4, RA1-RA2, TR1-TR3, TT1-TT6, YR1-YR4, LP1-LP4, TP1-TP4, AS1-AS3, AR1-AR3, SSRF1-3, DS1-DS4 | 77 | `SS:.../pattern_defaults.py:158-243` |
| `DEFAULT_EXPLANATIONS` only (uncategorised) | AST1-AST10, BH1-BH3 | 13 | `SS:.../pattern_defaults.py:47-157` |
| Emitted, absent from `pattern_defaults` | PE4, PE5 | 2 | `SS:.../static_patterns_privilege_escalation.py:600,623` |
| MCP rug-pull | RP1, RP2, RP3 | 3 | `SS:.../mcp_rug_pull.py:229,393,422` |
| Artifact integrity + ledger coverage | AE2-AE6; **AE1** | 6 | `SS:.../artifact_integrity.py:776,792,809,820,830`; `SS:.../finalize_inspection_ledger.py:77` |
| **Static subtotal (finding-emitting)** | | **101** | |
| Informational only — emits `structured_summaries`, `findings: []` | SSR-1 | 1 | `SS:.../structured_skill_roles.py:43,56-61` |
| **Static subtotal (all IDs)** | | **102** | scripted sweep of ID-shaped literals in `SS:src/` returns 105, minus the 3 example IDs in `suppression.py:25,26,38,40,46` |
| LLM — prompt-declared only | SSD-1..4, SDI-1..4, SQP-1..3 | 11 | `semantic_security_discovery.py:73,79,84,90`; `semantic_developer_intent.py:78-81`; `semantic_quality_policy.py:80-82` |
| **TOTAL** | | **113** (112 finding-emitting) | |

**Hybrid.** TP4 (description-vs-behaviour mismatch) lives inside the otherwise-static
`mcp_tool_poisoning.py` but runs through `_TP4Analyzer` and is skipped under `--no-llm`
(`SS:.../mcp_tool_poisoning.py:1707-1712`). Count it static-module / LLM-execution.

**Nuance no candidate states.** The 11 LLM rule IDs are **prompt conventions, not enforced
constants**: `LLMFinding.rule_id` is a free-form `str` with no enum or validator
(`SS:src/skillspector/llm_analyzer_base.py:301`), and severity is likewise a model choice
(`:303`). A model may return any string. Treat the LLM catalogue as advisory.

**Reconciling the three reported counts.** A **111** = missed AE1 **and** an off-by-one in its own
sum (its table lists 101, not the 100 it claims). B **113** = correct total, but folds the
informational SSR-1 into the finding-emitting figure. C **112 + 1** = correct and the most precise
framing. README **71** (`SS:README.md:28,358`) is stale against SkillSpector's own tables, which
enumerate **72** unique IDs across 17 category headings — so **41 of 113 IDs are undocumented**
(B's figure, confirmed).

**The LLM pass may add but never delete — VERIFIED TRUE.** `apply_filter`
(`SS:.../meta_analyzer.py:381-480`): docstring `:386` "*Enrich deterministic findings without
letting LLM output suppress them*"; `:395-397` "*Every deterministic finding remains in primary
output. Unconfirmed findings receive an annotation tag; confirmed findings may gain an explanation
or higher confidence, but are never downgraded.*"; the unconfirmed branch `:447-480` appends with
`llm-unconfirmed`. Low-confidence (`< 0.6`, `:411`) and `is_vulnerability: false` (`:408`) verdicts
both fall to that branch. Pinned by `SS:tests/nodes/test_meta_analyzer.py:171` and `:183`. All other
paths preserve: `--no-llm` -> `_fallback_filtered` (`:243-280`), LLM failure ->
`_passthrough_with_defaults` (`:283-290`, "*A security tool should fail-closed*"), batch miss ->
`:809-834`. **Verdict: C is right; A and B are wrong.**

**Two rule families never fire on real skills — VERIFIED TRUE (4 IDs, not 6).**
**RP3 is unreachable code**: `SS:.../mcp_rug_pull.py:422` reads `manifest.get("version")`;
`SS:.../build_context.py:1445-1503` sets only `name`, `description`, `triggers`, `permissions`,
`allowed-tools`, `parameters`. **TR1-TR3** gate on `manifest["triggers"]`
(`SS:.../static_patterns_supply_chain.py:1706-1711`), populated from a `triggers:` frontmatter key
no supported skill spec defines — the same file annotates `allowed-tools` as "(Agent Skills
standard)" at `build_context.py:1462` and gives `triggers` no such note. Open issues #458 and #472
confirm both, in the maintainers' own words. B's improvement #12 says "6 rules"; it is **4**.

**Repo facts (`gh api`, 2026-09-12).** 16,985 stars / 1,444 forks / Apache-2.0 / created 2026-03-21 /
pushed 2026-09-12. **True open issues = 61** (`search/issues is:issue is:open`); the
`open_issues_count=135` field that B and C quote counts pull requests too. **21** open issues match
"false positive" — i.e. ~**34% of the open issue backlog**, not ~16%. Not on PyPI
(`pypi.org/pypi/skillspector/json` -> **404**, re-verified); install is
`uv tool install git+https://github.com/NVIDIA/skillspector.git` (`SS:README.md:46`);
`requires-python = ">=3.12,<3.15"` (`SS:pyproject.toml:11`); Docker image is built locally
(`SS:README.md:82`). No Cursor anywhere: the only `Cursor` string in `src/ docs/ skills/
extensions/ README.md` is `nextCursor` at `SS:src/skillspector/mcp_registry.py:396`.
`extensions/` is a **Pi** extension (`SS:package.json:2-13`), not a Claude Code or Cursor plugin.

---

## 3. Ranking

**1. C · 2. B · 3. A.**

C is the only candidate to pass both hard gates, and the only one whose factual spine — rule count,
node count, static/LLM split, and the meta-analyzer invariant — survives independent re-derivation
intact. B is a close second: it has the best-evidenced section 6, the most compact prose, and the
two most valuable mechanism discoveries in the arena (diminishing-returns scoring; the dead rule
families), but it inverts the same invariant A does and builds an *Avoid* on the inversion. A is
third: two must-pass failures, the highest density of primary-source error (AE1, 25 vs 27 nodes,
TP4 column, its own arithmetic), and the longest document — though three of its verified findings
are load-bearing and must survive the merge.

---

## 4. Recommended base — **C**, by soundest shape

Take `candidates/C/finding.md` as the merge base. Rationale, in order:

1. **Its spine is correct.** The three facts the synthesis will be quoted on — 112/113 rule IDs,
   24 static + 3 LLM + 1 hybrid, and "the LLM cannot lower the deterministic floor" — are all right.
   Rebasing on A or B means repairing the spine before any improvement can be trusted.
2. **Its section 6 contains no false verdict.** C's defects are four citation slips (section 1,
   notes a-d), all correctable in place. A and B each carry a fabricated gap that they then promoted
   into a headline improvement.
3. **It aligns with the user's stated emphases better than either runner-up.** Safety for
   third-party skills first -> C #1 and #4 (#4 is the only treatment anywhere of hardening the
   auditor against the skill it audits). Static analysis excellence -> C #2 (negative criteria) and
   #8 (coverage ledger). ICM / progressive disclosure -> C is closest, though still short
   (section 6 item 13).
4. **Its structure is the cleanest to extend**: a 27-row node table keyed to `ANALYZER_ID` lines,
   per-facet tables, and a handoff that already names the load-bearing facts for the synthesizer.

---

## 5. Fold-ins — the single strongest idea from each runner-up

**From B -> "Own the Cursor surface" (`B/finding.md:147`, improvement #1).**
Audit `.cursor/rules/*.mdc` **frontmatter** — `description`, `globs`, `alwaysApply` — plus
`.cursor/mcp.json` and `AGENTS.md`, not just `SKILL.md`. **Where:** replace C's improvement #5
(`C/finding.md:136`), which today covers only `.cursor/hooks.json` / `rules` / `mcp.json` as *paths*
for BH-style hook analysis. B's version is strictly larger and is the only place in the arena where
a candidate names real Cursor frontmatter keys — precisely the user's "per-harness frontmatter
validity" emphasis. Carry B's evidence (`SS:.../bundled_execution_surface.py:38-43` is
Claude-Code-only: `hooks/hooks.json`, `.claude/settings.json`, `.claude/settings.local.json` —
verified) and C's `CP:spec.md:70` R-CS-21 anchor.
*Runner-up from B, also worth carrying:* the per-rule diminishing-returns + confidence-scaled +
docs-exempt scoring at `SS:.../report.py:447-462` -> into C's section 4 scoring row and as a new
improvement beside C #3.

**From A -> the integration-cost correction (`A/finding.md:94,153`).**
SkillSpector is **not on PyPI** (404, re-verified); the only install path is a git URL; it needs
Python `>=3.12,<3.15`, ~20 runtime deps including native `yara-python`, and the Docker image must be
built locally. **Where:** into C's improvement #1 (`C/finding.md:132`), which currently prices the
integrate call at effort **S** and says "installs via `uv tool`/Docker". The one-SKILL.md-plus-wrapper
effort is genuinely S; the **user-side install tax is the real cost**, and it re-prices C's Decision
#2 (`C/finding.md:150`) and B's Decision #2. This also corrects B's packaging row
(`B/finding.md:109`), which states `uv tool install skillspector` — a command that cannot work.
*Runner-up from A, also worth carrying:* the `SA:CHANGELOG.md:12` + `SA:RELEASE_NOTES.md:8`
"Claude Code has no skill-level events" vs `SA:docs/profiler-spec.md:200,209` (which reads
`skill.name` for `skill_activation: otel`) contradiction — dual-cited, verified exact, and **absent
from C's section 6 entirely**. It also ties to `CP:spec.md:104` R-AT-04.

---

## 6. Defects the merged finding must fix

1. **Rule count ->** 113 distinct IDs (102 static incl. informational SSR-1; 11 LLM), **112
   finding-emitting**. README claims 71, its own tables enumerate 72, so **41 IDs are undocumented**.
   Delete "111" and "~111" wherever A's figure propagated.
2. **Delete the "LLM deletes static findings" claim everywhere**, including the improvements built
   on it — A's *Avoid* #12 (`A/finding.md:160`) and B's *Avoid* #11 (`B/finding.md:157`). The
   invariant is the opposite and belongs in the **Borrow** column (C's #9 already has it right).
3. **27 registered analyzer modules (24 static + 3 LLM)**, not 25. `osv_client.py`,
   `whitespace_padding.py`, `common.py`, `pattern_defaults.py`, `static_runner.py` are helpers.
   TP4 is LLM-gated inside a static module.
4. **AE1 exists** — `SS:.../finalize_inspection_ledger.py:77`. Remove A's "No AE1 in v2.11.2".
5. **Issue denominator = 61 true open issues**, not 135 (that field counts PRs). ~21 mention
   "false positive" -> ~34% of the backlog. Use A's denominator with the corrected FP count.
6. **State that LLM rule IDs are unenforced.** `LLMFinding.rule_id` is a free-form `str`
   (`SS:.../llm_analyzer_base.py:301`) and severity is model-chosen (`:303`). No candidate says this,
   and it materially qualifies the "11 LLM rules" figure.
7. **Drop the `SA:PRINCIPLES.md:26` stale verdict** (A and C both carry it). It is a normative
   sequencing belief — "Live evaluation is Tier 3 ... Static checks come first" — not a status claim,
   and `SA:profiler/compare.go` does not contradict it. Same for C's `SA:docs/research.md:72`
   verdict, which is scoped to the skill-audit skill, not the plugin.
8. **Drop the skill-rewrite "no before/after loop" claim** (A and B). The loop is specified at
   `SA:skills/skill-rewrite/SKILL.md:95`, `:132`, `:140`. The **real** gaps are (a) the score loop is
   never wired to a *measured runtime* delta via `SA:profiler/compare.go` (C #6 has this right) and
   (b) there is no record of a finding reviewed and intentionally kept (C's baseline point).
9. **C's citation slips:** `LedgerReason` has **52** members starting at
   `SS:src/skillspector/inspection_ledger.py:41` (not "37 reasons" at `:23-70`); the skillscore
   "safety" dimension is at `SA:skills/skill-audit/SKILL.md:57`, not `SA:AGENTS.md:11`; plus item 7.
10. **Install path:** `uv tool install git+https://github.com/NVIDIA/skillspector.git`. Correct B's
    `uv tool install skillspector`. State the no-PyPI fact wherever the integrate call is priced.
11. **Keep A's README-drift proof** (`SS:README.md:466-478` vs `SS:.../behavioral_ast.py:148-158`;
    4 of 9 AST severities disagree and AST10 is omitted). It is the concrete evidence for the rule
    "read the source, not the README" — which A itself violated in its own section 4 scoring row.
12. **Keep B's dead-rule finding, corrected to 4 IDs** (TR1, TR2, TR3, RP3), with both the source
    proof and issues #458 / #472. Keep the process lesson ("never ship a rule without a
    fires-on-real-input test") — the cheapest transferable discipline in the arena.
13. **Gaps in all three, against the user's stated emphases — the merge must add these:**
    - **Project-level always-on token cost.** No candidate addresses it. `SA:PRINCIPLES.md:44`
      scores "Token discipline" per skill only; the cost of every always-on skill description summed
      across an installed estate (L0/L1 in `SA:PRINCIPLES.md:55-61`) is unmeasured, and it is the one
      static metric SkillSpector has no answer to at all.
    - **ICM / progressive disclosure as a first-class dimension.** All three note the L0-L4 table at
      `SA:PRINCIPLES.md:53-61` and none proposes promoting it from a table to a scored, enforced
      dimension. `SA:PRINCIPLES.md:81` already cites the ICM paper (arXiv:2603.16021).
    - **Check the Cursor differentiation against `agnix` before asserting it.**
      `CP:docs/research/skill-audit-tool/L1-static-report.md:126` records agnix as **455 rules across
      Claude Code, Codex, Cursor, Copilot ... CLAUDE.md, SKILL.md, hooks, MCP configs** with `CUR-*`
      rule IDs. B's "Cursor is unclaimed" and C's "nothing else covers Cursor hooks" are both
      asserted only against SkillSpector, and both may be false against agnix. Neither candidate
      checked. This is the single largest unexamined risk to the recommended differentiation.
