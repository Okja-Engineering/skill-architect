# PR #4 — correctness-lens findings, for the round after the current fix pass

_Chief of staff, 2026-09-14. From the guard-correctness hunter on 60c7f44, which returned after the first fix round was already dispatched. Items already in that round (lib/ flattening, pretty-print disclosure, the `error` key, SKILL.md rule IDs, printf escaping, audit-report jq guard) are not repeated here._

Solid and not to be re-litigated: S1 and S2 are genuinely closed. All 54 base-to-head behavioural changes across a 400-cell tool-presence matrix are masked-tool cells, every one moving to exit 3. No cell anywhere shows exit 0 or `passed:true` alongside a non-empty stderr. Every reachable exit is in {0,1,2,3}. The child-exit enumeration closes the class: 143, 127, 126 and a garbage payload all produce `cannot_compute`.

## Root A — the new tests pin the named instances, not the boundaries the change created

Three non-equivalent mutants survive all 138 assertions, and each reproduces the S1/S2 class this PR exists to close. **One fix closes all three: for every `case` or `||` the change introduced, pin both the accepted set's edge and the rejected set, not one witness of the rejected set.**

- **A1** `check-structure.sh:93-94` — restoring `2>/dev/null || true` on the `jq -c '.findings[]'` read-back survives the whole suite. With a child printing `garbage not json` and exiting 0: head gives exit 3 and "did not produce a readable payload"; the mutant gives exit 0, `passed:true`, silently. The test at `tests/test_f01.sh:335` stubs a child that prints *nothing*, so it covers the empty payload and never the unreadable one. Invariant: an unparseable child payload must not read as zero path findings.
- **A2** `check-structure.sh:86-89` — widening `0|1)` to `0|1|3)` survives. With a child exiting 3 carrying a `DEP001` payload, the mutant emits `passed:true` next to a `level:"fail"` finding, because `fail` derives from `path_code` alone and is never cross-checked against the child's `.passed`. The suite tests exit 42, the `*` arm, but never the enumerated set's edge.
- **A3** `check-frontmatter.sh:43-45` — letting the `3)` arm fall through to the license gate survives, and prints `frontmatter OK` with exit 0. That is literally S2's symptom on a pre-existing branch. The suite stubs the validator at 42 and at 2, never at 3.

## Independent

- **A4 (MEDIUM)** — the `DEP001`/`DEP002` payload is correctly shaped but **unreachable by its only machine consumer**. `audit-report.sh:63` captures `2>&1` and `:64-66` has no else-branch, so the blob is never valid JSON and `policy_findings` stays `[]`. Reproduced with all tools present and `check-paths.sh` removed: `check-structure.sh` emits the DEP002 payload and exits 3, and `audit-report.sh` still reports `{"passed": true, "total_findings": 0}` with exit 0. A counterfactual capturing stdout only would flip it. This makes `lib/verdict-guard.sh:23-25`'s claim that "a machine consumer sees a failing verdict rather than a missing one" false end to end, and leaves mandate invariant 1 violated at the composition boundary. Not a regression, base does this too, and `audit-report.sh` was out of scope; it is the one place the change's stated purpose does not land. Invariant: a composer must not treat an unparseable child payload as "no findings". Note this is distinct from the jq guard already routed to `audit-report.sh` in round 1.
- **A5 (LOW-MED)** — the new `source` is itself an unguarded dependency that fails with a **verdict** code. If `lib/verdict-guard.sh` is missing or malformed, `set -e` aborts with exit 1, which the contract defines as "spec/path failure", with no payload in `--json`. Reproduced on all three scripts: `exit=1`, stdout `[]`, and `audit-report.sh` then reports `passed:true`. This is the one dependency `require_tool` structurally cannot cover, and the change introduced it. Reachability is low since the plugin ships `./skills/` wholesale. Invariant: no failure to reach a verdict may exit with a code that means a verdict was computed.

## Also confirmed, already routed in round 1
A6 (the `scripts/lib/` self-audit warning, 0 → 1, decision taken: flatten), F7 (pretty-print formatting change), the undocumented `error` key, the SKILL.md rule-ID omission, and the `printf` JSON-escaping hazard in `cannot_compute`.

## Notes
`--json` still does not always emit a payload: the usage branches and the SKILL.md-not-found paths exit 3 as plain text, and `check-structure.sh:34` and `check-frontmatter.sh:22` write that message to **stdout** while `check-paths.sh:33` correctly uses stderr. Pre-existing, but it makes `README.md:247`'s "a `passed:false` payload whatever the skill contains" untrue for a nonexistent directory.

`require_tool skill-validator false` at `check-frontmatter.sh:27` is redundant for exit codes — the `*` arm already maps 127 to exit 3 — but it buys a better diagnostic. Keep it; do not count it as load-bearing.
