# Gap audit — lens: docs-truth

Audited `origin/main` @ `5c847e1`. **10 findings: 0 P1, 5 P2, 5 P3. All REAL except F9.**

> Persisted by the chief of staff: the hunter's session had Write disabled, so it
> returned this report inline.

## Coverage

**4/4 files walked, 186 enumerated claims**: `README.md` 74 · `RELEASE_NOTES.md` 34 ·
`CHANGELOG.md` 44 · `docs/profiler-spec.md` 20 prose + 42 published Go symbols + AC1-AC14.

Executions behind that: the profiler run over **all 54 fixtures** plus **21 purpose-built
fixtures**; a 15-invocation CLI usage/exit matrix; a throwaway Go module compiled against
all 42 published spec symbols; the five audit scripts under 3 tool-absence and 4
child-failure regimes; all four shell suites (850 assertions) plus `go test`, `gofmt -l`,
`go vet`; a mutation experiment against `tests/test_f01.sh`; the README's Python OTLP
receiver run end-to-end against a live POST; and four external sources fetched raw.
Second sweep added F6, F8, F10; a third added nothing.

**Excluded per mandate**: `CHANGELOG.md:15`, `:47`, `RELEASE_NOTES.md:20`, every version
literal, and the missing 0.4.2 changelog section.

---

## Shared root across F1-F4 and F6

Each is a sentence asserting **completeness** — "nothing maps", "does the same", "names
the source", "cannot fall behind", "are null" — where the code delivers a narrower
guarantee.

The project already enforces the right rule on reason strings (`CHANGELOG.md:49`):
**a claim may say what this code does not do; it may not assert an absence or an
exhaustiveness it has not proven.** A holistic fix applies that rule to the five
sentences at once rather than editing them one by one.

---

## F1 · P2 · REAL · the `attribution` reason asserts something false about the harness

`README.md:150-151`, `docs/profiler-spec.md:239`, `:248`, and the string shipped in
**every** Claude Code profile: `"Claude Code telemetry carries no output-to-skill
mapping"`.

False, and the README contradicts it seven lines earlier (`README.md:141-144`). Verified
against Anthropic's live reference fetched raw: `skill.name` is documented as "Skill
active for the request", listed on the token counter, `api_request`, `api_error` and
`api_refusal`, and Anthropic explicitly lists attributing spend to specific skills via
`skill.name` as a supported use.

The harness emits it. This adapter does not read it. Those are different statements.

This is exactly the class 0.4.1 closed for `skill_activation` and left open beside it.

**Invariant**: no reason asserts an absence in the harness the harness does not have.
**0.4.3 patch** — one reason string plus two doc sentences. No schema change,
classification unchanged.

> Cross-lens: the adapter-honesty lens raised the same issue as its F9, SUSPECTED.
> Independently confirmed here against the vendor's live documentation.

## F2 · P2 · REAL · README claims `check-frontmatter.sh` guards three conditions; it guards one

`README.md:280-282`. `check-structure.sh` does all three, reproduced. `check-frontmatter.sh`
with a stub `skill-validator`:

| stub behaviour | result |
|---|---|
| `exit 42` | exit 3, DEP002 on stderr — correct |
| prints `not json`, exit 0 | exit **0**, stdout `frontmatter OK` — wrong |
| prints `{"errors":[]}`, exit 1 | exit **1**, `SPEC FAIL: {"errors":[]}` — wrong |

Structural, not a branch bug: `check-frontmatter.sh:44-63` reads only the child's **exit
status**. It passes `-o json` and never parses the result. The script's own header is
honest; the README inflates it. Row 2 is **a pass verdict computed from a source the
script could not read** — precisely the failure `README.md:273-276` exists to prevent.

> Same defect the install-truth lens raised as its IT-01 (rated P1 there). Two lenses,
> one root.

## F3 · P2 · REAL · the unread spec source is not named

`README.md:291-293` claims that when a source produces nothing readable, it **names the
source** in `spec_error` or `policy_error` and leaves `summary.passed` false.

Policy leg holds. Spec leg fails whenever `skill-validator` produces no output.
Reproduced with a silent stub at exits 0, 1 and 42: all three give `spec: null`,
**`spec_error: null`**, `summary.passed: false`. Control in the same run gives a correct
`policy_error` naming the child.

Root: `audit-report.sh:70-74` sets `spec_error` from the child's captured output, and
`:185` renders the empty string as `null`. A source that dies silently leaves a report
naming nothing. P2 rather than P1 because `summary.passed` stays false.

## F4 · P2 · REAL · the rule-ID list can fall behind

`README.md:286-287` claims the list "cannot fall behind what they produce". Disproved by
mutation: adding one new finding with an unregistered rule ID, leaving every existing
literal intact, gives `561 passed, 0 failed` while a consumer sees the new ID.

Root: `tests/test_f01.sh:824-833` greps four literal spellings. It catches **removal** of
a registered ID, not **addition** of an unregistered one, which is the direction the
sentence claims. Same absolute claim at `skills/skill-audit/SKILL.md:87`.

**0.4.3** for the prose; a behavioural census is an honest **0.5.0** deferral.

