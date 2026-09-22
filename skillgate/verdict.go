package skillgate

// computeVerdict applies the F12/F13 contract:
//   - REJECT: any unsuppressed Blocker/High finding, or (untrusted input only)
//     an incomplete ledger.
//   - CAUTION: any Medium/Low finding, any skipped check, or an incomplete
//     ledger on a trusted (local) input.
//   - APPROVE: no findings, complete ledger, no skipped checks.
//
// It never emits "safe" or "clean" (F12), and an incomplete ledger or a
// skipped check can never produce more than CAUTION (F13).
func computeVerdict(findings []Finding, cov Coverage, skipped []SkippedCheck, untrusted bool) string {
	for _, f := range findings {
		if f.Blocks() {
			return VerdictReject
		}
	}
	if untrusted && !cov.Complete {
		return VerdictReject
	}
	if !cov.Complete || len(skipped) > 0 {
		return VerdictCaution
	}
	for _, f := range findings {
		if !f.Suppressed {
			return VerdictCaution
		}
	}
	return VerdictApprove
}
