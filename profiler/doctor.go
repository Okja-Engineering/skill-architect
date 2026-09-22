// What this machine can and cannot measure.
//
// # Why this file is written defensively
//
// `doctor` is the most likely place in this product to overstate, because
// making claims about capability is its entire job. This release kept three
// adapters out for advertising signals their captures returned nothing for, and
// a tier is that claim with a shorter name: "hooks" reads as "hook capture is
// working here", and a reader who believes it expects a profile out of the
// spool. So the report has two halves that a reader cannot confuse:
//
//   - **measurement** — what a registered harness actually read. Its only input
//     is a capability report from a probe over a real file. Nothing found on the
//     machine can raise it, because nothing found on a machine is a measurement:
//     a file existing, a hook being registered and an environment variable being
//     set are all things that are true of installations that measure nothing.
//   - **observed** — what is on this machine. Counts and paths. No tier, no
//     state, no source, and one field that says in the data what the counts do
//     not yield.
//
// # The hook surface produces no signal in this release, and says so
//
// The spool is capture only: no adapter reads it, so there is no tier for it and
// no signal it can contribute. `doctor` may report that a spool exists and how
// much is in it — that is true, and it is what a user wants to know about a
// capture they set up. What it may not report is that this machine can capture
// tokens, tool calls or anything else from Cursor. The distinction is the
// subject of the last five releases, so it is carried in the document rather
// than left to the prose: every spool observation ships the sentence in
// spoolYieldsNothingMeasured beside its counts.
//
// # Three things the draft reported that this does not
//
//   - **A Cursor user-data directory.** The draft read
//     `~/Library/Application Support/Cursor` and reported `cursor_installed`.
//     That path is macOS's only, so `false` means "not on a Mac" as often as it
//     means "no Cursor" — and either way whether Cursor is installed is not a
//     statement about what can be measured. Dropped rather than hedged.
//   - **`CURSOR_ADMIN_API_KEY`, as a `server_api` tier.** There is no Admin API
//     client in this repository. An environment variable being set is not a
//     surface anything here can read.
//   - **`OTEL_EXPORTER_OTLP_ENDPOINT`, as an `enterprise` tier.** An endpoint
//     is where a harness sends telemetry, not a file this tool can read; the
//     one adapter that reads OTel reads an export file the caller names. The
//     variable being set says nothing about whether such a file exists.
//
// Reporting any of the three would put a hopeful word next to a tier, and a
// tier is exactly where a hopeful word becomes a capability claim.
package profiler

import (
	"fmt"
	"sort"
	"time"
)

// EnvironmentTier names what this build can turn into a measured profile here.
//
// The enumeration is short because it is the set of claims that can be
// demonstrated: adapter_contract_test.go walks these constants and requires,
// for every tier above none, an input from which this function reports it *and*
// from which a capture produces a `present` signal. A tier added without one
// turns that table red, which is the only thing standing between this
// enumeration and the four tiers the draft advertised over surfaces nothing
// reads.
type EnvironmentTier string

const (
	// TierNone: nothing was read that this build can turn into a profile.
	TierNone EnvironmentTier = "none"
	// TierExport: a registered harness probed the export supplied and reports
	// at least one signal from a real source.
	TierExport EnvironmentTier = "export"
)

// spoolYieldsNothingMeasured is what a spool is allowed to say about itself.
//
// It ships with the counts, in the data, because a stored report travels and a
// caveat in a README does not travel with it. The wording is deliberate: the
// files are real and are worth having, and what is absent is a reader, not the
// capture.
const spoolYieldsNothingMeasured = "no measurement: no adapter in this build reads the spool, so these lines are " +
	"a capture to be parsed later and no profile, token count or tool call can be produced from them"

// EnvironmentQuery is what one detection is asked about.
//
// A struct rather than five parameters, because two of them are paths and two
// are names and a caller swapping a pair would compile. Every field is supplied
// by the caller: this function reads no environment variable and resolves no
// home of its own, so a test can point it anywhere and the CLI is the one place
// that decides what "the user's own" means.
type EnvironmentQuery struct {
	// Home is the directory whose .cursor/hooks.json and spool are observed.
	Home string

	// SpoolDir overrides the default of <Home>/.skill-architect/spool.
	SpoolDir string

	// HookCommand is the registration to look for in hooks.json. An entry is
	// ours only if its command is exactly this, which is the same test
	// InstallHooks and UninstallHooks apply.
	HookCommand string

	// Harness and ExportFile are the measurement question: which registered
	// adapter to ask, and the export to ask it about. With no ExportFile no
	// adapter is asked anything, and the tier is none — because a tier is what
	// a probe read and there is nothing to read.
	Harness    string
	ExportFile string
}

