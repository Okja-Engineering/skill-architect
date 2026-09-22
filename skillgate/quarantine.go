package skillgate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// G0 — quarantine. The audited unit is the *package*, not the skill: a
// local skill dir or a plugin/package root (a manifest may ship `skills`
// beside executable `extensions`, and a manifest short-circuits the
// convention-directory walk). A remote target is cloned into a fresh temp
// directory before the ledger sees it. The clone is the quarantine:
// nothing outside it is read, and the target is marked untrusted so an
// incomplete ledger REJECTs instead of capping at CAUTION (F13).
// Provenance records the URL, the resolved commit SHA, the quarantine dir,
// and the manifest digest — the pin is over the whole package.
//
// Only git URLs are fetched — `npm:`/`git:` package specs arrive with the
// D9 packaging decision; a spec with no `@ref` would be unpinned. The gate
// never executes bundle code; clone hooks are disabled via
// -c core.hooksPath=/dev/null so a malicious .git/hooks or
// gitconfig-in-repo cannot run during fetch.

const fetchTimeout = 60 * time.Second

// IsRemoteTarget reports whether the arg is a fetchable URL rather than a
// local path. An existing local path wins over the .git heuristic.
func IsRemoteTarget(arg string) bool {
	if st, err := os.Stat(arg); err == nil && st.IsDir() {
		return false
	}
	return strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") ||
		strings.HasPrefix(arg, "git@") || strings.HasPrefix(arg, "ssh://") ||
		strings.HasSuffix(arg, ".git")
}

// FetchTarget clones url into a quarantine dir, resolves the commit, and
// returns the local dir plus its provenance. The caller decides whether to
// clean up (CLI keeps the dir so the report can point at it).
func FetchTarget(url string) (dir string, prov *Provenance, err error) {
	q, err := os.MkdirTemp("", "skillgate-quarantine-*")
	if err != nil {
		return "", nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.hooksPath=/dev/null",
		"clone", "--depth", "1", "--quiet", url, q)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(q)
		return "", nil, fmt.Errorf("fetch %s: %v (%s)", url, err, strings.TrimSpace(string(out)))
	}

	commit, _ := exec.Command("git", "-C", q, "rev-parse", "HEAD").Output()
	return q, &Provenance{
		URL:        url,
		Commit:     strings.TrimSpace(string(commit)),
		Quarantine: q,
	}, nil
}

// GateDir is Gate plus provenance: the path may be a local dir or a remote
// URL (fetched into quarantine first).
func (e *Engine) GateDir(target string, opts Options) (*Report, error) {
	if IsRemoteTarget(target) {
		dir, prov, err := FetchTarget(target)
		if err != nil {
			return nil, err
		}
		rep, err := e.Gate(dir, opts)
		if err != nil {
			return nil, err
		}
		rep.Target = target
		if rep.Provenance == nil {
			rep.Provenance = prov
		} else {
			rep.Provenance.URL = prov.URL
			rep.Provenance.Commit = prov.Commit
			rep.Provenance.Quarantine = prov.Quarantine
		}
		// Recompute the verdict: a fetched bundle is untrusted, so an
		// incomplete ledger REJECTs rather than capping at CAUTION.
		rep.Verdict = computeVerdict(rep.Findings, rep.Coverage, rep.ChecksSkipped, true)
		return rep, nil
	}
	return e.Gate(target, opts)
}

// QuarantinePath returns the quarantine root for cleanup.
func QuarantinePath(rep *Report) string {
	if rep != nil && rep.Provenance != nil {
		return rep.Provenance.Quarantine
	}
	return ""
}
