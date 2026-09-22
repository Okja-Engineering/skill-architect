# Changelog

All notable changes to `skill-architect`.

## Unreleased

## 0.5.0 — 2026-09-21

Six new profiler subcommands, a contract that fails the build when one of them claims more
than it read, and one flag on `skill-rewrite`'s drafter. The subject is 0.4.3's carried onto
new surface instead of onto old prose: a capability is what a probe read and nothing else,
and this release adds the machinery that refuses to let a new adapter say otherwise.
**Four things were deliberately not shipped** rather than shipped in a form that does not
work — the three unlanded harness adapters (Cursor, Codex *and* Devin, not one), three
`doctor` environment tiers, a time limit on `experiment run`, and a spool-to-profile
projection — and each is named below with the reason it was declined.

**No Cursor, Codex or Devin adapter ships, and none ever has.** Three adapter drafts existed
as unlanded work, and the audit of them is why none landed. Measured at this release by
copying each draft **alone** into `profiler/` at head and running `go build ./...`, one file
at a time: **`cursor.go` and `codex.go` do not compile, and `devin.go` compiles cleanly.**
The earlier measurement added all three at once, which makes Go stop at
`./codex.go:224:15: too many errors`, and the conclusion was generalised past the point the
compiler had reached. Both of the two read `otelMetric`, `otelLog`, `toInt` and `parseTime`,
which v0.4.1's OTLP rewrite removed — all four declared at `541af3e:profiler/claude_code.go`
(`:144`, `:150`, `:222`, `:244`), still declared at `30f374c`, and none of them declared at
`v0.4.1` — and `cursor.go` reads a **fifth** undefined name, `getBool`. That one has a
different history, and an earlier version of this entry had it backwards by saying it was
never declared here at all: `getBool` **is** declared, at
`0a83615:profiler/claude_code.go:521`, the same ref these citations resolve against. `git log
--all -S"func getBool" -- profiler/claude_code.go` names `1e4e845` — the commit that added
the three drafts — so it arrived with them, as the helper for `cursor.go:228` and read by
nothing else. It is undefined at this head for a reason that is not a removal: `1e4e845` is
not an ancestor of this release (`git merge-base 0a83615 <head>` is `30f374c`, which predates
it), so the only branch that ever declared `getBool` is the one that did not land. v0.4.1's
rewrite accounts for the other four and cannot account for this one, which postdates the
fork. `devin.go` reads none of the five.
Worse than stale, and this rather than the compile failure is why none of them lands — two
of them advertise what they cannot deliver, which is the one thing the contract below exists
to refuse. `cursor.go:60` sets `caps[MetricTokens] = SourceSQLite` whenever no OTel source
supplied tokens and the `--export-file` it was given can be `stat`ed, and `cursor.go:150`
answers that same capability with
`UnknownTokenResult("Cursor SQLite token schema not yet verified")`; `devin.go:37` reports
`tokens` from `session_data`, and `devin.go` never returns a count on any path —
`UnknownTokenResult` at `:75` and `:114`, `ErrorTokenResult` at `:88`, `:98` and `:107`.
There is no input for which either delivers what it advertised, so landing them was never a
porting job. **Those `file:line` citations resolve against `0a83615`**, currently the tip of
`origin/wip/0.5.0-profiler`; none of the three files exists at this release's head, so a
reader of this changelog needs the ref to open them. `README.md`'s harness
table said "Planned" for all three before this release and says it after. **The Cursor work
that is real does ship** — the hook spool, `analyze` and `doctor` — as capture
infrastructure rather than as an adapter, and the harness row draws that distinction.

**Added**

- **An adapter contract, enforced as a table that fails the build.** Three obligations, each
  asserted in the direction it fails: a capture names one session and refuses an empty id; a
  count that was not read has no key in the profile rather than a zero; and **AC9 in both
  directions** — a capability a probe advertised must be `present` in the profile from the
  source it named, and a value the profile carries must have been advertised. The checks are
  a function returning violations rather than assertions calling `t.Errorf`, so the table's
  own ability to fail is under test (`TestTheContractTableCanFail`), and the registry the CLI
  resolves `--harness` through is the same list the table walks — with a scan that fails when
  an adapter is constructed in `cmd/` and escapes it.
- **`profiler compare --baseline --candidate`** — the paired comparison, with the refusal
  that is the reason it can be trusted: two profiles whose `capability.adapter_version`
  differ are **refused, not reconciled**, because the difference between them may be the
  reader rather than the skill. The refusal is upstream of every subtraction, not a flag set
  afterwards, and both versions reach the report. Two differing `snapshot_hash`es are a
  **note and never a refusal** — comparing a skill before and after a change is the whole
  purpose. **`adapter_version` is the only identity mismatch that refuses**, and the other
  three are notes: a differing `snapshot_hash`, a differing `harness` and a differing
  `skill_dir` each emit a note, exit 0, and the subtraction happens. `snapshot_hash` is
  deliberate; the other two are a decision with a shelf life, named here so nobody reads
  the refusal as protection against subtracting two different meters. `compare` exits **2**
  when it ran and nothing was comparable — a new scriptable status, as is `experiment run`'s
  own 2 for the same condition.
- **`profiler experiment {design,plan,run}`** — a paired with-skill/without-skill run.
  Refusals at **four** times, and the design check is a table of rules rather than a list:
  a design may declare only what the runner actually does (`ordering: blocked`,
  `analysis_method: difference`, `stopping_rule: fixed`, `$PROFILE` in each command, every
  identifying field non-empty, and **both conditions naming the same harness**, ordered
  ahead of the registration check so it survives the day a second adapter is registered);
  a **plan document** given to `run --plan` is refused for no runs, a step with no command,
  a step with no profile path, or two steps writing the same path; a **run** stamps the
  profile path before and after the command, so a step that exits 0 and writes nothing is
  refused rather than compared as this run's numbers; and each profile is **read back** and
  checked against the step's declared `harness`, `snapshot_hash` and `skill_dir`. `run`
  calls the same comparison, so there is one copy of the adapter-version rule. The
  freshness stamp is `(exists, size, modTime)`, which catches a step that wrote nothing and
  does **not** catch a step that re-touches a byte-identical profile an earlier run left
  there.
- **`profiler ingest` and `profiler hooks {install,uninstall}`** — a Cursor hook spool. This
  is **capture only**: the payload is written as it arrived, so a reader written later works
  from files that already exist. Numbers survive the round trip exactly — the decoder uses
  `json.Decoder.UseNumber()`, because decoding through `map[string]any` puts every number
  through `float64` and a nanosecond epoch is about 1.8×10¹⁸ — the fixtures carry
  `1789332594304000000`, against the 2⁵³ ≈ 9.007×10¹⁵ a `float64` holds exactly. "No field
  is dropped and no value is changed" is the claim, and it has **three** named exceptions:
  object key order, a duplicate key, and **credential redaction, which is always on**.
  **The redaction split, stated the right way round** — an earlier draft of this entry had
  it inverted. `redactSecrets` runs in **both** modes: a field whose normalized name is
  `auth`, `authorization`, `credential` or `credentials`, or ends in `token`, `secret`,
  `password`, `passwd`, `apikey`, `privatekey`, `accesskey` or `sessionkey`, is replaced
  wholesale, and a credential-shaped substring inside a string (bearer, `sk-`, `AKIA`,
  `ghp_`, a PEM header) is replaced in place. `stripContent` runs **only under `--strict`**
  and replaces the fourteen content keys — `prompt`, `command`, `tool_input`, `tool_output`
  and their ten neighbours — with `{"_stripped_bytes": N}`, N being the JSON-encoded size.
  So **prompts, commands and tool I/O are written verbatim by default**, because on this
  tool they are the data, and `docs/profiler-spec.md`'s five-bullet contract is the
  statement of record. Two consequences named rather than left implied: `hooks install`
  registers no `--strict` and there is no environment variable, so metadata-only mode is
  reachable only by hand-writing `--command`; and `--strict` is not anonymity — Cursor
  documents `user_email` on every payload, it matches neither set, and it reaches disk in
  both modes. Installation merges into an existing `~/.cursor/hooks.json`, writes the
  `"version": 1` Cursor's schema marks required, and removes only our own entries.
- **`profiler analyze`** — summarises a spool in sixteen fields: file, line and byte counts,
  envelope fields, a census of payload keys, and how many lines were unreadable. It
  describes the **files**, not the session, and it makes no inference about tokens or tool
  calls — the guard is the type, not the prose. One of the sixteen is a **census of key
  values**: `payload_key_values` quotes the value of every key that is neither a content
  field nor numeric, up to 64 characters, unbounded in cardinality — so absolute working
  directories, conversation ids and any field not on the redaction list appear in it
  verbatim. It carries the same "read it before you publish it" caution as an experiment
  result, which this entry previously gave only to the experiment result.
- **`profiler doctor`** — what this machine can and cannot measure. **Two tiers, `none` and
  `export`**, and a tier is only ever what a probe read from a real file. The function that
  decides a tier takes a harness name and a path and cannot see the home, the spool or the
  environment, so there is nothing available to it from which a tier could be raised.
- **`profiler/queries/`** — eight independently runnable DuckDB queries over a spool, with
  the column list derived from the event type by reflection so a query that falls behind the
  schema turns a test red. "Independently runnable" is a **manual** claim standing beside
  automated ones, and the asymmetry is named rather than hidden: `duckdb` is a new external
  tool, it is not one of the ten the skills require, no Go test, shell suite or CI step
  invokes it — the tests read the `.sql` files as text — and the verification is that all
  eight were run by hand against a seeded spool on DuckDB v1.5.5 and all eight returned
  rows.
- **The Claude Code adapter reads skill activation.** The source is
  `claude_code.skill_activated`, **the event and not the `skill.name` attribute**. The event
  is logged when a skill is invoked and only then, so one record is one activation and its
  timestamp is an invocation time. The attribute also rides along on `token.usage`,
  `cost.usage`, `api_request`, `api_error` and `api_refusal`, where it marks a skill active
  *for that request* — a skill used across five requests carries it five times, at flush and
  request times. Reading those as activations would report a count and a set of times the
  harness never recorded, so an export thick with the attribute and carrying no event is
  `skill_activation: unknown`, and `profiler/testdata/otlp/skill_name_present.json` is the
  negative control that pins it.
- **`skill-rewrite`'s `draft-rewrite.sh` takes `-o|--output`.** It writes anywhere the caller
  can write, **except four destinations**: an existing directory (the redirect fails on its
  own and reports "could not open" when the truth is you named a folder), a symbolic link
  (`: >` follows it and truncates the far end, so the name and the landing place are two
  different places), any `SKILL.md` (a draft is not a skill, and written over one it destroys
  the document it was built by reading), and anything inside
  `$HOME/{.claude,.cursor,.codex,.devin,.config,.agents}` — **six** roots — because an agent
  reads `~/.claude/skills/` **as skills**, so a draft dropped in one is not a stray file but
  a document that may be loaded as instructions. `.agents` is in that list because the
  rationale demands it: `README.md` documents `~/.agents/skills/` as a global skills
  directory for Codex and for Cursor, a draft written there is read as instructions by the
  same argument, and the list is now compared against that README table as well as against
  `skills/skill-rewrite/SKILL.md` — the two used to agree with each other and both disagree
  with the document that says where an agent reads skills from. **The bound is decided after
  resolving the path, not from its spelling**: `$HOME/drafts/x.md` where `drafts` links into
  `.claude/skills` is refused, and `$HOME/.claude-notes/x.md` is accepted though
  `$HOME/.claude` is a string prefix of it. A path that cannot be resolved counts as
  protected and exits 3 — "I could not tell where this would land" is not "go ahead". Inside
  the bound `-o` is an ordinary redirect: it **overwrites an existing file and is
  idempotent**, so it re-runs over its own output the way the default destination always
  has. The four refusals apply to the destination `-o` names and not to the default, which
  still writes beside the target's own `SKILL.md` — and `-o ""` falls through to that
  default rather than refusing, so a caller passing an unset variable writes into the target
  skill's own directory, which is the outcome `-o` exists to avoid. No new exit status,
  and both of the two the header registers are reachable here: a destination this script
  **refuses** is the `1` it registers for a usage or target error, and one it **cannot
  resolve** is the `3` it registers for an execution error.
- **`tests/lib/mutation-runner.sh`** — the mutation runner, in the test library instead of
  rebuilt by hand in slice after slice. (No numeral here: the earlier drafts said five, and
  the thing being counted was uncommitted worker state rather than anything derivable from
  this tree — `tests/lib/mutation-runner.sh` landed once, in S9.) Seven verdicts, each with its own exit status
  and its reason on the same line: KILLED, SURVIVED, DID NOT BUILD, SKIPPED, DID NOT END,
  DID NOT RUN, NOT APPLIED. **The absence of a verdict is a verdict and never a pass** — the
  readers ask whether the suite reported a tally at all, rather than searching its output for
  a failure line, which asks "did anything fail" of a suite that may never have started. The
  restore is unconditional, verified by byte comparison, and is `cp` from a backup rather
  than any git operation.

**Changed**

- **`AdapterVersion` is `0.5.0`, and it is deliberately not held equal to the plugin
  version.** It tracks what a profile contains and moves as soon as that changes, which is
  inside a release rather than at the end of one; the five plugin manifests move when the
  release is cut. `tests/test_skill.sh` asserts both and asserts them separately.
  **The upgrade consequence, which this entry previously left for a reader to assemble from
  two bullets:** because `compare` refuses a pair whose `adapter_version` differ, **every
  profile captured with 0.4.x is refused by `compare` at this release.** Driven: a stored
  0.4.3 profile beside a fresh 0.5.0 one exits **2**, with the refusal naming both versions
  and every metric `comparable: false` upstream of any subtraction. That is the refusal
  doing its job, and it means a stored profile library has to be re-captured before it
  compares against anything from this release. 0.4.3 made the same disclosure for its own
  bump, in the release notes; this one is stated in both documents.
