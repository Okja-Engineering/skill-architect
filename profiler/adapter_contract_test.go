package profiler

// The adapter contract, enforced over every adapter rather than over one.
//
// Three rules decided this repository's last two releases, and all three lived
// in prose and in a single adapter's implementation:
//
//  1. A capture reads only the records carrying the session it was asked for,
//     so an empty session id is no assertion and is refused.
//  2. Probe and capture answer the same question (AC9). Every signal the
//     capability report advertises from a real source is present in the
//     capture; every signal it marks "none" is not.
//  3. A token count that was not read has no key. "The export said nothing
//     about this" and "the export said zero" are different answers.
//
// Prose cannot refuse an adapter, and a rule implemented once is a rule the
// second adapter reimplements from memory. Three unlanded adapters were written
// against these rules and all three broke (2); one of them advertised
// session_data for tokens, tool calls and timing on every input and returned
// "unknown" on every path that could reach a capture. Nothing mechanical stood
// between that adapter and a profile claiming it had measured something.
//
// So the rules are a table over the registry below, and registering an adapter
// means supplying an export it reads. That is the part that refuses: an adapter
// with no input for which capture delivers what probe advertised cannot be
// registered without turning this file red.
//
// This is the adapter denominator. The export-shape denominator — one adapter
// over every fixture under testdata/otlp — is captureCases in profiler_test.go.
// They are not the same coverage and neither substitutes for the other.

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// adapterEntry is one adapter, together with the inputs the contract is
// asserted over. The inputs are part of the registration rather than looked up
// per harness elsewhere, because "this adapter has an export for which its own
// claims hold" is the obligation, and an entry that cannot name one is the
// adapter this file exists to refuse.
type adapterEntry struct {
	// name is the harness identifier the CLI selects by, and Name() must agree.
	name string

	// newAdapter builds the adapter pointed at one export. The signature is the
	// whole harness-agnostic part: whatever kind of file a harness exports, the
	// contract is asserted by pointing the adapter at one.
	newAdapter func(export string) ProfilerAdapter

	// export is an export this adapter reads that carries session, used for the
	// probe/capture agreement. Its signals are not written down here: the
	// capability report is the denominator, so the table cannot drift from what
	// the adapter says about its own input.
	export string

	// session is the identity carried by both exports below. A capture of a
	// session an export does not contain reads nothing, which would make the
	// agreement vacuously true in the "none" direction.
	session string

	// cacheOnlyExport is an export carrying cache token counts and no input or
	// output count. It may be empty only for an adapter that advertises no
	// token signal at all, and that claim is checked rather than taken.
	cacheOnlyExport string
}

// adapterRegistry is every adapter this package ships.
//
// It is not a list anyone is asked to remember: TestEveryAdapterIsInTheRegistry
// reads the package's own source for types implementing ProfilerAdapter and
// requires each to be here. An adapter the CLI can dispatch is a type in this
// package, so an adapter that escapes this table cannot be reached at all.
var adapterRegistry = []adapterEntry{
	{
		name:            "claude_code",
		newAdapter:      func(export string) ProfilerAdapter { return ClaudeCodeAdapter{OtelExportFile: export} },
		export:          fixture("full_export.ndjson"),
		session:         fixtureSession,
		cacheOnlyExport: fixture("cache_only.json"),
	},
}

// contractOpts are the capture options every check below uses. Neither field
// participates in any of the three rules; they are recorded verbatim in the
// profile and are supplied so a capture is a well-formed one.
var contractOpts = CaptureOpts{SnapshotHash: "abc123", SkillDir: "/skills/my-skill"}

