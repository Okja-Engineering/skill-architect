# dogfood-1: diet103/claude-code-infrastructure-showcase

**Status caveat (read first):** the sandbox denied every shell command in this
session — `go build`, `git ls-remote`, `jq`, even `go version` — so
`/tmp/skillgate-infra` was never built and `/tmp/gate-infra.json` does not
exist. Everything below is derived from (a) a full static read of the gate
implementation (`skillgate/tripwires.go`, `graph.go`, `icm.go`, `scan.go`,
`hook_exec.go`, `gate.go`) and (b) complete file contents of the target pulled
via `raw.githubusercontent.com` / github.com web pages on `main`. Where a
finding count or coverage number could not be confirmed without running the
binary it is marked **(predicted)**; classifications and mechanisms are
mechanistically traced against the actual rule code, not guessed. The parent
agent should re-run the two commands below and spot-check counts before
treating this report as final.

```
cd skillgate && go build -o /tmp/skillgate-infra ./cmd/skillgate
/tmp/skillgate-infra gate -o /tmp/gate-infra.json \
  https://github.com/diet103/claude-code-infrastructure-showcase.git
```

## Verdict / coverage / skipped checks / provenance

- **Verdict (predicted): REJECT.** Two independent paths force it:
  `SK-T013` is `severityBlocker` and fires on every `SKILL.md` (bundle contains
  registered executables, no `allowed-tools` anywhere), and `SK-T017` is
  `severityHigh` and fires on `.claude/settings.json` (real) plus doc copies
  (false). No suppression file expected in a fresh clone.
- **Coverage (predicted): ~60 files, all inspected, no binary/oversize
  skips.** The repo is all text (markdown, .sh, .ts, .json, LICENSE,
  .env.example, .gitignore); largest files are `package-lock.json` and long
  markdown — all far under the 1 MiB/file limit. `.git/` is ledger-skipped by
  design. Root layout (verified): `.agents/skills/`, `.claude/`, `.codex/`,
  `dev/`, `editor-config/`, `.env.example`, `.gitignore`,
  `CLAUDE_INTEGRATION_GUIDE.md`, `LICENSE`, `README.md`, `setup.ts`.
- **checks_skipped (predicted, with reasons):**
  - `preactivation-bash-leg` — always-appended standing skip (F14): "read path
    covered only before load; bash leg remains open." Makes CAUTION the
    ceiling for any clean input — here moot, verdict is REJECT anyway.
  - `icm-token-budget` — no token counter available.
  - `skillspector`, `agnix`, `skill-validator` — external scanners absent
    (predicted "binary not found" skips; advisory only, not load-bearing).
- **Provenance (predicted):** remote URL → G0 quarantine clone;
  `report.target` = the GitHub URL; manifests = `.claude/hooks/package.json`
  (+ `package-lock.json` digest); no root `package.json`. No crashes, hangs,
  or clone failures were possible to observe — the run never happened.

## Findings by rule

