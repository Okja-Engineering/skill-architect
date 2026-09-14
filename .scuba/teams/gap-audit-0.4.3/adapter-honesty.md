# Gap audit — lens: adapter-honesty

Audited `origin/main` @ `5c847e1`. Verdict: **NOT CLEAN** — 1 P1, 2 P2, 7 P3.

> Persisted by the chief of staff: the hunter's session had Write disabled, so it
> returned this report inline. Findings below are the hunter's; the **Verified by
> chief of staff** notes are independent confirmation against the tree.

## Coverage

1 adapter (enumerated from the tree) x 5 signals x 67 inputs = **335 signal-outcomes
executed**, each through both `probe` and `capture`. Inputs = 55 committed fixtures +
12 synthetic exports built to reach uncovered conditions. Harness registry exercised
for all 4 names plus empty and wrong-case. Spec AC9 checked mechanically over all 67.
One cross-version differential run. Statement coverage confirmed no adapter branch
went unwalked.

**Only one adapter exists on main.** `grep 'Probe() CapabilityReport' / ') Capture('`
at `5c847e1` returns exactly one: `ClaudeCodeAdapter` (`profiler/claude_code.go:52`).

## Shared root

Six findings share one root: **the adapter applies no provenance filter to what it
reads.** `resolve()` (`claude_code.go:87`) takes no identity argument, so no extractor
below it can scope anything — no session scoping, no instrumentation-scope check, no
resource check — while the result is stamped with a caller-asserted identity the data
never justified. A root fix closes F1, F3, F6 and changes F5/F7. Patching one signal
leaves the other two. Fix at the root, not per signal.

---

## P1

### F1 — `Capture` ignores `sessionID` · `claude_code.go:173,188` · REAL · 0.4.3

`sessionID` is written to `Profile.SessionID` and passed nowhere else. The contract
says the opposite (`types.go:94`, `claude_code.go:172`, `docs/profiler-spec.md:121-122`).
Claude Code puts `session.id` on every record, README:236 says so, and the project's
own fixtures carry it. No test references it.

Reproduced through the README's own recipe. Both documented routes append to one file
by design:

```
session_id : 22222222-2222-4222-8222-222222222222
tokens     : present {"input": 49200, "output": 9440}  source=otel
TRUTH      :          input 48000 / output 9100
```

`tool_calls` lists both sessions' tools. A two-session synthetic gave
`total_ms: 99999000` (27.8 hours). A session ID appearing nowhere in the export yields
the same confident numbers.

**Invariant**: a profile stamped with a session ID must contain only signal derived
from records carrying it, or must not be `present`.

`--session` is required by `cmd/main.go:79` and then ignored. "Required and ignored"
should not survive the fix.

> **Verified by chief of staff.** `grep -n 'sessionID\|SessionID'` over
> `claude_code.go` at `5c847e1` returns exactly two lines: the parameter at `:173` and
> the struct field at `:188`. `resolve()` takes no argument. `tokens_only.json` carries
> 6 session references. `cmd/main.go:79` requires `--session`. Confirmed REAL.

---

## P2

### F2 — `AdapterVersion` stale across a value-changing fix · `types.go:234` · RESOLVED

`types.go:232-234` states the bump rule; README:54-57 promises profiles "stay
distinguishable". `bd32b26` (PR 3) exists to change what a profile contains. The
constant was still `"0.4.1"` at `2ed34b8`, `bd32b26` and `5c847e1`. Proven
differentially on one fixture:

```
v0.4.1-code (2ed34b8) -> adapter_version= 0.4.1  tokens= {"input": 200}
main        (5c847e1) -> adapter_version= 0.4.1  tokens= {"input": 300}
```

A stored 0.4.1 profile against a fresh one shows a 50% token regression that is
entirely the reader changing.

> **RESOLVED by PR #5**, which sets `AdapterVersion = "0.4.2"`. Verified by chief of
> staff against `origin/release/0.4.2-version:profiler/types.go:234`. The mislabelled
> window is `bd32b26..5c847e1`, which exists only on main and was never released. **No
> 0.4.3 action.** The hunter flagged this without knowing PR 5 was in flight; the
> finding was correct at the time it was made.

### F3 — Foreign log events attributed to Claude Code · `claude_code.go:221-230`, `otlp.go:428` · REAL · 0.4.3

`eventName` strips the `claude_code.` prefix then unconditionally re-adds it, and
`logRecords()` filters on no scope or resource. Any record named bare `tool_result` or
`api_request` is promoted. Reproduced with records from scope `some.other.product`:

