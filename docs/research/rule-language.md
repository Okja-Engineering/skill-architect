# Rule language for skill audit — typed Go rules, generated catalogue, code where it counts

Cites: `ss:` = SkillSpector clone `src/skillspector/…` (commit **8421a2e**; `artifacts.py`, `security_reconstruction.py`, `unicode_confusables.py` sit at that root, **not** under `nodes/analyzers/`); `sa:` = `skill-architect/skillgate`; `cp:` = cursor-profiler `docs/research`; `probe:` = `scratchpad/mergeprobe`, `scratchpad/ea1diff` (Go **1.27.1**, re-run this session). **[O]** observed; **[I]** inferred; every path cite re-read at head. The 15 worked re-expressions, the RE2-hostile site lists and the probe transcripts are in the companion **`rule-language-samples.md`**.

## 1. Five-line summary

1. SkillSpector's analyzer layer: 33 files / 22,311 lines / 225 `re.*` sites [O grep]. RE2-hostile = **24 lookaround/backreference/atomic lines in 11 files**, plus 3 raw-string `\uXXXX` sites Go rejects outright, 5 counted repeats >1000, 3 `\Z`, 2 `(?x)`, 10 `\p{bc=}` behind the third-party `regex` module [O §2].
2. No surveyed DSL fits zero-dep Go **and** the mandate's document scopes: each is non-RE2, YAML-bound, byte- or field-only, or engine-gated [O §3].
3. Proposal: the canonical rule is a **typed Go value** (`Rule{ID, Severity, Effort, Remediation, Scope, Detect, Ceded, FallbackOf, Expect, Tests…}`) over ten matcher constructors and one shared `Ctx`. Catalogue, SARIF `rules[]`, notices and fixtures are **generated from** those values; data files exist for three things only — policy profile, generated catalogue, fixtures.
4. Sample fidelity, **by lane, not by ID**: 8 of 15 fit whole — the seven lexical ones only **on the raw view** until the View axis lands (§2, §4.3) — 6 fit one lane of a multi-lane upstream check (OH1, TM1, AE6, SSR-1, TP1, BH2), 1 does not fit (TT2). Every gap is a missing parser, Unicode table, view or unported sibling lane — never a missing format feature [O §4.4].
5. Impact = four numbers never blended (Risk, Debt, Load, Lift); `expected_outcome` is a **rule-and-path-scoped re-run predicate**, not a fingerprint match; effective spend refuses to compute across basis stamps; effectiveness is **six named states with a token-direction axis**, a count per state plus a lift histogram, never a mean.

## 2. What SkillSpector's analyzers actually are

| Mechanism | Where | Note |
|---|---|---|
| Regex `(pattern, confidence)` tuples, `IGNORECASE\|MULTILINE` | 15 `static_patterns_*` + TP/RP/LP | 225 `re.*` sites [O]. The circulating "219 checks / 318 tuples" reproduce under no counting rule and are dropped [O] |
| Hybrid: regex + post-filter / downgrade / clause logic | AR triage, TM1, RA1, P6, MP2, P5-substance, SC2, BH2 | the FP layer: tags-never-suppresses (`anti_refusal.py:409-429`), LOW/0.15 downgrade (`tool_misuse.py:2174-2180`) [O] |
| Structural (Unicode classes, JSON walks, path/graph) | P2, P9, EA5, BH1/BH3, TP2, AE2-5, SC5-9, TR1-3 | procedural; no regex core |
| Python AST · taint | `behavioral_ast.py`, `behavioral_taint_tracking.py` | `.py` only; taint is flow- and scope-insensitive over `ast.walk` [O] |
| YARA · LLM · network | `static_yara.py` (26 rules → 4 IDs), TP4/SDI/SQP/SSD, SC4 OSV | LLM is add-only — `meta_analyzer.py:380-395` never deletes a static finding [O] |
| **Derived security text views** | `_scan_all_views_detailed` over `artifacts.security_text_views()` | every lexical `static_patterns_*` module runs on raw **plus** derived views (`ss:nodes/analyzers/static_runner.py:1290-1328`); AST modules never see one (`:1137-1138`, `:1230-1232`) [O] |

113 emitting IDs (`cp:recommendation.md:112`). Score: CRITICAL 50 / HIGH 25 / MEDIUM 10 / LOW 5 (`ss:report.py:381-385`), ×1.3 on executables only (`:459-462,:513-519`), LOW band labelled **"SAFE"** (`:70-72`) — the anti-pattern F12 forbids [O].

**The view layer — what "runs the pattern" means upstream.** All seven lexical whole-fit rules (AS3, AR2, EA1, P5, MP2, RA1, P6 — none sets `USES_PYTHON_AST` [O grep]; P9 is structural) run on **every** view of each raw window, never raw alone (`static_runner.py:1290-1328`), plus declared-marker (`:1192-1202`, `:1052-1091`) and continuity views (`:1377-1398`); all views are sliced at 256,000 chars with 8192 overlap first (`:90-92`, `:751-766`). Views [O `ss:artifacts.py:1721-1752`]: **raw**; **normalized** = per-char NFKC → UTS #39 ASCII confusable skeleton (table generated from Unicode 17 `confusables.txt`, `ss:unicode_confusables.py:4-9`) minus Cf/Cc controls and default-ignorable runs (`:1609-1630`, `:355-360`); **compact** = that fold minus letter-spacing separators in runs of ≥6 letters (`:1662-1700`, `:204`), non-ASCII Z-separators (`:1593-1602`), U+FFFD and context-bound fillers; **obfuscated-instruction** = the fold minus fillers inside `ignore|override|bypass|disregard|forget` phrases only (`:1633-1659`, `:241-247`). **Declared-marker** views rebuild the payload of "remove/strip/delete/replace … the following markers 'X'" directives (`ss:security_reconstruction.py:43-54`, `:1467-1543`) after decoding `&#NN;` letter entities (`:366-367`) and compacting spaced verbs (`:399-405`); ≤8 directives, 700-char payloads (`:21-22`). **Continuity** views bridge separator runs wider than the overlap: 2048 chars of context per side, ≤24 chained runs, separator truncated to 8192 head+tail (`static_runner.py:113-115`, `:838-900`); P9 excluded (`:1411`).

