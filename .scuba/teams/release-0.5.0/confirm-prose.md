# Confirming pass, lens 2: REWRITTEN PROSE TRUTH — PR #22 @ `011defa`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so, verified by
> `ls`, and did not fall back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 13 findings, 3 hard-false sentences. 105 of 118 assertions TRUE.**

## Coverage

4/4 files, **41/41 diff hunks** (`CHANGELOG.md` +601, `RELEASE_NOTES.md` +32, `README.md`
+133/−50, `docs/profiler-spec.md` +88/−20 at `011defa` vs `10326b3`). **473 sentences** in
added/modified lines reduced to **118 distinct fact-assertions**, all 118 walked and
**118/118 driven by command** — a `011defa` build in its own worktree, per-file compiles of
the three drafts, both bash versions, both vendor references fetched live. **Second sweep**
re-walked all four diffs indexed by numeral and by claim-unit, adding 14 checks and **0
findings beyond the 13 below.**

---

## F1 · HIGH · REAL — "`getBool` was never declared in this repository at all" is false, at the ref the same sentence cites

`CHANGELOG.md:28`, `RELEASE_NOTES.md:17`

```
$ git show 0a83615:profiler/claude_code.go | grep -n "func getBool"
521:func getBool(m map[string]any, key string) bool {
$ git log --all --oneline --reverse -S"func getBool" -- profiler/claude_code.go
1e4e845 F03 Slices 2-4: Cursor, Codex, Devin profiler adapters
```

`getBool` is declared at **`0a83615:profiler/claude_code.go:521` — the exact ref the same
paragraph names as where its citations resolve** — and at `a3673b8`, `236152c`, `e0e2a52`,
`f13173e`, `1e4e845`. It was introduced by `1e4e845`, the commit that added the three
drafts, as the helper for `cursor.go:228`, and is used by nothing else. **This release's own
rewrite of `claude_code.go` removed it.**

The *explanation* is wrong too: v0.4.1's OTLP rewrite "does not account for" `getBool`
because `getBool` **postdates** that rewrite. The other four names were removed by v0.4.1;
**`getBool` was removed by this release's own lane.**

## F2 · HIGH · REAL — Cursor documents `cwd` on **four** events, not three; two documents say "only"

`CHANGELOG.md:442`, `RELEASE_NOTES.md:13`, `README.md:530`, `docs/profiler-spec.md:540`

Fetched `https://cursor.com/docs/agent/hooks`: `cwd` appears in four payload blocks —
`preToolUse`, `postToolUse`, **`postToolUseFailure`** (`{"tool_name":…,"tool_use_id":…,
"cwd":"/project","error_message":…,"failure_type":…}`), and `beforeShellExecution`.

All four documents omit `postToolUseFailure`. `CHANGELOG.md:442` and `RELEASE_NOTES.md:13`
add the word **"only"**, making it hard-false inside the sentence claiming the reference was
checked "field for field" — and `postToolUseFailure` is **one of the 21 events `hooks
install` registers.**

The rest of that block is exactly right: 21 events with no 22nd, `~/.cursor/hooks.json` as
User scope below Enterprise/Team/Project, `workspace_roots` and `user_email` on every
payload, Tab hooks a separate class, and `version` required — *"Config schema version. Must
be a positive integer (use `1`)"* quoted verbatim.

## F3 · HIGH · REAL — `CHANGELOG.md:285` names a registration string this release no longer writes, and contradicts `CHANGELOG.md:296-298`

```
$ sed -n '569p' profiler/cmd/main.go
	return self + " ingest --spool-dir " + profiler.SpoolDirIn(home) + " || true"
```

`CHANGELOG.md:285` says "path plus `ingest || true`" — the pre-`--spool-dir` build. It is
**the only place the changelog tells a reader what `doctor` looks for**, and two bullets
below `:296-298` states it correctly. `README.md:~568` and `docs/profiler-spec.md:~124` are
right; only the changelog is stale. **`profiler/doctor.go:321` carries the same stale string
in a code comment** — Lane 1's surface, noted for the fixer.

