package check

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// groupingModel reads the production form that the source-derived grouping spike became.
func groupingModel(t *testing.T, form string) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", form+".json"))
	require.NoError(t, err)
	require.NoError(t, ir.Validate(m))
	return m
}

func TestSharedNexusProductSpike(t *testing.T) {
	for _, form := range []string{"workflow", "standalone"} {
		t.Run(form, func(t *testing.T) {
			m := groupingModel(t, "nexus-"+form)
			machines := built(t, m)
			product, system := machines["nexusProduct"], machines["nexusSystem"]
			require.Equal(t, "operation", product.Decl.GetEntity())
			require.Equal(t, "operation", system.Decl.GetEntity())
			require.Equal(t, []string{"accepted", "rejected-notFound", "rejected-alreadyExists", "rejected-failedPrecondition", "rejected-invalidArgument"}, product.Table.Outcomes)
			rows, err := refinementOf(t, m, "nexusSystem")
			require.NoError(t, err)
			require.Len(t, rows, len(system.Table.Rows))
			carriers := map[string]string{}
			for _, r := range rows {
				if r.Product != nil {
					carriers[r.Key] = *r.Product
				}
			}
			if form == "workflow" {
				for key, carrier := range map[string]string{
					"scheduled-0-unset-unset-unset-reply-async":         "reply-async",
					"scheduled-0-unset-unset-unset-reply-syncSuccess":   "reply-syncSuccess",
					"backingOff-1-unset-unset-unset-complete-succeeded": "complete-succeeded",
					"started-0-unset-unset-unset-complete-failed":       "complete-failed",
					"started-0-unset-unset-expires-startToClose":        "timeout",
				} {
					require.Equal(t, carrier, carriers[key], key)
				}
				for _, key := range []string{
					"unscheduled-0-unset-unset-unset-schedule-unset-unset-unset",
					"scheduled-0-unset-unset-unset-reply-handlerError-true",
					"backingOff-1-unset-unset-unset-backoff",
					"scheduled-0-unset-unset-unset-stop",
				} {
					require.Empty(t, carriers[key], key)
				}
			} else {
				for key, carrier := range map[string]string{
					"scheduled-false-reply-async":             "reply-async",
					"scheduled-false-reply-syncSuccess":       "reply-syncSuccess",
					"scheduled-false-reply-operationFailed":   "reply-operationFailed",
					"scheduled-false-reply-operationCanceled": "reply-operationCanceled",
					"started-false-complete-succeeded":        "complete-succeeded",
					"started-false-complete-failed":           "complete-failed",
					"started-false-complete-canceled":         "complete-canceled",
					"scheduled-false-terminate":               "terminate",
				} {
					require.Equal(t, carrier, carriers[key], key)
				}
				for _, state := range system.Table.States {
					for _, class := range []string{"reply-handlerError-false", "reply-handlerError-true"} {
						require.True(t, disabled(system, state, class), state+"-"+class)
					}
				}
				for _, key := range []string{"unstarted-false-start", "scheduled-false-requestCancel", "terminated-false-terminate", "succeeded-true-requestCancel"} {
					require.Empty(t, carriers[key], key)
				}
				require.Equal(t, "rejected-failedPrecondition", row(t, system, "succeeded-false-terminate").Results[0].Outcome)
				require.Equal(t, "accepted", row(t, system, "terminated-false-terminate").Results[0].Outcome)
			}

			// Each source form exports a Query that checks the same product Property through its map.
			checked := receiptOf(t, Check(m, DefaultScope), "query nexusSystem terminalHolds")
			require.Equal(t, Verified, checked.Kind, checked.Explanation)
			require.True(t, checked.Exercised)
			require.Equal(t, "temporal.features.nexus.product", checked.Property.Family)
		})
	}
}

