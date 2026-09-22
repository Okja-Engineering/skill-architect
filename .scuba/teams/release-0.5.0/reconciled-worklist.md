# Reconciled worklist — PR #22 @ `9c8ba53` · v0.5.0 release commit

Five gate lenses + two dogfood lanes. **Every lens returned NOT CLEAN.** Raw reports:
`gate-{prose-truth,numbers,stale-markers,omissions,mechanics}.md`,
`dogfood-{otlp,hooks}.md`.

**Verdict: PR #22 does not merge as it stands.** Nothing found is unfixable and nothing
found is in the release's core measurement path — the profiler's numbers are *right*,
proved against independent sources. What is wrong is (a) two containment barriers that do
not hold, (b) an installer that probably produces nothing, (c) prose that denies limits
the code has, and (d) guards that cannot fire.

## Deduplication — where lenses converged

Convergence is signal, not noise. Four findings were reached independently:

| finding | lenses | count |
|---|---|---|
| `devin.go` compiles cleanly — "all three drafts" is false | omissions O2, prose F6+F7, numbers F3 | **3** |
| "Four deferral markers" is seven, and lists three | prose F8, numbers F4, omissions F5 | **3** |
| Redaction claims inverted / `--strict` unreachable | dogfood-hooks #2+#3, prose F3+F4+F5, omissions O7 | **3** |
| False `0.5.0` forward-promises in Go comments | CoS seed (7), stale F1+F2+F3, numbers F7 | **3** |

## Classification

Order matters: **code first, prose second.** The notes must describe what actually ships,
so prose cannot be finalised until the code below settles. This also keeps one writer per
file.

---

# LANE 1 — CODE. Blocks merge.

## C1 · CRITICAL · containment is decided *before* symlink resolution, in both barriers

Both primitives fold `..` **lexically before** resolving symlinks, so a `..` crossing a
symlink is folded against the link's *name*, not its target. A destination that resolves
**into** a protected directory is judged outside it. Both documents assert the opposite as
the headline safety property.

- `skills/skill-rewrite/scripts/draft-rewrite.sh:256` (`path_absolute`, awk fold) then
  `:317` (`path_resolved`, `cd -P`) — reproduced: a draft landed in
  `$HOME/.claude/skills` with rc=0. Control refusal works, so **the bound is defeated
  purely by spelling.**
- `profiler/internal/homesafe/homesafe.go:163-165` (`filepath.Abs` → `filepath.Clean` →
  only then `EvalSymlinks`) — reproduced: `PathContains` returned `false` while the write
  landed inside the home.

**Invariant:** containment is decided on a path whose symlinks are resolved **before** any
`..` is folded. Neither `filepath.Clean` nor an awk fold may run ahead of resolution.
**Pin the test to the invariant, not the patch:** assert the barrier's verdict agrees with
where an actual write on the *spelled* path lands.

## C2 · CRITICAL · `hooks install` writes a `hooks.json` with no `version`, which Cursor documents as required

Likely consequence: Cursor ignores the file, the spool stays permanently empty, and
`doctor` reports all 21 events registered. The 31 hook tests only assert `version` is
*preserved* when pre-seeded (`hooks_install_test.go:146,180`) — **never that a virgin
install writes one.** Fixtures and code were written from the same documentation, so they
agree with each other and both disagree with the emitter.

Confidence note from the lane: the *consequence* is Medium — it follows from the schema
being documented required, and was not observed against a running Cursor, which is not
installed here. **The missing field is High and certain.**

## C3 · HIGH · `path_resolved` reports a resolvable path as unresolvable

`cd -P -- "$head"` fails for any existing non-directory leaf, and that failure surfaces as
exit 3 *"could not be resolved, so where the draft would be written is unknown"* — a false
diagnosis. Three consequences, one root:
- an **undocumented fifth refusal**: any destination that already exists as a regular file
- `-o` is **not idempotent** — it cannot re-run over its own output, while the default
  destination overwrites happily
- the symlink refusal branch `[[ -L "$resolved" ]]` (`:401`) is **unreachable dead code**;
  `tests/test_rewrite.sh:637-639` documents the dead rule as the one that fires

