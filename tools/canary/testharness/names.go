package testharness

// The names a live test sets up the harness with. They are the package's only untagged part: the
// harness itself, with its plaintext transport, compiles only under the canary_harness tag.

// The environment the harness reads, beyond the canary's own coordinates and workflow context.
const (
	// VariablePolicy names the test policy's file.
	VariablePolicy = "UMPIRE_CANARY_HARNESS_POLICY"
	// VariableCrash names a phase at which the process exits at once, as a lost process would.
	VariableCrash = "UMPIRE_CANARY_HARNESS_CRASH"
	// VariablePause is `<phase>:<file>`: at that phase the process waits until the file exists.
	VariablePause = "UMPIRE_CANARY_HARNESS_PAUSE"
)

// ProfileName is the only Evaluation Profile a harness policy may name.
const ProfileName = "canary-harness"

// CrashExit is the exit code of a crashed harness process, which no mode ever returns.
const CrashExit = 99
