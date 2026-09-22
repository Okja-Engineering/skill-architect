# Final PROSE lane (2b) — record. PR #22, `release/0.5.0`

**Base:** `c3a565c` · **Head:** `bf045ce` · **Worktree:**
`…/scratchpad/lane2b` · **One commit**, prose only.
**Files:** `CHANGELOG.md`, `RELEASE_NOTES.md`, `README.md`, `docs/profiler-spec.md`, and the
PR #22 body. **No code, test or script file touched** — `git status --porcelain` shows
exactly those four paths; `git diff --shortstat` is `4 files changed, 326 insertions(+),
122 deletions(-)`.

**Verdict: all 13 findings dispositioned — 12 REAL and fixed, 1 REAL with its prescription
refused and rewritten from git (F1). Two code-side residuals reported, not edited.**

---

## The rule this lane was given, and how it was applied

Every count or set claim in the shipped prose is now the output of a command run at this
head, recorded beside it. Where a numeral would only go stale next release, it was
**dropped rather than corrected** — two of them were. Where a sentence described a
mechanism that no longer exists, the sentence was rewritten rather than renumbered.

The one place this bit back is worth recording, because it is the guard working. Writing
F4's evidence into the changelog meant spelling a lowercase deferral verb beside the
version being cut — and `tests/lib/forward-promise-check.sh` **correctly turned the build
red on my own prose**:

```
$ bash tests/lib/forward-promise-check.sh 0.5.0
RELEASE_NOTES.md:26: [deferral verb] - **Every deferral marker that read "0.5.0" is gone …
files=162 lines=43507 examined=43142 skipped=365
rc=1
```

Both documents now describe the caught shape without spelling it, and say why. The
changelog's copy had survived only because a line break fell between the verb and the
version — a reflow would have reddened CI later, so it was rewritten too.

```
$ bash tests/lib/forward-promise-check.sh 0.5.0 | tail -1
files=162 lines=43512 examined=43147 skipped=365
rc=0
```

---

## Every number as shipped, with the command that produced it

Run in `…/scratchpad/lane2b` at `bf045ce` unless a ref is named.

