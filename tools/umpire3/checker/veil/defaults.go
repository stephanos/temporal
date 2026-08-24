package veil

import (
	"bytes"
	_ "embed"

	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

//go:embed testdata/generated/nexus-cancellation-mutated.json
var defaultMutatedBindingJSON []byte

//go:embed testdata/retained/nexus-cancellation-mutated-concrete.json
var defaultMutatedResultJSON []byte

func DefaultMutatedBinding() (BindingArtifact, error) {
	return DecodeBindingArtifact(bytes.NewReader(defaultMutatedBindingJSON), protocolexperiment.DefaultDecodeLimit)
}

func DefaultMutatedResult() (protocolchecker.BackendResult, error) {
	return protocolchecker.DecodeBackendResult(bytes.NewReader(defaultMutatedResultJSON), protocolexperiment.DefaultDecodeLimit)
}
