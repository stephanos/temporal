package testpilot

import (
	"context"

	"github.com/google/uuid"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
)

func (p *PreparedCase) Run(ctx context.Context, driver Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	executionDriver, monitor, err := p.preflight(ctx, driver)
	if err != nil {
		return nil, nil, err
	}
	return execution.Run(ctx, p.program, executionDriver, monitor, "testpilot.run."+uuid.NewString(), p.source.GetCaseId())
}
