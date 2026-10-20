package backends

import (
	"crypto/sha256"
	"os"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

type ExecutionRequest struct {
	Target           target.Prepared
	Seed             uint64
	RunnerBuild      string
	Command          string
	Args             []string
	Argv0            string
	Dir              string
	Env              []string
	ExecutionTimeout time.Duration
	TerminateGrace   time.Duration
	OutputLimit      uint64
	// Output heads are borrowed. The caller owns their sync and close lifecycle.
	StdoutHead  *os.File
	StderrHead  *os.File
	World       WorldCapability
	IO          *IOCapability
	Diagnostics bool
	Choice      *ChoiceCapability
	Simulation  *SimulationCapability
	Coordinator *CoordinatorRequest
}

type CoordinatorRequest struct {
	Input []byte
	// Execute takes ownership of Stdout and closes it after starting the child,
	// or before returning any pre-start failure. The caller retains the read end.
	Stdout   *os.File
	Deadline time.Time
}

type WorldCapability struct {
	RecordLimit     uint64
	TransitionLimit uint64
	Seed            uint64
	ExpectedInitial []byte
	ReplayPlan      []byte
}

type IOCapability struct {
	Profile       record.IOProfile
	Transcript    *IOTranscriptCapability
	ReadOnlyMount *ReadOnlyMountCapability
}

type IOTranscriptCapability struct {
	Limit    uint64
	Replay   bool
	Expected []byte
}

type ReadOnlyMountCapability struct {
	Mappings []captured.Mapping
	Limits   captured.Limits
	Replay   *captured.Snapshot
}

type ChoiceCapability struct {
	Mode                 choice.Mode
	Profile              string
	ImplementationSHA256 [sha256.Size]byte
	ExecutionIdentity    choice.ExecutionIdentity
	Limit                uint64
	ReplayPlan           *choice.ReplayPlan
	Trace                []byte
	TraceSHA256          [sha256.Size]byte
	TapeSHA256           [sha256.Size]byte
	Decisions            uint64
	PrefixControl        *ChoicePrefixControl
}

type SimulationRole string

const (
	SimulationRoleCoordinator SimulationRole = "coordinator"
	SimulationRoleNode        SimulationRole = "node"
)

type SimulationCapability struct {
	Role                   SimulationRole
	ExplorationPlan        []byte
	ExplorationRecordLimit uint64
	ExplorationRecordCount uint64
}

type Output struct {
	Bytes          []byte
	RawBytes       []byte
	FullSHA256     [sha256.Size]byte
	RetainedSHA256 [sha256.Size]byte
	TotalBytes     uint64
	RetainedBytes  uint64
	DiscardedBytes uint64
	Truncated      bool
}

type Transcript struct {
	Bytes            []byte
	SHA256           [sha256.Size]byte
	Records          uint64
	Complete         bool
	ReplayDivergence *uint64
}

type ChoiceTrace struct {
	Profile              string
	ImplementationSHA256 [sha256.Size]byte
	Limit                uint64
	Trace                choice.Trace
	TapeSHA256           [sha256.Size]byte
	Decisions            uint64
}

type ExecutionTermination string

const (
	TerminationExit   ExecutionTermination = "exit"
	TerminationSignal ExecutionTermination = "signal"
)

type ExecutionResult struct {
	SoftAssertions          *record.SoftAssertions
	BackendEvidence         []byte
	BackendEvidenceSchema   string
	Captured                bool
	Termination             ExecutionTermination
	ExitCode                int
	Signal                  string
	WatchdogTimeout         bool
	Cancelled               bool
	Stdout                  Output
	Stderr                  Output
	PID                     int
	PGID                    int
	GroupGone               bool
	WorldRecord             []byte
	IOTranscript            Transcript
	IOROMounts              captured.Snapshot
	DiagnosticTrace         choice.DiagnosticTrace
	ChoiceTrace             ChoiceTrace
	SimulationRecords       [][]byte
	VirtualTimeElapsedNanos uint64
	CoordinatorErrorReason  string
}
