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

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

// RetainedSchema v2 references the executable module by digest; the verified
// Artifact target supplies its bytes on replay. v1 embedded module and helper
// bytes and is refused, not migrated.
const RetainedSchema = "gomad.wasm-simulation-retained/v2"

const retainedSchemaV1 = "gomad.wasm-simulation-retained/v1"

// ErrRetainedSchemaUnsupported marks retained evidence from an older schema.
var ErrRetainedSchemaUnsupported = errors.New("unsupported retained WASM cluster evidence schema")

// IsRetainedSchema reports whether schema names current or historical retained
// cluster evidence, so callers route old artifacts to a versioned refusal
// instead of treating them as foreign evidence.
func IsRetainedSchema(schema string) bool {
	return schema == RetainedSchema || schema == retainedSchemaV1
}

// CheckRetainedSchema returns the versioned refusal for any schema other than
// RetainedSchema, so callers can refuse before resolving an installation or
// reading the payload.
func CheckRetainedSchema(schema string) error {
	if schema == RetainedSchema {
		return nil
	}
	return retainedSchemaError(schema)
}

func retainedSchemaError(schema string) error {
	if schema == retainedSchemaV1 {
		return fmt.Errorf("%w: %s embedded module bytes and cannot be replayed; this build reads %s (re-record the artifact)", ErrRetainedSchemaUnsupported, schema, RetainedSchema)
	}
	return fmt.Errorf("%w: %q; this build reads %s", ErrRetainedSchemaUnsupported, schema, RetainedSchema)
}

// maximumRetainedModuleBytes bounds the referenced module copy.
const maximumRetainedModuleBytes = 512 << 20

// CheckRetainedModuleSize rejects a module whose listed size exceeds the
// retained module bound, so callers refuse it before copying any bytes.
func CheckRetainedModuleSize(size uint64) error {
	if size > maximumRetainedModuleBytes {
		return errors.New("retained WASM module exceeds its bound")
	}
	return nil
}

var ErrRetentionCapacity = errors.New("WASM retained execution capacity exceeded")

type RetentionCapacityError struct{ Required, Maximum uint64 }

func (e *RetentionCapacityError) Error() string {
	return fmt.Sprintf("%v: required=%d maximum=%d", ErrRetentionCapacity, e.Required, e.Maximum)
}
func (e *RetentionCapacityError) Unwrap() error { return ErrRetentionCapacity }

// The record is base64 encoded by the existing JSON owner. Reject a known
// oversized bundle before encoding it.
func retainedCapacityLowerBound(recordBytes uint64) error {
	const maximum = uint64(512 << 20)
	if recordBytes > maximum {
		return &RetentionCapacityError{Required: recordBytes, Maximum: maximum}
	}
	if required := 4 * ((recordBytes + 2) / 3); required > maximum {
		return &RetentionCapacityError{Required: required, Maximum: maximum}
	}
	return nil
}

type retainedBoot struct {
	ID      gomadsim.BootID `json:"id"`
	Request wasi.Request    `json:"request"`
}
type retainedExecution struct {
	Schema            string                         `json:"schema"`
	Implementation    string                         `json:"implementation_sha256"`
	Spec              gomadsim.Spec                  `json:"spec"`
	Record            []byte                         `json:"record"`
	Broker            wasi.BrokerConfig              `json:"broker"`
	OutputCrashes     []OutputCrash                  `json:"output_crashes"`
	OutputStops       []OutputCrash                  `json:"output_stops"`
	IncarnationBoots  []IncarnationBoot              `json:"incarnation_boots"`
	OutputCrashGroups []OutputCrashGroup             `json:"output_crash_groups"`
	Oracles           *OraclePlan                    `json:"oracles"`
	Lifecycle         []gomadsim.LifecycleTransition `json:"lifecycle"`
	Boots             []retainedBoot                 `json:"boots"`
	// ModuleSHA256 is the single executable module every boot runs. Its bytes
	// are the Artifact target, never embedded here.
	ModuleSHA256 string `json:"module_sha256"`
}

