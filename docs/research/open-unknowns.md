# Open unknowns — what is NOT verified, and the experiment that settles each

Consolidated 2026-09-10 from `skill-architect/.scuba/teams/cursorscope-go/roadmap.md` §G (14 items)
and `.scuba/teams/research-profiling/ledger.md` "What is UNKNOWN or unverifiable" (12 items),
de-duplicated. Evidence labels per `AGENTS.md`.

**Nothing on this list has been confirmed on the research machine.** Every design that depends on one
of these is building around a hypothesis. Ordered by how much damage a wrong answer does.

---

## Tier 1 — premise-level. A wrong answer voids work already planned.

### U-01. Cursor is not installed on the research machine

**Repository fact.** `~/.cursor/` and `~/Library/Application Support/Cursor/` are **absent**. No hook
payload, no `state.vscdb`, and no `hooks.json` has ever been observed. Every hook field name in
`cursor-telemetry-surfaces.md` §1 is **Specification** from Cursor's docs, not a captured payload.

**Blast radius.** The entire hooks half of the design is unbuildable-as-verified. Without a real
session there are no fixtures, U-02's spike cannot run, and the attribution premise stays unproven.

**Experiment that settles it.** Install Cursor on a machine the team controls. Back up
`~/.cursor/hooks.json`, register a pass-through hook that appends raw stdin to a file, run one real
session, and diff every observed payload against the 21-event table. Promotes ~20 **Local
hypothesis** claims to **Repository fact** in one shot. *Gate: this is a user decision, not an
engineering one.*

### U-02. Does Cursor's skill loading fire `beforeReadFile` on a `SKILL.md` path?

**Local hypothesis, load-bearing.** The entire local skill-attribution heuristic — the stated prize of
the whole effort — depends on it. If Cursor loads `.cursor/skills/*/SKILL.md` internally without a
file-read tool call, the heuristic yields nothing and the attribution work produces `unknown`.

**Experiment.** Register a `beforeReadFile` hook. Put a skill in `.cursor/skills/`. Prompt in a way
that should trigger it. Check whether a `beforeReadFile` event with the `SKILL.md` path arrives, and
whether `file_path` is absolute, workspace-relative, or symlink-resolved. Record the answer as a
**Repository fact** with the captured payload as the citation.

**⚠ Sequencing defect carried from the spec-gate review:** this spike currently sits *downstream* of
the slice it can void. It should gate that slice, not follow it.

### U-03. Cursor plan tier

**Unknown.** Determines whether the only honest token numbers are reachable at all.

| Tier | Consequence |
|---|---|
| Free / Pro | Neither Enterprise OTel nor the Admin API exists. Tokens are estimates or nothing. |
| Team | `POST /teams/filtered-usage-events` unlocks **billed per-request tokens** joined on `conversationId`. |
| Enterprise | `cursor.skill.activated` becomes a **first-party** activation event, beating the file-read inference outright, and `cursor.api.request` gives per-request four-way splits. |

**Experiment.** Check the team's Cursor billing page. One minute. *Gate: user decision.*

---

## Tier 2 — contract-level. Design is written against an unexercised spec.

### U-04. The Cursor OTel Wire Reference was read, not exercised

**Specification, unverified in practice.** No real Enterprise OTel export has ever been parsed. All
proposed fixtures are hand-authored from the published wire spec — so a fixture and the code can agree
perfectly while both being wrong about what Cursor actually emits.

**Experiment.** Requires Enterprise (U-03). Configure the export to a local OTel collector with a file
exporter, run one session that reads a skill, and diff the emitted attribute keys against the wire
reference. Absent Enterprise: no experiment exists — document the fixtures as spec-derived and label
the adapter's confidence accordingly.

### U-05. The Admin API `filtered-usage-events` response shape

**Specification only.** No key, no call, no response body observed. The `httptest` fixture would be
authored entirely from docs.

