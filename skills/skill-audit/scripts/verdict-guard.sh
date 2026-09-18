# Shared primitives for the skill-audit check scripts: the verdict guards, and
# the one reading of a SKILL.md that every verdict about its contents is
# computed over.
#
# A check script must never report a verdict it could not compute. Whenever a
# tool it needs is missing, or a child check exits with a status it cannot
# interpret, it says so and exits 3 (execution error) instead of letting the
# failure read as a clean pass.
#
# Reading the source belongs here for the same reason the guards do, and the
# reason is not tidiness. "Where does the frontmatter end" is one question, and
# four scripts each answered it privately, with four answers that disagreed.
# check-frontmatter.sh and audit-report.sh exited at the second `---` and were
# right. check-paths.sh ran a toggle, so a third `---` put it back into
# frontmatter: one markdown horizontal rule silently ended every path check
# after it, and a skill with two broken references reported none. And
# check-structure.sh did not ask at all, so its whole house-policy verdict was
# satisfiable out of frontmatter — six required headings written as column-0
# YAML comments, a code fence inside a block scalar, and the `---` delimiter
# itself read as a list item gave `summary.passed: true` and zero findings to a
# skill whose body was one prose sentence. A question with four answers has no
# answer, and deleting only the two wrong copies would leave the next reader
# looking at two. So it is asked once, here, beside the guards all four of those
# scripts already load.
#
# The rule ID this file emits directly is DEP001, a required tool is absent.
# Its callers pass it DEP002 for a source they could not read — a child check
# whose status or payload they cannot interpret, or a source they could not get
# a usable answer out of at all.
#
# Source this from a check script, checking the load on both sides:
#
#   script_dir="$(CDPATH= cd -P -- "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
#   verdict_guard="$script_dir/verdict-guard.sh"
#   bash -n "$verdict_guard" 2>/dev/null \
#     || { echo "ERROR: cannot load $verdict_guard: ..." >&2; exit 3; }
#   source "$verdict_guard"
#   { declare -F verdict_guard_ready >/dev/null && verdict_guard_ready; } \
#     || { echo "ERROR: $verdict_guard did not load its guards: ..." >&2; exit 3; }
#
# `CDPATH=` and `--` in that first line are load-bearing. A script invoked as
# `skills/skill-audit/scripts/check-paths.sh` has a relative dirname, and `cd`
# consults CDPATH for anything that does not begin with `/`, `./` or `../`: with
# CDPATH exported, the caller's own environment decides which
# `skills/skill-audit/scripts` this script thinks it lives in, and `cd` echoes
# the one it picked, which puts a second line in the value. The guard below is
# then looked for somewhere else, the load check refuses it, and every call exits
# 3 having computed nothing. It fails safe, which is why it went unnoticed: the
# whole suite reported 58 labelled failures rather than a wrong verdict. But an
# absolute path is not the caller's to redirect, so it is spelled that way here
# and in every script that copies this block.
#
# What the caller confirms afterwards is verdict_guard_ready, not a list of the
# guards it happens to use. "The guard loaded" is not one proposition: a file
# that stopped short defines some of them and not others, and a caller checking
# only the names it remembered runs on until `command not found` at the point it
# needed the one that is gone. The list lives here, once, beside the definitions
# it covers.
#
# This file is the one dependency require_tool cannot announce: if it is not
# there, nothing is there to do the announcing. An unchecked `source` of a
# missing file aborts the calling script with status 1, and of a malformed one
# with status 2 — statuses every caller's contract reserves for a spec, path or
# policy verdict it actually computed. A file that loads but defines nothing is
# worse still: the caller runs to completion and only discovers the guard is
# gone on the path where it needed it. So the caller syntax-checks the file
# before sourcing it and confirms the guards exist afterwards, and reports
# either failure the way it reports every other unmet precondition — exit 3, no
# verdict. That exit carries no payload: the arguments have not been parsed yet,
# so the caller does not know whether a payload was even asked for, and the
# thing that builds payloads is the thing that is missing.

