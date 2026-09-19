#!/usr/bin/env bash
set -euo pipefail
. "$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/lib/harness.sh"

# This file's own path, resolved here because `harness_init` changes directory.
# One control below reads this suite's source text rather than its behaviour:
# "the destination is never evaluated by a shell" is a claim about an
# implementation, and a sweep of constructs can only sample it.
suite_source="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)/$(basename "${BASH_SOURCE[0]}")"

harness_init

# The suite for the documented install routes.
#
# This exists because the install instructions were written and never run. The
# manual standalone copy was the documented *update* path — README tells the
# reader they "pull updates when you choose" — and `cp -R src dst` copies
# *into* dst once dst exists, so the second run of the documented command
# nested a second SKILL.md inside the installed skill and any harness that
# walks the skills root recursively registered the skill twice.
#
# So the assertions below do not check that the README says a particular
# thing. They **extract the command the README documents and run it**, twice,
# and require the destination to end up identical to the source. That is the
# invariant: the documented install command is an install *and* an update, so
# running it again must converge on the source rather than accumulate, and no
# failure of any step may leave the destination worse than it started. Any
# command that satisfies that passes — the test is pinned to the behaviour, not
# to the particular command that happens to be documented today.
#
# Nothing here can touch a live config, and that is a property of how the block
# is run rather than an assertion about its text. See the containment section
# below: an earlier version of this suite rewrote the destination with `sed` and
# checked the rewrite with `grep`, which is a guard that holds only for the
# spellings someone thought of — and it reported its verdict through `assert`,
# which reports and returns, so the suite ran `rm -rf` against a live skills
# directory after its own guard had already said no.

install_scratch="$harness_scratch/install"
mkdir -p "$install_scratch"

# The bash the suite itself is running under. The block is a documented shell
# command and this project supports two shells, so a block that behaves
# differently on 3.2 has to be seen on 3.2 — `bash` off PATH would quietly run
# every extraction under whichever one is first there.
suite_bash="${BASH:-bash}"

# --- What the repository ships, as the denominator for everything below -------
#
# The skills are enumerated from the repository rather than named, so a third
# skill is covered the day it lands. Nothing below knows how many there are.

repository_skill_names() {
  local dir
  for dir in "$harness_repo_root"/skills/*/; do
    [ -f "$dir/SKILL.md" ] || continue
    dir="${dir%/}"
    printf '%s\n' "${dir##*/}"
  done
}

repository_ships_skills() {
  [ -n "$(repository_skill_names)" ]
}

# --- The block the README documents ------------------------------------------

# The commands the README documents under "Manual standalone copy", read out of
# the README itself so the suite cannot drift from the document it is proving.
#
# Heredoc bodies and fenced blocks elsewhere in the README are not candidates:
# the scan starts at the heading and takes the first bash block after it.
documented_manual_copy_block() {
  awk '
    /^### Manual standalone copy$/ { found = 1; next }
    found && /^```bash$/ { in_block = 1; next }
    in_block && /^```$/ { exit }
    in_block { print }
  ' README.md
}

# The block is a block, and it names every skill the repository ships. An
# extractor that silently returned nothing, or a block that named only one of
# the skills, would make "installs and converges" true of a script that
# installed nothing. The name is looked for on its own rather than as
# `skills/<name>`, because where the block says it — spelled into a path, or
# listed once for a loop to walk — is the block's business and not this check's.
documented_block_is_extractable() {
  local block name
  block="$(documented_manual_copy_block)" || return 1
  [ -n "$block" ] || return 1
  while read -r name; do
    [ -n "$name" ] || continue
    printf '%s\n' "$block" | grep -qF "$name" || return 1
  done <<BLOCK_SKILLS
$(repository_skill_names)
BLOCK_SKILLS
  return 0
}

# --- Containment, decided after resolution ------------------------------------
#
# The block runs with HOME and the working directory both inside the harness
# scratch root, with an environment that carries nothing else, and under
# `set -u`. None of that is containment on its own: a redirected home and a
# working directory inside the root are two places to climb out of, and what
# used to stand here was the claim that an absolute path and a `~name` were the
# whole set of ways out. They were not. `$HOME/../../../..`, `../../../..`, a
# climb to the filesystem root followed by an absolute tail, and a destination
# whose every character is innocent but whose path runs through a symlink all
# passed that guard; one of them installed a full skills tree outside the
# scratch root and one of them deleted a decoy home's files first. A list of
# spellings cannot be a containment proof, because the next spelling is not on
# it, and the claim of completeness is what let that survive a gate.
#
# So the verdict is the resolved write, and it is taken where the write is:
# `run_documented_block` does not write the block out or execute it until the
# destination it will use has been expanded *in that run's own environment* and
# resolved — `.` and `..` folded away, symlinks followed through the part of the
# path that exists — and shown to land strictly inside the harness scratch root.
# The climbs, the absolute path and the symlink all resolve to somewhere outside
# and are refused by where they land rather than by how they look — each handed
# to the resolved verdict on its own, below, with the shape pass out of the way,
# so that what refuses them is not in doubt. `~name` is the one exception and it
# is named there: nothing here reads the password database, so a `~name`
# destination is refused for being unexpandable rather than for where it lands.
#
# "Expanded" does not mean "evaluated", and that distinction is the whole of the
# fourth round of this defect. The expansion is a substitution over text with no
# shell behind it, because for three rounds it was a shell behind a screen of
# the expansion shapes someone had thought of, and the fourth round found the
# category the screen had never contained. See the sweep section below.
#
# The shape pass below is the other fence, and it is not the lesser one. The
# resolved verdict can only decide the *destination*, because that is the one
# word whose value is fixed before the block runs; `"$skills_dir/$skill"` has no
# value until the loop binding `$skill` is running, so there is nothing to
# resolve and the question has to be asked of the name instead. So the two
# fences divide the block between them: resolution decides where the write
# lands, and the shape pass decides every word resolution cannot reach. What it
# demands is stated the other way round from the old list — not the escapes it
# knows, but that a word be accountable: no absolute path, no `~name`, no `..`
# in any position, and no expansion whose value this suite cannot name. Only
# HOME, PWD and the names the block itself binds are accountable; anything else,
# including a command substitution, is refused. PATH is deliberately not on that
# list: the run provides it, its value is absolute, and nothing an install block
# does is resolved from it.
#
# Where that fence carries load is the words the resolution cannot reach, and
# only there — which is a narrower claim than the one that stood here. It used
# to be the thing that made the resolution safe to reach, because the
# resolution expanded the destination with a shell and a destination carrying a
# substitution had to be refused before it got there. It is not that any more:
# the expansion cannot run anything, so the destination is fenced by the
# expansion and the resolution, and every rule in the shape pass would still
# refuse a bad *destination* with the rule removed. That is why each rule is
# measured on a word that is not the destination
# (`containment_refuses_every_unaccountable_word_away_from_the_destination`);
# a control that handed a rule a destination would now pass without it.
# The shape pass still runs first, but as ordering rather than as a fence in
# front of a shell.
#
# Nothing above is left standing on its own word. Every mechanism either
# paragraph names has a control below that hands it something it must refuse,
# on the seam where it is the only thing that can refuse it, and reverting the
# mechanism reddens that control — which is the difference between this and
# what stood here before, where the argument *was* the claim. The residue is
# named rather than implied: resolution decides where the destination lands, the
# expansion decides whether the destination can be read at all, and the shape
# pass decides every other word by name. A destination the shape pass admits
# and the expansion cannot read is refused with the character it could not
# account for named, so "nothing to expand" is a reported refusal and not a
# silent one.
#
# All of these are `require`s, not `assert`s: a failed precondition has to stop
# the suite, because a reported failure is not a refusal.