// adapterContractViolations returns one message per rule the adapter breaks,
// and nothing when it keeps all of them.
//
// It returns violations rather than calling t.Errorf, and that is what makes the
// table provable: a check that reports through the testing package can only be
// shown to work by breaking the repository, so nobody breaks it and the check
// is never shown to work. As a function of an adapter it can be run over
// adapters built to break each rule — see TestTheContractTableCanFail — so the
// table's ability to fail is itself under test, on every run.
func adapterContractViolations(e adapterEntry) []string {
	var v []string
	report := func(format string, args ...any) {
		v = append(v, fmt.Sprintf(format, args...))
	}

	a := e.newAdapter(e.export)

	// The registry key and the adapter's own identity are the same name. They
	// have to be: the refusal below quotes Name(), and the CLI selects by the
	// key, so a disagreement means the harness a caller named and the harness
	// that answered are two different things.
	if got := a.Name(); got != e.name {
		report("registered as %q, Name() returns %q", e.name, got)
	}

	v = append(v, sessionIDRefusalViolations(a)...)

	capabilities, agreement := probeCaptureAgreementViolations(a, e.session)
	v = append(v, agreement...)

	v = append(v, unreadCountViolations(e, capabilities)...)

	return v
}

// --- Rule 1: a capture with no session id is refused --------------------------

// sessionIDRefusalViolations checks that the adapter, not only the CLI, refuses
// an empty session id. A library caller never passes through the flag layer, and
// a profile stamped with an empty id could only report every session the export
// happens to carry — which is the scoping this rule exists to make load-bearing.
func sessionIDRefusalViolations(a ProfilerAdapter) []string {
	var v []string
	_, err := a.Capture("", contractOpts)
	switch {
	case err == nil:
		v = append(v, "Capture with an empty session id returned no error: an empty id is no "+
			"assertion, and a profile stamped with it names a session nobody asked for")
	case err.Error() != SessionIDRequiredError(a.Name()).Error():
		v = append(v, fmt.Sprintf("Capture with an empty session id returned %q, want the "+
			"shared refusal %q", err, SessionIDRequiredError(a.Name())))
	}
	return v
}

// --- Rule 2: probe and capture answer the same question (AC9) -----------------

// probeCaptureAgreementViolations walks the capability report and requires the
// capture of a session the export carries to match it, in both directions. It
// returns the report too, because rule 3 needs to know whether this adapter
// claims a token signal at all and re-probing could read a different file.
//
// The capability report is the denominator, so no signal is assumed and a
// report that enumerates fewer signals than the profile carries is itself a
// violation — otherwise an adapter could satisfy this by advertising nothing.
func probeCaptureAgreementViolations(a ProfilerAdapter, session string) (map[MetricName]MetricSource, []string) {
	var v []string
	report := a.Probe()

	profile, err := a.Capture(session, contractOpts)
	if err != nil {
		return report.Capabilities, append(v, fmt.Sprintf("Capture of the registered session failed: %v", err))
	}

	states := profile.SignalStates()
	if len(report.Capabilities) != len(states) {
		v = append(v, fmt.Sprintf("the capability report covers %d signals and the profile carries %d: "+
			"a signal nobody advertised is a signal this contract never reads",
			len(report.Capabilities), len(states)))
	}

	signals := capturedSignals(profile)
	for _, metric := range sortedMetrics(report.Capabilities) {
		advertised := report.Capabilities[metric]
		got, ok := signals[metric]
		if !ok {
			v = append(v, fmt.Sprintf("%s: advertised by probe, absent from the profile", metric))
			continue
		}

		if advertised == SourceNone {
			// The direction that catches an adapter reporting a value it never
			// said it could read.
			if got.raw.State == MetricPresent {
				v = append(v, fmt.Sprintf("%s: probe advertised %q and capture returned %q — "+
					"a value the adapter never claimed a source for", metric, advertised, MetricPresent))
			}
			continue
		}

		// The direction that catches an adapter advertising a capability its
		// capture cannot deliver.
		if got.raw.State != MetricPresent {
			v = append(v, fmt.Sprintf("%s: probe advertised %q, capture state is %q (reason %q) — "+
				"the capability is a claim about what was read", metric, advertised, got.raw.State, got.raw.Reason))
			continue
		}
		if got.raw.Source != string(advertised) {
			v = append(v, fmt.Sprintf("%s: probe advertised %q, capture names source %q",
				metric, advertised, got.raw.Source))
		}
		if !got.hasValue {
			v = append(v, fmt.Sprintf("%s: state %q carrying no value — present means a value was read",
				metric, got.raw.State))
		}
	}
	return report.Capabilities, v
}

