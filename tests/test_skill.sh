#!/usr/bin/env bash
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"
harness_init

# The scratch directory belongs to the shell that installed the trap, which is
# the harness. A helper that made its own could be running inside a command
# substitution, where the assignment never escapes and the trap has nothing to
# clean up.
tmp_skill="$harness_scratch/skill"
mkdir -p "$tmp_skill"

manifest_is_valid() {
  python3 - "$1" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'name mismatch'
assert data['version'] == '0.4.2', 'version mismatch'
assert 'skills' in data, 'missing skills'
PY
}

marketplace_is_valid() {
  python3 - <<'PY'
import json
with open('.claude-plugin/marketplace.json') as f:
    data = json.load(f)
assert data['name'] == 'skill-architect', 'marketplace name mismatch'
plugins = data['plugins']
assert len(plugins) == 1, 'expected exactly one plugin entry'
assert plugins[0]['name'] == 'skill-architect', 'plugin entry name mismatch'
assert plugins[0]['version'] == '0.4.2', 'plugin entry version mismatch'
assert plugins[0]['source'] == './', 'plugin entry source mismatch'
PY
}

frontmatter_name_matches_directory() {
  local skill="$1"
  local name_in_file
  name_in_file="$(awk '
    BEGIN { in_fm = 0 }
    /^---$/ {
      if (in_fm) { exit }
      in_fm = 1
      next
    }
    in_fm && /^name:/ { sub(/^name:[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); print; exit }
  ' "skills/$skill/SKILL.md")" || return 1
  [[ "$name_in_file" == "$skill" ]]
}

draft_rewrite_produces_draft() {
  local work="$tmp_skill/skill-rewrite-test"
  cp -R skills/skill-rewrite "$work" || return 1
  skills/skill-rewrite/scripts/draft-rewrite.sh -t "$work" >/dev/null || return 1
  [[ -f "$work/REWRITE-DRAFT.md" ]]
}

# Plugin manifests are valid JSON and version matches.
for manifest in .devin-plugin/plugin.json .claude-plugin/plugin.json .cursor-plugin/plugin.json .codex-plugin/plugin.json; do
  assert "$manifest exists" test -f "$manifest"
  assert "$manifest is valid plugin.json" quietly manifest_is_valid "$manifest"
done

# The marketplace manifest is the fifth version surface. `claude plugin
# marketplace add` reads it, and it carries its own copy of the version, so a
# release that bumps the four plugin.json files and forgets this one advertises
# the previous release to anyone installing by name.
assert ".claude-plugin/marketplace.json exists" test -f .claude-plugin/marketplace.json
assert ".claude-plugin/marketplace.json is valid and at 0.4.2" quietly marketplace_is_valid

# The profiler records its own version in every profile it writes, and it is the
# sixth surface carrying this release's number. It is asserted here beside the
# manifests so one place shows all of them, and in Go by
# TestAdapterVersionIsThisRelease.
assert "profiler AdapterVersion is 0.4.2" \
  grep -q 'AdapterVersion = "0.4.2"' profiler/types.go

# Each skill has a valid SKILL.md with frontmatter and name matching directory.
for skill in skill-audit skill-rewrite; do
  assert "skills/$skill directory exists" test -d "skills/$skill"
  assert "skills/$skill/SKILL.md exists" test -f "skills/$skill/SKILL.md"
  assert "skills/$skill frontmatter name matches directory" \
    frontmatter_name_matches_directory "$skill"
done

# skill-audit carries its scripts and references.
assert "skill-audit check-frontmatter.sh is executable" \
  test -x skills/skill-audit/scripts/check-frontmatter.sh
assert "skill-audit check-structure.sh is executable" \
  test -x skills/skill-audit/scripts/check-structure.sh
assert "skill-audit best-practices reference exists" \
  test -f skills/skill-audit/references/best-practices.md
assert "skill-audit evaluation-matrix reference exists" \
  test -f skills/skill-audit/references/evaluation-matrix.md

# The skill-audit scripts can audit themselves.
assert "skill-audit passes its own frontmatter check" \
  quietly skills/skill-audit/scripts/check-frontmatter.sh skills/skill-audit
assert "skill-audit passes its own structure check" \
  quietly skills/skill-audit/scripts/check-structure.sh skills/skill-audit

# The shipped skills validate clean — warnings included.
#
# skill-audit is the repo's reference skill: it is what the audit skill points
# other skills at, and it invokes this very validator. A warning in our own
# artifact is a defect. Nothing checked for one before, which is how a
# `scripts/lib/` directory earned a "deep nesting detected" warning and carried
# it through every gate unnoticed.
#
# `skill-validator check` exits 0 only on a clean pass; 1 means errors and 2
# means warnings only. Asserting 0 pins both counts at zero.
for skill in skills/skill-audit skills/skill-rewrite; do
  assert "$skill validates with zero errors and zero warnings" \
    quietly skill-validator check "$skill"
done

# skill-rewrite carries its script and can draft a rewrite for itself.
assert "skill-rewrite draft-rewrite.sh is executable" \
  test -x skills/skill-rewrite/scripts/draft-rewrite.sh
assert "skill-rewrite draft-rewrite.sh produces REWRITE-DRAFT.md" \
  quietly draft_rewrite_produces_draft

# --- draft-rewrite.sh does not report success over an audit that did not run ---
#
# It ran both audit checks with `2>&1 || true`, which is two defects in one
# line. The status was discarded, so a check that reached no verdict was
# indistinguishable from one reporting a failing skill, and a draft was written
# and exited 0 either way. And merging stderr into the capture put the checks'
# own diagnostics into the draft's "Current state" section as findings about the
# skill: with skill-validator absent, a clean skill's draft opened with "ERROR:
# skill-validator exited with unexpected status 127" as its current state.
#
# A stub here is one directory holding one file, prepended to the real PATH.
# Nothing is mirrored, replaced or uninstalled.

DRAFT=skills/skill-rewrite/scripts/draft-rewrite.sh

# draft_target — a fresh copy of a real skill to draft over. Each case gets its
# own, so a draft one case wrote cannot satisfy the next.
draft_target() {
  local name="$1"
  local dir="$harness_scratch/draft-$name"
  rm -rf "$dir"
  cp -R skills/skill-rewrite "$dir"
  rm -f "$dir/REWRITE-DRAFT.md"
  printf '%s' "$dir"
}

# draft_stub_path <name> <tool> <exit> — a PATH whose <tool> is useless.
draft_stub_path() {
  local dir="$harness_scratch/draft-stub-$1-$3"
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    printf '#!/usr/bin/env bash\nexit %s\n' "$3" > "$dir/$2"
    chmod +x "$dir/$2"
  fi
  printf '%s:%s' "$dir" "$PATH"
}

# draft_refused <name> <tool> <exit> — the audit cannot answer, so no draft is
# written and the status says so. Run as a captured command: the expected
# diagnostics are noise on a passing run.
draft_refused() {
  local name="$1" tool="$2" tool_exit="$3"
  local target status=0 out
  target="$(draft_target "$name")"
  out="$(PATH="$(draft_stub_path "$name" "$tool" "$tool_exit")" "$DRAFT" -t "$target" 2>&1)" || status=$?
  if [ "$status" -ne 3 ]; then
    printf 'expected exit 3, got %s:\n%s\n' "$status" "$out" >&2
    return 1
  fi
  if [ -f "$target/REWRITE-DRAFT.md" ]; then
    printf 'a draft was written over an audit that reached no verdict:\n' >&2
    cat "$target/REWRITE-DRAFT.md" >&2
    return 1
  fi
  return 0
}

assert "draft-rewrite, the spec source cannot run: exits 3 and writes no draft" \
  draft_refused spec-127 skill-validator 127
assert "draft-rewrite, the spec source answers with a status nobody can read: exits 3 and writes no draft" \
  draft_refused spec-42 skill-validator 42
assert "draft-rewrite, jq is present and useless: exits 3 and writes no draft" \
  draft_refused jq-silent jq 0
assert "draft-rewrite, grep is present and useless: exits 3 and writes no draft" \
  draft_refused grep-err grep 2

# draft_state_is_findings_only — the control side of the same invariant. With a
# healthy toolchain a draft is written, and nothing on its "Current state" is a
# diagnostic about the tooling. The checks' own stderr must not be in there.
draft_state_is_findings_only() {
  local target
  target="$(draft_target healthy)"
  "$DRAFT" -t "$target" >/dev/null 2>&1 || return 1
  [ -f "$target/REWRITE-DRAFT.md" ] || return 1
  if grep -q 'ERROR:' "$target/REWRITE-DRAFT.md"; then
    printf 'the draft carries a tooling diagnostic as the skill state:\n' >&2
    grep -n 'ERROR:' "$target/REWRITE-DRAFT.md" >&2
    return 1
  fi
  return 0
}

assert "draft-rewrite, a healthy toolchain: the draft states findings, not diagnostics" \
  quietly draft_state_is_findings_only

# The guard is a dependency it cannot announce itself, so its absence must read
# as "no draft", like every other unmet precondition.
draft_without_guard() {
  # Not named draft-<something>: draft_target owns that prefix, and a tree
  # sharing a name with a target had the target's `rm -rf` delete it.
  local tree="$harness_scratch/skills-without-guard"
  local target status=0
  rm -rf "$tree"
  cp -R skills "$tree"
  rm -f "$tree/skill-audit/scripts/verdict-guard.sh"
  target="$(draft_target no-guard)"
  "$tree/skill-rewrite/scripts/draft-rewrite.sh" -t "$target" >/dev/null 2>&1 || status=$?
  [ "$status" -eq 3 ] || return 1
  [ ! -f "$target/REWRITE-DRAFT.md" ]
}

assert "draft-rewrite, verdict-guard.sh absent: exits 3 and writes no draft" \
  draft_without_guard

# It leaves no temporary audit behind, on the path that writes a draft or on the
# path that refuses to. The refusing path is new, and a new exit path that
# leaked a file each time would be this change's own doing.
# The temporary directory is a private one, not the shared /tmp: mktemp honours
# TMPDIR, and counting entries in a directory other processes also write to is
# a test that fails when something unrelated happens to run beside it.
draft_leaves_no_temp() {
  local target left
  local tmphome="$harness_scratch/draft-tmphome"
  rm -rf "$tmphome"
  mkdir -p "$tmphome"

  target="$(draft_target leak-ok)"
  TMPDIR="$tmphome" "$DRAFT" -t "$target" >/dev/null 2>&1 || return 1

  target="$(draft_target leak-refused)"
  TMPDIR="$tmphome" PATH="$(draft_stub_path leak-refused skill-validator 127)" \
    "$DRAFT" -t "$target" >/dev/null 2>&1 || true

  left="$(find "$tmphome" -type f | wc -l | tr -d '[:space:]')"
  if [ "$left" != 0 ]; then
    printf 'the draft left %s temporary file(s) behind:\n' "$left" >&2
    find "$tmphome" -type f >&2
    return 1
  fi
  return 0
}

assert "draft-rewrite leaves no temporary audit behind, on either path" \
  quietly draft_leaves_no_temp

# The templates it offers are decided by what the *body* holds. A skill whose
# section headings sit at column 0 inside its YAML frontmatter has none of them
# in its body, and used to be handed a draft with no templates at all.
draft_reads_the_body() {
  local target="$harness_scratch/draft-bleed"
  rm -rf "$target"
  mkdir -p "$target"
  cp tests/fixtures/f01/frontmatter-bleed/SKILL.md "$target/SKILL.md"
  "$DRAFT" -t "$target" >/dev/null 2>&1 || return 1
  grep -q '^### When to use' "$target/REWRITE-DRAFT.md" \
    && grep -q '^### Examples' "$target/REWRITE-DRAFT.md"
}

assert "draft-rewrite, headings only in frontmatter: offers the templates the body lacks" \
  quietly draft_reads_the_body

harness_summary
