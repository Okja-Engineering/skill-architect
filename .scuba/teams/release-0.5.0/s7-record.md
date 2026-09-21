# S7 record — the hook spool, and the claims it is allowed to make

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `d77d90e`
**Branch** `feat(profiler)/0.5.0-hook-spool` · **Commit** `5c84911` · **PR** [#19](https://github.com/Okja-Engineering/skill-architect/pull/19) (open, not draft, base `main`)
**Worktree** `<scratch>/wt-s7-hook-spool` (session scratchpad)
**Status** delivered, not merged. The user merges. Nothing tagged.

`<scratch>` = `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad`

---

## 1. The manager's ruling, sanity-checked rather than accepted

The ruling was: ship, because the spool is lossless by design, so a wrong parse
is recoverable from the same files — categorically unlike an adapter advertising
a capability its capture cannot produce.

**The reasoning holds, and it is the right line. But the draft did not satisfy
its own premise in two places, and I had to repair both before the ruling's
justification was true of the code.**

### 1.1 "Lossless" was false: numbers went through `float64`

The draft decoded each payload into `map[string]any` and re-encoded it. Every
number goes through `float64` that way, so **an integer above 2⁵³ comes back
changed**. A nanosecond epoch timestamp is about 1.7×10¹⁸ — exactly the kind of
field a hook payload carries, and exactly the field this repo already pairs runs
on (`startTimeUnixNano`, plan §4(ii)).

That is not a cosmetic gap. The manager's entire argument is that a wrong *parse*
is recoverable because the payload is still on disk. A payload whose numbers were
silently rounded on the way to disk is **not** recoverable — the better reader
written later reads the rounded number.

Repaired: `decodeOneJSONValue` uses `json.Decoder` with `UseNumber()`, so every
numeric literal reaches the spool as it arrived. `TestNormalize_KeepsLargeIntegersExactly`
pins it; M3 (UseNumber dropped) kills 3 tests.

### 1.2 "Lossless" is still not *literally* true, and the docs say which part

With `UseNumber`, **no field is dropped and no value is changed.** Two things are
still not preserved, and I wrote them down rather than let "lossless" imply more
than it delivers:

- the order of an object's keys;
- a duplicate key, which collapses to its last occurrence.

Neither changes any field's value and the JSON object model gives neither
meaning, so the recoverability argument survives. But "lossless" on its own was
a word doing more work than the code, so the spec states the claim as **"no
field is dropped and no value is changed"** and names the two exceptions.

### 1.3 So the ruling is right, and I narrowed where it applies

The honesty requirement being on the claims and not the code is correct — with
one amendment I made and am flagging: **the data is a claim too.** §3.1.

---

## 2. What landed

10 files, 3,896 insertions, 26 deletions. Six are new. This is exactly the
plan's `git add` list — no file was forced outside it.

| File | What |
|---|---|
| `profiler/hooks.go` | **new** — the spool writer, the envelope, redaction, strict mode |
| `profiler/hooks_install.go` | **new** — `hooks.json` merge and removal |
| `profiler/spool_profile.go` | **new** — the spool **reader**. Not the profile projection (§3.1) |
| `profiler/hooks_test.go` | **new** — the capture contract, and the home barrier |
| `profiler/hooks_install_test.go` | **new** — idempotence, preservation, the refusals |
| `profiler/spool_profile_test.go` | **new** — the read-back, and the session-scoping refusal |
| `profiler/cmd/main.go` | `ingest`, `hooks {install,uninstall}`, the usage blocks |
| `profiler/cmd/main_test.go` | the CLI half, the sandboxed runner, two AST derivations |
| `docs/profiler-spec.md` | `## Hook spool contract` |
| `README.md` | "Capturing Cursor hooks to a spool", and one stale table cell |

`git add` was those ten, one at a time. No `git add -A`, no `-f`, no `commit -a`.

---

## 3. The one place the plan turned out to be wrong

### 3.1 `profileFromSpool` does not land, and the plan lists the file that holds it

**The plan's S7 says `spool_profile.go` lands. I landed the file and left out 60%
of it.** Named here rather than buried, because it is a deviation from the
plan's letter.

The draft's `spool_profile.go` did two things: read a session's events out of the
spool, and project those events into a `Profile` — `pairToolCalls` joining
pre/post hooks on `tool_use_id`, `inferActivations` reading a skill activation
out of a `SKILL.md` file read, `attributeCalls` linking each call to the most
recent activation before it, `sessionTiming`, and `profileFromSpool` assembling
the result.

Four reasons the projection does not land:

1. **It is unreachable.** Its only caller in the draft is `cursor.go`, the
   adapter §4 deletes. `profileFromSpool` is unexported, so landing it means
   landing a function nothing can call, which `staticcheck` U1000 flags and no
   test outside its own would ever exercise.

2. **It does not compile against `main`.** `getString`, `toInt`, `parseTime` and
   `skillPathRe` do not exist on `main` — measured, `grep` over every non-test
   file in `profiler/`. The fifth independent confirmation that the snapshot is
   out of date on the merits.

3. **It makes the claim the manager forbade — in the data rather than the prose.**
   The ruling says nothing in the spec or readme may assert that this reads
   Cursor correctly. `p.ToolCalls = PresentToolCallResult(calls, "hooks")` is
   exactly that assertion: `state: present, source: hooks`, on the strength of
   guessed field names. The prose limit cannot reach it, because a stored profile
   travels — it is subtracted by `compare`, aggregated by `experiment`, and read
   months later by callers who never see the file. **It is the unrecoverable half
   of the manager's own distinction, one layer down.**

4. **It replicates the defect class §4 catalogued.** A `Profile` whose
   `tool_calls` says `present` on a payload nobody has observed is the Devin
   adapter's pattern with a different source name.

**What I landed instead** is `LoadSessionEvents` plus `sessionMatches`: the read
that is true regardless of whether the guesses are right. It is reachable and
load-bearing *in this slice* — `TestLoadSessionEvents_RoundTripsWhatIngestWrote`
is the only thing that actually proves the capture claim, because a line that
cannot be read back is a line that was lost — and it is the primitive S8's
`analyze` builds on.

**The file is now misnamed**, and I did not rename it: `spool_profile.go` holds
no profile. The plan's `git add` list names the path literally and renaming would
add files outside it. **The slice that lands the projection should rename both
files to `spool_read.go` / `spool_read_test.go` or bring the projection back.**
Flagged as S8's or the adapter slice's call, not taken here.

I did not stop and ask before doing this, and I want to be explicit about why:
the manager's dispatch already decided it — "Do not add a Cursor adapter… This
slice is capture infrastructure only," and "If you cannot write a true sentence
about some behaviour, say so rather than writing a hedged one." Applying that
ruling to a file the plan listed before the ruling existed is execution, and it
*reduces* scope. **If the manager reads it otherwise, restoring the projection is
one file and its tests, and I should be told.**

---

## 4. The home-safety proof, and how it decides after resolution

The plan was right that the boundary shape is already correct and I preserved it:
`InstallHooks(home, …)`, `UninstallHooks(home, …)`, `AppendSpool(dir, …)` and
`Ingest(…, dir, …)` all take the directory they write; `DefaultSpoolDir()` is the
only function that reads the real home, and it only computes a path. Nothing in
the library creates it.

The risk is the CLI wiring. Three mechanisms, each proved by reverting it.

### 4.1 The decision is made after resolution, not from the shape of a path string

`pathContains(parent, child)` resolves **both** sides and then asks
`filepath.Rel`. Two properties, and each is the one a naive check gets wrong:

| what a naive check does | the case that catches it |
|---|---|
| compares the unresolved strings | a symlink pointing into the protected directory: **outside** by every string comparison, inside in fact. Reachable by accident — macOS hands `t.TempDir()` a symlinked path of its own (`/var/folders` → `/private/var/folders`), and `$HOME` can be a symlink |
| `strings.HasPrefix` for containment | a sibling sharing the name as a prefix: `/Users/alice-backup` is **inside** `/Users/alice` by `HasPrefix` and outside in fact |

**Both are asserted against directories the test creates**, not against whatever
happens to exist beside the real home. That matters: my first version asserted
the sibling case as `$HOME + "-decoy"`, which does not exist, so
`EvalSymlinks` failed and **the case skipped** — the barrier's own proof passing
by not running, which is exactly S6's generalisation firing on my own work.
`TestPathContains_DecidesAfterResolution` now builds `alice`, `alice/.cursor`,
`alice-backup` and a symlink inside a `t.TempDir()`, and skips nothing.

`realHomeContains(dir)` joins the decision to the real home, and is asserted only
on paths that already exist. **It is the only code in the slice that looks at the
real home, and it looks only.**

An error resolves to `contained = true`. A path the barrier could not resolve is
not a path it has shown to be outside the home, and "I could not tell" must not
read as "go ahead" (`TestRealHomeContains_RefusesRatherThanGuessesWhenItCannotResolve`,
both packages).

### 4.2 The abort aborts — and the shipped defect no longer compiles

The defect PR #8's gate found was a guard that called `Errorf`: it marks the test
failed and then **returns**, so the install on the next line ran anyway.

The guard takes `fatalTB`, an interface carrying **only** `Helper()` and
`Fatalf()`. Writing the defect is now a type error, in both packages:

```
MB5   Fatalf → Errorf (profiler)  DID NOT BUILD
      ./hooks_test.go:233:6: tb.Errorf undefined (type fatalTB has no field or method Errorf)
M66b  Fatalf → Errorf (cmd)       DID NOT BUILD
      cmd/main_test.go:593:6: tb.Errorf undefined (type fatalTB has no field or method Errorf)
```

**These two "DID NOT BUILD" verdicts are kills, not non-results**, and that
distinction is the one place I depart from S6's rule that a non-compiling mutant
proves nothing. S6's rule is about *careless* mutations — the unused-variable
trap, which hit me six times (M14/M15-class: M35, M52, M53, M54, M55, and M15's
first form) and was re-run in compiling form each time. Here the compiler error
**is** the proof: the defect is unrepresentable. I have labelled them
`KILLED BY THE COMPILER` in §6 with the error text shown, so a reader can judge
rather than take my word.

The guard also `return`s after each `Fatalf`. With a real `*testing.T` that is
unreachable, and the point is that the guarantee is the function's own rather
than borrowed from `testing`.

### 4.3 The write after the barrier never runs — reproduced, not argued

`TestHomeBarrier_StopsTheWriteThatWouldFollow` runs **one test in a child
`go test`** whose `HOME` is a temp directory, so the home the child refuses to
touch is a fake one and the real home is never in play. The child calls the
barrier on a path inside that fake home and then writes a marker file standing
for `InstallHooks`. The parent asserts: the child failed, it said "refusing",
and **the marker does not exist.**

That last assertion is the one that would have caught the shipped bug. MB3 (the
barrier checks nothing) kills it.

### 4.4 Nothing can forget it — the mechanism, not the convention

This is the integration rather than the guard. Per `integrate-dont-bolt-on`, I
repaired the shape instead of adding a check at each call site:

- **`runCLIIn` is the only place in `cmd/main_test.go` that starts the profiler
  binary**, and it sets `HOME` to a checked sandbox before the process starts.
  So a case that forgets `--home` or `--spool-dir` writes into the sandbox
  rather than the real home. This required migrating **five pre-existing
  `exec.Command(bin, …)` call sites** (capture, help, usage errors, probe, the
  help derivation) onto the shared runner — a mechanical refactor guarded by the
  existing suite, which removed duplicated stream/exit-code handling as a side
  effect.
- **`TestEverySubprocessRunsThroughTheSandboxedRunner`** walks the test file's
  own AST: every `exec.Command` must sit in `buildProfiler` (which runs
  `go build`) or `runCLIIn`. M61 — one stray `exec.Command(bin, "version")` —
  kills it. M63, an allowance naming a function nothing declares, kills it too,
  so the allow-list cannot go stale.
- **The same test derives that `runCLIIn` calls the barrier.** M68b kills it;
  the control M69 proves the derivation reads a real function.

### 4.5 Two mutations survive by construction, and I am not calling them clean

| | |
|---|---|
| **M62** the derivation's own allow-list check → `if false` | **SURVIVED.** Mutating an assertion so it cannot fail is unkillable for *every* test in every suite; it is not a finding about this one. M61 is the meaningful direction and it kills. |
| **M67** `sandboxHome` stops calling the barrier | **SURVIVED.** A guard that never fires in a healthy tree changes nothing observable when removed. It is now *redundant* defence: `runCLIIn`'s guard is the load-bearing one and §4.4 derives it. `sandboxHome`'s is belt-and-braces and I kept it. |

**The residual risk, stated:** on a machine whose `TMPDIR` lives under `$HOME`,
`t.TempDir()` would be inside the real home. The barrier catches that — that is
the case `realHomeContains` exists for — and `runCLIIn`'s derived guard means
every subprocess is checked. `sandboxHome`'s guard being removable is a
maintenance risk, not a hole today.

### 4.6 Measured: the real home was never touched, and could not have been

- **`~/.cursor` does not exist on this machine at all** (`ls -ld ~/.cursor` →
  no such file), consistent with Cursor not being installed. `~/.skill-architect`
  likewise absent after the full suite and the live demo.
- Every write in every test derives from `t.TempDir()`; every write in the live
  demo (§8.4) went under `<scratch>/s7-demo/home`.
- Nothing installed, nothing reconfigured, no PATH mirror of symlinks built, no
  live config directory written.

### 4.7 The duplication, and what should be done about it

The barrier exists **twice** — in `profiler`'s tests and in `cmd`'s — because
they are two packages and the alternative, exporting it from `profiler`, is
production surface existing only for tests. Both copies now have the same shape
(`pathContains` + `realHomeContains` + `mustBeOutsideRealHome` + `sandboxHome`)
deliberately, so a merge is mechanical. **See §9.1: S8 makes it three.**

---

## 5. The rename, and confirmation nothing wrote the old namespace first

**Three surfaces carried the sibling repository's namespace, not one.** The plan
names the schema constant; the root is the same for all three and none had ever
been written, so all three are renamed and the decision is flagged here as
slightly beyond the plan's literal sentence.

| | draft | landed |
|---|---|---|
| spool schema constant | *`<sibling>/spool/v1`* | `skill-architect/spool/v1` |
| default spool directory | `~/.<sibling>/spool` | `~/.skill-architect/spool` |
| strict-mode env var | `<SIBLING>_STRICT` | dropped — `--strict` is the flag; no env var ships |

`~/.cursor` is **not** renamed, and must not be: that is Cursor's own directory,
which is where the hooks file has to go.

**Confirmation that nothing ever wrote the old one:**

| check | result |
|---|---|
| `git log --all --oneline -S '<sibling>/spool'` | **one commit: `a3673b8` "archive: the 0.5.0 working tree, as frozen before any slice was cut"** — the archive branch, never `main` |
| `git ls-tree -r --name-only origin/main \| grep -iE "hooks\|spool"` | empty — no spool file has ever been on `main` |
| `git grep -c "<sibling>" origin/main` | absent |
| any file written by this code | none existed before this commit; there is no spool writer on `main` |

So the rename is free: **no migration, because no file with the old namespace was
ever produced by anything that shipped.**

**Pinned against reintroduction.** `TestSpoolSchema_NamesThisProduct` asserts the
constant *and* searches every `.go` file in the package for the three spellings
of the old namespace. Two notes on it:

- The needles are **assembled at runtime** rather than written as literals,
  because this file is one of the files searched — spelling them would make the
  check fail on itself, which is a check satisfiable only by deleting it. It did
  fail on itself first, on a comment of mine that quoted the old name; the
  comment was reworded and the old spelling now appears nowhere in the package.
- M1 (schema keeps the old namespace) and M2 (spool dir keeps it) both kill.

**The old namespace is still live in 5 untracked draft files** in the primary
working tree — `doctor_test.go`, `compare_test.go`, `queries/subagent_usage.sql`
and the two I replaced. **`doctor_test.go` and `queries/` are S8's inputs**, so
S8 inherits the same rename. §9.2.

---

## 6. Mutation proofs — 91 mutations, 11 harness proofs, a verdict in every direction

S5's and S6's rule, carried: **any check whose pass signal is the absence of
output can pass by not running.** The runner prints one of five verdicts and
never leaves silence as the answer.

### 6.1 The runner was proved in six directions before any result was trusted

| harness proof | expected | got |
|---|---|---|
| H0/H4 an edit that does not apply | NOT APPLIED | **NOT APPLIED** |
| H1 a behaviour change | KILLED | **KILLED — 5** |
| H2 a reworded comment | SURVIVED | **SURVIVED** |
| H3 an undefined symbol | DID NOT BUILD | **DID NOT BUILD** |
| H5c an unconditional infinite loop | DID NOT END | **DID NOT END** |

Run twice — once on the barrier alone, once on the final file set.

### 6.2 A fifth way a check yields no verdict, found the hard way

**M15 (a failed read still writes a line) never terminated.** The mutation
removed `Ingest`'s early return, so a drained `SliceSource` stopped reporting
`io.EOF` — and my own `TestIngest_ReplaysACorpusInOrderAndStopsAtTheEnd` drained
it with `for { … }`. The run sat for **ten minutes** and reported
`DID NOT RUN — nonzero exit with no test failure` only because Go's own default
timeout eventually panicked.

Two repairs, because there were two faults:

1. **The test.** An unbounded `for {}` turns a regression into a *hang* rather
   than a failure — a verdict nobody gets. It now allows 8 reads and fails
   explicitly: "a drained source never reported io.EOF … a caller's loop over it
   would not end". M15b and M20 both kill it.
2. **The runner.** `-timeout 90s` plus a `DID NOT END` verdict, proved by H5c.

This is S6's generalisation extended: **not only "the absence of output", but
"the absence of an ending".**

### 6.3 The capture layer — `hooks.go`

| # | Mutation | Result |
|---|---|---|
| M1 | the spool schema keeps the sibling repo's namespace | KILLED |
| M2 | the default spool dir keeps the sibling repo's name | KILLED — 2 |
| **M3** | **`UseNumber` dropped (numbers through `float64`)** | **KILLED — 3** |
| M4 | trailing content accepted as one document | KILLED |
| **M5** | **redaction skipped for anything not an object** | **KILLED** |
| M6 | a payload that is not JSON is dropped instead of captured | KILLED |
| M7 | an unknown event name mapped into the known set | KILLED — 6 |
| M8 | the strict flag is not recorded on the line | KILLED — 4 |
| M9 | strict mode strips metadata too | KILLED — 4 |
| M10 | strict mode strips nothing | KILLED — 4 |
| M11 | an event with no capture time is filed anyway | KILLED |
| M12 | the spool file is truncated rather than appended to | KILLED — 5 |
| M13 | the spool file is world-readable | KILLED — 2 |
| M14 | the spool directory is world-readable | KILLED — 2 |
| M15 | a failed read still writes a line | **DID NOT END** — §6.2 |
| M15b | the same, test bounded | KILLED — 2 |
| M16 | an append error is swallowed | KILLED — 2 |
| M17 | a credential-named field survives | KILLED — 2 |
| M18 | a credential-shaped value survives | KILLED — 3 |
| M19 | a measurement named for tokens is redacted as a credential | KILLED — 2 |
| M20 | a drained source keeps returning events instead of `io.EOF` | KILLED |

### 6.4 Registration — `hooks_install.go`

| # | Mutation | Result |
|---|---|---|
| **M21** | **install is not idempotent (registers again every run)** | **KILLED — 7** |
| **M22** | **install replaces the event list instead of appending** | **KILLED — 8** |
| M23 | install rebuilds the document from only what it knows | KILLED — 14 |
| M24 | a no-op install still takes a backup | KILLED — 2 |
| M25 | uninstall takes no backup | KILLED — 2 |
| M26 | uninstall removes every entry, not only ours | KILLED — 7 |
| M27 | uninstall leaves an empty event behind | KILLED — 4 |
| M28 | an unparseable `hooks.json` is overwritten instead of refused | KILLED — 6 |
| M29 | the refusal does not name the file | KILLED — 6 |
| M30 | a read error that is not a missing file reads as a missing file | KILLED |
| M31 | a backup that failed does not stop the write | KILLED |
| M32 | `hooks.json` is written world-readable | KILLED — 2 |
| **M33** | **an entry's command matched by rendering, not by type** | **SURVIVED — a real gap. §6.7** |
| M33b | the same, after the test was added | KILLED |

### 6.5 The reader — `spool_profile.go`

| # | Mutation | Result |
|---|---|---|
| **M34** | **an empty session id is accepted** | **KILLED** |
| M35b | the refusal returns the events anyway | KILLED |
| M36 | a missing spool directory reads as an empty one | KILLED |
| M37 | every file in the directory is read as a spool file | KILLED |
| M38 | subdirectories are read as spool files | KILLED |
| **M39** | **files read in directory order rather than sorted** | **SURVIVED — the code was redundant. §6.7** |
| M40 | events are not ordered by capture time | KILLED |
| M41 | a line that cannot be decoded aborts the file | KILLED |
| M42 | the session is matched on the envelope only | KILLED |
| **M43** | **a `raw` that is not an object takes the read down** | **SURVIVED — the branch was dead. §6.7** |
| M43b | a nil `raw` read as naming the session (after the repair) | KILLED — 3 |
| M44 | the session id is matched loosely (prefix) | KILLED |

### 6.6 The CLI — `cmd/main.go` and its tests

| # | Mutation | Result |
|---|---|---|
| M46 | `ingest` dispatched and left out of the help | KILLED |
| M47 | `hooks` in the help and not dispatched | KILLED — 8 |
| M48 | a hooks subcommand dispatched and left out of its help | KILLED |
| M49 | a hooks subcommand in its help and not dispatched | KILLED — 7 |
| M50 | `ingest` echoes the payload back to stdout | KILLED |
| M51 | an `ingest` that failed exits 0 | KILLED |
| M52b | `--strict` is not passed through | KILLED |
| M53b/c | `--spool-dir` ignored, the default always used | KILLED — 3 / 5 |
| **M54b** | **`--home` ignored, the real home always used** | **SURVIVED — a real gap. §6.7** |
| M54c | the same, after the two homes were separated | KILLED — 2 |
| M55b | the registered command is not this binary's path | KILLED — 2 |
| M56 | a hooks error exits 0 with a result | KILLED — 3 |
| M57 | an unknown hooks subcommand treated as install | KILLED — 2 |
| M58 | `hooks` with no subcommand exits 0 | KILLED — 2 |
| **M59** | **cmd barrier: prefix test instead of `filepath.Rel`** | **SURVIVED — a real gap. §6.7** |
| M59b | the same, after the negative test was added | KILLED — 2 |
| M60 | `runCLI` does not override `HOME` | KILLED — 2 |
| M61 | a test starts the binary outside the sandboxed runner | KILLED |
| **M62** | **the exec.Command derivation allows every function** | **SURVIVED by construction. §4.5** |
| M63 | the allowance names a function that is gone | KILLED |
| M64 | cmd barrier: resolution dropped | KILLED — 2 |
| **M65** | **cmd barrier: an unresolvable path read as outside** | **SURVIVED — a real gap. §6.7** |
| M65b | the same, after the refusal test was added | KILLED |
| **M66** | **cmd barrier: `Fatalf` → `Errorf` (the shipped defect)** | **SURVIVED — a real gap. §6.7** |
| M66b | the same, after the narrow interface was added | **KILLED BY THE COMPILER** |
| **M67** | **`sandboxHome` stops calling the barrier** | **SURVIVED by construction. §4.5** |
| M68 | `runCLIIn` does not check the home it was handed | SURVIVED → M68b |
| M68b | the same, after the derivation was added | KILLED |
| M69 | **control:** the guard derivation names a missing function | KILLED |
| MB1/MB1b | barrier: prefix test instead of `filepath.Rel` | KILLED — 2 |
| MB2/MB2b | barrier: resolution dropped | KILLED — 3 |
| MB3 | the barrier checks nothing | KILLED |
| **MB4** | **an unresolvable path read as outside the home** | **SURVIVED — a vacuous skip of mine. §6.7** |
| MB4b | the same, after the test was repaired | KILLED |
| MB5 | barrier: `Fatalf` → `Errorf` | **KILLED BY THE COMPILER** |

### 6.7 The nine survivals that were real gaps — six in my own tests

This is the sixth slice in a row to catch its own vacuous checks, and the first
where the *majority* of the survivals were in test code rather than production
code.

| # | What it exposed | Repair |
|---|---|---|
| **MB4** | **A vacuous `t.Skip`.** The refusal test skipped when `realHomeContains` returned `err == nil` — i.e. the skip was decided **by the function under test**. The mutation that broke the function made the test *skip* rather than fail. | Reachability is now asked of the **filesystem** (`filepath.EvalSymlinks` directly), never of the code being tested. |
| **M54b** | **`--home` was indistinguishable from `$HOME`.** Every case ran with `HOME` pointed at the same directory it passed as `--home` — which is what keeps the suite off the real home, and means none of them could tell whether the flag was read at all. Harmless in the suite; in a user's hands it writes `~/.cursor` while the flag names somewhere else. | `TestTheFlagWinsOverTheEnvironment` gives the two **different** sandboxes and asserts which was written. Both are sandboxes, so the separation costs no safety. |
| **M59** | The cmd copy of the barrier had **no negative test**: degrading `filepath.Rel` to `strings.HasPrefix` left the suite green. Correct code, unproven mechanism. | `pathContains` split out and given the symlink and sibling cases, as in `profiler`. |
| **M65** | The cmd copy lacked the **refuse-on-unresolvable** direction the `profiler` copy had. | Added, with the reachability decided by the filesystem. |
| **M66** | The cmd copy took `*testing.T`, so `Errorf` compiled — the exact shipped defect, writable again. | Narrow `fatalTB`. M66b no longer compiles. |
| **M68** | The load-bearing guard could be deleted with nothing observable changing. | Derived from the AST (§4.4); control M69. |
| **M33** | The registered command was matched without asserting its **type**. An entry whose `command` is a number that prints the same is not the same entry — reachable, since `--command` takes whatever the caller gives it. | `TestHooks_MatchTheCommandByTypeAndNotByItsRendering`. |
| **M39** | Not a test gap — **redundant code.** `sort.Strings(names)` cannot change anything: `os.ReadDir` already returns entries sorted by filename. | **Removed**, and the dependency on `ReadDir`'s ordering written down instead of claimed twice. |
| **M43** | Not a test gap — a **dead branch.** `sessionMatches`'s `return false` on a failed decode is unreachable in effect, because the only caller refuses an empty id and a nil map's fields read as `""`, matching nothing. | **Branch removed**, the dependency on the caller's refusal documented, and `TestSessionMatches_AnEventThatNamesNoSessionBelongsToNone` added to pin the invariant at that layer. `sessionMatches` went 83.3% → 100%. |

### 6.8 The unused-variable trap fired six times

M15 (first form), M35, M52, M53, M54, M55 all made a variable unused and read as
`DID NOT BUILD` — the mechanism that gave S5 a false green. Because the verdict
is printed in every direction they were visible and each was re-run in a
compiling form (M15b, M35b, M52b, M53b, M54b, M55b). **Five of the six then
killed; the sixth is M54b, the real gap above.**

### 6.9 Restores

`cp` from `<scratch>/s7-backup`. **No `checkout`, `reset`, `clean`, `stash` or
`rebase` was run anywhere**, across all 102 runs. All eight mutated files were
diffed against the backup afterwards: **byte-identical, all eight.** (The
`reset: moving to HEAD` in the worktree's reflog is `git worktree add`'s own
internal reset at creation, not a command I ran.)

---

## 7. What I inherited from the snapshot, and what I had to change

Hash-checked against `snapshot-0.5.0.manifest` before reading — **all six
matched**, and nothing was copied:

| file | sha256 (head) |
|---|---|
| `hooks.go` | `179c5735…491c` |
| `hooks_install.go` | `a5390308…4330` |
| `spool_profile.go` | `01ea0345…f26e` |
| `hooks_test.go` | `b7fa2df8…d1e1` |
| `hooks_install_test.go` | `113602e8…828b` |
| `spool_profile_test.go` | `d9644797…8c1` |

### 7.1 Measured what I inherited, rather than assuming it worked

Per the "read for intent, port by hand, measure" instruction, I ran the draft's
two relevant test files against the draft's `hooks.go` and `hooks_install.go` in
a scratch module (`<scratch>/s7-draftcov`), with a stub for the missing
`getString`:

**Draft coverage: 81.7%.** And the shape of the gap is the finding:

| draft function | coverage |
|---|---|
| `NormalizeHookPayloadStrict` | **0.0%** |
| `stripContent` | **0.0%** |
| `DefaultSpoolDir` | **0.0%** |
| `writeHooksDoc` | 66.7% |
| `readHooksDoc` | 77.8% |
| `AppendSpool` | 78.6% |

**The draft shipped its privacy control with zero tests.** Strict mode — "the
metadata-only mode for shared machines" — had not one test, and neither did the
function that names the directory everything is written to. A privacy control
nobody exercised is the same defect class as a capability claim nothing
implements, and it is the third slice running to find that shape in the draft
(S6: not one refusal path executed at 76.9%).

**Landed: 97.5% for the package, with strict mode tested in four directions**
(the flag recorded, content gone, metadata kept, envelope unaffected) and
`DefaultSpoolDir` pinned to the product's own name.

### 7.2 Eight changes, by hand

| What the draft does | Why it could not be ported | What landed |
|---|---|---|
| decodes into `map[string]any` and re-encodes | **numbers go through `float64`**; an integer above 2⁵³ is silently rounded, and a nanosecond timestamp is 1.7×10¹⁸. Makes "lossless" — the reason this ships — false | `json.Decoder` with `UseNumber()`, and `More()` so trailing content is kept whole rather than half-read |
| non-object JSON falls to the "not JSON" path | **and that path applies no redaction**, so a credential inside a JSON array reached the spool in the clear | one decode for every shape; redaction over all of them; envelope fields promoted only from an object |
| backup taken before deciding whether to write | every rerun of `hooks install` dropped another `.bak` and reported a backup for a write that never happened | the merge is computed first; `saveHooksDoc` is reached only when something changed |
| `UninstallHooks` writes with no backup | the **more destructive** of the two operations was the one with nothing to go back to | one `saveHooksDoc`, so the two cannot disagree about when a backup is taken |
| `AppendSpool` files an event with no `Ts` as `.jsonl` | invisible to every reader listing the spool by date | refused, `errNoCaptureTime`, and nothing written |
| `em["command"] == command` inline, twice | two copies of a type-sensitive comparison | one `entryCommand`, answering `""` for any shape this build cannot interpret — which is how a foreign entry is left alone rather than removed |
| strict lines indistinguishable from whole ones | a reader pooling a spool cannot tell what it has | `strict` on the envelope |
| `profileFromSpool` and the inference layer | §3.1 | **not landed** |

Also dropped: `CURSOR_PROFILER_STRICT` (§5) and `flag.ExitOnError` in the
subcommands, which exits **2** on a flag typo where 2 means "nothing compared" —
S5's and S6's `parseFlags`/`ContinueOnError` is used instead.

### 7.3 Integrations rather than additions, in the files I touched

- **`runCLI` became a sandboxing runner and absorbed five call sites** (§4.4).
  That removed three hand-rolled copies of stream capture and exit-code reading.
- **`caseLiteralsIn` and `commandsUnder`** — S6's derivations — were **reused**
  for `cmdHooks` rather than a third variant being written. `hooksUsage` prints
  a `subcommands:` block in the same shape `experiment` does, so
  `commandsUnder(help, "subcommands:")` works unchanged.
- **`hooksFlags`** resolves `--home` and `--command` once for both subcommands,
  so the pair cannot come to disagree about what `--home` defaults to — which is
  the resolution that decides whether a run writes into a user's real config.
- **`getString`** is one helper in `hooks.go` used by both the writer and the
  reader, rather than the draft's assumption that it existed elsewhere.

---

## 8. Verification — measured, not carried

### 8.1 Shell suites, both shells

| suite | base `d77d90e` | head, bash 3.2.57 | head, bash 5.3.15 |
|---|---|---|---|
| test_f01 | 1868 | 1868 | 1868 |
| test_f02 | 350 | 350 | 350 |
| test_harness | 131 | 131 | 131 |
| test_install | 48 | 48 | 48 |
| test_rewrite | 85 | 85 | 85 |
| test_skill | 53 | 53 | 53 |
| test_walk | 20 | 20 | 20 |
| **total** | **2555** | **2555 passed, 0 failed** | **2555 passed, 0 failed** |

Unchanged and expected: the slice touches no shell surface. `test_harness.sh`
names `README.md` only in its stageability list, and `test_f01`'s README
assertions are about jq tooling.

### 8.2 Go, in `profiler/`

| gate | base `d77d90e` | head |
|---|---|---|
| `gofmt -l .` | empty | empty |
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `go test -race -count=1 ./...` | ok | **ok**, both packages |
| **top-level tests** | **177** | **237** (+60) |
| PASS incl. subtests | 754 | **876** (+122) |
| failures | 0 | **0** |
| skips | 0 | **1** (the barrier child, by design) |
| `profiler` coverage | 97.8% | **97.5%** |
| `profiler/cmd` coverage | 94.5% | **93.7%** |

**The plan's "175 top-level Go tests" was 177 at `d77d90e`** — re-measured, not
carried, as instructed. S6's record says 175 and 752; the tree says 177 and 754.

**Coverage moved down slightly and I am not hiding it.** Both drops are
unreachable-by-construction error branches in new code: `json.Marshal` of a
value just decoded from JSON cannot fail (3 branches), and `os.UserHomeDir()` /
`os.Executable()` / `DefaultSpoolDir()` failing has no reachable trigger on this
platform (4 branches). Every branch a test *can* reach is covered:
`hooks_install.go` is 100% on both exported functions, `sessionMatches` is 100%,
and `stripContent` went from the draft's 0% to 100%.

S6's `-cover` instrumentation of the subprocess (the `GOCOVERDIR` check in
`buildProfiler`) is untouched and still working — 93.7% is the honest figure,
not the 8.5%-style artifact. **L21's 12.6% in the deferred ledger is still
wrong** (S6 §9.3 stands).

