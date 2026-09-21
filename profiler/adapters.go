package profiler

import (
	"sort"
	"strings"
)

// The harnesses this package can dispatch, in one place.
//
// Until 0.5.0 the only list of them was a `switch` in the CLI's own package,
// with the same set spelled out again in two flag descriptions and two refusal
// messages beside it. Four copies is not a style problem: a second adapter had
// to be added to all of them and to the contract table, and the copy that was
// forgotten would have been a harness the CLI advertised and could not build,
// or built and never advertised.
//
// It also decides who can own an adapter. `NewAdapter` is the only way the CLI
// resolves a harness name, so an adapter the CLI can dispatch is an adapter
// registered here — and therefore a type in this package, which is what the
// contract table in adapter_contract_test.go asserts its three obligations
// over. An adapter defined in the command's own package could not be reached
// by that table, so it is refused outright rather than left as a gap somebody
// has to remember.

// adapterRegistration is one harness: the name a caller selects it by, and how
// to build it for a given export.
//
// The constructor takes the export path rather than a CaptureOpts because
// choosing an adapter happens before a capture is configured, and because what
// distinguishes one adapter from another at construction time is which file it
// reads. Everything else a capture needs is passed to Capture.
type adapterRegistration struct {
	name string
	new  func(exportFile string) ProfilerAdapter
}

// adapterRegistry is every harness this package ships.
var adapterRegistry = []adapterRegistration{
	{
		name: "claude_code",
		new:  func(exportFile string) ProfilerAdapter { return ClaudeCodeAdapter{OtelExportFile: exportFile} },
	},
}

// NewAdapter builds the adapter for a harness name, reporting whether there is
// one.
//
// The second return is an explicit "no such harness" rather than a nil
// interface, so a caller cannot mistake an unknown name for an adapter that
// declined to build itself.
func NewAdapter(harness, exportFile string) (ProfilerAdapter, bool) {
	for _, registered := range adapterRegistry {
		if registered.name == harness {
			return registered.new(exportFile), true
		}
	}
	return nil, false
}

// HarnessNames is every name NewAdapter accepts, sorted so that help text and
// refusals read the same way on every run.
func HarnessNames() []string {
	names := make([]string, 0, len(adapterRegistry))
	for _, registered := range adapterRegistry {
		names = append(names, registered.name)
	}
	sort.Strings(names)
	return names
}

// SupportedHarnesses is HarnessNames as one line, for the help text that offers
// the choice and the refusal that reports an unrecognised one. Both say the
// same thing because both ask here.
func SupportedHarnesses() string {
	return strings.Join(HarnessNames(), ", ")
}
