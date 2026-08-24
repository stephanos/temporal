package agentworkflow

import "time"

type RunID string

type Outcome string

const (
	OutcomeSucceeded               Outcome = "succeeded"
	OutcomeNeedsChanges            Outcome = "needs-changes"
	OutcomeProjectFailed           Outcome = "project-failed"
	OutcomeAgentFailed             Outcome = "agent-failed"
	OutcomeUnsupported             Outcome = "unsupported"
	OutcomeInconclusive            Outcome = "inconclusive"
	OutcomeCancelled               Outcome = "cancelled"
	OutcomeTimedOut                Outcome = "timed-out"
	OutcomeCapacityExhausted       Outcome = "capacity-exhausted"
	OutcomeInfrastructureFailed    Outcome = "infrastructure-failed"
	OutcomeRecoverableInterruption Outcome = "recoverable-interruption"
	OutcomeCorrupt                 Outcome = "corrupt"
)

type Result struct {
	Schema          string          `json:"schema"`
	RunID           RunID           `json:"run_id"`
	Outcome         Outcome         `json:"outcome"`
	Phase           string          `json:"phase"`
	Backend         BackendInfo     `json:"backend"`
	SourceDigest    string          `json:"source_digest,omitempty"`
	CandidateDigest string          `json:"candidate_digest,omitempty"`
	Changes         []Change        `json:"changes,omitempty"`
	Checks          []CheckResult   `json:"checks,omitempty"`
	Reviews         []ReviewResult  `json:"reviews,omitempty"`
	Findings        []FindingRecord `json:"findings,omitempty"`
	Repairs         int             `json:"repairs"`
	Message         string          `json:"message,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	FinishedAt      time.Time       `json:"finished_at,omitempty"`
}

type Status struct {
	Schema        string    `json:"schema"`
	RunID         RunID     `json:"run_id"`
	State         string    `json:"state"`
	Phase         string    `json:"phase,omitempty"`
	Outcome       Outcome   `json:"outcome,omitempty"`
	Recoverable   bool      `json:"recoverable"`
	StartedAt     time.Time `json:"started_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Result        *Result   `json:"result,omitempty"`
	CorruptReason string    `json:"corrupt_reason,omitempty"`
}

type Change struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Bytes  int64  `json:"bytes,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type CheckResult struct {
	Name       string        `json:"name"`
	Command    []string      `json:"command"`
	Directory  string        `json:"directory"`
	Required   bool          `json:"required"`
	Outcome    string        `json:"outcome"`
	ExitCode   int           `json:"exit_code"`
	Duration   time.Duration `json:"duration"`
	Stdout     string        `json:"stdout,omitempty"`
	Stderr     string        `json:"stderr,omitempty"`
	Truncated  bool          `json:"truncated,omitempty"`
	BeforeHash string        `json:"before_hash"`
	AfterHash  string        `json:"after_hash"`
}

type ReviewResult struct {
	Lens     string    `json:"lens"`
	Summary  string    `json:"summary"`
	Findings []Finding `json:"findings"`
}

type Finding struct {
	ID           string   `json:"id"`
	Lens         string   `json:"lens"`
	Severity     Severity `json:"severity"`
	Confidence   string   `json:"confidence"`
	Requirement  string   `json:"requirement,omitempty"`
	Location     string   `json:"location,omitempty"`
	Claim        string   `json:"claim"`
	Evidence     string   `json:"evidence"`
	Reproduction string   `json:"reproduction,omitempty"`
	Impact       string   `json:"impact,omitempty"`
	ProposedFix  string   `json:"proposed_fix,omitempty"`
}

type FindingRecord struct {
	Finding      Finding              `json:"finding"`
	FirstRound   int                  `json:"first_round"`
	LastRound    int                  `json:"last_round"`
	Dispositions []FindingDisposition `json:"dispositions"`
}

type FindingDisposition string

const (
	FindingConfirmed  FindingDisposition = "confirmed"
	FindingRepaired   FindingDisposition = "repaired"
	FindingRejected   FindingDisposition = "rejected"
	FindingUnresolved FindingDisposition = "unresolved"
)

type Severity string

const (
	SeverityAdvisory Severity = "advisory"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)