- **The profile schema stays `skill-architect/profile/v1`.** The rule is written down: v1
  stays while a v1 reader is merely *ignorant* of a new key, and v2 is required when a v1
  reader would be *wrong*. Set-diffing the profile's `json:` tags against `v0.4.3` gives
  **nine** added keys, and this entry used to name eight of them: `error_type`, `count` and
  `id` on a tool call; `category`, `detail`, `operation_name` and `confidence` on an
  attribution; `estimated_context_tokens`; and **`total`** inside it, which is the one added
  key without `omitempty` of its own. Each is optional and absent when it was not read —
  `total`'s predicate holds through its parent, since `*EstimatedTokens` is a pointer with
  `omitempty`, so a profile that estimated nothing carries no key at all.
  `estimated_context_tokens` is a **pointer** for exactly that reason: as a value
  type with `omitempty` it would emit `{"state":""}` on every profile, a state outside the
  vocabulary a v1 reader walks, and would have forced a v2 nobody wanted.
  **The rule's own hard case is in this release, and it is a value rather than a key.**
  `skill_activation` gained an `error` state (`ErrorActivationResult`), where v0.4.3 had
  `{present, unknown}` as the complete domain and said so in `types.go` — *"and has no Error
  constructor: skill activation is a property of the harness and of this adapter, not of any
  export, so no export can fail it"* (`v0.4.3:profiler/types.go:212`, quoted whole; an
  earlier version of this entry shortened it inside the quote marks). A supplied export that
  cannot be read now yields
  `skill_activation: error` alongside the other three. A v1 reader ignorant of the value
  still skips it; a v1 reader that **switched** on the two documented values is wrong about
  that one case. The schema stays v1 in this release, and the decision is recorded here
  rather than made silently, because by the rule quoted two sentences up this is the
  closest call in the release.
  **One key is neither optional nor absent, and it is in the capability report:** that
  report went from five keys at v0.4.3 to six, because `claude_code.go` sets
  `MetricEstimatedContextTokens: SourceNone` unconditionally. It is always present and
  always `none` — a signal reported as unavailable rather than a key nobody wrote — and it
  is in `capture`'s stored output as well as in `probe`'s, where `compare` reads it.
- **`doctor`'s `hooks`, `server_api` and `enterprise` tiers are gone, and the reason is the
  point.** A preview of this command reported a `hooks` tier when `~/.cursor/hooks.json`
  carried our entry, a `server_api` tier when `CURSOR_ADMIN_API_KEY` was set, and an
  `enterprise` tier when `OTEL_EXPORTER_OTLP_ENDPOINT` was. **None of the three is a
  measurement.** There is no Admin API client in this repository; an OTLP endpoint is where a
  harness *sends* telemetry, not a file this tool can read; and no adapter in this build
  reads a spool, so a registered hook is a capture waiting for a reader. A user upgrading
  from that preview will notice they are gone — they were removed because a tier is a claim
  about what can be measured, and each of those three raised one on the strength of a file
  existing or a variable being set. The `getenv` parameter went with them: nothing else read
  it. What the machine actually has is still reported, under `observed`, with counts.
- **`profiler/spool_profile.go` is now `profiler/spool_read.go`** (and its test with it).
  Internal, unexported, same package, no API change; nothing shipped the old name.
- `AGENTS.md` gains an Operating principles section, and `.gitignore` gains the boundary
  entries — `go.work`, `go.work.sum`, `skillgate/`, `skills/skill-gate/`, `.scuba/`, and
  `.venv/` widened to `.venv*/` — each with a comment saying why, checked in both directions
  so an over-broad pattern cannot silently make a shipped path unstageable.
- `tests/lib/out-of-scope-check.sh` is the out-of-scope barrier as a runnable script rather
  than a convention, held in both directions by `tests/test_harness.sh` over fixture
  repositories plus clean-index and near-miss controls.

**Fixed**

- **Four false documentation claims, found by sweeping rather than by trusting a list.**
  `README.md`'s "the comparison engine is not yet built" and "the integration point for
  **future** paired comparisons (F04)" were both falsified by `compare` landing.
  `.out-of-scope.md`'s F04 bullet now says what is genuinely out of scope — the plugin never
  launches a model or spends tokens of its own. And `docs/profiler-spec.md`'s "F04 is
  expected to compare only profiles carrying the same `snapshot_hash`" was false and **must
  stay false**: comparing a skill before and after a change is the paired comparison's
  entire purpose, so two snapshot ids are a note.
- **The coverage figure for `profiler/cmd` was an artifact, and the measurement is fixed
  rather than the number quoted.** Every CLI test runs the binary as a *subprocess*, and
  `go test` counts only statements executed in the test process — so adding a subcommand
  exercised end to end made the reported percentage *fall*. The test binary is now built
  with `-cover` when `GOCOVERDIR` is set, so the subprocess writes its counters into the same
  directory. `go test -cover ./...` at this release: **97.9%** for `profiler`, **94.7%** for
  `profiler/cmd`, **95.9%** for `profiler/internal/homesafe`. The ledger's recorded 12.6%
  for `profiler/cmd` was that artifact. Note the command: the `./cmd` figure holds under
  `go test -cover`, and `go test -race -cover ./...` reports a fraction of it, because
  `-race` breaks the `GOCOVERDIR` propagation the subprocess coverage depends on. The two
  only functions in `internal/homesafe` short of 100% are `PathContains` at **87.5%** and
  `resolve` at **95.8%**, by `go tool cover -func`. That package's figure went **down** over
  this release, from the 97.0% an earlier draft of this entry published, and the fall is the
  containment rewrite below rather than a regression in what is tested: the walk that
  replaced the string comparison asks the kernel more questions, so it has more error arms.
  Exactly **five** statements in the package are uncovered and every one of them is an error
  return — three from `os.Stat` on a component the walk has already reached
  (`homesafe.go:189`, `:199`, `:215`), one from `os.Getwd`, one from `os.Readlink` after an
  `Lstat` has already reported a symlink — so reaching them needs the filesystem to change
  underneath a decision that is already in progress. Reported rather than chased, because
  restructuring working code to move a percentage is the kind of change this release exists
  to refuse. And measured, so that nobody reports a
  false regression: `go test -race -cover ./cmd` gives **3.4%**.
- **Containment is decided by filesystem identity, not by comparing paths, in all three
  implementations of that decision.** Every one of them used to fold `.` and `..` textually
  over the whole string and only then resolve symlinks over the deepest existing ancestor,
  so a `..` that crossed a symlink was folded against the *link's own name* rather than
  against its target, and a destination that resolved **into** a protected directory was
  judged outside it. An earlier draft of this entry named resolution order as the repair —
  symlinks resolved before any `..` is folded — and that was the first fix rather than the
  property the barrier now rests on, because ordering the string operations correctly still
  leaves a string comparison, and **two more escapes lived in the comparison itself**: on a
  case-insensitive volume `~/.CLAUDE` is another name for `~/.claude`, and on APFS an NFD
  spelling is another name for an NFC one, so both resolved to strings that did not match
  and the write landed inside anyway. What ships instead compares the one handle no spelling
  can change: the shell halves walk the kernel's own resolution as a sequence of `cd -P` and
  ask `[ . -ef "$root_dir" ]`, device and inode; the Go half returns the directory the walk
  reached and asks `os.SameFile`. `filepath.Clean`, `Abs`, `Join`, `Rel`, `Dir`,
  `EvalSymlinks` and `strings.HasPrefix` appear in `homesafe.go` only inside comments
  explaining their absence, and `path_resolved`, `path_absolute` and `path_normalized` are
  gone from `tests/`, `skills/` and `docs/` entirely — the textual folders are **deleted**
  rather than left sitting in a file whose one invariant forbids them. Driven at this
  release, on the shipped drafter: with `drafts` a link into `$HOME/.claude/skills`, the
  destination `$HOME/drafts/../skills/x.md` is **refused** (exit 1, "lands in the live
  configuration directory … whatever it is spelled"), while `$HOME/.claude-notes/x.md` is
  accepted and written though `$HOME/.claude` is a string prefix of it — both directions,
  because a repair that closes only the hole costs a legitimate destination. **A name is
  compared in exactly one place, and there it is three-valued:** a root that does not exist
  yet has no identity, so the same bytes are inside, two all-ASCII names differing by more
  than case are outside, and anything else has **no answer**, which every caller turns into
  a refusal. Folding the case or normalising would each be a guess, and neither is
  fail-closed for both consumers — the drafter keeps writes *out* of a protected directory
  while the install suite keeps them *in* a scratch root, so "this may be the root's name"
  has to refuse rather than decide. The tests are pinned to the invariant and not to the
  patch: no case names an expected verdict, each spelling is performed twice over two trees
  — once as a real write, so the **filesystem** says where the bytes landed, and once
  through the barrier — and the two must agree
  (`TestPathContains_AgreesWithWhereTheWriteLands`,
  `TestPathContains_DecidesAfterResolution`). The case set is **generated** from the
  protected-root list crossed with a set of mutation operators rather than enumerated, which
  is what produced the case-folding and normalisation rows the hand-written table had none
  of. A path that cannot be resolved counts as
  protected. `profiler/internal/homesafe` is the Go half, one package rather than three
  copies. The
  suites that install hooks, write spools and run the CLI redirect `HOME` into a scratch
  directory, and the `$HOME` anchoring is itself asserted — a barrier that resolved a literal
  home would pass every refusal assertion on the machine that wrote it and would be reading a
  live configuration directory to do it.
- `doctor` looks for the command `hooks install` actually registers — not a literal written
  out here, which is how this entry came to name the pre-`--spool-dir` build while the
  `--home` bullet below stated the real one; it is built in one place in `main.go` and read
  through the same reader and the same match the installer and uninstaller use — instead of
  string-matching `hooks.json` for `"profiler ingest"`, which
  would have missed every real registration and counted a foreign hook that merely mentioned
  us. A mutation dropping the `|| true` survived the whole suite while both halves stayed
  self-consistent, so the shared call is now derived from `main.go` with a control.
- **`--home` scopes the capture as well as the registration.** The registered command
  carried no `--spool-dir`, so at hook time `ingest` resolved a spool from whatever `$HOME`
  was when the hook fired, while `doctor --home X` reported `X/.skill-architect/spool` — a
  directory the installed hook never wrote to, reported as the spool with every count in it
  zero, which made the README's own try-it-somewhere-else example point the reader at an
  empty directory. `hooks install` now registers
  `<absolute binary> ingest --spool-dir <home>/.skill-architect/spool || true`, and the
  layout is `profiler.SpoolDirIn(home)` in one place instead of written out three times and
  held together by a comment. Driven end to end: the registered command, run through a
  shell with `HOME` pointed somewhere else entirely, writes into the `--home` spool, and
  `analyze --spool-dir <home>/.skill-architect/spool` reads the line back.
- A corrupted line in a spool is skipped by the reader and counted, rather than costing the
  file.
- **`MetricSource`'s five unproducible values say so, and a guard holds them.** `hooks`,
  `hooks_estimated`, `session_data`, `server_api` and `sqlite` are legal values of a
  profile's `source` key that no shipped capture can produce, and none of them carried a
  comment, so the claim was made silently by the declaration — a reader seeing `sqlite` in
  the enumeration concludes some capture reads a SQLite database, and `server_api` is one of
  the three `doctor` tiers this release removed for advertising a surface nothing reads.
  Each now carries the note, and a test reads the constants **and their doc comments** out
  of `types.go` by AST, reads every non-test file in both packages for references, and
  requires each value to be either produced or annotated — in both directions, so a note
  left on a value that has *gained* a producer is also a failure. The values stay: they are
  the profile schema's documented `source` vocabulary.

**Documentation**

- `docs/profiler-spec.md` gains the adapter contract, the comparison contract, the experiment
  contract, the hook spool contract, and sections on summarising a spool and on reporting
  what a machine can measure. `README.md` gains "Comparing two profiles", "Running a paired
  experiment", "Capturing Cursor hooks to a spool", "Reading a spool back" and "What can this
  machine measure?".
- **Every marker in the tree that read "0.5.0" as an assignment is re-pointed, and a check
  holds that instead of a count.** This entry used to publish a number — four — and then
  enumerate three, and the real figure depended on whether you were counting lines or
  subjects; the class had by then been found and repaired in five rounds, each round in a
  place the previous round's reader did not look. So there is no number here.
  `tests/lib/forward-promise-check.sh`, driven from `tests/test_skill.sh`, reads **every
  tracked file** — the denominator is `git ls-files`, because two of the last round's
  markers lived in `tests/` and `skills/`, which the previous readers' roots (`.`,
  `../docs`, `../README.md`) do not reach — and fails the build on a line that *assigns*
  work to the release being cut, while leaving a line that records history alone: "0.5.0 did
  not add one" passes, and `tracked for <the version being cut>` does not. It reads a closed vocabulary of
  deferral verbs rather than the version string, and the reason is the ratio: `git grep -o`
  finds the version **101** times across 91 lines, the check reports no findings, so **every
  one of those 101 is correct** and a check on the string alone would fire on all of them.
  (An earlier draft of this entry said "about seventy … about sixty of those are correct",
  which undercounted the total and, worse, implied a third of them were wrong. Both places
  that carried a variant of it — the reader's own header comment and `tests/test_skill.sh` —
  now publish **no numeral at all**: a figure in a comment that nothing re-derives is stale
  by the next release, and the argument does not rest on it.)

  **That vocabulary is a set of verbs, and this release is where it stopped being a set of
  spellings.** The matcher tested the raw record, so every alternative in its own closed set
  was in effect a lowercase spelling, and a capitalised member of the set went past it
  silently — at a sentence start, at a bullet start, or inside a bolded lead-in, which is
  this repository's commonest prose shape. All 18 of the reader's fixture cases were
  lowercase, so no control held that half. The repair is not seven more alternatives in the
  list. The record is **case-folded once**, at the single point it reaches the shape tests,
  and the tests are left unanchored: capitalisation, ALL CAPS and mixed case become one
  `tolower`, and a leading `-`, `*` or `**`, indentation, a colon or an em dash before the
  verb stop being forms to enumerate at all, because a position was never a shape. All five
  shapes are now matched in any case and in any position in the record. (The caught form is
  still deliberately not spelled out here, because writing it into this section would be
  writing the thing the check refuses — and the check does refuse it: it fired on this
  reader's own header comment while this entry was being written, and that is the check
  working, not a false positive.)

  **And the verb and the version no longer have to share a line.** Each record is examined
  joined to the one before it, which is the same join a prose reflow performs, and a finding
  seen only that way prints `across a line break` in its shape. This was live rather than
  theoretical: this section's own earlier copy of this paragraph escaped the reader only
  because a break happened to fall between the verb and the version, so re-wrapping a
  paragraph nobody had edited would have reddened CI later. The join resets at a file
  boundary, at a section heading and at a blank line — none of those is a wrap — and
  sentence-ending punctuation blocks it, so a sentence merely ending in a deferral verb
  before the next begins with the version is not a finding.

  **The controls, because the previous ones could not reach this.** All 18 cases ran one
  line in a file of one line. There are **42** single-line cases now — 30 refusals covering
  each member of the case class, each of the five shapes capitalised, and the version word
  on the record but away from the `is`; and 12 acceptances holding the other side of the
  fold, each of which fails against a reader with no exemption at all. **That last property
  is not decoration.** The acceptance that stood for this boundary before did not have it:
  it ran on past the version into words the shape does not admit, so it never reached the
  exemption and passed identically whether the exemption was correct, widened or deleted —
  which is how the widening below shipped under a green suite. An acceptance nothing can
  make fail is not a control. On top of them the class is driven over a **real multi-line
  document** with the
  texture the reader actually meets: headings, bulleted lists with bolded lead-ins, wrapped
  paragraphs, an indented line and a fenced block, five promises planted in it and eight
  innocent lines naming the version planted beside them. The assertion is on the **set of
  line numbers the reader names**, written out as literals and checked against the
  document's own length, so a reader blind to a planted shape and a reader firing on an
  innocent line both fail it — an exit-status check would have passed on four of the five.
  The wrapped promise is asserted by the shape the reader reports, and driven both inverse
  ways: with the wrap closed up it must still be refused and no longer attributed to a
  break, and with the promise taken out those two lines must go quiet while the other four
  still fire.

  **What it still does not do, stated rather than left to be found.** The window is two
  records, so the verb and the version have to land on records next to each other; a promise
  spread wider than that is not seen. (An earlier wording here said "with neither the verb
  nor the version adjacent to the join", which reads as though a verb next to a break were
  enough. Driven, it is not: no single join carries both.) And a version
  surface standing beside the `is` exempts that occurrence of the bare `is <version>`
  shape. `CHANGELOG.md` and
  `RELEASE_NOTES.md` are
  scoped to the section for the release being cut, because 0.4.3's section saying what 0.4.3
  deferred is history and stays byte-identical — **and that scope now fails closed, which it
  did not when this entry was first written.** The section was found by an exact heading
  match, so a heading written any of several ordinary ways was not a narrower scope but no
  scope: every record left through the skip counter, the accounting added up perfectly, and
  the reader exited 0 over a document it had not read a line of. Driven end to end, an
  assignment planted inside the section went unreported that way. The repair is not a list
  of tolerated heading spellings, which would leave the next one — the probe folds case like
  everything else here, and a scoped file that never entered its section is a **finding**,
  named as one. The controls for it are new too, and they are the first in this file to
  exercise section scoping at all: every fixture case runs the reader in its one-file mode,
  which applies no scoping and no exemption, so the two halves of the reader that decide
  what it reads at all had nothing holding them. **Its own counts have other sides, and no
  numeral is published for any of them:** the files it opened are compared for equality
  against the tracked files that have a record in them (not against `git ls-files` outright —
  awk's per-file counter cannot fire for a zero-byte file, and
  `profiler/testdata/otlp/empty.json` is deliberately one, which is why an earlier draft's
  "162 of them" sat beside a `git ls-files` of 163); the records it read are compared against
  `grep -ac ''` over those same files, a different program counting the same thing rather
  than `wc -l`, which is short by one on each of the two files with no final newline; and the
  reader itself refuses `examined + skipped != lines`, so a limit inserted anywhere in the
  walk is caught by the reader on its own terms. What the markers were waiting for is still
  waiting: `probe` has no documented exit contract; there is no bundled receiver subcommand
  and `--otel-file` is still the only input; and schema v1 still has no field in which a
  `present` result can say what it did not count. The marker that *closed* is skill
  activation, which this release reads. The version-surface exemption named above is there
  because `AdapterVersion is` and "building it is" the same version are the same three
  words doing opposite jobs, and what it asks is a **relation, not a word count**: the
  subject standing beside *that* `is`, through any markup, has to be the surface, and each
  occurrence on the record is asked separately — so a record carrying an exempt clause and
  an assignment at once is still refused. It was a substring over the whole record until
  this round, and that was two holes rather than one: one exempt word anywhere excused every
  occurrence of the shape on that record, and the discrimination that remained was carried
  entirely by capitalisation, so **case-folding the record for the shape tests folded the
  exemption with it** and five assignments this reader had been refusing stopped being
  refused — caught here by a driven comparison against the previous head, not by the suite,
  because the accept case standing for that boundary turned out to pass against a reader
  with the exemption deleted outright. It is now six refusals and two acceptances, each of
  which fails against a reader with no exemption. What still gets past is a subject that
  genuinely is a version surface in a sentence that is nonetheless assigning work, which
  needs the sentence parsed.
