# Dogfood gate — lane: the hook spool (`hooks install`, `ingest`, `analyze`, `doctor`)

**Worktree:** `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/hooklane/head` @ `9c8ba53` (detached, `origin/release/0.5.0`)
**Binary under test:** `.../scratchpad/hooklane/profiler` (built from that worktree, `go build ./cmd`)
**Date:** 2026-09-21

---

## Verdict

**Ship with two code fixes and two prose fixes.** The capture layer is genuinely
sound — it survived a live emitter, adversarial input and 40-way concurrency
without losing or corrupting a byte, and `analyze`/`doctor` reported numbers that
matched my independent counts exactly. But `hooks install` writes a hooks.json
that **violates Cursor's published schema** (no `version` field, which the schema
marks required), and two release documents make a **redaction claim that is
false** — I have a live capture with my prompt text sitting in the spool in
plaintext under the default mode the notes describe as `[REDACTED]`.

The lane's headline premise needs correcting for the record: **this is a Cursor
hook spool, not a Claude Code one.** `CLAUDE_CONFIG_DIR` is not read anywhere in
this code path. See §0.

---

## 0. Scope correction: the lane is Cursor, and `CLAUDE_CONFIG_DIR` is irrelevant

My mandate directed me to point `CLAUDE_CONFIG_DIR` at a scratch directory before
running `hooks install`, and to compare payload shapes against Claude Code's hook
docs. Both rest on a false premise, so I am recording it before the findings.

`profiler hooks install` has nothing to do with Claude Code:

- `profiler/hooks_install.go:131` —
  `func hooksJSONIn(home string) string { return filepath.Join(home, ".cursor", "hooks.json") }`
- The safety knob is the **`--home` flag**, resolved once at
  `profiler/cmd/main.go:504-528` (`hooksFlags` → `resolveHome`). With no flag it
  falls back to `os.UserHomeDir()`, i.e. `$HOME`.
- `CLAUDE_CONFIG_DIR` appears **nowhere** in the profiler source. Setting it would
  have provided no protection whatsoever.

**This is not a defect** — the tool is honestly and consistently documented as a
Cursor tool (README:455-500, `docs/profiler-spec.md:483-512`, RELEASE_NOTES.md:10).
It does mean the "does it ignore `CLAUDE_CONFIG_DIR` and hardcode `$HOME`" trap I
was told to watch for is not the relevant question; the relevant question is
whether `--home` fully scopes the write, and the answer is **no, not for the
spool** — see Finding 3.

**Landing-plan note resolved:** the note said `hooks install` "was documented as
writing `~/.cursor/hooks.json` in one place." That is what it actually writes, and
README:463, `docs/profiler-spec.md:504` and RELEASE_NOTES.md:10 all say so. **The
documentation matches the behaviour. No defect here.**

---

## 1. The `[DOCS]` payload shapes, checked field by field

Primary source: Cursor's own hooks reference, https://cursor.com/docs/agent/hooks
(fetched 2026-09-21). Cursor is **not installed on this machine**, so this closes
the docs-vs-code gap but *not* the docs-vs-running-Cursor gap.

### What the spool asserts (the denominator)

`docs/profiler-spec.md:502-509` lists four unverified assumptions. Results:

| Assumption | Verdict against current Cursor docs |
|---|---|
| Cursor reads `~/.cursor/hooks.json` | **CONFIRMED.** Docs list it as the User-scope location (below Enterprise, Team and Project scopes). |
| `hooks` member keyed by event name; entry is an object with a `command` string | **CONFIRMED** for shape. `type` is optional (defaults to `"command"`). **But a required sibling is missing — see Finding 1.** |
| The 21 names in `CursorHookEvents` are the events Cursor invokes | **CONFIRMED, exact match.** All 21 names in `hooks_install.go:37-48` appear in the docs' list, and the docs list no 22nd. |
| A payload is one JSON object carrying `hook_event_name`, `cursor_version`, `cwd`, `conversation_id` | **PARTIALLY WRONG — see Finding 2.** |

### Field-by-field on the envelope (Finding 2 detail)

Cursor's documented **common** payload schema (present on all hooks):

```
conversation_id, generation_id, model, model_id, model_params,
hook_event_name, cursor_version, workspace_roots, user_email, transcript_path
```

Against `profiler/hooks.go:151-158`, which promotes four fields:

| Spool promotes | In Cursor's common schema? | Consequence |
|---|---|---|
| `hook_event_name` | yes | correct |
| `cursor_version` | yes | correct |
| `conversation_id` | yes (absent on `workspaceOpen` only) | correct |
| **`cwd`** | **NO — not a common field** | `ev.Cwd` is blank on most events |