### 8.3 The out-of-scope barrier

`tests/lib/out-of-scope-check.sh` over the staged index: `out-of-scope check:
clean`, exit 0 — the committed script, run before the commit.

**Proved non-vacuous against this index**: staging `skillgate/probe.md` gave
`BLOCKED: out-of-scope path staged`, exit 1, naming it; unstaged and removed, the
check returned clean and the index was back to exactly ten paths.

### 8.4 Live, against the built binary, in a sandbox home

`HOME=<scratch>/s7-demo/home`, binary built fresh:

| run | result |
|---|---|
| `hooks install` with no `--home` | 21 events registered under the sandbox home; `hooks_json` names it |
| `hooks install` again | `events_registered: 0`, one entry per event, **0 backup files** |
| three `ingest` runs, one with a credential, one not JSON | 3 lines, one daily file |
| the credential | **0 occurrences** of the key or the bearer token |
| the command around the credential | survived: `curl -H "Authorization: [REDACTED]" https://x` |
| the nanosecond integer | **exact**: `1758378600123456789` present |
| a foreign hook + an unknown top-level field, then `uninstall` | foreign entry intact, `aFutureTopLevelField` intact, only `sessionStart` left, one backup taken |
| an unparseable `hooks.json`, then `install` | exit **1**, the path named on stderr, **the file byte-identical afterwards** |
| the real home, after all of it | `~/.cursor` and `~/.skill-architect` both **absent** |

