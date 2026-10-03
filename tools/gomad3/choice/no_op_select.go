package choice

// NoOpSelectShape is the evidence a select-poll decision carries: how many
// cases its select polled, which is the alternatives of the select's last poll
// step, and the readiness the runtime recorded with the select's result.
type NoOpSelectShape struct {
	PolledCases uint32
	Readiness   SelectReadiness
}

// NoOpSelectShapes lists the shapes whose poll order cannot change behavior.
// Poll order matters only when at least two cases are ready, but a shape is
// listed only after the select-readiness fixture (internal/gomadtool/conformance)
// explored it to exhaustion with and without its poll decisions expanded and
// reached the same outcomes and deadlocks; the comparison is retained under
// .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/select-reduction/.
// The fixture requires its proven shapes and this list to be the same set, and
// the explorer expands every select-poll decision of a shape off the list.
func NoOpSelectShapes() []NoOpSelectShape {
	return []NoOpSelectShape{
		{PolledCases: 2, Readiness: SelectReadiness{Known: true}},                                // blocking-zero-ready
		{PolledCases: 2, Readiness: SelectReadiness{Known: true, Ready: 1}},                      // blocking-one-ready
		{PolledCases: 2, Readiness: SelectReadiness{Known: true, Default: true}},                 // nonblocking-default
		{PolledCases: 2, Readiness: SelectReadiness{Known: true, TimerChannel: true}},            // timer-channel
		{PolledCases: 2, Readiness: SelectReadiness{Known: true, Ready: 1, ClosedChannel: true}}, // closed-channel
		{PolledCases: 2, Readiness: SelectReadiness{Known: true, Ready: 1, NilChannel: true}},    // nil-channel
		{PolledCases: 2, Readiness: SelectReadiness{Known: true, Ready: 1, TimerChannel: true}},  // timer-channel-due
	}
}
