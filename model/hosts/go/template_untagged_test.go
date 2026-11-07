//go:build !test_dep

package temporal_test

import (
	"context"
	"testing"

	model "github.com/temporalio/umpire/hosts/go/host"
)

// templateClone prepares a fresh guest on every call: host.Clone exists only in test_dep
// builds.
func templateClone(t *testing.T, key string, prepare func(context.Context, *model.Host) (any, error)) (*model.Host, any) {
	t.Helper()
	h := unboundGuest(t)
	value, err := prepare(context.Background(), h)
	if err != nil {
		t.Fatalf("prepare guest %s: %v", key, err)
	}
	return h, value
}