- **The two release documents are now a guarded version surface.** Deleting the entire
  0.5.0 section from both of them used to leave every shell assertion green and
  byte-identical, and nothing in `tests/` mentioned either file — the lesson
  `tests/test_skill.sh` had already learned for `marketplace.json` was never applied to the
  two documents that *are* the release. The version is read from
  `.claude-plugin/plugin.json`; each document must carry a section for it; each section must
  have a non-blank line under it, because a heading with nothing beneath it is the same
  deletion one line later; and the two documents must name the **same set** of releases,
  derived from both and compared for equality, which fires whichever file a section went
  missing from.
- `README.md`'s signal summary and its Claude Code harness row name `skill_activation`. They
  read "tokens, tool calls, timing" and were one release stale.
- `README.md`'s statement that the install and update rows were executed "in this release"
  now names 0.4.3, the release that ran them. This release did not re-run them.
- The sentence that the hook spool section had — "this release reads nothing back out of it"
  — was falsified inside this same release by `analyze`, and now says the thing that is true:
  no profile is produced from a spool.

**Verification**

- **3137 assertions across seven shell suites, 0 failed, identical under bash 3.2.57 and
  bash 5.3.15**: `test_f01` 1912, `test_f02` 350, `test_harness` 276, `test_rewrite` 401,
  `test_skill` 126, `test_install` 52, `test_walk` 20. Against 0.4.3's 2499 across the same
  seven, re-measured from the `v0.4.3` tag rather than quoted — 1868 / 350 / 75 / 85 / 53 /
  48 / 20. Every figure here is each suite's own last line under
  `for s in tests/test_*.sh; do "$B" "$s" | tail -1; done` run once per shell; the three
  suites that moved most late in the release are the ones whose case sets became generated
  products rather than hand-written tables.
- **286 top-level Go test functions, of which 285 pass and one skips by design, and 1014
  subtests pass under `-race`** — 239 top-level in `profiler`, 35 in `profiler/cmd`, 12 in
  `profiler/internal/homesafe`; of the subtests, `profiler`'s 579 are 546 direct and 33
  nested one deeper, with 87 in `profiler/cmd` and 348 in `profiler/internal/homesafe`, of
  which 320 are nested one deeper because that package's bound is generated on two axes.
  Against 0.4.3's 93 and 452, measured the same way from the tag. The skip is
  `TestHomeBarrierChild`, whose body a parent test runs in a child process; it is named here
  because "286 pass" would be one short of true. 64 OTLP fixtures, against 59.
- Dogfooding, with `skillscore@2.0.2` and `skill-validator v1.6.1`: `skill-audit` **92.5
  (A-)**, `skill-rewrite` **89.5 (B+)**. Both validate with zero errors and zero warnings,
  asserted by `tests/test_skill.sh`.
- Every slice in this release proved its own checks by mutation — reverting each check and
  watching it redden — and the runner prints a verdict in every direction rather than
  treating the absence of a failure line as a pass. **Two of this release's own non-vacuity
  checks could not redden, and they were the checks standing behind everything else**, so
  they are now derived denominators compared for **equality** instead of hand-written
  floors — and by the end of the release the same treatment had been applied to everything
  in the file that stood behind a number somebody chose. Earlier drafts of this entry
  described the repair as it stood mid-release, with a floor of `examined -ge 200` against a
  real 761 and a reader taught the third of bash's function-definition spellings. Both
  sentences are about mechanisms that no longer exist, so here is what ships.
  **The assertion audit reads the closure under `source`, not the files it was handed.**
  Four of the seven suites get their code from `tests/lib/`, so one line added to a library
  could make a whole suite's assertions vacuous with the audit green; the set of files is now
  computed from the `source` directives, and a `source` whose path cannot be resolved is a
  violation rather than a shrug. At this head that closure is **nine** files — the seven
  suites plus `tests/lib/masked-path.sh` and `tests/lib/mutation-runner.sh` — reporting
  `sites=829 files=9 lines=11913 accounted=11913`. (The base this work started from measured
  `sites=767 files=7 lines=10288`; the 761/9397 pair an earlier draft published was an
  earlier commit still, which is exactly why the figure is given with its command rather than
  carried forward.) **Each counter has a second reader rather than a floor.** `lines` is held
  against `grep -ac ''` and deliberately not `wc -l`, because awk counts *records* and a file
  whose last line has no newline holds one more of them than `wc -l` reports. `accounted` is
  held against `lines`: every record leaves the walk through one of four paths and each
  increments it. And `sites` gets an other side from **outside the file entirely** —
  `tests/lib/harness.sh` records the line of every assertion it actually runs and holds that
  set against `--sites`, so a call site this reader never reached reddens the suite that ran
  it. bash is the other side. That pair was not sufficient on its own, and the way it failed
  is worth the sentence: `accounted` counts **disposal, not examination**, so one legal
  assertion whose *quoted* argument contained `<<WORD` was read as opening a heredoc, the
  rest of the file was consumed as its body, all three counters agreed, and two vacuous
  assertions sat in the swallowed region with the audit green. A heredoc is now detected from
  the **tokens** rather than from a regex over the raw line, and one that reaches end-of-file
  without its terminator is itself a violation — which refuses the whole class with no number
  in it. **There is no longer a list of function-definition spellings.** The reader reads the
  grammar — an optional `function`, a name, an optional parenthesis pair that may have
  whitespace in it — because `assert ( ) {` is legal on every bash this project supports and
  no list of three contained it; `tests/test_harness.sh` generates 14 spellings, sources each
  into a real shell, and holds the reader against what **bash** says each one defined. The
  EXIT-trap reader went the same way: any operand naming the exit pseudo-signal is read, in
  any case, including `0`, where a single `EXIT` literal let `trap cleanup 0` displace the
  abort guard silently. And the harness-name list is derived by reading the harness with the
  same reader that finds a shadow, held against the set bash reports having loaded (14 = 14),
  as is the guard-primitive set (16 = 16) — so neither carries a numeral at all.
  **All four floors are gone**, replaced by `-gt 0` beside a derived other side, or by an
  equality against the findings the same payload reports — the policy-failure count, for
  instance, is now held equal to the `PL`-prefixed failures the same report carries, with a
  `-gt 0` beside it so the equality cannot be satisfied by two zeroes. An earlier draft said
  the blanket exemption had become "an enumeration naming which floors remain"; that
  enumeration went stale inside the same release that wrote it, which is the argument
  against enumerations and not for a better one, so what stands in its place is a rule about
  shapes rather than a list of lines.
  **One residual is named rather than claimed away:** one
  assertion's *label* promises slightly more than its command decides; the mutation that
  matters kills through the load-bearing half, so it is a labelling defect and not a hole,
  and it is recorded in the slice record that found it.

**Known past changes, recorded late**

- **`skill-audit`'s quality score fell from 94 (A) to 92.5 (A-) in 0.4.3, and the 0.4.3 entry
  does not record it.** Measured here under one pinned `skillscore@2.0.2` across three trees,
  so the drop is content and not a tool version: v0.4.1 scores 94, v0.4.3 scores 92.5, and
  head scores 92.5. The category is `clarity`, 9/10 down to 8/10, on a
  `1 synonym pair(s) used interchangeably` warning against a `SKILL.md` that release rewrote.
  0.5.0 does not touch `skills/skill-audit/` and does not change the number. The 0.4.1 and
  0.4.3 entries are left as written; this is the record.
- **0.4.3's own entry about the install rows was false when it was written, and this file
  and `RELEASE_NOTES.md` both still carry it.** The 0.4.3 section below says *"`README.md`
  marks the Devin, Codex and Cursor install rows 'not verified in this release'"*, and
  `RELEASE_NOTES.md`'s v0.4.3 section says *"The Devin, Codex and Cursor install rows are
  marked unverified."* **`README.md` marked the Codex row verified at v0.4.3 and still
  does**, naming `codex-cli 0.153.4`, the `CODEX_HOME` isolation, the two commands and the
  `codex plugin list` result — and the row at this head is byte-identical to
  `v0.4.3:README.md:436`, so the contradiction was inside the v0.4.3 tree and not introduced
  since. It is the README that is right; the Codex route was run. Neither released section is
  edited, because a released section is a record of what was said, so the correction is here,
  where a reader who reaches the 0.4.3 entries has already passed it. The wider claim also
  reached the deferred ledger's install row, which has read "Devin / Codex / Cursor" since
  0.4.1; that row is not a published document (`.gitignore` excludes `.scuba/`, and
  `git ls-files | grep -c '^\.scuba'` is 0), so citing it was accurate reporting rather than
  part of the contradiction. What needed correcting was these two files, and this is that
  correction. **Devin and Cursor remain genuinely unverified**, each for a stated reason, and
  the 0.5.0 enumeration above names those two and not Codex.

**Known limits shipping with this release**

- **`experiment run` has no time limit. There is no cap of any kind.** It shells out to the
  command each step declares, in a loop, and waits. A step that never terminates never
  terminates. A draft of this command declared a `Budget{MaxTokens, MaxTimeMs}` that nothing
  read, and it was **dropped rather than shipped as a cap that does not cap** — a token cap
  in particular cannot prevent the spend it names, because the count is only known after the
  step. This release does prove a portable bash-3.2-safe deadline in
  `tests/lib/mutation-runner.sh` (`mutation_bounded`: background job, `kill -0` poll, TERM
  then KILL, no `wait -n`, no external `timeout`), and the Go standard library offers
  `exec.CommandContext`. **Neither softens the sentence above**: the first is a test library
  and nothing in `experiment run` sources it, and the second is not called. Run an experiment
  under a time limit you impose yourself.
