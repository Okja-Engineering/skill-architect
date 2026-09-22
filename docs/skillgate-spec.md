# skillgate contract spec

Normative contract for the `skillgate` safety gate. This is the document PR
slices are validated against; when code and this file disagree, one of them is
wrong and the disagreement gets resolved before the slice lands.

Normative source today: `skillgate/` (as-built). Research rationale lives in
`docs/research/recommendation.md` and `docs/research/pi-integration.md` — those
are working notes and may fall away at release; this file must not.

## Claim boundary

- **F12 — never "safe" / "clean".** The only verdict strings are `APPROVE`,
  `CAUTION`, `REJECT`, always beside the coverage ledger.
- **F13 — CAUTION ceiling.** No verdict above `CAUTION` when the ledger is
  incomplete or any check was skipped. Every skip is named in
  `checks_skipped[]`.
- **F14 — read-path scope.** The gate audits before load and never contains
  what has loaded. `preactivation-bash-leg` is a standing entry in
  `checks_skipped[]` on every report — reading a skill through the shell
  instead of the harness read path is open on every examined harness and is
  never claimed closed. Consequence: **`CAUTION` is the reachable ceiling**;
  `APPROVE` is unreachable until a harness closes that leg.

## Audited unit and inputs

- The audited unit is the **package** — a skill directory or a plugin/package
  root — not only a `SKILL.md` dir. Manifest files found are recorded in
  `provenance.package_manifests`.
- Local directory: `provenance.path` only.
- Remote git URL: cloned into a fresh quarantine dir via
  `git -c core.hooksPath=/dev/null clone --depth 1`; `provenance` records
  `url`, `commit`, `quarantine`, and `sha256_manifest` (digest over sorted
  per-file hashes). Bundle code is never executed during gating.

## Coverage ledger

Every file gets a terminal outcome — `inspected` (with SHA-256) or `skipped`
with an allow-listed reason: `too_large`, `binary_unparsed`, `unreadable`,
`outside_root`, `symlink_escape`, `unsupported_type`, `file_count_limit`.
`coverage.complete` is true iff `skipped == 0`.

## Rule catalog

Deterministic, in Go, no external dependency. Severity is in the rule; the
block set is Pack-B-owned.

The Rule and Sev columns below are **checked against the gate**, not
maintained beside it: `skillgate.RuleCatalog()` is assembled from the
registries that run the rules, and `catalog_test.go` compares it to this table
in both directions — a rule with no row here, and a row here naming no rule,
each fail by name. What it catches is prose and is not checked; the id and the
severity are. Edit a row and run `go test ./skillgate`.

