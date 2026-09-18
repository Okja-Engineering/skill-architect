#!/usr/bin/env bash
# Draft a rewritten SKILL.md for a skill, from a skill-audit report.
# Writes REWRITE-DRAFT.md into the target skill directory and names it on
# stdout. The draft is a skeleton plus the audit output for a reader to work
# from; it does not rewrite the skill and does not touch its SKILL.md.
# Exit codes: 0=draft written, 1=usage or input error.
# A usage error is a missing or unknown option, an option given without its
# value, a target directory or SKILL.md that is not there, or a report named
# with -a that is not there. Every one of them is reported on stderr, naming
# the option or the path, and leaves no draft behind.
# The audit's own verdict is not this script's exit status: a skill with
# failing house policy is the case the drafter exists for, so findings in the
# report are a successful run.
set -euo pipefail

target_skill=""
audit_report=""

usage() {
  cat <<EOF
Usage: draft-rewrite.sh -t <target-skill-dir> [-a <audit-report-path>]

Options:
  -t, --target    Target skill directory to rewrite (required)
  -a, --audit     Path to an existing skill-audit report (optional)
  -h, --help      Show this help
EOF
}

# value_of <option> <count> — the option's value, or a usage error naming it.
#
# Reading "$2" directly is what made a mistyped option report `$2: unbound
# variable`: a line of bash internals naming a position in the parser, for a
# caller who needs to be told which option they left empty. The count is passed
# in because `$#` inside a function is the function's own.
value_of() {
  if [[ "$2" -lt 2 ]]; then
    echo "Missing value for $1" >&2
    usage >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--target)
      value_of "$1" "$#"; target_skill="$2"; shift 2 ;;
    -a|--audit)
      value_of "$1" "$#"; audit_report="$2"; shift 2 ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "$target_skill" ]]; then
  usage >&2
  exit 1
fi

if [[ ! -d "$target_skill" ]]; then
  echo "Target skill directory not found: $target_skill" >&2
  exit 1
fi

if [[ ! -f "$target_skill/SKILL.md" ]]; then
  echo "SKILL.md not found in $target_skill" >&2
  exit 1
fi

# A report named with -a is the caller saying what the draft is to be built
# from, so it is validated here with the other inputs rather than tested again
# at the point of use. Folding the two questions into one condition there —
# "was a report given, and is it readable" — made a typo indistinguishable from
# no report at all: the drafter audited afresh, drafted over that instead, and
# reported that no report had been provided.
if [[ -n "$audit_report" && ! -f "$audit_report" ]]; then
  echo "Audit report not found: $audit_report" >&2
  echo "Omit -a to have the structural checks run instead." >&2
  exit 1
fi

skill_name="$(basename "$target_skill")"
output="$target_skill/REWRITE-DRAFT.md"

# skill-audit is skill-rewrite's sibling, so it is resolved by going up from
# this script to the skill directory that holds it and then across. Resolved
# from this script's own location and not from the caller's working directory,
# which is not part of the answer: `CDPATH=` and `--` are what keep it that way,
# for the reason verdict-guard.sh sets out at length.
script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
skill_audit_root="$(dirname "$script_dir")/../skill-audit"

# What the draft says it was built from.
#
# A machine-local mktemp path was the one thing it could not be. It names a
# file that is gone by the time anyone reads the draft — or worse, one that is
# still there, since nothing removed it — and it never meant anything outside
# the process that wrote it. So the two cases state what they actually are: a
# report the caller named is named, and checks the drafter ran itself are
# described as checks the drafter ran itself.
if [[ -z "$audit_report" ]]; then
  provenance="structural checks run by this script: skill-audit's check-frontmatter.sh and check-structure.sh"
  echo "No audit report provided; running structural checks..." >&2
  audit_report="$(mktemp)"
  # The report is this script's scratch and not an artifact, so it does not
  # outlive the run. The trap goes on with the file rather than after the
  # checks, so no exit path between the two — including an errexit abort — can
  # leave it behind.
  trap 'rm -f "$audit_report"' EXIT
  "$skill_audit_root/scripts/check-frontmatter.sh" "$target_skill" > "$audit_report" 2>&1 || true
  "$skill_audit_root/scripts/check-structure.sh" "$target_skill" >> "$audit_report" 2>&1 || true
else
  provenance="audit report $audit_report"
fi

cat > "$output" <<EOF
# Rewrite draft: $skill_name

Generated from: $provenance

## Current state

EOF

cat "$audit_report" >> "$output"

cat >> "$output" <<'EOF'

## Proposed structure

Follow the Agent Skills spec and ICM context-management principles:

- Frontmatter stays small and includes `name`, `description`, `license`, and optional `compatibility`/`metadata`.
- `SKILL.md` body is under ~500 lines.
- Required sections:
  - `## When to use`
  - `## Deterministic actions (60%)`
  - `## Orchestration (30%)`
  - `## AI judgment (10%)`
  - `## Examples`
  - `## Constraints`
- Mechanical work lives in `scripts/`; deep reference material lives in `references/`.

## Missing section templates

EOF

if ! grep -qiE "^#{2,6}[[:space:]]+When to use[[:space:]]*$" "$target_skill/SKILL.md"; then
  cat >> "$output" <<'EOF'
### When to use

Use this skill when:

- <specific scenario 1>
- <specific scenario 2>
- <specific scenario 3>

Do not use this skill when <negative scope>.

EOF
fi

if ! grep -qiE "^#{2,6}[[:space:]]+Examples?[[:space:]]*$" "$target_skill/SKILL.md"; then
  cat >> "$output" <<'EOF'
### Examples

#### Example 1: <scenario>

Input:

```text
<concrete user request>
```

Command:

```bash
<exact command the skill runs>
```

Expected output:

```text
<concrete output>
```

EOF
fi

if ! grep -qiE "^#{2,6}[[:space:]]+Validation" "$target_skill/SKILL.md"; then
  cat >> "$output" <<'EOF'
### Validation checklist

- [ ] <observable pass/fail criterion 1>
- [ ] <observable pass/fail criterion 2>
- [ ] <observable pass/fail criterion 3>

EOF
fi

cat >> "$output" <<'EOF'

## Action items

- [ ] Review the proposed structure against the skill's original intent.
- [ ] Fill in the missing section templates above with domain-specific content.
- [ ] Move mechanical steps to `scripts/` if they are currently described in prose.
- [ ] Move deep reference content to `references/` or `assets/`.
- [ ] Re-run the audit after applying changes.

## Notes

- Default ratio for a spec-compliant skill: Deterministic 60%, Orchestration 30%, AI judgment 10%.
- Preserve the skill's original domain knowledge; only restructure how it is disclosed.
- If the skill is doing more than one job, consider splitting it instead of expanding the rewrite.
EOF

echo "Rewrite draft written to: $output"
