# Confirming pass, lens 3: CAN THE NEW GUARDS FIRE? — PR #22 @ `011defa`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so, verified the
> directory by `ls`, and did not fall back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 6 REAL fail-opens, five sharing one root, plus one incomplete
enumeration and one false shipped figure. Every shipped number holds.**

## Coverage

**12 guards enumerated and walked** (the 7 briefed + 5 more on the diff surface: per-suite
sum equality, `PROMISE_CASES_EXPECTED`, the marketplace/plugin version guard, the audit's
harness-shadowing half, the `-o` idempotency group) · **29 regressions injected**, every one
on **both** `/bin/bash` 3.2.57 and bash 5.3.15 · **33 numeric comparisons enumerated in
`tests/**`**, of which **26 stand behind a count** · **41 added assertion sites** in the
`tests/` diff walked · **all 7 shipped-number rows re-derived**, most by a second method ·
**second sweep** added F3, F5, F7's omissions and F8's stale filename. All 29 injections
reverted; worktree clean at `011defa`; primary tree untouched at `0a83615`.

---

## The two settled questions

**The 218 claim is CONFIRMED — the old floor was worthless.** Injecting
`FNR > 400 { next }` into `tests/lib/audit-suites.sh` gives exactly:

```
sites=218 files=7 lines=9397 accounted=2594
  test_f01 69 · test_f02 52 · test_harness 24 · test_install 0
  test_rewrite 27 · test_skill 26 · test_walk 20
```

**218 ≥ 200.** The old `test "${examined:-0}" -ge 200` would have passed a regression
taking the audited surface from 761 to 218 (−71%) with `test_install.sh` audited at
**zero**. The new two-sided check reddens on both shells with 7 failures.

**`tests/lib/harness.sh:162`'s "819 assertions" is wrong under every derivation, and
nothing depends on it.**

| derivation | value |
|---|---|
| `assert_value` sites in the two large suites (audit tokenizer) | **396** |
| same, `grep -c` | 398 |
| all seven suites, `grep -c` | **402** ← the escalation's figure |
| all seven suites, word-start grep | 400 |
| assertions *executed* through `assert_value` in the two large suites | **2236** (1886 + 350) |
| call sites at `71cc866`, the commit that wrote the sentence | 217 |

819 was not true then and is not true now. `grep -rn 819` returns exactly one hit — that
comment. **A false shipped number, not a broken guard.**

---

## F1 · HIGH · REAL · both shells — `sites` has no independent other side; a runaway heredoc hides vacuous assertions while `lines == accounted == wc -l`

`tests/lib/audit-suites.sh:333-341` (`detect_hd`), `:358-365` (the `in_hd` branch, which
increments `accounted`), `tests/test_harness.sh:289` (`-gt 0`, the only thing behind the
site count).

One legal, non-vacuous assertion added at `tests/test_install.sh:2020`:

```sh
assert "the install route writes no heredoc terminator of its own" \
  quietly grep -qv '<<INSTALL_ROUTE' tests/test_install.sh
```

`detect_hd` reads `<<INSTALL_ROUTE` out of the **quoted** word — its
`sub(/^.*<<[-]?[ \t]*/, ...)` is greedy and does not know it is inside quotes — and sets
`in_hd=1`. No later line equals `INSTALL_ROUTE`, so the rest of the file is read as
heredoc body, **and every one of those records increments `accounted`**. Measured:
`sites=1 files=1 lines=2133 accounted=2133`, `wc -l` = 2133. **Both sides of the new
two-sided count agree.**

Then `assert "the destination is contained" true` and `assert_value "the copy landed" true`
added beneath: **audit exit 0.** With only the runaway line removed: exit 1, both vacuous
sites named. All seven suites green, `TOTAL=2744`, identical on both shells.

`audit-suites.sh:39-40` claims the pair refuses "a runaway heredoc terminator". **It does
not:** `accounted` counts *disposal*, not *examination*, and the heredoc skip path counts
itself.

**Invariant:** the audit must be unable to report clean over a file whose assertion sites
the tokenizer did not reach. `lines`/`accounted` prove the bytes were walked; only an
independently-derived site count proves the sites were seen.

## F2 · HIGH · REAL · both shells — the shadowing check fails open on `name ( ) {`

`tests/lib/audit-suites.sh:287-303` (`shadowed_name`), claiming at `:71-73` to cover "any
of bash's three spellings".

Planted after `harness_init` in `tests/test_walk.sh`:

```sh
assert ( ) {
  echo "PASS: $1"
  pass=$((pass + 1))
}
```

