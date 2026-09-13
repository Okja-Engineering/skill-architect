# dogfood-1: pi monorepo

**RUN STATUS: BLOCKED — no gate output was produced.** This agent ran in background mode; `go build -o /tmp/skillgate-pi ./cmd/skillgate` and every follow-on exec (including `go version`, `curl`, and reads of `/tmp/gate-pi.json`) were auto-denied. `/tmp/gate-pi.json` does not exist. Everything below is a *predicted* triage derived from (a) reading `skillgate/` source end-to-end and (b) manual web inspection of the target repo via `raw.githubusercontent.com` and GitHub-rendered pages. Counts marked `~` are estimates; every classification needs a real run to confirm. Re-run commands:

```bash
cd skillgate && go build -o /tmp/skillgate-pi ./cmd/skillgate
/tmp/skillgate-pi gate -o /tmp/gate-pi.json https://github.com/earendil-works/pi.git
```

**Target substitution (required note):** the requested `https://github.com/badlogic/pi-mono.git` is not the live repo. The pi coding-agent monorepo now lives at `https://github.com/earendil-works/pi.git` (badlogic = Mario Zechner; `earendil-works` is his org). Top-level `package.json` is `"name": "pi-monorepo"` with workspaces `packages/*`, `packages/session-backends/*`, and five coding-agent example-extension workspaces. All triage below targets `earendil-works/pi` @ `main` (commit SHA unverified — no git access).

## Predicted header fields

| field | prediction | basis |
|---|---|---|
| verdict | **REJECT** | blockers expected (T013 on ~15 fixture SKILL.mds, T017 lifecycle/extensions, I001/I002 on intentionally-broken fixtures) + almost-certain incomplete ledger (committed binaries, see below) on an *untrusted* remote bundle |
| coverage | **incomplete** | committed binary assets: `packages/coding-agent/docs/images/*.png`, `packages/coding-agent/src/modes/interactive/assets/*.png`, `packages/tui/src/native/prebuilds/*/*.node` (listed in `create-source-archive.sh` `required_paths`) → `skip_binary` entries → `coverage.complete=false` → untrusted remote forces REJECT regardless of findings |
| file_count_limit | possible | `ledger.go` maxFiles=4096 and `SkipLimit` calls `filepath.SkipAll` — i.e., the first file past the cap truncates the *entire remaining* walk, silently dropping everything alphabetically after it. Repo file count unverified (est. 2–5k files); if >4096, large subtrees may never be ledgered at all |
| too_large | unverified | per-file cap 1 MiB. `package-lock.json` is only 163 KB (confirmed via blob page). Candidates for `skip_too_large`: `packages/ai/src/providers/data/*.json` model catalogs, `packages/coding-agent/npm-shrinkwrap.json`, `test/fixtures/*/large-session.jsonl` — sizes not verified |
| checks_skipped | ≥4 | standing `preactivation-bash-leg` skip + `skillspector`, `agnix`, `skill-validator` not-found skips (none installed). Each caps verdict at CAUTION; irrelevant under REJECT |
| provenance | remote URL + fetched HEAD sha; quarantine dir under os temp | `gate.go` marks fetched targets untrusted and recomputes verdict |
| package_manifests | ~18 expected | root + `packages/{agent,agent-core,ai,coding-agent,mom,pods,tui,web-ui}` + `session-backends/{acp,databricks,ssh}` + 5 example-extension workspaces (+ possibly `install-lock/package.json`, `test/` fixture package.jsons — ledger counts every inspected `package.json`) |

## Findings by rule — rule | count | TP or FP | mechanism/evidence

Predicted fires, by confirmed repo content. "TP-mech / FP-context" = the regex/detect matched what it was designed to match, but the flagged code is a legitimate repo artifact.