| figure | command | value |
|---|---|---|
| shell assertions, bash 3.2.57 | `for s in tests/test_*.sh; do /bin/bash "$s" \| tail -1; done` | **3112**, 0 failed |
| shell assertions, bash 5.3.15 | same under `/opt/homebrew/bin/bash` | **3112**, 0 failed, identical per suite |
| — per suite, both shells | as above | `test_f01` **1912** · `test_f02` **350** · `test_harness` **276** · `test_install` **52** · `test_rewrite` **401** · `test_skill` **101** · `test_walk` **20** |
| top-level Go tests | `go test -list '.*' <pkg> \| grep -c '^Test'` | **286** = 239 + 35 + **12** |
| — pass / skip | `go test -race -v` per pkg, `^--- PASS` / `^--- SKIP` at top level | **285 pass + 1 skip**, 0 fail |
| subtests under `-race` | `grep -c '^=== RUN'` minus top-level, per pkg | **1014** = 579 + 87 + **348** |
| — depth split | `^=== RUN   Test[^/]*/[^/]*$` vs `…/[^/]*/` | `profiler` 546 direct + 33 nested; `cmd` 87 + 0; `homesafe` 28 + **320** |
| coverage | `go test -cover ./...` | **97.9 / 94.7 / 95.9** |
| — homesafe functions | `go tool cover -func` | `PathContains` **87.5%**, `resolve` **95.8%**, all others 100% |
| — the uncovered statements | coverage profile, blocks with count 0 | **five**, all error returns: `homesafe.go:189`, `:199`, `:215` (`os.Stat`), `:372` (`os.Getwd`), `:417` (`os.Readlink`) |
| assertion audit | `tests/lib/audit-suites.sh tests/test_*.sh \| tail -1` | **`sites=821 files=9 lines=11749 accounted=11749`** |
| — the closure, named | `tests/lib/audit-suites.sh --closure tests/test_*.sh` | 7 suites + `tests/lib/masked-path.sh` + `tests/lib/mutation-runner.sh` |
| harness names, two sides | `audit-suites.sh --names tests/lib/harness.sh \| wc -l` | **14** |
| guard primitives, two sides | `audit-suites.sh --names skills/skill-audit/scripts/verdict-guard.sh \| wc -l` | **16** |
| floors remaining | `grep -n -- '-ge [0-9]' tests/lib/audit-suites.sh tests/test_harness.sh` | **none in code** — three comment lines explaining their removal |
| generated definition spellings | `tests/test_harness.sh` own output | **14** (12 defining, 2 not) |
| generated EXIT-trap spellings | same | **13** (10 displacing, 3 leaving) |
| forward-promise reader | `tests/lib/forward-promise-check.sh 0.5.0 \| tail -1` | `files=162 lines=43512 examined=43147 skipped=365`, rc 0 — **no numeral published** |
| tracked files | `git ls-files \| wc -l` | **163** |
| — non-empty | per-file `[ -s ]` over `git ls-files` | **162**; the one empty file is `profiler/testdata/otlp/empty.json` |
| — files with no final newline | `tail -c 1` per tracked file | **two**: `malformed.json`, `truncated_final_line.ndjson`; `grep -ac ''` total 43401 vs `wc -l` 43399 at the time of measuring, a difference of exactly 2 |
| `0.5.0` occurrences | `git grep -o "0\.5\.0" -- . \| wc -l` | **99** across **88** lines, and the check reports no findings → **all 99 correct** |
| OTLP fixtures | `ls profiler/testdata/otlp/*.json *.ndjson \| wc -l` | **64** (unchanged, not touched) |
| DuckDB queries | `ls profiler/queries/*.sql \| wc -l` | **8** (unchanged, not touched) |
| — run by tests or CI | `grep -rln duckdb tests/ .github/` | **0 files** — the disclosure holds |
| skillscore, head | `skillscore skills/…` under 2.0.2 | `skill-audit` **92.5 (A-)**, `skill-rewrite` **89.5 (B+)** |
| skillscore, v0.4.1 | same on a `v0.4.1` worktree | `skill-audit` **94.0 (A)** |
| skillscore, v0.4.3 | same on a `v0.4.3` worktree | `skill-audit` **92.5 (A-)** |
| 0.4.3 shell baseline | each suite on a `v0.4.3` worktree | **2499** = 1868/350/75/85/53/48/20 — published figure **confirmed** |
| 0.4.3 Go baseline | `go test -list` and `-race -v` on the same | **93** top-level, **452** subtests — **confirmed** |
| frozen CHANGELOG tail | `tail -n +786 CHANGELOG.md \| md5` | **240** lines, `ef2fe48af306b4ce06ace48391eddcb8` = `v0.4.3` |
| frozen RELEASE_NOTES tail | `tail -n +36 RELEASE_NOTES.md \| md5` | **112** lines, `46d66b1e331726dbe56a830d449c19aa` = `v0.4.3` |
| go build / vet / gofmt | `cd profiler && go build ./… && go vet ./… && gofmt -l .` | clean / clean / no output |
| go test | `go test -count=1 ./...` | ok ×3 |
| skill-validator | `skill-validator check … -o json` | both: `errors 0 warnings 0 passed True` |
| commits on the branch | `git rev-list --count 10326b3..HEAD` | **27**, author and committer `imagineux` on every one, `%(trailers)` empty, attribution grep 0 |
| diff vs base | `git diff --shortstat 10326b3..HEAD` | **31 files, +6842 −523** |

### Two figures dropped rather than corrected

1. **The forward-promise reader's file count.** The published "every tracked file
   (`git ls-files`, 162 of them)" paired a `git ls-files` of **163** with the *non-empty*
   count of 162, because awk's `FNR == 1` cannot fire for a zero-byte file. Rather than
   print 163 beside a reader that reports 162, both documents now describe the **two-sided
   comparison** with no numeral: files against the non-empty tracked files, records against
   `grep -ac ''` over the same files, and `examined + skipped == lines` refused by the
   reader itself.
