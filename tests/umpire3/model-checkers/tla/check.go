package tla

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const (
	TLCVersion      = "1.7.4"
	ApalacheVersion = "0.61.0"
)

type ToolLimits struct {
	Timeout        time.Duration
	MaxOutputBytes int64
	CPUSeconds     int
	MemoryBytes    int64
}

type RawResult struct {
	Output   string
	ExitCode int
	Limits   ToolLimits
}

func CheckTLC(
	ctx context.Context,
	view protocol.TemporalView,
	javaPath string,
	tlaJar string,
	limits ToolLimits,
) (RawResult, error) {
	if javaPath == "" || tlaJar == "" {
		return RawResult{}, errors.New("pinned Java and tla2tools paths are required")
	}
	generated, err := Generate(view)
	if err != nil {
		return RawResult{}, err
	}
	return runGenerated(ctx, generated, limits, func(directory, tlaPath, configPath string) []string {
		return []string{
			javaPath, "-cp", tlaJar, "tlc2.TLC", "-cleanup", "-workers", "1",
			"-config", configPath, "-metadir", filepath.Join(directory, "states"), tlaPath,
		}
	})
}

func CheckApalache(
	ctx context.Context,
	view protocol.TemporalView,
	apalachePath string,
	limits ToolLimits,
) (RawResult, error) {
	if apalachePath == "" {
		return RawResult{}, errors.New("pinned Apalache path is required")
	}
	generated, err := Generate(view)
	if err != nil {
		return RawResult{}, err
	}
	return runGenerated(ctx, generated, limits, func(directory, tlaPath, _ string) []string {
		return []string{
			apalachePath, "--out-dir=" + filepath.Join(directory, "apalache"), "check",
			"--init=Init", "--next=Next", "--inv=TypeOK", "--no-deadlock",
			"--length=" + strconv.Itoa(view.Bounds.MaxTraceLength), tlaPath,
		}
	})
}

func runGenerated(
	ctx context.Context,
	generated Generated,
	limits ToolLimits,
	command func(directory, tlaPath, configPath string) []string,
) (RawResult, error) {
	if limits.Timeout <= 0 || limits.MaxOutputBytes <= 0 || limits.CPUSeconds <= 0 || limits.MemoryBytes <= 0 {
		return RawResult{}, errors.New("external checker timeout, output, CPU, and memory budgets are required")
	}
	directory, err := os.MkdirTemp("", "umpire3-tla-")
	if err != nil {
		return RawResult{}, fmt.Errorf("create checker directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	tlaPath := filepath.Join(directory, generated.Module+".tla")
	configPath := filepath.Join(directory, generated.Module+".cfg")
	if err := os.WriteFile(tlaPath, generated.TLA, 0o600); err != nil {
		return RawResult{}, fmt.Errorf("write generated TLA+ module: %w", err)
	}
	if err := os.WriteFile(configPath, generated.Config, 0o600); err != nil {
		return RawResult{}, fmt.Errorf("write generated TLC configuration: %w", err)
	}
	result, runErr := process.Run(ctx, process.Request{
		Command:        command(directory, tlaPath, configPath),
		Timeout:        limits.Timeout,
		MaxOutputBytes: limits.MaxOutputBytes,
		Limits: process.Limits{
			CPUSeconds:  limits.CPUSeconds,
			MemoryBytes: limits.MemoryBytes,
		},
	})
	raw := RawResult{Output: string(result.Output), ExitCode: result.ExitCode, Limits: limits}
	if runErr != nil && !knownCheckerOutcome(raw.Output) {
		return raw, fmt.Errorf("run external temporal checker: %w", runErr)
	}
	return raw, nil
}

func knownCheckerOutcome(output string) bool {
	return strings.Contains(output, "Model checking completed. No error has been found") ||
		strings.Contains(output, "Temporal properties were violated") ||
		strings.Contains(output, "Checker reports no error") ||
		strings.Contains(output, "Checker reports an error")
}
