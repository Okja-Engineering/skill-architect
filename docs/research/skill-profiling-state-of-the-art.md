# Skill profiling — state of the art (Sept 2026)

Collected 2026-09-10 from `skill-architect/.scuba/teams/research-profiling/ledger.md` Parts A and B.
Evidence labels per `AGENTS.md`.

**Companion file:** `skill-evaluation-research.md` already covers ACES, SkillReducer, SkillEval, and
agent-profiler at a summary level. This file does not repeat those; it adds the primary-source
figures, confidence intervals, first-party tooling, and the token-effect evidence that were missing,
and flags where the two disagree.

---

## 1. The field converged on paired (with-skill / without-skill) evaluation

**External evidence — SkillsBench** (arXiv:2602.12670), the anchor benchmark. Abstract figures:
87 tasks across 8 domains, 18 model-harness configurations, deterministic verifiers.

| Arm | Pass rate |
|---|---|
| Without Skills | **33.9%** |
| With curated Skills | **50.5%** |
| Delta | **+16.6 pp** (25.5% normalized gain) |

Configuration-level gains **+4.1 to +25.7 pp**. **Self-generated Skills give no benefit on average.**
"Focused Skills with at most three modules outperform larger or exhaustive bundles, and smaller
models with Skills can match larger models without them."
<https://arxiv.org/abs/2602.12670>

> **Conflicting figures — flagged, unresolved.** Secondary summaries report "86 tasks across 11
> domains", "7 agent-model configurations over 7,308 trajectories", and "+16.2 pp". The arXiv
> abstract (primary) says 87 / 8 / 18 / +16.6 pp. **Use the abstract numbers; treat trajectory counts
> as UNKNOWN** until the PDF body is read. Secondary:
> <https://www.alphaxiv.org/overview/2602.12670> · <https://evaluatingevals.substack.com/p/skillsbench-review>

**External evidence — NVIDIA SkillEvaluator** (open source) operationalizes this as a 3-tier
framework; Tier 3 is the reference implementation of the paired protocol.

| Tier | What it does |
|---|---|
| 1 — safety & structure | schema, prompt-injection/exfiltration scan, secret/PII detection, license, script lint |
| 2 — distinctiveness | embedding similarity, intra-skill duplication, cross-catalog overlap |
| 3 — live evaluation | **two isolated sandbox runs per case**, identical prompt/model/inputs/grading; the only variable is skill presence |

**Skill Lift** = with-skill score minus without-skill score, in points. Five scoring dimensions:
Correctness, Discoverability (activation), Effectiveness, Efficiency (steps), Security. Repetitions:
85% of skills single attempt, 15% two attempts. Supported harnesses in the blog: **Claude Code and
OpenAI Codex — not Cursor.** Benchmark 2026-08-12: average Skill Lift **+31 points** across
dimensions (**+39** excluding Security); Correctness **46 → 87 (+41)**.
<https://developer.nvidia.com/blog/evaluating-ai-agent-skill-performance-with-nvidia-skillevaluator/> ·
<https://github.com/NVIDIA/SkillEvaluator> · <https://docs.nvidia.com/skills/skillevaluator>

**External evidence — ACES** (arXiv:2608.20614), the methodology paper behind it. Numbers the
existing `skill-evaluation-research.md` reports without the interval:

| Statistic | Value |
|---|---|
| Skills analyzed | 145 real skills |
| Scored paired cases | **947**, from 58 production skills across four primary harnesses |
| Mean composite Skill Lift | **0.2134, 95% CI [0.1967, 0.2301]** |
| Mean outcome-only lift | 0.1799 |
| **Positive lift rate** | **72.8%** — i.e. **27.2% of skills hurt** |

<https://arxiv.org/abs/2608.20614>