2. **The concurrency round's absolute bytes.** `git grep "128003312\|8000206"` matches only
   the two release documents — no test, script or fixture pins the payload, so a reader
   cannot regenerate them. Re-driven here at 16 × 8 MB with a payload of my own:
   `writers=16 lines=16 bytes=128003088 distinct_lengths=1 (8000192) unparseable=0` —
   different absolutes, same structure, same `16 × (len + 1)` relation. The **structural**
   result is kept (one line per writer, every line parsed, one distinct length, nothing
   split or interleaved) and the absolutes are gone.

---

## Findings, one by one

### F1 · `getBool` — REAL, and **the prescribed fix was refused**

The published sentence ("never declared in this repository at all") is false, as the
finding said. But the prescribed replacement — that `getBool` was *"removed by this
release's own rewrite of `claude_code.go`"*, and is *"the one of the five whose
disappearance this release caused"* — **is also false**, and git says so:

```
$ git show 0a83615:profiler/claude_code.go | grep -n "func getBool"
521:func getBool(m map[string]any, key string) bool {

$ git log --all --oneline -S"func getBool" -- profiler/claude_code.go
a3673b8 archive: the 0.5.0 working tree, as frozen before any slice was cut
1e4e845 F03 Slices 2-4: Cursor, Codex, Devin profiler adapters

$ git merge-base --is-ancestor 1e4e845 c3a565c ; echo $?
1                                  # NOT an ancestor
$ git merge-base 0a83615 c3a565c
30f374c
$ git log c3a565c --oneline -S"getBool" -- .
7dcd67b The release documents say what the code at this head actually does   # docs only
```

So `getBool` **never existed in any Go file on this release's lineage.** No commit here
removed it; the branch that declared it is the one that did not land. The dates confirm the
shape: `541af3e` (Sep 9) declares the four other names, `30f374c` (Sep 9) still has them and
is the fork point, `1e4e845` (Sep 10) adds the drafts *and* `getBool` on the WIP branch,
and `v0.4.1` (Sep 13) has none of the five. v0.4.1's rewrite therefore accounts for the four
and **cannot** account for `getBool`, which postdates the fork — which is the true version
of the original sentence's instinct.

Also driven: at `0a83615`, `getBool` appears in exactly two files — its declaration in
`claude_code.go` and one use at `cursor.go:228` (`Success: !getBool(log.Attributes,
"is_error")`), so "read by nothing else" holds.

Shipped in both documents as that history. **I did not write the prescribed sentence**, and
this is the evidence for refusing it.

### F2 · Cursor's `cwd` — REAL, confirmed against the live page

I did **not** take the summary on faith, and I am glad: `WebFetch`'s summarising model
answered *"`postToolUseFailure` … its payload does NOT include a `cwd` field"*, which is
wrong. Fetching the raw page and reading the payload blocks directly:

```
$ curl -sSL https://cursor.com/docs/agent/hooks   # 200, 1,300,537 bytes
# `cwd` appears in exactly four per-event payload blocks:
preToolUse            "tool_use_id":"abc123", "cwd":"/project", …
postToolUse           "tool_use_id":"abc123", "cwd":"/project", "duration":5432 …
postToolUseFailure    "tool_use_id":"abc123", "cwd":"/project", "error_message":"Command
                      timed out after 30s", "failure_type":"timeout"|"error" …
beforeShellExecution  "command":"<full terminal command>", "cwd":"<current working
                      directory>", "sandbox":false
```

And `cwd` is **absent** from the reference's own "Input (all hooks)" block, which lists
`conversation_id`, `generation_id`, `model`, `model_id`, `model_params`, `hook_event_name`,
`cursor_version`, `workspace_roots`, `user_email`, `transcript_path`.
`postToolUseFailure` is among the 21 events we register
(`profiler/hooks_install.go:47-58`). Corrected in all four documents; the word "only" is
gone, and the count is four with three of them ours.

**Also re-driven while there**, because "21 events, no twenty-second" is a set claim of the
same class: all 17 event names appearing as `hooks` config keys in the reference are in
`CursorHookEvents`, none is missing from it, and the remaining four
(`afterAgentResponse`, `afterAgentThought`, `beforeReadFile`, `postToolUseFailure`) each
have their own documented section. **The claim holds; left alone.** The `version` quote —
*"Config schema version. Must be a positive integer (use 1)."* — is verbatim on the page.

