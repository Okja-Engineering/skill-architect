#!/usr/bin/env bash
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

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--target)
      target_skill="$2"; shift 2 ;;
    -a|--audit)
      audit_report="$2"; shift 2 ;;
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
output="$target_skill/REWRITE-DRAFT.md"

# Resolve skill-audit scripts relative to this script.
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
skill_audit_root="$(dirname "$script_dir")/../skill-audit"

if [[ -z "$audit_report" || ! -f "$audit_report" ]]; then
  echo "No audit report provided; running structural checks..." >&2
  audit_report="$(mktemp)"
  "$skill_audit_root/scripts/check-frontmatter.sh" "$target_skill" > "$audit_report" 2>&1 || true
  "$skill_audit_root/scripts/check-structure.sh" "$target_skill" >> "$audit_report" 2>&1 || true
fi

cat > "$output" <<EOF
# Rewrite draft: $skill_name

Generated from audit report: $audit_report

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