// EnvironmentReport is the doctor document.
type EnvironmentReport struct {
	DetectedAt     string `json:"detected_at"`
	AdapterVersion string `json:"adapter_version"`

	Measurement MeasurementSurface  `json:"measurement"`
	Observed    MachineObservations `json:"observed"`
}

// MeasurementSurface is the capability half: what a harness read, and nothing
// else.
type MeasurementSurface struct {
	Tier EnvironmentTier `json:"tier"`

	// Harness and Export are what was asked and what it was asked about, so a
	// tier can be re-derived by hand from the report.
	Harness string `json:"harness,omitempty"`
	Export  string `json:"export,omitempty"`

	// Signals is the probe's own capability report, carried verbatim rather
	// than summarised. The tier is derived from it, so a reader who disagrees
	// with the derivation can see what it was derived from.
	Signals map[MetricName]MetricSource `json:"signals,omitempty"`

	// Reason is why the tier is what it is, always present — including for a
	// tier above none, because "which file, read by whom" is the part a user
	// needs in order to reproduce it.
	Reason string `json:"reason"`
}

// MachineObservations is what is on this machine. Nothing here is a capability,
// and nothing here can raise a tier.
type MachineObservations struct {
	Home      string               `json:"home"`
	HooksJSON HooksJSONObservation `json:"hooks_json"`
	Spool     SpoolObservation     `json:"spool"`
}

