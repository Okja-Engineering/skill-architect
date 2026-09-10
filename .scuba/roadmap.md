# skill-architect — roadmap
**Updated** 2026-09-10 02:55 · `scribe` · mirror `scuba-state/imagineux-gmail-com@59d7dbf`

## Now active
_One or two lines per currently-moving thread — what's happening right now._
- 🟡 **E-CS · Cursor hook telemetry** — spec gate round 4: conformance 3 MED + 6 LOW (no HIGH); verifiability 1 HIGH (S0 bypasses the single token writer) + 8 MED. Final revision round 4 running; will present to user after with a Known-residue section. → [status](teams/cursorscope-go/status.md)
- ✅ **E-CS · deep research** — ✅ deep-research done: no existing tool attributes cost per skill run in Cursor; report in `../cursor-profiler/docs/research/existing-per-skill-cost-tools.md`. → [status](teams/cursorscope-go/status.md)
- ✅ **E-CS · research collection** — ✅ six research files + deep-research report collected into `../cursor-profiler/docs/research/`. → [status](teams/cursorscope-go/status.md)
- ✅ **research-profiling** — ledger landed at `teams/research-profiling/ledger.md`; folded into the E-CS grooming. → [status](teams/research-profiling/ledger.md)
- ✅ **F04 · paired comparisons** — slices 1–3 are on `main` (`0a83615`, `236152c`, `e0e2a52`); its status file `tmp/teams/architect/F04.status.md` still reads "parked" and is **stale** — trust git, not that file.

## Decisions waiting on me
_(Empty when none; never bury a decision.)_
1. **E-CS forks and the top-3 user questions** — ingest model, OTLP sequencing, code location, naming, deps, token honesty → [context](teams/cursorscope-go/roadmap.md) §D and §E
2. **Cursor availability** — `~/.cursor/` is **absent on this machine**; will the user install Cursor and the hook? If not, the hooks half (S1–S5, S7, S9–S11) is unbuildable-as-verified and the epic collapses to **S0 + T1** → [context](teams/cursorscope-go/roadmap.md) §E Q1
3. **Dispatch S0 now?** — bug fix to the already-merged `profiler/cursor.go`; independent of the epic once the spec gate is CLEAN → [context](teams/cursorscope-go/slices/S0.status.md)
4. **Concurrent writer in `../cursor-profiler/docs/research/`** — its `research-contradictions.md` claims the `cursor.*` names are unverified; the conformance hunter verified all 21 against Cursor's wire reference (keys wrong, names real). Reconcile which session owns that directory. → [context](teams/cursorscope-go/review/hunter-conformance.md)

## Roadmap

```mermaid
flowchart TD
  L([skill-architect]):::root

  L --> CS[🟡 E-CS · Cursor hook telemetry for the profiler]:::spec
  L --> RP[✅ Skill profiling + token efficiency research]:::done
  L --> T1[💤 T1 · static BPE token cost in skill-audit]:::parked

  CS --> SC[🟡 SC · profile contract v1.1]:::spec
  CS --> SP1[🟡 SP1 · beforeReadFile gating spike]:::spec
  CS --> S0[🟡 S0 · fix Cursor OTel adapter vs wire reference]:::spec

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

  click CS "teams/cursorscope-go/roadmap.md" "Groomed roadmap — 17 slices, revised round 3"
  click RP "teams/research-profiling/ledger.md" "Research ledger"
  click T1 "teams/cursorscope-go/slices/T1.status.md" "Parked — separate parallel thread"
  click SC "teams/cursorscope-go/slices/SC.status.md" "Contract slice — ships first"
  click SP1 "teams/cursorscope-go/slices/SP1.status.md" "Gating spike — hard gate ahead of S3"
  click S0 "teams/cursorscope-go/slices/S0.status.md" "Slice status — dispatchable independently"
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