- **T007** (pipe-to-shell) | ~1 | **TP-mech, FP-context** — `packages/coding-agent/README.md` install block: `curl -fsSL https://pi.dev/install.sh | sh`. Real pipe-to-shell, but it is the project's own documented installer. Possibly also in other install docs not fetched.
- **T010** (harness-config snoop, `.claude/`) | ~6 | **TP-mech, FP-context** — `packages/coding-agent/examples/extensions/claude-rules.ts` exists precisely to *scan* `.claude/rules/` for cross-harness compat; every `.claude/` path literal + `claude_settings` token fires. Confirmed file content.
- **T011** (`.cursor/` snoop) | ~0 expected | unverified — no `.cursor/` reference confirmed in script files (docs mention is Markdown → not scanned by T011).
- **T012** (editor state DB) | ~0 expected | unverified — no `state.vscdb`/`accessToken`+cursor co-occurrence confirmed.
- **T014** (confirmation/permission gating) | ~8–10 | **TP-mech, FP-context** — confirmed hits: `claude-rules.ts` (`requiresConfirmation: true`, `ctx.ui.confirm(`), `examples/extensions/permission-gate.ts` (`requiresConfirmation`, `confirm(` — file is literally an example of confirm-before-dangerous-bash), `confirm-destructive.ts` (same pattern). These are the repo's own *safety* examples.
- **T017** (manifest executables/lifecycle) | ~6 | **mixed** —
  - `pi.extensions` leg: **TP-mech** — five workspace example manifests declare `"pi": {"extensions": [...]}` (`with-deps`, `custom-provider-anthropic`, `custom-provider-gitlab-duo`, `sandbox`, `gondolin`; globs like `./extensions/*.ts` literal-match). This is exactly what the rule wants. Benign intent.
  - lifecycle leg: **FP-class** — root `package.json` has `"prepare": "husky"`. `manifestExecFindings` fires on *any* non-empty `preinstall|install|postinstall|prepare` regardless of payload → a dev-only git-hook bootstrap becomes a finding. `prepublishOnly` scripts elsewhere correctly do NOT fire (not in the key set).
