// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package matching

import (
	"context"
	"sync"

	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/resource"
	"go.temporal.io/server/common/tqid"
)

type physicalTaskQueueManagerPool struct {
	taskManager       persistence.TaskManager
	namespaceRegistry namespace.Registry
	config            *Config
	logger            log.Logger
	throttledLogger   log.ThrottledLogger
	metricsHandler    metrics.Handler
	matchingRawClient resource.MatchingRawClient
	partitionsLock    sync.RWMutex // locks mutation of partitions
	partitions        map[tqid.PartitionKey]taskQueuePartitionManager
}

func newPhysicalTaskQueueManagerPool(
	taskManager persistence.TaskManager,
	namespaceRegistry namespace.Registry,
	config *Config,
) *physicalTaskQueueManagerPool {
	return &physicalTaskQueueManagerPool{
		partitions: make(map[tqid.PartitionKey]taskQueuePartitionManager),
	}
}

// Returns taskQueuePartitionManager for a task queue. If not already cached, and create is true, tries
// to get new range from DB and create one. This blocks (up to the context deadline) for the
// task queue to be initialized.
//
// Note that task queue kind (sticky vs normal) and normal name for sticky task queues is not used as
// part of the task queue identity. That means that if getOrCreate
// is called twice with the same task queue but different sticky info, the
// properties of the taskQueuePartitionManager will depend on which call came first. In general, we can
// rely on kind being the same for all calls now, but normalName was a later addition to the
// protocol and is not always set consistently. normalName is only required when using
// versioning, and SDKs that support versioning will always set it. The current server version
// will also set it when adding tasks from history. So that particular inconsistency is okay.
func (p *physicalTaskQueueManagerPool) getOrCreate(
	ctx context.Context,
	engine *matchingEngineImpl,
	partition tqid.Partition,
	create bool,
	loadCause loadCause,
) (retPM taskQueuePartitionManager, retCreated bool, retErr error) {
	defer func() {
		if retErr != nil || retPM == nil {
			return
		}

		if retErr = retPM.WaitUntilInitialized(ctx); retErr != nil {
			p.unload(retPM, unloadCauseInitError)
		}
	}()

	key := partition.Key()
	p.partitionsLock.RLock()
	pm, ok := p.partitions[key]
	p.partitionsLock.RUnlock()
	if ok {
		return pm, false, nil
	}

	if !create {
		return nil, false, nil
	}

	namespaceEntry, err := p.namespaceRegistry.GetNamespaceByID(namespace.ID(partition.NamespaceId()))
	if err != nil {
		return nil, false, err
	}
	nsName := namespaceEntry.Name()

	tqConfig := newTaskQueueConfig(partition.TaskQueue(), p.config, nsName)
	tqConfig.loadCause = loadCause
	logger, throttledLogger, metricsHandler := p.loggerAndMetricsForPartition(nsName, partition, tqConfig)
	var newPM *taskQueuePartitionManagerImpl
	onFatalErr := func(cause unloadCause) { newPM.unload(cause) }
	userDataManager := newUserDataManager(p.taskManager, p.matchingRawClient, onFatalErr, partition, tqConfig, logger, p.namespaceRegistry)
	newPM, err = newTaskQueuePartitionManager(engine, namespaceEntry, partition, tqConfig, logger, throttledLogger, metricsHandler, userDataManager)
	if err != nil {
		return nil, false, err
	}

	// If it gets here, write lock and check again in case a task queue is created between the two locks
	p.partitionsLock.Lock()
	pm, ok = p.partitions[key]
	if ok {
		p.partitionsLock.Unlock()
		return pm, false, nil
	}
	p.partitions[key] = newPM
	p.partitionsLock.Unlock()

	newPM.Start()
	if newPM.Partition().IsRoot() {
		// Whenever a root partition is loaded we need to force all other partitions to load.
		// If there is a backlog of tasks on any child partitions force loading will ensure that they
		// can forward their tasks the poller which caused the root partition to be loaded.
		// These partitions could be managed by this matchingEngineImpl, but are most likely not.
		// We skip checking and just make gRPC requests to force loading them all.

		newPM.ForceLoadAllNonRootPartitions()
	}
	return newPM, true, nil
}

