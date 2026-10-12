package simulation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

const RetainedSchema = "gomad3.wasm-simulation-retained/v1"

var ErrRetentionCapacity = errors.New("WASM retained execution capacity exceeded")

type RetentionCapacityError struct{ Required, Maximum uint64 }

func (e *RetentionCapacityError) Error() string {
	return fmt.Sprintf("%v: required=%d maximum=%d", ErrRetentionCapacity, e.Required, e.Maximum)
}
func (e *RetentionCapacityError) Unwrap() error { return ErrRetentionCapacity }

// These byte fields are base64 encoded by the existing JSON owner. Reject a
// known oversized bundle before reading or marshaling its executable inputs.
func retainedCapacityLowerBound(recordBytes uint64, inputBytes []uint64) error {
	const maximum = uint64(512 << 20)
	required := uint64(0)
	for _, size := range append([]uint64{recordBytes}, inputBytes...) {
		if size > maximum {
			return &RetentionCapacityError{Required: size, Maximum: maximum}
		}
		required += 4 * ((size + 2) / 3)
		if required > maximum {
			return &RetentionCapacityError{Required: required, Maximum: maximum}
		}
	}
	return nil
}

type retainedBoot struct {
	ID      gomad3sim.BootID `json:"id"`
	Request wasi.Request     `json:"request"`
}
type retainedExecution struct {
	Schema            string                          `json:"schema"`
	Implementation    string                          `json:"implementation_sha256"`
	Spec              gomad3sim.Spec                  `json:"spec"`
	Record            []byte                          `json:"record"`
	Broker            wasi.BrokerConfig               `json:"broker"`
	OutputCrashes     []OutputCrash                   `json:"output_crashes"`
	OutputStops       []OutputCrash                   `json:"output_stops"`
	IncarnationBoots  []IncarnationBoot               `json:"incarnation_boots"`
	OutputCrashGroups []OutputCrashGroup              `json:"output_crash_groups"`
	Oracles           *OraclePlan                     `json:"oracles"`
	Lifecycle         []gomad3sim.LifecycleTransition `json:"lifecycle"`
	Boots             []retainedBoot                  `json:"boots"`
	Inputs            map[string][]byte               `json:"inputs"`
}

