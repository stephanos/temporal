package lower

// A Model written by hand, or changed after it was lifted, is answered with an error or a result and
// never with a panic. These tests change the lifted fixtures at random, in ways the schema allows and
// the lifter would never write, and put each through admission and lowering.

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// site is one field of one message of a tree.
type site struct {
	message protoreflect.Message
	field   protoreflect.FieldDescriptor
}

// sites lists every field of a message tree, set or not, and the texts the tree holds.
func sites(m protoreflect.Message, out *[]site, texts *[]string) {
	fields := m.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		*out = append(*out, site{m, f})
		if !m.Has(f) {
			continue
		}
		switch {
		case f.IsList() && f.Kind() == protoreflect.MessageKind:
			list := m.Get(f).List()
			for j := range list.Len() {
				sites(list.Get(j).Message(), out, texts)
			}
		case f.IsList() && f.Kind() == protoreflect.StringKind:
			list := m.Get(f).List()
			for j := range list.Len() {
				*texts = append(*texts, list.Get(j).String())
			}
		case f.Kind() == protoreflect.MessageKind:
			sites(m.Get(f).Message(), out, texts)
		case f.Kind() == protoreflect.StringKind:
			*texts = append(*texts, m.Get(f).String())
		default:
		}
	}
}

// change changes one field at random and says what it did: a text becomes empty or another text of
// the tree, a number becomes zero, negative or large, an enum any number near its values, a flag
// flips, a message is cleared, and a list loses, repeats or swaps an element.
func change(rng *rand.Rand, roots []protoreflect.Message) string {
	var all []site
	texts := []string{"", "nothing.declares.this"}
	for _, root := range roots {
		sites(root, &all, &texts)
	}
	s := all[rng.IntN(len(all))]
	m, f := s.message, s.field
	name := string(m.Descriptor().Name()) + "." + string(f.Name())
	if f.IsList() {
		list := m.Mutable(f).List()
		switch n := list.Len(); {
		case n == 0:
			return name + ": an empty list, unchanged"
		case rng.IntN(4) == 0:
			m.Clear(f)
			return name + ": cleared"
		case rng.IntN(3) == 0:
			i := rng.IntN(n)
			list.Append(list.Get(i))
			return fmt.Sprintf("%s: element %d repeated", name, i)
		case rng.IntN(2) == 0 && n > 1:
			i, j := rng.IntN(n), rng.IntN(n)
			a, b := list.Get(i), list.Get(j)
			if f.Kind() == protoreflect.MessageKind {
				a, b = protoreflect.ValueOfMessage(proto.Clone(a.Message().Interface()).ProtoReflect()),
					protoreflect.ValueOfMessage(proto.Clone(b.Message().Interface()).ProtoReflect())
			}
			list.Set(i, b)
			list.Set(j, a)
			return fmt.Sprintf("%s: elements %d and %d swapped", name, i, j)
		default:
			list.Truncate(n - 1)
			return name + ": last element dropped"
		}
	}
	switch f.Kind() {
	case protoreflect.StringKind:
		text := texts[rng.IntN(len(texts))]
		m.Set(f, protoreflect.ValueOfString(text))
		return fmt.Sprintf("%s = %q", name, text)
	case protoreflect.Int64Kind:
		n := []int64{0, -1, 1, 1 << 40}[rng.IntN(4)]
		m.Set(f, protoreflect.ValueOfInt64(n))
		return fmt.Sprintf("%s = %d", name, n)
	case protoreflect.Int32Kind:
		n := []int32{0, -1, 1, 1 << 30}[rng.IntN(4)]
		m.Set(f, protoreflect.ValueOfInt32(n))
		return fmt.Sprintf("%s = %d", name, n)
	case protoreflect.BoolKind:
		m.Set(f, protoreflect.ValueOfBool(!m.Get(f).Bool()))
		return name + ": flipped"
	case protoreflect.EnumKind:
		n := protoreflect.EnumNumber(rng.IntN(f.Enum().Values().Len() + 2))
		m.Set(f, protoreflect.ValueOfEnum(n))
		return fmt.Sprintf("%s = %d", name, n)
	default:
		m.Clear(f)
		return name + ": cleared"
	}
}