text_names_nothing_outside_a_redirected_home() {
  awk '
    BEGIN { sq = sprintf("%c", 39) }

    # Where a comment starts on a line, which is a question about quoting and
    # not about the first "#" on it. This used to stop scanning the line at its
    # first "#" token, so a "#" inside a quoted string hid every word after it
    # — including an absolute path — from the pass that exists to find them.
    function uncommented(s,   out, i, c, q, prev) {
      q = ""
      out = ""
      prev = " "
      for (i = 1; i <= length(s); i++) {
        c = substr(s, i, 1)
        if (q == "") {
          if (c == "\"" || c == sq) {
            q = c
          } else if (c == "#" && (prev == " " || prev == "\t")) {
            return out
          }
        } else if (c == q) {
          q = ""
        }
        out = out c
        prev = c
      }
      return out
    }

    # Why a word is not accountable, or "" when it is.
    function unaccountable(t,   rest, k, name) {
      if (t ~ /^\//) {
        return "an absolute path, which ignores a redirected HOME"
      }
      if (t ~ /^~[^\/]/) {
        return "a tilde with a user name, which expands from the password database and not from HOME"
      }
      if (t ~ /(^|\/)\.\.(\/|$)/) {
        return "a parent-directory climb, which leaves the directory it is resolved from"
      }
      if (index(t, "`") > 0) {
        return "a command substitution, whose value this pass cannot name"
      }
      rest = t
      while ((k = index(rest, "$")) > 0) {
        rest = substr(rest, k + 1)
        if (substr(rest, 1, 1) == "{") {
          if (match(rest, /^\{[A-Za-z_][A-Za-z0-9_]*\}/) == 0) {
            return "an expansion whose value this pass cannot name"
          }
          name = substr(rest, 2, RLENGTH - 2)
        } else {
          if (match(rest, /^[A-Za-z_][A-Za-z0-9_]*/) == 0) {
            return "an expansion whose value this pass cannot name"
          }
          name = substr(rest, 1, RLENGTH)
        }
        rest = substr(rest, RLENGTH + 1)
        if (!(name in bound)) {
          return "an expansion of $" name ", which nothing in the block binds"
        }
      }
      return ""
    }

    { line[NR] = uncommented($0) }

    END {
      # The names the block binds itself, collected before anything is judged,
      # because a word may use a name the line below it assigns. HOME and PWD
      # are bound by the run and are the redirected ones.
      bound["HOME"] = 1
      bound["PWD"] = 1
      for (i = 1; i <= NR; i++) {
        n = split(line[i], w, /[[:space:]]+/)
        for (j = 1; j <= n; j++) {
          t = w[j]
          gsub(/["]/, "", t)
          gsub(sq, "", t)
          if (t ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
            sub(/=.*$/, "", t)
            bound[t] = 1
          } else if (t == "for" && j < n) {
            bound[w[j + 1]] = 1
          }
        }
      }
      for (i = 1; i <= NR; i++) {
        n = split(line[i], w, /[[:space:]]+/)
        for (j = 1; j <= n; j++) {
          t = w[j]
          if (t == "") continue
          gsub(/["]/, "", t)
          gsub(sq, "", t)
          sub(/^[A-Za-z_][A-Za-z0-9_]*=/, "", t)
          if (t == "") continue
          why = unaccountable(t)
          if (why != "") {
            printf "%d: %s: %s\n", i, why, w[j]
            bad++
          }
        }
      }
      exit (bad > 0)
    }
  '
}

block_names_nothing_outside_a_redirected_home() {
  documented_manual_copy_block | text_names_nothing_outside_a_redirected_home
}

# --- The resolved verdict -----------------------------------------------------

# <path> — fold `.` and `..` away without touching the filesystem, so that a
# destination which does not exist yet still has an answer. `..` at the root
# stops there, the way the kernel stops it, which is what makes a climb with an
# absolute tail resolve to that tail.
path_normalized() {
  printf '%s\n' "$1" | awk '
    {
      n = split($0, part, "/")
      out = ""
      depth = 0
      for (i = 1; i <= n; i++) {
        p = part[i]
        if (p == "" || p == ".") continue
        if (p == "..") {
          if (depth > 0) {
            sub(/\/[^\/]*$/, "", out)
            depth--
          }
          continue
        }
        out = out "/" p
        depth++
      }
      if (out == "") out = "/"
      print out
    }
  '
}

# <absolute path> — the same path with symlinks followed through the part of it
# that exists. Two reasons it is not a plain `cd -P`: the destination is a place
# the block is about to create and need not exist yet, and on this platform the
# scratch root arrives through /var and lives at /private/var, so a compare
# against an unresolved path says "outside" for a path that is in fact inside.
path_resolved() {
  local abs head tail
  abs="$(path_normalized "$1")"
  head="$abs"
  tail=""
  while [ "$head" != / ] && [ ! -d "$head" ]; do
    if [ -n "$tail" ]; then
      tail="${head##*/}/$tail"
    else
      tail="${head##*/}"
    fi
    head="${head%/*}"
    if [ -z "$head" ]; then
      head=/
    fi
  done
  head="$(CDPATH= cd -P -- "$head" 2>/dev/null && pwd -P)" || return 1
  if [ -n "$tail" ]; then
    printf '%s\n' "${head%/}/$tail"
  else
    printf '%s\n' "$head"
  fi
}

# <path> <root> — path *resolves* to somewhere strictly inside root, whether or
# not it exists yet. Equal to the root is not inside it.
path_resolves_inside() {
  local resolved root
  resolved="$(path_resolved "$1")" || return 1
  root="$(path_resolved "$2")" || return 1
  case "$resolved" in
    "$root"/?*) return 0 ;;
  esac
  return 1
}

# <run dir> <destination text> — what the destination expands to in the
# environment the block runs in, or nothing with the reason on stderr, which is
# itself a refusal.
#
# Text in, text out. There is no process here that could run what the text
# says, and that is the design rather than a hardening of it — see the section
# on the sweep below for why a screen in front of an `eval` cannot be the
# safety argument. Three expansions are implemented: a leading `~` with no user
# name, `$HOME`, and `${HOME}`. Everything else has to be a character that is
# inert in a path. A destination that needs more than this is refused, and the
# refusal names what it could not account for.
#
# Kept to one command on purpose: `the_destination_expansion_uses_no_shell`
# reads this body and requires that it reaches no interpreter, so anything here
# that forks a shell — including a command substitution to tidy the output — is
# a control failure rather than a style question.
expanded_destination() {
  DESTINATION_TEXT="$2" DESTINATION_HOME="$1/home" awk '
    function refuse(why) {
      printf "the destination is not one this suite can expand without a shell: %s\n", why \
        > "/dev/stderr"
      exit 1
    }
    BEGIN {
      text = ENVIRON["DESTINATION_TEXT"]
      home = ENVIRON["DESTINATION_HOME"]
      n = length(text)
      if (n == 0) refuse("there is nothing to expand")
      out = ""
      i = 1
      # A tilde, and only at the front of the word. Further in, a shell expands
      # one in some positions and not in others, and "in some positions" is not
      # something implemented here.
      if (substr(text, 1, 1) == "~") {
        if (n == 1 || substr(text, 2, 1) == "/") {
          out = home
          i = 2
        } else {
          refuse("a tilde with a user name is read from the password database")
        }
      }
      while (i <= n) {
        c = substr(text, i, 1)
        if (c == "$") {
          if (substr(text, i + 1, 6) == "{HOME}") {
            out = out home
            i += 7
            continue
          }
          # $HOME and not $HOMEX: the name has to end where the expansion does.
          if (substr(text, i + 1, 4) == "HOME" && substr(text, i + 5, 1) !~ /^[A-Za-z0-9_]$/) {
            out = out home
            i += 5
            continue
          }
          refuse("the $ at character " i " begins something other than $HOME or ${HOME}")
        }
        if (c !~ /^[A-Za-z0-9._\/-]$/) {
          refuse("the character at " i " is [" c "], which is not one of the inert characters a path may be built from here")
        }
        out = out c
        i += 1
      }
      print out
      exit 0
    }
  ' </dev/null
}

# <run dir> <destination text> — the absolute, resolved path the block would
# write to, or nothing if it cannot be established, which is itself a refusal.
resolved_destination() {
  local dir text expanded nl
  dir="$1"
  text="$2"
  expanded="$(expanded_destination "$dir" "$text")" || return 1
  [ -n "$expanded" ] || return 1
  nl='
'
  # One path, or it is not a destination this suite will let anything run
  # against.
  case "$expanded" in
    *"$nl"*) return 1 ;;
  esac
  case "$expanded" in
    /*) ;;
    *) expanded="$dir/cwd/$expanded" ;;
  esac
  path_resolved "$expanded"
}

# <run dir> <destination text> — the verdict that matters.
destination_resolves_inside_the_scratch_root() {
  local resolved
  resolved="$(resolved_destination "$1" "$2")" || return 1
  if ! path_resolves_inside "$resolved" "$harness_scratch"; then
    printf 'the destination resolves outside the harness scratch root:\n  %s\n  resolves to %s\n  which is not inside %s\n' \
      "$2" "$resolved" "$harness_scratch" >&2
    return 1
  fi
  return 0
}

# The containment decision, in one place, so that what the controls below
# measure and what the run sites ask are the same question with one definition.
#
# <run dir> <block text> — may this block be run in this run's environment?
#
# Three fences. The shape pass over every word, which is the only thing that can
# reach the words with no value yet. Then the destination, read out of the block
# the way a shell reads a word, and expanded without one — a destination this
# suite cannot expand is refused there. Then the resolved verdict on what it
# expanded to, which is the proof: it is decided by where the write lands.
#
# The order is no longer a safety property. It was, while the expansion was a
# shell: the shape pass had to refuse a substitution before the resolution
# reached it, and for one revision it did not, and a substitution written into
# the README ran on the machine doing the verifying. The expansion cannot run
# anything now, so the second and third fences are safe to reach in any order,
# and the shape pass is first because reporting the broadest failure first is
# the more useful diagnostic.
#
# A block with no destination assignment is refused rather than waved through.
# There is then nothing to resolve, and "nothing to resolve" is not a licence
# to run.
containment_verdict() {
  local dir block dest
  dir="$1"
  block="$2"
  printf '%s\n' "$block" | text_names_nothing_outside_a_redirected_home || return 1
  dest="$(printf '%s\n' "$block" | first_assignment_value)"
  if [ -z "$dest" ]; then
    echo "the block assigns no destination, so there is nothing to resolve" >&2
    return 1
  fi
  destination_resolves_inside_the_scratch_root "$dir" "$dest" || return 1
  return 0
}

# A run directory for the decisions that are taken before anything is run: the
# same shape and the same depth under the scratch root as the staged runs
# below, so a decision taken in it is the decision a real run gets.
#
# Its home holds a symlink to the checkout, because a reader's home commonly
# does, and because a destination can leave the scratch root without a single
# suspicious character in it — `~/checkout/skills` is a path whose shape says
# nothing and whose resolution says everything.
containment_probe_dir() {
  local dir
  dir="$install_scratch/containment-probe"
  if [ ! -d "$dir/home" ]; then
    mkdir -p "$dir/home" "$dir/cwd"
    ln -s "$harness_repo_root" "$dir/home/checkout"
  fi
  printf '%s\n' "$dir"
}

# A block in the shape of the documented one with its destination respelled.
# The controls differ from the README's block in exactly the one thing the
# README asks a reader to change, because a wrong destination is what a reader
# — or an author editing that line — actually produces.
block_with_destination() {
  printf 'skills_dir=%s\n' "$1"
  printf 'mkdir -p "$skills_dir"\n'
  printf 'rm -rf "$skills_dir/skill-audit" && cp -R "skills/skill-audit" "$skills_dir/skill-audit"\n'
}

# The control for the decision. A decision that cannot refuse is not a proof of
# anything, so every escape containment has to exclude is handed to it and the
# refusal is required — before anything is run, because this is what the
# containment argument rests on.
containment_refuses() {
  if containment_verdict "$(containment_probe_dir)" "$1" >/dev/null 2>&1; then
    printf 'the containment decision accepted a block it must refuse:\n%s\n' "$1" >&2
    return 1
  fi
  return 0
}

containment_refuses_an_absolute_destination() {
  containment_refuses "$(block_with_destination '/Users/you/.claude/skills')"
}

containment_refuses_a_tilde_with_a_user_name() {
  containment_refuses "$(block_with_destination '~root/.claude/skills')"
}

# The climbs. A redirected HOME and a working directory inside the scratch root
# are not containment on their own: both are places to climb out of, and both
# were climbed out of while every check here passed.
containment_refuses_a_climb_out_of_a_redirected_home() {
  containment_refuses \
    "$(block_with_destination '$HOME/../../../../ESCAPED-THE-SCRATCH-ROOT/.claude/skills')"
}

containment_refuses_a_climb_out_of_the_working_directory() {
  containment_refuses \
    "$(block_with_destination '../../../../ESCAPED-THE-SCRATCH-ROOT/.claude/skills')"
}

# Enough climbs to reach the filesystem root, where `..` stops, and then an
# absolute tail. This is the one that reaches a named home directory, and with
# a real user's name in it, the live skills directory inside it.
containment_refuses_a_climb_that_ends_in_an_absolute_path() {
  containment_refuses "$(block_with_destination \
    '$HOME/../../../../../../../../../../../../../../../Users/decoy-operator/.claude/skills')"
}

# And the one no reading of the text can catch: a destination with nothing
# wrong in it that resolves outside the scratch root anyway, because something
# on the way is a symlink.
containment_refuses_a_destination_that_resolves_through_a_symlink() {
  containment_refuses "$(block_with_destination '~/checkout/skills')"
}

containment_refuses_a_block_whose_second_line_escapes() {
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit /etc/codex/skills/skill-audit'
}

# --- What the shape pass alone decides ----------------------------------------
#
# The resolved verdict cannot reach any of these: they are words that are not
# the destination, so there is nothing for it to resolve. Each was a mechanism
# the section above asserted and nothing measured, which is the same defect as
# the claim of completeness it replaced — a sentence doing the work a control
# should do.
#
# The two destination-shaped controls in here are honest about what they now
# hold. A substitution in the destination is refused twice over — by this pass
# and by the expansion, which cannot read a `$(` at all — so the control below
# states the *whole decision's* behaviour, and the control that isolates this
# pass's substitution rules is the away-from-the-destination one further down.

# The scan used to stop at a line's first `#` token, so anything after one went
# uninspected. A `#` is only a comment at the start of a word, and a `#` that is
# its own word *inside* a quoted string is not one — the text after it is code,
# and here it is an absolute path.
containment_refuses_an_escape_after_a_quoted_hash() {
  containment_refuses 'skills_dir=~/.claude/skills
echo "installing # into the skills directory" && cp -R skills/skill-audit /etc/codex/skills/skill-audit'
}

# The other half of that rule, and the half nothing measured. `uncommented`
# ends a line at a `#` only when a space or a tab is in front of it, and that
# test had no control: the one above puts its `#` inside quotes, so the quoting
# branch decides it and the `prev` test is never reached. Reverting the `prev`
# test — letting any `#` end the line — left this suite at 43 passed, 0 failed
# on both shells, which is a control asserting something it does not measure.
#
# What the test is actually for: a `#` is a comment only where a word begins.
# Mid-word it is an ordinary character, so `x#../..` is one word naming a climb
# and not a comment plus nothing, and a reader that stopped at the `#` would
# have read `x`. This is on a line that is not the destination, so the
# resolution never looks at it.
containment_refuses_a_climb_behind_a_mid_word_hash() {
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit x#../../../../../../ESCAPED-THE-SCRATCH-ROOT/skill-audit'
}

# A destination whose value is only known by running it, and the one control
# the ordering of the two fences exists for. Being refused is not enough here:
# the resolution expands the destination with a shell, so a substitution that
# gets as far as the resolution has already had its effect, and a verdict of
# "refused" arrives after the damage. So what is measured is the marker, not
# only the verdict — the substitution's one effect is to create a file, and the
# file must not exist.
#
# Both spellings are handed over, and each isolates one rule: `$(…)` is refused
# as an expansion this pass cannot name, and a backtick has no `$` in it at all,
# so nothing but the backtick rule can catch it. The effect is written as a
# redirection rather than an argument so the destination stays a single word,
# which is how everything here reads "the first assignment's value".
substitution_left_no_marker() {
  if [ -e "$1" ]; then
    printf 'a substitution in the destination ran before it was refused:\n  %s exists\n' "$1" >&2
    return 1
  fi
  return 0
}

containment_refuses_a_substitution_before_it_can_run() {
  local dir marker
  dir="$(containment_probe_dir)"
  marker="$dir/a-substitution-ran"

  rm -f "$marker"
  containment_refuses "$(block_with_destination \
    "\$(id>$marker)/.claude/skills")" || return 1
  substitution_left_no_marker "$marker" || return 1

  rm -f "$marker"
  containment_refuses "$(block_with_destination \
    "\`id>$marker\`/.claude/skills")" || return 1
  substitution_left_no_marker "$marker" || return 1

  return 0
}

# A name the block never binds, on a line that is *not* the destination. The
# resolution never looks there, so the shape pass is the only thing that can
# refuse it — which is what makes this a control on the shape pass rather than
# on the expansion, which refuses the destination form anyway.
containment_refuses_an_unbound_name_away_from_the_destination() {
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit "$SKILLS_BACKUP/skill-audit"'
}

# Every remaining rule in the shape pass, each handed something on a line that
# is *not* the destination — and that placement is the whole point of this
# control.
#
# Since the expansion stopped being a shell, the destination is no longer where
# the shape pass carries load: take its `~name` rule out and a `~name`
# destination is still refused, by the expansion; take its `..` rule out and a
# `..` destination is still refused, by where it resolves. Controls that hand
# those rules a *destination* therefore pass with the rule gone, which is the
# non-discriminating shape this cluster keeps producing. Where the shape pass
# is the only fence is every other word in the block, because
# `"$skills_dir/$skill"` has no value until the loop is running and there is
# nothing to resolve. So the rules are measured there.
#
# The absolute-path rule already has such a control of its own, in
# `containment_refuses_a_block_whose_second_line_escapes`, and so does the
# unbound-name rule directly above; these are the four that did not.
containment_refuses_every_unaccountable_word_away_from_the_destination() {
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit ~root/.claude/skills/skill-audit' || return 1
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit ../../../../ESCAPED-THE-SCRATCH-ROOT/skill-audit' || return 1
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit "$(id)/skill-audit"' || return 1
  containment_refuses 'skills_dir=~/.claude/skills
cp -R skills/skill-audit "`id`/skill-audit"' || return 1
  return 0
}

# A block with nothing to resolve is not a block that may run. Waving one
# through is how "the destination is contained" becomes true of a block that has
# no destination and does whatever it likes.
#
# The verdict is not enough to measure that, and this control used to ask only
# for the verdict. A block with no destination hands the expansion an empty
# text, and the expansion refuses an empty text on its own account — so taking
# the refusal in `containment_verdict` out left the suite 43/0 and the control
# printing PASS with the mechanism it is named for gone. That is the same shape
# as the rest of this cluster: a control whose name claims one fence and whose
# evidence is another.
#
# So what is required is the named cause and not only the refusal. The two
# refusals are about different things and both are worth having: one says this
# *block* names no destination, which is a fact about the document, and the
# other says this *text* cannot be expanded, which is a fact about a string.
# Pinning to the first is what makes removing it visible, and a decision that
# fails closed while naming the wrong cause is the failure mode this suite
# already refuses elsewhere.
containment_refuses_a_block_that_assigns_no_destination() {
  local block why
  block='mkdir -p skill-audit
cp -R skills/skill-audit skill-audit'
  containment_refuses "$block" || return 1
  why="$(containment_verdict "$(containment_probe_dir)" "$block" 2>&1 >/dev/null)" || :
  case "$why" in
    *"assigns no destination"*) return 0 ;;
  esac
  printf 'the block was refused, but not as a block that assigns no destination:\n  %s\n' \
    "$why" >&2
  return 1
}

# --- The resolved verdict, measured on its own --------------------------------
#
# The invariant is that containment is decided by where the write lands and not
# by how the destination is spelled. That is a claim about the resolved verdict
# *without* the shape pass in front of it, so it is asked of the resolved
# verdict directly. Every spelling that escaped the old guard is handed to it
# here, and each is refused with the shape pass out of the way — which is what
# says the fence holding the escapes out is the resolution and not a denylist
# wearing new words.
#
# One of them is refused for a different reason and it is called out rather than
# counted in: `~root` does not resolve outside the scratch root, it does not
# resolve at all, because nothing here reads the password database. It is in
# this list because the old guard let it through, not because the resolution is
# what decides it. The four climbs and the symlink are the ones that carry the
# invariant, and they carry it on where they land.
resolution_refuses() {
  if destination_resolves_inside_the_scratch_root \
       "$(containment_probe_dir)" "$1" >/dev/null 2>&1; then
    printf 'the resolved verdict accepted a destination it must refuse:\n  %s\n' "$1" >&2
    return 1
  fi
  return 0
}

# The other direction, because a verdict that refuses everything proves nothing.
# A literal the suite owns, not the README's text: handing the document's own
# words to this seam is what the fence ordering above exists to prevent.
resolution_accepts_an_ordinary_destination() {
  destination_resolves_inside_the_scratch_root \
    "$(containment_probe_dir)" '~/.claude/skills'
}

resolution_refuses_an_absolute_destination() {
  resolution_refuses '/Users/you/.claude/skills'
}

resolution_refuses_a_tilde_with_a_user_name() {
  resolution_refuses '~root/.claude/skills'
}

resolution_refuses_a_climb_out_of_a_redirected_home() {
  resolution_refuses '$HOME/../../../../ESCAPED-THE-SCRATCH-ROOT/.claude/skills'
}

resolution_refuses_a_climb_out_of_the_working_directory() {
  resolution_refuses '../../../../ESCAPED-THE-SCRATCH-ROOT/.claude/skills'
}

resolution_refuses_a_climb_that_ends_in_an_absolute_path() {
  resolution_refuses \
    '$HOME/../../../../../../../../../../../../../../../Users/decoy-operator/.claude/skills'
}

resolution_refuses_a_destination_that_resolves_through_a_symlink() {
  resolution_refuses '~/checkout/skills'
}

# The climb that leaves through a part of the path which does not exist yet, and
# the reason `.` and `..` are folded away before the filesystem is consulted.
# Following symlinks can only start from the longest prefix that exists, so a
# `..` beyond that point is still in the string when the two paths are compared
# — and a string beginning with the scratch root compares as inside it however
# far out of it the path actually goes. Folding first is what makes the compare
# mean what it says, and this is the only control that can tell.
resolution_refuses_a_climb_through_a_path_that_does_not_exist_yet() {
  resolution_refuses \
    '~/not-created-yet/../../../../../../ESCAPED-THE-SCRATCH-ROOT/.claude/skills'
}

# --- The destination is expanded, and evaluating it is not how ----------------
#
# Deciding where a destination lands means knowing what its text means, and
# "means" used to be answered by handing the text to a shell:
# `eval "printf %s\n $text"`, the text unquoted, screened by the shape pass
# above. That arrangement was closed one spelling at a time for three rounds —
# a `$(…)`, a backtick, an arithmetic expansion — and every round left its
# shape untouched: a list of the expansion shapes someone had thought of,
# standing in front of a shell. The fourth round found the category the list had
# never contained. `x;id>FILE` carries no expansion at all, so all five rules
# passed it, and the `eval` honoured the `;` and the `>` — `id` ran on the
# machine doing the verifying, inside the decision whose whole purpose is
# refusing unsafe input, with every check below printing PASS.
#
# A screen in front of an `eval` cannot be the safety argument, for the same
# reason a list of escape spellings could not be the containment argument: it is
# complete only about what its author enumerated, and the next category is never
# on it. So the shell is gone from here, and the list is inverted. The
# expansions a destination actually needs are three — a leading `~` with no user
# name, `$HOME`, and `${HOME}` — and those three are performed below as
# substitution over text. Every other character has to be one that is inert in a
# path: a letter, a digit, `.`, `_`, `-`, `/`. A separator, a redirection, a
# pipe, a quote, a backslash, a brace, a glob character, a newline, a `#`, a `$`
# in front of any other name — none of these is refused for being on a list of
# dangerous things. They are refused because expanding them is not something
# this suite implements, so the category nobody thought of is refused by
# default rather than found by a reviewer. A destination that cannot be expanded
# without a shell is a destination this suite refuses.
#
# The cost is stated rather than hidden. A legitimate destination spelled with a
# character outside that set is refused too, and the refusal names the
# character, so widening the set is a deliberate edit and not a reflex. `~name`
# is no longer read from the password database and `$OTHER` is no longer an
# unbound-variable error inside a subshell: both are refusals here now, the same
# verdict reached by a mechanism that cannot execute. `$PWD` is not implemented
# either — the shape pass accounts for it in the words resolution cannot reach,
# but no destination needs it, and an expansion nothing needs is surface.

# The one effect every spelling in the sweep below is written to have: a file
# that cannot exist unless something ran.
expansion_marker() {
  printf '%s\n' "$(containment_probe_dir)/the-expansion-ran"
}

# <destination text> <marker> — the construct in the text did not run.
#
# Not redundant with the verdict, and this is the lesson of all four rounds: a
# construct that reaches a shell has already had its effect by the time any
# verdict is reported, so a control that asked only "was it refused?" printed
# PASS through every one of them. It did.
expansion_ran_nothing() {
  if [ -e "$2" ]; then
    printf 'the destination text was evaluated — the construct in it ran:\n  destination: %s\n  it left:     %s\n' \
      "$1" "$2" >&2
    rm -f "$2"
    return 1
  fi
  return 0
}

# <destination text> — the resolution refuses this destination, and nothing in
# it ran.
#
# Asked of the resolution *alone*, with the shape pass out of the way, because
# the resolution is the seam where the expansion happens. While the shape pass
# refuses a spelling first, a shell standing behind it is invisible — which is
# exactly how three rounds of closing the shape pass left the shell in place.
# The same spellings are put through the whole decision as well, further down.
expansion_refuses() {
  local dir marker
  dir="$(containment_probe_dir)"
  marker="$(expansion_marker)"
  rm -f "$marker"
  if destination_resolves_inside_the_scratch_root "$dir" "$1" >/dev/null 2>&1; then
    printf 'the resolution accepted a destination carrying a shell construct:\n  %s\n' "$1" >&2
    rm -f "$marker"
    return 1
  fi
  expansion_ran_nothing "$1" "$marker" || return 1
  return 0
}

# Command lists. Each of these makes one word carry two commands, which is the
# category the five-shape screen never contained.
expansion_refuses_every_command_separator() {
  local m
  m="$(expansion_marker)"
  expansion_refuses "x;id>$m" || return 1
  expansion_refuses "x&&id>$m" || return 1
  expansion_refuses "x||id>$m" || return 1
  expansion_refuses "x&id>$m" || return 1
  expansion_refuses "$(printf 'x\nid>%s' "$m")" || return 1
  return 0
}

# Redirections. `>` and `>>` and `<>` create their target, so for those the
# marker is left by `printf` itself rather than by a second command; `<` can
# only read, so its proof is the refusal and the creating forms above carry the
# effect. A pipe is in here rather than with the separators because what it
# does to this seam is take the expansion's own output away.
expansion_refuses_every_redirection() {
  local m
  m="$(expansion_marker)"
  expansion_refuses "x>$m" || return 1
  expansion_refuses "x>>$m" || return 1
  expansion_refuses "x<>$m" || return 1
  expansion_refuses "x2>$m" || return 1
  expansion_refuses "x<$m" || return 1
  expansion_refuses "x|id>$m" || return 1
  return 0
}

# The category the three previous rounds closed, re-asked here of a mechanism
# that cannot run them rather than of a screen that had to recognise them.
# `${HOME:-/etc}` is in the sweep because an expander that matched `${HOME` and
# stopped looking would accept it.
expansion_refuses_every_substitution() {
  local m
  m="$(expansion_marker)"
  expansion_refuses "\$(id>$m)/.claude/skills" || return 1
  expansion_refuses "\`id>$m\`/.claude/skills" || return 1
  expansion_refuses 'x$((1+1))/.claude/skills' || return 1
  expansion_refuses 'x$OTHER/.claude/skills' || return 1
  expansion_refuses 'x${OTHER}/.claude/skills' || return 1
  expansion_refuses 'x$1/.claude/skills' || return 1
  expansion_refuses 'x$HOMEX/.claude/skills' || return 1
  expansion_refuses '${HOME:-/etc}/.claude/skills' || return 1
  return 0
}

# Everything else a shell does to a word: brace expansion, globbing, quote
# removal, escaping, word splitting, and a tilde anywhere but the front. None
# of these runs a command, and none of them is harmless — each one makes the
# value the resolution judged different from the value the block will use,
# which is the other half of the same defect.
expansion_refuses_every_other_shell_construct() {
  expansion_refuses 'x{a,b}/.claude/skills' || return 1
  expansion_refuses 'x*/.claude/skills' || return 1
  expansion_refuses 'x?/.claude/skills' || return 1
  expansion_refuses 'x[ab]/.claude/skills' || return 1
  expansion_refuses 'x"y"/.claude/skills' || return 1
  expansion_refuses "x'y'/.claude/skills" || return 1
  expansion_refuses 'x\;/.claude/skills' || return 1
  expansion_refuses 'x y/.claude/skills' || return 1
  expansion_refuses "$(printf 'x\ty/.claude/skills')" || return 1
  expansion_refuses 'x~root/.claude/skills' || return 1
  return 0
}

# A `#` is a comment only where a word begins. `x#;id>FILE` is one word, so the
# `#` in it is an ordinary character, the `;` after it is a separator, and a
# reader of the text who stops at the first `#` has read a different
# destination from the one the shell will use.
expansion_refuses_a_construct_after_a_hash() {
  local m
  m="$(expansion_marker)"
  expansion_refuses "x#;id>$m" || return 1
  expansion_refuses "x #;id>$m" || return 1
  return 0
}

# The same sweep through the whole decision, which is the path the README's own
# destination takes: a respelled line in the document, extracted, judged, and —
# if judged contained — run. This is where the defect was live, so this is where
# it has to be refused with nothing having run.
#
# Only the single-word spellings are here. A destination carrying whitespace
# cannot reach this seam as one value, because the extractor stops at
# whitespace; those spellings are swept at the resolution above, which is the
# seam that would be handed one.
containment_refuses_a_destination_that_runs_a_command() {
  local dir marker text
  dir="$(containment_probe_dir)"
  marker="$(expansion_marker)"
  for text in \
    "x;id>$marker" \
    "x&&id>$marker" \
    "x||id>$marker" \
    "x&id>$marker" \
    "x|id>$marker" \
    "x>$marker" \
    "x>>$marker" \
    "x<>$marker" \
    "x#;id>$marker" \
    "\$(id>$marker)/.claude/skills" \
    "\`id>$marker\`/.claude/skills"; do
    rm -f "$marker"
    if containment_verdict "$dir" "$(block_with_destination "$text")" >/dev/null 2>&1; then
      printf 'the containment decision accepted a block whose destination runs a command:\n  %s\n' \
        "$text" >&2
      rm -f "$marker"
      return 1
    fi
    expansion_ran_nothing "$text" "$marker" || return 1
  done
  return 0
}

# The other direction, because a function that refuses everything expands
# nothing. The three expansions this suite implements, each required to produce
# exactly the substitution, and two inert texts required back unchanged. A shell
# would agree with every line here — that is the point. These pin what the
# expansion is *for*; the sweep above pins what it must not do.
expansion_is() {
  local got
  got="$(expanded_destination "$1" "$2")" || return 1
  [ "$got" = "$3" ] && return 0
  printf 'the destination expanded to the wrong value:\n  %s\n  expanded to %s\n  expected    %s\n' \
    "$2" "$got" "$3" >&2
  return 1
}

the_expansion_substitutes_home_and_nothing_else() {
  local dir
  dir="$(containment_probe_dir)"
  expansion_is "$dir" '~/.claude/skills' "$dir/home/.claude/skills" || return 1
  expansion_is "$dir" '$HOME/.claude/skills' "$dir/home/.claude/skills" || return 1
  expansion_is "$dir" '${HOME}/.claude/skills' "$dir/home/.claude/skills" || return 1
  expansion_is "$dir" '~' "$dir/home" || return 1
  expansion_is "$dir" '.claude/skills' '.claude/skills' || return 1
  expansion_is "$dir" '../sibling/skills' '../sibling/skills' || return 1
  return 0
}

# A destination no filesystem could hold, which until now was *accepted* — and
# accepting it is what costs. Both the shape pass and the expansion build their
# result a character at a time, so both are quadratic in the length of what
# they accept: the whole decision took 25 seconds on a 200,000-character
# destination and 101 seconds on 400,000, which puts a megabyte past any CI
# timeout. A suite killed by a timeout prints no summary, and "no summary" is
# the silent-abort failure this harness exists to refuse — so an input that
# cannot be a path is refused rather than chewed on.
#
# 4096 is PATH_MAX on the more generous of the two platforms this runs on;
# macOS stops at 1024. A destination longer than that cannot be created, so
# refusing it costs nothing real, and the refusal names the length the way
# every other refusal here names its cause.
#
# Measured on the invariant and not on the bound: what must hold is that an
# unusable length is refused and an ordinary one is not. The exact limit is
# free to move.
a_destination_no_filesystem_could_hold() {
  awk 'BEGIN { s = ""; while (length(s) < 5000) s = s "aaaaaaaaaaaaaaaaaaaa"; print s }' </dev/null
}

the_expansion_refuses_a_destination_no_filesystem_could_hold() {
  local dir long
  dir="$(containment_probe_dir)"
  long="$(a_destination_no_filesystem_could_hold)"
  if expanded_destination "$dir" "~/$long" >/dev/null 2>&1; then
    printf 'the expansion accepted a destination of %s characters, which no filesystem can hold\n' \
      "${#long}" >&2
    return 1
  fi
  # And through the whole decision, because that is the path a destination in
  # the document takes, and the cost is the decision's cost.
  if containment_verdict "$dir" "$(block_with_destination "~/$long")" >/dev/null 2>&1; then
    printf 'the containment decision accepted a destination of %s characters\n' "${#long}" >&2
    return 1
  fi
  # The other direction: an ordinary destination is still accepted.
  expanded_destination "$dir" '~/.claude/skills' >/dev/null || return 1
  return 0
}

# <name> — the text of a function defined in this suite, read out of this
# suite's own source.
suite_function_body() {
  awk -v name="$1" '
    $0 == name "() {" { inside = 1; next }
    inside && $0 == "}" { exit }
    inside { print }
  ' "$suite_source"
}

# The structural half: this reads `expanded_destination`'s own source text and
# requires that the body cannot reach an interpreter. The sweep above can only
# refuse the spellings written into it, and this does not depend on the
# spelling of the *destination* at all.
#
# It used to be described here as "the one control that does not depend on
# anyone having thought of the construct", and that was false in both of its
# dimensions. It was a substring denylist, so it depended on the spelling
# exactly as much as the sweep did — ten of eleven alternate interpreter
# spellings went past it. And it reads one function, so it depends on the
# placement too. It is inverted now, which closes the spelling dimension; the
# placement dimension is stated below and stays open.
#
# Its boundary, measured rather than asserted: the check has controls of its
# own, directly below, that hand it bodies which do reach an interpreter and
# require refusal, and bodies which do not and require acceptance. A check that
# can only read its own subject cannot be measured, and that is how a false
# description of it survived a gate.
the_destination_expansion_uses_no_shell() {
  local body
  body="$(suite_function_body expanded_destination)"
  if [ -z "$body" ]; then
    printf 'expanded_destination was not found in %s, so this control measured nothing\n' \
      "$suite_source" >&2
    return 1
  fi
  body_reaches_no_interpreter "$body"
}

# <body text> — the check itself, taken as a function of a body so that it can
# be measured on bodies other than the one it is pointed at. While it could
# only read its own subject it was unfalsifiable: it printed PASS, and whether
# it would print FAIL for a body that did reach a shell was nobody's evidence.
#
# Inverted, the way the expansion itself was inverted, and for the same reason.
# What stood here was a substring denylist of ten interpreter spellings, and it
# missed ten of the eleven alternate spellings tried against it — including
# `system ("x")` with a space before the paren, which the BSD awk this platform
# ships executes, and an output pipe, which reaches a shell whatever the
# command string is spelled as and even when it is composed at run time.
#
# So this does not ask what the body must not contain. It asks whether the body
# is the one shape this check can account for, and refuses everything else:
#
#   - exactly one command, an `awk`, with a single-quoted program and nothing
#     before or after it but literal assignments;
#   - that program reads from /dev/null, so no text reaches its stdin either;
#   - inside the program, every call is one of the string and control builtins
#     named below — a call this check cannot account for is refused whether or
#     not anyone has heard of it;
#   - no `|` in any position other than as half of a `||`, because the other
#     end of a pipe is a command however it is spelled;
#   - the only redirection target is "/dev/stderr".
#
# The cost is the same cost the expansion pays and it is the same trade: a
# legitimate rewrite of `expanded_destination` outside that shape is refused
# too, and the refusal names what it could not account for, so widening this is
# a deliberate edit.
#
# Its remaining boundary, and this is a real one: it is about *a body*, and the
# control below points it at one function of the nine the document's text flows
# through. A recurrence that put an interpreter in a different function is not
# in its reach. That is a placement claim, not a spelling claim, and it is the
# half that stays open.
body_reaches_no_interpreter() {
  INTERPRETER_CHECK_BODY="$1" awk '
    function fail(why) {
      printf "the body may reach an interpreter: %s\n", why > "/dev/stderr"
      bad = 1
    }

    function trimmed(s) {
      sub(/^[[:blank:]]+/, "", s)
      sub(/[[:blank:]]+$/, "", s)
      return s
    }

    # <line> — the line with double-quoted literals and any comment removed, so
    # that a construct is judged on the code it is and not on a string that
    # merely mentions one. A single quote cannot open a literal here: the whole
    # program is inside a single-quoted shell word.
    function without_literals(s,   out, i, c, inq) {
      out = ""
      inq = 0
      for (i = 1; i <= length(s); i++) {
        c = substr(s, i, 1)
        if (inq) {
          if (c == bs) { i++; continue }
          if (c == dq) { inq = 0 }
          continue
        }
        if (c == dq) { inq = 1; continue }
        if (c == "#") return out
        out = out c
      }
      return out
    }

    # Asked of the raw line, because the target is a literal and the point is
    # which file it names.
    function redirection_target_is_stderr(s,   t, k, rest) {
      t = s
      while ((k = index(t, ">")) > 0) {
        rest = substr(t, k + 1)
        sub(/^[[:blank:]>]+/, "", rest)
        if (substr(rest, 1, 1) == dq) {
          if (match(rest, /^"[^"]*"/) == 0) return 0
          if (substr(rest, 1, RLENGTH) != dq "/dev/stderr" dq) return 0
        }
        t = substr(t, k + 1)
      }
      return 1
    }

    # The name of the first call this check cannot account for, or "". The
    # blanks are why: `system ("x")` is a call, and a scan for "system(" is not
    # a scan for calls.
    function first_unaccounted_call(s,   t, tok) {
      t = s
      while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:blank:]]*\(/)) {
        tok = substr(t, RSTART, RLENGTH)
        sub(/[[:blank:]]*\($/, "", tok)
        if (!(tok in allowed)) return tok
        t = substr(t, RSTART + RLENGTH)
      }
      return ""
    }

    # `||` is logical or. Every other `|` is a pipe, and `|&` is one too.
    function has_a_pipe(s,   i) {
      for (i = 1; i <= length(s); i++) {
        if (substr(s, i, 2) == "||") { i++; continue }
        if (substr(s, i, 1) == "|") return 1
      }
      return 0
    }

    BEGIN {
      dq = sprintf("%c", 34)
      sq = sprintf("%c", 39)
      bs = sprintf("%c", 92)
      split("refuse print printf exit length substr index match split sub gsub sprintf toupper tolower if while for do else return", a, " ")
      for (k in a) allowed[a[k]] = 1

      n = split(ENVIRON["INTERPRETER_CHECK_BODY"], line, "\n")

      opened = 0
      closed = 0
      for (i = 1; i <= n; i++) {
        if (opened == 0) {
          if (length(line[i]) >= 5 && substr(line[i], length(line[i]) - 4) == "awk " sq) opened = i
        } else if (closed == 0) {
          if (substr(trimmed(line[i]), 1, 1) == sq) closed = i
        }
      }
      if (opened == 0 || closed == 0) {
        fail("it is not one awk program in a quoted word, which is the only shape this check can account for")
        exit 1
      }

      for (i = 1; i < opened; i++) {
        if (trimmed(line[i]) != "") fail("a command runs before the awk program: " trimmed(line[i]))
      }
      for (i = closed + 1; i <= n; i++) {
        if (trimmed(line[i]) != "") fail("a command runs after the awk program: " trimmed(line[i]))
      }

      # The command word is awk, and everything in front of it is a literal
      # assignment: no substitution, no second command, no other interpreter.
      pre = trimmed(substr(line[opened], 1, length(line[opened]) - 5))
      m = split(pre, word, /[[:blank:]]+/)
      for (i = 1; i <= m; i++) {
        if (word[i] == "") continue
        if (word[i] !~ /^[A-Za-z_][A-Za-z0-9_]*="[^"]*"$/) {
          fail("the awk command is preceded by something other than a literal assignment: " word[i])
        } else if (index(word[i], sq) > 0 || index(word[i], "`") > 0 || index(word[i], "$(") > 0) {
          fail("an assignment in front of the awk command substitutes: " word[i])
        }
      }

      post = trimmed(substr(trimmed(line[closed]), 2))
      if (post != "</dev/null") {
        fail("the awk program does not read from /dev/null: [" post "]")
      }

      for (i = opened + 1; i < closed; i++) {
        code = without_literals(line[i])
        tok = first_unaccounted_call(code)
        if (tok != "") fail("the awk program calls " tok "(), which this check cannot account for")
        if (has_a_pipe(code)) fail("the awk program routes through a pipe, and the other end of a pipe is a command: " trimmed(line[i]))
        if (index(code, "`") > 0) fail("the awk program names a backtick")
        if (index(code, "$(") > 0) fail("the awk program names a command substitution")
        if (index(code, sq) > 0) fail("the awk program closes its own quoted word")
        if (!redirection_target_is_stderr(line[i])) fail("the awk program redirects somewhere other than /dev/stderr: " trimmed(line[i]))
      }

      exit (bad != 0)
    }
  ' </dev/null
}

