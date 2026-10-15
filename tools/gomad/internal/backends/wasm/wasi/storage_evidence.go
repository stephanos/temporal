package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"slices"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// storageStateMagic starts storage-state envelope v1 (see wasmhost protocol.md).
const storageStateMagic = "GMSTORE\x01"

// storageStateLimit bounds the envelope independently of its recovery image.
// Each snapshot tree and the operation history can use six JSON bytes per
// binary source byte (escaped UTF-8/control bytes, decimal numbers and record
// fields). All three are represented in the image's paths, objects and history.
// Enumeration Bytes counts raw state data, paths and 32-byte entry metadata;
// six times that also covers base64, decimal fields and JSON record syntax.
func storageStateLimit(request Request) uint64 {
	limit := uint64(256) // envelope framing, image identity and log binding
	add := func(count, bytes uint64) {
		if count > (math.MaxUint64-limit)/bytes {
			limit = math.MaxUint64
		} else {
			limit += count * bytes
		}
	}
	add(storageImageLimit(request.Config), 18)
	if enumeration := request.StorageCrashLimits; enumeration != nil {
		for _, configured := range request.Config.Storage.Volumes {
			add(enumeration.Bytes, 6)
			// A fresh enumeration retains at most one seen identity per
			// returned state and one cursor byte per pending operation.
			add(min(enumeration.States, enumeration.Work), 80)
			add(request.Config.Storage.Limits.PendingOperations, 2)
			// Escaped map/volume names, fixed exploration fields and padding.
			add(uint64(len(configured.ID)), 12)
			add(1, 1024)
		}
	}
	return limit
}

// helperStorageRequest is the execute request's storage section in
// helper-owned storage runs.
type helperStorageRequest struct {
	Model          *StorageModel                  `json:"model"`
	ModelIdentity  string                         `json:"model_identity"`
	CrashLimits    *volume.CrashEnumerationLimits `json:"crash_limits,omitempty"`
	CrashSelection map[string][]uint64            `json:"crash_selection,omitempty"`
	ReturnImage    bool                           `json:"return_image"`
	Image          *helperStorageImage            `json:"image,omitempty"`
}

