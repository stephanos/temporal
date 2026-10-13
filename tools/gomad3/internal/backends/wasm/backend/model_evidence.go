package backend

import (
	"encoding/json"
	"errors"

	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/model/volume"
)

// Keep the legacy raw-WASI projection byte-for-byte when no additional model
// owner produced evidence. The optional envelope binds deterministic durable
// and network effects, but does not invent an initial-image or crash bundle.
func modelEvidence(result wasi.Result, limit uint64) ([]byte, error) {
	if len(result.StorageTranscript) == 0 && len(result.StorageSnapshots) == 0 && len(result.NetworkTranscript) == 0 && result.StorageImage == nil {
		return result.Transcript, nil
	}
	image := ""
	if result.StorageImage != nil {
		image = result.StorageImage.Identity()
	}
	encoded, err := json.Marshal(struct {
		Schema                 string `json:"schema"`
		WASI, Network, Storage []byte
		StorageSnapshots       []volume.VolumeSnapshot
		StorageImageIdentity   string `json:"storage_image_identity,omitempty"`
	}{"gomad-wasm.modeled-evidence/v1", result.Transcript, result.NetworkTranscript, result.StorageTranscript, result.StorageSnapshots, image})
	if err != nil {
		return nil, err
	}
	if uint64(len(encoded)) > limit {
		return nil, errors.New("modeled WASM evidence byte capacity")
	}
	return encoded, nil
}