# <injected awk statement> — a body in the shape of `expanded_destination`'s,
# carrying one statement inside the awk program.
a_body_whose_awk_says() {
  printf '  DESTINATION_TEXT="$2" DESTINATION_HOME="$1/home" awk %s\n' "'"
  printf '    BEGIN {\n'
  printf '      text = ENVIRON["DESTINATION_TEXT"]\n'
  printf '      %s\n' "$1"
  printf '      print text\n'
  printf '    }\n'
  printf "  %s </dev/null\n" "'"
}

# <injected shell line> — the same, carrying one line in the shell layer
# outside the awk program.
a_body_whose_shell_says() {
  printf '  %s\n' "$1"
  printf '  DESTINATION_TEXT="$2" DESTINATION_HOME="$1/home" awk %s\n' "'"
  printf '    BEGIN { print ENVIRON["DESTINATION_TEXT"] }\n'
  printf "  %s </dev/null\n" "'"
}

# The control for the check, and the reason this round exists. The check used
# to be a substring denylist of ten interpreter spellings, and it was described
# in this file as the one control that "does not depend on anyone having
# thought of the construct" — which was false twice over. It is a scan for
# spellings, so every spelling nobody wrote down is missed; and it reads one of
# the nine functions the document's text flows through, so every placement
# outside that one function is missed.
#
# These are the spellings it missed, each of which reaches an interpreter, and
# each is required to be refused. `system ("x")` with a space before the paren
# is executed by the BSD awk this platform ships; an output pipe reaches a
# shell whatever the command string is spelled as, including one composed at
# run time from the environment, which is how a live shell was shipped at 43
# passed, 0 failed with both shell-detection controls green.
the_interpreter_check_refuses_every_escape_it_can_see() {
  local spelling
  while IFS= read -r spelling; do
    [ -n "$spelling" ] || continue
    if body_reaches_no_interpreter "$(a_body_whose_awk_says "$spelling")" 2>/dev/null; then
      printf 'the interpreter check accepted an awk body that reaches an interpreter:\n  %s\n' \
        "$spelling" >&2
      return 1
    fi
  done <<'AWK_ESCAPES'
system("touch " text)
system ("touch " text)
system	("touch " text)
print "touch " text | "/bin/sh"
print "touch " text | "sh"
print "touch " text | "/bin/zsh"
print "touch " text | "/usr/bin/perl -e $x"
cmd = "/bin/sh"; print "touch " text | cmd
print "touch " text | ENVIRON["PWD"] "/mysh"
print "touch " text |& "/bin/sh"
"id" | getline text
printf "%s", text > "/etc/codex/skills"
AWK_ESCAPES

  while IFS= read -r spelling; do
    [ -n "$spelling" ] || continue
    if body_reaches_no_interpreter "$(a_body_whose_shell_says "$spelling")" 2>/dev/null; then
      printf 'the interpreter check accepted a shell layer that reaches an interpreter:\n  %s\n' \
        "$spelling" >&2
      return 1
    fi
  done <<'SHELL_ESCAPES'
eval "printf %s $2"
x=$(id)
x=`id`
/bin/sh -c "$2"
python3 -c "import os; os.system('id')"
id >&2
. ./helper.sh
SHELL_ESCAPES
  return 0
}

