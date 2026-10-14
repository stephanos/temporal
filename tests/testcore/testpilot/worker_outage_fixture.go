package testpilot

const (
	// WorkerOutageFixture is the one Case the worker-outage Model's functional set produces: the
	// controller stops the worker of its own task queue, starts the workflow, resumes the worker
	// and waits for the workflow the resumed worker completes.
	WorkerOutageFixture = "workerOutageTests-survived"
	// WorkerOutageRuleID is the outage-order rule the Producer derives from the two faults on the
	// Case's path: bounded liveness from the stop to the resume, with an event-count deadline.
	WorkerOutageRuleID = "worker-outage-order"
	// WorkerOutageTaskQueueRole is the role both faults are injected on.
	WorkerOutageTaskQueueRole = "temporal.task-queue"

	// SystemInfoFixture is the one Case the system-info Model's functional set produces: one
	// unary GetSystemInfo call and no workflow.
	SystemInfoFixture = "systemInfoTests-answered"
)