// RetentionPayload produces an existing Artifact backend evidence payload. It
// references the module by digest; the Artifact target retains its bytes, and
// replay executes the installed helper, so no executable bytes are embedded.
func (p *Provider) RetentionPayload(spec gomadsim.Spec, observed gomadsim.ClusterRecord) (target.BackendPayload, error) {
	if p == nil || observed.Backend != gomadsim.BackendWASM || observed.WASM == nil {
		return target.BackendPayload{}, errors.New("WASM retention requires completed broker evidence")
	}
	encoded, err := gomadsim.EncodeClusterRecord(observed)
	if err != nil {
		return target.BackendPayload{}, err
	}
	if err := retainedCapacityLowerBound(uint64(len(encoded))); err != nil {
		return target.BackendPayload{}, err
	}
	broker := p.Broker
	broker.Diagnostics = false
	broker.Nodes = nil
	broker.Addresses = nil
	spec.Replay = nil
	bundle := retainedExecution{Schema: RetainedSchema, Implementation: ImplementationSHA256(), Spec: spec, Record: encoded, Broker: broker, OutputCrashes: p.OutputCrashes, OutputStops: p.OutputStops, IncarnationBoots: p.IncarnationBoots, OutputCrashGroups: p.OutputCrashGroups, Oracles: p.Oracles, Lifecycle: observed.Transitions}
	ids := make([]gomadsim.BootID, 0, len(p.Boots))
	for id := range p.Boots {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		r := p.Boots[id]
		if r.Runtime == nil || r.Runtime.Choice == nil || r.StorageImage != nil || r.Broker != nil {
			return target.BackendPayload{}, errors.New("WASM retention requires fresh bound boot inputs")
		}
		if !validRetainedDigest(r.ModuleSHA256) || bundle.ModuleSHA256 != "" && bundle.ModuleSHA256 != r.ModuleSHA256 {
			return target.BackendPayload{}, errors.New("WASM retention requires every boot to run the one Artifact target module")
		}
		bundle.ModuleSHA256 = r.ModuleSHA256
		r.ModulePath, r.HelperPath, r.CompileCache = "", "", ""
		r.StdoutMirror = nil
		bundle.Boots = append(bundle.Boots, retainedBoot{ID: id, Request: r})
	}
	if len(bundle.Boots) == 0 {
		return target.BackendPayload{}, errors.New("WASM retention requires bound boots")
	}
	// Size the single output buffer exactly with a counting pass, so encoding
	// neither doubles a growing buffer nor holds a second whole-bundle copy.
	var size retainedCountingWriter
	if err := writeRetainedCanonical(&size, bundle); err != nil {
		return target.BackendPayload{}, err
	}
	if size > 512<<20 {
		return target.BackendPayload{}, &RetentionCapacityError{Required: uint64(size), Maximum: 512 << 20}
	}
	data := bytes.NewBuffer(make([]byte, 0, int(size)))
	hasher := sha256.New()
	if err := writeRetainedCanonical(io.MultiWriter(data, hasher), bundle); err != nil {
		return target.BackendPayload{}, err
	}
	if data.Len() != int(size) {
		return target.BackendPayload{}, errors.New("WASM retained encoding changed size")
	}
	return target.BackendPayload{Reference: record.BackendPayload{Schema: RetainedSchema, File: "backend/simulation.json", SHA256: record.SHA256FromSum([sha256.Size]byte(hasher.Sum(nil))), Bytes: record.Uint64String(data.Len())}, Data: data.Bytes()}, nil
}

type retainedCountingWriter uint64

func (w *retainedCountingWriter) Write(data []byte) (int, error) {
	*w += retainedCountingWriter(len(data))
	return len(data), nil
}

