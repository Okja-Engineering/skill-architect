# S3 record — the profile surface, and one registry the CLI dispatches through

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `992a0fa`
**Branch** `feat(profiler)/0.5.0-profile-surface` · **Commit** `6dde7a6` · **PR** [#15](https://github.com/Okja-Engineering/skill-architect/pull/15) (open, not draft, base `main`)
**Worktree** `<scratch>/wt-s3-surface` (session scratchpad, per ruling 2)
**Status** delivered, not merged. The user merges. Nothing tagged.

---

## 1. What landed

12 files, 1092 insertions, 131 deletions. Two of the files are new.

| File | What |
|---|---|
| `profiler/types.go` | the 0.5.0 type surface, hand-ported onto `main`'s file |
| `profiler/profiler_test.go` | the four acceptance tests, two controls, and three derivations that replaced hardcoded counts |
| `profiler/adapters.go` | **new** — the production registry |
| `profiler/adapter_contract_test.go` | walks the production registry; scans `cmd/` for escapees |
| `profiler/claude_code.go` | the sixth capability |
| `profiler/cmd/main.go` | `getAdapter` deleted; both callers resolve through the registry |
| `profiler/cmd/main_test.go` | the sixth signal, two new exit-code cases, a derived capability count |
| `docs/profiler-spec.md` | the new types, the v1 rule stated, the contract section re-pointed |
| `README.md` | the adapter-version sentence, which said the two numbers match |
| `tests/test_skill.sh` | `AdapterVersion` 0.4.3 → 0.5.0, and why it need not equal the manifests |
| `tests/lib/out-of-scope-check.sh` | **new** — the barrier, as a script |
| `tests/test_harness.sh` | the barrier's two directions and its controls |

`git add` was those twelve, named. No `git add -A`, no `commit -a`.

---

## 2. The four acceptance criteria, and the mutation proof for each

**Every check was reverted and watched to redden.** A green run was not accepted
as evidence, per S2's finding.

### 2.1 A profile that estimated nothing has no `estimated_context_tokens` key

`TestAProfileThatEstimatedNothingHasNoEstimateKey`, three subtests: a real
capture through `ClaudeCodeAdapter`, the zero profile, and — the direction that
matters — a profile that **did** estimate, which must carry the key. Without
that third one, deleting the field outright passes the first two.

| mutation | result |
|---|---|
| drop `,omitempty` from the field tag | RED — `a profile carrying no estimate still has a "estimated_context_tokens" key: an empty result is a state outside {present, unknown, error}`, and `the zero profile has a "estimated_context_tokens" key, so every profile does` |
| hold the field **by value** (`EstimatedTokensResult`), exactly as the unlanded draft had it — the D5 trap | RED, both subtests |

The second mutation required patching three call sites that take the field's
address; it was run to completion rather than reported as a compile error,
because "it does not compile" is not the same evidence as "the profile carries
the key".

### 2.2 `SignalStates()` derived by reflection and asserted equal

`TestSignalStatesCoversEveryResultFieldOnTheProfile`. A field is a signal when
its type — or the type it points at — embeds `RawMetricResult`; its name is its
json tag. So the assertion also pins the tag to the `MetricName` constant, which
is the agreement the capability report is keyed by. Asserted over a real capture
and over the zero profile, because the nil estimate is the case a value-typed
field would silently change.

| mutation | result |
|---|---|
| delete `MetricAttribution: p.Attribution.State` from `SignalStates` | RED, printing both maps and naming the field present in one |
| add a seventh result field to `Profile`, not to `SignalStates` | RED — `seventh` in the derived set only |
| neuter the derivation (`f.Anonymous && f.Type == want` → never true) | RED — `the derivation found no result field on Profile, so this check reads nothing`, **and** its own control `TestTheSignalDerivationFindsResultFields` |

The third is the one that matters: without the `len(derived) == 0` guard, a
broken derivation would make the equality vacuously true, which is precisely
S2's `check-ignore` failure mode one file over. The control is handed a shape
carrying one of each thing the derivation must decide — a non-result field, a
non-result struct, a value result, a present optional result, a nil one.

### 2.3 Round trip over a profile carrying every new key

`TestProfileRoundTripsEveryKeyTheSurfaceAdds`. Thirteen keys are asserted
**present in the marshalled document** before it is reparsed, because the keys
0.5.0 adds are optional and a round trip that does not carry them proves nothing
about them.

| mutation | result |
|---|---|
| untag `ToolCallEntry.ErrorType` (`json:"-"`) | RED — the non-vacuity guard fires: `the profile under test does not carry "error_type":"timeout", so round-tripping it proves nothing about that key` |
| make `Profile.MarshalJSON` non-idempotent | RED — `round-trip mismatch`, with both documents |

### 2.4 The reverted rename returns nothing

`TestTheDroppedTokenKeyRenameHasNotReturned`, scanning `profiler/`, `docs/` and
`README.md` for three spellings, with a minimum-file-count guard.

| mutation | result |
|---|---|
| `CacheWrite *int \`json:"cache_write,omitempty"\`` back in `types.go` | RED, naming file and line |
| the rename in the spec and the readme only | RED for both, separately |
| the scan stops matching | RED in its control, `TestTheRenameScanSeesTheRename` |
| the scan stops counting files read | RED on the file-count guard **and** the control |

**A real defect found while writing it.** The first version named what it
refused — in the test's own name and comments — and the scan covers the
directory the test lives in, so it refused itself: five FAILs, all of them its
own prose. Fixed by spelling neither the needles nor the comment with the
refused name. That is not a workaround: it is the scan demonstrating that it
reads its own directory, and the control proves it can still see a hit.

---

## 3. The registry migration landed here. It did not need its own slice.

~120 lines. The three pieces — the registry, the CLI migration, and re-pointing
the contract table — are one change; splitting them would have left a slice
where the CLI and the table disagree about what an adapter is.

### 3.1 How many copies of the harness list were collapsed

**Four, plus the `switch` that defined them.** S2 reported three; the count is
four because the `--harness` flag description is written once in `cmdProbe` and
again in `cmdCapture`.

| was | now |
|---|---|
| `fs.String("harness", "", "harness name (claude_code)")` in `cmdProbe` | `harnessFlag(fs)`, one definition, describing `profiler.SupportedHarnesses()` |
| the same line in `cmdCapture` | the same call |
| `"unknown harness: %s (supported: claude_code)"` in `cmdProbe` | `unknownHarness()`, naming the set from the registry |
| the same line in `cmdCapture` | the same call |
| `switch harness { case "claude_code": … }` in `getAdapter` | **deleted** — both callers ask `profiler.NewAdapter` |

**One deviation from the instruction as written, named.** The instruction was
"have `getAdapter` delegate to it". `getAdapter` would then have been a
pass-through adding nothing, and a second place where the CLI's knowledge of
adapters lived. Both callers now ask the registry directly and the shim is gone
— the same decision carried to its end rather than half-made. Nothing else
called it.

### 3.2 The exported surface is two functions

`NewAdapter(harness, exportFile) (ProfilerAdapter, bool)` and
`HarnessNames() []string`, with `SupportedHarnesses()` as the one-line form the
help text and the refusal share. `NewAdapter` returns an explicit `bool` rather
than a nil interface, so an unknown name cannot be mistaken for an adapter that
declined to build itself. The registry itself is unexported; the contract test
reads it directly, being in-package.

### 3.3 The contract table now walks production

`adapterRegistry` (test-local) is gone. `contractFixtures` is keyed by harness
name and holds only the *inputs*; `contractEntries` walks the **production**
registry and pairs each entry with its fixtures. A registered harness with no
fixtures is a **failure, not a skip** — an adapter the CLI can dispatch and the
table cannot assert anything over is the exact state the contract exists to
refuse, and a skip would report green.

### 3.4 The residual gap S2 named is closed, and proved closed

S2: *"an adapter defined inside `package main` itself would escape the scan. It
cannot be closed from here — a `package profiler` test cannot import `cmd`."*

True about importing; not true about reading. `TestNoAdapterEscapesIntoTheCommand`
parses `cmd/*.go` with the same `go/ast` scan and requires it to find no adapter
type at all — because an adapter found there belongs in `package profiler` and in
the registry, which is what makes it dispatchable in the first place.

Three mutations, all three run:

| mutation | result |
|---|---|
| a second adapter in `package profiler`, **not** registered | RED — `MutantAdapter implements ProfilerAdapter and is not in adapterRegistry` |
| the same adapter **registered** | `probe --harness mutant` produced a report; `--harness nope` said `supported: claude_code, mutant`; both `-h` screens offered it — and `TestAdapterContract` saw it at once: `"mutant" is in the production registry and has no contract fixtures` |
| an adapter defined in **`package main`** | it compiled and was dispatchable by hand; `TestNoAdapterEscapesIntoTheCommand` RED |
| the registry **emptied** | the CLI stopped dispatching `claude_code` entirely and `main_test.go` went red end to end — no second path survives |

The second row is the manager's requested proof in both halves at once: the CLI
dispatched a new adapter, the flag help and the refusal picked it up with no edit
anywhere, and the contract test saw it.

---

## 4. The barrier script and its two-direction test

**`tests/lib/out-of-scope-check.sh`**, per S2 §8.4's recommendation, asserted by
`tests/test_harness.sh` the way the `.gitignore` boundary already is. Exit 0
clean, 1 blocked (naming the paths), 2 could-not-run.

**Why the plan's form is inverted.** As a script's last line,
`git diff --cached --name-only | grep -E '…' && { echo; exit 1; }` exits **1 on a
clean index**, because `grep` reports 1 when it matches nothing. A reader — or a
pre-commit hook — sees a refusal exactly when there was nothing to refuse, and
whoever then "fixes" the hook by deleting it has removed the barrier. The
committed form reports an explicit verdict in both directions.

**One correction the manager's snippet did not have to make, but the script
does.** The staged list is captured into a variable before it is searched rather
than piped into `grep -q`: `grep -q` exits on its first match and the writer
upstream takes a SIGPIPE for it, so under `set -o pipefail` the pipeline reports
141 — and a condition testing it reads a block as a clean run.

**The test: 29 assertions, both directions, over real repositories.** One
repository per refused pattern (a single fixture staging all of them would still
pass with one alternative working), an in-scope index, an empty index, and a
near-miss.

| mutation | result |
|---|---|
| the plan's inverted form dropped in | **5 FAIL** — all three clean-direction assertions, plus `.venv` (absent from the plan's pattern) and the near-miss |
| the pattern neutered to match nothing | **14 FAIL** — every refusal and every "names the path it found" |
| the `^` anchor dropped | **1 FAIL** — the near-miss, which stages `vendor/NOTICE` and `skills/skill-audit/SKILL.md` |

