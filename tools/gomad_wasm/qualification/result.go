package qualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/hostfs"
	gomadqualification "go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/record"
)

// ValidateResult needs neither source nor a compiler/helper. It validates the
// existing qualification owner's report and recomputes the measured cost from
// raw samples, refusing a cost pass when any required measurement is absent.
func ValidateResult(result Result) error {
	if err := validateNativeCrashCatalogues(result); err != nil {
		return err
	}
	if !result.Manifest.Provisional && result.OperatorProcessID == 0 {
		return errors.New("final result requires actual operator process identity")
	}
	if err := result.Manifest.Validate(); err != nil {
		return err
	}
	if len(result.IntegratedSourceManifest) != 0 {
		if record.HashBytes(result.IntegratedSourceManifest) != result.Manifest.IntegratedSourceSHA256 {
			return errors.New("retained integrated source manifest identity changed")
		}
		if err := validateSourceManifest(result.IntegratedSourceManifest, false); err != nil {
			return err
		}
	} else if !result.Manifest.Provisional {
		return errors.New("final result omitted integrated source manifest")
	}
	report, err := gomadqualification.DecodeQualificationReport(bytes.TrimSuffix(result.QualificationBytes, []byte{'\n'}))
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result.Qualification)
	if err != nil {
		return err
	}
	retained, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, retained) {
		return errors.New("WASM qualification report differs from retained canonical owner bytes")
	}
	if uint64(report.Seed) != result.Manifest.Seed || uint64(report.Repeat) != result.Manifest.Repeat {
		return errors.New("retained qualification report differs from frozen sample selection")
	}
	if len(result.RunOrder) != 0 || !result.Manifest.Provisional {
		if !reflect.DeepEqual(result.RunOrder, sampleOrder(result.Samples)) {
			return errors.New("retained measurement order differs from observed sample sequence")
		}
		budget, budgetErr := EvaluateBudget(result.Samples, result.CompilerCommands, result.InitialPreparationWallNanos, result.ResidentGuests, operatorPeak(result))
		if budgetErr != nil {
			if result.Budget != nil || result.BudgetFailure != budgetErr.Error() {
				return errors.New("unmeasured budget was claimed complete")
			}
		} else if result.BudgetFailure != "" || result.Budget == nil || !reflect.DeepEqual(*result.Budget, budget) {
			return errors.New("retained budget differs from raw measurements")
		}
	}
	cost, costErr := measuredCost(result)
	if costErr != nil {
		if result.Cost != nil || result.CostFailure != costErr.Error() {
			return errors.New("unmeasured WASM costs were claimed complete")
		}
		return nil
	}
	if result.CostFailure != "" || result.Cost == nil || !reflect.DeepEqual(cost, *result.Cost) {
		return errors.New("retained WASM cost differs from raw measured samples")
	}
	return nil
}

