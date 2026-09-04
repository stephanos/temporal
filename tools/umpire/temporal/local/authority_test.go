package local

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestFactoryRejectsInvalidPreparationBeforeAuthorityStart(t *testing.T) {
	request := testRequest(t, "umpire.local.authority.invalid")
	crossed := testRequest(t, "umpire.local.authority.crossed")
	realize, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	crossedPrepare, ok := crossed.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)

	for _, test := range []struct {
		name    string
		command umpireruntime.Command
	}{
		{name: "wrong command", command: realize},
		{name: "crossed run", command: crossedPrepare},
	} {
		t.Run(test.name, func(t *testing.T) {
			starter := &recordingStarter{}

			environment, receipt := newFactory(starter).Prepare(
				context.Background(), request, test.command,
			)

			require.Nil(t, environment)
			require.Equal(t, umpireruntime.ReceiptUnsupported, receipt.Status())
			require.Empty(t, receipt.AcquiredResources())
			require.Equal(t, 0, starter.calls)
		})
	}
}

func TestFactoryFailsClosedWhenAuthorityStartReturnsNil(t *testing.T) {
	request := testRequest(t, "umpire.local.authority.nil")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "without error"},
		{name: "with error", err: errors.New("private authority startup failure")},
	} {
		t.Run(test.name, func(t *testing.T) {
			starter := &recordingStarter{err: test.err}
			environment, receipt := newFactory(starter).Prepare(
				context.Background(), request, prepare,
			)

			require.Nil(t, environment)
			require.Equal(t, umpireruntime.ReceiptFailed, receipt.Status())
			require.Empty(t, receipt.AcquiredResources())
			require.NotContains(t, receiptText(receipt), "private")
			require.Equal(t, 1, starter.calls)
		})
	}
}

func TestFactoryRetainsPartialAuthorityForCleanup(t *testing.T) {
	request := testRequest(t, "umpire.local.authority.partial")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{{kind: ownedEnvironment}},
	}
	starter := &recordingStarter{
		authority: backend,
		err:       errors.New("private partial startup failure"),
	}

	runtimeEnvironment, receipt := newFactory(starter).Prepare(context.Background(), request, prepare)

	require.Equal(t, umpireruntime.ReceiptFailed, receipt.Status())
	require.Equal(t, []umpireruntime.ResourceKind{
		umpireruntime.ResourceEnvironment,
	}, resourceKinds(receipt.AcquiredResources()))
	require.NotContains(t, receiptText(receipt), "private")
	require.Equal(t, 1, starter.calls)
	require.Equal(t, 0, backend.connectCalls)
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	cleanup, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	closed := environment.Cleanup(context.Background(), cleanup)
	require.Equal(t, umpireruntime.ReceiptAccepted, closed.Status())
	require.Equal(t, []string{"environment"}, backend.releaseOrder)
	require.Equal(t, "0", receiptField(closed, umpireruntime.EvidenceFieldOpenHandleCount))
}
