# Gap audit — lens: skills-function

Audited `origin/main` @ `5c847e1`. **26 findings, all REAL and reproduced. P1 x3, P2 x13, P3 x10.**

> Persisted by the chief of staff: the hunter's session had Write disabled, so it
> returned this report inline. **Verified by CoS** notes are independent confirmation.

Reproduction artifacts, left in place for the fixer:
- Worktree at `5c847e1`: `scratchpad/wt-skillsfn`
- Fixtures: `scratchpad/fx/`

## Coverage

6/6 scripts enumerated from the tree. 22/22 documented items in
`skills/skill-audit/SKILL.md`; 13/13 in `skills/skill-rewrite/SKILL.md`. **8 fixture
targets x 7 invocation forms = 56 executed runs**, plus 6 stubbed-child composition
runs, 3 masked-tool runs, 1 failing-`grep` run, 1 unreadable-`SKILL.md` run, 5 zsh runs.
All 4 shipped suites re-run on macOS bash 3.2: 850 passed, 0 failed. Two full sweeps.

---

## Root A — no single "body" primitive; the checks read the wrong text

`check-frontmatter.sh:66-73` and `audit-report.sh:140-147` strip frontmatter correctly.
`check-paths.sh:64-71` uses a **toggle**. `check-structure.sh:69-93` **does not strip at
all**. Three answers to one question, two wrong.

### A1 — PL005 is dead for every skill that has frontmatter. **P1**

`check-structure.sh:83` greps the whole file; the frontmatter delimiter `---` matches
`^[[:space:]]*[-*]`.

```
$ grep -n '^[[:space:]]*[-*]' fx/nolist-skill/SKILL.md
1:---
5:---
$ check-structure.sh --json fx/nolist-skill
{ "findings": [], "passed": true }      exit=0
```

PL005 can only fire on a file with no frontmatter, which is why only a 0-byte fixture
raised it and no realistic target can.

> **Verified by CoS.** `check-structure.sh:83` is
> `grep -qE '^[[:space:]]*[-*]' "$skill_md"` over the whole file. A leading `---`
> matches. Confirmed.

### A2 — the whole policy verdict is satisfiable from frontmatter. **P1**

Same root. PL002 (`check-structure.sh:70`) and PL004 (`:77`) also read the whole file. A
YAML comment begins `#`, so `## When to use` inside frontmatter is valid YAML **and**
matches the heading regex; a block scalar supplies the code fence.

`fx/fmbleed3` has valid YAML frontmatter, passes skill-validator, and a body of one
prose line:

```
$ skill-validator validate structure -o json fx/fmbleed3 | jq -c '{passed,errors}'
{"passed":true,"errors":0}
$ audit-report.sh fx/fmbleed3 | jq -c '.summary'
{"passed":true,"spec_passed":true,"spec_errors":0,"spec_warnings":1,
 "quality_score":77.5,"quality_grade":"C+","policy_failures":0,
 "path_failures":0,"total_findings":0}
```

`summary.passed: true` for a skill with no `When to use`, no `Deterministic actions`, no
`Orchestration`, no `Constraints`, no `Examples`, no Process heading, no code block and
no list. **The headline command returns the wrong answer.**

**Invariant**: a house-policy check about the body must be computed over the body.

> **Verified by CoS.** Grepping `check-structure.sh` for `in_fm` or `frontmatter`
> returns **zero** matches. It performs no frontmatter stripping whatsoever. Confirmed.

### A3 — a markdown horizontal rule silently ends all path checking. **P1**

`check-paths.sh:64-71`:

```awk
/^---$/ { if (in_fm) { in_fm = 0; next } in_fm = 1; next }
!in_fm  { print }
```

Toggling means the **third** `---` re-enters "frontmatter" and everything after is
dropped.

```
$ check-paths.sh fx/hr-skill        # has a --- rule, then 2 missing refs + 1 missing link
exit=0                              # zero findings

$ check-paths.sh fx/hr-control      # identical file, rule removed
PATH FAIL [PT002]: markdown link target not found: references/definitely-gone.md
PATH FAIL [PT001]: script/reference path not found: ./scripts/definitely-missing.sh
exit=1
```

> **Verified by CoS.** The awk toggle at `check-paths.sh:64-71` is exactly as quoted.
> A third `---` sets `in_fm = 1`. Confirmed.

### A4 — PL003 counts frontmatter toward a documented *body* limit. P3

`check-structure.sh:89` uses `wc -l < "$skill_md"`. A 497-line body plus 5 frontmatter
lines raises `PL003: SKILL.md is 502 lines (max 500)`. The product's own generated draft
says the **body** is under ~500 lines (`draft-rewrite.sh:78`).

---

## Root B — PR 4's invariant covers jq, skill-validator and skillscore, and stops there

`grep`, `awk` and `wc` are never routed through `require_tool`, and `check-quality.sh`
was never brought inside the guard. `set -e` cannot catch these: `! grep ...` is exempt
by definition.

