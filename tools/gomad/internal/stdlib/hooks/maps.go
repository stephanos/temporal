package hooks

import (
	"github.com/temporalio/gomad/gomadruntime"
)

func Maps_clone(m any) any {
	return gomadruntime.CloneMap(m)
}