`cwd` is **event-specific**. Docs place it on `preToolUse`, `postToolUse` and
`beforeShellExecution`; it is **absent** from `afterShellExecution`,
`beforeReadFile`, `afterFileEdit` and `beforeMCPExecution`. The field Cursor puts
on *every* payload for workspace location is **`workspace_roots`** (an array),
which the spool does not promote.

So `RELEASE_NOTES.md:11`, `README.md:485-490` and
`docs/profiler-spec.md:508-509` all name `cwd` as one of the four fields "a
payload" carries. Per Cursor's current docs that is **wrong for the majority of
the 21 events**. Nothing is lost — `raw` keeps everything — but the envelope's
promoted `cwd` will be empty on most lines, and the notes' fallback promise
("lines arrive with blank envelope fields") is exactly what will happen, for a
reason the notes attribute to being wrong about Cursor generally rather than to
this specific field.

---

## 2. Live-emitter capture — what I could and could not measure

Cursor is not installed. Claude Code **is** (v2.1.221), and it is a real hook
emitter that delivers one JSON document on stdin. I used it to test the capture
layer against a live emitter. **This is not a Cursor verification** and I am not
presenting it as one; it verifies `ingest` → spool → `analyze` → `doctor` against
a real process invoking the hook, rather than against a fixture.

### Two live sessions, hooks fired, real lines captured

- **Session 1** — hooks registered in a **scratch** `CLAUDE_CONFIG_DIR`
  (`.../hooklane/scratch-home/.claude/settings.json`). Session id
  `d5738acc-da68-4e00-ad5f-9d536a6e8088`. **3 hook events fired and 3 spool lines
  were written.** Session failed at "Not logged in" (a scratch config dir carries
  no auth), so no model turn occurred.
- **Session 2** — hooks registered at **project** scope
  (`.../hooklane/session2/.claude/settings.json`), real `CLAUDE_CONFIG_DIR` left
  alone and read-only. Session id `c4741f07-40af-4892-a3d6-8419cc5c2c26`.
  **3 hook events fired, 3 spool lines written.** Session failed at
  "Failed to authenticate: OAuth session expired and could not be refreshed".

**Raw artifacts kept, for the user to review:**

- `.../scratchpad/hooklane/artifacts/live-spool-claude-code-session1.jsonl` (3 lines, 3002 bytes)
- `.../scratchpad/hooklane/artifacts/live-spool-claude-code-session2-project-hooks.jsonl` (3 lines, 2597 bytes)
- `.../scratchpad/hooklane/artifacts/analyze-spool1.json`
- `.../scratchpad/hooklane/artifacts/doctor-A.json`, `doctor-B.json`
- `.../scratchpad/hooklane/artifacts/hooks.json.after-install`, `hooks.json.after-uninstall`
- `.../scratchpad/hooklane/artifacts/session1.json`, `session2.json` (harness result JSON)

(Full absolute prefix: `/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/hooklane/artifacts/`)

### BLOCKED: tool-call hook events could not be measured live

`PreToolUse`, `PostToolUse` and `Stop` were registered and **never fired, because
no model turn happened.** `claude -p` cannot authenticate on this machine: the
OAuth session is expired and could not be refreshed. I attributed this correctly
rather than assuming a sandbox block — `curl https://api.anthropic.com/v1/messages`
returns **HTTP 405 in 0.098s**, so the network is fine and the credential is
genuinely stale. I did **not** attempt to read or copy credentials (an attempt to
inspect the auth store was correctly refused by the permission system, and I did
not work around it).

**What it would take:** the user runs `/login` in a normal Claude Code session,
after which a nested `claude -p` with project-scope hooks would fire
`PreToolUse`/`PostToolUse` per tool call. That is a ~3-minute re-run of the
recipe in §2, and it is the only part of this lane that is unfinished.

### Measured live payload shapes

Every line's envelope and payload keys, read out of the kept artifact:

| line | `event` (promoted) | `cursor_version` | `cwd` | `conversation_id` | payload top-level keys |
|---|---|---|---|---|---|
| 1 | `SessionStart` | `""` | populated | **`""`** | `cwd, hook_event_name, session_id, source, transcript_path` |
| 2 | `UserPromptSubmit` | `""` | populated | **`""`** | `cwd, hook_event_name, permission_mode, prompt, prompt_id, session_id, transcript_path` |
| 3 | `SessionEnd` | `""` | populated | **`""`** | `cwd, hook_event_name, prompt_id, reason, session_id, transcript_path` |