The near-miss exists because without it the pattern could be widened to bare
substrings and every other assertion would still pass, while a later slice found
its own files refused.

---

## 5. The schema stays `profile/v1`, and why

Rule applied: v1 stays while a v1 reader is merely **ignorant** of a new key; v2
is required when a v1 reader would be **wrong**.

Everything added here is optional and absent when it was not read —
`ToolCallEntry.{ErrorType,Count,ID}`, `Attribution.{Category,Detail,OperationName,Confidence}`,
`estimated_context_tokens`. A v1 reader skips what it does not know and is right
about everything it does.

The single condition that would have made that false is §2.1's trap: a
value-typed estimate emits `{"state":""}` on **every** profile, and `""` is not a
member of `{present, unknown, error}` — a v1 reader walking the state vocabulary
would be wrong, not ignorant. The pointer removes it, and two mutations keep the
pointer honest. `ProfileSchema` is untouched at
`"skill-architect/profile/v1"`; no bump was needed and none was made.

---

## 6. `AdapterVersion` — and the plan item it collided with

`AdapterVersion = "0.5.0"`, set once, per D5.

**The plan did not anticipate that this breaks `tests/test_skill.sh`.** That
suite asserts `grep -q 'AdapterVersion = "0.4.3"' profiler/types.go` beside five
plugin-manifest assertions at `0.4.3`, and its comment calls the adapter version
"the sixth surface carrying this release's number". Bumping the adapter without
touching the suite turns it red; bumping the five manifests too would be S10's
work done in S3.