## F4 · MEDIUM · REAL — the new forward-promise guard is case-sensitive, so its own declared vocabulary escapes it

`tests/lib/forward-promise-check.sh:148`; claimed at `CHANGELOG.md:329-334`,
`RELEASE_NOTES.md:26`

```
rc=1  the caveat channel is deferred to 0.5.0.        -> [deferral verb]
rc=0  Deferred to 0.5.0.                              -> files=1 lines=1
rc=0  Tracked for 0.5.0.                              -> files=1 lines=1
rc=0  Reserved for 0.5.0.                             -> files=1 lines=1
rc=0  - **Deferred to 0.5.0**: a receiver subcommand.  -> files=1 lines=1
```

The regex is lowercase-only. A capitalised verb at a sentence start, a bullet start, or
inside a bolded lead-in passes silently — and **bolded bullet lead-ins are this
repository's dominant prose shape** (`**Also deferred:**`, `**Deferred, with reasons**`).

The earlier `scanFor` blind spot **is** closed (denominator is now `git ls-files`, and the
two markers in `tests/` and `skills/` would now be read). A new blind spot replaced it. The
script's own header at `:32-35` disclaims a phrasing *outside* its set — a capitalised verb
is **inside** the set and missed. All 18 fixture cases at `tests/test_skill.sh:772-789` are
lowercase, so no control catches it.

## F5 · MEDIUM · REAL — "162 of them" is not `git ls-files`

```
$ bash tests/lib/forward-promise-check.sh 0.5.0   -> files=162 lines=40026
$ git ls-files | wc -l                            -> 163
EMPTY: profiler/testdata/otlp/empty.json
```

The prose pairs "**every tracked file** (`git ls-files`, 162 of them)" — `git ls-files`
gives **163**; 162 is the *non-empty* count, because awk's `FNR==1` cannot fire for a
zero-byte file. The guard itself is sound and self-aware (`tests/test_skill.sh:811`
compares against non-empty tracked files and names `empty.json`). Only the published number
and `CHANGELOG.md:338`'s "compared for equality against the tree" are wrong. **Same class
the bullet was written to close.**

## F6 · MEDIUM · REAL — two of three fixture counts in the provenance bullet are wrong

`CHANGELOG.md:484`, `:488`, `RELEASE_NOTES.md:28`

- `:484` "two fixtures carry `slash_command` and `skill_tool`" → **six** carry
  `slash_command`, **three** carry `skill_tool`, union six.
- `:488` / `RELEASE_NOTES.md:28` "**one** fixture carries `"user"`" → **two** do.
- `skill.kind` = `"skill"` in two fixtures is **correct**.

Everything else in that bullet is TRUE: five new activation fixtures exactly (`comm -13`
against `v0.4.3`), `skill.name` labelled `[DOCS]`, both labelled paragraphs byte-identical
to `v0.4.3`, and neither `claude_code.skill_activated` nor
`invocation_trigger`/`skill.source`/`skill.kind` in either labelled paragraph.

## F7 · MEDIUM · REAL — a published document still contradicts itself on the Codex row. **This is the CoS's P6 ruling being incomplete.**

`CHANGELOG.md:782`, `RELEASE_NOTES.md:93` vs `README.md:808`

```
README:808 BYTE-IDENTICAL to v0.4.3:436     # "verified — codex-cli 0.153.4 ..."
CHANGELOG.md:782: `README.md` marks the Devin, Codex and Cursor install rows "not verified in this release".
RELEASE_NOTES.md:93: **The Devin, Codex and Cursor install rows are marked unverified.**
```

The **0.4.3 entries in both shipped documents** say README marks Codex unverified; README
marked it **verified** at v0.4.3 and still does. Those entries were false when written and
the freeze preserves them. The new 0.5.0 prose (`RELEASE_NOTES.md:25`, `CHANGELOG.md:552`)
asserts "this release stops republishing it" — **true of the 0.5.0 section, false of the
file**: the same two documents carry the wider version 757 and 68 lines down.