- **No line in this repository was produced by a running Cursor, and nothing here can tell
  you the spool works against Cursor.** Cursor is not installed on the machine this was
  written and verified on. What is different from the earlier draft of this entry is that
  the documentary half was checked against Cursor's own current published hooks reference
  rather than left as an assumption, and it moved in both directions. **Confirmed:**
  `~/.cursor/hooks.json` is the User-scope location (below Enterprise, Team and Project
  scopes, which this tool does not write); the 21 names in `CursorHookEvents` are exactly
  the documented event set, with no twenty-second; and the top-level `"version"` the
  reference marks required — *"Config schema version. Must be a positive integer (use
  1)"* — is now written, where an earlier build of `hooks install` omitted it and left a
  file that would most likely have been ignored while `doctor` reported all 21 events
  registered. **Contradicted:** `cwd` is **not** a common payload field. The reference's
  "Input (all hooks)" block names `conversation_id`, `generation_id`, `model`, `model_id`,
  `model_params`, `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email` and
  `transcript_path`, and `cwd` is in none of them; it appears in **four** per-event payload
  blocks — `preToolUse`, `postToolUse`, `postToolUseFailure` and `beforeShellExecution` —
  three of which are events this build registers. (An earlier version of this sentence said
  those first two and `beforeShellExecution` "only", which dropped `postToolUseFailure`
  inside a sentence claiming the reference had been read field for field; it was checked
  again here against the live page.) The field carried on every payload for workspace
  location is `workspace_roots`, which this build does not promote — so the older version
  still, which listed `cwd` among the four fields "a payload" carries, was not merely
  unverified but wrong. What the promotion does with an absent `cwd` is emit **no `cwd` key
  at all**, not `""`: the field is `json:"cwd,omitempty"`, and driven at this head a payload
  with no `cwd` spools
  `{"ts":…,"event":"sessionStart","schema_version":…,"conversation_id":"c1","raw":{…}}` with
  no `cwd` member, and a payload carrying `"cwd":""` spools the same way. That is the right
  behaviour and a reader cannot mistake "Cursor sent nothing" for "Cursor sent an empty
  string"; `raw` keeps whatever arrived, either way.
  **Still unverified:** that a hook is invoked with one JSON document on stdin, and every
  tool-call payload shape — `preToolUse` and `postToolUse` were registered on a live emitter
  and never fired, because a nested `claude -p` could not authenticate here. What *was*
  exercised live is the lifecycle half and the whole read-back path, against a real hook
  emitter (Claude Code, not Cursor), with `analyze`'s and `doctor`'s every count checked
  against the raw file by hand.
- **`AppendSpool` takes no lock, and this was measured rather than reasoned about.**
  Concurrent invocation is the normal case, not an edge one: Cursor's reference lists Tab
  hooks (`beforeTabFileRead`, `afterTabFileEdit`) as a class separate from its agent hooks.
  Driven at this release: **40 concurrent `ingest` processes** appending to one daily file
  at 100 B and at 200 KB, and **16 at 8 MB** — a file of about 128 MB. Every round wrote
  **exactly one line per writer**, every line parsed, and in the 8 MB round all sixteen
  lines shared **one** distinct length: not one was split, truncated or interleaved. Those
  are the results that hold, and they hold structurally rather than by luck — each record is
  a **single `write(2)` under `O_APPEND`** with the newline in the same buffer, and each
  call opens its own descriptor, so in-process goroutines get the same protection.
  Earlier drafts of this entry published the absolute byte figures (a 128,003,312-byte file,
  every line 8,000,206 bytes). They are **dropped rather than corrected**, because nothing
  in this repository pins the payload they were measured over — no test, script or fixture
  produces it — so a reader cannot regenerate them, and re-running the round here with a
  payload of our own gives a different pair on the same structure (sixteen lines, one
  distinct length, a file exactly sixteen times one line plus its newline). A published byte
  count needs its generator in the tree; the structural result needs only the reasoning
  above, and that is what is claimed. **The residual risk, named
  rather than implied: `O_APPEND` atomicity is not guaranteed on a network filesystem**, so
  a spool under an NFS or SMB home is outside everything above and was **not tested** — no
  network mount was available here. A hook killed mid-write still leaves a truncated last
  line, which the reader skips and counts as unreadable rather than losing the file.
- **`hooks uninstall` does not return a virgin machine to as-found.** On a home with no
  `.cursor` directory, install-then-uninstall leaves the directory, a `hooks.json`
  containing `{"hooks": {}, "version": 1}`, and a timestamped backup carrying our absolute
  binary path in all 21 entries. The leftover file is schema-**valid** now that install
  writes `version`, so it will not break a hook config added by hand later; it is still not
  the machine as found, and the reasoning `hooks_install.go` applies to an empty event
  registration — *"a trace of us in a file we are meant to have left as we found it"* —
  applies one level up and is not applied there.
- **The activation fixtures' provenance is thinner than the disclosure covering them, and
  the disclosure names the wrong attribute.** `profiler/testdata/otlp/README.md` labels
  `skill.name` `[DOCS]` — and `skill.name` is exactly the attribute this release **rejected**
  as the source. What shipped keys on the *event*, and neither `claude_code.skill_activated`
  nor `invocation_trigger`, `skill.source` or `skill.kind` appears in either labelled
  paragraph: five new activation fixtures landed and the provenance ledger was not extended
  for the identifier the whole signal turns on. Both labelled paragraphs are byte-identical
  to `v0.4.3`. Checked against Anthropic's monitoring reference, three fixture attribute
  **values** are outside the documented sets, and the fixture counts below are `grep -l`
  over `profiler/testdata/otlp/` rather than a tally anyone kept:
  `invocation_trigger` is documented as
  `"user-slash"`, `"claude-proactive"` or `"nested-skill"`, and `slash_command` appears in
  **six** fixtures while `skill_tool` appears in **three** of those same six; `skill.kind`
  is documented as `"workflow"` when the
  skill is a workflow skill and **absent otherwise**, and two fixtures carry `"skill"`;
  `skill.source` is given as `"bundled"`, `"userSettings"`, `"projectSettings"` or
  `"plugin"` — introduced there with "for example", so calling it a closed set is slightly
  stronger than the source supports — and **two** fixtures carry `"user"`. **No number
  moves**: the extractor reads
  `skill.name`, passes `invocation_trigger` through as an opaque string, and reads neither
  `skill.source` nor `skill.kind`. It does mean the fixtures assert an emitter shape the
  vendor does not document, and rewriting fixture values is not release-gate work, so it is
  named here for the maintainer. Two attribute facts *improved*: `tool_name`, `success` and
  `decision` are now **observed** on a real Claude Code 2.1.221 export — `"Read"`, `"true"`
  and `"accept"`, each as a `stringValue`, which is what the fixtures assume — and the
  reference confirms `skill.name`'s redaction to `"custom_skill"` for a user-defined or
  third-party plugin skill without `OTEL_LOG_TOOL_DETAILS=1`, which a live capture also
  showed both ways. The envelope, the timestamp encodings and the scope names remain
  `[OBSERVED]`.
- **Every profile key this release adds is written by nothing that ships — all nine.** This
  entry used to say three while the schema entry above enumerated eight, and the answer is
  the whole set: `error_type`, `count` and `id` on a tool call; `category`, `detail`,
  `operation_name` and `confidence` on an attribution; and `estimated_context_tokens` with
  its `total`. The only place shipped code builds a tool-call entry sets `name`, `timestamp`
  and `success` and nothing else, so a rejected tool call and a failed one are still both
  `success: false` with no classification between them; no shipped code constructs an
  `AttributionData` at all, and `PresentAttributionResult` has no non-test caller; no
  shipped adapter produces an estimate. They are `omitempty` and absent, and the adapter
  contract forbids advertising them, so no profile claims otherwise; the channel landed and
  the distinction did not.
- **`attribution` is `unknown` for every profile this release can produce**, because Claude
  Code's telemetry carries no output-to-skill mapping at all, and
  `estimated_context_tokens` is never produced by any shipped adapter. Both are honest
  `unknown`s that the contract table holds to AC9 in both directions.
- **An experiment result embeds the whole normalized design, including both commands.** That
  is deliberate — a stored result must say what it ran — and it means a secret spelled
  literally into a command travels with the document. Read a result before you publish it.
- **`doctor` exits 0 whatever it finds.** "Nothing here measures anything" is the answer for
  most machines today and a status that called it a failure would make the command useless in
  the situation it exists for, so a script asking "can this machine measure?" has to parse
  the JSON.
- **CI runs one job on `ubuntu-latest`, and two of this release's own invariants are only
  exercised off it. This is an accepted risk, and it is load-bearing rather than
  procedural.** `.github/workflows/ci.yml` has one job, one runner, one bash, so **every
  "green on CI" claim in this release is a bash-5, Linux claim.** Two things follow, and the
  second is new in this release.
  **The shell half.** bash 3.2 exempts `[[ ]]` from `errexit` while bash 5 does not, which
  was 0.4.3's headline defect: a whole suite reported PASS for failing checks on the older of
  the two shells this project supports. Every suite here was therefore run locally under
  both, and returns the same count and the same per-suite split under each.
  **The filesystem half, and this one is sharper.** The containment barrier decides by
  filesystem identity, and two of the three escapes it closes are **case-folding and
  Unicode normalization** — `~/.CLAUDE` naming `~/.claude`, and an NFD spelling naming an
  NFC one. Those are properties of the *volume*, not of the code: on the macOS APFS home
  this was written on, `[ ROOT -ef root ]` is true and an NFD path is the same object as its
  NFC spelling, so the barrier must refuse a destination that a string comparison calls
  outside. **`ubuntu-latest` is case-sensitive and normalization-sensitive**, so on CI those
  same spellings genuinely name *different* objects, and the generated case set — which
  takes its expected verdict from where a real write lands rather than from a table — quite
  correctly asserts the opposite answer there and stays green. That is the generator working,
  and it also means **the macOS half of the invariant is never exercised by CI at all**: it
  runs on a developer machine, and a regression in it reddens there and nowhere else. So the
  local both-shells, macOS run is not a belt-and-braces step in the release checklist; for
  these cases it is the only place the assertion exists. Closing it means a second CI job on
  `macos-latest` — new CI surface rather than a gap in what shipped, named here as 0.6.0
  surface and deliberately not added in this release, because adding an untried runner to
  the workflow that gates the release is not a change to make at a release gate.

**Known limits, carried past this release**

- **The deferred ledger has 15 rows that name 0.5.0 as their home and are marked open**
  (`.scuba/teams/gap-audit-0.4.3/deferred-ledger.md`, audited at `5c847e1`, before 0.4.3
  shipped). Re-walked at this release: two are closed — the composite-attribute identity
  defect, closed by 0.4.3, and the `profiler/cmd` coverage figure, whose artifact is fixed
  above — and **the remaining 13 are open and this release closes none of them**. They are:
  per-finding `jq` re-invocation in both `--json` writers; `isMonotonic` unread; a data
  point's `flags` unread; a `--json` exit 3 with no payload on three caller-error paths;
  schema v1 having no channel for a refused series; the whole-export slurp; a truncated final
  line costing the whole capture; rejected-versus-failed tool calls being indistinguishable;
  the not-a-count reason's upper bound being a hair generous; the **Devin and Cursor**
  install rows being unverified; cumulative temporality never having been seen on real
  output; the tool attribute names being `[DOCS]` rather than `[OBSERVED]`; and
  `CaptureOpts.APIKey` being read by nothing. **Two of those rows are narrower than the
  ledger's own wording, and the 0.5.0 enumeration above states the narrow version.** The
  install
  row reads "Devin / Codex / Cursor" and has since 0.4.1, but `README.md`'s Codex row
  records that route as **verified** — both steps run against `codex-cli 0.153.4` with
  `CODEX_HOME` pointed at a scratch directory, `codex plugin list` then reporting the plugin
  installed and enabled — and has recorded it since 0.4.3, byte-identically. The row's
  narrative was already false about Codex against the tree it was audited over; the row
  stays open on Devin and Cursor, which genuinely are unverified and say why. **That
  narrowing is true of this section and not of this file**: the 0.4.3 section below carries
  the wider claim, and because a released section is never edited here, it still does. See
  "Known past changes, recorded late" above, where that is recorded as the past error it is.
  And the tool
  attribute row is narrower too: `tool_name`, `success` and `decision` are now observed on a
  live export, so what remains documentary there is the activation attribute set and
  cumulative temporality.
- **A `present` result still has no field in which to say what it did not count.** A refused
  mixed-temporality series, a skipped unreadable record, and records the provenance
  projection removed while others survived are all invisible inside a `present` total. It is
  a schema gap rather than a counting one, and it needs the caveat channel v1 does not have.
- **`probe` still has no documented exit contract.** It exits 0 in every case, including one
  where the export it was given could not be read; the reason goes to stderr and stdout
  carries the same report shape a caller already parses. Stated precisely, because an
  earlier draft of this line escalated "same shape" into "byte-identical either way", which
  is false: a readable export and an unreadable one differ on stdout in the capabilities
  they name, `otel` against `none`. What **is** byte-identical is every way of reading
  nothing — all **six** of a nonexistent path, a non-JSON file, an empty file, a file
  truncated mid-object, a `chmod 000` file, and no `--otel-file` at all produce **one**
  distinct stdout report between them, apart from `probed_at` — every capability `none`, no
  reason anywhere, nothing naming which of the six it was — so the only differentiator
  between "no telemetry was
  configured" and "your export is broken" is whether stderr is empty. A caller that must
  branch on a bad export uses `capture`, which exits 2.
- **There is still no bundled receiver subcommand.** `--otel-file` is the only input.
- **Two `hooks.json` backups taken in the same second collide**, the suffix being
  second-resolution. Reaching it needs **two** runs inside one second, not the three this
  entry used to claim: `hooks install` followed by `hooks uninstall` each take a backup, and
  driven inside one second they leave a single `.bak-<stamp>` file holding the
  post-install state — so the file the user started with is not recoverable from it. A plain
  re-run of `install` takes no backup at all, which is why it takes two runs rather than one.
  **And the first install on a machine with no `hooks.json` takes no backup either** — driven
  at this head, `hooks install --home <fresh>` reports `events_registered: 21` and
  `schema_version_added: true`, and the `.cursor` directory afterwards holds `hooks.json` and
  nothing else. A backup names a file that was replaced, and on a virgin home there was none;
  the reason the result has a `schema_version_added` field at all is that "wrote something,
  backed up nothing" has to be sayable. Worth stating plainly because "backed up before any
  write" is the natural shorthand and is false here: what is true is *backed up before any
  file is overwritten*.
