package main

import (
	_ "go.temporal.io/server/chasm/lib/activity"
	_ "go.temporal.io/server/chasm/lib/nexusoperation"
	_ "go.temporal.io/server/chasm/lib/scheduler"
	_ "go.temporal.io/server/common/nexus"
	_ "go.temporal.io/server/service/history/hsm/callbacks"
	_ "go.temporal.io/server/service/history/hsm/nexusoperations"

	callbackconfig "go.temporal.io/server/chasm/lib/callback"
	"go.temporal.io/server/common/dynamicconfig"
)

func init() {
	productionFixtureGenerator = func(catalogSettings []ProjectedSetting) ([]ResolutionFixture, error) {
		return computeProductionFixtures(productionSettings{
			Global:        dynamicconfig.AdminEnableListHistoryTasks,
			Namespace:     callbackconfig.MaxPerExecution,
			NamespaceID:   dynamicconfig.SkipReapplicationByNamespaceID,
			TaskQueue:     dynamicconfig.MatchingUpdateAckInterval,
			ShardID:       dynamicconfig.ReplicationTaskProcessorErrorRetryMaxAttempts,
			TaskType:      dynamicconfig.StandbyTaskMissingEventsResendDelay,
			Destination:   callbackconfig.RequestTimeout,
			ChasmTaskType: dynamicconfig.ChasmStandbyTaskDiscardDelay,
		}, catalogSettings)
	}
}
