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

package cassandra

import (
	"context"
	"fmt"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/common/convert"
	"go.temporal.io/server/common/log"
	p "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/nosql/nosqlplugin/cassandra/gocql"
	"go.temporal.io/server/common/primitives/timestamp"
)

const (
	templateCreateTaskQuery_v2 = `INSERT INTO tasks_v2 (` +
		`namespace_id, task_queue_name, task_queue_type, type, pass, task_id, task, task_encoding) ` +
		`VALUES(?, ?, ?, ?, ?, ?, ?, ?)`

	templateCreateTaskWithTTLQuery_v2 = `INSERT INTO tasks_v2 (` +
		`namespace_id, task_queue_name, task_queue_type, type, pass, task_id, task, task_encoding) ` +
		`VALUES(?, ?, ?, ?, ?, ?, ?, ?) USING TTL ?`

	templateGetTasksQuery_v2 = `SELECT task_id, task, task_encoding ` +
		`FROM tasks ` +
		`WHERE namespace_id = ? ` +
		`and task_queue_name = ? ` +
		`and task_queue_type = ? ` +
		`and type = ? ` +
		`and pass = 0 ` + // TODO
		`and task_id >= ? ` +
		`and task_id < ?`

	templateCompleteTasksLessThanQuery_v2 = `DELETE FROM tasks_v2 ` +
		`WHERE namespace_id = ? ` +
		`AND task_queue_name = ? ` +
		`AND task_queue_type = ? ` +
		`AND type = ? ` +
		`and pass = 0 ` + // TODO
		`AND task_id < ? `

	templateGetTaskQueueQuery_v2 = `SELECT ` +
		`range_id, ` +
		`task_queue, ` +
		`task_queue_encoding ` +
		`FROM tasks_v2 ` +
		`WHERE namespace_id = ? ` +
		`and task_queue_name = ? ` +
		`and task_queue_type = ? ` +
		`and type = ? ` +
		`and pass = 0 ` + // TODO
		`and task_id = ?`

	templateInsertTaskQueueQuery_v2 = `INSERT INTO tasks_v2 (` +
		`namespace_id, ` +
		`task_queue_name, ` +
		`task_queue_type, ` +
		`type, ` +
		`pass, ` +
		`task_id, ` +
		`range_id, ` +
		`task_queue, ` +
		`task_queue_encoding ` +
		`) VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?) IF NOT EXISTS`

	templateUpdateTaskQueueQuery_v2 = `UPDATE tasks_v2 SET ` +
		`range_id = ?, ` +
		`task_queue = ?, ` +
		`task_queue_encoding = ? ` +
		`WHERE namespace_id = ? ` +
		`and task_queue_name = ? ` +
		`and task_queue_type = ? ` +
		`and pass = 0 ` +
		`and type = ? ` +
		`and task_id = ? ` +
		`IF range_id = ?`

	templateUpdateTaskQueueQueryWithTTLPart1_v2 = `INSERT INTO tasks_v2 (` +
		`namespace_id, ` +
		`task_queue_name, ` +
		`task_queue_type, ` +
		`type, ` +
		`pass, ` +
		`task_id ` +
		`) VALUES (?, ?, ?, ?, 0, ?) USING TTL ?`

	templateUpdateTaskQueueQueryWithTTLPart2_v2 = `UPDATE tasks_v2 USING TTL ? SET ` +
		`range_id = ?, ` +
		`task_queue = ?, ` +
		`task_queue_encoding = ? ` +
		`WHERE namespace_id = ? ` +
		`and task_queue_name = ? ` +
		`and task_queue_type = ? ` +
		`and type = ? ` +
		`and pass = 0 ` +
		`and task_id = ? ` +
		`IF range_id = ?`

	templateDeleteTaskQueueQuery_v2 = `DELETE FROM tasks_v2 ` +
		`WHERE namespace_id = ? ` +
		`AND task_queue_name = ? ` +
		`AND task_queue_type = ? ` +
		`AND type = ? ` +
		`and pass = 0 ` +
		`AND task_id = ? ` +
		`IF range_id = ?`

	templateGetTaskQueueUserDataQuery_v2 = `SELECT data, data_encoding, version
	    FROM task_queue_user_data
		WHERE namespace_id = ? AND build_id = ''
		AND task_queue_name = ?`

	templateUpdateTaskQueueUserDataQuery_v2 = `UPDATE task_queue_user_data SET
		data = ?,
		data_encoding = ?,
		version = ?
		WHERE namespace_id = ?
		AND build_id = ''
		AND task_queue_name = ?
		IF version = ?`

	templateInsertTaskQueueUserDataQuery_v2 = `INSERT INTO task_queue_user_data
		(namespace_id, build_id, task_queue_name, data, data_encoding, version) VALUES
		(?           , ''      , ?              , ?   , ?            , 1      ) IF NOT EXISTS`

	templateInsertBuildIdTaskQueueMappingQuery_v2 = `INSERT INTO task_queue_user_data
	(namespace_id, build_id, task_queue_name) VALUES
	(?           , ?       , ?)`
	templateDeleteBuildIdTaskQueueMappingQuery_v2 = `DELETE FROM task_queue_user_data
	WHERE namespace_id = ? AND build_id = ? AND task_queue_name = ?`
	templateListTaskQueueUserDataQuery_v2       = `SELECT task_queue_name, data, data_encoding, version FROM task_queue_user_data WHERE namespace_id = ? AND build_id = ''`
	templateListTaskQueueNamesByBuildIdQuery_v2 = `SELECT task_queue_name FROM task_queue_user_data WHERE namespace_id = ? AND build_id = ?`
	templateCountTaskQueueByBuildIdQuery_v2     = `SELECT COUNT(*) FROM task_queue_user_data WHERE namespace_id = ? AND build_id = ?`
)

