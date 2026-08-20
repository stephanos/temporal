package protocol

type ClaimKind string

const (
	ClaimProved         ClaimKind = "proved"
	ClaimBoundedSafe    ClaimKind = "bounded-safe"
	ClaimCounterexample ClaimKind = "counterexample"
	ClaimConforming     ClaimKind = "conforming"
	ClaimViolating      ClaimKind = "violating"
	ClaimUnsupported    ClaimKind = "unsupported"
	ClaimInconclusive   ClaimKind = "inconclusive"
)
