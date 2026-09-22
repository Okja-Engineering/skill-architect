# Lane 2 — PROSE. Record. PR #22, `release/0.5.0`

Base `d904c30` (Lane 1's head). My head **`011defa`**. 3 commits, author and committer
`imagineux <imagineux@gmail.com>` on all three, `%(trailers)` empty on all three, and
`git log d904c30..HEAD --format='%H%n%B' | grep -icE 'co-authored|generated with|noreply@anthropic'`
→ **0**. Pushed to `origin/release/0.5.0`. CI `test` → **pass** (2m58s) on `011defa`;
per-PR `mergeable: true`, `mergeable_state: clean`. PR #22 body rewritten (16 commits from
base, the grade fix among the corrections).

Worktree: `…/scratchpad/lane2-prose`, `git worktree add --force … release/0.5.0`. One
throwaway detached worktree (`…/scratchpad/lane2-red` at `d904c30`) for the three RED
checks, since two of them require editing files and one requires editing `tests/lib/`;
removed afterwards. Nothing was written in the primary tree except this file; it is still
at `0a83615` with 47 dirty entries, matching the session-start snapshot.

**Lane 1's files are untouched.** `git diff d904c30..HEAD --name-only` →
`CHANGELOG.md`, `README.md`, `RELEASE_NOTES.md`, `docs/profiler-spec.md`. No Go file, no
`tests/**`, no `skills/**/scripts/`. 4 files, +576 −165.

---

## Verification, at head

| check | command | result |
|---|---|---|
| shell, bash 3.2.57 | `for s in tests/test_*.sh; do /bin/bash "$s"; done` | **2741 passed, 0 failed** |
| shell, bash 5.3.15 | same with `/opt/homebrew/bin/bash` | **2741 passed, 0 failed**, identical per-suite |
| — per suite, both shells | | `test_f01` 1886 · `test_f02` 350 · `test_harness` 191 · `test_install` 50 · `test_rewrite` 158 · `test_skill` 86 · `test_walk` 20 |
| go build | `cd profiler && go build ./...` | clean |
| go vet | `go vet ./...` | clean |
| gofmt | `gofmt -l .` | no output |
| go test | `go test ./...` | ok ×3 |
| skill-validator | `skill-validator check skills/skill-audit`, `… skills/skill-rewrite` | exit 0 both; `-o json` → `errors=0 warnings=0 passed=True` |
| frozen sections | `## 0.4.3`→EOF vs `v0.4.3` tag, both documents, plus each header | **byte-identical**, verified after the edits |
| PR threads | `pulls/22/comments`, `issues/22/comments`, `pulls/22/reviews`, `per_page=100` | **0 / 0 / 0** — nothing to reply to or resolve |
| mergeable | `gh api repos/…/pulls/22` (per-PR, not the list endpoint) | `mergeable: true`, `mergeable_state: blocked`, head `64de8c0` |

The forward-promise check and the release-document guard both run from `test_skill.sh`, so
the 86-assertion suite passing is the statement that **no sentence I wrote assigns work to
the release being cut**. One draft of mine did trip it, deliberately-by-accident: I had
written the literal phrase `tracked for 0.5.0` into the 0.5.0 changelog section as an
example of what the check refuses. It reddens. I replaced the literal with
`tracked for <the version being cut>`. Recording that because it is the guard earning its
keep on the first writer after it landed.

## Numbers, each re-derived by me at `64de8c0`

Every figure below was taken with the command beside it, in my own worktree, after my
edits. The published prose carries exactly these.

