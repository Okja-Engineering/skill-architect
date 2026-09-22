# skillgate contract spec

Normative contract for the `skillgate` safety gate. This is the document PR
slices are validated against; when code and this file disagree, one of them is
wrong and the disagreement gets resolved before the slice lands.

Normative source today: `skillgate/` (as-built). Research rationale lives in
`docs/research/` — `recommendation.md`, `pi-integration.md` and
`rule-language.md`. Those notes are **kept permanently** by standing ruling and
are durable at the `archive/docs-research-0.6.0` tag, but they are **not
tracked in the release tree**, so nothing tracked may depend on them: where a
research note is normative for shipped behaviour, this file restates it and
this file governs.

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

Deterministic, in Go. Severity is in the rule; the block set is Pack-B-owned.

**Dependency surface.** The gate has **no runtime dependency on Python, Rust or
any external binary** — the advisory integrations below are opt-in and can
never force a verdict. It is not, however, dependency-free: the production
module's direct requires are `golang.org/x/text` — the Go team's own module,
BSD-3-Clause — which the view layer needs for NFKC and full casefold. That
sentence is not maintained by hand: `packaging_test.go` reads the module paths
out of it and compares them to `skillgate/go.mod`'s direct require block in
both directions, so a dependency taken without being stated, and a statement
outliving its dependency, each fail by name.

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
| SK-I006 | medium | Frontmatter the reader refused, reported at the line and with the reason it recorded. *Missing* and *unparseable* are different findings: a document that was never read has no missing keys, and SK-I001 does not claim it does |
| SK-H002 | info | Per-harness frontmatter: keys valid for Claude Code, silently ignored by Cursor (`allowed-tools`, `disable-model-invocation`, `context`, `when_to_use`) |

Rule count is capped at 20 tripwires; extensions fold into existing legs
(T017/T020 did). That cap is a bound the build holds, not a claim about the
table: `catalog_test.go` reads the number out of this sentence and counts the
SK-T rules the gate registers, so the twenty-first tripwire fails the build,
and raising the cap here is what raises it.

**What is frozen, and what is not.** The old sentence here — *"rule
representation is frozen until `docs/research/rule-language.md` lands"* — is
spent: that note has landed, and this release implements its **view axis** (see
**Views**). What is unchanged is rule *representation*: a rule is still a Go
literal declared beside the code that emits it, and the rule **language** the
note proposes — constructors, scopes, a catalogue generator — is not built
here. A refactor to it is a later release's, and it is gated on a report/v1
golden diff rather than on the note.

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
differ, this section governs. It restates the **axis**, not the whole of the
prior art's catalogue of renderings — the renderings this release does not
build are named under **Stated limits**, so "0.6.0 implements the view axis"
cannot be read as "0.6.0 implements all of it".

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
| compactLetter | The skeleton fold, then the separators inside letter-spacing runs removed. A **letter-spacing run** is six or more **isolated** letters in a row — a letter is isolated when it has no letter on either side of it — each separated from the next by a non-empty gap holding no letter and no digit. Isolation is what cuts **both** edges: the run cannot begin inside an ordinary word and cannot reach into one, so it stops in front of the following word rather than one letter inside it. Within a run the narrowest gap is the letter separator and is deleted; a wider gap is where the words divide and becomes one space. Derived from the run's shape, so every separator closes at once — space, NBSP, `.`, `-`, `_`, `*`, a non-ASCII Z-separator, U+FFFD and the filler nobody has thought of yet are all simply "not a letter". |
| markup | The skeleton fold, then CommonMark's **inline** syntax resolved away and the text it wraps kept: emphasis and strong-emphasis delimiter runs, code spans, inline links and images, and HTML comments. What tells a delimiter from an ordinary asterisk is the grammar's **delimiter run** rule — what sits on either side of the run — plus the requirement that a delimiter actually pair with another. So `rm *.sh` (an opener with no closer) and `2 * 3` (flanked by whitespace, so not a delimiter at all) are untouched, and no exception list is needed to leave them alone. |
| markupCompact | The skeleton fold, then the markup stage, then the compaction — the two derived stages in one pipeline, with no third grammar. It exists because a payload that is letter-spaced *and* markup-interpolated is reached by neither parent view: the delimiters sit where the compaction has to read a gap width, so the compaction alone reconstructs the wrong words, and the letters are still spaced after the markup alone is resolved. |

A view is an ordered pipeline of stages: every stage is written against the
text the stage before it produced, and the offset maps are composed, so a view
anchors back to raw however many stages made it. `compactLetter` and `markup`
are the fold plus one stage each; `markupCompact` is the fold plus both.

