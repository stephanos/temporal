package runtimeengine

import umpireruntime "go.temporal.io/server/tools/umpire/runtime"

var (
	CanonicalPhaseLimits      = umpireruntime.CanonicalPhaseLimits
	CheckRequest              = umpireruntime.CheckRequest
	NewAuthority              = umpireruntime.NewAuthority
	NewControlReceipt         = umpireruntime.NewControlReceipt
	NewFact                   = umpireruntime.NewFact
	NewFactField              = umpireruntime.NewFactField
	NewHistoryCapacityReceipt = umpireruntime.NewHistoryCapacityReceipt
	NewOccurrence             = umpireruntime.NewOccurrence
	NewProgram                = umpireruntime.NewProgram
	NewReceipt                = umpireruntime.NewReceipt
	NewResource               = umpireruntime.NewResource
)