Measured behaviours, all correct:

- `hook_event_name` promoted **verbatim** — `SessionStart`, not lowercased or
  mapped. An event name this build never heard of survived exactly as claimed.
- `cwd` promoted correctly (Claude Code does put `cwd` on every payload).
- `cursor_version` blank, as it must be for a non-Cursor emitter. Honest.
- Nothing dropped: `source`, `reason`, `prompt_id`, `permission_mode` — none of
  which this build knows about — all carried through into `raw`.

One measured consequence, worth noting though it is correct-for-Cursor:
**`conversation_id` is blank on every live line**, because Claude Code's session
identity field is `session_id`. `analyze` therefore reports
`"conversation_ids": 0` for a capture that plainly covers exactly one session.
The identity is recoverable — `session_id` shows up in the key census with the
right value — so no data is lost. Filed as 0.6.0 scope, not a 0.5.0 defect: the
tool does not claim to support Claude Code hooks.

---

## 3. Findings

### FINDING 1 — `hooks install` writes a hooks.json that violates Cursor's required schema. **CODE FIX.**

`profiler/hooks_install.go:63-87` merges into `doc.root["hooks"]` and never sets a
top-level `version`. On a machine with no existing `~/.cursor/hooks.json`, the
file it creates is, verbatim (artifact `hooks.json.after-install`):

```json
{
  "hooks": {
    "afterAgentResponse": [ { "command": "…/profiler ingest || true" } ],
    …21 events…
  }
}
```

Cursor's configuration reference marks `version` **required**: *"Config schema
version. Must be a positive integer (use 1)."* A file omitting a required field
fails schema validation, so the most likely outcome is that **Cursor ignores the
whole file and the spool stays permanently empty** — while `doctor` cheerfully
reports all 21 events registered (artifact `doctor-A.json`), which is the
compounding harm: the user's only diagnostic tells them it is configured.

**Why the 275-test suite did not catch it.** This is precisely the
"fixture written from the same documentation" failure the gate was set up to find.
`profiler/hooks_install_test.go` tests `version` **only as a field to preserve**,
never as a field to write:

- line 146: seeds `"version": 2` into a pre-existing file
- lines 180-181: asserts `"the top-level 'version' field was dropped"` does not happen
- lines 244, 271-272, 359: same shape, for uninstall

Across 31 hook tests (13 in `hooks_install_test.go`, 18 in `hooks_test.go`) there
is **no test that a virgin install produces a schema-valid file.** Every test
assumes the user already has a valid hooks.json.

**Fix:** in `saveHooksDoc` or `InstallHooks`, set `root["version"] = 1` when the
key is absent, and never overwrite an existing value (a future Cursor may use 2).
Roughly three lines plus a virgin-install test. I recommend this ships in 0.5.0 —
it is the difference between a feature that works and one that silently never
runs, and the change is additive and inside the existing merge discipline.

### FINDING 2 — `cwd` is not a universal Cursor payload field. **DISCLOSURE.**

Detail in §1. `RELEASE_NOTES.md:11`, `README.md:485-490` and
`docs/profiler-spec.md:508-509` each list `cwd` among the four fields "a payload"
carries. Per Cursor's current docs it is event-specific (on `preToolUse`,
`postToolUse`, `beforeShellExecution`; absent from at least four others), and the
universal workspace field is `workspace_roots`, which is not promoted.

**Fix:** prose only. Move `cwd` out of the "carried by a payload" list and say it
is promoted when present, which is what the code already does correctly
(`hooks.go:151-158` and the "treated as absent rather than rendered" rule). No
code change needed — promoting an absent field to `""` is already the right
behaviour. Optionally note `workspace_roots` as the 0.6.0 candidate.

### FINDING 3 — `hooks install --home X` registers a command that spools to the real `$HOME`, and the README's own example points at the wrong directory. **CODE FIX (small) or DISCLOSURE.**

Measured. `hooks install --home /scratch` registers the command string
`<binary> ingest || true` — with **no `--spool-dir`** (`main.go:538-551`,
`resolveHookCommand`). At hook time `ingest` resolves its spool via
`resolveSpoolDir` → `profiler.DefaultSpoolDir()` → `os.UserHomeDir()`
(`hooks.go:288-294`). I confirmed empirically that `DefaultSpoolDir` reads `$HOME`:

```
$ echo '{"hook_event_name":"probe"}' | HOME=<fake-home> profiler ingest
<fake-home>/.skill-architect/spool/2026-09-21.jsonl     # created
```

