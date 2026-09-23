package skillgate

import (
	"context"
	"encoding/json"
	"time"
)

// External scanners — deciding what a non-zero exit *means*.
//
// A scanner exits non-zero for two unrelated reasons, and the gate read them
// as one:
//
//   - **"I ran, and the subject has findings."** `skill-validator` exits 1
//     on any conformance error and prints its whole report to stdout anyway.
//   - **"I did not run usefully."** Binary missing, bad arguments, crash,
//     killed.
//
// Deciding from the exit status made the first case invisible. `SV-CONFORM`
// was gated on `rep.Errors > 0`, and the only path that could set that field
// was the path that returned a skip and threw the report away — so a rule
// this gate publishes could not fire at all. Measured across 63 bundles in
// four estates (S14): **zero SV-CONFORM findings ever**, while seven bundles
// had errors in a report the gate had already read and dropped. It reached
// 10 of 29 installed skills and 29 of 31 third-party bundles, and none of
// this repository's own three, which exit 0 — invisible to self-dogfooding
// by construction.
//
// The same discarded report carries the **token counts**, so every bundle
// with a conformance error also lost its measured budget: `icm-token-budget`
// gained a skip and `report.tokens.basis` fell back from `measured`. One
// conflation, three published things wrong, and all of them wrong in the
// direction of saying less about the bundles that deserved more scrutiny.
//
// So the **output** decides and the exit status is only ever evidence for
// the skip's wording:
//
//	exit   report parses   outcome
//	----   -------------   -----------------------------------------
//	0      yes             result
//	≠0     yes             result  ← the case that was unreachable
//	0      no              skip, named
//	≠0     no              skip, named, quoting the exit error
//
// with one rule that outranks all four: **a deadline is always a skip.** A
// truncated report is not a report however well the bytes that arrived
// happen to parse, and "did not finish" must never read as "clean". That is
// the direction this whole change must not get wrong — a gate that reports
// nothing because its scanner died is worse than one that misses a finding,
// because it looks the same as success.
//
// Note on the mechanics, because the obvious patch is unnecessary: `Output()`
// does **not** discard stdout on a non-zero exit — it returns the collected
// bytes in its first result and reports the exit in `err`. (`exec.ExitError`
// has no `Stdout` field at all; only `Stderr`.) Nothing was being thrown away
// by the standard library. The report was being thrown away by *this package*,
// which branched on `err` before looking at what it already held.

// externalOutcome decides whether a scanner produced a usable report.
//
// It unmarshals into `into` on success and returns nil; otherwise it returns
// the named skip that belongs in checks_skipped. `verb` is the scanner's own
// word for what it does ("check", "scan"), so the reason reads in the
// tool's language rather than the gate's.
func externalOutcome(ctx context.Context, check, verb string, timeout time.Duration, runErr error, report []byte, into any) *SkippedCheck {
	// First, and regardless of what arrived: a deadline is not a result.
	if ctx.Err() == context.DeadlineExceeded {
		return &SkippedCheck{
			Check:  check,
			Reason: verb + " timed out after " + timeout.String(),
		}
	}
	if err := json.Unmarshal(report, into); err != nil {
		reason := verb + " produced no readable report: " + err.Error()
		if runErr != nil {
			// The exit status is not the decision, but it is the best
			// evidence for why there was nothing to read.
			reason = verb + " failed (" + runErr.Error() + ") and produced no readable report: " + err.Error()
		}
		return &SkippedCheck{Check: check, Reason: reason}
	}
	return nil
}
