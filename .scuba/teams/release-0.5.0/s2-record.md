# S2 record — the adapter contract, enforced

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `01dce17`
**Branch** `feat(profiler)/0.5.0-adapter-contract` · **Commit** `dc88942` · **PR** [#14](https://github.com/Okja-Engineering/skill-architect/pull/14) (open, not draft, base `main`)
**Worktree** `<scratch>/wt-s2-contract` (session scratchpad, per ruling 2)
**Status** delivered, not merged. The user merges.

---

## 1. What landed

Three files, 892 insertions, 0 deletions.

| File | Lines | What |
|---|---|---|
| `profiler/adapter_contract_test.go` | +730 | the contract table, the registry, the source scan, the lying stubs |
| `docs/profiler-spec.md` | +41 | `## Adapter contract` — the three obligations and where they are enforced |
| `tests/test_harness.sh` | +121 | the `.gitignore` boundary check and its two controls (ruling 3) |

`git add` was the three paths, named. No `git add -A`, no `commit -a`. `profiler/types.go`
was **not** touched — the registry did not need exporting, see §5.

---

## 2. The three obligations, and how each is asserted

The checks are a **function returning violations**, `adapterContractViolations(adapterEntry)
[]string`, not assertions calling `t.Errorf`. That is the structural decision the whole
slice rests on: a check that reports through the testing package can only be shown to work
by breaking the repository, so nobody breaks it and it is never shown to work. As a
function it can be run over adapters built to break each rule, which is what makes §3
possible.

### Obligation 1 — a capture names one session

`Capture("", contractOpts)` must return an error whose text equals
`SessionIDRequiredError(a.Name())`. Compared against the shared constructor rather than
against a literal, so an adapter cannot invent its own wording for the same refusal and
still pass. `a.Name()` is separately required to equal the registry key, because the
refusal quotes it and the CLI selects by it.

### Obligation 2 — AC9, both directions

`Probe()` then `Capture(e.session, …)` of a session the registered export carries.

- **The capability report is the denominator.** Signals are walked out of the report, not
  out of a written-down list, and a report enumerating fewer signals than
  `Profile.SignalStates()` carries is itself a violation. Without that, an adapter
  satisfies AC9 by advertising nothing — the vacuity the whole slice exists to refuse.
  Derived from `SignalStates()` rather than hardcoded to 5, so S3 widening it to six does
  not silently narrow this check.
- **Forward:** `advertised != SourceNone` ⟹ capture state is `present`, `Source` equals the
  advertised source, and the result carries a value. This is the direction that catches
  Devin.
- **Reverse:** `advertised == SourceNone` ⟹ capture state is not `present`. This is the
  direction that catches an adapter reporting a value it never claimed a source for.
- Violations are emitted in sorted metric order, so a run reporting several is stable.

### Obligation 3 — a count that was not read has no key

Two halves, deliberately split.

- **Structural**, once for the package (`TestTokenCountsAreUnreadableAsZero`): every
  `TokenCounts` field is `*int` and carries `,omitempty`. It is not per-adapter because no
  adapter can keep the rule if the type cannot express "not read" — a value field turns an
  absent count into a zero before any adapter gets a say. The `omitempty` half was added
  because a pointer without it serializes an unread count as `null` rather than leaving it
  out.
- **Behavioural**, per adapter: capture the registered **cache-only export**, marshal
  `Tokens.Value`, and require no `input` and no `output` key. Asked of the serialized form
  because a nil pointer and a zero one are the same field until they are marshalled. It
  also requires the fixture to actually yield a cache count, so the rule is asserted over
  something rather than over an export the adapter could not read.
- **The exemption is checked, not declared.** An entry may omit `cacheOnlyExport` only if
  its probe advertises `SourceNone` for tokens. Registering no cache-only export while
  advertising tokens is a violation, so the exemption cannot be used to opt out.
  Both directions are pinned by `TestAnAdapterWithNoTokenSignalNeedsNoCacheExport`.

### What makes registration itself the refusal

`adapterEntry` carries the adapter's constructor **and the two exports it must read**.
Registering an adapter therefore means naming an input for which its own claims hold. An
adapter with no such input — which is exactly what §4 of the plan measured for Devin and
Cursor — cannot be registered without turning the file red. That is the mechanical gate D6
says a future `CursorAdapter` must pass.

### The registry cannot be escaped

`TestEveryAdapterIsInTheRegistry` parses the package's non-test `.go` files with `go/ast`
and collects every type carrying `Name`, `Probe` and `Capture` on either receiver form —
method set, not naming convention, because a type not called `…Adapter` is the one nobody
would think to register. Each must be in `adapterRegistry`, and the counts must match in
both directions. Go cannot enumerate a package's types at runtime, so source is the only
available denominator. The scan carries its own control
(`TestTheAdapterScanFindsAnAdapter`): a synthetic file it must recognise and one with two
of the three methods it must not.

---

## 3. The mutation proof

**A green run alone proves nothing here, so it was not accepted as proof.**

### Live mutation — the Devin shape in the live registry

A stub entry was added to `adapterRegistry` reproducing the Devin adapter's exact shape:
`Probe()` advertising `session_data` for tokens, `Capture()` returning
`UnknownTokenResult("Devin ATIF schema not yet verified; token, tool, and timing fields are
unmapped")`, accepting an empty session, and reporting `input: 0` / `output: 0` on a
cache-only export.

```
--- FAIL: TestAdapterContract (0.01s)
    --- FAIL: TestAdapterContract/devin-shaped-liar (0.00s)
        Capture with an empty session id returned no error: an empty id is no assertion, …
        tokens: probe advertised "session_data", capture state is "unknown" (reason "Devin
          ATIF schema not yet verified; token, tool, and timing fields are unmapped") — the
          capability is a claim about what was read
        the cache-only export carries no input count and the profile reports "input": 0 — a
          measurement nobody made
        the cache-only export carries no output count and the profile reports "output": 0 …
FAIL	github.com/Okja-Engineering/skill-architect/profiler	0.266s
```

**All three obligations fired independently on one adapter.** Removing the entry:
`go test -race -count=1 -run 'TestAdapterContract$' ./...` → **exit 0, ok**.

### Committed mutation — the table's ability to fail is under test

The same liars ship as `TestTheContractTableCanFail`, six cases, each required to produce a
violation **containing its own expected text** rather than merely to produce one:

| stub | must be refused for |
|---|---|
| over-advertising | `tokens: probe advertised "otel", capture state is "unknown"` |
| under-advertising | `tokens: probe advertised "none" and capture returned "present"` |
| accepts-any-session | `Capture with an empty session id returned no error` |
| fabricates-zeroes | `the cache-only export carries no input count and the profile reports "input": 0` |
| advertises-nothing | `the capability report covers 0 signals and the profile carries 5` |
| wrong name | `registered as "wrong-name", Name() returns "declared-name"` |

And the other half, `TestTheContractTableAcceptsAnHonestAdapter`: a stub keeping all three
rules must produce **zero** violations. Without it, every row above is satisfied by a table
that refuses everything — which would also refuse every adapter that keeps the contract.

The stubs are types in the test file, which is why the registry scan skips test sources.
They are not registered: a permanently red suite is a suite that gets deleted.

---

## 4. The `.gitignore` check (ruling 3) and its non-vacuity

**Home: `tests/test_harness.sh`**, not a new file. That suite is "the suite for the
substrate the other suites are written on", and its stated invariant is that a check which
cannot fail is the defect it exists to refuse. S1 §4.3 called the unenforced `.gitignore`
"the same class, one level up", and it is — a guard whose removal is invisible.

**Asked of `git check-ignore`, not of the file's text**, so each assertion is about the
effect an entry has rather than how it is spelled. Reordering, recommenting, or rewriting
`.venv/` as `.venv*/` leaves it alone; removing the enforcement does not.

**Both directions**, 8 + 12 assertions:

- must stay hidden: `tmp/ .venv/ .venv-skillspector/ .scuba/ go.work go.work.sum
  skillgate/ skills/skill-gate/`. `.venv-skillspector/` is named alongside `.venv/` because
  the S1 widening is what keeps the 15k-file virtualenv out, and an entry narrowed back
  would pass a check that only asked about `.venv/`.
- must stay stageable: `README.md AGENTS.md .gitignore .out-of-scope.md
  docs/profiler-spec.md .github/workflows/ci.yml profiler/types.go profiler/claude_code.go
  profiler/cmd/main.go skills/skill-audit/SKILL.md skills/skill-rewrite/SKILL.md
  tests/test_harness.sh` — every one a remaining slice has to be able to stage. An
  over-broad pattern is the more expensive mistake: it blocks a later slice silently.

**The list is written down, deliberately, and the comment says why.** Every other count in
that file is derived from a second place that already holds it. This one has no second
place — the boundary's only statement of itself *is* `.gitignore`, and deriving the list
from the file under test would make deleting an entry delete the check for it. Writing it
down is what makes the deletion visible.

### A real defect found by trying to prove non-vacuity

The first version used plain `git check-ignore`. The removal mutation reddened; **the
over-broad mutation did not** — appending `profiler/` to `.gitignore` left all twelve
"stageable" assertions green.

Cause: `git check-ignore` consults the index first and reports every **tracked** path as
not ignored whatever the patterns say. All twelve paths are tracked, so that whole
direction was asserting nothing. Fixed with `--no-index` on both predicates, which asks the
pattern question rather than the index question — and that is the right invariant anyway: a
path already in the index is safe either way; the file about to be written beside it is the
one that disappears.

**This is the exact failure mode the slice exists to catch, found in the slice's own work
by refusing to accept a green run.**

### Committed controls

Two fixture git repositories, because `check-ignore` is the thing under control and a grep
over text would prove nothing about it:

- `boundary-without-skillgate` — this repository's `.gitignore` minus `skills/skill-gate/`.
  `stageable_in` must report the removed entry, and `ignored_in` must still hold
  `skillgate/`, so a control tree broken outright is caught. A precondition asserts the
  fixture is the repository's boundary minus exactly one line.
- `boundary-too-wide` — this repository's `.gitignore` plus `profiler/`. `ignored_in` must
  report `profiler/types.go`, and `README.md` must still be stageable.

### Live non-vacuity, run against the real `.gitignore`

| mutation | result |
|---|---|
| remove `skills/skill-gate/` | **exit 1** — `FAIL: the boundary hides skills/skill-gate/`, plus the control's own precondition refusing (it can no longer remove a line already gone — correct, and it said so rather than passing) |
| append `profiler/` | **exit 1** — 3 FAILs, one per `profiler/` path a remaining slice needs |
| restored | **exit 0** — 102 passed, 0 failed |

---

## 5. The plan was wrong about one thing: there is no harness registry

S2 is specified as iterating "the harness registry", with `profiler/types.go` staged "only
if the registry needs exporting". **Neither exists.** The only harness list on `main` is the
`switch` inside `getAdapter` in `package main`; `profiler/types.go` has nothing to export,
and a `package profiler` test cannot reach `package main`.

**Resolution, inside the ratified `git add` list:** the registry is defined in the test
file, and bound to reality by the source scan in §2. Since `getAdapter` can only return
`profiler.*` types, an adapter the CLI can dispatch is necessarily a type in `package
profiler`, hence necessarily in the table. The registry is therefore not a decorative second
list — forgetting to register an adapter is a test failure, which is the property D7 needs.

**Residual gap, named:** an adapter defined inside `package main` itself would escape the
scan. It cannot be closed from here — a `package profiler` test cannot import `cmd` — and
closing it means `main.go`, which is outside this slice's `git add` list.

**Recommendation for S3** (which already edits `types.go` and the CLI surface): give
`profiler` a production registry and have `getAdapter` delegate to it. That closes the gap
and also collapses the **three** hardcoded copies of the harness list now in `main.go` —
the `--harness` flag help and the two `unknown harness: %s (supported: claude_code)`
messages, in `cmdProbe` and `cmdCapture`. Adding a second adapter today means editing four
places and the contract table; after that refactor it means editing one.

---

## 6. Drawn from the snapshot: nothing

S2 is new test code and a new spec section. The snapshot was checked rather than assumed:

- `<scratch>/snapshot-0.5.0/profiler/*.go` — grepped for any registry, `allAdapters`, or
  adapter-table prior art. **None.** The unlanded adapters are three hand-written files
  beside a fourth, which is the shape D7 refuses.
- `<scratch>/snapshot-0.5.0/docs/profiler-spec.md` — diffed against the branch. It is
  **behind** `main` (dated 2026-09-10 vs 2026-09-13; still describes the generic
  `MetricResult[T]` that v0.4.1 removed). Nothing to draw; drawing from it would have
  regressed the spec.

No file was copied, so no manifest hash needed checking. The snapshot was not rebuilt, and
no `checkout`, `reset`, `clean`, `stash` or `rebase` was run anywhere. **Primary tree
verified untouched after the slice: still `0a83615`, 47 `git status --porcelain` lines, the
same count as at the start.**

---

## 7. Verification

### Shell suites, measured on both shells

Driven from a bash script, not an inline zsh loop (S1 §4.7: zsh does not word-split
unquoted `$VAR`, so `for s in $SUITES` silently becomes one iteration).

| suite | bash 3.2.57 | bash 5.3.15 |
|---|---|---|
| test_f01 | 1868 | 1868 |
| test_f02 | 350 | 350 |
| **test_harness** | **102** | **102** |
| test_install | 48 | 48 |
| test_rewrite | 85 | 85 |
| test_skill | 53 | 53 |
| test_walk | 20 | 20 |
| **total** | **2526 passed, 0 failed** | **2526 passed, 0 failed** |

2499 → 2526. The 27 are entirely `test_harness.sh`'s 75 → 102 (8 ignored + 12 stageable + 2
preconditions + 4 control assertions + 1 control precondition). Identical on both shells;
the new code uses `local` on separate lines and no bash-4 constructs.