// sortedMetrics orders a capability report's signals, so a run that finds
// several violations reports them in the same order every time.
func sortedMetrics(caps map[MetricName]MetricSource) []MetricName {
	names := make([]MetricName, 0, len(caps))
	for m := range caps {
		names = append(names, m)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// --- Rule 3: a count that was not read has no key -----------------------------

// unreadCountViolations captures an export carrying only cache counts and
// requires the profile to say nothing about input and output.
//
// The failure this refuses is arithmetic on a value field: `tc.Input += n`
// against a count nobody exported leaves a zero that reads as a measurement,
// and a cache-only export then reports a session that used no input and no
// output tokens. Two releases were spent closing that class in one adapter;
// this is where the next adapter meets it.
//
// An entry with no cache-only export is allowed only when the adapter
// advertises no token signal, and the allowance is checked against the probe
// rather than declared, so it cannot be used to opt out.
func unreadCountViolations(e adapterEntry, capabilities map[MetricName]MetricSource) []string {
	var v []string

	if e.cacheOnlyExport == "" {
		if capabilities[MetricTokens] != SourceNone {
			v = append(v, fmt.Sprintf("advertises tokens as %q but registers no cache-only export: "+
				"an adapter that reports counts has to show that an unread count has no key",
				capabilities[MetricTokens]))
		}
		return v
	}

	profile, err := e.newAdapter(e.cacheOnlyExport).Capture(e.session, contractOpts)
	if err != nil {
		return append(v, fmt.Sprintf("Capture of the cache-only export failed: %v", err))
	}
	if profile.Tokens.State != MetricPresent || profile.Tokens.Value == nil {
		return append(v, fmt.Sprintf("the registered cache-only export yielded tokens %q (reason %q): "+
			"the export has to carry cache counts for this rule to be asserted over anything",
			profile.Tokens.State, profile.Tokens.Reason))
	}

	keys, err := tokenCountKeys(*profile.Tokens.Value)
	if err != nil {
		return append(v, fmt.Sprintf("token counts do not marshal: %v", err))
	}
	for _, unread := range []string{"input", "output"} {
		if raw, ok := keys[unread]; ok {
			v = append(v, fmt.Sprintf("the cache-only export carries no %s count and the profile "+
				"reports %q: %s — a measurement nobody made", unread, unread, raw))
		}
	}
	if _, read := keys["cache_read"]; !read {
		if _, created := keys["cache_creation"]; !created {
			v = append(v, "the registered cache-only export yielded no cache count at all, so "+
				"nothing distinguishes it from an export this adapter cannot read")
		}
	}
	return v
}

// tokenCountKeys is the profile's own view of a token result: the keys a reader
// of the JSON finds. Asked of the serialized form rather than of the struct,
// because absence is expressed by omitempty and a nil pointer and a zero one
// are the same field until they are marshalled.
func tokenCountKeys(tc TokenCounts) (map[string]string, error) {
	b, err := json.Marshal(tc)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	keys := make(map[string]string, len(raw))
	for k, val := range raw {
		keys[k] = string(val)
	}
	return keys, nil
}

// --- The table --------------------------------------------------------------

// TestAdapterContract is the table. Every adapter this package ships, against
// all three rules.
func TestAdapterContract(t *testing.T) {
	if len(adapterRegistry) == 0 {
		t.Fatal("the registry is empty, so every rule below holds over nothing")
	}
	for _, e := range adapterRegistry {
		t.Run(e.name, func(t *testing.T) {
			for _, violation := range adapterContractViolations(e) {
				t.Error(violation)
			}
		})
	}
}

// TestTokenCountsAreUnreadableAsZero is the structural half of rule 3, and it is
// one assertion for the package rather than one per adapter: no adapter can
// keep the rule if the type cannot express "not read". A value field turns an
// absent count into a zero before any adapter gets a say.
func TestTokenCountsAreUnreadableAsZero(t *testing.T) {
	tc := reflect.TypeOf(TokenCounts{})
	if tc.NumField() == 0 {
		t.Fatal("TokenCounts has no fields, so this check reads nothing")
	}
	for i := range tc.NumField() {
		f := tc.Field(i)
		if f.Type.Kind() != reflect.Pointer || f.Type.Elem().Kind() != reflect.Int {
			t.Errorf("TokenCounts.%s is %s, want *int: a value field cannot tell a count "+
				"the export never carried from one it carried as zero", f.Name, f.Type)
		}
		if !strings.Contains(f.Tag.Get("json"), ",omitempty") {
			t.Errorf("TokenCounts.%s has json tag %q, want omitempty: without it an unread "+
				"count is serialized as null rather than left out", f.Name, f.Tag.Get("json"))
		}
	}
}

// --- The registry is the whole package ----------------------------------------

// TestEveryAdapterIsInTheRegistry makes the package source the denominator. A
// table that only covers the adapters somebody remembered to add is the same
// defect as the prose it replaces, one level up.
//
// The scan is over source rather than over runtime types because Go cannot
// enumerate a package's types at runtime, and over non-test files because the
// stubs below deliberately break the contract and must not be required to keep
// it.
func TestEveryAdapterIsInTheRegistry(t *testing.T) {
	sources, err := packageSources(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("no package sources found, so this check reads nothing")
	}

	defined, err := adapterTypesIn(sources)
	if err != nil {
		t.Fatal(err)
	}
	if len(defined) == 0 {
		t.Fatal("the scan found no adapter type in the package, and one is shipped: " +
			"the scan is broken, not the package")
	}

	registered := make(map[string]bool, len(adapterRegistry))
	for _, e := range adapterRegistry {
		registered[concreteTypeName(e.newAdapter(""))] = true
	}

	for _, name := range defined {
		if !registered[name] {
			t.Errorf("%s implements ProfilerAdapter and is not in adapterRegistry: "+
				"its probe/capture agreement, its session-id refusal and its token "+
				"counts are asserted nowhere", name)
		}
	}
	if len(registered) != len(defined) {
		t.Errorf("%d types registered, %d adapters defined in the package: an entry names "+
			"something the package does not define", len(registered), len(defined))
	}
}

// TestTheAdapterScanFindsAnAdapter is the control for the scan above, which can
// only ever report "nothing unregistered" and would report exactly that if it
// stopped recognising adapters. It is handed a source file it must recognise
// and one it must not.
func TestTheAdapterScanFindsAnAdapter(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	full := write("full.go", `package p
type FullAdapter struct{}
func (a FullAdapter) Name() string { return "full" }
func (a FullAdapter) Probe() CapabilityReport { return CapabilityReport{} }
func (a *FullAdapter) Capture(s string, o CaptureOpts) (Profile, error) { return Profile{}, nil }
`)
	partial := write("partial.go", `package p
type PartialAdapter struct{}
func (a PartialAdapter) Name() string { return "partial" }
func (a PartialAdapter) Probe() CapabilityReport { return CapabilityReport{} }
`)

	found, err := adapterTypesIn([]string{full, partial})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"FullAdapter"}
	if !reflect.DeepEqual(found, want) {
		t.Errorf("scan found %v, want %v: a type is an adapter when it has all three "+
			"methods, on either receiver form, and is not one when it has two", found, want)
	}
}

// packageSources lists the package's own Go files, excluding tests.
func packageSources(dir string) ([]string, error) {
	all, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	var sources []string
	for _, path := range all {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		sources = append(sources, path)
	}
	return sources, nil
}

// adapterTypesIn returns the names of the types in these files that implement
// ProfilerAdapter, sorted.
//
// A type is an adapter when it carries all three of the interface's methods, on
// either receiver form. Matching on the method set rather than on a naming
// convention is deliberate: a type called something other than "…Adapter" is
// still dispatchable, and it is the one nobody would think to register.
func adapterTypesIn(files []string) ([]string, error) {
	methods := map[string]map[string]bool{}
	fset := token.NewFileSet()
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			recv := receiverTypeName(fn.Recv.List[0].Type)
			if recv == "" {
				continue
			}
			if methods[recv] == nil {
				methods[recv] = map[string]bool{}
			}
			methods[recv][fn.Name.Name] = true
		}
	}

	var adapters []string
	for name, have := range methods {
		if have["Name"] && have["Probe"] && have["Capture"] {
			adapters = append(adapters, name)
		}
	}
	sort.Strings(adapters)
	return adapters, nil
}

