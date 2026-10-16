//go:build !gomad && !gomad_wasm_slim

package temporal

import (
	"go.temporal.io/server/common/membership/ringpop"
	"go.uber.org/fx"
)

func defaultMembershipModule() fx.Option {
	return ringpop.MembershipModule
}