type (
	// matchingTaskStoreV2 is a fork of matchingTaskStoreV1 that uses a new task schema.
	// All methods and queries were duplicated (even if they didn't change) to reduce the shared code.
	// Eventually, the original matchingTaskStoreV1 will be removed; and it this can be
	// renamed to matchingTaskStoreV1.
	matchingTaskStoreV2 struct {
		Session gocql.Session
		Logger  log.Logger
	}
)

func newMatchingTaskStoreV2(
	session gocql.Session,
	logger log.Logger,
) *matchingTaskStoreV2 {
	return &matchingTaskStoreV2{
		Session: session,
		Logger:  logger,
	}
}

func (d *matchingTaskStoreV2) CreateTaskQueue(
	ctx context.Context,
	request *p.InternalCreateTaskQueueRequest,
) error {
	query := d.Session.Query(templateInsertTaskQueueQuery_v2,
		request.NamespaceID,
		request.TaskQueue,
		request.TaskType,
		rowTypeTaskQueue,
		taskQueueTaskID,
		request.RangeID,
		request.TaskQueueInfo.Data,
		request.TaskQueueInfo.EncodingType.String(),
	).WithContext(ctx)

	previous := make(map[string]interface{})
	applied, err := query.MapScanCAS(previous)
	if err != nil {
		return gocql.ConvertError("CreateTaskQueue", err)
	}

	if !applied {
		previousRangeID := previous["range_id"]
		return &p.ConditionFailedError{
			Msg: fmt.Sprintf("CreateTaskQueue: TaskQueue:%v, TaskQueueType:%v, PreviousRangeID:%v",
				request.TaskQueue, request.TaskType, previousRangeID),
		}
	}

	return nil
}

func (d *matchingTaskStoreV2) GetTaskQueue(
	ctx context.Context,
	request *p.InternalGetTaskQueueRequest,
) (*p.InternalGetTaskQueueResponse, error) {
	query := d.Session.Query(templateGetTaskQueueQuery_v2,
		request.NamespaceID,
		request.TaskQueue,
		request.TaskType,
		rowTypeTaskQueue,
		taskQueueTaskID,
	).WithContext(ctx)

	var rangeID int64
	var tlBytes []byte
	var tlEncoding string
	if err := query.Scan(&rangeID, &tlBytes, &tlEncoding); err != nil {
		return nil, gocql.ConvertError("GetTaskQueue", err)
	}

	return &p.InternalGetTaskQueueResponse{
		RangeID:       rangeID,
		TaskQueueInfo: p.NewDataBlob(tlBytes, tlEncoding),
	}, nil
}

