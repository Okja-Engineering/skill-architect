# S8a — Guardrails: make four footguns fail CI

- **Stage:** parked on SC + S0 + S4 — **no user answer, no docs**
- **Owner:** unassigned
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** **SC** (rule 2 binds to the `estimated` constant), **S0** (rule 4 binds to `profiler/cursor_wirekeys.go`), **S4** (rule 3 binds to the redaction write path)
- **Blocks:** none

## Goal
Four rules that fail CI on a seeded violation and pass once it is removed. Pure enforcement — this half of the old S8 needs no attribution, no docs and no user answer, which is what keeps **both** minimum viable epics closed.

**Why the split.** Round 1's single S8 depended on **S3**, because it also wrote the docs that describe attribution. The fallback MVE (SP1 = no) withdraws S3 — and therefore did not close. The guardrails never needed attribution to exist; the docs did. Splitting on that seam is the fix. `S8b` carries the documentation half.

## Guardrail rules (grep + `go/ast` + Go tests — not semgrep)
1. **No `python3` invocation in any shell script** (`AGENTS.md:6`). Mechanism: `grep -rn 'python3' tests/ scripts/`.
2. **No mislabelled estimate.** A `MetricResult` built as `present` with a `Source` **other than** `estimated`, inside or returned from a function whose name matches `(?i)estimate`. **Rewritten at the spec gate:** the old rule forbade `present` outright from an `estimate*` function, which forbade exactly what Fork (vi) option 2 mandates — a direct design conflict. The rule forbids the **mislabel**, not the state.
   **Mechanism, named (round 2):** a **`go/ast` walk** in `profiler/guardrails_test.go`. Parse the package with `go/parser.ParseDir`; for every `*ast.FuncDecl` whose `Name.Name` matches `(?i)estimate`, walk its body for composite literals of `RawMetricResult`/`TokenResult`/`ToolCallResult`/`ActivationResult` and for calls to the `Present*Result` constructors, and fail if `State` is `MetricPresent` while `Source` is any value other than `SourceEstimated`. Round 1 said only "a Go test over the package's declarations, using the same source scan" and named no analyzer — the one rule not reducible to `grep`, and the one with no stated mechanism. If a reviewer judges the AST walk too much machinery for one rule, the sanctioned fallback is to **drop rule 2 to a documented manual check in `S8b`'s doc** and say so in the PR; what is not acceptable is an unnamed mechanism.
3. **No unredacted payload on the write path.** Every call site that appends to the spool goes through S4's redactor. Mechanism: `grep` for direct calls to the spool-append function outside `profiler/privacy.go` and `profiler/hooks.go`'s single redacted path (R-CS-18/19).
4. **No `cursor.` literal outside the two places D5 allows** — `profiler/cursor_wirekeys.go` and `*_test.go` files. Mechanism: `grep -rn 'cursor\.' profiler/ --include='*.go'` minus those two. Test files are included in the allowance because fixtures must spell wire keys; **D5 states both**, so this file and the roadmap now agree (round 1 widened the allowance here without widening D5). Per D4, S10 emits no `cursor.*` string at all, so there is no S10 conflict. **This rule is why S0 is a dependency:** `profiler/cursor.go:175,182,184,186,197,224,242,262` holds 8 live `cursor.` literals today, and CI would be red from the moment this slice merged until S0 relocates them.

## Test approach
`tests/test_guardrails.sh` (new): for each of rules 1, 3 and 4, seed a violation in a scratch copy of the tree, assert CI's check exits non-zero, remove it, assert it exits zero — red→green per rule, so each guardrail is proven to catch something rather than merely existing. Rule 2 is `profiler/guardrails_test.go`, the `go/ast` walk above, with the same seeded-violation pattern: a test fixture function named `estimateBogus` returning a `present` result with `Source: SourceHooks` must make the test fail.

## Requirements
R-SA-14 (guardrail 4 enforces it), R-RL-15 (guardrails are `grep` + `go/ast` + Go tests, not semgrep)

## Files
`tests/test_guardrails.sh` (new), `profiler/guardrails_test.go` (new), `.github/workflows/ci.yml`

## Next
Dispatchable in wave 3 alongside S3/S5/S6 — all three dependencies land in waves 1–2. `.github/workflows/ci.yml` is also written by S1a in wave 1 (the latency ceiling); two waves apart, no contention.

Ledger §C3:308 settled semgrep: it cannot count tokens (`skill-validator` does, with `o200k_base`), parses markdown only in `generic` mode (≈`grep`), and is a Python package `AGENTS.md:6` excludes from scripts. The ledger flags its own reasoning as unverified — semgrep is not installed here (§G12) — so this is a **Design decision** over a **Local hypothesis**, not a measurement. Do not add it.
