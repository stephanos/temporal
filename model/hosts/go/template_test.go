//go:build test_dep

package temporal_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	model "github.com/temporalio/umpire/hosts/go/host"
	temporalmodel "umpire.local/temporal"
)

// guestTemplate is a guest prepared once per test process; tests receive clones of it.
type guestTemplate struct {
	once  sync.Once
	host  *model.Host
	value any
	err   error
}

var guestTemplates sync.Map // string -> *guestTemplate

// templateClone clones the process-wide guest prepared under key, with the value its
// preparation returned. The first admission of a three-attempt start computes its World
// inside the guest and dominates a fresh guest's cost; a clone copies that state instead of
// recomputing it. host.Clone exists only in test_dep builds.
func templateClone(t *testing.T, key string, prepare func(context.Context, *model.Host) (any, error)) (*model.Host, any) {
	t.Helper()
	entry, _ := guestTemplates.LoadOrStore(key, new(guestTemplate))
	template := entry.(*guestTemplate)
	template.once.Do(func() {
		ctx := context.Background()
		h, err := model.New(ctx, temporalmodel.Guest())
		if err == nil {
			template.value, err = prepare(ctx, h)
		}
		if err != nil {
			template.err = fmt.Errorf("prepare guest template %s: %w", key, err)
			return
		}
		template.host = h
	})
	if template.err != nil {
		t.Fatal(template.err)
	}
	h, err := template.host.Clone(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := h.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return h, template.value
}
