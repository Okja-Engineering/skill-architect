#!/usr/bin/env bash
# Read a shell suite's source text and refuse what breaks the harness
# invariant: an assertion whose verdict is not a function of repository state,
# or a private copy of the harness that is free to drift from it.
#
# Both are source-text questions, and neither is answerable at runtime, which
# is why they are asked here together rather than by two weaker checks.
#
# By the time `assert` is called, a verdict computed from the repository and a
# verdict written as a literal are the same bytes — `assert "the moon is made
# of green cheese" true` prints PASS, and so does `:`, and so does `echo`.
# Making `assert` take a command instead of a value does not close that: a
# literal truthy *command* is still a literal.
#
# And a suite that defines its own `assert`, or installs its own EXIT trap, has
# a copy of the harness that a repair to the shared one will not reach. That is
# not a hypothetical: it is how one suite's assertion took a command while
# another's took a value, and how three of four suites had a bare `rm -rf`
# where the fourth had an abort guard.
#
# A grep cannot ask either question, because these suites hold heredocs with
# Python that starts lines with `assert`, and fixtures that break both rules on
# purpose so the checks below can be seen to fire. This reads shell words, and
# skips heredoc bodies.
#
# Usage:
#   audit-suites.sh [--harness <file>] <file>...   audit, following `source`
#   audit-suites.sh --names <file>...              the names <file> defines
#   audit-suites.sh --closure <file>...            the files an audit would read
#   audit-suites.sh --sites <file>...              <path> <first> <last> per site
#
# Exit 0 when every file is clean, 1 on any violation, 2 on a usage error.
# The last line of an audit is always `sites=<N> files=<M> lines=<L>
# accounted=<A>`, preceded by one `read=<path> sites= lines= accounted=` line
# per file, so a caller can tell "nothing is wrong" from "nothing was
# examined" — per file, because one file going unexamined is invisible in a sum.
#
# # Nothing here is a hand-kept list of the thing it is checking
#
# This file audited a hand-enumerated world four times over, and every one of
# the four was a fail-open:
#
#   1. the harness names were a literal in this file, so a new shared primitive
#      was shadowable the day it landed and until somebody remembered to come
#      here. It is now *derived*, by reading the harness with the same reader
#      that finds a shadow (`--names`), and `tests/test_harness.sh` holds that
#      derivation against the set bash itself reports having loaded.
#   2. the definition spellings were three literals, so `assert ( ) {` — legal
#      on every bash this project supports — was not a redefinition. There is
#      now no list of spellings: `defined_name_at` reads the *grammar*, an
#      optional `function`, a name, and an optional parenthesis pair that may
#      have whitespace in it.
#   3. the EXIT-trap spellings were one literal, `EXIT`, so `trap cleanup 0` and
#      `trap cleanup exit` displaced the abort guard silently. Any operand that
#      names the exit pseudo-signal is now read, in any case, including `0`.
#   4. the set of files was whatever the caller passed, and four of the seven
#      suites get their code from `tests/lib/`, so one line added to a library
#      made a whole suite's assertions vacuous with the audit green. The set is
#      now the **closure under `source`**, computed here, and a `source`
#      directive whose path cannot be resolved is a violation rather than a
#      shrug.
#
# The same discipline governs the counters. `lines` is what the reader saw, and
# a second program counting the records of the same file is its other side —
# `grep -ac ''` in tests/test_harness.sh, and not `wc -l`, because awk counts
# *records* and a file whose last line has no newline holds one more of them
# than `wc -l` reports. `accounted` is what the walk disposed of, and `lines` is
# its other side: every record leaves the walk through one of four paths and
# each increments it.
#
# That pair is not sufficient on its own and the failure is worth recording,
# because the sentence that used to stand here claimed it was. `accounted`
# counts **disposal, not examination** — and the heredoc skip path counts
# itself. So one legal assertion whose *quoted* argument contained `<<WORD` was
# read as opening a heredoc, the rest of the file was consumed as its body,
# `lines == accounted == wc -l` all agreed, `sites` collapsed to 1, and two
# vacuous assertions sat in the swallowed region with this audit green. Two
# repairs close that class at its cause rather than by another counter:
#
#   - a heredoc is detected from the **tokens**, not from a regex over the raw
#     line, so a `<<` inside a quoted word is text and not a redirection; and
#   - a heredoc that reaches the end of its file without its terminator is
#     itself a **violation**. A runaway heredoc is by definition one that never
#     closes, so this refuses the whole class with no number in it.
#
# And `sites` gets the other side it never had, from outside this file
# entirely: `tests/lib/harness.sh` records the line of every assertion it
# actually runs and holds that set against `--sites`, so a call site this
# reader did not reach reddens the suite that ran it. bash is the other side.
#
# Run it over suites and fixtures. `tests/lib/harness.sh` is excluded from the
# closure by identity, not by name: it is where those names are supposed to be
# defined, and it is the file `--harness` points at.
#
# What it reads:
#
#   assert <label> <command> [args...]   the verdict is the command's exit
#                                        status, so the command must be able to
#                                        have one. A constant command (true,
#                                        false, :, echo, printf) is refused,
#                                        including underneath a transparent
#                                        wrapper such as quietly, which passes
#                                        its command's status through unchanged.
#
#   assert_value <label> <verdict>       the verdict is a string. A literal is
#                                        refused; anything with an expansion or
#                                        a command substitution in it is a
#                                        value the repository decides.
#
#   require <label> <command> [args...]  an assertion that also halts the suite,
#                                        read exactly as `assert` is: a
#                                        precondition that cannot fail is a
#                                        guard that can never refuse, which is
#                                        worse than a vacuous assertion.
#
#   <harness name> ()                    a redefinition of anything the shared
#                                        harness provides, in any spelling bash
#                                        accepts, derived from the harness.
#
#   trap ... EXIT                        an exit handler of the suite's own,
#                                        which displaces the abort guard. `0`
#                                        and any casing of `exit` are the same
#                                        signal and are read as such.
#
#   . <path> / source <path>             a file whose text is part of this
#                                        suite, so it joins the audited set.
#
# The boundary, stated rather than implied: this reads text, so a verdict whose
# command word is itself an expansion (assert "x" "$cmd") is accepted, and a
# variable that happens to hold a constant is not traced. Closing that needs a
# behavioural census of *vacuity* — run every assertion and see which ones can
# be made to fail — which is a larger piece of work and is not what this file
# does. What it does close is the whole class of *literal* verdict, which is the
# class that was live. The census the harness does run is of *reachability*:
# which call sites executed, not which of them could have failed.