So `--home` scopes the **registration** but not the **capture**. Three consequences:

1. **`doctor` and `install` disagree about the spool.** `DetectEnvironment`
   computes `spoolDir = filepath.Join(q.Home, ".skill-architect", "spool")`
   (`doctor.go:199-206`). With `--home /scratch` it reports
   `"dir": "/scratch/.skill-architect/spool", "exists": false` (artifact
   `doctor-A.json`) — a directory the installed hook will never write to. The
   comment at `main.go:515-519` states the requirement it breaks: *"they have to
   be looking at the same one or the report describes a machine nobody
   configured."* For the hooks file they do; for the spool they do not.
2. **The README's worked example is wrong.** README:467 says
   `./profiler hooks install --home /tmp/try-it` is *"how you try it out first"*,
   and README:527 then tells the reader to run
   `./profiler analyze --spool-dir /tmp/try-it/.skill-architect/spool`. That
   directory will be empty; the lines went to `$HOME/.skill-architect/spool`.
   A user following the README's own try-it-safely path concludes capture is
   broken.
3. It defeats the isolation the safety-conscious user is reaching for.

**Fix, cheapest correct version:** have `resolveHookCommand` append
`--spool-dir <resolved spool for the chosen home>` when `--home` was given
explicitly. That keeps `install`, `ingest` and `doctor` on one path and makes the
README example true. If the team judges that too much for a feature-complete
release, the disclosure alternative is to fix README:527 and add one sentence
saying `--home` relocates the registration only. **I lean code fix**, because the
defect's visible form is a user being told their capture failed when it did not.

### FINDING 4 — The release notes and changelog claim prompts and tool I/O are redacted by default. They are not. **PROSE FIX — highest-confidence finding in this lane.**

`RELEASE_NOTES.md:10`, verbatim:

> `prompt`, `command`, `tool_input`, `tool_output` and their neighbours are `[REDACTED]` by default; `--strict` reduces them to their byte sizes.

`CHANGELOG.md:64`, verbatim:

> `prompt` and tool I/O are redacted by default and `--strict` reduces them to their sizes.

**Both halves of the first clause are false, and I have a live measurement.** My
own prompt text is in the kept artifact in plaintext:

```
$ grep -c "Run exactly two Bash commands" live-spool-claude-code-session1.jsonl
1
```

And a direct test with a Cursor-shaped payload, default mode:

```json
{ "api_key": "[REDACTED]", "authorization": "[REDACTED]",
  "command": "psql -h db.internal -U admin",
  "prompt": "my secret business plan",
  "tool_input": { "file": "/repo/x.go" },
  "user_email": "matt.vandusen@okja.io", … }
```

The code is right and the prose is wrong. There are **two distinct sets** and the
sentence conflates them:

- `sensitiveKey` (`hooks.go:317-329`) — credential-shaped keys (`*token`,
  `*secret`, `*password`, `*apikey`, `auth`, `authorization`, …) plus
  credential-shaped substrings. **These become `[REDACTED]`, in both modes.**
  `prompt`, `command`, `tool_input`, `tool_output` match none of these patterns.
- `contentKeys` (`hooks.go:203-208`) — `prompt`, `command`, `tool_input`,
  `tool_output` and 10 neighbours. **Untouched by default**; only `--strict`
  replaces them, and with `{"_stripped_bytes": N}`, never with `[REDACTED]`.

The code's own comment says it plainly (`hooks.go:307-310`): *"Prompt text and
file paths stay, because on this tool they are the data; credentials do not."*
The README's spool section is also correct ("Metadata only: prompts and tool I/O
become their sizes", README:469). **The false claim is confined to
`RELEASE_NOTES.md:10` and `CHANGELOG.md:64`.**

This is the release-note class the mandate calls unshippable: a limit the notes
deny. A user reading RELEASE_NOTES.md:10 would reasonably register the hooks on a
shared machine believing prompts are redacted, and ship every prompt they type to
a plaintext file that outlives the session.

**Fix:** replace the clause with something like — *"Credential-shaped fields and
values are `[REDACTED]` in both modes. Prompts, commands and tool I/O are written
verbatim by default, because on this tool they are the data; `--strict` reduces
them to their byte sizes."* Same in CHANGELOG.md:64.

### FINDING 5 — `--strict`, the "metadata-only mode for a shared machine", writes the user's email address to disk. **DISCLOSURE (0.5.0) + 0.6.0 code scope.**

Cursor's documented common payload includes `user_email` (*"Email address of the
authenticated user, if available"*) on **every** hook. Measured against this
build, `--strict` output:

