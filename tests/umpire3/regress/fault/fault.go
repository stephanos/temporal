package fault

import "go.temporal.io/server/tests/umpire3/regress"

func StaleWorkerCompletion(identifier string) regress.FaultIntent {
	return regress.StaleWorkerCompletion(identifier)
}
