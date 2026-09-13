# Session: skillgate dogfood-and-fix loop

Repo: `/Users/matthewvandusen/Development/Auraprix/skill-architect`
Module: `skillgate/` (Go, in `go.work` beside `profiler/`)

## Hard rules — read first
- **PROTOTYPE MODE.** Do NOT commit, branch, push, stash, or open PRs.
  Everything stays uncommitted on `main`. The user has enforced this twice —
  do not drift back to PR workflow no matter how "ready" the tree looks.
- No Python for scripts (`AGENTS.md`). Go for logic, Bash for simple glue.
- Verdicts are `APPROVE`/`CAUTION`/`REJECT` only — never "safe"/"clean".
- No rule-representation refactor until `docs/research/rule-language.md` lands.

## What exists
- `skillgate/`: G0 quarantine (remote `git clone --depth 1`, hooks disabled,
  commit+manifest-digest provenance pin), G1 coverage ledger (SHA-256 per
  file, allow-listed skip reasons), 20 deterministic tripwires SK-T001..T020,
  advisory shell-outs (skillspector/agnix/skill-validator — never
  load-bearing), verdict engine, fingerprint+reason baselines, SARIF,
  reference graph (SK-G001 dangling / SK-G002 cycles), ICM mechanicals
  (SK-I001..I005), per-harness frontmatter info finding (SK-H002), Pack E
  token budget with per-item classes.
- F14: `preactivation-bash-leg` is a standing `checks_skipped[]` entry —
  **CAUTION is the reachable ceiling** by design.
- Durable docs (keep current, validate against them):
  `docs/skillgate-intent.md` (mandate, D1–D9, don't-builds),
  `docs/skillgate-spec.md` (normative contract). Control plane:
  `.scuba/roadmap.md` (temporarily un-ignored — do not re-ignore without
  asking; do not commit it).
- Validation: `go test ./...` + `go vet ./...` in `profiler/` and
  `skillgate/`; `tests/test_{skill,walk,f01,f02}.sh`. All green at handoff.

## Where the last session ended
Dogfooding on real repos. `anthropics/skills@34040c9c` run found+fixed:
- T007 FP: markdown table `| Python | https://…` → match can't cross `|`
- T002 FP: `system prompt:` doc prose → needs instruction-like content after `:`
- G001: refs resolve file-dir → ancestors → package root (221 → 10)
- `~/` excluded as a bundle-relative ref

263 → 47 findings; survivors mostly true positives (20× T013 no
`allowed-tools`, 9× T006 placeholder creds, T008/T010 baseline cases) and
56 `binary_unparsed` → REJECT on untrusted (F13 working as designed).

## The loop
1. `cd skillgate && go build -o /tmp/skillgate ./cmd/skillgate`
2. `/tmp/skillgate gate -o /tmp/out.json <git-url>` — flags BEFORE the target
   (Go flag parsing stops at the first positional arg).
3. Triage: FP class → fix rule + add a no-fire fixture; true positive →
   leave it. Verify: positive fixture still fires, negative doesn't,
   `go test ./...` green.
4. Next targets: `obra/superpowers` (plugin manifest, many skills, heavy
   scripts), a small single-skill repo, a Cursor-layout skill, a known-bad
   example if you can find one.
5. Report per round: repo@commit, verdict, findings delta, fixes applied.

## Open judgment calls — ask the user before deciding
- Binary assets (docx/pptx/pdf templates) → `binary_unparsed` → incomplete
  ledger → REJECT on untrusted. Keep strict vs. an `inspected-as-binary`
  outcome that records the hash but admits content wasn't inspected?
- T005 env+sink is file-level correlation — narrow to a proximity window?
- T006 placeholder creds (`ghp_your_github_token`) fire as credential-shaped —
  baseline vs. a placeholder-vocabulary exclusion (trades recall)?
- Cursor cross-root collision detection and the pi ground-truth bench are
  slice-2+ scope — don't build them in this loop.