**Resolved the way D5 implies, not the way the prose did.** The shell assertion
moves to `0.5.0`; the five manifests stay at `0.4.3` until the release slice. The
comment now states the real rule — the manifests version the plugin and move
when the release is cut, the adapter version tracks what a profile contains and
moves as soon as that changes — so the suite no longer claims six surfaces must
agree. `README.md:55` carried the same false claim (`"though the two match
here"`) and was corrected. `TestAdapterVersionIsThisRelease` was re-commented to
match.

---

## 7. Files outside the plan's `git add` list, and why each was forced

The plan named three: `profiler/types.go profiler/profiler_test.go docs/profiler-spec.md`.
Nine more were staged. None is a new feature; each is a consequence of an
instruction in the slice.

| File | Forced by |
|---|---|
| `profiler/adapters.go`, `profiler/cmd/main.go` | attached item A, ratified. Named in the instruction. |
| `profiler/cmd/main_test.go` | `SignalStates()` → six. `main_test.go` carries a deliberate guard — *"a new signal must be set here too"* — that Fatals on exactly this. Also held `len(report.Capabilities) != 5`. |
| `profiler/adapter_contract_test.go` | attached item A (walk the production registry) **and** the sixth signal: S2's rule that a report must enumerate every signal the profile carries fired on every stub. |
| `profiler/claude_code.go` | the same rule, on the shipped adapter: its capability report must now advertise `estimated_context_tokens: none`. One entry; the adapter reads an OTel export and has nothing to estimate over. |
| `tests/test_skill.sh`, `README.md` | §6 — the `AdapterVersion` bump the plan required. |
| `tests/lib/out-of-scope-check.sh`, `tests/test_harness.sh` | attached item B, ratified. |

