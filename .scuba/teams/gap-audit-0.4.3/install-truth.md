# install-truth — gap audit of `origin/main` @ `5c847e1`

**Method.** Clean worktree cut from `origin/main` at `5c847e1`
(`.../scratchpad/wt-install`). The primary tree was never touched. Every install
path and first-run flow below was **executed**, not read. Missing tools were
simulated by PATH masking (a symlink farm of the real PATH minus one binary,
using the repo's own `tests/lib/masked-path.sh`); broken tools by shims on a
prefixed PATH. Nothing on the system was installed, moved or uninstalled.
`claude` runs used an isolated `CLAUDE_CONFIG_DIR`; the user's real
`~/.claude/settings.json` was verified unchanged. `devin` probes were
non-mutating (the plugin store was verified identical before and after).

Local `bash` is 3.2.57; the scripts were additionally re-run with bash 3.2
forced first on PATH and behaved identically, so no finding below is a
bash-version artifact.

## Coverage

- **5/5 documented install paths**: `.claude-plugin` (GitHub marketplace route
  **and** local `./` route, both installed live), `.devin-plugin`,
  `.cursor-plugin`, `.codex-plugin`, manual standalone copy.
- **4/4 harnesses**: Claude Code and Devin exercised with their real CLIs
  (both are installed here); Cursor and Codex verified against the vendors'
  own current docs, since neither CLI is present.
- **46/46 executable lines** in `README.md`, `skills/skill-audit/SKILL.md` and
  `skills/skill-rewrite/SKILL.md` run verbatim (enumerated by grep, walked
  one by one).
- **3 external tools × 3 states (present / absent / present-but-broken) × 6
  scripts** = the full prerequisites matrix, plus `verdict-guard.sh` in its 3
  documented states (missing / malformed / truncated) × 5 scripts.
- **13/13 markdown link targets** (10 in README, 3 in SKILL.md) resolved; 4/4
  external URLs fetched; both install sources (`brew` tap, npm registry)
  confirmed to exist.
- **Profiler**: build + `version`/`probe`/`capture`, 8 usage-error forms, 6
  input states (valid ndjson, missing file, empty file, directory, no flag,
  `--export-file`), and the README's route-(a) OTLP capture recipe run
  end-to-end against a live receiver.
- **5 test suites** (4 shell + `go test ./...`) run from the repo root and
  again from `/`.
- Second full sweep performed; it added IT-09, IT-12, IT-13 and IT-16 and then
  went dry.

**Verdict: NOT CLEAN — 25 findings (1 P1, 9 P2, 15 P3).**

## Shared roots

Most of the list collapses into four roots. Fixing the root closes the class.

- **Root A — `require_tool` proves a tool is *present*, never that it *works*.**
  `verdict-guard.sh:143-152` answers one question (`command -v`), and every
  caller then consumes that tool's output unchecked. README:278 promises the
  opposite ("The same rule covers the case where every tool is present but one
  of them answers with something the caller cannot interpret"), but that
  coverage exists **only** for the `check-structure.sh` → `check-paths.sh`
  child relationship and only for `skill-validator`'s *exit status*. It does
  not exist for `jq` at all, for `skill-validator`'s *payload*, or for
  `skillscore`. → IT-01, IT-02, IT-03, IT-11, IT-12, IT-13.
- **Root B — an empty string from a source is rendered as "no error".**
  `audit-report.sh` uses the source's own output as the error message
  (`:74`, `:97`) and then maps `""` to `null` (`:185`, `:187`), so a source
  that said nothing at all is reported as a source with nothing wrong.
  → IT-09.
- **Root C — the exit-code contract in `SKILL.md:87` is one sentence covering
  scripts that do not share one contract.** `audit-report.sh` documents its own
  different contract at `audit-report.sh:26` (`0=report generated`), and two
  scripts leak a child tool's status outside their enumerated set. → IT-08,
  IT-11, IT-12.
- **Root D — install/update instructions were written, not executed.** README's
  own hedge at :329-331 says the three non-Claude routes were never run; two of
  them are wrong in ways a single execution would have caught, and the
  manual-copy command is not idempotent. → IT-04, IT-05, IT-06, IT-22.

---

# P1

## IT-01 — `check-frontmatter.sh` reports a spec verdict it did not compute when `skill-validator` answers with an unreadable payload

**REAL. P1. 0.4.3 patch.** `skills/skill-audit/scripts/check-frontmatter.sh:44-63`.
Documented claim: `README.md:278-282` — "The same rule covers the case where
every tool is present but one of them answers with something the caller cannot
interpret. … `check-frontmatter.sh` does the same on stderr for
`skill-validator`."

`check-frontmatter.sh` runs `skill-validator validate structure -o json` and
then inspects **only `$?`**. The JSON payload it asked for is never read except
to `sed` it into the failure message. So a `skill-validator` that exits 0 while
printing non-JSON, a bare number, two documents, or nothing at all is accepted
as a clean spec pass.

```text
$ PATH=<shim printing "not json at all", exit 0>:$PATH check-frontmatter.sh <skill>
frontmatter OK
exit=0

$ PATH=<shim printing nothing, exit 0>:$PATH check-frontmatter.sh <skill>
frontmatter OK
exit=0

$ PATH=<shim printing "7", exit 0>:$PATH check-frontmatter.sh <skill>
frontmatter OK
exit=0
```

Only the status arm works, and it emits no rule ID in the text:

```text
$ PATH=<shim exit 42>:$PATH check-frontmatter.sh <skill>
ERROR: skill-validator exited with unexpected status 42
boom
exit=3
```

The script's own header (`check-frontmatter.sh:7-8`) is honest — it says DEP002
means "skill-validator returned a **status** this script cannot interpret". It
is `README.md:282` that overclaims, and the behaviour is the invariant hole:
a version drift in `skill-validator` that changed the `validate structure`
subcommand's output shape while keeping exit 0 would silently pass every skill.

**Invariant that must hold:** a script must not emit a spec verdict unless it
read a payload it proved it could interpret. `verdict-guard.sh` already owns the
predicate (`json_document_conforms`); the spec source is simply not put through
it.

## Note on P1 count

IT-02 and IT-03 break the same invariant with the same shape and could
reasonably be P1 too. They are held at P2 only because their trigger — a `jq` on
PATH that runs but produces nothing — is rarer in the field than a
`skill-validator` version drift. A fix at Root A closes all three at once.

---

# P2

## IT-02 — `audit-report.sh` exits 0 with no report at all when `jq` is present but non-functional

**REAL. P2. 0.4.3 patch.** `skills/skill-audit/scripts/audit-report.sh:60`,
`:155-192`. Documented claim: `README.md:271` — "without jq there is no report
to generate: it exits 3 with `required tool not found: jq` and writes nothing to
stdout, **rather than dying part-way through**"; `README.md:273-276` — "a script
that cannot reach a verdict says which tool is missing and exits 3, rather than
reporting a result it did not compute."

`require_tool jq false` passes because `jq` is on PATH. Everything after it
consumes `jq` output unchecked.

```text
# jq present but exits 5 with a diagnostic
$ audit-report.sh <skill>
exit=5   stdout=''   stderr='jq: error
ERROR: check-paths.sh exited with unexpected status 5
jq: error'

# jq present but silent (exit 0, no output)
$ audit-report.sh <skill>
exit=0   stdout=''   stderr='ERROR: check-paths.sh --json payload contradicts its exit status 0'
```

The second line is the defect: **exit 0 and an empty stdout**. A caller doing
`audit-report.sh "$skill" | jq '.summary.passed'` (the pipeline `SKILL.md:82`
tells them to write) gets nothing and a success status. The first line is
`exit=5`, outside the script's own documented set of `{0,3}`
(`audit-report.sh:26`) and outside `SKILL.md:87`'s `{0,1,2,3}`.

## IT-03 — `check-paths.sh --json` exits 0 with an empty payload when `jq` is present but non-functional

**REAL. P2. 0.4.3 patch.** `skills/skill-audit/scripts/check-paths.sh:57`,
`:118`, `:121`, `:137`. Same root as IT-02, one level down.

```text
# jq present but silent
$ check-paths.sh --json <skill>
exit=0   stdout=''   stderr=''
```

Exit 0 with no payload is a claimed pass the script did not compute. The script
promised at `check-paths.sh:13-15` that `--json` "builds its verdict with jq and
requires it" and that the output is `{"findings": [...], "passed": bool}`.
Its direct consumer `check-structure.sh` does catch this (DEP002) — but any
other `--json` consumer, including a user following `SKILL.md:67`, does not.

## IT-04 — the documented Devin local install `devin plugins install .` fails

**REAL. P2. 0.4.3 patch (doc fix).** `README.md:349-350`. Reproduced against the
installed Devin CLI, in an empty scratch directory so nothing could be
installed:

```text
$ devin plugins install . -y
Error: local path sources can't sync to Devin Cloud; run `devin plugins install --local .` to install on this machine only.

$ devin plugins install ./ -y
Error: local path sources can't sync to Devin Cloud; run `devin plugins install --local ./` to install on this machine only.
```

The documented command is missing `--local`. Both `.` and `./` fail identically,
so README:335's worry that "`.` and `./` are not interchangeable" does not apply
to Devin — the missing flag does.

Two things this run *does* confirm, and they are worth keeping:
`devin plugins install <owner>/<repo>` is an accepted source form (it attempted
the clone), and Devin reads the manifest at exactly
`<root>/.devin-plugin/plugin.json`:

```text
$ devin plugins install --local . -y      # in a dir with no manifest
Error: invalid manifest: could not read manifest at <cwd>/.devin-plugin/plugin.json: No such file or directory
```

## IT-05 — `.codex/skills/` is not a location Codex loads skills from

**REAL. P2. 0.4.3 patch (doc fix).** `README.md:376` — "The exact path depends
on the agent (`~/.claude/skills/`, `.cursor/skills/`, `.codex/skills/`,
`.devin/skills/`, etc.)."

Verified against the vendors' current primary docs:

| Path in README:376 | Real? | Source |
|---|---|---|
| `~/.claude/skills/` | yes | matches README:365-366's own `cp` target |
| `.cursor/skills/` | yes | Cursor docs (also `~/.cursor/skills/`, `.agents/skills/`) |
| `.codex/skills/` | **no** | Codex docs list only `$CWD/.agents/skills`, `$REPO_ROOT/.agents/skills`, `$HOME/.agents/skills`, `/etc/codex/skills` |
| `.devin/skills/` | yes | confirmed live: `devin skills paths` prints `.devin/skills/<skill-name>/SKILL.md` |

A Codex user following README:376 for the manual-copy install puts the skill
somewhere Codex never reads, and it silently never loads. The correct
project-scoped path is `.agents/skills/`.

## IT-06 — the documented manual copy is not idempotent; re-running it nests the skill

**REAL. P2. 0.4.3 patch.** `README.md:364-366`, against `README.md:362` ("You own
the files and **pull updates when you choose**") — i.e. re-running these two
lines is the documented update path.

```text
$ cp -R skills/skill-audit ~/.claude/skills/skill-audit   # first run: correct
$ cp -R skills/skill-audit ~/.claude/skills/skill-audit   # second run
$ ls ~/.claude/skills/skill-audit
SKILL.md  references  scripts  skill-audit      <-- nested duplicate
```

`cp -R src dst` copies *into* `dst` when `dst` already exists. The nested copy
carries a second `SKILL.md`, and harnesses that walk the skills root recursively
(Cursor documents that it does) will register two `skill-audit` skills. The
invariant: the documented install/update command must be safe to run twice.

## IT-07 — `skill-rewrite`'s documented script invocation is cwd-relative and resolves nowhere

**REAL. P2. 0.4.3 patch.** `skills/skill-rewrite/SKILL.md:52` and `:119`.

```text
$ cd <repo root> && scripts/draft-rewrite.sh -t <skill>
no such file or directory: scripts/draft-rewrite.sh
exit=127
```

Every other script invocation in both SKILL.md files is anchored
(`"$skill_root/scripts/..."` — `skill-audit/SKILL.md:49-51,66-70,76`,
`skill-rewrite/SKILL.md:41-42`). These two are bare `scripts/…`, presuming
cwd is the `skill-rewrite` directory — and `skill-rewrite/SKILL.md` never
defines a `skill_root` for itself at all (Stage 0, `:29-32`, declares only
`target_skill` and `audit_report`). That missing input is the shared root of
both hits. Run with an absolute path the script works from any cwd, so this is
purely the documented form.

Related, same file: `skill-rewrite/SKILL.md:45` points at
`references/evaluation-matrix.md` "from `skill-audit`" — `skill-rewrite` has no
`references/` directory, so that path does not resolve as written either.

## IT-08 — `audit-report.sh` always exits 0, while `SKILL.md:87` tells the reader 0 means pass

**REAL. P2. 0.4.3 patch.** `skills/skill-audit/SKILL.md:87` — "Exit codes:
0=pass, 1=spec/path failure, 2=policy failure, 3=execution error" — sits
directly beneath the `audit-report.sh` usage at `SKILL.md:73-85`.
`audit-report.sh:26` states a *different* contract for itself
("0=report generated, 3=execution error"), and only that one is implemented.

```text
$ audit-report.sh tests/fixtures/f01/no-license
exit=0  {"passed":false,...,"policy_failures":8,"total_findings":8}
$ audit-report.sh tests/fixtures/f01/malformed-yaml
exit=0  {"passed":false,"spec_passed":false,"spec_errors":1,...}
$ audit-report.sh <skill>   # with skill-validator masked off PATH
exit=0  {"passed":false,...}  spec_error:"skill-validator not found. …"
```

A user wiring `audit-report.sh "$skill" && echo PASS` — the natural reading of
`SKILL.md:87` — prints PASS for a failing skill and for an unchecked one. That
last line is precisely the failure mode the v0.4.1 prerequisites section and the
PR-4 guards exist to prevent; the prerequisites table's own claim
(`README.md:269`) is narrower and **is** true (`spec` null, `spec_error` named,
`summary.passed` false — all verified). The gap is the exit code and the
`SKILL.md:87` sentence that covers it.

## IT-09 — a source that produces *nothing* is reported as a source with *no error*

**REAL. P2. 0.4.3 patch.** `skills/skill-audit/scripts/audit-report.sh:70-75`,
`:88-101`, `:185`, `:187`. Documented claim: `README.md:292-294` — "when a source
produces nothing it can read, it **names the source** in `spec_error` or
`policy_error` and leaves `summary.passed` false".

The error message *is* the source's own captured output. When the source emits
nothing, the message is `""`, and `:185`/`:187` map `""` to `null`.

```text
# skill-validator exits 0 printing nothing
$ audit-report.sh <skill>
exit=0  {"p":false,"se":null,"qe":null,"qs":80}
        i.e. summary.passed=false, spec=null, spec_error=null — a failing
        verdict with no stated reason at all

# skillscore exits 0 printing nothing
$ audit-report.sh <skill>
exit=0  {"p":true,"qe":null,"qs":null}
        contradicts README:270 ("audit-report.sh emits quality: null with quality_error")
```

Every *non-empty* misbehaviour is named correctly, which is why this survived —
verified good: garbage (`spec_error:"not json at all"`), a bare number
(`"7"`), two documents (`"{}\n{}"`), a crash (`"boom"`), a non-executable
`check-structure.sh`, unreadable policy stdout, and a relayed `DEP002` payload
(which does reach `.policy.findings` with diagnostics on stderr, exactly as
`README.md:289-292` promises).

## IT-10 — `profiler probe` cannot distinguish a wrong `--otel-file` path from a silent session

**REAL. P2. 0.4.3 patch.** `profiler/claude_code.go:154-170`. The README's very
first profiler example (`README.md:47`) run verbatim:

```text
$ ./profiler probe --harness claude_code --otel-file ./otel-export.json
{ "harness": "claude_code", "adapter_version": "0.4.1", ...,
  "capabilities": { "attribution":"none","skill_activation":"none",
                    "timing":"none","tokens":"none","tool_calls":"none" } }
exit=0
```

The file does not exist. Nothing in the output or the status says so. `capture`
on the identical input classifies it correctly:

```text
$ ./profiler capture --harness claude_code --session abc123 --snapshot sha123 \
    --skill-dir ./skills/my-skill --otel-file ./otel-export.json
"tokens": {"state":"error","reason":"failed to read OTel export file: open ./otel-export.json: no such file or directory"}
exit=2
```

`README.md:131-136` warns about exactly this hazard for `capture` ("exiting 0
after reading nothing is how a broken `--otel-file` path becomes a row of
zeros") and gives it exit 2 as protection. `probe` has neither the protection
nor the warning, and README documents no exit codes for `probe` at all. The
capability vocabulary is `{otel, none}` so it structurally cannot say "error" —
but the run status can, and does not.

---

# P3

## IT-11 — `check-quality.sh` relays `skillscore`'s exit status outside its documented set

**REAL. P3. 0.4.3 patch.** `skills/skill-audit/scripts/check-quality.sh:20`
(`skillscore "$skill_dir" --json` is the last command, so its status is the
script's). Documented set, `check-quality.sh:5`: "0=success, 3=execution error";
`SKILL.md:87`: `{0,1,2,3}`.

```text
$ PATH=<shim: skillscore prints "{}" and exits 9>:$PATH check-quality.sh <skill>
exit=9  stdout='{}'
$ PATH=<shim: skillscore prints "nope" and exits 0>:$PATH check-quality.sh <skill>
exit=0  stdout='nope'
```

The second line is Root A again: a non-JSON body is passed straight through to a
caller `SKILL.md:70` told to expect "quality scoring (JSON)". Lower severity
than IT-01/02/03 because quality is a score, not a verdict
(`README.md:294-296` makes that distinction explicitly).

`check-quality.sh` is also the one script in `skills/skill-audit/scripts/` that
does not source `verdict-guard.sh` at all — see IT-19.

## IT-12 — `check-paths.sh --json` leaks `jq`'s exit status

**REAL. P3. 0.4.3 patch.** `check-paths.sh:118`, `:121` under
`set -euo pipefail`. Documented set, `check-paths.sh` header and `SKILL.md:87`:
`{0,1,3}`.

```text
$ PATH=<shim: jq exits 5>:$PATH check-paths.sh --json <skill>
exit=5  stdout=''  stderr='jq: error'
```

Same root and same file as IT-03; separated because it is a distinct
contract breach (an unenumerated status vs a silent pass) and a fixer may close
one without the other.

## IT-13 — a broken `jq` is misattributed to `check-paths.sh` in the DEP002 message

**REAL. P3. 0.4.3 patch.** `check-structure.sh:135-146`. Proven with a
`check-paths.sh` shim emitting a perfectly conforming payload and exit 0, while
`jq` is the thing that is broken:

```text
# child healthy, jq exits 5
exit=3  {"...","rule":"DEP002","message":"check-paths.sh --json did not produce a readable payload"}

# child healthy, jq silent
exit=3  {"...","rule":"DEP002","message":"check-paths.sh --json payload contradicts its exit status 0"}
```

`payload_is_conforming` answers with `jq` (`verdict-guard.sh:198-199` says so),
so a `jq` failure is indistinguishable from a misshapen child payload and the
operator is sent to debug the wrong component. The exit status is right; the
diagnosis is not.

## IT-14 — `draft-rewrite.sh` writes outside where the docs say, and never cleans up

**REAL. P3. 0.4.3 patch.** `skills/skill-rewrite/scripts/draft-rewrite.sh:55`,
`:63`. `README.md:392` and `skill-rewrite/SKILL.md:55` say the output is
`REWRITE-DRAFT.md` next to the target's `SKILL.md`. It also writes a `mktemp`
file per invocation, leaves it, and embeds its absolute path in the draft:

```text
$ draft-rewrite.sh -t <skill>
Rewrite draft written to: <skill>/REWRITE-DRAFT.md
$ head -3 <skill>/REWRITE-DRAFT.md
# Rewrite draft: my-skill

Generated from audit report: /var/folders/tt/9rh023ks47lbz9wzqfqq316c0000gn/T/tmp.KUF6Nrvn5Z
$ ls -la /var/folders/.../tmp.KUF6Nrvn5Z
-rw-------  1 … 16 …    # still there
```

One orphaned file per run, and a draft whose provenance line points at a machine-
local temp path that is meaningless to any reviewer.

## IT-15 — the documented profiler build leaves an untracked binary that `.gitignore` does not cover

**REAL. P3. 0.4.3 patch.** `README.md:40-41` (`cd profiler && go build -o profiler ./cmd/`)
against `.gitignore` (two lines: `tmp/`, `.venv/`).

```text
$ cd profiler && go build -o profiler ./cmd/ && cd .. && git status --short
?? profiler/profiler        # 4.0 MB
```

A user who follows the README's first profiler instruction has a dirty tree and
a 4 MB binary one `git add -A` away from the repo.

## IT-16 — `tests/test_skill.sh` is the only suite that does not normalise its cwd

**REAL. P3. 0.4.3 patch.** `tests/test_skill.sh` (no `cd`), against
`tests/test_f01.sh:4`, `tests/test_walk.sh:4`, `tests/test_f02.sh:7` — all three
of which begin `cd "$(dirname "$0")/.."`. `README.md:410-414` lists all four
side by side with no stated cwd.

```text
$ cd / && /path/to/tests/test_skill.sh
PASS: .devin-plugin/plugin.json exists
Traceback (most recent call last):
  File "<stdin>", line 2, in <module>
FileNotFoundError: [Errno 2] No such file or directory: '.devin-plugin/plugin.json'
exit=1
$ cd / && /path/to/tests/test_walk.sh   -> 19 passed, exit 0
$ cd / && /path/to/tests/test_f01.sh    -> 561 passed, exit 0
$ cd / && /path/to/tests/test_f02.sh    -> 243 passed, exit 0
```

From the repo root all four pass (27 / 19 / 561 / 243), as does
`cd profiler && go test ./...`. One missing line in one file is the whole
finding.

(Adjacent, for the test-integrity lens rather than this one: the assertion that
crashed is `[[ -f "$manifest" ]]` followed by `assert "$manifest exists" true` —
the literal `true` makes the assertion vacuous, and `set -e` on the preceding
test is what actually enforces it.)

## IT-17 — a Go struct field name is offered to CLI users as a configuration option

**REAL. P3. 0.4.3 patch.** Reason string emitted by `profiler capture` with no
`--otel-file`:

```text
"reason": "OTel export not configured. Provide an OTel export file via --otel-file or OtelExportFile."
```

`OtelExportFile` is `ClaudeCodeAdapter`'s Go field (`profiler/claude_code.go:56`),
not anything a CLI user can set. It contradicts `README.md:248` — "`--otel-file`
is the only input today."

## IT-18 — README's DEP002 claim does not scope itself to `--json`, and text mode catches only one of the three cases

**REAL. P3. 0.4.3 patch (doc fix).** `README.md:279-281`. Measured, with
`check-paths.sh` shimmed:

| child behaviour | `check-structure.sh` text | `check-structure.sh --json` |
|---|---|---|
| unenumerated status (42) | exit 3, named | exit 3, DEP002 |
| unreadable payload | exit 0, relayed verbatim | exit 3, DEP002 |
| payload contradicts status | exit 0 / exit 1, relayed | exit 3, DEP002 |
| empty stdout | exit 0 | exit 3, DEP002 |

The behaviour is defensible — in text mode the child emits no payload to read
(`check-structure.sh:100-103`) — but the README sentence reads as unconditional.

## IT-19 — "The scripts share `verdict-guard.sh` … Each checks that it loads" is true of four of five

**REAL. P3. 0.4.3 patch (doc fix).** `README.md:298-303` against
`check-quality.sh`, which sources nothing. Verified across all three documented
guard states:

```text
verdict-guard.sh missing / malformed / truncated-at-100-lines:
  check-frontmatter.sh  exit=3, no payload   (all three states)
  check-structure.sh    exit=3, no payload   (all three states)
  check-paths.sh        exit=3, no payload   (all three states)
  audit-report.sh       exit=3, no payload   (all three states)
  check-quality.sh      exit=0, full output  (all three states)
```

The messages are exactly as documented — `cannot load …: missing or malformed;
no verdict was computed` for the first two states, `… did not load its guards;
no verdict was computed` for the truncated one, and no payload on any of them,
including `--json`. `check-quality.sh` is unaffected because it uses no guard;
the README sentence simply says "each".

## IT-20 — `.codex-plugin/plugin.json` omits the `$schema` field Codex's own minimal example carries

**SUSPECTED. P3. 0.5.0 deferral.** `.codex-plugin/plugin.json`. The Codex
build-plugins docs show the minimal manifest as
`{"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
"name": …, "version": …, "description": …}`. The repo's manifest has
`name`/`version`/`description`/`license`/`skills` and no `$schema`. The docs do
not state whether `$schema` is mandatory, and no Codex CLI is installed here to
settle it — hence SUSPECTED. The manifest **location** is correct
(`.codex-plugin/plugin.json` is the documented scaffolded layout), and skills
under `skills/` are auto-discovered, so this is at worst a missing hint.

## IT-21 — the four plugin manifests do not agree on how they spell the skills path

**REAL. P3. 0.4.3 patch.** `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`,
`.cursor-plugin/plugin.json` all say `"skills": "./skills/"`;
`.devin-plugin/plugin.json` says `"skills": "skills"`. Both forms work (the
Claude route was installed live; Devin resolves the plugin and exposes
`/skill-architect:skill-audit`), so this is presentation, not breakage — but it
is the kind of drift that makes a later "fix them all" miss one.

## IT-22 — the Cursor install row names neither what to copy nor where to put it

**REAL. P3. 0.4.3 patch (doc fix).** `README.md:337` — "Copy or symlink the
plugin directory to your Cursor plugins folder". Cursor's docs give both
concretely: the destination is `~/.cursor/plugins/local/`, and the thing to
place there is the plugin root (the directory containing
`.cursor-plugin/plugin.json`, i.e. the repo root — not `.cursor-plugin/`, which
is the reading the current wording invites). Cursor's docs also give the exact
symlink form, `ln -s /path/to/my-plugin ~/.cursor/plugins/local/my-plugin`.
The manifest location `.cursor-plugin/plugin.json` and the `name`-only
requirement are both correct as shipped.

## IT-23 — every plugin manifest lacks `author`; the Claude validator warns on it

**REAL. P3. 0.4.3 patch.** `claude plugin validate .` on the worktree:

```text
Validating marketplace manifest: …/.claude-plugin/marketplace.json
⚠ Found 1 warning:
  ❯ plugins[0] plugin.json → author: No author information provided. Consider adding author details for plugin attribution
✔ Validation passed with warnings
```

`marketplace.json` has an `owner` block; none of the four `plugin.json` files
has `author`.

## IT-24 — both skills declare `git` as a compatibility requirement and no script uses it

**REAL. P3. 0.4.3 patch.** `skills/skill-audit/SKILL.md:5` and
`skills/skill-rewrite/SKILL.md:5`: `compatibility: POSIX shell (bash 3.2+ or
zsh), git.` No script under either `scripts/` invokes `git`. The `bash 3.2+`
half of the claim is true — every script was re-run with `/bin/bash` (3.2.57)
forced first on PATH and behaved identically.

## IT-25 — the repo's own PATH-masking test harness can build a farm with broken entries

**REAL. P3. 0.4.3 patch.** `tests/lib/masked-path.sh:32-33`.
`[[ -e "$farm/$b" ]] && continue` is false for a **broken** symlink, and
`ln -s "$f" …` copies the PATH entry verbatim, so a *relative* PATH entry
becomes a relative (therefore broken) symlink inside the farm and permanently
shadows the real binary of the same name.

Reproduced by accident while building this audit's own harness: with `./bin` on
PATH, the farm got `bash -> ./bin/bash`, and every masked run died with
`env: bash: No such file or directory` and exit 127 instead of exercising the
script. `[[ -L ... ]]` in the dedupe test and an absolutised link target are the
two invariants. This only bites where a user's PATH carries a relative entry or
a broken link, which is why CI has never seen it — but it silently converts a
real test into a vacuous 127.

---

# What is CLEAN — walked and verified, so a fixer does not re-derive it

These are the enumerated items that earned an explicit "clean". They are load-
bearing: several are exactly the claims v0.4.1's prerequisites section and PR 4
were written to make true, and they **are** true.

**Prerequisites, tool-absent (`README.md:269-271`) — every claim verified:**

| Claim | Result |
|---|---|
| `skill-audit` Stage 2 stops with `skill-validator not found`, exit 1 | ✔ ran `SKILL.md:46-51` verbatim |
| `skill-audit` Stage 2 stops with `skillscore not found`, exit 1 | ✔ |
| `audit-report.sh`: `spec` null, `spec_error` names the tool, `summary.passed` false | ✔ |
| `check-frontmatter.sh` alone: exit 3, `required tool not found: skill-validator`, license gate never reached | ✔ (proved on a no-license fixture: exit 3, no `PL001`) |
| `check-quality.sh` exits 3 when `skillscore` is gone | ✔ |
| `audit-report.sh`: `quality` null + `quality_error`, `quality_score`/`quality_grade` null, verdict unchanged | ✔ |
| no `jq`: `audit-report.sh` exits 3, `required tool not found: jq`, writes nothing to stdout | ✔ |
| no `jq`: `check-paths.sh --json` and `check-structure.sh --json` exit 3 with a `passed:false` payload carrying `DEP001`, whatever the skill contains | ✔ (verified on both a passing and a failing fixture) |
| no `jq`: text mode unaffected | ✔ (`check-structure.sh` text on a failing fixture: exit 2, full `PL002`/`PL004` findings) |

**`DEP002` on an unreadable child (`README.md:279-281`), `--json` mode:** all
four child behaviours (unenumerated status, unreadable payload, self-
contradicting payload, empty stdout) produce exit 3 with a `DEP002` finding and
a `passed:false` payload. See IT-18 for the text-mode caveat.

**`audit-report.sh` composition (`README.md:289-296`):** verified for a
non-executable `check-structure.sh` (`policy_error` named, `passed` false), for
unreadable policy stdout (named, `passed` false), and for a relayed `DEP002`
payload (reaches `.policy.findings` with the child's diagnostics on stderr, not
in the payload). See IT-09 for the empty-output hole.

**`verdict-guard.sh` (`README.md:298-303`):** all three states × the four
scripts that use it — exit 3, no payload, guard named. See IT-19.

**Claude Code install — both documented routes, run live in an isolated config:**

```text
$ claude plugin marketplace add Okja-Engineering/skill-architect
✔ Successfully added marketplace: skill-architect
$ claude plugin install skill-architect@skill-architect
✔ Successfully installed plugin: skill-architect@skill-architect (scope: user)
$ claude plugin list
  ❯ skill-architect@skill-architect   Version: 0.4.1   Status: ✔ enabled
```

and the local route, including README:356-358's trailing-slash claim:

```text
$ claude plugin marketplace add ./
✔ Successfully added marketplace: skill-architect
$ claude plugin marketplace add .
✘ Invalid marketplace source format. Try: owner/repo, https://..., or ./path
```

`claude plugin details` resolves both skills; both skills' scripts were then run
**from the plugin cache**, from cwd `/`, and produced correct results — exec
bits survive the install, and `verdict-guard.sh` resolves from
`${BASH_SOURCE[0]}`, so nothing in the audit scripts is cwd-dependent.

**Manual standalone copy:** copied both skills into a project `.devin/skills/`
and confirmed with the real Devin CLI that they load:

```text
$ devin skills list
  /skill-audit   (./.devin/skills/skill-audit) - Evaluate an Agent Skill directory…
  /skill-rewrite (./.devin/skills/skill-rewrite) - Draft a rewritten SKILL.md…
```

**`skill-rewrite` without its `skill-audit` sibling (`README.md:369-374`):**
behaves *exactly* as documented — still exit 0, still writes `REWRITE-DRAFT.md`,
and the "Current state" section contains `No such file or directory` for
`check-frontmatter.sh` and `check-structure.sh`. Resolution is
`<skill-rewrite>/../skill-audit`, as stated. Honest documentation of a real gap.

**Profiler:**

- build clean, `./profiler version` → `profiler 0.4.1`
- `probe`/`capture` agree on a real export (`tokens`/`tool_calls`/`timing` all
  `otel`/`present`, exit 0)
- usage surface matches `README.md:133-136` exactly: unknown command, unknown
  harness, missing `--session`/`--snapshot`/`--skill-dir`, and `--export-file`
  all exit **1**; an unreadable supplied export exits **2**; no `--otel-file` at
  all exits **0** with an all-`unknown` profile
- `--export-file` message is verbatim what `README.md:256` prints
- a `--otel-file` pointing at a directory is classified `error`, not `unknown`
- `go test ./...` green

**The README's OTLP capture recipe, route (a) (`README.md:163-197`), executed
end-to-end** — its Python receiver extracted verbatim from the README, started,
POSTed two real OTLP batches, and the resulting `otel-export.ndjson` fed
straight into `capture`:

```text
POST -> 200 ; POST -> 200
{"tokens":{"state":"present","source":"otel",
 "value":{"input":1523,"output":412,"cache_read":20480,"cache_creation":3072}},
 "tool_calls":"present","timing":"present"}
exit=0
```

**Other clean items:** all 13 markdown link targets resolve; all 4 external URLs
return 200 (npmjs 403s bots — the registry API confirms `skillscore` exists at
2.0.2, and `brew info agent-ecosystem/tap/skill-validator` confirms the tap);
`skill-validator validate structure -o json` and `check -o json`
(`SKILL.md:64-65`) both exist and run; all three `jq` pipelines at
`SKILL.md:82-84` produce the documented values; a literal un-expanded
`~/.claude/skills/my-skill` argument fails loudly and names the path rather than
producing a wrong answer; nothing in any documented flow wrote outside its
documented location except IT-14 and IT-15.

---

# Triage

## 0.4.3 patch — behaviour fixes, no new capability, no schema change

| ID | Sev | One-line |
|---|---|---|
| IT-01 | P1 | `check-frontmatter.sh` never reads `skill-validator`'s payload |
| IT-02 | P2 | `audit-report.sh`: broken `jq` → exit 0, no report |
| IT-03 | P2 | `check-paths.sh --json`: broken `jq` → exit 0, no payload |
| IT-08 | P2 | `audit-report.sh` always exits 0 vs `SKILL.md:87` "0=pass" |
| IT-09 | P2 | empty source output rendered as `spec_error: null` |
| IT-10 | P2 | `probe` exit 0 + all-`none` on an unreadable `--otel-file` |
| IT-11 | P3 | `check-quality.sh` relays `skillscore`'s status |
| IT-12 | P3 | `check-paths.sh --json` leaks `jq`'s status |
| IT-13 | P3 | broken `jq` misattributed to `check-paths.sh` |
| IT-14 | P3 | `draft-rewrite.sh` leaks a temp file and embeds its path |
| IT-15 | P3 | `profiler/profiler` not in `.gitignore` |
| IT-16 | P3 | `tests/test_skill.sh` missing the cwd `cd` the others have |
| IT-17 | P3 | `OtelExportFile` offered to CLI users |
| IT-21 | P3 | manifests disagree on the skills-path spelling |
| IT-23 | P3 | no `author` in any `plugin.json` |
| IT-24 | P3 | `git` declared as a compatibility requirement, unused |
| IT-25 | P3 | `masked-path.sh` farm can shadow a real binary with a broken link |

## 0.4.3 patch — documentation-only, same release

| ID | Sev | One-line |
|---|---|---|
| IT-04 | P2 | `devin plugins install .` needs `--local` |
| IT-05 | P2 | `.codex/skills/` is not a Codex skills location |
| IT-06 | P2 | `cp -R` update path nests the skill on a second run |
| IT-07 | P2 | `skill-rewrite/SKILL.md:52,119` cwd-relative script path |
| IT-18 | P3 | DEP002 claim not scoped to `--json` |
| IT-19 | P3 | "each checks that it loads" — `check-quality.sh` does not |
| IT-22 | P3 | Cursor row names neither the source nor the destination |

## Honest 0.5.0 deferral

| ID | Sev | Why |
|---|---|---|
| IT-20 | P3 | `$schema` in `.codex-plugin/plugin.json` — SUSPECTED; needs a real Codex CLI to settle, and the manifest location and skills discovery are already correct |

## Note for the fixer

Nothing here would be caught by the delivered CI (`.github/workflows/ci.yml`):
it installs `skill-validator` and `skillscore`, relies on the runner's
preinstalled `jq`, never runs `claude plugin validate`, and never exercises an
install path. Whatever closes Root A should arrive with a test that puts a
*present-but-broken* tool on PATH, not only an absent one — the existing
suites cover absence thoroughly (`tests/test_f01.sh`, `tests/test_f02.sh`, 804
assertions between them, all green) and presence-without-function not at all.