// RetentionPayload produces an existing Artifact backend evidence payload. It
// captures verified executable inputs; no source or compiler is needed to reopen.
func (p *Provider) RetentionPayload(spec gomad3sim.Spec, observed gomad3sim.ClusterRecord) (target.BackendPayload, error) {
	if p == nil || observed.Backend != gomad3sim.BackendWASM || observed.WASM == nil {
		return target.BackendPayload{}, errors.New("WASM retention requires completed broker evidence")
	}
	encoded, err := gomad3sim.EncodeClusterRecord(observed)
	if err != nil {
		return target.BackendPayload{}, err
	}
	broker := p.Broker
	broker.Diagnostics = false
	broker.Nodes = nil
	broker.Addresses = nil
	spec.Replay = nil
	bundle := retainedExecution{Schema: RetainedSchema, Implementation: ImplementationSHA256(), Spec: spec, Record: encoded, Broker: broker, OutputCrashes: p.OutputCrashes, OutputStops: p.OutputStops, IncarnationBoots: p.IncarnationBoots, OutputCrashGroups: p.OutputCrashGroups, Oracles: p.Oracles, Lifecycle: observed.Transitions, Inputs: map[string][]byte{}}
	ids := make([]gomad3sim.BootID, 0, len(p.Boots))
	for id := range p.Boots {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	seen := map[string]bool{}
	var inputSizes []uint64
	for _, id := range ids {
		r := p.Boots[id]
		for _, input := range []struct{ path, digest string }{{r.ModulePath, r.ModuleSHA256}, {r.HelperPath, r.HelperSHA256}} {
			if seen[input.digest] {
				continue
			}
			info, err := os.Stat(input.path)
			if err != nil {
				return target.BackendPayload{}, err
			}
			if !info.Mode().IsRegular() || info.Size() < 0 {
				return target.BackendPayload{}, errors.New("WASM retained execution input is not a regular file")
			}
			seen[input.digest] = true
			inputSizes = append(inputSizes, uint64(info.Size()))
		}
	}
	if err := retainedCapacityLowerBound(uint64(len(encoded)), inputSizes); err != nil {
		return target.BackendPayload{}, err
	}
	for _, id := range ids {
		r := p.Boots[id]
		if r.Runtime == nil || r.Runtime.Choice == nil || r.StorageImage != nil || r.Broker != nil {
			return target.BackendPayload{}, errors.New("WASM retention requires fresh bound boot inputs")
		}
		for _, input := range []struct{ path, digest string }{{r.ModulePath, r.ModuleSHA256}, {r.HelperPath, r.HelperSHA256}} {
			if _, ok := bundle.Inputs[input.digest]; ok {
				continue
			}
			data, err := hostfs.ReadBounded(input.path, 512<<20)
			if err != nil {
				return target.BackendPayload{}, err
			}
			if fmt.Sprintf("%x", sha256.Sum256(data)) != input.digest {
				return target.BackendPayload{}, errors.New("WASM execution input changed before retention")
			}
			bundle.Inputs[input.digest] = data
		}
		r.ModulePath, r.HelperPath, r.CompileCache = "", "", ""
		r.StdoutMirror = nil
		bundle.Boots = append(bundle.Boots, retainedBoot{ID: id, Request: r})
	}
	data, err := json.Marshal(bundle)
	if err != nil {
		return target.BackendPayload{}, err
	}
	if len(data) > 512<<20 {
		return target.BackendPayload{}, &RetentionCapacityError{Required: uint64(len(data)), Maximum: 512 << 20}
	}
	return target.BackendPayload{Reference: record.BackendPayload{Schema: RetainedSchema, File: "backend/simulation.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}, Data: data}, nil
}

// ReopenProvider verifies retained detached inputs before creating fresh helper
// files. Only the retained module/helper bytes are used, never original paths.
func verifyRetained(payload target.BackendPayload) (retainedExecution, *Provider, gomad3sim.ClusterRecord, error) {
	bundle, err := decodeRetained(payload)
	if err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, err
	}
	if bundle.Implementation != ImplementationSHA256() {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("retained WASM model implementation differs from current owners")
	}
	observed, err := gomad3sim.DecodeClusterRecord(bundle.Record)
	if err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, err
	}
	if observed.Backend != gomad3sim.BackendWASM || observed.WASM == nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("retained payload lacks WASM execution evidence")
	}
	actions, _ := json.Marshal(bundle.Lifecycle)
	expectedActions, _ := json.Marshal(observed.Transitions)
	if !bytes.Equal(actions, expectedActions) {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("retained WASM actions differ from observed lifecycle")
	}
	plan, err := gomad3sim.ReplayPlanFor(observed)
	if err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, err
	}
	bundle.Spec.Replay = &plan
	if err := gomad3sim.ValidateSpec(bundle.Spec); err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, err
	}
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{}, Broker: bundle.Broker, OutputCrashes: bundle.OutputCrashes, OutputStops: bundle.OutputStops, IncarnationBoots: bundle.IncarnationBoots, OutputCrashGroups: bundle.OutputCrashGroups, Oracles: bundle.Oracles}
	for _, boot := range bundle.Boots {
		if boot.ID == "" || p.Boots[boot.ID].Runtime != nil || boot.Request.Runtime == nil || boot.Request.Runtime.Choice == nil || boot.Request.ModulePath != "" || boot.Request.HelperPath != "" || boot.Request.Broker != nil || boot.Request.StorageImage != nil {
			return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("invalid retained WASM boot")
		}
		p.Boots[boot.ID] = boot.Request
		for _, digest := range []string{boot.Request.ModuleSHA256, boot.Request.HelperSHA256} {
			data, ok := bundle.Inputs[digest]
			if !ok || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
				return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("retained WASM module/helper identity mismatch")
			}
		}
	}
	for _, n := range bundle.Spec.Nodes {
		r, ok := p.Boots[n.Boot]
		if !ok {
			return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("retained WASM node lacks boot")
		}
		identity, err := BootConfiguration(r)
		if err != nil || !bytes.Equal(identity, n.Config) {
			return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, errors.New("retained WASM configuration mismatch")
		}
	}
	// Reuse provider admission for every incarnation configuration before any
	// retained executable is written or a helper can start.
	admitted, err := p.Open(context.Background(), bundle.Spec)
	if err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, err
	}
	config := admitted.(*execution).provider.Broker
	if err := verifyWorldEvidence(observed.WASM, config.WorldProfile, world.Config{Seed: world.Seed(config.Seed), Limits: config.Limits}); err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, fmt.Errorf("retained WASM World owner evidence: %w", err)
	}
	if err := validateRetainedLifecycle(bundle.Spec, p); err != nil {
		return retainedExecution{}, nil, gomad3sim.ClusterRecord{}, err
	}
	return bundle, p, observed, nil
}

