# Runbook — cutting a release

How `skill-architect` gets from trunk to the people who install it, and the one
ordering rule that must not be broken.

## The problem this solves

`.claude-plugin/marketplace.json` carries `"source": "./"`, and every install
route this repository documents clones the repository **without naming a ref**.
So an installer gets whatever the repository's default branch is at that moment.
While the default branch was `main`, that meant **merging to `main` published** —
measured, not assumed: every tag in this repository's history sits on exactly the
commit `main` advanced to, and `v0.5.0 == origin/main`.

There was no ref between trunk and users, which is why a release had to be
batched: any increment toward it shipped the moment it merged.

The fix operates below the plugin layer, because one of the four agents has no
ref mechanism to operate at. **The default branch becomes `release`.** `main` is
trunk and moves freely; `release` moves only when a release is cut. Every
harness that clones without a ref then gets the release by construction, and the
user-facing install command does not change.

### Why not pin the manifest instead

Because it was tried and measured. Replacing `"source": "./"` with
`{ "source": "github", "repo": …, "ref": "v0.5.0" }` sends the documented
local-checkout route to GitHub instead of the working copy — **silently**:
`marketplace add` succeeds, `install` prints success, `plugin list` reports a
plausible version, and the installed tree is byte-identical to
`git archive v0.5.0`. A developer testing their own edits is running someone
else's commit and nothing says so. Keeping both forms is not expressible either:
two entries with one plugin name fail `claude plugin validate --strict`, and a
second marketplace pointing back at the repository root is rejected outright
(`Path contains ".."`).

### Why the default branch reaches all four agents

| Agent | How it installs | Resolves the default branch? |
|---|---|---|
| Claude Code | `claude plugin marketplace add Okja-Engineering/skill-architect` | **Yes — proved.** Its git invocation adds `--branch` only when a ref is supplied; with none, `git clone` resolves remote `HEAD`. Run against a repository whose default branch was `release` while a `main` branch also existed carrying a different version, it installed the `release` one. |
| Codex | `codex plugin marketplace add Okja-Engineering/skill-architect` | **Yes — proved**, same experiment, `codex-cli 0.156.1`. It reads **this repository's `.claude-plugin/marketplace.json`**; `codex plugin list` prints that path. |
| Cursor | copy or symlink the repository root into the plugins folder | **Not applicable — there is no ref to resolve.** A user clones and copies; a plain `git clone` lands on the default branch. Cursor never parses a ref, so there is nothing that could be pinned to `main`. |
| Devin | `devin plugins install Okja-Engineering/skill-architect` | **Documented, not executed.** Devin's plugin reference states: *"`sha` and `ref` … are mutually exclusive: a `sha` is a pin, a `ref` floats. Without either, the source tracks the repository's default branch."* Not run here, deliberately — that form syncs to Devin Cloud and would rewrite a live account's plugin list. The fetch happens server-side, so it is not provable locally at all. |

## Cutting a release

Versions live in five files and they must agree: `.claude-plugin/plugin.json`,
`.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`,
`.devin-plugin/plugin.json`, and the `version` on the entry in
`.claude-plugin/marketplace.json`. `tests/test_release.sh` refuses a tree where
they do not.

1. **On `main`, land the version bump** as an ordinary PR: all five files, plus
   `CHANGELOG.md` and `RELEASE_NOTES.md`. Trunk now declares a version that has
   not been released, which is allowed — `main` is trunk.

2. **Tag the commit on `main`.** The tag comes *before* the branch moves.

   ```bash
   git tag -a v0.6.0 -m "v0.6.0" <the commit on main>
   git push origin v0.6.0
   ```

3. **Open a PR from `main` into `release`** and merge it. CI runs the full suite
   on that PR, and `tests/test_release.sh` additionally checks the half it
   cannot check anywhere else: that the version the merge result declares is one
   a tag names. Because step 2 already happened, it passes. Had you merged
   first, it would fail — which is the point.

4. **Verify what a user gets**, against a disposable config directory so the
   check cannot touch your own:

   ```bash
   CLAUDE_CONFIG_DIR="$(mktemp -d)" sh -c '
     claude plugin marketplace add Okja-Engineering/skill-architect &&
     claude plugin install skill-architect@skill-architect &&
     claude plugin list --json'
   ```