// receiverTypeName unwraps a pointer receiver and a generic one to the type's
// own name, and returns "" for anything that is not a named type.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// concreteTypeName is the name of the type behind a ProfilerAdapter, whether it
// was registered by value or by pointer.
func concreteTypeName(a ProfilerAdapter) string {
	t := reflect.TypeOf(a)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil {
		return ""
	}
	return t.Name()
}

// --- Proof that the table can fail --------------------------------------------
//
// A green run of a contract table proves nothing on its own: a table that
// checks the wrong field, reads an empty denominator, or compares a value to
// itself is green for every input, and the adapters it was written to refuse
// pass it. So the table is run here over adapters built to break exactly one
// rule each, and the violation it must report is named.
//
// The stubs are types in this file, which is why the registry scan above skips
// test sources. They are not registered: a permanently red suite is a suite
// that gets deleted.

const (
	stubPrimaryExport   = "stub://primary"
	stubCacheOnlyExport = "stub://cache-only"
)

// stubAdapter is a synthetic adapter whose every answer is dictated by the
// fields below, so a liar differs from the honest one by one field and the
// violation can be attributed to it.
type stubAdapter struct {
	harness string
	// acceptsNoSession makes Capture("") return a profile instead of the refusal.
	acceptsNoSession bool
	capabilities     map[MetricName]MetricSource
	tokens           TokenResult
	cacheTokens      TokenResult
	// pointedAt is the export this instance was built for, so one stub can
	// answer differently for the primary and the cache-only export the way a
	// real adapter reading two files does.
	pointedAt string
}