- **No spool-to-profile projection ships**, and that was a choice rather than an omission.
  The draft of one set `tool_calls` to `present` from `hooks` on the strength of guessed
  field names — a capability claim **in the data rather than in the prose**, which no
  documented limit can reach, because a stored profile is subtracted by `compare`, aggregated
  by `experiment` and read months later by callers who never see the source. What landed
  instead is the read that is true whether or not the guesses are right.

**Deferred, with reasons**

- **The skill `metadata.version` fields are not bumped, and whether they move is an open
  decision for the maintainer rather than something to do quietly.**
  `skills/skill-audit/SKILL.md` declares `0.2.0` and `skills/skill-rewrite/SKILL.md` declares
  `0.1.0`, while the five plugin manifests go to `0.5.0`. **The skill versions are
  independent of the plugin version by design** — nothing asserts a relationship between
  them, and nothing in this release adds one. It is more conspicuous than it was: this
  release changed `skill-rewrite`'s documented interface by adding a flag to it, and that
  skill still declares `0.1.0`. Named here rather than moved, and no assert pins them.
- **The shell containment primitive now exists three times** — in `tests/test_install.sh`, in
  `profiler/internal/homesafe`, and in `draft-rewrite.sh`. They are not copies of one library
  and cannot be: one is a test suite, one is a Go package, and one is a shipped script that
  may depend only on its sibling skill. They are three implementations of one decision, and
  the drift is real rather than theoretical — the first shell version written from the Go one
  was weaker than its source. Named as a decision for a later release rather than started
  here.
- **`draft-rewrite.sh` can still exit 7**, from a failing `rm` in its EXIT trap, outside the
  `{0, 1, 3}` its header states. Unchanged from 0.4.3 and recorded there in full.

## 0.4.3 — 2026-09-20

Nine clusters, one release, one subject: claims that asserted more than the code delivered.
No new features, and no profile schema keys changed. The verification substrate could not
report a failure on the bash this project supports, so the suites standing behind every other
claim were partly decorative. The audit scripts asserted a completeness their own tool
handling did not have. The documented install and update routes had been written rather than
executed. `skill-rewrite`'s documentation described a draft its script does not write. And the
profiler required a session id, stamped it on the output, and filtered nothing.

**The profiler reports different numbers for the same export**, and that is the fix rather
than a regression. Both documented capture routes append to one file by design, so a profile
covered every session in it. On `profiler/testdata/otlp/two_sessions.ndjson`, 0.4.2 reports
49200 input tokens and three tool calls for *either* session; 0.4.3 reports 48000 and two for
one, 1200 and one for the other. A session that is not in the file at all came back `present`
with the same confident 49200; it is now `unknown`, with a reason naming how many records do
not carry the asserted id. `capability.adapter_version` moves to 0.4.3, which is what tells a
profile written by this adapter apart from one written by 0.4.2 — a stored 0.4.2 profile
beside a fresh 0.4.3 one is two different measurements, not a disagreement to reconcile.

**`check-frontmatter.sh` and `check-quality.sh` gained hard tool requirements**, `jq` among
them, on paths that previously worked without one. Measured by masking each tool out of PATH
in turn: at 0.4.2 `check-frontmatter.sh` refused for exactly one absent tool, `skill-validator`;
it now refuses for five — `awk`, `dirname`, `grep`, `jq` and `skill-validator` — in text mode,
and it has no `--json` mode, so text mode is where the change lands. `check-quality.sh` refused
only for `skillscore` and now refuses for `dirname`, `jq` and `skillscore`. The trade is
deliberate and stated in each script's header: reading *what* a source answered needs the
predicate that proves a payload readable, and that predicate is jq. Inferring a verdict from an
exit status alone is what let a `skill-validator` exiting 0 while printing non-JSON produce
`frontmatter OK`. The full ten-tool set is in `README.md`'s Prerequisites and is compared
against the scripts in both directions by `tests/test_f01.sh`.

**Fixed**

- `assert` runs a command instead of taking a value, so a failing check can report FAIL. 16 of `tests/test_skill.sh`'s 17 shell `assert` call sites wrote the real check as a bare `[[ ]]` on the preceding line and passed the literal `true` — and bash 3.2 does not apply `errexit` to `[[ ]]`, so on the older of the two shells this project supports a failing check printed PASS. Reproduced on the 0.4.2 tree: with `skills/skill-audit/references/best-practices.md` deleted, bash 3.2 printed `PASS: skill-audit best-practices reference exists`, and neither shell printed a single `FAIL:` line or reached the summary. Taking a command removes the shape rather than patching the call sites: there is no value left to hand in, and the guard and the assertion are the same statement. The Python heredocs and the multi-line awk became named functions, and `quietly` keeps a passing check silent while surfacing everything a failing one said. The population is the suite's own `assert` call sites: a count of 25 in this release's earlier records added the eight Python `assert` statements inside two heredocs, which are a different statement in a different language and were never the defect. `tests/test_harness.sh` carries the corrected figure — 23 across two suites, these 16 and `tests/test_walk.sh`'s 7 — and holds it mechanically.
- The suite runs from any working directory, and an abort before the summary is loud. `tests/test_skill.sh` had no `cd` of its own: run from `/` on the 0.4.2 tree under bash 3.2 it printed `PASS: .devin-plugin/plugin.json exists` and then died in the Python heredoc that could not open that file, with no summary and no `FAIL:` line. It is the same defect as the `assert` shape and is repaired with it, because fixing either alone re-opens the other. An EXIT trap now makes a run that ends before its summary a failure rather than a silence, and reclaims the scratch directory on every path.
- Four body-extraction implementations are one primitive in the shared guard. A markdown horizontal rule no longer ends the body and disables path checking: `tests/fixtures/f01/rule-before-refs` and `tests/fixtures/f01/no-rule-before-refs` differ by a `---` above the references section, and otherwise only in their frontmatter `name` and the one line in which each names the other — as each fixture's own body says, "byte-identical but for the rule below". At 0.4.2 `check-paths.sh` exited 0 on the first and 1 on the second. Both are 1 now. A skill whose body is a single prose line no longer reports a pass.
- The verdict guard's contract is "the tool is present **and** answered something this file proved it could read", not "a name resolved". An external's status is read where it is called, so — over the ten tools of `README.md`'s Prerequisites, which is the denominator this was measured on — no script leaks a status outside the set its own header states. Those ten are every external these scripts invoke except two: `rm`, which the drafter's cleanup calls and which is in neither the census nor the sweep, and `bash`, which the scripts invoke as `bash -n` on the guard and which cannot be absent while a script of theirs is running. `rm` is why the drafter's **exit 7**, recorded under known limits below, stands beside this sentence rather than against it. At 0.4.2 the sentence failed over that same denominator, in both of the ways it can: masking `grep` from `check-paths.sh` produced **exit 0 — a pass** — with `grep: command not found` as its only output; masking `wc` from `check-structure.sh` or `awk` from `check-paths.sh` produced **127**, outside the set those headers state; and masking `grep` from `check-structure.sh` produced **2** — inside the set its header states, but the status that contract reserves for a policy failure, so a verdict reported where none had been computed. Every one of those is now exit 3, naming the tool.
- Exit contracts derive from each script's own header rather than restating one contract across five scripts that never shared it. `check-quality.sh` and `skill-rewrite`'s `draft-rewrite.sh` are inside the shared guard for the first time.
- Two dependencies are deleted rather than guarded. `sed` and `tr` each did one thing the shell does itself — prefixing lines, and stripping the padding `wc` puts around a count — and each was a tool whose status nothing read. Neither is computed with anywhere now; both survive only as rows in the guard's known-answers table.
- The rule-ID census reads the IDs a script *can emit*, not the ones it spells. Three routes to a `--json` consumer leave no literal in the emitting script: an ID handed to the guard is quoted at some call sites and bare at others; `require_tool` raises `DEP001` on behalf of every script that states a tool precondition; and a relayed finding is rebuilt from variables or spliced in whole by the parent. The registry side had the mirror defect, matching a fixed list of the prefixes already in use, so a rule announced under a new prefix was invisible to the line that claims to name the whole set. Both sides read a rule-shaped token now, and relay attribution is transitive and cycle-guarded, which is what makes one unregistered ID fail `check-paths.sh`, `check-structure.sh` and `audit-report.sh` together.
- The masked-PATH farm masks the one binary it was asked to mask and shadows nothing else. The link target was copied verbatim from the PATH entry, so a relative entry became a link relative to the farm — which is not the working directory, and therefore broken — and the dedupe test was `[[ -e ]]`, which is false for a broken symlink, so the dangling name survived in front of every working binary a later PATH directory offered. A test that then died at 127 looked like a test of the script, when it was a test that never reached one. Nothing enters the farm now under a name that does not resolve.
- The documented update route converges on its second run. It copied into an existing destination, which nested a second copy of a skill inside itself, and it left the destination empty when a copy failed part-way.
- `skill-rewrite`'s documented invocations are anchored on a root the document defines. `scripts/draft-rewrite.sh` resolved only from a working directory the document never named, and `references/evaluation-matrix.md` resolved against neither skill from the repository root — `skill-rewrite` has no `references/` directory. Stage 0 now defines `skill_root` and `audit_root`, and every command and path downstream is written against one of them, which is what `skills/skill-audit/SKILL.md` has done since it shipped.
- `draft-rewrite.sh` refuses a `-a` report that is not there, instead of auditing afresh, building the draft from that, and reporting "No audit report provided". The two questions the old condition asked at once — was a report given, and is it readable — are separated, so the branch at the point of use is left with the only one it can answer.
- `draft-rewrite.sh` has an exit contract (`0=draft written, 1=usage or target error, 3=execution error`), and an option given without its value names the option. It read `$2` when there might be no `$2`, so `nounset` answered a mistyped option with `$2: unbound variable` — bash internals naming a position in the parser. It is refused once at the read, for both options and for any option added later.
- `draft-rewrite.sh` removes the temporary audit it makes for itself and states a provenance a reader can act on. The file outlived every run, one orphan per invocation, and its path was written into the draft as the draft's stated provenance — a `/var/folders/…/tmp.XXXXXXXX` that never meant anything outside the process that wrote it.
- CI pins `skillscore@2.0.2`, the one test dependency npm could change underneath the suites. `skill-validator` was already pinned at `v1.6.1` on the step above; `skillscore` was installed as `npm install -g skillscore` and took whatever the registry served. `tests/test_f02.sh` composes its assertions over `audit-report.sh`'s merge of `skillscore --json`, so that tool's output shape is load-bearing and a new major version breaks the suite with no change in this repository behind it.
- `tests/test_rewrite.sh` is committed executable. CI invokes each suite as a bare path, so it died at `Permission denied`, exit 126, before a single assertion ran; local runs passed because they went through `bash`, which does not need the bit.
- A profiler map attribute's identity no longer depends on the order its members arrive in, so a reordered attribute stops splitting one series in two and doubling its total.
- The skill-audit scripts resolve their own directory without consulting the caller's exported `CDPATH`, which could otherwise put a second line into the resolved value.

**Documentation**

- `skills/skill-audit/SKILL.md` no longer asserts a completeness the code does not deliver. The documented preflight exited 1 for a missing tool — the status the skill's own table reserves for a spec failure — and checked two tools of seven while never checking the one this release makes unconditional. The shared `verdict-guard.sh` that every audit script depends on was named nowhere.
- `skills/skill-rewrite/SKILL.md` describes the draft `draft-rewrite.sh` actually writes. It promised preserved frontmatter with a corrected `name`, five section templates, and a checklist mapping each failed or weak audit dimension to a concrete fix. The script writes none of the first, three of the second — one of which, `Validation checklist`, the document did not list at all — and a fixed five-item review list for the third, byte-identical for a skill that audits clean and one that does not, with the evaluation matrix never read. Building that capability is 0.5.0 work; describing it for another release was the worse of the two options. The section list is a `text` block so `tests/test_rewrite.sh` can hold it to the headings a run writes, and `tests/fixtures/rewrite/all-sections` and `tests/fixtures/f01/valid-minimal` between them cover every template the script can fire.
- `skills/skill-rewrite/SKILL.md` states its prerequisites: `Required tools: awk, cat, grep, jq, mktemp, skill-validator, wc.`, and the sibling `skill-audit` directory it borrows the check scripts and the evaluation matrix from. Mask any one of the seven and `draft-rewrite.sh` exits 3 naming that tool and leaves no draft. It names `verdict-guard.sh`, which every borrowed check and the drafter itself exit 3 without, and which no command in either document invokes — it is sourced, not run — so to a reader of the commands it looks like an unused file, and a pruner removing it takes out every check in both skills. What is missing is a command naming it, not a mention: the name is in nine files under `skills/`, which is every shell file the two skills bundle plus both `SKILL.md`s. The document's first attempt at this sentence claimed instead that nothing else named the file, which was false when written; `tests/test_skill.sh` now reads every shipped `SKILL.md` for that claim shape and carries the original sentence as the control it must refuse.
- Both shipped skills declare the compatibility they have. `compatibility: POSIX shell (bash 3.2+ or zsh), git.` was wrong three ways in both files: every bundled script is bash throughout (`[[ ]]`, `set -o pipefail`, `${BASH_SOURCE[0]}`), nothing works under zsh, and no script in either skill runs `git`. The line is now `bash 3.2+.` The claim is what was wrong; making seven scripts genuinely POSIX-sh and zsh clean — the six the two skills ship plus the `verdict-guard.sh` they all source, which is bash throughout as well and which the premise above counts — is new work with a new failure surface, which a gap-closing patch release is the wrong place for. Nothing declines zsh on purpose: under zsh `BASH_SOURCE[0]` is unset, the sibling resolution collapses to the caller's working directory, and the guard goes out of reach.
- The machinery holding a frontmatter claim walks a derived denominator over the shipped skills rather than reading one file. That is why the identical false compatibility line next door went unheld for two releases: the check was built for one skill and pinned to that skill's `SKILL.md`. A skill with no witness now fails.
- `README.md`'s install and update rows carry the status of each route **as executed in this release**, and the rows that were wrong against the vendors' own CLIs are corrected. A hedge written from vendor documentation is replaced with what running the routes showed.
- `tests/test_install.sh` states what it bounds and what it does not, rather than claiming containment it has not got. The destination is bounded and the bound is proven; the block is executed and can do anything a shell can; the protection is a redirected home and a scratch directory. It is not a sandbox, because a hostile document implies a hostile suite.
- `tests/test_harness.sh` holds every suite to the shared harness and holds the suite glob, `.github/workflows/ci.yml` and `README.md` to the same list — a suite CI never runs reports nothing, and neither does one a reader is never told to run. The list derives from the workflow rather than from a hand-maintained count.
- `docs/profiler-spec.md` describes the provenance projection: it is applied once on the read path before any extractor, so an extractor cannot be handed the whole export by accident; a record contributes only if it carries the asserted `session.id` and its instrumentation scope is not another product's; and the scope test is an exclusion rather than an allowlist, because a scope is optional in OTLP and a collector may drop it, so refusing an unnamed scope would trade a wrong number for no number. `probe` stays unscoped and says so.
- 93 Go test functions pass under `-race` — 86 in the profiler package, 7 in the CLI — and 452 subtests, 422 direct and 30 nested one deeper, against 0.4.2's 66 and 230. The shell suites are at **2499** across seven suites, against 0.4.2's 850 across four: `test_f01` 1868, `test_f02` 350, `test_rewrite` 85, `test_harness` 75, `test_skill` 53, `test_install` 48, `test_walk` 20. Every suite was run under bash 3.2.57 and bash 5.3.15 and returns the same count under each. 59 OTLP fixtures, against 55.