type helperStorageImage struct {
	Chunks uint64 `json:"chunks"`
	Bytes  uint64 `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func helperStorageConfig(e *Environment, request Request) (*helperStorageRequest, []byte, error) {
	model, err := json.Marshal(e.config.Storage)
	if err != nil {
		return nil, nil, err
	}
	storage := &helperStorageRequest{Model: e.config.Storage, ModelIdentity: storageModelIdentity(model), CrashLimits: request.StorageCrashLimits, CrashSelection: request.StorageCrashSelection, ReturnImage: request.ReturnStorageImage}
	image := e.storageRecovery
	if image == nil {
		return storage, nil, nil
	}
	if image.data == nil {
		return nil, nil, invalid("durable storage image")
	}
	if uint64(len(image.data)) > storageImageLimit(e.config) {
		return nil, nil, capacity("storage-image-bytes")
	}
	size := uint64(len(image.data))
	storage.Image = &helperStorageImage{Chunks: (size + namespaceChunkBytes - 1) / namespaceChunkBytes, Bytes: size, SHA256: image.identity}
	return storage, image.data, nil
}

func (sink *frameSink) sendHelperStorageImage() error {
	data := sink.storageImage
	for sequence := 0; len(data) > 0; sequence++ {
		count := min(len(data), namespaceChunkBytes)
		if err := sink.send(struct {
			Type     string `json:"type"`
			Sequence int    `json:"sequence"`
			Data     []byte `json:"data_base64"`
		}{"storage_image_input", sequence, data[:count]}); err != nil {
			return err
		}
		data = data[count:]
	}
	return sink.send(struct {
		Type string `json:"type"`
	}{"storage_image_inputs_end"})
}

// helperStorageState is the decoded storage-state envelope.
type helperStorageState struct {
	snapshots     []volume.VolumeSnapshot
	operations    map[string][]volume.Operation
	enumerations  map[string]volume.CrashEnumeration
	imageIdentity string
	logLength     uint64
	logDigest     [32]byte
}

// decodeStorageState strictly decodes envelope v1. Truncation, trailing data,
// unknown versions, misordered sections, a length or digest mismatch, and
// JSON that does not decode exactly into the volume types are all refused.
func decodeStorageState(data []byte) (helperStorageState, error) {
	var state helperStorageState
	refuse := func(reason string) (helperStorageState, error) {
		return helperStorageState{}, invalid("storage evidence: " + reason)
	}
	if len(data) < len(storageStateMagic)+8+40 || string(data[:len(storageStateMagic)]) != storageStateMagic {
		return refuse("envelope header")
	}
	body := data[:len(data)-40]
	if binary.LittleEndian.Uint64(data[len(body):]) != uint64(len(body)) || sha256.Sum256(body) != [32]byte(data[len(body)+8:]) {
		return refuse("envelope length or digest")
	}
	reader := body[len(storageStateMagic):]
	if binary.LittleEndian.Uint32(reader) != 1 || binary.LittleEndian.Uint32(reader[4:]) != 5 {
		return refuse("envelope version")
	}
	reader = reader[8:]
	sections := make([][]byte, 5)
	for tag := range sections {
		if len(reader) < 12 || binary.LittleEndian.Uint32(reader) != uint32(tag+1) {
			return refuse("envelope sections")
		}
		size := binary.LittleEndian.Uint64(reader[4:])
		reader = reader[12:]
		if size > uint64(len(reader)) {
			return refuse("truncated envelope section")
		}
		sections[tag], reader = reader[:size], reader[size:]
	}
	if len(reader) != 0 {
		return refuse("trailing envelope bytes")
	}
	if err := strictjson.Decode(sections[0], &state.snapshots); err != nil {
		return refuse("snapshots")
	}
	if err := strictjson.Decode(sections[1], &state.operations); err != nil || state.operations == nil {
		return refuse("operations")
	}
	if !bytes.Equal(sections[2], []byte("null")) {
		if err := strictjson.Decode(sections[2], &state.enumerations); err != nil || state.enumerations == nil {
			return refuse("enumerations")
		}
	}
	identity := string(sections[3])
	if decoded, err := hex.DecodeString(identity); err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != identity {
		return refuse("image identity")
	}
	state.imageIdentity = identity
	if len(sections[4]) != 8+sha256.Size {
		return refuse("storage log binding")
	}
	state.logLength, state.logDigest = binary.LittleEndian.Uint64(sections[4]), [32]byte(sections[4][8:])
	return state, nil
}

// collectHelperStorage fills the storage results of a helper-owned storage
// run from the helper's evidence streams. Undecodable, truncated or
// inconsistent evidence is an `invalid` refusal, never a silent pass.
func (sink *frameSink) collectHelperStorage(result *Result) error {
	evidence := &sink.helperEvidence
	if len(evidence.storageState) == 0 {
		if len(evidence.storage) != 0 || len(evidence.storageImage) != 0 {
			return invalid("storage evidence without state")
		}
		switch sink.terminal.Termination {
		case "exit", "trap", "storage_crash", "crash", "stop":
			return invalid("missing helper storage evidence")
		}
		return nil
	}
	state, err := decodeStorageState(evidence.storageState)
	if err != nil {
		return err
	}
	if state.logLength != uint64(len(evidence.storage)) || state.logDigest != sha256.Sum256(evidence.storage) {
		return invalid("storage evidence: storage log length or digest")
	}
	request := sink.request
	ids := make([]string, 0, len(request.Config.Storage.Volumes))
	for _, configured := range request.Config.Storage.Volumes {
		ids = append(ids, configured.ID)
	}
	slices.Sort(ids)
	if len(state.snapshots) != len(ids) || len(state.operations) != len(ids) {
		return invalid("storage evidence: volume set")
	}
	for index, id := range ids {
		if state.snapshots[index].ID != id || state.operations[id] == nil {
			return invalid("storage evidence: volume set")
		}
	}
	if (request.StorageCrashLimits == nil) != (state.enumerations == nil) || state.enumerations != nil && len(state.enumerations) != len(ids) {
		return invalid("storage evidence: enumerations")
	}
	for id := range state.enumerations {
		if _, found := state.operations[id]; !found {
			return invalid("storage evidence: enumerations")
		}
	}
	model, err := json.Marshal(request.Config.Storage)
	if err != nil {
		return err
	}
	image := &StorageImage{identity: state.imageIdentity, model: model}
	if request.ReturnStorageImage {
		digest := sha256.Sum256(evidence.storageImage)
		if len(evidence.storageImage) == 0 || hex.EncodeToString(digest[:]) != state.imageIdentity {
			return invalid("storage evidence: recovery image")
		}
		image.data = slices.Clone(evidence.storageImage)
	} else if len(evidence.storageImage) != 0 {
		return invalid("storage evidence: unrequested recovery image")
	}
	result.StorageTranscript = slices.Clone(evidence.storage)
	result.StorageSnapshots = state.snapshots
	result.StorageOperations = state.operations
	result.StorageCrashEnumeration = state.enumerations
	result.StorageImage = image
	return nil
}

// storageCrashConfigured admits a helper-originated storage_crash terminal
// only for a configured crash-before fault, by its exact Go message.
func storageCrashConfigured(model *StorageModel, message string) bool {
	if model == nil {
		return false
	}
	for _, fault := range model.Faults {
		if fault.Action == "crash-before" && message == (&StorageCrash{Fault: fault.Name}).Error() {
			return true
		}
	}
	return false
}