func (s stubAdapter) Name() string { return s.harness }

func (s stubAdapter) Probe() CapabilityReport {
	return CapabilityReport{Harness: s.harness, AdapterVer: AdapterVersion, Capabilities: s.capabilities}
}

func (s stubAdapter) Capture(sessionID string, _ CaptureOpts) (Profile, error) {
	if sessionID == "" && !s.acceptsNoSession {
		return Profile{}, SessionIDRequiredError(s.Name())
	}
	tokens := s.tokens
	if s.pointedAt == stubCacheOnlyExport {
		tokens = s.cacheTokens
	}
	return Profile{
		Schema:    ProfileSchema,
		Harness:   s.harness,
		SessionID: sessionID,
		Capability: CapabilityReport{
			Harness: s.harness, AdapterVer: AdapterVersion, Capabilities: s.capabilities,
		},
		Tokens:          tokens,
		ToolCalls:       UnknownToolCallResult("the stub reads no tool calls"),
		SkillActivation: UnknownActivationResult("the stub reads no activations"),
		Timing:          UnknownTimingResult("the stub reads no timing"),
		Attribution:     UnknownAttributionResult("the stub maps no outputs to skills"),
	}, nil
}

// honestStub keeps all three rules: it refuses an empty session, advertises
// exactly the one signal it delivers, and reports no count it did not read.
func honestStub() stubAdapter {
	return stubAdapter{
		harness: "stub",
		capabilities: map[MetricName]MetricSource{
			MetricTokens:          SourceOtel,
			MetricToolCalls:       SourceNone,
			MetricSkillActivation: SourceNone,
			MetricTiming:          SourceNone,
			MetricAttribution:     SourceNone,
		},
		tokens:      PresentTokenResult(TokenCounts{Input: Count(10), Output: Count(5)}, string(SourceOtel)),
		cacheTokens: PresentTokenResult(TokenCounts{CacheRead: Count(7)}, string(SourceOtel)),
	}
}

// stubEntry registers a stub the way a real adapter is registered, so the
// checks below run through exactly the path adapterRegistry's entries do.
func stubEntry(name string, s stubAdapter) adapterEntry {
	s.harness = name
	return adapterEntry{
		name: name,
		newAdapter: func(export string) ProfilerAdapter {
			built := s
			built.pointedAt = export
			return built
		},
		export:          stubPrimaryExport,
		session:         "stub-session",
		cacheOnlyExport: stubCacheOnlyExport,
	}
}