# json_string <text>
#
# Echo <text> as a JSON string literal, surrounding quotes included.
#
# The guard encodes in the shell rather than shelling out to jq, because a
# missing jq is one of the things it is called to report: the payload has to be
# buildable with the encoder absent. Interpolating the text instead would make
# the documented shape conditional on every caller passing a string with no
# quote, backslash or control character — a convention no caller is checked
# against, and one a future caller relaying a tool's own output would break.
# JSON asks for three things to be escaped: the quote, the backslash, and the
# control characters below U+0020. Everything else is passed through exactly as
# it arrived, every byte of a multi-byte character included.
#
# Which of those a byte is, is settled by its ordinal, because the decision and
# the format it decides on have to be one question rather than two. `\u00xx`
# can spell an ordinal below 0x80 and nothing else, so the range that reaches
# that arm is exactly the range it can spell, and there is no way left to ask
# it for something it has no room for. Splitting the question has now been
# wrong here twice, in the same direction both times: bash yields a *negative*
# ordinal for any byte at or above 0x80, so a bare `< 32` test sent every byte
# of a UTF-8 character down the escape path and emitted ￿ffffffffffc3 for
# it, and asking `[[:cntrl:]]` instead did the same under a UTF-8 locale, where
# that class matches a byte at or above 0x80 — an invalid one, and the valid C1
# controls U+0080-U+009F, which JSON asks no one to escape. The format then ran
# on an ordinal it could not spell and emitted sixteen hex digits where it
# promised four. Both payloads still parsed, so the shape promise held while the
# message no longer said what it was given — and carrying the message is the
# reason this encoder exists at all. Asking the ordinal, and only about the
# range the format covers, also makes the encoding the same under every locale,
# which is the property that made the second of those two a live defect and the
# first a latent one.
#
# DEL at 0x7f is escaped as well, as it has been here throughout. It is inside
# what the format can spell and outside what a message usefully carries.
json_string() {
  local s="$1"
  local out='"'
  local i c ord
  for (( i = 0; i < ${#s}; i++ )); do
    c="${s:i:1}"
    case "$c" in
      '"')   out+='\"' ;;
      '\')   out+='\\' ;;
      $'\n') out+='\n' ;;
      $'\r') out+='\r' ;;
      $'\t') out+='\t' ;;
      $'\b') out+='\b' ;;
      $'\f') out+='\f' ;;
      *)
        printf -v ord '%d' "'$c"
        if (( ord >= 0 && (ord < 0x20 || ord == 0x7f) )); then
          printf -v c '\\u%04x' "$ord"
        fi
        out+="$c"
        ;;
    esac
  done
  printf '%s"\n' "$out"
}

# cannot_compute <rule> <message> <emit_json:true|false> [detail]
#
# Report that no verdict could be reached, and exit 3. In JSON mode the payload
# keeps the documented {"findings": [...], "passed": bool} shape so a machine
# consumer sees a failing verdict rather than a missing one, and carries the
# reason as an extra "error" field. Any detail — a failed tool's own output —
# follows the reason on stderr, never on stdout, which in JSON mode is the
# payload channel.
cannot_compute() {
  local rule="$1"
  local message="$2"
  local emit_json="${3:-false}"
  local detail="${4:-}"

  echo "ERROR: $message" >&2
  [[ -n "$detail" ]] && echo "$detail" >&2
  if [[ "$emit_json" == "true" ]]; then
    printf '{"findings": [{"level": "fail", "rule": %s, "message": %s}], "passed": false, "error": %s}\n' \
      "$(json_string "$rule")" "$(json_string "$message")" "$(json_string "$message")"
  fi
  exit 3
}

# require_tool <tool> <emit_json:true|false>
#
# A stated precondition, checked once where the dependency is established rather
# than at each call site — so the requirement never depends on what the audited
# skill happens to contain.
require_tool() {
  local tool="$1"
  local emit_json="${2:-false}"

  command -v "$tool" >/dev/null 2>&1 && return 0

  cannot_compute DEP001 "required tool not found: $tool" "$emit_json"
}

# json_document_conforms <text> <claim>
#
# Succeed when <text> is exactly one JSON document and <claim> — a jq expression
# evaluated with that document as its input — answers true of it.
#
# Shape is a question about the text alone, so it is asked about the text alone:
# before anything is read out of it, and whatever the things read out of it turn
# out to hold. "It parsed" is not that question. A number parses, and a caller
# that took parsing for shape then indexed it and died inside jq with nothing on
# stdout at all — which is how every consumer in this skill has been broken at
# least once. Every read a caller makes after this returns 0 is total; there is
# nothing left for it to trip over.
#
# The claim reaches as far as the caller reads and no further. Proving the top
# level and then indexing a level down is the same defect one level in, and
# proving more than is read makes a source unreadable for a field nobody wanted.
#
# `-s` is what makes "exactly one document" part of every claim, and it lives
# here rather than in each caller's claim because it is the same sentence for
# all of them: a stream of documents slurps to an array longer than one and is
# refused, whichever position a conforming document holds in it, and no input at
# all slurps to an empty array. What differs between callers is only the shape,
# so only the shape is theirs to state.
#
# A claim is written total rather than short-circuiting — each `if` settles a
# type before anything indexes through it — so a wrong type answers false where
# it would otherwise raise. Totality is not observable from outside, since a
# raise leaves jq non-zero and is already read here as "not that shape". It is
# the rule anyway because the point of a claim is to decide the question rather
# than to survive being wrong about it, and an expression whose answer depends
# on which branch happens to be evaluated is the exact defect this exists to
# close.
#
# It answers with jq, so it is callable only where jq is already a proven
# precondition. Every caller requires jq before reading any source.
json_document_conforms() {
  printf '%s' "$1" | jq -se "length == 1 and (.[0] | $2)" >/dev/null 2>&1
}

