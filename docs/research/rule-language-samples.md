# Rule language — sample annex: the 15 rules re-expressed

Annex to **`rule-language.md`** (same cite legend: `ss:` = SkillSpector clone `src/skillspector/…` at commit **8421a2e**; `sa:` = `skill-architect/skillgate`; `cp:` = cursor-profiler `docs/research`; `probe:` = `scratchpad/mergeprobe`, Go 1.27.1, run this session). **[O]** observed, **[I]** inferred. This file exists because the main body is page-capped; it carries the worked re-expressions, the full RE2-hostile site lists and the probe transcript that the body cites.

## A. Probe transcript (Go 1.27.1, this session)

```
lb-python(\u)          COMPILE FAIL: error parsing regexp: invalid escape sequence: `\u`
lb-go(\x{})            compiles
bidi-ctl-python        COMPILE FAIL: error parsing regexp: invalid escape sequence: `\u`
bidi-ctl-go            compiles
P5-3 verbatim          compiles
  P5-3 across newlines: true
  OH1 "subprocess.run(output)"             optional=true  mandatory=false
  OH1 "subprocess.run(output, shell=True)" optional=true  mandatory=false
  OH1 "subprocess.run(cmd)output"          optional=false mandatory=true
  OH1 "subprocess.run([\"ls\",\"--output\"])" optional=false mandatory=false
  AE6 "i g n o r e"    ascii-class=true  pL-class=true
  AE6 "ｉ ｇ ｎ ｏ ｒ ｅ"    ascii-class=false pL-class=true
  AE6 "и г н о р е"    ascii-class=false pL-class=true
{0,1000}               compiles
{0,1001}               COMPILE FAIL: invalid repeat count: `{0,1001}`
{0,4096}               COMPILE FAIL: invalid repeat count: `{0,4096}`
python \Z              COMPILE FAIL: invalid escape sequence: `\Z`
go \z                  compiles
Zs/Zl/Zp class         compiles
```

EA1 ordered-alternation fidelity (`probe:mergeprobe/ea1`), pattern
`(?im)(?:tools?|permissions?)\s*:[ \t]*\[?[ \t]*['"]?\*(?:['"][ \t]*\]?|[ \t]*\]|[^*\w\r\n]|$)`:

```
"tools: [\"*\"]"         match=true  evidence="tools: [\"*\"]"
"tools: '*'"             match=true  evidence="tools: '*'"
"tools: *"               match=true  evidence="tools: *"
"tools: *\nnext"         match=true  evidence="tools: *"
"permissions: [ '*' ]"   match=true  evidence="permissions: [ '*' ]"
"tools: [*]"             match=true  evidence="tools: [*]"
"tools: *,"              match=true  evidence="tools: *,"      <- the one drifting case
"tools: **"              match=false
"permissions: *_read"    match=false
```

**Superseded.** That nine-form run carries no trailing-whitespace, NBSP or CRLF case; the nineteen-form differential in the body (§4.4 I3) shows **seven** drifts and one missed detection (`tools: *\r\n`), and rules the ordered alternation out in favour of keeping upstream's tail `['"]?[ \t]*\]?` verbatim, capturing `(\*)` and rejecting when the text at group 1's end matches `[*\p{L}\p{N}_]`. The block is kept because the body cites it as the artefact it corrects.

**OH1 evidence differential** (`probe:oh1diff`, Go 1.27.1 + python3, this session). Upstream `ss:static_patterns_output_handling.py:66-72` — `\bsubprocess\s*\.\s*(?:CALLS)\s*\([^)]{0,1000}?(?<![-\w'\"])(?:NAMES)(?!\w)` under `IGNORECASE|VERBOSE`, evidence `match.group(0)[:200]` (`:544`) — against the **consuming-tail** form `(?:[^\w]|\z)` and the **corrected** form (pattern ends at `(?P<name>…)`, post-filtered by `.RejectAfter("name", [\p{L}\p{N}_])`, leading class widened to `[^-'\")\p{L}\p{N}_]`):

| input | upstream (Python) | consuming tail | corrected |
|---|---|---|---|
| `subprocess.run(output)` | `subprocess.run(output` | `subprocess.run(output)` | `subprocess.run(output` |
| `subprocess.run(output, shell=True)` | `subprocess.run(output` | `subprocess.run(output,` | `subprocess.run(output` |
| `subprocess.check_output(["sh","-c",result])` | `…"-c",result` | `…"-c",result]` | `…"-c",result` |
| `subprocess.run( output)` | `subprocess.run( output` | `subprocess.run( output)` | `subprocess.run( output` |
| `subprocess.run(output)x` | `subprocess.run(output` | `subprocess.run(output)` | `subprocess.run(output` |
| `x = subprocess.run(output` (EOF) | `subprocess.run(output` | `subprocess.run(output` | `subprocess.run(output` |
| `subprocess.run(outputü)` | **silent** | `subprocess.run(outputü` — **FP** | **silent** |
| `subprocess.run(üoutput)` | **silent** | `subprocess.run(üoutput)` — **FP** | **silent** |
| `subprocess.run(cmd)output` | silent | silent | silent |
| `subprocess.run(["ls","--output"])` | silent | silent | silent |

Five of the six matching forms drift by exactly one evidence byte — every one of them baseline-visible (`sa:baseline.go:17-24`) — and two more are false positives that Go's ASCII `\w` manufactures on either side of the name. The corrected form is byte-identical on all six and agrees with upstream on all ten rows, including the four I1 cases (rows 1-2 and the last two). `.TrimGroup` on the consumed neighbour would repair the five evidence rows and neither false positive.

**P5 lane 2 ladder** (`python3`, the three predicates transcribed verbatim from `ss:static_patterns_harmful_content.py:136-196`). Input: a five-line recipe whose line 3 is `Add hemlock to the pot, then stir.` and line 4 `WARNING: do not serve this to guests; it is toxic.` — instructional via `add `/`stir `/`step `/`prepare` in the ±250-char window, educational via `warning`/`do not`/`danger` in the ±5-line context, warning via the same:

```
instructional=True  educational=True  warning=True
upstream if/elif ladder   -> 0.95  emits: True
flattened last-wins chain -> 0.2   emits: False
```

Python side, MP2 (`python3`, this session): `re.search(r'(.{2,20}?)\1{20,}', 'abAB'*11, re.I|re.M)` is **True** at 44 chars; without `re.I` it is **False**; with a `\n` inside the unit it is False (no DOTALL upstream) [O].