# The other direction, because a check that refuses every body proves nothing
# about the one it is for: the body this suite actually ships has to pass, and
# a body doing the same legitimate things a different way has to pass too.
the_interpreter_check_accepts_a_body_that_only_substitutes() {
  body_reaches_no_interpreter "$(suite_function_body expanded_destination)" || return 1
  body_reaches_no_interpreter "$(a_body_whose_awk_says \
    'if (length(text) == 0 || substr(text, 1, 1) == "~") printf "%s\n", "no" > "/dev/stderr"')" \
    || return 1
  return 0
}

# The precondition the whole containment argument rests on, and it is now a
# question about the block. What stood here compared the install scratch path
# with the harness scratch path it had just been built from — true by
# construction of the line that made it, never once looking at the block, while
# carrying the name of the thing that mattered.
#
# It is the same decision the run sites take, on the README's own block, in a
# probe run of the same shape and depth as the staged runs below, taken early so
# that it halts the suite before anything is run. Deliberately the same
# `containment_verdict` and not a second path to the resolved verdict: the
# shape pass is what refuses a destination carrying a substitution, and going
# straight to the resolution would hand the README's text to a shell that the
# shape pass had not seen yet.
the_documented_block_is_contained() {
  containment_verdict "$(containment_probe_dir)" "$(documented_manual_copy_block)"
}