func validRetainedDigest(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	for _, c := range digest {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// verifyRetained admits retained detached inputs before any module file is
// created. Replay reuses only the referenced Artifact target, never original paths.
func verifyRetained(payload target.BackendPayload) (retainedExecution, *Provider, gomadsim.ClusterRecord, error) {
	bundle, err := decodeRetained(payload)
	if err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	if bundle.Implementation != ImplementationSHA256() {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained WASM model implementation differs from current owners")
	}
	observed, err := gomadsim.DecodeClusterRecord(bundle.Record)
	if err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	if err := ValidateInspectionChoices(observed.WASM); err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	if observed.Backend != gomadsim.BackendWASM || observed.WASM == nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained payload lacks WASM execution evidence")
	}
	actions, _ := json.Marshal(bundle.Lifecycle)
	expectedActions, _ := json.Marshal(observed.Transitions)
	if !bytes.Equal(actions, expectedActions) {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained WASM actions differ from observed lifecycle")
	}
	plan, err := gomadsim.ReplayPlanFor(observed)
	if err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	bundle.Spec.Replay = &plan
	if err := gomadsim.ValidateSpec(bundle.Spec); err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	p := &Provider{Boots: map[gomadsim.BootID]wasi.Request{}, Broker: bundle.Broker, OutputCrashes: bundle.OutputCrashes, OutputStops: bundle.OutputStops, IncarnationBoots: bundle.IncarnationBoots, OutputCrashGroups: bundle.OutputCrashGroups, Oracles: bundle.Oracles}
	if len(bundle.Boots) == 0 || !validRetainedDigest(bundle.ModuleSHA256) {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained WASM module reference is invalid")
	}
	for _, boot := range bundle.Boots {
		if boot.ID == "" || p.Boots[boot.ID].Runtime != nil || boot.Request.Runtime == nil || boot.Request.Runtime.Choice == nil || boot.Request.ModulePath != "" || boot.Request.HelperPath != "" || boot.Request.Broker != nil || boot.Request.StorageImage != nil {
			return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("invalid retained WASM boot")
		}
		p.Boots[boot.ID] = boot.Request
		if boot.Request.ModuleSHA256 != bundle.ModuleSHA256 {
			return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained WASM boot module differs from the referenced module")
		}
	}
	for _, n := range bundle.Spec.Nodes {
		r, ok := p.Boots[n.Boot]
		if !ok {
			return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained WASM node lacks boot")
		}
		identity, err := BootConfiguration(r)
		if err != nil || !bytes.Equal(identity, n.Config) {
			return retainedExecution{}, nil, gomadsim.ClusterRecord{}, errors.New("retained WASM configuration mismatch")
		}
	}
	// Reuse provider admission for every incarnation configuration before any
	// retained executable is written or a helper can start.
	admitted, err := p.Open(context.Background(), bundle.Spec)
	if err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	config := admitted.(*execution).provider.Broker
	if err := validateInspectionNetworks(observed.WASM, &admitted.(*execution).provider, bundle.Spec); err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	if err := verifyWorldEvidence(observed.WASM, config.WorldProfile, world.Config{Seed: world.Seed(config.Seed), Limits: config.Limits}); err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, fmt.Errorf("retained WASM World owner evidence: %w", err)
	}
	if err := validateRetainedLifecycle(bundle.Spec, p); err != nil {
		return retainedExecution{}, nil, gomadsim.ClusterRecord{}, err
	}
	return bundle, p, observed, nil
}

func verifyWorldEvidence(evidence *gomadsim.WASMExecutionEvidence, profile string, expected world.Config) error {
	if evidence == nil || evidence.WorldEncoding != profile {
		return errors.New("world encoding differs from selected broker profile")
	}
	if profile != "" && profile != world.CompressedSnapshotSchema {
		return errors.New("unsupported World profile")
	}
	summary, err := world.DecodeCompressedSnapshot(evidence.World, world.CompressedSnapshotProfile())
	if err != nil {
		return err
	}
	if summary.Config != expected {
		return errors.New("world configuration differs from bound broker configuration")
	}
	return nil
}

// VerifyRetained performs complete owner admission without writing helper files.
func VerifyRetained(payload target.BackendPayload) (gomadsim.Spec, gomadsim.ClusterRecord, error) {
	bundle, _, observed, err := verifyRetained(payload)
	return bundle.Spec, observed, err
}