**RE2-incompatible set.** *Invariant: the census is derived by compiling every extracted pattern under Go `regexp`, never by grepping for named constructs — the `\u` row is what grepping misses.*

| Construct | Count · exemplar | Rewrite |
|---|---|---|
| Raw-string `\uXXXX` | 3 · `common.py:31` `LOGICAL_LINE_BREAK`, imported by 6 analyzers into the P6/MP2 triage regexes; `prompt_injection.py:83,89` | Go: `invalid escape sequence: \u`; spell `\x{2028}` [O probe]. **Loader lint rejects `\u`** |
| Lookaround (neg 13 · behind 5 · pos 2) | 20 · `excessive_agency.py:47`, `output_handling.py:70` | consume the neighbour + `Reject`/`RejectAfter`; `(?<!X)` at a possible string/group start → an **optional** consuming group (I1) |
| Backreference `\1`, `(?P=q)` | 3 · `memory_poisoning.py:84` | `Repeat{}` codepoint period scan **with case folding**; enumerate quote alternatives |
| Atomic `(?>…)` | 1 · `tool_misuse.py:47` | drop (RE2 never backtracks) |
| Counted repeat >1000 | 5 in 2 files · `tool_misuse.py:119` `{0,8192}`, `mcp_least_privilege.py:84,85,94,95` `{0,4096}` | bounded Go extractor + a small regex; `{0,1000}` compiles, `{0,1001}` does not [O probe] |
| `\Z` · `(?x)` | 3 · 2 · `supply_chain.py:823` · `anti_refusal.py:194` | `\z`; flatten at authoring |
| `\p{bc=…}` | 10 · `bundled_execution_surface.py:51-66` | no Bidi_Class tables in stdlib — D6 |
| Class drift `\b \w \d \s` | every ported pattern | Python Unicode vs Go ASCII: `(?:[^\W\d_](?:[^\w]\|_)+){5}[^\W\d_]` is **false** on `ｉ ｇ ｎ ｏ ｒ ｅ` and `и г н о р е` in Go, true in Python (`ss:artifacts.py:196-202`, `re.UNICODE`); `\p{L}` is true on both [O probe] |

The 24 hostile lines sit in **11** files [O grep]; 10 of 11 `regex.` hits are the third-party module, the 11th (`deserialization.py:131`) is a loop variable from `_COMPILED` [O].

## 3. Landscape against the constraints

| Language | Zero-dep Go? | RE2? | Doc scopes? | Borrow |
|---|---|---|---|---|
| Semgrep CE / OpenGrep | No — OCaml core | No — PCRE2 | no `markdown`; extract mode removed 1.65.0; interfile Pro-only | operator names, `ruleid:/ok:` tests, metadata discipline |
| YARA | No — cgo | no backrefs | bytes only | `N of ($fam_*)`, typed `meta`, proximity |
| Sigma | No — YAML; 6 deps | backend-defined | field-scoped over a *parsed* doc | `field\|modifier: value`, `falsepositives:` |
| Vale | No — ~30 deps | No — backtracking | per-key frontmatter; drops fences | rule kinds, scope grammar, in-rule `tests:` |
| markdownlint / remark / textlint | No — JS | n/a | mdast nodes incl. `code.lang`, `Yaml` | node names as scopes; `valid[]/invalid[]` |
| ast-grep / tree-sitter | No Go binding | Rust regex | syntax only | `{id, valid, invalid}`; opt-in shell-out |
| CodeQL | No — closed CLI, no Bash | n/a | n/a | nothing |
| OPA · CUE · CEL | No — 47 / 25 / 6 requires | n/a | policy layer only | ladder + unification shapes, for the profile |
| **Go stdlib** | **Yes** (`sa:go.mod`, no require block) [O] | linear-time | fence splitter must be built; frontmatter reader exists (`sa:frontmatter.go:15-64`) | the substrate |

All rows [O landscape]; Go row [O re-read].

## 4. The proposed expression

### 4.1 Where a DSL is worth it

| Question | Answer |
|---|---|
| Who authors rules? | The engine team, in Go. Pack B caps at 20 (`cp:recommendation.md:114`) plus ~30 across Packs C-H; no third-party matcher requirement exists [O] |
| What must be data? | Rule **metadata** — id, severity, effort, remediation, origin, tests. Today SARIF `rules[]` is built only from *fired* findings (`sa:sarif.go:91-101`) and no live rule carries remediation (`sa:tripwire.go:19-32`) [O] |
| Does matcher logic fit data? | **7 of 15** need a Go predicate no matcher DSL supplies (MP2 period scan, TM1 clause extractor, P9, AE6, TP1 decode gate, BH2 proof, SSR-1 fact); the other 8 are regex + scope + capture-reject [O §4.4] |
| What a DSL costs | A loader (JSON is stdlib; YAML is not — `sa:frontmatter.go:8-10` "deliberately not a YAML parser"), validation, a second syntax to document [O] |
| Where data is right | policy profile · generated catalogue · fixtures · word tables as Go slices beside the rule [I] |

The typed Go value is the canonical rule; the "human-readable rule language" is the **generated catalogue**, not an input format.

### 4.2 Format

```go
type Rule struct {
    ID, Pack, Quality string   // "SK-T004", "B", security|reliability|maintainability
    Severity   Severity        // the only rule field the verdict reads (sa:verdict.go:12-30)
    Effort     int             // minutes; uncalibrated placeholder until n≥20 fixes (F4)
    Message, Background, Remediation string  // §6 problem / context / proposed solution
    Origin     []string        // upstream provenance; generates THIRD_PARTY_NOTICES.md (D5)
    Expect     Expect          // §6 re-run predicate
    Scope      Scope           // scope + View; widens when FallbackOf's engine is skipped
    Exclude    []string        // basename globs (P9's *.min.js…)
    Detect     Detector        // the only executable part; nil when Ceded
    Weight     float64         // Pack I scoring prior; never a verdict input
    Tags       []string        // category quirks travel here (P5)
    Ceded      string          // "external:skillspector" — the shell-out owns this lane
    FallbackOf string          // covers <id>'s surface when <id>'s engine is skipped
    Tests      Tests           // Fires, Quiet — ≥1 each, enforced
}
```