# --- Running the block, contained --------------------------------------------

# A home the block installs into, and a working directory it runs from. The
# working directory holds a copy of the repository's skills tree rather than the
# repository itself, so a block that wrote to its source would write to the copy
# — and so that a run from a directory *without* the tree can be staged just by
# not making it.
staged_run_dir() {
  local name="$1"
  local dir="$install_scratch/$name"
  rm -rf "$dir"
  mkdir -p "$dir/home" "$dir/cwd"
  printf '%s\n' "$dir"
}

stage_the_repository_skills() {
  cp -R "$harness_repo_root/skills" "$1/cwd/skills"
}

run_documented_block() {
  local dir script block
  dir="$1"
  script="$dir/documented-block.sh"
  # Containment, at the site of the run and for this run's environment. The
  # block is not written out and not executed until its destination has been
  # shown to resolve inside the harness scratch root, so a run that escapes is
  # not a run that is reported — it is a run that does not happen. A run site
  # added later inherits this instead of having to remember it.
  #
  # Read once, then judged and run: what is executed is the text that was
  # judged, and not a second reading of the document it came from.
  block="$(documented_manual_copy_block)" || return 1
  [ -n "$block" ] || return 1
  quietly containment_verdict "$dir" "$block" || return 1
  printf '%s\n' "$block" > "$script" || return 1
  (
    cd "$dir/cwd" || exit 1
    env -i HOME="$dir/home" PATH="$PATH" "$suite_bash" -euo pipefail "$script"
  )
}