### B1 — a `grep` that errors produces false verdicts. P2

Stub `grep` exiting 2 (the status a real grep uses for an unreadable file, a bad locale,
or a busybox grep without `-E`), against `fx/clean-skill`, whose frontmatter has
`license: MIT`:

```
$ check-frontmatter.sh fx/clean-skill
POLICY FAIL [PL001]: missing license (house policy)        exit=2

$ check-structure.sh --json fx/clean-skill | jq -c '{passed,rules:[.findings[].rule]}'
{"passed":false,"rules":["PL002",...,"PL004","PL005"]}

$ audit-report.sh fx/clean-skill | jq -c '.summary, .policy_error'
{"passed":false,...,"policy_failures":9,"total_findings":9}
null
```

Nine fabricated failures presented as computed verdicts, with `policy_error: null`. Same
shape as the PL001 misclassification PR 4's own commit message names as the bug it
fixed, one tool over. Sites: `check-frontmatter.sh:75`, `check-structure.sh:70,77,83`,
`check-paths.sh:79,99`, `audit-report.sh:149`.

### B2 — `audit-report.sh` exits 2 with no report. P2

`audit-report.sh:140-147` runs an unguarded `awk`; `set -e` aborts.

```
$ chmod 000 fx/unreadable/SKILL.md
$ audit-report.sh fx/unreadable ; echo exit=$?
awk: can't open file fx/unreadable/SKILL.md
exit=2          # stdout: 0 bytes
```

`audit-report.sh:26` documents `{0,3}`. Exit 2 is outside its own set, with no report and
none of the `*_error` keys `SKILL.md:79` promises.

### B3 — `check-structure.sh --json` exits 1 with an empty payload channel. P2

`check-structure.sh:89`'s `wc -l` is unguarded.

```
$ chmod 000 fx/unreadable/SKILL.md
$ check-structure.sh --json fx/unreadable ; echo exit=$?
check-structure.sh: line 89: Permission denied
exit=1          # stdout: 0 bytes
```

`:6` reserves 1 for a computed path-failure verdict; `:18-21` enumerates exactly three
no-payload exits. This is a fourth, wearing a verdict status. Exactly the invariant PR 4
exists to enforce.

### B4 — `check-quality.sh` is entirely outside the guard. P2

The 20-line `check-quality.sh` sources no guard, calls no `require_tool`, emits no
`DEP001`, never checks that `SKILL.md` exists, and passes skillscore's status through.

```
$ check-quality.sh fx/missing-skill   -> exit=1, 52 bytes of spinner padding, not JSON
$ check-quality.sh fx/nosk2           -> exit=1  (dir with no SKILL.md)
```

Three claims broken: `:5` documents `{0,3}`; `SKILL.md:70` documents JSON output;
`SKILL.md:87` says 1 means "spec/path failure". With skillscore absent it prints a bare
error with **no rule ID** while every sibling emits `DEP001`.

**Why nothing caught it**: `tests/test_f01.sh:823-828` greps for rule literals.
`check-quality.sh` has none, so it contributes the empty set and the registry assertion
at `:841` passes **vacuously** over it.

---

## Root C — skill-rewrite was untouched by PR 4

`git diff bd32b26 5c847e1 -- skills/skill-rewrite/` is empty. Its SKILL.md describes a
draft the script does not produce.

- **C1 — the drafter reports success over a failed audit. P2.** `draft-rewrite.sh:53-58`
  folds diagnostics into the report body with `2>&1` and discards the status with
  `|| true`. With tools masked it writes a draft, exits 0, and the draft's "Current
  state" is a wall of policy failures for a skill that has none. Same class as B1, on
  the skill PR 4 did not reach.
- **C2 — "Preserved frontmatter (with corrected `name` if mismatched)" does not exist.
  P2.** `SKILL.md:57`. `draft-rewrite.sh:60-160` never reads, copies or corrects
  frontmatter.
- **C3 — three of five documented section templates do not exist. P2.** `SKILL.md:59`
  documents five. The script emits three, one of which (`Validation checklist`) is not on
  the documented list and fires unconditionally.
- **C4 — the audit-dimension checklist does not exist. P2.** `SKILL.md:60`.
  `draft-rewrite.sh:145-160` emits a fixed five-item boilerplate, byte-identical for a
  clean, a faulty and an empty skill. `evaluation-matrix.md` is never loaded.
- **C5 — the documented invocation does not run as written. P2.** `SKILL.md:52` and
  `:119` give a bare relative path that only resolves when cwd is
  `skills/skill-rewrite`, which the file never states. `skill-audit/SKILL.md` gets this
  right everywhere.
- **C6 — `-a <nonexistent>` silently falls back. P3.** A typo'd path runs a fresh audit
  and reports "No audit report provided", which is not what happened.
- **C7 — the mktemp report leaks. P3.** Never removed, and its path becomes the draft's
  stated provenance.
- **C8 — no exit-code contract. P3.** `-t` with no value gives a raw
  `$2: unbound variable`.