`Detect` returns `Hit{Line, Evidence, Severity, Weight, Tags, Summary}` over a `Ctx{Ledger, Docs, Refs, Configs, facts}` built once per run.

**Split-rule engine delegation (the F13 coupling).** A rule may be cleaved into a native and a ceded half sharing an ID family — `SS-OH1-fallback`/`SS-OH1-ast`, `SS-TM1-regex`/`SS-TM1-shell`. With the engine absent the ceded half lands in `checks_skipped[]` and caps the verdict at CAUTION (`sa:gate.go:133-136`; F13 `cp:recommendation.md:192`), **and** its `FallbackOf` sibling lifts its path exclusion over the ceded surface, tagged `fallback`. A bare `Ceded` routes every `.py` to the shell-out and silently loses that coverage when it is absent.

Generated by `go test ./rules` + `go generate`: `catalogue.md`, `catalogue.json` (feeds `rules[]` with `help.text`=Remediation, `properties.effortMinutes`, `partialFingerprints`), `THIRD_PARTY_NOTICES.md` from `Origin`, the fixture tables. `TestCatalogue` enforces: every regex compiles under Go `regexp` (a bad pattern fails a test, not `MustCompile` at init), **no `\u` escape**, Remediation non-empty, Effort>0, ≥1 firing and ≥1 quiet fixture, Pack B ≤20 — **and, for every rule marked byte-identical, a Python-vs-Go evidence differential over its whole fixture set**, EA1's carrying the trailing-whitespace, NBSP, CRLF and non-ASCII-word forms of I3.

### 4.3 Matcher vocabulary, scopes, composition

| Constructor | Meaning |
|---|---|
| `Lines(re…)` · `Span(P(w,re)…)` | per-line / finditer over the scoped text; evidence ≤160 chars (`sa:tripwire.go:62`) |
| `.Reject(pred)` · `.RejectAfter(g,re)` · `.Tag(cond,tags…)` | drop on a predicate over the capture; drop when the text at **group g's end** matches `re` — the general lookahead substitute (I3); keep and tag (SS triage) |
| `.Downgrade(cond,sev,w)` · `.Boost` · `.Weight(cond,w)` · `WeightLadder(W(cond,w)…)` · `.Dedup(PerLineMax)` · `.TrimGroup(g,pred)` | severity/weight change, never deletion (TM1 LOW/0.15); **`WeightLadder` is ordered — the first arm whose condition holds sets the weight and no later arm may override**, the only form that expresses a source `if/elif` ladder (P5 lane 2, **I7**); a bare `.Weight` chain is **last-wins**; max-weight hit per (file,line); drop a consumed neighbour from the evidence |
| `SameLine`·`SameFile`·`Near(n)`·`CtxLines`·`CtxChars`·`ClausePrefix/After`·`LineIs`·`MatchIs` | co-occurrence, clause and window conditions (T005/T009; RA1, AR2, TM1, P6, P5) |
| `Repeat{MinUnit,MaxUnit,MinReps,FoldCase,NoNewlineInUnit}` | O(19·n) codepoint period scan — MP2's `\1` is not a regular language |
| `Native(fn)` | Go function over Ctx/Doc; must still declare Scope and Tests (P9, AE6, BH2, TP1, G002) |

Scopes: `AllText` · `LoadedText` · `Scripts(ext…)` (`sa:frontmatter.go:104-122`) · `Configs(paths…)` (JSON parsed once into `Ctx`; **MCP manifests live here** — `parameters[].name/.description` are manifest fields, not frontmatter: `_extract_metadata_texts(manifest: dict)`, `ss:mcp_tool_poisoning.py:168-177` [O]) · `Frontmatter(keys…)` (top-level only) · `Body` · `Prose` · `Fences(langs…)` · `Bundle`. The fence splitter is new (~40 lines; skillgate has none [O grep]). Composition: `All`/`Any` over one unit; negation only as `Reject`/`RejectAfter` or a second regex — never inline.

**View axis — the missing half of Scope.** `Lines`/`Span` run over scoped **raw** text (every sample is `Scope: AllText`, `cp:rule-language-samples.md:75,100,123,146,177`) and skillgate's `FileContent.Text` is `string(data)` unmodified (`sa:ledger.go:116-134`; `lineMatches` `sa:tripwire_a.go:88-99`) [O], while upstream scans each window through the views of §2. Every "whole" verdict therefore loses: fullwidth/confusable/NFKC spellings; zero-width/Cf-interleaved tokens (T001 *flags* those runes, `sa:tripwire_a.go:102-127`, but no rule matches *through* them); letter-spaced `i g n o r e` (P6, AR2, P5 phrases); entity-encoded or marker-scrambled payloads (P6/AR2); tokens split across windows by a long separator (any `\s+` pattern) [I]. AE6's rule-local `FoldFullwidthAndConfusableThenCasefold` (`samples.md:326`) is the only fold here, not a shared view. **Recommend** adding `View: Raw | Skeleton | CompactLetter | DeclaredMarker | Continuity` to `Scope` with upstream's contract as invariants: a view is `(name, text, source_offsets)` whose `source_offset(i)` returns the raw index that produced derived char `i` (`ss:artifacts.py:72-86`); the runner stores each finding's view-start as a source offset and tags non-raw hits `normalized-view`, marker hits also `declared-marker-view` (`static_runner.py:736-747`); drops hits whose source start leaves the window's owned span (`:1337-1342`); rewrites `start_line`/`end_line` by mapping the derived line-start back and bisecting the raw line table (`:981-1005`); dedups **raw-wins** — a derived hit carrying a raw hit's key is discarded, but a derived non-benign classification is never suppressed by a benign raw one (`:689-718`; marker views also dedup on text + first/last source offset, `:1069-1076`). `Skeleton`/`CompactLetter` need NFKC tables ⇒ **D6**; `DeclaredMarker`/`Continuity` are pure-Go offset arithmetic. Until they land each rule carries "unported: raw view only — loses {classes}" and the seven lexical "whole" verdicts read **whole on the raw view**.