// UpdateTaskQueue update task queue
func (d *matchingTaskStoreV2) UpdateTaskQueue(
	ctx context.Context,
	request *p.InternalUpdateTaskQueueRequest,
) (*p.UpdateTaskQueueResponse, error) {
	var err error
	var applied bool
	previous := make(map[string]interface{})
	if request.TaskQueueKind == enumspb.TASK_QUEUE_KIND_STICKY { // if task_queue is sticky, then update with TTL
		if request.ExpiryTime == nil {
			return nil, serviceerror.NewInternal("ExpiryTime cannot be nil for sticky task queue")
		}
		expiryTTL := convert.Int64Ceil(time.Until(timestamp.TimeValue(request.ExpiryTime)).Seconds())
		if expiryTTL >= maxCassandraTTL {
			expiryTTL = maxCassandraTTL
		}
		batch := d.Session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
		batch.Query(templateUpdateTaskQueueQueryWithTTLPart1_v2,
			request.NamespaceID,
			request.TaskQueue,
			request.TaskType,
			rowTypeTaskQueue,
			taskQueueTaskID,
			expiryTTL,
		)
		batch.Query(templateUpdateTaskQueueQueryWithTTLPart2_v2,
			expiryTTL,
			request.RangeID,
			request.TaskQueueInfo.Data,
			request.TaskQueueInfo.EncodingType.String(),
			request.NamespaceID,
			request.TaskQueue,
			request.TaskType,
			rowTypeTaskQueue,
			taskQueueTaskID,
			request.PrevRangeID,
		)
		applied, _, err = d.Session.MapExecuteBatchCAS(batch, previous)
	} else {
		query := d.Session.Query(templateUpdateTaskQueueQuery_v2,
			request.RangeID,
			request.TaskQueueInfo.Data,
			request.TaskQueueInfo.EncodingType.String(),
			request.NamespaceID,
			request.TaskQueue,
			request.TaskType,
			rowTypeTaskQueue,
			taskQueueTaskID,
			request.PrevRangeID,
		).WithContext(ctx)
		applied, err = query.MapScanCAS(previous)
	}

	if err != nil {
		return nil, gocql.ConvertError("UpdateTaskQueue", err)
	}

	if !applied {
		var columns []string
		for k, v := range previous {
			columns = append(columns, fmt.Sprintf("%s=%v", k, v))
		}

		return nil, &p.ConditionFailedError{
			Msg: fmt.Sprintf("Failed to update task queue. name: %v, type: %v, rangeID: %v, columns: (%v)",
				request.TaskQueue, request.TaskType, request.RangeID, strings.Join(columns, ",")),
		}
	}

	return &p.UpdateTaskQueueResponse{}, nil
}

func (d *matchingTaskStoreV2) ListTaskQueue(
	_ context.Context,
	_ *p.ListTaskQueueRequest,
) (*p.InternalListTaskQueueResponse, error) {
	return nil, serviceerror.NewUnavailable("unsupported operation")
}

func (d *matchingTaskStoreV2) DeleteTaskQueue(
	ctx context.Context,
	request *p.DeleteTaskQueueRequest,
) error {
	query := d.Session.Query(
		templateDeleteTaskQueueQuery_v2,
		request.TaskQueue.NamespaceID,
		request.TaskQueue.TaskQueueName,
		request.TaskQueue.TaskQueueType,
		rowTypeTaskQueue,
		taskQueueTaskID,
		request.RangeID,
	).WithContext(ctx)
	previous := make(map[string]interface{})
	applied, err := query.MapScanCAS(previous)
	if err != nil {
		return gocql.ConvertError("DeleteTaskQueue", err)
	}
	if !applied {
		return &p.ConditionFailedError{
			Msg: fmt.Sprintf("DeleteTaskQueue operation failed: expected_range_id=%v but found %+v", request.RangeID, previous),
		}
	}
	return nil
}

