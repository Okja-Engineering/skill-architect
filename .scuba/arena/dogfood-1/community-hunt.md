# dogfood-1: community hunt

**Status: manual emulation, not authoritative gate output.** The background session was denied
shell execution — `go build`, `git`, and `/tmp/skillgate*` all refused — so no `gate` JSON was
produced for these repos. Everything below is a source-level replay of the registered checks
(`skillgate/tripwire_a.go`, `tripwire_b.go`, `icm.go`, `refgraph.go`, `gate.go`, `ledger.go`)
against file contents fetched from `raw.githubusercontent.com` / GitHub tree pages. Findings are
labeled *predicted*; verdicts are predictions with the firing mechanism proven against the source.
Commit SHAs are unresolved (HEAD of `main` at fetch time; BaluRaut repo has 2 commits, DVC2 repo
was archived 2026-06-13 at 27 commits, mattpocock/skills at v1.2.3 / 1.2.3 plugin version).

To produce authoritative runs (foreground, where exec is permitted):

```bash
cd skillgate && go build -o /tmp/skillgate-hunt ./cmd/skillgate
/tmp/skillgate-hunt gate -o /tmp/gate-hunt-1.json https://github.com/BaluRaut/claude-skills-demo
/tmp/skillgate-hunt gate -o /tmp/gate-hunt-2.json https://github.com/mattpocock/skills
/tmp/skillgate-hunt gate -o /tmp/gate-hunt-3.json https://github.com/DVC2/cursor-agent-configs
```

Avoided sibling-agent repos: `anthropics/skills`, `obra/superpowers`,
`diet103/claude-code-infrastructure-showcase`, `badlogic/pi-mono`. Crashes: none observed or
expected — no run was performed; all three targets are plain text trees, nothing exercises the
crash paths (unparseable JSON is handled, binary files are `binary_unparsed` skips).

`checks_skipped` expected on every run (assuming the optional binaries are absent — unverifiable
without exec): `skillspector`, `agnix`, `skill-validator`, `preactivation-bash-leg` (F14 standing
entry, `gate.go:107-110`). **But see the I005 bug below: `icm-token-budget` will NOT appear** on
any of these repos even though no tokenizer ran.

## repo 1: BaluRaut/claude-skills-demo @main (unresolved SHA) — predicted REJECT

Shape: small demo repo (2 commits). `.claude/skills/` worked-example set
(javascript-standards, typescript-standards, react-patterns, react-data-fetching, react-testing,
antd-ui, antd-form, skill-generator), `plugins/skill-generator/` Claude Code plugin
(`bin/skill-recon` extensionless node script, `scripts/recon.mjs`, `.claude-plugin/plugin.json`),
`packages/skill-recon/`, root `README.md`, `GUIDE.md`, `.gitignore`. Package manifests:
`plugins/skill-generator/.claude-plugin/plugin.json`, `packages/skill-recon/package.json` (likely),
possibly more under `packages/`.

- Verdict: **REJECT** (T013 blockers; remote target is untrusted but ledger completeness is moot —
  the blockers fire first).
- Coverage: expected complete — all files are text; no binaries observed; nothing over 1 MiB.
- Skip reasons (ledger): none expected. `checks_skipped`: skillspector, agnix, skill-validator,
  preactivation-bash-leg.
- Provenance: URL + resolved commit + quarantine dir + `sha256_manifest` + `package_manifests`
  (plugin.json and any package.json; `quarantine.go:44-65`, `ledger.go:204-213`).
- Crashes: none expected.

### findings — rule | TP or FP | mechanism/evidence