# --- The invariant ------------------------------------------------------------

# Where the block put things, discovered rather than assumed. The suite does not
# need to know the path the README documents: it needs the installed tree to be
# the repository's tree, wherever the README says to put it.
installed_skill_dirs() {
  find "$1" -name SKILL.md | sed 's|/SKILL\.md$||' | sort
}

# The destination root, read back from where the block actually installed. Named
# from the repository's own skill list, so it is discovered and not spelled.
installed_destination_root() {
  local home="$1" first
  first="$(repository_skill_names | head -1)"
  [ -n "$first" ] || return 1
  find "$home" -type d -name "$first" | head -1 | sed "s|/$first\$||"
}

# What is installed is what is in the repository: every shipped skill present,
# one manifest each however many times the command has run, and byte-identical
# contents. `diff -r` catches a file the update failed to refresh and a file
# deleted upstream that the update left behind; the manifest count catches the
# nested copy, which adds a SKILL.md inside a skill that is otherwise right.
every_repository_skill_is_installed_intact() {
  local dest="$1" name
  [ -n "$dest" ] || return 1
  while read -r name; do
    [ -n "$name" ] || continue
    [ -d "$dest/$name" ] || return 1
    [ "$(find "$dest/$name" -name SKILL.md | wc -l | tr -d '[:space:]')" = 1 ] || return 1
    diff -r "$harness_repo_root/skills/$name" "$dest/$name" >/dev/null || return 1
  done <<SKILL_NAMES
$(repository_skill_names)
SKILL_NAMES
  return 0
}

