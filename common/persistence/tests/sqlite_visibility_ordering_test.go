//go:build gomad

package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/server/common/namespace"
	persistencetests "go.temporal.io/server/common/persistence/persistence-tests"
	"go.temporal.io/server/common/persistence/visibility/manager"
)

// sqliteVisibilityOrderingSuite pins the order the gomad build of the SQLite
// store gives executions with equal times; the stock build, like the MySQL
// and PostgreSQL stores, orders run IDs ascending there.
type sqliteVisibilityOrderingSuite struct {
	suite.Suite
	base *VisibilityPersistenceSuite
}

func TestSQLiteVisibilityOrderingSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(sqliteVisibilityOrderingSuite))
}

func (s *sqliteVisibilityOrderingSuite) SetupSuite() {
	s.base = &VisibilityPersistenceSuite{TestBase: persistencetests.NewTestBaseWithSQL(persistencetests.GetSQLiteMemoryTestClusterOption())}
	s.base.SetT(s.T())
	s.base.SetupSuite()
}

func (s *sqliteVisibilityOrderingSuite) SetupTest() {
	s.base.SetT(s.T())
	s.base.SetupTest()
}

func (s *sqliteVisibilityOrderingSuite) TearDownTest() {
	s.base.TearDownTest()
}

func (s *sqliteVisibilityOrderingSuite) TearDownSuite() {
	s.base.TearDownSuite()
}

// Run IDs are UUIDv7, so with equal start times the larger run ID started
// later; it lists first, on the first page and across the page token.
func (s *sqliteVisibilityOrderingSuite) TestEqualStartTimesListTheLaterRunFirst() {
	namespaceID := namespace.ID(uuid.NewString())
	startTime := time.Now().UTC().Add(-time.Minute)
	earlier := "00000000-0000-7000-8000-00000000000a"
	later := "00000000-0000-7000-8000-00000000000b"
	for _, runID := range []string{later, earlier} {
		s.base.taskID++
		s.Require().NoError(s.base.VisibilityMgr.RecordWorkflowExecutionStarted(s.base.ctx, &manager.RecordWorkflowExecutionStartedRequest{
			VisibilityRequestBase: &manager.VisibilityRequestBase{
				NamespaceID:      namespaceID,
				Execution:        &commonpb.WorkflowExecution{WorkflowId: "same-instant", RunId: runID},
				WorkflowTypeName: "visibility-workflow",
				StartTime:        startTime,
				ExecutionTime:    startTime,
				Status:           enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
				TaskQueue:        "test-queue",
				TaskID:           s.base.taskID,
			},
		}))
	}
	executions := s.base.listWithPagination(namespaceID, 1)
	s.Require().Len(executions, 2)
	s.Require().Equal([]string{later, earlier}, []string{executions[0].GetExecution().GetRunId(), executions[1].GetExecution().GetRunId()})
}