| rule | TP/FP | mechanism / evidence |
| --- | --- | --- |
| SK-T013 ×~9 | **FP (bundle-scoped poisoning)** | `tripwire_b.go:89-129`: `hasExec` is set by *any* inspected `isScript` file repo-wide — `plugins/skill-generator/scripts/recon.mjs` (`.mjs`) and `plugins/skill-generator/scripts/` (path leg, `frontmatter.go:107-114`) trip it — then **every** SKILL.md without an `allowed-tools` frontmatter key gets a Blocker: "bundle ships scripts/ but SKILL.md has no allowed-tools". Fetched frontmatter confirms zero `allowed-tools` keys. The doc-only convention skills (javascript-standards, antd-ui, …) inherit a Blocker from an unrelated plugin's census script they never mention. Mechanism fires correctly; per-skill granularity is lost. **Highest-value FP signal: a benign doc skill gets a security Blocker.** |
| SK-I002 ×1 | TP | `icm.go:102-126` collision leg: `name: skill-generator` claimed by both `.claude/skills/skill-generator/SKILL.md` and `plugins/skill-generator/skills/skill-generator/SKILL.md` (both fetched; identical frontmatter). Real silent-shadow hazard — legitimate maintainability catch. |
| SK-G001 | possible, unverified | `refgraph.go:136` dangling in-bundle refs; the example SKILL.mds reference sibling files I did not exhaustively enumerate. Expect none/few. |
| SK-T002/T004-T009/T010-T012/T015-T020 | predicted none | No override phrasing, credentials, network calls, persistence writes, or harness-snoop literals observed in fetched files. `recon.mjs` runs `npx tsc`/`npm run` via `execSync` — T008's `reUnpinned` (`tripwire_a.go:66-73`) covers `npm i/install`, `pip install`, `go install @latest`, `gem install` only — `npx` is not matched (correct: it's not an install). Note `recon.mjs` deliberately *walks* target repos' `.claude` dirs (`e.name !== '.claude'` in `walk()`) yet evades T010 because `reClaudePaths` requires a literal `.claude/` slash that never appears. |

## repo 2: mattpocock/skills @main (unresolved SHA, v1.2.3) — predicted REJECT

Shape: plugin-shaped multi-skill repo. `skills/engineering/` (~18 skills per
`.claude-plugin/plugin.json` `skills` array: ask-matt, diagnosing-bugs, grill-with-docs, triage,
improve-codebase-architecture, setup-matt-pocock-skills, tdd, to-spec, to-tickets, wayfinder,
implement, prototype, research, domain-modeling, codebase-design, code-review,
resolving-merge-conflicts, wizard) + `skills/productivity/` (~7), plus `deprecated/`, `misc/`,
`in-progress/` buckets (per `link-skills.sh` comments — unverified extra SKILL.mds there).
`scripts/` (link-skills.sh, list-skills.sh, sync-plugin-version.mjs), `.claude-plugin/`
(plugin.json, marketplace.json), `package.json`, `.github/workflows/release.yml`, `.agents/` docs,
`docs/`, `AGENTS.md`, `CLAUDE.md`. Package manifests: `package.json`, `.claude-plugin/plugin.json`.

- Verdict: **REJECT** (T010 High + T013 Blockers).
- Coverage: expected complete — all text; `.changeset/` md files inspected too.
- Skip reasons (ledger): none expected. `checks_skipped`: skillspector, agnix, skill-validator,
  preactivation-bash-leg.
- Provenance: remote fetch; commit + quarantine + manifest digest; `package_manifests` =
  `package.json`, `.claude-plugin/plugin.json` (`plugin.json` base matches `isManifestFile`,
  `budget.go:31-38`).
- Crashes: none expected.

### findings — rule | TP or FP | mechanism/evidence

| rule | TP/FP | mechanism / evidence |
| --- | --- | --- |
| SK-T010 (High) | **TP-mechanism / mislabeled-intent** | `tripwire_b.go:53-59` + `reClaudePaths` (`:15`): `scripts/link-skills.sh` contains `"$HOME/.claude/skills"` — `$HOME/\.claude` and `\.claude/` both match, plus the comment `~/.claude/skills:`. The script is the repo's documented *installer* (symlinks skills into `~/.claude/skills` + `~/.agents/skills`). Message says "touches another harness's config directory" — it does touch it, but the check is read/write-blind and `.claude` is the *same* harness these skills target, not "another". Notably T020 does **not** fire here — `rePersistPath` covers only `.claude/settings` and `.claude/CLAUDE`, so writing into `~/.claude/skills` escapes the persistence rule while the snoop rule catches it. The capability flag is correct; the framing is wrong. |
| SK-T013 ×~25+ | **FP (same class as repo 1)** | `scripts/*.sh|*.mjs` set `hasExec`; no fetched SKILL.md declares `allowed-tools` (tdd, setup-matt-pocock-skills, grill-me all checked). Every SKILL.md in `skills/` — including `deprecated/`/`misc/`/`in-progress/` if present — gets a Blocker. A repo whose entire purpose is distributing benign prompt-skills gets 25+ security Blockers for not using a Claude-Code-specific key. |
| SK-H002 ×≥2 (Info) | TP | `frontmatter.go:141-166`: `disable-model-invocation: true` present on `setup-matt-pocock-skills/SKILL.md` and `grill-me/SKILL.md` (fetched) — silently ignored by Cursor. Informational only; correct and useful. |
| SK-T017 | predicted none | `package.json` scripts are `changeset`/`version`/`check-plugin-version` — none are in the lifecycle set (`preinstall/install/postinstall/prepare`, `tripwire_b.go:351`). plugin.json `skills:` array is not `extensions` → not scanned. `.claude-plugin/` ≠ `.claude/` so leg 1 doesn't reach it. |
| SK-G001/G002, SK-I001-I004 | possible, unverified | ~25 SKILL.mds + docs not exhaustively fetched; names appear to match dirs in fetched samples; bodies are short. Deprecated/misc buckets could hold name collisions — confirm on real run. |
| SK-T020 | none | `link-skills.sh` writes `.claude/skills` (not covered — see bugs). `sync-plugin-version.mjs` writes `.claude-plugin/plugin.json` — `\.claude/` requires the slash, `.claude-plugin` evades. |

