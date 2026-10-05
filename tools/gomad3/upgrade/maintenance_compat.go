package upgrade

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/upgrade/adapterregen"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

// RegenerateAdapter keeps the original upgrade entry point while the adapter
// regeneration transaction and review digest have one owner.
func RegenerateAdapter(root, module, version, approval string, output io.Writer) (int, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return 2, err
	}
	goCommand := os.Getenv("GOMAD3_BOOTSTRAP_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	resolvedGo, err := exec.LookPath(goCommand)
	if err != nil {
		return 3, err
	}
	resolvedGo, err = filepath.Abs(resolvedGo)
	if err != nil {
		return 3, err
	}
	result, err := adapterregen.Run(context.Background(), adapterregen.Spec{
		Root: absoluteRoot, Module: module, Version: version, GoCommand: resolvedGo,
		Environment: os.Environ(), Approval: approval,
	})
	if err != nil {
		var input *adapterregen.InputError
		var blocked *adapterregen.BlockedError
		switch {
		case errors.As(err, &input):
			return 2, err
		case errors.As(err, &blocked):
			return 1, err
		default:
			return 3, err
		}
	}
	if err := adapterregen.Render(output, result); err != nil {
		return 3, err
	}
	return 0, nil
}

type PinImpact struct {
	Schema          string      `json:"schema"`
	CandidateSHA256 string      `json:"candidate_sha256"`
	BaselineSHA256  string      `json:"baseline_sha256"`
	Pins            []PinResult `json:"pins"`
}

type PinResult struct {
	Class           string   `json:"class"`
	ID              string   `json:"id"`
	Module          string   `json:"module,omitempty"`
	Version         string   `json:"version,omitempty"`
	Sum             string   `json:"sum,omitempty"`
	Platforms       []string `json:"platforms,omitempty"`
	SourceSetSHA256 string   `json:"source_set_sha256,omitempty"`
	Status          string   `json:"status"`
	Reason          string   `json:"reason"`
}

type InvalidPinImpactInput struct{ Err error }

func (err *InvalidPinImpactInput) Error() string { return err.Err.Error() }
func (err *InvalidPinImpactInput) Unwrap() error { return err.Err }

func (report PinImpact) Invalidated() bool {
	for _, pin := range report.Pins {
		if pin.Status == "invalidated" || pin.Status == "unknown" {
			return true
		}
	}
	return false
}

func (report PinImpact) CanonicalJSON() ([]byte, error) { return canonicaljson.CanonicalJSON(report) }

// ReadPinImpact preserves the original file-based entry point and uses the
// graph-aware evaluator that the command uses for indirect dependencies.
func ReadPinImpact(root, candidatePath, baselinePath string) (PinImpact, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return PinImpact{}, &InvalidPinImpactInput{Err: err}
	}
	candidate, err := readPinModule(candidatePath)
	if err != nil {
		return PinImpact{}, &InvalidPinImpactInput{Err: err}
	}
	baseline, err := readPinModule(baselinePath)
	if err != nil {
		return PinImpact{}, &InvalidPinImpactInput{Err: err}
	}
	goCommand := os.Getenv("GOMAD3_BOOTSTRAP_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	resolvedGo, err := exec.LookPath(goCommand)
	if err != nil {
		return PinImpact{}, err
	}
	resolvedGo, err = filepath.Abs(resolvedGo)
	if err != nil {
		return PinImpact{}, err
	}
	resolver, err := pinimpact.NewGoResolver(resolvedGo, os.Environ())
	if err != nil {
		return PinImpact{}, err
	}
	report, err := pinimpact.Evaluate(context.Background(), pinimpact.Spec{
		Root: absoluteRoot, Candidate: candidate, Baseline: baseline, Resolver: resolver, IncludeAll: true,
	})
	err = errors.Join(err, resolver.Close())
	if err != nil {
		if pinimpact.IsInputError(err) {
			return PinImpact{}, &InvalidPinImpactInput{Err: err}
		}
		return PinImpact{}, err
	}
	result := PinImpact{
		Schema: pinimpact.Schema, CandidateSHA256: moduleFilesDigest(candidate),
		BaselineSHA256: moduleFilesDigest(baseline), Pins: make([]PinResult, 0, len(report.Pins)),
	}
	for _, pin := range report.Pins {
		class := string(pin.Class)
		switch pin.Class {
		case pinimpact.ClassAdapter:
			class = "adapter"
		case pinimpact.ClassPackRule:
			class = "pack_rule"
		case pinimpact.ClassInterception:
			class = "interception_fingerprint"
		case pinimpact.ClassClockReference:
			class = "clock_inventory_reference"
		}
		status := strings.ReplaceAll(string(pin.Status), "-", "_")
		reason := pin.Reason
		if reason == "" && (pin.Class == pinimpact.ClassInterception || pin.Class == pinimpact.ClassClockReference) {
			reason = "runtime_pin_dependency_bump_only"
		}
		id := pin.ID
		if pin.Class == pinimpact.ClassAdapter {
			id = pin.Module
		} else if pin.Pack != "" && pin.ImportPath != "" {
			id = pin.Pack + ":" + pin.ImportPath
		}
		result.Pins = append(result.Pins, PinResult{
			Class: class, ID: id, Module: pin.Module, Version: pin.PinnedVersion,
			Sum: pin.PinnedSum, Platforms: pin.Platforms, SourceSetSHA256: pin.SourceSetSHA256,
			Status: status, Reason: reason,
		})
	}
	return result, nil
}

func readPinModule(path string) (pinimpact.ModuleFiles, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return pinimpact.ModuleFiles{}, err
	}
	path = absolutePath
	if filepath.Base(path) != "go.mod" {
		return pinimpact.ModuleFiles{}, fmt.Errorf("module file must be named go.mod: %s", path)
	}
	goMod, err := os.ReadFile(path)
	if err != nil {
		return pinimpact.ModuleFiles{}, err
	}
	goSum, err := os.ReadFile(filepath.Join(filepath.Dir(path), "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return pinimpact.ModuleFiles{}, err
	}
	return pinimpact.ModuleFiles{GoMod: goMod, GoSum: goSum, Directory: filepath.Dir(path)}, nil
}

func moduleFilesDigest(files pinimpact.ModuleFiles) string {
	joined := make([]byte, 0, len(files.GoMod)+1+len(files.GoSum))
	joined = append(joined, files.GoMod...)
	joined = append(joined, 0)
	joined = append(joined, files.GoSum...)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(joined))
}