func verifyWorldEvidence(evidence *gomad3sim.WASMExecutionEvidence, profile string, expected world.Config) error {
	if evidence == nil || evidence.WorldEncoding != profile {
		return errors.New("world encoding differs from selected broker profile")
	}
	switch profile {
	case "":
		snapshot, err := world.DecodeSnapshot(evidence.World)
		if err != nil {
			return err
		}
		if snapshot.Config != expected {
			return errors.New("world configuration differs from bound broker configuration")
		}
	case world.CompressedSnapshotSchema:
		summary, err := world.DecodeCompressedSnapshot(evidence.World, world.CompressedSnapshotProfile())
		if err != nil {
			return err
		}
		if summary.Config != expected {
			return errors.New("world configuration differs from bound broker configuration")
		}
	default:
		return errors.New("unsupported World profile")
	}
	return nil
}

// VerifyRetained performs complete owner admission without writing helper files.
func VerifyRetained(payload target.BackendPayload) (gomad3sim.Spec, gomad3sim.ClusterRecord, error) {
	bundle, _, observed, err := verifyRetained(payload)
	return bundle.Spec, observed, err
}

// RetainedStorageCrashSelection is detached intent for one sealed boot. Nil
// Selection is the native durable-only default; nil Limits means enumeration
// was not requested, not an inferred or omitted enumeration result.
type RetainedStorageCrashSelection struct {
	Boot      gomad3sim.BootID
	Selection map[string][]uint64
	Limits    *volume.CrashEnumerationLimits
}

// RetainedStorageCrashSelections reuses full owner preflight without writing
// executable inputs or launching helpers. It returns no executable paths.
func RetainedStorageCrashSelections(payload target.BackendPayload) ([]RetainedStorageCrashSelection, error) {
	bundle, _, _, err := verifyRetained(payload)
	if err != nil {
		return nil, err
	}
	return detachedStorageCrashSelections(bundle.Boots), nil
}

func detachedStorageCrashSelections(boots []retainedBoot) []RetainedStorageCrashSelection {
	selections := make([]RetainedStorageCrashSelection, 0, len(boots))
	for _, boot := range boots {
		selection := RetainedStorageCrashSelection{Boot: boot.ID}
		if boot.Request.StorageCrashSelection != nil {
			selection.Selection = make(map[string][]uint64, len(boot.Request.StorageCrashSelection))
			for volume, operations := range boot.Request.StorageCrashSelection {
				var cloned []uint64
				if operations != nil {
					cloned = make([]uint64, len(operations))
					copy(cloned, operations)
				}
				selection.Selection[volume] = cloned
			}
		}
		if boot.Request.StorageCrashLimits != nil {
			limits := *boot.Request.StorageCrashLimits
			selection.Limits = &limits
		}
		selections = append(selections, selection)
	}
	return selections
}

