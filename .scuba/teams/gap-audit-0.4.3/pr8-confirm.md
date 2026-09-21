# PR #8 confirming pass — cluster C6, `fix/0.4.3-install-truth`

Confirmed `origin/fix/0.4.3-install-truth` @ `84a00cf` against base `71cc866` (24 commits).
Report only; nothing fixed. Own worktree at
`.../scratchpad/c6confirm/head`; all decoys/destinations under my scratch root; HOME and
CODEX_HOME redirected; no PATH mirror, no stubs, primary tree untouched.

## Verdict: NOT CLEAN — 1 P1 blocker (the same class the mandate raised), plus 1 minor test-integrity residue.

**The `eval` is still exploitable at head.** The ordering was fixed and `$()`/backtick/
arithmetic substitutions are now refused, but the shape screen does not screen shell
command-list / redirection metacharacters (`;`, `&&`, `&`, single-word `|`, `>`), so a README
destination like `x;id>FILE` reaches the unquoted `eval` and executes — arbitrary command
execution from a README edit, on the verifying machine, with every containment `require`
printing PASS. Reproduced end-to-end on bash 3.2.57 and 5.3.15.

## Coverage

17/17 diff files enumerated and walked · 34/34 install assertions run (baseline green both
shells) · 7 mutation-controls reverted for discrimination (6 discriminate, 1 does not) ·
9 injection spellings driven through `containment_verdict` · 3 named escapes re-run through
resolution-alone + decoy survival · empty-workflow and missing-workflow re-run live ·
run-site read-count re-counted · 6 suites × 2 shells (969/0 both) · profiler build+vet+test
green · gate B1–B5 + confirm F-8a/F-8b re-walked. Second sweep added the no-destination
residue; a third added nothing.

Per-file: `tests/test_install.sh` (deep), `tests/lib/harness.sh` (`require`), 
`tests/lib/audit-suites.sh` (`require` shape audit), `.github/workflows/ci.yml` (install step +
derived-count comment), `README.md` (all hunks), `tests/test_harness.sh` (empty denominator),
`tests/test_walk.sh` (20/0), `profiler/{types.go,cmd/main.go,claude_code.go,probe_diagnostics_test.go,cmd/main_test.go}`
(build+test), 4 plugin manifests (via suite), `docs/profiler-spec.md` (gate P3, non-blocking, not re-litigated).

---

## C6-CONFIRM-1 — BLOCKER — P1 — REAL — command injection through the containment `eval`

`tests/test_install.sh:352` — `expanded_destination` runs
`eval "printf %s\\n $2"` with `$2` = the README's destination text, **unquoted**, under a
real shell (`env -i HOME=… PATH=$PATH "$suite_bash" -u -c …`).

The only gate in front of that `eval` is the shape pass `unaccountable()` at
`tests/test_install.sh:183-216`, reached via `containment_verdict:407`. It refuses exactly
five shapes: a leading `/`, a leading `~name`, a `..` segment, a backtick, and a `$`-expansion
whose name the block does not bind. **It never inspects `;`, `&`, `|`, `<`, `>`, `(`.** A
destination is treated as a single whitespace-delimited "word" that is safe if it carries no
*expansion* — but a single word can carry a command separator and a redirection, and the
`eval` honours them.

Proven end-to-end at head, suite unmodified, only the README destination line respelled to
`skills_dir=x;id>$DECOY/PWNED-FROM-README`:

```
bash 3.2.57 and bash 5.3.15:
  PASS: the destination the README's block resolves to is inside the harness scratch root
  ... all 5 containment requires PASS ...
  marker exists? YES-EXECUTED
  marker contents: uid=501(...) gid=20(staff) ...   <- `id` ran on the host
```

It runs **during the containment decision itself**: `expanded_destination` returns `x`
(the `printf` output), which resolves inside the scratch root, so
`the_documented_block_is_contained` (`:648`) reports the block *contained* while the injected
`id` has already executed. The block is then run a second time by `run_documented_block`.

Class breadth through `containment_verdict` (marker = did the command run):

```
semicolon   x;id>M    ACCEPTED  ran=YES
and-list    x&&id>M   ACCEPTED  ran=YES
background  x&id>M    ACCEPTED  ran=YES
or-list     x||id>M   ACCEPTED  ran=no (printf succeeds, `||` short-circuits)
$(id>M)/x            refused   ran=no   <- newly closed, correct
`id>M`/x             refused   ran=no   <- newly closed, correct
$((1+1))/x           refused   ran=no   <- newly closed, correct
```

Confirmed the execution site by calling `expanded_destination` directly:
`expanded returned [x]; ran=YES`.

**Why the fixer missed it, and why this is the same root a fourth time.** F-8a's stated root
(confirm-789 §23) was "a sentence doing the work a control should do": the section enumerated
the mechanisms it could think of and gave each a control. But the *enumeration of what makes a
word unaccountable* is itself asserted-complete-and-isn't — it lists expansions and omits
command separators/redirections — and there is **no mutation/control for the missing
category because the category was never conceived**. The completeness defect the round exists
to kill recurs one layer further down: the shape pass is a denylist of five shapes wearing the
words of a proof, and F-8a's own invariant ("a denylist over spellings cannot be a containment
proof; only the resolved write can") is violated by the very seam that was supposed to make the
resolution safe to reach.

