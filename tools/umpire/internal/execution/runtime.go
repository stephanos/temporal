package execution

import (
	"context"
	"errors"
	"time"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
)

func Run(
	ctx context.Context,
	program *PreparedProgram,
	driver Driver,
	monitor Monitor,
	runID string,
	caseID string,
) (*umpirespb.Run, *umpirespb.Verdict, error) {
	if isNil(ctx) || program == nil || isNil(driver) || isNil(monitor) || !validID(runID) || !validID(caseID) {
		return nil, nil, invalid(ir.Malformed, "execution", "context, prepared Program, Driver, Monitor and Run identity required")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	limits := program.source.GetLimits()
	runCtx, cancelRun := context.WithTimeout(ctx, time.Duration(limits.GetMaxTotalDurationMilliseconds())*time.Millisecond)
	session, err := driver.Open(runCtx, runID, program)
	if err != nil {
		cancelRun()
		return nil, nil, err
	}
	if isNil(session) {
		cancelRun()
		return nil, nil, invalid(ir.Malformed, "execution", "Driver returned no Session")
	}
	if err := runCtx.Err(); err != nil {
		cancelRun()
		closeCtx, cancelClose := freshContext(limits.GetMaxCleanupDurationMilliseconds())
		closeErr := session.Close(closeCtx)
		cancelClose()
		return nil, nil, errors.Join(err, closeErr)
	}
	scheduler, err := newScheduler(program, runID, caseID, session, monitor, time.Now)
	if err != nil {
		cancelRun()
		closeCtx, cancelClose := freshContext(limits.GetMaxCleanupDurationMilliseconds())
		closeErr := session.Close(closeCtx)
		cancelClose()
		return nil, nil, errors.Join(err, closeErr)
	}
	ordinaryErr := scheduler.execute(runCtx)
	abort := ordinaryErr != nil || scheduler.recorder.shouldAbort()
	terminationCtx, cancelTermination := freshContext(limits.GetMaxCleanupDurationMilliseconds())
	terminationErr := scheduler.settle(terminationCtx, scheduler.outstanding(), false, abort)
	cancelTermination()
	if terminationErr != nil {
		ordinaryErr = errors.Join(ordinaryErr, scheduler.fail("termination_failed", terminationErr))
	}
	disposition := scheduler.recorder.terminalDisposition(ordinaryErr)

	cleanup := &umpirespb.CleanupOutcome{Status: umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED}
	cleanupStart := scheduler.ownedCount()
	cleanupCtx, cancelCleanup := freshContext(limits.GetMaxCleanupDurationMilliseconds())
	cleanupErr := scheduler.executeCleanup(cleanupCtx)
	cleanupSettleErr := scheduler.settle(cleanupCtx, scheduler.outstandingSince(cleanupStart), true, cleanupErr != nil)
	cancelCleanup()
	if cleanupErr = errors.Join(cleanupErr, cleanupSettleErr); cleanupErr != nil {
		cleanup.Status = umpirespb.RUN_CLEANUP_STATUS_FAILED
		if id := scheduler.recorder.report(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "cleanup_failed", cleanupErr); id != "" {
			cleanup.DiagnosticIds = append(cleanup.DiagnosticIds, id)
		}
	}

	closeCtx, cancelClose := freshContext(limits.GetMaxCleanupDurationMilliseconds())
	closeErr := session.Close(closeCtx)
	if closeErr == nil {
		closeErr = closeCtx.Err()
	}
	cancelClose()
	if closeErr != nil {
		scheduler.recorder.report(umpirespb.RUN_DIAGNOSTIC_KIND_HOST_CONTRACT, "host_close_failed", closeErr)
	}
	cancelRun()
	scheduler.beginClose()
	verdictCtx, cancelVerdict := freshContext(limits.GetMaxTotalDurationMilliseconds())
	run, verdict, recorderErr := scheduler.recorder.close(verdictCtx, disposition, cleanup)
	cancelVerdict()
	scheduler.finishClose()
	_ = recorderErr
	return run, verdict, nil
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