## The ordering rule

**A ref must never lag its tag.** Publish the tag before, or in the same breath
as, the thing that points at it.

This is measured, and the failure is total rather than partial. A marketplace
entry pinned to a ref that does not exist yet fails at install for **everyone**,
including users who never pinned anything:

```
✘ Failed to install plugin "skill-architect@skill-architect": Failed to clone
  repository: … fatal: Remote branch v0.6.0 not found in upstream origin
```

Exit 1, `claude plugin list --json` returns `[]`, nothing is cached. The good
news inside the bad news is that there is **no silent fallback to the default
branch** — a dangling ref fails loudly rather than quietly serving trunk. But
between a pin landing and its tag being pushed, the plugin is uninstallable.
`marketplace add` still succeeds during that window, because the catalogue is
valid; only `install` fails. So the window is invisible to the person who opened
it and total for everyone else.

The same rule is why step 2 precedes step 3 above, and `tests/test_release.sh`
is what enforces it: a `release` branch whose declared version has no tag on
that commit fails CI.

## What is *not* checked automatically

- **That `release` is the default branch.** That is a repository setting, not a
  file, so nothing in the tree can assert it. See below.
- **Devin's and Cursor's install routes**, for the reasons in the table above.
- **Tag naming.** `claude plugin tag` creates tags named
  `skill-architect--v0.6.0`. This repository's history uses `vX.Y.Z`, and
  `tests/test_release.sh` looks for `vX.Y.Z`. Tag by hand, as in step 2.

## Switching the default branch (maintainer only)

This is an outward-facing repository setting and is not something a contributor
or CI can change. Everything else is prepared so that it is one switch.

**Before it can be flipped, two refs must move.** GitHub refuses to create
`refs/heads/release` while `refs/heads/release/0.4.3` and
`refs/heads/release/0.5.0` exist — git's ref namespace is a path, so a ref
cannot be both a file and a directory (`! [remote rejected] release -> release
(directory file conflict)`). Neither of those two branches is an ancestor of
`main` and neither is contained in any tag, so they hold history that exists
nowhere else; archive rather than delete them:

```bash
git push origin origin/release/0.4.3:refs/heads/archive/release-0.4.3
git push origin origin/release/0.5.0:refs/heads/archive/release-0.5.0
git push origin :refs/heads/release/0.4.3
git push origin :refs/heads/release/0.5.0
git push origin 2e6e25ec8b6b65cd417fa000d5b21d58bbba925c:refs/heads/release
```

The last line creates `release` at `v0.5.0`, which is what is released today —
deliberately not at trunk's head.

**Then the switch itself:** Settings → General → Default branch → the switch
icon → choose `release` → Update.

**What changes the instant it is clicked:**

- **Every unpinned install route starts serving `release` instead of `main`.**
  Anyone who installs from that moment on gets `v0.5.0`. Existing users are not
  stranded: a `claude plugin marketplace update` re-clones and picks up the new
  default branch, after which `claude plugin update` moves the installed plugin
  — measured, including a move to a *lower* version number, which the CLI
  accepts without complaint. `codex plugin marketplace upgrade` does the same.
- **The branch ruleset follows the default branch, and so leaves `main`.** The
  ruleset `deafult-branch-protection` targets `~DEFAULT_BRANCH`, not
  `refs/heads/main`. The instant `release` becomes the default, its rules —
  pull request required, linear history, the `test` status check, no deletion,
  no force-push, Copilot review — apply to `release` and **stop applying to
  `main`. Trunk becomes an unprotected branch.** That is the opposite of what is
  wanted and is not a side effect to discover later: a second ruleset naming
  `refs/heads/main` explicitly should be added in the same sitting, or the
  existing one changed to include both.
- **New PRs default to basing on `release`.** Contributors opening a PR will
  target the release branch unless they change the base, and slices belong on
  `main`. Worth a line in `AGENTS.md` and in the PR template if one is added.
- **Fresh `git clone` of this repository lands on `release`**, including for
  anyone developing on it. `git clone -b main` or a `git switch main` after
  cloning is the developer's first step from then on.
- **Nothing about the user-facing install command changes.** That is the whole
  point of doing it at this layer.