// HooksJSONObservation is the state of Cursor's hooks file and our registration
// in it.
type HooksJSONObservation struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`

	// Unreadable is why the file could not be parsed, when it could not.
	//
	// A file that exists and cannot be read is not a machine with nothing
	// registered. Reporting it as "no events" would be the reader's own failure
	// told to the user as a fact about their machine — and it is the failure a
	// user is most likely to have caused, by hand-editing the file.
	Unreadable string `json:"unreadable,omitempty"`

	// RegisteredEvents is the events whose entries carry exactly the command
	// the query named, sorted so two runs over one file read the same.
	RegisteredEvents []string `json:"registered_hook_events,omitempty"`
}

// SpoolObservation is what is in the spool, and what it yields.
type SpoolObservation struct {
	Dir    string `json:"dir"`
	Exists bool   `json:"exists"`

	// Unreadable is why the directory could not be read, when it could not. An
	// unreadable spool is not an empty one, for the same reason an unparseable
	// hooks.json is not an empty registration.
	Unreadable string `json:"unreadable,omitempty"`

	Files           int    `json:"files"`
	Lines           int    `json:"lines"`
	UnreadableLines int    `json:"unreadable_lines"`
	PayloadBytes    int64  `json:"payload_bytes"`
	LastCaptureAt   string `json:"last_capture_at,omitempty"`

	// Yields is spoolYieldsNothingMeasured, always. It is a field rather than a
	// comment because the counts above travel and a reader who has only the
	// counts concludes that the capture is feeding something.
	Yields string `json:"yields"`
}

// DetectEnvironment inspects one home and one export and reports what can be
// measured here.
//
// It never fails. Detection is a report, not a gate: "this command errored" is
// not something a user can act on, and every way a read can fail is reported as
// the reason it failed, in the half of the document that read is about.
func DetectEnvironment(q EnvironmentQuery) EnvironmentReport {
	spoolDir := q.SpoolDir
	if spoolDir == "" {
		// The writer's own layout, relative to the home being asked about.
		// SpoolDirIn is that layout, and it is called rather than repeated:
		// this join used to be spelled out here and held to the writer's by a
		// comment, and a comment is not a mechanism.
		spoolDir = SpoolDirIn(q.Home)
	}

	return EnvironmentReport{
		DetectedAt:     time.Now().UTC().Format(time.RFC3339),
		AdapterVersion: AdapterVersion,
		Measurement:    measure(q.Harness, q.ExportFile),
		Observed: MachineObservations{
			Home:      q.Home,
			HooksJSON: observeHooksJSON(q.Home, q.HookCommand),
			Spool:     observeSpool(spoolDir),
		},
	}
}

// measure asks one registered adapter what it can read out of one export, and
// is the only thing in this file that can return a tier above none.
//
// Its inputs are a harness name and a file — deliberately not the home, not the
// spool and not the environment. A function that could see those could be made
// to raise a tier from one of them, which is the defect this shape forecloses
// rather than guards against.
func measure(harness, exportFile string) MeasurementSurface {
	if exportFile == "" {
		return MeasurementSurface{
			Tier:    TierNone,
			Harness: harness,
			Reason: "no export was supplied, so no harness was asked to read anything. A measurement surface is " +
				"what a probe read: pass --harness and --otel-file to have one read, or run " +
				"`profiler capture --harness <name> --otel-file <path>` to produce a profile",
		}
	}
	if harness == "" {
		return MeasurementSurface{
			Tier:   TierNone,
			Export: exportFile,
			Reason: fmt.Sprintf("an export was supplied and no harness was named, so nobody was asked to read %s (supported: %s)",
				exportFile, SupportedHarnesses()),
		}
	}

	adapter, ok := NewAdapter(harness, exportFile)
	if !ok {
		return MeasurementSurface{
			Tier:    TierNone,
			Harness: harness,
			Export:  exportFile,
			Reason:  fmt.Sprintf("there is no harness named %q (supported: %s)", harness, SupportedHarnesses()),
		}
	}

	report := adapter.Probe()
	surface := MeasurementSurface{
		Tier:    TierNone,
		Harness: adapter.Name(),
		Export:  exportFile,
		Signals: report.Capabilities,
		Reason:  "",
	}

	readable := readableSignals(report.Capabilities)
	if len(readable) == 0 {
		surface.Reason = fmt.Sprintf("%s read %s and reports no signal from a real source, so there is nothing here to measure with",
			adapter.Name(), exportFile)
		return surface
	}

	surface.Tier = TierExport
	surface.Reason = fmt.Sprintf("%s read %s and reports %s from a real source",
		adapter.Name(), exportFile, joinMetrics(readable))
	return surface
}

// readableSignals is the signals a capability report names a real source for,
// sorted. SourceNone is the adapter saying it has no source for that signal,
// which is the answer that must not add up to a tier.
func readableSignals(capabilities map[MetricName]MetricSource) []MetricName {
	var readable []MetricName
	for metric, source := range capabilities {
		if source != SourceNone {
			readable = append(readable, metric)
		}
	}
	sort.Slice(readable, func(i, j int) bool { return readable[i] < readable[j] })
	return readable
}

// joinMetrics is a metric list for a sentence a person reads.
func joinMetrics(metrics []MetricName) string {
	names := make([]string, 0, len(metrics))
	for _, m := range metrics {
		names = append(names, string(m))
	}
	switch len(names) {
	case 1:
		return names[0]
	default:
		return fmt.Sprintf("%d signals (%s)", len(names), joinComma(names))
	}
}

func joinComma(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// observeHooksJSON reads Cursor's hooks file through the same reader the
// install uses, and matches our entry the same way.
//
// The draft string-matched the file for "profiler ingest". Two things are wrong
// with that and both matter: the command `hooks install` registers is this
// binary's absolute path plus `ingest || true`, so the match would miss a real
// registration, and a foreign hook whose command merely mentioned us would
// count as one. Asking readHooksDoc and entryCommand means "our entry" has one
// definition, and install, uninstall and doctor all answer to it.
func observeHooksJSON(home, command string) HooksJSONObservation {
	obs := HooksJSONObservation{Path: hooksJSONIn(home)}

	doc, err := readHooksDoc(obs.Path)
	if err != nil {
		// readHooksDoc reports an absent file as not-exists rather than as an
		// error, so an error here is a file that is there and could not be
		// read.
		obs.Present = true
		obs.Unreadable = err.Error()
		return obs
	}
	if !doc.exists {
		return obs
	}
	obs.Present = true

	if command == "" {
		// Nothing was named, so nothing can be recognised as ours. Reporting
		// every event in the file would be reporting somebody else's hooks as
		// this tool's registration.
		return obs
	}
	for event, raw := range hooksTable(doc.root) {
		entries, _ := raw.([]any)
		if hasCommand(entries, command) {
			obs.RegisteredEvents = append(obs.RegisteredEvents, event)
		}
	}
	sort.Strings(obs.RegisteredEvents)
	return obs
}

// observeSpool counts what is in the spool, through AnalyzeSpool.
//
// It goes through the summary rather than listing the directory itself, because
// the draft's version was a third reader of the same files with its own idea of
// what a spool line is — it stat'ed every *.jsonl for a byte count and then
// re-read each one for a timestamp. Payload bytes rather than bytes on disk:
// the question is how much was captured, not how much of the filesystem it
// takes up, and one is a property of the capture while the other is a property
// of the format.
func observeSpool(dir string) SpoolObservation {
	obs := SpoolObservation{Dir: dir, Yields: spoolYieldsNothingMeasured}

	summary, err := AnalyzeSpool(dir)
	if err != nil {
		obs.Unreadable = err.Error()
		return obs
	}

	obs.Exists = true
	obs.Files = summary.Files
	obs.Lines = summary.Lines
	obs.UnreadableLines = summary.UnreadableLines
	obs.PayloadBytes = summary.PayloadBytes
	obs.LastCaptureAt = summary.LastCaptureAt
	return obs
}