| figure | value | command |
|---|---|---|
| shell assertions | **2741**, 0 failed, both bashes | `for s in tests/test_*.sh; do $SH "$s" \| tail -1; done` |
| — per suite | **1886/350/191/50/158/86/20** | as above |
| top-level Go tests | **285** = 239 + 35 + 11 | `go test -list '.*' <pkg> \| grep -c '^Test'` |
| — of which | **284 pass, 1 skips by design** (`TestHomeBarrierChild`) | `go test -race -v`, `--- SKIP` |
| subtests under `-race` | **696** = 579 + 87 + 30; `profiler`'s 579 = 546 depth-1 + 33 depth-2, depth-3+ = 0 | `go test -race -count=1 -v <pkg>`, `=== RUN` minus top-level |
| coverage | **97.9 / 94.7 / 97.0** | `go test -cover ./...` |
| — and the trap | `go test -race -cover ./cmd` → **3.4%** | `-race` breaks `GOCOVERDIR` propagation |
| — homesafe detail | only `PathContains` **90.0%** and `resolve` **96.8%** short of 100 | `go tool cover -func` |
| assertion call sites | **761**; `files=7 lines=9397 accounted=9397` | `bash tests/lib/audit-suites.sh tests/test_*.sh \| tail -1` |
| — other side | `wc -l tests/test_*.sh` → **9397** | equality holds |
| OTLP fixtures | **64** | `ls profiler/testdata/otlp/*.json profiler/testdata/otlp/*.ndjson \| wc -l` |
| DuckDB queries | **8**, all 8 exit 0 and return rows on DuckDB **v1.5.5** | `SPOOL_DIR=… duckdb -c ".read <q>"` ×8 |
| v0.4.3 baselines | **2499 / 93 / 452 / 59** | `git archive v0.4.3` tree, suites + `go test -list` + `-race -v` + `ls` |
| skillscore | head **92.5 (A-)** / **89.5 (B+)**; v0.4.1 **94 (A)**; v0.4.3 **92.5 (A-)** | `skillscore <dir> --json`, `overallScore.{percentage,letterGrade}`, `skillscore --version` → 2.0.2 |
| ledger rows naming 0.5.0 | **15**, all marked OPEN in the ledger; notes say 13 remain open after the re-walk | `awk -F'\|'` over the ledger's Home column |
| subcommands | **six new** — `compare`, `experiment`, `hooks`, `ingest`, `analyze`, `doctor` | `profiler -h` at head vs a `v0.4.3` build |

**Two figures I refused to publish.** `tests/lib/harness.sh:162` says "819 assertions"
compute their verdict inside a command substitution; `grep -c assert_value` over the seven
suites gives **402**. I could not reproduce 819 by any derivation I tried, so the PR body
now says "every assertion that computes its verdict inside a command substitution" instead
of quoting the number. It is a Lane 1 file and I did not edit it — **escalated below.**
Likewise I did not carry the dogfood lane's `8,000,143`-byte line length: my own 8 MB round
produced `8,000,206`, because the payload differed. The published figure is mine.

## RED / GREEN / RED — the three guards my prose now claims, driven by me

Prose has no unit test, so the non-vacuity evidence available is the guards that stand
behind the *class*. All three were driven in the throwaway worktree at `d904c30`.

| guard | RED | GREEN |
|---|---|---|
| the assertion audit's derived denominator | limit injected into the reader → `sites=218 files=7 lines=2594 accounted=2594`, `test_install.sh` at **zero**, **7 FAILs**, `184 passed, 7 failed`. **218 clears the old floor of 200** — that is the whole point | audit restored → `sites=761 lines=9397 accounted=9397`, `191 passed, 0 failed` |
| the release-document guard | both 0.5.0 sections deleted → `82 passed, 4 failed`: "CHANGELOG.md has a section for the release the manifests declare", "…says something", same two for RELEASE_NOTES | sections restored → `86 passed, 0 failed` |
| the forward-promise check | `- A template for \`Constraints\` is deferred to 0.5.0.` planted in the 0.5.0 RN section → `RELEASE_NOTES.md:5: [deferral verb] …`, check exit 1, `FAIL: no tracked file assigns work to the release being cut`, `85 passed, 1 failed` | line removed → `86 passed, 0 failed` |

The first row is the one that matters: the RED was produced against the *defect* (a reader
that stops early), not merely against old code, and the injected count is above the floor
the old assertion used, so the old assertion would have passed.

---

# P-items, dispositions, and the command that proved each new sentence

## P1 · CRITICAL · the redaction claims were inverted — **FIXED, re-derived**

Driven at the CLI, one payload through both modes:

```
default: {"api_key":"[REDACTED]","my_token":"[REDACTED]",
          "command":"psql -h db.internal -U admin",
          "prompt":"my secret business plan",
          "tool_input":{"file":"/repo/x.go"},"tool_output":"rows: 3",
          "user_email":"matt.vandusen@okja.io"}
--strict: command/prompt/tool_input/tool_output → {"_stripped_bytes": N};
          api_key/my_token still [REDACTED]; user_email still present
```

and value-level redaction in place:
`"command":"curl -H \"Authorization: [REDACTED]\" https://x"`, `"note":"[REDACTED]"` for an
`sk-` key of 16+ chars.

So: **three** transformations on the write path (key order, duplicate keys, credential
redaction), not two; credential redaction is always on; content stripping is `--strict`
only. Both documents now state the split the way `docs/profiler-spec.md`'s five-bullet
contract already had it, and the byte-fidelity sentence in both documents and at
`README.md`'s spool summary now names credential removal.