rule | count (est.) | TP or FP | mechanism / evidence
--- | --- | --- | ---
SK-T017 | 5 | **TRUE POSITIVE** | `.claude/settings.json` is valid JSON, so `hookExecFindings` regex `"command"\s*:\s*"([^"]+)"` yields 5 hits, each resolving under `$CLAUDE_PROJECT_DIR` → `isBundledCommand` → finding. Evidence: `"command": "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-activation-prompt.sh"` (UserPromptSubmit), `skill-verification-guard.sh` (PreToolUse `Edit\|MultiEdit\|Write`), `post-tool-use-tracker.sh` + `skill-activation-tracker.sh` (PostToolUse), `session-doc-updater.sh` (Stop). The shipped config really does execute bundled shell scripts at five lifecycle points.
SK-T017 | ~13 | **FALSE POSITIVE** | T017's file filter is `p == "hooks/hooks.json" \|\| strings.HasPrefix(p, ".claude/") \|\| HasSuffix settings.json` — the `.claude/` leg matches **any** file under `.claude/`, not just JSON config. `.claude/hooks/CONFIG.md` is a markdown guide whose embedded example `settings.json` blocks contain ~11 `"command":` lines (5 in the full example, 3 in the "execution order" example, 2 in "build checking only", 1 in "minimal setup"); the file fails `json.Unmarshal` so the unparseable fallback emits one finding per `"command":` line. Same for `.claude/hooks/README.md` (~2 `"command"` example lines). These are documentation samples, never loaded as config.
SK-T010 | ~30–80 | **MOSTLY FALSE POSITIVE, partial TP** | Rule scans script-class files for literal `.claude/` `.codex/` etc. Nearly every hook script references its own bundle home: `_run-node-hook.sh` `NODE_SCRIPT="${CLAUDE_PROJECT_DIR}/.claude/hooks/$1"`; `post-tool-use-tracker.sh`/`skill-activation-prompt.sh`/`session-doc-updater.sh` state paths under `.claude/hooks/state/`; `stop-build-check-enhanced.sh` `.claude/tsc-cache`; `verify-setup.sh` `.claude/` throughout; `sync-agent-skills.sh` `.claude/skills`→`.agents/skills`; `setup.ts` log strings (`cp .claude/hooks/.env.example …`, `bash .claude/scripts/verify-setup.sh`). A bundle whose *home is `.claude/`* cannot avoid tripping this. Genuine TP subset: `tsc-check.sh` writes to `$HOME/.claude/tsc-cache/$SESSION_ID` — a real cross-boundary write into the user's *home* harness dir — and `.codex/hooks/_codex-adapter.sh` reaches into `.claude/hooks/` (a true cross-harness bridge, though benign by design).
SK-T003 | ~20–40 | **MIXED** | Path filter `(^|/)\.(claude|codex|…)/` selects every file under `.claude/`/`.codex/`; the *text* trigger is content matching `\.(md|markdown)`. Every `.claude/**/*.md` that links to or mentions a `.md` file fires (all SKILL.md files, agents/commands docs, CONFIG.md, READMEs) — message "instruction-bearing files inside dotdirs" is semantically true for those, so TP-ish. But the trigger is content-based, so `.sh`/`.ts` files under `.claude/` that merely mention `SKILL.md`/`*.md` (e.g. `verify-setup.sh`) also get labeled "instruction-bearing files" — wrong message for a shell script.
SK-T013 | 9 | **TP per rule / FP in spirit (nominated)** | `hasExec` is set by any registered hook command (`.claude/settings.json` → 5 bundled `.sh`). Nine `SKILL.md` files exist (4 under `.claude/skills/`, 5 under `.agents/skills/` incl. `source-command-verify-setup`), none declares `allowed-tools` → 9 blocker findings: "bundle contains executable content; skill does not declare allowed-tools". The executables are *harness-invoked hooks*, not skill-invokable scripts — the skills never instruct tool use — but the rule cannot distinguish.
SK-T020 | ~5–15 | **MOSTLY TRUE POSITIVE, 1 borderline FP** | Concrete persistence writes exist and match: `session-doc-updater.sh`/`.ts` write `.claude/hooks/state/*` and prune state; `metrics.ts` appends `.claude/hooks/state/metrics.jsonl`; `trigger-build-resolver.sh` writes `.claude/hooks/debug.log`; `stop-build-check-enhanced.sh` + `tsc-check.sh` write tsc caches (incl. `$HOME/.claude/tsc-cache`); `session-doc-worker.ts` creates a SQLite vector DB; `session-state.ts` writes `skills-used-<sessionId>.json`. This bundle is persistence-surface-heavy by design — correct signal. Borderline FP: `_run-node-hook.sh` `source ~/.bashrc >/dev/null 2>&1` — `~/.bashrc` + `>` on one line reads as "write to shell startup file" but is a source-and-discard env probe.
SK-T005 | 1–3 | **OPEN (judgment call per brief)** | Fires on co-occurrence of `process.env` + `https?://` in one script file. `setup.ts` reads `process.env.GEMINI_API_KEY`/`OPENAI_API_KEY`/`ANTHROPIC_API_KEY` and contains `https://aistudio.google.com/apikey` etc. in log strings → 1 finding ("reads env in N line(s) AND reaches the network"). Possibly `types.ts`/provider files if a default `https://…` endpoint literal sits next to `process.env` reads. These are env *reads of named keys* plus *documentation URLs in comments* — the same-file correlation is the known-coarse mechanism → marked OPEN, not re-nominated.
SK-G001 | ~11–22 | **TRUE POSITIVE** | `.claude/skills/backend-dev-guidelines/SKILL.md` links `[architecture-overview.md](architecture-overview.md)` etc. — ~11 unique bare-filename md links — but the files live in `resources/` (verified: dir contains only `resources/` + `SKILL.md`). `refTargets` resolves siblings → missing → dangling. The `.agents/skills/backend-dev-guidelines` mirror presumably repeats all 11. The gate caught a real defect in the target repo (links missing the `resources/` prefix).
SK-I002 | 4 | **TRUE POSITIVE (with context)** | The `.agents/skills/` tree is a byte-mirror of `.claude/skills/` (sync script `sync-agent-skills.sh`), so names `backend-dev-guidelines`, `frontend-dev-guidelines`, `skill-developer`, `error-tracking` are each claimed by two `SKILL.md` files → "name claimed by multiple skills in one package — silent shadow under Cursor's undocumented root precedence." The duplicate is intentional cross-harness mirroring, but the collision the rule describes is real.
SK-T001, T002, T004, T007, T008, T009, T011, T012, T014, T015, T016, T018, T019, G002, H002, I001, I003, I004 | 0 (expected) | — | No override phrasing ("You are a specialized…" ≠ `you are now`), no `curl|bash`/`curl\|sh`, no line-anchored install commands in scripts, no `.cursor/` content, no `mcp`/`autoApprove` config, `allowed-tools` absent → no T014 (command file `route-research-for-testing.md` does declare a scoped `allowed-tools` list — correctly not flagged), no `..` escapes observed that leave the bundle, all skills have name+description and matching dir names, bodies <500 lines. T001 (invisible unicode) could not be verified over webfetch — treat as unverified, not confirmed-clean.
SK-T006 | 0 (expected) | **OPEN** | `.env.example` / `.claude/hooks/.env.example` contain `# GEMINI_API_KEY=your-key-here` etc. — the credential regex requires a *quoted* ≥16-char value, so placeholders don't match. The placeholder-credential judgment call stays OPEN per brief.
(pre-run sanity) | — | — | No root `package.json`, no `hooks/hooks.json`, no `.cursor/`, no binary assets observed — the binary-asset REJECT judgment call is OPEN but not exercised here.

