# S8 record — `analyze` describes the spool, `doctor` reports only what a probe read

**Implementer** senior-implementer · **Date** 2026-09-21 · **Base** `origin/main` = `c104f3c`
**Branch** `feat(profiler)/0.5.0-analyze-doctor` · **Commits** `d540aff`, `5eae640`, `5365a46`
**PR** [#20](https://github.com/Okja-Engineering/skill-architect/pull/20) (open, not draft, base `main`)
**Worktree** `<scratch>/wt-s8-analyze-doctor` (session scratchpad)
**Status** delivered, not merged. The user merges. Nothing tagged.

`<scratch>` = `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad`

---

## 1. The three things S7 handed over, all executed

### 1.1 The barrier was lifted first, and re-proved after the move

`profiler/internal/homesafe` now holds it: `PathContains`, `RealHomeContains`,
`MustBeOutside`, `SandboxHome`, `FatalTB`, `TempDirTB`. One package, one set of
tests. Imported only from `_test.go` files, so it is linked into no shipped
binary; `internal/` bounds the promise as S7 recommended.

**The stronger of the two copies was taken.** The `profiler`-package copy
resolved a path whose leaf does not exist yet (`resolveForBarrier`); the `cmd`
copy used plain `EvalSymlinks` and errored on one. The lifted version keeps the
former, so the barrier can be asked about a directory a test is about to create
rather than only after the thing that creates it has run. That is strictly
stronger: the permission-denied case still errors and still counts as contained.

**One consequence, and it is the S7 trap.** The `cmd` copy's
unresolvable-path test used a *never-created* path. Under the stronger resolver
that path resolves, so the case would have **skipped** — the barrier's proof
passing by not running, for the second slice in a row. The lifted test uses the
sealed-directory (EACCES) form, asks the filesystem whether the case is
reachable rather than asking the function under test, and does not skip on this
machine (verified in a `-v` run).

Every property re-proved after the move, by reverting its mechanism:

| | mutation | verdict |
|---|---|---|
| MB1 | compare the unresolved strings | DID NOT BUILD (unused variable) |
| MB1b | same, compiling form | **KILLED 2** — the symlink case |
| MB2 | containment by `strings.HasPrefix` | **KILLED 2** — the `alice-backup` sibling |
| MB3 | the guard checks nothing | **KILLED 3** |
| MB4 | unresolvable reads as outside the home | **KILLED 1** |
| MB5 | `Fatalf` → `Errorf` | **DID NOT BUILD, and the compiler error is the kill**: `tb.Errorf undefined (type FatalTB has no field or method Errorf)`. The shipped defect — a guard that reported and then returned, letting the install run — is unrepresentable |
| MB6 | `SandboxHome` stops asking the barrier | **KILLED 1** |
| MB7 | resolution skips `EvalSymlinks` | **KILLED 3** |
| MB8 | aborts on the error but not on the match | **KILLED 3** |
| MB9 | `runCLIIn` stops checking its home | **KILLED 1** |
| MB10 | the AST derivation matches loosely | **KILLED 1** |
| MB11 | a home that cannot be named reads as outside | **KILLED 2** |
| MB12 | the parent's resolve error is ignored | **SURVIVED — equivalent** (§4.2) |

**Two are stronger than before the move, and that is the point of having lifted
it.** S7's M67 (`sandboxHome` stops calling the barrier) **survived**, because a
guard that never fires in a healthy tree changes nothing observable when
removed. Driving `SandboxHome` with a recorder whose `TempDir()` returns the real
home makes the call observable, and MB6 now dies. The same recorder gives MB3 an
in-process half; the old copies could only watch the abort from a child
`go test`.

**The AST derivation went red on the move, which is the check working.**
`callsFunc(file, "runCLIIn", "mustBeOutsideRealHome")` matched a bare identifier
and the call is now `homesafe.MustBeOutside` — a selector. It failed rather than
silently passing only because it was already looking for a call it could name.
`calleeName` now compares the whole spelling (`pkg.Helper`, not the trailing
word, which would match any package's function of that name), and a control
asserts a spelling nothing calls comes back false. MB10 proves the loose form
dies.

**One incidental repair.** The child-process proof runs `go test` with `HOME`
overridden, and `GOCACHE` derives from `HOME` — so the child was building the
standard library into a cold cache inside a temp directory to prove a path
comparison. It now inherits this run's `GOCACHE` and takes 0.37s. This is a
plausible cause of S7's "one mutation that never terminated".

Files beyond S8's `git add` list, named rather than added quietly, all forced by
the lift: `profiler/internal/homesafe/homesafe.go` and `homesafe_test.go` (the
manager authorised these), `profiler/hooks_test.go`, `profiler/hooks_install_test.go`.

### 1.2 The renames, and the widened check

**The queries carried the sibling namespace in all eight files**, and S5's rename
guard only read `*.go` — so all eight would have slipped past it. Confirmed by
watching it fail: the widened check named every one of `README.md`,
`conversations_per_day.sql`, `estimated_tokens.sql`, `events_per_day.sql`,
`model_mix.sql`, `subagent_usage.sql`, `time_of_day.sql`, `tool_call_mix.sql`.

`TestSpoolSchema_NamesThisProduct` now reads this package's `*.go` **and
everything under `queries/`**, and `filesNamingTheSpool` fatals if either group
is empty — reading nothing is how a search for an absent string passes without
running. The needles are still assembled at runtime, because `hooks_test.go` is
one of the files searched.

`doctor_test.go` did not need renaming in the end: it was rewritten from scratch
(§3), and `doctor.go`'s own `.cursor-profiler` spool path went with it.

### 1.3 The tier claim, sharpened to what the release can demonstrate

S7's ruling: after the hook spool there is no adapter reading a spool, so **the
hook tier produces no `present` signal at all.** That is now the shape of the
code rather than a note in it. §3 has the exact wording of every claim.

---

## 2. What landed

15 files, 3 commits. Six are new.

| File | What |
|---|---|
| `profiler/internal/homesafe/homesafe.go` | **new** — the barrier, lifted |
| `profiler/internal/homesafe/homesafe_test.go` | **new** — its proofs, re-run after the move |
| `profiler/analyze.go` | **new** — `SpoolSummary`, `AnalyzeSpool` |
| `profiler/analyze_test.go` | **new** — the summary, and the query assertions |
| `profiler/doctor.go` | **new** — `EnvironmentQuery`, `EnvironmentReport`, `DetectEnvironment` |
| `profiler/doctor_test.go` | **new** — what doctor may and may not say |
| `profiler/queries/` | 7 SQL + README: renamed, corrected, one dropped, one added |
| `profiler/spool_profile.go` | `forEachSpoolLine`, shared with `LoadSessionEvents` |
| `profiler/adapter_contract_test.go` | rule 4: the tier table |
| `profiler/hooks_test.go` | barrier removed; namespace check widened |
| `profiler/hooks_install_test.go` | `sandboxHome` → `homesafe.SandboxHome` |
| `profiler/cmd/main.go` | `analyze`, `doctor`, three shared resolvers, usage |
| `profiler/cmd/main_test.go` | barrier removed; the CLI half; two derivations |
| `docs/profiler-spec.md` | `## Summarising a spool`, `## Reporting what a machine can measure` |
| `README.md` | "Reading a spool back", "What can this machine measure?" |

Files **forced beyond the plan's `git add` list**, each named here:
`profiler/internal/homesafe/*` (authorised by the manager),
`profiler/hooks_test.go`, `profiler/hooks_install_test.go` (the barrier lift),
`profiler/spool_profile.go` (the shared spool walk),
`profiler/adapter_contract_test.go` (the plan's own integration requirement says
to extend S2's contract table, which lives there).

`git add` was those paths, one at a time. No `git add -A`, no `-f`, no
`commit -a`. `tests/lib/out-of-scope-check.sh` before every commit: **clean**,
exit 0, three times.

---

## 3. The exact wording of every claim, and which are demonstrable

### 3.1 The tiers

| tier | the claim, verbatim from `doctor.go` | demonstrable? |
|---|---|---|
| `none` | "nothing was read that this build can turn into a profile" | **yes** — proved over an empty home with no export |
| `export` | "a registered harness probed the export supplied and reports at least one signal from a real source" | **yes** — proved present over `testdata/otlp/full_export.ndjson`, the adapter contract's own fixture, and the capture of that same export delivers four `present` signals |

That is the whole enumeration. The draft's `hooks`, `server_api` and
`enterprise` are gone.

**The mechanism that keeps it honest is structural, not a rule.** `measure()`
takes a harness name and a path. It cannot see the home, the spool or the
environment, so there is nothing available to it from which a tier could be
raised. The only input to a tier is a `CapabilityReport` from a probe over a
real file.

### 3.2 The four reasons, verbatim

- no export: *"no export was supplied, so no harness was asked to read
  anything. A measurement surface is what a probe read: pass --harness and
  --otel-file to have one read, or run `profiler capture --harness <name>
  --otel-file <path>` to produce a profile"*
- export, no harness: *"an export was supplied and no harness was named, so
  nobody was asked to read %s (supported: %s)"*
- unknown harness: *"there is no harness named %q (supported: %s)"*
- read nothing: *"%s read %s and reports no signal from a real source, so there
  is nothing here to measure with"*
- read something: *"%s read %s and reports %s from a real source"* — live:
  *"claude_code read …/full_export.ndjson and reports 4 signals
  (skill_activation, timing, tokens, tool_calls) from a real source"*

The harness set in all of them comes from `SupportedHarnesses()`, so a refusal
cannot name a set the CLI does not accept.

### 3.3 The spool sentence — the one the manager's ruling is about

Shipped in the data, on **every** spool observation, as
`SpoolObservation.Yields`:

> no measurement: no adapter in this build reads the spool, so these lines are a
> capture to be parsed later and no profile, token count or tool call can be
> produced from them

Chosen against three alternatives:

- *"hooks capture is available"* — false in the way that matters. Capture is
  available; measurement is not, and "available" is the word a reader turns into
  the latter.
- *"the spool is not yet supported"* — false the other way. It is supported: the
  lines are written, redacted, readable back and summarised. What is absent is
  an adapter, not the capture.
- a `reason` on a `hooks` tier — the bolt-on. A tier is the claim; a caveat
  beside it is the shape this release has rejected four times.

It is a **field and not a comment** because a stored report travels: it is
pasted into issues and quoted months later by readers who never see the source.
`TestSpoolYields_IsTheSentenceTheSpecQuotes` pins it to the sentence
`docs/profiler-spec.md` quotes, the way `fallbackReasonInSpec` already is.

### 3.4 What is reported as an observation, and what was dropped

Reported: `hooks_json` (path, present, unreadable-with-reason, the sorted events
carrying **exactly** our command) and `spool` (dir, exists,
unreadable-with-reason, files, lines, unreadable_lines, payload_bytes,
last_capture_at, yields).

**Three things the draft reported and this does not**, each named in `doctor.go`
and in the spec:

- the Cursor user-data directory, reported as `cursor_installed`. The path is
  macOS's only, so `false` means "not on a Mac" as often as "no Cursor" — and
  whether Cursor is installed is not a statement about what can be measured;
- `CURSOR_ADMIN_API_KEY` as a `server_api` tier. There is no Admin API client in
  this repository;
- `OTEL_EXPORTER_OTLP_ENDPOINT` as an `enterprise` tier. An endpoint is where a
  harness sends telemetry, not a file this tool can read.

`getenv` left the signature with them. Nothing in this build reads either
surface, so the parameter existed only to produce those two claims.

### 3.5 The registration, matched the way the install writes it

The draft string-matched `hooks.json` for `"profiler ingest"`. Two defects in
one line: the command `hooks install` registers is the binary's absolute path
plus `ingest || true`, so a real registration would have been missed; and a
foreign hook whose command merely mentioned us would have counted.

`observeHooksJSON` asks `readHooksDoc` and `hasCommand`/`entryCommand` — the
same reader and the same match `InstallHooks` and `UninstallHooks` use — so
"our entry" has one definition. The CLI resolves that command once, in
`resolveHookCommand`, for `hooks install`, `hooks uninstall` and `doctor`.

**A mutation proved this needed a derivation.** MC3 (drop the `|| true`)
**survived the whole suite**: both halves stayed self-consistent, because they
share the function. So `TestDoctorLooksForTheCommandHooksInstallRegisters`
derives the shared call from `main.go` with a control, and S7's assertion on the
registered command now checks the whole string rather than a prefix. MC3b and
MC5 both kill.

---

## 4. `analyze`: what it describes, and how the inference was kept out of the data

### 4.1 It makes no inference. The guard is the type, not the prose

`SpoolSummary` carries counts and maps of counts. It has **no** `MetricState`,
`MetricSource`, `RawMetricResult` or signal result of any kind, and
`TestSpoolSummary_MakesNoSignalClaim` walks the struct by reflection — through
pointers, slices and maps — and refuses eleven types by name. A later slice
cannot put `PresentToolCallResult(...)` in it without turning that red.

That is the mechanism S7 asked for: the ruling was that a present-claim
constructor makes the assertion *in the data*, where a prose limit cannot reach
it. So the limit is in the data too.

### 4.2 What the draft claimed, and what replaced it

The draft had four hand-guessed accumulators — `tool_calls` from `tool_name`,
`models` from `model`, `mcp_servers` from `mcp_server_name`, `subagent_runs`
from `subagent_type` — plus `skill_activations` inferred from a `SKILL.md` read
via `skillPathRe`, `estimated_tokens` as payload bytes over four, and
`last_context_tokens` read from a `context_tokens` field. It also did not
compile against `main`: `toInt` does not exist there.

Each accumulator reports a field-name guess under the name of a thing the
harness does. If Cursor calls the field something else, the summary says a
session made **no tool calls** rather than that this build could not find the
field.

Replaced by one census: `payload_keys` (how many payloads carried each
top-level key) and `payload_key_values` (their values, bounded). A census of
what is there cannot be wrong that way, and it answers the question this release
actually needs answered — what does a hook payload carry. The live run makes
the point: `payload_keys` came back with `api_key`, `conversation_id`,
`hook_event_name`, `model`, `prompt`, `tool_input`, `tool_name`.

Two bounds on the value census:

- **content keys are counted and never quoted**, asked of the capture layer's
  own `contentKeys` rather than restated, because a summary is a document a user
  pastes into an issue. A prompt is counted; its text is not printed;
- **numbers are not valued.** A histogram of every number in a spool is a table
  of measurements nobody made — the same defect in a different shape. `json.Number`
  is not a `string`, so the type switch excludes them by construction.

`payload_bytes` is bytes. The draft divided them by four and called them
estimated tokens.

The payload is decoded with `decodeOneJSONValue`, the writer's own `UseNumber`
decoder. Nothing here reports a number, but decoding into `any` without it
rounds an integer above 2^53 — the defect S7 repaired in the writer, and it
would have come back in the reader.

### 4.3 What it counts that the draft dropped in silence

`unreadable_lines`. A line that cannot be decoded is skipped — a hook killed
mid-write leaves a truncated last one — but skipping it silently makes the
summary report a smaller corpus than the files hold, with nothing saying so.
`object_payloads` + `other_payloads` = `envelopes`, asserted, so the two always
account for the whole.

### 4.4 One reader of the spool, not three

`forEachSpoolLine` in `spool_profile.go` is now the single decision about what a
spool line is: `*.jsonl` only, no subdirectories, oldest file first, and a
directory that is not there is an **error** rather than an empty spool.
`LoadSessionEvents` had that loop; the draft's `AnalyzeSpool` had a second copy
with different answers, and the draft's `doctor` had a third (it stat'ed every
file for bytes and then re-read each one for a timestamp). `doctor` now goes
through `AnalyzeSpool`.

`spool_profile.go` is still misnamed (S7 §9.3) and I did not rename it: the
manager ruled on three things and this was not one of them. §8.1.

---

## 5. Mutation proofs — 45 runs, a verdict in every direction

S5/S6/S7's rule, carried and extended: **any check whose pass signal is the
absence of output can pass by not running.** The runner prints one of six
verdicts and now **counts skips beside every one of them**, because S7's fifth
way was a case that *skipped* and a skip is silent in a pass/fail tally.

| verdict | runs |
|---|---|
| KILLED | 38 |
| SURVIVED | 1 (equivalent — below) |
| DID NOT BUILD | 4 |
| NOT APPLIED | 1 |
| DID NOT END | 0 |
| SKIPPED | 0 (no run reported a skip other than the barrier child, which is by design) |

Tallies by area: barrier §1.1 (13), `analyze` (10), `doctor` (11), the tier
table (3), the CLI (8).

### 5.1 The one survival, and why it is not a gap

**MB12** — `PathContains` ignores the error from resolving the parent.
**SURVIVED, and behaviourally equivalent.** `resolvedParent` becomes `""`,
`filepath.Rel` then fails on an empty base (verified with a two-line program:
`Rel: can't make /tmp/x relative to `), `PathContains` returns the same error,
and `RealHomeContains` returns the same `contained=true` with the same reason.
There is no observable difference to assert. Named here rather than excluded
from the count, as S7 did with M62 and M67.

### 5.2 Four in my own tests were real gaps, found by mutation

- **MA5** payload bytes divided by four **survived**: the assertion was `> 0`,
  which cannot tell a size from a quarter of one — exactly the difference
  between bytes and the draft's estimated token count. Now the exact total,
  summed by reading the spool back.
- **MA10** the value-length bound removed **survived**: no case existed. Worse,
  the case I then wrote built its inputs *from the constant it was checking*, so
  raising the bound raised the "too long" value with it and the case could never
  fail — **the third of this release's five ways to pass without running, in my
  own fresh test.** The lengths are literals now and the constant is asserted to
  be what they were chosen for.
- **MA2** the undecodable-payload counter removed **survived**: my "not an
  object" case used inputs that reached a different branch. A new test covers an
  envelope with no `raw` member at all, which is the branch's only reachable
  trigger.
- **MC3** the registered command losing `|| true` **survived the whole suite**:
  §3.5.

### 5.3 The unused-variable trap fired four times

MB1, MA3, MA4, MT2 all first appeared as DID NOT BUILD from an unused variable
or import, and each was re-run in compiling form (MB1b, MA3b, MA4b, MT2b) before
its verdict was trusted. S6's rule that a non-compiling mutant proves nothing
holds for all four. **MB5 is the exception S7 identified**: there the compiler
error *is* the kill, because the defect is unrepresentable, and it is labelled
that way rather than counted as a non-result.

### 5.4 Restores

Every mutation is applied by a script that diffs against a `cp` backup, reports
`NOT APPLIED` when the edit changed nothing, and restores from the backup after
the run. Backups verified by sha256 at the end; `git status --porcelain` clean
before each commit. **No `checkout`, `reset`, `clean`, `stash` or `rebase` in
any tree, ever.**

---

## 6. What I inherited, measured rather than assumed

The seventh independent confirmation that the snapshot is wrong on the merits.
All twelve files hash-checked against the manifest before use — all twelve
matched.

| What the draft did | Measured |
|---|---|
| `analyze.go` calls `toInt` | Does not exist on `main`. The draft does not compile |
| `analyze_test.go` asserts `skill_activations[commit] == 1` | The inference S7 dropped, asserted as a feature |
| `estimated_tokens` = payload bytes / 4 | A token claim from a file size |
| `last_context_tokens` from `context_tokens` | A Cursor-reported token count from a guessed field, **decoded through `float64`** — the rounding defect S7 repaired in the writer, reintroduced in the reader |
| unreadable lines | Skipped and **not counted** |
| non-object payloads | Not counted |
| `doctor.go` spool path | `~/.cursor-profiler/spool`, and a second definition of the layout |
| `doctor.go` hooks detection | String match for `"profiler ingest"` — not the command the install writes |
| `doctor_test.go` | Asserts `TierServerAPI` from an env var and `TierEnterprise` from an env var, as features |
| `doctor_test.go` homes | `t.TempDir()` with **no barrier**, while writing `.cursor/hooks.json` inside them |
| `queries/*.sql` | Sibling namespace in all 8; `strict` in none of them |
| `queries/estimated_tokens.sql` | chars/4 as "estimated context tokens", plus a per-skill regex over `SKILL.md` paths |
| `queries/tool_call_mix.sql` | `avg_duration_ms` — a unit the payload documentation does not state |

**The queries were run, not read.** DuckDB 1.5.5, against a spool seeded through
the real `ingest` including a non-object payload and a `--strict` line. All eight
return rows. On an empty spool DuckDB reports `IO Error: No files found that
match the pattern …` and names the path — the same distinction the Go reader
makes, so it is documented as wanted rather than papered over.

**One defect I introduced and fixed in my own diff.** Adding
`AND raw->>'$.field' IS NOT NULL` to a query's outer `WHERE` makes DuckDB push
the JSON path test into the scan, where it fails the whole query against a
non-object payload — the shape the capture layer keeps on purpose. Reproduced,
narrowed to the predicate by bisecting four query forms, and repaired by
projecting the payload fields in a CTE behind `WHERE json_type(raw) = 'OBJECT'`.
The draft's forms did not have the bug because they had no such predicate; mine
did, so it was mine to fix.

---

## 7. Verification — measured, not carried

### 7.1 Shell suites, both shells

| suite | bash 3.2.57 | bash 5.3.15 |
|---|---|---|
| test_f01 | 1868 | 1868 |
| test_f02 | 350 | 350 |
| test_harness | 131 | 131 |
| test_install | 48 | 48 |
| test_rewrite | 85 | 85 |
| test_skill | 53 | 53 |
| test_walk | 20 | 20 |
| **total** | **2555 passed, 0 failed** | **2555 passed, 0 failed** |

Re-run after the README and spec edits, not before. Unchanged and expected: the
slice touches no shell surface, and `test_harness`/`test_f01` reference
`README.md` only in a stageability list and in jq-tooling assertions.

### 7.2 Go

| gate | base `c104f3c` | head `5365a46` |
|---|---|---|
| `gofmt -l .` | empty | empty |
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `go test -race -count=1 ./...` | ok | **ok**, all three packages |
| **top-level tests** | **236** | **274** (+38) |
| incl. subtests | 876 | **943** (+67) |
| failures | 0 | **0** |
| skips | 1 | **1** (the barrier child, by design) |
| `profiler` coverage | 97.5% | **97.9%** |
| `profiler/cmd` coverage (instrumented) | 94.5% | **94.5%** |
| `internal/homesafe` coverage | — | **93.5%** |

**The base is 236 top-level, not S7's 237.** Re-measured in a worktree at
`c104f3c` rather than carried; the difference is one test S7 counted that the
tree reports as a skip. S6's `-cover` instrumentation of the subprocess is
untouched and still working — 111 coverage files written, and 94.5% is the
honest figure, not the plan's 12.6% artifact (L21 in the deferred ledger is
still wrong).

`internal/homesafe` at 93.5% has three uncovered statements and all three are
unreachable by construction: `filepath.Rel` erroring (both paths are absolute),
`filepath.Abs` erroring (only when the working directory cannot be read), and
`resolve` reaching the filesystem root (the root always resolves). Written down
in the test file rather than reached with an indirection in production code
existing only for a test.

### 7.3 Live, against the built binary, in a sandbox home

`HOME=<scratch>/s8-demo/home`:

| run | result |
|---|---|
| `hooks install` | 21 events registered |
| four `ingest` runs (one credential, one non-JSON) | 4 lines, 1 file |
| `analyze` | 4 lines, 4 envelopes, 0 unreadable, 3 object + 1 other payloads, 7 keys censused |
| the prompt text, the `sk-` key, the bearer token in the summary | **0 occurrences each** |
| `api_key` in the value census | `[REDACTED]` — the value the capture layer stored, so nothing leaks through the census |
| `doctor` | `tier: none`, all **21** registered events reported back, spool counted, the yields sentence present |
| `doctor --harness claude_code --otel-file <fixture>` | `tier: export`, four signals from `otel`, the reason naming the file |
| `doctor --harness cursor` | `tier: none` — *"there is no harness named \"cursor\" (supported: claude_code)"* |
| `doctor --otel-file <fixture>` with no harness | `tier: none`, the reason saying no harness was named |
| `doctor --harness claude_code --otel-file no_envelope.json` | `tier: none` — read nothing |
| `analyze --spool-dir <missing>` | exit **1**, the path named on stderr, no summary printed |
| `doctor` against a fresh home | exit 0, and the home still holds **0 files** |
| the real home, after all of it | `~/.cursor` and `~/.skill-architect` both **absent** |

### 7.4 Hygiene

- Author and committer `imagineux <imagineux@gmail.com>` on all three commits.
  AI-attribution grep over every commit object, the PR body file and the **live**
  PR title and body: **0 matches each**.
- **No version surface touched.** `git diff c104f3c..HEAD` shows no change to
  `profiler/types.go`, the five plugin manifests, `CHANGELOG.md` or
  `RELEASE_NOTES.md`. `AdapterVersion` = `0.5.0`, `ProfileSchema` =
  `skill-architect/profile/v1`, manifests = `0.4.3`.
- **Never wrote to a live config directory.** Nothing installed, nothing
  reconfigured, no PATH mirror of symlinks.
- **The primary working tree was never written to.** Re-checked after the push:
  still `0a83615`, 47 `git status --porcelain` lines — the figure S3 through S7
  all recorded.
- The snapshot stayed read-only; all twelve files hash-checked before use.
- Every file deliverable written with the Write/Edit tools, never a heredoc.

---

## 8. Findings — what the release slice needs to know

### 8.1 `spool_profile.go` still holds no profile. **[open, S7 §9.3]**

I added `forEachSpoolLine` to it and did not rename the pair. The manager ruled
on three things and this was not one of them, and a rename changes the `git add`
list. **The release slice or the adapter slice should rename it to
`spool_read.go` / `spool_read_test.go`.** It is now slightly worse than S7 left
it: the file holds the shared spool walk as well as the session read, and
neither is a profile.

### 8.2 `RELEASE_NOTES.md` and `CHANGELOG.md` need three entries. **[S10]**

Untouched here, as S7 left them. What 0.5.0 gained from this slice:

- `profiler analyze` — summarises a spool. Describes the files, not the session.
- `profiler doctor` — two tiers, `none` and `export`, and a tier is only ever
  what a probe read. **Worth a release-notes line of its own**: a user upgrading
  from a preview that advertised `hooks`, `server_api` and `enterprise` will
  notice they are gone, and the reason is the point of the release.
- `profiler/queries/` — eight DuckDB queries. `estimated_tokens.sql` is replaced
  by `payload_bytes.sql`; nothing shipped the old name, so no migration.
- S7 §9.5's note about `AppendSpool` having no locking still wants a line.

### 8.3 The `--strict` env var is still absent, and `analyze` now makes it visible. **[note]**

S7 dropped `<SIBLING>_STRICT` rather than renaming it. `analyze` reports
`stripped_envelopes`, so a user who wanted metadata-only capture and forgot the
flag can now see that none of their lines were stripped. That is the cheap half
of the problem; the flag still has to be baked into the registered command by
hand. Unchanged, and no new surface added.

### 8.4 `doctor` has no exit code for "nothing measures here". **[decide — later]**

It exits 0 whatever it finds, because detection is a report and not a gate, and
"nothing here measures anything" is the answer for most machines today. A script
asking "can this machine measure?" has to parse the JSON. If that turns out to
be wanted, `2` is the status this repo already uses for "ran and read nothing" —
but it is new surface and nobody asked.

### 8.5 The mutation runner should print the skip count itself. **[carried]**

It does now, for this slice. The change is in `<scratch>/s8-mutrun.sh` and not in
the repository — the runners have been scratch scripts since S5. **Worth
considering whether the runner belongs in `tests/lib/`**, since five slices have
now each derived it from the last by `sed`, which is the copy-by-hand pattern
`tests/lib/harness.sh` exists to end. Not done: it is not this slice's, and it
would be a new committed surface.

### 8.6 The eight queries each repeat the `read_json` column block. **[note]**

DuckDB has no way to share a source definition across independently `.read`-able
files without a prelude, and independence is what the files are for. The drift
risk is closed mechanically instead: the column list is derived from `SpoolEvent`
by reflection and the default directory from `DefaultSpoolDir`, so a query that
falls behind either turns a test red. The repetition remains, checked.

### 8.7 Smaller notes

- `SourceHooksEstimated` in `types.go` is declared and used by nothing. It was
  the source name for a chars/4 estimate over hook payloads — the thing this
  slice removed from `analyze`. Left alone; deleting a declared constant is a
  type-surface decision, and the adapter slice may want it.
- S6 §9.1's step timeout is still open. Untouched.
- `doctor`'s `EnvironmentQuery` takes five fields and no `getenv`. If an Admin
  API or OTel client ever lands, the tier for it goes in the same table as the
  other two and needs a fixture from which a capture delivers a `present`
  signal. That is the only way to add one.

---

## 9. PR

[#20](https://github.com/Okja-Engineering/skill-architect/pull/20), open, **not
draft**, base `main`, head `feat(profiler)/0.5.0-analyze-doctor` at `5365a46`.
AI-attribution grep over the live PR title and body: **0 matches**.

---

## 10. Not done, and why

| Not done | Why |
|---|---|
| Merging or tagging | The user merges. Nothing tagged. |
| A Cursor adapter, or any spool → `Profile` projection | §4 of the plan and the manager's standing ruling. `doctor` and `analyze` are why one is not needed to make the spool useful. |
| Renaming `spool_profile.go` | §8.1 — outside the manager's three rulings and outside the `git add` list. |
| An Admin API or OTel tier | §3.4. Nothing in this build reads either surface. |
| A `getenv` parameter on `DetectEnvironment` | §3.4. It existed only to produce the two tiers that are gone. |
| A Cursor-installed check | §3.4. macOS-only, and not a statement about measurement. |
| An exit code for a `none` tier | §8.4 — named, not added. |
| Touching any version surface | §7.4. The manifests are S10's. |
| Moving the mutation runner into the repo | §8.5 — new committed surface, and not this slice's. |
| Verifying anything against a real Cursor | Impossible here, and the reason `analyze` reports a key census rather than a tool-call count. |