**Known past changes, recorded late**

- 0.4.2's Fixed section says "No `skill-audit` script reports a verdict a missing tool could not compute", and the v0.4.2 release note says the same as "No `skill-audit` script tells you a skill passed when it could not check". Both are false as written and were false when written. Measured against the v0.4.2 tree: `check-paths.sh` with `grep` absent exits **0** — a pass — having checked nothing, its only output the shell's `grep: command not found`; and `check-structure.sh` with `grep` absent exits **2**, which its header reserves for a policy failure, so a missing tool is reported as a verdict about the skill. The claim was true of the three tools that release's suites actually masked (`jq`, `skill-validator`, `skillscore`) and not of the set the sentence names. This release closes it for the tools the scripts compute with. The 0.4.2 entries are left as written; this is the correction.
- 0.4.2's same entry closes "Swept across every fixture, script, mode and masked tool, no invocation exits outside `{0, 1, 2, 3}`", and the v0.4.2 release note repeats it. Also false when written, and by the same mechanism — the sweep ran over three masked tools. Measured against the v0.4.2 tree, three invocations land outside the set: `check-structure.sh` with `wc` absent, `check-paths.sh` with `awk` absent, and `audit-report.sh` with `awk` absent, all three exiting **127**. Those three carry this correction on their own. The fourth measurement in this release's sweep — `check-structure.sh` with `grep` absent, exiting **2** — does not falsify this sentence, because 2 is inside the set the sentence names; it falsifies the one above, and is cited there. Every one of the four is exit 3 here. The claim is still not universally true at this release, and the one member outside it is recorded below rather than asserted away. The 0.4.2 entries are left as written; this is the correction.
- 0.4.1's Documentation section says `README.md`'s Prerequisites block names `skill-validator`, `skillscore` and `jq`, and that "`jq` has no guard". That stopped being true at 0.4.2, which made `--json` require `jq` unconditionally, and it is further from true here: `jq` is a hard precondition of all five `skill-audit` scripts — unconditionally in `check-frontmatter.sh`, `check-quality.sh` and `audit-report.sh`, and in the `--json` modes of `check-paths.sh` and `check-structure.sh`. The 0.4.1 entry is left as written; this is the correction.

**Known limits shipping with this release**

- **A durability clause in `tests/test_install.sh`'s prose is false.** The file says taking the destination first "keeps a pathological document from ending this suite in a CI timeout with no summary printed". Only the destination is length-bounded. The path-word pass is quadratic over *every* word, and a long word that is not the destination reaches no bound at all: measured at this release, that pass alone takes 0.9s on a 100,000-character non-destination word and 3.6s on 200,000 — four times the cost for twice the input — and **accepts** the block on the way there. The bound is real; the sentence claims more of it than it does.
- **`draft-rewrite.sh` can exit with a status its own header does not name.** Its EXIT trap removes the temporary audit with an `rm` that is the last command of an AND-list, so `errexit` is not exempt from it: when `rm` runs and fails, the trap dies before its `return 0` and the script exits with `rm`'s status. Reproduced on both shells with `rm` shadowed by one directory holding one real file: **exit 7**, nothing on stderr naming `rm`, the draft written, and the temporary audit left behind. Three things wrong, all of them the original signature — the status is the tool's, nothing names the tool, and 7 is outside the `{0, 1, 3}` the header states. `rm` is also absent from the tool census in `README.md`, because both the census and the break-mode sweep derive their denominator from the guard's registration sites and `rm` is in neither.
- **The second of those arrived in the round that closed its own class.** The cluster whose subject is that every script reads its externals' statuses is the one that rewrote the drafter's inline trap into a `cleanup()` function and added the line giving the trap ownership of an incomplete draft. That is the cluster this limit is a member of. It is recorded here rather than omitted because a release whose subject is claims that overstate what was proven cannot quietly drop the counterexample its own work produced.

**Known limits, carried to 0.5.0**

- **A `present` result has no field in which to say what it did not count**, and this release adds a third case to that one gap. A refused mixed-temporality series and a skipped unreadable record were already invisible inside a `present` total; now records the provenance projection removed while others survived are too — when *some* of a signal's records lack the asserted `session.id`, or carry another's, the survivors make the signal `present` and the value is silently reduced with no reason attached. When *no* record survives, the signal is `unknown` and the reason does name how many were passed over and why. It is a schema gap rather than a counting one, and the caveat channel is the 0.5.0 item `docs/profiler-spec.md` already reserves.
- **`isMonotonic` and a data point's `flags` are still unread.** Neither field is read anywhere in the profiler's non-test sources: `isMonotonic` appears in none of them, and no OTLP struct declares a `flags` member at all — `otlpDataPoint` carries `attributes`, `startTimeUnixNano`, `timeUnixNano`, `asDouble` and `asInt`, and nothing else. The lowercase token `flags` does occur in `profiler/cmd/main.go`, where it means the CLI's own command-line flags; that is a different thing with the same name, and the sentence used to overreach by not saying so. Both are unchanged from 0.4.2 and carried for the reasons recorded there.
- **A `--json` exit 3 still carries no payload on three caller-error paths** — a usage error, a target that cannot be resolved, and a guard file that will not load. Verified unchanged at this release.
- **`skill-rewrite`'s draft is still not the draft a rewrite wants.** No frontmatter is carried through, there is no template for `Deterministic actions`, `Orchestration`, `AI judgment` or `Constraints` — four of the six sections the draft's own `Proposed structure` requires, as `skills/skill-rewrite/SKILL.md` says — the `Action items` list is fixed text identical for a skill that audits clean and one that fails, and `skill-audit`'s evaluation matrix is never read. All four gaps are pinned as absences in `tests/test_rewrite.sh`, deliberately the wrong way round for a wish list: when the capability is built they go red, which is what stops the document being left describing the old draft a second time.

**Deferred, with reasons**

- **The skill metadata versions are not bumped and this remains an open user decision.** `skills/skill-audit/SKILL.md` is at `0.2.0` and `skills/skill-rewrite/SKILL.md` at `0.1.0`. Both skills changed substantially this release — `skill-rewrite`'s documentation and script more than any release since it shipped — which by the convention the 0.3.1 entry set would be a bump. Nothing in this release's mandate calls for one, so the numbers are left where they are and named here rather than moved quietly.
- **The bash 3.2 half of the suites is not in CI.** `.github/workflows/ci.yml` has one job on `ubuntu-latest` with one bash, so a failure that occurs only under bash 3.2 passes CI. That asymmetry is the exact one the `assert`-shape defect above lived in for two releases, and it is why every suite in this release was run locally under both shells. Closing it means a second CI job with a bash 3.2 to run under, which is new infrastructure rather than a gap in what shipped.

## 0.4.2 — 2026-09-14

Two fixes, one release. The profiler's cumulative token merge was wrong in three ways that
each read as "fewer tokens than the session spent", and no `skill-audit` script could tell
"this passed" from "I could not check". No new features, and no profile schema keys changed.

**Totals for the same cumulative export differ from 0.4.1**, in the direction of being
correct: a capture spanning several resources or scopes now adds its series instead of letting
one replace another, a counter that restarted inside the capture keeps the run before the
restart, and a flush that reported less than an earlier one on the same run no longer takes
that run down with it. Delta temporality — Claude Code's default, `aggregationTemporality: 1`,
observed live — is unaffected throughout. `capability.adapter_version` moves to 0.4.2, which
is what tells a profile written by this adapter apart from one written by 0.4.1.

**Fixed**

- A token time series is identified the way OTel identifies one: the resource it was exported from, the instrumentation scope that recorded it, the metric's name, and the data point's whole attribute set. This adapter keyed on the data-point attributes alone, so a capture aggregating several resources or scopes under cumulative temporality merged their series and let one running total replace another — two resources reporting 100 and 200 gave 200, not 300, where the same pair under delta correctly gave 300. `schemaUrl` is deliberately not part of that identity: it declares which version of the semantic conventions the attributes follow, not a different origin, and including it would split one series in two and double-count it. An absent `resource` and a resource carrying no attributes are one resource, for the same reason: a resource is its attribute set.
- A cumulative series is a set of runs, and `startTimeUnixNano` says which run a point belongs to. It was not read at all, so a counter that restarted inside one capture discarded everything before the restart — 100 followed by a restart reaching 20 gave 20, not 120. The points sharing a start are one run; a point carrying a different start is a counter that restarted, and its run sits beside the earlier one rather than replacing it, so the series contributes the sum of its runs. Runs are keyed by their start rather than by file order, so batches written out of order still add up. A `startTimeUnixNano` of `0` is a `startTimeUnixNano` that is absent: it is a proto3 `fixed64` and OTLP mandates the proto3 JSON mapping, in which an explicit zero and an omitted field are two encodings of one message — `protojson` writes `"0"` with `EmitUnpopulated` on and omits the field with it off. The same rule now holds for `timeUnixNano`, which read them apart in the same reader. A delta point's `startTimeUnixNano` is the start of that point's own interval rather than of a run, and is not read.
- What a run holds is the **greatest** running total its points reported, and `timeUnixNano` is not read by the merge at all. These are monotonic counters: a running total only goes up, so the values carry their own order, every point of a run is a prefix of the greatest, and a flush reporting less than an earlier one on that run is a capture contradicting itself rather than tokens given back. Ordering by `timeUnixNano` let such a flush take its run down with it: a run reporting 900 at t1001 and then 500 at t1002 gave 500 at v0.4.1 and gives 900 here. The time-ordering apparatus is deleted rather than corrected, and that is the more interesting half. The obvious correction — keep the ordering, and let each run hold its latest point by `timeUnixNano` — breaks the property this merge is built on. Measured against a build of that candidate, since it never shipped: one run at start 1000 carrying an untimed 900, a 500 at t2000 and an untimed 1000 reports 500, and erasing the start time raises it to 1000, so supplying the field that says which run a point is from would make the profile report *fewer* tokens — adding information would lower the number. v0.4.1 reports 500 either way, because it does not read `startTimeUnixNano` at all; this release reports 1000 either way. Monotonicity is now stated as the assumption it always was; `isMonotonic` is still unread, and that is named under known limits below.
- A cumulative point whose `startTimeUnixNano` is absent, zero or unreadable cannot name its run, but it came from one — either a run that named itself, or a run nothing else in the capture observed — so it is neither a run of its own nor free. The series now contributes the **least total its runs can account for**: the runs added, plus whatever such a point reported over and above the largest of them. Both ends of that are wrong in a direction already shipped. Counting the point as a run of its own adds mass no point ever reported, so one export flush that omitted one field doubled a session; taking the series to be the greatest running total observed anywhere on it throws away the runs the point did not join, which are tokens the session really spent. Runs of 100 and 20 beside an unplaceable 500 give **520** — 500 drops a run that named itself, and 620 invents a third — and runs of 100, 100 and 100 beside an unplaceable 150 give **350**, the case where the unplaceable total is above the largest run but below their sum, so a rule comparing it only with the sum never fires and 50 reported tokens go missing with no case to notice. The property behind all of it, and the one to check a change here against: taking information away — erasing a start time — may lower the total and may never raise it.
- A series carrying both delta and cumulative points is refused whole and counted, rather than resolved one way. The two are opposite instructions — add, or supersede — so every resolution invents a number the export does not contain: adding the increments to a running total counts them twice, dropping them counts them not at all. 0.4.1 treated such a series as cumulative and let the point with the latest `timeUnixNano` win, so which number it produced turned on the timestamps rather than on write order — a cumulative 50 at t9000 written before a delta 100 at t1000 gave 50. The refusal is counted per *series* rather than per data point, because the points are individually fine and it is their company that is malformed, and the `tokens` reason names it. Refusing one series is not refusing the export: the well-formed series beside it still count, and `tokens` is `unknown` only when no series survives.
- No `skill-audit` script reports a verdict a missing tool could not compute. `check-structure.sh --json` with `jq` absent swallowed the missing binary with `2>/dev/null || true`, lost every path finding, and took the zero-findings branch that hard-coded `passed: true` — a clean bill of health at exit 0 from a check that never ran. `check-paths.sh --json` died instead at a bare `jq: command not found`, exit 127 with no payload at all. `check-frontmatter.sh` with `skill-validator` absent captured its 127, special-cased only exits 3 and 1, fell through to the license gate and printed `frontmatter OK` at exit 0 — and printed `POLICY FAIL [PL001]` at exit 2 for a malformed-YAML skill it had not validated. A new `skills/skill-audit/scripts/verdict-guard.sh` owns the one sentence behind all three: `cannot_compute` writes the reason to stderr, adds a `passed: false` payload on stdout when its caller is in `--json` mode, and exits 3, and `require_tool` is its missing-tool caller. `check-frontmatter.sh` has no `--json` mode and so reports on stderr alone. `--json` now requires `jq` unconditionally in both writers and in `audit-report.sh` — a stated precondition, not a data-dependent one. Text mode is unaffected.
- A child result a parent does not recognise is an execution error, not a pass. `check-structure.sh` mapped only child exit 1 onto failure, so `check-paths.sh`'s 127 read as success; it now enumerates the child's documented statuses and treats anything unenumerated as an execution error, stops folding the child's stderr into the payload it parses, and no longer reads an unparseable or absent payload as "no findings". A child exiting 0 having printed nothing did the same thing by a different route and is closed by the same pass. `check-frontmatter.sh` enumerates 0, 1, 2 and 3 with `*` as an execution error. Both `--json` writers derive `passed` from the computed verdict in one place, and the zero-findings shortcut is gone.
- `audit-report.sh` no longer reads an unreadable source as a clean one. It captured its only guarded child with `2>&1`, so a `passed: false` payload never parsed and findings stayed empty. Caught on this release's own branch, at the point where `check-structure.sh` had gained `DEP002` and the composer had not yet been told to read it: with `check-paths.sh` removed, `check-structure.sh --json` returned a `DEP002` payload at exit 3 and the composed report was `{"passed": true, "total_findings": 0}` at exit 0. stdout is now captured alone, per the child's own contract that the payload is stdout and diagnostics are stderr, and a source yielding nothing readable is named in a `policy_error` with `summary.passed` false, mirroring the existing `spec_error`. The composer also proves each source's shape before it reads a verdict out of it: `skill-validator` emitting `0` or `[1,2]`, `skillscore` emitting `true`, and either emitting a two-document stream all took `jq` down — exit 5 or exit 2, stdout empty, no report at all. One `json_document_conforms` in the guard owns the half that is the same for every source, that there is exactly one document, and each source states only the shape it is actually read as, so the quality source's claim reaches inside `overallScore` because the read does, while a `skillscore` payload with no `overallScore` at all is still read and simply leaves the score null.
- The guard's JSON string encoder is settled by ordinal, so its output no longer depends on the locale. Built by interpolation at first, it emitted invalid JSON for a message containing a quote; the replacement escaped through `[[:cntrl:]]`, which under a UTF-8 locale matches bytes bash reports as *negative* ordinals — `printf 'a\x80z'` came out as a sixteen-hex-digit escape under `en_US.UTF-8` and passed through untouched under `C`, and the valid C1 controls U+0080–U+009F, which JSON asks no one to escape, were corrupted identically. The decision is now the ordinal alone, asked only about the range `\u00xx` can spell, so no escape is wider than four hex digits and the same bytes come out under every locale.
- A guard file that will not load is an execution error rather than a policy failure. `verdict-guard.sh` is new in this release, so both of the states below were caught on the branch rather than shipped: one that would not parse exited **2** from every script, the status the contract reserves for "policy failure"; one missing entirely exited **1** from every script, the status that means "spec failure". Both reached `audit-report.sh` too, which documents only 0 and 3, so neither was a status it claims to return. Each script now checks the load on both sides, `bash -n` before and `declare -F` after, and exits 3 with no payload. `check-structure.sh` and `check-frontmatter.sh` wrote their SKILL.md-not-found diagnostic to stdout, which in `--json` mode is the payload channel; both write it to stderr now, matching the other two scripts. `audit-report.sh` exits 3 naming a missing `jq` instead of exiting 127 from the shell mid-report, with no partial report; `skill-validator` and `skillscore` stay soft dependencies there. Swept across every fixture, script, mode and masked tool, no invocation exits outside `{0, 1, 2, 3}`.
- The rule IDs the scripts emit are registered where the skill enumerates them. `DEP001`, `DEP002` and `PATH` are now in `skills/skill-audit/SKILL.md`'s rule-ID enumeration with both levels stated, in the header of every script that emits them, and in the README. `scripts/lib/`, the nested layout the guard was first written into and which no release ever carried, was flattened to `scripts/verdict-guard.sh` before it shipped, which removes the `deep nesting detected` warning it had earned the skill from `skill-validator`, and `tests/test_skill.sh` now asserts that both shipped skills validate with zero errors **and** zero warnings — nothing checked for a warning before, which is how that one rode through every gate unnoticed.
- `AdapterVersion` moved from `0.4.1` to `0.4.2`. It is recorded in every profile, and this release changed what a profile contains for the same export. The five manifests, the `tests/test_skill.sh` asserts and the README's two version references move with it.

