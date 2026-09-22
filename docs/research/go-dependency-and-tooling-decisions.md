# Go dependency and tooling decisions

Collected 2026-09-10 from `skill-architect/.scuba/teams/research-profiling/ledger.md` Q2/Q3/Q4,
`.scuba/teams/cursorscope-go/mandate-draft.md` §5, and
`.scuba/teams/cursorscope-go/review/hunter-verifiability.md`. Evidence labels per `AGENTS.md`.

Four decisions, each with the tradeoff named.

---

## 1. Go OTel SDK vs hand-rolled OTLP/JSON — **hand-roll**

**Repository fact — measured 2026-09-10 with Go 1.27.1 via `proxy.golang.org`, two throwaway builds.**

Current module versions:

| Module | Version |
|---|---|
| `go.opentelemetry.io/otel` | v1.46.0 (2026-08-25) |
| `go.opentelemetry.io/otel/sdk` | v1.46.0 |
| `.../exporters/otlp/otlptrace/otlptracehttp` | v1.46.0 |
| `.../exporters/otlp/otlpmetric/otlpmetrichttp` | v1.46.0 |
| `.../exporters/otlp/otlplog/otlploghttp` | **v0.22.0** (logs still pre-1.0) |
| `go.opentelemetry.io/otel/log` | v0.22.0 |
| `go.opentelemetry.io/proto/otlp` | v1.11.0 |

Measured cost of a minimal `otlptracehttp` + `otlploghttp` program vs a stdlib-only program that
POSTs OTLP/JSON to `:4318/v1/traces`:

| Metric | OTel Go SDK | stdlib OTLP/JSON |
|---|---:|---:|
| Direct modules | 2 | **0** |
| Indirect modules | 22 | **0** |
| Packages compiled | **408** | — |
| `google.golang.org/grpc` packages compiled | **66** (despite HTTP-only transport) | 0 |
| Also pulls | `genproto`, `grpc-gateway/v2`, `protobuf` | — |
| arm64 binary | **17.7 MB** | **6.4 MB** |

**Context:** `skill-architect`'s `profiler/go.mod` declares **no dependencies at all** today
(**Repository fact**), and its self-contained principle exists precisely to protect that.

**Recommendation: do not adopt the OTel Go SDK for export.** Hand-roll OTLP/HTTP JSON
(`Content-Type: application/json` to `{endpoint}/v1/traces|/v1/logs|/v1/metrics`), a first-class,
spec-mandated OTLP encoding every collector accepts, using `encoding/json` + `net/http` only.

**Tradeoff, named honestly.** You give up batching, retry/backoff, the semconv constant packages,
context propagation, and automatic resource detection — and you take on schema drift against a moving
OTLP JSON schema. Acceptable here because the profiler emits a bounded, hand-authored set of ~6 span
shapes and ~5 metrics, not arbitrary third-party instrumentation. cursorscope's own 71-line
`test/fake-collector.mjs` shows how small the receiving surface is.

**Escape hatch.** If a future requirement forces the SDK (gRPC transport, or consuming third-party
instrumentation), isolate it: put the exporter behind an interface in a **separate Go module** under
`profiler/exporters/otlp/` so the core module stays dependency-free.

**Note separately:** for *reading* Cursor's OTel export — what the Cursor adapter actually does today
— **no OTel dependency is needed at all**. `encoding/json` over a collector-written file is correct
and already implemented.

---

## 2. Semgrep — **do not adopt**

**Environment fact first: semgrep is not installed on the research machine**
(`command -v semgrep` → not found), so nothing below was empirically verified. It is reasoning from
documented capability and should carry a **Local hypothesis** label until a spike runs.

