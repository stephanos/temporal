package model

import (
	"fmt"
	"math"
	"math/big"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// Ceilings bound the work interpreting a Model may take, each checked against a count of the work
// made before any of it is done.
type Ceilings struct {
	// Members is the most values one catalog may list.
	Members int64
	// Evaluations is the most state and class pairs one machine's rows may evaluate.
	Evaluations int64
}

// defaultCeilings are the ceilings Build and NewInterpreter interpret a Model within.
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

// count is how many values a catalog, or how much work, has: exactly, or, past what an int64 holds,
// only that it overflows.
type Count struct {
	n        int64
	overflow bool
}

func CountOf(n int64) Count { return Count{n: n} }

// Int64 returns the count and whether it fits an int64.
func (c Count) Int64() (int64, bool) { return c.n, !c.overflow }

var overflowed = Count{n: math.MaxInt64, overflow: true}

func counted(n *big.Int) Count {
	if !n.IsInt64() {
		return overflowed
	}
	return Count{n: n.Int64()}
}

// times multiplies two counts; a product with an exact zero is zero, however large the other.
func (c Count) Times(o Count) Count {
	switch {
	case (c.n == 0 && !c.overflow) || (o.n == 0 && !o.overflow):
		return Count{}
	case c.overflow || o.overflow:
		return overflowed
	default:
		return counted(new(big.Int).Mul(big.NewInt(c.n), big.NewInt(o.n)))
	}
}

func (c Count) Plus(o Count) Count {
	if c.overflow || o.overflow {
		return overflowed
	}
	return counted(new(big.Int).Add(big.NewInt(c.n), big.NewInt(o.n)))
}

func (in *Interpreter) Within(resource string, ceiling int64, needed Count) error {
	if needed.overflow || needed.n > ceiling {
		return &LimitError{Resource: resource, Ceiling: ceiling, Needed: needed.n, Overflow: needed.overflow}
	}
	return nil
}

// size counts a finite type's catalog without listing it.
// A type or channel whose catalog contains itself has none, and is refused rather than followed.
func (in *Interpreter) Size(t *umpirespb.TypeRef) (Count, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		return Count{n: 2}, nil
	case *umpirespb.TypeRef_IntRange:
		if r.IntRange.GetHigh() < r.IntRange.GetLow() {
			return Count{}, nil
		}
		return counted(new(big.Int).Add(new(big.Int).Sub(big.NewInt(r.IntRange.GetHigh()), big.NewInt(r.IntRange.GetLow())), big.NewInt(1))), nil
	case *umpirespb.TypeRef_Named:
		leave, err := in.enter("type " + r.Named)
		if err != nil {
			return Count{}, err
		}
		defer leave()
		return in.sizeOfDeclared(r.Named)
	case *umpirespb.TypeRef_Channel:
		leave, err := in.enter("channel " + r.Channel)
		if err != nil {
			return Count{}, err
		}
		defer leave()
		c, err := in.channel(r.Channel, nil)
		if err != nil {
			return Count{}, err
		}
		messages, err := in.Size(c.GetMessage())
		if err != nil {
			return Count{}, err
		}
		entries := messages.Times(Count{n: int64(c.GetDuplicates()) + 1})
		return channelSize(entries, int64(c.GetCapacity()), c.GetOrder() == umpirespb.Channel_ORDER_UNORDERED), nil
	default:
		return Count{}, &Error{Message: "a finite type is a named type, the Booleans, an integer range, or a channel's contents"}
	}
}

func (in *Interpreter) sizeOfDeclared(name string) (Count, error) {
	decl, ok := in.types[name]
	if !ok {
		return Count{}, &Error{Message: "no type " + name}
	}
	switch s := decl.GetShape().(type) {
	case *umpirespb.Type_Enum:
		var n Count
		for _, c := range s.Enum.GetCases() {
			k, err := in.SizeOfProduct(c.GetFields())
			if err != nil {
				return Count{}, err
			}
			n = n.Plus(k)
		}
		return n, nil
	case *umpirespb.Type_Record:
		return in.SizeOfProduct(s.Record.GetFields())
	default:
		return Count{}, ErrorAt(decl.GetPosition(), "type %s has no shape", name)
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
func channelSize(e Count, n int64, unordered bool) Count {
	switch {
	case n <= 0:
		return Count{n: 1}
	case e.overflow:
		return overflowed
	case e.n <= 1:
		return Count{n: e.n*n + 1}
	case unordered && (min(e.n, n) >= 64 || e.n > math.MaxInt64-n), !unordered && n >= 64:
		return overflowed
	case unordered:
		return counted(new(big.Int).Binomial(e.n+n, n))
	default:
		power := new(big.Int).Exp(big.NewInt(e.n), big.NewInt(n+1), nil)
		return counted(power.Sub(power, big.NewInt(1)).Div(power, big.NewInt(e.n-1)))
	}
}

func (in *Interpreter) SizeOfProduct(fields []*umpirespb.Field) (Count, error) {
	n := Count{n: 1}
	for _, f := range fields {
		k, err := in.Size(f.GetType())
		if err != nil {
			return Count{}, err
		}
		n = n.Times(k)
	}
	return n, nil
}
