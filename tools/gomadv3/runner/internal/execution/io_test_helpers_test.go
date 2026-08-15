package execution_test

import (
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
)

func recordAdapters(adapters []deterministicio.BuildAdapter) []evidence.TargetAdapter {
	result := make([]evidence.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = evidence.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}
