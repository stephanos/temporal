package record

// SamePreparedTargetIdentity compares provenance before execution; the full
// target identity continues to bind any retained execution evidence.
func SamePreparedTargetIdentity(left, right Target) (bool, error) {
	left.Backend = CloneBackendMetadata(left.Backend)
	right.Backend = CloneBackendMetadata(right.Backend)
	if left.Backend != nil {
		left.Backend.Evidence = nil
	}
	if right.Backend != nil {
		right.Backend.Evidence = nil
	}
	return SameTargetIdentity(left, right)
}
