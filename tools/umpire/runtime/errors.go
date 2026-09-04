package runtime

// PreflightErrorKind is one closed, deterministic request-rejection category.
type PreflightErrorKind string

const (
	PreflightInputSet      PreflightErrorKind = "input-set"
	PreflightProfile       PreflightErrorKind = "profile"
	PreflightConfiguration PreflightErrorKind = "configuration"
	PreflightTarget        PreflightErrorKind = "target"
	PreflightAction        PreflightErrorKind = "action"
	PreflightFault         PreflightErrorKind = "fault"
	PreflightOccurrence    PreflightErrorKind = "occurrence"
	PreflightParticipant   PreflightErrorKind = "participant"
	PreflightProtocol      PreflightErrorKind = "protocol"
	PreflightCapability    PreflightErrorKind = "capability"
	PreflightBudget        PreflightErrorKind = "budget"
	PreflightRunIdentity   PreflightErrorKind = "run-identity"
	PreflightSeed          PreflightErrorKind = "seed"
	PreflightAttempt       PreflightErrorKind = "attempt"
	PreflightDuplicate     PreflightErrorKind = "duplicate"
)

// PreflightError reports only a closed kind and sanitized subject.
type PreflightError struct {
	kind    PreflightErrorKind
	subject string
}

func (e *PreflightError) Error() string {
	if e == nil {
		return ""
	}
	if e.subject == "" {
		return string(e.kind)
	}
	return string(e.kind) + ": " + e.subject
}

// Kind returns the stable rejection category.
func (e *PreflightError) Kind() PreflightErrorKind {
	if e == nil {
		return ""
	}
	return e.kind
}

// Subject returns the bounded identifier naming the rejected contract field.
func (e *PreflightError) Subject() string {
	if e == nil {
		return ""
	}
	return e.subject
}

func (e *PreflightError) Is(target error) bool {
	other, ok := target.(*PreflightError)
	return ok && e != nil && other != nil && e.kind == other.kind
}

func preflightError(kind PreflightErrorKind, subject string) error {
	return &PreflightError{kind: kind, subject: boundedSubject(subject)}
}

func boundedSubject(subject string) string {
	if len(subject) <= MaximumIdentityBytes {
		return subject
	}
	return subject[:MaximumIdentityBytes]
}
