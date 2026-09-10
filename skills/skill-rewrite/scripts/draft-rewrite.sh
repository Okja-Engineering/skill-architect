#!/usr/bin/env bash
set -euo pipefail

target_skill=""
audit_report=""
output=""

usage() {
  cat <<EOF
Usage: draft-rewrite.sh -t <target-skill-dir> [-a <audit-report-path>] [-o <output-path>]

Options:
  -t, --target    Target skill directory to rewrite (required)
  -a, --audit     Path to an existing skill-audit report (markdown or JSON)
  -o, --output    Path for the rewrite draft (default: ./REWRITE-DRAFT-<skill>.md)
  -h, --help      Show this help
EOF
}

require_arg() {
  if [[ $# -lt 2 ]]; then
    usage >&2
    exit 1
  fi
  printf '%s' "$2"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--target)
      target_skill="$(require_arg "$@")"; shift 2 ;;
    -a|--audit)
      audit_report="$(require_arg "$@")"; shift 2 ;;
    -o|--output)
      output="$(require_arg "$@")"; shift 2 ;;
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

skill_name="$(basename "$target_skill")"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
skill_audit_root="$(dirname "$script_dir")/../skill-audit"

if [[ ! -d "$skill_audit_root" ]]; then
  echo "skill-audit not found at $skill_audit_root" >&2
  exit 3
fi

if [[ -z "$audit_report" || ! -f "$audit_report" ]]; then
  if "$skill_audit_root/scripts/audit-report.sh" "$target_skill" > "/tmp/draft-rewrite-audit-${skill_name}.json" 2>/dev/null; then
    audit_report="/tmp/draft-rewrite-audit-${skill_name}.json"
  else
    audit_report="$(mktemp)"
    "$skill_audit_root/scripts/check-frontmatter.sh" "$target_skill" > "$audit_report" 2>&1 || true
    "$skill_audit_root/scripts/check-structure.sh" "$target_skill" >> "$audit_report" 2>&1 || true
  fi
fi

if [[ -z "$output" ]]; then
  output="./REWRITE-DRAFT-${skill_name}.md"
fi

cat > "$output" <<EOF
# Rewrite draft: $skill_name

Generated from audit report: $audit_report
EOF

echo "" >> "$output"
echo "## Current state" >> "$output"
echo "" >> "$output"

if [[ "$audit_report" == *.json ]]; then
  if command -v jq >/dev/null 2>&1; then
    jq -r '.quality.categories[] 
      | select(.score < 10) 
      | "- [ ] \(.name) (score \(.score)): \([.findings[]? | select(.type != "pass") | .message] | join("; "))"' \
      "$audit_report" >> "$output" || true
  else
    echo "- [ ] Audit JSON present but jq not available; review manually." >> "$output"
  fi
else
  awk -F'|' '
    BEGIN { in_table = 0 }
    /^\|[[:space:]]*Dimension[[:space:]]*\|/ { in_table = 1; next }
    in_table && /^\|[-]+/ { next }
    in_table && /^\|/ {
      dim = $2; gsub(/^[[:space:]]+|[[:space:]]+$/, "", dim)
      score = $3; gsub(/^[[:space:]]+|[[:space:]]+$/, "", score)
      notes = $4; gsub(/^[[:space:]]+|[[:space:]]+$/, "", notes)
      if (score ~ /^[0-9]+$/ && score <= 1) {
        printf("- [ ] %s (score %s): %s\n", dim, score, notes)
      }
    }
    in_table && !/^\|/ { in_table = 0 }
  ' "$audit_report" >> "$output"
fi

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

## Notes

- Default ratio for a spec-compliant skill: Deterministic 60%, Orchestration 30%, AI judgment 10%.
- Preserve the skill's original domain knowledge; only restructure how it is disclosed.
- If the skill is doing more than one job, consider splitting it instead of expanding the rewrite.
EOF

echo "Rewrite draft written to: $output"