### 4.4 The 15 sample rules — fit, by lane

Worked re-expressions and per-rule fidelity notes: **`rule-language-samples.md`**. Seven (AR2, P5, OH1, TM1, P6, TT2, SSR-1) are in the plan's ceded set (`cp:recommendation.md:36`); this proves the format, it does not propose adding them to Pack B.

| Rule | Upstream lanes | Native here | Fit |
|---|---|---|---|
| AS3 · AR2 · RA1 · P6 | 1 (+triage / +suppressors) | all | **whole** (raw view) |
| EA1 | 1 | 1 | **whole** — byte-identical only with `.RejectAfter` (I3) |
| P5 | **2**: 10 action branches + 15 bare-substance, context-adjusted confidence, `≥0.5` emit gate (`harmful_content.py:82`, loop `:109-132`) | 2 | **whole** (raw view) — lane 2 only as an **ordered** `WeightLadder` (I7); the flattened `.Weight` chain drops a 0.95 Blocker |
| MP2 | 1 backreference + 2 post-filters (`:312`,`:314-316`) | 3 | **whole** (I2; raw view) |
| P9 | 5 detectors + overlap suppression | 5 | **whole** — `Native`, stdlib; `\p{Zs}\p{Zl}\p{Zp}` compiles [O probe] |
| OH1 | **3**: python AST · fallback regex · JS regexp-literal suppressor (`output_handling.py:200-510`) | 1 + `FallbackOf` | partial — and the one ported lane is byte-identical **only** with `.RejectAfter` (I3) |
| TM1 | **2**: 20-branch regex · 1,100-line shell lexer | 1 + `FallbackOf` | partial |
| AE6 | **3**: letter-spacing · contextual-ignorable · targeted-instruction (`min()`, `artifact_integrity.py:641`) | 1, ID **must** be `SS-AE6/letter-spacing` | partial |
| TP1 | **5**: data URI · HTML comment · markdown comment · zero-width · base64 (`mcp_tool_poisoning.py:239-398`) | 1, ID `SS-TP1/base64`, fields via `Configs` | partial |
| BH2 | argv/URL proof · BiDi/punycode · WHATWG | proof only | partial (I4) |
| SSR-1 | gate · `summaries[]` channel · AISOP payload parser | 2 | partial (I6) |
| TT2 | flow-insensitive taint | 0, `Ceded` | **no** — no zero-dep Python parser; identity, both FP modes, effort and fixtures still carried |

**The seven invariants the sample proves** (probe transcripts in the companion):

- **I1 — lookaround direction.** `(?<!X)` where the position may be a string/group start rewrites to an **optional** consuming group. OH1's optional form matches `subprocess.run(output)` and `…(output, shell=True)`, rejects `…run(cmd)output` and `…run(["ls","--output"])`; the mandatory form inverts all four [O probe].
- **I2 — a native must reproduce the construct under the source's flags.** MP2's `(.{2,20}?)\1{20,}` runs under `IGNORECASE|MULTILINE` (`:310`), so the run detector must **fold case** and bound the unit in **codepoints**: `('abAB'*11)` fires with `re.I`, not without [O]. No DOTALL ⇒ the unit cannot span a line.
- **I3 — evidence must stay byte-identical, and only a differential proves it.** `Fingerprint(ruleID, path, evidence, severity)` hashes the evidence and "on any content drift it suppresses nothing (fail-closed)" (`sa:baseline.go:17-24`). The earlier "drifts one char only on `tools: *,`" claim was an artefact of a nine-form probe with no trailing whitespace, NBSP or CRLF (`probe:mergeprobe/ea1/main.go:11`) [O]. A 19-form differential of upstream `\*(?!\*|\w)['"]?[ \t]*\]?` under `IGNORECASE|MULTILINE`, evidence `match.group(0)[:200]` (`ss:static_patterns_excessive_agency.py:47,375,386`), against the shipped alternation `\*(?:['"][ \t]*\]?|[ \t]*\]|[^*\w\r\n]|$)` (`samples.md:127`) shows **seven** drifts [O probe `scratchpad/ea1diff`]:

  | form | upstream | alternation |
  |---|---|---|
  | `tools: *,` | `tools: *` | `tools: *,` |
  | `*  # all` · `*\t\t` · `*   \n` | keeps every trailing `[ \t]` | keeps exactly one |
  | `tools: *\xa0x` | `tools: *` | `tools: *\xa0` |
  | `tools: *\r\n…` | fires | **silent** — `[^*\w\r\n]` and `(?m)$` both fail before `\r` |
  | `tools: *ü` | silent (Python `\w` is Unicode) | fires (Go `\w` = `[0-9A-Za-z_]`) |

  Every drift is baseline-visible; the CRLF row is a **missed detection**, and skillgate has no `\r` normalisation [O grep]. `RejectNext` at match end is not the fix: upstream's lookahead sits after `\*`, *before* the optional tail, so rejecting on the char after the whole match repairs six rows, still fires on `*ü`, and silences four forms upstream matches (`tools: * x`, `[*]x`, `"*"x`, `* ]*`) [O]. The substitute must anchor to the `*`: keep the tail verbatim, capture `(\*)`, reject when the text at **group 1's end** matches `[*\p{L}\p{N}_]` (Python str `\w` ≈ `isalnum()||'_'`) — byte-identical on all 19 forms, where the ASCII class `[*\w]` differs on `*ü` alone. Hence `.RejectAfter` (§4.3) and the differential in `TestCatalogue` (§4.2); a golden diff over a clean corpus catches none of it.

  **OH1 is the same defect at the tail, and worse.** Upstream's fallback ends on the *zero-width* `(?!\w)` and emits `match.group(0)[:200]` (`ss:output_handling.py:70`, `:528-547`), so the match stops at the name. The shipped `(?:[^\w]|\z)` tail **consumes** the boundary character, which moves the match end and appends that byte to **every** evidence string — `subprocess.run(output)` yields `subprocess.run(output)` where upstream yields `subprocess.run(output`; `…(output, shell=True)` yields `…(output,`; `check_output(["sh","-c",result])` yields `…result]` — **and** it fires where upstream is silent, because Go's `[^\w]` is ASCII: `subprocess.run(outputü)` matches in Go and not in Python. That is a match-set divergence (a false positive on a High/0.85 rule), not only evidence drift, so `TrimGroup` — which repairs the bytes alone — is not the fix. The fix is EA1's non-consuming substitute: end the pattern at the name and post-filter with a `.RejectAfter` on group `name` against `[\p{L}\p{N}_]`, and widen the *leading* optional group's class to `[^-'")\p{L}\p{N}_]` for the same reason — upstream's lookbehind `(?<![-\w'"])` is Unicode, so the ASCII spelling also fires on `subprocess.run(üoutput)`. Corrected, the rule is byte-identical to upstream on all six matching forms, silent on all four the source is silent on, and the four I1 cases are unchanged [O probe `scratchpad/oh1diff`; full table in the annex §A].
