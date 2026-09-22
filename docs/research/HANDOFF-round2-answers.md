# Devin — round 2: my answers, branch discipline, and the pi findings you haven't folded in yet

Good round. Slices 1–3 and the PR-0 repairs are what I wanted. Three things before you continue.

## 1. My answers to your two decisions

**T013 (no `allowed-tools` + bundled scripts ⇒ REJECT).** Keep the rule exactly as it is. Fix our skills, not the gate: add `allowed-tools` to `skills/skill-audit`, `skills/skill-rewrite`, and `skills/skill-gate`, scoped to the tools their scripts actually need. Then make the per-harness frontmatter check (`frontmatter.go`) report `allowed-tools` as *valid* for Claude Code and *silently ignored* for Cursor — that is the per-harness validity dimension doing its job, as an informational finding, not a downgrade of T013. Our own skills must pass our own gate on merit; a rule we re-scope to let ourselves through is a rule nobody should trust.

**skill-rewrite's sibling dependency (T019 `../skill-audit/` escape + 5× G001).** Make skill-rewrite self-contained. Copy into `skills/skill-rewrite/references/` only the subset of the five files it actually reads (ICM says trim, don't mirror), delete the `../` paths, and let the duplication show up as the informational finding it is. Do not invent a cross-skill `depends_on` field no harness honors, and do not baseline a T019 escape — that is the exact class of finding the gate exists for. If after trimming the shared content is large, the right follow-up is a plugin-level `shared/` reference root with a declared, non-escaping path convention; open that as a proposal, don't do it silently.

## 2. Branch discipline — this is the one I need you to change now

You are on `main`, uncommitted, seven commits ahead of origin, nothing pushed. That is the state I told you not to leave things in. Before any more building:

1. Cut a branch from the current tree, e.g. `skillgate/slices-1-3`.
2. Open **PR-0**: `docs/research/**` (untracked, not gitignored) plus the PR-0 repairs (claude_code.go comment, D-6, H3, README/CHANGELOG/RELEASE_NOTES, `.out-of-scope.md`, skill-audit degrade, F04.status.md) — nothing else.
3. Open **PR-1**: the `skillgate/` module + `skills/skill-gate` + `go.work` (slice 1, and the slice 2/3 statics if they can't be split cleanly; say so in the description if not).
4. Push both. I merge; you never do.

Small PRs, each green on its own. If splitting slices 2–3 out of PR-1 costs more than an hour, keep them in and tell me.

## 3. Findings you haven't folded in yet — from `docs/research/pi-integration.md`

`recommendation.md` took 15 tagged `[pi]` edits after your last read. Each of these touches code you've already written; verify against the file, then apply in the PR where it belongs:

| Finding | What it changes in the tree |
|---|---|
| **F14** — a gate audits *before* load, never contains what has loaded; the `bash` read leg is open on every harness | `verdict.go` / report text: any pre-activation claim is scoped to the **read path**; the bash leg is listed in `checks_skipped[]`, never claimed closed |
| **Manifest trap** — a package manifest that lists only executables silently drops the skill that is the interface | Slice-1 acceptance test: after install via both plugin manifests, `skills/skill-gate/SKILL.md` is discoverable; fail the build if not |
| **G0 unit widened skill → package** | `quarantine.go`: the unit of quarantine and provenance pin is the package (a skill dir *or* a plugin/package root), not only a skill dir |
| **T017 / T020 extended** without breaking the 20-rule cap | `tripwire_b.go`: T017 covers extension-style executables declared in a manifest; T020 covers writes into harness state roots (`.pi/`, `~/.pi/agent/trust.json`, `.claude/settings*.json`, `.cursor/`) — extend the existing rules, add none |
| **Cursor ships a stock skills system** (eight roots incl. `~/.claude/skills`, recursive, precedence undocumented) | `frontmatter.go` / SK-I001..I005: Cursor `SKILL.md` conformance and root-collision detection are **ours**; agnix `CUR-*` covers `.mdc` only |
| **Pack E gains two item classes** — injected tool description/guideline text, and tool input schemas | `skillvalidator.go` / the always-on budget: count both classes as separate line items with per-item attribution |
| **New don't-build** — never use a cache-miss detector as a per-skill cost signal | Don't. Cost deltas come only from quantities that change when and only when a skill body enters context |
| **D9** — pi is a bench fixture, not a target | No pi package in this program. Slice 2 gets the always-on acceptance bench (see rec §6) only if it costs < a day; otherwise note it and move on |

## 4. What's coming

A rule-language research pass is running: it inventories SkillSpector's regex analyzers, races three declarative-rule designs against a fixed 15-rule sample, and proposes how our 27 Go rules should be expressed and migrated, plus the 3–4-dimensional impact model and the problem / background / proposed solution / expected outcome audit contract. It lands at `docs/research/rule-language.md` in this repo. **Do not refactor the rule representation until it lands** — finish PR-0 and PR-1 first.

## Order
PR-0 → PR-1 → apply §1 and §3 in PR-1 or a PR-2 → wait for `rule-language.md` before touching how rules are expressed.

## 5. Update — `docs/research/rule-language.md` has landed (both repos, byte-identical)

Read it before touching rule representation. The short version:
- **Canonical rule = a typed Go value** (`Rule{ID, Severity, Effort, Remediation, Origin, Scope, Detect, Ceded, FallbackOf, Expect, Tests}`) over ten matcher constructors and one shared `Ctx`. The "rule language" humans read is the catalogue, SARIF `rules[]`, NOTICE file and fixture tables **generated from** those values. Data files only for the policy profile, catalogue and fixtures. No YAML DSL, no Semgrep engine.
- **Split rules** (`Ceded` + `FallbackOf`) make engine absence a declared, verdict-capping state — the F13 mechanism, formalized.
- **Impact model = four numbers, never blended:** Risk · Debt · Load · Lift, each stamped billed / measured / estimated.
- **Audit contract:** `expected_outcome` is a rule-and-path re-run predicate, so a before/after run confirms it mechanically.
- **RE2 reality:** ~18 SkillSpector pattern sites use lookaround, backreferences, atomic groups, `{0,8192}` repeats or Unicode bidi classes that Go `regexp` cannot express; each has a stated rewrite, post-filter, or "ceded" disposition. Sample fit: 8 of 15 whole, 6 partial, 1 ceded.
- **Do this first, before any rule migrates:** the determinism prerequisites the judges found in the live engine — map-order Tarjan in `refgraph.go` (~line 259) and map ranges in `tripwire_b.go` (~lines 207-209) — plus a permanent repeated-run probe test.
- Section 9 has new decisions for me (ID namespace for ported patterns, page yardstick). Flag, don't decide.
- Correction from the judges' scorecards: **no candidate cleared the 12/15 fidelity bar** (A 10, B 7, C 10); C is the base by soundest shape. Treat every "fits whole" verdict in the sample annex as needing a differential test (Python `re` vs Go RE2 on the same corpus) before it migrates.
- `go test ./...` from the repo root fails (no root module under `go.work`); both modules pass individually. CI already runs them per module — document the per-module command in README, or add a root `Makefile`/script target, so nobody trusts a false red.
