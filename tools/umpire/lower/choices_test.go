package lower

// Named choices are inert (model/SEMANTICS.md, Named choices): a Case lowered from a Model whose step
// records name their alternatives is the Case of the Model without the names, byte for byte.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// namedChoices is a copy of a Model that names the step records it constructs, each by a name of its
// own, and the number it named. A construct that a row evaluates twice, as a helper a step calls twice
// does, would name two of its results alike, which the reader refuses; such a construct stays unnamed.
func namedChoices(t *testing.T, m *umpirespb.Model) (*umpirespb.Model, int) {
	t.Helper()
	out := proto.CloneOf(m)
	var steps []*umpirespb.Construct
	var visit func(protoreflect.Message)
	visit = func(msg protoreflect.Message) {
		if c, ok := msg.Interface().(*umpirespb.Construct); ok && c.GetType() == umpiremodel.StepType {
			c.Choice = fmt.Sprintf("alternative-%d", len(steps))
			steps = append(steps, c)
		}
		msg.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			switch {
			case f.Message() == nil:
			case f.IsList():
				for i := range v.List().Len() {
					visit(v.List().Get(i).Message())
				}
			default:
				visit(v.Message())
			}
			return true
		})
	}
	visit(out.ProtoReflect())
	named := len(steps)
	twice := regexp.MustCompile(`has two results named alternative-(\d+)$`)
	for {
		_, err := umpiremodel.Build(out)
		if err == nil {
			return out, named
		}
		match := twice.FindStringSubmatch(err.Error())
		require.NotNil(t, match, "%v", err)
		i, err := strconv.Atoi(match[1])
		require.NoError(t, err)
		steps[i].Choice = ""
		named--
	}
}

// loweredAll is every Query of a Model lowered, as its standing, its Case's bytes, its inventory and
// gaps, or its error; and how many Cases it lowered.
func loweredAll(t *testing.T, m *umpirespb.Model) (out []string, cases int) {
	t.Helper()
	p, err := NewProducer(m)
	require.NoError(t, err)
	for _, q := range m.GetQueries() {
		l, err := p.Lower(q.GetName(), IdentityFor("temporal.case", "choices", q.GetName()))
		if err != nil {
			out = append(out, q.GetName()+": "+err.Error())
			continue
		}
		encoded, err := protojson.MarshalOptions{}.Marshal(l.Case)
		require.NoError(t, err)
		out = append(out, fmt.Sprintf("%s: %s %s %+v %+v %+v", q.GetName(), l.Standing, encoded, l.Inventory, l.Unsupported, l.OffPath))
		if l.Standing == Lowered {
			cases++
		}
	}
	return out, cases
}

// TestNamedChoicesLowerTheSameCases names every step record of each Model with a realization and
// lowers each of its Queries: the standings, the Cases' bytes, their inventories and gaps, and the
// errors are the unnamed Model's. activity-race.json branches: one of its rows has several results,
// each now named.
func TestNamedChoicesLowerTheSameCases(t *testing.T) {
	branches := map[string]bool{"model/ir/activity-race.json": true}
	for _, path := range []string{
		"model/ir/activity.json", "model/ir/activity-race.json", "model/ir/nexus-caller.json", "model/ir/nexus-control.json",
		"model/irgen/testdata/lifts/expected/realizations.json",
	} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			plain, err := umpiremodel.Load(filepath.Join("..", "..", "..", path))
			require.NoError(t, err)
			named, n := namedChoices(t, plain)
			require.Positive(t, n)
			require.NoError(t, umpiremodel.Validate(named))
			machines, err := umpiremodel.Build(named)
			require.NoError(t, err)
			reported, branching := 0, 0
			for _, mm := range machines {
				for _, row := range mm.Table.Rows {
					named := 0
					for _, res := range row.Results {
						if res.Choice != "" {
							named++
						}
					}
					reported += named
					if named > 1 {
						branching++
					}
				}
			}
			require.Positive(t, reported, "a result of %s is named", path)
			if branches[path] {
				require.Positive(t, branching, "a row of %s has several named results", path)
			}
			was, cases := loweredAll(t, plain)
			is, _ := loweredAll(t, named)
			require.Equal(t, was, is)
			require.Positive(t, cases, "a Query of %s is lowered to a Case", path)
		})
	}
}