### 8.5 Hygiene

- `git add` was the plan's **ten named paths, one at a time**. No `git add -A`,
  no `-f`, no `commit -a`. **No file was forced outside the plan's list.**
- Author and committer `imagineux <imagineux@gmail.com>`. AI-attribution grep
  over the whole commit object and over the live PR title and body: **0 matches
  each**.
- **No version surface touched.** `AdapterVersion` stays `0.5.0`,
  `ProfileSchema` stays `skill-architect/profile/v1`, the five plugin manifests
  stay `0.4.3`. `skill-architect/spool/v1` is a new document, not a bump.
- No `checkout`, `reset`, `clean`, `stash` or `rebase` in any tree, ever.
- **The primary working tree was never written to.** Re-checked after the push:
  still `0a83615`, 47 `git status --porcelain` lines — the figure S3, S4, S5 and
  S6 all recorded.
- The snapshot was read-only and hash-checked; nothing written to it, nothing
  copied from it.

---

## 9. Findings — what should change before slice eight

### 9.1 The home barrier is about to exist three times. **[decide — S8]**

`doctor`'s `DetectEnvironment(home, getenv)` takes a home, so S8's tests need the
same barrier, and there is no third package to put it in either. **Recommendation:
lift it into `profiler/internal/homesafe` before S8 writes its tests** — one
package, one set of tests, `internal/` bounding the promise so no external
consumer can bind to it.