`tests/test_rewrite.sh:580-590` asserts only `code != 0`, which is why this survived 136
passing assertions. **It must assert the status**, or the class returns.

## C4 · HIGH · the protected-root list does not satisfy its own stated rationale

The rationale is that an agent reads a skills directory **as instructions**.
`$HOME/.agents/skills/` is unprotected — and this same README documents it as a skills
directory at `:836` (Codex) and `:842` (Cursor). Reproduced: rc=0, file written.

**Invariant:** the protected-root list and the set of directories this repo documents as
agent-read skills directories are the same set. The suite already compares the list
against `skill-rewrite/SKILL.md`; the missing side is the README's own table. All 9 other
evasion spellings probed were correctly refused.

## C5 · HIGH · two non-vacuity floors cannot fire, and one of them is the only thing behind 819 assertions

- `tests/test_harness.sh:211` — floor `200`, real count **727**. Injecting a plausible
  limit regression dropped it to 234 with `test_install.sh` audited at **zero**, and all
  seven suites stayed green at exactly 2649 on both shells. **68% of the audited surface
  gone, nothing fires.** This audit is the only thing standing behind non-vacuity for the
  819 `assert_value` sites `tests/lib/harness.sh:~150` says rest on it.
- `tests/test_f01.sh:736` — floor `6`, real count **16**. Re-spelling 8 primitives as
  `name ()  {` dropped the set to 8; **24 assertions vanished from the published 1868 with
  zero failures.**

`test_harness.sh:140-148` already records this exact lesson (*"The number used to be a
floor — `-ge 6` … the floor still passed, one suite went unexamined"*) and applied the fix
to the suite count and not to the site count **in the same file**. F2 also falsifies the
documented exemption at `test_f01.sh:1573-1577`.

**Invariant:** a count that stands behind non-vacuity is derived from the artifact and
compared for **equality**. Also fix `tests/test_f01.sh:1290` (floor 15, real 20 — its
neighbour is one-directional and would not catch a shrinking walk). The five remaining
floors are equal-to-real today; convert or document each.

## C6 · MEDIUM · `--home` scopes the registration but not the spool

The registered command carries no `--spool-dir`, so it writes to the real `$HOME`;
`doctor --home X` then reports a spool dir the hook never writes to, and
`README.md:467,527`'s own try-it example points the reader at an empty directory.

## C7 · MEDIUM · eight false `0.5.0` forward-promises in Go files

`profiler/types.go:81,84,85` · `claude_code.go:475,611` · `otlp.go:1130` ·
`cmd/main.go:118` · `cmd/main_test.go:369`. Each is a copy of a sentence whose prose
sibling S10 already repaired. **Group by marker topic, not by file:** probe-exit
(`main.go:118`, `main_test.go:369`), caveat-channel (`claude_code.go:475,611`,
`otlp.go:1130`), reserved-fields (`types.go:81,84,85`).

Also `tests/test_rewrite.sh:1170-1176` — says the capability "is 0.5.0" while the shipped
`skills/skill-rewrite/SKILL.md:117` frames the same gap as a permanent design boundary.
**One of the two is wrong and neither cites the other** — decide which, then make them agree.

**Do NOT touch:** `profiler/profiler_test.go:325` (`retiredActivationDeferral`) is a
negative control asserting the sentence is gone. `docs/profiler-spec.md:72` and
`claude_code.go:837-838` are correct as written.

## C8 · MEDIUM · `MetricSource` declares five values no shipped capture can produce

`SourceHooks`, `SourceSessionData`, `SourceServerAPI`, `SourceSQLite`,
`SourceHooksEstimated` (`types.go:54,60,61,62,63`) have **zero** production references.
`SourceServerAPI`'s string is one of the three removed `doctor` tiers, and
`cmd/main_test.go:1704` asserts the doctor report may not contain it *"which no capture in
this release delivers"* — while it is declared a valid profile source 1,600 lines away.
Unlike the reserved struct fields, **none carries any comment**, so the claim is made
silently by the declaration.

The release built the AST-reading exhaustiveness guard for the **other** vocabulary
(`adapter_contract_test.go:1080-1124`, `declaredTiers`) and not this one.
`git grep declaredSources` → empty. **`declaredTiers` is the model.**

## C9 · GUARD · stop the fifth recurrence of the marker class

Denominator **must** be `git ls-files` — `scanFor`'s roots (`.`, `../docs`,
`../README.md`) miss `tests/` and `skills/`, which is where two of these live. The check
must distinguish *assignment* ("is 0.5.0", "for 0.5.0", "reserved for") from *history*
("0.5.0 did not add", "gained … in 0.5.0"), or it fires on ~40 correct lines.