// ResolveRetainedInspection admits all immutable evidence before selecting a
// boundary. It never materializes executable inputs or starts a helper.
func ResolveRetainedInspection(payload target.BackendPayload, request gomadsim.WASMInspectionRequest) (gomadsim.WASMInspectionResolution, error) {
	_, _, observed, err := verifyRetained(payload)
	if err != nil {
		return gomadsim.WASMInspectionResolution{}, err
	}
	return ResolveInspection(observed.WASM, request, true)
}

// RetainedStorageCrashSelection is detached intent for one sealed boot. Nil
// Selection is the native durable-only default; nil Limits means enumeration
// was not requested, not an inferred or omitted enumeration result.
type RetainedStorageCrashSelection struct {
	Boot      gomadsim.BootID
	Selection map[string][]uint64
	Context   map[string]wasi.StorageCrashContext
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
		if boot.Request.StorageCrashContext != nil {
			selection.Context = cloneStorageCrashContext(boot.Request.StorageCrashContext)
		}
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

// InstalledHelper is the operator installation's pinned engine helper. Payload
// hashes establish only self-consistency, so retained replay executes this
// helper and never native helper bytes carried by an artifact.
type InstalledHelper struct {
	Path             string
	SHA256           string
	Engine           wasi.EngineIdentity
	ConfigureRequest func(*wasi.Request) error
}

// RetainedModule supplies the module bytes that retained evidence references
// by digest, normally the verified Artifact target. Copy must create
// destination exclusively; reopen re-hashes the copy before any helper starts.
type RetainedModule struct {
	SHA256 string // "sha256:<hex>"
	Copy   func(destination string) error
}

func copyRetainedModule(destination string, source io.Reader) (retErr error) {
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	_, err = io.Copy(f, source)
	return err
}

// materializeRetainedModule copies the referenced module into a fresh private
// file and fails closed unless its bounded bytes hash to the retained digest.
func materializeRetainedModule(module RetainedModule, digest, destination string) error {
	if module.Copy == nil {
		return errors.New("retained WASM replay requires the Artifact target module")
	}
	if err := module.Copy(destination); err != nil {
		return fmt.Errorf("retained WASM module: %w", err)
	}
	f, info, err := hostfs.OpenPath(destination)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	n, err := io.Copy(hasher, io.LimitReader(f, maximumRetainedModuleBytes+1))
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	if info.Size() != n || n > maximumRetainedModuleBytes || fmt.Sprintf("%x", hasher.Sum(nil)) != digest {
		return errors.New("retained WASM module bytes differ from the referenced digest")
	}
	return nil
}

func ReopenProvider(payload target.BackendPayload, module RetainedModule, directory string, installed InstalledHelper) (gomadsim.Spec, *Provider, error) {
	bundle, p, _, err := verifyRetained(payload)
	if err != nil {
		return gomadsim.Spec{}, nil, err
	}
	admission := retainedAdmission{bundle: bundle, provider: p}
	return admission.reopen(module, directory, installed)
}

func bindInstalledHelper(p *Provider, installed InstalledHelper) error {
	if !filepath.IsAbs(installed.Path) || installed.SHA256 == "" {
		return errors.New("retained WASM replay requires an installed helper")
	}
	ids := make([]gomadsim.BootID, 0, len(p.Boots))
	for id := range p.Boots {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		r := p.Boots[id]
		if r.HelperSHA256 != installed.SHA256 || r.Engine != installed.Engine {
			return fmt.Errorf("retained WASM boot %s helper sha256:%s engine %s differs from installed helper sha256:%s engine %s", id, r.HelperSHA256, engineIdentity(r.Engine), installed.SHA256, engineIdentity(installed.Engine))
		}
	}
	data, err := hostfs.ReadBounded(installed.Path, 512<<20)
	if err != nil {
		return fmt.Errorf("installed WASM helper: %w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != installed.SHA256 {
		return errors.New("installed WASM helper identity changed")
	}
	return nil
}

func engineIdentity(engine wasi.EngineIdentity) string {
	return fmt.Sprintf("%s/%s/%s/%s-%s", engine.Name, engine.Version, engine.Configuration, engine.HostOS, engine.HostArch)
}

type retainedAdmission struct {
	bundle   retainedExecution
	provider *Provider
}

func (a *retainedAdmission) reopen(module RetainedModule, directory string, installed InstalledHelper) (gomadsim.Spec, *Provider, error) {
	if a == nil || a.provider == nil {
		return gomadsim.Spec{}, nil, errors.New("retained WASM admission was already consumed")
	}
	bundle, p := a.bundle, a.provider
	a.bundle, a.provider = retainedExecution{}, nil
	if module.SHA256 != "sha256:"+bundle.ModuleSHA256 {
		return gomadsim.Spec{}, nil, errors.New("retained WASM module reference differs from the Artifact target")
	}
	if err := bindInstalledHelper(p, installed); err != nil {
		return gomadsim.Spec{}, nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return gomadsim.Spec{}, nil, errors.New("retained WASM directory is not a real directory")
	}
	private, err := os.MkdirTemp(directory, "wasm-replay-")
	if err != nil {
		return gomadsim.Spec{}, nil, err
	}
	name := filepath.Join(private, bundle.ModuleSHA256)
	if err := materializeRetainedModule(module, bundle.ModuleSHA256, name); err != nil {
		return gomadsim.Spec{}, nil, errors.Join(err, os.RemoveAll(private))
	}
	for _, boot := range bundle.Boots {
		r := p.Boots[boot.ID]
		r.ModulePath, r.HelperPath = name, installed.Path
		r.CompileCache = filepath.Join(private, "compiled-cache")
		if installed.ConfigureRequest != nil {
			if err := installed.ConfigureRequest(&r); err != nil {
				return gomadsim.Spec{}, nil, err
			}
		}
		p.Boots[boot.ID] = r
	}
	return bundle.Spec, p, nil
}

func bindRetainedPrimary(observed gomadsim.ClusterRecord, primary string) error {
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
// binding before any executable writes. The primary module is the Artifact
// target that supplies the referenced module bytes. OriginalRecord is the
// digest of the admitted owner's exact canonical bytes, not of a second
// encoding of the record; the bytes themselves are released before replay.
func ReplayBoundRetained(ctx context.Context, payload target.BackendPayload, module RetainedModule, directory string, installed InstalledHelper) (observed gomadsim.Result, originalRecord record.SHA256, completed *Provider, err error) {
	bundle, p, retained, err := verifyRetained(payload)
	if err != nil {
		return gomadsim.Result{}, "", nil, err
	}
	if err := bindRetainedPrimary(retained, module.SHA256); err != nil {
		return gomadsim.Result{}, "", nil, err
	}
	originalRecord = record.HashBytes(bundle.Record)
	bundle.Record = nil
	admission := retainedAdmission{bundle: bundle, provider: p}
	spec, p, err := admission.reopen(module, directory, installed)
	if err != nil {
		return gomadsim.Result{}, "", nil, err
	}
	observed, err = replayRetainedProvider(ctx, spec, p)
	return observed, originalRecord, p, err
}

func decodeRetained(payload target.BackendPayload) (retainedExecution, error) {
	if err := CheckRetainedSchema(payload.Reference.Schema); err != nil {
		return retainedExecution{}, err
	}
	if payload.Reference.File != "backend/simulation.json" && payload.Reference.File != "backend/evidence.bin" || uint64(payload.Reference.Bytes) != uint64(len(payload.Data)) || record.HashBytes(payload.Data) != payload.Reference.SHA256 || len(payload.Data) > 512<<20 {
		return retainedExecution{}, errors.New("WASM retained payload reference mismatch")
	}
	var bundle retainedExecution
	if err := strictjson.Decode(payload.Data, &bundle); err != nil {
		return retainedExecution{}, err
	}
	if bundle.Schema != RetainedSchema || bundle.ModuleSHA256 == "" {
		return retainedExecution{}, errors.New("invalid retained WASM payload identity")
	}
	return bundle, nil
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
		{"module_sha256", bundle.ModuleSHA256},
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

// RetainedRecord validates the existing backend payload envelope and delegates
// canonical cluster evidence decoding to its single owner.
func RetainedRecord(payload target.BackendPayload) (gomadsim.ClusterRecord, error) {
	_, observed, err := VerifyRetained(payload)
	return observed, err
}