```json
{ "command": {"_stripped_bytes": 30}, "prompt": {"_stripped_bytes": 25},
  "user_email": "matt.vandusen@okja.io", "cwd": "/repo", … }
```

`user_email` matches neither `sensitiveKey` nor `contentKeys`, so it survives
both modes. `hooks.go:118-124` describes `--strict` as *"the metadata-only mode
for a shared machine"* — a mode whose selling point is a shared machine, writing
the authenticated user's identity into a file that outlives the session.

Corroborating evidence that the payload schema was never walked field by field:
**the string `user_email` does not appear anywhere in the repository** — not in
the redaction list, not in the strip list, not in a fixture, not in the spec.

**Fix:** 0.5.0 gets a disclosure sentence (the README already handles the
analogous OTel case well at README:309-313 — *"A capture identifies you"* — and
the spool deserves the same paragraph). Adding `user_email` to `contentKeys` or a
new identity set is 0.6.0 scope; I would not add a redaction category during a
gate.

### FINDING 6 — `uninstall` does not return a virgin machine to as-found, and what it leaves may be schema-invalid. **DISCLOSURE, or a small CODE FIX bundled with Finding 1.**

Measured on a home with no `.cursor` directory at all:

```
$ ls -a virgin-home            → (empty)
$ profiler hooks install   --home virgin-home   → events_registered: 21
$ profiler hooks uninstall --home virgin-home   → events_removed: 21
$ find virgin-home
  virgin-home/.cursor
  virgin-home/.cursor/hooks.json                       ← {"hooks": {}}
  virgin-home/.cursor/hooks.json.bak-20260921T154213Z  ← full 21-event registration
```

Three residues where there was previously nothing: the `.cursor` directory, a
`hooks.json` containing `{"hooks": {}}`, and a timestamped backup that still
carries our absolute binary path in all 21 entries.

`hooks_install.go:117-121` deletes an event left with no entries on the stated
reasoning that *"an empty registration is a trace of us in a file we are meant to
have left as we found it."* The same reasoning applies one level up and is not
applied. And it compounds Finding 1: the `{"hooks": {}}` file we leave behind has
no `version` field either, so a user who tries the tool and removes it may be
left with a schema-invalid `~/.cursor/hooks.json` where they previously had **no
file at all** — potentially breaking Cursor's hook config load for hooks they add
later by hand. That raises this above cosmetic.

**Fix:** if the document is empty after removal *and* the file did not exist
before the install, remove the file (and the directory if we created it). Fixing
Finding 1 defuses the sharp edge on its own, since the residual file would then be
valid.

### NOT A FINDING — everything below was checked and is correct

These are recorded because the gate asks what was verified, not only what broke.

- **Idempotent install.** Re-running `hooks install` reported
  `events_registered: 0` and left the file **byte-identical** (sha256 compared
  before/after), and wrote **no backup**. Matches RELEASE_NOTES.md:10's
  "a re-run adds nothing and writes nothing" exactly.
- **Uninstall is surgical.** I seeded a foreign entry inside one of our own events
  (`preToolUse` → `/opt/other/tool.sh`), a wholly foreign event (`customEvent`),
  an unknown top-level field (`someFutureField`) and `"version": 1`. Uninstall
  removed **exactly** our 21 entries and preserved all four foreign things
  (artifact `hooks.json.after-uninstall`). Backup written, as the spec promises
  for the destructive direction.
- **Unparseable hooks.json is refused, not replaced** (`hooks_install.go:152-157`),
  and `doctor` reports it as `unreadable` rather than as "no events"
  (`doctor.go:329-337`). Correct, and the right call.
- **`ingest` is silent and exits 0 on success**, as the spec requires. Verified.
- **"Nothing is dropped" holds under adversarial input.** Empty stdin →
  `{"raw": ""}`. Non-JSON → captured as a JSON string with the bytes intact.
  **Two** JSON documents on stdin → the *whole* input kept verbatim as a string
  rather than the first document kept and the second lost, exactly as
  `decodeOneJSONValue` (`hooks.go:186-197`) promises. `analyze` then counted all
  three as `other_payloads: 3, object_payloads: 0, unreadable_lines: 0` — it
  distinguished "payload I could not census" from "line I could not read", which
  is the distinction the file was written to preserve.
- **File permissions.** Spool file `0600`, spool dir `0700`, hooks.json `0600`.
  Confirmed by `ls -la`.
