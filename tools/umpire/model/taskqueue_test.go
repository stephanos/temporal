package model

// An independent consumer of the shared task queue, lifted from
// model/irgen/testdata/lifts/TaskQueue.scala: a job of its own over the opaque queue, over the
// matching provider that replaces it, and over the forgetful provider, the negative control. It names
// nothing of the standalone activity, so these answers are the queue's as any feature relies on it.

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

func taskQueueModel(t *testing.T) *checkedModel {
	t.Helper()
	c, err := checkedOnce(lifted(t, "taskqueue"))
	require.NoError(t, err)
	return c
}

// Every declaration has one answer: the job keeps its promise over the opaque and the matching
// queue, and every check over the forgetful provider is its rejected replacement.
func TestTaskQueueResults(t *testing.T) {
	c := taskQueueModel(t)
	require.Empty(t, c.report.Unsupported())
	require.Equal(t, map[string]ReceiptKind{
		"refinement taskQueueSystem taskQueueProduct":  Verified,
		"refinement forgetfulQueue taskQueueProduct": RefinementRejected,
		"composition jobOverMatching":             Verified,
		"composition jobOverForgetful":            RefinementRejected,

		"query jobOverQueue jobOverQueue.duplicateDelivery":                Found,
		"query jobOverQueue jobOverQueue.any.settledLeavesNothing":         Verified,
		"query jobOverMatching jobOverMatching.crashAfterInvocation":       Found,
		"query jobOverMatching jobOverMatching.any.settledLeavesNothing":   Verified,
		"query jobOverForgetful jobOverForgetful.crashAfterInvocation":     RefinementRejected,
		"query jobOverForgetful jobOverForgetful.any.settledLeavesNothing": RefinementRejected,
	}, kinds(c.report))
}

// The queue may hand one message out twice: the job meets the second delivery started, and settles.
// Over the matching provider a crash after the invocation loses nothing durable, so history invokes
// AddActivityTask again and the job still settles.
func TestTaskQueueWitnesses(t *testing.T) {
	c := taskQueueModel(t)
	duplicate := receiptOf(t, c.report, "query jobOverQueue jobOverQueue.duplicateDelivery")
	require.Equal(t, []string{"send", "start", "start", "settle"}, taken(duplicate.Witness))
	require.Equal(t, "settled_empty", last(t, duplicate.Witness).State.Value)

	crashed := receiptOf(t, c.report, "query jobOverMatching jobOverMatching.crashAfterInvocation")
	require.Equal(t, []string{"send", "queue_addActivityTask", "queue_crash", "queue_addActivityTask",
		"queue_persistTask", "start", "settle"}, taken(crashed.Witness))
	require.Equal(t, "settled_nowhere-false-never", last(t, crashed.Witness).State.Value)
}

// The forgetful provider drops history's task at the invocation, so a crash there leaves a message the
// interface calls committed with no custodian. Each check over it is that rejection, never an answer
// of its own.
func TestTaskQueueForgetfulProvider(t *testing.T) {
	c := taskQueueModel(t)
	rejected := receiptOf(t, c.report, "refinement forgetfulQueue taskQueueProduct")
	require.Equal(t, umpire.RefinementUnmatched, rejected.Failure)
	require.Equal(t, []string{"enqueue", "addActivityTask", "crash"}, taken(rejected.Witness))
	require.Equal(t, "nowhere-false-never", last(t, rejected.Witness).State.Value)
	for _, key := range []string{"composition jobOverForgetful",
		"query jobOverForgetful jobOverForgetful.crashAfterInvocation",
		"query jobOverForgetful jobOverForgetful.any.settledLeavesNothing"} {
		r := receiptOf(t, c.report, key)
		require.Equal(t, rejected.Failure, r.Failure, key)
		require.Equal(t, rejected.Witness, r.Witness, key)
	}
}