// CreateTasks add tasks
func (d *matchingTaskStoreV2) CreateTasks(
	ctx context.Context,
	request *p.InternalCreateTasksRequest,
) (*p.CreateTasksResponse, error) {
	batch := d.Session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	namespaceID := request.NamespaceID
	taskQueue := request.TaskQueue
	taskQueueType := request.TaskType

	for _, task := range request.Tasks {
		ttl := getTaskTTL(task.ExpiryTime)

		if ttl <= 0 || ttl > maxCassandraTTL {
			batch.Query(templateCreateTaskQuery_v2,
				namespaceID,
				taskQueue,
				taskQueueType,
				rowTypeTaskInSubqueue(task.Subqueue),
				task.Pass,
				task.TaskId,
				task.Task.Data,
				task.Task.EncodingType.String())
		} else {
			batch.Query(templateCreateTaskWithTTLQuery_v2,
				namespaceID,
				taskQueue,
				taskQueueType,
				rowTypeTaskInSubqueue(task.Subqueue),
				task.Pass,
				task.TaskId,
				task.Task.Data,
				task.Task.EncodingType.String(),
				ttl)
		}
	}

	// The following query is used to ensure that range_id didn't change
	batch.Query(templateUpdateTaskQueueQuery_v2,
		request.RangeID,
		request.TaskQueueInfo.Data,
		request.TaskQueueInfo.EncodingType.String(),
		namespaceID,
		taskQueue,
		taskQueueType,
		rowTypeTaskQueue,
		taskQueueTaskID,
		request.RangeID,
	)

	previous := make(map[string]interface{})
	applied, _, err := d.Session.MapExecuteBatchCAS(batch, previous)
	if err != nil {
		return nil, gocql.ConvertError("CreateTasks", err)
	}
	if !applied {
		rangeID := previous["range_id"]
		return nil, &p.ConditionFailedError{
			Msg: fmt.Sprintf("Failed to create task. TaskQueue: %v, taskQueueType: %v, rangeID: %v, db rangeID: %v",
				taskQueue, taskQueueType, request.RangeID, rangeID),
		}
	}

	return &p.CreateTasksResponse{UpdatedMetadata: true}, nil
}

// GetTasks get a task
func (d *matchingTaskStoreV2) GetTasks(
	ctx context.Context,
	request *p.GetTasksRequest,
) (*p.InternalGetTasksResponse, error) {
	// Reading taskqueue tasks need to be quorum level consistent, otherwise we could lose tasks
	query := d.Session.Query(templateGetTasksQuery_v2,
		request.NamespaceID,
		request.TaskQueue,
		request.TaskType,
		rowTypeTaskInSubqueue(request.Subqueue),
		request.InclusiveMinTaskID,
		request.ExclusiveMaxTaskID,
	).WithContext(ctx)
	iter := query.PageSize(request.PageSize).PageState(request.NextPageToken).Iter()

	response := &p.InternalGetTasksResponse{}
	task := make(map[string]interface{})
	for iter.MapScan(task) {
		_, ok := task["task_id"]
		if !ok { // no tasks, but static column record returned
			continue
		}

		rawTask, ok := task["task"]
		if !ok {
			return nil, newFieldNotFoundError("task", task)
		}
		taskVal, ok := rawTask.([]byte)
		if !ok {
			var byteSliceType []byte
			return nil, newPersistedTypeMismatchError("task", byteSliceType, rawTask, task)

		}

		rawEncoding, ok := task["task_encoding"]
		if !ok {
			return nil, newFieldNotFoundError("task_encoding", task)
		}
		encodingVal, ok := rawEncoding.(string)
		if !ok {
			var byteSliceType []byte
			return nil, newPersistedTypeMismatchError("task_encoding", byteSliceType, rawEncoding, task)
		}
		response.Tasks = append(response.Tasks, p.NewDataBlob(taskVal, encodingVal))

		task = make(map[string]interface{}) // Reinitialize map as initialized fails on unmarshalling
	}
	if len(iter.PageState()) > 0 {
		response.NextPageToken = iter.PageState()
	}

	if err := iter.Close(); err != nil {
		return nil, serviceerror.NewUnavailable(fmt.Sprintf("GetTasks operation failed. Error: %v", err))
	}
	return response, nil
}

