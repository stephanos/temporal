package umpire

import (
	"context"

	"github.com/google/uuid"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
)

func (p *PreparedCase) Run(ctx context.Context, host Host) (*umpirespb.Run, *umpirespb.Verdict, error) {
	driver, monitor, err := p.preflight(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	return execution.Run(ctx, p.program, driver, monitor, "umpire.run."+uuid.NewString(), p.source.GetCaseId())
}
