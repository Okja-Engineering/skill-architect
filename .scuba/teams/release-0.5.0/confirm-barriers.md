# Confirming pass, lens 1: ESCAPE THE REWRITTEN BARRIERS — PR #22 @ `011defa`

> **Persisted by the chief of staff.** The hunter had no Write tool, said so, verified by
> `ls`, and did not fall back to a heredoc. This is that report.

**VERDICT: NOT CLEAN — 11 findings, 3 of them live escapes, and F3 is a REGRESSION this
PR introduced.**

## Coverage

**Denominator (barriers) established by behaviour, not name — and it really is 3.**
Swept every path-resolution primitive across the tree (`cd -P`, `pwd -P`, `realpath`,
`readlink`, `-ef`, `os.Readlink`, `os.SameFile`,
`filepath.{Clean,Abs,Join,Rel,EvalSymlinks,IsAbs,Dir}`) and every containment idiom
(`strings.HasPrefix`/`TrimPrefix`, `case "$x" in "$y"/*`, `${x#$y}`, `${x%…}`). Walked
**all 22 shell files**, **all 14 Go non-test files**, and **all 30 diff files** one by one.

Ruled out as non-containment: `tests/lib/masked-path.sh:41` and `tests/test_f01.sh:1740`
use `cd -P` as *script locators*; `tests/lib/out-of-scope-check.sh` gates on a
git-relative name regex with no filesystem traversal; `otlp.go:312`,
`claude_code.go:375`, `adapter_contract_test.go:1250` are `HasPrefix` on event/identifier
names. Production write sites (`hooks_install.go`, `hooks.go`, `experiment.go`) have **no**
path gate by design. **B1/B2/B3 are the complete set.**

**The deletion claim is verified.** `path_absolute` / `path_normalized` survive **only in
prose** (`draft-rewrite.sh:271,281`; `test_install.sh:344`) — no definition, no caller, no
reachable copy.

**Spellings driven**, all against a real-write oracle over two identical trees: 49
absolute + 17 relative × 3 cwds through B1 and B3; 53 through B2; then a **second sweep
with entirely fresh spellings** — 14 shell + 18 Go + 18 end-to-end drafter runs (6
protected roots × 3 cases, plus 11 combined). Covered: symlink crossed by `..` both
directions, chains, chain > link limit (kernel = **32**, measured), dangling symlink,
symlink-to-symlink-to-parent, relative from inside/outside, `.`/`..` non-leading, trailing
`..`, exactly-the-root, one-char-longer prefix (`roo`/`rootX`/`root-backup`/`Root-2`),
empty path, `/`, `//`, `///`, leaf-symlink vs mid-path symlink, hardlink, TOCTOU,
case-insensitivity, Unicode NFC/NFD, ~1.1 kB path, newline, tab, `\x01`, `...`, `..x`,
`a.../b..`.

**Dryness:** the second sweep added **0 new classes** — all 5 shell + 3 Go mismatches
collapse into F1. Suites green on both shells before and after (`test_install` 50,
`test_rewrite` 158, `test_harness` 191, `test_skill` 86, `test_f01` 1886, `test_f02` 350,
`test_walk` 20; `go test ./...` ok) — **the escapes below are silent in the shipped
suites.**

## Shared root (F1–F4)

All three barriers decide containment by **comparing the bytes of a string they built from
the caller's spelling**, while the write is performed on the caller's **original** spelling
through the kernel. **A name is not an identity.** On this platform three independent
mechanisms make name ≠ identity: case folding, Unicode normalization, and (shell only) the
command substitution mutating the string. The only spelling-independent handle is
`(st_dev, st_ino)`.

**Invariant that must hold:** for every spelling `s` and root `r`, the verdict equals
whether the object a write on `s` creates-or-truncates is the same filesystem object as
`r`, or is reachable from `r` without leaving it — decided by **identity** for every
component that exists, and by name only for components that do not yet exist.

---

