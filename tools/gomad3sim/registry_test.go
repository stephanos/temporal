package gomad3sim

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBootRegistry(t *testing.T) {
	id := uniqueBootID("registry-valid")
	boot := func(context.Context, NodeContext) error { return nil }
	require.NoError(t, RegisterBoot(id, boot))
	registered, ok := RegisteredBoot(id)
	require.True(t, ok)
	require.NotNil(t, registered)

	before := RegisteredBootIDs()
	require.Error(t, RegisterBoot(id, boot))
	require.Error(t, RegisterBoot("Bad ID", boot))
	require.Error(t, RegisterBoot(uniqueBootID("registry-nil"), nil))
	require.Equal(t, before, RegisteredBootIDs())

	before[0] = "changed"
	require.NotEqual(t, before, RegisteredBootIDs())
	_, ok = RegisteredBoot("missing")
	require.False(t, ok)
}

func TestBootRegistryConcurrentRegistration(t *testing.T) {
	const registrations = 16
	prefix := uniqueBootID("concurrent")
	var waitGroup sync.WaitGroup
	errors := make(chan error, registrations)
	for index := 0; index < registrations; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			id := BootID(fmt.Sprintf("%s-%02d", prefix, index))
			errors <- RegisterBoot(id, func(context.Context, NodeContext) error { return nil })
			RegisteredBootIDs()
		}(index)
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	ids := RegisteredBootIDs()
	require.True(t, slices.IsSorted(ids))
	for index := 0; index < registrations; index++ {
		_, ok := RegisteredBoot(BootID(fmt.Sprintf("%s-%02d", prefix, index)))
		require.True(t, ok)
	}
}
