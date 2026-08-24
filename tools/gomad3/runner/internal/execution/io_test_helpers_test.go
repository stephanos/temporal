package execution_test

import (
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
)

func recordAdapters(adapters []deterministicio.BuildAdapter) []record.TargetAdapter {
	result := make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}