| Rule | Sev | What it catches |
|---|---|---|
| SK-T001 | blocker | Invisible / bidi / tag control chars in loaded text (hidden-instruction channel) |
| SK-T002 | blocker | Instruction-override phrasing in skill text |
| SK-T003 | blocker | Confusable non-Latin chars mixed into name/description |
| SK-T004 | blocker | Bundled script transmits to a literal remote host (loopback URLs masked — `localhost`/`127.*`/`::1` are dev plumbing) |
| SK-T005 | blocker | Wholesale env dump (`printenv`, `os.environ` enumeration, `process.env` spread) within 10 lines of a network/file sink — single-var reads, distant co-occurrence, and dumps bound to `env:`/env-named variables (child-env plumbing) don't count |
| SK-T006 | blocker | Credential-shaped literal in a bundled file |
| SK-T007 | blocker | Network output piped to a shell or interpreter |
| SK-T008 | blocker | Remote fetch/install without a pinned version |
| SK-T009 | high | Encoded payload feeding an interpreter — same line or decode within 10 lines of exec; `.exec()` regex methods and short ANSI/regex hex escapes don't count |
| SK-T010 | high | Touches another harness's config dir (.claude/.codex/.gemini/.continue) |
| SK-T011 | high | Reads Cursor config/state paths (.cursor/) |
| SK-T012 | blocker | References the Cursor credential store (state.vscdb / cursorAuth) |
| SK-T013 | blocker | `allowed-tools` wildcard, or skill ships executables *in its own directory tree* with none declared (scripts elsewhere in the package don't flag a doc-only skill) |
| SK-T014 | medium | `allowed-tools` declares far more tools than the bundle invokes |
| SK-T015 | blocker | MCP config carries plaintext secret or wildcard HTTP binding |
| SK-T016 | blocker | MCP tool auto-approved without consent |
| SK-T017 | blocker | Hook config (JSON only — `.claude`/`.codex`/`hooks/` dirs, incl. nested) **or package manifest** registers bundled executable content — incl. `pi.extensions`/`extensions` entries, npm lifecycle scripts, `bin` entries, and convention-discovered `.pi/extensions/` or top-level `extensions/` scripts |
| SK-T018 | blocker | Cursor hook config executes bundled content |
| SK-T019 | blocker | Path escape: reference resolves outside the bundle root |
| SK-T020 | blocker | Writes to persistence/self-modification surfaces — shell rc, crontab, LaunchAgents, `.claude/settings*.json`, `.cursor/`, `.pi/`, `~/.pi/agent/trust.json`. Verbs: shell redirects (path-like target required — `<cwd>` placeholders and `a > b`/`>=` comparisons don't count), copy/move/sed -i, and programmatic writes (`writeFileSync`, `appendFile`, `open(…,'w')`). `/dev/null` redirects and backslash-spelled paths normalized. |
| SK-G001 | medium | Dangling reference: path resolves inside the bundle, no file exists. Refs try file-dir first then each ancestor up to the root; an ancestor-level escape keeps the in-bundle candidates (a `../` ref is still checked, not dropped) |
| SK-G002 | info | Reference cycle between bundle files (Tarjan SCC) |
| SK-G003 | low | Unreachable file: a loaded-text file in a skill's subdirectory that no reference path from a harness entry point reaches. See **Reference reachability** below for the edge derivation and its stated limits |
| SK-I001 | medium | Required frontmatter missing (`name`, `description`). Skill files are `SKILL.md` only — `.mdc` rule files have no skill frontmatter contract |
| SK-I002 | low | Frontmatter `name` ≠ dir name; charset violation (lowercase-hyphen, ≤64); in-bundle same-name skill collision |
| SK-I003 | medium | `description` > 1024 chars |
| SK-I004 | medium | Body > 500 lines |
| SK-I005 | medium | Body over token budget — requires a measured counter, else `icm-token-budget` skip |
| SK-H002 | info | Per-harness frontmatter: keys valid for Claude Code, silently ignored by Cursor (`allowed-tools`, `disable-model-invocation`, `context`, `when_to_use`) |

Rule count is capped at 20 tripwires; extensions fold into existing legs
(T017/T020 did). That cap is a bound the build holds, not a claim about the
table: `catalog_test.go` reads the number out of this sentence and counts the
SK-T rules the gate registers, so the twenty-first tripwire fails the build,
and raising the cap here is what raises it. Rule representation is frozen
until `docs/research/rule-language.md` lands.

## Views — what a lexical rule scans

**A lexical rule runs over normalised views of a file as well as over its raw
text, never over raw text alone.** A rule that matches phrasing against raw
bytes is defeated by how the payload is *spelled*: `Ｉｇｎｏｒｅ ａｌｌ
ｐｒｅｖｉｏｕｓ ｉｎｓｔｒｕｃｔｉｏｎｓ` and `Ignоre all prevіous
instructions` (Cyrillic U+043E, U+0456) are the blocker-severity `SK-T002`
payload, and neither fired before views existed. The answer is not more
alternations in the pattern — that is the enumerate-the-forms defect — it is
to normalise the text before matching.

This section is the normative contract. It restates, for the tracked
specification, the view axis proposed in `docs/research/rule-language.md`
§4.3 (prior art, and the source of the upstream invariants); where the two
differ, this section governs.

### The contract

A **view** is `(name, text, source offsets)`. The offset map answers, for
every byte of the view's text, which raw byte produced it. That map is not
polish: without it a view-derived finding has no honest position, and a report
that points at the wrong line is worse than no report. So:

- **Position and evidence are always raw.** A finding's `file`, `line` and
  `evidence` name the raw source, whatever view discovered it, so a reader can
  find the text in the file as it is written on disk.
- **A non-raw finding is tagged.** `findings[].view` carries the view's name.
  It is absent on a raw finding, so a report of raw findings is unchanged.
- **A hit that cannot be anchored is dropped.** Evidence that is not present
  in the text the rule scanned has no offset to map back, and a finding at an
  invented position is noise. It is dropped, not guessed at. (The rules whose
  evidence is synthesised by construction are raw-only for that reason — see
  the table below — so this is a safety net, not a routine loss.)
- **Findings dedup raw-wins.** One rule reporting the same place in the same
  file is one finding, preferring the raw discovery. An engine that reported
  each raw hit once per view would double every existing finding.
- **Views are content, not configuration.** They are built once with the
  coverage ledger and are the same for every run over the same bytes.

### Registered views

| View | What it renders |
|---|---|
| raw | The file's bytes, unchanged. Always present, always scanned first. |
| skeleton | Per-character NFKC, then the UTS #39 ASCII confusable skeleton, then removal of `Cf`, the non-whitespace `Cc` and `Other_Default_Ignorable_Code_Point`. Three Unicode classes and one generated table — not a list of spellings. NFKC is applied per character so one character maps to one span, which is what makes the offset map exact; the cost is that a combining sequence spelled base + mark is not composed. |

`skillgate.ViewNames()` returns this list from the registry that builds them,
and `view_test.go` compares the two: a view with no row here, and a row here
naming no view, each fail by name.

### Rules that run on raw text only

Every rule that scans text runs on every view **unless it declares a reason
not to**, so a view added later covers every rule with no edit at the rule,
and a rule added later is view-covered by construction. The opt-outs are
derived from that declaration, not maintained here:
`skillgate.ViewCoverage()` reads the reason off the rule, and `view_test.go`
compares the set to this table in both directions.

| Rule | Why it is raw-only |
|---|---|
| SK-T001 | Its subject *is* the invisible control characters a normalised view removes. The skeleton view strips exactly this rule's set, so a derived view can never contain what it looks for. |
| SK-T003 | Its subject *is* the confusable code points the skeleton folds onto ASCII. The fold is what makes the text unmixed, so the mixed-script condition cannot survive into a derived view. |
| SK-T005 | Its evidence is a synthesised pair of lines (the dump line and the sink line), not a substring of the text it scanned, so a derived hit has no offset to map back to raw. |
| SK-T006 | Its evidence is masked before it leaves the rule — the gate never republishes a secret — so it is never a substring of the text it scanned. |
| SK-T009 | Its evidence is a synthesised decode-line / exec-line pair, not a substring of the text it scanned. |
| SK-T019 | Its subject is path *syntax*, and a normalised view manufactures path syntax out of prose: NFKC maps typographic punctuation onto ASCII, so U+2025 TWO DOT LEADER becomes `..` and a bare `‥` in an ordinary sentence fires this blocker. Erring towards a missed escape rather than a false blocker is the direction SK-G003 already ruled for. |

### Rules that read executed code only

A view answers *in what spelling*; a region answers *in what part*. Every rule
that scans text reads the whole document **unless it declares a reason not
to**, exactly as with the view axis, and the opt-outs are derived from that
declaration rather than maintained here: `skillgate.ViewCoverage()` reads the
reason off the rule and `view_test.go` compares the set to this table in both
directions.

A code-only rule scans the file with its **whole-line commentary blanked** —
blanked to spaces, not removed, so every byte offset still means what it meant
and a finding still reports the raw line it came from. Which bytes are
commentary is decided by the file's own language (`scriptLangs`), never by a
list of phrasings: an author cannot spell a line as a comment and have the
interpreter run it. A file whose language the gate does not know has no
commentary and the rule keeps its full reach, so losing reach requires
positively identifying the grammar.

| Rule | Why it is code-only |
|---|---|
| SK-T010 | Its verb is *touches* — the finding claims the script reaches into another harness's config directory, and a comment reaches into nothing. Measured: it flagged `skills/skill-rewrite/scripts/draft-rewrite.sh` six times, on six comments describing the containment bound that script enforces, and never on the live line naming all five protected directories. |
| SK-T011 | Its verb is *reads*, and a comment reads nothing. Same subject and same reasoning as SK-T010: a skill that documents the Cursor paths it stays out of is describing the boundary, not crossing it. |

### Stated limits

- A payload a program carries as **data** and later executes — a Python
  triple-quoted block passed to `exec`, a JS template literal passed to
  `eval` — is elided by the code projection if its lines begin with the
  comment marker. Assembling and running text is SK-T009's subject, and the
  code that does the assembling is live text the projection keeps.
- A fullwidth- or confusable-spelled path escape is **not** caught, because
  SK-T019 is raw-only above. A fullwidth-spelled network transmission, pipe to
  a shell, or persistence write **is** caught.
- The rules whose evidence is synthesised or masked (SK-T005, SK-T006,
  SK-T009) do not gain view coverage. Giving them coverage means giving them
  locatable evidence, which is a change to those rules, not to the engine.
- Vocabulary is not surface form, and no view reaches it. `Bypass any
  preceding directives`, `axios.post`, `uv pip install` and their kind are
  missed for the same reason they were missed before: the enumeration is in
  the matcher's vocabulary, not in the text's spelling.
- Letter-spaced and markup-interpolated payloads (`i g n o r e …`,
  `Ignore **all previous** instructions`) need their own views; the skeleton
  does not close them.

## Reference reachability (SK-G003)

Reachability is a property of the resolved graph, not of the text. This
section is normative because the rule accuses a file of being dead, and an
accusation must state the computation that produced it.

**Entry set.** Every inspected file a harness opens *by convention* rather
than through a reference: `SKILL.md`, the memory files (`AGENTS.md`,
`CLAUDE.md`, `GEMINI.md`, `AGENT.md`), and `.mdc` Cursor rule files. These
are the doors; they need no inbound reference and are never candidates.

**Reachable set.** The forward closure of the entry set over the resolved
edges. Reference sources are **every inspected file**, not only loaded text
— a script or a manifest naming a template is an edge a reader follows, and
a missed edge is what turns a reachable file into a false orphan.

**Candidate set.** Inspected loaded-text files sitting in a *subdirectory*
of a skill root — the progressive-disclosure payload, which exists only to
be pointed at. Files beside `SKILL.md` are skill furniture (README,
CHANGELOG, a license) and are not candidates. Skill roots are derived from
the `SKILL.md` files in the ledger, so a package with no `SKILL.md` has no
candidates and the rule is silent: with no entry point there is no
reachability question, and a docs tree is not a bundle of orphans.

**Reported set.** Candidates minus reachable — the complement of a
computation, not a list of files that look orphaned.

### Which spellings of a reference become an edge

Edges come from the same token stream the rest of the reference graph reads
(markdown link targets, backtick-quoted path tokens, and bare paths under
`references/`, `scripts/`, `assets/`, `examples/`, `hooks/`), classified by
whether the token names a file or a directory:

- **File reference** — a token ending in a known source/document extension
  that resolves to a ledger file: one edge to that file. Bare filenames
  count only from an explicit markdown link target.
- **Directory reference** — a path-shaped token that names no file and
  resolves to a ledger directory: an edge to **every** ledger file beneath
  it, at any depth. Naming a directory discloses what is in it; only
  reachability reads these, never SK-G001.

Everything else produces **no edge**: URLs, anchors, `~/` home-relative
paths, `$VAR`/`<placeholder>` forms, command strings, and bare prose
mentions of a path outside the five conventional directories.

### The three rulings, and the direction the rule errs

- **A cycle is not reachability.** An island of files referencing only each
  other is reached by nobody, so every member is reported. A cycle hanging
  off an entry point is reached through its entry edge and is silent — that
  one is SK-G002's to mention.
- **An edge may leave the referencing skill's directory.** The audited unit
  is the package, so a file another skill reaches is reachable. An edge
  whose target leaves the package *root* resolves to no ledger file and
  contributes no edge at all; that reference is SK-T019's.
- **A missed edge is the dangerous direction**, because it accuses a file
  that is in fact reached in a spelling the resolver could not follow. So a
  candidate whose base name is named in a file that produced no edge to it
  is **not** reported: the resolver's blind spot is not the skill's defect.

### Stated limits — where SK-G003 is silent by construction

These are gaps, accepted deliberately in exchange for not accusing correct
skills. Each is asserted by a test in `skillgate/refgraph_test.go`, not
promised here.

1. **A directory reference blinds the directory.** A layout line naming
   `references/` is indistinguishable from "use the templates in
   `assets/contracts/`", so a bundle that names its payload directory as a
   whole can never produce an SK-G003 for a file inside it. Measured: this
   silences the rule for
   `skills/skill-audit/references/` (`SKILL.md:152` names `references/`,
   `scripts/` and `assets/` bare).
2. **Scripts and other non-loaded-text files are not candidates.** The
   resolver reads references out of text and cannot see a script's own
   imports, a hook config's command target, or a package manifest's `bin`
   entry, so an unreferenced script is never reported as unreachable. Those
   registration surfaces are SK-T013's and SK-T017's.
3. **A base-name mention anywhere spares the file**, including a mention
   that has nothing to do with it. A genuine orphan whose file name happens
   to appear in unrelated prose goes unreported.

## checks_skipped contract

Every stage or scanner that did not run is a named entry:
`skillspector`, `agnix`, `skill-validator`, `icm`, `icm-token-budget`,
`preactivation-bash-leg`, or `<name>` with reason `skipped by option`
(`--skip-checks`). Absent optional tools are named skips, never silent, and
never load-bearing for `REJECT`.

## Verdict and exit codes

- `REJECT` — any unsuppressed, non-advisory blocker/high finding; or
  incomplete ledger on untrusted input.
- `CAUTION` — any other unsuppressed finding, any skipped check, or an
  incomplete ledger on local input.
- `APPROVE` — clean, complete, zero skipped checks. Unreachable while F14's
  standing skip exists (see claim boundary).

Exit codes: `0` = APPROVE/CAUTION · `1` = REJECT, or incomplete under
`--fail-on-incomplete` (the F14 standing skip is a claim boundary, not
coverage, and does not trigger it) · `2` = the gate itself failed.

## Baselines

Suppressions are content-bound: `fingerprint` + mandatory `reason`. Drifted
content fails closed — a stale fingerprint un-suppresses the finding.

## Advisory integrations

SkillSpector, agnix, and skill-validator shell out when present; findings
carry `source` and `advisory: true`. They report at true severity but can
never force `REJECT` — the block set has no external dependency (D2).

## Pack E — always-on token budget

`report.tokens`: exact per-file counts, `counter` (e.g.
`skill-validator/o200k_base`) and `basis` (`measured` | `estimated` |
`inferred` | `billed`) named. `tokens.items[]` attributes content injected
outside the file list — classes `skill_description`, `tool_description`,
`tool_input_schema` — with exact chars; tokens only when a real tokenizer
covered the item, never chars/4. No figure derives from cache-miss or
prefix-invalidation signals.

## Report schema

`skillgate/report/v1`: `schema`, `generated_at`, `tool{name,version}`,
`target`, `provenance`, `verdict`, `findings[]`, `ledger[]`, `coverage`,
`checks_skipped[]`, `refusals[]`, `tokens`. SARIF 2.1.0 via `--format sarif`
(blocking → `error`, advisory/suppressed → `warning`, info → `note`).

A finding carries `view` when it was discovered on a normalised rendering of
the file rather than on its raw text — see **Views**. The field is absent on a
raw finding, so a consumer written before views existed reads an unchanged
report; `file`, `line` and `evidence` are the raw source either way.

CLI: `skillgate gate <dir-or-url> [--baseline f] [--fail-on-incomplete]
[--format json|sarif] [-o file] [--skip-checks a,b]`. A flag may appear
before or after the target and the report is identical either way; `--` ends
flag parsing, so a target whose name begins with `-` is written
`gate -- -target`. Exactly one target is required, an undefined flag is
refused wherever it appears, and a flag whose value is missing is refused
rather than read from the arguments beside it.
