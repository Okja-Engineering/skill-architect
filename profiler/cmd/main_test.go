package main

import (
	"strings"
	"testing"
)

// The invariant: a flag the selected adapter cannot honour fails loudly. It is
// never accepted and silently dropped, which turns a supplied export into a
// profile that looks like the user configured no telemetry at all.

func TestCaptureFlagError_RefusesAnExportFileNoAdapterReads(t *testing.T) {
	err := captureFlagError("claude_code", "/var/tmp/session.atif.json")
	if err == nil {
		t.Fatal("--export-file accepted by an adapter that cannot read it: the path would be silently ignored")
	}
	if !strings.Contains(err.Error(), "--otel-file") {
		t.Errorf("error = %q, want it to name the flag that does work", err)
	}
	if !strings.Contains(err.Error(), "claude_code") {
		t.Errorf("error = %q, want it to name the adapter that cannot honour the flag", err)
	}
}

func TestCaptureFlagError_AllowsTheOtelOnlyInvocation(t *testing.T) {
	if err := captureFlagError("claude_code", ""); err != nil {
		t.Fatalf("capture without --export-file rejected: %v", err)
	}
}