- **C9 — skill-rewrite states no tool prerequisites. P3.** Stage 1 hard-requires
  `skill-validator`; neither it nor `jq` is mentioned.

---

## Root D — documentation PR 4 left stale

PR 4's entire SKILL.md delta is two sentences. The rest still describes pre-PR-4
behavior.

- **D1 — the Stage 2 preflight still exits 1 for a missing tool. P2.** `SKILL.md:46-47`.
  The skill's own recipe for the condition PR 4 standardised on `DEP001` and exit 3.
- **D2 — the preflight never checks `jq`, which PR 4 made unconditional. P2.** A user
  who runs the documented preflight, sees it pass, then runs the documented headline
  command gets exit 3 and empty stdout. The preflight exists to prevent exactly this.
- **D3 — the exit-code line is false for two of the five scripts it governs. P2.**
  `audit-report.sh` **exits 0 on a failed audit**, so `SKILL.md:97`'s "treat a nonzero
  exit as a deterministic failure" never fires for the headline command.
- **D4 — the rule-registry guard is vacuous for `check-quality.sh`. P2.**
- **D5 — `compatibility: POSIX shell (bash 3.2+ or zsh)` is false for zsh. P2.** Under
  zsh, `BASH_SOURCE[0]` is unset, `script_dir` collapses to cwd and `verdict-guard.sh`
  is not found. The four audit scripts correctly refuse with exit 3.
  **`draft-rewrite.sh` writes a draft anyway and exits 0.** The scripts are also
  bash-only, not POSIX shell.
- **D6 — worked example uses a path convention this repo does not use. P3.**
- **D7 — `verdict-guard.sh` appears nowhere in the SKILL.md. P3.** Every script now
  hard-depends on it and exits 3 without it.

**bash 3.2, checked specifically: clean.** No bash-5-only construct present. All four
suites green on macOS bash 3.2. Note that CI has one job on `ubuntu-latest`, so the
bash-3.2 claim is asserted by the frontmatter and verified by nothing. It happens to be
true today.

---

## Root E — check-paths.sh reference detection

- **E1, P3** — `check-paths.sh:107` matches only paths containing `scripts/`,
  `references/` or `assets/`. Dogfooded: skill-rewrite's own documented invocation
  produces zero findings, neither resolved nor reported unverified.
- **E2, P3** — one path referenced twice yields two identical findings, so
  `summary.path_failures` counts references, not broken paths.

## Root F — cosmetic

- **F1, P3** — `check-structure.sh:162` prints a leading blank line in text mode.
- **F2, P3** — all four skill-audit scripts loop over `"$@"` and silently keep only the
  last positional argument.
- **F3, P3** — `audit-report.sh:180-181` counts only `PL*`/`PT*`, so a relayed
  fail-level `DEP002` lands in `total_findings` and in neither counter.

---

## What PR 4 got right — verified, NOT findings, do not re-open

- `--json` requires `jq` unconditionally; both scripts emit a `DEP001` payload and exit 3.
- The composer never reads an unparseable child as "no findings". Six stubbed children,
  all yield exit 3, a `DEP002` payload and `summary.passed:false`.
- All three soft-source claims at `SKILL.md:79` hold exactly.
- All three documented `jq` expressions return the documented shapes.
- Unverified PATH findings do not change exit status.
- The not-a-repository-root guard is enforced.
- Channel separation is clean, verified with separate output files.
- Both shipped skills pass their own audit: skill-audit A/94, skill-rewrite B+/89.5.

---

## Triage

**0.4.3 patch**: A1, A2, A3, A4, B1, B2, B3, B4, C1, C5, C6, C7, C8, C9, D1, D2, D3,
D4, D5, D6, D7, E2, F1, F2, F3.

A1 through A3 alone justify the patch: the headline command returns `passed: true` for a
skill that fails three of its own house rules.

**Advisory fix directions — hypotheses, re-derive them at the root:**

- **One fix closes A1-A4**: a single body-extraction primitive in `verdict-guard.sh`,
  used by both `check-structure.sh` and `check-paths.sh`, that ends frontmatter at the
  second `---` and never re-enters.
- **One fix closes B1-B4 and C1**: bring the remaining tool dependencies and the two
  remaining scripts inside `require_tool`/`cannot_compute`, and stop `|| true` and bare
  `!` from carrying a tool failure onto the verdict path.
- **One fix closes D1-D4**: generate the Stage 2 recipe and the exit-code line from the
  scripts' own headers rather than restating them, as `tests/test_f01.sh:835` already
  does for rule IDs.

**0.5.0 honest deferrals** (new capability, not repair): C2, C3, C4, E1.

C2, C3 and C4 have a legitimate 0.4.3 alternative that is a patch rather than a feature:
correct `skill-rewrite/SKILL.md:57-60` to describe the draft the script actually
produces. Shipping a false description for another release is the worse option.
**Recommendation: doc-honesty in 0.4.3, capability in 0.5.0.**