- **I4 — a closed-proof rule resolves an unmodeled predicate toward NO proof**, upstream's own direction: `_parse_http_url` (`:445-455`) returns None unless `_hostname_satisfies_bidi_rule`, and `_is_valid_literal_host` (`:285-311`) returns False on invalid punycode, so such a host is not remote and **no BH2 fires** [O]. Treating `xn--` as remote fires where the source is silent, on a Blocker at 0.99.
- **I5 — port at full width or qualify the ID.** Bare `SS-AE6` would report full coverage from a third of the detector; sample-fit tallies count **lanes, not IDs**.
- **I6 — no nondeterminism crosses the boundary.** RE2's linear time makes P5's DOTALL branch safe **verbatim** (the 1000 cap is on *counted* repeats, not lazy `.*?`) [O probe]; SSR-1's upstream wall-clock budget (`ss:structured_skill.py:449-453`, abort at `:137`) must **not** be carried into a deterministic engine by a future native port.
- **I7 — an ordered source ladder must stay ordered.** P5 lane 2's confidence is an `if/elif/elif` (`ss:harmful_content.py:113-119`): instructional wins at **0.95**; educational (0.3) and warning (0.2) apply **only** when instructional does not. Three chained `.Weight` calls are *last-wins*, and the marker sets overlap by construction — `warning`, `danger`, `never` and `do not` sit in **both** the educational list (`:157-180`) and the warning list (`:182-196`) — so a recipe line naming a substance beside a `WARNING: do not serve…` line in its ±5-line context scores **0.95 and emits** upstream but **0.2 and is deleted** by the `confidence >= 0.5` gate (`:120`) once flattened: a Blocker silently suppressed [O probe, the three predicates transcribed verbatim]. Hence `WeightLadder` (§4.3), a `TestCatalogue` lint that rejects a second `.Weight` on a detector already carrying one, and a fixture pinning that case at 0.95.

### 4.5 What does not fit, honestly

| Gap | Why | Disposition |
|---|---|---|
| Python AST / taint (TT2, OH1-ast, AST1-10, TT1-6) | no zero-dep Python parser | `Ceded` + Advisory findings + a `FallbackOf` sibling where one exists |
| NFKC + confusable skeleton (AE6, P2/TP2) | `x/text/unicode/norm` not importable under zero-dep [O] | fullwidth arithmetic + T003 ranges (`sa:tripwire_a.go:156-157`) = a documented **subset**, missed detections only; or the in-tree route, D6 |
| Shared derived views for every lexical rule (§2) | `Skeleton`/`CompactLetter` need those same tables [O] | port `DeclaredMarker`/`Continuity` now — pure offset arithmetic; `Skeleton`/`CompactLetter` behind **D6**; until then each rule is annotated "raw view only" |
| Bidi_Class · WHATWG URLs (BH2, BH1) | no bidi tables; `net/url` is RFC 3986 | unmodeled ⇒ no proof; prenormalize `https:/{1,3}` + fixtures for divergences |
| Nested frontmatter (EA5 `metadata.*`) | reader is top-level only (`sa:frontmatter.go:40-43`) | keep the reader minimal; TP1 does **not** need it |
| Unicode class semantics; SS ±3/±5-line `context` | Go classes are ASCII [O probe]; `Finding` carries `Evidence` only | `\p{L}` rewrite + fixture re-run via `TestCatalogue`; evidence = directive line |
| YARA · OSV · LLM | cgo / network / non-deterministic | ceded; G6 add-only lane |

## 5. The impact model — four numbers, never blended

| Dimension | Unit | Computation | Basis: billed / measured / estimated |
|---|---|---|---|
| **Risk** | 0-100 | Σ points(50/25/10/5) × per-rule 1/.5/.25 × `Weight` × 1.3 on script files; cap 100 (`ss:report.py:381-385,459-462,513-519`; `cp:recommendation.md:161`) | computed; **no money**; never the word "safe" (F12, `:191`) |
| **Debt** | minutes | Σ `Rule.Effort` over unsuppressed findings | **estimated placeholder**; grade and debt ratio withheld until n ≥ 20 observed fixes per rule (F4, `:163`) |
| **Load** | tokens (o200k), chars | always-on per item vs 0.01 × window and 1,536 chars (`:157`); per-activation body vs `icmBodyTokenLimit = 8000` / `icmBodyLineLimit = 500` (`sa:icm.go:21-22`) | **measured (static exact count), never billed**: `sa:skillvalidator.go:76-81` stamps `Basis: measured` for `skill-validator/o200k_base` — honest as a static count, but o200k_base is not the target's billing tokenizer and a static count is not tier-2 harness usage [O] |
| **Lift** | Δ tok/task, Δ pass rate, precision/recall, N, MDE | paired arms A/B/C; BCa 95% CI on paired means; McNemar on success; precision on the should-not-fire stratum; **Arm-A floor `lift(C) > 0`, strictly** (`synthesis.md:186`) | tier 1 billed unavailable; tier 2 measured tokens / estimated $; tier 4 `chars/4` **ordinal only**; both sides stamped, never cross-tier (`synthesis.md:166-175`) |

