# `skillgate` — a state-of-the-art skill audit / doctor / inspector for Claude Code and Cursor

**Merged recommendation · 2026-09-12 · synthesis only, no new research.**

**Keys**, under `cursor-profiler/docs/research/` unless noted: `[SS]` `skillspector.md` · `[SY]` `skill-audit-tool/synthesis.md` · `[L1]`/`[L2]`/`[L3]` `skill-audit-tool/L{1,2,3}-*-report.md` · `[WD]` `warp-skill-doctor.md` · `[OT]` `otel-genai-alignment.md` · `[SOTA]` `skill-profiling-state-of-the-art.md` · `[EPC]` `existing-per-skill-cost-tools.md` · `[CTS]` `cursor-telemetry-surfaces.md` · `[Qn]`/`[spec]` `cursor-profiler/` · `[SA:…]`/`[PR:n]` `skill-architect/` · `[AGX]` agnix enumeration (`agent-sh/agnix`, `rules.json` v1.1.0, **455 rules**, @2026-09-11) · `[pi.md §n]` `pi.md`, the arena-merged pi finding (pi @`71dca87`, 2026-09-12), folded in only where it changed something — each such edit tagged **[pi]**; delta log in `pi-integration.md`. ● covers · ◐ partial · ○ no.

---

## 1. The pitch

1. **What.** One Go binary plus a `skill-gate` Agent Skill wrapping it — shaped like NVIDIA's `skill-inspector`: a security tool distributed as a skill wrapping its own deterministic CLI, degrading to manual review when the binary is absent `[SS §9]`. Three questions, one pass: **is this safe to install**, **what does it cost my estate every session**, **did my last refactor pay**.
2. **For whom.** The engineer pulling a skill off the internet (gate); the author refactoring their own (debt + proof); the platform lead ranking the estate `[L3 §3.1]`.
3. **Why state of the art.** Two crowded lanes, four empty. Security *depth* is solved by SkillSpector: **113 rule IDs (112 emitting + informational SSR-1), 18 categories, 27 analyzer nodes**, OSV, YARA, taint, ~2,500 tests `[SS §3]`. Spec *conformance* is solved by agnix: 20 `CUR-*` over `.cursor/rules/*.mdc`, `.cursor/hooks.json` schema, 26 `MCP-*` over `.cursor/mcp.json`, `AGM-001..006` over `AGENTS.md`, **30 `CC-SET-*` over `.claude/settings.json`** `[AGX]`. **We reuse both and build only the four things neither does:** project-level **always-on token budget** with per-item attribution `[SS §7 #4]`; **ICM / progressive-disclosure** enforcement `[SS §7 #5]`; **reference graph + cycles** `[L1 §2c]`; **Cursor security** — agnix checks a hook's *shape*, never what it executes; SkillSpector reads no Cursor path at all `[SS §1.4, AGX]`. Remediation cost in minutes is unclaimed by every tool surveyed `[L1 §3, WD §5]`.
4. **Why now.** Claude Code emits `skill.name` on `claude_code.token.usage`/`cost.usage` `[L2 §2]`; Cursor hooks and Enterprise OTel `cursor.skill.activated` are the only first-party skill-activation event anywhere `[CTS §2]`. The dynamic half rides on surfaces that exist today.
5. **The bet.** Safety blocks; economics ranks; nothing is claimed that cannot be traced to a measured token, an observed minute, or a deterministic rule. Everything else prints `unknown` with a reason `[SA:docs/profiler-spec.md:11-36]`.

---

## 2. The threat and quality model for a skill entering an enterprise

Complete against SkillSpector's catalogue — **113 IDs / 18 categories / 27 nodes** `[SS §3]` — plus what it structurally cannot reach. **S** static · **L** LLM · **D** dynamic. Coverage order **SS**·**SA** (`skill-audit` today)·**AG** (agnix)·**WP** (Warp)·**CD** (`/skill-doctor`).