func measuredCost(result Result) (Cost, error) {
	if result.InitialPreparationWallNanos == 0 || len(result.CompilerCommands) == 0 {
		return Cost{}, errors.New("actual preparation and compiler measurements are required")
	}
	if result.InitialCompilerCommandCount == 0 || result.InitialCompilerCommandCount > uint64(len(result.CompilerCommands)) || preparationCompiled(result.CompilerCommands[:result.InitialCompilerCommandCount]) != result.InitialPreparationCompiled {
		return Cost{}, errors.New("initial preparation command classification differs from actual observations")
	}
	if _, err := record.ParseSHA256(string(result.PreparationProvenanceSHA256)); err != nil {
		return Cost{}, errors.New("initial preparation provenance identity required")
	}
	cost, err := Summarize(result.Samples)
	if err != nil {
		return Cost{}, err
	}
	for _, command := range result.CompilerCommands {
		if command.ResourceUsage == nil {
			return Cost{}, errors.New("actual compiler resource usage is unavailable")
		}
		usage := command.ResourceUsage
		delta := usage.UserNanos + usage.SystemNanos
		if delta < usage.UserNanos || ^uint64(0)-cost.CPUNanos < delta {
			return Cost{}, errors.New("WASM cost counter overflow")
		}
		cost.CPUNanos += delta
		if ^uint64(0)-cost.CompilerCPUNanos < delta {
			return Cost{}, errors.New("WASM cost counter overflow")
		}
		cost.CompilerCPUNanos += delta
		cost.CompilerPeakRSSBytes = max(cost.CompilerPeakRSSBytes, usage.PeakRSSBytes)
		cost.HostPeakRSSBytes = max(cost.HostPeakRSSBytes, usage.PeakRSSBytes)
	}
	if ^uint64(0)-cost.PreparationWallNanos < result.InitialPreparationWallNanos {
		return Cost{}, errors.New("WASM cost counter overflow")
	}
	cost.PreparationWallNanos += result.InitialPreparationWallNanos
	cost.InitialPreparationWallNanos = result.InitialPreparationWallNanos
	// One process interval includes config/input capture, preparation, models,
	// retained artifacts and replay. Intermediate snapshots are diagnostic only;
	// summing separate intervals would omit setup between Freeze and Run.
	for _, snapshots := range [][2]*hostexec.ResourceUsage{{result.OperatorPreparationBefore, result.OperatorExecutionAfter}} {
		before, after := snapshots[0], snapshots[1]
		if before == nil || after == nil || after.PeakRSSBytes == 0 {
			return Cost{}, errors.New("actual operator self CPU and lifetime peak RSS measurements are required")
		}
		if after.UserNanos < before.UserNanos || after.SystemNanos < before.SystemNanos {
			return Cost{}, errors.New("operator CPU counters decreased")
		}
		user, system := after.UserNanos-before.UserNanos, after.SystemNanos-before.SystemNanos
		if ^uint64(0)-user < system || ^uint64(0)-cost.OperatorCPUNanos < user+system || ^uint64(0)-cost.CPUNanos < user+system {
			return Cost{}, errors.New("WASM cost counter overflow")
		}
		cost.OperatorCPUNanos += user + system
		cost.CPUNanos += user + system
		cost.OperatorPeakRSSBytes = max(cost.OperatorPeakRSSBytes, after.PeakRSSBytes)
	}
	var logicalVirtual, logicalWall uint64
	for _, execution := range result.Qualification.Executions {
		evidence := execution.Evidence
		// The canonical qualification owner retains one aggregate evidence for
		// matched nondiagnostic runs, with each execution bound by its digest.
		// Only that exact match permits projecting the shared virtual clock.
		if evidence == nil && result.Qualification.EvidenceDigest != "" && execution.EvidenceDigest == result.Qualification.EvidenceDigest {
			evidence = result.Qualification.Evidence
		}
		if evidence == nil || execution.WallElapsedNanos == 0 {
			return Cost{}, errors.New("actual logical execution wall and virtual time evidence required")
		}
		virtual, wall := uint64(evidence.VirtualTimeElapsedNanos), uint64(execution.WallElapsedNanos)
		if ^uint64(0)-logicalVirtual < virtual || ^uint64(0)-logicalWall < wall {
			return Cost{}, errors.New("WASM cost counter overflow")
		}
		logicalVirtual += virtual
		logicalWall += wall
	}
	if logicalWall == 0 {
		return Cost{}, errors.New("actual logical execution wall and virtual time evidence required")
	}
	cost.VirtualToWallRatio = float64(logicalVirtual) / float64(logicalWall)
	cost.ReplayableBugsPerComputeHour = float64(cost.ReplayableBugs) * 3_600_000_000_000 / float64(cost.CPUNanos)
	return cost, nil
}

func WriteResult(root string, result Result) (string, error) {
	return writeResult(root, result, nil)
}