Thresholds come from the shipping constant, never the plan doc (D11). Join key: frontmatter `name` (== directory, SK-I002) ↔ `skill.name` (`:178`). The plan's third number, SDY `(Waste$ + Risk$) × c ÷ E`, decomposes into Load and Lift so each side carries its own basis; it stays a **ranking**, not a claim (F8/F9), with a hotspot lane ordered by blast radius × permission scope jumping the queue (`:180`), Waste$/Risk$ always printed separately with the `C_f` sensitivity pair.

## 6. The audit contract per finding

| Field | Deterministic | LLM-drafted | Made measurable by |
|---|---|---|---|
| **problem** | `Message` + `Evidence` + file:line + `Fingerprint` (`sa:baseline.go:17-24`) | never | the fingerprint is the identity a before/after run compares |
| **background** | `Background` + `Origin` + the rule's own `Fires` fixture as the illustrated case | may append one paragraph tagged `llm-drafted`; G6 adds, never deletes (`recommendation.md:61`) | n/a |
| **proposed solution** | `Remediation` + `Effort` | may draft the concrete edit, tagged; never replaces the deterministic text | effort is an uncalibrated placeholder (F4) |
| **expected outcome** | `Expect{Rerun, Metric, Op, Target}` — `Rerun ∈ {silent, silent_or_tagged, silent_or_low}` ∧ an optional metric over a **closed** enum: `always_loaded_tokens, body_tokens, body_lines, dead_refs, cycles, activation_precision, lift_tokens, pass_rate` | never | `skillgate verify --before r1.json --after r2.json` → **confirmed / regressed / unconfirmed(reason)** |

**Why a re-run, not a fingerprint.** The fingerprint hashes the evidence string, so `finding_absent: <fingerprint>` reports "met" when an author merely **rewords the matched text** and leaves the problem in place [O]. The predicate is *"this rule is silent at this path"*. `unconfirmed` reasons: rule silent but a new finding at the same path · below funded N (prints achieved MDE; N≈12/54/216, `synthesis.md:188`) · basis mismatch between the reports (F1, `:196`) · unmeasured at tier N — which reports "unmeasured", **never "met"**. Dynamic outcomes need all four conditions including the strict `lift(C) > 0` floor; `lift ≥ 0` admits the zero case the condition exists to exclude. Examples: T013 → `silent`; I004 → `body_lines ≤ 500`; a description rewrite → `activation_precision` not down ∧ `lift_tokens > 0` with CI, else MDE.

## 7. The effective-spend knob and "are skills effective anyhow"

**One knob object** in the policy profile, every field an uncalibrated labelled placeholder: `k = {k_L, k_A, r_max}` — `k_L` scales the listing budget (0.01 × context window), `k_A` the per-activation ceiling (8,000 tok / 500 lines), `r_max` the **spend tolerance**, the maximum tokens-per-success ratio against the no-skill arm. **Metric:** `ES_ratio = tokens_per_success(with) ÷ tokens_per_success(without)` on the **paired arm**. *ES refuses to compute when its inputs carry different basis stamps*, as F1 refuses a cross-source delta (`synthesis.md:166-175`); on mixed bases it prints **"unmeasured at tier N"**.

**Inputs to `ES_ratio` — three, each with its source, none in hand.** (1) **Verifier verdict per task**: a deterministic pass/fail hook (R-AT-09, `spec.md:143`; metric row `L2-dynamic-report.md:66`), unbuilt — no grading hook and no price table anywhere in the package (D-9, `L2:237`; fix-order item 6, `:248`) — and neither reference scanner has one: no skillspector analyzer emits a verdict or cost (its `estimate_tokens` calls size LLM prompts and batches — `ss:mcp_tool_poisoning.py:1121,1293`, `ss:llm_analyzer_base.py:726,741,804`), and skillgate's `ExitPass` is a lint-gate exit, not a task grader (`sa:gate.go:150-171`) [O]. (2) **Per-task, per-arm token count carrying a tier stamp** (`synthesis.md:166-175`): on Cursor the hook spool has exactly one real token number, `preCompact.context_tokens`, and it fires only when a session compacts (`cp:cursor-telemetry-surfaces.md:51,56-58`); per-skill attribution (`skill.name` on `claude_code.token.usage`) exists only on Claude Code (R-AT-04, `spec.md:138`; `L2:65,227`) [O]. (3) **An Arm-A (no-skill) run** for the denominator (R-AT-05, `spec.md:139`; `L2:19-27`): `sa:profiler/experiment.go:191` hardcodes `Steps = []ExperimentStep{baseStep, candStep}`, so Arm A cannot be expressed (D-8, `L2:235`, which cites `:188` — stale at head) [O]. Until the three-arm harness plus verifier exist, ES prints `unmeasured at tier N` for every skill — the same refusal path as a mixed-basis delta. **The only static proxy** is always-on tokens × turn count: always-on from skillgate Pack E (`sa:budget.go:8-21`, stamped `skillgate/exact-chars` / `measured` at `sa:gate.go:85`), turns from `preCompact.message_count` (`cursor-telemetry-surfaces.md:51`) [O]. That product is tier 4 (`synthesis.md:173`) and an **ordinal signal only** — enough to rank two skills in a paired A/B with comparable turn counts, never an absolute or dollar claim (`:82-84`) [O]. It carries no success term, so it ranks listing cost, not effectiveness, and never feeds `ES_ratio` [I]. Sub-file token attribution lands only when a measured counter covers the item; none does (`sa:budget.go:21-22`) [O]. (The gap statement's `L2:64-66` should read `:65-67`; line 64 is tool-call success rate [O].)

