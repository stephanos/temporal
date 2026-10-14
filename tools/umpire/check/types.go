package check

import core "go.temporal.io/server/tools/umpire/internal/engine"

type (
	TraceStep         = core.TraceStep
	Trace             = core.Trace
	Answer            = core.Answer
	Limits            = core.Limits
	Query             = core.Query
	QueryForm         = core.QueryForm
	Outcome           = core.Outcome
	PropertyDecl      = core.PropertyDecl
	ScenarioDecl      = core.ScenarioDecl
	Model             = core.Model
	ComposeCeiling    = core.ComposeCeiling
	ComposeLimitError = core.ComposeLimitError
	RefinementRow     = core.RefinementRow
	RefinementError   = core.RefinementError
	RefinementFailure = core.RefinementFailure
	MonitorVerdict    = core.MonitorVerdict
	Verdict           = core.Verdict
	ProgressKind      = core.ProgressKind
	ProgressVerdict   = core.ProgressVerdict
	Assumption        = core.Assumption
	Group             = core.Group
	Requirement       = core.Requirement
	RequirementKind   = core.RequirementKind
	UnknownReach      = core.UnknownReach
	UnknownKind       = core.UnknownKind
	Monitor           = core.Monitor
	Evaluation        = core.Evaluation
)

const (
	StateRequirement         = core.StateRequirement
	OutcomeRequirement       = core.OutcomeRequirement
	FactRequirement          = core.FactRequirement
	FindForm                 = core.FindForm
	VerifyForm               = core.VerifyForm
	VerifiedWithinLimits     = core.VerifiedWithinLimits
	CounterexampleFound      = core.CounterexampleFound
	UnknownRow               = core.UnknownRow
	UnknownClaim             = core.UnknownClaim
	MonitorHeld              = core.MonitorHeld
	MonitorViolated          = core.MonitorViolated
	MonitorUnread            = core.MonitorUnread
	MonitorUnknown           = core.MonitorUnknown
	DeadlockKind             = core.DeadlockKind
	CycleKind                = core.CycleKind
	DeadlineKind             = core.DeadlineKind
	RefinementCatalog        = core.RefinementCatalog
	RefinementInitial        = core.RefinementInitial
	RefinementUnmatched      = core.RefinementUnmatched
	RefinementVisibleStutter = core.RefinementVisibleStutter
	RefinementIncomplete     = core.RefinementIncomplete
)

func Fingerprint(canonical string) string { return core.Fingerprint(canonical) }

func Quote(value string) string { return core.Quote(value) }