Every other figure was **re-measured, not carried forward** — 2499, 93 and 545 all
reproduced exactly at base before the change.

### Go, in `profiler/`

| gate | result |
|---|---|
| `gofmt -l .` | empty |
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `go test -race -count=1 ./...` | **ok**, both packages |
| top-level tests | **100** (was 93, +7) |
| PASS lines incl. subtests | **559** (was 545, +14) |
| failures | **0** |

### CI, which is a third environment

`gh pr checks 14` → **pass, 3m03s** (ubuntu-latest, bash 5, Linux git). That matters more
than usual for this slice: the boundary controls create real git repositories with `git
init` and depend on `check-ignore --no-index` semantics, and both were developed against
macOS git. The `test_harness.sh` glob-versus-workflow equality check also still passes, so
the suite list did not drift.

### Hygiene

- `git add profiler/adapter_contract_test.go docs/profiler-spec.md tests/test_harness.sh` —
  three named paths.
- D4 barrier 3 in the **explicit-verdict form** (ruling 1), run before the commit:
  `out-of-scope check: clean`, **exit 0**. Script at
  `<scratch>/s2-out-of-scope-check.sh`; it prints which of the two states it found rather
  than leaving a bare `grep` status as the verdict.
- Author and committer `imagineux <imagineux@gmail.com>`.
- AI-attribution grep over commit message, author, committer, PR title and body:
  **0 matches** before push and re-checked after.