set -euo pipefail

self_dir="$(CDPATH= cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"

mode=audit
harness="$self_dir/harness.sh"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --harness)
      harness="${2:-}"
      shift 2 || true
      ;;
    --names)   mode=names;   shift ;;
    --closure) mode=closure; shift ;;
    --sites)   mode=sites;   shift ;;
    --)        shift; break ;;
    -*)
      echo "audit-suites.sh: unknown option: $1" >&2
      exit 2
      ;;
    *) break ;;
  esac
done

if [ "$#" -eq 0 ]; then
  echo "usage: audit-suites.sh [--harness <file>] [--names|--closure|--sites] <file>..." >&2
  exit 2
fi

for f in "$@"; do
  if [ ! -f "$f" ]; then
    echo "audit-suites.sh: not a file: $f" >&2
    exit 2
  fi
done

if [ ! -f "$harness" ]; then
  echo "audit-suites.sh: the harness is not a file: $harness" >&2
  exit 2
fi

# The reader. One awk program, and the mode decides what it reports; the
# tokenizer, the definition grammar and the heredoc tracking are the same code
# in every mode, because a second reader is a second thing to be wrong.
AUDIT_READER='
BEGIN {
  SQ = sprintf("%c", 39)
  DQ = sprintf("%c", 34)
  BT = sprintf("%c", 96)

  # Commands whose exit status is fixed before the repository is consulted.
  n = split("true false : echo printf", list, " ")
  for (i = 1; i <= n; i++) constant[list[i]] = 1

  # Wrappers that return the status of the command they were handed, so the
  # verdict is that command s. Defined in tests/lib/harness.sh.
  n = split("quietly", list, " ")
  for (i = 1; i <= n; i++) wrapper[list[i]] = 1

  # Everything the shared harness provides, read out of the harness by this
  # same program in --names mode and handed in. There is no list here.
  n = split(harness_names, list, " ")
  for (i = 1; i <= n; i++) if (list[i] != "") harness_name[list[i]] = 1

  n = split("; && || | & then else do { (", list, " ")
  for (i = 1; i <= n; i++) separator[list[i]] = 1

  sites = 0
  lines = 0
  accounted = 0
  bad = 0
  in_hd = 0
  hd_term = ""
  hd_start = 0
  acc = ""
  startline = 0
  prev_file = ""
}

