# Shared PATH-masking harness for the shell suites.
#
# A masked PATH is a symlink farm of the real PATH minus one binary, so a test
# can observe how a script behaves when a tool it needs is absent. Nothing is
# deleted, moved or uninstalled.
#
# Source this after setting `mask_root` to a writable scratch directory:
#
#   mask_root="$(mktemp -d)"
#   trap 'rm -rf "$mask_root"' EXIT
#   source tests/lib/masked-path.sh

# masked_path <tool>
#
# Echo a PATH holding every binary the real PATH offers except <tool>.
# Cached per tool.
masked_path() {
  local hide="$1"
  local farm="$mask_root/without-$hide"
  if [[ -d "$farm" ]]; then
    echo "$farm"
    return 0
  fi
  mkdir -p "$farm"
  local dirs d f b
  IFS=: read -ra dirs <<< "$PATH"
  for d in "${dirs[@]}"; do
    [[ -d "$d" ]] || continue
    for f in "$d"/*; do
      b="${f##*/}"
      [[ "$b" == "$hide" ]] && continue
      [[ -e "$farm/$b" ]] && continue
      ln -s "$f" "$farm/$b" 2>/dev/null || true
    done
  done
  echo "$farm"
}

# run_on_path <path> <cmd> [args...]
#
# Run a command under a given PATH, capturing stdout in `output`, stderr in
# `errout` and the exit status in `code` — separately, so a --json payload can
# be parsed without the diagnostics mixed into it.
run_on_path() {
  local use_path="$1"
  shift
  local errfile="$mask_root/stderr"
  code=0
  output=$(PATH="$use_path" "$@" 2>"$errfile") || code=$?
  errout="$(cat "$errfile")"
}

# run_masked <tool> <cmd> [args...] — run with <tool> absent from PATH.
run_masked() {
  local hide="$1"
  shift
  run_on_path "$(masked_path "$hide")" "$@"
}

# run_present <cmd> [args...] — the control: the real PATH, nothing hidden.
run_present() {
  run_on_path "$PATH" "$@"
}
