# The shipped skills, and the declarations their frontmatter carries.
#
# This lives in tests/lib/ rather than in one suite because two suites now ask
# the same questions of the same files, and a per-suite copy of a reader is the
# drift tests/lib/harness.sh exists to end — the same reader written twice
# answers differently the moment one copy is repaired. tests/test_skill.sh
# witnesses each skill's declared carrier; tests/test_gate.sh builds the carrier
# skill-gate declares and runs it. Both need the census and the reader, and
# neither of them owns it.
#
# Nothing here asserts. Which declarations are *valid* is a suite's question and
# the two suites ask different ones: test_skill.sh holds every skill to the
# carrier contract, and test_gate.sh only needs to find the source path so it
# has something to build. A reader that refused an unexpected value here would
# take that judgement away from both of them.

# The shipped skills, derived from the tree rather than written down.
#
# This is the denominator both suites walk, and it is the reason the walk
# exists: a compatibility line and a claim-shape refusal were both built for one
# of the two skills shipped at the time and pinned to that skill's own SKILL.md,
# so the identical sentence next door was un-held and went on being false. A
# written-down pair reproduces that the day a third skill lands — and a third
# one has landed since.
shipped_skills() {
  local d
  for d in skills/*/; do
    d="${d%/}"
    [ -f "$d/SKILL.md" ] || continue
    printf '%s\n' "${d##*/}"
  done
}

skill_md_of() { printf 'skills/%s/SKILL.md' "$1"; }

# metadata_value <skill> <key> -- a key nested one level under `metadata:` in
# the skill's frontmatter. Nothing, rather than an error, when it is absent: an
# absent declaration is a verdict a suite reports, not a reason to abort.
#
# The file is read here rather than through skillgate's own frontmatter reader
# on purpose. A suite that read the tree with the program under test would agree
# with it by construction and assert nothing about it.
metadata_value() {
  awk -v want="$2" '
    NR == 1 && $0 == "---" { fm = 1; next }
    fm && $0 == "---" { exit }
    fm && /^metadata:[[:space:]]*$/ { in_md = 1; next }
    in_md && /^[^[:space:]]/ { in_md = 0 }
    in_md {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      key = line
      sub(/:.*$/, "", key)
      if (key == want) {
        val = line
        sub(/^[^:]*:[[:space:]]*/, "", val)
        gsub(/^"|"$/, "", val)
        print val
        exit
      }
    }
  ' "$(skill_md_of "$1")"
}

# What carries a skill's mechanical work: `bundled-scripts` or `installed-binary`
# today. The tag only; the binary kind's payload (`carrier-command`,
# `carrier-source`) is read with metadata_value by whoever needs it.
carrier_of() { metadata_value "$1" carrier; }
