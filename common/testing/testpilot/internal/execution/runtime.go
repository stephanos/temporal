package execution

import (
	"context"
	"errors"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

func Run(
	ctx context.Context,
	program *PreparedProgram,
	driver Driver,
	monitor Monitor,
	runID string,
	caseID string,
) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	if ir.IsNil(ctx) || program == nil || ir.IsNil(driver) || ir.IsNil(monitor) || !ir.ValidID(runID) || !ir.ValidID(caseID) {
		return nil, nil, ir.Invalid(ir.Malformed, "execution", "context, prepared Program, Driver, Monitor and Run identity required")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	limits := program.limits
	runCtx, cancelRun := context.WithTimeout(ctx, time.Duration(limits.GetMaxDuration().AsDuration().Milliseconds())*time.Millisecond)
	session, err := driver.Open(runCtx, runID, program)
	if err != nil {
		cancelRun()
		return nil, nil, err
	}
	if ir.IsNil(session) {
		cancelRun()
		return nil, nil, ir.Invalid(ir.Malformed, "execution", "Driver returned no Session")
	}
	if err := runCtx.Err(); err != nil {
		return nil, nil, abandon(err, cancelRun, session, limits)
	}
	scheduler, err := newScheduler(program, runID, caseID, session, monitor, time.Now)
	if err != nil {
		return nil, nil, abandon(err, cancelRun, session, limits)
	}
	ordinaryErr := scheduler.execute(runCtx)
	abort := ordinaryErr != nil || scheduler.recorder.shouldAbort()
	terminationCtx, cancelTermination := freshContext(limits.GetCleanupDuration().AsDuration().Milliseconds())
	terminationErr := scheduler.settle(terminationCtx, scheduler.outstanding(), false, abort)
	cancelTermination()
	if terminationErr != nil {
		ordinaryErr = errors.Join(ordinaryErr, scheduler.fail("termination_failed", terminationErr))
	}
	disposition := scheduler.recorder.terminalStatus(ordinaryErr)

	cleanup := &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}
	cleanupStart := scheduler.ownedCount()
	cleanupCtx, cancelCleanup := freshContext(limits.GetCleanupDuration().AsDuration().Milliseconds())
	cleanupErr := scheduler.executeCleanup(cleanupCtx)
	cleanupSettleErr := scheduler.settle(cleanupCtx, scheduler.outstandingSince(cleanupStart), true, cleanupErr != nil)
	cancelCleanup()
	if cleanupErr = errors.Join(cleanupErr, cleanupSettleErr); cleanupErr != nil {
		cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED
		if id := scheduler.recorder.report(testpilotspb.RUN_DIAGNOSTIC_KIND_EXECUTION, "cleanup_failed", cleanupErr); id != "" {
			cleanup.DiagnosticIds = append(cleanup.DiagnosticIds, id)
		}
	}

	closeCtx, cancelClose := freshContext(limits.GetCleanupDuration().AsDuration().Milliseconds())
	closeErr := session.Close(closeCtx)
	if closeErr == nil {
		closeErr = closeCtx.Err()
	}
	cancelClose()
	if closeErr != nil {
		cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED
		if id := scheduler.recorder.report(testpilotspb.RUN_DIAGNOSTIC_KIND_DRIVER_CONTRACT, "driver_close_failed", closeErr); id != "" {
			cleanup.DiagnosticIds = append(cleanup.DiagnosticIds, id)
		}
	}
	cancelRun()
	scheduler.beginClose()
	verdictCtx, cancelVerdict := freshContext(limits.GetMaxDuration().AsDuration().Milliseconds())
	run, verdict, recorderErr := scheduler.recorder.close(verdictCtx, disposition, cleanup)
	cancelVerdict()
	scheduler.finishClose()
	// The recorder's own close failure is the caller's to see: the Run and Verdict it produced are
	// still the authoritative record, so they are returned beside the error rather than dropped.
	return run, verdict, recorderErr
}

// abandon ends a Run that failed before its schedule started: it cancels the Run and closes the
// Session under a fresh cleanup bound, joining the close error to err.
func abandon(err error, cancelRun context.CancelFunc, session contract.Session, limits *testpilotspb.ProgramLimits) error {
	cancelRun()
	closeCtx, cancelClose := freshContext(limits.GetCleanupDuration().AsDuration().Milliseconds())
	defer cancelClose()
	return errors.Join(err, session.Close(closeCtx))
}

func freshContext(milliseconds int64) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(milliseconds)*time.Millisecond)
}

func (p *PreparedProgram) cleanupGraph() *graph {
	for _, candidate := range p.graphs {
		if candidate.cleanup {
			return candidate
		}
	}
	return nil
}
