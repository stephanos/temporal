package backend

import (
	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
)

func CapabilitiesForProfile(profile string) backends.Capabilities {
	capabilities := backends.Capabilities{
		Name: "wasm", ReplayModes: []string{record.ReplayObserved},
		Commands: []string{"install", "discover", "explore", "replay", "resume", "recover", "inspect", "minimize", "qualify", "report"},
	}
	if profile == wasi.CooperativeProfile {
		capabilities.ChoiceProfiles = []string{choice.Profile}
		capabilities.ReplayModes = []string{record.ReplayExact, record.ReplayObserved}
	}
	return capabilities
}

func (provider *Provider) Capabilities() backends.Capabilities {
	return CapabilitiesForProfile(provider.options.Config.Profile)
}