func (p *physicalTaskQueueManagerPool) get(
	partitionKey tqid.PartitionKey,
) (taskQueuePartitionManager, bool) {
	p.partitionsLock.RLock()
	pm, ok := p.partitions[partitionKey]
	p.partitionsLock.RUnlock()
	return pm, ok
}

func (p *physicalTaskQueueManagerPool) getLoadedPartitions() []tqid.Partition {
	p.partitionsLock.RLock()
	defer p.partitionsLock.RUnlock()

	partitions := make([]tqid.Partition, 0, len(p.partitions))
	for _, pm := range p.partitions {
		partitions = append(partitions, pm.Partition())
	}
	return partitions
}

func (p *physicalTaskQueueManagerPool) getAll(
	maxCount int,
) (lists []taskQueuePartitionManager) {
	p.partitionsLock.RLock()
	defer p.partitionsLock.RUnlock()

	lists = make([]taskQueuePartitionManager, 0, len(p.partitions))
	count := 0
	for _, tlMgr := range p.partitions {
		lists = append(lists, tlMgr)
		count++
		if count >= maxCount {
			break
		}
	}
	return
}

// Unloads the given task queue partition. If it has already been unloaded (i.e. it's not present in the loaded
// partitions map), then does nothing.
// partitions map), unloadPM.Stop(...) is still called.
func (p *physicalTaskQueueManagerPool) unload(
	unloadPM taskQueuePartitionManager,
	unloadCause unloadCause,
) {
	p.unloadByKey(unloadPM.Partition(), unloadPM, unloadCause)
}

// Unloads a task queue partition by id. If unloadPM is given and the loaded partition for queueID does not match
// unloadPM, then nothing is unloaded from matching engine (but unloadPM will be stopped).
// Returns true if it unloaded a partition and false if not.
func (p *physicalTaskQueueManagerPool) unloadByKey(
	partition tqid.Partition,
	unloadPM taskQueuePartitionManager,
	unloadCause unloadCause,
) bool {
	key := partition.Key()
	p.partitionsLock.Lock()
	foundTQM, ok := p.partitions[key]
	if !ok || (unloadPM != nil && foundTQM != unloadPM) {
		p.partitionsLock.Unlock()
		return false
	}
	delete(p.partitions, key)
	p.partitionsLock.Unlock()
	foundTQM.Stop(unloadCause)
	return true
}

// For use in tests
func (p *physicalTaskQueueManagerPool) updateTaskQueue(partition tqid.Partition, mgr taskQueuePartitionManager) {
	p.partitionsLock.Lock()
	defer p.partitionsLock.Unlock()
	p.partitions[partition.Key()] = mgr
}

func (p *physicalTaskQueueManagerPool) loggerAndMetricsForPartition(
	nsName namespace.Name,
	partition tqid.Partition,
	tqConfig *taskQueueConfig,
) (log.Logger, log.Logger, metrics.Handler) {
	logger := log.With(p.logger,
		tag.WorkflowTaskQueueName(partition.RpcName()),
		tag.WorkflowTaskQueueType(partition.TaskType()),
		tag.WorkflowNamespace(nsName.String()))
	throttledLogger := log.With(p.throttledLogger,
		tag.WorkflowTaskQueueName(partition.RpcName()),
		tag.WorkflowTaskQueueType(partition.TaskType()),
		tag.WorkflowNamespace(nsName.String()))
	metricsHandler := metrics.GetPerTaskQueuePartitionIDScope(
		p.metricsHandler,
		nsName.String(),
		partition,
		tqConfig.BreakdownMetricsByTaskQueue(),
		tqConfig.BreakdownMetricsByPartition(),
		metrics.OperationTag(metrics.MatchingTaskQueuePartitionManagerScope),
	)
	return logger, throttledLogger, metricsHandler
}
