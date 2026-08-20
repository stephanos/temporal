package tests

import (
	"testing"
)

func TestUmpire3SparseRegressionOrdinaryNexusCompletion(t *testing.T) {
	runUmpire3Behavior(t, "SparseRegressionOrdinaryNexusCompletion", "")
}

func TestUmpire3SparseRegressionCompletionBeforeStartResponse(t *testing.T) {
	runUmpire3Behavior(t, "SparseRegressionCompletionBeforeStartResponse", "")
}

func TestUmpire3SparseRegressionCancellationRetry(t *testing.T) {
	runUmpire3Behavior(t, "SparseRegressionCancellationRetry", "")
}

func TestUmpire3SparseRegressionSharedHandlerWorkflow(t *testing.T) {
	runUmpire3Behavior(t, "SparseRegressionSharedHandlerWorkflow", "hsm")
}

func TestUmpire3SparseRegressionStartToCloseTimeout(t *testing.T) {
	for _, chasmEnabled := range []bool{false, true} {
		t.Run(umpire3CHASMName(chasmEnabled), func(t *testing.T) {
			runUmpire3Behavior(t, "SparseRegressionStartToCloseTimeout", umpire3CHASMNameLower(chasmEnabled))
		})
	}
}

func TestUmpire3SparseRegressionCallbackAfterCallerCompletion(t *testing.T) {
	for _, chasmEnabled := range []bool{false, true} {
		t.Run(umpire3CHASMName(chasmEnabled), func(t *testing.T) {
			runUmpire3Behavior(t, "SparseRegressionCallbackAfterCallerCompletion", umpire3CHASMNameLower(chasmEnabled))
		})
	}
}

func TestUmpire3SparseRegressionBidirectionalNexusActivityLinks(t *testing.T) {
	for _, chasmEnabled := range []bool{false, true} {
		t.Run(umpire3CHASMName(chasmEnabled), func(t *testing.T) {
			runUmpire3Behavior(t, "SparseRegressionBidirectionalNexusActivityLinks", umpire3CHASMNameLower(chasmEnabled))
		})
	}
}

func umpire3CHASMName(enabled bool) string {
	if enabled {
		return "CHASM"
	}
	return "HSM"
}

func umpire3CHASMNameLower(enabled bool) string {
	if enabled {
		return "chasm"
	}
	return "hsm"
}