| # | Threat / defect | SS IDs | Det | SS·SA·AG·WP·CD | What we add |
|---|---|---|---|---|---|
| 1 | Prompt injection: override, zero-width, exfil steering, padding | P1-4, P9 | S | ●·○·**○**·○·○ | Same rules over Cursor `.mdc` + `AGENTS.md`, unread by SS `[SS §1.4]`; harden our own auditor `[SS §7 #1]`. *agnix has **no** injection rules — `CC-SK-009 "Too Many Injections"` counts `` !`cmd` `` authoring injections, not attacks* `[AGX]` |
| 2 | Semantic/paraphrased injection, NL exfil; description↔behaviour mismatch, scope creep | SSD-1..4, SDI-1..4 | L | ●·◐·○·○·○ | Deterministic subset first, LLM for the residue; add-never-delete + "Do NOT flag if" negatives. SS's LLM rule IDs are **free-form strings, not enforced constants** `[SS §2-3, §7 #6]` |
| 3 | Data exfiltration: transmission, env harvest, FS enum, context leak, cloud upload | E1-5 | S | ●·○·○·○·○ | Blocks from the floor (T004-T006); observed egress vs declared `allowed-tools` feeds blast radius `B` `[L3 §2 R8]` |
| 4 | Supply chain: unpinned deps, `curl \| bash`, obfuscation, **live OSV**, typosquat | SC1-9 | S+net | ●·○·○·○·○ | Install-time **provenance pin** (source URL + per-file SHA-256); re-scan on hash change |
| 5 | Agent snooping: reads `.claude`/`.codex`/`.gemini`/`.continue`, MCP config | AS1-3 | S | ●·○·○·○·○ | **`.cursor/` paths + Cursor's credential store `state.vscdb`, holding `cursorAuth/accessToken`** (`SELECT value FROM ItemTable WHERE key='cursorAuth/accessToken'`) `[EPC F-08]` — a cost-research artifact reframed as a snooping target |
| 6 | Excessive agency (unrestricted tools, autonomy, scope creep) + MCP least privilege | EA1-5, LP1-4 | S | ●·○·◐·○·○ | `allowed-tools` **grants, does not restrict** `[L3 §2 R8]`: declared breadth vs invoked set; `disable-model-invocation`/`context: fork` first-class `[L3 §4]`. **Unambiguous leg blocks from the floor** (T013-T016). agnix ◐ understated: MCP-005, MCP-018, MCP-019, MCP-021 `[AGX]` |
| 7 | Bundled-hook execution surface | BH1-3 (**CC paths only**) | S | ◐·○·◐·○·○ | **Port to `.cursor/hooks.json`, `.claude/settings.json`, `hooks/hooks.json`** `[SS §7 #3]`; **blocks from the floor** (T017-T018). agnix checks hook *schema* (CUR-010..013, 017-019), never the command `[AGX]`. **[pi]** The audited unit is the **package, not the skill**: one pi `package.json#pi` manifest ships `skills` *and* `extensions` — in-process TypeScript, never type-checked by the loader, run with full user privileges and resolved provider auth — so a "skill install" can deliver auth-bearing executable code `[pi.md §5 #4, §8 P3]` |
| 8 | MCP tool poisoning + rug pull | TP1-4, RP1-2; **RP3 dead** — reads `manifest["version"]`, never set, #472 | S+L | ●·○·◐·○·○ | Homoglyph blocks from the floor (T003); re-scan on hash change replaces dead RP3's intent. TP4 is LLM-gated, silently absent under `--no-llm` ⇒ `checks_skipped[]` `[SS §3]` |
| 9 | Trigger abuse | TR1-3 — **dead: gate `manifest["triggers"]`, a key no spec defines, #458** | S | ○(dead)·◐·●·◐·○ | Replacement over the real `description`/`when_to_use` and `.mdc` `description`/`globs`; collision ≥60% overlap `[L1 §2c]` |
| 10 | Artifact integrity: mismatch, NUL, mixed script, oversize, **uninspected artifact (AE1)** | AE1-6 | S | ●·○·○·○·○ | AE1 wired into **our** exit code: "N read, M skipped, reason each" + `--fail-on-incomplete`; path escape blocks from the floor (T019) `[SS §7 #8]` |
| 11 | Memory poisoning: persistent injection, context stuffing | MP1-3 | S | ●·○·○·○·○ | Extend to writes into **always-on** surfaces (`CLAUDE.md`, `AGENTS.md`, `.mdc`); self-modification blocks from the floor (T020) |
| 12 | **Ceded — shell out, never restate:** P5 · P6-8 · PE1-5 · OH1-3 · TM1-4 · RA2 · AR1-3 · SSRF1-3 · DS1-4 · AST1-10 · TT1-6 · YR1-4 · SSR-1 | 60 IDs | S | ●·○·○·○·○ | Nothing — reimplementing YARA + taint + AST is the gold-plating trap `[SS §7 #15]`. SS's README publishes **wrong severities for 4 AST rules**; P6 fires on a docs heading (#512) — the baseline with mandatory `reason:` is the answer `[SS §3, §5]` |
| 13 | **Silent failure**: dead reference, cycle, missing required heading | — | S | ○·◐·◐·○·○ | `SK-R001` resolves `$var`/glob paths `check-paths.sh:50,71` **silently skips**; `SK-R002` Tarjan cycles — *"no evidence found of any existing tool doing this for skills"*; **no cycle rule in agnix's 455** `[L1 §2c-2d, AGX]` |
| 14 | **Never triggers / over-triggers** | — | S + **D** | ○·◐·●·◐·◐ | Catalogue-scope collision + dynamic activation **precision and recall** on a should-not-fire stratum `[L2 §2.1]` |
| 15 | **Context bloat — per-activation** | — | S | ○·◐·◐·○·● | Body vs CC's constants: 5,000 tok, 500 lines; re-attach keeps first 5,000 tok/skill in a 25,000-tok combined budget `[SOTA §3]` |
| 16 | **Project-level always-on tax** (every description, CLAUDE.md/AGENTS.md, `alwaysApply` rules, hooks, MCP manifests) | — | S | ○·○·○·○·◐ | **New — the one static metric neither SS nor agnix answers** `[SS §7 #4]`. `/skill-doctor` is per-session, CC-only, ≥v2.1.252 `[SOTA §2]`; agnix's nearest, `CC-MEM-009`, is a **per-file** cap `[AGX]` |
| 17 | **ICM / progressive-disclosure non-conformance** (L0-L4, depth, deferral ratio) | — | S | ○·◐·○·○·○ | **New as enforced** — codified `[PR:53-61]`, paper cited `[PR:82]`, never operationalised; **zero ICM rules in agnix** `[SS §7 #5, AGX]` |
| 18 | **Catalogue duplication** (two skills, one job) | — | S | ○·○·○·○·○ | jscpd behind a flag; cross-skill scope `[L1 §2c]` |
| 19 | **Trojanized third-party skill / marketplace supply chain** | SC/YR partial | S | ◐·○·○·○·○ | Quarantine-before-install, hash pin, re-scan. *ClawHavoc counts conflict — 1,184 vs 341 — unresolved* `[L3 §2 R11]` |
| 20 | **Negative lift — the skill actively hurts** | — | **D only** | ○·○·○·○·○ | **27.2% of measured skills have negative lift** (ACES, N=947, mean 0.2134, 95% CI [0.1967, 0.2301]) `[L2 §1.1]`; three-arm paired experiment `[L2 §1]` |
| 21 | **Cost / waste per skill**, incl. sticky context | — | D | ○·○·○·○·◐ | Cost-per-**activation**, not per-session `[L3 §2 R7]`. CC free from `skill.name` `[L2 §2]`; Cursor has **no per-skill cost tool in existence** `[EPC]` |

**Where inputs disagree, the pick.** `[L1 §3]` says "71 patterns / 17 categories" and "security is solved"; the security lens re-enumerated the source at **113 IDs / 18 categories** with **41 undocumented**, **4 dead rules**, **no Cursor coverage**, **no published FP/FN rate**, **21 of 61 open issues (~34%) about false positives** `[SS §3, §5, §106]`. **We take `[SS]`.** SkillSpector is a strong *detector*, a weak *oracle*: integrate for recall; never present its score as truth without the ledger and a human verdict.

---

## 3. The gate — "run this before I trust it"

| Stage | What it does | Budget |
|---|---|---|
| **G0 Quarantine + provenance** | Fetch into `~/.skillgate/quarantine/<sha>/`, **never a skills directory**; record source, commit, per-file SHA-256. **Never execute the target; never let the raw body enter the reviewing agent's context unlabelled** `[SS §4]`. **[pi]** Provenance inputs are **package specs, not only skill paths**: `npm:x@ver` / `git:repo@ref` / local — a missing `@ref` or version is **unpinned**, a local install is verified by an existence check alone, and the git leg runs `npm install --omit=dev` with **no `--ignore-scripts`**, so a lifecycle script is a supply-chain finding; `pi.extensions` entries are **executables** and take the ×1.3 weight (§4 Pack I) `[pi.md §4 G0, §5 #4]` | <1 s |
| **G1 Coverage ledger** | Every file gets a terminal outcome from an allow-listed reason enum; `--fail-on-incomplete` `[SS §4, §7 #8]`. **[pi]** The enum needs a **`silently skipped`** terminal reason: harnesses drop skills with no error at all — pi drops a skill whose `description` is missing and prunes by ignore-file without a diagnostic — and a silent drop must land as a ledger row, never as an absence; collisions record `{winnerPath, loserPath}`, which is also R005's shape `[pi.md §4 G1, §5 #5]` | <100 ms |
| **G2 Tripwire — Pack B, 20 Go rules, hard cap** | The blocking floor, working with **no Python and no Rust present**. `SK-T001..T020` in §4; **the block set below is derived from that table, not written beside it** | **<400 ms** |
| **G3 Deep scan** | `skillspector scan --no-llm --format json`, deterministic under `--no-llm` `[SS §2]`. Absent ⇒ `checks_skipped[]`, **CAUTION ceiling** (F13) | sec–min |
| **G4 Conformance** | `agnix --format json` over `.cursor/rules/*.mdc`, `.cursor/hooks.json`, `.cursor/mcp.json`, `AGENTS.md`, `.claude/settings.json` `[AGX]`; `skill-validator check -o json` for BPE counts | <1 s |
| **G5 Budget + ICM + graph** | Packs E/F/G: always-on budget with per-item attribution, ICM conformance, reference graph + cycles, effort minutes | <1 s |
| **G6 Semantic review (optional)** | Agent reads the source at every HIGH/CRIT → APPROVE/CAUTION/REJECT; **may add findings, never delete one** `[SS §2]` | 1–2 calls |
| **G7 Verdict + artifacts** | `report/v1.json` + SARIF 2.1.0; baseline entry with **mandatory `reason:`**; on APPROVE, install and pin the hash. **[pi]** The pin also records **the harness name and version the integration was tested against** — for any pi artifact, the pi version, since pi's documented peer convention is `*` and it ships no back-compat guarantee, so the supported range is ours to assert and to re-test `[pi.md §4 G7, §6 #11]` | <100 ms |

