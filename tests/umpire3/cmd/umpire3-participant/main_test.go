package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateConfigRequiresCompleteNexusAndHardBudget(t *testing.T) {
	configuration := config{
		programPath: "program.json", outputPath: "result.json", address: "localhost:7233",
		namespace: "namespace", taskQueue: "queue", workflowID: "workflow", timeout: time.Minute,
	}
	require.NoError(t, validateConfig(configuration))
	configuration.nexusEndpoint = "endpoint"
	require.EqualError(t, validateConfig(configuration), "nexus endpoint, service, and operation must be supplied together")
	configuration.nexusService = "service"
	configuration.nexusOperation = "operation"
	configuration.timeout = 0
	require.EqualError(t, validateConfig(configuration), "participant timeout must be positive")
}

func TestLoadProgramRejectsUnknownAndTrailingInput(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "program.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"formatVersion":"umpire3/participant-program/v1","identifier":"program","commands":[],"unknown":true}`), 0o600))
	_, err := loadProgram(path)
	require.ErrorContains(t, err, "unknown field")

	require.NoError(t, os.WriteFile(path, []byte(`{"formatVersion":"umpire3/participant-program/v1","identifier":"program","commands":[]} {}`), 0o600))
	_, err = loadProgram(path)
	require.EqualError(t, err, "participant program must contain one JSON document")
}
