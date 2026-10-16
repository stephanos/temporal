package backend

import (
	"encoding/json"
	"errors"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// Keep the legacy raw-WASI projection byte-for-byte when no additional model
// owner produced evidence. The optional envelope binds deterministic durable
// and network effects, but does not invent an initial-image or crash bundle.
func modelEvidence(result wasi.Result, limit uint64) ([]byte, error) {
	if len(result.StorageTranscript) == 0 && len(result.StorageSnapshots) == 0 && len(result.NetworkTranscript) == 0 && result.StorageImage == nil && len(result.InspectionMetadata) == 0 {
		return result.Transcript, nil
	}
	image := ""
	if result.StorageImage != nil {
		image = result.StorageImage.Identity()
	}
	schema := "gomad-wasm.modeled-evidence/v1"
	if len(result.InspectionMetadata) != 0 {
		schema = "gomad-wasm.modeled-evidence/v2"
	}
	encoded, err := json.Marshal(struct {
		Schema                 string `json:"schema"`
		WASI, Network, Storage []byte
		StorageSnapshots       []volume.VolumeSnapshot
		StorageImageIdentity   string `json:"storage_image_identity,omitempty"`
		Inspection             []byte `json:"inspection,omitempty"`
	}{schema, result.Transcript, result.NetworkTranscript, result.StorageTranscript, result.StorageSnapshots, image, result.InspectionMetadata})
	if err != nil {
		return nil, err
	}
	inspectionBytes := uint64(len(result.InspectionMetadata))
	var inspectionAllowance uint64
	if inspectionBytes != 0 {
		inspectionAllowance = 4*((inspectionBytes+2)/3) + 128
	}
	if inspectionBytes > 64<<20 || uint64(len(encoded)) > limit+inspectionAllowance {
		return nil, errors.New("modeled WASM evidence byte capacity")
	}
	return encoded, nil
}