function is_op(t) {
  return (t == ";" || t == "&&" || t == "||" || t == "|" || t == "&")
}

function dequote(t,   r) {
  r = t
  gsub(DQ, "", r)
  gsub(SQ, "", r)
  return r
}

function has_expansion(t) {
  return (index(t, "$") > 0 || index(t, BT) > 0)
}

# Index just past the ")" that closes the "$(" starting at i.
function scan_subst(s, i,   L, depth, c, d) {
  L = length(s)
  i += 2
  depth = 1
  while (i <= L && depth > 0) {
    c = substr(s, i, 1)
    if (c == "\\") { i += 2; continue }
    if (c == SQ) {
      i++
      while (i <= L && substr(s, i, 1) != SQ) i++
      i++
      continue
    }
    if (c == DQ) {
      i++
      while (i <= L) {
        d = substr(s, i, 1)
        if (d == "\\") { i += 2; continue }
        if (d == DQ) { i++; break }
        if (d == "$" && substr(s, i + 1, 1) == "(") { i = scan_subst(s, i); continue }
        i++
      }
      continue
    }
    if (c == "$" && substr(s, i + 1, 1) == "(") { i = scan_subst(s, i); continue }
    if (c == "(") { depth++; i++; continue }
    if (c == ")") { depth--; i++; continue }
    i++
  }
  return i
}

# Split a logical line into top-level shell words and operators, keeping quotes
# and substitutions whole so a label containing a space, a pipe or a nested
# command substitution stays one token.
function tokenize(s, toks,   L, i, n, c, d, t, k) {
  L = length(s)
  i = 1
  n = 0
  while (i <= L) {
    c = substr(s, i, 1)
    if (c == " " || c == "\t") { i++; continue }
    if (c == ";") { toks[++n] = ";"; i++; continue }
    if (c == "&") {
      if (substr(s, i, 2) == "&&") { toks[++n] = "&&"; i += 2 } else { toks[++n] = "&"; i++ }
      continue
    }
    if (c == "|") {
      if (substr(s, i, 2) == "||") { toks[++n] = "||"; i += 2 } else { toks[++n] = "|"; i++ }
      continue
    }
    t = ""
    while (i <= L) {
      c = substr(s, i, 1)
      if (c == " " || c == "\t" || c == ";" || c == "&" || c == "|") break
      if (c == "\\") { t = t substr(s, i, 2); i += 2; continue }
      if (c == SQ) {
        t = t c
        i++
        while (i <= L) { d = substr(s, i, 1); t = t d; i++; if (d == SQ) break }
        continue
      }
      if (c == DQ) {
        t = t c
        i++
        while (i <= L) {
          d = substr(s, i, 1)
          if (d == "\\") { t = t substr(s, i, 2); i += 2; continue }
          if (d == "$" && substr(s, i + 1, 1) == "(") {
            k = scan_subst(s, i)
            t = t substr(s, i, k - i)
            i = k
            continue
          }
          t = t d
          i++
          if (d == DQ) break
        }
        continue
      }
      if (c == "$" && substr(s, i + 1, 1) == "(") {
        k = scan_subst(s, i)
        t = t substr(s, i, k - i)
        i = k
        continue
      }
      if (c == BT) {
        t = t c
        i++
        while (i <= L) { d = substr(s, i, 1); t = t d; i++; if (d == BT) break }
        continue
      }
      t = t c
      i++
    }
    toks[++n] = t
  }
  return n
}

function report(ln, reason, text) {
  if (mode != "audit") return
  printf "%s:%d: %s: %s\n", FILENAME, ln, reason, text
  bad++
}

function check_command_shape(verb, toks, i, nt, ln, text,   j, cw) {
  if (i + 1 > nt || is_op(toks[i + 1])) {
    report(ln, verb " has no label", text)
    return
  }
  j = i + 2
  while (j <= nt && !is_op(toks[j]) && wrapper[dequote(toks[j])]) j++
  if (j > nt || is_op(toks[j])) {
    report(ln, verb " has no verdict to run", text)
    return
  }
  if (has_expansion(toks[j])) return
  cw = dequote(toks[j])
  if (constant[cw]) {
    report(ln, "verdict is the constant command " SQ cw SQ ", so this assertion cannot fail", text)
  }
}

