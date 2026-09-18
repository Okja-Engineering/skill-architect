# PR #8 gate — cluster C6, install and update truth

Gated `fix/0.4.3-install-truth` @ `f28bb09` against base `71cc866`.

**Verdict: NOT CLEAN — 5 blockers, 14 P3s.**

The behaviour fix converges and every non-vacuity count reproduces exactly. Three things do
not hold: the new suite's own safety guard does not guard, the Codex hedge rests on a
tool-availability claim that is **false on this host**, and the fixed command is destructive
in a failure mode nobody documented.

> Persisted by the chief of staff: the hunter's session had Write disabled.
> **Verified by CoS** notes are independent confirmation.

## Coverage

14/14 diff files · 17/17 hunks · 7/7 C6 entries · **8/8 assertions in the new suite
mutation-proved** · 5/5 of the fixer's non-vacuity claims re-run · 13 probe failure modes
swept · 12 base-vs-head stdout comparisons · 6 suites x 2 shells · 4/4 tool-availability
claims re-checked · 5/5 cited URLs fetched. Second sweep added 9 findings; a third added
nothing.

---

## B1 — BLOCKER — P1 — the new suite will destroy a live skills directory

`tests/test_install.sh` computes a redirect guard, reports it through `assert`, **and then
runs the extracted README commands anyway.** `assert` in `tests/lib/harness.sh` has no abort
path: it prints FAIL, increments a counter, and returns.

Reproduced by respelling the README destination to the exact residual the record names, with
`HOME` pointed at a decoy holding a precious file in each skill:

```
FAIL: the documented destination is redirected away from any live config
FAIL: the documented manual copy installs two skills and is safe to re-run
...
precious files survive? audit=NO rewrite=NO
```

The guard fired, the suite proceeded, and `rm -rf $HOME/.claude/skills/skill-{audit,rewrite}`
ran against the live `HOME`. This contradicts the suite's own comment ("this suite becomes the
thing that corrupts an install") and the lane record's claim that it **refuses to run**.

**Invariant**: no path in a suite may execute the extracted command unless the destination is
*proven* inside the harness scratch root. **A reported failure is not a refusal.**

> **Verified by CoS.** `assert` is `if "$@"; then …PASS… else …FAIL… fi` with no `exit` and no
> `return 1` propagation, and the two install assertions follow the guard unconditionally.
> Confirmed REAL.

## B2 — BLOCKER — the tool-availability claim is false; Codex was runnable here

The record says Codex is "not on PATH, no app bundle". The readme says "no Codex CLI was
reachable to run it".

```
$ /Applications/ChatGPT.app/Contents/Resources/codex --version
codex-cli 0.153.4
```

`command -v codex` is empty, but the app bundle ships the CLI, and `~/.codex/` is a live Codex
home with `auth.json`, `config.toml`, `plugins/` and `skills/`.

Consequences: `codex plugin` has **no `install` subcommand** (only `add` and
`marketplace add`), so the row was settleable with one `--help`; and the binary reads
`.codex-plugin/plugin.json` with fallback to the Claude and Cursor manifests, also checkable.

**This is C6's own defect class recurring inside C6's fix.**

> **Verified by CoS.** The bundle binary exists and reports `codex-cli 0.153.4`.
> `~/.codex/skills/.system` holds six vendor skills. Confirmed REAL.

## B3 — BLOCKER — the hedge steers readers away from a path the vendor's own artifact documents

The readme withdraws `.codex/skills/` and says OpenAI's documentation "names `.agents/skills`
paths instead", to be treated as the vendor's claim.

But OpenAI's own skill, shipped **inside** Codex at
`~/.codex/skills/.system/skill-installer/SKILL.md`, states it installs into
`$CODEX_HOME/skills/<skill-name>`, defaulting to `~/.codex/skills`. The binary's string table
carries that path, and six vendor skills sit there now.

The web docs **do** list `.agents/skills` and `/etc/codex/skills`, so the original finding was
right about those. **Both are true.** The readme presents one and casts doubt on the other.

On the attribution question this gate was asked: **no**, on two counts. The antecedent of
"treat that as the vendor's claim" is ambiguous, and the sentence is substantively wrong
regardless, so clearer attribution would not repair it.

## B4 — BLOCKER — the fixed command is non-atomic: a failed copy leaves nothing

The `&&` guards `cp` against a failed `rm`; **nothing guards the install against a failed
`cp`.** Reproduced from a non-repo working directory over an existing install:

