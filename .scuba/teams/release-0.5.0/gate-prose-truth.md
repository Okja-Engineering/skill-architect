# Gate lens: PROSE TRUTH — PR #22 @ `9c8ba53`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so, verified by
> `ls`, and did not fall back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 16 REAL, 7 low/SUSPECTED.**

## Coverage

Denominator: every fact-asserting sentence in the added/modified lines of the three prose
files, plus every claim in S10 record §3's ten-owed-items table.

- `RELEASE_NOTES.md` **24/24** added lines walked sentence by sentence.
- `CHANGELOG.md` **338/338** added lines, all nine blocks.
- `README.md` **9/9** modified hunks, plus the three install-table rows `:725-727` and
  the five skills-dir rows `:827-844` the modified sentences scope.
- S10 §3 — **13/13** cited `file:line` pairs confirmed present and saying what the
  record claims, in **both** documents.
- `-o|--output`: **4/4** stated refusals driven + 2 stated acceptances + **9 additional
  evasion spellings**.
- **8/8** subcommands driven against a binary built at `9c8ba53`.
- Independently re-measured: 7 suites × 2 shells, `go test -race -v`, `-cover`, 0.4.3
  baselines from a `git archive` tree, skillscore across 3 trees, the three adapter
  drafts compiled **individually**.
- Grounded against the real vendor doc (`code.claude.com/docs/en/monitoring-usage`, raw,
  160 kB).

Second sweep added F13–F16 and L1–L7; a third pass added nothing.

---

## Root A — "decided after resolving the path" is FALSE in both implementations

**Shared root:** both containment primitives normalize `..` **lexically, before** resolving
symlinks. A `..` that crosses a symlink is folded against the link's *name*, not its
target — so a destination that resolves *into* a protected directory is judged *outside*
it. The documents assert the opposite as the headline safety property.

### F1 · CRITICAL · REAL (reproduced) · `RELEASE_NOTES.md:18`, `CHANGELOG.md:96-97`

> "The decision is made after resolving the path, not from its spelling"

`path_absolute` (`skills/skill-rewrite/scripts/draft-rewrite.sh:256`) folds `..` with awk;
`path_resolved` (`:317`) only then runs `cd -P`.

```
$ ln -s $HOME/.claude/skills $W/outside/lnk
$ bash draft-rewrite.sh -t $W/skill -o "$W/outside/lnk/../evade.md"
Rewrite draft written to: .../outside/lnk/../evade.md      rc=0
$ find $HOME/.claude -type f
.../home/.claude/evade.md            # inside the protected directory
```

Control: `-o $HOME/.claude/skills/evade.md` → rc=1, refused. **The refusal is defeated
purely by spelling.**

### F2 · CRITICAL · REAL (reproduced) · `CHANGELOG.md:165-168`

> "`profiler/internal/homesafe` holds it, and it decides after resolution"

Same root. `resolve` (`profiler/internal/homesafe/homesafe.go:163-165`) calls
`filepath.Abs` then `filepath.Clean` — lexical `..` removal — **before** `EvalSymlinks`.

```
PathContains(home, ".../outside/lnk/../evade.md") = false, err=<nil>
THE FILE LANDED AT .../home/.claude/evade.md (inside the protected home)
--- FAIL: barrier said OUTSIDE but the write landed inside the home
```

**Invariant:** containment must be decided on a path whose symlinks are resolved *before*
any `..` is folded. Neither `filepath.Clean` nor textual awk folding may run ahead of
resolution.

---

## Root B — the spool-fidelity prose mis-transcribes the spec

`docs/profiler-spec.md:545-571` states the contract **correctly** in five bullets,
including *"A credential is removed, and that is the one deliberate loss"* and that
prompts survive, and that `--strict` **additionally** replaces content fields. The release
prose collapsed that to "two named exceptions" and swapped the `--strict`-only
`stripContent` (`profiler/hooks.go:202-217`) for the always-on `redactSecrets` (`:346`).

