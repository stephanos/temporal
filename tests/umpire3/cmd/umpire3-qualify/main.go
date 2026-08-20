package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tests/umpire3/qualification"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

type receipt = qualification.Receipt

type options struct {
	releasePath    string
	experimentPath string
	resultPath     string
	profile        string
	outputPath     string
}

func main() {
	configuration := options{}
	flag.StringVar(&configuration.releasePath, "release", "", "candidate release manifest")
	flag.StringVar(&configuration.experimentPath, "experiment", "", "executed semantic experiment")
	flag.StringVar(&configuration.resultPath, "result", "", "runtime or canary result artifact")
	flag.StringVar(&configuration.profile, "profile", "", "external profile being qualified")
	flag.StringVar(&configuration.outputPath, "output", "", "qualification receipt output")
	flag.Parse()
	if err := run(configuration, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(configuration options, stdout io.Writer) error {
	if configuration.releasePath == "" || configuration.experimentPath == "" ||
		configuration.resultPath == "" || configuration.profile == "" {
		return errors.New("release, experiment, result, and profile are required")
	}
	releaseBytes, err := os.ReadFile(configuration.releasePath)
	if err != nil {
		return fmt.Errorf("read release: %w", err)
	}
	experimentBytes, err := os.ReadFile(configuration.experimentPath)
	if err != nil {
		return fmt.Errorf("read experiment: %w", err)
	}
	resultBytes, err := os.ReadFile(configuration.resultPath)
	if err != nil {
		return fmt.Errorf("read result: %w", err)
	}
	value, err := qualification.Qualify(qualification.Request{
		ReleaseBytes: releaseBytes, ExperimentBytes: experimentBytes,
		ResultBytes: resultBytes, Profile: configuration.profile,
	})
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode qualification receipt: %w", err)
	}
	encoded = append(encoded, '\n')
	if configuration.outputPath != "" {
		if err := os.WriteFile(configuration.outputPath, encoded, 0o600); err != nil {
			return fmt.Errorf("write qualification receipt: %w", err)
		}
		return nil
	}
	_, err = stdout.Write(encoded)
	return err
}

func decodeResult(encoded []byte) (umpire3runtime.Result, error) {
	return qualification.DecodeResult(encoded)
}