- **T020** (persistence surfaces) | ~1 | **TP-mech, borderline** — `project-trust.ts` doc comment: `cp packages/.../project-trust.ts ~/.pi/agent/extensions/` → `.pi/` + `cp` verb on the same comment line. It *is* a write to pi's persistence root; flagging a usage-comment install line is arguably intended. Notable asymmetry: the same `cp … ~/.pi/agent/` instruction inside fenced Markdown (`examples/extensions/README.md`, `docs/skills.md`, `docs/packages.md`) does NOT fire because T020 only scans `isScript` files.
- **G001** (dangling refs) | ~8–10 | **TP-mech, FP-context** — confirmed unresolved-from-root refs: `.pi/skills/add-llm-provider.md` references `src/core/model-resolver.ts`, `src/core/provider-display-names.ts`, `src/cli/args.ts`, `docs/providers.md` — all *package-relative* (`packages/coding-agent/…`) paths with no bundle-root target. `docs/skills.md` example-block refs `./scripts/process.sh`, `references/REFERENCE.md` (hypothetical skill layout) and backticked `` `.pi/settings.json` `` (file doesn't exist in repo — it's a runtime path). `docs/packages.md` same `.pi/settings.json` ref. Ancestor-relative retry (`./pi-test.sh` from `.pi/skills/` → root `pi-test.sh`) correctly resolves — known-fixed class holding.
- **T013** (no `allowed-tools` on exec bundle) | ~15 | **FP-class (largest)** — `packages/coding-agent/test/fixtures/skills/` contains ~13 intentionally-malformed fixture `SKILL.md`s (missing-description, name-mismatch, no-frontmatter, invalid-name-chars, consecutive-hyphens, long-name, invalid-yaml, disable-model-invocation, unknown-field, multiline-description, nested/child-skill, root-skill-preferred, …) plus `skills-collision/{first,second}/calendar`. None declare `allowed-tools`; the monorepo ships `.ts`/`.sh` executables → every fixture gets the "executable bundle declares no allowed-tools boundary" blocker. Pi's *own* skills (`.pi/skills/*.md`) escape entirely because they are `<name>.md`, not `SKILL.md`.
- **I001** (missing name/description) | ~2–3 | **TP-mech, FP-context** — fixture `missing-description/SKILL.md` (confirmed `name` only), `no-frontmatter/`, `invalid-yaml/` → missing-key findings on deliberately broken test fixtures.
- **I002** (name charset / collision) | ~5–6 | **TP-mech, FP-context** — `name-mismatch` (name `different-name` vs dir `name-mismatch`), `skills-collision` (two `calendar` skills — the fixture exists specifically to test collision handling), `invalid-name-chars`, `consecutive-hyphens`, `long-name`.
- **I003–I005** | unknown | fixture contents not all fetched; possible hits on description-length/field checks.
- **H002** (per-harness frontmatter) | ~1 | **TP, info** — `disable-model-invocation/SKILL.md` fixture.
- **T001/T003** | unknown | no BOM/hidden-unicode confirmed; generated provider `.ts` files are large non-ASCII-capable surfaces — could not grep. Predict 0–few.
- **T002** | ~0 | no override phrasing confirmed in fetched files.
- **T004/T005** | 0–few, unverified | candidates: `packages/ai/scripts/generate-models.ts` (~3k lines, `fetch()` + provider base URLs + likely `process.env`), `scripts/publish-model-catalog.mjs` (`process.env.AWS_*` + `aws s3` — sink regex may not match `s3://`), `packages/ai/src/env-api-keys.ts` (env reads, sink unknown). Coarse file-level ANDing means any `process.env` + `https://` literal in the same file fires — likely some hits across `src/` — **OPEN per brief**.
- **T006** | ~0 | workflows use `${{ secrets.X }}` (no quoted-literal credentials); `pi-test.{sh,ps1}` only *unset* keys. Placeholder-credential edge cases unverified — **OPEN per brief**.
- **T008** | unknown | `prebuild-install`/`node-gyp` presence in shrinkwrap not verified.
- **T009** | unknown.
- **T015/T016** | ~0 | no `mcp.json`/`.vscode/` files confirmed in tree listings.
- **T018** | 0 | no `.cursor/hooks.json`.
- **T019** | ~0 | `../`-prefixed refs found (`docs/skills.md` `"skills": ["../.claude/skills"]` in fenced JSON — and fenced blocks aren't ref-extracted; `examples/extensions/README.md` md-link `../rpc-extension-ui.ts` resolves *inside* the bundle). Nothing confirmed to escape the bundle root.
- **T021** | 0 | no `hooks.json`/`settings.json` with hook/command keys confirmed (a `settings.json` fixture could still exist under `test/` — unverified).

## FP classes nominated — class | rule | minimal fix sketch | est. blast radius

| class | rule | minimal fix sketch | est. blast radius |
|---|---|---|---|
| **Any `prepare`/`postinstall` string = finding, payload-blind** | T017 lifecycle | Gate severity on payload, not key: treat bare local-tool invocations (`husky`, `changeset`, `patch-package`, `node ./scripts/x.mjs` w/o network/write verbs) as INFO; escalate only when the script body contains `curl|wget|node -e|chmod|>` etc. Alternatively require the script to reference a bundled executable. | High — `prepare: husky` is in a large fraction of modern repos; every monorepo root manifest will fire |
| **Test fixtures scanned as first-class skills** | T013, I001, I002, H002 | Downgrade or annotate findings whose path matches `test/`, `tests/`, `fixtures/`, `__fixtures__/`, `testdata/`; or apply ICM/T013 only to skill dirs reachable from the bundle root (top-level `skills/`, `SKILL.md` not under a `test/` segment). At minimum, collapse per-fixture blockers into one aggregate note. | High — any repo shipping fixture skills gets N blockers + REJECT; dominant noise source here (~15 of ~40 predicted findings) |
| **Whole-monorepo `hasExec` poisons every SKILL.md** | T013 exec-bundle leg | Compute `hasExec` relative to the skill's own directory subtree (or package dir), not the entire ledger — a fixture skill beside 5k source files shouldn't "declare no boundary" for the whole repo. | High on monorepos; near-zero on real skill bundles |
| **Doc-comment/code-identifier false context on `.claude/`** | T010, T014 | Suppress/annotate when matches occur in files under `examples/`, `docs/`, `test/` or when the line is a comment/string-table entry; alternatively require a file-access verb near the `.claude/` literal. | Medium — compat shims and docs in agent-tool repos mention `.claude/` legitimately |
| **Package-relative doc refs unresolved from bundle root** | G001 | For `.md` under `packages/<p>/`, resolve bare relative refs against the containing package dir (up to the package's manifest root) in addition to bundle root; and/or treat refs inside fenced example blocks as illustrative (skip or downgrade). | Medium — docs-heavy monorepos will emit a steady drip of dangling-ref findings on every page that documents per-package layout |
| **Runtime-path refs (`~/.pi/agent/…`, `.pi/settings.json`) treated as bundle refs** | G001 | Skip backticked refs that resolve to known runtime/config locations (`.pi/settings.json`, `~/.pi/**`, `.claude/**`) rather than files expected to ship in the bundle. | Low–medium |
| **T020 write-detection blind to programmatic writes** | T020 (gap, not FP) | Coverage note rather than FP fix: `import-repro.ts` does `writeFileSync` to session paths and `bash-spawn-hook.ts` does `source ~/.profile` — neither fires (no shell verb / read not write). If intent is persistence-surface coverage, add `writeFileSync|appendFileSync|mkdirSync` + `.pi/|trust.json` co-location detection. | Medium — pi extensions persist via fs APIs, not shell redirects; the regex misses the actual mechanism used here |
| **`$HOME/.pi/agent` ≠ `agent/trust.json`** | T020 (gap) | Path regex keys on `agent/trust.json` literally; `migrate-sessions.sh` sets `AGENT_DIR="${PI_AGENT_DIR:-$HOME/.pi/agent}"` and `mv`s session trees into it — no fire because `trust.json` isn't named and `.pi/agent` isn't in the path list. Consider adding `\.pi/agent` (and `$PI_AGENT_DIR`) to the persistence path set when paired with a write verb. | Medium — pi's persistence root is `~/.pi/agent/` wholesale; narrowing to `trust.json` misses session/skill/extension dirs |

OPEN items carried per brief: binary-asset REJECT policy (here it guarantees REJECT on any repo with committed images/native prebuilds — arguably over-strong for a triage gate), T005 file-level coarseness, T006 placeholder credentials — all unverified pending a real run.

## Gate bugs / surprises / scale issues

1. **`filepath.SkipAll` on file-count cap** (`ledger.go`): when `files_seen` hits 4096 the walk *terminates entirely* — one `file_count_limit` skip can hide an unbounded tail of the repo from every check. On a monorepo this turns a budget guard into a coverage cliff; consider `SkipDir` per-directory or continuing the walk in "record-only" mode.
2. **Untrusted-remote + any skip = unconditional REJECT**: committed `.png` screenshots alone force `coverage.complete=false` → REJECT. For source-repo triage this makes the verdict a constant; consider distinguishing "binary asset skipped" (hash recorded, fine) from "uninspected text".
3. **T017 `pi.extensions` fires even when…** the declared glob points at missing/empty paths? Predicted TP here, but verify whether the check validates that declared extension files exist — a declared-but-absent extension should probably be a different finding (or a G001-style dangling ref).
4. **Convention-discovered resources are invisible to T017**: pi auto-loads `extensions/`, `skills/`, `prompts/`, `themes/` dirs with *no* manifest entry (documented in `docs/packages.md`; `.pi/extensions/*.ts` in this repo are convention-loaded). The gate only audits the `pi.*` manifest keys, so the dominant pi mechanism — zero-manifest convention dirs — produces zero T017 findings. Design question for the rule.
5. **Markdown-vs-script asymmetry on T020**: `cp … ~/.pi/agent/extensions/` fires inside a `.ts` comment but not inside a `.md` install doc — same instruction, opposite verdict, purely on file extension.
6. **CLI flag order**: `skillgate gate -o out.json <target>` works (flags before positional); `gate <target> -o out.json` does not (flags parsed only before `fs.Arg(0)`). Documented gotcha, not a bug.
7. **Performance**: could not measure — wall clock, token budget, and hang behavior all unverified. The repo is a realistic stress case (~2–5k files, ~3000-line generated TS, lockfiles) once exec is permitted.

## Evidence appendix (confirmed repo content)

- `.pi/extensions/{tps,import-repro,prompt-url-widget,redraws}.ts` — real event-driven extensions (agent_start/agent_end hooks, session-file writes, `gh` CLI calls, GitHub API fetches).
- `.pi/skills/{interactive-testing,add-llm-provider,skill-docs-todo,postmortem-*}.md` — flat `.md` skills (not `SKILL.md` convention).
- `scripts/migrate-sessions.sh` — `AGENT_DIR="${PI_AGENT_DIR:-$HOME/.pi/agent}"` + `mv` of session trees (no `trust.json` literal → T020 silent).
- `test/fixtures/skills/` — ~13 intentionally-broken SKILL.md fixtures + `skills-collision/{first,second}/calendar/SKILL.md` (confirmed frontmatter).
- `packages/coding-agent/test/fixtures/empty-agent/` — `.gitkeep` only.
- `.github/workflows/build-binaries.yml` — pinned actions, no curl-pipes; `${{ secrets.* }}` never as quoted literals.
- `package-lock.json` — 163 KB / 5311 lines (under the 1 MiB per-file cap).
- `packages/ai/src/models.generated.ts` — 6.4 KB (imports only; the large data lives in `src/providers/data/*.json`, sizes unverified).

## Blockers to completing the real run

- Background-mode exec denial prevented `go build`, `git ls-remote`, and the gate invocation itself. Needs a foreground re-run or pre-approved `go build`/`skillgate` exec.
- No commit SHA captured → provenance pin unknown; report version claims (`0.0.3` root manifest vs `0.85.1` packages) reflect upstream drift between fetch times.