### F3 · the stale registration string — REAL, fixed in prose; code side reported

```
$ python3 -c "import json;print(json.load(open('/tmp/v2b/.cursor/hooks.json'))['hooks']['preToolUse'])"
[{'command': '/tmp/prof2b ingest --spool-dir /tmp/v2b/.skill-architect/spool || true'}]
$ sed -n '569p' profiler/cmd/main.go
	return self + " ingest --spool-dir " + profiler.SpoolDirIn(home) + " || true"
```

The changelog bullet no longer writes the literal out at all — that is what let it go stale
while the `--home` bullet two below it was right. It now says the command is built in one
place in `main.go` and read through the same reader, and names its own past error.

### F4 · the guard's case-sensitive matcher — REAL at head, disclosed (code not mine)

Driven at `c3a565c`/`bf045ce` over one-line scratch files:

```
rc=1  the caveat channel is deferred to <version>.      -> [deferral verb]
rc=0  Deferred to <version>.                            -> files=1 lines=1 examined=1
rc=0  Tracked for <version>.                            -> files=1 lines=1 examined=1
rc=0  **Deferred to <version>**: a receiver subcommand. -> files=1 lines=1 examined=1
```

The vocabulary at `tests/lib/forward-promise-check.sh:207` is lowercase-only, and all 18
fixture cases at `tests/test_skill.sh:772-789` are lowercase, so no control holds that half.
**This is a code defect and outside this lane.** What I could do, and did: both documents now
state the boundary — the match is case-sensitive, it closes the five lowercase spellings this
repository has actually used five rounds running, and it is *not* a check on the class.
Overclaiming it was the original defect; the guard now has an honest published scope.

### F5 · "162 of them" — REAL, dropped. See "Two figures dropped" above

Also corrected: "compared for equality against the tree" → the actual two-sided comparison,
against non-empty tracked files and against `grep -ac ''`. The `grep`-not-`wc` reason is now
published with its evidence (two files with no final newline, a difference of exactly 2).

### F6 · fixture counts — REAL, all three re-derived

```
$ cd profiler/testdata/otlp
$ grep -l 'slash_command' *.json *.ndjson | wc -l   -> 6
$ grep -l 'skill_tool'    *.json *.ndjson | wc -l   -> 3
$ grep -lE 'slash_command|skill_tool' … | wc -l     -> 6      # union
$ grep -l '"user"'        *.json *.ndjson | wc -l   -> 2      # skill.source
$ grep -o 'skill.kind[^}]*}[^}]*}' skill_activated.json full_export.ndjson
  -> "skill.kind" … "stringValue": "skill"   in two fixtures  # already correct
```

Shipped as six / three-of-those-six / two / two. **Also verified the vendor side**, since
that is the same class — Anthropic's monitoring reference, fetched live, line 1021:
*"`skill.source`: Where the skill was loaded from (**for example**, `"bundled"`,
`"userSettings"`, `"projectSettings"`, `"plugin"`)"*. The "for example" is real, so both
documents now say that calling it a closed set is slightly stronger than the source
supports. Lines 1020 and 1022 confirm the `invocation_trigger` and `skill.kind` wordings
exactly as published.

### F7 · the surviving self-contradiction — REAL. **Treatment and reasoning**

Driven:

```
$ sed -n '808p' README.md        # "verified — … codex-cli 0.153.4 … CODEX_HOME …"
$ git show v0.4.3:README.md | sed -n '436p'
IDENTICAL                                     # byte-for-byte
$ git show v0.4.3:CHANGELOG.md | grep -n 'marks the Devin, Codex and Cursor install rows'
182: … marks the Devin, Codex and Cursor install rows "not verified in this release".
$ git show v0.4.1:README.md | grep -c "codex-cli 0.153.4"
0                                             # so "recorded since 0.4.3" holds
$ grep -n scuba .gitignore   ->  9:.scuba/
$ git ls-files | grep -c '^\.scuba'   ->  0
```

