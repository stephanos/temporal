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

// storageStateMagic starts storage-state envelopes (see wasmhost protocol.md).
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
		limit = storageStateBudgetAdd(limit, count, bytes)
	}
	add(storageImageLimit(request.Config), 18)
	if enumeration := request.StorageCrashLimits; enumeration != nil {
		add(1, storageCrashContextLimit(request.Config)+12)
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

func storageStateBudgetAdd(limit, count, size uint64) uint64 {
	if count != 0 && size > (math.MaxUint64-limit)/count {
		return math.MaxUint64
	}
	return limit + count*size
}

// Complete images bound recovered history, but a partial envelope can contain
// more new history than fits in an image. Calls bounds new operations (two per
// call); Files bounds every allocated namespace node, including unlinked nodes.
// A rename has at most two paths per node and three parent/inode dependencies.
// Persisted snapshots may retain old names. Initial Files plus Calls*Files
// inserted paths fits 2*Calls*Files for valid Calls>=1, doubled for both trees.
func storagePartialStateLimit(request Request) uint64 {
	const operationFixedJSON = 361 // scalar/hash fields, null dependencies and array comma
	const snapshotFixedJSON = 195  // scalar/hash fields and array comma
	const numberJSON = 21          // uint64 decimal digits plus comma
	const escapedByteJSON = 6
	limit := storageStateLimit(request)
	calls, files := request.Config.Limits.Calls, request.Config.Limits.Files
	callFiles := storageStateBudgetAdd(0, calls, files)
	callFilesSquared := storageStateBudgetAdd(0, callFiles, files)
	limit = storageStateBudgetAdd(limit, calls, 2*(operationFixedJSON+3*numberJSON))
	limit = storageStateBudgetAdd(limit, callFiles, 2*2*escapedByteJSON*volume.MaximumPathBytes+2*2*numberJSON+4*snapshotFixedJSON)
	limit = storageStateBudgetAdd(limit, callFilesSquared, 4*escapedByteJSON*volume.MaximumPathBytes)
	for _, configured := range request.Config.Storage.Volumes {
		limit = storageStateBudgetAdd(limit, uint64(len(configured.ID)), 3*escapedByteJSON)
		limit = storageStateBudgetAdd(limit, uint64(len(configured.Path)), escapedByteJSON)
		limit = storageStateBudgetAdd(limit, 1, 1024)
	}
	return limit
}

func storageStateStreamLimit(request Request, previous, next []byte) uint64 {
	var header [16]byte
	n := copy(header[:], previous)
	n += copy(header[n:], next)
	if n == len(header) && string(header[:8]) == storageStateMagic && binary.LittleEndian.Uint32(header[8:]) == 2 && binary.LittleEndian.Uint32(header[12:]) == 6 {
		return storagePartialStateLimit(request)
	}
	return storageStateLimit(request)
}

// helperStorageRequest is the execute request's storage section in
// helper-owned storage runs.
type helperStorageRequest struct {
	Model          *StorageModel                  `json:"model"`
	ModelIdentity  string                         `json:"model_identity"`
	CrashLimits    *volume.CrashEnumerationLimits `json:"crash_limits,omitempty"`
	CrashSelection map[string][]uint64            `json:"crash_selection,omitempty"`
	CrashContext   *helperChunkedInput            `json:"crash_context,omitempty"`
	ReturnImage    bool                           `json:"return_image"`
	Image          *helperChunkedInput            `json:"image,omitempty"`
}

type helperChunkedInput struct {
	Chunks uint64 `json:"chunks"`
	Bytes  uint64 `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func helperStorageConfig(e *Environment, request Request) (*helperStorageRequest, []byte, []byte, error) {
	model, err := json.Marshal(e.config.Storage)
	if err != nil {
		return nil, nil, nil, err
	}
	storage := &helperStorageRequest{Model: e.config.Storage, ModelIdentity: storageModelIdentity(model), CrashLimits: request.StorageCrashLimits, CrashSelection: request.StorageCrashSelection, ReturnImage: request.ReturnStorageImage}
	var context []byte
	if request.StorageCrashContext != nil {
		context, err = json.Marshal(request.StorageCrashContext)
		if err != nil {
			return nil, nil, nil, err
		}
		digest := sha256.Sum256(context)
		storage.CrashContext = &helperChunkedInput{Chunks: (uint64(len(context)) + namespaceChunkBytes - 1) / namespaceChunkBytes, Bytes: uint64(len(context)), SHA256: hex.EncodeToString(digest[:])}
	}
	image := e.storageRecovery
	if image == nil {
		return storage, nil, context, nil
	}
	if image.data == nil {
		return nil, nil, nil, invalid("durable storage image")
	}
	if uint64(len(image.data)) > storageImageLimit(e.config) {
		return nil, nil, nil, capacity("storage-image-bytes")
	}
	size := uint64(len(image.data))
	storage.Image = &helperChunkedInput{Chunks: (size + namespaceChunkBytes - 1) / namespaceChunkBytes, Bytes: size, SHA256: image.identity}
	return storage, image.data, context, nil
}

func (sink *frameSink) sendStorageInput(stream string, data []byte) error {
	for sequence := 0; len(data) > 0; sequence++ {
		count := min(len(data), namespaceChunkBytes)
		if err := sink.send(struct {
			Type     string `json:"type"`
			Sequence int    `json:"sequence"`
			Data     []byte `json:"data_base64"`
		}{stream + "_input", sequence, data[:count]}); err != nil {
			return err
		}
		data = data[count:]
	}
	return sink.send(struct {
		Type string `json:"type"`
	}{stream + "_inputs_end"})
}

// helperStorageState is the decoded storage-state envelope.
type helperStorageState struct {
	snapshots     []volume.VolumeSnapshot
	operations    map[string][]volume.Operation
	enumerations  map[string]volume.CrashEnumeration
	imageIdentity string
	logLength     uint64
	logDigest     [32]byte
	failedStage   byte
	crashContext  map[string]StorageCrashContext
}

// decodeStorageState strictly decodes complete v1 and v3 (crash-context) and partial v2 envelopes. Truncation, trailing data,
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
	version, count := binary.LittleEndian.Uint32(reader), binary.LittleEndian.Uint32(reader[4:])
	if (version != 1 || count != 5) && (version != 2 || count != 6) && (version != 3 || count != 6) {
		return refuse("envelope version")
	}
	reader = reader[8:]
	sections := make([][]byte, count)
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
	if version == 2 {
		if len(sections[5]) != 1 || sections[5][0] < 1 || sections[5][0] > 2 {
			return refuse("collection failure stage")
		}
		state.failedStage = sections[5][0]
	}
	if version == 3 {
		if err := strictjson.Decode(sections[5], &state.crashContext); err != nil || state.crashContext == nil {
			return refuse("crash context")
		}
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
	if decoded, err := hex.DecodeString(identity); state.failedStage == 0 && (err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != identity) || state.failedStage != 0 && identity != "" {
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
	if (request.StorageCrashLimits == nil) != (state.enumerations == nil) || state.failedStage != 1 && state.enumerations != nil && len(state.enumerations) != len(ids) {
		return invalid("storage evidence: enumerations")
	}
	for id := range state.enumerations {
		if _, found := state.operations[id]; !found {
			return invalid("storage evidence: enumerations")
		}
	}
	if state.failedStage == 0 && (request.StorageCrashLimits == nil) != (state.crashContext == nil) {
		return invalid("storage evidence: crash context")
	}
	if state.crashContext != nil {
		contextRequest := Request{Config: request.Config, StorageCrashContext: state.crashContext}
		if err := ValidateStorageCrashSelection(contextRequest); err != nil || len(state.crashContext) != len(ids) {
			return invalid("storage evidence: crash context")
		}
		for _, snapshot := range state.snapshots {
			context, exists := state.crashContext[snapshot.ID]
			if !exists || context.PendingSHA256 != snapshot.PendingSHA256 || uint64(len(context.Operations)) != snapshot.PendingOperations {
				return invalid("storage evidence: crash context snapshot")
			}
			history := make(map[uint64]volume.Operation, len(state.operations[snapshot.ID]))
			for _, operation := range state.operations[snapshot.ID] {
				history[operation.ID] = operation
			}
			for _, operation := range context.Operations {
				if recorded, exists := history[operation.ID]; !exists || !storageOperationEqual(operation, recorded) {
					return invalid("storage evidence: crash context operation")
				}
			}
		}
	}
	if state.failedStage != 0 {
		if len(evidence.storageImage) != 0 || sink.terminal.Message == "" {
			return invalid("storage evidence: partial recovery image or terminal")
		}
		switch state.failedStage {
		case 1:
			if sink.terminal.Termination != "invalid" || request.StorageCrashLimits == nil || len(state.enumerations) >= len(ids) {
				return invalid("storage evidence: enumeration failure terminal")
			}
			for index, configured := range request.Config.Storage.Volumes {
				_, found := state.enumerations[configured.ID]
				if found != (index < len(state.enumerations)) {
					return invalid("storage evidence: enumeration failure prefix")
				}
			}
		case 2:
			if sink.terminal.Termination != "infrastructure" && (sink.terminal.Termination != "capacity" || sink.terminal.Message != "WASI capacity: storage-image-bytes") && (sink.terminal.Termination != "invalid" || sink.terminal.Message != "storage crash context diverged from prior observation" || request.StorageCrashContext == nil) {
				return invalid("storage evidence: image failure terminal")
			}
		}
		result.StorageTranscript = slices.Clone(evidence.storage)
		result.StorageSnapshots = state.snapshots
		result.StorageOperations = state.operations
		result.StorageCrashEnumeration = state.enumerations
		return nil
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
	result.StorageCrashContext = state.crashContext
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
