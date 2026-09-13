# Problem brief — what skillgate is and how it detects

_Step-back synthesis, 2026-09-13. Sources: `docs/research/{recommendation,skillspector,rule-language}.md` re-read at head; `skillgate/` as-built._

## The problem, stated plainly

A skill is **prompt-shipped software**: markdown instructions, scripts,
configs, and references that an agent loads into context and executes with
the user's privileges. The consumer is an LLM that does what the text says —
so the bundle is simultaneously a *malware surface* (injection, exfil,
persistence, snooping) and a *defect surface* (dead refs, bloat, silent
skips). Three questions, one tool:

1. **Is this safe to install?** — the gate (v0.5.0's subject)
2. **What does it cost every session?** — always-on budget (partially built)
3. **Did my refactor pay?** — paired proof (profiler, slices 4–6)

## The detection surface decomposes into lanes, each with a right mechanism

| Lane | Where the badness lives | Right mechanism | In tree? |
|---|---|---|---|
| Lexical | instruction-override phrasing, injection in prose | patterns over **normalized text views** | partial — raw view only |
| Semantic | scripts that exfil, decode, persist | AST/taint for real languages; regex floor | regex floor only |
| Structural | hooks.json, MCP manifests, settings, frontmatter | parsed JSON/manifests | yes (T015–T018, I001…) |
| Graph | path escapes, dangling refs, cycles, **unreachable files** | resolved-reference graph | yes minus reachability |
| Coverage | what wasn't inspected | ledger + allow-listed skip reasons | yes |
| Provenance | unpinned fetch, mutated bundle | quarantine + SHA-256 manifest | yes |
| Estate | always-on token tax per item | exact counts + attribution | yes (Pack E) |

Two conclusions fall out:

- **"Regex is the wrong tool" is really two gaps, not one.** (a) The lexical
  lane scans *raw* text — upstream scans every window through six derived
  views (NFKC+confusable skeleton, compact-letter, declared-marker,
  continuity, obfuscated-instruction), so `i g n o r e`, `ｉｇｎｏｒｅ`,
  entity-encoded and marker-scrambled payloads all still match upstream and
  silently miss here. This is the largest detection gap and it is a
  **view problem, not a regex problem**. (b) The code lane has no
  AST/flow — genuinely ceded; a real engine belongs behind the advisory
  contract.
- **The rule count is not the architecture.** 20 tripwires is a floor, not
  the ceiling — SkillSpector's 113 IDs stay the deep-scan leg.

## On semgrep (and every other engine)

Already surveyed head-on (`rule-language.md:43-57`). As the canonical engine
it fails: OCaml core (not zero-dep), **no `markdown` language** (most
tripwires scan prose), interfile Pro-only, extract mode removed at 1.65.0.
**As an advisory leg it's cheap and honest** — same contract as
skillspector/agnix/skill-validator: on PATH → runs (over `Scripts` scope),
absent → named skip. Same for ast-grep later.

## The check model — two kinds, one contract

There are exactly two check kinds, and the user's "Go or bash scripts that
parallel-eval" is already both of them:

- **Native checks** — `Check{Name, Run}` over the shared read-only Ledger.
  Deterministic, fingerprinted, load-bearing. Currently sequential for no
  reason — the Ledger is immutable post-build, so `errgroup` makes them
  parallel in ~20 lines.
- **External checks** — binaries on PATH, advisory-only, named-skip when
  absent. The generalization of "bash scripts that eval a skill."

What's missing for scriptability: `--list-checks` and `--only a,b` so each
named unit is individually invocable (~15 lines). That gives per-check bash
composition without splitting the engine into N processes that each
re-parse the bundle and lose the shared context + fingerprint contract.

## Reachability — the missing graph leg (G003)

refgraph already resolves every file→file reference (dangling G001, cycles
G002). The leg the user wants: seed at entry points (SKILL.md files +
manifest-declared executables/hooks), walk forward, report files nothing
reaches. Unreachable = dead weight *or* staged payload. ~60 lines, no new
machinery, security-relevant, cheap enough for v0.5.0.

## What the research already decided (don't relitigate)

- SkillSpector stays an unmodified shell-out (Apache-2.0 vendoring burden,
  ~20 Python deps, no PyPI) — D2 ratified. Advisory, never load-bearing.
- Reimplementing 113 IDs / 27 analyzers / ~2,500 tests is the gold-plating
  trap — ceded set is deliberate.
- The typed-`Rule` migration (catalogue generation, matcher vocabulary,
  view axis) is specced at 7–11 days with fidelity proven by `difftest`
  differentials — it's the next epic, spec already written.
- F12/F13/F14 claim boundaries are the product's spine — already enforced.

## So what is v0.5.0 — the fork (DECIDED, 2026-09-13)

- **A. Ship the contract as staged** — the five-PR chain as-is. Fastest tag;
  detection depth all lands in 0.6.
- **B. Hold 0.5 for the engine** — typed rules + views + catalogue before
  any tag. Right destination, 7–11 day slip.
- **C. Ship contract + cheap detection wins** — recommended cut.

**Decision: a narrower C.** The user took C minus views/tables/artifact/
semgrep — see "Revised 0.5 boundary — DECIDED" below. Everything past the
fork section is 0.6 design input; only the zero-risk wins ship in 0.5.

---

## Refinement — the zero-dep constraint (user, 2026-09-13)

> "Zero deps, and anything valuable enough to add to the repo is something
> we rewrite — bash, Go, semgrep, etc. Understand and improve the design
> prior to including."

This resolves the fork structurally: **external engines stay advisory
forever; anything load-bearing is authored in-repo.** The decisions it
creates:

### 1. What "zero deps" binds

`go.mod` zero requires + a self-contained static binary + zero required
external tools. Go source compiled in is fine; a module require is not.
Consequences:

- **`x/text` currently violates it.** `difftest/ae6.go` imports
  `cases`/`norm` and `go mod tidy` added the require. Resolution: carve
  `difftest/` into its own module (dev harness, joined via `go.work`) so
  the gate module returns to zero requires — OR point difftest at the
  in-tree tables below once they exist. Prefer the module carve now;
  tables later.
- **Unicode tables (D6) resolve to "generate in-tree."** Upstream's
  skeleton uses a table generated from Unicode `confusables.txt` —
  BSD-licensed data. Our version: a Go generator (dev tool, `go:generate`,
  pinned Unicode version + checksum) that emits committed
  `*_tables.go`. go.mod stays at zero requires; difftest can diff our
  skeleton against upstream's rune-for-rune. The honest fallback is a
  curated subset (fullwidth arithmetic + T003 ranges — "missed detections
  only"), but generating the real table is strictly better and is the
  "rewrite in Go" answer.

### 2. Views become a pipeline stage — and join the coverage ledger

Design improvement over upstream, not just a port: views are
`(name, text, source_offsets)` artifacts built between ledger and checks.
Checks opt in per file class; hits map back through `source_offsets`;
raw-wins dedup preserved. **New: which views ran per file is recorded in
the ledger** — an unbuilt view is a named gap under F13, exactly like an
uninspected file or a missing binary. That turns the table decision from
a hidden limitation into honest coverage.

Pure-Go views (DeclaredMarker, Continuity — offset arithmetic only) need
no decision. Skeleton + CompactLetter wait on the generated tables.

### 3. Semgrep under zero-dep: authored rules, external engine

The repo owns a semgrep ruleset (YAML, ~8-15 rules: env→sink, curl|sh,
decode→exec, unpinned fetch — `Scripts` scope only). The `semgrep` binary
is an advisory leg: on PATH → runs our rules, absent → named skip. The
*rules* are the ported value; the *engine* is interchangeable. Identical
pattern for ast-grep later. Never load-bearing.

### 4. "Understand and improve before including" is a process, now

Every ported rule gets a design card before implementation: lane, views
required, divergences from upstream, fixtures, unported sub-lanes named.
The typed `Rule` struct is that card — which is why the catalogue
migration stays coupled to porting, not to the release.

### Revised 0.5 boundary — DECIDED (user, 2026-09-13)

**0.5 is the contract plus the zero-risk wins. Views, tables, the
artifact, semgrep, and the typed-Rule migration are the 0.6 epic** —
`rule-language.md` is already its spec; the G003 card and the view/table
sections below are its design input, not 0.5 scope.

In 0.5, on top of the staged five-PR chain:

- **G003 reachability** on the existing refgraph, emitting the additive
  `reachability` block. Extract the small "convention loads" seed table
  now so G003 and T015–T018 read one list; per-rule config parsing stays
  where it is (consolidation is 0.6's first slice).
- **`--list-checks` / `--only` + parallel native checks** over the
  immutable ledger.
- **`difftest/` module carve** — required regardless of boundary: the
  `x/text` require in the gate module's go.mod makes the zero-requires
  claim false as staged.
- **Fix the raw-lane misses E2 already shows** — structural and raw-lane
  gaps, not view gaps; cheapest detection wins available, no new
  machinery:
  - difftest ports: BH1's three hooks.json/settings.json document-level
    misses; TP1's escaped-comment miss (`<\!--`).
  - live gate (verified this session): **T017 sees only `"command"`
    hook values — a `"type": "http"` hook posting to `exfil.example`
    produced zero findings**; a `data:text/plain;base64,` blob decoding
    to injection text produced zero findings.
- **The spec sentence naming ceded lanes** stays.

Deferred to 0.6, one epic: all four views (Continuity and DeclaredMarker
included), generated Unicode tables, the authored semgrep ruleset + leg,
the artifact model, the typed-Rule migration. Rationale, in the user's
words: the mandate freezes rule representation this release and a view is
the front half of that representation (a rule must declare which views it
reads); position honesty is the hard part and a source-offset bug ships a
provenance lie — the exact failure the product defines itself against —
so no view ships on the first tag without the view-level difftest corpus
proving it. Semgrep rides along because each rule needs a design card
under our own process, and an advisory leg that names a skip on most
machines adds nothing load-bearing to 0.5.

The two seams, resolved for 0.6: manifest parsing consolidates into
`Artifact.Manifests` as the artifact epic's first slice (the 0.5 seed
table is its down payment — do not refactor T015–T018 before then);
typed `Rule` lands **with** the artifact, migrating lane-by-lane behind
difftest fidelity per rule — checks keep their current `rule` shape in
0.5 and read from the artifact once it exists.

---

## The eval artifact — formalizing the graph view (0.6 epic design input)

The eval pipeline adopts the 60/30/10 shape it enforces:

```
bundle → [BUILD: artifact] → [RUN: checks] → [JUDGE: advisory]
          ~deterministic      deterministic      re-runnable
          60%                 30%                10%
```

**Artifact** — built once, pure function of the bundle, pinned by the
ledger's manifest SHA. Members: `Ledger` (exists), `Views` (new),
`Graph` (refgraph + seeds + reachability), `Manifests` (frontmatter +
configs, parsed once — today T015–T018 re-parse ad hoc), `Budget`
(exists). Checks become `f(*Artifact) []Finding` — never touching the
filesystem — which is what makes them parallel, `--only`-invocable, and
replayable.

**The graph is the spine.** G001 (dangling) and G003 (unreachable) are
dual queries over one edge set; BFS depth from seeds gives ICM its layer
axis — `branch_count`, deferred fraction, depth>1, and the token share per
layer all become graph queries. The skill's own 60/30/10 stops being a
judgment estimate (`skill-audit/SKILL.md:154`) and becomes measurable.

**Re-run property.** Artifact pinned by content hash ⇒ a new check is a
new pure function over old bundles; baselines become
`(artifact-sha, rule-id, fingerprint)`; difftest diffs the artifact itself
view-for-view against upstream. Judgment (the ~10%) stays advisory,
additive-only, re-runnable against the same pinned input.

**Serialization discipline.** Internal boundary first
(`BuildModel(dir) → *Model`); serialize into report/v1 only where a
consumer exists — the `reachability` block is the first slice ("the dep
tree comes back" = emit the graph section). Schema grows additively.

**Independent convergence:** SkillSpector's own pipeline is
`build_context → parallel analyzers over one file cache → meta`. Same
shape reached from the opposite direction — evidence the boundary is real.

### G003 design card — reachability (ships in 0.5)

- **Seeds** = files a harness loads *by convention*, no pointer required:
  every `SKILL.md`; convention configs (`hooks/hooks.json`,
  `.claude/settings.json`, `.cursor/hooks.json`, `mcp.json`/`.mcp.json`,
  `.codex/` hook configs, `package.json#pi`); convention-loaded text
  (root `AGENTS.md`/`CLAUDE.md`/`GEMINI.md`, `.cursor/rules/*.mdc`).
  The seed table is the same surface T015–T018 parse — one
  "convention loads" table feeds both.
- **Edges** = `extractRefs` over **every inspected file**, not only
  `isLoadedText` — scripts `source`/`exec`, JSON configs name commands.
  (G001/G002 keep the narrower prose scope deliberately; G003's edge set
  is wider and that's a stated divergence.)
- **Classification** = three-way, never overclaim: `reachable` /
  `unreachable` (finding — executable/binary unreachable is the staged-
  payload case, doc/asset is cruft at Info) / `dynamic_only` (dir-iterated
  or glob/`$var`-referenced — recorded, not flagged).
- **Scope** = skill roots only; files outside every SKILL.md root are
  repo cruft on repo-shaped targets, exempt or Info.
- **Output** = `reachability` block in report/v1: `seeds`, `reached`,
  `unreached`, `dynamic_only` — sorted, the tree itself is evidence.

---

## The view pipeline — design card (0.6 epic design input)

### Contract (invariants worth keeping from upstream)

- A view is `(name, text, source_offsets)`; `source_offsets[i]` = the raw
  byte index that produced derived char `i` (byte-indexed for Go; a
  rune-expanded char maps all its derived bytes to the source rune's
  start).
- Findings always carry **raw-file** position: derived match → source
  offset → raw line via bisect. Non-raw hits tagged with the view name.
- Dedup is **raw-wins**: a derived hit carrying a raw hit's key is
  discarded — but a derived *non-benign* classification is never
  suppressed by a benign raw one.
- Views are additive and per-rule: a rule declares `views: [raw,
  skeleton, …]`; raw always runs.
- **Improvement over upstream**: which views ran per file joins the
  coverage ledger — an unbuilt view is a named gap under F13.

### Port order (dependency-driven)

| View | Needs | Evasion it defeats |
|---|---|---|
| Continuity | pure Go offset arithmetic | long separator runs splitting phrases (`ignore` … 50KB dashes … `all instructions`) |
| DeclaredMarker | pure Go | "strip the markers 'X'" directives; `&#NN;` entities; spaced verbs (`r e m o v e`) |
| Skeleton | generated tables | fullwidth `ｉｇｎｏｒｅ`, homoglyphs, math alphanumeric 𝐢𝐠𝐧𝐨𝐫𝐞, zero-width-interleaved tokens |
| CompactLetter | skeleton + tables | letter-spaced `i g n o r e`, non-ASCII separators, U+FFFD, fillers |
| ObfuscatedInstruction | skeleton | filler removal inside trigger phrases only — defer, it's a refinement |

### The table decision — a simplification hypothesis

Upstream does per-char **NFKC → UTS #39 confusable skeleton → strip
Cf/Cc/default-ignorable**. Under zero-dep we generate tables in-tree —
but `confusables.txt` *already is* a skeleton map: for defeating
ASCII-phrase obfuscation, skeleton-only may subsume NFKC (fullwidth,
homoglyphs, math alphanumeric are all confusables entries). **Hypothesis:
skeleton + category tables (Cf/Cc ranges, default-ignorable) ≈
NFKC→skeleton for our rule set.** Three small generated tables, one
generator (`go:generate`, pinned Unicode version + SHA256 of the data
file), committed `*_tables.go`, go.mod stays zero requires.

If difftest shows real misses (compatibility decompositions not covered
by confusables — circled letters, ligature-like forms), add the NFKC
table as a fourth generated artifact. Divergence is measured, not
argued.

### Fidelity harness

difftest already runs upstream-vs-port on fixtures. Add a view-level
corpus: per evasion class, a positive fixture (payload present in
obfuscated form) **and** a negative (legit text exercising the same
script — a CJK reference doc, a Unicode-explainer) — the negative is what
keeps skeleton folding honest. Expected finding set per fixture is
asserted on the artifact, not on final findings, so view behavior is
tested in isolation.

### Consumption

`FileContent.Views []View` (or per-file record on the artifact). Runner
builds the union of views the enabled rules declare for that file class —
deterministic, lazy. T002 (instruction-override) is the canonical
skeleton+compact consumer; T001 already detects the runes, views are what
let rules match *through* them; T003's hand-rolled script ranges fold
onto the skeleton table.
