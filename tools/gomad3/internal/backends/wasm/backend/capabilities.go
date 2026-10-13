package backend

import (
	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
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

func (p *Provider) Capabilities() backends.Capabilities {
	return CapabilitiesForProfile(p.options.Config.Profile)
}
