package entity_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/entity"
	"go.temporal.io/server/tools/umpire/event"
	"go.temporal.io/server/tools/umpire/identity"
	"go.temporal.io/server/tools/umpire/model"
)

// TestLostTaskUnit verifies the unit-level behavior of lost task detection.
// This test doesn't require full Temporal infrastructure - it directly tests
// the entity and model behavior.
func TestLostTaskUnit(t *testing.T) {
	// Setup
	logger := log.NewNoopLogger()
	registry, err := entity.NewEntityRegistry(logger, "")
	require.NoError(t, err)
	defer registry.Close()

	entity.RegisterDefaultEntities(registry)

	// Create task matching model
	taskMatchingModel := &model.TaskMatchingModel{}
	err = taskMatchingModel.Init(context.Background(), umpire.Deps{
		Registry: registry,
		Logger:   logger,
	})
	require.NoError(t, err)

	// Timestamp for events
	baseTime := time.Now().Add(-1 * time.Minute) // Old enough to trigger checks
	taskQueue := "test-queue"
	workflowID := "test-workflow-1"
	runID := "test-run-1"

	// Create identities
	workflowTaskID := identity.NewEntityIDFromType("WorkflowTask", taskQueue+":"+workflowID+":"+runID)
	taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
	taskIdentity := &identity.Identity{
		EntityID: workflowTaskID,
		ParentID: &taskQueueID,
	}
	taskQueueIdentity := &identity.Identity{
		EntityID: taskQueueID,
		ParentID: nil,
	}

	// Simulate the sequence of events:
	// 1. Task added to matching
	// 2. Task stored to persistence (removed from matching)
	// 3. Empty poll (task not in matching anymore)
	events := []event.Event{
		&event.AddWorkflowTaskEvent{
			Timestamp:  baseTime,
			TaskQueue:  taskQueue,
			Identity:   taskIdentity,
			WorkflowID: workflowID,
			RunID:      runID,
		},
		&event.StoreWorkflowTaskEvent{
			Timestamp:  baseTime.Add(1 * time.Second),
			TaskQueue:  taskQueue,
			Identity:   taskIdentity,
			WorkflowID: workflowID,
			RunID:      runID,
		},
		&event.PollWorkflowTaskEvent{
			Timestamp:    baseTime.Add(2 * time.Second),
			TaskQueue:    taskQueue,
			Identity:     taskQueueIdentity,
			WorkflowID:   "",
			RunID:        "",
			TaskReturned: false,
		},
	}

	err = registry.RouteEvents(context.Background(), events)
	require.NoError(t, err)

	// Run the check - should detect the lost task
	taskMatchingModel.Check(context.Background())

	// Verify the entities are in the expected state
	workflowTasks := registry.QueryEntities(entity.NewWorkflowTask())
	require.Len(t, workflowTasks, 1)

	wt := workflowTasks[0].(*entity.WorkflowTask)
	require.Equal(t, "stored", wt.FSM.Current(), "Task should be in 'stored' state")
	require.False(t, wt.StoredAt.IsZero(), "StoredAt timestamp should be set")
	require.True(t, wt.PolledAt.IsZero(), "Task should never have been polled")

	taskQueues := registry.QueryEntities(entity.NewTaskQueue())
	require.Len(t, taskQueues, 1)

	tq := taskQueues[0].(*entity.TaskQueue)
	require.False(t, tq.LastEmptyPollTime.IsZero(), "LastEmptyPollTime should be set")
	require.True(t, tq.LastEmptyPollTime.After(wt.StoredAt), "Empty poll should occur after task was stored")
}

// TestTaskQueueTracksEmptyPolls verifies that TaskQueue entities properly track empty polls
func TestTaskQueueTracksEmptyPolls(t *testing.T) {
	logger := log.NewNoopLogger()
	registry, err := entity.NewEntityRegistry(logger, "")
	require.NoError(t, err)
	defer registry.Close()

	entity.RegisterDefaultEntities(registry)

	taskQueue := "test-queue"
	taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
	taskQueueIdentity := &identity.Identity{
		EntityID: taskQueueID,
		ParentID: nil,
	}

	pollTime := time.Now()

	// Send an empty poll event
	emptyPollEvent := &event.PollWorkflowTaskEvent{
		Timestamp:    pollTime,
		TaskQueue:    taskQueue,
		Identity:     taskQueueIdentity,
		WorkflowID:   "",
		RunID:        "",
		TaskReturned: false,
	}

	err = registry.RouteEvents(context.Background(), []event.Event{emptyPollEvent})
	require.NoError(t, err)

	// Verify TaskQueue tracked the empty poll
	taskQueues := registry.QueryEntities(entity.NewTaskQueue())
	require.Len(t, taskQueues, 1)

	tq := taskQueues[0].(*entity.TaskQueue)
	require.Equal(t, taskQueue, tq.Name)
	require.Equal(t, pollTime, tq.LastEmptyPollTime)
}

// TestWorkflowTaskStoreTransition verifies the FSM transition to stored state
func TestWorkflowTaskStoreTransition(t *testing.T) {
	logger := log.NewNoopLogger()
	registry, err := entity.NewEntityRegistry(logger, "")
	require.NoError(t, err)
	defer registry.Close()

	entity.RegisterDefaultEntities(registry)

	taskQueue := "test-queue"
	workflowID := "test-workflow"
	runID := "test-run"

	workflowTaskID := identity.NewEntityIDFromType("WorkflowTask", taskQueue+":"+workflowID+":"+runID)
	taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
	taskIdentity := &identity.Identity{
		EntityID: workflowTaskID,
		ParentID: &taskQueueID,
	}

	baseTime := time.Now()

	// Send events: add then store
	events := []event.Event{
		&event.AddWorkflowTaskEvent{
			Timestamp:  baseTime,
			TaskQueue:  taskQueue,
			Identity:   taskIdentity,
			WorkflowID: workflowID,
			RunID:      runID,
		},
		&event.StoreWorkflowTaskEvent{
			Timestamp:  baseTime.Add(1 * time.Second),
			TaskQueue:  taskQueue,
			Identity:   taskIdentity,
			WorkflowID: workflowID,
			RunID:      runID,
		},
	}

	err = registry.RouteEvents(context.Background(), events)
	require.NoError(t, err)

	// Verify task transitioned to stored state
	workflowTasks := registry.QueryEntities(entity.NewWorkflowTask())
	require.Len(t, workflowTasks, 1)

	wt := workflowTasks[0].(*entity.WorkflowTask)
	require.Equal(t, "stored", wt.FSM.Current())
	require.False(t, wt.AddedAt.IsZero())
	require.False(t, wt.StoredAt.IsZero())
	require.True(t, wt.PolledAt.IsZero())
	require.True(t, wt.StoredAt.After(wt.AddedAt))
}