**Three hardcoded `5`s in tests were replaced by derivations, not by `6`.** In
`profiler_test.go` (×2) and `main_test.go` (×1), the capability count is now
`len(profile.SignalStates())` — the invariant is "a report covers every signal a
profile carries", and a literal stops asserting it the moment one is added. The
same repair was made to the stubs in the contract table: `advertising()` builds a
report from `SignalStates` instead of three hand-written five-entry maps, which
is the defect that had just fired on all of them.

One further shape repair in `profiler_test.go`: the exemption
`if metric == MetricSkillActivation || metric == MetricAttribution` was a
conditional chain about to grow a third name. It is now
`if otelBackedSignals[metric]`, which states the actual rule — only a signal this
adapter reads out of the export can be failed by one — and does not grow with
each non-OTel signal.

---

## 8. Drawn from the snapshot: nothing copied, and the warning confirmed

Both files were hash-checked against `snapshot-0.5.0.manifest` before being read:

- `profiler/types.go` — `f7a9849…c786c`, matches the manifest.
- `docs/profiler-spec.md` — `0cfb63b…c5c96`, matches the manifest.

**S2's warning is correct and was verified independently.** The snapshot's spec
is dated 2026-09-10 against `main`'s 2026-09-13 and still describes the generic
`MetricResult[T]` that v0.4.1 removed, along with a "snapshot hash … F04
revalidates" purpose 0.4.x corrected. Its `types.go` is computed against the
pre-0.4.2 file: `Input int` where `main` has `Input *int`, no `SignalStates`, no
`ProbeDiagnoser`, no `SessionIDRequiredError`, `AdapterVersion = "0.1.0"`, and
the `CacheWrite` rename.

**So nothing was copied.** Both files were read for intent and the additions
hand-ported onto `main`'s versions. Applying either wholesale would have reverted
0.4.1, 0.4.2 and 0.4.3 in one commit.

No `checkout`, `reset`, `clean`, `stash` or `rebase` was run anywhere — including
during the mutation runs, which restore by `cp` from a scratch backup rather than
by git. **Primary tree verified untouched after the slice: still `0a83615`, 47
`git status --porcelain` lines, unchanged from the start.**

---

## 9. Verification — measured, not carried

Every base figure was re-derived at `992a0fa` before any edit, and all three
reproduced S2's record exactly (2526 / 100 / 559).

### Shell suites, both shells

| suite | base | head (bash 3.2.57) | head (bash 5.3.15) |
|---|---|---|---|
| test_f01 | 1868 | 1868 | 1868 |
| test_f02 | 350 | 350 | 350 |
| **test_harness** | **102** | **131** | **131** |
| test_install | 48 | 48 | 48 |
| test_rewrite | 85 | 85 | 85 |
| test_skill | 53 | 53 | 53 |
| test_walk | 20 | 20 | 20 |
| **total** | **2526** | **2555 passed, 0 failed** | **2555 passed, 0 failed** |

+29, all of them `test_harness.sh`: the barrier's two directions, its four
fixture preconditions and the near-miss. Identical on both shells; the new code
uses `local` on separate lines, no bash-4 constructs, and no
`local a=$1 b="$a"` left-to-right assumption.

### Go, in `profiler/`

| gate | base | head |
|---|---|---|
| `gofmt -l .` | empty | empty |
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `go test -race -count=1 ./...` | ok | **ok**, both packages |
| top-level tests | 100 | **107** |
| PASS lines incl. subtests | 559 | **571** |
| failures | 0 | **0** |

The seven new top-level tests: the four acceptance criteria, two controls
(`TestTheSignalDerivationFindsResultFields`, `TestTheRenameScanSeesTheRename`)
and `TestNoAdapterEscapesIntoTheCommand`.

### CI

`gh pr checks 15` → **pass, 2m56s** (ubuntu-latest, bash 5, Linux git). It
matters here: the barrier's controls create real repositories with `git init`
and were developed against macOS git.

### Hygiene