Compounding facts, each driven:

- **`hooks install` registers no `--strict`.** Registered command on a virgin home:
  `<abs>/profiler ingest --spool-dir <home>/.skill-architect/spool || true`,
  `any --strict: False`, 21 events. There is no environment variable. Both documents now
  say metadata-only mode is reachable only by hand-writing `--command`, and the README
  spells the command out. I drove that recipe end to end: the registered strict command,
  run through `sh` with `HOME` pointed at a decoy, wrote
  `{"…","strict":true,"raw":{…,"prompt":{"_stripped_bytes":25},"user_email":"x@y.z"}}`.
- **`--strict` writes `user_email`.** Confirmed in the payload above *and* in Cursor's own
  reference, which lists `user_email` among the fields present on **all** hooks. Disclosed
  in all three documents. The string still appears nowhere in the repo.
- **`analyze` emits `payload_key_values`.** Driven: a one-line spool produced
  `"cwd": {"/repo": 1}` and `"conversation_id": {"c1": 1}`. `analyze.go`'s
  `maxCensusedValueLen = 64`, content keys excluded, numbers excluded. Disclosed in both
  release documents with the same "read it before you publish it" caution the experiment
  result already had, and the asymmetry the worklist flagged is closed.

## P2 · HIGH · `README.md:315` told the reader to disable the flag the adapter reads — **FIXED, the highest-value change in the lane**

Re-derived at head against the dogfood lane's two live exports, one variable apart:

```
capture --harness claude_code --otel-file run4-skill.otel-export.ndjson  → skill_name: custom_skill
capture --harness claude_code --otel-file run5-skill-tooldetails…ndjson  → skill_name: dogfood-probe
```

