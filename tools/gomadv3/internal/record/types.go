package record

const SchemaVersion uint32 = 1

const (
	ArtifactTargetFailure   = "gomadv3.target-failure/v1"
	ArtifactWatchdogTimeout = "gomadv3.watchdog-timeout/v1"
	ArtifactRunnerFailure   = "gomadv3.runner-failure/v1"
)

const (
	ReplayExact      = "exact"
	ReplayDiagnostic = "diagnostic"
	ReplayNone       = "none"
)

type SHA256 string

type Manifest struct {
	SchemaVersion    uint32        `json:"schema_version"`
	ArtifactKind     string        `json:"artifact_kind"`
	RecordHash       SHA256        `json:"record_hash"`
	CreatedAt        string        `json:"created_at"`
	BatchID          string        `json:"batch_id"`
	SelectionOrdinal Uint64String  `json:"selection_ordinal"`
	Seed             Uint64String  `json:"seed"`
	ReplayMode       string        `json:"replay_mode"`
	Runner           Runner        `json:"runner"`
	Toolchain        Toolchain     `json:"toolchain"`
	Target           Target        `json:"target"`
	Environment      []Environment `json:"environment"`
	Limits           Limits        `json:"limits"`
	World            World         `json:"world"`
	Outcome          Outcome       `json:"outcome"`
	Streams          Streams       `json:"streams"`
	Files            []File        `json:"files"`
	Host             Host          `json:"host"`
}

type Runner struct {
	RecordContract string `json:"record_contract"`
	RunnerBuild    string `json:"runner_build"`
	HostOS         string `json:"host_os"`
	HostArch       string `json:"host_arch"`
}

type Toolchain struct {
	GoVersion    string `json:"go_version"`
	BuildKey     string `json:"build_key"`
	TargetGOOS   string `json:"target_goos"`
	TargetGOARCH string `json:"target_goarch"`
}

type Target struct {
	Kind      string       `json:"kind"`
	Source    string       `json:"source"`
	File      string       `json:"file"`
	SHA256    SHA256       `json:"sha256"`
	Size      Uint64String `json:"size"`
	Argv      []string     `json:"argv"`
	BuildTags []string     `json:"build_tags"`
	BuildInfo BuildInfo    `json:"build_info"`
}

type BuildInfo struct {
	GoVersion  string         `json:"go_version"`
	Path       string         `json:"path"`
	MainModule string         `json:"main_module"`
	Settings   []BuildSetting `json:"settings"`
}

type BuildSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Environment struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Limits struct {
	RunTimeoutNanos      Uint64String `json:"run_timeout_nanos"`
	OverallTimeoutNanos  Uint64String `json:"overall_timeout_nanos"`
	TerminateGraceNanos  Uint64String `json:"terminate_grace_nanos"`
	OutputBytes          Uint64String `json:"output_bytes"`
	WorldTransitionBytes Uint64String `json:"world_transition_bytes"`
}

type World struct {
	Initial     WorldPayload     `json:"initial"`
	Transitions WorldTransitions `json:"transitions"`
	Final       WorldPayload     `json:"final"`
	Adapters    []WorldAdapter   `json:"adapters"`
	Terminal    WorldTerminal    `json:"terminal"`
}

type WorldTerminal struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

type WorldAdapter struct {
	Schema        string `json:"schema"`
	InitialDigest SHA256 `json:"initial_digest"`
	FinalDigest   SHA256 `json:"final_digest"`
}

type WorldPayload struct {
	Schema         string `json:"schema"`
	File           string `json:"file"`
	RawSHA256      SHA256 `json:"raw_sha256"`
	SemanticDigest SHA256 `json:"semantic_digest"`
}

type WorldTransitions struct {
	Schema           string       `json:"schema"`
	File             string       `json:"file"`
	RawSHA256        SHA256       `json:"raw_sha256"`
	Count            Uint64String `json:"count"`
	TranscriptDigest SHA256       `json:"transcript_digest"`
}

type WorldPayloads struct {
	Initial     []byte
	Transitions []byte
	Final       []byte
}

type Outcome struct {
	Domain           string        `json:"domain"`
	Reason           string        `json:"reason"`
	Termination      string        `json:"termination"`
	ExitCode         *Uint64String `json:"exit_code"`
	Signal           *string       `json:"signal"`
	Deadline         *string       `json:"deadline"`
	FailureSignature SHA256        `json:"failure_signature"`
	ReplayMatch      *bool         `json:"replay_match"`
}

type Streams struct {
	Stdout Stream `json:"stdout"`
	Stderr Stream `json:"stderr"`
}

type Stream struct {
	File           string       `json:"file"`
	RetainedSHA256 SHA256       `json:"retained_sha256"`
	FullSHA256     SHA256       `json:"full_sha256"`
	TotalBytes     Uint64String `json:"total_bytes"`
	RetainedBytes  Uint64String `json:"retained_bytes"`
	DiscardedBytes Uint64String `json:"discarded_bytes"`
	Truncated      bool         `json:"truncated"`
}

type File struct {
	Path   string       `json:"path"`
	Mode   string       `json:"mode"`
	Size   Uint64String `json:"size"`
	SHA256 SHA256       `json:"sha256"`
}

type Host struct {
	StartedAt    string       `json:"started_at"`
	FinishedAt   string       `json:"finished_at"`
	ElapsedNanos Uint64String `json:"elapsed_nanos"`
}
