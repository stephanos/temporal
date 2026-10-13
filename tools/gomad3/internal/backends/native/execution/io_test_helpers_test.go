package execution_test

import (
	"go.temporal.io/server/tools/gomad3/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

func recordAdapters(adapters []transcript.BuildAdapter) []record.TargetAdapter {
	result := make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}
