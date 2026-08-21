package evidence

const LegacySchemaVersion uint32 = 2

const PreviousSchemaVersion uint32 = 3

const PriorSchemaVersion uint32 = 4

const SchemaVersion uint32 = 5

const LegacyRecordContract = "gomadv3.run-record/v2"

const PreviousRecordContract = "gomadv3.run-record/v3"

const PriorRecordContract = "gomadv3.run-record/v4"

const RecordContract = "gomadv3.run-record/v5"

const (
	ArtifactSuccess         = "gomadv3.success/v1"
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

type ExecutionRecord struct {
	SchemaVersion     uint32             `json:"schema_version"`
	ArtifactKind      string             `json:"artifact_kind"`
	RecordHash        SHA256             `json:"record_hash"`
	CreatedAt         string             `json:"created_at"`
	CampaignID        string             `json:"batch_id"`
	SelectionOrdinal  Uint64String       `json:"selection_ordinal"`
	Seed              Uint64String       `json:"seed"`
	ReplayMode        string             `json:"replay_mode"`
	Runner            Runner             `json:"runner"`
	Toolchain         Toolchain          `json:"toolchain"`
	Target            Target             `json:"target"`
	IOProfile         IOProfile          `json:"io_profile"`
	ChoiceProfile     *ChoiceProfile     `json:"choice_profile,omitempty"`
	SimulationProfile *SimulationProfile `json:"simulation_profile,omitempty"`
	Minimization      *Minimization      `json:"minimization,omitempty"`
	Environment       []Environment      `json:"environment"`
	Limits            Limits             `json:"limits"`
	World             World              `json:"world"`
	Outcome           Outcome            `json:"outcome"`
	Streams           Streams            `json:"streams"`
	Files             []File             `json:"files"`
	Host              Host               `json:"host"`
}

type IOProfile struct {
	Name                 string          `json:"name"`
	ImplementationSHA256 SHA256          `json:"implementation_sha256"`
	Inventory            string          `json:"inventory"`
	InventorySHA256      SHA256          `json:"inventory_sha256"`
	Transcript           *IOTranscript   `json:"transcript,omitempty"`
	ReadOnlyMounts       *ReadOnlyMounts `json:"read_only_mounts,omitempty"`
}

type IOTranscript struct {
	Schema  string       `json:"schema"`
	File    string       `json:"file"`
	SHA256  SHA256       `json:"sha256"`
	Bytes   Uint64String `json:"bytes"`
	Records Uint64String `json:"records"`
}

type ChoiceProfile struct {
	Name                 string      `json:"name"`
	ImplementationSHA256 SHA256      `json:"implementation_sha256"`
	Trace                ChoiceTrace `json:"trace"`
}

type ChoiceTrace struct {
	Schema           string       `json:"schema"`
	File             string       `json:"file"`
	SHA256           SHA256       `json:"sha256"`
	Bytes            Uint64String `json:"bytes"`
	Records          Uint64String `json:"records"`
	BranchingRecords Uint64String `json:"branching_records"`
	TerminalState    string       `json:"terminal_state"`
	Limit            Uint64String `json:"limit"`
	TapeSHA256       SHA256       `json:"tape_sha256,omitempty"`
	Decisions        Uint64String `json:"decisions,omitempty"`
}

type SimulationProfile struct {
	Name             string           `json:"name"`
	ControllerSHA256 SHA256           `json:"controller_sha256"`
	ExecutionSHA256  SHA256           `json:"execution_sha256"`
	CandidateSHA256  SHA256           `json:"candidate_sha256"`
	OutcomeSHA256    SHA256           `json:"outcome_sha256"`
	FailureSHA256    SHA256           `json:"failure_sha256,omitempty"`
	Plan             SimulationPlan   `json:"plan"`
	Record           SimulationRecord `json:"record"`
}

type SimulationPlan struct {
	Schema string       `json:"schema"`
	File   string       `json:"file"`
	SHA256 SHA256       `json:"sha256"`
	Bytes  Uint64String `json:"bytes"`
}

type SimulationRecord struct {
	Schema string       `json:"schema"`
	File   string       `json:"file"`
	SHA256 SHA256       `json:"sha256"`
	Bytes  Uint64String `json:"bytes"`
	Limit  Uint64String `json:"limit"`
}

type Minimization struct {
	Schema                  string                  `json:"schema"`
	ImplementationSHA256    SHA256                  `json:"implementation_sha256"`
	ParentRecordHash        SHA256                  `json:"parent_record_hash"`
	ParentFailureSignature  SHA256                  `json:"parent_failure_signature"`
	OriginalCandidateSHA256 SHA256                  `json:"original_candidate_sha256"`
	FinalCandidateSHA256    SHA256                  `json:"final_candidate_sha256"`
	AttemptBudget           Uint64String            `json:"attempt_budget"`
	Attempts                Uint64String            `json:"attempts"`
	OriginalForcedDecisions Uint64String            `json:"original_forced_decisions"`
	FinalForcedDecisions    Uint64String            `json:"final_forced_decisions"`
	Accepted                []MinimizationReduction `json:"accepted"`
	Predicate               MinimizationPredicate   `json:"predicate"`
}

type MinimizationReduction struct {
	Kind         string                 `json:"kind"`
	BeforeSHA256 SHA256                 `json:"before_sha256"`
	AfterSHA256  SHA256                 `json:"after_sha256"`
	Removed      []MinimizationDecision `json:"removed"`
}

type MinimizationDecision struct {
	Dimension string       `json:"dimension"`
	Ordinal   Uint64String `json:"ordinal"`
	Identity  SHA256       `json:"identity"`
}

type MinimizationPredicate struct {
	FailureSignature SHA256 `json:"failure_signature"`
	Domain           string `json:"domain"`
	Reason           string `json:"reason"`
	Termination      string `json:"termination"`
	ReplayMatch      bool   `json:"replay_match"`
	ChoiceReplay     string `json:"choice_replay"`
	SimulationReplay string `json:"simulation_replay"`
}

type ReadOnlyMounts struct {
	Schema     string              `json:"schema"`
	File       string              `json:"file"`
	SHA256     SHA256              `json:"sha256"`
	Bytes      Uint64String        `json:"bytes"`
	Entries    Uint64String        `json:"entries"`
	NotExist   Uint64String        `json:"not_exist,omitempty"`
	TotalBytes Uint64String        `json:"total_bytes"`
	Mappings   []string            `json:"mappings"`
	Limits     ReadOnlyMountLimits `json:"limits"`
}

type ReadOnlyMountLimits struct {
	PathBytes        Uint64String `json:"path_bytes"`
	Requests         Uint64String `json:"requests"`
	Files            Uint64String `json:"files"`
	DirectoryEntries Uint64String `json:"directory_entries"`
	SingleFileBytes  Uint64String `json:"single_file_bytes"`
	TotalBytes       Uint64String `json:"total_bytes"`
}

type ReadOnlyMountDescriptor struct {
	Schema     string               `json:"schema"`
	Mappings   []string             `json:"mappings"`
	Limits     ReadOnlyMountLimits  `json:"limits"`
	Requests   Uint64String         `json:"requests"`
	TotalBytes Uint64String         `json:"total_bytes"`
	NotExist   []string             `json:"not_exist,omitempty"`
	Entries    []ReadOnlyMountEntry `json:"entries"`
}

type ReadOnlyMountEntry struct {
	Path     string               `json:"path"`
	Mode     string               `json:"mode"`
	Kind     string               `json:"kind"`
	Size     Uint64String         `json:"size"`
	SHA256   SHA256               `json:"sha256,omitempty"`
	Payload  string               `json:"payload,omitempty"`
	Children []ReadOnlyMountChild `json:"children"`
}

type ReadOnlyMountChild struct {
	Name string `json:"name"`
	Mode string `json:"mode"`
	Kind string `json:"kind"`
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
	Kind               string                    `json:"kind"`
	Source             string                    `json:"source"`
	File               string                    `json:"file"`
	SHA256             SHA256                    `json:"sha256"`
	Size               Uint64String              `json:"size"`
	Argv               []string                  `json:"argv"`
	BuildTags          []string                  `json:"build_tags"`
	Adapters           []TargetAdapter           `json:"adapters"`
	Compatibility      []CompatibilityPack       `json:"compatibility"`
	BuildInfo          BuildInfo                 `json:"build_info"`
	CapabilityMode     string                    `json:"capability_mode,omitempty"`
	CapabilityManifest *TargetCapabilityManifest `json:"capability_manifest,omitempty"`
}

type TargetCapabilityManifest struct {
	Schema                       string       `json:"schema"`
	File                         string       `json:"file"`
	SHA256                       SHA256       `json:"sha256"`
	Bytes                        Uint64String `json:"bytes"`
	Facts                        Uint64String `json:"facts"`
	ProducerImplementationSHA256 SHA256       `json:"producer_implementation_sha256"`
	GuardImplementationSHA256    SHA256       `json:"guard_implementation_sha256,omitempty"`
	CapabilityUniverseSHA256     SHA256       `json:"capability_universe_sha256"`
}

type TargetAdapter struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
}

type CompatibilityPack struct {
	ID     string `json:"id"`
	SHA256 SHA256 `json:"sha256"`
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
	IOTranscriptBytes    Uint64String `json:"io_transcript_bytes"`
	ChoiceTraceBytes     Uint64String `json:"choice_trace_bytes,omitempty"`
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