**The block set, derived from the catalogue.** Every dangerous class the bar names has an implementing rule in Pack B, so the floor blocks it with zero external runtime.

| Dangerous class | Rules | Blocks with **no** SkillSpector and **no** agnix? |
|---|---|---|
| Prompt injection (P1-4, P9) · Data exfiltration (E1-5) | T001-T003 · T004-T006 | **Yes** · **Yes** |
| Supply chain (SC1-9) | T007-T009 + G0 pin | **Yes**; OSV/typosquat depth ⇒ `checks_skipped[]` |
| Over-broad tools / permissions (EA1-5, LP1-4) | T013-T016 | **Yes** on the unambiguous leg (wildcard grant, plaintext secret, auto-consent, wildcard bind); graded leg warns (T014) |
| Bundled-hook execution (BH1-3) · Agent snooping incl. Cursor credential store (AS1-3) | T017-T018 · T010-T012 | **Yes** · **Yes** |
| Integrity / persistence (AE2, RA1) · uninspected artifact (AE1) | T019-T020 · G1 ledger | **Yes** · **Yes** |
| The 60 ceded IDs, OSV, YARA, AST, taint | none — by design | **No** ⇒ `checks_skipped[]` ⇒ **CAUTION ceiling** (F13) |

### (a) a skill I wrote vs (b) a skill from the internet

