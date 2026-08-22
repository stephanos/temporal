package experiment

type ActionOutcome string

const (
	ActionOutcomeApplied          ActionOutcome = "applied"
	ActionOutcomeSuppressed       ActionOutcome = "suppressed"
	ActionOutcomeRejected         ActionOutcome = "rejected"
	ActionOutcomeRetried          ActionOutcome = "retried"
	ActionOutcomeFaultIntercepted ActionOutcome = "fault-intercepted"
)

func validActionOutcome(outcome ActionOutcome) bool {
	switch outcome {
	case ActionOutcomeApplied, ActionOutcomeSuppressed, ActionOutcomeRejected,
		ActionOutcomeRetried, ActionOutcomeFaultIntercepted:
		return true
	default:
		return false
	}
}
