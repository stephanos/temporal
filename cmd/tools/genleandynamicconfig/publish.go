package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/common/artifactio"
)

var dynamicConfigArtifactSet = artifactio.Set{
	Roots: []string{
		dynamicConfigFacadePath,
		"Temporal/DynamicConfig",
	},
	Paths: []string{
		dynamicConfigFacadePath,
		dynamicConfigTypesPath,
		dynamicConfigSettingsPath,
	},
}

func publishCatalog(
	outputRoot string,
	artifacts map[string][]byte,
	validate func(candidateRoot string) error,
) error {
	return dynamicConfigArtifactSet.Publish(outputRoot, artifacts, validate)
}

func validateLeanCandidate(ctx context.Context, moduleRoot string, candidateRoot string) error {
	modelRoot := filepath.Join(moduleRoot, "model")
	leanPathCommand := exec.CommandContext(ctx, "mise", "exec", "--", "lake", "env", "printenv", "LEAN_PATH")
	leanPathCommand.Dir = modelRoot
	leanPathOutput, err := leanPathCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("resolve Lean path: %w: %s", err, strings.TrimSpace(string(leanPathOutput)))
	}
	buildRoot := filepath.Join(candidateRoot, ".lean-validation")
	leanPath := strings.Join(
		[]string{buildRoot, candidateRoot, strings.TrimSpace(string(leanPathOutput))},
		string(os.PathListSeparator),
	)
	for _, relative := range []string{
		dynamicConfigTypesPath,
		dynamicConfigSettingsPath,
		dynamicConfigFacadePath,
	} {
		source := filepath.Join(candidateRoot, filepath.FromSlash(relative))
		output := filepath.Join(
			buildRoot,
			filepath.FromSlash(strings.TrimSuffix(relative, ".lean")+".olean"),
		)
		if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
			return fmt.Errorf("prepare Lean candidate output %q: %w", relative, err)
		}
		command := exec.CommandContext(
			ctx,
			"mise",
			"exec",
			"--",
			"lean",
			"-R",
			candidateRoot,
			"-o",
			output,
			source,
		)
		command.Dir = modelRoot
		command.Env = append(os.Environ(), "LEAN_PATH="+leanPath)
		var outputBuffer bytes.Buffer
		command.Stdout = &outputBuffer
		command.Stderr = &outputBuffer
		if err := command.Run(); err != nil {
			diagnostic := strings.ReplaceAll(outputBuffer.String(), candidateRoot, "<candidate>")
			return fmt.Errorf(
				"lean elaborate %q: %w: %s",
				relative,
				err,
				strings.TrimSpace(diagnostic),
			)
		}
	}
	return nil
}
