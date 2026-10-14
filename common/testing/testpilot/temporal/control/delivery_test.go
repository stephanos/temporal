package control

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeliveryGateHoldsOnlyItsDeclaredDelivery(t *testing.T) {
	gate := NewDeliveryGate("delivery")
	defer gate.Close()
	require.NoError(t, gate.Arrive(t.Context(), "other"))
	done := make(chan error, 1)
	go func() { done <- gate.Arrive(t.Context(), "delivery") }()
	require.NoError(t, gate.WaitHeld(t.Context()))
	select {
	case err := <-done:
		t.Fatalf("delivery crossed the hold before release: %v", err)
	default:
	}
	require.NoError(t, gate.Release())
	require.NoError(t, <-done)
	require.NoError(t, gate.Arrive(t.Context(), "delivery"))
	require.NoError(t, gate.Release())
}

func TestDeliveryGateRejectsReleaseBeforeArrival(t *testing.T) {
	gate := NewDeliveryGate("delivery")
	defer gate.Close()
	require.ErrorIs(t, gate.Release(), ErrNotHeld)
}

func TestDeliveryGateCancellationDoesNotReleaseDelivery(t *testing.T) {
	gate := NewDeliveryGate("delivery")
	defer gate.Close()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- gate.Arrive(ctx, "delivery") }()
	require.NoError(t, gate.WaitHeld(t.Context()))
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.NoError(t, gate.Release())
	require.ErrorIs(t, gate.Arrive(ctx, "delivery"), context.Canceled)
}

func TestDeliveryGateCloseUnblocksBothWaits(t *testing.T) {
	for _, arrived := range []bool{false, true} {
		t.Run(map[bool]string{false: "before arrival", true: "after arrival"}[arrived], func(t *testing.T) {
			gate := NewDeliveryGate("delivery")
			done := make(chan error, 1)
			if arrived {
				go func() { done <- gate.Arrive(t.Context(), "delivery") }()
				require.NoError(t, gate.WaitHeld(t.Context()))
			} else {
				go func() { done <- gate.WaitHeld(t.Context()) }()
			}
			gate.Close()
			gate.Close()
			require.ErrorIs(t, <-done, ErrClosed)
			require.ErrorIs(t, gate.Arrive(t.Context(), "delivery"), ErrClosed)
			require.ErrorIs(t, gate.WaitHeld(t.Context()), ErrClosed)
			require.ErrorIs(t, gate.Release(), ErrClosed)
		})
	}
}

func TestDeliveryGateWaitHonorsCanceledContext(t *testing.T) {
	gate := NewDeliveryGate("delivery")
	defer gate.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, gate.WaitHeld(ctx), context.Canceled)
}

func TestDeliveryGateRejectsIncomparableIdentity(t *testing.T) {
	for _, test := range []struct {
		name                string
		declared, delivered any
	}{
		{"declared", []int{1}, "delivery"},
		{"delivered", "delivery", []int{1}},
		{"both", []int{1}, []int{1}},
		{"nested", struct{ Key any }{[]int{1}}, struct{ Key any }{[]int{1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			gate := NewDeliveryGate(test.declared)
			defer gate.Close()
			require.ErrorIs(t, gate.Arrive(t.Context(), test.delivered), ErrInvalid)
		})
	}
}
