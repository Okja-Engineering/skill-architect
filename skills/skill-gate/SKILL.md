---
name: skill-gate
description: Run the skillgate safety gate against an Agent Skill bundle (local directory or git URL) before the skill is installed or invoked. Use when vetting first-party or third-party skills, reviewing a skill PR, or checking whether a skill is safe to let run. Produces a report/v1.json (or SARIF) verdict — APPROVE, CAUTION, or REJECT — beside a full coverage ledger.
license: MIT
compatibility: "Go toolchain or a built skillgate binary; POSIX shell for examples. Optional: skillspector (Python 3.12+ venv) for the deep advisory scan."
allowed-tools: Read, Bash
metadata:
  version: "0.1.0"
  carrier: installed-binary
  carrier-command: skillgate
  carrier-source: ./skillgate/cmd/skillgate
---

# skill-gate

Wraps the `skillgate` Go binary (slice 1: the safety gate — G0 quarantine, G1 coverage ledger, G2 tripwire Pack B, G3 SkillSpector shell-out, G7 verdict).

## Contract

- Verdicts are `APPROVE` / `CAUTION` / `REJECT`. The tool never says "safe" or "clean".
- Every file in the bundle gets a terminal ledger outcome — inspected (with SHA-256) or skipped with a named reason.
- Any skipped check or uninspected file caps the verdict at CAUTION; on a fetched (untrusted) target, an incomplete ledger REJECTs.
- A baseline may suppress a finding, but only with a mandatory written `reason` and only while the finding's content fingerprint is unchanged — drift fails closed.
- SkillSpector, agnix, and skill-validator are opt-in and advisory only: their findings report at true severity but can never force REJECT. The block set is owned by the 20-rule Go tripwire pack, which needs no Python and no Rust.
- `report.tokens` carries the always-on context budget: exact o200k_base counts per file via skill-validator, plus `items[]` — per-item char-exact attribution for skill description lines, manifest-declared tool descriptions, and tool input schemas (the classes a file list misses). Never presented as billed usage.
- The audited unit is the package, not the skill dir: a manifest may ship `skills` beside executable `extensions`, and manifest-declared executables are the bundled-hook class (SK-T017).
- F14 — the gate audits before load and never contains what has loaded. Every claim is scoped to the read path and the moment before load; `preactivation-bash-leg` is a standing entry in `checks_skipped` because the bash leg is open on every harness. CAUTION is the reachable ceiling today.
- The gate never executes the target bundle. Remote targets are cloned into a quarantine dir with git hooks disabled.

## Usage

Binary on PATH:

```sh
skillgate gate <dir-or-git-url> [-o report.json] [--format json|sarif] [--baseline b.json] [--fail-on-incomplete] [--skip-checks a,b] [--only a,b]
skillgate gate --list-checks
```

Flags may be written before or after the target; the report is the same either
way. `--` ends flag parsing, for a target whose name begins with `-`.

`--list-checks` prints every registered check and the rules it runs, and exits;
it is where the names `--skip-checks` and `--only` accept come from, and a name
matching none of them is refused. Whatever `--only` leaves out is named in
`checks_skipped`, so a narrowed run never reads as a full one.

From the repo checkout:

```sh
(cd skillgate && go run ./cmd/skillgate gate <dir>)
```

Exit codes: `0` = APPROVE/CAUTION · `1` = REJECT or incomplete ledger under `--fail-on-incomplete` · `2` = the gate itself failed.

## Interpreting the report

- `verdict` + `coverage` + `checks_skipped[]` are the summary contract — read them together, never the verdict alone.
- `ledger[]` shows every file's outcome and hash; `findings[]` carry `rule_id`, `severity`, `source`, `fingerprint`, and optional `advisory`/`suppressed` flags.
- `SS-*` findings are SkillSpector advisories (opt-in). `SK-T001`–`SK-T020` are the deterministic tripwire floor.
- `provenance.sha256_manifest` is the bundle digest — re-gate if it changes (rug-pull detection lands with baselines per bundle).

## Optional deep scan

Install SkillSpector into a venv and put it on PATH before invoking the gate:

```sh
python3.12 -m venv /tmp/ss-venv
/tmp/ss-venv/bin/pip install "skillspector @ git+https://github.com/NVIDIA-AI-Blueprints/skillspector"
PATH=/tmp/ss-venv/bin:$PATH skillgate gate <dir>
```

If the binary is absent, `checks_skipped` names `skillspector` and the verdict caps at CAUTION — nothing fails silently.

agnix (conformance: CC/Cursor/MCP/AGENTS.md rules) and skill-validator (token budget + spec errors) follow the same contract: on PATH → they run and report advisory findings; absent → named in `checks_skipped`, verdict capped at CAUTION.