## FP classes nominated

class | rule | minimal fix sketch | est. blast radius
--- | --- | --- | ---
Doc-embedded hook-config examples under `.claude/` | SK-T017 | Restrict the `.claude/` leg of T017's file filter to JSON (`*.json`, or basenames `settings.json`/`settings.local.json`/`hooks.json`); drop the unparseable-markdown fallback for non-JSON files. | Any bundle that documents its own hook setup under `.claude/` (README/CONFIG docs) gets ~1 bogus high-sev "executes bundled content" finding per example line — hits exactly the class of repo this gate is meant to evaluate.
Self-referential harness-dir paths | SK-T010 | When the script lives *inside* the same dotdir it references (file under `.claude/` mentioning `.claude/`), suppress or downgrade; keep severity for `$HOME`/`~`-prefixed paths and for references to a *different* harness's dotdir (the codex→claude case). | Every self-contained `.claude/` bundle (i.e., the standard packaging shape) produces dozens of warning-noise findings that drown the real cross-boundary signal.
hasExec conflates harness-registered hooks with skill-invokable exec | SK-T013 | Count exec surface reachable *from skills* (scripts adjacent to skills or referenced in skill text), not hook commands registered in `settings.json`/`.codex/hooks.json`; alternatively treat `.claude/hooks/` as expected infrastructure, not skill exec. | Every hook+skills bundle without `allowed-tools` becomes an automatic blocker → REJECT by construction for the dominant packaging shape. Largest blast radius of the nominated classes.
T003 fires on content, not filename | SK-T003 | Change file filter to `\.(md|markdown)$` *and* dotdir prefix (path-based), instead of scanning file text for the string `.md`. | `.sh`/`.ts` files under `.claude/` that mention a `.md` filename get mislabeled "instruction-bearing files"; moderate noise wherever scripts reference docs.