## C10 · GUARD · the release documents are an unguarded version surface

Deleting the **entire** 0.5.0 section from both documents leaves all 2649 assertions green,
byte-identical. `grep -rn 'CHANGELOG\|RELEASE_NOTES' tests/*.sh tests/lib/*.sh` returns
nothing. `tests/test_skill.sh:93-96` learned this lesson for `marketplace.json` and did not
apply it to the two files that **are** the release.

---

# LANE 2 — PROSE. Blocks merge. Runs after Lane 1 settles.

## P1 · CRITICAL · the redaction claims are inverted — a privacy claim that is false

`RELEASE_NOTES.md:10` and `CHANGELOG.md:64` say `prompt`, `command`, `tool_input`,
`tool_output` are `[REDACTED]` by default. **They are plaintext.** `profiler/hooks.go:308-311`
says so in the code. Three sub-errors, one transcription root — `docs/profiler-spec.md:545-571`
states the contract **correctly** and the notes collapsed it:
- content stripping is `--strict`-only (`hooks.go:202-217`), not default
- there is an always-on **third** exception (credential redaction, `hooks.go:346`), not "two"
- so "a line written is a line read back, byte for byte" (`:11`, `README.md:482`) is false

Compounding: **`hooks install` never registers `--strict`** and there is no environment
variable, so the metadata-only mode is unreachable through the documented path. Either C6's
fix makes it reachable, or the notes must say it requires a hand-written `--command`.

Also `analyze` emits `payload_key_values` — every `cwd`, conversation id, and unlisted
field — undisclosed in the notes while `experiment` gets a dedicated "read before you
publish" warning. README and spec do disclose it.

## P2 · HIGH · `README.md:315` tells the user to disable the flag the adapter reads

> "Leave `OTEL_LOG_TOOL_DETAILS` unset. It is not needed — **this adapter reads nothing it
> adds**"

False and self-contradictory in one sentence. Measured, one variable changed: unset →
`skill_name: "custom_skill"`; set → `skill_name: "dogfood-probe"`. The adapter reads
`skill.name` at `profiler/claude_code.go:786`. **A user following the README's own recipe
gets `custom_skill` for every user-authored skill** — which is every skill this project
exists to author. The code is right; the operative instruction is wrong.

Lower, same area: `RELEASE_NOTES.md:90` / `CHANGELOG.md:528` claim the event carries
`skill.kind`. It does not on 2.1.221 — **two fixtures assert one the emitter never sends.**

## P3 · HIGH · "all three drafts do not compile" — `devin.go` compiles cleanly

Three lenses reproduced it, one using S10's own recipe. Go printed `too many errors` and
truncated; the conclusion was generalised from a truncated compiler run. Also incomplete:
`cursor.go:228` needs `getBool`, a **fifth** undefined name, never present at `541af3e`
either — so "removed by v0.4.1's rewrite" does not explain it.

**The decision to delete all three still stands** on Devin's other, verified defect
(`devin.go:37` → `SourceSessionData`, `:114` → `UnknownTokenResult`). Narrow the sentence
to what the compiler printed, **per file**, and name the ref the `file:line` citations
resolve against — they point at `1e4e845`/`0a83615`, unreachable from the release.

## P4 · HIGH · the upgrade consequence nothing disclosed

`AdapterVersion` moved to 0.5.0 **and** `compare` refuses any pair whose versions differ
(exit 2). Both facts are in the changelog; **neither document joins them.** Every profile
captured with 0.4.x is refused by this release's headline command. 0.4.3 made exactly this
disclosure for its own bump, so this is a measured regression in candor against the section
this release is modelled on. `RELEASE_NOTES.md` never states `AdapterVersion` moved at all.