**Experiment.** With a Team admin key: one `POST /teams/filtered-usage-events` for a 1-day range,
capture the raw body, assert `tokenUsage` and `conversationId` exist and that `conversationId` matches
a hook-emitted `conversation_id` from the same session. **This single call also proves or disproves
the whole hooks-for-structure / Admin-API-for-totals architecture.** Highest value per minute on this
list, if U-03 permits.

### U-06. Does Enterprise `cursor.skill.activated` fire for workspace `.cursor/skills/`?

**Inference, not confirmation.** The `cursor.skill.source` enum includes `workspace`, which strongly
suggests yes — but a strong suggestion is not evidence. If it fires only for marketplace/plugin
skills, then even on Enterprise the first-party activation path does not cover the skills we care
about.

**Experiment.** Same setup as U-04; place the skill in `.cursor/skills/` specifically and assert
`cursor.skill.source == "workspace"` on the emitted event.

### U-07. `chars/4` accuracy on code, JSON, and non-English text

**Local hypothesis.** Calibrated only on English markdown (−7%…+15% vs `o200k_base`). BPE efficiency
diverges most from 4 chars/token exactly on code, JSON, and non-Latin scripts — which is most of what
a `tool_output` payload contains.

**Experiment.** Cheap and runnable today: run `skill-validator check -o json` (or any `o200k_base`
tokenizer) over a corpus of Go source, a large JSON blob, a shell transcript, and a non-English
document; compare against `len/4`. Publish the per-content-type error band. **Do this before any
report quotes an error bar**, because the current ±15% figure will be quoted at payloads it was never
measured on.

### U-08. Cursor `state.vscdb` schema

**External evidence only**, verified against Cursor 2.6 / 3.0 with a documented **one-way breaking
migration** between them. Version-fragile. This is why the SQLite token capability should be retracted
rather than implemented.

**Experiment.** Requires U-01. `sqlite3 state.vscdb ".tables"` and a key sample; confirm whether
`composer.composerData` or `composer.composerHeaders` is present, and grep the bubble JSON for any
token field. Expect to find none.

### U-09. Does `CompareProfiles` handle hook-sourced profiles unchanged?

**Read, not exercised.** Assumed to work; never run against a hook-sourced profile.

**Experiment.** Runnable today with hand-authored fixtures — no Cursor required. Construct two
synthetic hook-sourced profiles and run the existing comparison. If it does not work unchanged, the
"fit the existing contract" requirement becomes a real constraint rather than a formality. See also
the proven `compareTokens` source-laundering defect in
`go-dependency-and-tooling-decisions.md` §4 L-2 — that one is already known to be broken.

---

## Tier 3 — measurement gaps. Numbers we will be asked for and do not have.

### U-10. Hook process latency in a live editor

**Unmeasured.** The requirement is "invisible in the editor"; there is no budget number attached to
it, and a slow hook on `preToolUse` sits directly in the user's interaction loop.

**Experiment.** Measure cold-start + parse + append for the hook binary (`hyperfine` or a loop of
`time`), then measure again with the hook installed in a live session and compare perceived latency.
Publish a number and make it a regression test.

### U-11. `/skill-doctor` output parseability

**UNKNOWN — the ledger calls it the single highest-value spike** for a Claude Code token-attribution
path, because it is the only first-party per-skill context-cost number in the ecosystem. Docs say
`-p` "prints as text"; whether that text is stable or parseable is unknown. Local Claude Code is
**2.1.221**, below the documented **≥2.1.252**.

**Experiment.** Upgrade Claude Code to ≥2.1.252, run `/skill-doctor` in an interactive session and
`claude -p` non-interactively, and check whether output is stable across runs and whether any JSON
mode exists. Out of scope for a Cursor-first tool, but it sets the ceiling for what a per-skill cost
number can look like.

### U-12. `claude plugin eval` CLI surface

**No canonical reference page reached.** Subcommands, flags, and output schema unverified. Everything
known comes from the `skill-creator` SKILL.md and secondary write-ups.