## Gate bugs / surprises

1. **`.codex/hooks.json` is invisible to T017 (false negative).** The file
   literally registers bundled commands —
   `.codex/hooks/_codex-adapter.sh` invoked from four lifecycle events — but
   T017's path filter (`hooks/hooks.json` | `.claude/*` | `*settings.json`)
   doesn't match `.codex/hooks.json`. Meanwhile `isHarnessConfig` *does* cover
   it (basename `hooks.json` → T015/T016 scan it). The config surface is
   defined inconsistently across rules; the flagship cross-harness file in
   this target is silently unexamined by the exact rule it should trip.
2. **G001 `refTargets` drops in-bundle `../` refs.** `refTargets` walks the
   file's dir upward and `return nil` — discarding *all* candidates — as soon
   as an ancestor join escapes the root. So `[vite.config.ts](../../vite.config.ts)`
   in `.claude/skills/frontend-dev-guidelines/SKILL.md` resolves to
   `.claude/vite.config.ts` at the file-dir level (in-bundle, dangling —
   no such file) yet is never checked. Escapes are T019's job, but T019's
   own join *also* lands in-bundle, so neither rule covers this ref.
3. **T020 misses the real home-dir writes and flags a non-write.** `setup.ts`
   copies `editor-config/init.lua` → `~/.config/nvim/init.lua` and `vimrc` →
   `~/.vimrc` via `fs.copyFileSync` — genuine persistence-surface writes — but
   `.config/nvim` and `.vimrc` aren't in the persist-path list, so nothing
   fires. Meanwhile `_run-node-hook.sh`'s `source ~/.bashrc >/dev/null`
   (a read + null redirect) does. Worst-case pairing: false negative on the
   real write, false positive on the benign probe.
4. **T008's line-start anchor misses wrapped install commands.** `setup.ts`
   runs `execSync('npm install', …)` and `execSync('chmod +x *.sh', …)` —
   real subprocess installs — but the pattern anchors
   `npm install` to line start, and both occurrences are nested inside
   call expressions / log strings. Coverage gap, not an FP.
5. **Surprise (positive):** G001 caught a real defect in the *target* —
   `backend-dev-guidelines/SKILL.md` ships ~11 resource links missing the
   `resources/` prefix while its sibling skill uses the prefix correctly.
6. **Surprise (design):** `.agents/skills/` is a wholesale mirror (a
   `source-command-verify-setup` skill even contains `bash .Codex/scripts/…`
   — a sloppy `s/claude/Codex/` mirror artifact). The I002 name-collision
   findings are real but the repo *wants* the shadow; worth noting the rule
   has no notion of intentional cross-root mirroring.

**Summary:** Against `diet103/claude-code-infrastructure-showcase`, the gate's
predicted verdict is REJECT driven by one genuine signal class (T017 on
`.claude/settings.json`, ~5 TPs) and one structurally-forced blocker (T013 on
all 9 SKILL.md files, correct per rule but semantically dubious for
harness-registered hooks). The dominant findings are T010 self-referential
`.claude/` mentions (~30–80, mostly FP), T003 dotdir-content warnings
(~20–40, mixed), T017 doc-example FPs (~13), T020 persistence TPs (~5–15),
G001's ~11 real broken links in the target's own docs, and 4 I002 mirror
collisions; the flagship miss is `.codex/hooks.json`, whose bundled-command
registrations no rule examines. All counts are predicted pending an actual
run — every exec call was sandbox-denied in this session; the parent agent
should run the two commands at the top and diff counts before shipping.