func ReopenProvider(payload target.BackendPayload, directory string) (gomad3sim.Spec, *Provider, error) {
	bundle, p, _, err := verifyRetained(payload)
	if err != nil {
		return gomad3sim.Spec{}, nil, err
	}
	admission := retainedAdmission{bundle: bundle, provider: p}
	return admission.reopen(directory)
}

type retainedAdmission struct {
	bundle   retainedExecution
	provider *Provider
}

func (a *retainedAdmission) reopen(directory string) (gomad3sim.Spec, *Provider, error) {
	if a == nil || a.provider == nil {
		return gomad3sim.Spec{}, nil, errors.New("retained WASM admission was already consumed")
	}
	bundle, p := a.bundle, a.provider
	a.bundle, a.provider = retainedExecution{}, nil
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return gomad3sim.Spec{}, nil, errors.New("retained WASM directory is not a real directory")
	}
	private, err := os.MkdirTemp(directory, "wasm-replay-")
	if err != nil {
		return gomad3sim.Spec{}, nil, err
	}
	paths := map[string]string{}
	for _, boot := range bundle.Boots {
		r := p.Boots[boot.ID]
		for _, digest := range []string{r.ModuleSHA256, r.HelperSHA256} {
			if paths[digest] != "" {
				continue
			}
			name := filepath.Join(private, digest)
			f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
			if err != nil {
				return gomad3sim.Spec{}, nil, err
			}
			_, writeErr := f.Write(bundle.Inputs[digest])
			if err = errors.Join(writeErr, f.Close()); err != nil {
				return gomad3sim.Spec{}, nil, err
			}
			paths[digest] = name
		}
		r.ModulePath, r.HelperPath = paths[r.ModuleSHA256], paths[r.HelperSHA256]
		r.CompileCache = filepath.Join(private, "compiled-cache")
		p.Boots[boot.ID] = r
	}
	return bundle.Spec, p, nil
}

func bindRetainedPrimary(observed gomad3sim.ClusterRecord, primary string) error {
	if _, err := record.ParseSHA256(primary); err != nil {
		return errors.New("invalid primary module identity")
	}
	if observed.WASM != nil {
		for _, incarnation := range observed.WASM.Incarnations {
			if "sha256:"+incarnation.ModuleSHA256 == primary {
				return nil
			}
		}
	}
	return errors.New("retained cluster primary module differs from artifact target")
}

// ReplayBoundRetained joins complete retained admission and Artifact primary
// binding before any executable writes. OriginalRecord is the admitted owner's
// exact canonical bytes, not a second encoding of the decoded record.
func ReplayBoundRetained(ctx context.Context, payload target.BackendPayload, primary, directory string) (observed gomad3sim.Result, originalRecord []byte, completed *Provider, err error) {
	bundle, p, retained, err := verifyRetained(payload)
	if err != nil {
		return gomad3sim.Result{}, nil, nil, err
	}
	if err := bindRetainedPrimary(retained, primary); err != nil {
		return gomad3sim.Result{}, nil, nil, err
	}
	originalRecord = bundle.Record
	admission := retainedAdmission{bundle: bundle, provider: p}
	spec, p, err := admission.reopen(directory)
	if err != nil {
		return gomad3sim.Result{}, nil, nil, err
	}
	observed, err = replayRetainedProvider(ctx, spec, p)
	return observed, originalRecord, p, err
}