### F3 · CRITICAL · REAL (reproduced) · `RELEASE_NOTES.md:10`, `CHANGELOG.md:64`

> "`prompt`, `command`, `tool_input`, `tool_output` and their neighbours are `[REDACTED]`
> by default"

**FALSE and exactly inverted.**

```
$ echo '{"hook_event_name":"...","prompt":"secret text","command":"rm -rf /","tool_output":"out"}' | profiler ingest
{"ts":…,"raw":{"command":"rm -rf /",…,"prompt":"secret text",…,"tool_output":"out"}}
```

`profiler/hooks.go:308-311` says so in the code: *"Prompt text and file paths stay,
because on this tool they are the data; credentials do not."* The notes tell a user their
prompts and shell commands are redacted in a file on disk that outlives the session.

### F4 · HIGH · REAL (reproduced) · `RELEASE_NOTES.md:10`, `CHANGELOG.md:62-64`

> "with object key order and duplicate keys as **the two named exceptions**"

FALSE — an always-on **third** exception the spec itself names:

```
in:  {"api_key":"plainvalue","my_token":"abc","command":"curl -H \"Authorization: Bearer xyz123\"…","note":"sk-abc…"}
out: {"api_key":"[REDACTED]","my_token":"[REDACTED]","command":"curl -H \"Authorization: [REDACTED]\"…","note":"[REDACTED]"}
```

### F5 · HIGH · REAL · `RELEASE_NOTES.md:10,11`, `CHANGELOG.md:59`, `README.md:482`

> "the payload is written as it arrived" / "a line written is a line read back, byte for
> byte" / "What the spool does is put on disk what it was handed"

FALSE by F4's evidence. **Invariant:** a byte-fidelity claim must enumerate every
transformation on the write path, credential redaction included.

---

## Root C — enumerations asserted complete that were never walked to the end

### F6 · HIGH · REAL (reproduced) · `RELEASE_NOTES.md:15`, `CHANGELOG.md:19`

> "Measured … by copying all three into the profiler package at head and building: they do
> not compile"

Built each draft **alone** against `9c8ba53`:

```
cursor alone: undefined otelMetric, otelLog, toInt, getBool, parseTime
codex  alone: undefined otelMetric, otelLog, toInt ×5, parseTime
devin  alone: (no output — compiles cleanly)
```

The original measurement added all three at once; **Go printed `too many errors` and
truncated.** The conclusion was generalized from a truncated compiler run.

### F7 · HIGH · REAL (reproduced) · `RELEASE_NOTES.md:15`, `CHANGELOG.md:20-21`

> "`otelMetric`, `otelLog`, `toInt` and `parseTime` … and all three drafts read them"

`devin.go` references **none** of the four (0/0/0/0). Also incomplete: `cursor.go:228`
needs `getBool`, a fifth undefined name the notes omit — and it was never present at
`541af3e` either, so "removed by v0.4.1's rewrite" does not explain it. (The four-name
claim about v0.4.1 *is* true: all four present at `541af3e`, gone at `46d292d`.)

### F8 · MEDIUM · REAL · `CHANGELOG.md:189`, `RELEASE_NOTES.md:21`

> "Four deferral markers reading '0.5.0' are re-pointed"

**Seven** were: `README.md:98`, `:186`, `:323` and `docs/profiler-spec.md:788`, `:792`,
`:806`, `:837`. And the sentence then names only **three**, omitting the spec's
`duration_ms`/`active_time` marker. A sentence asserting four and listing three, when the
answer is seven.

### F9 · MEDIUM · REAL · `RELEASE_NOTES.md:21`, `CHANGELOG.md:304`

> "the Devin, Codex **and Cursor** install rows being unverified"

Contradicts `README.md:726` **in the same release**: *"verified — both steps run against
`codex-cli 0.153.4` … `codex plugin list` then reports the plugin installed and
enabled."* The ledger's L22 narrative (`deferred-ledger.md:343`) was already false against
the v0.4.3 tree it was audited over; 0.5.0 re-publishes it verbatim. **One of the two
claims is false; the release ships both.**