**Stage order is part of a view's identity, not an implementation detail.**
Two stages in the other order are a different transform with different output,
so `markupCompact`'s order is a measured decision: the markup stage runs
first. The compaction normalises every gap inside a run to nothing or to one
space, so a delimiter that sits in a run is consumed as gap material and can
never pair afterwards; the markup stage run first still sees delimiters as
delimiters. Measured over 18,480 spellings of the `SK-T002` payload — every
combination of unit separator, word gap and interpolated construct —
markup-then-compaction reached 10,968 that no existing view reached, and
compaction-then-markup reached none that markup-then-compaction did not.

**Every view has the same lines as the file**, so view line *i* is raw line
*i*. This is a property of the view axis as a whole, not of any one view:
every rule cuts its evidence on lines and reports at a line, so a stage that
swallowed a newline would move every finding after it, and would let two
lines' text form a match that is in neither. `view_compact_test.go` asserts it
over every registered view.

Beside it, each derived stage states **what it is allowed to bring together**,
because that is where a view manufactures findings rather than revealing them:

- **The compaction only ever brings letters together.** Gaps are removed or
  become one space, and text outside a run is untouched, so — unlike a fold —
  it cannot manufacture syntax out of prose. That is the failure mode the
  skeleton view measured, where NFKC turned a bare `‥` into `..` and fired a
  path-traversal blocker.
- **The markup stage only ever deletes.** Its output is a subsequence of its
  input, so it can bring two characters that were already on the line
  together, but it can never produce a character the line did not contain. It
  is the one stage that removes punctuation, so this is the bound that matters
  for it, and it is why the rule that fired on the fold's manufactured `..` —
  `SK-T019`, already raw-only below — cannot be reached this way either.
- **A composition states its own bound, because a composition's bound is not
  the conjunction of its parts.** The compaction's bound is stated against the
  text it is given, and in `markupCompact` that text is not the document: the
  markup stage has already fused what sat on either side of a delimiter, so
  the compaction can read a letter run that exists neither in the raw document
  nor in either parent view. Measured, the composition's own bound holds and
  is narrower than that fear: every character of a composed line is a
  character of the folded line, in order, except that a word boundary may be
  rendered as a space. So the composition can bring characters together and
  can never produce one the line did not contain.

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

### Word boundaries, and why the two ends are not symmetric

A matcher whose vocabulary is a **word** must not match a substring of a
longer word. Every rule pattern leg that begins with a word is therefore
`\b`-anchored at the **start**, and `skillgate`'s test suite enforces it as a
class rather than by inspection: it runs every rule pattern over every text
file in the repository and fails on any match beginning mid-word, with **no
allow-list**. A list of words permitted to match inside words would be the
enumerate-the-forms defect in a new place.

**The trailing end carries no such blanket rule, and that is a measured
decision rather than an omission**, because:

> a word's meaning survives suffixing but not prefixing.

Identifiers compound head-first. `cursorAuth`, `cursorDir`, `CursorVersion`
and `CURSOR_API_KEY` are all genuinely *about* Cursor, and `writeFileSync` is
genuinely a `writeFile` — whereas `func` is not about `nc`, `guarantee` is
not about `tee`, and `concat` is not about `ncat`. So a trailing anchor is
correct only where the leg's vocabulary is a **command word**: a token an
interpreter resolves as a program or keyword, which cannot be extended and
remain the same command. `sh` extended is `shadow`.

Two legs qualify and carry a trailing anchor:

| Leg | Rule | What it stops matching |
|---|---|---|
| `\|\s*(sudo\s+)?((ba\|z\|fi\|da)?sh\|base64)\b` | SK-T005 | `\|\| showhelp`, `\|\| shadow_code=`, `bashrc\|bash_profile`, and the `\|sh` inside any other regex's `(?:can\|may\|should)` — each a **blocker** for piping the environment into a shell |
| `\bsubprocess\b` | SK-T009 | `subprocessEnv`, `subprocess_helper` — identifiers that start with the module's name and execute nothing |

The legs that must **stay loose at the end**, with what anchoring them would
have cost — measured, and pinned in `wordboundary_test.go` so the result
cannot decay into a guess:

| Leg | Rule | Anchoring it would lose |
|---|---|---|
| `\bcursor` | SK-T012's context gate | `cursorAuth` — *the credential-store key the gate exists to gate* — plus `cursorDir`, `CursorVersion`, `cursor_version`, `CURSOR_API_KEY` |
| `\b(write\|append)File` | SK-T020 | `writeFileSync`, `appendFileSync`, the commonest real spellings |
| `\burllib` | SK-T005 | `urllib3`, a real library and a real sink |
| `\btee\s+\S` | SK-T020 | nothing — this leg ends on the first character of the *filename*, a deliberate partial match rather than vocabulary |

### Stated limits