So the contradiction lived **inside the v0.4.3 tree** — that release's own changelog and
notes entries were false against the README shipped beside them — and the freeze preserves
them at `CHANGELOG.md:782` (now `:961`) and `RELEASE_NOTES.md:93`. The ledger row is **not**
a published document, so citing it was accurate reporting and never the contradiction; the
two files I own were.

**Treatment, and why.** Three options were available and two are wrong:

- *Edit the frozen sections* — refused. A released section records what was said. Editing it
  destroys the only evidence that the error was ever made, and breaks the byte-identity
  invariant the release asserts and tests.
- *Leave it and cite the ledger* — refused. That was the previous round's move, and it is
  what left the contradiction standing: the ledger is untracked, so a reader of the shipped
  documents never sees the correction.
- *Use the pattern the release already has* — taken. **"Known past changes, recorded
  late"**, the section this release already uses for the skillscore drop, now carries a
  second entry in each document naming **0.4.3's own two entries** (not only a ledger row),
  stating that the README was and is right, and saying explicitly that neither released
  section is edited and why. It sits *above* the 0.4.3 section in both files, so a reader
  reaches the correction before the error.

And the compounding overclaim is fixed: "this release stops republishing it" was true of the
section and false of the file. Both documents now say which is which — "that narrowing is
true of this section and not of this file" — and point at the recorded-late entry. Frozen
tails re-verified byte-identical afterwards (240 / 112 lines, md5 match).

### F8 · the spec's stale "remain unverified" list — REAL, fixed

Two of four bullets are plainly documented, confirmed on the live page: the
`hooks`-keyed-by-event-name / `command`-string shape is the form of **every** config example
in the reference (`{ "version": 1, "hooks": { "afterFileEdit": [{ "command": "…" }] }}`),
and `hook_event_name`, `cursor_version` and `conversation_id` are all three in the
"Input (all hooks)" block. They are removed from the list and stated as documented, with a
sentence saying `README.md:538-541`'s parallel list was already right and the spec was the
one left behind. The list keeps the two that are genuinely still documentary — the
stdin-document assumption and the tool-call payload shapes — so the two documents now agree.

### F9 · the elided quotation — REAL, quoted whole

```
$ git show v0.4.3:profiler/types.go | sed -n '211,214p'
// slice, like ToolCallResult's, and has no Error constructor: skill activation
// is a property of the harness and of this adapter, not of any export, so no
// export can fail it.
```

The changelog quoted *"has no Error constructor: no export can fail it"* — two
non-adjacent fragments inside quote marks. Now quoted in full with its ref
(`v0.4.3:profiler/types.go:212`) and a note that an earlier version shortened it.

### F10 · "about seventy … about sixty correct" — REAL, re-derived

```
$ git grep -o "0\.5\.0" -- . | wc -l          -> 99
$ git grep -n "0\.5\.0" -- . | wc -l          -> 88   (lines)
$ bash tests/lib/forward-promise-check.sh 0.5.0 ; echo $?   -> 0
```

99 occurrences, zero findings, therefore **all 99 correct** — not "about sixty". The
changelog now says so and names its own past error. **The same wording survives in two code
comments** and is reported below, not edited.

### F11 · the 8 MB bytes — REAL (durability). Dropped, structure kept

See "Two figures dropped". The structural claim was re-driven rather than inherited:
40 writers at 100 B → `writers=40 lines=40 unparseable=0`; 40 at 200 KB → same; 16 at 8 MB →
`lines=16 distinct_lengths=1 unparseable=0`. No generator was invented.

### F12 · "backed up before any write" on a virgin install — REAL, fixed

```
$ /tmp/prof2b hooks install --home /tmp/v2b        # no pre-existing hooks.json
{ "hooks_json": "/tmp/v2b/.cursor/hooks.json", "events_registered": 21,
  "events_removed": 0, "schema_version_added": true }
$ ls -a /tmp/v2b/.cursor/
.  ..  hooks.json                                  # no .bak-*
$ /tmp/prof2b hooks install --home /tmp/v2b        # re-run
{ …, "events_registered": 0, "events_removed": 0 }  # still no .bak-*
```

