package runner

import (
	"context"
	"errors"
)

// collectRoundCompletions waits for every candidate of an exploration round
// and returns the completions in candidate order. cancel stops the candidates
// still running; subject names the strategy in an ordering failure.
//
// A parent context that ended fails the round with that end's classification.
// It is read after the last completion, not from the receive that happened to
// win: a candidate the end interrupted reports an error of its own, and a
// round that received it first would pass it on as a supervision failure.
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
	parentDone := ctx.Done()
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
		case <-parentDone:
			cancel()
			parentDone = nil
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	return completions, nil
}
