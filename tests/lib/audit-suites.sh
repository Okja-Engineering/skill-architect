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
# Usage: audit-suites.sh <file>...
# Exit 0 when every file is clean, 1 on any violation, 2 on a usage error.
# The last line is always `sites=<N> files=<M>`, so a caller can tell "nothing
# is wrong" from "nothing was examined".
#
# Run it over suites and fixtures, never over tests/lib/harness.sh: the harness
# is where those names are supposed to be defined.
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
#   <harness name> ()                    a redefinition of anything the shared
#                                        harness provides, in any of bash's
#                                        three spellings.
#
#   trap ... EXIT                        an exit handler of the suite's own,
#                                        which displaces the abort guard.
#
# The boundary, stated rather than implied: this reads text, so a verdict whose
# command word is itself an expansion (assert "x" "$cmd") is accepted, and a
# variable that happens to hold a constant is not traced. Closing that needs a
# behavioural census — run every assertion and see which ones can be made to
# fail — which is a larger piece of work and is not what this file does. What
# it does close is the whole class of *literal* verdict, which is the class
# that was live.

set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: audit-suites.sh <file>..." >&2
  exit 2
fi

for f in "$@"; do
  if [ ! -f "$f" ]; then
    echo "audit-suites.sh: not a file: $f" >&2
    exit 2
  fi
done

awk '
BEGIN {
  SQ = sprintf("%c", 39)
  BT = sprintf("%c", 96)

  # Commands whose exit status is fixed before the repository is consulted.
  n = split("true false : echo printf", list, " ")
  for (i = 1; i <= n; i++) constant[list[i]] = 1

  # Wrappers that return the status of the command they were handed, so the
  # verdict is that command s. Defined in tests/lib/harness.sh.
  n = split("quietly", list, " ")
  for (i = 1; i <= n; i++) wrapper[list[i]] = 1

  # Everything the shared harness provides. A suite that defines one of these
  # has its own copy of it.
  n = split("assert assert_value quietly witness_exit harness_init harness_exit harness_summary harness_ready", list, " ")
  for (i = 1; i <= n; i++) harness_name[list[i]] = 1

  n = split("; && || | & then else do { (", list, " ")
  for (i = 1; i <= n; i++) separator[list[i]] = 1

  sites = 0
  bad = 0
  in_hd = 0
  hd_term = ""
  acc = ""
  startline = 0
}

function is_op(t) {
  return (t == ";" || t == "&&" || t == "||" || t == "|" || t == "&")
}

function dequote(t,   r) {
  r = t
  gsub(/"/, "", r)
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
    if (c == "\"") {
      i++
      while (i <= L) {
        d = substr(s, i, 1)
        if (d == "\\") { i += 2; continue }
        if (d == "\"") { i++; break }
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
      if (c == "\"") {
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
          if (d == "\"") break
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
  printf "%s:%d: %s: %s\n", FILENAME, ln, reason, text
  bad++
}

function check_command_shape(toks, i, nt, ln, text,   j, cw) {
  if (i + 1 > nt || is_op(toks[i + 1])) {
    report(ln, "assert has no label", text)
    return
  }
  j = i + 2
  while (j <= nt && !is_op(toks[j]) && wrapper[dequote(toks[j])]) j++
  if (j > nt || is_op(toks[j])) {
    report(ln, "assert has no verdict to run", text)
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

# bash spells a function definition three ways: name(), name () and
# `function name`. All three shadow the shared harness, so all three are read.
function shadowed_name(toks, nt,   t, bare) {
  t = toks[1]
  if (t == "function" && nt >= 2) {
    bare = toks[2]
    sub(/\(\)$/, "", bare)
    if (harness_name[bare]) return bare
    return ""
  }
  bare = t
  if (bare ~ /\(\)$/) {
    sub(/\(\)$/, "", bare)
    if (harness_name[bare]) return bare
    return ""
  }
  if (nt >= 2 && toks[2] == "()" && harness_name[bare]) return bare
  return ""
}

function check_shadowing(toks, nt, ln, text,   name, j) {
  name = shadowed_name(toks, nt)
  if (name != "") {
    report(ln, "a private copy of the harness name " SQ name SQ ", free to drift from the shared one", text)
  }
  if (toks[1] != "trap") return
  for (j = 2; j <= nt; j++) {
    if (dequote(toks[j]) == "EXIT") {
      report(ln, "a private EXIT trap, which displaces the harness abort guard", text)
      return
    }
  }
}

function process(s, ln,   nt, toks, i, name) {
  nt = tokenize(s, toks)
  if (nt == 0) return
  check_shadowing(toks, nt, ln, s)
  for (i = 1; i <= nt; i++) {
    name = toks[i]
    if (name != "assert" && name != "assert_value") continue
    if (i > 1 && !separator[toks[i - 1]]) continue
    sites++
    if (name == "assert") check_command_shape(toks, i, nt, ln, s)
    else check_value_shape(toks, i, nt, ln, s)
  }
}

function detect_hd(l,   tok) {
  if (l !~ /<</) return
  tok = l
  sub(/^.*<<[-]?[ \t]*/, "", tok)
  if (substr(tok, 1, 1) == SQ || substr(tok, 1, 1) == "\"") tok = substr(tok, 2)
  if (tok !~ /^[A-Za-z_]/) return
  sub(/[^A-Za-z0-9_].*$/, "", tok)
  if (tok != "") { hd_term = tok; in_hd = 1 }
}

FNR == 1 { files++; in_hd = 0; acc = ""; hd_term = "" }

{
  if (in_hd) {
    probe = $0
    sub(/^[ \t]+/, "", probe)
    if (probe == hd_term) in_hd = 0
    next
  }

  probe = $0
  sub(/^[ \t]+/, "", probe)
  if (acc == "" && (probe == "" || substr(probe, 1, 1) == "#")) next

  cur = $0
  if (acc == "") startline = FNR
  if (cur ~ /\\[ \t]*$/) {
    sub(/\\[ \t]*$/, "", cur)
    acc = acc cur
    detect_hd($0)
    next
  }
  acc = acc cur
  detect_hd($0)
  process(acc, startline)
  acc = ""
}

END {
  if (acc != "") process(acc, startline)
  printf "sites=%d files=%d\n", sites, files
  if (bad > 0) exit 1
  exit 0
}
' "$@"