function check_value_shape(toks, i, nt, ln, text,   v) {
  if (i + 1 > nt || is_op(toks[i + 1])) {
    report(ln, "assert_value has no label", text)
    return
  }
  if (i + 2 > nt || is_op(toks[i + 2])) {
    report(ln, "assert_value has no verdict", text)
    return
  }
  v = toks[i + 2]
  if (has_expansion(v)) return
  report(ln, "verdict is the literal " SQ dequote(v) SQ ", so this assertion cannot fail", text)
}

# The name a function definition starting at token i defines, or "".
#
# bash has no fixed number of spellings for this and the three that used to be
# listed here were not all of them: `assert ( ) {` is legal on 3.2 and on 5,
# defines the function, and was not a redefinition as far as this file was
# concerned. So this reads the grammar instead — an optional `function`
# keyword, a name, and an optional parenthesis pair whose contents may be
# nothing but whitespace — and has no list to be short.
#
# The tokenizer has already collapsed the whitespace, so `name ( )`, `name ()`
# and `name()` arrive as three different token shapes of one construct and all
# three are read here. `tests/test_harness.sh` holds the whole of this against
# bash: every spelling it generates is sourced into a real shell, `declare -F`
# is asked which name that shell defined, and this reader must agree.
function defined_name_at(toks, nt, i,   t, name, k) {
  t = toks[i]
  if (t == "function") {
    if (i + 1 > nt) return ""
    name = toks[i + 1]
    sub(/\([ \t]*\).*$/, "", name)
    sub(/\{.*$/, "", name)
    if (name !~ /^[A-Za-z_][A-Za-z0-9_]*$/) return ""
    return name
  }
  name = t
  # The name with its parentheses glued on: name() / name(){ / name( ){
  if (name ~ /^[A-Za-z_][A-Za-z0-9_]*\([ \t]*\)/) {
    sub(/\(.*$/, "", name)
    return name
  }
  if (name !~ /^[A-Za-z_][A-Za-z0-9_]*$/) return ""
  # The parentheses as their own token(s): `name ()` or `name ( )`.
  k = i + 1
  if (k > nt) return ""
  if (toks[k] ~ /^\([ \t]*\)/) return name
  if (toks[k] == "(") {
    k++
    if (k <= nt && toks[k] ~ /^\)/) return name
    return ""
  }
  return ""
}

# Whether the token names the exit pseudo-signal. bash takes `EXIT`, any other
# casing of it, and `0`, and every one of them displaces the abort guard;
# measured on 3.2.57 and on 5.3.15 by firing each. There is no list of
# spellings because case is not a spelling.
function is_exit_spec(t,   d) {
  d = dequote(t)
  return (tolower(d) == "exit" || d == "0")
}

# The exit sigspec a `trap` command starting at token i names, or "".
#
# Every operand after the action is a sigspec, and a `trap` with one operand
# resets that signal — which removes the abort guard just as surely as
# replacing it. `-p` and `-l` only ask, so they are left alone.
function trap_exit_spec_at(toks, nt, i,   j, a) {
  if (toks[i] != "trap") return ""
  j = i + 1
  if (j <= nt && toks[j] == "--") j++
  if (j <= nt && (toks[j] == "-p" || toks[j] == "-l")) return ""
  if (j > nt || is_op(toks[j])) return ""
  if (j == nt || is_op(toks[j + 1])) {
    if (is_exit_spec(toks[j])) return dequote(toks[j])
    return ""
  }
  for (a = j + 1; a <= nt; a++) {
    if (is_op(toks[a])) break
    if (is_exit_spec(toks[a])) return dequote(toks[a])
  }
  return ""
}

# The heredoc terminator a logical line opens, or "".
#
# Read from the tokens and not from the raw text, which is the repair for the
# hole that made this whole audit walkable: the old reader ran
# `sub(/^.*<<[-]?[ \t]*/, ...)` over the line, so one legal assertion whose
# quoted argument contained `<<WORD` opened a heredoc that never closed, and
# the rest of the file became its body with every counter agreeing. A `<<`
# inside a quoted word is text. `<<<` is a here-string and opens nothing.
function heredoc_term(toks, nt,   i, t, L, j, c, q, rest, term) {
  for (i = 1; i <= nt; i++) {
    t = toks[i]
    L = length(t)
    j = 1
    while (j <= L) {
      c = substr(t, j, 1)
      if (c == "\\") { j += 2; continue }
      if (c == SQ || c == DQ) {
        q = c
        j++
        while (j <= L) {
          if (substr(t, j, 1) == "\\" && q == DQ) { j += 2; continue }
          if (substr(t, j, 1) == q) { j++; break }
          j++
        }
        continue
      }
      if (c == "<" && substr(t, j + 1, 1) == "<") {
        j += 2
        if (substr(t, j, 1) == "<") { j++; continue }
        if (substr(t, j, 1) == "-") j++
        rest = substr(t, j)
        if (rest == "") {
          if (i + 1 > nt) return ""
          rest = toks[i + 1]
        }
        term = rest
        gsub(DQ, "", term)
        gsub(SQ, "", term)
        gsub(/\\/, "", term)
        sub(/[^A-Za-z0-9_].*$/, "", term)
        if (term != "") return term
        return ""
      }
      j++
    }
  }
  return ""
}

# A `source`/`.` word, resolved as far as text can take it.
#
# Emitted for the caller to turn into a path, because only the caller can test
# whether a file is there. Three resolutions, in order, and anything else is
# reported as unresolvable rather than skipped: a literal; a variable this file
# assigned a literal to; a variable in the environment. A directive this cannot
# resolve is a file whose text would be part of the suite and would not be
# read, which is the defect, so it fails closed.
function emit_source(word, ln,   w, name, tail, p, q, expanded) {
  w = word
  gsub(DQ, "", w)
  gsub(SQ, "", w)
  expanded = 0
  if (index(w, "$") > 0 || index(w, BT) > 0) {
    name = w
    sub(/^\$\{/, "", name)
    sub(/^\$/, "", name)
    sub(/\}$/, "", name)
    if (name ~ /^[A-Za-z_][A-Za-z0-9_]*$/) {
      if (name in assign) { print "SRC\t0\t" FILENAME "\t" ln "\t" assign[name]; return }
      if (name in ENVIRON && ENVIRON[name] != "") { print "SRC\t0\t" FILENAME "\t" ln "\t" ENVIRON[name]; return }
      print "SRCX\t" FILENAME "\t" ln "\t" word
      return
    }
    p = 0
    for (q = length(w); q >= 1; q--) {
      if (substr(w, q, 1) == ")" || substr(w, q, 1) == "}") { p = q; break }
    }
    if (p == 0) { print "SRCX\t" FILENAME "\t" ln "\t" word; return }
    tail = substr(w, p + 1)
    if (tail == "" || index(tail, "$") > 0) { print "SRCX\t" FILENAME "\t" ln "\t" word; return }
    print "SRC\t1\t" FILENAME "\t" ln "\t" tail
    return
  }
  print "SRC\t0\t" FILENAME "\t" ln "\t" w
}

function process(s, ln, endln,   nt, toks, i, name, spec, term) {
  nt = tokenize(s, toks)
  if (nt == 0) return

  for (i = 1; i <= nt; i++) {
    if (i > 1 && !separator[toks[i - 1]] && !is_op(toks[i - 1])) continue

    name = defined_name_at(toks, nt, i)
    if (name != "") {
      if (mode == "names") {
        print name
      } else if (harness_name[name]) {
        report(ln, "a private copy of the harness name " SQ name SQ ", free to drift from the shared one", s)
      }
    }

    spec = trap_exit_spec_at(toks, nt, i)
    if (spec != "" && mode == "audit") {
      report(ln, "a private EXIT trap (spelled " SQ spec SQ "), which displaces the harness abort guard", s)
    }

    if ((toks[i] == "." || toks[i] == "source") && mode == "closure" && i + 1 <= nt && !is_op(toks[i + 1])) {
      emit_source(toks[i + 1], ln)
    }
  }

  for (i = 1; i <= nt; i++) {
    name = toks[i]
    if (name != "assert" && name != "assert_value" && name != "require") continue
    if (i > 1 && !separator[toks[i - 1]]) continue
    sites++
    if (mode == "sites") print FILENAME "\t" ln "\t" endln
    if (mode == "audit") {
      if (name == "assert_value") check_value_shape(toks, i, nt, ln, s)
      else check_command_shape(name, toks, i, nt, ln, s)
    }
  }

  # A variable this file assigns a literal to, for a `source` of it below.
  if (mode == "closure") {
    for (i = 1; i <= nt; i++) {
      if (toks[i] !~ /^[A-Za-z_][A-Za-z0-9_]*=/) continue
      name = toks[i]
      sub(/=.*$/, "", name)
      term = toks[i]
      sub(/^[^=]*=/, "", term)
      gsub(DQ, "", term)
      gsub(SQ, "", term)
      if (term != "" && index(term, "$") == 0 && index(term, BT) == 0) assign[name] = term
      break
    }
  }

  term = heredoc_term(toks, nt)
  if (term != "") { hd_term = term; in_hd = 1; hd_start = endln }
}

# A heredoc that reached the end of a file without its terminator swallowed
# everything after it, and that is the shape a runaway opener takes. It is a
# violation in its own right, which refuses the whole class with no counter.
function close_file() {
  if (in_hd && mode == "audit") {
    printf "%s:%d: a heredoc opened with %s%s%s is never terminated, so the rest of the file was read as its body\n", \
      prev_file, hd_start, SQ, hd_term, SQ
    bad++
  }
  in_hd = 0
  hd_term = ""
  hd_start = 0
}

# Every input record, counted before any rule below can skip one. First, and on
# its own, so that a heredoc body, a continuation and a comment are all counted
# as read: the question this answers is how much of the file this program saw.
{ lines++ }

FNR == 1 {
  if (prev_file != "") { close_file(); flush_file() }
  prev_file = FILENAME
  files++
  file_lines = 0
  file_accounted = 0
  file_sites = sites
  acc = ""
  split("", assign)
}

{ file_lines++ }

function flush_file() {
  if (mode == "audit") {
    printf "read=%s sites=%d lines=%d accounted=%d\n", prev_file, sites - file_sites, file_lines, file_accounted
  }
}

# Every record leaves the rule below through exactly one of four paths, and each
# of them says so. `accounted` is therefore `lines` unless something skipped a
# record without accounting for it — which is what a limit, an off-by-one in a
# bound, or a rule inserted above this one all look like from the outside. The
# pair answers "was the whole file walked", and it does not answer "were the
# sites seen": `accounted` counts disposal, and the heredoc path disposes of
# whatever it was given. That second question is answered by the unterminated
# heredoc check above, by reading `<<` from the tokens, and by the harness
# holding the sites it ran against --sites.
{
  if (in_hd) {
    probe = $0
    sub(/^[ \t]+/, "", probe)
    if (probe == hd_term) in_hd = 0
    accounted++
    file_accounted++
    next
  }

  probe = $0
  sub(/^[ \t]+/, "", probe)
  if (acc == "" && (probe == "" || substr(probe, 1, 1) == "#")) { accounted++; file_accounted++; next }

  cur = $0
  if (acc == "") startline = FNR
  if (cur ~ /\\[ \t]*$/) {
    sub(/\\[ \t]*$/, "", cur)
    acc = acc cur
    accounted++
    file_accounted++
    next
  }
  acc = acc cur
  process(acc, startline, FNR)
  acc = ""
  accounted++
  file_accounted++
}

END {
  if (acc != "") process(acc, startline, FNR)
  if (prev_file != "") { close_file(); flush_file() }
  if (mode == "audit") printf "sites=%d files=%d lines=%d accounted=%d\n", sites, files, lines, accounted
  if (bad > 0) exit 1
  exit 0
}
'

# The names the harness provides, derived from the harness by the reader above.
# There is no list of them anywhere in this file.
harness_names="$(awk -v mode=names -v harness_names="" "$AUDIT_READER" "$harness" | sort -u | tr '\n' ' ')"

# And the derivation has to be total, which is the same rule the closure below
# is held to and the rule this whole file was rewritten for: a derived side that
# comes back empty and says nothing has replaced a short list with no list at
# all. Measured, before this refusal existed: pointed at a harness that defines
# no function, this script read a suite whose first line was `assert() {` and
# reported nothing about it — the shadowing half silently disabled, with the
# exit status coming from an unrelated finding. Every caller in the repository
# passes the real harness, so it was not live; it is the shape that is the
# defect, and a check whose subject can be emptied from outside has no floor
# under it.
#
# The refusal is scoped to `audit`, which is the only mode that consults the
# set. `--names` *is* the derivation and is handed an empty set on purpose;
# `--sites` and `--closure` never look at a harness name, and refusing there
# would take the diagnostic away from the caller that asked a different
# question — tests/lib/harness.sh's census reads `--sites` out of a deliberately
# crippled reader to prove it can refuse one.
if [ "$mode" = names ]; then
  awk -v mode=names -v harness_names="" "$AUDIT_READER" "$@" | sort -u
  exit 0
fi

if [ "$mode" = sites ]; then
  awk -v mode=sites -v harness_names="$harness_names" "$AUDIT_READER" "$@" || true
  exit 0
fi

if [ "$mode" = audit ]; then
  case "$harness_names" in
    '' | ' ')
      echo "audit-suites.sh: no function definition was found in the harness at $harness: there is nothing a suite could shadow, so the check would pass over one that does" >&2
      exit 2
      ;;
  esac