## F5 · P2 · REAL · the published contract cannot do what the spec's own reason instructs

`docs/profiler-spec.md:254` quotes the shipped reason: "Provide an OTel export file via
`--otel-file` or `OtelExportFile`." `OtelExportFile` appears nowhere else in the spec.
What the spec publishes is `ProfilerAdapter` and `CaptureOpts`, whose only file field,
`ExportFile`, the spec itself says is **refused**.

A library caller building strictly to the published contract has **no documented way to
supply an OTel export at all**, so every profile they can construct is all-`unknown`, and
the one reason meant to tell them what to do names an identifier the contract does not
define. Proven: a throwaway module compiled against all 42 published symbols builds clean
and none accepts an export path. **0.4.3**, documentation only.

## F6 · P3 · REAL · two keys named at the wrong level

`README.md:270`. `has("quality_score")` and `has("quality_grade")` are both false at top
level; they exist only under `summary`. The adjacent cell uses the qualified
`summary.passed`, so the unqualified pair reads as top-level. Worse than cosmetic:
`jq '.quality_score'` returns `null` both when skillscore is missing and when it scored
94, so the exact distinction the row teaches is undetectable at the path it names.

## F7 · P3 · REAL · stale version attribution

`README.md:28` and `:423`. The delivered profiler is not 0.4.0's. The OTLP/JSON parser,
one-predicate capability model, absent-vs-zero keys, exit-2 contract and the whole series
model are 0.4.1/0.4.2 work. The roadmap records both lines as explicitly out of the
version PR's scope, so they belong here.

## F8 · P3 · REAL · "each of the scripts" is four of five

`README.md:298-299`. `check-quality.sh` neither sources nor checks `verdict-guard.sh`.
Proven by deleting and by truncating the guard in two throwaway trees: all four others
exit 3 naming the guard with empty stdout; `check-quality.sh` runs to completion and
exits 0 with a full report in both. One qualifier fixes it.

## F9 · P3 · SUSPECTED · third-party site cited as product docs

`README.md:336` cites a non-OpenAI domain as "Codex plugin docs". The install row is
honestly marked "not verified", but the **citation** is presented as the product's own
documentation, which is a different claim from the command beside it. SUSPECTED because
it rests on domain ownership, not execution.

## F10 · P3 · REAL · the "snapshot-pinned" rewording missed surfaces it claims to have covered

`CHANGELOG.md:45` claims the rewording landed across five surfaces. Still un-reworded:
`docs/profiler-spec.md:131`, `profiler/types.go:73`, and `profiler capture -h`, captured
live. `Profile.SnapshotHash` itself **was** corrected, so the entry is right about what it
names and wrong about being complete.

---

## Walked and clean — the auditable denominator

- **Commands, flags, exits** all exist and behave as written. Exit matrix across 15
  invocations: usage errors give 1; unusable supplied export gives 2 with the profile
  still on stdout; everything else 0.
- **Published Go contract**: all 42 symbols compile, including the 13 constructors and
  `*ClaudeCodeAdapter` satisfying `ProfilerAdapter`. Round-trip byte-identical.
- **Failure and degradation, all induced**: absent, `chmod 000`, empty, whitespace-only,
  top-level array, malformed, type mismatch, truncated tail, stray brace, null and empty
  `resourceMetrics`, non-OTLP object, BOM. Every documented distinction holds, including
  the byte-offset convention verified against actual file sizes.
- **Token model**: every published number reproduced — 300, 400, 120, 520, 350, 1200050,
  900. Zero start time equals absent; runs identified by start not file order;
  `schemaUrl` not part of resource identity; mixed temporality refused per series with a
  healthy series beside it still present, reduced, and reason-free.
- **Numeric leaves**: out-of-range refused, NaN-string with a valid sibling takes the
  sibling, `-0.4` rounds to 0, `-0.5` refused, exact int64 max preserved.
- **Temporality spellings, attribute-kind canonicalisation, absent-vs-zero, probe/capture
  agreement (AC2/AC3/AC9/AC10), tool-call and timing semantics** all confirmed.
- **The capture recipe works as written**, receiver extracted from the README and run
  against live POSTs.
- **All external facts confirmed** against Anthropic's telemetry reference and the
  collector's `fileexporter` README, including the redaction split and the export-interval
  defaults.
- **Prerequisites table** reproduced under an induced empty `PATH`.
- **Test counts are true at the commit they describe**: 58 functions and 179 subtests at
  `v0.4.1`, exactly as claimed. At `5c847e1` they are 66 and 230, but those statements are
  0.4.1-scoped history, so **not** a finding.
- **Quality scores re-measured and true**: skill-audit 94/A, skill-rewrite 89.5/B+.

## Triage

All ten are **0.4.3**, except the behavioural rule census behind F4, which is an honest
0.5.0 deferral, and F9/F10 which could fold into 0.5.0.

**No P1.** No delivered flow fails and no wrong answer is produced: F3's report still
refuses to pass, F2's gap needs a tool that lies about its own output, and F1 changes no
classification. These are false claims, which is P2 on this mandate's scale.