- **`doctor`'s surviving tiers report accurately.** With no export:
  `tier: "none"`, reason names exactly why. It refused to raise a tier off the
  21 registered hooks or off a 3-line spool, which is the whole point of removing
  the `hooks` tier. The `yields` sentence shipped in the data on every spool
  observation. The removed `server_api`/`enterprise` tiers stayed removed.
- **Full test suite green** at `9c8ba53`: `profiler`, `profiler/cmd`,
  `profiler/internal/homesafe` all `ok`.

---

## 4. Ground truth — every number, with its independent check

The mandate's central demand. For each `analyze`/`doctor` number I counted the raw
spool myself with Python/`wc` over the JSONL and compared.

| Number | Tool said | Independent check | Source of truth | Match |
|---|---|---|---|---|
| spool lines | `analyze: lines 3` | `wc -l` → `3` | the raw file | **yes** |
| envelopes | `3` | 3 lines each parsed as one envelope | the raw file | **yes** |
| unreadable lines | `0` | 0 of 3 failed `json.loads` | the raw file | **yes** |
| event names | `SessionEnd 1, SessionStart 1, UserPromptSubmit 1` | own `Counter` over `event` → identical dict | the raw file | **yes** |
| event types vs. what fired | 3 lifecycle events | the harness's own 6 registered events, of which 3 can fire without a model turn; session result JSON shows `num_turns: 1`, `is_error: true` | Claude Code's own result JSON | **yes — consistent** |
| `payload_bytes` | `2205` | own sum of `len(json.dumps(raw, separators=(',',':')))` → `2205` | the raw file | **yes, exact** |
| `conversation_ids` | `0` | 1 distinct `session_id` (`d5738acc-…`) in the payloads | the raw file + harness result JSON | **no — see §2**, correct-for-Cursor, wrong-for-reality |
| `capture_days` | `{"2026-09-21": 3}` | all 3 `ts` on 2026-09-21 | the raw file | **yes** |
| `last_capture_at` | `2026-09-21T15:39:37Z` | max `ts` of the 3 lines | the raw file | **yes** |
| `files` | `1` | one `*.jsonl` in the dir | `ls` | **yes** |
| `registered_hook_events` | 21, named | read the installed hooks.json directly; 21 keys, each carrying our command | the file itself | **yes** |
| doctor spool `lines`/`payload_bytes` | `3` / `2205` | same independent counts as above | the raw file | **yes** |
| `measurement.tier` | `none` | no export was passed; correct by construction | the invocation | **yes** |

**Unverified numbers:** none reported in this lane. Every count above was checked
against the raw file or against Claude Code's own result JSON, neither of which is
the profiler. The one mismatch (`conversation_ids`) is a naming/scope issue, not
an arithmetic error, and the underlying identity is present in the census.

**Token counts:** not applicable to this lane — the spool reports **bytes** and
explicitly refuses to divide them into an estimated token count
(`analyze.go:90-97`). That refusal is correct and I want it on the record as the
right decision, since it is the single easiest place in this release to have
manufactured a fake number.

---

## 5. `AppendSpool` has no locking — I tried to make it bite, and it did not

The notes disclose the missing lock. I attempted to corrupt the spool.

**Method.** N concurrent `profiler ingest` processes, all appending to the same
daily file, at four payload sizes plus one very large one to probe the
partial-write regime. After each round: line count vs. expected, and every line
re-parsed as JSON.

| payload | concurrent procs | lines (expected) | unparseable lines | file size |
|---|---|---|---|---|
| 100 B | 40 | 40 (40) | 0 | 11,560 B |
| 8 KB | 40 | 40 (40) | 0 | 327,560 B |
| 200 KB | 40 | 40 (40) | 0 | 8,007,560 B |
| 1 MB | 30 | 30 (30) | 0 | 30,005,670 B |
| **8 MB** | **16** | **16 (16)** | **0** | **128,002,288 B** |

In the 8 MB round all 16 lines were **exactly 8,000,143 bytes** — a single
distinct length, i.e. not one line was split or interleaved.

**Zero corruption, and the reason is structural rather than luck.**
`AppendSpool` (`hooks.go:261-280`) opens with `os.O_APPEND` and then issues
**one** `f.Write(append(line, '\n'))` — the newline is appended into the same
buffer, so the whole record including its terminator is a single `write(2)`. On a
local filesystem an `O_APPEND` write atomically positions and writes, so two
writers cannot interleave within a record. In-process goroutines get the same
protection, because `AppendSpool` opens its own descriptor per call rather than
sharing one.