installed_tree_is_the_repository_tree() {
  local home="$1" dest
  dest="$(installed_destination_root "$home")" || return 1
  every_repository_skill_is_installed_intact "$dest" || return 1
  # And nothing else anywhere under the home: a nested copy, a staging directory
  # an interrupted run left behind, and a second install somewhere else all show
  # up as a manifest the repository does not account for.
  [ "$(find "$home" -name SKILL.md | wc -l | tr -d '[:space:]')" \
    = "$(repository_skill_names | wc -l | tr -d '[:space:]')" ] || return 1
  return 0
}

# --- Never worse than it started ----------------------------------------------
#
# The second invariant, and the one the chosen command broke. Converging is not
# enough: no failure of any step may leave the destination worse than it was,
# because the documented command is the *update* path and it is run over a
# working install.

# The failure a reader actually hits: the block run from a directory that is not
# a checkout, so its source is not there. `rm -rf dst && cp -R src dst` guards
# the copy against a failed remove and leaves the install *empty* when the copy
# is what fails.
documented_update_survives_a_failing_step() {
  local dir dest before after
  dir="$(staged_run_dir failing-step)" || return 1
  stage_the_repository_skills "$dir" || return 1
  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1

  dest="$(installed_destination_root "$dir/home")" || return 1
  [ -n "$dest" ] || return 1
  before="$(cd "$dest" && find . | sort)" || return 1

  # The same block, from a directory with no skills tree in it. Its exit status
  # is not what is asserted — the destination is.
  rm -rf "$dir/cwd/skills"
  run_documented_block "$dir" >/dev/null 2>&1 || true

  after="$(cd "$dest" && find . | sort)" || return 1
  [ "$before" = "$after" ] || return 1
  every_repository_skill_is_installed_intact "$dest" || return 1
  return 0
}

# A run that fails part-way must also leave the *next* run able to converge. A
# staged form that does not clear its own staging directory passes the check
# above and then nests the next update inside what it left behind — which is the
# nesting defect this cluster exists to fix, arriving by another route.
#
# The interruption is a read-only installed skill directory rather than a signal
# or a stubbed tool: it stops whichever step replaces the install, on every
# command form, with no timing to get right and nothing installed or faked.
#
# What is asserted after the interruption is recovery, not survival, and the
# distinction is deliberate. A `rm -rf` on a directory it cannot finish removing
# deletes what it can reach and leaves the rest, so *no* remove-then-replace
# form — and that is every dependency-free form — keeps an install intact
# through this one. Measured: the installed skill loses the contents of every
# writable subdirectory. Demanding survival here would be demanding a
# rename-swap with an `.old` directory and six steps in a README block, to
# protect against a permissions state the reader would have had to create.
# Recovery is the property that is both reachable and the one that matters: the
# documented command, run again, converges.
an_interrupted_update_leaves_the_next_one_able_to_converge() {
  local dir dest first
  dir="$(staged_run_dir interrupted)" || return 1
  stage_the_repository_skills "$dir" || return 1
  run_documented_block "$dir" || return 1
  dest="$(installed_destination_root "$dir/home")" || return 1
  first="$(repository_skill_names | head -1)"
  [ -n "$first" ] || return 1

  chmod 500 "$dest/$first" || return 1
  run_documented_block "$dir" >/dev/null 2>&1 || true
  chmod -R 700 "$dest/$first" || return 1

  # The documented command, run again, converges on the repository instead of
  # merging into whatever the interrupted run left behind.
  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1
  return 0
}

# The README claims it in prose — "nothing else under the skills directory is
# touched" — and the reader whose other skills live there is the one who pays if
# it is wrong. This is also what catches the adaptation hazard: a destination
# substituted one level too high turns the update into `rm -rf` on the directory
# holding every skill the reader has.
documented_update_leaves_the_other_skills_alone() {
  local dir dest
  dir="$(staged_run_dir siblings)" || return 1
  stage_the_repository_skills "$dir" || return 1
  run_documented_block "$dir" || return 1
  dest="$(installed_destination_root "$dir/home")" || return 1
  [ -n "$dest" ] || return 1

  mkdir -p "$dest/someone-elses-skill" || return 1
  echo "not ours" > "$dest/someone-elses-skill/SKILL.md"
  run_documented_block "$dir" || return 1
  run_documented_block "$dir" || return 1

  [ "$(cat "$dest/someone-elses-skill/SKILL.md" 2>/dev/null)" = "not ours" ] || return 1
  every_repository_skill_is_installed_intact "$dest" || return 1
  return 0
}

# --- One destination, spelled once, in the words the README's own list uses ----
#
# The other half of the same blocker. The command embedded the skills directory
# *and* the skill name at every destructive step, while the list of paths a
# reader takes the destination from gives skills directories. A reader adapting
# one into the other produces `rm -rf` on the directory holding all their
# skills. Neither the form of the command nor a warning fixes that: the two
# places have to agree, and there has to be exactly one of them.

# The one thing the block asks the reader to change: its first assignment. Read
# as "the first assignment" rather than by variable name, so the block stays
# free to call it something else.
#
# Split in two because the containment decision above needs the same reading of
# a block it was handed rather than of the README's, and one reading of "the
# destination" is what keeps the decision and the assertions talking about the
# same string.
#
# The value ends at whitespace and nowhere else, because that is where a shell
# ends a word. It used to end at a `#` as well, and that made the destination
# this suite judged a different string from the one the block would use:
# `skills_dir=x#;id>FILE` is a single word to a shell, so the assignment is
# `x#` and the `;id>FILE` after it is a command — while a reader that stopped
# at the `#` saw a harmless `x`, found it contained, and let the block run. A
# `#` that really does open a comment has whitespace in front of it, so
# stopping at whitespace already stops there.
first_assignment_value() {
  sed -n 's/^[A-Za-z_][A-Za-z0-9_]*=\([^ 	]*\).*$/\1/p' | head -1
}

documented_destination() {
  documented_manual_copy_block | first_assignment_value
}

# The same path as the README's own list of where each agent reads skills from
# gives it. Claude Code is the row the command is written against.
readme_listed_destination() {
  sed -n 's/^- \*\*Claude Code\*\* — `\([^`]*\)`.*$/\1/p' README.md | head -1
}

the_list_and_the_block_agree_on_the_destination() {
  local from_block from_list
  from_block="$(documented_destination)"
  from_list="$(readme_listed_destination)"
  [ -n "$from_block" ] || return 1
  [ -n "$from_list" ] || return 1
  [ "${from_block%/}" = "${from_list%/}" ]
}

the_destination_has_exactly_one_substitution_point() {
  local dest count
  dest="$(documented_destination)"
  [ -n "$dest" ] || return 1
  count="$(documented_manual_copy_block | grep -oF "$dest" | wc -l | tr -d '[:space:]')"
  [ "$count" = 1 ]
}

documented_install_is_an_update_that_converges() {
  local dir
  dir="$(staged_run_dir idempotent)" || return 1
  stage_the_repository_skills "$dir" || return 1

  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1

  # The second run is the documented update path. Before it, the two states a
  # merge-into-the-destination command leaves behind for good: a file the
  # repository no longer has, and a file whose contents are stale.
  local installed
  installed="$(installed_skill_dirs "$dir/home" | head -1)"
  [ -n "$installed" ] || return 1
  : > "$installed/withdrawn-upstream.md"
  echo "stale" > "$installed/SKILL.md"

  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1

  # And a third, because "converges" is the claim, not "survives one repeat".
  run_documented_block "$dir" || return 1
  installed_tree_is_the_repository_tree "$dir/home" || return 1
  return 0
}