## B. Full RE2-hostile site lists [O grep over `ss:nodes/analyzers`, 33 files / 22,311 lines]

**Lookaround / backreference / atomic — 24 lines in 11 files** (`grep -nE '\(\?=|\(\?!|\(\?<=|\(\?<!|\(\?>|\(\?P=|\\[1-9]'`):

| File | Lines |
|---|---|
| `bundled_execution_surface.py` | 183, 808 |
| `mcp_tool_poisoning.py` | 689 |
| `static_patterns_agent_snooping.py` | 114 |
| `static_patterns_excessive_agency.py` | 47 |
| `static_patterns_memory_poisoning.py` | 84 (`\1`), 167 |
| `static_patterns_output_handling.py` | 70 |
| `static_patterns_privilege_escalation.py` | 59, 97, 129 (`(?P=quote)`), 184, 185, 295, 367, 375 |
| `static_patterns_rogue_agent.py` | 129 |
| `static_patterns_supply_chain.py` | 154, 823, 1268 |
| `static_patterns_system_prompt_leakage.py` | 165 |
| `static_patterns_tool_misuse.py` | 47 (`(?>…)`), 76, 269 (`(?P=quote)`) |

Others: raw-string `\uXXXX` at `common.py:31` (`LOGICAL_LINE_BREAK`, imported by `artifact_integrity`, `memory_poisoning`, `privilege_escalation`, `prompt_injection`, `system_prompt_leakage`, `static_runner`, and spliced as `_LOGICAL_BREAK` at `system_prompt_leakage.py:155`) and `static_patterns_prompt_injection.py:83,89` · counted repeats >1000 at `tool_misuse.py:119` (`{0,8192}` via `_ROOT_GLOB_COMMAND_CHARS`) and `mcp_least_privilege.py:84,85,94,95` (`{0,4096}`), while `privilege_escalation.py:125` sits at `{0,1000}` and compiles · `\Z` at `memory_poisoning.py:167`, `system_prompt_leakage.py:165`, `supply_chain.py:823` · `re.VERBOSE` at `anti_refusal.py:194`, `output_handling.py:72` · `\p{bc=…}` at `bundled_execution_surface.py:51-66` behind `import regex` (`:19`). The 11th `regex.` grep hit, `static_patterns_deserialization.py:131`, is a loop variable unpacked from `_COMPILED[language]` at `:130` — not the third-party module.

## C. The 15 rules

Severity maps CRITICAL→Blocker as `sa:skillspector.go:131-147`; SS per-pattern confidence is carried as `Weight` (Pack I only); effort values are uncalibrated placeholders (F4). Seven of these are in the plan's ceded set (`cp:recommendation.md:36`) — the sample proves the format, it does not propose adding them to Pack B. **The IDs below are shown in the upstream `SS-*` spelling, and AS3 in `Pack: "A"`, pending D13**: `sa:skillspector.go:115` already emits `"SS-" + is.ID` for every shell-out finding, so a native and a shell-out `SS-AS3` are one ID with two producers and two fingerprints, and Pack A is defined as the *borrowed* 113 (`cp:recommendation.md:125`) under a `never vendor` ruling (`:112`). The body recommends `SK-P*` in a new Pack **P**, with `Origin` carrying the SkillSpector rule id and the Apache-2.0/NVIDIA attribution that generates the NOTICE entry (D5).

### 1. AS3 — fits whole
Source `ss:static_patterns_agent_snooping.py:95-120` (7 patterns), `:114` `(?!CURRENT)`, `:167-181` emit MEDIUM.