**Invariant that must hold**: the README's destination text must never reach a shell that can
execute a command it carries — the screen in front of the `eval` must reject a word containing
*any* shell-active character it cannot account for (command separators, redirections,
groupings), not only expansions; or the destination must be expanded by a mechanism that cannot
execute (no `eval` of untrusted text). Fix direction is the `bug-fixer`'s to re-derive; the
control that proves it must assert a **marker** (the injected effect), not the verdict, because
a substitution/command that reaches the `eval` has already run — the same reason
`containment_refuses_a_substitution_before_it_can_run` (`:536-552`) asserts a marker, but that
control only exercises `$(…)` and `` `…` ``, never `;`/`&`/`|`.

## C6-CONFIRM-2 — MINOR — P3 — SUSPECTED — the no-destination control does not discriminate

`tests/test_install.sh:566-569` / require `:1053-1054`. confirm-789 §23 claims "all seven
mutations now redden their own control." Six do (verified below). The seventh does not:
reverting the explicit no-destination refusal (`containment_verdict:409-412`, `return 1` → no-op)
leaves the suite **34/0, no FAIL** — the block is still refused, by the empty-expansion backstop
(`resolved_destination:363`, `[ -n "$expanded" ] || return 1`), not by the mechanism the
control names. The fixer's own comment (`:146-148`) concedes that backstop exists, so this is a
non-discriminating control, not a hole: a block with no destination is still refused. It is a
residue of the same "each mechanism needs a control that isolates it" discipline, one the
completeness claim overstates. Not blocking on its own.

---

## Confirmed closed / clean (re-proven, not trusted)

- **Item 2 — the three named escapes.** `$HOME/../../../..`, `../../../..`, and the
  fifteen-`../`+absolute-tail spelling are each **refused by the resolved verdict alone**
  (shape pass out of the way, `resolution_refuses_*` at `:1058-1071`), the decoy `SKILL.md`
  survives (`PRECIOUS` intact), and an ordinary `~/.claude/skills` is still **accepted** — so
  the resolution refuses on where the write lands, not by refusing everything. For these three
  spellings the shape pass is **not** load-bearing; resolution carries containment. (The
  blocker above is a *different* class the shape pass alone must catch, and doesn't.)
- **Item 3 — six of seven mutation controls discriminate**, both shells: dotdot-fold, quoted-#
  scan (verified with the correct quote-tracking mutation, not the prev-space guard),
  unbound-name-away-from-destination, `$()`-cmdsub, backtick, and absolute-path each redden a
  control when reverted. The two subtle ones hold: the `..`-fold control is a climb through a
  path that does not exist yet (`:1070`), and the substitution control asserts a marker
  (`:528-534`). The seventh (no-destination) is C6-CONFIRM-2. Discarded-controls claim
  (`$SKILLS_ROOT` / `${IFS}`) is consistent but not independently verifiable from the tree.
- **Item 4 — handover correction is right.** The tautology
  (`everything_the_block_can_reach_is_inside_the_scratch_root`, `path_is_inside "$install_scratch"
  "$harness_scratch"`, true by construction) was closed at **`5be22c8`** (replaced by a
  resolution of the *documented* destination), not by the recovered edit. `fdaa596` only routed
  it through `containment_verdict` (shape-pass-first) and made the single read. The chief-of-staff
  handover was wrong on this point; the fixer's correction is correct.
- **Item 5 — empty/missing denominator.** Live: an empty workflow →
  `FAIL: the CI workflow names suites…` + `REFUSED: the precondition above failed` (refused
  precondition, not a silent abort). Missing workflow file → `grep: …/ci.yml: No such file or
  directory` then the same refusal — fails closed with the real cause named.
- **Item 6 — read once, judged, run.** `run_documented_block:682` reads the block once;
  `containment_verdict:684` is handed that string; `:685-689` writes and runs the same string.
  No run site re-reads the document. Eleven `run_documented_block` call sites, all inherit the
  single-read guard.
- **Item 7 — no regressions.** Both shells: test_install **34/0**, test_harness 70/0,
  test_skill 27/0, test_walk 20/0, test_f01 575/0, test_f02 243/0 (**969/0** total, matching the
  fixer's count). Profiler build/vet/test green. The `-ge 6` floor is gone (derived equality
  count). Version surface untouched; `skills/skill-audit/SKILL.md` untouched. The non-atomic
  update command (staged `mv`), other-skills-alone, converge-on-rerun, and interrupted-recovery
  assertions all pass. `require` is on the shared harness and the assertion auditor rejects a
  vacuous `require` (test_harness `:254-258`).

## What blocks

C6-CONFIRM-1 alone. It is the exact defect the mandate flagged ("the `eval` is still present at
head; what changed is the ordering") — the ordering and the `$`/backtick screen closed the
*substitution* vector the replacement fixer demonstrated, but the *command-separator* vector
into the same `eval` remains open. One root fix (screen the full set of shell-active characters
before the `eval`, or expand without an executing shell) closes both C6-CONFIRM-1 and removes
the need for the incomplete per-shape denylist; C6-CONFIRM-2 is closed by pinning the
no-destination control to a mechanism the backstop does not also satisfy.
