# S10 record — the v0.5.0 release commit

**Implementer** S10 · **Date** 2026-09-21 · **Branch** `release/0.5.0`
**Base** `origin/main` = **`10326b3`** ("0.5.0 S9: the drafter gains an output flag, and a bound to go with it")
**Head** `9c8ba53` · **PR** [#22](https://github.com/Okja-Engineering/skill-architect/pull/22), open, **not draft**, base `main`
**Worktree** `<scratch>/wt-s10-release`. The primary tree was never touched: still at
`0a83615` with 47 uncommitted paths intact. No `checkout`, `reset`, `clean`, `stash` or
`rebase` was run in any tree but my own.

**Nothing is tagged.** See §7.

Two commits:

| commit | what |
|---|---|
| `c232907` | the five version surfaces and the three asserts that hold them |
| `9c8ba53` | CHANGELOG, RELEASE_NOTES, and the claims this release falsified |

---

## 1. Test-first, and the RED that earned the bump

The only behaviour change in this slice is five version strings, and the asserts that hold
them are the test. So they were moved first.

`tests/test_skill.sh:44`, `:58` and `:96` were bumped `0.4.3` → `0.5.0` **before** any
manifest, and the suite was run on both shells:

```
/bin/bash tests/test_skill.sh                 exit=1  48 PASS  5 FAIL
/opt/homebrew/bin/bash tests/test_skill.sh    exit=1  48 PASS  5 FAIL
```

Five `FAIL:` lines, one per version surface, each carrying
`AssertionError: version mismatch` from the Python manifest reader:

```
FAIL: .devin-plugin/plugin.json is valid plugin.json
FAIL: .claude-plugin/plugin.json is valid plugin.json
FAIL: .cursor-plugin/plugin.json is valid plugin.json
FAIL: .codex-plugin/plugin.json is valid plugin.json
FAIL: .claude-plugin/marketplace.json is valid and at 0.5.0
```

**RED for exactly the right reason**, and on both shells — which matters here more than
usual, because a suite that could not report this failure on bash 3.2 is the defect 0.4.3
exists to have closed. The five manifests were then bumped and the suite returned
**53 passed / 0 failed, exit 0, on both shells**.

The fifth surface is the one the mandate warned about: `.claude-plugin/marketplace.json:13`
carries its own copy of the version and `claude plugin marketplace add` reads it, so
bumping the four `plugin.json` files and forgetting it would advertise 0.4.3 to anyone
installing by name. `jq -e .` over all five confirms each is still valid JSON.

---

## 2. Every number, with the command that produced it

**Nothing is quoted.** Five figures went stale during this release, so every count below
was produced on this branch, and the 0.4.3 baselines the notes compare against were
**re-measured on a `git archive v0.4.3` tree** rather than read out of the 0.4.3 entry.

### 2.1 Shell suites — 2649, identical on both shells

```
for s in tests/test_{harness,skill,install,walk,rewrite,f01,f02}.sh; do
  "$BASH_BIN" "$s" > log; grep -c '^PASS:' log; grep -c '^FAIL:' log; done
```
Runner: `<scratch>/s10-run-suites.sh`, run four times — baseline and head, under each shell.

| suite | bash 3.2.57 | bash 5.3.15 |
|---|---|---|
| test_f01 | 1868 | 1868 |
| test_f02 | 350 | 350 |
| test_harness | 174 | 174 |
| test_rewrite | 136 | 136 |
| test_skill | 53 | 53 |
| test_install | 48 | 48 |
| test_walk | 20 | 20 |
| **Total** | **2649 passed, 0 failed** | **2649 passed, 0 failed** |

Measured at `10326b3` before I changed anything and again at `9c8ba53`: **identical**, which
is the proof this slice regressed nothing. **S9's 2649 reproduces exactly** — the first
figure in five slices that did not go stale.

Bash versions: `/bin/bash` is `3.2.57(1)-release (arm64-apple-darwin25)`;
`/opt/homebrew/bin/bash` is `5.3.15(1)-release (aarch64-apple-darwin25.4.0)`.

### 2.2 Go tests — 275 top-level, 669 subtests

```
cd profiler && go test -list '.*' ./... | grep -cE '^(Test|Example|Fuzz|Benchmark)'
```
and the same per package:

| package | top-level test functions |
|---|---|
| `profiler` | 233 |
| `profiler/cmd` | 34 |
| `profiler/internal/homesafe` | 8 |
| **Total** | **275** |

**S9's 275 and its 233/34/8 breakdown reproduce exactly.** Both figures S9 told me to treat
as unverified are correct at `10326b3`.

```
go test -race -count=1 -v ./...    # rc=0
grep -cE '^--- PASS'          →  274   top-level passes
grep -cE '\-\-\- SKIP'        →    1   TestHomeBarrierChild (274 + 1 = 275)
grep -cE '^ +--- PASS'        →  669   subtests
… of those, names with 2+ '/' →   33   nested one deeper (so 636 direct)
```

A note on that split: **Go 1.27.1 no longer indents a nested subtest's `--- PASS` line**
— every subtest line has four spaces regardless of depth — so the 0.4.3 entry's "422 direct
and 30 nested" cannot be reproduced by indentation today. Counting slashes in the subtest
name gives exactly 30 at v0.4.3, which is what that entry meant, so the two are consistent
and my 636/33 is measured the same way.

### 2.3 Coverage — and L21's figure was an artifact

```
cd profiler && go test -cover ./...
profiler                     97.9%
profiler/cmd                 94.5%
profiler/internal/homesafe   93.5%
```

The ledger records `profiler/cmd` at 12.6%. That is the artifact S6 diagnosed: every CLI
test runs the binary as a subprocess and `go test` counts only the test process, so
`-cover` on the built binary is what makes the number mean anything. `git log -S'GOCOVERDIR'`
confirms the fix landed in this release (`d77d90e`, S6).

### 2.4 Counts of things

```
ls -1 profiler/testdata/otlp/ | grep -cE '\.(json|ndjson)$'    →  64    (v0.4.3: 59)
ls -1 profiler/queries/*.sql | wc -l                           →   8
grep -n 'case "' profiler/cmd/main.go                          →  9 top-level subcommands
git show v0.4.3:profiler/cmd/main.go | grep -n 'case "'        →  3 (probe, capture, version)
```

So **six subcommands are new**: `compare`, `experiment`, `ingest`, `hooks`, `analyze`,
`doctor`. S8 §2's table says "7 SQL + README" and §8.2 says "eight DuckDB queries"; the
tree has **eight** `.sql` files and a README, so §8.2 is the correct one.

### 2.5 The 0.4.3 baselines — measured, not quoted

```
git archive v0.4.3 | tar x   (into <scratch>/v043)
cd v043/profiler && go test -list '.*' ./... | grep -cE …       →  93
cd v043/profiler && go test -race -count=1 -v ./...             →  93 top-level, 452 subtests
cd v043 && seven suites under /bin/bash                          →  2499 passed, 0 failed
git ls-tree --name-only v0.4.3 profiler/testdata/otlp/ | grep -cE … → 59
```

All four match what the 0.4.3 entry published. They are now my measurements rather than its
claims.

### 2.6 The deferred ledger — 15 rows, 13 still open

```
grep -E '^\| L[0-9]+ \|' deferred-ledger.md | awk -F'|' '$8 ~ /0\.5\.0/' | wc -l   →  15
… same, adding '&& $5 ~ /OPEN/'                                                     →  15
grep -cE '^\| L[0-9]+ \|' deferred-ledger.md                                        →  33
```

15 rows name 0.5.0 as their home and **all 15 are marked OPEN** — but the ledger was audited
at `5c847e1`, *before 0.4.3 shipped*, so "open" is a claim about that commit. I re-walked all
15 against this branch rather than repeating the label:

| row | re-walked at `9c8ba53` | how |
|---|---|---|
| L5 per-finding `jq` in both `--json` writers | **open** | `check-paths.sh:176` calls `jq_answer` inside the finding loop |
| L6 `isMonotonic` unread | **open** | no non-test file in `profiler/` names it |
| L7 `flags` unread | **open** | no `flags` member on any OTLP struct in `otlp.go` |
| L8 `--json` exit 3 without payload | **open, deliberate** | unchanged |
| L9 no channel for a refused series | **open** | schema is still `profile/v1`; `Reason` is `present when State != "present"` |
| L13 whole-export slurp | **open** | `os.ReadFile` still on the read path |
| **L14 composite attribute identity** | **CLOSED, by 0.4.3** | `otlp.go:698–702` canonicalises a `kvlistValue` as a map and keeps an `arrayValue`'s order. This is 0.4.3's "map attribute's identity no longer depends on order". Not this release's. |
| L15 truncated final line costs the capture | **open** | `otlp.go:353` still `malformedJSON(…, "unexpected end of JSON input")` |
| L16 rejected vs failed indistinguishable | **open** | `ErrorType` is declared in `types.go:189` and **set by nothing** outside tests |
| L19 not-a-count bound a hair generous | **open** | unchanged |
| **L21 `profiler/cmd` coverage 12.6%** | **CLOSED, by this release** | §2.3 — 94.5%, and the 12.6% was the artifact |
| L22 Devin/Codex/Cursor install rows unverified | **open, disclosed** | not re-run here |
| L23 cumulative temporality never seen live | **open, honest** | unchanged |
| L24 tool attrs `[DOCS]` not `[OBSERVED]` | **open, honest** | `testdata/otlp/README.md:26–29` still lists them under `[DOCS]` |
| L25 `CaptureOpts.APIKey` read by nothing | **open, documented** | only declaration sites in `types.go` |

**13 open, and this release closes none of the 13.** That is the sentence in the notes.
The plan's §S10 said "~15 open (L5–L25)", which is the ledger's own count; the 13 is the
re-walk, and it is stated as such in both documents rather than as a correction of the plan.

### 2.7 Dogfooding — and a 0.4.3 regression nobody recorded

AGENTS.md makes dogfooding a release gate, so it was run and the numbers recorded.

```
skillscore skills/<skill> --json | jq -c '.overallScore | {percentage, letterGrade}'
skillscore 2.0.2 · skill-validator v1.6.1
```

| skill | at head |
|---|---|
| `skill-audit` | **92.5 (A-)** |
| `skill-rewrite` | **89.5 (B+)** |

**`skill-audit` was 94 (A) at v0.4.1 and is 92.5 (A-) now, and the 0.4.3 entry does not
record the drop.** Measured across three trees under *one* pinned skillscore, so it is
content and not a tool version:

```
git archive v0.4.1 skills | tar x → skillscore → 94   (A)
git archive v0.4.3 skills | tar x → skillscore → 92.5 (A-)
head                              → skillscore → 92.5 (A-)
```

Diagnosed to the category and the finding:

| category | v0.4.1 | head | the finding that changed |
|---|---|---|---|
| `clarity` | 9/10 | 8/10 | `pass 2 Consistent terminology throughout` became `warning 1 1 synonym pair(s) used interchangeably` |

0.5.0 does not touch `skills/skill-audit/` (`git diff --name-status v0.4.3..HEAD` lists only
`skills/skill-rewrite/*` under `skills/`), so this release neither caused nor changes it. It
is recorded in both documents as a past change, in the house pattern: the 0.4.1 and 0.4.3
entries are left as written. **I did not chase the synonym pair** — editing a shipped
`SKILL.md` to move a score is behaviour, out of this slice, and it wants its own decision.

---

## 3. Every owed item, and where it is stated

All ten are in **both** documents. `CHANGELOG.md` line numbers first, `RELEASE_NOTES.md`
second.

| owed | CHANGELOG.md | RELEASE_NOTES.md |
|---|---|---|
| The removed `doctor` tiers — `hooks`, `server_api`, `enterprise` — **with the reason**, as its own item | `:126` (and `:14`, the overview paragraph) | `:14` |
| **`experiment run` has no time limit — no cap of any kind**; `mutation_bounded` and `exec.CommandContext` named and explicitly not softening it | `:232` | `:9` |
| The hook spool's verification is documentary only; the payload shapes are Cursor's docs | `:243` | `:11` |
| **`AppendSpool` has no locking** | `:251` | `:12` |
| No Cursor, Codex or Devin adapter ships; README still says "Planned"; the Cursor spool does ship and `analyze` summarises it | `:17` | `:15` |
| `draft-rewrite.sh`'s `-o|--output`, all four refusals named, bound decided after resolution | `:88` | `:18` |
| **What this release does not close** — the ledger's open 0.5.0 rows, counted and enumerated | `:294` | `:21` |
| **R4 as an explicitly accepted risk** — CI is `ubuntu-latest` only, so every "green on CI" claim here is a bash-5 claim; 0.4.3's `[[ ]]`/`errexit` skew named as the reason it matters; no `macos-latest` job added | `:282` | `:20` |
| The skill `metadata.version` fields — independent of the plugin version, where they stand, the maintainer's call, not pre-empted and not hidden | `:327` | `:25` |
| The skill-activation read is `[DOCS]` too (found while verifying, not on the list) | `:259` | `:23` |

Three more that the records asked for and I judged owed:

| | CHANGELOG.md | RELEASE_NOTES.md |
|---|---|---|
| An experiment result embeds both commands, so a literal secret travels with it (S6 §9.6) | `:275` | `:24` |
| `error_type`, `count` and `id` are declared and written by nothing that ships; `attribution` is `unknown` on every profile this release can produce | `:265`, `:271` | `:22` |
| No spool-to-profile projection ships, and why that was a choice (S7 §3.1) | `:318` | `:27` |

---

## 4. The prose gate — what I drove against the code before writing it

Every sentence in the two documents that asserts a fact was checked. The ones that were
**not** simply readable off a file:

- **"The three adapter drafts do not compile against head."** Not quoted from the plan —
  measured. `git archive HEAD profiler | tar x` into a scratch tree, then
  `git show 1e4e845:profiler/{cursor,codex,devin}.go` copied in, then `go build ./...`:
  `undefined: otelMetric`, `undefined: otelLog`, `undefined: toInt`, `undefined: parseTime`.
  The four names in the notes are the four the compiler printed.
- **"Two of them advertise what they cannot deliver."** Read out of the drafts themselves:
  `cursor.go:60` sets `caps[MetricTokens] = SourceSQLite` and `cursor.go:150` answers that
  same capability `UnknownTokenResult("Cursor SQLite token schema not yet verified")`;
  `devin.go:37` sets `SourceSessionData` and `devin.go:114` returns `UnknownTokenResult`.
- **"None ever has."** `git ls-tree --name-only v0.4.3 profiler/` has no cursor, codex or
  devin file, and `git log --all -- profiler/{cursor,codex,devin}.go` returns only
  `1e4e845` (the wip branch) and `a3673b8` (the archive). The drafts were **never on `main`
  and never in a release**, so the notes say "none ever has" rather than "deleted" — a
  reader must not infer that a shipped adapter was removed.
- **"21 event names."** `sed -n '37,49p' profiler/hooks_install.go | grep -oE '"[a-zA-Z]+"' | wc -l` → 21.
- **Redaction defaults.** `hooks.go:204–205` — `prompt`, `text`, `content`, `tool_input`,
  `tool_output`, `output`, `command`, `edits` → `[REDACTED]`; strict mode replaces the value
  with `{"_stripped_bytes": N}`.
- **The design refusals.** `experiment.go:48–50` (`blocked`/`difference`/`fixed`), `:59`
  (`$PROFILE`), and `designRefusal` at `:294` as a rule table rather than an `if` chain.
- **`compare` treats a snapshot difference as a note.** `compare.go:247` — "A different
  snapshot hash is the *point* of a paired run".
- **The seven mutation verdicts.** `grep -oE 'KILLED|SURVIVED|…' tests/lib/mutation-runner.sh`
  returns all seven plus `mutation_bounded`.
- **`analyze` reports `stripped_envelopes` and `payload_keys`** — `analyze.go:61`, `:110`.
- **The contract table's shape.** `adapter_contract_test.go` carries
  `TestTheContractTableCanFail`, `TestNoAdapterEscapesIntoTheCommand`,
  `TestEveryAdapterIsInTheRegistry` and the AC9 both-directions checks at `:221–247`.
- **Zero errors and zero warnings** is `tests/test_skill.sh:153`, over a derived denominator.
- **CI is one job on `ubuntu-latest`** — read off `.github/workflows/ci.yml`, which has a
  single `test` job and no `strategy.matrix`.

### 4.1 Claims I removed rather than assert

- I did **not** write that the Claude Code adapter delivers "four `present` signals" as a
  general fact. It does over `testdata/otlp/full_export.ndjson` (S8 measured it), which is a
  fixture; the notes say the adapter *reads* four signals and leave the count of `present`
  ones to the export.
- I did **not** re-run any install or update route, so the README's rows now say they were
  executed in **0.4.3** and that 0.5.0 did not re-run them. The
  `claude plugin list … at version 0.5.0` line is derived from
  `.claude-plugin/marketplace.json`, which I changed, and not from a live install.
- I did **not** claim a `skillscore`/`skill-validator` delta for `skill-rewrite` beyond the
  number: it is 89.5 (B+) at v0.4.1, v0.4.3 and head, unchanged even though S9 rewrote parts
  of its `SKILL.md`. Stated as the number, not as a conclusion about the tool.

---

## 5. Files beyond the mandate's enumerated list, and why

The mandate names five things that land. I touched **one file beyond them**, and one of the
five in a way worth stating.

### 5.1 `docs/profiler-spec.md` — **beyond the list. [flag]**

Four hunks, docs only, all the same defect: a deferral marker that reads "0.5.0" for work
this release does not do. `:788` and `:792` (the schema-v1 caveat channel), `:806`
(`duration_ms` and `active_time`), `:837` (`probe`'s exit contract). Each now names what it
is waiting for instead of naming this release.

**Why I did it rather than surface it.** Shipping 0.5.0 with a document that says a thing is
"tracked for 0.5.0" is a false claim in the release surface, which is the exact defect class
the mandate opens by naming, and the release notes assert the opposite on the same day. The
mandate's own Verification section makes the prose gate a release gate. Fixing README's four
and leaving the spec's four would also have been incoherent — they are one class in two
files. **It is docs-only, surgical, and revertible in one hunk per marker**; if the manager
reads the enumerated list as exhaustive, revert the `docs/profiler-spec.md` hunks and nothing
else moves.

### 5.2 `README.md` — in the list, but wider than "capability tables"

Eight hunks. Two are the capability surface the mandate names (the signal summary at `:28`
and the Claude Code harness row at `:34`, both of which omitted `skill_activation` and were
flagged by S5 §7.4 and S6 §9.5). The other six are claims the release itself falsified:

| README | was | why it had to change |
|---|---|---|
| `:98` | "`probe`… **deferred to 0.5.0**" | 0.5.0 ships without it |
| `:186` | "…is tracked for 0.5.0" | schema is still v1 |
| `:323` | "A bundled receiver subcommand is 0.5.0" | **flatly false** once 0.5.0 ships |
| `:477` | "this release reads nothing back out of it" | falsified **inside this release** by `analyze`, which is documented 40 lines further down the same file. S7 wrote it before S8 landed `analyze` |
| `:713` | `claude plugin list` … "at version 0.4.3" | the manifests moved |
| `:715`, `:839` | install rows "executed in **this release**" | 0.5.0 did not run them; 0.4.3 did |

The three "Planned" rows for Cursor, Codex and Devin are **untouched**, and the Cursor row's
distinction — the spool ships, the adapter does not — is preserved verbatim.

### 5.3 Everything the mandate forbade, and stayed forbidden

No Go file changed. No shell script under `skills/` or `tests/` changed except
`tests/test_skill.sh`'s three version asserts. `git diff --name-status 10326b3..HEAD` is ten
files and `tests/lib/out-of-scope-check.sh` returned **`out-of-scope check: clean`, exit 0**
before each of the two commits.

---

## 6. Verification, and the plan's definition of done re-checked on this branch

| check | result |
|---|---|
| All seven suites, bash 3.2.57 | **2649 passed, 0 failed**, every suite exit 0 |
| All seven suites, bash 5.3.15 | **2649 passed, 0 failed**, every suite exit 0 |
| `cd profiler && go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l .` | empty |
| `go test ./...` | all three packages ok |
| `go test -race -count=1 ./...` | ok, 275 top-level, 669 subtests, 1 by-design skip |
| `tests/test_skill.sh` with asserts at 0.5.0 | 53 passed / 0 failed, both shells |

The plan §8's five mechanical items, re-run on `9c8ba53` rather than trusted from the CoS's
check at `10326b3`:

| DoD item | command | result |
|---|---|---|
| no `skillgate`/`skill-gate`/`go.work`/`NOTICE` in the tree | `git ls-tree -r HEAD --name-only \| grep -E …` | **nothing** |
| no `cache_write`/`CacheWrite` | `grep -rn … . --exclude-dir=.git` | **nothing** |
| `profiler/{cursor,codex,devin}.go` absent | `ls` | all three absent |
| README still says "Planned" | `grep -c Planned README.md` | 3 rows, all three harnesses |
| `AdapterVersion == "0.5.0"`, `ProfileSchema == "skill-architect/profile/v1"` | `grep -n … profiler/types.go` | `:377`, `:367` — both as required |

The sixth DoD item — "`wip/0.5.0-profiler` is tagged `archive/0.5.0-wip`" — is **satisfied as
a branch, not a tag**: `refs/heads/archive/0.5.0-wip-tree` exists and is pushed to `origin`,
as is `refs/remotes/origin/wip/0.5.0-profiler`. The archive is preserved and reachable; only
the ref *kind* differs from the plan's wording. Tags are the maintainer's (§7), so I did not
convert it.

CI on PR #22: **`test` pass**
([run 35618761354](https://github.com/Okja-Engineering/skill-architect/actions/runs/35618761354)).
Per the notes' own R4 item, that green is a **bash-5 result only** — one job, `ubuntu-latest`,
one bash. The bash 3.2 half is the local run in the table above and nowhere else.
`gh pr view 22`: `mergeable: MERGEABLE`, `mergeStateStatus: BLOCKED` (review required).

---

## 7. The tag — left undone, deliberately

**No tag was created and none was pushed.** `git tag` on this branch returns
`v0.2.0 v0.4.1 v0.4.2 v0.4.3` — the four the maintainer has pushed by hand — and
`git tag --contains HEAD` returns nothing.

`v0.5.0` is the maintainer's to create, after the ship gate, and after the one decision this
release deliberately leaves open (§8.1). Annotating and pushing it is the last act of the
release and it is not an agent's.

**Nothing was merged.** PR #22 is open, not draft, base `main`.

---

## 8. Open, and not mine

### 8.1 The skill `metadata.version` decision. **[user]**

`skills/skill-audit/SKILL.md` declares `0.2.0`; `skills/skill-rewrite/SKILL.md` declares
`0.1.0`; the five manifests are now `0.5.0`. Untouched, no assert added, and stated in both
documents as independent-by-design and as an open decision. It is more conspicuous than it
was — S9 added a flag to `skill-rewrite`'s documented interface and left it at `0.1.0` — and
`tests/test_skill.sh:102–107` already explains why `AdapterVersion` is deliberately not held
equal to the manifests, which is the nearest precedent for how to reason about it. **Decide
before tagging.**

### 8.2 `profiler/types.go:85` carries a claim this release falsifies. **[0.6.0 or the bug-fixer]**

```go
APIKey       string `json:"api_key,omitempty"`     // server API auth; reserved for the Devin and Cursor adapters in 0.5.0
```

There is no Devin or Cursor adapter in 0.5.0 and this release is 0.5.0, so the comment is
false at head. It is a Go file, which this slice may not touch, so it is **reported and not
edited** — the only false "0.5.0" marker left in the tree after this commit, and the reason
it is left is the mandate's own constraint rather than a judgement that it is fine.
Ledger row L25 covers the field itself; this is the comment on it.

### 8.3 `skill-audit`'s 94 → 92.5 is recorded, not repaired. **[decide]**

§2.7. One `clarity` point, on a synonym-pair warning against a `SKILL.md` 0.4.3 rewrote.
Recovering it means editing a shipped skill's prose to move a score, which is behaviour and
wants its own slice and its own argument. The measurement and the diagnosis are above so
whoever takes it does not start from zero.

### 8.4 `README.md`'s install and update rows are 0.4.3's measurement. **[decide]**

I changed the sentence to say so rather than let it claim this release ran them. That leaves
the rows themselves as a one-release-old execution record, which is honest but will keep
ageing — and ledger row L22 (the Devin, Codex and Cursor rows never verified at all) sits
inside the same table. A release that re-runs the routes, or a table that carries the release
each row was last executed in, would end the drift. Not started.

### 8.5 The overview paragraph says "four things were deliberately not shipped". **[note]**

I counted them: a Cursor adapter, the three `doctor` tiers, `experiment run`'s cap, and the
spool-to-profile projection. The mandate's framing said three. The fourth is S7 §3.1's
projection, which is the same class and is separately owed a line, so the number in the
notes is four and it is enumerated in the same sentence rather than left to be counted.

### 8.6 `test_harness.sh` derives the suite count from `ci.yml` and `README.md`. **[note]**

Which is why R4 cannot be closed casually: adding a `macos-latest` job to `ci.yml` touches
the file that suite counts suites from. S9 §6 hit the same coupling from the other side. Not
a reason to leave R4 open — it is open because new CI surface is 0.6.0's — but the next
person should know the cost is two files, not one.

---

## 9. Not done, and why

| not done | why |
|---|---|
| Creating or pushing `v0.5.0` | §7. The maintainer's. All four existing tags were pushed by hand. |
| Merging | The user merges. PR #22 is open and not draft. |
| Adding a `macos-latest` CI job | R4 permits a job **or** a recorded risk; S1 did not add the job, so the risk is recorded (`CHANGELOG.md:282`, `RELEASE_NOTES.md:20`). New CI surface belongs to 0.6.0, and the mandate says so. |
| Bumping any skill `metadata.version`, or adding an assert pinning one | §8.1. The user's call; the notes state it without pre-empting it. |
| Fixing `profiler/types.go:85` | §8.2. A Go file. Reported. |
| Fixing `skill-audit`'s synonym-pair warning | §8.3. Editing a shipped skill is behaviour, not a release commit. |
| Re-running the install or update routes | §4.1. Not measured, so the README now says which release measured them. |
| Converting `archive/0.5.0-wip-tree` from a branch to a tag | §6. It is pushed and reachable; tags are the maintainer's. |
| Editing the 0.4.3, 0.4.2, 0.4.1 or 0.4.0 entries | The mandate forbids it and the house pattern is to correct in the new entry. The 94 → 92.5 drop is recorded that way. |
| Touching any Go file, or any shell script under `skills/` or `tests/` other than `test_skill.sh`'s three asserts | The mandate. §5.3, and `git diff --name-status` proves it. |
| Reading or draining PR #22's review findings | Ship-gate work, and the `bug-fixer`'s. Handed off. |
