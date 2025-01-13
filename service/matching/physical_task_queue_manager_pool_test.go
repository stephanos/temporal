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
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	enumspb "go.temporal.io/api/enums/v1"
)

type (
	physicalTaskQueueManagerPoolSuite struct {
		suite.Suite
		*require.Assertions
	}
)

func TestPhysicalTaskQueueManagerPoolSuite(t *testing.T) {
	s := new(physicalTaskQueueManagerPoolSuite)
	suite.Run(t, s)
}

func (s *physicalTaskQueueManagerPoolSuite) TestOnlyUnloadMatchingInstance() {
	prtn := newRootPartition(
		uuid.New(),
		"makeToast",
		enumspb.TASK_QUEUE_TYPE_ACTIVITY)
	tqm, _, err := s.matchingEngine.physicalTaskQueueManagerPool.getOrCreate(context.Background(), s.matchingEngine, prtn, true, loadCauseUnspecified)
	s.Require().NoError(err)

	tqm2 := s.newPartitionManager(prtn, s.matchingEngine.config)

	// try to unload a different tqm instance with the same taskqueue ID
	s.matchingEngine.physicalTaskQueueManagerPool.unload(tqm2, unloadCauseUnspecified)

	got, _, err := s.matchingEngine.getTaskQueuePartitionManager(context.Background(), prtn, true, loadCauseUnspecified)
	s.Require().NoError(err)
	s.Require().Same(tqm, got,
		"Unload call with non-matching taskQueuePartitionManager should not cause unload")

	// this time unload the right tqm
	s.matchingEngine.unloadTaskQueuePartition(tqm, unloadCauseUnspecified)

	got, _, err = s.matchingEngine.physicalTaskQueueManagerPool.getOrCreate(context.Background(), s.matchingEngine, prtn, true, loadCauseUnspecified)
	s.Require().NoError(err)
	s.Require().NotSame(tqm, got,
		"Unload call with matching incarnation should have caused unload")
}
