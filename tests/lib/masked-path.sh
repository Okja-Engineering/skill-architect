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
  local dirs d f b abs_d
  IFS=: read -ra dirs <<< "$PATH"
  for d in "${dirs[@]}"; do
    [[ -d "$d" ]] || continue
    # A PATH entry may be relative, and the farm is not the cwd it is relative
    # to. Resolving it here is what keeps the links below pointing at the binary
    # they name.
    #
    # `CDPATH=` and `--` are what make the resolution the caller's environment's
    # to decide and not CDPATH's. With CDPATH exported, `cd bin` searches it
    # first, lands in some other `bin`, and *echoes the directory it chose* —
    # which puts a second line into this value, so every `ln` below names a path
    # that does not exist, the farm comes up empty, and the run the caller built
    # it for dies at exit 127 having tested nothing. That vacuous 127 is the
    # outcome this file exists to prevent; CDPATH is another door into it. `--`
    # keeps an entry beginning with a dash from being read as an option.
    abs_d="$(CDPATH= cd -P -- "$d" && pwd -P)" || continue
    for f in "$d"/*; do
      b="${f##*/}"
      [[ "$b" == "$hide" ]] && continue
      # A name already taken stays taken, whether or not what it points at
      # resolves. `-e` alone is false for a broken symlink, so a dangling entry
      # would fall through to the `ln` below, fail because the name exists, and
      # be left shadowing the working binary a later PATH directory offers.
      [[ -e "$farm/$b" || -L "$farm/$b" ]] && continue
      # Nothing enters the farm under a name that does not resolve — which also
      # drops the literal `dir/*` an empty directory's glob leaves behind. A
      # broken link masks a second binary the caller never asked to hide, and
      # turns the run it was built for into a vacuous exit 127 rather than a
      # test of the script.
      [[ -e "$abs_d/$b" ]] || continue
      ln -s "$abs_d/$b" "$farm/$b" 2>/dev/null || true
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
