package testtier

import "fmt"

type Mode struct {
	Tiers   []string
	Success string
}

var modes = map[string]Mode{
	"test": {
		Tiers:   []string{"test-builder", "test-runtime", "test-upstream"},
		Success: "gomadv3 all black-box tiers passed",
	},
	"test-builder": {
		Tiers:   []string{"test-builder"},
		Success: "gomadv3 builder tier passed",
	},
	"test-runtime": {
		Tiers:   []string{"test-runtime"},
		Success: "gomadv3 runtime tier passed",
	},
	"test-upstream": {
		Tiers:   []string{"test-upstream"},
		Success: "gomadv3 upstream-compatibility tier passed",
	},
	"test-interception": {
		Tiers:   []string{"test-interception"},
		Success: "gomadv3 interception tier passed",
	},
}

func Resolve(name string) (Mode, error) {
	mode, found := modes[name]
	if !found {
		return Mode{}, fmt.Errorf("unknown gomadv3 test mode: %s", name)
	}
	mode.Tiers = append([]string(nil), mode.Tiers...)
	return mode, nil
}
