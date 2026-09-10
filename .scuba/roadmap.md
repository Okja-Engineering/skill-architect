# skill-architect — roadmap
**Updated** 2026-09-10 00:47 · `scribe` · mirror `scuba-state/imagineux-gmail-com@pending`

## Now active
_One or two lines per currently-moving thread — what's happening right now._
- 🟡 **E-CS · Cursor hook telemetry** — mandate drafted; awaiting research fold-in and the user's fork calls before grooming into slices. → [status](teams/cursorscope-go/status.md)
- 🟢 **research-profiling** — researcher running on skill profiling + token-efficiency evidence; ledger not yet written. → [status](teams/research-profiling/ledger.md)
- ✅ **F04 · paired comparisons** — slices 1–3 are on `main` (`0a83615`, `236152c`, `e0e2a52`); its status file `tmp/teams/architect/F04.status.md` still reads "parked" and is **stale** — trust git, not that file.

## Decisions waiting on me
_(Empty when none; never bury a decision.)_
1. **E-CS mandate forks** — ingest model, OTLP, location, naming, deps, token honesty → [context](teams/cursorscope-go/mandate-draft.md) §7 and §10

## Roadmap

```mermaid
flowchart TD
  L([skill-architect]):::root

  L --> CS[🟡 E-CS · Cursor hook telemetry for the profiler]:::spec
  L --> RP[🟢 Skill profiling + token efficiency research]:::exec

  subgraph PRIOR["Prior epics (superseded control plane in tmp/teams/architect/)"]
    F01[✅ F01 · format/policy split]:::done
    F02[✅ F02 · audit.json static scope]:::done
    F03[✅ F03 · harness-agnostic profiler adapters]:::done
    F04[✅ F04 · paired comparisons, slices 1-3]:::done
    F05[💤 F05 · evidence-linked candidates]:::parked
    F06[💤 F06 · regression coverage]:::parked
  end

  L --> PRIOR
  F01 --> F02
  F02 --> F03
  F03 --> F04
  F04 --> F05
  F05 --> F06
  F03 --> CS
  F04 --> CS

  click CS "teams/cursorscope-go/mandate-draft.md" "Mandate draft — spec → plan → brief"
  click RP "teams/research-profiling/ledger.md" "Research ledger"
  click F01 "../tmp/teams/architect/F01.status.md" "Done — status"
  click F02 "../tmp/teams/architect/F02.status.md" "Done — status"
  click F03 "../tmp/teams/architect/profiler-design-space.md" "Done — design space (no status file)"
  click F04 "../tmp/teams/architect/F04.status.md" "Done for slices 1-3 — status file stale"
  click F05 "../tmp/teams/architect/F05.status.md" "Parked — status"
  click F06 "../tmp/teams/architect/F06.status.md" "Parked — status"

  classDef root fill:#F1EFE8,stroke:#5F5E5A,color:#2C2C2A
  classDef spec fill:#FAEEDA,stroke:#854F0B,color:#412402
  classDef plan fill:#E6F1FB,stroke:#185FA5,color:#042C53
  classDef exec fill:#EAF3DE,stroke:#3B6D11,color:#173404
  classDef review fill:#EEEDFE,stroke:#534AB7,color:#26215C
  classDef blocked fill:#FCEBEB,stroke:#A32D2D,color:#501313
  classDef done fill:#E1F5EE,stroke:#0F6E56,color:#04342C
  classDef parked fill:#F1EFE8,stroke:#888780,color:#2C2C2A
```

_Node labels carry the stage emoji (🟡 spec · 🔵 plan · 🟢 execution · 🔎 review · ⛔ blocked · ✅ done · 💤 parked); colour comes from the matching `classDef` — don't invent new ones. Click a node to open its artifact; artifacts chain **spec → plan → executive brief**. Per-thread recovery detail (branch · worktree · last SHA · next · blocker) lives in each thread's `status.md`._
