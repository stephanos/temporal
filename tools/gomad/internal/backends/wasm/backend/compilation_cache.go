package backend

import (
	"encoding/json"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

type compilationInputs struct {
	Profile, CompilerSHA256, GoVersion, RuntimeSHA256, RuntimeImplementationSHA256 string
	Sources                                                                        []sourceIdentity
	Kind                                                                           target.Kind
	Source                                                                         string
	BuildTags, BuildFlags, BuildEnvironment                                        []string
}

type compilationReceipt struct {
	Schema, BuildKey, ModuleSHA256  string
	Inputs                          compilationInputs
	Imports                         []wasi.Import
	InitialMemoryPages, ModuleBytes uint64
}

func compilationFor(pr provenance) (compilationReceipt, error) {
	inputs := compilationInputs{
		Profile: pr.Profile, CompilerSHA256: pr.CompilerSHA256, GoVersion: pr.GoVersion,
		RuntimeSHA256: pr.RuntimeSHA256, RuntimeImplementationSHA256: pr.RuntimeImplementationSHA256,
		Sources: pr.Sources, Kind: pr.Kind, Source: pr.Source,
		BuildTags: pr.BuildTags, BuildFlags: pr.BuildFlags, BuildEnvironment: pr.BuildEnvironment,
	}
	data, err := json.Marshal(inputs)
	if err != nil {
		return compilationReceipt{}, err
	}
	return compilationReceipt{
		Schema: "gomad-wasm.compilation/v1", BuildKey: strings.TrimPrefix(string(record.HashBytes(data)), "sha256:"), Inputs: inputs,
		ModuleSHA256: pr.ModuleSHA256, ModuleBytes: pr.ModuleBytes, Imports: pr.Imports, InitialMemoryPages: pr.InitialMemoryPages,
	}, nil
}
