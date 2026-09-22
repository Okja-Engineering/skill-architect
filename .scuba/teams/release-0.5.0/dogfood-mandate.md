# Dogfood gate — run the profiler on a real session, then review the data

**Raised by the user, directly:** *"for example the profiler we want to be able to
run this profiler and verify that it correctly collects some data and then review
that."*

**This gate blocks PR #22.** The v0.5.0 release commit does not merge until this lane
reports. The release ships nine profiler subcommands and **every one of them is
verified against fixtures only** — 64 OTLP fixtures, 275 Go tests, 2649 shell
assertions, and not one observation of a session the profiler actually watched.

`AGENTS.md:28` makes dogfooding a release gate but scopes it to "skill-architect's own
tools on its own skills" — the skill audit. **The profiler is outside that scope**, so
it has never been gated this way. That scope gap is itself a finding to report.

## The question

Not "do the tests pass." **Does the profiler, pointed at a real Claude Code session,
produce numbers that are true about that session?**

Three sub-questions, which are the project's own stated 1.0 promise:
1. Does it collect data at all, from a live emitter rather than a fixture?
2. Are the numbers **right** — checkable against something that is not the profiler?
3. What does it silently fail to see?

## Hard safety boundary — read this before running anything

**Never write to the user's real config.** `~/.claude` is live and in use by the
session reading this. `profiler hooks install` writes hook configuration; run it only
with `CLAUDE_CONFIG_DIR` pointed at a scratch directory you created. Same for
`CODEX_HOME`. Before any command that writes config, print the target path and
confirm it is inside your scratch area.

**Do not touch the primary working tree** at
`/Users/matthewvandusen/Development/Auraprix/skill-architect` — it is at `0a83615`
with ~2,900 lines of uncommitted work. Use your own worktree at PR head
(`origin/release/0.5.0` @ `9c8ba53`).

**Do not break the global `skillscore` install.** A previous worker destroyed it by
building a PATH mirror of symlinks and writing a stub through one. Do not build PATH
mirrors.

**Telemetry is not currently configured on this machine** — no OTLP env vars, no
telemetry keys in `~/.claude/settings.json`. Enabling it is part of your job, in a
scratch config only. Leave the machine as you found it and say so explicitly.

## Ground truth is the point

A profile that says "1.2M tokens" is worthless to this gate unless you can check it
against something that is not the profiler. Establish ground truth independently —
the session's own transcript, `/cost`-style reporting, the raw export you fed in,
counting records yourself with `jq`. **State for every number whether you verified it
against an independent source, and what that source was.** A number you could only
check against the profiler's own output is unverified; say so rather than blessing it.

This matters more than coverage here. One token count proved right against an
independent source is worth more than nine subcommands that ran without crashing.

## Report

Write `.scuba/teams/release-0.5.0/dogfood-<lane>.md` with the **Write tool**, by
absolute path. **Never a Bash heredoc** — it truncates silently on a broken shell and
reports success, which has already cost this project two reports that were claimed
written and did not exist.

Include: the commands you ran, the raw artifacts you produced and where they are
(keep the export and the profile — the user wants to review the actual data), every
number with its independent check or an explicit "unverified", every defect with
`file:line`, and a plain verdict.

**Classify each defect:** does it need a CODE FIX before 0.5.0 ships, or a DISCLOSURE
line in the release notes, or is it 0.6.0 scope? Be conservative about code fixes —
0.5.0 is feature-complete and the gate is not the place to add capability. A limit
that is disclosed honestly is shippable; a limit the notes deny is not.

If you cannot generate real telemetry at all, that is a legitimate and important
finding. **Report it as a blocked verification, with what you tried and what it would
take.** Do not simulate a session and present it as a real one — a fixture wearing the
word "live" is the exact defect this whole release is about.
