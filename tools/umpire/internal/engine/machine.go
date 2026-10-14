package engine

// Model is anything with a finite table: a declared, derived or composed machine.
type Model interface {
	Name() string
	Table() (*Table, error)
}

// compareStrings orders keys the way Lean's `String` order does: by code point, which for the
// ASCII keys models use is Go's byte order.
func compareStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
