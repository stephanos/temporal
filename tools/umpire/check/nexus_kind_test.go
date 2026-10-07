package check

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/ir"
)

func TestBothProductionNexusFormsRefineKindProduct(t *testing.T) {
	for _, form := range []string{"workflow", "standalone"} {
		t.Run(form, func(t *testing.T) {
			m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "nexus-"+form+".json"))
			require.NoError(t, err)
			machines := built(t, m)
			product, system := machines["nexusProduct"], machines["nexusSystem"]
			require.NotNil(t, product)
			require.NotNil(t, system)
			require.Equal(t, "temporal.features.nexus.product", product.Decl.GetFamily())
			require.Equal(t, []string{"accepted", "rejected-notFound", "rejected-alreadyExists", "rejected-failedPrecondition", "rejected-invalidArgument"}, product.Table.Outcomes)
			rows, err := refinementOf(t, m, "nexusSystem")
			require.NoError(t, err)
			require.Len(t, rows, len(system.Table.Rows))
			claim := receiptOf(t, Check(m, DefaultScope), "query nexusSystem terminalHolds")
			require.Equal(t, Verified, claim.Kind, claim.Explanation)
			require.True(t, claim.Exercised)
			require.Equal(t, ClaimKey{Family: "temporal.features.nexus.product", Owner: "nexusProduct", Name: "terminalIsFinal"}, claim.Property)
			for _, state := range system.Table.States {
				if form == "standalone" {
					for _, class := range []string{"reply-handlerError-false", "reply-handlerError-true"} {
						require.True(t, disabled(system, state, class), state+"-"+class)
					}
				}
			}
		})
	}
}