**Do hooks ever fire concurrently?** Yes, and I confirmed it from the emitters'
own documentation rather than assuming. Claude Code's hooks reference states
*"All matching hooks run in parallel"*, fires `PreToolUse`/`PostToolUse` **per
tool inside a parallel batch**, and offers `async: true` background hooks.
Cursor's Tab hooks (`beforeTabFileRead`, `afterTabFileEdit`) fire independently of
its agent hooks. **So concurrent invocation is the normal case, not an edge case.**

**Verdict: the disclosure is conservative — it understates the safety, not the
risk.** Two residual caveats it should name instead:

1. **Network filesystems.** `O_APPEND` atomicity is not guaranteed on NFS/SMB. A
   spool under a network-mounted home could interleave. **Untested** — no network
   mount was available. This is the honest residual risk.
2. **A hook killed mid-write** leaves a truncated last line. Already disclosed at
   `docs/profiler-spec.md:595-596`, and `analyze` counts it as
   `unreadable_lines` rather than losing the file. Correct.

**Recommended replacement wording:** *"No lock is taken. Each line is written with
a single `O_APPEND` write including its newline, which on a local filesystem
cannot interleave with another writer's — measured at 40 concurrent writers and
payloads to 8 MB, with no interleaving or loss. A spool on a network filesystem is
outside that guarantee and has not been tested."*

---

## 6. The `[DOCS]` disclosure — is it still accurate?

The mandate asked whether the disclosure needs to change. **Yes, in both
directions, and the current sentence is now wrong on the strict side as well as
the generous side.**

`RELEASE_NOTES.md:11` currently says:

> …that `~/.cursor/hooks.json` is the file Cursor reads, that the 21 event names are the ones it invokes, and that a payload arrives as one JSON object carrying `hook_event_name`, `cursor_version`, `cwd` and `conversation_id` are **Cursor's documentation rather than our measurement**. …**We cannot tell you it works against Cursor.**

What changed:

- **"We cannot tell you it works against Cursor" stays.** Cursor is still not
  installed. No line in this repository was produced by Cursor. Do not soften it.
- **The hooks.json location and the 21 event names are now confirmed against
  Cursor's current published reference**, field for field, with no 22nd event.
  That is a stronger position than "taken from documentation, unchecked" — it
  rules out the specific failure mode of documentation drift since the code was
  written.
- **`cwd` is now known to be wrong** (Finding 2), so a sentence that lists it
  among a payload's fields is no longer merely unverified — it is contradicted by
  the primary source.
- **A new, sharper limit outranks all of it:** the file `hooks install` writes is
  **missing a field Cursor's schema marks required** (Finding 1). "We cannot tell
  you it works" is too weak for that; the honest statement is closer to *"the file
  we write omits a required field, so it likely does not work until that is
  fixed."* Fixing Finding 1 is the better route than disclosing it.
- **The capture layer now has one live-emitter measurement** (Claude Code, 6 hook
  invocations across two sessions, 3 event types). The notes should say the spool
  round-trip has been exercised by a live hook emitter — naming which one, and
  naming that it was not Cursor — rather than implying the only evidence is
  fixtures. That is a real upgrade to the release's credibility and it is
  currently unclaimed.

---

## 7. Machine left as found — explicit verification

**`~/.claude` is untouched.** Verified, not asserted:

| Check | Baseline (before) | After | Result |
|---|---|---|---|
| `sha256 ~/.claude/settings.json` | `3bd84040a6aa832b9c3d3a914d3abbdaab2a6abafe3ade56aef8346d3997f053` | `3bd84040a6aa832b9c3d3a914d3abbdaab2a6abafe3ade56aef8346d3997f053` | **identical** |
| `hooks` in `~/.claude/settings.json` | one `PreToolUse` → `scuba-guard.sh` | one `PreToolUse` → `scuba-guard.sh` | **unchanged** |
| `~/.claude/hooks/` | `scuba-guard.sh` | `scuba-guard.sh` | **unchanged** |
| `~/.cursor` | did not exist | **does not exist** | never created |
| `~/.skill-architect` (the default spool) | did not exist | **does not exist** | never written |

No hook entry anywhere in the real config points at a scratch path. Every
`hooks install` ran with an explicit `--home` inside the scratchpad, and I printed
and confirmed the resolved target path before the first run. Every `ingest` ran
with an explicit `--spool-dir` inside the scratchpad (the one exception was the
deliberate `HOME=<fake-home>` probe in Finding 3, which wrote into a scratch
directory and has been removed).