### F10 · MEDIUM · REAL (reproduced) · `CHANGELOG.md:315-317`

> "Reaching it needs three runs inside one second"

**Two suffice, and the original file is lost:**

```
$ profiler hooks install && profiler hooks uninstall     # 2 runs, same second
$ ls ~/.cursor/  →  hooks.json  hooks.json.bak-20260921T153740Z
$ cmp backup original → NO                               # original unrecoverable
```

### F11 · HIGH · REAL (reproduced) · `RELEASE_NOTES.md:18`, `CHANGELOG.md:88-89`

> "It writes anywhere you can write, except four destinations"

An undocumented **fifth** refusal: **any destination that already exists as a regular
file**, refused at `exit 3` with the false diagnosis "could not be resolved".

```
$ echo old > existing.md; draft-rewrite.sh -t skill -o existing.md
ERROR: the destination …/existing.md could not be resolved … rc=3
$ draft-rewrite.sh -t skill -o new.md   # rc=0
$ draft-rewrite.sh -t skill -o new.md   # rc=3  — not idempotent
$ draft-rewrite.sh -t skill             # rc=0, twice — the default overwrites happily
```

Root: `path_resolved` (`:317-336`) reaches `cd -P -- "$head"`, which fails for any
existing non-directory leaf, and that failure is reported as unresolvability.

### F12 · MEDIUM · REAL (reproduced) · `RELEASE_NOTES.md:18`, `CHANGELOG.md:101-103`

> "No new exit status — a destination the script will not write to is the `1` the header
> already registers"

FALSE. Exit **3** for a symlink destination and for an existing-file destination.
`tests/test_rewrite.sh:580-590` (`refused_out`) only asserts `code != 0`, which is why
this survived 136 passing assertions.

### F13 · MEDIUM · REAL (reproduced) · `RELEASE_NOTES.md:18`, `CHANGELOG.md:90-92`

The symlink refusal branch `[[ -L "$resolved" ]]` (`draft-rewrite.sh:401`) is
**unreachable**: `path_resolved` never returns a symlink. Observed instead — symlink→
nonexistent: exit 3; symlink→file: exit 3; symlink→dir: exit 1 "is a directory". The
outcome is safe; the enumeration and the reason shown to the user are wrong.
`tests/test_rewrite.sh:637-639` even documents the dead rule as the one that fires.

---

## F14 · HIGH · REAL (reproduced) — the stated rationale is not met by the stated bound

`RELEASE_NOTES.md:18`, `CHANGELOG.md:93-96`:

> "an agent reads `~/.claude/skills/` as skills, so a draft dropped in one … may be loaded
> as instructions"

**`$HOME/.agents/skills/` is unprotected — and this same README documents it as a skills
directory**, at `:836` (*"Codex also discovers skills from … `~/.agents/skills/`"*) and
`:842` (*"Cursor — … `~/.agents/skills/` globally"*).

```
$ mkdir -p $HOME/.agents/skills
$ draft-rewrite.sh -t skill -o $HOME/.agents/skills/draft.md   # rc=0, file written
```

**Invariant:** the protected-root list and the set of directories this repo documents as
agent-read skills directories must be the same set. All 9 other spellings probed (double
slash, relative-from-inside, `..` chains without a symlink, `$HOME/.claude` itself,
trailing slash, lowercase `skill.md`, literal tilde, symlinked-dir leaf) were correctly
refused.

---

## Two more false capability claims

### F15 · MEDIUM · REAL (reproduced) · `README.md:480-481`

> "`analyze` … **says in its own output that it yields no measurement**"

FALSE. `analyze`'s output has 16 fields and no such statement — the `yields` sentence
lives only on `doctor`'s `observed.spool.yields` (`profiler/doctor.go:189`).
`RELEASE_NOTES.md:14`'s *"every spool observation ships with the sentence"* is true of
`doctor`'s and false of `analyze`'s.