and on the wire, from the exports themselves:
`{'skill.name': 'custom_skill', 'invocation_trigger': 'claude-proactive', 'skill.source': 'userSettings'}`
versus the same with `skill.name: dogfood-probe`. The adapter reads `skill.name` at
**`profiler/claude_code.go:788`** (786 at the base; the line moved under Lane 1's commits).

Grounded in the vendor reference as well, fetched raw (160,711 bytes,
`code.claude.com/docs/en/monitoring-usage.md:1016`): *"`skill.name`: Name of the skill. For
user-defined and third-party plugin skills the value is the placeholder `"custom_skill"`
unless `OTEL_LOG_TOOL_DETAILS=1`"*.

`README.md:315` is now the tradeoff it actually is, with the two-row measured table, an
explicit "if you are profiling a skill you wrote, set it", the full list of what the flag
widens (accurate against the reference's own wording), and a note that the old sentence was
true before v0.5.0. The Route (a) recipe gains the flag as a **commented-out** line with
the tradeoff named, rather than silently enabling it.

**I walked the rest of the telemetry recipe for the same shape**, as instructed, and found
two more false sentences, both new in this diff:

- `README.md:213` said an export with the attribute and no event "reports
  `skill_activation: none`, and the reason names the event". Measured on the live run-3
  export: the **capability** says `"skill_activation": "none"`, and the **profile signal**
  says `{"state": "unknown", "reason": "no claude_code.skill_activated log events found in
  OTel export"}`. A capability value carries no reason. Fixed in the README and in
  `docs/profiler-spec.md`'s matching numbered item, with the two vocabularies named once so
  the distinction carries.
- `README.md:224` said `attribution` "is the signal that is still always `none`". Measured:
  the capability is `none`, the signal is `unknown` with a reason. Fixed the same way, and
  the reason itself verified — 30 distinct attribute keys across the live exports, nothing
  maps an output to a skill.

Sentences in that recipe I checked and **left alone** because they hold: `prompt` and
`response` `<REDACTED>` by default (reference: *"Redacted by default. Set
`OTEL_LOG_USER_PROMPTS=1`"*, and the same for responses); the identity attribute list;
"Claude Code has no file exporter"; the console-exporter advice.

**`skill.kind` — I am pushing back on the worklist's diagnosis.** The worklist says the
event "does not carry `skill.kind`" and that "two fixtures assert one the emitter never
sends". The first half is right about my observation and wrong as a general claim: the
vendor reference (`:1019`) says *"`skill.kind`: `"workflow"` when the skill is a workflow
skill. Absent otherwise."* My probe skill was not a workflow skill, so absence is the
documented behaviour, not a missing attribute. The real defect is sharper and I verified it
directly: the fixtures carry **`skill.kind: "skill"`**, which is not a documented value at
all; `invocation_trigger: "slash_command"` and `"skill_tool"`, where the reference
documents `"user-slash"`, `"claude-proactive"`, `"nested-skill"`; and
`skill.source: "user"`, where it documents `"bundled"`, `"userSettings"`,
`"projectSettings"`, `"plugin"`. Three attributes, values outside the vendor's sets.
**No number moves** — `claude_code.go:788-793` reads `skill.name` and passes
`invocation_trigger` through opaquely, and reads neither `skill.source` nor `skill.kind`.
Fixtures are Lane 1's, so **reported, not edited**, in both release documents and in the PR
body. The prose claims only what I observed and what the reference says.

`RELEASE_NOTES.md:90` and `CHANGELOG.md:528` are inside the frozen **v0.4.1** sections
(RN sections at 30/50/69→ `:90` is v0.4.1; CHANGELOG `## 0.4.1` spans 486–535). Left
byte-identical; the correction is in the 0.5.0 sections. Same for `RELEASE_NOTES.md:87`,
which carries the "reads nothing it adds" sentence and was true when written.

## P3 · HIGH · "all three drafts do not compile" — **FIXED, measurement re-run per file**

`git archive d904c30 profiler`, each draft copied in **alone**, `go build ./...`:

```
0a83615 cursor.go alone:  undefined getBool, otelLog, otelMetric, parseTime, toInt
0a83615 codex.go  alone:  undefined otelLog, otelMetric, parseTime, toInt (×5)
0a83615 devin.go  alone:  COMPILES CLEANLY (exit 0)
all three at once:        ./codex.go:224:15: too many errors    ← what truncated the original
```

- The four names were declared at `541af3e:profiler/claude_code.go` (`type otelMetric` :144,
  `type otelLog` :150, `func toInt` :222, `func parseTime` :244) and none at `v0.4.1`, so
  "removed by v0.4.1's OTLP rewrite" is **true for the four**.
- `getBool` is declared at **neither** ref and at no ref I checked, so the rewrite does not
  explain it. Named as a fifth undefined name in both documents.
- Devin's real defect stands and is now stated precisely: `devin.go:37` →
  `SourceSessionData`, and it never returns a count — `UnknownTokenResult` at `:75` and
  `:114`, `ErrorTokenResult` at `:88`, `:98`, `:107`. The old "returns an unknown result on
  every path" was wrong in a way that matters here, because error ≠ unknown is load-bearing
  elsewhere in this release.
- Cursor's over-advertisement narrowed to what the code does: `cursor.go:60` sets
  `SourceSQLite` only when no OTel source supplied tokens **and** the `--export-file` can be
  `stat`ed (the draft guards on `caps[MetricTokens] == SourceNone`), then `:150` answers
  with `UnknownTokenResult("Cursor SQLite token schema not yet verified")`.
- **The ref is named.** The four cited `file:line` pairs resolve at **`0a83615`** and not at
  `origin/archive/0.5.0-wip-tree` (`a3673b8`), where `cursor.go:60` is a blank line and the
  draft reads `profileFromSpool` instead of `getBool`. `0a83615` is on
  `origin/wip/0.5.0-profiler`. Both documents now say so.
- The decision to delete all three stands, and both documents now say the **advertisement**
  is the reason rather than the compile failure.

## P4 · HIGH · the upgrade consequence nothing disclosed — **FIXED, both halves, both documents**

```
compare --baseline <0.4.3 profile> --candidate <0.5.0 profile>  → exit 2
refusal: adapter version mismatch: baseline "0.4.3" vs candidate "0.5.0" — …
every metric comparable: false
AdapterVersion:  v0.4.3 types.go:274 = "0.4.3";  head types.go:433 = "0.5.0"
```

A new bullet in `RELEASE_NOTES.md` and a paragraph on the `AdapterVersion` entry in
`CHANGELOG.md`. 0.4.3's own precedent quoted and verified verbatim in its frozen section
(*"A stored 0.4.2 profile beside a fresh 0.4.3 one is not a regression; it is the fix"*).

Second half, the value domain. Driven: an unreadable export gives
`skill_activation: error` alongside the other three. `ErrorActivationResult` exists at head
and not at `v0.4.3`, whose `types.go` said *"has no Error constructor: skill activation is a
property of the harness and of this adapter, not of any export, so no export can fail it"*
— quoted and checked. Stated with the changelog's own bumping rule beside it, shipped as
v1, and **flagged for the maintainer rather than decided quietly** (see escalations).

Also here, driven: the capability report went **5 → 6** keys (`probe` at a `v0.4.3` build
vs head), because `estimated_context_tokens` is hard-set to `SourceNone`. It is the one
added surface that is neither optional nor absent, so it is stated separately from the
nine-keys sentence rather than folded into it.

## P5 · MEDIUM · every counted enumeration that was short — **FIXED as a class**

Per the mandate I preferred a derivation or no numeral over a corrected numeral. Row by row:

| claim | written | disposition |
|---|---|---|
| deferral markers re-pointed | 4, listing 3 | **numeral dropped.** Replaced by the guard: `forward-promise-check.sh` over `git ls-files` (162 files at head, reported on its own last line), with the topics still open named and the one that closed named |
| keys this release adds | 8 | **derived and merged with the row below.** `json:` tag set-diff `v0.4.3`→head = **9**: `category, confidence, count, detail, error_type, estimated_context_tokens, id, operation_name, total`. All nine named |
| profile keys written by nothing | 3 | **same nine.** Derived: the only shipped `ToolCallEntry{…}` (`claude_code.go:731`) sets `Name/Timestamp/Success`; `PresentAttributionResult` and `PresentEstimatedTokensResult` have no non-test caller; `Category/Detail/OperationName/Confidence/ErrorType` appear in no non-test file outside `types.go`. One true sentence now replaces two false numerals |
| `-o` destinations refused | 4 | **4 is correct at head, and the root list is six.** Driven: symlink → 1, directory → 1, `SKILL.md` → 1, each of `.claude/.cursor/.codex/.devin/.config/.agents` → 1; `.claude-notes` → 0; `$HOME/drafts/../skills/x.md` across a symlink → 1; `chmod 000` parent → 3. The spurious fifth (existing regular file) is gone: two runs to the same `-o` both exit 0 |
| runs to reach the backup collision | 3 | **2, driven**, and the surviving `.bak` holds the post-install state, so `cmp` against the original says NO. Corrected in the changelog and **added to the release notes**, which had never carried it — the one limit of nineteen that diverged |
| things deliberately not shipped | "a Cursor adapter" | **three adapters**, named, in both documents' ledes |
| `probe` stdout | "byte-identical" | **corrected.** A readable export differs (`otel` vs `none`); the six ways of reading nothing give **one** report ignoring `probed_at`. Both facts stated, and the stderr-is-the-differentiator consequence named |
| `analyze` "says in its own output that it yields no measurement" | — | **false, fixed.** The `yields` sentence is `doctor`'s `observed.spool.yields`, driven; `analyze` has **16** fields and no such statement. `README.md` now points at `doctor` and says why it matters for a pasted summary |
| `experiment` "three refusals at three times" | 3 | **four times, driven**: design (cross-harness, exit 1, message verbatim), plan document (`no runs`; `both steps write the same profile path`, exit 1), run (`exited 0 and wrote no profile`, exit 1), read-back (`its snapshot_hash is "WRONG" and the plan declared "a"`, exit 1). The design check is described as a table rather than a quotable list |
| mutation runner "five consecutive slices" | 5 | **numeral dropped**, with the reason: nothing in this tree derives it |
| nanosecond epoch "1.7×10¹⁸" | 1.7 | **1.8**, with the fixture literal `1789332594304000000` and 2⁵³ named |
| "275 … pass" | 275 | **285, of which 284 pass and one skips** — the total no longer claims what its parts do not |

Undisclosed shipped surface added: `compare` exit **2** and `experiment run` exit **2**
(both driven); `compare` treating **`harness`** and **`skill_dir`** as notes (driven: exit
0, `notes: ["harness differs: baseline \"claude_code\" vs candidate \"cursor\""]`), so only
`adapter_version` refuses; the cross-harness design refusal; the plan-time refusals;
`experiment run`'s freshness-stamp hole; `duckdb` as a new external tool absent from
Prerequisites and run by no test or CI step (8/8 by hand on v1.5.5); `-o ""` falling through
to the default destination (driven: `Rewrite draft written to: <T>/skill/REWRITE-DRAFT.md`,
exit 0).

**On the two rows the mandate told me not to merge:** I did not. C8's five `MetricSource`
*values* are recorded as a Fixed entry about the `source` **vocabulary** and its AST guard;
the "written by nothing" sentence is about the nine profile **keys**. Different sentences,
different sections, neither citing the other's count.

## P6 · DECISION · the Codex install contradiction — **RULED: the README row is true; L22's published narrative is narrowed**

The two claims:

- `README.md:726`: Codex row, *"verified — both steps run against `codex-cli 0.153.4` with
  `CODEX_HOME` pointed at a scratch directory. `codex plugin list` then reports the plugin
  installed and enabled"*.
- `RELEASE_NOTES.md:21` / `CHANGELOG.md:304`, ledger row **L22**: *"the Devin, Codex and
  Cursor install rows being unverified"*.

What I checked:

1. The three install-table rows at head (`README.md:725-727`) are **byte-identical** to
   `v0.4.3:README.md:435-437` — `cmp` on the extracted ranges says identical. So the Codex
   row is not new prose in this release; it is a v0.4.3-era record that 0.5.0 carried
   forward unchanged, exactly as `README.md:715-716` says it should be (*"the status of the
   route as executed in 0.4.3, the release that ran them"*).
2. The ledger's own row: `L22 | Devin / Codex / Cursor install rows unverified | 0.4.1 R1 fnf #3 | OPEN`,
   and the narrative paragraph beside it points at `README.md:329-337`.
3. `codex` is **not on PATH** here, so I cannot re-run the route and cannot falsify the row.

**Ruling: the README row is the true claim, and L22's published narrative is the false one.**
Reasoning, in order of weight:

1. **The README row is evidence and the ledger row is bookkeeping.** It names a CLI version
   string, the isolation used (`CODEX_HOME` at a scratch directory), the confirming command,
   and that command's result. That is the shape of a measurement. L22 names no run, no
   version and no date; it is a tracking row that was opened at 0.4.1 R1 and carried.
2. **L22 was already false about Codex against the tree it was audited over.** The ledger
   was audited at `5c847e1`, before v0.4.3 shipped, and the v0.4.3 README already said
   "verified" for Codex — I read it off the tag. So this is not a 0.5.0 regression; 0.5.0
   re-published a claim that was wrong when it was written.
3. **The repository's own convention decides which document owns the fact.** `README.md`
   is the document that records per-route install status "as executed", by release. A
   deferred-ledger row does not overrule it.
4. **The direction of the correction is the safe one.** Narrowing a published "three rows
   unverified" to "two rows unverified" is a claim I can support from a document in the
   release; widening it to call the README row false would require asserting that a recorded
   run did not happen, which I have no evidence for and no way to test here.

So: both release documents now say **"the Devin and Cursor install rows being unverified"**,
with a sentence naming why Codex is excluded and what the README row records. The row stays
**open** — Devin and Cursor genuinely are unverified and each says why — so the count of 13
open rows is unchanged. The README row is left byte-identical.

**What the maintainer still owns, and I am not doing it silently:** the ledger row itself
(`.scuba/teams/gap-audit-0.4.3/deferred-ledger.md:53` and the L20/L22 paragraph at `:343`)
still carries the three-harness wording, and it is outside my file list. Either narrow the
row to Devin + Cursor, or record that the Codex route was verified at 0.4.3. Escalated
below. This does **not** block the tag: the published documents no longer contradict each
other or the README.

## P7 · disclosures — all added, each driven

- **Tool-call hook payload shapes unverified against a live emitter.** Stated as the user's
  decision directs, with **both halves precise**: `preToolUse`/`postToolUse` were registered
  and never fired (a nested `claude -p` could not authenticate), while the lifecycle hooks
  and the whole `ingest` → spool → `analyze` → `doctor` path **were** verified live, against
  a real hook emitter that was not Cursor, with every count checked against the raw file.
  In all three prose files.
- **`AppendSpool` reworded, not deleted.** My own run, at head:

  | payload | procs | lines (expected) | unparseable | file size | distinct line lengths |
  |---|---|---|---|---|---|
  | 100 B | 40 | 40 (40) | 0 | 12,280 | 1 (306) |
  | 200 KB | 40 | 40 (40) | 0 | 8,008,280 | 1 (200,206) |
  | 8 MB | 16 | 16 (16) | 0 | **128,003,312** | **1 (8,000,206)** |

  With the structural reason (`hooks.go`: `os.OpenFile(..., O_APPEND|O_CREATE|O_WRONLY)` then
  **one** `f.Write(append(line, '\n'))`, a fresh descriptor per call), concurrent firing
  named as normal (Cursor's reference lists Tab hooks as a class separate from its agent
  hooks), and the **untested NFS/SMB caveat** named in place of the old "could in principle
  interleave" hazard. No Go comment carries this disclosure, so no sibling went out of sync —
  checked with a tree grep for `no lock|locking|O_APPEND|interleav`.
- **The `[DOCS]` sentence, both directions.** Verified myself against
  `cursor.com/docs/agent/hooks`: `~/.cursor/hooks.json` is the **User**-scope location, with
  Enterprise/Team/Project above it; the common payload schema is
  `conversation_id, generation_id, model, model_id, model_params, hook_event_name,
  cursor_version, workspace_roots, user_email, transcript_path` — **no `cwd`**, which the
  docs put on `preToolUse`, `postToolUse`, `beforeShellExecution` only; `version` required,
  *"Must be a positive integer (use 1)"*; and the event list, which I compared
  programmatically against the installed set: **21 = 21, no difference either way.** So:
  location and names **confirmed**, `cwd` **contradicted**, `version` now written, and
  *"We cannot tell you it works against Cursor"* **kept** verbatim in spirit in all three
  files, because Cursor is still not installed.
- **`--strict` writes `user_email`** — driven, and corroborated by the reference. Disclosed.
- **`hooks uninstall` on a virgin home.** Driven: leaves `.cursor/`,
  `hooks.json` = `{"hooks": {}, "version": 1}`, and `hooks.json.bak-<stamp>` with our
  absolute path in all 21 entries. **P7's bullet changes as the mandate says:** the leftover
  is schema-**valid** now, so the disclosure says "not as-found" and drops the
  "schema-invalid" hazard the dogfood lane inferred. In all three files.
- **The activation provenance sentence cites the attribute the adapter does not read.**
  Driven: `profiler/testdata/otlp/README.md` carries labels on exactly two lines,
  `[OBSERVED]` and `[DOCS]`; `skill.name` is in the `[DOCS]` list; `claude_code.skill_activated`,
  `invocation_trigger`, `skill.source` and `skill.kind` are in **neither**; and both labelled
  paragraphs are unchanged in `git diff v0.4.3..HEAD` (the file changed only in its fixture
  table). Stated in both release documents, together with the three out-of-vocabulary
  fixture values above. `tool_name`/`success`/`decision` relaxed to **observed** — I read
  them off the live run-3 export myself: `"Read"`, `"true"`, `"accept"`, each a
  `stringValue`, out of 30 distinct attribute keys.
- **`experiment run`'s freshness stamp** accepts a re-touched stale profile. Disclosed in
  both documents; the refusal it *does* make is driven, and its own message names the hole
  (*"or the file left there is an earlier run's"*).
- **The same-second backup collision** added to `RELEASE_NOTES.md`, which had never carried
  it, and corrected from three runs to two in `CHANGELOG.md`.
- **PR body: `94 (A) → 92.5 (A-)`** — fixed, and verified by measurement
  (`skillscore` at `v0.4.1` → `94 A`, at `v0.4.3` → `92.5 A-`, at head → `92.5 A-`).

---

## DEFERRED items I took, because they are disclosure-shaped and in my files

- `README.md:26,28` — the profiler is no longer introduced as a "preview" attributed to
  v0.4.0; it names the six subcommands 0.5.0 added and says the preview label is what
  earlier editions used.
- `README.md:34` — Status `✅ Slice 1` → `✅ Ships`. Checked first that no test reads the
  string (`grep -rn 'Slice 1' tests/` → nothing; the three `| Planned |` rows DoD #4 asserts
  are untouched).
- `docs/profiler-spec.md:3` — header was 8 days and 5 slices stale and said `Status: spec`;
  now `Status: reference for shipped behaviour · Date: 2026-09-21 · Covers: v0.5.0`, with an
  opening note saying what the document became.
- `docs/profiler-spec.md:7` — Purpose claimed all four harnesses in the present tense. Now
  says one adapter ships, with the refusal driven:
  `capture --harness cursor|codex|devin` → `unknown harness: <n> (supported: claude_code)`,
  exit 1, all three.
- `docs/profiler-spec.md:776,778,844` — the "(Slice 1)" scope labels over content spanning
  four releases, with AC2/AC4's activation clauses being 0.5.0's own normative criteria. The
  labels are gone and the AC section says what it covers.
- `docs/profiler-spec.md:159-160` (F9) — the spec's copy of the reserved `CaptureOpts`
  fields was **silent** where `types.go` says plainly that nothing in this release reads
  either. Matched to the Go comment's words, including its reason for naming no release.
- `-o ""` falling through to the default destination — driven, disclosed in both documents.
- `duckdb` absent from Prerequisites — disclosed in all three, with the reason it is *not*
  added to that list: the list is the skills' ten preconditions and is held in both
  directions by `tests/test_f01.sh`, a Lane 1 file.

## Left alone deliberately

- **The two sentences the mandate protects.** `docs/profiler-spec.md`'s *"0.5.0 did not add
  one"* (three instances in README, one in the spec) and *"needs a channel 0.5.0 did not
  add"* are untouched; `grep` confirms they still read as before, so the divergence Lane 1
  closed stays closed. I checked for a Go sibling before every spec sentence I did change.
- **`docs/profiler-spec.md:100,130` and `types.go:69`** — the `Harness` field's declared
  vocabulary `"cursor" | "claude_code" | "codex" | "devin"`. Three of the four are names
  `NewAdapter` refuses, but this is a *declared schema vocabulary*, the same shape as
  `MetricSource`'s values, and the spec's copy is byte-matched to a Go comment Lane 1 left.
  Changing one side only would re-open a divergence. Reported instead.
- The three install-table rows, byte-identical to v0.4.3, including the Codex row.
- Both documents' frozen sections, verified byte-identical afterwards.
- Every Go file, `tests/**`, and `skills/**` — including the two `skill.kind` fixtures and
  `tests/lib/harness.sh:162`'s "819".

---

## Escalated to the manager — five, and what each costs

1. **P6's ledger row is still wrong, in a file that is not mine.**
   `.scuba/teams/gap-audit-0.4.3/deferred-ledger.md:53` and the paragraph at `:343` say
   "Devin / Codex / Cursor install rows unverified". The published documents no longer
   repeat it. The maintainer should either narrow the row or record the Codex verification.
   One line. **Does not block the tag.**
2. **A schema judgment the maintainer may want to take rather than inherit.**
   `skill_activation` gained `error`, widening a key's value domain that v0.4.3 documented
   as complete and asserted in a code comment could not fail. By the changelog's own rule
   ("v2 is required when a v1 reader would be wrong") this is the closest call in the
   release. It ships **v1**, and both documents now say so explicitly and name the tension.
   If the answer is v2, that is a code change and a new lane.
3. **Two `skill.kind` fixtures and two `invocation_trigger` values assert an emitter shape
   the vendor does not document** (`skill.kind: "skill"` vs documented `"workflow"`/absent;
   `slash_command`/`skill_tool` vs `user-slash`/`claude-proactive`/`nested-skill`;
   `skill.source: "user"` vs the four documented values). No number moves. Fixtures are Lane
   1's territory and Lane 1 is closed, so this is **reported, not fixed**, in both documents
   and the PR body. It wants a 0.6.0 ledger row rather than a gate edit.
4. **`tests/lib/harness.sh:162`'s "819 assertions" does not reproduce.** `grep -c
   assert_value` over the seven suites gives 402, and I found no derivation that yields 819.
   It is a hand-written numeral in a Lane 1 file standing behind a non-vacuity story, which
   is precisely the class C5 was opened for — and C5 converted the audit's floor while
   leaving this figure. Worth one look before the tag; I did not publish the number.
5. **The three adapter drafts are reachable only from `0a83615`, on
   `origin/wip/0.5.0-profiler`** — the tip of the branch the primary tree sits on, so a
   routine push moves the point the changelog's citations resolve against. My prose names
   the commit sha as well as the branch for that reason, but M3's recommendation stands and
   is one command: an immutable tag for the frozen WIP. **The maintainer's, not mine.**

## Threads replied to / resolved

**None, because there are none.** Verified live at my head rather than from Lane 1's
report: `pulls/22/comments?per_page=100` → `0`, `issues/22/comments?per_page=100` → `0`,
`pulls/22/reviews?per_page=100` → `0`. No external reviewer thread exists to reply to, and
nothing was resolved. `gh pr checks 22` → **pass**. Per-PR `mergeable: true`,
`mergeable_state: blocked`, head pinned at `64de8c0`.

## Commits, in order

```
7dcd67b  The release documents say what the code at this head actually does
64de8c0  README and the spec stop telling a reader to defeat the signal 0.5.0 added
011defa  The ledger bullet's CLI coverage figure is 94.7% at this head, not 94.5%
```

The third commit is the one found by the closing sweep: I enumerated **every** numeral in
both 0.5.0 sections (`awk` the section out, `grep -onE '[0-9]+\.[0-9]%|[0-9]{3,4}'`, sort
unique) and traced each to a command. One survived — the L21 re-walk sentence still said the
CLI coverage is 94.5%, which was true at the release commit and is 94.7% after Lane 1's
tests. Every other numeral resolved to a measurement in the tables above, to a `file:line`
I opened, to a commit sha, or to a file mode.
