package goir

import (
	"fmt"
	"math"
	"math/big"

	modelirspb "go.temporal.io/server/api/modelir/v1"
)

// Ceilings bound the work interpreting a Model may take, each checked against a count of the work
// made before any of it is done.
type Ceilings struct {
	// Members is the most values one catalog may list.
	Members int64
	// Evaluations is the most state and class pairs one machine's rows may evaluate.
	Evaluations int64
}

// DefaultCeilings are the ceilings Build and NewInterpreter interpret a Model within.
var DefaultCeilings = Ceilings{Members: 1 << 16, Evaluations: 1 << 20}

// LimitError is a ceiling some work would exceed, with the work it needs, counted before any of it is
// done and saturating at math.MaxInt64. It says nothing about the Model but that it was not
// interpreted.
type LimitError struct {
	// Machine is the machine whose interpretation needed the work, or empty outside one.
	Machine  string
	Resource string
	Ceiling  int64
	Needed   int64
}

func (e *LimitError) Error() string {
	what := "the catalog"
	if e.Machine != "" {
		what = e.Machine
	}
	return fmt.Sprintf("%s needs %d %s, above the ceiling of %d", what, e.Needed, e.Resource, e.Ceiling)
}

// BuildWithin is Build within explicit ceilings.
func BuildWithin(m *modelirspb.Model, c Ceilings) (map[string]*Machine, error) {
	in := NewInterpreter(m)
	in.ceilings = c
	return in.build(m)
}

func (in *Interpreter) within(resource string, ceiling, needed int64) error {
	if needed > ceiling {
		return &LimitError{Resource: resource, Ceiling: ceiling, Needed: needed}
	}
	return nil
}

// size counts a finite type's catalog without listing it.
func (in *Interpreter) size(t *modelirspb.TypeRef) (int64, error) {
	switch r := t.GetRef().(type) {
	case *modelirspb.TypeRef_Bool:
		return 2, nil
	case *modelirspb.TypeRef_IntRange:
		if r.IntRange.GetHigh() < r.IntRange.GetLow() {
			return 0, nil
		}
		return saturate(new(big.Int).Add(new(big.Int).Sub(big.NewInt(r.IntRange.GetHigh()), big.NewInt(r.IntRange.GetLow())), big.NewInt(1))), nil
	case *modelirspb.TypeRef_Named:
		decl, ok := in.types[r.Named]
		if !ok {
			return 0, &Error{Message: "no type " + r.Named}
		}
		switch s := decl.GetShape().(type) {
		case *modelirspb.Type_Enum:
			var n int64
			for _, c := range s.Enum.GetCases() {
				k, err := in.sizeOfProduct(c.GetFields())
				if err != nil {
					return 0, err
				}
				n = add(n, k)
			}
			return n, nil
		case *modelirspb.Type_Record:
			return in.sizeOfProduct(s.Record.GetFields())
		default:
			return 0, errorAt(decl.GetPosition(), "type %s has no shape", r.Named)
		}
	case *modelirspb.TypeRef_Channel:
		c, err := in.channel(r.Channel, nil)
		if err != nil {
			return 0, err
		}
		messages, err := in.size(c.GetMessage())
		if err != nil {
			return 0, err
		}
		entries := saturate(new(big.Int).Mul(big.NewInt(messages), big.NewInt(int64(c.GetDuplicates())+1)))
		return channelSize(entries, int64(c.GetCapacity()), c.GetOrder() == modelirspb.Channel_ORDER_UNORDERED), nil
	default:
		return 0, &Error{Message: "a finite type is a named type, the Booleans, an integer range, or a channel's contents"}
	}
}

// channelSize counts the lists of at most n of e entries in closed form: e⁰ + … + eⁿ of them in
// order, and C(e+n, n) multisets unordered. Either is at least 2^64 once both e ≥ 2 and n ≥ 64.
func channelSize(e, n int64, unordered bool) int64 {
	switch {
	case e <= 1:
		return e*n + 1
	case unordered && (min(e, n) >= 64 || e > math.MaxInt64-n), !unordered && n >= 64:
		return math.MaxInt64
	case unordered:
		return saturate(new(big.Int).Binomial(e+n, n))
	default:
		power := new(big.Int).Exp(big.NewInt(e), big.NewInt(n+1), nil)
		return saturate(power.Sub(power, big.NewInt(1)).Div(power, big.NewInt(e-1)))
	}
}

func (in *Interpreter) sizeOfProduct(fields []*modelirspb.Field) (int64, error) {
	n := big.NewInt(1)
	for _, f := range fields {
		k, err := in.size(f.GetType())
		if err != nil {
			return 0, err
		}
		n.Mul(n, big.NewInt(k))
	}
	return saturate(n), nil
}

func add(a, b int64) int64 { return saturate(new(big.Int).Add(big.NewInt(a), big.NewInt(b))) }

func saturate(n *big.Int) int64 {
	if !n.IsInt64() {
		return math.MaxInt64
	}
	return n.Int64()
}