// changed is a Model with up to three random changes to what a realization and its lowering read:
// the realizations, and with actions also the actions, whose examples lowering reads.
func changed(rng *rand.Rand, base *umpirespb.Model, actions bool) (*umpirespb.Model, []string) {
	m := proto.Clone(base).(*umpirespb.Model)
	var done []string
	for range 1 + rng.IntN(3) {
		var roots []protoreflect.Message
		for _, r := range m.GetRealizations() {
			roots = append(roots, r.ProtoReflect())
		}
		if actions {
			for _, a := range m.GetActions() {
				roots = append(roots, a.ProtoReflect())
			}
		}
		done = append(done, change(rng, roots))
	}
	return m, done
}

// finds is the find Queries of a Model.
func finds(m *umpirespb.Model) []string {
	var out []string
	for _, q := range m.GetQueries() {
		if q.GetForm() == umpirespb.Query_FORM_FIND {
			out = append(out, q.GetName())
		}
	}
	return out
}

// sample is how many changed Models a test tries: all of them, or a twentieth under -short, which
// keeps the same seed and so the first of the same changes.
func sample(iterations int) int {
	if testing.Short() {
		return max(iterations/20, 2)
	}
	return iterations
}

// answered runs one step of a random Model and fails with what was changed if it panics.
func answered(t *testing.T, seed uint64, iteration int, done []string, run func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			require.FailNowf(t, "a changed Model panicked", "seed %d, iteration %d, changes %q: %v", seed, iteration, done, r)
		}
	}()
	run()
}

func randomModels(t *testing.T) map[string]*umpirespb.Model {
	return map[string]*umpirespb.Model{"nexus-caller": loaded(t, "nexus-caller"), "realizations": liftedRealizations(t),
		"activity": loaded(t, "activity")}
}

// Admission answers every changed Model: it admits it or says what is wrong with it.
func TestAdmissionOfAChangedModelNeverPanics(t *testing.T) {
	const seed = 20261001
	iterations := sample(2000)
	for name, base := range randomModels(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel() // each Model has its own fixture, generator and seed
			rng := rand.New(rand.NewPCG(seed, 1))
			admitted := 0
			for i := range iterations {
				m, done := changed(rng, base, true)
				answered(t, seed, i, done, func() {
					if ir.Validate(m) == nil {
						admitted++
					}
				})
			}
			require.Positive(t, admitted, "some changes leave an admissible Model")
			require.Less(t, admitted, iterations, "some changes are rejected")
		})
	}
}

// Lowering answers every changed realization admission lets through: a Case, a standing, or an
// error. The machines, the claims and the answers of the unchanged Model are bound once; a changed
// realization changes none of them.
func TestLoweringAChangedRealizationNeverPanics(t *testing.T) {
	const seed = 20261001
	iterations := sample(2000)
	for name, base := range randomModels(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel() // each Model has its own fixture, producer, generator and seed
			bound, err := NewProducer(base)
			require.NoError(t, err)
			queries := finds(base)
			rng := rand.New(rand.NewPCG(seed, 2))
			standings := map[Standing]int{}
			errored, lowered := 0, 0
			for i := range iterations {
				m, done := changed(rng, base, false)
				answered(t, seed, i, done, func() {
					if ir.Validate(m) != nil {
						return
					}
					lowered++
					p := &Producer{realizer: bound.realizer, found: bound.found, realizations: m.GetRealizations()}
					query := queries[i%len(queries)]
					l, err := p.Lower(query, cp.IdentityFor("temporal.case", "random", query))
					if err != nil {
						errored++
						return
					}
					standings[l.Standing]++
				})
			}
			require.Positive(t, lowered, "some changed realizations are admitted")
			require.Positive(t, errored, "some admitted changes are refused when lowered")
			require.Positive(t, standings[Lowered]+standings[NotSupported], "some admitted changes still lower or name their gaps")
		})
	}
}

// The whole way from a changed Model to a Case, with nothing bound beforehand, panics nowhere either:
// the actions change too, so the claims a producer reads are read from what admission let through.
func TestProducingFromAChangedModelNeverPanics(t *testing.T) {
	const seed = 20261001
	iterations := sample(40)
	for name, base := range randomModels(t) {
		t.Run(name, func(t *testing.T) {
			rng := rand.New(rand.NewPCG(seed, 3))
			for i := range iterations {
				m, done := changed(rng, base, true)
				answered(t, seed, i, done, func() {
					p, err := NewProducer(m)
					if err != nil {
						return
					}
					for _, query := range finds(m) {
						_, _ = p.Lower(query, cp.IdentityFor("temporal.case", "random", query))
					}
				})
			}
		})
	}
}