## repo 3: DVC2/cursor-agent-configs @main (unresolved SHA, archived) — predicted REJECT

Shape: Cursor-layout config collection (the requested shape). `.cursor/hooks.json`,
`.cursor/hooks/` (format.sh, guard-destructive.sh), `.cursor/rules/` (javascript.mdc,
typescript.mdc), `.cursor/skills/` (debug, init-agents-md, write-adr — SKILL.md each),
`.cursor/agents/` (code-reviewer.md, test-author.md, verifier.md), `scripts/` (install.sh,
install.ps1, sync-agents.sh), `examples/hooks/` (README + 3 .sh cookbooks), `templates/`,
`docs/`, `.github/workflows/ci.yml`, root AGENTS.md/README/MIGRATION/SUNSET/CONTRIBUTING/LICENSE.
No package.json → no manifest leg; `package_manifests` empty.

- Verdict: **REJECT** (T018 ×2 + T020 + T011 High + T013 ×5).
- Coverage: expected complete — all text.
- Skip reasons (ledger): none expected. `checks_skipped`: skillspector, agnix, skill-validator,
  preactivation-bash-leg.
- Provenance: remote fetch; commit + quarantine + digest; no package manifests.
- Crashes: none expected.

### findings — rule | TP or FP | mechanism/evidence