**Design decision candidate.** A comparison engine should adopt **Skill Lift** as the headline
statistic and **report a confidence interval, not a point estimate**. ACES's CI over ~950 paired
cases sets the bar: a single paired run is not evidence. The 27.2% negative-lift rate (and
SkillsBench's 16/84 negative-delta tasks) means the engine must be able to report **"this skill
hurt."**

---

## 2. Anthropic shipped in-harness skill profiling: `/skill-doctor`

**Specification** (Claude Code docs, <https://code.claude.com/docs/en/skills>):

- Reports, per skill in the session: **context cost** and **how often it was invoked**; flags listing
  entries never invoked and says where to turn them off.
- Scope: session skills **other than bundled and enterprise skills**.
- Requires **Claude Code v2.1.252 or later**. Not available in sessions that skip feature-flag
  fetching. Not available over Remote Control (`Skill usage reports are not available on this
  connection.`).
- Output: interactive → `/plugin` manager **Stats** tab; non-interactive (`-p`) → **plain text**.

> **Conflict — flagged.** Third-party reporting says it shipped in **2.1.261 on 2026-09-04**
> (<https://www.implicator.ai/anthropic-claude-code-skill-doctor-context-audit/>); official docs say
> **≥ v2.1.252**. Both can be true under a feature-flagged rollout. Exact first version UNKNOWN.

**Repository fact.** The research machine runs `claude 2.1.221` — `/skill-doctor` is **not available
there**. Any design consuming it must gate on version.

**External evidence (limitation).** It records *whether* a skill ran, not whether it *improved*
results — a rarely-used-but-valuable skill is indistinguishable from dead weight. It is a **cost**
instrument, not an **efficacy** instrument. That is exactly the gap paired evaluation fills.

**Related — `claude plugin eval`.** Anthropic's `skill-creator` plugin ships an eval loop: test cases
in `evals/evals.json`, each run as an isolated spawned subagent, assertion-based grading, HTML review
viewer; `claude plugin eval init --bare regression` scaffolds a skills-directory plugin
(**External evidence**,
<https://github.com/anthropics/claude-plugins-official/blob/main/plugins/skill-creator/skills/skill-creator/SKILL.md>).
**No canonical CLI reference page was reached** — subcommands, flags, and output schema are
unverified. Note it is *assertion* grading, not *paired* grading: it answers "did the skill produce
the right output", not "did the skill help versus not having it".

---

## 3. Claude Code skill token budgets — the most precise public accounting

**Specification** — <https://code.claude.com/docs/en/skills>:

- Three-level progressive disclosure: **(1) skill listing — always loaded, every turn**: only
  `description` + `when_to_use`. **(2) full `SKILL.md`** on invocation. **(3) supporting files** only
  when Claude opens them.
- > "Every skill in the skill listing adds to your context on every turn, whether or not Claude ever
  > uses it."
- > "the combined `description` and `when_to_use` text is truncated at **1,536 characters** in the
  > skill listing to reduce context usage."
- > "Unlike CLAUDE.md content, a skill's body loads only when it's used, so long reference material
  > costs almost nothing until you need it."
- Recommendation: keep `SKILL.md` under **500 lines**.
- **Stickiness:** > "the rendered `SKILL.md` content enters the conversation as a single message and
  **stays there across later turns**." A skill invoked once keeps paying every subsequent turn.
- **Compaction budget:** Claude Code re-attaches the most recent invocation of each skill after the
  summary, keeping the **first 5,000 tokens** of each. Re-attached skills share a combined budget of
  **25,000 tokens**, filled from the most recently invoked skill — so older skills can be dropped
  entirely after compaction.
- Mitigations: `disable-model-invocation: true`, `skillOverrides`, `/skill-doctor`.

**External evidence — listing budget knobs:**

| Knob | Default | Notes |
|---|---|---|
| `skillListingBudgetFraction` | **0.01** (1% of context window) | shipped in Claude Code **v2.1.129** |
| `skillListingMaxDescChars` | **1536** | truncates individual descriptions before the overall budget |
| `SLASH_COMMAND_TOOL_CHAR_BUDGET` | — | env var for a fixed character budget |

When the listing overflows, Claude Code **drops descriptions starting with the least-invoked skills**.
**Known bug:** the fraction is computed against a fixed ~200K-token reference rather than the model's
real context window, so on a 1M-context model the effective budget is **~5× smaller than documented**.
<https://github.com/anthropics/claude-code/issues/57941> ·
<https://github.com/anthropics/claude-code/issues/56966> ·
<https://claudefa.st/blog/guide/mechanics/skill-listing-budget>

**Repository fact — `skill-architect`'s own measured numbers (2026-09-10):**

| Skill | SKILL.md bytes | lines | char/4 est | frontmatter `description` chars | est. always-loaded tokens |
|---|---:|---:|---:|---:|---:|
| `skills/skill-audit` | 8,069 | 195 | 2,017 | 262 | ~65 |
| `skills/skill-rewrite` | 7,249 | 185 | 1,812 | 441 | ~110 |

Both are well under the 1,536-char listing cap and the 500-line body guidance.

---

## 4. Static BPE token counting — `skill-validator`

**Repository fact — verified by running the tool.** `skill-validator` (Go,
`/opt/homebrew/bin/skill-validator`) emits token counts **only under `check`, not under
`analyze content`**:

```
skill-validator check skills/skill-audit -o json          → .token_counts.files[] {file, tokens}, .token_counts.total
skill-validator analyze content skills/skill-audit -o json → NO token_counts key
```

Measured on `skills/skill-audit`:

| File | o200k_base tokens |
|---|---:|
| `SKILL.md` body | 1,814 |
| `references/best-practices.md` | 1,135 |
| `references/checklists.md` | 432 |
| `references/evaluation-matrix.md` | 649 |
| **Total** | **4,030** |

**55% of this skill's token mass sits behind progressive disclosure** and is only paid on demand.

**External evidence.** skill-validator counts with **`o200k_base`** encoding — real BPE, not an
estimate. Purely static: no runtime or profiling capability.
<https://github.com/agent-ecosystem/skill-validator>

**Other static tooling (External evidence, less verified):**

- `skillscore` — offline **Dart** CLI (also npm and pub.dev); lints/scores SKILL.md across
  discoverability, conciseness, structure, instruction design, hygiene, safety, with a cited source
  per rule. <https://github.com/sayed3li97/skillscore> · <https://pub.dev/packages/skillscore>
  — **Note:** `skill-architect`'s `AGENTS.md` calls it "npm, 7-dimension"; upstream describes **six**
  dimensions and a Dart implementation. Doc-fix candidate in that repo.
- `@crafter/skillkit` (npm) — claims usage analytics, conflict detection, cost analysis,
  context-budget monitoring; **requires Bun**. **UNVERIFIED** (npmjs.com returned HTTP 403 to
  automated fetch). Bun runtime conflicts with a self-contained-tooling rule.
- `moutons/skills-validator`, `getskillcheck.com`, `vercel-labs/skills` (`npx skills`) — spec
  validation, no runtime profiling.
- **PluginEval** — third-party three-layer framework (static analysis + LLM judge + Monte Carlo)
  producing calibrated scores with CIs. **UNVERIFIED**, not run.
  <https://github.com/wshobson/agents/blob/main/docs/plugin-eval.md>
- **MalSkills** — neuro-symbolic static analyzer for malicious skills.
  <https://github.com/security-pride/MalSkills>

Newer benchmarks worth tracking (abstracts only): **SkillGenBench** (skill *generation* pipelines,
<https://arxiv.org/abs/2605.18693>), **SkillResolve-Bench** (same-capability ambiguity in skill
*retrieval*, <https://arxiv.org/pdf/2606.10388>).

---

## 5. Token effects of skills are bimodal — measure, do not assume

**External evidence — NVIDIA SkillEvaluator Tier 3 token tracking** (2026-08-12 benchmark). Published
per-skill token deltas:

| Skill | Token effect |
|---|---|
| `jetson-optimize-memory` | **−76.9%** |
| `cuopt-install` | **+120.3%** |

**This is the strongest published evidence that a skill's token effect is bimodal and must be
measured per-skill, not assumed.** A skill that shortcuts exploration saves massively; a skill that
adds procedure can more than double token use. **Both can be good skills** — which is why token delta
must be reported alongside correctness, never alone.

**External evidence — SkillReducer** (arXiv:2603.29919, submitted 2026-03-31, revised 2026-06-24).
Beyond what `skill-evaluation-research.md` records: it analyzed **55,315 public skills** and found
**26.4% missing routing descriptions** and **>60% excessive non-actionable content**. Validated on
600 skills + SkillsBench; model retention mean **0.965** across **5 models from 4 families**.
**The abstract does NOT report absolute runtime token or cost reduction — only compression ratios.
Do not cite it for runtime savings.**

**External evidence — SkillJuror** (arXiv:2606.11543, 2026-06-11, 82-task SkillsBench study,
<https://github.com/zhiyuchen-ai/skill-juror>):

- **From the abstract (primary, confirmed):** distinct Skill resources touched per trajectory rise
  **1.18 → 3.85**; effective uptake events rise **1.33 → 3.92**; **+17 verifier-passing trials out of
  410 matched trials (+4.1%)**.
- **From secondary summarization (lower confidence — not confirmed against the PDF body):**
  Progressive Disclosure **46.1% pass** vs **29.0% No Skill**; yield-normalized wall-clock
  **20.1 → 17.8 min per strict pass**; **cost/pass nearly tied, $1.31 vs $1.28**.
- **Well-supported conclusion:** progressive disclosure **raises resource touches ~3×**, converting
  always-loaded cost into on-demand cost, but **does not reduce total cost per successful task**.
  It buys *accuracy*, not *cheapness*.

**Low-confidence / UNVERIFIED — do NOT use.** Blog claims of "60–92% token savings from progressive
disclosure" (dev.to, chudi.dev) and "85× token savings" (matthewkruczek.ai) are reported **without
accuracy control** — they measure tokens *not loaded*, with no check that task success held constant.
One secondary source cites a measured band of 26.8%–43.2% that could not be traced to a primary
paper. **Treat all of these as marketing.** The only accuracy-controlled runtime numbers available
are SkillEvaluator's and SkillJuror's.

---

## 6. OTel GenAI semantic conventions — no skill convention exists

**Specification.**
<https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md>
(semconv **1.44.0**, OTel spec v1.56.0).

- **Status: Development.** All GenAI attributes and span types carry development badges.
  **Not stable — do not treat as a frozen contract.**
- Agent operations: `create_agent`, `invoke_agent`, `execute_tool`.
- Token attributes: `gen_ai.usage.input_tokens`, `.output_tokens`, `.cache_read.input_tokens`,
  `.cache_write.input_tokens`, `.reasoning.output_tokens`.
- **There is NO convention for skills, rules, or prompt-fragment attribution.**

**Consequence + Design decision.** Any skill-level attribute is necessarily vendor-specific. Cursor
invented `cursor.skill.*`; cursorscope invented `cursor.attribution.category`/`.name`/`.detail`/
`.token_source`. **Namespace this project's own skill attributes (e.g. `skill_architect.skill.name`,
`.token_source`) rather than squatting on `gen_ai.*`, and map to `gen_ai.*` only where the convention
actually defines the field. Never put a chars/4 estimate in `gen_ai.usage.*_tokens`.**
**Repository fact:** cursorscope itself pins `@opentelemetry/semantic-conventions@^1.38.0` and imports
agent attributes from the `/incubating` entrypoint — upstream treats these as unstable too.

---

## 7. Techniques for attributing tokens to a skill

| Technique | What it gives | Availability | Verdict |
|---|---|---|---|
| First-party per-skill accounting — `/skill-doctor` | per-skill context cost + invocation count | Claude Code ≥2.1.252 only | best-in-class, wrong harness for a Cursor tool |
| First-party activation events — `cursor.skill.activated` | name, trigger, source | Cursor Enterprise only | see `cursor-telemetry-surfaces.md` §2 |
| Static BPE — `skill-validator check -o json` | exact cost *if fully loaded* | always | exact, but says nothing about what was actually loaded |
| Per-event `chars/4` summation | context bytes admitted | always | proxy; ordinal only |
| **Paired A/B differencing** | token delta attributable to the skill | **every harness** | **the only method that needs no vendor cooperation** |

**The recommendation the ledger lands on:** paired A/B differencing is the load-bearing method. It
requires no per-fragment instrumentation, works on any harness that reports session totals, and is
the method SkillsBench, ACES, and SkillEvaluator Tier 3 all use.

---

## 8. Where this disagrees with `skill-evaluation-research.md`

| # | Existing file says | This research says | Resolution needed? |
|---|---|---|---|
| C-1 | ACES lift = 0.2134, positive in 72.8% (no interval) | Same, plus **95% CI [0.1967, 0.2301]** | No conflict; the CI is the load-bearing part for a comparison engine (a single paired run is not evidence). |
| C-2 | SkillReducer: "token efficiency comes from compressing/removing skill content" | SkillReducer's abstract reports **compression ratios only, no runtime token or cost reduction** | Tension. The existing implication over-reads the paper. Runtime savings are unproven by that source. |
| C-3 | SkillEval (arXiv:2608.06891v1) cited as a source | Not surveyed in the ledger; ledger surveys SkillJuror (2606.11543) instead | Not a conflict — non-overlapping coverage. SkillEval remains unverified against the ledger. |
| C-4 | agent-profiler (github.com/cleverb/agent-profiler) capabilities | Not surveyed in the ledger | Unverified against a primary source. |
| C-5 | "Skill-Lift-style dimensions: skill execution, behavior check, skill efficiency, token delta" | SkillEvaluator's five published dimensions are Correctness, Discoverability, Effectiveness, Efficiency, Security; ACES's are six unnamed runtime metrics | Two different vocabularies conflated. Pick one and cite it. |

---

## Sources

- SkillsBench — <https://arxiv.org/abs/2602.12670>
- ACES / Skill Lift — <https://arxiv.org/abs/2608.20614>
- NVIDIA SkillEvaluator — <https://developer.nvidia.com/blog/evaluating-ai-agent-skill-performance-with-nvidia-skillevaluator/> · <https://github.com/NVIDIA/SkillEvaluator> · <https://docs.nvidia.com/skills/skillevaluator>
- SkillReducer — <https://arxiv.org/abs/2603.29919>
- SkillJuror — <https://arxiv.org/abs/2606.11543> · <https://github.com/zhiyuchen-ai/skill-juror>
- SkillGenBench — <https://arxiv.org/abs/2605.18693> · SkillResolve-Bench — <https://arxiv.org/pdf/2606.10388>
- Claude Code skills (budgets, `/skill-doctor`) — <https://code.claude.com/docs/en/skills>
- `skillListingBudgetFraction` bug — <https://github.com/anthropics/claude-code/issues/57941> · <https://github.com/anthropics/claude-code/issues/56966>
- `/skill-doctor` press — <https://www.implicator.ai/anthropic-claude-code-skill-doctor-context-audit/>
- skill-validator — <https://github.com/agent-ecosystem/skill-validator>
- skillscore — <https://github.com/sayed3li97/skillscore> · <https://pub.dev/packages/skillscore>
- OTel GenAI semconv — <https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md>