func writeResult(root string, result Result, observer func(string)) (string, error) {
	observeHostMemory(observer, "result.validate.before")
	err := ValidateResult(result)
	observeHostMemory(observer, "result.validate.after")
	if err != nil {
		return "", err
	}
	observeHostMemory(observer, "result.encode.before")
	encoded, err := json.MarshalIndent(result, "", "  ")
	observeHostMemory(observer, "result.encode.after")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')
	observeHostMemory(observer, "result.publish.before")
	path, err := publishResultBytes(root, encoded)
	observeHostMemory(observer, "result.publish.after")
	return path, err
}

// This mirrors the existing qualification owner's staging/fsync publication
// discipline, with a hard link providing immutable no-clobber publication.
func publishResultBytes(root string, encoded []byte) (path string, retErr error) {
	name := "result-" + strings.TrimPrefix(string(record.HashBytes(encoded)), "sha256:") + ".json"
	return publishQualificationBytes(root, name, encoded, 64<<20)
}

func publishQualificationBytes(root, name string, encoded []byte, limit int) (path string, retErr error) {
	if len(encoded) == 0 || len(encoded) > limit || filepath.Base(name) != name {
		return "", errors.New("WASM qualification result byte limit")
	}
	if !filepath.IsAbs(root) {
		return "", errors.New("absolute WASM qualification result root required")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	directory := root
	for _, component := range []string{"", "wasm-qualifications", "v1"} {
		if component != "" {
			directory = filepath.Join(directory, component)
			if err := os.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return "", err
			}
		}
		info, err := os.Lstat(directory)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", errors.New("WASM qualification publication directory is not a real directory")
		}
		if component != "" {
			if err := os.Chmod(directory, 0700); err != nil {
				return "", err
			}
		}
	}
	path = filepath.Join(directory, name)
	file, err := os.CreateTemp(directory, ".result-*.partial")
	if err != nil {
		return "", err
	}
	temporary := file.Name()
	defer func() {
		if err := os.Remove(temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, err)
		}
	}()
	if _, err := file.Write(encoded); err != nil {
		return "", errors.Join(err, file.Close())
	}
	if err := errors.Join(file.Sync(), file.Close()); err != nil {
		return "", err
	}
	if err := os.Link(temporary, path); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
		if err := awaitCompletePublication(path, encoded, uint64(limit)); err != nil {
			return "", err
		}
	}
	if err := os.Remove(temporary); err != nil {
		return path, err
	}
	dir, err := os.Open(directory)
	if err != nil {
		return path, err
	}
	return path, errors.Join(dir.Sync(), dir.Close())
}

// The winning link temporarily has final+stage links until its publisher
// removes its own stage. Every retry retains hostfs's strict single-link,
// no-symlink, bounded read; a crashed or foreign hardlink never becomes accepted.
func awaitCompletePublication(path string, expected []byte, limit uint64) error {
	deadline := time.NewTimer(250 * time.Millisecond)
	defer deadline.Stop()
	tick := time.NewTicker(2 * time.Millisecond)
	defer tick.Stop()
	var original error
	for {
		prior, err := hostfs.ReadBounded(path, limit)
		if err == nil {
			if bytes.Equal(prior, expected) {
				return nil
			}
			return errors.New("immutable WASM qualification result collision")
		}
		if original == nil {
			original = err
		}
		select {
		case <-deadline.C:
			return errors.Join(errors.New("immutable WASM qualification result collision"), original, err)
		case <-tick.C:
		}
	}
}

func OpenResult(path string) (Result, error) {
	encoded, err := hostfs.ReadBounded(path, 64<<20)
	if err != nil {
		return Result{}, err
	}
	if filepath.Base(path) != "result-"+strings.TrimPrefix(string(record.HashBytes(encoded)), "sha256:")+".json" {
		return Result{}, errors.New("WASM qualification result content identity changed")
	}
	var result Result
	if err := strictDecode(encoded, &result); err != nil {
		return Result{}, err
	}
	if err := ValidateResult(result); err != nil {
		return Result{}, err
	}
	return result, nil
}