// CompleteTasksLessThan deletes all tasks less than the given task id. This API ignores the
// Limit request parameter i.e. either all tasks leq the task_id will be deleted or an error will
// be returned to the caller
func (d *matchingTaskStoreV2) CompleteTasksLessThan(
	ctx context.Context,
	request *p.CompleteTasksLessThanRequest,
) (int, error) {
	query := d.Session.Query(
		templateCompleteTasksLessThanQuery_v2,
		request.NamespaceID,
		request.TaskQueueName,
		request.TaskType,
		rowTypeTaskInSubqueue(request.Subqueue),
		request.ExclusiveMaxTaskID,
	).WithContext(ctx)
	err := query.Exec()
	if err != nil {
		return 0, gocql.ConvertError("CompleteTasksLessThan", err)
	}
	return p.UnknownNumRowsAffected, nil
}

func (d *matchingTaskStoreV2) GetTaskQueueUserData(
	ctx context.Context,
	request *p.GetTaskQueueUserDataRequest,
) (*p.InternalGetTaskQueueUserDataResponse, error) {
	query := d.Session.Query(templateGetTaskQueueUserDataQuery_v2,
		request.NamespaceID,
		request.TaskQueue,
	).WithContext(ctx)
	var version int64
	var userDataBytes []byte
	var encoding string
	if err := query.Scan(&userDataBytes, &encoding, &version); err != nil {
		return nil, gocql.ConvertError("GetTaskQueueData", err)
	}

	return &p.InternalGetTaskQueueUserDataResponse{
		Version:  version,
		UserData: p.NewDataBlob(userDataBytes, encoding),
	}, nil
}

func (d *matchingTaskStoreV2) UpdateTaskQueueUserData(
	ctx context.Context,
	request *p.InternalUpdateTaskQueueUserDataRequest,
) error {
	batch := d.Session.NewBatch(gocql.UnloggedBatch).WithContext(ctx)

	for taskQueue, update := range request.Updates {
		if update.Version == 0 {
			batch.Query(templateInsertTaskQueueUserDataQuery_v2,
				request.NamespaceID,
				taskQueue,
				update.UserData.Data,
				update.UserData.EncodingType.String(),
			)
		} else {
			batch.Query(templateUpdateTaskQueueUserDataQuery_v2,
				update.UserData.Data,
				update.UserData.EncodingType.String(),
				update.Version+1,
				request.NamespaceID,
				taskQueue,
				update.Version,
			)
		}
		for _, buildId := range update.BuildIdsAdded {
			batch.Query(templateInsertBuildIdTaskQueueMappingQuery_v2, request.NamespaceID, buildId, taskQueue)
		}
		for _, buildId := range update.BuildIdsRemoved {
			batch.Query(templateDeleteBuildIdTaskQueueMappingQuery_v2, request.NamespaceID, buildId, taskQueue)
		}
	}

	previous := make(map[string]any)
	applied, iter, err := d.Session.MapExecuteBatchCAS(batch, previous)
	for _, update := range request.Updates {
		if update.Applied != nil {
			*update.Applied = applied
		}
	}
	if err != nil {
		return gocql.ConvertError("UpdateTaskQueueUserData", err)
	}
	defer iter.Close()

	if !applied {
		// No error, but not applied. That means we had a conflict.
		// Iterate through results to identify first conflicting row.
		for {
			name, nameErr := getTypedFieldFromRow[string]("task_queue_name", previous)
			previousVersion, verErr := getTypedFieldFromRow[int64]("version", previous)
			update, hasUpdate := request.Updates[name]
			if nameErr == nil && verErr == nil && hasUpdate && update.Version != previousVersion {
				if update.Conflicting != nil {
					*update.Conflicting = true
				}
				return &p.ConditionFailedError{
					Msg: fmt.Sprintf("Failed to update task queues: task queue %q version %d != %d",
						name, update.Version, previousVersion),
				}
			}
			clear(previous)
			if !iter.MapScan(previous) {
				break
			}
		}
		return &p.ConditionFailedError{Msg: "Failed to update task queues: unknown conflict"}
	}

	return nil
}