I did **not** do it here: it means new files outside the plan's `git add` list,
which is scope I was told to surface rather than take. Both copies are now
deliberately the same shape (`pathContains`, `realHomeContains`,
`mustBeOutsideRealHome`, `sandboxHome`, `fatalTB`), so the merge is mechanical.

**It is ~90 lines duplicated and S8 makes it ~135.** That is the moment it stops
being cheap.

### 9.2 S8's inputs still carry the sibling repo's namespace. **[S8's, flagged]**

`git grep` over the primary working tree finds it live in `profiler/doctor_test.go`
and `profiler/queries/subagent_usage.sql` — both S8 inputs. §5's test only
guards `profiler/*.go`, so **the SQL file would slip past it.** S8 should either
widen the check to `queries/` or rename as it ports.

### 9.3 `spool_profile.go` now holds no profile. **[decide — S8 or the adapter slice]**

§3.1. Renaming it to `spool_read.go` was outside the plan's `git add` list so I
left it. Whichever slice touches it next should rename the pair or restore the
projection.

### 9.4 `doctor`'s `EnvironmentTier` is the next capability claim. **[S8, per the plan]**

The plan already says this (S8's "integration requirement"), and this slice makes
it sharper: **a tier `doctor` reports as reachable must be a tier from which a
capture actually produces a `present` signal — and after this slice, the hook
tier produces no `present` signal at all**, because there is no adapter. So
`doctor` must not report the hook tier as delivering `tokens`, `tool_calls` or
anything else. It can report that a spool exists and has lines in it. That is a
true and useful thing to say; "this machine can capture tokens from Cursor" is
not.

### 9.5 `AppendSpool` has no locking, and Cursor may fire hooks concurrently. **[note]**

One `os.File.Write` of the whole line under `O_APPEND`. POSIX guarantees the
offset update is atomic, and a single `write(2)` of a line is not split in
practice on APFS or ext4 — but it is not *guaranteed* for a regular file, and a
line longer than the FS's atomic write size could in principle interleave with
another hook's. Not fixed: it is new behaviour, unmeasurable without a running
Cursor, and a corrupted line is skipped by the reader rather than losing the
file (M41). **Worth a line in the release notes; worth real locking if anyone
ever observes an interleaved line.**

### 9.6 Two backups in the same second collide. **[note]**

`hooks.json.bak-<YYYYMMDDTHHMMSSZ>` is second-resolution, so
install→uninstall→install inside one second overwrites the first backup. Reaching
it needs three runs in a second; the idempotence reorder (§7.2) means a plain
rerun takes no backup at all, which removes the common case. Not fixed because
the fix is either a counter or sub-second precision, and neither is this slice's.

### 9.7 S6 §9.1's step timeout is still open, and now has a sibling. **[unchanged]**

`RunPlan` shells out with no wall clock. Untouched by this slice.

### 9.8 Smaller notes

- **`DID NOT BUILD` needs two labels**, not one. S6's rule that a non-compiling
  mutant proves nothing is right for the unused-variable trap (which fired six
  times here) and wrong when the compiler error *is* the kill — MB5 and M66b are
  proofs that a defect is unrepresentable. §4.2 distinguishes them; a future
  runner should print the distinction itself rather than leaving it to the
  writer.
- **A mutation of an assertion is unkillable by that assertion** (M62), and
  **removing a guard that never fires changes nothing observable** (M67). Both
  are properties of testing, not findings about this slice, but a reader tallying
  survivals will hit them — so they are named in §4.5 rather than quietly
  excluded from the count.
- The strict-mode env var was dropped rather than renamed. If the metadata-only
  mode needs to be settable without editing the registered command, the flag has
  to come back as an env var — and the registered command is written by
  `hooks install`, which could take `--strict` and bake it in. Not added: new
  surface, and nobody asked.
- `CursorHookEvents` is a `var`, not a `const` slice, so a caller can mutate it.
  Left alone; changing it to a function returning a copy is a boundary decision
  the adapter slice can take when something depends on it.

---

## 10. PR

[#19](https://github.com/Okja-Engineering/skill-architect/pull/19), open,
**not draft**, base `main`, head `feat(profiler)/0.5.0-hook-spool` at `5c84911`.
CI: `gh pr checks 19` -> **pass, 2m33s** (ubuntu-latest, bash 5, Linux git).
AI-attribution grep over the live PR title and body: **0 matches**.

---

## 11. Not done, and why

| Not done | Why |
|---|---|
| Merging or tagging | The user merges. Nothing tagged. |
| A `CursorAdapter` | §4 of the plan and the manager's ruling. None of the three lands. |
| `profileFromSpool` and the inference layer | §3.1 — the one place the plan turned out to be wrong, named rather than silently done or silently skipped. |
| Renaming `spool_profile.go` | §9.3 — outside the plan's `git add` list. |
| `profiler/internal/homesafe` | §9.1 — new files outside the list; surfaced instead. |
| Touching any version surface | §8.5. The manifests are S10's. |
| `analyze`, `doctor`, `queries/` | S8. |
| `.out-of-scope.md` | Re-read; nothing this slice does makes any of it false. Not staged, unlike S6, which had to. |
| `CHANGELOG.md` / `RELEASE_NOTES.md` | S10's. |
| Locking `AppendSpool` | §9.5 — named, not silently added. |
| Verifying anything against a real Cursor | Impossible here, and the whole point of §1. |
