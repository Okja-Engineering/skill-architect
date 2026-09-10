# SP1 — GATE: does Cursor's skill loading fire `beforeReadFile`? (+ fixture harvest)

- **Stage:** ready in code terms — **blocked only on User Question 1**
- **Owner:** unassigned (needs the user at a machine running Cursor)
- **Branch:** none (target: `epic/skill-scope`)
- **Depends on:** none — needs no code from us
- **Blocks:** **S3 (hard gate)**; S5 and S7's value case follow S3

## Goal
Answer R-RL-17 **before** anything is built on it, and harvest the one real session that turns every hand-authored fixture in this epic into a verified one. Moved ahead of S3 at the spec gate: a spike that can void S3/S5/S7 must gate them, not sit downstream of them.

**Widened at round 3.** The source-supply sweep found three assertions that the docs cannot settle and only a real capture can. They are cheap to answer in the same session, and each one currently forces a Design decision made blind. They are now DoD items 2–4.

## Definition of done
1. **The gate.** With a hand-installed `beforeReadFile` hook (a five-line script appending its stdin to a file — no `skill-scope` binary required) and a skill in `.cursor/skills/`, a prompt that exercises the skill produces a capture that either **does** or **does not** contain a `beforeReadFile` event whose `file_path` matches `SKILL_PATH_RE` — `/(?:^|[/\\])skills[/\\]|SKILL\.md$/i`, **External evidence**, cursorscope `src/attribution.js:1`. The answer is recorded as a **Repository fact** with the Cursor version, and one hand-redacted capture is committed as a fixture.
2. **The attribution join (S3 DoD 3, D11).** Does the same `(conversation_id, generation_id)` window also contain a `preToolUse` / `postToolUse` whose `tool_input.file_path` matches `SKILL_PATH_RE`? **Yes → S3 ships Form A** (`Attribution.Target` = `tool_use_id`, resolving to a `ToolCallEntry.ID`). **No → S3 ships Form B** (`Target` = `file://<path>`, generation-scoped). `beforeReadFile` itself carries no `tool_use_id` (ledger §A5.1:127), so this cannot be answered from the docs and the roadmap must not pretend otherwise.
3. **The outcome carriers (S1b DoD 2, D11).** Register `preToolUse`, `postToolUse`, `postToolUseFailure`, `beforeShellExecution` and `afterShellExecution`, then run one command that succeeds and one that fails, and interrupt a third mid-flight. Record: (a) does a shell command produce a `postToolUse`/`postToolUseFailure` pair, or only the `*ShellExecution` pair? (b) does the interrupted call arrive as `postToolUseFailure` with `is_interrupt: true` (ledger §A5.1:122)? (c) does `afterShellExecution` carry an undocumented `exit_code` (R-CS-15, ledger §A5.1:124)? If (a) is "only `*ShellExecution`", shell rows have no outcome carrier and S1b's orphan rule makes them `aborted` — lossy, and the manager must be told before S1b is dispatched.
4. **Timestamps (D11).** Confirm by inspection that **no** captured event carries a timestamp field of any kind. The ten documented base fields contain none (§A5.1:113) and no payload row at :117-137 has one, but D11 commits the whole epic's timing story to a hook-stamped `received_at`, so a single counter-example is worth catching now rather than after S1a ships. If a timestamp *is* present, D11's timing half is reopened and `TimingData` gains a real Cursor source.

## Test approach
`tests/test_spike_activation.sh` (new) asserts against the **committed capture**, not against a live editor:
- if the spike answers **yes**: the capture contains ≥1 `beforeReadFile` whose `file_path` matches `SKILL_PATH_RE`, and ≥1 event carrying each of the base fields **that event is documented to supply** — the 10 documented base fields are `conversation_id`, `generation_id`, `model`, `model_id` (opt), `model_params` (opt), `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email`, `transcript_path` (**Specification**, ledger §A5.1:113), and `workspaceOpen` supplies only four of them (§A5.1:137), so the assertion is per-event, not blanket. This is the fixture S1b/S3 assert against;
- if the spike answers **no**: the same test pins the negative (no `SKILL.md` read in a session that demonstrably used the skill), and **S3 is withdrawn rather than built**.
- **DoD 2–4 are pinned the same way, either direction:** a grep for a `preToolUse` with a `SKILL_PATH_RE` `tool_input.file_path` (records Form A vs Form B); a grep for `postToolUseFailure` + `is_interrupt`, and for `exit_code`; and a **negative** grep asserting no captured event carries a timestamp field. Each records its answer in the spike file as a **Repository fact**, so the Design decisions D11 rests on stop being blind.
Either way the test is deterministic in CI because it reads a committed file.

## Requirements
R-RL-17 · confirms or refutes R-CS-09 (**Local hypothesis**, ledger §A5.1:143: *"Local hypothesis (needs a spike)"*) and R-CS-15 (whether a real `afterShellExecution` payload carries `exit_code`, which Cursor's docs do not list — ledger §A5.1:124) · settles the open half of **D11** (attribution join form, outcome carriers, timestamps)

## Files
**Committed to the branch:** `tests/fixtures/cursor-hooks/session-01.jsonl` (new, hand-redacted), `tests/test_spike_activation.sh` (new).
**Written to the control plane, NOT to the branch:** `.scuba/teams/cursorscope-go/spike-r-rl-17.md` — the recorded answers to DoD 1–4. Control-plane state lives on the human's branch, not inside a worker's tree; round 1 wrongly listed it among the files this PR commits onto `epic/skill-scope`.

## Next
**This is not a "~40-line PR."** It commits a hand-redacted real session capture plus a bash test, and writes four recorded answers to the control plane. Size it as a fixture-bearing PR whose review cost is *reading the fixture*.

**Redaction is manual here.** S4's redaction code does not exist yet, so the committed capture must be hand-reviewed line by line for emails, absolute paths, bearer tokens and prompt text before it is committed. Commit only what a reviewer has read.

While the session is open, capture one instance of **every** hook event that fires, not just `beforeReadFile` — that same capture retires §G1 ("no hook payload has ever been observed locally"), validates S1a's 21 hand-authored fixtures, and checks S1a DoD 6's non-base field list, which is transcribed from the docs rather than observed. If the user is on Enterprise (User Q2), also export the OTel stream: that would retire §G5 and give S0 a real fixture instead of a hand-authored one.