```
tool_calls: present [{"name":"SomeOtherProductTool",...},{"name":"ForeignViaAttr",...}]
```

Reachable: the documented route listens on `127.0.0.1:4318`, the standard OTLP port.
Metrics are unaffected (exact-name match); the log path only.

---

## P3

- **F4** `claude_code.go:441-443` REAL 0.4.3 — comment claims "the reason below says
  so" for mid-run accepts; the reason is built only when `len(calls)==0`. 1 result + 2
  accepts gives `present` with an empty reason, one call for a session that made three.
  `spec:244` gets it right; the comment contradicts it.
- **F5** `claude_code.go:466` vs `:477` REAL 0.5.0 — reject and ran-and-failed both emit
  `Success:false`. Schema change, so not patch-legal.
- **F6** `claude_code.go:577-616` REAL 0.5.0 — span excludes `api_error`/`api_refusal`;
  `spec:231` names two other exclusions but not this.
- **F7** `otlp.go:1048`, `:1024`, `claude_code.go:331` REAL 0.5.0 — reports
  `{"input": 9223372036854775807}` with no marker that it is a ceiling.
- **F8** `types.go:287,322` REAL 0.5.0 — `PresentActivationResult` and
  `PresentAttributionResult` at 0.0% coverage, unreachable rather than untested.
  `SourceHooks`/`SessionData`/`ServerAPI`/`SQLite` and `Reasoning` have no producer;
  `types.go:56` advertises four harnesses.
- **F9** `claude_code.go:208` SUSPECTED 0.5.0 — "carries no output-to-skill mapping" vs
  `spec:239`'s own rule; `prompt.id` joins `api_request(skill.name)` to
  `tool_result(tool_use_id)`, so it is derivable.
- **F10** `claude_code.go:190` REAL, no action — `skill_dir` verbatim and unvalidated,
  explicitly documented as such.

---

## Verified clean — explicit negatives

- **The Devin suspicion is REFUTED.** No Devin, Cursor or Codex adapter exists at
  `5c847e1`. All three exit 1; README marks them "Planned", accurately. `ExportFile`
  and `APIKey` are refused in two places deliberately so they cannot drift — the honest
  pattern to measure the rest against. The placeholder Devin adapter is unpushed local
  work and should be audited before it lands in 0.5.0.
  > **Verified by chief of staff**: `git ls-tree origin/main profiler/` returns no
  > cursor, codex or devin file.
- **The v0.4.1 partial-export class is genuinely closed** on all three signals, verified
  in both directions on every fixture and synthetic partial.
- **The proto3-zero class is closed in the value path.** Absent and `null` value leaves,
  absent vs explicit `0` vs `UNSPECIFIED` temporality, and `startTimeUnixNano: 0` all
  return `unknown`, never a zero. Inverse holds: `zero_token_count` gives explicit
  zeros; `cache_only` leaves input and output absent.
- **Wrong-kind attributes never fabricate.** `success` as `intValue:1` drops the record
  rather than becoming `false`.
- **AC9 holds over all 67 inputs, zero violations.** The single shared `resolve()` makes
  probe and capture one question, which is why there is no probe-overclaim finding.
- **`Probe` never fails**; file-access classification matches `spec:254`/`:269` in every
  case.
- **The partial-present undercount is real but correctly ledgered** — reproduced on all
  three signals, and the spec states it plainly and defers it to 0.5.0. Largest honesty
  gap in the product, and honestly disclosed. Only F4 defects the area.
- **Harness claims check out against Anthropic's live telemetry docs.** `type` is exactly
  `input`/`output`/`cacheRead`/`cacheCreation`, so `Reasoning` genuinely has no source
  as `types.go:111-116` claims. The hunter tried to refute the
  `claude_code.skill_activated` claim and could not.

## Triage

**0.4.3 patch: F1, F3, F4.** F1 and F3 are one root fix — provenance filtering in
`resolve()` and the extractors. F4 is a comment. None needs a schema change, so all are
patch-legal. F2 is resolved by PR #5.

**0.5.0: F5, F6, F7, F8, F9.** F5 and F7 want the same channel the skipped-record count
already wants; design together.

## Reproduction artifacts

Baseline `go test ./...` green before and after. No product code edited.

- Worktree: `scratchpad/wt-adapter`
- Synthetic inputs: `scratchpad/synth`
- Binaries: `scratchpad/prof` (main), `scratchpad/prof041` (v0.4.1 code)

Sources: Claude Code monitoring docs; anthropics/claude-code issue 35319.