**Six states, with a token-direction axis.** Three buckets (proven positive / no-difference / negative) carry no direction, so `jetson-optimize-memory` (−76.9%) and `cuopt-install` (+120.3%) — the cases that prove bimodality — both land in "proven positive", collapsing the distinction the source demands: "token delta must be reported alongside correctness, never alone" (`cp:skill-profiling-state-of-the-art.md:212-218`) [O].

| State | Condition |
|---|---|
| saves and holds accuracy | Δtok < 0, ΔS ≥ 0, four conditions met |
| costs more and raises accuracy, **within tolerance** | Δtok > 0, ΔS > 0, `ES_ratio ≤ r_max` — still a good skill |
| costs more and raises accuracy, **over tolerance** | Δtok > 0, ΔS > 0, `ES_ratio > r_max` |
| **delete this skill** | ΔS ≤ 0 or `lift(C) ≤ 0` — costs more with no accuracy change, or hurts (F10, `synthesis.md:205`) |
| saves but loses recall | Δtok < 0 with activation precision/recall down (F6, `:201`) |
| no difference at MDE = X · unmeasured at tier N | below funded N, or static-only |

Estate roll-up = a **count per state plus a lift histogram, never a mean**, the two extremes named: the distribution is bimodal and 27.2% of skills have negative lift (`synthesis.md:186`). Progressive disclosure buys *accuracy*, not cheapness — it "raises resource touches ~3× … but does not reduce total cost per successful task" (`SOTA:237-239`). The $1.31 vs $1.28 figure is **not cited**: the source labels it "From secondary summarization (lower confidence — not confirmed against the PDF body)" (`SOTA:233-235`) [O]. F2 applies: `sum(tokens) by skill.name` is attribution, not causation (`:197`). `refusals[]` carries F1-F14 verbatim.

## 8. Migration from the 28 live Go rules

**Prerequisites — before any rule migrates, or every baseline is unstable.** `refgraph.go:259` seeds Tarjan from `for v := range graph`, and G002 emits `File: scc[0]`, `Evidence: strings.Join(scc, " <-> ")` (`:209-211`) in that order, so a cycle finding's fingerprint flips between runs; `tripwire_b.go:207-209` ranges over decoded JSON maps for T015 [O]. Fix: sort graph keys before DFS, sort each SCC, sort map keys before emission, sort findings `(rule, file, line, evidence)` before fingerprinting — **plus a permanent repeated-run probe test**, not a one-time sort. ½ day.

| Rules | Today | Target |
|---|---|---|
| T002 T004 T007 T008 T010 T011 T016 | `lineMatches` over a file class (`tripwire_a.go:117-122,169-176,216-232`; `tripwire_b.go:51-64,229-234`) | `Lines(re…)` + `Scope` — mechanical |
| T005 · T009 · T012 · T020 | hand-coded co-occurrence (`tripwire_a.go:183-193,239-251`; `tripwire_b.go:74-80,337-346`) | `SameFile` · `Any(SameLine,SameFile)` · `Any(Lines,All(FileHas,Lines))` · `SameLine(…)` |
| T001 · T003 · T006 | rune loop · frontmatter mixed-script · masked evidence (`tripwire_a.go:102-127,136-179,223-241`) | `Span(P(1,class))` · `Frontmatter(…)`+`Native` · `Lines(…).Mask("***")` |
| T013 T014 · T015 T017 T018 · T019 G001 G002 · I001-I005 | `scanBundle` · per-rule JSON parse · path + refgraph · one function with five inline findings (`icm.go:27-149`) | `Scope: Bundle/Configs` parsed once into `Ctx`; `Native` over `Ctx.Refs`; five `Rule` values, I005's missing counter a named skip |
| **H002** | `harnessFrontmatterCheck` — Pack D, `Info`, `EffortMinutes: 0`, one finding per Claude-Code-only frontmatter key Cursor ignores (`frontmatter.go:159-186`; key list `:158`) [O] | `Frontmatter(allowed-tools, disable-model-invocation, context, when_to_use)` + `Native`. **The 28th live rule**: `grep -oh '"SK-[A-Z0-9]*"' skillgate/*.go \| sort -u` yields 29 strings, one of which (`"SK-T"`) is a prefix constant — so the mandate's "27" and this document's earlier count were both off by one [O]. `Effort: 0` and no fixture, so §4.2's `Effort>0` and ≥1-fires/≥1-quiet rules need an explicit `Info` exemption or H002 needs both |
| Catalogue + SARIF + tests | `rules[]` from fired findings only (`sarif.go:91-101`) | generated catalogue → `rules[]` with help/effort/fingerprint |

**Test-coverage gap at head** (mandate:33 wants one positive and one negative per rule): positives cover **23 of 28** SK-* IDs — `firesTable` T001-T020 (`tripwire_test.go:12-36`), `TestRefGraphDanglingAndCycles` pins G001/G002 (`:157-187`), `TestSameNameCollisionFires` pins SK-I002 both ways (`gate_test.go:182-218`). Negatives cover ≥6 rules — `noFireTable` T019, T012, T018, T017 (×2) (`:95-121`), T008 (`:140-153`), I002, plus G001's exact-evidence assertion [O]. The gap is real — I001/I003/I004/I005 **and SK-H002** are untested (`grep H002 skillgate/*_test.go` is empty [O]) — but smaller than "I001-I005 have no tests".

Effort [I]: types + constructors + 13 pure rules 2-4 d; fence splitter, scopes, ICM split 2-3 d; catalogue generator + goldens 1 d; determinism ½ d; `DeclaredMarker`/`Continuity` views + offset mapping 1-2 d ⇒ **≈ 7-11 days**, rule-by-rule behind a report/v1 golden diff, with a baseline rewrite for any rule whose evidence string moves.

## 9. Decisions above my level