`README.md` now reads "backed up before it is **overwritten** — on install and on uninstall
alike", with the virgin case stated explicitly, and both release documents carry the same
fact (the changelog's backup-collision bullet and the release notes' uninstall bullet), since
the notes are the user-facing file.

### F13 · an absent `cwd` yields no key — REAL, fixed, driven both ways

```
$ echo '{"hook_event_name":"sessionStart","conversation_id":"c1"}' | prof ingest …
{"ts":"…","event":"sessionStart","schema_version":"…","conversation_id":"c1","raw":{…}}
                                                       # no cwd member at all
$ echo '{"hook_event_name":"preToolUse","cwd":"/project"}' | prof ingest …
{…,"cwd":"/project","raw":{"cwd":"/project",…}}
$ echo '{"hook_event_name":"preToolUse","cwd":""}' | prof ingest …
{…,"raw":{"cwd":"",…}}                                 # also no promoted cwd
```

`profiler/hooks.go:63` is `Cwd string \`json:"cwd,omitempty"\``. The changelog's "Promoting
an absent field as `""` is the documented behaviour" is replaced by what happens, and the
loose "is blank otherwise" in `README.md` and the spec is corrected the same way — with the
point that this is *why* a reader can tell "sent nothing" from "sent an empty string".

---

## The new disclosure this release owed — R4, sharpened

Driven locally, which is the whole point of the disclosure:

```
$ mkdir -p fstest/root && cd fstest
$ [ ROOT -ef root ] && echo SAME        -> SAME OBJECT (case-insensitive)
$ mkdir -p "café"; nfd=$(printf 'cafe\xcc\x81')
$ [ "$nfd" -ef "café" ] && echo SAME    -> SAME OBJECT (normalization-insensitive)
$ mount | grep apfs                     -> /dev/disk3s1s1 on / (apfs, …)
$ grep -n runs-on .github/workflows/ci.yml  -> 11:    runs-on: ubuntu-latest
```

Two of the three containment escapes the barrier lane closed are case-folding and Unicode
normalization. Those are properties of the **volume**. `ubuntu-latest` is case-sensitive and
normalization-sensitive, so there `ROOT` and `root` genuinely *are* different objects, and
the generated case set — which takes each expected verdict from where a real write lands
rather than from a table (`homesafe_test.go:107-127`) — correctly asserts the opposite
answer and stays green. **That is the generator working, and it means the macOS half of the
invariant is never exercised by CI at all.** It runs on a developer machine and reddens
there and nowhere else.

So R4 is no longer procedural hygiene: for these cases the local macOS run is the *only*
place the assertion exists. Both documents now say that plainly, name a `macos-latest` job
as **0.6.0 surface**, and say why it was not added at a release gate. **No job was added.**
(Checked that naming 0.6.0 is safe: the release-set guard derives releases from
`^## v?N.N.N` headings only — `documented_release_versions()` in `tests/test_skill.sh` — so
a prose mention creates no phantom release, and the forward-promise reader only scans for
the version being cut.)

---

## Judged INVALID or refused, with evidence

1. **F1's prescribed wording — REFUSED.** "Removed by this release's own rewrite of
   `claude_code.go`", and "the one of the five whose disappearance this release caused", are
   both false: `1e4e845` is not an ancestor of `c3a565c`, the lineages fork at `30f374c`,
   and `git log c3a565c -S"getBool"` finds only a documentation commit. Nothing on this
   lineage ever declared or removed it. Full evidence under F1. Shipped what git shows
   instead.
2. **`WebFetch`'s reading of the Cursor page — REJECTED.** Its summarising model said
   `postToolUseFailure`'s payload does **not** carry `cwd`. The raw page shows it does. The
   finding was right and the tool was wrong; this is the second time in this release that a
   *read* rather than a *run* produced the false claim, which is exactly the root the
   confirming pass named.
