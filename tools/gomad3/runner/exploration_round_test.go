package runner

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

// finalCompletionFirst is the parent context a round sees when its final
// completion is received ahead of an end the parent has already reached: Err
// reports the end and Done is never ready.
type finalCompletionFirst struct{ context.Context }

func (finalCompletionFirst) Done() <-chan struct{} { return nil }

// interruptedCandidate is the completion of a candidate its parent's end
// interrupted: a cancelled result beside the journal's refusal to advance.
func interruptedCandidate(ordinal uint64, cause error) runCompletion {
	return runCompletion{job: runJob{ordinal: ordinal}, result: execution.Result{Cancelled: true}, err: errors.Join(cause)}
}

// A round whose parent context ended fails with that end's classification,
// whichever of the parent's Done channel and the final completion the round
// receives first.
func TestRoundClassifiesAParentEndIndependentOfReceiveOrder(t *testing.T) {
	const start = 7
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, release := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer release()
	for _, parent := range []struct {
		name   string
		ctx    context.Context
		reason string
	}{
		{name: "cancelled", ctx: cancelled, reason: "cancelled"},
		{name: "overall timeout", ctx: expired, reason: "overall_timeout"},
	} {
		completion := interruptedCandidate(start, parent.ctx.Err())
		for _, order := range []struct {
			name string
			ctx  context.Context
			// reported is whether the candidate has reported before the round
			// stops it; otherwise it reports once the round does.
			reported bool
		}{
			{name: "final completion first", ctx: finalCompletionFirst{parent.ctx}, reported: true},
			{name: "parent end first", ctx: parent.ctx},
		} {
			t.Run(parent.name+"/"+order.name, func(t *testing.T) {
				completions := make(chan runCompletion, 1)
				stopped := 0
				stop := func() {
					stopped++
					if !order.reported {
						completions <- completion
					}
				}
				if order.reported {
					completions <- completion
				}
				collected, err := collectRoundCompletions(order.ctx, stop, completions, start, 1, "exploration_order", "exploration")
				if want := (&HostError{Reason: parent.reason, Err: parent.ctx.Err()}); collected != nil || !reflect.DeepEqual(err, want) {
					t.Fatalf("round = %d completions and %v, want none and %v", len(collected), err, want)
				}
				if want := map[bool]int{true: 0, false: 1}[order.reported]; stopped != want {
					t.Fatalf("round stopped its candidates %d times, want %d", stopped, want)
				}
			})
		}
	}
}

// A round whose parent context is live returns each candidate's own outcome
// for its strategy to classify, and rejects a completion its candidates
// cannot have produced.
func TestRoundKeepsCandidateOutcomesUnderALiveParent(t *testing.T) {
	const start = 7
	failed := runCompletion{job: runJob{ordinal: start + 1}, err: errors.New("executor failed")}
	succeeded := runCompletion{job: runJob{ordinal: start}}
	for _, test := range []struct {
		name     string
		reported []runCompletion
		want     []runCompletion
		err      error
		stopped  int
	}{
		{name: "supervision failure", reported: []runCompletion{failed, succeeded}, want: []runCompletion{succeeded, failed}},
		{
			name: "ordinal before the round", reported: []runCompletion{{job: runJob{ordinal: start - 1}}, succeeded},
			err: &HostError{Reason: "exploration_order", Err: errors.New("exploration completion ordinal is outside its round")}, stopped: 1,
		},
		{
			name: "ordinal after the round", reported: []runCompletion{{job: runJob{ordinal: start + 2}}, succeeded},
			err: &HostError{Reason: "exploration_order", Err: errors.New("exploration completion ordinal is outside its round")}, stopped: 1,
		},
		{
			name: "repeated ordinal", reported: []runCompletion{succeeded, succeeded},
			err: &HostError{Reason: "exploration_order", Err: errors.New("exploration completion ordinal is duplicated")}, stopped: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			completions := make(chan runCompletion, len(test.reported))
			for _, completion := range test.reported {
				completions <- completion
			}
			stopped := 0
			collected, err := collectRoundCompletions(context.Background(), func() { stopped++ }, completions, start, 2, "exploration_order", "exploration")
			if !reflect.DeepEqual(collected, test.want) || !reflect.DeepEqual(err, test.err) || stopped != test.stopped {
				t.Fatalf("round = %#v, %#v after %d stops, want %#v, %#v after %d", collected, err, stopped, test.want, test.err, test.stopped)
			}
		})
	}
}
