package primitive

import (
	"context"
	"errors"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestNilValue(t *testing.T) {
	var pointer *int
	var unsafePointer unsafe.Pointer
	var ctx context.Context
	for name, tc := range map[string]struct {
		value any
		want  bool
	}{
		"nil":                {value: nil, want: true},
		"nil pointer":        {value: pointer, want: true},
		"nil unsafe pointer": {value: unsafePointer, want: true},
		"nil map":            {value: map[string]int(nil), want: true},
		"nil context":        {value: ctx, want: true},
		"value":              {value: 0, want: false},
		"non-nil pointer":    {value: new(int), want: false},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, NilValue(tc.value))
		})
	}
}

func TestContextErrorReturnsCallerSentinel(t *testing.T) {
	sentinel := errors.New("caller sentinel")
	var typedNil *nilContext
	require.ErrorIs(t, ContextError(nil, sentinel), sentinel) //nolint:staticcheck // nil context is the case under test
	require.ErrorIs(t, ContextError(typedNil, sentinel), sentinel)
	require.NoError(t, ContextError(t.Context(), sentinel))
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, ContextError(canceled, sentinel), context.Canceled)
}

func TestMutexLockContext(t *testing.T) {
	sentinel := errors.New("caller sentinel")
	mutex := NewMutex()
	require.ErrorIs(t, mutex.LockContext(nil, sentinel), sentinel) //nolint:staticcheck // nil context is the case under test
	require.NoError(t, mutex.LockContext(t.Context(), sentinel))
	held, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, mutex.LockContext(held, sentinel), context.Canceled)
	mutex.Unlock()
	require.NoError(t, mutex.LockContext(t.Context(), sentinel))
}

type nilContext struct{ context.Context }
