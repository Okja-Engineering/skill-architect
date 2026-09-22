# Ship gate — PR #22, the v0.5.0 release commit

**PR:** #22 · head `9c8ba53` · base `main` · MERGEABLE/CLEAN · CI `test` pass · 10 files.
**Branch:** `release/0.5.0`. **Not merged. Not tagged.**
Worker's record: `.scuba/teams/release-0.5.0/s10-record.md`.

## Why this PR gets a full gate

It changes almost no code. It is gated hard anyway, because **it is the only artifact
in the release that makes claims about the whole release.** The equivalent gate on
0.4.3's release commit found three false claims, one of which the repo's own suite
asserts must be refused. The gate on PR #5 found six. This is the highest
false-claim-density artifact this project produces.

The release's own subject matter is claims that assert more than the code delivers.
A false sentence in these notes is not a typo; it is the defect the release is about.

## Seeded findings — already CONFIRMED by the chief of staff, do not re-litigate

**Seven false `0.5.0` forward-promises survive in Go files at PR head.** S10's record
says `types.go:85` is "the only false 0.5.0 marker left in the tree". That
enumeration is wrong. Confirmed by `git grep` at `origin/release/0.5.0`:

- `profiler/types.go:81` — "the shape the Devin and Cursor adapters need in 0.5.0"
- `profiler/types.go:84` — "reserved for 0.5.0"
- `profiler/types.go:85` — "reserved for the Devin and Cursor adapters in 0.5.0"
- `profiler/claude_code.go:475` — "tracked with it for 0.5.0"
- `profiler/claude_code.go:611` — "tracked with it for 0.5.0"
- `profiler/otlp.go:1130` — "tracked for 0.5.0 with the skipped data points"
- `profiler/cmd/main_test.go:369` — "is new surface for 0.5.0"

All three adapters were **deleted deliberately** on a measured finding, so "reserved
for the Devin and Cursor adapters in 0.5.0" is a promise the release refutes. The
"tracked for 0.5.0" sentences are the *same sentence* S10 correctly rewrote in
`docs/profiler-spec.md` — it fixed the spec's copies and left the code's copies.
`docs/profiler-spec.md:72` ("gained its `Error…` constructor in 0.5.0") is **true
past tense and correct** — do not flag it.

Root cause is the chief of staff's own mandate, which said no Go file may change and
so forced a report instead of a repair. **Do not spend your round re-finding these.
Your job is to find the ones I have not found.** Assume my enumeration is also
incomplete, because that is this project's recurring defect and it has now recurred
in the mandate, in S10's record, and in every release so far.

## Your posture

Each of you takes ONE lens. Work in **your own worktree**; never `checkout`,
`reset`, `clean` or `stash` in a shared tree. The primary tree at
`/Users/matthewvandusen/Development/Auraprix/skill-architect` is at `0a83615` with
~2,900 lines of uncommitted work — **do not touch it.**

- **Prove, don't reason.** Run the command. A claim you did not drive against the
  code is SUSPECTED, not REAL.
- **Enumerate the class, not instances.** Give a **coverage line**: the denominator
  you walked and how you know it is the denominator. A findings list with no
  denominator is an early-stop and will be rejected.
- Cite `file:line`. Label REAL or SUSPECTED. State the invariant each finding breaks.
- You do **not** fix. Read and run only.
- Any fix you prescribe is **advisory** — it will be re-derived at the root.

Return your report inline AND write it to
`/Users/matthewvandusen/Development/Auraprix/skill-architect/.scuba/teams/release-0.5.0/gate-<your-lens>.md`
by absolute path, with the Write tool. **Never a Bash heredoc** — it truncates
silently on a broken shell and reports success, which has already cost this project
two lost reports. If you have no Write tool, say so explicitly, verify by `ls`, and
return the full report inline.