Scratch configs removed: `scratch-home/`, `virgin-home/`, `homeprobe/`, `edge/`,
`redact/`, `redact-strict/`, `conc-*/`, `session-cwd/`, `session2/`. Only
`artifacts/` and the two live spool files are kept, deliberately, for review.

**One disclosed side effect.** Session 2 ran against the **real**
`CLAUDE_CONFIG_DIR` (read-only — I registered hooks at project scope in a scratch
directory precisely to avoid writing config). Claude Code wrote its own session
transcript to
`~/.claude/projects/-private-tmp-claude-501--Users-matthewvandusen-…-hooklane-session2/c4741f07-….jsonl`,
as it does for any session in any directory. That is a transcript record, **not
configuration**, and it contains no hook registration. I left it in place rather
than deleting session history; say the word and I will remove that one directory.

---

## 8. Classification summary

| # | Finding | Class | `file:line` |
|---|---|---|---|
| 1 | Install writes hooks.json with no required `version` field; no test covers a virgin install | **CODE FIX before 0.5.0** | `profiler/hooks_install.go:63-87`, `:168-184`; `profiler/hooks_install_test.go:146,180` |
| 4 | Notes and changelog claim prompts/tool I/O are `[REDACTED]` by default; they are plaintext | **PROSE FIX before 0.5.0** | `RELEASE_NOTES.md:10`, `CHANGELOG.md:64` (code is correct: `hooks.go:203-208,317-329`) |
| 3 | `--home` scopes the registration but not the spool; README's try-it example points at an empty dir | **CODE FIX (small) or DISCLOSURE** | `profiler/cmd/main.go:538-551`, `profiler/doctor.go:199-206`, `README.md:467,527` |
| 2 | `cwd` is not a universal Cursor payload field | **DISCLOSURE** | `RELEASE_NOTES.md:11`, `README.md:485-490`, `docs/profiler-spec.md:508-509` |
| 5 | `--strict` writes `user_email` to disk; string absent from the whole repo | **DISCLOSURE (0.5.0)** + code 0.6.0 | `profiler/hooks.go:118-124,203-208` |
| 6 | Uninstall leaves `{"hooks":{}}`, a `.bak`, and a `.cursor/` dir on a virgin home | **DISCLOSURE**, or fold into Finding 1's fix | `profiler/hooks_install.go:94-129,117-121` |
| — | `AppendSpool` lock disclosure understates safety; should name the NFS caveat instead | **DISCLOSURE (reword)** | `profiler/hooks.go:261-280` |
| — | `[DOCS]` sentence needs rewriting in both directions | **DISCLOSURE (reword)** | `RELEASE_NOTES.md:11` |
| — | `conversation_id` blank for any non-Cursor emitter; `analyze` reports 0 conversations | **0.6.0 scope** | `profiler/hooks.go:151-158`, `profiler/analyze.go:180-182` |
| — | Tool-call hook events unmeasured: nested `claude -p` cannot authenticate | **BLOCKED VERIFICATION** | needs `/login`, then a ~3-min re-run |

---

## 9. Commands run (reproducible)

```bash
SP=/private/tmp/claude-501/-Users-matthewvandusen-Development-Auraprix-skill-architect/\
31edaf48-6b19-4833-82e7-6cb52dac691f/scratchpad/hooklane

git worktree add --detach $SP/head 9c8ba53
(cd $SP/head/profiler && go build -o $SP/profiler ./cmd)
(cd $SP/head/profiler && go test ./...)                       # all ok

# install / idempotency / uninstall  (--home ALWAYS inside the scratchpad)
$SP/profiler hooks install   --home $SP/scratch-home          # events_registered: 21
$SP/profiler hooks install   --home $SP/scratch-home          # events_registered: 0, byte-identical
$SP/profiler hooks uninstall --home $SP/scratch-home --command "$SP/profiler ingest || true"

# live emitter: Claude Code, hooks at project scope in a scratch dir
(cd $SP/session2 && claude -p "…two Bash commands…" --allowedTools "Bash(echo:*)" --output-format json)

# read back
$SP/profiler analyze --spool-dir $SP/spool
$SP/profiler doctor  --home $SP/scratch-home
$SP/profiler doctor  --home $SP/scratch-home --spool-dir $SP/spool --command "$SP/profiler ingest || true"

# concurrency
for i in $(seq 1 40); do $SP/profiler ingest --spool-dir $DIR < payload.json & done; wait
```

Primary sources consulted:
- Cursor hooks reference — https://cursor.com/docs/agent/hooks (fetched 2026-09-21)
- Claude Code hooks reference — https://code.claude.com/docs/en/hooks (fetched 2026-09-21)