func TestTheContractTableCanFail(t *testing.T) {
	overAdvertising := honestStub()
	overAdvertising.tokens = UnknownTokenResult("schema not yet verified; token fields are unmapped")

	underAdvertising := honestStub()
	underAdvertising.capabilities = map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}

	acceptsAnySession := honestStub()
	acceptsAnySession.acceptsNoSession = true

	fabricatesZeroes := honestStub()
	fabricatesZeroes.cacheTokens = PresentTokenResult(
		TokenCounts{Input: Count(0), Output: Count(0), CacheRead: Count(7)}, string(SourceOtel))

	advertisesNothing := honestStub()
	advertisesNothing.capabilities = map[MetricName]MetricSource{}

	misnamed := honestStub()

	cases := []struct {
		name  string
		entry adapterEntry
		want  string
	}{
		{
			name:  "advertises a capability its capture cannot deliver",
			entry: stubEntry("over-advertising", overAdvertising),
			want:  `tokens: probe advertised "otel", capture state is "unknown"`,
		},
		{
			name:  "returns a value it never advertised a source for",
			entry: stubEntry("under-advertising", underAdvertising),
			want:  `tokens: probe advertised "none" and capture returned "present"`,
		},
		{
			name:  "captures without a session id",
			entry: stubEntry("accepts-any-session", acceptsAnySession),
			want:  "Capture with an empty session id returned no error",
		},
		{
			name:  "reports counts the export never carried",
			entry: stubEntry("fabricates-zeroes", fabricatesZeroes),
			want:  `the cache-only export carries no input count and the profile reports "input": 0`,
		},
		{
			name:  "advertises nothing at all",
			entry: stubEntry("advertises-nothing", advertisesNothing),
			want:  "the capability report covers 0 signals and the profile carries 5",
		},
		{
			name:  "answers to a name other than the one it is registered under",
			entry: stubEntry("declared-name", misnamed),
			want:  `registered as "wrong-name", Name() returns "declared-name"`,
		},
	}
	// The one case that is a property of the entry rather than of the adapter.
	cases[len(cases)-1].entry.name = "wrong-name"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := adapterContractViolations(tc.entry)
			if len(got) == 0 {
				t.Fatalf("the contract table found nothing wrong with an adapter that %s", tc.name)
			}
			if !containsSubstring(got, tc.want) {
				t.Errorf("the table refused the adapter for the wrong reason.\ngot:  %v\nwant a violation containing: %q", got, tc.want)
			}
		})
	}
}

// TestTheContractTableAcceptsAnHonestAdapter is the other half of the control.
// Without it every check above is satisfied by a table that refuses everything,
// which would also refuse every adapter that keeps the contract.
func TestTheContractTableAcceptsAnHonestAdapter(t *testing.T) {
	if got := adapterContractViolations(stubEntry("stub", honestStub())); len(got) != 0 {
		t.Errorf("the contract table refused an adapter that keeps all three rules: %v", got)
	}
}

// TestAnAdapterWithNoTokenSignalNeedsNoCacheExport pins the one exemption in
// rule 3, in both directions: an adapter that advertises no tokens may register
// no cache-only export, and one that advertises tokens may not.
func TestAnAdapterWithNoTokenSignalNeedsNoCacheExport(t *testing.T) {
	noTokens := honestStub()
	noTokens.capabilities = map[MetricName]MetricSource{
		MetricTokens:          SourceNone,
		MetricToolCalls:       SourceNone,
		MetricSkillActivation: SourceNone,
		MetricTiming:          SourceNone,
		MetricAttribution:     SourceNone,
	}
	noTokens.tokens = UnknownTokenResult("this harness exports no token counts")

	exempt := stubEntry("no-token-signal", noTokens)
	exempt.cacheOnlyExport = ""
	if got := adapterContractViolations(exempt); len(got) != 0 {
		t.Errorf("an adapter advertising no token signal was required to register a "+
			"cache-only export: %v", got)
	}

	claimsTokens := stubEntry("claims-tokens", honestStub())
	claimsTokens.cacheOnlyExport = ""
	got := adapterContractViolations(claimsTokens)
	if !containsSubstring(got, "registers no cache-only export") {
		t.Errorf("an adapter advertising tokens opted out of rule 3 by registering no "+
			"cache-only export: %v", got)
	}
}

func containsSubstring(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