**Documentation**

- `README.md` and `docs/profiler-spec.md` describe the merge the code now performs: a series is resource + scope + metric + attributes, a cumulative run is identified by `startTimeUnixNano` and holds the greatest running total its points reported, an unplaceable point joins the run it is cheapest to have come from, a `0` start time is an absent start time, and a mixed-temporality series is refused whole. The paragraph in both files naming the two known limits as deferred to a later release is deleted — this release is where they were fixed.
- `docs/profiler-spec.md` gains a **Deliberate deviations from the proto3 JSON mapping** note enumerating two deviations, stated so they are not mistaken for a conformance claim, and running in opposite directions. Narrower: a 64-bit integer field written in exponent form (`1.7893e18`) is refused, though the mapping accepts it. Reading it means going through a `float64`, whose spacing at 1.789e18 is 256ns, and `startTimeUnixNano` is a run identity — two points of one run that round apart would become two runs. Exact exponent parsing was considered and not taken: it needs decimal big-integer arithmetic in a hot leaf for a form `protojson` never emits and nothing observed produces. Wider: an enum written as a quoted decimal (`"aggregationTemporality": "2"`) is accepted, though the mapping reads a JSON string as an enum name and refuses `"2"` as the name of nothing. Nothing observed produces the quoted form and the reader's own comment puts it no higher than a *may*; it is accepted because refusing a temporality costs an entire series' merge, so all three spellings read. The count is an enumeration of what was measured, not a claim that nothing else deviates.
- What v0.4.1 actually did is now stated from measurement rather than from memory, in `README.md`, `docs/profiler-spec.md` and `profiler/testdata/otlp/README.md`. Two earlier claims were checked against the v0.4.1 binary and disproved: that a capture carrying no start times "merges exactly as it did before runs existed" (900 at t1001 then 500 at t1002 gave 500 at v0.4.1 and gives 900 here), and that "the last value written won" on a mixed-temporality series (it turned cumulative and the latest by `timeUnixNano` won).
- Two code comments that taught a model the code no longer has are corrected: the `array_attribute_series` reason in `profiler/profiler_test.go`, which explained a 200 by "the later point supersedes the earlier" — the right number from a deleted mechanism, and false as stated, since with the values reversed head reports 200 where "later supersedes" gives 100 — and `counterAccumulator`'s doc comment in `profiler/otlp.go`, which offered the type to a second OTLP-speaking adapter "unchanged" while neither refusing a negative point value nor handling one the same way on every branch: a cumulative point carrying one is discarded, and a delta point carrying one is added and takes the total down with it, so a delta series can total below zero. The guarantee lives in the caller, which refuses a value that is not a count before it arrives, and the comment now says so branch by branch.
- `profiler/testdata/otlp/README.md` rewrites the rows whose explanations described the superseded model, and documents the sixteen fixtures this release adds. The fixture set goes from 39 to 55.
- `README.md` rewrites its `jq` row — the row is not new; what it describes is — to say what `jq` is required for and when, corrects the `skill-validator` row's now-false clause, removes the sentence stating there is no guard for `jq`, and gains a `DEP002` paragraph.
- 66 profiler test functions pass — 61 in the profiler package, 5 in the CLI — and 230 subtests, 200 direct and 30 nested deeper, under `-race`. The shell suites are at **850**: `test_f01` 561, `test_f02` 243, `test_skill` 27, `test_walk` 19, against 0.4.1's 134. Both shipped skills validate with zero errors and zero warnings. All four suites were run under `LC_ALL=C` and under a UTF-8 locale, because one defect this release fixes was visible only under one of them.

**Known past changes, recorded late**

- 0.4.1's first Fixed entry says the adapter "accepts every 64-bit integer as a JSON number or a decimal string as the OTLP spec requires". The conformance clause is false and was false when written. The proto3 JSON mapping, which OTLP mandates, accepts exponent notation for 64-bit integer fields; this adapter refuses it. The behaviour is deliberate — reading an exponent form means going through a `float64`, which cannot represent every nanosecond instant, and two points of one run that round apart would become two runs — and it is recorded in this release under "Deliberate deviations from the proto3 JSON mapping" in `docs/profiler-spec.md`. What the adapter does is accept a 64-bit integer as a JSON number or a decimal string; what it does not do is conform to the mapping. The 0.4.1 entry is left as written; this is the correction.
- 0.4.1's Documentation section says `docs/profiler-spec.md` and `README.md` "state two known limits of the token series key". They no longer do: both limits are fixed in this release and the paragraph describing them is deleted from both files. The numbers in that entry are still the right numbers for v0.4.1 — two resources reporting 100 and 200 gave 200, and 100 followed by a restart at 20 gave 20 — and they are what this release changes. The 0.4.1 entry is left as written; this is the correction.
- `RELEASE_NOTES.md`'s v0.4.1 entry carries the same two claims and disagrees with itself about when they land: its heading says the limits are "both 0.4.2" and its body says they are "fixed together in 0.5.0". 0.4.2 is the right number and they are fixed here. "Both are described in the README and the spec" was true at v0.4.1 and is no longer, for the reason above. The v0.4.1 release note is left as written; the correction is in the v0.4.2 note.

**Known limits, carried to 0.5.0**

- **`isMonotonic` is unread.** A `claude_code.token.usage` sum declaring `isMonotonic: false` with a genuinely decreasing run reports the greatest value where v0.4.1 reported the latest — 1000 against 200 in the reproduced case. Judged safe for this metric and left deliberately: greatest-wins cannot over-report a monotonic cumulative sum, the OTel data model says a reader should expect non-decreasing values, every fixture in the repo declares `true`, and `claude_code.token.usage` is a Counter. It is the same class as `flags` and belongs beside it.
- **`flags` is unread**, so a stale non-zero `NO_RECORDED_VALUE` data point still reads as a running total. Fixing it is a new leaf on the data point, a new refusal class, its own reason clause, fixtures and a spec paragraph, and no capture route the README documents produces one today. It is the next one to take.
- **A `--json` exit 3 does not always carry a payload.** Three paths deliberately carry none: a usage error and a target that cannot be resolved, where there is no skill to render a verdict about, and a guard file that will not load, where there is nothing left to build a payload with. All three are enumerated in the header of each `--json` writer and pinned by tests; the README describes the guard-load one of the three. Minting rule IDs for them would enlarge the consumer-observable surface to describe caller errors rather than verdicts, so it was not done.
- **Schema v1 has no channel to say that a malformed series was excluded from a `present` total.** A `present` result carries no reason, so a series refused beside a healthy one reduces a total by a defect the profile has no field to name — the same gap as a skipped data point, and tracked with it. Inventing a channel inside v1 would mean a reason on a `present` result, which is a schema change wearing a bug fix's clothes. The caveat channel is the 0.5.0 item the spec already reserves.

## 0.4.1 — 2026-09-13

Makes 0.4.0's promises true. No new features: one family of profiler bugs — a
signal reported available on structural evidence rather than on a value read —
and documentation corrected to match the code.

**Fixed**