## F1 · REAL · HIGH — case-insensitive volume: every protected root is bypassed by spelling

`draft-rewrite.sh:340` + `:438`; `test_install.sh:397` + `:417`; `homesafe.go:230-256` + `:151`.

Reproduced end-to-end on the shipped drafter:

```
draft-rewrite.sh -t <target> -o "$HOME/.CLAUDE/skills/ESCAPED.md"   -> rc=0
$HOME/.claude/skills/ESCAPED.md                                      (bytes landed)
```

**All six roots escape**: `.CLAUDE .CURSOR .CODEX .DEVIN .CONFIG .AGENTS` (the exact
spelling correctly returns rc=1 in each case). Go: `PathContains(p,
base+"/PROTECTED/skills/x")` = `false` while the write lands inside — same for a link whose
**target** is spelled in the wrong case.

**Mechanism:** bash's **builtin** `pwd -P` returns the caller's spelling; `/bin/pwd -P`
(i.e. `getcwd`) returns the stored name — measured on bash 3.2, bash 5.3, sh and zsh.
`homesafe.resolve` never asks the kernel for the name at all.

**Pre-existing at base `10326b3`** (verified), so inherited, not introduced.

## F2 · REAL · HIGH — Unicode normalization (APFS is normalization-insensitive)

Same lines, same root. HOME spelled NFC (`caf\303\251`), `-o` spelled NFD
(`cafe\314\201`) → rc=0, draft lands in the live `.claude/skills/`. Go barrier: same
divergence. `/bin/pwd -P` canonicalizes to NFC; the builtin does not (byte-diffed).

## F3 · REAL · HIGH · **A REGRESSION THIS PR INTRODUCED** — trailing-newline truncation

`draft-rewrite.sh:350` (`printf '%s\n' "$resolved"`) consumed by `$( )` at `:418` strips
the trailing newline, so the barrier decides on a **different path** from the one
`: > "$output"` writes at `:578` (`$output` is the raw `$output_arg`, set at `:456`).
**The new leaf-append branch at `:344-345` is what makes it reachable.**

```
ln -s "$HOME/.claude/skills/PWNED.md" "$HOME/plain/notes.md"$'\n'
draft-rewrite.sh -t <target> -o "$HOME/plain/notes.md"$'\n'
  -> HEAD rc=0, lands at ~/.claude/skills/PWNED.md
  -> BASE rc=3 (refused)
```

`test_install.sh:407`/`:414-415` has the identical shape but is currently saved by the
newline refusal at `:510-512`.

## F4 · REAL · MEDIUM — the leaf symlink is not followed by either shell walk, so "all three are now the same walk" (`test_install.sh:371`) is false

`draft-rewrite.sh:344-345` and `test_install.sh:401-402` append an existing non-directory
leaf **as named**; `homesafe.go:231` calls `os.Readlink` on every component including the
leaf. Oracle table: `@B/root/lnkfile` and `@B/root/dangling-out` → shell says **inside**,
the write lands **outside**; Go says outside (correct).

Not currently exploitable: B1 is closed by the separate `-L` rule at `:421` (verified — two
leaf-symlink attacks into `.claude` both refused), and B3's consumer's first command is
`mkdir -p`, which fails on such a leaf (verified). **It is a hole standing behind two
accidents** — and `test_install.sh:986`'s oracle **cannot express** the case: `mkdir -p`
fails, so it is discarded as *"the oracle could not create … so this case has no verdict to
compare against"* rather than measured.

## F5 · REAL · MEDIUM — TOCTOU: nothing re-checks between decision and write

Decision at `draft-rewrite.sh:454`, write at `:578`, window ≈ 250 ms (the whole audit run).
Race won deterministically by swapping a path component for a symlink into
`.claude/skills` at t+0.15 s → draft landed inside. Present at base too. **Different fix
direction from F1–F3:** verify the opened handle's `dev/ino` against the decision
(`O_NOFOLLOW` / same-handle write).