fi

# --- The closure under `source` ----------------------------------------------
#
# The audited set is not what a caller passed: four of the seven suites get
# their code from tests/lib/, and a definition there is as live as one in the
# suite. So every file reachable by `source` from an argument joins the set,
# transitively, and the harness leaves it by identity — it is the file
# `--harness` names, whatever path a suite spelled to reach it.
#
# A directive this cannot resolve to a file is refused. That is the whole
# difference between a derivation and a list: the derivation has to be total,
# and where it cannot be, it says so instead of returning a smaller world.
same_file() {
  [ -f "$1" ] && [ -f "$2" ] && [ "$1" -ef "$2" ]
}

# The queue and the closure are newline-terminated lists, and they are grown
# and shortened with parameter expansion rather than through `$( )`, which
# strips the trailing newline and silently concatenates the next entry onto the
# last one.
closure=""
pending=""
for f in "$@"; do
  pending="$pending$f
"
done
unresolved=""

while [ -n "$pending" ]; do
  next="${pending%%
*}"
  pending="${pending#*
}"
  [ -n "$next" ] || continue
  if same_file "$next" "$harness"; then continue; fi
  seen=no
  while IFS= read -r have; do
    [ -n "$have" ] || continue
    if same_file "$next" "$have"; then seen=yes; break; fi
  done <<CLOSURE_SEEN