- The Claude Code adapter parses real OTLP/JSON. It read a bespoke `{"metrics": [...], "logs": [...]}` envelope carrying `token_type`, `cache_read`, `reasoning` and `decision: "approved"` — a shape nothing has ever emitted. Claude Code emits OTLP: `resourceMetrics[].scopeMetrics[].metrics[]` and `resourceLogs[].scopeLogs[].logRecords[]`, with `type` ∈ `input`/`output`/`cacheRead`/`cacheCreation` and `decision` ∈ `accept`/`reject`. A real export therefore produced an all-`unknown` profile while only a hand-written file produced values — the truth inverted. The adapter now reads one `Export*ServiceRequest` per JSON object (a single object, NDJSON, or concatenated objects), accepts every 64-bit integer as a JSON number or a decimal string as the OTLP spec requires, sums delta data points and supersedes cumulative ones per time series, and refuses a temporality that reads as neither rather than guessing a direction. Three further consequences: `ToolCallEntry.Success` now means the tool ran and succeeded, read from `claude_code.tool_result`, with rejected calls listed from `claude_code.tool_decision` as `success: false`; `TokenCounts.Reasoning` is no longer populated, because Claude Code has no reasoning token type and a zero there would claim a measurement nobody made; and `CaptureOpts.ExportFile` is refused by the adapter itself, not only by the CLI, so a library caller is told rather than handed an all-unknown profile. No profile schema keys changed, and `capability.adapter_version` already records this release. The bullets below stay as written — they are true as history of what the `token_type` and `decision` handling did, inside the envelope this release replaced.
- A profiler signal is `present` only when a value was actually read from the export. `Probe` scanned the export for structure while `Capture` extracted values from it — two predicates per signal, over two reads of the same file — and they disagreed in both directions. A `claude_code.token.usage` metric whose `token_type` the adapter does not recognise, or whose value is not a number, produced `present` with all-zero counts; a `tool_decision` with no `tool_name` and an `api_request` with no timestamp did the same. Both predicates are now one: the export is read and parsed once, each extractor is its signal's only predicate, and the capability report is derived from what those extractors returned, so probe and capture cannot disagree.
- Profiler capture no longer discards signals a partial OTel export does carry. `Capture` used the token capability as a proxy for "is any OTel data available", so an export with `claude_code.tool_decision` and `claude_code.api_request` events but no `claude_code.token.usage` metric was reported as "OTel export not configured" and its tool-call and timing data was dropped.
- An OTel export that was supplied but cannot be read or parsed is now `error` for all three signals, naming the read or parse failure. It previously reported "OTel export not configured", sending the caller to fix the one thing that was not wrong, because the probe's parse error was swallowed. `MetricError` was unemittable by this adapter — every branch that produced it was unreachable, and `ErrorToolCallResult` and `ErrorTimingResult` did not exist at all — and is now emitted and covered by tests.
- Session timing spans the earliest `claude_code.api_request` event to the latest, not the first in file order to the last. An out-of-order export produced a negative `total_ms`.
- `capture --export-file` now fails loudly instead of accepting a path nothing reads. No shipped adapter consults `CaptureOpts.ExportFile`, and the Claude Code adapter's fallback to it was unreachable, so the flag silently produced an all-unknown profile that looked like missing telemetry.
- Added `.claude-plugin/marketplace.json`, without which neither documented Claude Code install command could work: `claude plugin install` resolves a plugin name against configured marketplaces, not a repo path or `.`.
- `AdapterVersion` moved from `0.1.0` to `0.4.1`. It is recorded in every profile, and this release changed what a profile contains for the same export.
- Corrected the `profiler` module path from `github.com/auraprix/skill-architect/profiler` to `github.com/Okja-Engineering/skill-architect/profiler`, which is where the repo actually lives.
- CI now reads the Go toolchain version from `profiler/go.mod` instead of a hardcoded `1.23`, four minors behind the `1.27.1` the module requires, and disables `setup-go`'s cache, which warned on every run looking for a `go.sum` this dependency-free module does not have.
- Removed `CaptureOpts.OtelEndpoint`, which nothing read.
- A token count reaches the profile only if it is a count: a whole number from 0 to 2⁶³−1. The adapter carried a data point's `asDouble` through `int64(math.Round(f))`, which Go leaves to the architecture when the value is out of range — the same export read as `input: 9223372036854775807` on arm64 and `input: -9223372036854775808` on amd64. A negative value was assimilated verbatim as a token count, and `"NaN"`, `"Inf"` and `"Infinity"` all read as numbers, because `strconv.ParseFloat` accepts them from a quoted string. None of them reaches a profile now, and the two are counted and named apart by how a leaf *read*, not by how large the number is. A leaf that **decoded as a finite number** whose rounded value is negative or 2⁶³ or greater is refused by `count` as *not a count*. A leaf that **did not decode as a finite number at all** says nothing, so the next leaf is read: `NaN`, infinity and a literal that overflows its own reader are all refused by that reader, so `{"asDouble":"NaN","asInt":"9"}` is the count 9, while `{"asInt":"9223372036854775808"}` carries no readable value — 2⁶³ is finite and numeric, but `strconv.ParseInt` refuses it, so no leaf ever yielded a number to judge. A point is reported as carrying no readable value only when neither leaf yields one.
- Two time series that differ only in an attribute this adapter does not interpret are two series again. The series key rendered every attribute as bare text, so an `arrayValue`, a `kvlistValue`, a `bytesValue` and an absent value all keyed as the empty string — two series merged into one and a session's tokens were reported as a fraction of themselves. `{}` and `{"stringValue": ""}` collided, `intValue` `5` and `"5"` split one series in two, and an attribute value carrying the separator byte could forge another attribute set's key. The key is now a kind tag plus a canonical scalar, length-prefixed.
- A token count the export said nothing about has no key in the profile, instead of serialising as `0`. A cache-only export reported `input: 0, output: 0` beside its real `cache_read` — two measurements nobody made. A count read as zero keeps its key, with `0` in it. `TokenCounts` fields are pointers; the JSON key names are schema v1 and unchanged.
- Four reasons said something the export had not done. An export carrying `"resourceMetrics": []` — a session that emitted nothing — was told it is "not OTLP/JSON", sending its owner to fix an exporter protocol that was fine; it now reaches the extractors and each signal reports its own absence. A `sum` declaring no `aggregationTemporality` was told it "declared an aggregationTemporality that is neither 1 nor 2"; absent, unreadable and declared-but-neither are now three clauses. Six clauses said "1 events" and "1 data points". And byte offsets followed two conventions at once — a syntax error named the byte after the offending one, while a file that ended mid-object named the batch's first byte, so an 87-byte truncated capture reported byte 0. One convention now: the 0-based offset in the file of the first byte the decoder could not accept, or the file's length when the file ended.
- A capture is read to the end of the file or reported as a failure; it is never read as far as its first unreadable byte and then reported as a measurement. The reader asked the JSON decoder whether another element followed, and that question answers no for a stray `}` or `]` exactly as it does for the end of the file — so a capture carrying a doubled write, or a JSON array unwrapped by hand and left with its closing bracket, stopped at that byte and the batches before it were reported `present` as though they were the whole file. A stray `}` between two batches reported the first batch's 100 tokens and dropped the second batch's 500 without a word, which is the one failure a profile has no way to show its reader: a `present` result is supposed to be a complete measurement. Decoding now ends the walk. The end of the file ends it cleanly — whitespace and blank lines after the last batch included — and everything else is `error` naming the batch and the byte, with nothing from earlier batches used, which is what the spec said all along.
- `profiler capture` exits 2 when it read nothing and something failed, and 0 otherwise. It exited 0 for a missing or unreadable `--otel-file`, so a script wrapping it stored the all-unknown profile as a successful capture. The profile still goes to stdout in every case.
- CI runs `gofmt -l`, `go vet` and `go test -race`, not `go test` alone.

**Documentation**

- `README.md` documents the Claude Code install route that works — `claude plugin marketplace add` followed by `claude plugin install skill-architect@skill-architect`. Verified live against a clean install from a local clone, which is what was exercised; the same commands against the GitHub remote work once this release is on `main`, because that is where `marketplace add` looks for `.claude-plugin/marketplace.json`.
- `README.md` gains a Prerequisites block for `skill-validator`, `skillscore`, and `jq`, naming what fails without each. `skill-validator` and `skillscore` fail loudly; `jq` has no guard. Without `skill-validator`, `check-frontmatter.sh` run on its own skips spec validation silently; only its license gate still runs.
- `README.md` notes that `skill-rewrite` needs a sibling `skill-audit` directory — `draft-rewrite.sh` resolves the audit scripts at `../skill-audit` from the `skill-rewrite` directory, and copying `skill-rewrite` alone produces a draft with "No such file or directory" where the audit should be.
- `README.md` documents three things that shipped without mention: the `profiler version` subcommand and the `capture --export-file` flag, both in 0.4.0 (the flag is reserved for a future session-export adapter; passing it today is an error), and per-signal probe detection, which arrived after 0.4.0 in `30f374c` and was rewritten in this release. The README describes what the code does now, which is the only version of it that has been released.
- `README.md` privacy note: every metric data point and log record in a real capture carries `user.email`, `user.id`, `user.account_id`, `user.account_uuid`, `organization.id` and `session.id`. The capture recipe no longer sets `OTEL_LOG_TOOL_DETAILS=1` — this adapter reads nothing it adds, and it widens the export to what Anthropic's docs list for it: tool parameters and input, Bash commands, MCP server and tool names, skill names, and user-authored workflow names. It is named once, as a thing to leave unset.
- `README.md` marks the Devin, Codex and Cursor install rows "not verified in this release". Only the Claude Code route was run against a clean install; listing three unrun commands beside one verified one implied a check nobody made.
- `docs/profiler-spec.md` replaces the `MetricResult[T any]` generic, which was never implemented, with the real `RawMetricResult` plus the typed wrapper per metric category. The same name in the `profiler/types.go` comments is corrected too.
- `docs/profiler-spec.md` AC2 described the all-or-nothing probe that 30f374c replaced, contradicting the probe-logic section above it. Probe logic, capture logic, and the acceptance criteria now state one predicate, and AC9, AC10 and AC12 pin probe/capture agreement, "present means a value was read", and the error classification. AC11 and AC14 are new in round 2: one profile per export whatever the architecture, and the capture exit status.
- `docs/profiler-spec.md` fallback reason named `CLAUDE_CODE_ENABLE_TELEMETRY` and `OTEL_METRICS_EXPORTER`, which this adapter never reads, and claimed every metric carries it. `skill_activation` and `attribution` never do, and a supplied-but-unreadable export no longer does either.
- "Snapshot-pinned" overstated what the profiler does: `--snapshot` is a caller-supplied label copied into the profile verbatim, and nothing hashes or validates `--skill-dir` against it. Reworded across the README, release notes, spec, `Profile` doc comment, and the 0.4.0 entry below.
- 0.4.0's profiler test count is stated per commit: 14 at 0.4.0 itself, 15 after 30f374c. This release has **58** test functions (53 in the profiler package, 5 in the CLI) and **179** subtests (152 direct, 27 nested deeper). The shell suites are at **134**, three more than 0.4.0's 131: the two new marketplace-manifest assertions and the adapter-version one.
- `docs/profiler-spec.md` and `README.md` state two known limits of the token series key, both fixed together in 0.4.2 and both confined to cumulative temporality — Claude Code's default is delta (`aggregationTemporality: 1`, observed live), where neither arises. Resource and instrumentation-scope identity are not part of the series key: OTel identifies a series by resource + scope + metric + attributes, this adapter by the data-point attributes alone, so a capture aggregating several resources under cumulative temporality merges their series and undercounts — two resources reporting 100 and 200 give 200, not 300, where the same pair under delta correctly gives 300. And `startTimeUnixNano`, which is what marks a cumulative series' reset boundary, is not read, so a counter that restarts inside one capture discards the earlier run — 100 followed by a restart at 20 gives 20, not 120. Both are real and reproduced; they share one cause and are one change, which is why they are named here rather than half-fixed.
- 0.4.0 cited the design-space survey at `tmp/teams/architect/profiler-design-space.md`, a gitignored path no reader can open. Path reference removed.
- 0.4.0's "Claude Code has no skill-level events" was false, and round 3's correction of it did not go far enough: Claude Code emits a dedicated `claude_code.skill_activated` event — logged whenever a skill is invoked, through the Skill tool or a `/` command, carrying `skill.name`, `invocation_trigger`, `skill.source` and `skill.kind` — and attaches `skill.name` to `token.usage`, `cost.usage`, `api_request`, `api_error` and `api_refusal` besides. The `skill_activation` reason printed in every Claude Code profile still said the harness emitted no such event. It now says what is true: this adapter does not read that telemetry yet, and `claude_code.skill_activated` is the source it will read. The classification is unchanged — `skill_activation` stays `unknown` with capability `none`, because capability reports what this adapter can produce. A reason may say what this adapter does not read; it may not say what the harness does not emit unless that is true.

**Known past changes, recorded late**

- 0.3.1's entries describe the evaluation checklists, scoring dimensions table, and report format template as inline in `skill-audit/SKILL.md`. They are not: 3e2efd8 moved the checklists to `references/checklists.md` and pointed the scoring table and report template at `references/evaluation-matrix.md`, applying progressive disclosure. The 0.3.1 entries are left as written; this is the correction.
- `ToolCallEntry.Duration` (`duration_ms`) was removed from profile schema v1 in 30f374c because nothing populated it. That went unrecorded at the time. The schema string is unchanged at `skill-architect/profile/v1`; consumers should not expect a `duration_ms` key on tool-call entries.
- 0.3.1 recorded `skill-audit` at 92.5 (A-), which was true of the 0.3.1 release commit (`9897acc`). 3e2efd8 took it to **94 (A)**, and that is still the score at this release's head; `skill-rewrite` is unchanged at 89.5 (B+). The 0.3.1 entry is left as written; this is the update. (Measured with `skillscore --json` on all three commits.)

## 0.4.0 — 2026-09-10

Profiler preview: harness-agnostic runtime signal capture with graceful degradation.

- Added `profiler/` Go module — adapter interface (`ProfilerAdapter`), per-metric results (present/unknown/error), `CapabilityReport`, and a serialized `Profile` format carrying the caller-supplied snapshot id.
- Added Claude Code adapter (Slice 1) — reads OTel export data for token counts, tool calls, and timing. Skill activation and attribution are honestly `unknown` (the adapter does not yet read skill-level attributes).
- Added `profiler` CLI — `probe` reports capabilities, `capture` produces a profile JSON.
- Added `docs/profiler-spec.md` — the adapter interface contract, profile format, and acceptance criteria.
- 14 profiler tests pass at this release, 15 after 30f374c (serialization, capability reports, capture with/without OTel, round-trip, no-value-in-JSON for unknown/error states).
- All 131 existing tests still pass; no regressions.
- Design space survey of 4 harnesses (Cursor, Claude Code, Codex, Devin) informed the adapter-per-harness architecture.

## 0.3.1 — 2026-09-09

Dogfooding fixes: ran `skill-architect` against its own skills and addressed the findings.

- Added dependency-verification guards (`command -v skill-validator`, `command -v skillscore`) before script invocations in `skill-audit` SKILL.md — a missing dependency now fails loudly instead of silently.
- Added inline evaluation checklists (spec, trigger, body, determinism, ICM, validation, self-contained) directly in `skill-audit` SKILL.md body so the auditor is self-contained for criteria that scripts don't cover mechanically.
- Added scoring dimensions table with "Strong signal" column inline in `skill-audit` SKILL.md, keeping the `references/evaluation-matrix.md` reference for the tier model and output format.
- Added inline report format template to Stage 4 of `skill-audit` SKILL.md.
- Added Inputs/Outputs stage contracts to the Orchestration section of `skill-audit` SKILL.md.
- Fixed hardcoded `/tmp/release-check-audit.md` path in `skill-rewrite` SKILL.md example — replaced with a relative path.
- Bumped `skill-audit` metadata version to 0.2.0.
- Quality scores improved: `skill-audit` 89→92.5 (A-), `skill-rewrite` 86.5→89.5 (B+).
- All 131 tests pass; no spec regressions, no orphaned files.

## 0.3.0 — 2026-09-09

- Added `audit-report.sh` — produces a unified machine-readable JSON report composing `skill-validator` (spec, structure, content, contamination), `skillscore` (7-dimension quality scoring), and house-policy checks (PL001–PL005, PT001–PT002) into one document with a top-level summary.
- Added `--json` flag to `check-paths.sh` and `check-structure.sh` for structured output; text mode is unchanged.
- Documented `audit-report.sh` and `--json` flags in `skill-audit` SKILL.md.
- Added 58 tests covering the unified report shape, summary fields, all three sources, policy/path findings, spec failures, self-audit, and backward compatibility.
- Wired `test_f02.sh` into CI.

## 0.2.0 — 2026-09-07

- Added `skill-rewrite` skill for drafting rewrites from audit reports.
- Added `draft-rewrite.sh` helper that runs the audit and generates a `REWRITE-DRAFT.md` with templates for missing sections.
- Updated README and tests to cover both skills.

## 0.1.0 — 2026-09-07

**Initial release.**

- Added `skill-audit` skill for evaluating Agent Skills against the spec, Anthropic best practices, and ICM context-management criteria.
- Bundled `check-frontmatter.sh` and `check-structure.sh` deterministic checks.
- Added `references/best-practices.md` and `references/evaluation-matrix.md` for interpretation and scoring.
- Added plugin manifests for Devin, Claude Code, Cursor, and Codex.
- Added layout and script tests in `tests/`.