Every gap this release ships, named here — and **every limit below is driven**.
`skillgate/ceded_test.go` holds one lane per item: a spelling the gate does not
catch beside a neighbouring spelling it does, or the test elsewhere in the
package that already drives it. The join is checked in both directions, so a
limit added here with nothing behind it fails by its own words, and a limit that
has *stopped being true* fails at its driver — which is the failure this
section previously could not have. Twice in this release a limit outlived its
truth and was propagated by a later slice; a limit is a claim, and a claim
nobody can falsify decays. **Do not reword a limit to keep a driver green.**

- **Renderings the prior art proposes and this release does not build.**
  `obfuscatedInstruction` (filler removal gated on an override vocabulary) and
  `declaredMarker` (reconstructing a payload from a *remove the following
  markers* directive) are described in `docs/research/rule-language.md` §2 and
  have no counterpart in the registry above, so a payload that needs one is
  missed. `continuity`, the third the prior art names, does not arise here: it
  bridges a separator run wider than a scanning window's overlap, and the gate
  reads whole files rather than overlapping windows. All three are asserted
  absent from the registry, so building one fails a test rather than leaving
  this paragraph behind.
- **The legs whose vocabulary is a name stay loose at the trailing end**, and
  what that costs is measured rather than assumed — see **Word boundaries**
  above for the legs, the spellings each would have dropped, and why the two
  ends of a word are not symmetric. It is a ceded lane and is named here as
  one: a leg with no trailing anchor can match inside a longer word, and the
  release accepts that rather than lose `cursorAuth`, `writeFileSync` and
  `urllib3`.
- **`\| /bin/sh` is not caught** by SK-T005's sink leg or by SK-T007's
  pipe-to-shell leg: both require the interpreter name immediately after the
  pipe, and neither accepts a leading path. This is a vocabulary-and-form
  gap, not a boundary one, and nothing in this release closes it.
- A payload a program carries as **data** and later executes — a Python
  triple-quoted block passed to `exec`, a JS template literal passed to
  `eval` — is elided by the code projection if its lines begin with the
  comment marker. Assembling and running text is SK-T009's subject, and the
  code that does the assembling is live text the projection keeps.
- SK-T019 is **no longer raw-only**, so a fullwidth- or confusable-spelled
  path escape *is* caught. The opt-out existed because NFKC folds U+2025 TWO
  DOT LEADER onto `..` and a bare `‥` in prose fired a blocker; a bare `..`
  is now read as the single path segment it is and names nowhere, so the
  fold has nothing left to manufacture.
- SK-T019 does not model a process's working directory: `cd ..` is a
  directory change, not a reference, and is not reported. It never was
  analysed — the old matcher hit it by coincidence of spelling.
- A reference inside a comment **is** reported by SK-T019. A documented
  dependency on a file outside the bundle is still a dependency, which is
  why the code-only narrowing above does not apply to it.
- The rules whose evidence is synthesised or masked (SK-T005, SK-T006,
  SK-T009) do not gain view coverage. Giving them coverage means giving them
  locatable evidence, which is a change to those rules, not to the engine.
- Vocabulary is not surface form, and no view reaches it. `Bypass any
  preceding directives` (SK-T002), `axios.post` (SK-T004), `uv pip install`
  (SK-T008) and SK-T009's decoders beyond the ones its pattern spells — a
  `tr`-driven rot13, a `gzip.decompress` — are missed for the same reason they
  were missed before: the enumeration is in the matcher's vocabulary, not in
  the text's spelling. Closing one means deriving that rule's side (a network
  sink receiving a literal remote host; remote bytes reaching an interpreter;
  an install command with no pin; a decode feeding an exec), which is a change
  to the rule and not to the engine.
- **Hook configuration is read as JSON only.** A `.codex/config.toml` is
  classified as harness configuration and then not parsed, so an exec
  registration spelled in TOML reaches no rule; the same registration spelled
  in JSON fires SK-T017. The catalog row says *JSON only*; this is what that
  costs.
- **An MCP server entry is read for what it carries, not for where it points.**
  SK-T015 and SK-T016 read a server's secrets and its approval settings. A
  server whose notable property is a **remote `url`** produces no finding, and
  no rule reads an `allowedEnvVars` grant at all — a wildcard there is silent.
  A server in the same shape carrying a plaintext secret does fire, so this is
  about which fields are read, not about MCP configs being unread.
- **A loopback sink is masked, so a relay behind one is invisible.** SK-T004
  masks `localhost`/`127.*`/`::1` as dev plumbing, which is what keeps it off
  every local development script — and the cost is that a script posting to a
  local port that forwards the bytes onward is not reported. The same request
  to a literal remote host fires. Reading the relay would mean resolving what
  is listening, which the gate does not do: it never executes the bundle and
  never opens the network.