3. **My own first draft of an NFD claim — CAUGHT AND CORRECTED.** I wrote into
   `RELEASE_NOTES.md` that the shipped drafter refuses "an NFD spelling of an NFC directory
   name". Driving it showed my test case passed only because the directory was *genuinely*
   inside `.claude`: all six of the drafter's protected roots are ASCII, so the
   normalization axis is unreachable through the drafter and is covered by the generated
   case set over a non-ASCII root. The sentence now says that. Recorded because it is the
   same failure mode as the 13 findings, caught by driving rather than by review.
4. **The audit's mid-release injection figures (`sites=218 files=7 lines=2594`) — NOT
   republished.** They were measured against a seven-file reader that no longer exists (the
   audit now reads a nine-file closure), so re-deriving them would have meant re-injecting a
   limit into a script this lane may not touch. The sentence was rewritten to describe the
   mechanism and its two sides instead, with no injection numerals.
5. **One inherited claim trimmed for lack of evidence.** A draft of mine repeated the guards
   record's detail that the removed floor enumeration "stated a real value of 6 where the
   measured value was 5". That is a claim about the contents of a deleted sentence, which I
   could not re-derive at head, so I cut it and kept only what I drove — that the
   enumeration went stale inside the same release that wrote it, and that the policy-failure
   count is now an equality against the `PL`-prefixed failures the same payload reports
   (`tests/test_f01.sh:833-857`), with a `-gt 0` beside it.
6. **"64 OTLP fixtures" and "8 DuckDB queries" — verified TRUE, left alone**, as instructed.
   Also re-confirmed that `duckdb` is exercised by nothing in `tests/` or `.github/`, so
   that disclosure still holds.
7. **The 0.4.3 baselines (2499 / 93 / 452) — verified TRUE, left alone.** Re-derived from
   the tag rather than trusted; both matched exactly.
8. **PR body `94 (A) → 92.5 (A-)` — verified correct, no edit needed.** The previous round's
   report was accurate. Re-derived the underlying scores on three worktrees: v0.4.1 **94.0
   (A)**, v0.4.3 **92.5 (A-)**, head **92.5 (A-)**.

---

## Reported, not changed — code surfaces outside this lane

1. **`tests/lib/forward-promise-check.sh:207` — the deferral-verb match is
   case-sensitive.** A capitalised verb at a sentence start, a bullet start, or inside a
   bolded lead-in passes silently, and bolded lead-ins are this repository's commonest prose
   shape. All 18 fixture cases at `tests/test_skill.sh:772-789` are lowercase, so no control
   reaches it. Disclosed in both release documents as a stated boundary; the fix is a code
   change (lowercase the record before matching, plus capitalised fixture cases).
2. **`profiler/doctor.go:319-321` — a code comment still carries the pre-`--spool-dir`
   registration string** ("this binary's absolute path plus `ingest || true`"). The code is
   right; only the comment is stale.
3. **The "about seventy times" wording survives in two code comments** —
   `tests/lib/forward-promise-check.sh:51` ("about seventy … roughly sixty") and
   `tests/test_skill.sh:723` ("69 times"). The real figure is 99, all correct. Named in the
   changelog as noted-for-the-maintainer.
