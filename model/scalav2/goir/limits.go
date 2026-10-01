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
// Needed is the exact count; with Overflow the count is more than an int64 holds, Needed is
// math.MaxInt64, and no int64 ceiling admits the work.
type LimitError struct {
	// Machine is the machine whose interpretation needed the work, or empty outside one.
	// At admission it is also the composition whose class keys needed it.
	Machine string
	// Resource is "members" for one catalog or "classes" for a machine's classes of every action,
	// both under the Members ceiling, or "evaluations" for its state and class pairs.
	Resource string
	Ceiling  int64
	Needed   int64
	Overflow bool
}

func (e *LimitError) Error() string {
	what := "the catalog"
	if e.Machine != "" {
		what = e.Machine
	}
	if e.Overflow {
		return fmt.Sprintf("%s needs more than %d %s, above the ceiling of %d", what, e.Needed, e.Resource, e.Ceiling)
	}
	return fmt.Sprintf("%s needs %d %s, above the ceiling of %d", what, e.Needed, e.Resource, e.Ceiling)
}

// BuildWithin is Build within explicit ceilings.
func BuildWithin(m *modelirspb.Model, c Ceilings) (map[string]*Machine, error) {
	in := NewInterpreter(m)
	in.ceilings = c
	return in.build(m)
}

// count is how many values a catalog, or how much work, has: exactly, or, past what an int64 holds,
// only that it overflows.
type count struct {
	n        int64
	overflow bool
}

var overflowed = count{n: math.MaxInt64, overflow: true}

func counted(n *big.Int) count {
	if !n.IsInt64() {
		return overflowed
	}
	return count{n: n.Int64()}
}

// times multiplies two counts; a product with an exact zero is zero, however large the other.
func (c count) times(o count) count {
	switch {
	case (c.n == 0 && !c.overflow) || (o.n == 0 && !o.overflow):
		return count{}
	case c.overflow || o.overflow:
		return overflowed
	default:
		return counted(new(big.Int).Mul(big.NewInt(c.n), big.NewInt(o.n)))
	}
}

func (c count) plus(o count) count {
	if c.overflow || o.overflow {
		return overflowed
	}
	return counted(new(big.Int).Add(big.NewInt(c.n), big.NewInt(o.n)))
}

func (in *Interpreter) within(resource string, ceiling int64, needed count) error {
	if needed.overflow || needed.n > ceiling {
		return &LimitError{Resource: resource, Ceiling: ceiling, Needed: needed.n, Overflow: needed.overflow}
	}
	return nil
}

// size counts a finite type's catalog without listing it.
// A type or channel whose catalog contains itself has none, and is refused rather than followed.
func (in *Interpreter) size(t *modelirspb.TypeRef) (count, error) {
	switch r := t.GetRef().(type) {
	case *modelirspb.TypeRef_Bool:
		return count{n: 2}, nil
	case *modelirspb.TypeRef_IntRange:
		if r.IntRange.GetHigh() < r.IntRange.GetLow() {
			return count{}, nil
		}
		return counted(new(big.Int).Add(new(big.Int).Sub(big.NewInt(r.IntRange.GetHigh()), big.NewInt(r.IntRange.GetLow())), big.NewInt(1))), nil
	case *modelirspb.TypeRef_Named:
		leave, err := in.enter("type " + r.Named)
		if err != nil {
			return count{}, err
		}
		defer leave()
		return in.sizeOfDeclared(r.Named)
	case *modelirspb.TypeRef_Channel:
		leave, err := in.enter("channel " + r.Channel)
		if err != nil {
			return count{}, err
		}
		defer leave()
		c, err := in.channel(r.Channel, nil)
		if err != nil {
			return count{}, err
		}
		messages, err := in.size(c.GetMessage())
		if err != nil {
			return count{}, err
		}
		entries := messages.times(count{n: int64(c.GetDuplicates()) + 1})
		return channelSize(entries, int64(c.GetCapacity()), c.GetOrder() == modelirspb.Channel_ORDER_UNORDERED), nil
	default:
		return count{}, &Error{Message: "a finite type is a named type, the Booleans, an integer range, or a channel's contents"}
	}
}

func (in *Interpreter) sizeOfDeclared(name string) (count, error) {
	decl, ok := in.types[name]
	if !ok {
		return count{}, &Error{Message: "no type " + name}
	}
	switch s := decl.GetShape().(type) {
	case *modelirspb.Type_Enum:
		var n count
		for _, c := range s.Enum.GetCases() {
			k, err := in.sizeOfProduct(c.GetFields())
			if err != nil {
				return count{}, err
			}
			n = n.plus(k)
		}
		return n, nil
	case *modelirspb.Type_Record:
		return in.sizeOfProduct(s.Record.GetFields())
	default:
		return count{}, errorAt(decl.GetPosition(), "type %s has no shape", name)
	}
}

// enter marks a type or channel as being sized, and refuses one already being sized: its catalog
// would contain itself.
func (in *Interpreter) enter(catalog string) (func(), error) {
	if in.sizing[catalog] {
		return nil, &Error{Message: "the catalog of " + catalog + " contains itself"}
	}
	in.sizing[catalog] = true
	return func() { delete(in.sizing, catalog) }, nil
}

// channelSize counts the lists of at most n of e entries in closed form: e⁰ + … + eⁿ of them in
// order, and C(e+n, n) multisets unordered. Either is at least 2^64 once both e ≥ 2 and n ≥ 64.
func channelSize(e count, n int64, unordered bool) count {
	switch {
	case n <= 0:
		return count{n: 1}
	case e.overflow:
		return overflowed
	case e.n <= 1:
		return count{n: e.n*n + 1}
	case unordered && (min(e.n, n) >= 64 || e.n > math.MaxInt64-n), !unordered && n >= 64:
		return overflowed
	case unordered:
		return counted(new(big.Int).Binomial(e.n+n, n))
	default:
		power := new(big.Int).Exp(big.NewInt(e.n), big.NewInt(n+1), nil)
		return counted(power.Sub(power, big.NewInt(1)).Div(power, big.NewInt(e.n-1)))
	}
}

func (in *Interpreter) sizeOfProduct(fields []*modelirspb.Field) (count, error) {
	n := count{n: 1}
	for _, f := range fields {
		k, err := in.size(f.GetType())
		if err != nil {
			return count{}, err
		}
		n = n.times(k)
	}
	return n, nil
}
