package veil

import (
	"bytes"
	_ "embed"

	"go.temporal.io/server/tests/umpire3/protocol"
)

//go:embed bindings/nexus-cancellation-mutated.json
var defaultMutatedBindingJSON []byte

//go:embed results/nexus-cancellation-mutated-concrete.json
var defaultMutatedResultJSON []byte

func DefaultMutatedBinding() (BindingArtifact, error) {
	return DecodeBindingArtifact(bytes.NewReader(defaultMutatedBindingJSON), protocol.DefaultDecodeLimit)
}

func DefaultMutatedResult() (protocol.BackendResult, error) {
	return protocol.DecodeBackendResult(bytes.NewReader(defaultMutatedResultJSON), protocol.DefaultDecodeLimit)
}