func decodeRetained(payload target.BackendPayload) (retainedExecution, error) {
	if payload.Reference.Schema != RetainedSchema || payload.Reference.File != "backend/simulation.json" || uint64(payload.Reference.Bytes) != uint64(len(payload.Data)) || record.HashBytes(payload.Data) != payload.Reference.SHA256 || len(payload.Data) > 512<<20 {
		return retainedExecution{}, errors.New("WASM retained payload reference mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload.Data))
	decoder.DisallowUnknownFields()
	var bundle retainedExecution
	if err := decoder.Decode(&bundle); err != nil {
		return retainedExecution{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return retainedExecution{}, errors.New("trailing retained WASM payload")
	}
	if !retainedCanonicalEqual(bundle, payload.Data) || bundle.Schema != RetainedSchema {
		return retainedExecution{}, errors.New("noncanonical retained WASM payload")
	}
	return bundle, nil
}

type retainedCanonicalComparator struct {
	expected []byte
	offset   int
}

func (w *retainedCanonicalComparator) Write(data []byte) (int, error) {
	if len(data) > len(w.expected)-w.offset || !bytes.Equal(data, w.expected[w.offset:w.offset+len(data)]) {
		return 0, errors.New("retained canonical bytes differ")
	}
	w.offset += len(data)
	return len(data), nil
}

func retainedCanonicalEqual(bundle retainedExecution, expected []byte) bool {
	w := retainedCanonicalComparator{expected: expected}
	return writeRetainedCanonical(&w, bundle) == nil && w.offset == len(expected)
}

type retainedCheckedWriter struct{ io.Writer }

func (w retainedCheckedWriter) Write(data []byte) (int, error) {
	n, err := w.Writer.Write(data)
	if n != len(data) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}

// The closed retained schema keeps stdlib JSON bytes while avoiding a second
// whole-bundle encoding of the large Record and executable input byte fields.
func writeRetainedCanonical(destination io.Writer, bundle retainedExecution) error {
	w := retainedCheckedWriter{destination}
	fields := []struct {
		name  string
		value any
	}{
		{"schema", bundle.Schema},
		{"implementation_sha256", bundle.Implementation},
		{"spec", bundle.Spec},
		{"record", nil},
		{"broker", bundle.Broker},
		{"output_crashes", bundle.OutputCrashes},
		{"output_stops", bundle.OutputStops},
		{"incarnation_boots", bundle.IncarnationBoots},
		{"output_crash_groups", bundle.OutputCrashGroups},
		{"oracles", bundle.Oracles},
		{"lifecycle", bundle.Lifecycle},
		{"boots", bundle.Boots},
		{"inputs", nil},
	}
	if _, err := io.WriteString(w, "{"); err != nil {
		return err
	}
	for i, field := range fields {
		if i != 0 {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		key, err := json.Marshal(field.name)
		if err != nil {
			return err
		}
		if _, err := w.Write(key); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ":"); err != nil {
			return err
		}
		switch field.name {
		case "record":
			err = writeRetainedBytes(w, bundle.Record)
		case "inputs":
			err = writeRetainedInputs(w, bundle.Inputs)
		default:
			var encoded []byte
			encoded, err = json.Marshal(field.value)
			if err == nil {
				_, err = w.Write(encoded)
			}
		}
		if err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "}")
	return err
}

func writeRetainedBytes(w io.Writer, data []byte) error {
	if data == nil {
		_, err := io.WriteString(w, "null")
		return err
	}
	if _, err := io.WriteString(w, `"`); err != nil {
		return err
	}
	encoder := base64.NewEncoder(base64.StdEncoding, w)
	n, err := encoder.Write(data)
	if n != len(data) && err == nil {
		err = io.ErrShortWrite
	}
	if err := errors.Join(err, encoder.Close()); err != nil {
		return err
	}
	_, err = io.WriteString(w, `"`)
	return err
}

func writeRetainedInputs(w io.Writer, inputs map[string][]byte) error {
	if inputs == nil {
		_, err := io.WriteString(w, "null")
		return err
	}
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if _, err := io.WriteString(w, "{"); err != nil {
		return err
	}
	for i, key := range keys {
		if i != 0 {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		encoded, err := json.Marshal(key)
		if err != nil {
			return err
		}
		if _, err := w.Write(encoded); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ":"); err != nil {
			return err
		}
		if err := writeRetainedBytes(w, inputs[key]); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "}")
	return err
}

// RetainedRecord validates the existing backend payload envelope and delegates
// canonical cluster evidence decoding to its single owner.
func RetainedRecord(payload target.BackendPayload) (gomad3sim.ClusterRecord, error) {
	_, observed, err := VerifyRetained(payload)
	return observed, err
}
