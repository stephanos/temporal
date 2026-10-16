package record

import "go.temporal.io/server/tools/gomad/internal/model/choice"

const ExecutionEvidenceSchema = "gomad.execution-evidence/v1"

type OutcomeEvidence struct {
	SoftAssertions *SoftAssertions `json:"soft_assertions,omitempty"`
	Domain         string          `json:"domain"`
	Reason         string          `json:"reason"`
	Termination    string          `json:"termination"`
	ExitCode       *Uint64String   `json:"exit_code,omitempty"`
	Signal         *string         `json:"signal,omitempty"`
	Deadline       *string         `json:"deadline,omitempty"`
}

type ExecutionLimitsEvidence struct {
	ExecutionTimeoutNanos Uint64String `json:"execution_timeout_nanos"`
	TerminateGraceNanos   Uint64String `json:"terminate_grace_nanos"`
	OutputBytes           Uint64String `json:"output_bytes"`
	WorldTransitionBytes  Uint64String `json:"world_transition_bytes"`
	IOTranscriptBytes     Uint64String `json:"io_transcript_bytes"`
	ChoiceTraceBytes      Uint64String `json:"choice_trace_bytes,omitempty"`
}

type ChoiceEvidence struct {
	Profile                string           `json:"profile"`
	ImplementationSHA256   SHA256           `json:"implementation_sha256"`
	Limit                  Uint64String     `json:"limit"`
	SHA256                 SHA256           `json:"sha256"`
	Records                Uint64String     `json:"records"`
	BranchingRecords       Uint64String     `json:"branching_records"`
	TerminalState          string           `json:"terminal_state"`
	TapeSHA256             SHA256           `json:"tape_sha256"`
	Decisions              Uint64String     `json:"decisions"`
	Runnable               Uint64String     `json:"runnable"`
	SelectPoll             Uint64String     `json:"select_poll"`
	SelectResult           Uint64String     `json:"select_result"`
	PeakGoroutines         Uint64String     `json:"peak_goroutines"`
	Features               []choice.Feature `json:"features"`
	AdjacentPairsObserved  Uint64String     `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool             `json:"adjacent_pairs_truncated"`
}

type ExecutionEvidence struct {
	Schema                  string                     `json:"schema"`
	Seed                    Uint64String               `json:"seed"`
	RunnerBuild             string                     `json:"runner_build"`
	Toolchain               Toolchain                  `json:"toolchain"`
	Target                  Target                     `json:"target"`
	IOProfile               IOContract                 `json:"io_profile"`
	Environment             []Environment              `json:"environment"`
	Limits                  ExecutionLimitsEvidence    `json:"limits"`
	Outcome                 OutcomeEvidence            `json:"outcome"`
	GroupGone               bool                       `json:"group_gone"`
	Stdout                  Stream                     `json:"stdout"`
	Stderr                  Stream                     `json:"stderr"`
	IOTranscriptSHA256      SHA256                     `json:"io_transcript_sha256"`
	IOTranscriptRecords     Uint64String               `json:"io_transcript_records"`
	IOTranscriptComplete    bool                       `json:"io_transcript_complete"`
	BackendEvidenceSHA256   SHA256                     `json:"backend_evidence_sha256,omitempty"`
	VirtualTimeElapsedNanos Uint64String               `json:"virtual_time_elapsed_nanos"`
	Diagnostics             *DiagnosticEvidence        `json:"diagnostics,omitempty"`
	Choices                 *ChoiceEvidence            `json:"choices,omitempty"`
	World                   World                      `json:"world"`
	ReadOnlyMountsSHA256    *SHA256                    `json:"read_only_mounts_sha256,omitempty"`
	SemanticCoverage        SemanticCoverage           `json:"semantic_coverage"`
	ChoiceExploration       *ChoiceExplorationEvidence `json:"choice_exploration,omitempty"`
}

type ChoiceExplorationEvidence struct {
	ImplementationSHA256 SHA256       `json:"implementation_sha256"`
	Round                Uint64String `json:"round"`
	CandidateSHA256      SHA256       `json:"candidate_sha256"`
	ParentSHA256         SHA256       `json:"parent_sha256,omitempty"`
	PrefixSHA256         SHA256       `json:"prefix_sha256,omitempty"`
	ForcedDepth          Uint64String `json:"forced_depth"`
	OutcomeSHA256        SHA256       `json:"outcome_sha256"`
}

type DiagnosticEvidence struct {
	Profile string       `json:"profile"`
	SHA256  SHA256       `json:"sha256"`
	Records Uint64String `json:"records"`
}

type DiagnosticTraceReference struct {
	Path string `json:"path"`
	DiagnosticEvidence
}

type ChoiceTraceSummary struct {
	Seed             uint64
	Profile          string
	Limit            uint64
	SHA256           SHA256
	Records          uint64
	BranchingRecords uint64
	Runnable         uint64
	SelectPoll       uint64
	SelectResult     uint64
	TerminalState    string
	TapeSHA256       SHA256
	Decisions        uint64
	PeakGoroutines   uint32
}
