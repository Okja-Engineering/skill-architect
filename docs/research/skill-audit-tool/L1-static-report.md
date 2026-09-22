# L1 — Static analysis: the SonarQube model applied to Agent Skills

**Researcher:** L1 · **Date:** 2026-09-11 · **Evidence key:** **[O]** = observed (I read/ran it) · **[I]** = inferred

## Summary (five lines)

1. SonarQube's transferable machinery is four things: **rules → issues with an `effortMinutes` remediation cost**, a **quality profile** (which rules are on), a **quality gate** (pass/fail conditions, biased to *new code*), and **technical debt ratio = Σ effort ÷ (dev-cost-per-line × LOC)** rendered as an A–E rating. All of it is money-convertible because effort is minutes. **[O]**
2. Every one of those maps cleanly onto skill packages. The only concept that needs redefinition is **complexity**: cyclomatic/cognitive complexity assume control flow, and a `SKILL.md` has none — the honest analog is a **context-cost + branch-count + reference-depth** triple, not a single number. **[I]**
3. The landscape is crowded at the **spec-conformance** layer (agnix 455 rules, skill-validator, skillcheck, doodle, pulser, skill-lint) and well-served at **security** (NVIDIA SkillSpector, 71 patterns, SARIF) and **duplication** (jscpd, Markdown-aware, SARIF). It is **empty** at: cross-skill reference graphs / cycle detection, remediation-cost economics, and portfolio trend. **[O]**
4. SonarQube itself has **no Markdown analyzer** (so `SKILL.md` is invisible to it) but **does** ship Shell and Secrets analyzers (so `scripts/*.sh` is already covered) — which decides the build shape. **[O]**
5. **Recommendation: build a standalone Go rule engine that emits Sonar *generic issue* JSON as its primary interop format (SARIF as secondary), and keep the Claude Code/Cursor plugin as the UX.** Do **not** write a SonarQube Java plugin. SARIF alone is disqualifying: Sonar's SARIF importer hard-codes every imported issue to `SECURITY`/`CONVENTIONAL` and drops effort, which destroys the technical-debt model. **[O]**

---

## 1. SonarQube's model, precisely