Audit **exit 0**. Vacuity proven independently: with `skills/skill-audit/SKILL.md` replaced
by the single word `BROKEN`, the shadowed suite still reports **20 passed, 0 failed**, where
the real harness reports **18 passed, 2 failed**.

Complete class — 10 spellings, all legal on both bashes. Caught: `assert() {`,
`assert () {`, `assert  ()  {`, all four `function` forms, and the one-line double
definition. **MISSED: `assert ( ) {` and `assert (  ) {`** — whitespace inside the
parentheses, no `function` keyword.

## F3 · HIGH · REAL — same check, EXIT-trap side: `trap cleanup 0` and `trap cleanup exit` escape

`tests/lib/audit-suites.sh:310-316`.

| spelling | audit | installs a real EXIT trap? |
|---|---|---|
| `trap cleanup EXIT` | caught | yes |
| `trap 'cleanup' EXIT` | caught | yes |
| **`trap cleanup 0`** | **MISSED** | **yes** (verified by firing it) |
| **`trap cleanup exit`** | **MISSED** | **yes** |

Both missed spellings displace the harness abort guard — the precise failure the check
names.

## F4 · HIGH · REAL · both shells — the audit reads the suites but not the libraries they source; one line in `tests/lib/masked-path.sh` makes all 1886 of `test_f01.sh`'s assertions vacuous

`tests/test_harness.sh:275`, `:283-290` drive the audit over `$suites` only, while
`audit-suites.sh:46-48` says "Run it over suites and fixtures". Four of seven suites source
libraries (`test_f01.sh:165`, `test_f02.sh:238`, `test_skill.sh:18`,
`test_rewrite.sh:36`).

Appended to `tests/lib/masked-path.sh`:

```sh
assert_value() { echo "PASS: $1"; pass=$((pass + 1)); }
```

and `skills/skill-audit/SKILL.md` replaced with `BROKEN`. Both shells:

- audit over the 7 suites: `sites=761 files=7 lines=9397 accounted=9397`, **exit 0**
- `test_harness.sh`: **191 passed, 0 failed**
- `test_f01.sh`: **1886 passed, 0 failed** — **its exact published figure, over a broken tree**

This is the "roots too narrow" shape one level in: the old `scanFor` missed `tests/` and
`skills/`; this misses **`tests/lib/`**, where four of the seven suites get their code.

## F5 · HIGH · REAL — the harness-name list is a hand-kept literal with no other side, and `harness_ready` has no caller

`tests/lib/audit-suites.sh:116` vs `tests/lib/harness.sh:222-229`.

Adding `assert_all()` to `harness.sh` — a plausible new shared primitive, since `require`
and `witness_exit` both arrived that way — and shadowing it in `test_walk.sh`: audit **exit
0**, `test_harness 191/0`.

**This is exactly the defect the PR fixed one level over.** `tests/test_f01.sh:798-800` now
holds `verdict-guard.sh`'s defined set equal to `verdict_guard_ready`'s own list, with no
number. The same repair was not applied to the harness list read by the same audit — the
identical shape as `test_harness.sh:140-148` recording the lesson and not applying it to
the site count.

**Second half:** `harness_ready` (`harness.sh:222`) is **never invoked**.
`grep -rnE '(^|[^a-z_#])harness_ready( |$|;|\))' tests/` returns only the definition and two
comments. Its own doc says *"Defined last on purpose, so 'stopped short' fails by the same
route as 'never loaded'"* — but with no caller, a truncated `harness.sh` is detected by
nothing, while `audit-suites.sh:116` lists `harness_ready` among the names a suite may not
shadow, implying it is live. `verdict_guard_ready`, its analogue, **is** called.

**Shared root for F1–F5:** every side of `audit-suites.sh`'s shadowing half is a
hand-enumerated literal — the name list, the definition spellings, the trap spellings, the
set of files read — and **not one is held against an independent derivation.** The
assertion-shape half has fixture controls; the byte count has `wc -l`; the shadowing half
has neither. One root fix closes F2–F5 together: derive each side and hold it two-sided,
the way `test_f01.sh:798-800` now does for the guard.

## F6 · HIGH · REAL · both shells — `forward-promise-check.sh` prints `lines` and nothing compares it

`tests/lib/forward-promise-check.sh:123` (`{ lines++ }`), `:171` (the printf),
`tests/test_skill.sh:806-817` (only `files` gets an other side).