| Question | Answer |
|---|---|
| Can it estimate token cost? | **No.** That is a character/BPE counting problem, and `skill-validator check -o json` already answers it exactly with `o200k_base`. |
| Can it analyze `SKILL.md` prose? | Only via `generic` mode (whitespace-tolerant token matching) — roughly what `grep` already gives you. Markdown is not a language semgrep parses semantically. |
| What *can* it do that grep cannot? | (a) Parse **YAML frontmatter** structurally to enforce always-loaded budget rules. (b) Lint a skill bundle's **`scripts/` bash and Go** for determinism / error-handling / no-absolute-path rules — squarely its home turf and the one place it beats every alternative. (c) Detect always-loaded bloat patterns (large fenced blocks inlined in `SKILL.md`) — but a small Go pass over the markdown does that more accurately without a Python runtime. |
| Can it tell us whether a skill was loaded, when, how often, at what cost? | **No.** That is unreachable by static analysis and needs runtime signals. |

**Two cautions that decided it:**

1. **Semgrep is distributed as a Python package.** `AGENTS.md` bans Python for skill scripts to
   preserve self-containment. Introducing a Python CI dependency **to enforce a "no Python" rule** is
   at minimum ironic and deserves an explicit decision, not a silent one.
2. Adopting it adds a CI install step for a tool that duplicates what is already installed.

**Recommendation: do not add semgrep for skill profiling or token estimation.** Ship the four
guardrail rules — no `python3` in a shell script; no dishonest `present` metric; no unredacted
payload written to disk; no hardcoded `cursor.*` telemetry string outside the one declaring file — as
`grep` + `go vet` + Go tests. Consider semgrep only as an optional, opt-in linter for the `scripts/`
directory of a *target* skill bundle, invoked as an external tool with graceful degradation when
absent, and never on the critical path of an audit.

**Tradeoff:** you forgo a mature rule ecosystem for bundled scripts, in exchange for keeping the
toolchain self-contained and single-runtime.

**⚠ Conflict to resolve before implementing guardrail rule 2.** The proposed rule — "reject a
`MetricResult` constructed as `present` from a function whose name contains `estimate`" — **forbids
exactly what R-RL-06 mandates** (`SourceEstimated` with `state: present`). One of the two must move.
Recommended fix: the rule should reject `present` + `Source` ∈ {`otel`, `hooks`} from an `estimate*`
function, not `present` itself.

---

## 3. jq vs Go for the `hooks.json` merge — **decide it; currently ambiguous**

**Repository fact.** cursorscope's `scripts/install-global-hooks.sh` performs its
`~/.cursor/hooks.json` merge via an embedded `python3` heredoc (verified in the clone), which
`AGENTS.md` forbids. The requirement itself (R-CS-21) is right and carefully done: non-clobbering
merge, timestamped backup, cleanup of its own legacy entries, uninstall removes only its own entries.

| Option | For | Against |
|---|---|---|
| **`jq`** | Already installed (`/usr/bin/jq`, **Repository fact**); `AGENTS.md` says prefer existing tools; the merge is ~10 lines of jq; keeps install logic in the same bash script that already does backup and chmod | An external binary the install path now hard-depends on; jq is not universally present on every machine; error handling in bash is weaker |
| **Go subcommand** | Zero external dependency; the same binary already has to exist for the hook itself; testable with Go table tests; consistent error handling and `--dry-run` | Install script becomes "bash that shells into our own Go binary", which is a bootstrapping wrinkle if the binary is not yet built |

**Recommendation: Go subcommand** (`<binary> hooks install|uninstall`), with the bash installer as a
thin wrapper. Rationale: the merge is the single highest-risk operation in the whole install path — it
edits a user's live editor config — and it is the one place where `--dry-run`, byte-identical
round-trip testing, and precise error messages are worth more than brevity. A jq one-liner cannot be
unit-tested to the standard "a pre-existing foreign hook survives install *and* uninstall
byte-identical" demands.

**Tradeoff:** the installer can no longer run before the binary is built. Mitigate by having the
installer fail loudly with a build instruction rather than silently falling back.

**⚠ Note:** the current plan says `jq` in one place and a Go command in another. This is an open
ownership question, not a settled decision.

---

## 4. Profiler contract limits — constraints a new design must respect