```
before: SKILL.md scripts
cp: skills/skill-audit: No such file or directory   exit=1
after:  []    skill-audit exists? NO
```

The same wrong-directory run under the **pre-fix** command exits 1 with the install intact.
The section documents no working-directory precondition, while the surrounding prose claims
"the `rm -rf` is what makes these safe to re-run".

**Same root, also reproduced**: a reader adapting the destination from the readme's own path
list — which gives **roots** while the command embeds **root plus skill name** — produces
`rm -rf <root>`. On this host, applied to `~/.codex/skills`, that deletes six vendor skills no
reinstall restores.

**Invariant**: the documented update path must never leave the destination worse than it
started, for any failure of any step.

## B5 — BLOCKER — the suite-count floor under-counts once PR #9 lands

The floor reads `-ge 6`, and its stated purpose is refusing a shrunken glob. PR #9 adds a
suite and does not touch that file. Staged together:

```
7 suites present → 54 passed, 0 failed
remove one suite → 50 passed, 0 failed     <-- no FAIL. The guard stopped guarding.
```

A release-integration blocker owned by whichever of #8 and #9 merges second.

---

## The three command forms — the record's safety claim does not hold

| Form | Converges | Drops upstream deletions | A failed run leaves | Deps |
|---|---|---|---|---|
| `rm -rf dst && cp -R src dst` (chosen) | yes | yes | **nothing** | none |
| `cp -R src/ dst` (rejected) | no | **no** | previous install | none |
| `rsync -a --delete src/ dst/` (rejected) | yes | yes | **previous install** | rsync |

The trailing-slash rejection is **correct**. The rsync rejection is **not** correct on safety
grounds — rsync updates in place, so a failed run never empties the destination. Its real
costs are the dependency and two load-bearing slashes.

A fourth form the record never considered is dependency-free and atomic-ish:
`cp -R src dst.new && rm -rf dst && mv dst.new dst`. Offered as a direction to test.

**Destructive-vector sweep, all executed, all safe**: `HOME` unset, `HOME=""`, a space in
`HOME`, `dst` a symlink to a directory, an outbound symlink nested inside `dst`, and `dst`
absent.

## Shared roots

- **R-A, hand-maintained denominators where a derived one exists** — the floor, the workflow
  comment, the readme's suite list, and the manifest list. The irony is local: the harness
  praises its own glob because "a suite added tomorrow is covered the day it lands", and the
  new suite beside it introduces two literals that are not.
- **R-B, a guarantee stated in a comment and not enforced in code** — B1, and the probe test
  that claims to hold diagnostics off the report but stays green when the report is widened.
- **R-C, unavailability asserted without executing the check** — B2, B3, and the Devin row.
- **R-D, idempotence bought with destructiveness, no staging, no stated precondition** — B4.

## Notable P3s

- The readme's "run the tests" block lists **four** suites, omitting the meta-check and the new
  install suite. A reader following it never runs either.
- The new suite's manifest list is hardcoded to four. Proved blind: a fifth manifest directory
  with a bogus skills path, a stray file, and a missing author all pass.
- The suite supplies a `mkdir -p` the readme never documents, so one assertion is true only
  given undocumented setup.
- The Devin row says the release "had no way to exercise" a logged-in account; `devin plugins
  list` returns two plugins here. The honest reason is a deliberate refusal to mutate the
  user's cloud plugin list.
- Cursor's plugins page is cited as its **skills** docs; a separate skills page exists.
- The probe diagnostics test claims to hold diagnostics off the capability report. Adding a
  field to that report leaves the suite fully green. The claim is sound; the assertion is
  mislabelled.
- The spec still declares the old adapter interface and describes probe with no stderr
  channel, while the PR adds an exported interface, method and output stream.
- The extracted readme commands run under PATH `bash`, so they never execute under 3.2 even
  when the suite is started with it.

## Confirmed clean

The Devin `--local` correction is independently confirmed by `--help`. Cursor is genuinely
absent. All four manifests validate, and removing the author block reddens it. **G6-06's
deferral is right, not an evasion** — zero consumers of that field, it is a version surface,
and it is already tracked as an open user decision. Probe exit status is 0 in **all 13**
failure modes with the assertion proven live. Probe's stdout is **byte-identical** between the
base and head binaries in all 12 modes compared. The meta-check is not weakened and does
inspect the new suite. **All 8 new assertions are mutation-provable and none vacuous.** Every
one of the fixer's non-vacuity counts reproduces exactly. Gate totals both shells: **923
passed, 0 failed**.