Appended to `README.md` at line 992: `- the count of skipped records is deferred to 0.5.0`.
Then inserted `FNR > 400 { next }` below the `{ lines++ }` rule — the same regression that
motivated `audit-suites.sh`'s two-sided count. Reader: `files=162 lines=40028`, **exit 0,
violation not reported.** `test_skill.sh`: **86 passed, 0 failed** on both shells, printing
*"files the forward-promise reader read: 162 of 162 non-empty tracked files"*.

The 18 fixture controls at `test_skill.sh:748-791` **cannot** catch this: all run in
`--over` single-file mode over one-line fixtures, so none reaches line 400 — the exact "a
control that cannot reach the limit it is testing" shape `test_harness.sh:326-329` calls out
and deliberately avoids.

**Invariant:** the reader must be unable to report "none found" over a prefix of a file.
`files` has an other side (`git ls-files`); `lines` has none.

## F7 · MEDIUM · REAL — the enumeration that replaced the blanket exemption is itself incomplete, and one stated real value is wrong

`tests/test_f01.sh:1683-1697`.

- **Omission:** omits `tests/test_f01.sh:2724`, `.summary.policy_failures -ge 8`. Derived
  real value **8** — tight today, but a count-backed floor absent from the list written to
  end blanket exemptions.
- **Wrong value:** "the scripts whose exit contract was compared (`-ge 5`, real 6)" —
  measured **5**, so floor == real, not floor < real. The companion
  `exit_witness_scripts -ge 6` is correct.
- **Incomplete generalisation:** "Everything else numeric in this file is `-gt 0` … a
  fixture's own content, or a timing bound" does not cover `:1610`, `bcalls -gt bprobe` —
  two measured values compared. (Covered in substance by `BROKEN_CASES_EXPECTED=173`.)
- **Dead threshold helper:** `tests/test_f02.sh:33-39` defines `assert_jq_gt` with
  **zero call sites** — a floor waiting for a caller.

## F8 · LOW · REAL — `tests/lib/harness.sh` ships a false figure and four references to a file that does not exist

`:162` — the 819 figure, resolved above. `:20`, `:117`, `:167`, `:189` all name
**`tests/lib/audit-assertions.sh`**; the file is `tests/lib/audit-suites.sh` and no
`audit-assertions.sh` exists. `:167` is the sentence asserting that the non-vacuity of
every `assert_value` "rests on" it. Pre-existing, not in this diff — **but it is what this
release ships, and F1/F2/F4 make that sentence load-bearing.**

## F9 · LOW · SUSPECTED — the f01 drop-one loop derives its case count from its own subject

`tests/test_f01.sh:807-822`. The equality at `:798-800` **does** catch a re-spelling —
proven: a trailing comment after `mktemp_answers() {` (legal, `bash -n` clean) took the
derived set 16 → 15 and reddened both shells at 1881/1 with the naming diagnostic. But a
**coordinated** removal — a primitive deleted from `verdict-guard.sh` and from
`verdict_guard_ready`'s list in one commit — satisfies the equality and silently removes 4
assertions per primitive, with no `..._EXPECTED` number anywhere. An *unused* primitive
would go quietly.

Also (fail-closed robustness, not a hole): `guard_names_ready_requires` at `:781-790`
parses only via `sed -n '/^verdict_guard_ready()/,/^}$/p'`; re-spelling that function as
`function verdict_guard_ready {` collapses the required list to one name and reddens the
suite for no real defect. Reader-spelling audit of `guard_definition_names`: misses
`name ( ) {` and `name() {  # comment`.

---

## Guards that HELD — with the attack listed

