package cli

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/backends"
)

type optionalFlags struct {
	set     *flag.FlagSet
	enabled bool
}

func (flags optionalFlags) String(name, value, usage string) *string {
	result := new(string)
	*result = value
	if flags.enabled {
		flags.set.StringVar(result, name, value, usage)
	}
	return result
}

func (flags optionalFlags) Bool(name string, value bool, usage string) *bool {
	result := new(bool)
	*result = value
	if flags.enabled {
		flags.set.BoolVar(result, name, value, usage)
	}
	return result
}

func (flags optionalFlags) Uint64(name string, value uint64, usage string) *uint64 {
	result := new(uint64)
	*result = value
	if flags.enabled {
		flags.set.Uint64Var(result, name, value, usage)
	}
	return result
}

func (flags optionalFlags) Var(value flag.Value, name, usage string) {
	if flags.enabled {
		flags.set.Var(value, name, usage)
	}
}

type enumValue struct {
	value   *string
	allowed []string
}

func (value enumValue) String() string {
	if value.value == nil {
		return ""
	}
	return *value.value
}
func (value enumValue) Get() any { return *value.value }
func (value enumValue) Set(input string) error {
	if !slices.Contains(value.allowed, input) {
		return fmt.Errorf("must be one of %s", strings.Join(value.allowed, ", "))
	}
	*value.value = input
	return nil
}

func supportsCommand(capabilities *backends.Capabilities, command string) bool {
	return capabilities == nil || slices.Contains(capabilities.Commands, command)
}

func featureFlags(set *flag.FlagSet, capabilities *backends.Capabilities) (native, choices, simulation, mounts optionalFlags) {
	return optionalFlags{set, capabilities == nil || capabilities.IOTranscript},
		optionalFlags{set, capabilities == nil || len(capabilities.ChoiceProfiles) != 0},
		optionalFlags{set, capabilities == nil || capabilities.Simulation},
		optionalFlags{set, capabilities == nil || capabilities.ReadOnlyMounts}
}