## F6 · REAL · LOW — the "oracle, not a table" argument does not hold as claimed

`test_install.sh:1023-1032` (9 templates), `test_rewrite.sh:848-860` (12),
`homesafe_test.go:156-169` (12). The oracle *design* is real and correct, but it is driven
by an **enumerated table**, so it is still bounded by what an author thought of. Proven:
inserting one spelling (`'@B/ROOT/home/inner/dX'`) into the existing table at
`test_install.sh:1036` turns the shipped assertion RED with exactly *"a write on that
spelling lands inside the root, and the resolved verdict says outside"*.

`CHANGELOG.md:273-276` claims the tests are "pinned to the invariant and not to the patch";
**two whole classes (F1/F2, F4) sit outside every table.**

## F7 · REAL · LOW — `path_resolved ""` returns `$PWD` at rc=0 in both shell copies; `homesafe.go:202-204` refuses

Guarded at both shell callers (`draft-rewrite.sh:413`, `test_install.sh:451`/`:505`), so
latent — but a second way the three "identical" walks differ.

## F8 · REAL · LOW — opposite polarity for a degenerate root

`test_install.sh:417` (`"$root"/?*` becomes `//?*`) **accepts everything** when the root
resolves to `/`; `draft-rewrite.sh:366-369` (`parent=""`, pattern `/*`) **refuses
everything**. Fail-open vs fail-closed in two copies claimed identical. Neither root is
ever `/` today.

## F9 · REAL · LOW (factual) — `maxSymlinkHops = 40`

`homesafe.go:162`, documented at `:158-161` as bounding "the way the kernel does".
Measured: **macOS refuses at 32.** So `resolve` answers for chains of 32–40 the kernel
rejects. Fail-safe in effect; the comment is wrong.

## F10 · REAL · LOW (consistency) — `filepath.Dir` at `homesafe.go:226` is a surviving textual folder

`Dir` calls `Clean`, in the file whose header (`:169-177`) and sibling
(`draft-rewrite.sh:281-282`) say the textual folders were **deleted** rather than left
present for exactly this reason. Attacked with `...`, `..x`, `a.../b..`, `///`, `././.`,
trailing `../../`, 120-deep — **15/15 clean**, so not exploitable. The documented setup for
a regression, not a live bug.

## F11 · SUSPECTED · LOW — hardlinks defeat path containment and the oracle

A hardlink inside the protected root to a file outside means a write judged "outside"
mutates content inside; no path-resolution barrier can see it. **Observed live:** a
hardlink fixture made the hunter's own real-write oracle mis-report two cases, so a suite
fixture containing one would silently corrupt the oracle's verdict. Probably a stated
non-goal rather than a fix.

---

## Fix direction — ADVISORY, re-derive at the root

**F1–F3 are one repair, not three: resolve to *identity* rather than to a name.**

- **Shell:** `path_resolved` can get the kernel's stored name from the external `pwd`
  (`command pwd -P` / `/bin/pwd -P`) instead of the builtin — that alone closes F1 and F2
  — **but the newline truncation (F3) survives any change that still returns the value
  through `$( )`.** So the containment comparison must stop being a string compare of a
  captured value: compare `dev:ino` of the deepest existing ancestor via
  `stat -f '%d:%i'`, and compare names only for the not-yet-existing tail.
- **Go:** compare `os.SameFile` on the existing prefix instead of `filepath.Rel` on names.

**Pin the test to the invariant stated above — not to these spellings — or F6 recurs as
the next round.**

**Reproducers and drivers:** `scratchpad/atk/` (`drive.sh`, `drive-rel.sh`, `tree.sh`,
`cases-abs.txt`, `cases-rel.txt`, `cases-sweep2.txt`, `inst_barrier.sh`,
`draft_barrier.sh`, `base/`). Worktree `scratchpad/lens1-barriers`, clean at `011defa`.
Primary tree untouched.