func (d *matchingTaskStoreV2) ListTaskQueueUserDataEntries(ctx context.Context, request *p.ListTaskQueueUserDataEntriesRequest) (*p.InternalListTaskQueueUserDataEntriesResponse, error) {
	query := d.Session.Query(templateListTaskQueueUserDataQuery_v2, request.NamespaceID).WithContext(ctx)
	iter := query.PageSize(request.PageSize).PageState(request.NextPageToken).Iter()

	response := &p.InternalListTaskQueueUserDataEntriesResponse{}
	row := make(map[string]interface{})
	for iter.MapScan(row) {
		taskQueue, err := getTypedFieldFromRow[string]("task_queue_name", row)
		if err != nil {
			return nil, err
		}
		data, err := getTypedFieldFromRow[[]byte]("data", row)
		if err != nil {
			return nil, err
		}
		dataEncoding, err := getTypedFieldFromRow[string]("data_encoding", row)
		if err != nil {
			return nil, err
		}
		version, err := getTypedFieldFromRow[int64]("version", row)
		if err != nil {
			return nil, err
		}

		response.Entries = append(response.Entries, p.InternalTaskQueueUserDataEntry{TaskQueue: taskQueue, Data: p.NewDataBlob(data, dataEncoding), Version: version})

		row = make(map[string]interface{}) // Reinitialize map as initialized fails on unmarshalling
	}
	if len(iter.PageState()) > 0 {
		response.NextPageToken = iter.PageState()
	}

	if err := iter.Close(); err != nil {
		return nil, serviceerror.NewUnavailable(fmt.Sprintf("ListTaskQueueUserDataEntries operation failed. Error: %v", err))
	}
	return response, nil
}

func (d *matchingTaskStoreV2) GetTaskQueuesByBuildId(ctx context.Context, request *p.GetTaskQueuesByBuildIdRequest) ([]string, error) {
	query := d.Session.Query(templateListTaskQueueNamesByBuildIdQuery_v2, request.NamespaceID, request.BuildID).WithContext(ctx)
	iter := query.PageSize(listTaskQueueNamesByBuildIdPageSize).Iter()

	var taskQueues []string
	row := make(map[string]interface{})

	for {
		for iter.MapScan(row) {
			taskQueueRaw, ok := row["task_queue_name"]
			if !ok {
				return nil, newFieldNotFoundError("task_queue_name", row)
			}
			taskQueue, ok := taskQueueRaw.(string)
			if !ok {
				var stringType string
				return nil, newPersistedTypeMismatchError("task_queue_name", stringType, taskQueueRaw, row)
			}

			taskQueues = append(taskQueues, taskQueue)

			row = make(map[string]interface{}) // Reinitialize map as initialized fails on unmarshalling
		}
		if len(iter.PageState()) == 0 {
			break
		}
	}

	if err := iter.Close(); err != nil {
		return nil, serviceerror.NewUnavailable(fmt.Sprintf("GetTaskQueuesByBuildId operation failed. Error: %v", err))
	}
	return taskQueues, nil
}

func (d *matchingTaskStoreV2) CountTaskQueuesByBuildId(ctx context.Context, request *p.CountTaskQueuesByBuildIdRequest) (int, error) {
	var count int
	query := d.Session.Query(templateCountTaskQueueByBuildIdQuery_v2, request.NamespaceID, request.BuildID).WithContext(ctx)
	err := query.Scan(&count)
	return count, err
}

func (d *matchingTaskStoreV2) GetName() string {
	return cassandraPersistenceName
}

func (d *matchingTaskStoreV2) Close() {
	if d.Session != nil {
		d.Session.Close()
	}
}