### F16 · MEDIUM · REAL (reproduced) · `CHANGELOG.md:311-313`

> "the reason goes to stderr and stdout is **byte-identical either way**"

FALSE:

```
$ diff <(probe --otel-file ./typo.json) <(probe --otel-file full_export.ndjson)
<   "skill_activation": "none", "timing": "none", "tokens": "none", "tool_calls": "none"
>   "skill_activation": "otel", "timing": "otel", "tokens": "otel", "tool_calls": "otel"
```

`README.md:95-97` (unmodified) says it correctly — *"the report is the same JSON a caller
already parses"*. The CHANGELOG escalated "same shape" into "byte-identical", and that
escalation is **new in this diff**.

---

## LOW / SUSPECTED

| # | `file:line` | Finding |
|---|---|---|
| L1 | `README.md:98-99` | "no release has committed to adding one" — the line this diff *removed* read "deferred to 0.5.0", a commitment a shipped release made. REAL, low. |
| L2 | `RELEASE_NOTES.md:15` | Devin draft "returns an **unknown** result on every path" — `devin.go:88,98,107` return `ErrorTokenResult`. Conclusion survives; error≠unknown is load-bearing elsewhere. REAL, low. |
| L3 | `RELEASE_NOTES.md:15` | "reports `tokens` from `sqlite` for **any** stat-able export" — `cursor.go:55` guards on `caps[MetricTokens] == SourceNone`. SUSPECTED over-broad. |
| L4 | `RELEASE_NOTES.md:10` | "`--strict` reduces them to their byte sizes" — the number is the JSON-encoded size; a 10-byte string reports `{"_stripped_bytes":12}`. REAL, low. |
| L5 | `RELEASE_NOTES.md:23` | "the attribute names and values are **not** `[OBSERVED]`" — `testdata/otlp/README.md:18-23` labels `event.name` and the resource attribute set `[OBSERVED]`. Errs conservative. REAL, low. |
| L6 | `README.md:713` | "`claude plugin list` shows … 0.5.0" — derived from the manifest, never run, and it sits *above* the sentence scoping verification to 0.4.3. UNVERIFIABLE-AS-WRITTEN. |
| L7 | `RELEASE_NOTES.md:23` | Fixtures use `invocation_trigger` = `slash_command`/`skill_tool`; the reference documents `"user-slash"`, `"claude-proactive"`, `"nested-skill"`. "From Anthropic's monitoring reference" does not hold for that attribute's values. REAL, low. |

---

## Walked and TRUE — what could not be broken

**Every published number is correct.** 2649 assertions, 0 failures, **identical** under
bash 3.2.57 and 5.3.15, per-suite 1868/350/174/136/53/48/20. 275 top-level Go tests
(233/34/8), 274 PASS + 1 SKIP (`TestHomeBarrierChild`), 669 subtests (636/33) under
`-race`. Coverage 97.9/94.5/93.5. 64 OTLP fixtures. 8 `.sql` queries. **0.4.3 baselines
re-measured on a `git archive v0.4.3` tree: 2499, 93, 452, 59 — all four exact.**
skillscore: skill-audit 92.5 (A-), skill-rewrite 89.5 (B+); v0.4.1 94 (A) → v0.4.3 92.5,
`Clarity & Instructions` 9→8, warning verbatim; the 0.4.3 entries indeed do not record it.

**The removed `doctor` tiers are genuinely gone.** Only `TierNone`/`TierExport`
(`doctor.go:71,74`). Driven with `CURSOR_ADMIN_API_KEY` and `OTEL_EXPORTER_OTLP_ENDPOINT`
set: tier stays `none`. No `net/http` in `profiler/`. `getenv` parameter gone.
`measure(harness, exportFile)` takes exactly those two. `doctor` exits 0 in every case.

**`compare`**: version mismatch refused with both versions named and every metric
`comparable: false` upstream of subtraction; differing `snapshot_hash` a note, never a
refusal.