```go
var AS3 = Rule{ID: "SS-AS3", Pack: "A", Quality: "security", Severity: Medium, Effort: 15,
  Message: "Skill enumeration: reads or lists other installed skills", Scope: AllText,
  Origin: []string{"ss:static_patterns_agent_snooping.py:95-120 (pattern text copied)"},
  Remediation: "Restrict file access to this skill's own directory; remove enumeration of .claude/.codex/.gemini skill roots.",
  Expect: Expect{Rerun: Silent},
  Detect: Span(
    P(0.9,  `(?i)(?:os\.listdir|os\.scandir|glob\.glob|Path\.iterdir)\s*\([^)]*\.(?:claude|codex|gemini)/skills?`),
    P(0.85, `(?i)(?:ls|find|dir)\s+[^|&;\n]*\.(?:claude|codex|gemini)/skills?`),
    P(0.85, `(?i)open\s*\(\s*['"][^'"]*SKILL\.md['"].*?\bother\b`),
    P(0.85, `(?i)(?:read|access|inspect|enumerate)\s+(?:all\s+)?(?:installed|available|other)\s+skills?(?:\s+in\s+(?:the\s+)?(?:skills?|agent)\s+(?:directory|folder))?`),
    P(0.85, `(?i)(?:list|discover|find|enumerate|identify)\s+(?:all\s+)?(?:other|installed|available)\s+(?:skills?|agents?|tools?)\s+(?:in\s+)?(?:the\s+)?(?:\.(?:claude|codex|gemini)|\$HOME)`),
    P(0.8,  `(?i)skills?/(?P<seg>[A-Za-z][A-Za-z0-9_-]+)/SKILL\.md`).       // was (?!CURRENT), :114
      Reject(func(h Hit) bool { return strings.HasPrefix(strings.ToLower(h.Group("seg")), "current") }),
    P(0.9,  `(?i)(?:read|access|load)\s+(?:the\s+)?(?:SKILL|skill)\.md\s+(?:file\s+)?(?:of|from|for)\s+(?:another|other|different|all)\s+(?:skill|agent|tool)`)),
  Tests: Tests{
    Fires: {{"SKILL.md": "Read skills/Other/SKILL.md first."}},
    Quiet: {{"SKILL.md": "See skills/CURRENT/SKILL.md and skills/currently-used/SKILL.md."}}}}
```

Fidelity: match-then-reject reproduces the IGNORECASE lookahead exactly, including `skills/current-skill/…` [O]. The reported FP (#500, self-reference) needs `h.Group("seg") == c.SkillName()` — a semantic change, D4.

### 2. AR2 — fits whole (regex + four triage lanes)
Source `ss:static_patterns_anti_refusal.py:71-90` (6 patterns), `:183-195` schema-field regex (`(?x)` flattened), `:269-309` clause bounds and context rewrite, `:361-393` triage, `:401-461` emit + dedup.

```go
var AR2 = Rule{ID: "SS-AR2", Severity: High, Effort: 15, Message: "Anti-refusal statement: disclaimer or caveat suppression",
  Scope: AllText, Expect: Expect{Rerun: SilentOrTagged},
  Remediation: "Delete instructions that forbid warnings, disclaimers or moralizing; if the text means a JSON field, name it explicitly.",
  Detect: Span(
    P(0.8,  `(?i)\b(?:do\s+not|don'?t|never|no)\s+(?:include|add|give|provide|use|append)?\s*(?:any\s+)?(?:warnings?|disclaimers?|caveats?)\b`),
    P(0.8,  `(?i)\bwithout\s+(?:any\s+)?(?:warnings?|disclaimers?|caveats?|moralizing|moralising|judg(?:e?ment|ing))\b`),
    P(0.85, `(?i)\b(?:do\s+not|don'?t|never)\s+(?:lecture|moralize|moralise|preach|judge|warn)\b`),
    P(0.8,  `(?i)\b(?:skip|omit|leave\s+out|drop)\s+(?:the\s+)?(?:disclaimer|warning|ethics|safety)\b`),
    P(0.8,  `(?i)\b(?:no|without)\s+(?:ethical|moral)\s+(?:judg(?:e?ment)|considerations?|commentary|objections?)\b`),
    P(0.55, `(?i)\b(?:do\s+not|don'?t)\s+(?:apologize|apologise|say\s+sorry)\b`)).
    Tag(All(MatchIs(arLive), ClauseAfter(schemaFieldRE)), "contextual-triage", "likely-benign-context").   // :361-370, :183-195
    Tag(retrospectiveNarrativeClause, "contextual-triage", "likely-benign-context").                        // :348-358
    Tag(All(quotedMatch, defensiveFraming), "contextual-triage", "likely-benign-context").                  // :312-346
    Tag(All(codeExample, explicitExampleScaffold), "contextual-triage", "likely-benign-context").           // :264-266, SKILL.md exempt
    Evidence(DirectiveLine(`(?i)^\s*documentation\s*:\s*`)).                                                // :293-309
    Dedup(PerLineMax)}
```

Fidelity: tags never suppress (`:428-429`). SS emits a ±3-line `context` plus `matched_text`; our `Finding` carries `Evidence` only, so the rewritten directive line becomes the evidence — and evidence is fingerprinted (`sa:baseline.go:17-24`), so the label strip is baseline-visible and declared (I3).

### 3. EA1 — fits whole, evidence byte-identical
Source `ss:static_patterns_excessive_agency.py:46-69` (10 patterns), `:47` `\*(?!\*|\w)['"]?[ \t]*\]?`, `:374-388` emit MEDIUM.

```go
var EA1 = Rule{ID: "SS-EA1", Severity: Medium, Effort: 10, Message: "Unrestricted tool access", Scope: AllText,
  Remediation: "Replace the wildcard grant with an explicit allowed-tools list.", Expect: Expect{Rerun: Silent},
  Detect: Span(
    // SUPERSEDED by §4.4 I3 (nineteen-form differential): this ordered alternation drifts on seven forms and
    // silences `tools: *\r\n`. Ship instead the verbatim tail `['"]?[ \t]*\]?` with `(\*)` captured and
    // `.RejectAfter(1, `[*\p{L}\p{N}_]`)` standing in for `(?!\*|\w)`. Kept here as the artefact the body corrects.
    P(0.85, `(?im)(?:tools?|permissions?)\s*:[ \t]*\[?[ \t]*['"]?\*(?:['"][ \t]*\]?|[ \t]*\]|[^*\w\r\n]|$)`),
    P(0.8,  `(?i)(?:allow|grant|enable)\s+(?:access\s+to\s+)?(?:all|any|every)\s+tools?`),
    P(0.75, `(?i)(?:no|without)\s+(?:tool|permission|access|capability)\s+(?:restrictions?|constraints?|limitations?)`),
    P(0.8,  `(?i)(?:call|invoke|use|execute)\s+(?:any|all|every)\s+(?:available\s+)?tools?`),
    P(0.85, `(?i)(?:unrestricted|unlimited|unconstrained)\s+(?:tool|function|api)\s+(?:access|use|calls?)`),
    P(0.75, `(?i)(?:can|may|should)\s+(?:freely|always)\s+(?:use|call|invoke)\s+(?:any|all)\s+(?:tools?|functions?|apis?)`),
    P(0.7,  `(?i)tools?\s*:\s*\[\s*['"]shell['"].*?['"](?:file_write|network|http)['"]`),
    P(0.85, `(?i)(?:grant|give)\s+(?:full|complete|total)\s+(?:tool|function|api)\s+access`),
    P(0.8,  `(?i)(?:execute|run)\s+(?:arbitrary|any)\s+(?:commands?|code|scripts?)`),
    P(0.7,  `(?i)(?:no\s+)?(?:tool\s+)?(?:allow|block|deny)\s*(?:list|listing)\s*(?:is\s+)?(?:empty|disabled|off)`))}
```

Fidelity: **the "one drifting case" claim is withdrawn.** The body's nineteen-form differential (§4.4 I3) finds seven drifts — every trailing-`[ \t]` run collapses to one char, NBSP is consumed, `tools: *ü` fires where Python is silent, and `tools: *\r\n` is a **missed detection** — so the alternation above needs the `.RejectAfter(1, …)` replacement, not a baseline rewrite. See §A.

### 4. P5 — fits whole, **both lanes** (lane 2 only as an *ordered* ladder)
Source `ss:static_patterns_harmful_content.py:35-51` substances, `:52-81` `DANGEROUS_ACTIONS` (10), `:82` `SUBSTANCE_PATTERNS` (15), `:92` category tag, `:96` `IGNORECASE|MULTILINE|DOTALL`, `:109-132` substance loop, `:198-214` dedup.

```go
var subst = `(cyanide|arsenic|ricin|botulinum|strychnine|thallium|polonium|sarin|vx gas|mustard gas|anthrax|deadly nightshade|aconite|hemlock|oleander)`
var P5 = Rule{ID: "SS-P5", Severity: Blocker, Effort: 20, Message: "Harmful content injection", Scope: AllText,
  Tags: []string{"prompt-injection"},   // quirk preserved verbatim: :92 tags PROMPT_INJECTION, not harmful-content — D3
  Expect: Expect{Rerun: Silent},
  Detect: Any(
    // Lane 1 — DANGEROUS_ACTIONS, verbatim. Branch 3 keeps its unbounded lazy spans: RE2 is linear and caps
    // only *counted* repeats, so no window is needed (probe: compiles, matches across newlines).
    Span(
      P(0.95, `(?is)add\s+(?:a\s+)?(?:dash|pinch|bit|drop|amount)\s+of\s+`+subst),
      P(0.95, `(?is)(?:include|incorporate|mix\s+in|stir\s+in)\s+(?:some\s+)?`+subst),
      P(0.98, `(?is)for\s+(?:every|each|all)\s+(?:recipe|dish|meal|food).*?add.*?`+subst),
      P(0.9,  `(?is)mix\s+bleach\s+(?:and|with)\s+ammonia`), P(0.9, `(?is)mix\s+ammonia\s+(?:and|with)\s+bleach`),
      P(0.95, `(?is)(?:drink|consume|ingest)\s+(?:the\s+)?(?:bleach|antifreeze|drain\s+cleaner)`),
      P(0.95, `(?is)(?:you\s+should|try\s+to|please)\s+(?:hurt|harm|injure)\s+yourself`),
      P(0.9,  `(?is)(?:cut|burn|starve)\s+yourself`),
      P(0.95, `(?is)(?:instructions?\s+(?:for|to|on)\s+)?(?:make|build|create)\s+(?:a\s+)?(?:bomb|explosive|weapon)`),
      P(0.95, `(?is)(?:how\s+to\s+)?(?:poison|kill|murder)\s+(?:someone|a\s+person|people)`)),
    // Lane 2 — SUBSTANCE_PATTERNS (:82), IGNORECASE only, base 0.7, context-adjusted, emit gate >= 0.5 (:109-132).
    Span(P(0.7, `(?i)\b`+subst+`\b`)).               // base_confidence 0.7 (:113)
      // ORDERED. Upstream is an if/elif/elif (:114-119), NOT three independent adjustments: the
      // first arm that holds sets the confidence and the rest never run (I7). A chained
      // .Weight().Weight().Weight() is last-wins, and the marker lists deliberately overlap —
      // "warning", "danger", "never", "do not" appear in BOTH the educational list (:157-180)
      // and the warning list (:182-196) — so an instructional passage that also reads as a
      // warning lands on 0.2 and the :120 gate DELETES a Blocker that upstream emits at 0.95 (§A).
      WeightLadder(
        W(CtxChars(250, instructionalRE), 0.95),      // :114-115 -> :136-154, ±250 CHARS of content
        W(CtxLines(5, educationalRE), 0.3),           // :116-117 -> :157-180, literal substring over
                                                      //   context.lower(), ±5 LINES (:112, :178)
        W(CtxLines(5, warningRE), 0.2)).              // :118-119 -> :182-196, re.search over the same
      Reject(WeightBelow(0.5))).                      // :120 `if confidence >= 0.5`
    Dedup(PerLineMax)}                                // :198-214
```

Fidelity: A's bounded rewrite (`.{0,400}?` + a 5-line window) was dropped — the probe shows the verbatim branch compiles and matches across newlines, so bounding only loses matches. Lane 2 is what makes "Hemlock tea: prepare by steeping…" fire upstream; omitting it silently halves the rule — and **flattening its ladder silently halves it again in the other direction**: the probe in §A shows the same passage scoring 0.95/emit upstream and 0.2/silent under a last-wins chain. Fit stays **whole**, but only with `WeightLadder`.

### 5. MP2 — fits whole (native run detector + two post-filters)
Source `ss:static_patterns_memory_poisoning.py:83-109` (7 patterns), `:84` backreference, `:194-219` layout filter, `:309-329` emit MEDIUM under `IGNORECASE|MULTILINE` (`:310`).

```go
var MP2 = Rule{ID: "SS-MP2", Severity: Medium, Effort: 15, Message: "Context window stuffing", Scope: AllText,
  Expect: Expect{Rerun: Silent},
  Detect: Any(
    // was (.{2,20}?)\1{20,} under re.IGNORECASE|re.MULTILINE: 21 total copies, case-folded, unit in
    // CODEPOINTS, and the unit cannot contain \n because the source passes no DOTALL.
    Repeat{MinUnit: 2, MaxUnit: 20, MinReps: 21, FoldCase: true, NoNewlineInUnit: true}.W(0.8),
    Span(
      P(0.85, `(?i)(?:repeat|duplicate|echo)\s+(?:this|the\s+following)\s+(?:\d{3,}|many|hundreds?|thousands?)\s+times?`),
      P(0.85, `(?i)(?:fill|pad|stuff|flood|saturate)\s+(?:the\s+)?(?:context|memory|input|prompt|window|buffer)`),
      P(0.8,  `(?i)(?:generate|produce|output|write)\s+(?:\d{4,}|thousands?\s+of|millions?\s+of)\s+(?:words?|characters?|tokens?|lines?)`),
      P(0.85, `(?i)(?:include|add|insert)\s+(?:enough|sufficient)\s+(?:text|content|padding|filler)\s+(?:to|until)\s+(?:fill|overflow|exhaust|push\s+out)`),
      P(0.9,  `(?i)(?:displace|push\s+out|overwrite|crowd\s+out|evict)\s+(?:the\s+)?(?:original|system|previous|existing|safety)\s+(?:instructions?|prompt|context|rules?)`),
      P(0.8,  `(?i)(?:exhaust|overflow|exceed)\s+(?:the\s+)?(?:context|token|memory)\s+(?:window|limit|budget|capacity)`))).
    Reject(layoutOnlySpan).      // :312 -> :194-219: len<=256; any letter/digit => false; all chars in "|-_=+" U+2500-259F => true
    Reject(singleCharNoSpace)}   // :314-316: <=1 distinct non-whitespace char and no ' '/'\t' in the span
```

Fidelity: a byte-exact period scan diverges from the source on the very construct it replaces — `'abAB'*11` fires upstream at 44 chars and needs a 4-char unit / 84 chars without folding (§A). Both post-filters attach to the run branch only; prose matches can never be layout-only, so that is semantically identical to the source's placement.

### 6. OH1 — partial (1 of 3 lanes native), with the fallback contract
Source `ss:static_patterns_output_handling.py:52-65` name tables and the 1,000-char bound, `:66-72` pattern (`(?x)`, lookbehind `:70`, lookahead), `:528-547` emit HIGH 0.85, `:648-653` dispatch, `:200-510` the JavaScript regexp-literal recognizer.

```go
var OH1fallback = Rule{ID: "SS-OH1-fallback", Severity: High, Effort: 20, FallbackOf: "SS-OH1-ast",
  Message: "Unvalidated output injection: model output reaches a subprocess sink",
  Remediation: "Never pass model output as a command; validate against an allow-list and use argv arrays.",
  Scope: AllTextExcept(".py"),   // :648-653 — .py goes to the AST lane; widens to include .py when SS-OH1-ast is skipped
  Expect: Expect{Rerun: Silent},
  Detect: Span(P(0.85, `(?i)\bsubprocess\s*\.\s*(?:Popen|call|check_call|check_output|getoutput|getstatusoutput|run)\s*\((?:[^)]{0,999}?[^-'")\p{L}\p{N}_])?(?P<name>answer|completion|generated|output|reply|response|result)`)).
    RejectAfter("name", `[\p{L}\p{N}_]`)}
// (?<![-\w'"]) => ONE OPTIONAL consumed boundary char that is neither ')' nor [-\w'"], or the name directly
// after '(' (I1). The class is spelled [^-'")\p{L}\p{N}_] and NOT [^-\w'")]: Python's \w is Unicode and Go's
// is ASCII, so the ASCII spelling treats 'ü' as a boundary and fires on `subprocess.run(üoutput)`, which is
// silent upstream. 999+1 preserves the 1,000-char window (:65).
// (?!\w) is ZERO-WIDTH (:70). The pattern must therefore END at the name and reject on the FOLLOWING char via
// .RejectAfter. The earlier consuming tail `(?:[^\w]|\z)` moved match.end(), so every evidence string gained
// the boundary byte — upstream `subprocess.run(output`, consuming form `subprocess.run(output)` — an I3
// violation on EVERY fire, and it also over-fired on `subprocess.run(outputü)`. Differential in §A.

var OH1ast = Rule{ID: "SS-OH1-ast", Severity: High, Effort: 20, Ceded: "external:skillspector", Scope: Scripts(".py"),
  Background: "alias-resolved AST walk over subprocess calls (:550-603); absent engine => checks_skipped[] + CAUTION cap (F13)"}
```

Fidelity: the optional **leading** group is the only form that reproduces all four I1 probe cases, and the **trailing** assertion must stay zero-width. As written above the ported lane is byte-identical to upstream's `match.group(0)[:200]` (`:544`) on all six matching forms and silent on all four the source is silent on (§A); the consuming-tail form it replaces drifted every evidence string by one byte and added two false positives on a High/0.85 rule. `.TrimGroup` would repair the bytes and not the over-fire, so `.RejectAfter` is the fix. Unported and named: the JS regexp-literal recognizer (`:200-510`) that suppresses OH1 pattern 1 on `.js`/`.ts` sources, and the file-type confidence bump. Fit remains **partial** — 1 of 3 lanes — with that one lane now declared byte-identical.

### 7. TM1 — partial (regex half native, shell lexer ceded)
Source `ss:static_patterns_tool_misuse.py:44-53` constants, `:107-164` (21 patterns), `:118-120` bounded rm clause (atomic `:47`, `{0,8192}` `:46,119`), `:250-302` safe idioms (`(?P=quote)` `:269`), `:2168-2207` emit.

```go
var tm1Clause = ShellClause{MaxAtoms: 8192}   // from the match end, atoms per _BOUNDED_SHELL_CLAUSE_ATOM (:48)
var TM1 = Rule{ID: "SS-TM1-regex", Severity: High, Effort: 25, FallbackOf: "SS-TM1-shell", Scope: AllText,
  Message: "Tool parameter abuse", Expect: Expect{Rerun: SilentOrLow},
  Remediation: "Remove shell=True, force/skip/insecure flags and destructive root paths; scope deletions to owned directories.",
  Detect: Any(
    Span(P(0.9, `(?i)\b(?:rm|del|erase)\b`)).Then(tm1Clause).
      Match(`(?i)^(?:\\\r?\n)*[ \t](?:[ \t]|\\\r?\n)*(?s:.*?)(?:\\)?-(?:rf|fr|r)(?:\\\r?\n)*[ \t](?:[ \t]|\\\r?\n)*[/~]`),  // :118-120, (?>…) dropped
    Span(
      P(0.8,  `(?i)subprocess\.\w+\s*\([^)]*shell\s*=\s*True`), P(0.8, `(?i)Popen\s*\([^)]*shell\s*=\s*True`),
      P(0.7,  `(?i)--force\s+(?:delete|remove|push|reset|clean)`),
      P(0.75, `(?i)--no-?(?:check|validate|confirm|protect|safe)\b`), P(0.7, `(?i)--skip-?(?:validation|verification|checks?|auth|tests?)`),
      P(0.65, `(?i)--allow-?(?:root|unrelated|unsafe)\b`),
      P(0.85, `(?i)\brm\b\s+(?:-[A-Za-z]+\s+)*(?:--\s+)?(?:['"][^'"]*/[^'"]*['"]|[^\s|;&>]*/[^\s|;&>]*)`),
      P(0.85, `(?i)\bshutil\.rmtree\s*\(\s*['"]\s*/`), P(0.8, `(?i)(?:chmod|chown)\s+[^|]*(?:777|666|a\+rwx)`),
      P(0.7,  `(?i)git\s+push\s+[^|]*--force`), P(0.65, `(?i)git\s+reset\s+--hard`), P(0.7, `(?i)git\s+clean\s+-[fd]+x`),
      P(0.75, `(?i)\bgit\s+(?:am|commit|merge|push)\b[^\n|]*--no-verify\b`),
      P(0.6,  `(?i)curl\s+[^|]*-k\b`), P(0.65, `(?i)curl\s+[^|]*--insecure\b`), P(0.65, `(?i)wget\s+[^|]*--no-check-certificate`),
      P(0.80, `(?i)\b(?:delete|remove)\s+['"]?/[^\s'"]{1,100}`),
      P(0.85, `(?i)(?:execute|query)\s*\(\s*f?['"].*?\{.*?\}.*?\b(?:DROP|DELETE|UPDATE|INSERT|ALTER|TRUNCATE)\b`),
      P(0.8,  `(?i)(?:set|pass|use)\s+(?:the\s+)?(?:parameter|argument|flag|option)\s+(?:to\s+)?(?:shell\s*=\s*True|--force|-rf)\b`))).
    Downgrade(Any(containerIdiom, dockerfileIdiom, cacheCleanup), Low, 0.15).   // :250-302, never suppressed
    Boost(FileClass(Script), 0.1).                                             // :2181-2187
    Dedup(PerLineMaxBy(NormalizedMatch))}                                      // :2188-2194

var TM1shell = Rule{ID: "SS-TM1-shell", Ceded: "external:skillspector", Severity: High, Effort: 25,
  Background: "the 1,100-line shell lexer half (:1166-1900): ANSI-C quoting, printf evaluation, brace expansion"}
```

`cacheCleanup` enumerates `(?P=quote)` as three alternatives (bare / `'` / `"`) plus the segment rule at `:292-302`. Deviation to declare: our `isScript` also covers `.ts/.rb/.pl/.ps1/.zsh/.mjs` where the source bumps only python/shell/javascript.

### 8. RA1 — fits whole
Source `ss:static_patterns_rogue_agent.py:43-84` (12 patterns), `:156-173` emit HIGH, `:192-210` same-clause negation.

```go
var RA1 = Rule{ID: "SS-RA1", Severity: High, Effort: 25, Message: "Self-modification", Scope: AllText,
  Remediation: "A skill must not rewrite its own SKILL.md, scripts or config, nor disable its safety checks.",
  Expect: Expect{Rerun: Silent},
  Detect: Span(
    P(0.95, `(?i)open\s*\(\s*__file__\s*,\s*['"]w`), P(0.95, `(?i)(?:Path|pathlib)\s*\(\s*__file__\s*\)\s*\.\s*write_text`),
    P(0.85, `(?i)(?:write|modify|edit|update|overwrite|patch)\s+(?:this\s+)?(?:skill(?:'s)?|SKILL\.md|skill\.md)`),
    P(0.9,  `(?i)(?:modify|edit|change|rewrite|update|alter)\s+(?:your\s+own|its\s+own|this\s+skill(?:'s)?)\s+(?:code|source|file|script|config|configuration|instructions?|rules?)`),
    P(0.9,  `(?i)(?:self[_-]?modify|self[_-]?update|self[_-]?rewrite|self[_-]?patch|self[_-]?evolve)`),
    P(0.85, `(?i)(?:rewrite|replace|overwrite)\s+(?:the\s+)?(?:current|existing|original)\s+(?:code|script|file|config|skill)`),
    P(0.85, `(?i)(?:append|add|inject|insert)\s+(?:code|instructions?|rules?)\s+(?:to|into)\s+(?:this|the\s+current|your\s+own)\s+(?:file|skill|script)`),
    P(0.8,  `(?i)(?:generate|create|write)\s+(?:new|updated|replacement)\s+(?:code|version)\s+(?:for|of)\s+(?:this|itself|the\s+skill)`),
    P(0.9,  `(?i)open\s*\(\s*['"](?:SKILL\.md|skill\.md)['"].*?['"]w`),
    P(0.75, `(?i)(?:update|modify|change|edit)\s+(?:the\s+)?(?:config|configuration|settings|parameters)\s+(?:of\s+)?(?:this|the\s+current)\s+(?:skill|agent|tool)`),
    P(0.9,  `(?i)(?:disable|remove|delete|bypass)\s+(?:the\s+)?(?:safety|security|guard|protection|constraint)\s+(?:check|rule|mechanism|feature)`),
    P(0.7,  `(?i)(?:comment\s+out|remove)\s+(?:the\s+)?(?:validation|verification|check|guard|assertion)`)).
    Reject(ClausePrefix(`(?i)(?:must\s+not|do\s+not|don't|never|should\s+not)\s+(?:(?:ever|again|directly|intentionally|explicitly|attempt\s+to|try\s+to)\s+){0,2}$`))}
```

`ClausePrefix`: on the hit's line, prefix = text after the last of `. ; :` before the hit; reject when the regex matches at the prefix end — equivalent to `:192-210` without a per-hit compile. Caps at two hedges, as the source does.

### 9. P6 — fits whole
Source `ss:static_patterns_system_prompt_leakage.py:43-76` (8 patterns), `:154-167` taxonomy (`\A`, `\Z`, `(?=\Z|…)`), `:169-191` directive/reference regexes, `:194-202` heading suppressor, `:205-262` bounded-line helpers, `:277-295` emit HIGH.

```go
const brk = `(?:\r\n|[\r\n\v\f\x1c-\x1e\x{85}\x{2028}\x{2029}])`   // common.py:31, Go spelling (probe: the \u form is rejected)
var P6 = Rule{ID: "SS-P6", Severity: High, Effort: 15, Message: "Direct prompt extraction", Scope: AllText,
  Expect: Expect{Rerun: Silent},
  Detect: Span(
    P(0.85, `(?i)(?:print|output|show|display|reveal|expose|return|echo)\s+(?:your\s+)?(?:full\s+)?(?:system\s+)?(?:prompt|instructions?|rules?|guidelines?|directives?)`),
    P(0.8,  `(?i)(?:what\s+(?:are|is)\s+your|tell\s+me\s+your)\s+(?:system\s+)?(?:prompt|instructions?|rules?|guidelines?|directives?)`),
    P(0.8,  `(?i)(?:output|print|display)\s+(?:the\s+)?(?:text|content|message)\s+(?:above|before)\s+(?:this|the\s+user)`),
    P(0.85, `(?i)(?:share|disclose|leak|give\s+me)\s+(?:your\s+)?(?:internal|hidden|system|secret)\s+(?:prompt|instructions?|rules?|config)`),
    P(0.8,  `(?i)(?:dump|extract|copy)\s+(?:the\s+)?(?:system\s+)?(?:prompt|message|instructions?)\s+(?:to|into|as)`),
    P(0.85, `(?i)(?:begin|start)\s+(?:your\s+)?(?:response|output|reply)\s+with\s+(?:your\s+)?(?:system\s+)?(?:prompt|instructions?)`),
    P(0.85, `(?i)(?:first|before\s+anything)\s+(?:output|print|show|display)\s+(?:your\s+)?(?:full\s+)?(?:system\s+)?(?:prompt|instructions?)`),
    P(0.9,  `(?i)(?:verbatim|exactly|word\s+for\s+word)\s+(?:repeat|output|print)\s+(?:your\s+)?(?:system\s+)?(?:prompt|instructions?)`)).
    Reject(All(FileIs(Markdown), MatchIs("Output Rules"), LineIs(`^## Output Rules \(Both Modes\)$`))).   // :194-202
    Reject(Native(benignPrintRulesTaxonomy))}                                                            // :205-262
```

`benignPrintRulesTaxonomy`: within ±256 chars find the CSS-taxonomy sentence whose `target` span equals the hit span; require candidate end == text end or a `brk` there (this replaces `(?=\Z|brk)`, which the source re-checks at `:243-247` anyway); then the neighbouring non-blank logical lines within 512 chars must be complete and must not match `_PRECEDING_DIRECTIVE` / `_NEXT_LINE_REFERENCE`.

### 10. P9 — fits whole (Native, stdlib only)
Source `ss:whitespace_padding.py:41-101` char sets and thresholds, `:207-221` fence flags, `:224-413` detectors, `:416-481` overlap suppression; consumer `static_patterns_prompt_injection.py:45-59` skip globs, `:344-374` severity map.

```go
var P9 = Rule{ID: "SS-P9", Severity: Varies, Effort: 10, Message: "Whitespace padding", Scope: AllText,
  Exclude: []string{"*.min.js", "*.min.css", "*.lock", "package-lock.json", "yarn.lock", "*.svg", "*.map"},
  Expect: Expect{Rerun: Silent},
  Detect: Native(WhitespacePadding{VerticalLines: 20, VerticalHighLines: 40, HorizontalChars: 80, BlockBytes: 2048,
    Ratio: 0.90, RatioMinBytes: 4096, RepeatChar: 512, RepeatLine: 64, FFFDDensityBail: 0.30,
    FenceSkip: FileIs(Markdown)})}
```

| Run kind | Detection | Severity / weight | Suppression |
|---|---|---|---|
| vertical | ≥20 blank lines | High 0.8 if followed by content ∧ ≥40 lines; else Medium; 0.6 when not followed | primary |
| horizontal | ≥80 padding chars in one logical line; md fences skipped | Medium 0.7 | primary |
| block / ratio | span >2048 bytes; padding >90% of files >4096 bytes | Low 0.4 | dropped when overlapping a primary |
| repetition | ≥512 identical non-padding chars; ≥64 identical non-blank lines | Medium 0.8 | dropped when overlapping a primary |

Padding chars = ASCII space, `\t\n\r\v\f`, `unicode.Zs/Zl/Zp`, {U+200B, 200C, 200D, 2060, FEFF}, U+0085, U+180E. Evidence = `run.summary`; multiple hits per file with distinct severities is why `Hit` carries `Severity`.

### 11. AE6 — partial (1 of 3 detectors), ID qualified
Source `ss:artifact_integrity.py:49-76` term lists, `:77-341` grammar tables and compiler, `:407-538` matcher, `:438-479` benign notation, `:605-641` emit — where `first_obfuscation_line = min(first_spacing_line, first_contextual_ignorable_line, first_targeted_instruction_line)` at `:641`; run spans from `ss:artifacts.py:196-202,468-508`.

```go
var AE6 = Rule{ID: "SS-AE6/letter-spacing", Severity: High, Effort: 20, Scope: LoadedText,
  Message: "Concealed instruction text (letter-spacing)", Expect: Expect{Rerun: Silent},
  Detect: Native(LetterSpacing{
    // The upstream prefilter compiles with re.UNICODE; Go's \w/\W are ASCII, so the candidate scan MUST be
    // \p{L}-based or a rune walk, or the fold table is unreachable on fullwidth and Cyrillic runs (probe §A).
    Runs:  ConcealedRuns{MinLetters: 6, LetterClass: UnicodeLetter},
    Fold:  FoldFullwidthAndConfusableThenCasefold,   // subset of NFKC + the SS confusable skeleton (:36) — D6
    Terms: []string{"bypass","disregard","ignore","instructions","jailbreak","override","previousinstructions",
      "restrictions","securityconstraints","silentlysend","sshkey","unfiltered","unrestricted","userdata"},
    Exact: []string{"accesstoken","apikey","credential","credentials","password","privatekey","secrettoken","systemprompt"},
    Grammar: fourCommandFamilies,           // prefix? action connector{0,3} target suffix? — compiles in RE2
    BenignNotation: benignNotation96,       // :438-479
    TrailingLetterRetry: true, EmitFirstOnly: true, Weight: 0.9})}
```

Fidelity: this is **one of three** AE6 detectors; the contextual-ignorable and targeted-instruction lanes are unported, which is why the ID is lane-qualified. The fold is a strict subset (missed detections only) unless D6 admits in-tree `x/text` tables.

### 12. SSR-1 — partial (channel fits, payload parser ceded)
Source `ss:structured_skill_roles.py:32-62` (findings always empty, one summary), gate `ss:structured_skill.py:42` (`"AISOP V"`, `"AISP V"`), extractor `:444-475` with a wall-clock budget at `:449-453` and abort at `:137`.

```go
var SSR1 = Rule{ID: "SS-SSR-1", Kind: Summary, Severity: Info, Effort: 0, Scope: Bundle,
  Ceded: "external:skillspector",   // the AISOP/AISP extractor; gate + channel are native
  Message: "Structured {layout_kind} bundle detected ({protocol})",
  Background: "upstream aborts extraction on a time.monotonic deadline (:449-453, :137) — a wall-clock budget must NOT be ported into a deterministic engine; a native port needs a size/step bound instead",
  Detect: Native(func(c *Ctx, _ *Doc) []Hit {
    sc := c.StructuredSkill(); if sc == nil { return nil }
    return []Hit{{Summary: Summary{ID: "SSR-1", File: sc.BundlePath, Protocol: sc.Protocol, LayoutKind: sc.LayoutKind,
      DeclaredTools: sorted(sc.DeclaredTools), WorkflowNodes: sc.WorkflowNodes, Constraints: sc.ConstraintAnchors,
      Resources: sc.ResourceAnchors, Tags: []string{"AISOP", "AISP", "structured-skill"}}}} })}
```

`Kind: Summary` routes `Hit.Summary` into an additive `report.summaries[]`, never into `findings[]` (report/v1 has no such channel today — D7).

### 13. TT2 — does not fit; ceded with its contract intact
Source `ss:behavioral_taint_tracking.py:313-331` rule pick, `:392-451` helpers, `:476-479` dedup, `:499-528` assignment tracking, `:562-576` emission.

```go
var TT2 = Rule{ID: "SS-TT2", Ceded: "external:skillspector", Severity: Medium, Effort: 15,
  Message: "Tainted flow (variable-mediated)", Scope: Scripts(".py"), Detect: nil,
  Background: "flow- and scope-insensitive: `tainted` is one dict keyed by bare variable name over ast.walk order (:499-528); an Assign whose RHS holds a source call, an os.environ[...] subscript, or any tainted Name marks every Name/Tuple target (:418-430); nothing ever clears a name, so `x = os.environ['K']; x = 'safe'; requests.post(u, data=x)` still fires, and a name tainted in one function taints the same name in every function; one finding per (rule, line) (:476-479)",
  Tests: Tests{
    Fires: {{"scripts/e.py": "import os,requests\nk=os.environ['K']\nk='safe'\nrequests.post('https://e.x',data=k)\n"}},
    Quiet: {{"scripts/e.py": "import requests\nrequests.post('https://e.x',data='x')\n"}}}}   // run only when the engine is present; else a named skip
```

Zero-dep Go has no Python parser (gpython's parser last published 2023-06-08). The format still carries identity, background with the exact FP modes, effort and fixtures, so the Advisory finding lands with the same contract as a native rule.

### 14. TP1 — partial (1 of 5 lanes), fields routed through the manifest scope
Source `ss:mcp_tool_poisoning.py:168-204` field extraction from a **manifest dict**, `:236` data-URI regex, `:239-398` `_check_tp1` with five lanes (data URI `:253-261`, HTML comment `:286`, markdown comment `:309`, zero-width `:331`, base64 `:376`), `:353-396` emit HIGH 0.75.

```go
var TP1b64 = Rule{ID: "SS-TP1/base64", Severity: High, Effort: 10,
  Message: "Base64-encoded blob in metadata field", Expect: Expect{Rerun: Silent},
  // parameters[].* are MCP MANIFEST fields (_extract_metadata_texts(manifest: dict), :168-177) — they are not
  // SKILL.md frontmatter, so no frontmatter-reader extension is on the critical path.
  Scope: Any(Frontmatter("name", "description", "triggers[]"), Configs(MCPManifests).Fields("tools[].name",
    "tools[].description", "tools[].parameters[].name", "tools[].parameters[].description")),
  Detect: Span(P(0.75, `[A-Za-z0-9+/]{50,}={0,2}`)).
    Reject(WithinAfter(`data:text/[^;]+;base64,`, 200)).   // :355-361
    Reject(func(h Hit) bool {                              // :363-372 decode-validity gate
      raw := h.Match(); pad := raw + strings.Repeat("=", (4-len(raw)%4)%4)
      b, err := base64.StdEncoding.DecodeString(pad); return err != nil || !utf8.Valid(b) })}
```

Fidelity: one lane of five — the ID says so, and the sample-fit tally counts lanes. Probe: a 116-char base64 of English decodes and is UTF-8 (kept); a 64-char hex SHA-256 decodes but is not UTF-8 (rejected) [O].

### 15. BH2 — partial (proof native; BiDi/punycode and WHATWG unmodeled, failing closed)
Source `ss:bundled_execution_surface.py:144-190` events and tables, `:245-311` host predicates, `:285-311` `_is_valid_literal_host`, `:445-455` `_parse_http_url`, `:741-790` sensitive paths, `:851-1035` argv helpers and proofs, `:1296-1322` strict parse, `:1422-1448` disable gate, `:1546-1565` finding, `:1650-1657` emission.

```go
var BH2 = Rule{ID: "SS-BH2", Severity: Blocker, Effort: 30,
  Message: "A bundled hook directly sends sensitive event or file content remotely.",
  Remediation: "Remove the remote transfer or require explicit, narrowly scoped user action.",
  Scope: Configs("hooks/hooks.json", ".claude/settings.json", ".claude/settings.local.json"),
  Expect: Expect{Rerun: Silent},
  Detect: Native(func(c *Ctx, d *Doc) []Hit {
    doc, ok := c.StrictJSON(d)                    // duplicate keys => partial, zero findings (:1296-1322)
    if !ok || c.HooksDisabled() { return nil }    // cross-file disableAllHooks gate (:1422-1448, :1650)
    var proofs []Proof
    for _, h := range hookDeclarations(doc, c.PreviousSettingsHookIDs(d)) {
      switch {
      case h.Type == "http" && sensitiveEvents[h.Event] && remoteHTTP(h):
        proofs = append(proofs, Proof{"event_http_body", "http"})                       // :1028-1032
      case h.Type == "command":
        if p := commandProof(h); p != nil { proofs = append(proofs, *p) }                // :995-1025
      }
    }
    if len(proofs) == 0 { return nil }            // closed proof: no proof, no finding
    return []Hit{{Line: 1, Weight: 0.99, Evidence: "document:" + d.Path,
      Tags: []string{"activation_state=conditional", "proof_status=closed",
        "proof_kinds=" + kinds(proofs), "transport_kinds=" + transports(proofs)}}} })}
```

| Piece | Go |
|---|---|
| 18 sensitive events (`:144-165`) | slice |
| curl / wget / scp / rsync / shell-form argv models (`:851-1025`) | direct port of the exactly-one-flag rules |
| sensitive absolute path (`:166-184`, `:741-769`) | `^/(?:Users/([^/]+)\|home/([^/]+)\|root)(/.*)$` + a Go check that the user segment is not `.`/`..` (replaces two lookaheads) |
| non-remote host (`:245-269`) | `netip.ParseAddr` + `Unmap()`; `inet_aton` shorthand (`127.1`, `0x7f000001`) needs ~40 lines of hand parsing |
| **BiDi hostname rule** (`:51-66`, `:386-431`) and **punycode validity** (`:285-311`) | **unmodeled ⇒ host is not remote ⇒ no proof ⇒ no finding**, which is upstream's own direction: `_parse_http_url` returns None unless `_hostname_satisfies_bidi_rule`, and `_is_valid_literal_host` returns False on invalid punycode. Treating `xn--` as remote would fire where the source is silent |
| URL parse (`:434-442`, pywhatwgurl) | `net/url` after normalizing `https:/{1,3}` → `https://`; WHATWG vs RFC 3986 edge cases differ [I] |

---

**Provenance.** Annex to `rule-language.md`; same merge: SkillSpector clone **8421a2e**; five inventory readers → a fixed 15-rule sample → six landscape readers → three candidates → three lensed verifiers per candidate → a two-judge panel → merge on base **C** with fold-ins from **A** and **B**. Probes re-run this session under Go 1.27.1 (`scratchpad/mergeprobe`). Date: **2026-09-12**.