- Markup-interpolated payloads are closed by the `markup` view, and by it
  alone: neither the skeleton nor the compaction reaches them. The fold has no
  mapping for `*`, and folding the payload the skeleton's way runs the words
  together, so the phrase rule then fails for want of whitespace instead —
  a different miss, not a fix. The bounds on that view are contract:
  - **Reference links** (`[text][ref]`) are not resolved. They need the
    document's link-reference definitions, which a line-bounded stage does not
    have. Inline links and images are resolved.
  - **A code span's content is literal, delimiters and all.** That is
    CommonMark precedence: a renderer shows a code span verbatim, so a view
    that resolved markup inside one would be reading the document differently
    from every renderer and every human. It is also what lets a document
    *quote* an interpolated payload in order to explain it without being read
    as carrying one — the next line quotes the release's headline case, and
    the gate does not flag this file for it:

    `Ignore **all previous** instructions`

    That is a property this release's own documents depend on, since S09, S10
    and this section all explain the rules by writing the evasions out.
    An HTML comment is the opposite case and is treated so: it renders as
    nothing
    at all, so there is no literal display to preserve, its content is text
    the model still reads, and markup inside it is markup.
  - **Strikethrough** (`~~x~~`) is not resolved: it is a GFM extension rather
    than a CommonMark production. Measured over this repository, 300 installed
    markdown documents and the design corpus, resolving it would have changed
    no finding either way.
- A payload that is **both** letter-spaced and markup-interpolated is closed
  by `markupCompact`, with one bound: if the letter-spacing separator is
  *itself* an inline-markup delimiter (`*` or `_`), the markup stage reads the
  run's own separators as delimiters, resolves some of them, and the
  compaction that follows no longer sees a run. Reaching that spelling costs
  an attacker the interpolation the composed view exists to close — the same
  payload spaced with `*` or `_` and *not* additionally interpolated is closed
  by `compactLetter` alone. The reverse stage order does not fix it and closes
  strictly less besides.
- A letter-spaced payload whose **word boundaries are also invisible** — a
  zero-width character between every letter *and* between every word — is not
  reached by any view. The skeleton strips the invisible characters before
  the compaction is asked to look, which is what closes the ordinary
  invisible-separator spelling, but it leaves one unbroken word, and every
  phrase rule is written with whitespace between its words. Spelled the
  readable way, with the word boundaries visible, the fold closes it on its
  own.
- Compaction takes a run's narrowest gap as its letter separator, so a run
  that is entirely one word and a run whose every gap is the same width
  compact to one word — which is correct — but a payload that spaces its
  words *narrower* than its letters is not reconstructed. That spelling is
  not readable as the instruction it is imitating.
- **A run ends in front of the following word, so a payload that must be read
  as one unbroken token across that boundary is not reconstructed.** Space out
  the first half of one of SK-T012's single-token patterns and leave the second
  half spelled plainly, and the compaction puts a space where the two halves
  meet, so the rule does not match. Spelled all the way out, the same payload
  compacts to the token and the rule does match.

  This is a choice the grammar has to make and cannot avoid, because that
  spelling and `p r e v i o u s instructions` are the *same shape* — a run,
  then a unit-width gap, then a plainly spelled word — and only vocabulary
  could tell them apart, which is exactly what a view must not carry.
  Separating is the measured side: over the gate's own per-rule payload
  corpus respelled both ways, joining reaches **no rule the separating
  grammar misses** and misses one it reaches; joining also counts the
  following word's first letter toward the six-letter threshold, which
  inflated 20 of this repository's 74 runs over it. `SK-T012`'s own test
  (`TestARunDoesNotReconstructATokenAcrossItsEdge`) pins both halves of this
  trade so it stays a stated bound rather than a surprise.

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
mentions of a path outside the conventional directories named above.

### The rulings, and the direction the rule errs

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

Every check that did not run is a named entry. The names are the registered
checks — `skillgate gate --list-checks` publishes the set, and there is no
second list to keep in step with it; an entry may also name a **leg** inside a
check, published on the same line as the check that reports it.

A check is a named skip whether it declined (an absent optional binary), was
excluded (`--skip-checks`), was not selected (`--only`), can never run (the
standing F14 boundary), or crashed. A run narrowed by `--only` therefore names
every check it did not inspect with, and cannot read as a full run. Absent
optional tools are named skips, never silent, and never load-bearing for
`REJECT`.

A name that matches no registered check is refused rather than absorbed, in
both `--skip-checks` and `--only`.

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
[--format json|sarif] [-o file] [--skip-checks a,b] [--only a,b]`. A flag may
appear before or after the target and the report is identical either way; `--`
ends flag parsing, so a target whose name begins with `-` is written
`gate -- -target`. Exactly one target is required, an undefined flag is
refused wherever it appears, and a flag whose value is missing is refused
rather than read from the arguments beside it.

`skillgate gate --list-checks` prints the registered checks and the rules each
runs, and exits. It needs no target and produces no report, so the report
flags do not apply to it. It is the source of the names `--skip-checks` and
`--only` accept — see **checks_skipped contract**.