**Answering the mandate directly:** the *ledger* is **not** a published document —
`.gitignore:9` ignores `.scuba/`, and `git ls-files | grep -c '^\.scuba'` → 0 — so citing
`deferred-ledger.md:53` is accurate reporting, **not** a contradiction. The contradiction
that survives is in `CHANGELOG.md` and `RELEASE_NOTES.md`.

**Fix direction:** the release already has the pattern — "**Known past changes, recorded
late**", used for the skillscore drop. The same treatment, naming 0.4.3's own two entries
rather than only the ledger row, satisfies the invariant without editing a frozen section.

## F8 · LOW-MEDIUM · REAL — the spec's "remain unverified" list is stale against the reference the same paragraph says it re-read field by field

`docs/profiler-spec.md:516-521`. Two of four bullets are plainly documented by Cursor:
the `hooks`-keyed-by-event-name / `command`-string shape (the reference's own quickstart),
and `hook_event_name`/`cursor_version`/`conversation_id` (all three in the common field
block). `README.md:538-541`'s parallel "**Still documentary**" bullet correctly lists only
the stdin-document assumption and the tool-call payload shapes — **the two documents
disagree and the spec is the wrong one.**

## F9 · LOW · REAL — a quotation presented as verbatim is an elision of two non-adjacent fragments

`CHANGELOG.md:205-206` quotes *"has no Error constructor: no export can fail it"*; the
source reads *"…and has no Error constructor: skill activation is a property of the harness
and of this adapter, not of any export, so no export can fail it."* Fair paraphrase, but
inside quote marks and attributed. The substance is TRUE and driven.

## F10 · LOW · REAL — "about seventy times … about sixty of those are correct" undercounts

```
$ git grep -o "0\.5\.0" -- . | wc -l   -> 99   (across 88 lines)
$ bash tests/lib/forward-promise-check.sh 0.5.0 ; echo $?   -> 0
```

**99 occurrences, and the check reports zero findings, so ALL of them are correct** — not
"about sixty". The number is the load-bearing justification for reading a verb vocabulary
instead of the version string, and it is ~40% low.

## F11 · LOW · REAL (durability, not falsity) — the 8 MB concurrency bytes are unreproducible from the tree

Re-ran all three rounds: 100 B × 40 → 40 lines, 0 unparseable, 1 length; 200 KB × 40 →
8,195,560 bytes; 8 MB × 16 → **128,002,384** bytes, 1 distinct length (8,000,149 incl.
`\n`).

**Every structural claim reproduces exactly**, and it is structural, not lucky:
`profiler/hooks.go:273-279` opens its own descriptor with `O_APPEND|O_CREATE|O_WRONLY` and
does a single `f.Write(append(line,'\n'))`. The published figures are internally consistent
(`128,003,312 = 16 × (8,000,206 + 1)`), so almost certainly a faithful transcription — no
slip. But they are payload-dependent and **no test, script or fixture pins the payload**
(`git grep "128003312\|8000206"` matches only the two release documents). A reader cannot
regenerate them. **Invariant if kept:** a published byte figure needs its generator in the
tree, or publish the structural result and drop the absolute bytes.

## F12 · LOW · REAL — "backed up before any write — install *and* uninstall" is false for a virgin install

```
$ profiler hooks install --home /tmp/v2t      # no pre-existing hooks.json
{"events_registered": 21, "schema_version_added": true}
$ ls /tmp/v2t/.cursor/   ->  hooks.json       # no .bak-*
```

`README.md:560-561`. A virgin install **writes** and takes **no** backup. The changelog's
"a plain re-run takes no backup at all" covers the no-write case but not this one.
Everything else in the backup story is TRUE and driven, including that two runs inside one
second collide into one `.bak` holding the **post-install** state, making the original
unrecoverable — exactly as the corrected text says.

## F13 · LOW · REAL — an absent `cwd` yields no key, never `""`

`profiler/hooks.go:63` is `Cwd string \`json:"cwd,omitempty"\``. Driven: the `cwd` key is
**absent**, not empty. `CHANGELOG.md:447-448`'s "Promoting an absent field as `""` is the
documented behaviour" is the falsifiable one; `README.md:531-533` and the spec are loose in
the same direction.

---

## Shared roots

- **Root A — F1, F2, F5, F6, F10 (and F8).** Every hard-false sentence asserts a **count or
  a set derived by reading** — a vendor reference, the fixture directory, `git ls-files`,
  git history — rather than by running a command and pasting its output. The same
  documents' *behavioural* claims, which were driven, came back clean at **118/118**.
  **Invariant:** every count, set, and negative-existence claim in the prose must be the
  output of a command, recorded beside it, re-runnable at head.
- **Root B — F3, F12, F13.** A sentence describing a literal or behaviour a sibling lane
  changed underneath it (`--spool-dir` added; `omitempty` on `Cwd`; backup-on-virgin-write).
- **Root C — F7.** The freeze preserves 0.4.3's own false sentences, and the 0.5.0
  narrowing was applied to the **untracked** ledger instead of the two shipped documents
  that carry the error.
- **Root D — F4.** A new guard whose matcher is narrower than the vocabulary its own header
  declares closed, with all 18 controls inside the narrow half.

## The two judgments

**The P6 ruling — reasoning SOUND, scope INCOMPLETE.** `README.md:808` is byte-identical to
`v0.4.3:README.md:436`, and the row names a version string, isolation, a command and a
result; `v0.4.1:README.md` contains no `codex-cli 0.153.4`, so "recorded since 0.4.3"
holds. Narrowing L22 is correct. **The gap: it audited the ledger, which is untracked and
therefore unpublished, and stopped.** The surviving self-contradiction is F7.

**`skill.kind` — the confirming pass is RIGHT; the prior finding was wrong.** Anthropic's
current reference documents `skill.kind` as `"workflow"` for workflow skills and **absent
otherwise**; `invocation_trigger` as `"user-slash"`/`"claude-proactive"`/`"nested-skill"`;
`skill.source` "for example" `"bundled"`/`"userSettings"`/`"projectSettings"`/`"plugin"`;
and `skill.name` as the placeholder `"custom_skill"` for user-defined and third-party
plugin skills unless `OTEL_LOG_TOOL_DETAILS=1`. So the emitter **does** send `skill.kind`,
for workflow skills only — "never sends" was false and the real defect is out-of-vocabulary
fixture values. **Two riders:** the fixture *counts* attached to that finding are wrong
(F6), and the reference introduces `skill.source` with "for example", so calling it a
closed set is marginally stronger than the source supports.

## What came back TRUE — 105 clean assertions, condensed

**Redaction, all of it.** 14 content keys counted in `hooks.go:203-208` and 14 named in the
spec; the five-bullet contract is exactly five bullets. Default leaves
`prompt`/`command`/`tool_input`/`tool_output` **plaintext**; `--strict` gives
`{"_stripped_bytes": …}` and records `strict: true`. Cross-cases both ways: a
non-credential-shaped value under a credential key is `[REDACTED]` wholesale;
credential-shaped substrings under innocent keys are replaced **in place** with surrounding
text surviving; `user_email` reaches disk in **both** modes; `1789332594304000000` intact.

**The `--strict` recipe end to end** — the documented hand-written `--command` produced a
strict line; reachable by no other path and no env var.

**README try-it** — registers `<abs> ingest --spool-dir /tmp/try-it/.skill-architect/spool
|| true`; fired with `HOME` elsewhere, `analyze` read the line back. Re-install
byte-identical md5, no backup, 0 events. `0600` in `0700`. Foreign entries, unknown events
and unknown top-level fields all survive uninstall. Corrupted line: `lines 2, envelopes 1,
unreadable 1`. `analyze` = exactly **16** fields, exit **1** on a missing dir.

**`README.md:326-341` tradeoff table, both rows** — wire `custom_skill` → `custom_skill`;
wire real name → real name; count and timestamp identical either way.
`profiler/claude_code.go:788` is `r.Attributes.String("skill.name")`, so the citation is
right and the old "reads nothing it adds" was indeed false.

**The two vocabularies** — negative control → capability `"none"`, profile `unknown` with a
reason; supplied-unreadable → capability `"none"`, profile `error` for all four
export-backed signals, exit **2**.

**Adapter drafts, per file at head** — `cursor.go` alone → 5 undefined (incl. `getBool`);
`codex.go` alone → 4; **`devin.go` alone compiles cleanly**; all three at once →
`./codex.go:224:15: too many errors`, the exact cited string. All citations resolve at their
named refs; none of the five names exists at `v0.4.1`; `0a83615` **is** the tip of
`origin/wip/0.5.0-profiler`.

**`-o`** — six roots each refused exit 1; four refusals driven; resolve-then-decide both
directions; unresolvable → exit 3; overwrote a pre-existing regular file and re-ran to a
byte-identical md5.

**`compare`/`experiment`** — 0.4.3-vs-0.5.0 exits 2 with both versions named and every
metric `comparable: false`; `harness`/`snapshot_hash`/`skill_dir` each a note at exit 0.
**Four** refusal times driven, with same-harness ordered at `experiment.go:318` *ahead* of
the registry check at `:320`. **No cap:** `mutation_bounded` exists in the test library,
nothing in `experiment run` sources it, `exec.CommandContext` called nowhere.

**`probe`'s six ways** — all exit **0**, **one** distinct stdout report apart from
`probed_at`, every capability `none`, no reason anywhere; stderr the only differentiator.

**Counts** — nine added json tags by set-diff, none removed. Capability report 5 → **6**
keys. `ToolCallEntry` built in one place. No non-test caller of `PresentAttributionResult`
or `PresentEstimatedTokensResult`. Six new subcommands. `doctor`: two tiers, exit 0,
`observed.spool.yields` carries the caveat, 21 events read back, no `getenv`.

**Verification block** — 2741 = 1886+350+191+158+86+50+20, **0 failed, identical under bash
5.3.15 and 3.2.57**. v0.4.3 from the tag: **2499**. 285 Go tests (239+35+11), **284 PASS +
1 SKIP**, **696** subtests, 0 FAIL under `-race`. v0.4.3: 93 and 452. Coverage 97.9 / 94.7
/ 97.0; `-race -cover ./cmd` → 3.4%. 64 fixtures vs 59. 8 queries, all exit 0 on DuckDB
v1.5.5; **`duckdb` not among the ten Prerequisites** and invoked by nothing in
`tests/`/`.github/`. skillscore 2.0.2: head 92.5 (A-) / 89.5 (B+); `v0.4.1` **94 (A)**.
Assertion audit: `sites=761 files=7 lines=9397 accounted=9397` against `wc -l` = 9397.

**Frozen sections** — CHANGELOG from `## 0.4.3` down = **240** lines byte-identical;
RELEASE_NOTES from `## v0.4.3` down = **112** lines byte-identical.

**No forward-looking 0.5.0 promise survives** the tracked tree. **PR #22 body line 109
reads "fell 94 (A) → 92.5 (A-)" — corrected.**

**Ledger** — exactly **15** rows name 0.5.0, all OPEN; the remaining-13 enumeration has
exactly 13 items. `MetricSource`'s five unproducible values each annotated and read by AST
at `adapter_contract_test.go:1157-1181`.

**UNVERIFIABLE-AS-WRITTEN (2, not findings):** `README.md:231-232` ("checked against every
distinct attribute key on a live export") and `README.md:795` (`claude plugin list` at
0.5.0) — no live export and no plugin install reachable here; the second follows
deductively from the five manifests.