Same root: `skill_activation` gained the `error` state, widening an existing key's value
domain where v0.4.3 documented `{present, unknown}` as complete. The changelog states the
schema rule verbatim and then omits the only change in the release that triggers it.

## P5 · MEDIUM · every counted enumeration that is short

Each is *"N things"* where the walk finds more. Derive from the artifact or drop the numeral.

| claim | written | real |
|---|---|---|
| deferral markers re-pointed | 4 (lists 3) | **7** |
| keys this release adds | 8 | **9** (`total`, `types.go:284`) |
| `-o` destinations refused | 4 | **5** |
| runs to reach the backup collision | 3 | **2**, and the original is lost |
| profile keys written by nothing | 3 | **8** |
| things deliberately not shipped | "a Cursor adapter" | **three** adapters |
| `probe` stdout across error cases | "byte-identical" | differs (`README.md:95-97` says it correctly; the escalation is new in this diff) |
| `analyze` "says in its own output that it yields no measurement" | — | it does not; that is `doctor`'s field |

Plus undisclosed shipped surface: `compare`/`experiment run` exit **2**; `experiment`'s
cross-harness design refusal and its fourth plan refusal; `compare` treating `harness` and
`skill_dir` as **notes, not refusals**; `duckdb` as a new external tool absent from
Prerequisites and exercised by no test or CI step.

## P6 · DECISION — the release ships two contradictory claims

`README.md:726` says the Codex install route is **verified** against `codex-cli 0.153.4`.
`RELEASE_NOTES.md:21` / `CHANGELOG.md:304` publish L22 saying the **Codex** row is
unverified. The ledger's L22 narrative was already false against the v0.4.3 tree it was
audited over, and 0.5.0 re-publishes it verbatim. **Either narrow L22 to Devin+Cursor in
both documents, or correct the README row. The release cannot ship both.**

## P7 · disclosures to add

- **Tool-call hook payload shapes are unverified against a live emitter.** User decision,
  taken: ship with it disclosed. `PreToolUse`/`PostToolUse` never fired — a nested
  `claude -p` could not authenticate. Lifecycle hooks and the spool/analyze/doctor path
  **were** verified live.
- **Reword `AppendSpool`'s note: the disclosure understates the safety.** 40 concurrent
  writers, payloads to 8 MB, a 128 MB file, **zero interleaving** — every 8 MB line exactly
  8,000,143 bytes. It holds structurally: one `O_APPEND` `write(2)` per record. Concurrent
  firing *is* normal. Name the untested NFS caveat instead of implying a live hazard.
- **The `[DOCS]` sentence changes in both directions**: location and all 21 event names are
  now confirmed field-for-field against Cursor's current reference; `cwd` is now known
  **wrong** (not a universal Cursor payload field); and C2 is a sharper limit than "we
  cannot tell you it works". Keep "we cannot tell you it works against Cursor" — Cursor is
  still not installed.
- `--strict` writes `user_email` to disk and that string appears nowhere in the repo.
- `hooks uninstall` leaves `{"hooks":{}}` plus a `.bak` on a virgin home.
- The activation provenance sentence cites `skill.name`, the attribute the adapter
  deliberately does **not** read. What shipped keys on `claude_code.skill_activated`, which
  appears in **neither** provenance list, and both lists are byte-identical to v0.4.3.
- `experiment run`'s freshness stamp accepts a re-touched stale profile (S6 named it).
- The `hooks.json` same-second backup collision is in the changelog and **not** in the
  release notes — the only one of 19 limits that diverges, and RN is the user-facing file.
- Fix the PR body: `94 (A) → 92.5 (A-)`. The body puts `(A-)` on the from-number and
  inverts the fact the bullet exists to record.

---

# DEFERRED — disclose or carry to 0.6.0, not blockers