4. **`tests/test_harness.sh:131` and `:339` cite `740` and `sites=767 files=7` without
   naming their commits** (carried over from the guards lane's residuals). Both were true of
   different commits; neither is false, and both will go stale.

---

## Verification, at `bf045ce`

```
$ for B in /bin/bash /opt/homebrew/bin/bash; do for s in tests/test_*.sh; do
    printf '%-24s %s\n' "$s" "$($B "$s" 2>&1 | tail -1)"; done; done
=== /bin/bash (3.2.57(1)-release)          === /opt/homebrew/bin/bash (5.3.15(1)-release)
tests/test_f01.sh     1912 passed, 0 failed   tests/test_f01.sh     1912 passed, 0 failed
tests/test_f02.sh      350 passed, 0 failed   tests/test_f02.sh      350 passed, 0 failed
tests/test_harness.sh  276 passed, 0 failed   tests/test_harness.sh  276 passed, 0 failed
tests/test_install.sh   52 passed, 0 failed   tests/test_install.sh   52 passed, 0 failed
tests/test_rewrite.sh  401 passed, 0 failed   tests/test_rewrite.sh  401 passed, 0 failed
tests/test_skill.sh    101 passed, 0 failed   tests/test_skill.sh    101 passed, 0 failed
tests/test_walk.sh      20 passed, 0 failed   tests/test_walk.sh      20 passed, 0 failed
                    3112, 0 failed                                3112, 0 failed
```

`test_skill.sh` is the suite that runs the forward-promise reader and the release-document
guard over the prose in this commit, so its green is load-bearing here rather than
incidental — and it went red once on my own draft and back to green, above.

```
$ cd profiler && go build ./... && go vet ./... && gofmt -l . && go test ./...
BUILD_OK / VET_OK / (gofmt: no output) / ok ×3
$ skill-validator check skills/skill-audit   -o json  -> errors 0 warnings 0 passed True
$ skill-validator check skills/skill-rewrite -o json  -> errors 0 warnings 0 passed True
$ git status --porcelain
 M CHANGELOG.md
 M README.md
 M RELEASE_NOTES.md
 M docs/profiler-spec.md
```

Frozen sections, after all edits: CHANGELOG `## 0.4.3` at line 786, tail **240** lines,
md5 `ef2fe48af306b4ce06ace48391eddcb8`; RELEASE_NOTES `## v0.4.3` at line 36, tail **112**
lines, md5 `46d66b1e331726dbe56a830d449c19aa`. **Both identical to `v0.4.3`.**

Author hygiene: `git log 10326b3..HEAD --format='%an|%cn' | sort -u` → `imagineux|imagineux`
only, across all 27 commits; `%(trailers)` empty on every one; an attribution grep over
every message returns 0. The only `claude` string in my commit message is the filename
`profiler/claude_code.go`.

---

## PR #22

Body rewritten and pushed (`gh pr edit 22 --body-file`, 224 lines round-tripped): the figure
table re-derived with a command per row plus two rows saying which figures were *dropped*;
the containment paragraph now names identity rather than resolution order; the `getBool`,
`cwd`, fixture-count and concurrency claims corrected; the R4 bullet rewritten as
load-bearing with the macOS/CI asymmetry; a new bullet recording the F7 treatment; the
verification paragraph updated to 3112 / 27 commits / md5-verified frozen tails; and a
closing "still the maintainer's to decide before tagging" list. No attribution or
"Generated with" line anywhere in it.

**Review threads: none to reply to or resolve.** Paginated via GraphQL —
`reviewThreads.totalCount` is **0**, with 0 reviews and 0 issue comments. The 13 findings
reached this lane through the steward's reconciled worklist and the two fix records, not
through PR threads, so there is no thread for a fixing reply to attach to and none for the
steward to resolve.

`mergeable` read per-PR (not from a list endpoint) at head `bf045ce`: **MERGEABLE**.
`mergeStateStatus` went CLEAN → BLOCKED on the push purely because the single `test` job
re-queued on the new head; its result is recorded by the steward/maintainer at the tip.

---

## Still owed before the maintainer tags

1. **The `v0.5.0` tag itself.** Not created, not pushed. All four existing tags were pushed
   by hand and this one is the maintainer's.
2. **The skill `metadata.version` decision** — `skill-audit` 0.2.0 and `skill-rewrite` 0.1.0
   against the plugin's 0.5.0. Disclosed in both documents, untouched, nothing asserting a
   relationship, no assert pinning them. Left open exactly as instructed. It is more
   conspicuous this release because `skill-rewrite`'s documented interface changed.
3. **An immutable tag for the frozen 0.5.0 WIP tree.** The three adapter drafts — and
   `getBool`, per F1 — are reachable only from `0a83615` on `origin/wip/0.5.0-profiler`, a
   branch a routine push moves. Both documents cite that ref so a reader can open the files;
   if the branch moves, those citations die. One command.
4. **Four code residuals**, listed under "Reported, not changed" — the guard's
   case-sensitive matcher (the only one with a live blind spot behind it), the stale
   `doctor.go` comment, the "about seventy" wording in two comments, and two uncommitted
   figures in `test_harness.sh` comments.