**Experiment.** Run `claude plugin eval --help` on a ≥2.1.252 build. **Do not design a hard
integration against it before that.** Note it is *assertion* grading, not *paired* grading.

### U-13. Semgrep behaviour on `SKILL.md`

**Not installed.** The recommendation against it reasons from documented capability, not measurement.

**Experiment.** `pip install semgrep` in a throwaway venv, write one `generic`-mode rule against a
`SKILL.md`, and one real rule against a bundled `scripts/*.sh`. Confirms or refutes the claim that
generic mode ≈ `grep`. Low priority: the decision also rests on the Python-runtime objection, which no
experiment changes.

### U-14. cursorscope's tests as a behavioral oracle

**Listed, not executed.** `attribution.test.js` and `privacy.test.js` are assumed correct for
transcription into Go table tests.

**Experiment.** `npm install && npm test` in the clone. If they pass, transcribe the assertions with
confidence; if any fail or are skipped, do not treat them as a specification.

---

## Tier 4 — citation quality. Affects what we may claim, not what we build.

### U-15. SkillsBench task / domain / trajectory counts

The arXiv abstract (87 tasks / 8 domains / 18 configurations / +16.6 pp) **conflicts** with secondary
summaries (86 / 11 / 7 configs / 7,308 trajectories / +16.2 pp). PDF body not read.
**Experiment:** read the PDF body. Until then, cite the abstract only and never cite trajectory
counts.

### U-16. SkillJuror per-arm pass rates and cost/pass

The 46.1% / 29.0% and $1.31 vs $1.28 figures come from a search summary, not the PDF body (PDF text
streams did not extract). The abstract figures (1.18→3.85 resources touched, +4.1% on 410 matched
trials) **are** confirmed. **Experiment:** obtain and read the PDF. Until then, do not quote the
per-arm numbers.

### U-17. Exact `/skill-doctor` first version

Docs say ≥2.1.252; press says 2.1.261 on 2026-09-04. Both can be true under a feature-flagged rollout.
**Experiment:** check the Claude Code changelog for the release that first mentions it.

### U-18. `@crafter/skillkit` capabilities

npmjs.com returned HTTP 403 to automated fetch; all detail is from search snippets. It also requires
Bun. **Experiment:** fetch the package README manually, or `npm view @crafter/skillkit`. Low priority
— the Bun runtime dependency likely rules it out regardless.

### U-19. `gen_ai.*` semconv stability

**Status: Development** at semconv 1.44.0. Attribute names may change under any OTLP export work.
**Experiment:** none that settles it — this is upstream's roadmap. Mitigate by pinning the semconv
version referenced in any exporter and reviewing on upgrade.

### U-20. `skillscore` dimension count and implementation language

`skill-architect`'s `AGENTS.md` says "npm, 7-dimension"; upstream describes **six** dimensions and a
**Dart** implementation. **Experiment:** read the upstream README and fix whichever doc is wrong. One
minute; doc hygiene only.

---

## What to run first

| Order | Item | Why | Cost |
|---|---|---|---|
| 1 | **U-03** plan tier | Gates U-05 and U-04/U-06; changes what "honest tokens" even means | 1 min, user |
| 2 | **U-01 → U-02** Cursor install + activation spike | Voids or validates the central premise | hours, user-gated |
| 3 | **U-05** one Admin API call | Proves the hooks-for-structure / API-for-totals architecture in a single request | minutes, if U-03 permits |
| 4 | **U-07** chars/4 on code/JSON | Runnable today, no Cursor needed; stops a wrong error bar being quoted | ~1 hour |
| 5 | **U-09** compare on synthetic hook profiles | Runnable today; surfaces contract defects before they are designed around | ~1 hour |
| 6 | **U-10** hook latency | Needed for a non-negotiable requirement with no number attached | ~1 hour, needs U-01 |

Everything below U-10 is either citation hygiene or gated on a decision above this level.