| # | Decision |
|---|---|
| D1 | **Canonical form**: typed Go values + generated catalogue vs a runtime-loaded JSON catalogue. Recommend Go — 7/15 need a predicate anyway; authoring-by-non-Go is not a requirement |
| D2 | Carry SS per-pattern confidence as `Weight` for Pack I (`recommendation.md:161`), or drop it |
| D3 | P5 category quirk: keep the `prompt-injection` tag verbatim (`harmful_content.py:92`) or correct it — affects SARIF tags and baselines |
| D4 | AS3 self-reference: reject the literal `current` prefix (verbatim) or the bundle's own name (fixes #500; semantic change) |
| D5 | **Licensing — the mechanism, not the question.** SkillSpector is **Apache-2.0** (`ss:pyproject.toml:10`), skill-architect **MIT** (`LICENSE:1`), **no NOTICE file exists today** [O]; the plan already ruled that vendoring needs LICENSE + NOTICE + change notices while shelling out to an unmodified binary creates none (`recommendation.md:121`). Copying ~80 pattern *texts* is the vendoring case: Apache-2.0 §4 attribution/NOTICE retention is a **shipping obligation**, discharged by `Rule.Origin` generating `THIRD_PARTY_NOTICES.md`. The catalogue must mark **re-expressed ideas** (thresholds, severity mapping, word lists, control flow) versus **copied text** (alternation bodies) |
| D6 | **Zero-dep vs the in-tree route.** `x/text/unicode/norm` is not importable [O] — but x/text is **BSD-3-Clause**, and copying the norm/bidi tables in-tree with attribution leaves `go.mod` with zero requires, which is what "zero external dependencies" means in Go practice. A decision, not an impossibility; it gates AE6's "strict subset", BH2's hostname leg, **and the `Skeleton`/`CompactLetter` views seven lexical rules' coverage depends on (§4.3)** |
| D7 | report/v1 additive fields: `summaries[]` (SSR-1), `expected_outcome`, `weight`, `tags`, rule `background`/`origin`; SARIF `partialFingerprints` |
| D8 | Ceded rules and the cap: does a `FallbackOf` sibling count against Pack B's 20 (`recommendation.md:114`)? |
| D9 | Policy profile owning thresholds, block set, pack enablement and `k = {k_L, k_A, r_max}`, monotone over the Go floor, never read from the bundle |
| D10 | Fallback duplication: when the external engine **is** present, does the `FallbackOf` sibling stay silent (coverage gap on engine error) or run and dedup (duplicate findings)? |
| D11 | Per-activation ceiling: shipping constant 8,000 tok (`sa:icm.go:22`) vs plan doc 5,000 (`recommendation.md:158,160`) — which is normative? |
| D12 | **Page budget — now a live call, and wider than at first pass.** Body ≈ **42.9k characters**: ~10.7 pages at 4,000 chars/page, ~12.3 at the 3,486 chars/page density used for the three candidates (62-80k, i.e. 18-23 pages). The first-pass gap fills (view layer, `ES_ratio` inputs, EA1 differential) added ~7k and the second pass (OH1 tail, I7/P5 ladder, D13, SK-H002) ~5k more, all of it cited evidence; prose tightening has recovered ~1k total. Holding seven pages now means cutting **§2 and §3 to counts-plus-verdict** (−6k) **and** moving the view layer plus I3/I7 into the annex (−5k). Name the yardstick and I will cut; I will not drop cited rows unasked |
| D13 | **ID namespace for ported rules — and the `never vendor` contradiction it hides.** The samples label native re-expressions `SS-*` and file them under `Pack: "A"`. Both collide. **(a) Namespace.** `sa:skillspector.go:115` emits `RuleID: "SS-" + is.ID` for every shell-out issue, so a native `SS-AS3` and a shell-out `SS-AS3` are one ID with two producers; their evidence differs (upstream `matched_text` vs our capture), so the fingerprints differ (`sa:baseline.go:17-24`) and **both survive as duplicates** while neither baseline entry suppresses the other. D10 covers only `FallbackOf` siblings, not whole-fit natives. **(b) Plan contradiction.** `cp:recommendation.md:125` defines Pack A as "all 113 SkillSpector IDs via G3 (*borrowed*)" — i.e. delivered by the shell-out — and `:112` rules SkillSpector "integrate — shell out, opt-in, **never vendor**", with `:121` making vendoring the trigger for LICENSE + NOTICE + change notices. Copying pattern text **is** the vendoring case (D5), so filing a native re-expression under Pack A silently overturns `:112`. **Option 1 (recommended):** ported natives take their own namespace — `SK-P*` (`SK-P-AS3`) in a new Pack **P** ("ported, native") — with `Origin` recording the SkillSpector rule id plus Apache-2.0 / NVIDIA / `8421a2e`, generating the NOTICE entry (D5); `SS-*` then keeps one meaning, *came from the engine*, and the dedup key when both run is `(Origin upstream id, path, line)` with the native yielding to the engine. **Option 2:** `SS-*` stays shell-out-only and the re-expressions ship as format proof, never as rules. **Recommend Option 1** — it is the only one that lets §4.2's fallback contract cover the ceded surface, and it makes the overturn of `:112` explicit and countable ("N of the 113 move from *borrowed* to *ported*"). Either way the sample's `Pack: "A"` is wrong |

---

**Provenance.** SkillSpector clone at commit **8421a2e**. Workflow: five inventory readers → a fixed 15-rule sample → six landscape readers → three independent candidates → three lensed verifiers each → a two-judge panel → this merge on base **C**, with fold-ins from **A** (effectiveness-class vocabulary, the billed/measured/estimated basis column, the split-rule delegation contract) and **B** (`expected_outcome` re-run semantics, the six-state vocabulary with `spend_tolerance`), plus three gap fills: the derived-view layer, the `ES_ratio` input ledger, the EA1 evidence differential. Probes re-run this session under Go 1.27.1 (`scratchpad/mergeprobe`, `scratchpad/ea1diff`, `scratchpad/oh1diff`). Date: **2026-09-12**.

Second-pass fill 2026-09-12: OH1, P5, ID namespace, SK-H002.
