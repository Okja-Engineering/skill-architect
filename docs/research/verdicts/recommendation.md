# Arena verdict — final recommendation (A / B / C)

**Judge:** fresh hunter, authored no candidate. **Date:** 2026-09-12.
**Bar:** `.scuba/teams/recommendation/arena.md` (B1-B3 must-pass; B4-B7 weighted).

**Coverage walked.** 21 criterion-cells (7 × 3); 27/27 SkillSpector analyzer families mapped
against each candidate's threat table (81 mappings); 5 dangerous classes × 3 candidates × 2
legs (modelled / blocked) = 30 determinations; ~20 claim spot-checks per candidate (61 total)
recomputed against `docs/research/**`, `intent.md`, `spec.md`, `questions.md` and
`skill-architect/{PRINCIPLES,README,AGENTS,.out-of-scope,CHANGELOG}.md`,
`skills/skill-audit/SKILL.md`, `.scuba/roadmap.md`, both plugin manifests; 8 mandate-named
harness surfaces × 3 candidates = 24 gate/packaging checks; agnix independently enumerated
(455/455 rules from `crates/agnix-rules/rules.json` v1.1.0, shallow clone of
`agent-sh/agnix` @ 2026-09-11) with CUR-*, MCP-*, AGM-*, CC-SET-* inspected by hand.
Two full sweeps; the second added items 6, 8 and 12 of §6 and nothing else.

---

## 1. Per-candidate scorecard

| # | Criterion | A | B | C |
|---|---|---|---|---|
| **B1** | Safety-first: threat model complete vs SkillSpector; gate **blocks** the dangerous classes before a skill runs | **FAIL** | **FAIL** | **PASS** |
| **B2** | Traceability: every claim traces; nothing invented; numbers reconcile | **FAIL** | **PASS** | **PASS** |
| **B3** | Claude Code and Cursor each addressed concretely at the gate and in packaging | PASS (thin) | PASS | PASS |
| **B4** | Integrate-vs-reimplement decided, licensed, reasoned; no gold-plating | Good | Weakest | **Best** |
| **B5** | First three slices independent; slice 1 is the safety gate | Good | Weakest | **Best** |
| **B6** | Team lead in one sitting; ≤ 7 pages; tables | ~8.8 pp | ~8.9 pp | ~8.9 pp |
| **B7** | Candour: "don't build" list and user decisions are real | Good | Better | **Best** |

### B1 — evidence

**All three model the catalogue completely.** A rows 1-27 + SSD folded into row 1; B rows 1-17;
C rows 1-28. Each accounts for all 27 SkillSpector analyzers and all 113 IDs, and each also
covers the mandate's own quality half (silent failure, negative lift, context bloat, cost).
**The failures are all on the second leg — does the gate block.**

- **A — FAIL, two items.**
  (i) *Over-broad tools and permissions never blocks.* `A/recommendation.md:91`: "Block = the
  dangerous classes (1, 4, 5, 6, 12, 13, 14, 17, 18, 19, 20, 21, 24)" — class 8 (excessive
  agency / unrestricted tools, EA1-5) is absent, and the same line demotes it: "Hotspot-style
  review states (class 8, 9, 20 at MEDIUM) are review-required, not auto-fail". Its only
  implementing rule, `SK-H004` at `A:171`, is severity **Medium**. REAL.
  (ii) *The blocking floor is unbacked by its own catalogue.* `A:82` has stage S2, "Deterministic
  floor (Go, zero-dep)", **BLOCK** on classes 1, 4, 5, 6, 12, 13, 14, 19, 20. A's static
  catalogue `A:157-162` ships exactly five Go security rules — `SK-S001` (pipe-to-shell),
  `SK-S002` (credential literal), `SK-S028` (`.cursor/hooks.json`), `SK-S029` (`.cursor/` reads),
  `SK-S030` (unpinned) — and delegates `SK-S003..S027` to the SkillSpector shell-out (`A:159`).
  Classes 1, 5, 12, 13 and 19 have **no Go rule** yet are named as blocked at a stage that runs
  with zero dependencies. REAL.