| rule | TP/FP | mechanism / evidence |
| --- | --- | --- |
| SK-T018 ×2 | **TP — the rule working as designed** | `tripwire_b.go:395-412` + `hookExecFindings`: `.cursor/hooks.json` parses as JSON; `"command": "./.cursor/hooks/format.sh"` and `"./.cursor/hooks/guard-destructive.sh"` both hit `isBundledCommand` (`./` prefix → bundled). The repo genuinely ships agent-executed hook config — its README even warns "Hooks run shell scripts on agent events. Review them before installing." The hook scripts themselves are defensive (guard-destructive blocks `rm -rf`/`git push --force`). Correct capability detection; REJECT is harsh but semantically honest for a bundle whose purpose is installing executors. |
| SK-T020 ×1 | **TP-mechanism / OPEN on intent** | `tripwire_b.go:415-429`: `install.sh` line `chmod +x "$DEST"/.cursor/hooks/*.sh 2>/dev/null \|\| true` satisfies `rePersistPath` (`.cursor/`) + `rePersistVerb` — where the "verb" is the bare `>` in `2>/dev/null` (stderr redirect, not a write). The same file's actual copy calls (`backup_then_copy "$SRC/.cursor/rules" "$DEST/.cursor/rules"`) carry no verb token on the line and don't fire — the check lands on a coincidental line instead of the real copy operations. File-level verdict (this script does write `.cursor/` config — that is its job) vs. line-level evidence (a chmod + stderr redirect) — classify as mechanism-TP with a noisy-evidence caveat. **OPEN** judgment call: is an installer's documented copy operation a "persistence write"? |
| SK-T011 ×2 files (High) | **FP-leaning / TP-mechanism** | `reCursorPaths` on isScript files: `install.sh` (`.cursor/` literals throughout — copies config *into* projects) and `examples/hooks/test-on-save.sh` (`log=".cursor/test-on-save.log"`). Message says "reads Cursor config or state paths" — both are *writes/installs*, not snoops. **`install.ps1` performs the identical operation with `.cursor\` backslash paths and escapes T011 entirely** — the pattern is forward-slash only. |
| SK-T013 ×5 | **FP (.mdc has no such concept) + repo-1 class** | `hasExec` set by `scripts/*.sh`, `*.ps1`, `.cursor/hooks/*.sh`, `examples/hooks/*.sh`. Fires on all 3 SKILL.md (debug, init-agents-md, write-adr — `name`/`description` only, no `allowed-tools`) **and both `.mdc` rules** — `skillFiles()` (`frontmatter.go:90-104`) includes `.mdc`, whose frontmatter is `description`/`globs`/`alwaysApply`; `allowed-tools` is not a Cursor-rules key at all. A Cursor-layout repo gets security Blockers for omitting a Claude-Code-only key — on files that can never carry it. |
| SK-I001 ×2 | **FP (.mdc format)** | `icm.go:39-52` iterates `skillFiles` = SKILL.md **+ .mdc**, requiring `name` + `description`. `javascript.mdc`/`typescript.mdc` carry `description`/`globs`/`alwaysApply` — `.mdc` has no `name` field (the filename is the identifier). "frontmatter missing required keys: name" is a spec misapplication, same over-inclusion as T013. |
| SK-I002 | predicted none | `write-adr`, `debug`, `init-agents-md` names match dirs (fetched/listed); no cross-dir collisions observed. |
| SK-T004-T009/T012/T015-T017/T019 | predicted none | `guard-secrets.sh` embeds credential-regex literals (`AKIA[0-9A-Z]{16}` etc.) — verified they do **not** self-match `reCreds` (the pattern text can't satisfy its own char-class requirements). `inject-context.sh`'s `cat >/dev/null` is not a persist-path line. `test-on-save.sh`'s `>> "$log"` escapes T020 via `$log` indirection (path and verb never share a line). `examples/hooks/README.md` embeds `"command": "./.cursor/hooks/…"` JSON blocks — correctly NOT scanned (T017/T018 only parse real config paths; a `.md` is not `hooks.json`). Nested `examples/hooks/*.json` would evade T017's exact `hooks/hooks.json` match — gap noted below. |

## FP classes nominated — class | rule | minimal fix sketch

| class | rule | minimal fix sketch |
| --- | --- | --- |
| **Repo-scoped `hasExec` poisoning** | SK-T013 | Compute the exec/boundary relation per skill root, not per repo: attribute each script to the nearest enclosing skill dir (or the SKILL.md that references it); only flag a SKILL.md with no `allowed-tools` when *its own* subtree ships executables. Alternatively downgrade the no-boundary leg to Medium when the script lives outside any skill dir (`plugins/`, `scripts/` top-level tooling). |
| **`.mdc` treated as SKILL.md** | SK-T013, SK-I001 (and I002 name leg is already skipped-by-`n != ""` but I004 still applies) | Split `skillFiles` consumers: ICM legs (I001–I003) should apply to `SKILL.md` only — `.mdc`'s required keys differ (`description`, `globs`, `alwaysApply`); either exempt `.mdc` from T013 or treat a Cursor `globs`/`alwaysApply` declaration as a boundary equivalent. One-line gate: `if strings.HasSuffix(p, ".mdc") → skip I001/I002`. |
| **Snoop checks are read/write-blind + slash-asymmetric** | SK-T010, SK-T011 | (a) Accept `[\\/]` after `.claude`/`.cursor` so `install.ps1`'s `.cursor\rules` can't evade. (b) Reword messages from "reads"/"snoops" to "references harness config dir" — install scripts are the dominant community pattern. Optionally require a read verb or downgrade when the path appears only as a copy *destination*. |
| **Persistence-path surface gaps** | SK-T020 | Add `.claude/skills`, `.claude/commands`, `.agents/` to `rePersistPath` (a skills installer writing `~/.claude/skills` currently escapes); accept backslash separators; and consider a var-assignment leg (`VAR=.*persistpath` then `$VAR` in a redirect) — `$log=".cursor/…"` + `>> "$log"` currently evades. |
| **`>\s*` as a write verb** | SK-T020 | `2>/dev/null` and `cat >/dev/null`-adjacent redirects satisfy the verb leg on any line mentioning `.cursor/`. Require the redirect target to be the persist path (or at least exclude `>/dev/null`, `>&2`, `2>&1` shapes). |

## Gate bugs / surprises

1. **SK-I005 is effectively dead code — and its skip is silent (F13 hole).**
   `gate.go:83-88` builds a non-nil `TokenBudget{Counter:"skillgate/exact-chars"}` whenever any
   `budgetItems` exist — i.e., whenever any SKILL.md has a `description` (always) or a manifest has
   name+description. Then `icm.go:129-147` only names the `icm-token-budget` skip `if budget == nil`.
   Result: on every real bundle, `budget.Files` is empty when skill-validator is absent → the I005
   loop iterates zero times → **no finding, no named skip**. The check "runs" vacuously and F13's
   "no verdict above CAUTION when a check was skipped" silently fails to record that the token
   budget was never measured. Fix sketch: track whether `budget.Files` was populated by a measured
   counter, not whether `budget` is non-nil.
2. **`isScript` has no shebang leg** (`frontmatter.go:107-114`): extension + `scripts/` path only.
   Repo 1's `plugins/skill-generator/bin/skill-recon` is `#!/usr/bin/env node`, on PATH when the
   plugin is enabled, and the file the SKILL.md tells the agent to run — yet it is invisible to
   T004/T005/T008/T009/T010/T011/T020. An extensionless `bin/` payload is a clean evasion. Fix:
   sniff `#!` on extensionless regular files during ledger build.
3. **Manifest `bin` field unchecked**: `manifestExecFindings` (`tripwire_b.go:329-358`) covers
   `pi.extensions`/`extensions` + four lifecycle scripts, but `package.json`'s `"bin"` map —
   the canonical npm mechanism for putting a bundled executable on PATH — is not inspected.
4. **T010/T011 literal-slash evasions**: `reClaudePaths` needs `.claude/` — recon.mjs walks
   `.claude` dirs via `e.name !== '.claude'` (no trailing slash) and escapes; `reCursorPaths`
   misses `.cursor\` backslash paths entirely (`install.ps1`). Both are same-file, same-intent
   coverage holes.
5. **T020 path/verb co-linearity is fragile both directions**: misses `$var` indirection
   (`log=".cursor/x.log"` + `>> "$log"`) and `.claude/skills` writes (repo 2's actual skill
   *installer* only trips T010-as-snoop), while firing on `2>/dev/null` noise.
6. **Nested hook-config paths evade T017**: leg 1 exact-matches `hooks/hooks.json` at root;
   `examples/hooks/hooks.json` would slip through (T018's `.cursor/hooks.json` *suffix* match is
   fine — use suffix matching for both legs).
7. **OPEN judgment calls carried over**: binary-asset REJECT policy (untrusted + incomplete →
   REJECT, `verdict.go:18`); T005 env-harvest coarseness (same-file `process.env` + any `https://`
   sink); T006 placeholder credentials (16+ char literal requirement helps but `token:"xxx…"`
   docs examples are still in scope).

## Benign-bundle UX verdict — did a clean small repo get CAUTION with zero SK-T?

**No — and the three repos found suggest it is nearly unreachable in the wild.** All three predict
REJECT, and the dominant driver is SK-T013: `hasExec` is set by *any* script-extension file or
*anything under `scripts/`*, and then fires on *every* SKILL.md (and `.mdc`) lacking
`allowed-tools` — a key that is (a) Claude-Code-only per SK-H002, and (b) absent from every
community frontmatter observed. The compositional result: **any repo containing even one `.sh`/
`.mjs`/`.ts` file — or any file under a `scripts/` directory, even a `.txt` — REJECTs unless every
skill file carries a harness-specific key.** A truly clean CAUTION (zero SK-T*, only the F14 skip)
requires a repo with zero script-class paths at all — pure markdown, no tooling — which describes
almost no real skills repo. The standing-skip CAUTION ceiling is working as designed
(`verdict.go:21-22`); the problem is that the floor below it is also REJECT for the common
"docs + one helper script" shape. Recommend the per-skill-root attribution fix plus a severity
rationale review before calling small-repo UX healthy.