- **M3 · durability.** `origin/feat(profiler)/0.5.0-analyze-doctor` **was pushed and is
  gone from the remote**; its three commits are reachable from no remote ref. No data lost
  (byte-identical in `10326b3`), but it is this repo's own counter-example to substituting
  a branch for a tag. `origin/wip/0.5.0-profiler` is the tip of the branch the primary tree
  sits on, so a routine push moves the "frozen" point; `git tag --points-at a3673b8` is
  empty. **Recommend an immutable tag now — it is one command and the 0.5.0 WIP is
  currently verified byte-identical to the archive (47/47 entries, zero differ).**
- **M2 · `.gitignore`** claims tool enforcement it lacks for `NOTICE` and
  `docs/skillgate-*.md` — both untracked in the tree right now and both matching the DoD's
  own grep. Compounding: the primary tree's *working* `.gitignore` is the three-line
  pre-boundary version, so `git add -A --dry-run` there stages all 150+ forbidden files.
  Resolves on checkout of `main`.
- **M4** · 186 untracked files (`skillgate/` 139, `docs/research/` 41, plus `NOTICE`,
  `go.work*`) exist on **no** remote ref or tag — one disk only. Outside DoD #6's letter;
  0.6.0 scope.
- `README.md:26,28` still calls the profiler a **preview** and attributes it to v0.4.0,
  while the notes lead with six new subcommands. `README.md:34` Status column reads
  `✅ Slice 1`. `docs/profiler-spec.md:3` header is 8 days and 5 slices stale; `:776,778,844`
  label the release's normative ACs "(Slice 1)". `docs/profiler-spec.md:7` Purpose claims
  all four harnesses in present tense.
- `tests/lib/out-of-scope-check.sh:6-7` mis-attributes `docs/research/` and `NOTICE` to
  0.6.0 — the ladder puts them at 0.7.0 and nowhere. Behaviour is correct; the comment is
  one release out.
- `-o ""` is silently accepted and writes to the default destination (`:394` refusal
  unreachable behind `:434`'s `-n` guard).
- `profiler/go.mod:3` pins `go 1.27.1` — patch-pinned on a release artifact.
- `profiler/cmd/main_test.go:63` cites 91.5% at `e26b2a5`; the figure is at neither commit
  (8.5% there, 94.5% at `d77d90e` where the mechanism landed, 94.5% at head).
- `tests/lib/harness.sh:20` points at `audit-assertions.sh`; the file is `audit-suites.sh`.
- The 13 open ledger rows, unchanged.

---

# What is verified GOOD — do not re-open, and do not "repair"

- **The profiler tells the truth on real data.** Tokens `1111/2222/3333/4444` verified
  against **three** independent sources (the upstream API's served log,
  `claude --output-format json`'s `.usage`, and `jq` over the raw wire bytes).
  `tool_calls` = 2 verified three ways; `timing.total_ms` = 15 by `jq` arithmetic on raw
  `timeUnixNano`. **Session scoping genuinely isolates one session out of a real
  three-session export**, byte-identical single- vs multi-session. Both honest `unknown`s
  confirmed honest across 32 observed attribute keys.
- **Every published number is correct**, re-derived twice by two lenses: 2649 on both
  bashes with identical per-suite splits, 275 (233/34/8), 669 (636/33), coverage
  97.9/94.5/93.5, 64 fixtures, 8 queries, and all four 0.4.3 baselines re-measured from
  the tag rather than quoted.
- **Structural honesty clean.** Both documents byte-identical to `v0.4.3` below the new
  sections — **no history rewritten**. `## Unreleased` correctly empty. Candor is *up*
  versus 0.4.3, not down.
- **The removed `doctor` tiers are exemplary** — guarded twice, including an AST-read
  constant set, and driven with the old env vars set. Do not touch.
- **`skillscore` history confirmed** across five revisions: the 94→92.5 drop is entirely
  inside 0.4.3, 0.5.0 changed neither score, and the record is correct. Global install
  intact at 2.0.2.
- Tag absent and attribution clean across all 34 commits and the PR body.
- The five version asserts are non-vacuous, proven RED independently by two lenses.
- `analyze`'s schema-drift guard is RED-proven. Spool numeric fidelity exact on three
  hostile literals. Install merges, re-run writes nothing, uninstall removes exactly ours.
- Skill activation verified against the real vendor doc, including a negative control.