- **B — FAIL, one item, structural: the gate fails open.** B's Go core (stage 2, `B:64`) checks
  "frontmatter validity per harness, structure, dead refs, cycles, ICM placement, always-on cost,
  trigger collision" — **zero security rules**; it blocks only on "Blocker/High **reliability** or
  ICM L0 violation". Every dangerous class is delegated to stage 3, the SkillSpector shell-out
  (`B:65`), which is optional and degrades: "a missing external tool produces `checks_skipped[]`,
  never `exit 1`" (`B:71`). Nothing caps the verdict when it is skipped — B's F12 (`B:182`) caps
  only on an **incomplete inspection ledger**, which counts artifacts read, not checks skipped.
  On the stated default environment (Python 3.9.6, no `uv`, SkillSpector not on PyPI —
  `skillspector.md:90,104`) **B's gate blocks on none of the five dangerous classes and can still
  emit APPROVE.** REAL. Secondary: `B:65` column (a) says the security scan is "run, warn-only"
  while the Blocks? column on the same row says "block on any CRITICAL" — contradiction.

- **C — PASS.** The G2 tripwire (`C:83`, ~15 Go rules, <300 ms, "must work with **no Python
  present**") blocks zero-width/bidi (P1-3), credential literals, `curl|sh` (SC2-3), `.claude`/
  `.cursor` config reads (AS1-2), egress + env harvest (E2/E4), path escape, self-modification
  (RA1), homoglyph description (TP2) — enumerated again as Pack B at `C:156`. G3 absent ⇒
  "`checks_skipped[]`, verdict **capped at CAUTION, never APPROVE**" (`C:84`), and F13 (`C:197`)
  states the invariant once: "No verdict above CAUTION when the ledger is incomplete — any
  uninspected artifact (AE1) **or skipped check** caps the verdict". C is the only candidate whose
  refusal covers skipped checks, and therefore the only one that fails closed.
  *Named gaps, not failures:* EA1-5 and BH1-3 are absent from the 15-rule tripwire, so with no
  Python they produce a CAUTION ceiling rather than a block (§6 item 1).

### B2 — evidence (61 spot-checks; every recomputed number reconciles unless noted)

- **A — FAIL, one item, load-bearing and invented.** `A:100`: "Pre-activation enforcement — a
  Claude Code hook invoking `skillaudit gate --skill <dir>` **before the Skill tool resolves**;
  S2 only (<2 s)". **No input supports a Claude Code hook that can block skill activation.** The
  only two inputs touching it say the opposite: `otel-genai-alignment.md:194` (gap G9, issue #320
  "Agent Harness Hook Semantic Conventions" — *no PR*, no blocking semantics) and
  `skill-profiling-state-of-the-art.md:127`, which names the actual documented mitigations
  (`disable-model-invocation: true`, `skillOverrides`, `/skill-doctor`). B flags the same thing
  as "**Unverified**" (`B:80`); C states "*No evidence of a Claude Code hook that can block skill
  activation; we do not design one*" (`C:106`). This is A's only Claude Code pre-activation
  mechanism, so it is not a stray sentence. REAL.
  A's arithmetic is otherwise sound — the worked always-on budget at `A:185` reconciles exactly
  against `skill-profiling-state-of-the-art.md:145-151` (262 chars → ~65 tok; 441 → ~110; 175
  total; 0.01 × ~200K = 2,000; 175/2,000 = 8.75%).
- **B — PASS, strongest traceability in the set.** Verified exact: ACES 0.2134, 95% CI
  [0.1967, 0.2301], N=947, 27.2% negative (`L2-dynamic-report.md:23,102,274`); MDE N≈12/54/216
  (`L2:125`); 1,536 **characters** with `SYN§7a` correcting `L1§4` (`synthesis.md:248`);
  `.claude-plugin/plugin.json` `"skills": "./skills/"` (verified in the file); `preCompact.
  context_tokens` "the only real Cursor-reported token number" (`cursor-telemetry-surfaces.md:56`);
  `cursor.hook.outcome: success|blocked|failed|timeout` (`:114`); the `SYN§6` landed-work list
  (`synthesis.md:235`); $6 at ½`C_f` (`synthesis.md:124`); SDY ≈ $232 (`L3:72`). Best single
  original synthesis in the set: `B:32` traces `state.vscdb` → `cursorAuth/accessToken` to
  `existing-per-skill-cost-tools.md:97` (F-08, `SELECT value FROM ItemTable WHERE key =
  'cursorAuth/accessToken'`) and reframes a cost-research artifact as an agent-snooping target.
  Only defect: `B:25` marks agnix ◐ on prompt injection — agnix has no injection rules (§6 item 6).
- **C — PASS, two trivial slips.** Verified exact: `description` ≤1024 / `name` 64 /
  `compatibility` 500 (`L1-static-report.md:63`); `skillListingBudgetFraction` 0.01 **at v2.1.129**
  (`SOTA:133`); `preCompact.context_window_size` as the Cursor budget denominator
  (`cursor-telemetry-surfaces.md:51`); `disable-model-invocation` as the documented CC mitigation
  (`SOTA:127`); `.out-of-scope.md:10` quoted verbatim; both manifests at **v0.4.0** (verified);
  D-1/D-2/D-3/R1/R2 landed, D-6 + `claude_code.go:83` open (`synthesis.md:235`); $101 → $6
  (`synthesis.md:124`); SkillSpector's README publishing wrong severities for **4** AST rules
  (`skillspector.md:77`); the `skillscore` 7-dim-npm vs 6-dim-Dart conflict (`L1:135`) — which
  **only C** flags. Uniquely verified: `C:117`'s "SP1 · beforeReadFile gating spike as a *hard
  gate*" is real — `skill-architect/.scuba/roadmap.md:32,59` ("Gating spike — hard gate ahead of
  S3"). No other candidate found it. Slips: `C:27` says "26 rule families" against its own 28
  SS-derived rows and SkillSpector's 27 analyzers / 18 categories; `C:122` cites
  `SA:README.md:63` for the Cursor install row, which is line 64.

### B3 — evidence (8 mandate-named surfaces × 3)

| Surface | A | B | C |
|---|---|---|---|
| Cursor `.cursor/rules/*.mdc` frontmatter (`description`/`globs`/`alwaysApply`) | ✔ `:108` | ✔ `:88` | ✔ `:119` |
| Cursor `.cursor/mcp.json` | ✔ `:108` | ✔ `:88` | ✔ `:119` |
| Cursor `hooks.json` | ✔ `:108,110` | ✔ `:88` | ✔ `:117,119` |
| Cursor `AGENTS.md` | ✔ `:108` | ✔ `:88` | ✔ `:119` |
| Claude Code `.claude/settings` | **absent** | **absent** | **absent** |
| Claude Code `hooks/hooks.json` | ~ "plugin bundled hooks" `:97` | ~ "hooks" `:79` | ~ "hook definitions" `:159` |
| Claude Code plugin manifest | ✔ `:102` | ✔ `:81` exact | ✔ `:110` exact, v0.4.0 |
| Claude Code OTel `skill.name` | ✔ `:14,221` | ✔ `:82` | ✔ `:109` |

All three pass: Cursor is addressed concretely and separately by all three (4/4 surfaces each),
and Claude Code on 3 of 4. **`.claude/settings.json` is absent from all three** — a shared root,
not a differentiator: each enumerated the Cursor surface from `skillspector.md:131` (§7 #3, which
lists the Cursor paths) but enumerated the Claude Code surface from SkillSpector's *analyzer*
list, which is skill-bundle-shaped, so nobody walked Claude Code's **config** surface. See §6
item 3. Packaging: B (`:168`) and C (`:184`) each carry a dedicated packaging paragraph naming
both manifests and applying ICM to the tool itself; A's is two distributed one-line table rows
(`:102`, `:114`) — thinnest, still concrete. A is also the only candidate that declares a
five-tool coverage key (`A:23`) without populating five columns per row; B (`:23-50`) and C
(`:30-69`) both satisfy mandate §2's column requirement in full.

### B4 — integrate-vs-reimplement, install tax, licensing, and C's tripwire

All three reach the same correct call — **integrate: shell out to an unmodified binary, never
vendor, never hard-depend** — and all three state the licensing correctly: Apache-2.0 against
skill-architect's MIT, vendoring would need LICENSE + NOTICE + change notices plus a permanent
Python sync burden, shelling out to an unmodified binary creates no obligation
(`skillspector.md:143`, verified). All three name the install tax honestly: not on PyPI (404),
git-URL install, Python ≥3.12 <3.15, ~20 deps, native `yara-python`, Docker built locally, dev
machine on 3.9.6 (`skillspector.md:90`). **The decision is a wash; what separates them is what
happens when the binary is absent** — see B1.

**Is C's capped Go tripwire sound or gold-plating? Sound — and it is the opposite of
gold-plating.** It is bounded by construction ("**Caps at 15 rules.** No YARA, no OSV, no AST, no
taint — those are SkillSpector-only and the report says so", `C:134`); it is justified by
constraints the inputs establish rather than by ambition (`AGENTS.md:6` bans Python — verified,
and the actual text bans it "for skill scripts", supporting all three readings; `questions.md`
Q8 "No Semgrep, no Python"; dev machine at 3.9.6); and it is the mechanism that *resolves* the
standing `L1§4`-says-shell-out vs `Q8`-says-no-Python conflict rather than deferring it to a
ratification request, which is all A and B do. It is also what makes C's fail-closed posture
affordable: without a zero-dependency floor, "block" and "opt-in Python" cannot both be true.
One real defect in the reasoning, not the design: the tripwire mirrors SkillSpector's P/E/SC/AS/
PE/RA/TP/AE semantics and therefore inherits its false-positive profile (~34% of its 61 open
issues, `skillspector.md:101`), but C's named go/no-go spike measures **SkillSpector's** FP rate,
not the tripwire's (§6 item 8). A's Go floor has the same instinct ("the Go floor exists so the
gate is never toothless", `A:138`) but is not capped, not enumerated, and — per B1(ii) — not
backed by its own catalogue. B has no floor at all.

### B5 — first three slices

| | Slice 1 = safety gate? | Slices 1-3 independent? | PRs map to slices? |
|---|---|---|---|
| **A** | ✔ `:217` `skillaudit gate` | ✔ independence argued per slice, and the arguments hold ("slice 1 already gates without it"; slice 3 "emits warnings only, so it cannot break slice 1's gate") | ✔ 1:1 |
| **B** | ✔ `:193` | ◐ slice 2 is **"Gated on: agnix diff (D4)"** (`:194`) | ✘ **PRs ≠ slices**: PR1 (`:206`) is a `skill-audit` hardening/degradation change, PR2 is slice 1, PR3 is slice 2 |
| **C** | ✔ `:205` "**1 · THE SAFETY GATE**" | ◐ slice 2 gated on the same agnix diff (`:206`) | ✔ 1:1, with an explicit disjointness claim: "Each touches disjoint code paths and disjoint report sections; none depends on another's output" (`:222`) |

A's independence reasoning is the best-argued, but its slice 1 ships only the five Go security
rules and therefore does not deliver the gate A's own §3 specifies (B1(ii)). B's slice/PR
mismatch means the document answers "the first three PRs" with a different list than "the first
three slices", and its PR1 is not the gate. C is cleanest. The shared agnix gate on slice 2 is a
*spike*, not a code dependency — and my agnix enumeration confirms it was the right call to gate
it (§7).

### B6 — readability

Effectively a tie and a uniform miss. A 5,750 words / 177 table rows / 53 prose lines; B 5,776 /
147 / 61; C 5,786 / 157 / 53. At ~650 words per dense page all three land at **~8.8-8.9 pages
against a ≤ 7-page bar**. A has the highest table-to-prose density; B the most prose. No
candidate is separable here; all need the same cut (§6 item 9).

### B7 — candour

All three pass. A: 13 don't-build rows, 9 conflict picks, D1-D8, 7 risks. B: 9 don't-build rows,
D1-D8 each carrying an explicit **recommendation** ("Recommend: narrow it" / "keep" / "block on
CRITICAL only"), 7 conflict picks, 6 risks with evidence and mitigation. C: 13 don't-build rows,
D1-D8 each with a **"Default if you say nothing"** column — the only candidate whose decisions
carry defaults, which is what makes a decision list actionable rather than a stall — plus a
standalone "Thin evidence, stated plainly" paragraph (`C:259`) naming seven gaps including one
nobody else surfaces (that `effortMinutes` → `sqale_index` is docs-assistant asserted, not
primary-page verified). C > B > A.

---

## 2. Coverage matrix — dangerous threat classes vs each candidate's gate

`BLOCK` = the gate refuses before the skill runs, with a rule that exists in that candidate's own
catalogue · `BLOCK*` = blocks only when the optional SkillSpector binary is present · `WARN` =
detected, never blocks · `ABSENT` = not in the gate's block/warn decision at all.
"Fails open / closed" = behaviour when SkillSpector is absent, which is the stated default
environment (`skillspector.md:90,104`).

| Dangerous class | SS IDs | A | B | C |
|---|---|---|---|---|
| **Prompt injection** (override, zero-width, steering, padding) | P1-4, P9 | `BLOCK*` — named at S2 `:82` but **no Go rule** in `:157-162`; real blocking is S3 | `BLOCK*` — stage 3 only `:65` | **`BLOCK`** — G2 tripwire, zero-width/bidi + homoglyph `:83,156` |
| **Data exfiltration** (transmit, env harvest, FS enum, context leak, cloud upload) | E1-5 | `BLOCK*` — same defect; `SK-S002` covers credential literals only | `BLOCK*` — stage 3 only | **`BLOCK`** — egress + env harvest in tripwire `:83`, E2/E4 `:156` |
| **Supply chain** (unpinned, `curl\|bash`, obfuscation, OSV, typosquat) | SC1-9 | **`BLOCK`** — `SK-S001` pipe-to-shell + `SK-S030` unpinned provenance `:157,162` | `BLOCK*` — stage 3; provenance hash at stage 1 `:63` | **`BLOCK`** — `curl\|sh` + SC2-3 + G0 provenance pin `:81,83,156` |
| **Over-broad tools and permissions** (excessive agency, MCP least-privilege) | EA1-5, LP1-4 | **`WARN`** — class 8 omitted from the block list `:91`; `SK-H004` is **Medium** `:171` | `BLOCK*` — stage 3 only; row 7 `:31` adds observed-vs-declared but no block | `BLOCK*` — row 9/23 `:40,54`; **not in the 15-rule tripwire** |
| **Bundled-script execution surface** (hooks that execute bundled content) | BH1-3 | **`BLOCK`** — `SK-S028` `.cursor/hooks.json` `:160`, class 20 in the block list `:91` | `BLOCK*` — row 9 `:33` ports BH to `.cursor/hooks.json`, delivered in slice 2 | `BLOCK*` — row 21 `:52` ports BH, delivered in Pack D / slice 2; **not in the tripwire** |
| *Supporting guard:* uninspected artifact / incomplete ledger | AE1 | **`BLOCK`** `:81`, F12 `:206` | **`BLOCK`** `:66`, F12 `:182` | **`BLOCK`** `:82,97`, F13 `:197` |
| *Supporting guard:* **skipped check caps the verdict** | — | **ABSENT** — F12 covers the ledger only `:206` | **ABSENT** — F12 covers the ledger only `:182` | **PRESENT** — F13 covers "or skipped check" `:197`; G3 absent ⇒ CAUTION ceiling `:84` |
| **Net posture with SkillSpector absent** | — | 3 of 5 blocked; **over-claims 5 more** | **0 of 5 blocked — fails open, APPROVE still reachable** | 3 of 5 blocked, other 2 capped at CAUTION — **fails closed** |

---

## 3. Ranking

**1st — C.** The only candidate that passes all three must-pass criteria. **2nd — B.** One
must-pass failure (B1, fails open), the strongest traceability in the set, and the best-populated
tool-coverage matrix. **3rd — A.** Two must-pass failures (B1 over-broad tools never blocks and a
blocking floor its own catalogue cannot back; B2 an invented Claude Code pre-activation hook),
against the best static-rule catalogue and the only worked always-on budget that reconciles.

A is third on the must-pass arithmetic, not on quality of thought: it is the strongest document
on the addendum's "enumerate the full static catalogue" instruction and the only one to
demonstrate the headline metric with arithmetic. But a fabricated platform capability is the most
expensive defect class here, because it is the kind that gets built.

---

## 4. Recommended base — **C** (`candidates/C/recommendation.md`)

Soundest *shape*, on four counts that are structural rather than stylistic:

1. **It is the only gate that fails closed.** The G2 tripwire + the CAUTION ceiling + F13's
   "or skipped check" clause form one invariant stated once and enforced at every stage. A's floor
   is over-claimed; B's is absent.
2. **It refuses to invent a platform capability.** Where the evidence runs out it says so and
   substitutes the documented mitigation (`disable-model-invocation: true`, `SOTA:127`) — and it
   is the only candidate to find `SP1 · beforeReadFile gating spike` as an existing *hard gate* in
   skill-architect's own roadmap rather than asserting a Cursor gating mechanism as settled.
3. **Its roadmap is coherent.** Slices 1-3 map 1:1 to PRs 1-3 with an explicit disjointness claim;
   slice 1 is labelled and scoped as the safety gate and actually ships one.
4. **Its decision list carries defaults.** D1-D8 with "Default if you say nothing" plus a
   standalone thin-evidence paragraph is the form that lets the user ratify by exception.

---

## 5. Fold-ins — one strongest idea per runner-up

**From B → fold into C §3a (G2 tripwire) and §4b (Pack B / Pack D).**
`B/recommendation.md:32`: `state.vscdb` holds `cursorAuth/accessToken` — "a real credential
store" — traced to `existing-per-skill-cost-tools.md:97` (F-08, `SELECT value FROM ItemTable
WHERE key = 'cursorAuth/accessToken'`). This converts a cost-research artifact into a concrete
Cursor credential-store path and is the single best original synthesis move in the arena. C's
row 15 (`C:46`) says only "**Add `.cursor/` paths**"; B names the file that actually holds the
token. Fold the path into C's tripwire agent-snooping rule (AS1-2 leg) and into Pack D's Cursor
surface list. *Secondary, if a second fold is affordable:* B's five-column tool-coverage matrix
(`B:23-50`, SS | SA | AGX | WARP | ASD populated for all 26 rows) is the cleanest satisfaction of
mandate §2's column requirement; C's five-symbol single column is equivalent in content but
harder to scan.

**From A → fold into C §4b (Pack E) and the catalogue tables.**
`A/recommendation.md:155-185`: the per-rule static catalogue — 25 rule IDs, each with
`{softwareQuality, severity}`, an effort-minutes estimate and a named origin (new / borrowed +
source) — **plus the worked always-on budget at `A:185`**, which is the only place in the arena
where the headline new metric is *demonstrated* rather than asserted: 262 chars → ~65 tokens,
441 → ~110, estate total 175, budget 0.01 × ~200K = 2,000, therefore 8.75% consumed by two
skills, every input cited and every step reproducible. C's Pack E (`C:159`) specifies the metric
correctly but never runs it. Fold A's worked example into Pack E and A's per-rule ID/origin/effort
granularity into C's pack tables, keeping A's own honesty label ("effort minutes are illustrative
placeholders until calibrated", `A:183`).

---

## 6. Defects the merged document must fix

Ordered by severity. Items 1, 2, 5 and 11 share one root: **the blocking set was written
separately from the rule catalogue that has to implement it**, so each document names classes it
blocks without a rule behind them, or ships rules that never reach the block set. One root fix —
derive the block list *from* the catalogue, one rule per blocked class, and state the fail-closed
invariant once in the measurement contract — closes all four.

| # | Defect | Where | Severity | Class |
|---|---|---|---|---|
| 1 | **Over-broad tools/permissions (EA1-5, LP1-4) and the bundled-hook execution surface (BH1-3) must block from the zero-dependency floor, not only via the optional shell-out.** C's 15-rule tripwire omits both; A demotes EA to Medium/review-required; B has no floor. These are two of the five dangerous classes the bar names. | `C:83,156`; `A:91,171`; `B:31,33` | **High** | B1 |
| 2 | **Delete the Claude Code pre-activation hook.** No input supports a CC hook that blocks skill activation; `otel-genai-alignment.md:194` (G9, issue #320, no PR) and `SOTA:127` say the opposite. Use `disable-model-invocation: true` + pre-install/CI as the CC enforcement point, and say plainly that CC has no blocking pre-activation surface today. | `A:100` | **High** | B2 |
| 3 | **Add `.claude/settings.json` to the Claude Code gate surface** — absent from all three. It carries `permissions.allow`/`deny`, MCP server enablement and hook registration: it is Claude Code's over-broad-permissions surface, and agnix ships **30 CC-SET-\*** rules against it. Name `hooks/hooks.json` by path rather than "bundled hooks" / "hooks" / "hook definitions". | all three | **High** | B3 |
| 4 | **Shrink the per-harness Cursor frontmatter pack to a shell-out to agnix** and re-aim the slice at what agnix does not do. See §7 — the 20 CUR-\* rules already cover the `.mdc` surface all three claim as new. All three hedged correctly ("verify against agnix first"); the merged doc must resolve the hedge, not carry it. | `A:170` `SK-H003`; `B:139`; `C:158` Pack D | **High** | B4/B5 |
| 5 | **State the fail-closed invariant once and propagate it.** Adopt C's F13 wording — a skipped check, not only an uninspected artifact, caps the verdict — and delete B's stage-3 contradiction, where column (a) says "run, warn-only" and the Blocks? column on the same row says "block on any CRITICAL". | `C:197` (adopt); `B:65` (delete) | **High** | B1 |
| 6 | **Correct the agnix marks.** agnix has **no** prompt-injection detection: `CC-SK-009 "Too Many Injections"` is an authoring rule about the count of `` !`cmd` `` injections in a SKILL.md, not a security rule. B row 1 `AGX ◐` and C row 1 `AG ◐` → `○`. Conversely, agnix's `.cursor/mcp.json` security coverage is **understated**: MCP-005 (tool without user consent), MCP-018 (plaintext secret in env), MCP-019 (dangerous stdio command, bad-example `curl …\| sh`), MCP-021 (wildcard HTTP binding) all run against `.cursor/mcp.json`. | `B:25`; `C:32`; `A:48` | Medium | B2 |
| 7 | **Reconcile C's family count.** `C:27` says "all **26** rule families / 113 IDs" against its own 28 SS-derived rows and against `skillspector.md:75`'s 27 analyzers / 18 categories. Pick one denominator and use it everywhere. | `C:27` | Medium | B2 |
| 8 | **Calibrate the tripwire's own false-positive rate, not only SkillSpector's.** The named go/no-go spike scans skill-architect's skills with SkillSpector; the ~15 Go rules mirror SkillSpector's P/E/SC/AS/PE/RA/TP/AE semantics and inherit its FP profile (~34% of 61 open issues, `skillspector.md:101`). Every tripwire rule needs both a fires-on-real-input test **and** a does-not-fire-on-the-clean-corpus test — otherwise the house rule against dead rules (`skillspector.md:144`) is only half-enforced. | `C:83,134,259` | Medium | B4 |
| 9 | **Cut to ≤ 7 pages.** All three land at ~8.8-8.9. The cheapest cut is the ceded rows: C rows 3, 4, 6, 10, 12-14, 16-20, 22, 24, 26 all read "— (cede)" and collapse into one row, as B already does at its rows 13-14. | all three | Medium | B6 |
| 10 | **Fix the inherited line cite for `skill-audit`'s hard exit.** All three cite `SKILL.md:46-47`; the `command -v … \|\| exit 1` pair is at `:45-46` (`L1-static-report.md:80,113` says `:44-45`). Inherited from `skillspector.md:114`. | `A:127,261`; `B:71`; `C:138` | Low | B2 |
| 11 | **Every blocked class needs an implementing rule in the shipped catalogue.** A's S2 blocks 9 classes; A's catalogue ships 5 Go security rules. Whatever floor the merged doc adopts, the block list must be derived from the catalogue rather than written beside it. | `A:82` vs `A:157-162` | **High** | B1 |
| 12 | **`SA:README.md:63` → `:64`** — the Cursor plugin-install row. | `C:122` | Low | B2 |

---

## 7. The agnix Cursor-coverage answer

**Question:** B flags as "the largest unexamined risk in this plan" (`B:238`, D4) that agnix may
already cover Cursor rules with `CUR-`-prefixed checks, which would invalidate the Cursor
differentiation all three candidates assert. **Answer: the risk is real and the flag is
confirmed.** The Cursor *spec-conformance* lane is already built; the Cursor *security, budget,
ICM and graph* lanes are not.

**Method.** `L1-static-report.md:126` records agnix at 455 rules with `CC-*`/`CUR-*`/`MCP-*` IDs,
MIT + Apache-2.0, 410★, pushed 2026-09-11 — but never enumerates `CUR-*`. I resolved it directly:
`gh api repos/agent-sh/agnix` (reachable; 410★, Apache-2.0, pushed 2026-09-11T19:46Z), shallow
clone, and enumeration of `crates/agnix-rules/rules.json` (`total_rules: 455`, v1.1.0, updated
2026-09-05). **The 455 figure is confirmed exactly.**

**What agnix already covers — 20 `CUR-*` rules:**

| Surface | agnix rules | Verdict |
|---|---|---|
| `.cursor/rules/*.mdc` frontmatter | CUR-002 missing frontmatter · CUR-003 invalid YAML · **CUR-004 invalid glob in `globs`** · **CUR-005 unknown frontmatter keys** · **CUR-007 `alwaysApply` with redundant globs** · **CUR-008 invalid `alwaysApply` type** · **CUR-009 missing `description` for agent-requested rule** · CUR-001 empty rule file · CUR-006 legacy `.cursorrules` · **CUR-020 ignored plain-markdown rule** | **Already built.** This is precisely the `description`/`globs`/`alwaysApply` surface all three candidates claim as new. CUR-005 is the *invalid* leg and CUR-020 is a *silently-ignored* leg of the proposed tri-state. |
| `.cursor/hooks.json` | CUR-010 invalid hooks schema · CUR-011 unknown hook event name · CUR-012 hook entry missing `command` · CUR-013 invalid hook type · CUR-017 invalid hook entry field types · CUR-018/019 prompt-hook fields | **Schema validity already built.** Execution-surface *security* (the BH1-3 port) is **not** — agnix validates the shape of a hook, never what the hook executes. |
| `.cursor/mcp.json` | 26 `MCP-*` rules; `crates/agnix-core/src/rules/mcp.rs:3031+` calls `validate_path(".cursor/mcp.json", …)` explicitly. Includes MCP-005 tool without user consent, MCP-018 plaintext secret in env, MCP-019 dangerous stdio command (`curl … \| sh`), MCP-021 wildcard HTTP binding | **Already built, including partial least-privilege/security.** Understated by all three candidates. |
| `AGENTS.md` | AGM-001..006 (markdown structure, section headers, character limit, project context, platform-specific features without a guard, nested hierarchy) | **Already built** for structure. |
| Cursor per-client skills | CR-SK-001 "Cursor Skill Uses Unsupported Field" | Already built, minimally. |
| *(bonus, relevant to defect 3)* Claude Code `.claude/settings.json` | **CC-SET-\*, 30 rules** | Already built — and omitted by all three candidates. |

**What agnix does not do, so the differentiation survives here:**

- **No project-level always-on token budget with per-item attribution.** The nearest rule is
  `CC-MEM-009 "Token Count Exceeded"`, a per-file cap on a memory file. Nothing sums descriptions
  + `CLAUDE.md`/`AGENTS.md` + `alwaysApply` rules + hooks + MCP manifests into one budget.
  (Adjacent size rules only: AGM-003, AS-015, CDX-AG-004, WS-002.)
- **No ICM / progressive-disclosure enforcement** — no L0-L4 placement, deferral ratio or
  reference-depth rule anywhere in the 455.
- **No cross-skill reference graph or cycle detection.**
- **No security threat detection on any surface** — zero injection, exfiltration, taint, YARA or
  supply-chain-of-the-skill rules. agnix is a *linter*, not a scanner.
- **No remediation-cost economics** — no effort/minutes field on any rule.

**Consequence for the merged plan.** The agnix diff should be treated as **settled, not spiked**:
slice 2's per-harness Cursor frontmatter pack (A `SK-H003`, B's frontmatter-validity row, C's
Pack D) collapses to a shell-out to `agnix --format …` plus the tri-state *per harness version*
verdict that agnix does not attempt, and the slice re-aims at Pack E (always-on budget), Pack F
(ICM), Pack G (graph) and the Cursor **security** port (BH1-3 to `.cursor/hooks.json`, AS1-3 to
`.cursor/` paths and `state.vscdb`) — none of which agnix touches. B, C and A all hedged this
correctly (`B:139,238`; `C:136,158,252`; `A:61,274` D5); the merged document should close the
hedge with the enumeration above rather than carry a one-day spike into the roadmap. Note also
that this **strengthens** the case for C's tripwire and for the security lane generally: the
crowded lane is conformance, and neither agnix nor SkillSpector covers Cursor security.