- `git add` was twelve named paths. No `git add -A`, no `commit -a`.
- The barrier script, run over its own commit's index: `out-of-scope check: clean`, exit 0.
- `grep -rn 'cache_write\|CacheWrite\|cacheWrite' profiler/ docs/ README.md` → **no matches**.
- Author and committer `imagineux <imagineux@gmail.com>`.
- AI-attribution grep over commit message, author, committer, PR title and body:
  **0 matches**, before and after push. The one hit from a looser grep was the
  string `claude_code` in the commit body — the harness identifier.
- `ProfileSchema` unchanged. No plugin manifest touched. Nothing merged, nothing tagged.
- Nothing installed, nothing reconfigured, no PATH mirror built, no live config
  directory written. All fixture repositories live in the suite's own `mktemp -d`.

---

## 10. Findings — what should change before slice four

### 10.1 S4 inherits a sixth signal and must widen the adapter, not the test. **[carry]**

`claude_code`'s capability report now has six entries and `SignalStates` six
signals. S4 turns `skill_activation` from `unknown` to `present`; the contract
table will check **both** directions of it automatically, and the `absent`
exemption in `TestCaptureDeliversEverySignalProbeAdvertises` currently routes
activation through `otelBackedSignals` as *not* OTel-backed. **S4 must add
`MetricSkillActivation` to `otelBackedSignals`** once activation is read from the
export, or an export that fails will assert `unknown` where the adapter now
correctly says `error`.

### 10.2 The estimate is advertised and never produced. **[expected, but name it]**

`estimated_context_tokens` exists as a signal with one honest answer —
`SourceNone`, absent key — because no adapter estimates yet. **S7 (the hook
spool) is what makes it real**, and it is the slice that must use
`SourceHooksEstimated` and `PresentEstimatedTokensResult`. The contract table
already enforces AC9 both ways over it, so S7 cannot advertise the estimate
without delivering it.

### 10.3 The five plugin manifests now differ from `AdapterVersion`. **[S10]**

By design (§6), but S10 must move `.devin-plugin`, `.claude-plugin`,
`.cursor-plugin`, `.codex-plugin` and `.claude-plugin/marketplace.json` to
`0.5.0` **and** `README.md:424`, which still says `claude plugin list` shows
`0.4.3`. `tests/test_skill.sh` asserts all five, so it fails closed.

### 10.4 The barrier script is asserted but not *run* by anything. **[decide]**

`tests/test_harness.sh` proves it works; nothing invokes it on a real commit.
The cheapest next step is a `.git/hooks/pre-commit` one-liner in a slice's
checklist — but a hook is per-clone and cannot be committed, so it stays a
human step. **Worth deciding**: CI cannot run it either (a PR's index is
already a commit). It is a pre-commit tool for the writer, and every remaining
slice should be told to run `tests/lib/out-of-scope-check.sh` before committing,
now that there is something to run.

### 10.5 `main_test.go`'s `profileWith` is positional over six states. **[note]**

It gained a sixth positional argument across eight cases. Its own guard Fatals
when the count drifts from `len(SignalStates())`, so it fails closed — but a
seventh signal means editing eight call sites again. If S7 adds one, the helper
should take a map. Not worth changing for six.

### 10.6 Smaller notes

- The `go/ast` scan over `cmd/` is the mechanism that closed S2's residual gap.
  The same technique is available to any future "this package must not contain
  X" rule, and costs no dependency.
- `TestCaptureDeliversEverySignalProbeAdvertises` in `profiler_test.go` and the
  contract table remain different denominators, per S2 §8.5. Neither was
  collapsed into the other.
- The worktree went in the session scratchpad per ruling 2. The permission guard
  raised one objection in the whole slice, and correctly: a compound command
  containing `git checkout` was refused even though it targeted the worktree.
  Mutation restores were done with `cp` from a scratch backup instead, which is
  safer anyway — nothing in this slice ran a destructive git command anywhere.

---

## 11. Not done, and why

| Not done | Why |
|---|---|
| Merging or tagging | The user merges. |
| Bumping `ProfileSchema` | §5 — it is not needed, and the pointer is what keeps that true. |
| Bumping the five plugin manifests | S10's, §6 and §10.3. |
| Reading `skill.name` activation | S4's. This slice only widened the surface it will fill. |
| Producing an estimate | S7's, §10.2. |
| Rebuilding the snapshot | Frozen by ruling 4. |
| The `.out-of-scope.md` bullet S1 deferred | Still S5's, per S2 §8.3. Untouched. |
