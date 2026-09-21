# S4 record — the activation is the event, not the attribute

**Implementer** senior-implementer · **Date** 2026-09-20 · **Base** `origin/main` = `b5338c8`
**Branch** `feat(profiler)/0.5.0-skill-activation` · **Commit** `4b5d45e` · **PR** [#16](https://github.com/Okja-Engineering/skill-architect/pull/16) (open, not draft, base `main`, CI **pass 3m7s**)
**Worktree** `<scratch>/wt-s4-activation` (session scratchpad)
**Status** delivered, not merged. The user merges. Nothing tagged.

---

## 1. The design decision this slice turned on, and why it went the way it did

The slice was handed to me as "the Claude Code adapter reads `skill.name`
activation", with the note that "the fixture for a present `skill.name` already
ships". Read literally, that means `skill_name_present.json` flips to `present`.
**It must not, and it does not.** This is the one place the brief's summary
wording and the repo's own contract point in different directions, so it is
recorded first.

Claude Code exposes skill identity in two shapes:

| Shape | What it is | Timestamp on the record |
|---|---|---|
| `claude_code.skill_activated` | logged when a skill is invoked, through the Skill tool or a `/` command, and **only then** | the invocation |
| `skill.name` attribute on `token.usage`, `cost.usage`, `api_request`, `api_error`, `api_refusal` | marks a skill active **for that request** | a metric flush, or a request |

`ActivationEntry` is `{skill_name, timestamp, trigger}` — an activation event.
Reading the *attribute* as an activation manufactures both halves: a skill used
across five requests carries it five times, so the count is five activations the
harness logged none of, and each entry's timestamp is a flush or request time
rather than an invocation time. That is the proto3-zero fabrication class 0.4.1
and 0.4.2 spent two releases closing, and the standard the plan states in its own
words — never report what you cannot measure.

**`main` already decided this, four times over**, and the plan's S4 cites exactly
that authority ("`main:205` names it as 0.5.0 work in its own source"):

- `profiler/claude_code.go:205` — the deferral names `claude_code.skill_activated`.
- `README.md:196` — "Reading `claude_code.skill_activated` is 0.5.0."
- `docs/profiler-spec.md` step 6 — the same sentence.
- `CHANGELOG.md:190` (0.4.3) — "`claude_code.skill_activated` … the source it will read".

**The snapshot disagrees, and the snapshot is the superseded draft.** Drawn per D3
and hash-checked before reading (`docs/profiler-spec.md` →
`0cfb63b…c5c96`, matches the manifest; `README.md` → `33980fe…7420f`, matches),
the frozen spec says at line 209:

> `SkillActivation` → one `ActivationEntry` per distinct `skill.name` on token/cost
> usage metrics, `present`/`otel`; `unknown` when no `skill.name` attributes exist.

That is the pre-0.4.x design, in the same file that still describes the generic
`MetricResult[T]` that v0.4.1 removed — **the "snapshot's spec is behind `main`"
warning, landing on this slice's own subject.** It also cannot be implemented
honestly: a distinct-name set over metric data points has no invocation time to
put in `ActivationEntry.Timestamp` and no trigger. **Nothing was copied.**

So: the adapter reads `skill.name` **of the activation event**, which satisfies
the brief's phrasing and the repo's contract at once. `skill_name_present.json`
becomes the **negative control** — an export thick with the attribute, logging no
event, staying `unknown` — which is a stronger test than flipping it would have
been.

**Manager: this is the one place to overrule me if I read it wrong.** It is a
three-line change to the extractor and a fixture row if you want the attribute
read as well; it is not a three-line change to make that honest.

---

## 2. How activation is extracted and scoped

`extractActivations(export scopedExport) ActivationResult`, beside the three
extractors that were already there and following their shape exactly.

- One entry per `claude_code.skill_activated` record carrying a `skill.name`.
- `Trigger` from `invocation_trigger` when the event carried one; absent
  (`omitempty`) when it did not. The trigger is optional; the name is not — an
  activation of no named skill is one a reader can do nothing with, so it is
  counted and not listed.
- A record whose clock could not be read is **kept** and sorted last, the same
  rule a tool call gets: the event was read, only its time was not.
- Names are reported exactly as spelled. `skill_activated` redacts user-defined
  and third-party skills to `custom_skill` without `OTEL_LOG_TOOL_DETAILS=1`;
  un-redacting would invent a name nobody recorded. `skill_activated.json`
  carries a `custom_skill` entry so that rule is pinned, not just documented.

**Scoping is by construction, not by discipline.** The extractor takes a
`scopedExport` and can never be handed the file — which is what the type is
*for*, per `provenance.go`'s own comment, and it is why adding a fourth signal
did not reopen the defect. Read in full before editing, as the plan required.
Three consequences, each pinned:

1. Another session's activation is not in this profile.
2. A record carrying **no `session.id` at all** is in nobody's profile.
   `ownsAttrs` treats "carries a different id" and "carries none" as the same
   answer, which is the honest one: an activation the adapter cannot attribute
   is not reported under whichever session was asked for.
3. A `skill_activated` under `some.other.product` is that product's, whatever
   session id it carries.

Every `unknown` reason carries `export.logsNotRead()`, so "no
`claude_code.skill_activated` log events found in OTel export" cannot be read as
"the export logs none" — it names how many records the projection removed and
why.

---

## 3. Fixtures

| Fixture | Case | New? |
|---|---|---|
| `skill_activated.json` | **present** — three events written newest-first (so file order ≠ timestamp order), one with no trigger, one with no clock, one named `custom_skill` | new |
| `skill_name_present.json` | **absent**, and the negative control: thick with `skill.name`, logs no event | existing, row rewritten |
| `unreadable_activation_events.json` | seen, nothing readable: no `skill.name`, and an empty one | new |
| `foreign_scope_activation.json` | **foreign scope** beside the harness's own, same session id — only the harness's is read | new |
| `foreign_scope_activation_only.json` | the file's only activation is another product's — `unknown`, reason names the scope exclusion | new |
| `two_sessions_activation.json` | **multi-session** — four records interleaved: 1× session A, 2× session B, 1× carrying no `session.id` | new |
| `full_export.ndjson` | gained one `skill_activated`, so "the happy path carries every signal" stays true and the contract table gets the advertised direction | edited |

The multi-session fixture is asserted in **both** identities (A's profile holds
only A's; B's holds only B's, in clock order, not file order) plus a third
identity absent from the file — the half that catches a filter matching
everything. The unattributable record is asserted absent from both.

`TestEveryFixtureIsAContractCase` makes the fixture directory the denominator, so
all five new files have a `captureCases` row; that check is what stopped a
fixture being added and never asserted over.

---

## 4. S3's pre-planted trap: found before it was tripped

S3 §10.1: *"S4 must add `MetricSkillActivation` to `otelBackedSignals` … or an
export that fails will assert `unknown` where the adapter now correctly says
`error`."*

**It did not bite. It was read from the handoff and set up front, before the
implementation existed**, so it drove the design instead of ambushing it. The
first RED run — with only the event-name constant added and no extractor —
printed it verbatim, seven times:

```
--- FAIL: TestCaptureDeliversEverySignalProbeAdvertises/malformed_JSON
    profiler_test.go:974: skill_activation: state = "unknown", want "error"
```

That is what forced the two repairs the trap is really about:

- **`ErrorActivationResult` had to exist.** `types.go` said, in a doc comment,
  that `ActivationResult` has no `Error` constructor because "skill activation is
  a property of the harness and of this adapter, not of any export, so no export
  can fail it". True until the read existed. The constructor is added, the
  comment is repaired, and `AttributionResult`'s and `EstimatedTokensResult`'s
  comments — which pointed at `ActivationResult` as their precedent — are
  re-pointed at `AttributionResult`, which is now the one signal the claim is
  still true of.
- **`erroredSignals`/`unknownSignals` had to settle activation.** A failing
  export now errors activation with the other three.

Both directions were then mutated (§6, M5 and M6): removing
`ErrorActivationResult` from `erroredSignals` reddens 7 cases, and removing
`MetricSkillActivation` from `otelBackedSignals` reddens the same 7 the other
way. The trap entry is load-bearing, proved.

### S3's derivations: verified, not assumed

S3 §7 said three hardcoded counts became derivations and a two-name condition
became a lookup, so "the sixth signal should mostly flow". **Verified. It did.**

| S3's derivation | Held? |
|---|---|
| `len(report.Capabilities) != len(profile.SignalStates())` (×2 in `profiler_test.go`, ×1 in `main_test.go`) | yes — untouched by this slice; activation was already the sixth signal |
| `advertising()` built from `SignalStates` in the contract stubs | yes — the new activation stubs needed no capability-map edit |
| `if otelBackedSignals[metric]` replacing the two-name condition | **yes, and this is the one that mattered** — widening the set was a one-line edit, where the old `metric == MetricSkillActivation \|\| metric == MetricAttribution` form would have needed the condition rewritten |
| `(Profile{}).SignalStates()` as the probe denominator | yes |

Nothing in `profiler/cmd` needed changing at all: `SignalStates()` is still six,
so `main_test.go`'s positional `profileWith` guard (S3 §10.5) did not fire.

---

## 5. Integrations, not a fourth branch

Three shape repairs, each because the alternative was a second copy of a rule.

**5.1 `otelSignals.resolved()` — one enumeration of the export-backed signals.**
Activation had to be added to `capabilityReport`'s map *and* to `failureReasons`'
slice. Two hand-written lists of the same set is how a signal comes to be
advertised by one and explained by neither. Both now derive from
`resolved() []resolvedSignal`, ordered (not a map) because the diagnostics are
printed and a reader comparing two runs should not have to sort them.
`capabilityReport` seeds the two signals with no read behind them —
`attribution`, `estimated_context_tokens` — and walks `resolved()` for the rest.

**5.2 `timedEntry[T]` / `orderedEntries[T]`.** Tool calls and activations share a
rule with a subtle edge: *untimed entries last, in file order*. A second copy is a
second place to get it wrong. `timedCall` became `timedEntry[ToolCallEntry]`,
`orderedEntries` became generic, and the timestamp read is factored into
`recordTime(r) (string, int64, bool)`. The existing tool-call ordering tests
cover the refactor, so it is a mechanical one under TDD's stated exception.
Generics are already in the file's idiom (`entries[T any]` in `otlp.go`); the
repo's "type parameters are avoided" rule is stated about the JSON result types
and is untouched.

**5.3 Two hand-written three-signal walks in the tests became derivations.**
`TestProvenance_ASessionAbsentFromTheExportYieldsNoValue` and the fallback-reason
loop in `TestCapture_ClaudeCode_WithoutOtel` each listed tokens/tool_calls/timing
by hand. Both now walk `sortedMetrics(capabilities)` filtered by
`otelBackedSignals`. That is what made activation covered there automatically —
and it is where the one real defect in this slice turned up (§6).

---

## 6. Mutation proofs — twelve, and one of them caught me

Every check was reverted and watched to redden. Restores were done by `cp` from a
scratch backup; **no `checkout`, `reset`, `clean`, `stash` or `rebase` was run
anywhere**, in the worktree or out of it.

| # | Mutation | Result |
|---|---|---|
| M1 | `extractActivations` not called; activation back to a constant | RED — **12 top-level tests** |
| M2 | keyed on the `skill.name` attribute instead of the event | RED — reports an activation of `third-party` read off an `api_request` record: the fabrication, demonstrated |
| M3a | the extractor handed a projection that drops the session test | RED — session A's profile lists `[session-a-skill unattributable-skill session-b-first session-b-second]` |
| M3b | `ownsScope` stops excluding a foreign scope | RED — both new foreign-scope tests, plus the three that already existed |
| M4 | an event with no `skill.name` becomes an activation of nothing | RED — `[{SkillName: …} {SkillName: …}]` |
| M5 | `erroredSignals` leaves activation `unknown` | RED — 7 cases (**S3's trap from the other side**) |
| M6 | `MetricSkillActivation` removed from `otelBackedSignals` | RED — 7 cases (**S3's trap as planted**) |
| M7 | activations returned in file order | RED — the newest-first fixture inverts |
| M8 | the activation reason drops `logsNotRead()` | RED — all three provenance reason tests |
| M9 | the contract drops the **none** direction of rule 2 | RED — the tokens case **and** the new activation case |
| M10 | `invocation_trigger` not read | RED |
| M11 | the contract drops the **advertised** direction of rule 2 | RED — the tokens case **and** the new activation case |
| M12 | the derived signal walk finds nothing | **first run: GREEN — a real defect, in this slice's own test** |

### M12 — the guard that did not guard

The non-vacuity guard I wrote for the derived walk was

```go
if want := len(otelBackedSignals); walked != want { … }
```

and emptying `otelBackedSignals` satisfies it on **both** sides: `0 != 0` is
false, so every assertion in the loop passed by reading nothing and the suite
reported green. That is precisely the vacuity S3 caught one file over
(`len(derived) == 0`), recreated one slice later by a different route — which
suggests the pattern is worth a lint rather than a memory (§9.4).

Repaired to two guards that are not one guard twice: `walked == 0` catches the
derivation finding nothing, and the equality catches the capability report
falling behind the set. Same repair applied to the fallback-reason loop, which
had no guard at all. M12 re-run: **RED in both**, naming each.

---

## 7. The contract table sees activation in both directions

**Direction 1 — advertised must be delivered.** `full_export.ndjson` now logs an
activation, so `claude_code`'s contract fixture makes `probe` advertise
`skill_activation: otel` and `probeCaptureAgreementViolations` requires the
capture to be `present`, with that source, carrying a value, carrying no reason.
Before this change the contract's only export had no activation, so only the
`none` direction was ever exercised for it.

**Direction 2 — unadvertised must not appear.** Exercised over every other
fixture in `captureCases`, and now mechanically at the contract table too.

**Both directions are proved failable, over activation specifically.**
`stubAdapter` gained an `activation` field so `TestTheContractTableCanFail` can
lie about a second signal, and two cases were added:

| Stub | Violation it must produce |
|---|---|
| `over-advertising-activation` | `skill_activation: probe advertised "otel", capture state is "unknown"` |
| `undeclared-activation` | `skill_activation: probe advertised "none" and capture returned "present"` |

M9 and M11 confirm each is what catches its direction: neutering one branch
reddens exactly one of the two new cases alongside its tokens counterpart. A
table whose only liar lies about tokens is a table nobody has shown reads the
rest of the report it iterates; that is no longer true here.

---

## 8. Verification — measured, not carried

Base figures re-derived at `b5338c8` before any edit. All three reproduced S3's
record exactly (2555 / 107 / 571).

### Shell suites, both shells

| suite | base | head, bash 3.2.57 | head, bash 5.3.15 |
|---|---|---|---|
| test_f01 | 1868 | 1868 | 1868 |
| test_f02 | 350 | 350 | 350 |
| test_harness | 131 | 131 | 131 |
| test_install | 48 | 48 | 48 |
| test_rewrite | 85 | 85 | 85 |
| test_skill | 53 | 53 | 53 |
| test_walk | 20 | 20 | 20 |
| **total** | **2555** | **2555 passed, 0 failed** | **2555 passed, 0 failed** |

Unchanged, and that is the expected result rather than a null one: this slice
touches no shell-suite surface. `tests/test_walk.sh` walks the skills, not the
profiler docs (verified by grep: no suite references `docs/profiler-spec.md`
content, and `test_skill.sh`'s only profiler assertion is the `AdapterVersion`
string, which S3 set and this slice did not touch).

### Go, in `profiler/`

| gate | base | head |
|---|---|---|
| `gofmt -l .` | empty | empty |
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `go test -race -count=1 ./...` | ok | **ok**, both packages |
| top-level tests | 107 | **114** |
| PASS lines incl. subtests | 571 | **607** |
| failures | 0 | **0** |

The seven new top-level tests: three activation extraction tests
(`ReadsTheSkillActivatedEvent`, `AnEventWithNoSkillNameIsNotAnActivation`,
`SkillNameOnARequestSignalIsNotAnActivation`) and four provenance tests
(`ActivationsAreTheProfiledSessionsOnly`,
`ASessionWithNoActivationIsNotGivenAnothersOrToldTheExportHasNone`,
`AForeignScopesActivationIsNotAttributed`,
`AnExportWhoseOnlyActivationIsForeignYieldsNone`).
`TestCapture_TheHarnessLevelReasonsAreTheSpecsWordForWord` was renamed to
`TestCapture_TheAttributionReasonIsTheSpecsWordForWord`, because its premise —
"skill_activation and attribution are a property of the harness, not of any
export" — is now half false. It also asserts the retired deferral sentence
appears in no profile, over six inputs.

### CI

`gh pr checks 16` → **pass, 3m7s** (ubuntu-latest, bash 5, Linux git).

---

## 9. Findings — what should change before slice five

### 9.1 The snapshot's activation draft is wrong on the merits, not just stale. **[carry]**

Recorded in §1 because a later reader of the snapshot could reasonably implement
it. The snapshot's spec §209 is the design `main` itself corrected in 0.4.3. If
the snapshot is ever re-taken or re-read for S5–S8, this is the second
independent confirmation (after S2 and S3) that **its spec must be read for
intent only, never applied.**

### 9.2 `cmd/main_test.go` carries a comment that is now doubly wrong. **[note, not mine]**

Line 74: `{name: "an export carrying every signal", otel: fixture("full_export.ndjson"), want: 0, about: "three signals were read"}`.
The case captures under `--session s`, which matches no record in the fixture, so
**zero** signals are read and the expected exit code of 0 is right for a reason
the `about` string does not give. It was already wrong before this slice; the
fixture now carrying four signals makes it wrong twice. Left alone deliberately:
it is a pre-existing inaccuracy in a file outside this slice's `git add` list, and
S5 touches `cmd/main_test.go` anyway.

### 9.3 A reason that could mislead, and the case for leaving it. **[decide]**

An export full of `skill.name` attributes and no `skill_activated` event reports
`no claude_code.skill_activated log events found in OTel export`. Literally true,
and it does not claim the export says nothing about skills — but a user whose
skill plainly ran may read it that way. The house pattern for exactly this is
`logsNotRead()`: a clause stopping a reason being read as a stronger claim than
it is. **I did not add one**, on the ground that `README.md` now explains the
distinction at length and the clause would be a new counter for a case that is
not a defect. If a reader trips on it, the clause is ~10 lines plus a fixture row.

### 9.4 The vacuous-derived-denominator pattern has now bitten twice. **[decide]**

S3 caught it in `SignalStates` (`len(derived) == 0`); §6/M12 caught it here, in a
guard written *by someone who had read that record*. Two occurrences in two
slices, both caught only by mutation. The cheapest mechanical answer is a
convention the remaining slices are told once: **a derived denominator gets a
`> 0` guard, never an equality against another derivation of the same thing.**
S7 adds a signal and will derive again.

### 9.5 `estimated_context_tokens` is still advertised and never produced. **[unchanged, S7's]**

S3 §10.2 stands. This slice did not touch it. The contract table enforces AC9
both ways over it, so S7 cannot advertise it without delivering it.

### 9.6 The five plugin manifests still differ from `AdapterVersion`. **[unchanged, S10's]**

S3 §10.3 stands, untouched here.

### 9.7 Smaller notes

- The activation extractor's `activationCounters` has one defect class today.
  It is a struct with a `reason()` method anyway, matching the token and
  tool-call counters, so a second class is an added clause and not a reshaped
  sentence. Judged worth the ten lines; flag if you disagree.
- Attribute names (`skill.name`, `invocation_trigger`) are inline string
  literals, matching how `tool_name`, `success`, `decision` and `type` are
  already spelled in that file. Only `otelSessionAttr` is a constant, because
  `provenance` needs it by name. No new convention introduced.
- `full_export.ndjson` was edited rather than a sixth fixture added. It is
  documented as "the happy path … all signals present", and after this release
  the happy path includes an activation. Blast radius was four assertions, all
  of which are the slice's headline changing.

---

## 10. Not done, and why

| Not done | Why |
|---|---|
| Merging or tagging | The user merges. Nothing tagged. |
| Touching any version surface | `AdapterVersion` stays at S3's `0.5.0`, `ProfileSchema` stays `profile/v1`, the five plugin manifests stay `0.4.3` for S10. |
| Reading `skill.name` off request-scoped signals | §1 — it would fabricate a count and a set of times. Named, not silently skipped. |
| `CHANGELOG.md` / `RELEASE_NOTES.md` | S10's. Their 0.4.3 entries describing the deferral are historical records of 0.4.3 and remain true of it. |
| Fixing `cmd/main_test.go`'s `about` string | §9.2 — pre-existing, out of this slice's add list. |
| The `.out-of-scope.md` bullet S1 deferred | Still S5's. Untouched. |
| Rebuilding the snapshot | Frozen. Read-only, hash-checked, not written to. |

## 11. Hygiene

- `git add` was **fourteen named paths**. No `git add -A`, no `commit -a`.
- `tests/lib/out-of-scope-check.sh` run over the staged index before committing:
  `out-of-scope check: clean`, exit 0. Used the committed script rather than
  re-deriving the pipeline, per S3 §10.4.
- Author and committer `imagineux <imagineux@gmail.com>`. AI-attribution grep
  over commit message, author, committer, PR title and PR body: **0 matches**.
  The only hits from a looser grep are `claude_code` (the harness identifier) and
  "Claude Code" (the product being profiled).
- No PATH mirror built, nothing installed or reconfigured, no live config
  directory written. No network writes beyond `git push` and `gh pr create`.
- **Primary working tree verified untouched after the slice:** still `0a83615`,
  47 `git status --porcelain` lines — identical to the figure S3 recorded.