**`experiment`**: all four design refusals driven; the exits-0-writes-nothing refusal;
`harness`/`snapshot_hash`/`skill_dir` identity refusals each driven. **No cap:** zero hits
for `CommandContext|context\.|WithTimeout|Budget|time.After|Deadline` in `experiment.go`;
`exec.Command("sh","-c",…)` at `:505`; nothing sources `mutation-runner.sh`. Archived
drafts confirm the history: `a3673b8:profiler/experiment.go:19-22` declares
`Budget{MaxTokens, MaxTimeMs}`.

**Spool**: `UseNumber` fidelity proven exact on three hostile numerics. 21 events
registered. Install **merges** (foreign entries survive), re-run adds nothing and writes
nothing, uninstall removes exactly our 21. Corrupted line skipped and counted.

**`analyze`**: **RED-proven** — adding one json-tagged field to `SpoolEvent` turns all 8
`TestSpoolQueries_ReadTheEnvelopeThatIsWritten` subtests red.

**Skill activation — verified against the real vendor doc**:
`claude_code.skill_activated` exists with the documented sentence (line 1006);
`skill.name` appears on exactly the five named surfaces (§588, §606, §735, §762, §786);
"Skill active for the request" verbatim (line 600). Negative control driven.

**Contract**: all four named guard tests exist. Empty session id refused (exit 1). A count
not read has **no key**. `attribution` `unknown` on every profile.
`estimated_context_tokens` absent, and it is a pointer — a value type would emit
`{"state":""}`.

**Ledger**: 33 L-rows, **15** name 0.5.0, all 15 OPEN; audited at `5c847e1` before v0.4.3
shipped. 15 − L14 − L21 = **13**, and the bullet enumerates exactly 13.

**CI / R4**: one `test` job, `ubuntu-latest`, no matrix. The bash-3.2 premise **driven**:
`/bin/bash -c 'set -e; [[ 1 == 2 ]]; echo REACHED'` prints REACHED and exits 0; bash
5.3.15 exits 1.

**exit 7 reproduced** with `rm` shadowed; `cleanup()` byte-identical to v0.4.3 and
`CHANGELOG.md:419` records it in full, so "unchanged from 0.4.3 and recorded there" is
TRUE.

Also TRUE: six new subcommands (9 vs 3); five manifests at 0.5.0; `README.md:715-721`'s
attribution is accurate — the three table rows are **byte-identical** to
`v0.4.3:README.md:435-437`; Cursor genuinely unreachable here, and Devin's CLI *is*
present, consistent with its row's "deliberately not run" reason.

---

## Fix direction — ADVISORY, re-derive at the root

- **Root A** is one invariant in two places: *never lexically fold `..` ahead of symlink
  resolution.* One correct resolver (resolve the deepest existing ancestor, then append
  the untouched remainder **without** cleaning `..`, or reject a destination containing
  `..` outright) closes F1 and F2 together. **Pin the test to the invariant:** assert the
  barrier's verdict agrees with where an actual write on the *spelled* path lands.
- **Root B** is a transcription error, not a code bug — the spec is correct. F3/F4/F5
  close by making the two documents say what the spec says: three exceptions, credentials
  always removed, content stripped only under `--strict`.
- **Root C** wants a mechanical answer, not a careful re-read: every "N things" sentence
  derived from a counted denominator the way `test_skill.sh` already derives its
  registries. **F6/F7 need the measurement re-run per file** — Go's `too many errors` cap
  is what truncated it.
- **F11/F12/F13** share the `path_resolved`-fails-on-a-file root. Fixing that resolver
  surfaces the real answers. `tests/test_rewrite.sh:580-590` must assert the *status*, not
  just non-zero, or the class returns.
- **F9 needs a decision, not a patch:** either the Codex row's "verified" is right and L22
  must be narrowed to Devin+Cursor in both documents, or the row is wrong. **The release
  cannot ship both.**