- **No version surface touched.** `AdapterVersion` still `"0.4.3"`, `ProfileSchema` still
  `skill-architect/profile/v1`.
- Nothing installed, nothing reconfigured, no PATH mirror built, no live config directory
  written. The two fixture git repositories are inside the suite's own `mktemp -d` scratch
  and are removed by the shared harness's EXIT handler.

---

## 8. Findings — things that should change before slice three

### 8.1 The plan's "harness registry" does not exist. **[ratify the resolution]**

§5 above. The slice is delivered without it, but S3 should be told to build the production
registry and migrate `getAdapter`, both to close the residual gap and because `main.go`
currently holds three copies of the harness list. S3 already edits `types.go`, so it is the
cheap moment.

### 8.2 `git check-ignore` without `--no-index` cannot see an over-broad pattern. **[note]**

§4. It is fixed here, but it is a trap with a wider blast radius than this suite: anything
in this release that reasons about "can this path be staged" by asking git, including a
pre-commit hook someone may write from the barrier-3 snippet, gets the reassuring answer
for every tracked path. Worth a line wherever the barrier is written down.

### 8.3 The `.out-of-scope.md` bullet S1 deferred is still deferred. **[carry]**

S1 §4.1 moved it to S5. S2 did not touch `.out-of-scope.md` beyond asserting it stays
stageable. Still open, still S5's, together with `README.md:612` and `:303`.

