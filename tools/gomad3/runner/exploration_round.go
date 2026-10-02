package runner

import (
	"context"
	"errors"
)

// collectRoundCompletions waits for every candidate of an exploration round
// and returns the completions in candidate order. cancel stops the candidates
// still running; subject names the strategy in an ordering failure.
func collectRoundCompletions(
	ctx context.Context,
	cancel context.CancelFunc,
	completionChannel <-chan runCompletion,
	startOrdinal uint64,
	candidates int,
	orderReason string,
	subject string,
) ([]runCompletion, error) {
	completions := make([]runCompletion, candidates)
	seen := make([]bool, candidates)
	received := 0
	var contextErr error
	for received < candidates {
		select {
		case completion := <-completionChannel:
			if completion.job.ordinal < startOrdinal || completion.job.ordinal >= startOrdinal+uint64(len(completions)) {
				cancel()
				return nil, &HostError{Reason: orderReason, Err: errors.New(subject + " completion ordinal is outside its round")}
			}
			index := int(completion.job.ordinal - startOrdinal)
			if seen[index] {
				cancel()
				return nil, &HostError{Reason: orderReason, Err: errors.New(subject + " completion ordinal is duplicated")}
			}
			completions[index] = completion
			seen[index] = true
			received++
		case <-ctx.Done():
			contextErr = ctx.Err()
			cancel()
			ctx = context.WithoutCancel(ctx)
		}
	}
	if contextErr != nil {
		return nil, &HostError{Reason: contextFailureReason(contextErr), Err: contextErr}
	}
	return completions, nil
}