| | (a) My skill | (b) Internet skill |
|---|---|---|
| Provenance · G3 | git worktree, G0 skipped · optional, absence warns | **G0 mandatory**, no install before verdict · **mandatory**, absence ⇒ CAUTION ceiling, install refused in CI |
| Baseline · Scope | inherited, content-bound fingerprints · **diff-scoped** (Sonar's new-code primitive) `[L1 §1, L3 §3.3]` | **empty — no inherited suppressions**, fail-closed `[SS §4]` · whole bundle |
| Blocks | new Blocker/High security; dead refs = 0; cycles = 0; estate budget exceeded | all of (a) **plus** any tripwire hit, any CRITICAL/HIGH from G3, any uninspected artifact, any unpinned/mutated artifact |
| Warns · Re-gate | ICM, body >5,000 tok, collision, house headings, debt minutes · on commit | same plus every MEDIUM · **on every version/hash change** (rug-pull class) |

### Claude Code specifics

| Where | Mechanism |
|---|---|
| Pre-install · CI | `/skill-architect:skill-gate <url>` → G0-G7, install only on APPROVE `[SA:README.md:88-92]` · `skillgate scan --format sarif` → GitHub Code Scanning; exit `0`/`1` (over threshold **or incomplete**)/`2` `[SS §4]` |
| Pre-activation | **Claude Code has no blocking pre-activation surface today.** Ship quarantined skills with **`disable-model-invocation: true`**, plus `skillOverrides` and `/skill-doctor` `[SOTA §3]`. *No input supports a CC hook that blocks activation: `[OT G9]` records issue #320 "Agent Harness Hook Semantic Conventions" as open, **no PR**, no blocking semantics. We do not design one.* Pre-install and CI enforce |
| Config surface | **`.claude/settings.json`** — `permissions.allow`/`deny`, MCP enablement, hook registration: CC's over-broad-permissions surface; `hooks/hooks.json` by path for plugin-bundled hooks. agnix ships **30 `CC-SET-*`** here; we shell out for schema and add only the execution-surface security leg (T017) `[AGX]` |
| Always-on inputs | `CLAUDE.md`; `description`+`when_to_use` truncated at **1,536 characters** (`skillListingMaxDescChars`); `skillListingBudgetFraction` **0.01** of the context window, shipped in **v2.1.129** `[SOTA §3]` |
| Dynamic · Packaging | `skill.name` on token/cost usage `[L2 §2]` · `.claude-plugin/plugin.json` at **v0.4.0**, `"skills": "./skills/"`; add `skill-gate` beside `skill-audit`/`skill-rewrite` |

### Cursor specifics

| Where | Mechanism |
|---|---|
| Pre-install · CI | Same CLI and `SKILL.md`; `skillspector mcp` (FastMCP) is a second editor-native path `[SS §7 #2]` · identical binary and exit codes, but **no native Cursor SARIF viewer** `[SS §7 #12]` |
| Skill roots — **[pi] correction** | **Cursor ships a stock skills system.** The rows below described the Cursor surface as `.mdc` rules + `AGENTS.md`, and that silence was read as absence. Cursor discovers `SKILL.md` from **four native roots** — `.agents/skills/`, `.cursor/skills/`, `~/.agents/skills/`, `~/.cursor/skills/` — plus **four compatibility roots**: `.claude/skills/`, `.codex/skills/`, `~/.claude/skills/`, `~/.codex/skills/`. Each is walked **recursively**, a `.cursor/skills/` or `.agents/skills/` folder **anywhere in the repo** is picked up and scoped to that subtree, and `name` must be lowercase/hyphenated and **match the parent folder**. Three consequences: a `~/.claude/skills` skill is live in Cursor with **zero config**, so root, portability and per-harness-validity findings hit Cursor **first** (`skillgate env`, slice 2); agnix's `CUR-*` cover `.mdc`, **not `SKILL.md`**, so Cursor `SKILL.md` conformance is ours, not shelled out; and **precedence across the eight roots is undocumented**, making multi-root collision a Cursor finding class that needs its own two-roots-one-name fixture `[pi.md §4a, §5 #5]` (re-verified against `cursor.com/docs/skills.md`, 2026-09-12) |
| Pre-activation | `.cursor/hooks.json` `beforeReadFile` deny on an unapproved skill path. **Conditional:** skill-architect's roadmap carries `SP1 · beforeReadFile gating spike` as a *hard gate* ahead of S3 `[SA:.scuba/roadmap.md:32,59]`; until it lands Cursor pre-activation is **warn-only** and pre-install/CI enforces. Hook fail-open, bounded timeout `[spec R-CS-03, R-CS-30]`. **[pi]** **Scope the claim to the read path.** A `beforeReadFile` deny gates reads only; the `bash` leg is open on every harness examined — pi's own docs instruct the agent to use `bash` when `read` is unavailable, and nothing stops `bash cat SKILL.md` when it is — so "no un-pinned `SKILL.md` enters context by any tool" is **not achievable today**. Ship it as "blocks the read path; the bash path is unsolved," with the bash leg named in `checks_skipped[]` (F14) `[pi.md §7 row 2, §6 #4]` |
| Conformance — **agnix already covers it** | `.mdc` frontmatter CUR-002/003, **CUR-004 invalid glob**, **CUR-005 unknown keys**, **CUR-007 `alwaysApply` with redundant globs**, **CUR-008 bad `alwaysApply` type**, **CUR-009 missing `description`**, CUR-020 ignored plain-markdown rule; hooks schema CUR-010..013/017-019; 26 `MCP-*`; `AGM-001..006`; CR-SK-001 `[AGX]`. **We shell out and add only the tri-state *per harness version* agnix does not attempt** |
| Security — **nothing covers it** | Hook *execution* content (BH port), `.cursor/` path reads, **`state.vscdb` / `cursorAuth/accessToken`** `[EPC F-08]`, injection over `.mdc`/`AGENTS.md`. SS reads no Cursor path `[SS §1.4]`; agnix runs no security rule on any surface `[AGX]` |
| Always-on · Dynamic · Packaging | `AGENTS.md` + every `alwaysApply: true` rule; denominator `preCompact.context_window_size` `[CTS §1]` · hook spool → attribution (`beforeReadFile` path heuristic, **untested — U-02**); `preCompact.context_tokens` is *the only real Cursor-reported token number in the entire hook surface* `[CTS §1]`; Enterprise OTel `cursor.skill.activated` when available `[CTS §2]` · `.cursor-plugin/plugin.json` at v0.4.0; copy/symlink into the Cursor plugins folder `[SA:README.md:64]` |

---

## 4. Architecture

| Component | Call | Role and boundary |
|---|---|---|
| **`skillgate` (Go, zero-dep, single binary)** | **new** | Tripwire, reference graph + Tarjan cycles, always-on budget, ICM scoring, ledger, emitters. **Never prices anything** — minutes and tokens only `[SY §3]` |
| **SkillSpector CLI** | **integrate — shell out, opt-in, never vendor** | 113 IDs, OSV, YARA, taint, MCP lanes. Unmodified binary, `--no-llm`. Absent ⇒ `checks_skipped[]`, CAUTION ceiling, never abort `[SS §7 #2, #11]` |
| **agnix (Rust, MIT+Apache-2.0, 455 rules)** | **integrate — shell out. Settled, not spiked** | `CUR-*`, `MCP-*`, `AGM-*`, `CC-SET-*` conformance. We add the per-harness-version tri-state, budget, ICM, graph and all security — **none of which exist in the 455** `[AGX]` |
| **Pack B tripwire (20 Go rules)** | **new, deliberately small** | The blocking floor when neither binary is present. **Hard cap at 20.** No YARA, OSV, AST or taint — SkillSpector-only, and the report says so |
| `skill-validator` · jscpd · `skillscore` | reuse · flagged off · degrade | Real `o200k_base` counts (`check -o json`, **not** `analyze content`) `[L1 §2d]` · duplication · *conflict: `skill-audit` says 7 dimensions/npm, upstream 6 + Dart*; degrade, never `exit 1` `[L1 §3, SS §7 #11]` |
| `skill-audit` / `skill-rewrite` | **policy reused, code superseded** | PL001-PT002 → `SK-*`; the rewrite loop's "re-audit ≥ original" has no SkillSpector analogue `[SS §6]`. Today it hard-`exit 1`s on missing `skill-validator`/`skillscore` (`SA:skills/skill-audit/SKILL.md:46-47`, verified live) against its own "degrades gracefully" promise, and is single-skill by contract (`:33`) — the engine must be catalogue-scoped and degrade `[L1 §2d]` |
| `cursor-profiler` spool · `skill-architect/profiler` | reuse · **reuse after repair** | Lossless JSONL capture, redact before disk; `profile/v1` additive only `[spec, Q10]` · D-1/D-2/D-3/R1/R2 **already landed in the working tree**; **D-6 and the stale `claude_code.go:83` comment remain open** `[SY §6]` |
| DuckDB · `calibration.jsonl` · baseline file | reuse · **new** · **new (clean-room, Go)** | JSONL is truth, `.duckdb` a rebuildable cache `[Q9]` · observed fix minutes from git history; records, never estimates `[SY §3]` · content-bound fingerprints, **mandatory `reason:`**, fail-closed on drift; suppressed findings stay in machine output and never score `[SS §7 #7]` |
| Emitters · OTel `gen_ai.*` | **new, thin** · **alias, gated off** | ~150 lines each; SARIF never the sole output — Sonar's importer forces `SECURITY` and drops effort `[SY §5, L1 §4]` · `gen_ai.skill.name` is unmerged PR #498, every name `development`; **no `gen_ai.*` literal in `types.go`** `[OT S-1, S-2]` |

**Integrate vs reimplement — decided, with licensing. Integrate both; shell out to unmodified binaries.** Reimplementing means owning 113 IDs / 27 analyzers / ~2,500 tests / OSV / native YARA `[SS §7 #15]`, or agnix's 455 rules `[AGX]`. SkillSpector is **Apache-2.0**, agnix **MIT + Apache-2.0**, against skill-architect's **MIT**: **vendoring** needs LICENSE + NOTICE + change notices plus a permanent Python sync burden; **shelling out to an unmodified binary creates no such obligation** `[SS §7 #15]`. The tax is named: SkillSpector is **not on PyPI** (404), git-URL install, **Python ≥3.12,<3.15** (dev machine 3.9.6), ~20 deps, native `yara-python`, Docker built locally `[SS §4]`, and `[SA:AGENTS.md:6]` bans Python. That is why **Pack B exists and is hard-capped at 20 rules**: the gate blocks all five dangerous classes with zero external runtime and degrades honestly for the rest — resolving the conflict where `[L1 §4]` says shell out and `[Q8]`/`[SY §7a]` says no Python. **Opt-in, off by default, never load-bearing for a REJECT.** Ratification: §8 D2.

### The static catalogue we ship

**Pack A · Security depth** — all 113 SkillSpector IDs via G3 (*borrowed*). **Pack C · Spec conformance** — frontmatter validity, `description` ≤1024 / `name` ≤64 / `compatibility` ≤500 `[L1 §2b]`, link resolution, BPE counts, contamination (*borrowed: skill-validator + agnix*).

**Pack B · Tripwire `SK-T001..T020` — new implementation, borrowed semantics, hard cap 20.**

| ID | Rule | Sev | Min | Origin |
|---|---|---|---|---|
| T001 | Zero-width or bidi control characters in loaded text | Blocker | 10 | P2/P9 `[SS §3]` |
| T002 | Instruction-override phrasing in `description`/body | Blocker | 15 | P1 |
| T003 | Homoglyph / mixed-script in `description` or MCP tool name | Blocker | 20 | TP2 |
| T004 | Bundled script transmits to a literal remote host (`curl`/`wget`/`nc`) | Blocker | 20 | E1/E2 |
| T005 | Env harvest (`env`/`printenv`/`os.environ` dump) reaching a sink | Blocker | 20 | E4 |
| T006 | Credential-shaped literal in a bundled file | Blocker | 15 | L1 min set `[L1 §3-4]` |
| T007 | Network output piped to a shell (`curl … \| sh`) | Blocker | 30 | SC2 `[L1 §4]` |
| T008 | Remote fetch without a pinned SHA/tag; third-party bundle unpinned | Blocker | 10 | **new** — dead RP3's intent |
| T009 | Obfuscation: base64/hex decode feeding `eval`/`exec`/shell | High | 25 | SC3 |
| T010 | Reads `.claude/`, `.codex/`, `.gemini/`, `.continue/` | High | 15 | AS1-2 |
| T011 | Reads `.cursor/`, `~/.cursor/hooks.json`, `.cursor/mcp.json` | High | 20 | **new** — AS off CC-only paths `[SS §1.4]` |
| T012 | Reads or queries Cursor `state.vscdb` / `cursorAuth/accessToken` | Blocker | 15 | **new** — Cursor's credential store `[EPC F-08]` |
| T013 | `allowed-tools` wildcard, or absent with an executable bundle | Blocker | 20 | **new** — EA1 at the floor `[L3 §2 R8]` |
| T014 | Declared tool set exceeds invoked set beyond threshold | Medium (**warn**) | 20 | → `B` input `[L3 §1.3]` |
| T015 | MCP server with plaintext secret in `env`, or wildcard HTTP binding | Blocker | 20 | MCP-018/021 `[AGX]` |
| T016 | MCP tool invocable without user consent / auto-approved | High | 15 | MCP-005, LP1-4 `[AGX]` |
| T017 | `.claude/settings.json` or `hooks/hooks.json` registers a hook executing bundled content — **[pi]** or a `package.json#pi` manifest names bundled TypeScript under `extensions` | Blocker | 30 | **new** — BH1-3 at the floor `[SS §3]`; **[pi]** the pi surface **extends T017, it is not a 21st rule — the cap at 20 holds** `[pi.md §4 G2, §5 #4]` |
| T018 | `.cursor/hooks.json` hook command executes a bundled path | Blocker | 30 | **new** — BH ported to Cursor `[SS §7 #3]` |
| T019 | Path escape: `../` outside the bundle root, or a symlink leaving it | Blocker | 15 | AE2 |
| T020 | Self-modification / persistence: writes into `.claude`, `.cursor`, a shell rc — **[pi]** or `.pi/`, `~/.pi/agent/trust.json`, `settings.json#packages`; the trust file is pi's only input-loading guard, so a write there auto-answers the trust prompt | Blocker | 25 | RA1; `[pi.md §4 G2]` |

**Every tripwire rule ships two tests: fires-on-real-input *and* does-not-fire-on-the-clean-corpus.** Pack B mirrors SkillSpector's P/E/SC/AS/PE/RA/TP/AE semantics and inherits its false-positive profile (~34% of 61 open issues `[SS §5]`); measuring SkillSpector's FP rate does not measure ours. Clean corpus: `skill-architect/skills/*` plus PR 2's fixture estate.

| Pack | Rules / measures (ID · severity · effort min) | Status |
|---|---|---|
| **D · Per-harness validity** | `SK-H001` key **invalid** for the harness (High, 10) · `H002` key **silently ignored** (Medium, 10). CC fields: `name`, `description`, `when_to_use`, `allowed-tools`, `disable-model-invocation`, `context: fork`, `license`, `compatibility` | **Shrunk to a shell-out plus a thin delta.** agnix CUR-004/005/007/008/009/020 already cover the `.mdc` surface every candidate claimed as new — CUR-005 is the *invalid* leg, CUR-020 the *silently-ignored* leg `[AGX]`. **NEW only:** the tri-state **per harness version**; no input enumerates per-version acceptance, so the matrix needs its own spike |
| **E · Always-on budget** | `SK-B001..B005` (High, 45): one budget, **per-item attribution** — each skill's L0/L1 description line, `CLAUDE.md`/`AGENTS.md`, every `alwaysApply: true` rule, hook definitions, MCP manifests, plugin metadata. **[pi] Two item classes were missing, both larger than the manifest *file* that list counts:** (a) the **description and guideline text an installed tool injects into the prompt every turn** — counted for *active* tools, and it **survives compaction** while a loaded skill body does not; (b) the **tool input schema** — `name` + `description` + `input_schema` ride the separate `tools` request parameter, never appear in the system-prompt string, and are billed every turn: measured on one real tool, **832 chars of schema, 1,007 for the full entry, against 255 chars of injected prose — 3.9×** `[pi.md §5 #1, #2a, §6 #3]`. Denominators: CC 0.01 × context window and 1,536 chars/description; Cursor `preCompact.context_window_size`. **Distinct from per-activation cost** | **NEW — the one static metric neither SkillSpector nor agnix answers.** agnix's nearest, `CC-MEM-009`, is a per-file cap `[SS §7 #4, AGX]` |
| **F · ICM** | `SK-I001` L2 content belonging in L3 (High, 60) · `I002` reference depth >1 (Medium, 45) · `I003` deferred fraction below floor (Medium, 30) · `I004-I006` `branch_count`, 60/30/10 layer ratio, `activation_tokens` vs the 5,000-tok/500-line budget. `always_loaded_tokens` is a **measure, in characters against the 1,536 truncation cap** `[SY §7a]` | **NEW as enforced** — in `[PR:30-38, 53-61]`, never machine-checked; **zero ICM rules in agnix** `[SS §7 #5, AGX]` |
| **G · Reliability / graph** | `SK-R001` dead reference **resolving `$var` and glob paths** (High, 10) · `R002` cycle, Tarjan (High, 60) · `R003` undeclared binary vs `compatibility:` (Medium, 10) · `R004` weak/absent `description` (Blocker, 15) · `R005` trigger collision ≥60% (Medium, 30) | **NEW** — *"no evidence found of any existing tool doing this for skills"*; no cycle rule in agnix's 455 `[L1 §2c-2d, AGX]` |
| **H · Maintainability** | `SK-M001` body >500 lines or >5,000 tok (Medium, 90) · `M002` catalogue metadata over the 25,000-tok re-attach budget (High, 45) · `M003` duplicated block ≥N tokens (Low, 25) · configurable required headings replacing PL002's hard-coded loop (Low, 5) | **borrowed + configurable** `[L1 §2d, §4, SOTA §3]` |
| **I · Scoring & hygiene** | Severity + remediation minutes per rule; **diminishing returns** (full/half/quarter/ignored), confidence-scaled, **×1.3 executables only — docs at base weight**; baseline with mandatory `reason:`; ledger + `--fail-on-incomplete`; every rule needs both tests | **borrowed** `[SS §7 #7, #8, #9, #16]` |

**Effort minutes are illustrative placeholders until calibrated** `[L1 §4]` — refusal F4, §5.

**Pack E, worked** `[SOTA §3-4]`: `skill-audit` `description` 262 chars ≈ **65** always-loaded tokens; `skill-rewrite` 441 chars ≈ **110**; estate total **175**. Budget = 0.01 × the ~200K reference window = **2,000 tokens**. **175 / 2,000 = 8.75% of the listing budget consumed by two skills.** (Arithmetic re-verified step by step.) Per-item attribution makes that actionable; a single number does not.

### How the LLM-assisted checks stay honest

| Rule | Source |
|---|---|
| **The LLM may add findings and raise confidence; it may never delete a deterministic one.** Unconfirmed additions are tagged `llm-unconfirmed`, not dropped. **Fail closed**: LLM failure ⇒ passthrough of the deterministic findings, never a pass | `[SS §2]` verified invariant + test |
| **The target is untrusted input**; instructions inside skill content are ignored, and text asserting *"verified safe"* **raises** suspicion. Every judgment dimension carries explicit "Do NOT flag if…" negative criteria | `[SS §2, §7 #1, #6]` |
| **`insufficient_evidence` is an excluded state, not a middling score** — replacing `skill-audit`'s *"when in doubt, mark the dimension as 1"* (`SKILL.md:155`), a live upward bias | `[WD B3]` |
| **Arithmetic happens in code, never in the model**; never re-ask a judge what a script answered; determinism claimed only under `--no-llm` | `[WD R3, R4, D32; SS §2]` |

### Dynamic, data store, scoring, reporting

- **Dynamic rides only on surfaces the targets expose.** CC: `skill.name` on token/cost usage `[L2 §2]`. Cursor: hook spool (`preCompact.context_tokens`) and, when enabled, Enterprise OTel `cursor.skill.activated` `[CTS §1-2]`. Nothing is designed against a surface the targets lack.
- **Data store: there isn't a new one.** JSONL spool + DuckDB + `report/v1.json` + `calibration.jsonl` + baseline. No server, no daemon `[SY §3, Q9]`.
- **Scoring: three numbers, never blended.** **Risk** — 0-100, diminishing returns, confidence-scaled, docs exempt from the ×1.3 executable weight `[SS §7 #9]`. **Debt** — Σ remediation minutes; **letter grade and debt ratio withheld until n ≥ 20 observed fixes per rule** `[SY §4c F4]`. **Yield** — SDY `(Waste$ + Risk$) × c ÷ E` in a **two-lane queue**: a hotspot lane ordered by blast radius × permission scope that jumps the queue regardless of EV, plus an absolute floor dropping skills under an org-set recoverable total `[L3 §1.2, SY §2d-2e]`.
- **Reporting and packaging.** `report/v1.json` is the contract; SARIF 2.1.0 and Sonar generic-issue are thin emitters behind flags `[SY §5]`; HTML is a later slice, borrowed in shape from Warp `[WD B7]`. One repo, one binary, two plugin manifests that already exist (both **v0.4.0**), one new `skill-gate` skill built the way it preaches — a short `SKILL.md` with `references/` per rule pack. **The tool must be built to the ICM standard it enforces.**

---

## 5. Measurement contract

**Reused verbatim from `[SY §4]`; not re-derived.** Four tiers: tier 1 billed (Admin API, **unavailable today** `[Q2]`), tier 2 measured tokens / estimated dollars, tier 3 vendor-estimated, tier 4 `chars/4` **ordinal only**. Three non-negotiables: never compare across tiers; stamp **both** sides of a delta; a tier-4 side forbids any dollar. **On today's environment the word "billed" never appears in the product.** "Proved" requires all four of: paired bootstrap BCa 95% CI excluding 0; McNemar on task success; activation **precision** not falling; `lift(C) > 0` against the no-skill arm `[SY §4b]`. Nulls print their achieved MDE (N ≈ 12 / 54 / 216 `[L2 §125]`). Refusals **F1-F11** ship as a machine-readable `refusals[]` array `[SY §4c]`.

| # | Addition this mandate requires | Why |
|---|---|---|
| **F12** | **The tool never says "safe" or "clean."** Verdicts are APPROVE / CAUTION / REJECT, always beside the coverage ledger; *"don't rely on the score alone"* | `[SS §4]`; SkillSpector publishes **no FP/FN rate**, ~34% of open issues are FPs `[SS §5]` |
| **F13** | **The fail-closed invariant, stated once here and propagated to every stage: no verdict above CAUTION when the ledger is incomplete.** Any uninspected artifact (AE1) **or any skipped check** caps the verdict and is named in `checks_skipped[]`. No stage may both "run warn-only" and "block on CRITICAL" — a stage either participates in the block set (§3) or its absence caps the verdict | `[SS §7 #8, #11]` |
| **F14** | **[pi] A runtime gate audits *before* load; it never contains what has already loaded, and it never claims the `bash` leg.** Handlers run in **load order**, later handlers see earlier mutations, the first block wins, and any extension may re-register `read`/`bash`: **order, not privilege, decides.** The same candour is owed to Claude Code hooks and to Cursor `beforeReadFile` — neither isolates what is already in-process. Every gating claim is therefore scoped to **the read path** and to **the moment before load**; the `bash` leg ships in `checks_skipped[]`. F13 bounds what the *ledger* may claim; **F14 bounds what a *runtime gate* may** | `[pi.md §7 rows 1-2, §8 P2]` |

---

## 6. Roadmap

| Slice | Ships | Independent because | Gated on |
|---|---|---|---|
| **1 · THE SAFETY GATE** | G0-G3 + G7: quarantine, provenance pin, ledger, **the 20-rule tripwire covering all five dangerous classes**, SkillSpector shell-out with honest degradation, APPROVE/CAUTION/REJECT, `report/v1.json` + SARIF + exit codes, shipped as `skill-gate` in **both** plugin manifests. **[pi]** If D9 admits a third packaging target, its manifest **must list `skills` beside `extensions`**: a package manifest **short-circuits** the convention-directory walk, so an extensions-only manifest ships the binary and **silently drops the `skill-gate` skill — the interface** (SkillSpector ships exactly this defect today, and §1's product shape is the skill). **Acceptance test:** after install, the *skill* is discoverable, not only the tool `[pi.md §3 row 1, §6 #1]` | Pure filesystem. No telemetry, no scoring, no profiler | nothing |
| **2 · Environment-aware static** | G4 agnix shell-out (`CUR-*`, `MCP-*`, `AGM-*`, `CC-SET-*`) + Pack D's tri-state delta + **Pack E always-on budget with per-item attribution** | Separate rule packs, own report section | **nothing — the agnix diff is settled by enumeration, not spiked** `[AGX]` |
| **3 · ICM + graph + debt** | Packs F and G, `effortMinutes` on every finding, `calibration.jsonl`, baseline with mandatory `reason:` | Filesystem + git history only | nothing |
| **4 · Dynamic, Claude Code** | Read `skill.name`; per-task activation comparison; close D-6 `stopping_rule`; delete the stale `claude_code.go:83` claim | Touches `skill-architect/profiler` only | nothing (CC OTel exists today) |
| **5 · Dynamic, Cursor** | Hook spool → attribution; per-skill aggregates (R-AT-10) | Reuses slice 4's comparator | **Cursor install + hook**; U-02 untested `[Q]` |
| **6 · Rank + prove** | SDY two-lane portfolio, three-arm paired experiments, HTML report | Degrades: with no report it ranks on Waste$/Risk$ and prints `E: unknown` | Org constants `C_f`; funded N |
| *Waits on a Cursor admin key* · *on Enterprise OTel* | Billed `C_a` via `filtered-usage-events` joined on `conversationId`, `c` above 0.5 · first-party `cursor.skill.activated` with `.trigger`/`.source` | — | Admin API key `[Q2, CTS §3]` · work admin enables it |

**The first three PRs — 1:1 with slices 1-3.** Each touches disjoint code paths and disjoint report sections; none depends on another's output.

| # | PR | Proves | Test |
|---|---|---|---|
| **1** | `skillgate gate <path\|url>` — slice 1 end to end | A third-party skill is quarantined, scanned, verdicted and pinned **before it can run**, on a machine with **no Python and no Rust** | Malicious fixture (zero-width instruction, `curl\|sh`, `state.vscdb` read, wildcard `allowed-tools`, `.cursor/hooks.json` executing a bundled script, oversize artifact) → REJECT + ledger; the same bundle with both binaries absent → still REJECT on the tripwire, **never APPROVE**; clean corpus → zero tripwire findings |
| **2** | `skillgate env` — G4 + Pack D delta + Pack E | The estate's always-on tax, itemised, for both harnesses — a number nothing else computes | Fixture estate (3 skills, `CLAUDE.md`, 2 `.mdc` rules one `alwaysApply`, 1 MCP manifest, 1 `.claude/settings.json`) → per-item table summing to the budget; agnix findings pass through un-restated. **[pi] Ground-truth bench — named acceptance test for slice 2:** run the same fixture estate under pi and check `skillgate env`'s per-item arithmetic against the **assembled system-prompt string** that pi's `before_agent_start` hands an extension alongside the structured build options — **exact, in characters**. Claude Code and Cursor expose no such string, so this is the only place the metric can be proved before it ships for them. **Tokens are a separate claim:** pi never tokenizes, so the token leg reconciles the first request's `usage` minus the user message, the active-tool array, Anthropic's hidden tool-use preamble and any OAuth block, priced with `count_tokens` and reported within its stated estimate tolerance `[pi.md §5 #2, §6 #9, §8 P1]` |
| **3** | `skillgate scan` — Packs F + G, effort minutes, `report/v1.json`, `calibration.jsonl` | Debt in **minutes** for a real skill, plus cycle and `$var` dead-reference detection no tool does | Golden skill dir with each defect → expected report; fixture git history → expected ledger rows |

---

## 7. What we deliberately do not build

| Not building | Why |
|---|---|
| A reimplementation of SkillSpector's catalogue · a **vendored** copy · a **hard** dependency | 113 IDs, 27 analyzers, ~2,500 tests, OSV, native YARA `[SS §7 #15]` · Apache-2.0 into an MIT repo ⇒ LICENSE + NOTICE + change notices plus a permanent Python sync burden · Python ≥3.12 vs `[SA:AGENTS.md:6]`; dev machine runs 3.9.6 `[SS §4]` |
| **A Cursor `.mdc` / `mcp.json` / `AGENTS.md` conformance linter** | agnix already ships 20 `CUR-*`, 26 `MCP-*`, `AGM-001..006`, 30 `CC-SET-*`; we shell out and build only the tri-state delta `[AGX]` |
| A Claude Code pre-activation blocking hook · a 21st tripwire rule | **No such surface exists**: `[OT G9]` issue #320 is open with no PR; `disable-model-invocation: true` is the documented mitigation `[SOTA §3]` · the cap is what makes "opt-in Python" and "the gate still blocks" both true |
| A SonarQube Java language plugin · SARIF as the sole output | Java in a Go shop, a Markdown grammar, a server as a hard dependency `[L1 §4a]` · Sonar's importer forces `SECURITY`/`CONVENTIONAL` and has no effort field — it would destroy the debt model `[L1 §4b]` |
| Session-transcript harvesting across six harnesses (Warp's moat) | Six undocumented formats, a Python codebase, a privacy posture; `skill_coverage` **measures the sampler, not the user** (true 0.95 → reported 0.75) `[WD §3.1]` |
| A curved letter grade · any letter grade before calibration | `curve(s)=0.5+0.5s` lifts the floor to 0.51 and makes the A band a function of how many skills you installed, beside a sales CTA `[WD R1, R2]` · Sonar's 30 min/line denominator is certainly wrong for skills; **F4** holds until n ≥ 20 `[SY §4c]` |
| A duplication engine or secrets scanner · a daemon, HTTP ingestor, OTLP export, Postgres/ClickHouse · any `gen_ai.*` literal in `types.go` | jscpd and Sonar's Shell+Secrets analyzers cover them `[L1 §3, §4]` · `[Q4, Q9, spec forks (i)(ii)]` · every name is `development`; `gen_ai.skill.name` is unmerged PR #498 `[OT S-1, S-2]` |
| **[pi]** A cache-miss / cache-invalidation detector as the per-skill cost signal | pi's `missedTokens = min(prev.promptTokens, promptTokens) − cacheRead` (discarded at ≤1,024) measures prefix **invalidation** — idle gaps, model switches, compaction resets — not skill load. A `SKILL.md` **appends** to a cached prefix, so `cacheRead` ≈ the previous prompt and the detector reads ≈0 on every skill; it has no skill attribution and emits **dollars**, against §4's "never prices anything." **Invariant:** a per-skill cost number derives only from a quantity that changes when *and only when* a skill body enters context — the turn-over-turn **prompt-token delta** (`input + cacheRead + cacheWrite`) after the `SKILL.md` read, paid share `input + cacheWrite`, reported as an **upper bound** `[pi.md §5 #7, §6 #12]` |
| Executing or sandboxing an untrusted skill · a rule without both tests | The gate's first rule is *never execute the target*; dynamic signal comes from **our own** sessions · SkillSpector shipped **4 documented dead rules** and carries a ~34% FP backlog `[SS §5, §7 #16]` |

---

## 8. Risks and decisions for the user

| # | Decision | Why it is yours | Default if you say nothing |
|---|---|---|---|
| **D1** | **Does skill-architect take a safety position at all?** `[SA:.out-of-scope.md:10]` says the plugin does not *"guarantee security… not a substitute for a full security audit."* Slice 1 walks that back | The product's public boundary | Narrow to "we gate, we do not certify"; ship slice 1 `[SS §8.1]` |
| **D2** | **Is an opt-in external Python binary acceptable?** `[SA:AGENTS.md:6]` bans Python and `[Q8]` says no Semgrep/no Python; yet `skill-audit` already hard-requires `skillscore` via npm and `[L1 §4]` recommends shelling out | A convention call, not a research finding. My read: the ban targets *authored* scripts | Yes — opt-in, off by default, never load-bearing; the 20-rule tripwire covers the no-Python case `[SS §8.2]` |
| **D3** | **Accept agnix as a shipped dependency of slice 2** (shell-out, MIT+Apache-2.0), and accept that our Cursor *conformance* differentiation is gone — the differentiation is Cursor **security**, budget, ICM, graph | Was "the largest unexamined risk in the plan"; **resolved by enumeration, not deferred** `[AGX]` | Adopt the shell-out; re-aim slice 2 at Pack E and the security port; no spike |
| **D4** | **Does the gate block merges or only warn?** Sonar and Datadog PR Gates both block `[L3 §3.4]` | A heuristic scan will false-positive on day one — SkillSpector's backlog is ~34% FP reports `[SS §5]` | Block on the tripwire + CRITICAL/HIGH; warn on the rest; baseline entries need a written `reason:` |
| **D5** | **Own `C_f` and the engineer-hour rate.** `Risk$` dominates `Waste$` in every row; at ½`C_f` the EV gap between two skills collapses from $101 to **$6** `[SY §2e]` | These two numbers set the ranking | Ship slices 1-3 without them; SDY waits for slice 6 |
| **D6** | **Will the team time 20 real fixes per rule?** Without the calibration ledger, **F4** means minutes ship and a letter never does `[SY §4c]` | A measurement commitment, not a coding task | Minutes only, forever, honestly |
| **D7** | **Cursor access.** `~/.cursor/` is absent on the dev machine; Cursor lives on an enterprise-managed work machine `[Q, SA:.scuba/roadmap.md]`. Slices 2-3 are static and unblocked; slice 5 and the `beforeReadFile` gate need an install and the SP1 spike | Environment, yours alone | Static Cursor support against fixtures now; dynamic Cursor warn-only |
| **D8** | **Rule-ID rename `PL*`/`PT*` → `SK-*` and catalogue-first scope** — two user-visible breaks to a shipped plugin's exit-code contract `[L1 decisions 4-5, SY §7c]` | Breaking change to something already published | Take both in one release or neither |
| **D9** | **[pi] Is pi a third target, or only a bench fixture?** Static support (its skill roots, `pi.skills` / `pi.extensions`, its diagnostic vocabulary) is small and rides slices 2-3. A published pi package is a **new artifact class** — TypeScript beside the Go binary, never type-checked by the host loader, in lockstep with a harness that ships no back-compat guarantee — against `[SA:AGENTS.md:6]`'s language ban and §4's "one Go binary, zero-dep" `[pi.md §8 P1, P4, P5]` | Product scope plus a standing maintenance commitment: the question D2 answered for Python, asked again for TypeScript | **Static: yes**, slices 2-3 — cheap, and it is where the Cursor compatibility roots get exercised. **Package: yes, but only as a thin shell** — all logic in Go, binary resolved from `SKILLGATE_BIN`/PATH, **absent ⇒ `checks_skipped[]` + CAUTION ceiling (F13)**, **never** a `postinstall` download (that is the pattern G0 flags, and it silently no-ops on 2 of 3 package managers), per-platform `optionalDependencies` deferred; pinned version range plus a nightly canary. **pi dynamic deferred behind slice 4** — except the Pack E ground-truth bench, which lands in **slice 2** |

**Thin evidence, stated plainly.** SkillSpector was never run live against skill-architect's own skills (Python 3.9.6, no `uv`, Docker down) — **the cheapest go/no-go before slice 1 is one `--no-llm` scan in a 3.12 venv against `skills/skill-audit` and `skills/skill-rewrite`** `[SS §9]`. It publishes no FP/FN rate `[SS §5]`, and **Pack B's own FP rate is unmeasured** until the clean-corpus tests run. `/skill-doctor` could not be run (local CC 2.1.221 < 2.1.252) and its first version is disputed (docs ≥2.1.252 vs press 2.1.261) `[SOTA §2]`. The Cursor hook activation heuristic (U-02) is untested and is the sole source of `A` for Cursor `[L3 handoff]`. ClawHavoc counts conflict (1,184 vs 341) `[L3 §2 R11]`. `skillscore`'s dimension count and runtime conflict between `skill-audit` and upstream `[L1 §3]`. That `effortMinutes` rolls into `sqale_index` is docs-assistant asserted, not primary-page verified `[L1 handoff]`. Pack D's per-harness-version matrix has **no primary source**. One correction to the arena verdict: it placed `skill-audit`'s hard `exit 1` pair at `SKILL.md:45-46`; the live file puts it at **`:46-47`**, matching `[SS §6]` — the inherited cite was already right.

---

*Provenance.* Three independent candidates (A, B, C) wrote this recommendation to one mandate with no cross-visibility; a fresh hunter that authored none of them judged all three against a stated bar (B1-B3 must-pass, B4-B7 weighted) using a threat-class coverage matrix over the five dangerous classes × both legs (modelled / blocked). **Base: C** — the only candidate passing all three must-pass criteria, and the only gate that fails closed. **Fold-in from B:** Cursor's `state.vscdb` / `cursorAuth/accessToken` exfiltration surface, into the tripwire (T012) and the Cursor surface list. **Fold-in from A:** the per-rule catalogue with effort and origin, and the worked always-on token budget, arithmetic re-verified. All twelve verdict defects applied — including deriving the block set from the catalogue, adding `.claude/settings.json`, and collapsing Pack D onto agnix. Date: **2026-09-12**. Rests on seven merged research documents, all under `cursor-profiler/docs/research/`: `skill-audit-tool/synthesis.md`, `skillspector.md`, `warp-skill-doctor.md`, `otel-genai-alignment.md`, `skill-profiling-state-of-the-art.md`, `existing-per-skill-cost-tools.md`, `cursor-telemetry-surfaces.md`.
