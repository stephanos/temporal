package target

import (
	"errors"
	"slices"

	"go.temporal.io/server/tools/gomad3/record"
)

type BackendPayload struct {
	Reference record.BackendPayload
	Data      []byte
}

func (prepared Prepared) CloneBackend() Prepared {
	prepared.Argv = slices.Clone(prepared.Argv)
	prepared.BuildTags = slices.Clone(prepared.BuildTags)
	prepared.Adapters = slices.Clone(prepared.Adapters)
	prepared.Compatibility = slices.Clone(prepared.Compatibility)
	prepared.BuildInfo.Settings = slices.Clone(prepared.BuildInfo.Settings)
	if prepared.CapabilityManifest != nil {
		manifest := *prepared.CapabilityManifest
		manifest.Payload = slices.Clone(manifest.Payload)
		prepared.CapabilityManifest = &manifest
	}
	prepared.Backend = record.CloneBackendMetadata(prepared.Backend)
	payloads := make([]BackendPayload, len(prepared.BackendPayloads))
	for index, payload := range prepared.BackendPayloads {
		payloads[index] = BackendPayload{Reference: payload.Reference, Data: append([]byte(nil), payload.Data...)}
	}
	if prepared.BackendPayloads != nil {
		prepared.BackendPayloads = payloads
	}
	return prepared
}

func (spec Spec) Clone() Spec {
	spec.Args = slices.Clone(spec.Args)
	spec.BuildTags = slices.Clone(spec.BuildTags)
	spec.AdapterReplacements = slices.Clone(spec.AdapterReplacements)
	return spec
}

func (prepared Prepared) ValidateBackendPayloads() error {
	if prepared.Backend == nil {
		return errors.New("external backend metadata is required")
	}
	if err := record.ValidateBackendMetadata(*prepared.Backend); err != nil {
		return err
	}
	references := record.BackendReferences(*prepared.Backend)
	if len(references) != len(prepared.BackendPayloads) {
		return errors.New("external backend payload set is incomplete")
	}
	for index, reference := range references {
		payload := prepared.BackendPayloads[index]
		if payload.Reference != reference || uint64(len(payload.Data)) != uint64(reference.Bytes) || record.HashBytes(payload.Data) != reference.SHA256 {
			return errors.New("external backend payload bytes do not match their reference")
		}
	}
	return nil
}