| Concept | Definition (SonarSource docs) | Cite |
|---|---|---|
| **Rule** | Executed on source to generate issues. In **MQR mode** a rule is categorised by *software quality* (security / reliability / maintainability); it may impact more than one. In **Standard Experience** there are four types: code smell, bug, vulnerability, security hotspot. **[O]** | [rules](https://docs.sonarsource.com/sonarqube-server/quality-standards-administration/managing-rules/rules.md) |
| **Software qualities** | Security, Reliability, Maintainability. Each rule→quality pairing carries a severity: **Blocker / High / Medium / Low / Info**. "When a rule is broken, an issue is raised. The issue affects one or more software qualities with varying severity as inherited from the rule." **[O]** | [software-qualities](https://docs.sonarsource.com/sonarqube-server/quality-standards-administration/managing-rules/software-qualities.md) |
| **Clean-code attributes** | Consistency (FORMATTED, CONVENTIONAL, IDENTIFIABLE), Intentionality (CLEAR, LOGICAL, COMPLETE, EFFICIENT), Adaptability (FOCUSED, DISTINCT, MODULAR, TESTED), Responsibility (LAWFUL, TRUSTWORTHY, RESPECTFUL). **[O]** | [generic-issue-import-format](https://docs.sonarsource.com/sonarqube-server/analyzing-source-code/importing-external-issues/generic-issue-import-format.md) |
| **Quality profile** | Per-language rule set; built-in "Sonar way" is read-only; custom profiles may override severity and rule parameters and may **inherit** from a parent profile. Enterprise adds "prioritized rules". **[O]** | [quality-profiles](https://docs.sonarsource.com/sonarqube-server/quality-standards-administration/managing-quality-profiles/understanding-quality-profiles.md) |
| **Quality gate** | Set of conditions = *(metric, comparison operator, error value)*, each scoped to new code or overall code; **if one condition is met the gate fails**. Built-in **Sonar way** has exactly four: no new issues; all new security hotspots reviewed; new-code coverage ≥ 80%; new-code duplication ≤ 3%. On PRs, **only new-code conditions apply**. A **fudge factor** suppresses duplication/coverage conditions until ≥ 20 new lines. **[O]** | [quality-gates](https://docs.sonarsource.com/sonarqube-server/quality-standards-administration/managing-quality-gates/introduction-to-quality-gates.md) |
| **New code** | "Code you've recently added or modified." Definitions: previous version, number of days (default 30, max 90), specific analysis, reference branch. Overall and new-code ratings are computed separately. **[O]** | [about-new-code](https://docs.sonarsource.com/sonarqube-server/user-guide/about-new-code.md) |
| **SQALE / technical debt** | `sqale_index` = **Σ of per-issue remediation cost in minutes**, "taken over from the effort assigned to the rule that raised the issue". 8-hour day when shown in days. **[O]** | [metrics-definition](https://docs.sonarsource.com/sonarqube-server/user-guide/code-metrics/metrics-definition.md) |
| **Technical debt ratio** | `sqale_debt_ratio = technical debt / (cost to develop one line of code × LOC)`; **dev cost defaults to 30 min/line**, settable via `sonar.technicalDebt.developmentCost`. Worked example in the docs: 122,563 min debt ÷ (30 × 63,987) = **6.4%**. **[O]** | [metrics-definition](https://docs.sonarsource.com/sonarqube-server/user-guide/code-metrics/metrics-definition.md), [metrics-parameters](https://docs.sonarsource.com/sonarqube-server/instance-administration/analysis-functions/metrics-parameters.md) |
| **Maintainability rating** | A ≤5% · B 5–<10% · C 10–<20% · D 20–<50% · E ≥50%. Grid overridable via `sonar.technicalDebt.ratingGrid`. **[O]** | same |
| **Security hotspot** | "Security-sensitive code that the developer needs to review" — *not* a defect. Distinguished from a vulnerability by **the need for review before deciding whether to fix**. Statuses start at **To review**. Sonar's stated target: >80% resolved as "reviewed" after review. **[O]** | [security-hotspots](https://docs.sonarsource.com/sonarqube-server/user-guide/security-hotspots.md) |
| **Duplication** | Non-Java default: **≥100 successive duplicated tokens** spread over **≥10 lines** (20 ABAP, 30 COBOL); Java: ≥10 successive duplicated statements. Indentation and string-literal differences ignored. `duplicated_lines_density = duplicated_lines / lines × 100`. Not supported for Terraform-like IaC or CSS. **[O]** | [metrics-definition](https://docs.sonarsource.com/sonarqube-server/user-guide/code-metrics/metrics-definition.md) |
| **Cyclomatic complexity** | `complexity` = 1 + number of conditional branches, computed per function then summed. Function-level values are not exposed in the UI. **[O]** | same |
| **Cognitive complexity** | `cognitive_complexity` = "a qualification of how hard it is to understand the code's control flow"; canonical definition is the SonarSource *Cognitive Complexity* white paper (Campbell). Rule **S3776** default threshold **15** — reported by Sonar's docs assistant; `rules.sonarsource.com` was unreachable from this host so the rule page itself is unverified. **[I]** | [white paper](https://www.sonarsource.com/resources/white-papers/cognitive-complexity/) |

**Two things it is easy to get wrong.** (a) Complexity is a **measure**, not an issue — it only becomes debt when a *rule* (S3776) fires on a threshold. (b) The gate is deliberately **new-code-first**: "This quality gate focuses on keeping high quality standards for new code, rather than spending a lot of effort remediating old code." **[O]** That is the mechanism that makes SonarQube adoptable on a legacy estate, and it transfers directly to a skill portfolio.

---

## 2. Mapping each concept to skill packages

### 2a. Concept map

| Sonar concept | Skill-package analog | Computable from | Already covered? |
|---|---|---|---|
| Language | `SKILL.md` (Markdown+YAML) + `scripts/*` (shell/python/js) — **two languages in one artifact** | filesystem | partly |
| Software qualities | **Security** (bundled scripts, injection, exfil) · **Reliability** (will it trigger, will it execute) · **Maintainability** (context cost, structure, duplication) | rule design | no product does all three |
| Issue + severity | rule-ID finding with Blocker→Info | rule engine | agnix/skillcheck have severity, no effort |
| **Remediation cost** | minutes to fix *this* finding (e.g. "add a `## Constraints` heading" = 5 min; "split a 900-line SKILL.md" = 120 min) | rule metadata table | **nothing does this — the white space** |
| Dev cost / LOC | **cost to author one line of a skill** — the denominator. Sonar's 30 min/line is wrong here; skills are authored far faster | team-set constant | no |
| Technical debt ratio | Σ effort ÷ (author-cost-per-line × SKILL.md+refs lines) → A–E per skill | derived | no |
| Quality profile | which skill rules are active for this repo (house policy vs spec) | config file | skill-architect encodes house policy in shell, not config **[O]** |
| Quality gate | CI condition set, e.g. *0 new Blocker, 100% hotspots reviewed, cross-skill duplication ≤ 3%, trigger-overlap < 60%* | CI | agnix `--strict`, skill-validator exit codes — no metric conditions |
| **New code** | changed skills since ref branch / last release — score the **diff**, not the estate | git | no |
| Security hotspot | a `curl \| sh`, an unpinned `npm i -g`, a `$(...)` on untrusted input, a secret-shaped literal inside a bundled script — **review-required, not auto-fail** | script AST/grep | SkillSpector (as findings, not as a review lifecycle) |
| Duplication | same prose block across two skills; same script copy-pasted | jscpd (Markdown-aware) | jscpd, unbadged |
| Cyclomatic complexity | — **no faithful analog**; a SKILL.md has no control flow | — | — |
| Cognitive complexity | see 2b | — | — |

### 2b. What "complexity" should mean for an instruction file

Sonar's two complexity metrics both assume executable control flow. A `SKILL.md` is read, not executed. Proposal — **three separate measures, not one score**, because they drive different fixes: **[I]**

| Measure | Definition | Source of truth | Threshold candidate |
|---|---|---|---|
| `always_loaded_tokens` | BPE tokens of `name` + `description` (+ `when_to_use`) | `skill-validator check -o json` → `.token_counts` (real `o200k_base`) **[O]** | Claude Code truncates combined `description`+`when_to_use` at **1,536 chars** **[O]** ([docs](https://code.claude.com/docs/en/skills)); spec caps `description` at **1024 chars**, `name` 64, `compatibility` 500 **[O]** ([spec](https://agentskills.io/specification)) |
| `activation_tokens` | BPE tokens of the `SKILL.md` body (paid on every activation) | same | spec: "**Instructions (< 5000 tokens recommended)**", "Keep your main SKILL.md under 500 lines" **[O]** |
| `deferred_tokens` | BPE tokens behind `references/` + `assets/` | same | measured: skill-audit is 1,814 body / 4,030 total → **55% deferred [O]** (`docs/research/skill-profiling-state-of-the-art.md:155-180`) |
| `branch_count` | count of conditional instructions — "if X then…", "when…", "otherwise", "unless" | markdown parse | the closest true cognitive-complexity analog: each branch is a decision the model must resolve **[I]** |
| `reference_depth` | longest chain `SKILL.md` → `references/a.md` → `references/b.md` | reference graph | spec: "**Keep file references one level deep from SKILL.md. Avoid deeply nested reference chains.**" → depth > 1 is a rule with a citation **[O]** |
| `directive_density` | imperative sentences ÷ total sentences | `skill-validator` already emits `imperative_ratio`, `instruction_specificity`, `information_density`, `code_block_ratio` **[O]** (ran it: see 2d) | unset |

**Do not roll these into one "cognitive complexity" number.** Sonar can, because minutes-to-fix is a single axis for source. Here, high `always_loaded_tokens` costs money on *every* session while high `deferred_tokens` costs nothing until referenced — they are not fungible. **[I]**

### 2c. Reference graph: circular refs, dead refs, duplication

| Edge type | How to extract | Failure it catches |
|---|---|---|
| skill → file (`[text](references/x.md)`) | markdown link parse | dead reference |
| skill → script (`scripts/foo.sh` in a fence) | fenced-block path scan | "the agent runs a command that does not exist" — the exact silent failure `PRINCIPLES.md:14` names **[O]** |
| script → skill (a bundled script that `cat`s another skill, or invokes `claude`/a slash command) | shell parse | **cycle**, and a permission-scope leak |
| skill → skill (prose "then run the `foo` skill", `[[wikilinks]]`, slash-command chains) | name-reference scan against the skill index | **cycle**: unbounded context growth / non-terminating orchestration |
| skill → external binary (`skill-validator`, `skillscore`) | command scan vs `compatibility:` frontmatter | **undeclared dependency** — skill-audit hard-requires both tools and aborts if missing (`skills/skill-audit/SKILL.md:44-45` **[O]**) |

Cycle detection is a plain SCC/Tarjan pass over that graph. **No evidence found of any existing tool doing this for skills** — GitHub searches for `skill dependency graph circular`, `SKILL.md circular reference`, `agent skill dead link checker` returned 2, 0 and 1 repos respectively, none relevant. **[O]**

**Duplication across skills** needs no new engine: `jscpd` tokenizes **Markdown by splitting it into embedded languages first**, reports `exact`/`renamed`/`similar` clone kinds, and emits **SARIF** (`jscpd/duplicate-code`, `jscpd/renamed-code`, `jscpd/similar-code`) among 15 reporters. MIT, 6,192★, pushed 2026-09-11. **[O]**

**Trigger-quality duplication** is a distinct axis: two skills whose descriptions overlap make the router coin-flip. `skillcheck` implements `trigger-collision` at **≥60% shared keywords**; `doodle surface` measures overlap between a description and realistic user phrasings offline, and reports a **mean 24% vocabulary overlap across 77 published skills, with description length explaining <8% of variance**. **[O]**

### 2d. What skill-architect's `skill-audit` already checks statically — and what it does not

| Rule ID | Check | File:line | Sonar analog |
|---|---|---|---|
| PL001 | `license:` present in frontmatter (house policy, not spec) | `check-frontmatter.sh:50` **[O]** | maintainability, Low |
| PL002 | six required headings present (`When to use`, `.*Process`, `Examples?`, `Deterministic`, `Orchestration`, `Constraints`) via case-insensitive grep | `check-structure.sh:40-42` **[O]** | maintainability, Medium |
| PL003 | `SKILL.md` ≤ 500 lines | `check-structure.sh:61-62` **[O]** | maintainability (a *measure* threshold, correctly modelled as a rule) |
| PL004 / PL005 | ≥1 fenced code block / ≥1 list item | `check-structure.sh:49,55` **[O]** | maintainability, Low |
| PT001 | script/reference path in a fenced block resolves on disk | `check-paths.sh:76` **[O]** | **reliability, High — the dead-reference rule** |
| PT002 | markdown link target resolves on disk | `check-paths.sh:56` **[O]** | reliability |
| (delegated) | spec validation, link resolution, token counts, contamination | `skill-validator validate structure` / `check` **[O]** | — |
| (delegated) | 7-dimension quality score | `skillscore --json` **[O]** | — |

Measures already available for free, from running `skill-validator check skills/skill-audit -o json` **[O]**: `token_counts.files[]`/`.total`, and `content_analysis` = `{word_count: 1074, code_block_count: 7, code_block_ratio: 0.216, sentence_count: 86, imperative_count: 12, imperative_ratio: 0.1395, information_density: 0.1778, strong_markers: 6, weak_markers: 1, instruction_specificity: 0.8571, section_count: 18, list_item_count: 47}`, plus `contamination_analysis.contamination_score`.

**Gaps in the current static surface** (all **[O]** by absence):

| Gap | Why it matters |
|---|---|
| No remediation effort on any finding | without minutes there is no debt, no ratio, no dollars — this is the single biggest gap vs the SonarQube model |
| No cross-skill analysis at all | `target_skill` "must be one skill directory, not a repository root or collection" (`skills/skill-audit/SKILL.md:33` **[O]**) — so duplication, trigger collision, and cycles are structurally unreachable |
| No cycle detection | see 2c |
| No security rules on bundled scripts | scripts are only checked for *existence*, never for content |
| No new-code scoping | every run scores the whole skill; nothing scores the diff |
| Policy encoded in bash, not a profile | PL002's six headings are a hard-coded `for` loop (`check-structure.sh:40`) — cannot be turned off per-repo without editing the script |
| Hard external dependency | aborts if `skill-validator` or `skillscore` is absent (`SKILL.md:44-45`) — ironic given `PRINCIPLES.md:24` "Self-contained ... No hidden external dependencies" **[O]** |
| PT001/PT002 emit `unverified` for any path containing `$` or a glob | `check-paths.sh:50,71` **[O]** — every `"$skill_root/scripts/..."` reference silently escapes validation, including in skill-audit's own SKILL.md |

### 2e. Security hotspots in bundled scripts

Sonar's *hotspot* semantics — review-required, not auto-fail, with a status lifecycle — is the **right** model for skills, because the same construct (`curl`, `rm -rf`, a network call) is legitimate in one skill and an exfiltration path in another. **[I]** Candidate hotspot families, all statically detectable: network egress (`curl`/`wget`/`nc`), `eval`/`$(...)` on non-literal input, unpinned installs (`npm i -g`, `pip install` without a version), credential-shaped literals, writes outside the skill root, `sudo`, `--dangerously-skip-permissions`-class flags, and `allowed-tools` breadth vs what the scripts actually invoke. SonarQube's own **Shell** analyzer already covers Bash/POSIX and its **Secrets** analyzer is language-agnostic **[O]** — so for `scripts/*.sh` the platform is not the gap; `SKILL.md` is.

---

## 3. Verified landscape of static tools for skills and prompts

| Tool | What it measures statically | How | License / maturity | URL |
|---|---|---|---|---|
| **agnix** | **455 rules** across Claude Code, Codex, Cursor, Copilot, OpenCode, Kiro, Gemini, MCP: CLAUDE.md, SKILL.md, hooks, MCP configs. Rule IDs (`CC-*`, `CUR-*`, `MCP-*`), `agnix explain <ID>`, auto-fix (safe/unsafe tiers), `--strict`, `--format github`, GitHub Action, VS Code/JetBrains/Neovim/Zed extensions | Rust core, npm/pip/cargo/brew | MIT + Apache-2.0, 410★, pushed 2026-09-11 — **most mature spec-conformance engine** | [agent-sh/agnix](https://github.com/agent-sh/agnix) **[O]** |
| **skill-validator** | spec/frontmatter validation, internal link resolution, **real `o200k_base` BPE token counts** (`check` only, not `analyze content`), content analysis (see 2d), cross-language contamination score; `-o json\|markdown\|compact`, `--emit-annotations` for GH Actions | Go binary, `brew` | v1.6.1 installed locally; used by skill-architect | [agent-ecosystem/skill-validator](https://github.com/agent-ecosystem/skill-validator) **[O]** |
| **NVIDIA SkillSpector** | **71 vulnerability patterns / 17 categories**: prompt injection, data exfil, privilege escalation, supply chain, excessive agency, system-prompt leakage, memory poisoning, tool misuse, trigger abuse, dangerous code (AST), taint tracking, YARA, MCP least-privilege. 0–100 risk score. **Terminal/JSON/Markdown/SARIF output.** Baseline suppression so re-scans surface only *new* findings. Optional LLM stage. Cites research: **26.1% of skills contain vulnerabilities, 5.2% likely malicious** | Python 3.12, Docker, MCP server | Apache-2.0, **16,984★**, pushed 2026-09-11 — the security layer, do not rebuild | [NVIDIA/SkillSpector](https://github.com/NVIDIA/SkillSpector) **[O]** |
| **jscpd** | copy/paste detection, **Markdown split into embedded languages before tokenizing**, exact/renamed/similar clones, `--min-tokens`/`--min-lines` thresholds, 15 reporters incl. **SARIF** and codeclimate, exit codes to gate on | Rust engine, npm | MIT, 6,192★, pushed 2026-09-11 | [kucherenko/jscpd](https://github.com/kucherenko/jscpd) **[O]** |
| **skillcheck** | frontmatter errors, name/description length (>1024 chars), **duplicate names across skills**, secret detection (AWS/OpenAI/GitHub PAT/JWT/private keys), description too short/generic, **`trigger-collision` at ≥60% shared keywords** | Python, pip | MIT, 18★, pushed 2026-07-20 | [RiriXt1/skillcheck](https://github.com/RiriXt1/skillcheck) **[O]** |
| **doodle** | **`doodle surface`** — offline vocabulary-overlap between a description and realistic user phrasings (no API key); 16 static rules each citing the authoring guide; plus `doodle eval` (dynamic, L2's ground). Published: mean 24% overlap over 77 skills | TS, VS Code ext | MIT, 25★, pushed 2026-08-21 | [krishyaid-coder/doodle](https://github.com/krishyaid-coder/doodle) **[O]** |
| **pulser** | portfolio-level scan ("54 skills scanned · Score 89/100"), classifies + prescribes fixes with AUTO/MANUAL fix type, GitHub Action | npm `pulser-cli` | MIT, 18★, pushed 2026-06-30 | [TheStack-ai/pulser](https://github.com/TheStack-ai/pulser) **[O]** |
| **skill-lint** | Claude.ai-upload-blocking errors: name format, description length, angle brackets, disallowed frontmatter fields, `marketplace.json` | npm, GH Action | MIT, 15★, pushed 2026-08-15 | [himself65/skill-lint](https://github.com/himself65/skill-lint) **[O]** |
| **skills-ref** | reference validator from the spec authors: `skills-ref validate ./my-skill` — frontmatter validity + naming conventions | — | official reference impl | [agentskills/agentskills](https://github.com/agentskills/agentskills/tree/main/skills-ref) **[O]** |
| **skillscore** | 7-dimension quality score (identity, conciseness, clarity, routing, robustness, safety, portability), `--json` | Dart CLI, also npm | v2.0.2 installed locally. Note: skill-architect calls it "npm, 7-dimension"; upstream describes six dimensions + Dart — pre-existing doc conflict, unresolved **[O]** | [sayed3li97/skillscore](https://github.com/sayed3li97/skillscore) |
| **Claude Code `/skill-doctor`** | per-skill **context cost + invocation count**; flags never-invoked skills and unused plugins; recommends starting with highest context cost. Interactive → `/plugin` Stats tab; `-p` prints text. Requires **≥ v2.1.252** | first-party | proprietary. **Local machine runs 2.1.221 — unavailable, unverified [O]** | [code.claude.com/docs/en/skills](https://code.claude.com/docs/en/skills) **[O]** |
| **`claude plugin eval`** | runs each prompt in an isolated session **with and without the plugin**, scores with graders, **exits non-zero below a threshold** so CI can gate | first-party | proprietary; **dynamic — L2's ground**, noted here only because it is the CI gate skills already have | [plugin-evals](https://code.claude.com/docs/en/plugin-evals) **[O]** |
| **promptfoo** | 25,038★ MIT — primarily dynamic eval + red-team. Static surface is config validation only. **L2's ground.** | — | MIT, pushed 2026-09-12 | [promptfoo/promptfoo](https://github.com/promptfoo/promptfoo) **[O]** |
| **SonarQube markdown plugin** | **No evidence found.** GitHub searches `sonar plugin markdown in:name` and `sonarqube markdown analyzer` both returned **0 repositories**. Sonar docs confirm no general `.md` analyzer; only language-in-Markdown (R chunks in `.Rmd`) | — | — | **[O]** |
| **Sonar "Sonar way for agentic AI" / Agentic Analysis** | Sonar gates **AI-generated code**, in Beta on Cloud (Team/Enterprise). It does **not** analyze the skill/prompt artifacts. Prior art for the gate, not a competitor | proprietary | Beta | [quality-gate-for-agentic-ai](https://docs.sonarsource.com/sonarqube-server/quality-standards-administration/ai-code-assurance/quality-gate-for-agentic-ai.md) **[O]** |

**Landscape verdict.** Spec conformance is solved (agnix). Security is solved (SkillSpector). Duplication is solved (jscpd). **Unsolved and unclaimed: remediation cost in minutes, a debt ratio per skill, cross-skill reference graphs with cycle detection, new-code scoping, and a portfolio ranking.** That is precisely the SonarQube half of the SonarQube model, and precisely the user's stated goal. **[I]**

---

## 4. Build options and recommendation

| # | Option | What it requires | Gets you | Costs you |
|---|---|---|---|---|
| **a** | **Real SonarQube plugin** (new "language" = Agent Skill) | `sonar-plugin-api` is a **Java** API; Java 8 + Maven 3.1+, `sonar-packaging-maven-plugin`, jar copied to `extensions/plugins/`, server restart, `Sonar-Version` manifest pin **[O]**. "Supporting a new language" = write a grammar, a parser, parse-tree visitors, a scanner Sensor, and compute issues + raw measures + duplications + highlighting + symbol table **[O]** | Native rules page, quality profiles, native debt/ratio/rating, hotspot review lifecycle, trend history — the full model, free | A Java codebase in a Go/TS shop; a SonarQube server as a hard dependency; a grammar for Markdown; distribution via jar. Nobody without a SonarQube install can use the tool. **Highest cost, narrowest reach.** |
| **b** | **Standalone engine → Sonar generic-issue JSON** (+ SARIF) | Emit `{rules:[{id,name,description,engineId,cleanCodeAttribute,impacts:[{softwareQuality,severity}]}], issues:[{ruleID, effortMinutes, primaryLocation:{message,filePath,textRange}}]}`; point `sonar.externalIssuesReportPaths` at it **[O]**. **No plugin required** **[O]** | Runs standalone *and* lands in SonarQube. External issues **are taken into account when calculating quality gate status** **[O]**, and `effortMinutes` + `softwareQuality: MAINTAINABILITY` feeds technical debt **[I — asserted by Sonar's docs assistant, consistent with the debt definition, not stated verbatim on a primary page]** | Imported rules are **not visible on the Rules page and not in any quality profile** — profile management stays in our tool **[O]**. Two report formats to maintain. |
| **b′** | Same, but **SARIF only** | `sonar.sarifReportPaths`, SARIF 2.1.0 | GitHub Code Scanning for free | **Disqualifying for the debt model:** Sonar's SARIF importer **assigns `CONVENTIONAL` and forces `SECURITY` software quality to every imported issue**, maps severity only from `defaultConfiguration.level` (error→HIGH, warning→MEDIUM, note/none→LOW), and has **no effort field** **[O]**. Every maintainability finding would be mislabelled as a security issue with zero debt. |
| **c** | **Stay a Claude Code / Cursor plugin only** | what skill-architect is today | Zero new infra; lives where the author works; `/skill-doctor`-adjacent UX | No CI gate that a non-Claude user can run, no machine-readable interop, no trend store. The current implementation also can't cross skill boundaries (2d) — which blocks duplication, trigger collision, and cycles no matter how good the prose gets. |

### Recommendation

**Build option (b): a standalone rule engine — Go, single static binary, no runtime — whose native output is a report object, with Sonar generic-issue JSON as the primary interop emitter and SARIF as a secondary one. Keep the Claude Code/Cursor plugin (option c) as a thin front end over that binary, and do not build a SonarQube Java plugin.**

Because:

1. **Effort is the whole product.** The user wants dollars. Dollars come from minutes. Only the generic-issue format carries `effortMinutes`; SARIF does not, and the SonarQube plugin route buys effort at the price of a Java codebase and a server dependency. **[O]**
2. **The platform gap is exactly `SKILL.md`.** SonarQube already analyzes `scripts/*.sh` (Shell) and secrets (Secrets analyzer), and has **no Markdown analyzer at all** **[O]**. So the tool that adds value is one that analyzes the instruction file and hands findings to whatever gate the team already runs — not one that re-implements shell linting inside SonarQube.
3. **Standalone is the only shape that works for the majority who have no SonarQube.** Generic-issue JSON is an *emitter*, not a dependency: teams without Sonar get the binary, the exit code, and the report; teams with Sonar get the same findings on their dashboard by setting one property. **[I]**
4. **Go matches the existing stack** (`profiler/*.go`, `skill-validator` is Go) and the repo's stated self-contained-tooling rule (`docs/research/go-dependency-and-tooling-decisions.md:70-73` **[O]**).
5. **Don't rebuild solved layers.** Shell out to / vendor `SkillSpector` for security hotspots, `jscpd` for duplication, `skill-validator` for BPE counts and spec validation. Own the three unclaimed layers: **reference graph + cycles**, **remediation-cost economics**, **new-code scoping and portfolio ranking**.

**Named tradeoff.** Choosing (b) over (a) permanently gives up native quality-profile management and the native hotspot review lifecycle inside SonarQube — imported external rules are invisible on the Rules page and cannot be activated/deactivated there **[O]**. We must therefore own the profile (a checked-in config file: which rules, which severities, which effort minutes) and own the hotspot "reviewed" state (a baseline file, the pattern SkillSpector already uses **[O]**). That is real work, and it should be in the spec, not discovered later.

### Minimum rule set to ship first (with effort, so debt is computable on day one)

| ID | Rule | Quality / Severity | Effort |
|---|---|---|---|
| `SK-S001` | Bundled script pipes network output to a shell | Security / **Blocker** (hotspot) | 30 min |
| `SK-S002` | Credential-shaped literal in a bundled file | Security / Blocker | 15 min |
| `SK-R001` | Referenced script or file does not exist (supersedes PT001/PT002, and resolves `$var` paths instead of skipping them) | Reliability / **High** | 10 min |
| `SK-R002` | Reference cycle detected in the skill/script graph | Reliability / High | 60 min |
| `SK-R003` | Undeclared external binary invoked but absent from `compatibility:` | Reliability / Medium | 10 min |
| `SK-R004` | `description` absent, <20 chars, or filler-prefixed | Reliability / Blocker | 15 min |
| `SK-R005` | Trigger collision: ≥60% keyword overlap with another skill's description | Reliability / Medium | 30 min |
| `SK-M001` | `always_loaded_tokens` over the 1,536-char listing cap | Maintainability / High | 20 min |
| `SK-M002` | `SKILL.md` body > 500 lines or > 5,000 tokens | Maintainability / Medium | 90 min |
| `SK-M003` | Reference depth > 1 | Maintainability / Medium | 45 min |
| `SK-M004` | Duplicated block ≥ N tokens shared with another skill | Maintainability / Low | 25 min |
| `SK-M005` | Required house-policy heading missing (configurable; replaces hard-coded PL002) | Maintainability / Low | 5 min |

Effort values above are **[I] — illustrative placeholders**. They must be calibrated by timing real fixes before any dollar figure is published; an uncalibrated SQALE model produces confident, wrong numbers. Sonar's own denominator (30 min/line) is certainly wrong for skills and must be re-measured.

---

## Decisions needed above my level

1. **Is SonarQube a target or a metaphor?** If no team member actually runs a SonarQube server, the generic-issue emitter is speculative surface area and should be deferred behind SARIF + native JSON. I found no evidence either way in this repo.
2. **Who calibrates `effortMinutes` and the author-cost-per-line denominator, and with what data?** Without this the debt ratio and every dollar figure downstream are unfounded. This is a measurement commitment, not a coding task.
3. **Vendor vs shell out to SkillSpector (Apache-2.0, Python 3.12) and jscpd (MIT, Node/Rust).** Shelling out re-introduces exactly the runtime dependencies the repo's self-contained rule rejects; vendoring means owning 71 security patterns. Third option: reimplement a thin subset. This is a product-scope call.
4. **Does the tool analyze one skill or a portfolio?** Cross-skill rules (duplication, trigger collision, cycles) are where the unclaimed value is, but the current `skill-audit` contract is explicitly single-skill. Changing it is a breaking interface change to a shipped plugin.
5. **Rule-ID namespace ownership.** `PL*`/`PT*` are already published in `skill-audit`'s SKILL.md and exit-code contract. A move to `SK-*` is a user-visible break.

## Handoff to synthesis

- **To L2 (dynamic):** static analysis can supply the *predictor* side of a paired experiment — `always_loaded_tokens` and `activation_tokens` from `skill-validator check -o json` are real `o200k_base` BPE, not estimates, and are the one honest token number available without billing data. The before/after story should pair a **static delta** (tokens, findings, debt-minutes) with the **runtime delta**; the static delta is exactly reproducible and costs nothing, so it belongs in every report as the floor. Also: `doodle surface` gives an *offline* proxy for activation likelihood that can be measured before spending any model calls.
- **To L3 (prioritization):** SonarQube's ranking primitive is the **technical debt ratio** (debt ÷ dev cost), and its adoption primitive is **new-code focus** — rank by *debt introduced recently*, not total debt, so a large old skill doesn't permanently dominate the list. `/skill-doctor`'s heuristic ("start with the highest context cost", "flag skills never invoked") is first-party prior art for the portfolio view and should be compared against your scoring model. The static side can supply, per skill: debt-minutes, debt ratio, A–E rating, blast-radius proxies (script count, network calls, `allowed-tools` breadth, permission scope), and a never-referenced flag from the reference graph.
- **Unresolved / thin evidence, flagged honestly:** S3776's default threshold of 15 comes from Sonar's docs assistant, not a primary rule page (`rules.sonarsource.com` was unreachable from this host). That imported generic-format issues with `softwareQuality: MAINTAINABILITY` + `effortMinutes` roll into `sqale_index` is consistent with the debt definition and asserted by the docs assistant, but I did not find it stated verbatim on a primary page — **verify with a throwaway SonarQube instance before designing the economics on top of it.** `/skill-doctor` could not be run (local Claude Code 2.1.221 < 2.1.252).