# payload_is_conforming <text>
#
# Succeed when <text> is one findings payload in the shape these scripts
# document: an object with a boolean `passed` and an array `findings` whose
# every element is an object carrying a string `level`, `rule` and `message`.
#
# This is the findings payload's claim, named once because two scripts read that
# payload and a shape proven twice is a shape proven two ways. The elements are
# part of it because that is how far both of them read.
payload_is_conforming() {
  json_document_conforms "$1" '
    if type != "object" then false
    elif (.passed | type) != "boolean" then false
    elif (.findings | type) != "array" then false
    else [.findings[] |
      if type != "object" then false
      else (.level | type) == "string"
           and (.rule | type) == "string"
           and (.message | type) == "string"
      end] | all
    end'
}

# skill_section <frontmatter|body> <skill-md>
#
# Echo one of the two halves of a SKILL.md. The shared reading skill_frontmatter
# and skill_body are both spellings of: a SKILL.md opens with YAML frontmatter
# delimited by `---`, and everything after that is body.
#
# Three decisions, each stated because each was a defect somewhere:
#
# Frontmatter opens on line 1 or it does not open at all. A `---` further down
# is a markdown horizontal rule, which is body, and a file that never had
# frontmatter has none to find. Scanning for the first `---` anywhere made a
# rule in the body open a frontmatter that was not there, so a `license:` line
# written in prose satisfied the license gate and the real frontmatter above it
# was never read.
#
# Frontmatter closes once and never re-opens. The next `---` after the opener
# closes it; every `---` after that is body, because there is only one
# frontmatter block and it has already ended. A toggle reads the third
# delimiter as a second opening — the horizontal-rule defect above.
#
# Frontmatter that never closes leaves no body. A file that opens a block and
# runs to EOF inside it has frontmatter and nothing else, on any reading, so the
# body is empty rather than being the frontmatter over again.
#
# The delimiter is exactly `---` on a line of its own, as it was in all four of
# the private copies: no leading space, no trailing space, no `...`. Widening
# that is a separate question from asking it in one place, and is not smuggled
# in here.
#
# It answers with awk, so it is callable only where awk is a proven
# precondition, and it passes awk's status through: an unreadable file leaves
# awk non-zero, and a caller that cannot read the source has no verdict to
# report about it. Every caller checks.
skill_section() {
  local want="$1"
  local file="$2"
  awk -v want="$want" '
    NR == 1 {
      if ($0 == "---") { in_fm = 1; next }
      past_fm = 1
    }
    in_fm {
      if ($0 == "---") { in_fm = 0; past_fm = 1; next }
      if (want == "frontmatter") print
      next
    }
    past_fm && want == "body" { print }
  ' "$file"
}

# skill_frontmatter <skill-md> — the YAML between the delimiters.
# skill_body <skill-md>        — everything after the closing delimiter.
#
# Two names rather than one with an argument, because the argument would be the
# same string at every call site and a caller that mistyped it would silently
# get the other half.
skill_frontmatter() {
  skill_section frontmatter "$1"
}

skill_body() {
  skill_section body "$1"
}

# verdict_guard_ready
#
# Succeed when every primitive this file exists to provide is defined. This is
# the one question a caller asks after sourcing, so the set lives here rather
# than being re-listed at every call site — where it would be re-listed
# incompletely, and each omission would be a primitive whose absence nothing
# catches.
#
# Defined last on purpose: a file that did not reach the end does not define
# this either, so "stopped short" fails by the same route as "never had it".
verdict_guard_ready() {
  local g
  for g in json_string cannot_compute require_tool json_document_conforms \
           payload_is_conforming skill_section skill_frontmatter skill_body; do
    declare -F "$g" >/dev/null 2>&1 || return 1
  done
  return 0
}
