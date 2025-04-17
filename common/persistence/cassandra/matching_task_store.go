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
	"time"

	"go.temporal.io/server/common/convert"
	"go.temporal.io/server/common/log"
	p "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/nosql/nosqlplugin/cassandra/gocql"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Not much of a need to make this configurable, we're just reading some strings
	listTaskQueueNamesByBuildIdPageSize = 100

	// Row types for table tasks. Lower bit only: see rowTypeTaskInSubqueue for more details.
	rowTypeTask = iota
	rowTypeTaskQueue
)

func NewMatchingTaskStore(
	session gocql.Session,
	logger log.Logger,
	enableFairness bool,
) p.TaskStore {
	if enableFairness {
		return newMatchingTaskStoreV2(session, logger)
	}
	return newMatchingTaskStoreV1(session, logger)
}

// We steal some upper bits of the "row type" field to hold a subqueue index.
// Subqueue 0 must be the same as rowTypeTask (before subqueues were introduced).
// 00000000: task in subqueue 0 (rowTypeTask)
// 00000001: task queue metadata (rowTypeTaskQueue)
// xxxxxx1x: reserved
// 00000100: task in subqueue 1
// nnnnnn00: task in subqueue n, etc.
func rowTypeTaskInSubqueue(subqueue int) int {
	return subqueue<<2 | rowTypeTask // nolint:staticcheck
}

func getTaskTTL(expireTime *timestamppb.Timestamp) int64 {
	var ttl int64 = 0
	if expireTime != nil && !expireTime.AsTime().IsZero() {
		expiryTtl := convert.Int64Ceil(time.Until(expireTime.AsTime()).Seconds())

		// 0 means no ttl, we dont want that.
		// Todo: Come back and correctly ignore expired in-memory tasks before persisting
		if expiryTtl < 1 {
			expiryTtl = 1
		}

		ttl = expiryTtl
	}
	return ttl
}