**Repository fact** — from `skill-architect/profiler/types.go`, `profiler/compare.go`, and
`docs/profiler-spec.md`, reproduced by the spec-gate hunter with a probe program. These are hard
limits of the **existing v1 contract**; a design that assumes otherwise is untestable as written.

### L-1. `CapabilityReport` has exactly one source per metric, and no reason field

```go
type CapabilityReport struct {
    Harness      string                      `json:"harness"`
    AdapterVer   string                      `json:"adapter_version"`
    ProbedAt     string                      `json:"probed_at"`
    Capabilities map[MetricName]MetricSource `json:"capabilities"`
}
```

Consequences, all of which invalidate DoDs written against them:

- **"`Probe()` reports `tokens: none` with a reason" is untestable.** There is no reason field on a
  `CapabilityReport`. Either add one, or move the reason to the `MetricResult` that `Capture()`
  returns and rewrite the DoD accordingly.
- **"`Probe()` reports hooks and OTel independently" is impossible.** `map[MetricName]MetricSource`
  admits one source per metric; a second assignment silently overwrites the first. Supporting two
  simultaneous sources for one metric requires a `types.go` change (e.g.
  `map[MetricName][]MetricSource`), which is a **schema-affecting decision above slice level.**
- **"the fallback says so in `Source`" has no vocabulary.** The `MetricSource` enum is
  `otel|hooks|session_data|server_api|sqlite|none` — there is no member meaning "aggregate fallback".

### L-2. `compareTokens` drops the baseline's source label

```go
func compareTokens(baseline, candidate TokenResult) MetricComparison {
    if baseline.State != MetricPresent || candidate.State != MetricPresent {
        return MetricComparison{Comparable: false, Reason: "token data not present in both profiles"}
    }
    return MetricComparison{
        Comparable: true,
        Source:     candidate.Source,   // ← baseline.Source is never read
        ...
    }
}
```

**Proven failure mode.** An `estimated` baseline compared against a `server_api` candidate yields
`comparable: true, source: "server_api", delta: {input: 3000}` — **the estimate's honesty label
vanishes from the report, and the delta is arithmetic between a guess and a bill.**

This is the single most important constraint on the whole `SourceEstimated` design: adding the
constant to `types.go` does **not** make the system honest, because the comparison layer launders it.
A new design must either (a) refuse to compare across differing sources, (b) carry both sources in
`MetricComparison` and mark such comparisons `comparable: false` or heavily qualified, or (c) both.
**No slice in the current plan owns this.**

### L-3. `present` is defined as "captured from telemetry"

`types.go:12` — `MetricPresent MetricState = "present" // value captured from telemetry` — and
`docs/profiler-spec.md` repeats it: `State == "present"` → `Value` populated, `Reason` empty, `Source`
always populated "so the profile is auditable".

**A chars/4 estimate is not captured from telemetry.** So `SourceEstimated` with `state: present` is
in direct tension with the contract's own words, even though `Source` is the right mechanism to carry
the honesty level. **Whichever way this lands, `docs/profiler-spec.md` must be amended in the same
change** — and currently only one slice owns that file, while four DoDs depend on distinctions it
does not yet express.

### L-4. Additive-field claims still touch the spec

Adding `ActivationEntry.Source` (R-RL-05) is additive to the Go struct but **contradicts the
documented v1 shape** in `docs/profiler-spec.md` unless that doc is updated in the same change. The
slice proposing it excludes the spec doc from its file list.

---

## 5. Summary of recommendations

| Decision | Recommendation | Tradeoff accepted |
|---|---|---|
| OTel Go SDK | **Hand-roll OTLP/HTTP JSON**; isolate the SDK in a separate module if ever forced | Own the wire format; no batching/retry; track spec drift |
| Semgrep | **Do not adopt**; `grep` + `go vet` + Go tests | Forgo a mature rule ecosystem for bundled scripts |
| `hooks.json` merge | **Go subcommand**, bash wrapper | Installer depends on a built binary |
| Profiler contract | **Treat `types.go` + `docs/profiler-spec.md` as one owned, versioned artifact**, not incidental edits | A schema-owning work item must land before the DoDs that depend on it |