### 8.4 Ruling 1's barrier script is still not committed. **[decide]**

The explicit-verdict form was run before this commit and passed, but it lives in
`<scratch>` and dies with the session. S1 §4.2 recommended committing it; ruling 1 only
required running it. Every remaining slice re-derives it from the ruling text, which is the
copy-by-hand pattern `tests/lib/harness.sh` exists to end. Cheapest fix: one script under
`tests/lib/`, asserted by `test_harness.sh` the way the boundary now is. It is outside S2's
add list. **Manager to decide, before slice three rather than after.**

### 8.5 Smaller notes

- The worktree went in the session scratchpad per ruling 2. The permission guard raised no
  objection on any Go or shell write, which confirms ruling 2's diagnosis from the other
  side.
- `go/parser.ParseFile` over a `filepath.Glob` was used rather than `parser.ParseDir`,
  which is deprecated. No dependency was added; the module still has no `go.sum`.
- `TestCaptureDeliversEverySignalProbeAdvertises` in `profiler_test.go` is **not**
  duplicated by this slice and was not touched. It iterates 60-odd export shapes over one
  adapter; the new table iterates adapters over one export each. Different denominators,
  and neither substitutes for the other. A reader who removes one because "AC9 is already
  tested" would lose real coverage; the new file's header comment says so.
- CI runs `tests/test_harness.sh` already, so the boundary check is enforced on every push
  with no workflow change. `.github/workflows/ci.yml` was not touched, and
  `test_harness.sh`'s own glob-versus-workflow equality check still passes.

---

## 9. Not done, and why

| Not done | Why |
|---|---|
| A production registry in `profiler/types.go` | It was not needed to make the table complete, and migrating `getAdapter` onto it means staging `profiler/cmd/main.go`, which is outside the ratified `git add` list. Surfaced as §8.1 rather than done quietly. |
| Committing the barrier-3 script | Outside the `git add` list. Recommended, §8.4. |
| Touching any version surface | D5 — `AdapterVersion` moves once, in a later slice. |
| Rebuilding the snapshot | Frozen by the manager's ruling 4. |
| Merging or tagging | The user merges. |