| guard | regression injected | verdict |
|---|---|---|
| `audit-suites.sh` two-sided count | `FNR > 400 { next }` above `FNR==1` | **RED**, 7 fails, both shells |
| its own control (`:314-336`) | ran as-is | **fires**, both shells |
| per-suite sum equality (`test_harness.sh:297-303`) | `sites = 0` reset added (total → 20) | **RED**, both shells |
| guard-primitive equality (`test_f01.sh:798-800`) | trailing comment after a definition | **RED** 1881/1, both shells |
| `REQUIRED_TOOL_PRECONDITIONS_EXPECTED=20` | deleted `require_tool grep` | **RED**, "state 19 … says 20" |
| `BROKEN_CASES_EXPECTED=173` | same edit (79 assertions vanished) | **RED**, "is 160 … says 173" |
| `PREFLIGHT_CANDIDATES_EXPECTED=8` | deleted `require_tool wc` | **RED**, "require 7 … says 8" |
| `test_f01.sh:2132` `-ge 8` (real 10) | narrowed the readme reader to 8 of 10 | **RED** via `readme_missing` — floor genuinely redundant |
| release-document guard | ① section cut ② body emptied, each doc ③ manifest → 0.5.1 ④ 0.4.2 dropped from one doc | **RED** on all four, both shells |
| marketplace/plugin version guard | marketplace → 0.4.9 | **RED** |
| `forward-promise-check.sh` root | violation planted in `tests/`, `skills/`, `docs/`, `profiler/`, `README.md`, `.github/` | **RED** on all six |
| its `files` denominator | narrowed to three roots | **RED**, "read 105 of 162" |
| its self-checking exemption | `EXEMPT_PATTERN` made to match nothing | **RED**, names the stale exemption |
| " | deleted a tracked file | **RED**, fail-closed (rc=2) |
| `PROMISE_CASES_EXPECTED=18` | deleted one heredoc case | **RED** |
| `refused_out` | destination refusal `exit 1` → `exit 3` | **RED** 7 fails, both shells |
| protected-roots list, both directions | dropped `.agents` from `protected_home_dirs` | **RED** 4 fails: behavioural + SKILL.md + README |
| `homesafe.PathContains` vs a real write | `filepath.Clean` before the walk | **RED** 6 subtests, both directions |
| drafter containment vs kernel oracle (12 spellings) | inverted `path_inside` args | **RED** 15 fails, both shells |
| install containment vs kernel oracle (9 spellings) | dropped symlink resolution | **RED** and **REFUSED** — precondition halts the suite |
| `-o` idempotency group | deleted the existing-file leaf branch | **RED** 6 fails, both shells |
| bash 3.2 fail-open sweep | grepped every added `tests/` line for a bare `[[ ]]` statement | **none**; four bare `[ ]` uses are all `X \|\| Y` lists | **clean** |

## The numeric-comparison denominator — the true figure

**33 comparisons in `tests/**`; 26 stand behind a count.** No prior enumeration's "seven
floors" matches this surface, and `test_f01.sh:1290` **no longer exists as a floor** — at
`011defa` it is the equality at `:1374-1378`.

Of the 26: **8 equalities against a derived other side** · **4 floors where floor == real**
(`f01:2517` 5/5, `f01:2724` 8/8, `f01:2828` 6/6, `skill:116` 2/2) · **1 floor with slack**
(`f01:2132`, proven redundant) · **1 dead** (`f02:39`) · **12 `-gt 0`/`-ge 1` emptiness
guards or derived-vs-derived**. The remaining 7 are loop bounds, fixture generators, timing
and summary plumbing.

**The one that matters is `tests/test_harness.sh:289`, `-gt 0`** — the only comparison
standing behind the site count, which F1 walks through at a value of 1 against a real 50.

## The shipped numbers — every row re-derived, ALL CONFIRMED

| figure | claimed | derived |
|---|---|---|
| shell assertions, both bashes | 2741 · 1886/350/191/50/158/86/20 | **identical on both**; sum = 2741 ✓ |
| Go top-level | 285 (239/35/11) | **285** ✓ |
| pass/skip | 284 pass, 1 skip | **284/1**; `TestHomeBarrierChild`, legitimately env-gated |
| subtests under `-race` | 696 (579/87/30) | **696** ✓ (818−239, 122−35, 41−11) |
| 579 = 546 + 33 | — | **546 + 33** ✓ by slash depth |
| coverage | 97.9 / 94.7 / 97.0 | **exact** ✓ |
| under `-race -cover` | 3.4% (CLI) | **3.4%** ✓ |
| assertion sites | 761, `lines = accounted = 9397 = wc -l` | **761 / 9397 / 9397** ✓ |
| fixtures / queries | 64 / 8 | **64 / 8** ✓ |
| v0.4.3 baselines | 2499 / 93 / 452 / 59 | **all four exact** ✓, measured at tag `5b60fca` |
| tracked-file denominator | 162 of 162 | **163 tracked, 1 empty** → 162 ✓ |

**Every sum checks. No total disagrees with its parts.**

## Two smaller notes, not findings

- `tests/test_skill.sh:815` builds its denominator with `xargs -0 -I{} sh -c '[ -s "{}" ]
  && echo x'`, interpolating each filename into a `sh -c` string. No tracked filename
  currently contains a shell metacharacter, so inert today; a filename with a quote or `$`
  would miscount or execute.
- Both sides of that comparison come from `git ls-files` in the process CWD. The suites
  `cd` to the repo root via `harness_init`, so it is correct in practice — but run from a
  subdirectory both sides would narrow together and stay green: **the F4/F6 shape in its
  third location.**