# --- What "the plugin directory" means, for every native route --------------
#
# Each agent reads its manifest from its own subdirectory of the repository
# root, and the manifest's `skills` value resolves from the *root*, not from the
# directory the manifest sits in. That makes the repository root the plugin root
# and the manifest directory merely where the manifest lives — a distinction the
# install instructions have to get right, because `.cursor-plugin/` on its own
# holds one JSON file and no skills, so a reader who copies that installs a
# plugin with nothing in it and gets no error telling them so.
#
# Asserted here rather than stated in prose, because prose is what was wrong.
#
# The manifest directories are a glob for the same reason the skills are: a
# fifth agent's manifest is covered the day it lands. Hardcoding the four made
# every check below blind to exactly the manifest most likely to be wrong.

plugin_manifest_dirs() {
  local dir
  for dir in .*-plugin; do
    [ -d "$dir" ] || continue
    printf '%s\n' "$dir"
  done
}

repository_has_plugin_manifests() {
  [ -n "$(plugin_manifest_dirs)" ]
}

manifest_skills_value() {
  python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["skills"])' "$1"
}

skills_resolve_from_the_repository_root() {
  local dir value name
  while read -r dir; do
    [ -n "$dir" ] || continue
    [ -f "$dir/plugin.json" ] || return 1
    value="$(manifest_skills_value "$dir/plugin.json")" || return 1
    [ -n "$value" ] || return 1
    # Resolves from the repository root...
    [ -d "$value" ] || return 1
    while read -r name; do
      [ -n "$name" ] || continue
      [ -f "$value/$name/SKILL.md" ] || return 1
    done <<MANIFEST_SKILLS
$(repository_skill_names)
MANIFEST_SKILLS
    # ...and not from the manifest's own directory, which is the misreading the
    # install instructions have to rule out.
    [ ! -d "$dir/$value" ] || return 1
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  return 0
}

# The manifest directory is not itself a distributable plugin: it holds the
# manifest and nothing else. If a skill ever lands inside one, "copy the plugin
# root" stops being unambiguous and this fails.
manifest_dir_holds_only_manifests() {
  local dir entry
  while read -r dir; do
    [ -n "$dir" ] || continue
    for entry in "$dir"/* "$dir"/.*; do
      case "${entry##*/}" in
        '.'|'..'|'*'|'.*') ;;
        plugin.json|marketplace.json) ;;
        *) return 1 ;;
      esac
    done
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  return 0
}

# The manifests are maintained by hand and nothing compared them, so they
# drifted: three spelled the skills path `./skills/` and one spelled it
# `skills`. Both worked, which is why it survived — the cost of that kind of
# drift is not a broken install, it is a later "fix them all" that misses one.
# Comparing them is the only thing that stops it recurring.
manifests_agree_on_the_skills_path() {
  local dir value first=""
  while read -r dir; do
    [ -n "$dir" ] || continue
    value="$(manifest_skills_value "$dir/plugin.json")" || return 1
    if [ -z "$first" ]; then
      first="$value"
    elif [ "$value" != "$first" ]; then
      return 1
    fi
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  [ -n "$first" ]
}

# `claude plugin validate --strict` fails on missing metadata, and author was
# missing from all four. Claude Code is not installed in CI, so the requirement
# is asserted against the manifests directly — a check that skipped itself when
# the CLI was absent would report nothing in the one place it has to report.
manifests_carry_author_attribution() {
  local dir
  while read -r dir; do
    [ -n "$dir" ] || continue
    python3 - "$dir/plugin.json" <<'PY' || return 1
import json, sys
a = json.load(open(sys.argv[1])).get("author")
assert isinstance(a, dict), "author must be an object"
assert a.get("name"), "author.name must be non-empty"
assert a.get("url"), "author.url must be non-empty"
PY
  done <<MANIFEST_DIRS
$(plugin_manifest_dirs)
MANIFEST_DIRS
  return 0
}

# --- Preconditions ------------------------------------------------------------
#
# Everything below runs the README's commands, so these come first and each one
# stops the suite rather than reporting on its way past.

require "the containment decision refuses an absolute destination" \
  containment_refuses_an_absolute_destination
require "the containment decision refuses a tilde with a user name" \
  containment_refuses_a_tilde_with_a_user_name
require "the containment decision refuses a climb out of a redirected home" \
  containment_refuses_a_climb_out_of_a_redirected_home
require "the containment decision refuses a climb out of the working directory" \
  containment_refuses_a_climb_out_of_the_working_directory
require "the containment decision refuses a climb that ends in an absolute path" \
  containment_refuses_a_climb_that_ends_in_an_absolute_path
require "the containment decision refuses a destination that resolves through a symlink" \
  containment_refuses_a_destination_that_resolves_through_a_symlink
require "the containment decision reads the whole block, not just its first line" \
  containment_refuses_a_block_whose_second_line_escapes
require "the containment decision inspects what follows a quoted # on a line" \
  containment_refuses_an_escape_after_a_quoted_hash
require "the containment decision reads a # that is mid-word as an ordinary character" \
  containment_refuses_a_climb_behind_a_mid_word_hash
require "a substitution in the destination is refused before it can run" \
  quietly containment_refuses_a_substitution_before_it_can_run
require "the containment decision refuses a name the block never binds, away from the destination" \
  containment_refuses_an_unbound_name_away_from_the_destination
require "the containment decision refuses every unaccountable word away from the destination" \
  quietly containment_refuses_every_unaccountable_word_away_from_the_destination
require "the containment decision refuses a block that assigns no destination" \
  containment_refuses_a_block_that_assigns_no_destination

require "the resolved verdict accepts an ordinary destination" \
  resolution_accepts_an_ordinary_destination
require "the resolved verdict alone refuses an absolute destination" \
  resolution_refuses_an_absolute_destination
require "the resolved verdict alone refuses a tilde with a user name" \
  resolution_refuses_a_tilde_with_a_user_name
require "the resolved verdict alone refuses a climb out of a redirected home" \
  resolution_refuses_a_climb_out_of_a_redirected_home
require "the resolved verdict alone refuses a climb out of the working directory" \
  resolution_refuses_a_climb_out_of_the_working_directory
require "the resolved verdict alone refuses a climb that ends in an absolute path" \
  resolution_refuses_a_climb_that_ends_in_an_absolute_path
require "the resolved verdict alone refuses a destination that resolves through a symlink" \
  resolution_refuses_a_destination_that_resolves_through_a_symlink
require "the resolved verdict alone refuses a climb through a path that does not exist yet" \
  resolution_refuses_a_climb_through_a_path_that_does_not_exist_yet

require "the destination expansion substitutes this run's home and nothing else" \
  quietly the_expansion_substitutes_home_and_nothing_else
require "the destination expansion refuses a destination no filesystem could hold" \
  quietly the_expansion_refuses_a_destination_no_filesystem_could_hold
require "the destination expansion refuses every command separator, and runs none of them" \
  quietly expansion_refuses_every_command_separator
require "the destination expansion refuses every redirection, and runs none of them" \
  quietly expansion_refuses_every_redirection
require "the destination expansion refuses every substitution, and runs none of them" \
  quietly expansion_refuses_every_substitution
require "the destination expansion refuses everything else a shell does to a word" \
  quietly expansion_refuses_every_other_shell_construct
require "the destination expansion refuses a construct written after a #" \
  quietly expansion_refuses_a_construct_after_a_hash
require "the containment decision refuses a destination that runs a command, and it does not run" \
  quietly containment_refuses_a_destination_that_runs_a_command
require "the destination expansion reaches no shell" \
  quietly the_destination_expansion_uses_no_shell
require "the check for that refuses every escape spelling it is handed" \
  quietly the_interpreter_check_refuses_every_escape_it_can_see
require "the check for that accepts a body that only substitutes" \
  quietly the_interpreter_check_accepts_a_body_that_only_substitutes

require "the README's manual-copy block names nothing outside a redirected home" \
  quietly block_names_nothing_outside_a_redirected_home
require "the destination the README's block resolves to is inside the harness scratch root" \
  quietly the_documented_block_is_contained
require "the repository ships skills for the checks below to be about" \
  repository_ships_skills
require "the repository holds plugin manifests for the checks below to be about" \
  repository_has_plugin_manifests
require "the README documents a manual-copy block naming every shipped skill" \
  documented_block_is_extractable

echo "  skills examined: $(repository_skill_names | tr '\n' ' ')"
echo "  plugin manifests examined: $(plugin_manifest_dirs | tr '\n' ' ')"

# --- Assertions ---------------------------------------------------------------

assert "every native route resolves its skills from the repository root" \
  skills_resolve_from_the_repository_root
assert "no manifest directory is mistakable for the plugin root" \
  manifest_dir_holds_only_manifests
assert "every plugin manifest agrees on how the skills path is spelled" \
  manifests_agree_on_the_skills_path
assert "every plugin manifest carries author attribution" \
  quietly manifests_carry_author_attribution

assert "the documented manual copy installs every shipped skill and converges on re-run" \
  documented_install_is_an_update_that_converges
assert "a failing step leaves the previous install exactly as it was" \
  documented_update_survives_a_failing_step
assert "an interrupted update leaves the next one able to converge" \
  an_interrupted_update_leaves_the_next_one_able_to_converge
assert "the documented update leaves the reader's other skills alone" \
  documented_update_leaves_the_other_skills_alone
assert "the destination is spelled in exactly one place in the block" \
  the_destination_has_exactly_one_substitution_point
assert "the block's destination is the path the README's own list gives" \
  the_list_and_the_block_agree_on_the_destination

harness_summary