func TestSharedProductCatalogPrecedesOutcomeVisibility(t *testing.T) {
	m := groupingModel(t, "nexus-standalone")
	product := admMachine(m, "nexusProduct")
	outcome := proto.Clone(admType(m, product.GetOutcomeType())).(*umpirespb.Type)
	outcome.Name = "fixture.productOnlyOutcome"
	var rejectedCase *umpirespb.Case
	for _, c := range outcome.GetEnum().Cases {
		if c.GetName() == "rejected" {
			rejectedCase = c
		}
	}
	require.NotNil(t, rejectedCase)
	require.Len(t, rejectedCase.GetFields(), 1)
	rejectionType := rejectedCase.GetFields()[0].GetType().GetNamed()
	require.NotEmpty(t, rejectionType)
	rejection := proto.Clone(admType(m, rejectionType)).(*umpirespb.Type)
	rejection.Name = "fixture.productOnlyRejection"
	rejection.GetEnum().Cases = slices.DeleteFunc(rejection.GetEnum().Cases, func(c *umpirespb.Case) bool { return c.GetName() == "failedPrecondition" })
	rejectedCase.GetFields()[0].Type.Ref = &umpirespb.TypeRef_Named{Named: rejection.Name}
	m.Types = append(m.Types, rejection, outcome)
	for _, f := range m.GetFunctions() {
		if strings.HasPrefix(f.GetName(), "temporal.features.nexus.product.") {
			encoded, err := protojson.Marshal(f)
			require.NoError(t, err)
			encoded = bytes.ReplaceAll(encoded, []byte(`"`+product.GetOutcomeType()+`"`), []byte(`"`+outcome.Name+`"`))
			encoded = bytes.ReplaceAll(encoded, []byte(`"`+rejectionType+`"`), []byte(`"`+rejection.Name+`"`))
			require.NoError(t, protojson.Unmarshal(encoded, f))
		}
	}
	product.OutcomeType = outcome.Name
	require.False(t, function(m, admMachine(m, "nexusSystem").GetRefines().GetVisibleOutcomes()).GetBody().GetLiteral().GetBool(), "the missing source outcome is excluded from stutter observations")
	_, err := refinementOf(t, m, "nexusSystem")
	require.ErrorContains(t, err, "which is no fixture.productOnlyOutcome")
}

func TestSharedProductRejectsMissingAndWrongCarriers(t *testing.T) {
	for _, effect := range []string{"start", "succeed", "fail", "cancel", "terminate"} {
		t.Run("missing fact "+effect, func(t *testing.T) {
			m := groupingModel(t, "nexus-standalone")
			f := function(m, "temporal.features.nexus.product.NexusProduct$.effects$."+effect)
			f.Body.GetList().Items[0].GetConstruct().Args[2].GetList().Items = nil
			_, err := refinementOf(t, m, "nexusSystem")
			require.ErrorContains(t, err, "nexusSystem refines nexusProduct: the row")
		})
	}
	for name, change := range map[string]func(*umpirespb.Model){
		"missing termination action": func(m *umpirespb.Model) {
			p := admMachine(m, "nexusProduct")
			p.Steps = slices.DeleteFunc(p.Steps, func(b *umpirespb.StepBinding) bool {
				return b.GetAction() == "temporal.features.nexus.client.terminate"
			})
		},
		"wrong termination fact": func(m *umpirespb.Model) {
			f := function(m, "temporal.features.nexus.product.NexusProduct$.effects$.terminate")
			f.Body.GetList().Items[0].GetConstruct().Args[2].GetList().Items[0].GetLiteral().GetEnum().Case = "nexusOperationCanceled"
		},
		"observed termination hidden as cancellation": func(m *umpirespb.Model) {
			f := function(m, admMachine(m, "nexusSystem").GetRefines().GetMap())
			cases := f.Body.GetMatch().Cases
			cases[len(cases)-1].Body.GetConstruct().Args[0].GetLiteral().GetEnum().Case = "canceled"
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := groupingModel(t, "nexus-standalone")
			change(m)
			_, err := refinementOf(t, m, "nexusSystem")
			require.ErrorContains(t, err, "nexusSystem refines nexusProduct: the row")
		})
	}
}