$closure
CLOSURE_SEEN
  [ "$seen" = no ] || continue
  closure="$closure$next
"
  while IFS=$'\t' read -r kind a b c d; do
    case "$kind" in
      SRC)
        # a=expanded flag, b=file, c=line, d=the path or its literal tail
        cand=""
        if [ -f "$d" ]; then
          cand="$d"
        elif [ "$a" = 1 ]; then
          # The literal tail of an expanded word is a suffix, so it is offered
          # the directory of the file that spelled it as well as the cwd.
          try="$(dirname -- "$b")/${d#/}"
          if [ -f "$try" ]; then cand="$try"; fi
        fi
        if [ -n "$cand" ]; then
          pending="$pending$cand
"
        else
          unresolved="$unresolved$b:$c: a source directive naming $d, which is not a file this can read
"
        fi
        ;;
      SRCX)
        unresolved="$unresolved$a:$b: a source directive this cannot resolve to a path: $c
"
        ;;
    esac
  done <<CLOSURE_SOURCES
$(awk -v mode=closure -v harness_names="$harness_names" "$AUDIT_READER" "$next" || true)
CLOSURE_SOURCES
done

if [ "$mode" = closure ]; then
  if [ -n "$unresolved" ]; then
    printf '%s' "$unresolved" >&2
    printf '%s' "$closure"
    exit 1
  fi
  printf '%s' "$closure"
  exit 0
fi

status=0
if [ -n "$unresolved" ]; then
  printf '%s' "$unresolved"
  status=1
fi

set --
while IFS= read -r have; do
  [ -n "$have" ] || continue
  set -- "$@" "$have"
done <<CLOSURE_AUDIT
$closure
CLOSURE_AUDIT

if [ "$#" -eq 0 ]; then
  echo "audit-suites.sh: the closure is empty, so nothing was read" >&2
  exit 2
fi

awk -v mode=audit -v harness_names="$harness_names" "$AUDIT_READER" "$@" || status=1
exit "$status"
