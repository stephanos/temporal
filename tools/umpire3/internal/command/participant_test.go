package command

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateParticipantConfigRequiresCompleteNexusAndHardBudget(t *testing.T) {
	configuration := participantConfig{
		programPath: "program.json", outputPath: "result.json", address: "localhost:7233",
		namespace: "namespace", taskQueue: "queue", workflowID: "workflow", timeout: time.Minute,
	}
	require.NoError(t, validateParticipantConfig(configuration))
	configuration.nexusEndpoint = "endpoint"
	require.EqualError(t, validateParticipantConfig(configuration), "nexus endpoint, service, and operation must be supplied together")
	configuration.nexusService = "service"
	configuration.nexusOperation = "operation"
	configuration.timeout = 0
	require.EqualError(t, validateParticipantConfig(configuration), "participant timeout must be positive")
}

func TestLoadParticipantProgramRejectsUnknownAndTrailingInput(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "program.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"formatVersion":"umpire3/participant-program/v1","identifier":"program","commands":[],"unknown":true}`), 0o600))
	_, err := loadParticipantProgram(path)
	require.ErrorContains(t, err, "unknown field")

	require.NoError(t, os.WriteFile(path, []byte(`{"formatVersion":"umpire3/participant-program/v1","identifier":"program","commands":[]} {}`), 0o600))
	_, err = loadParticipantProgram(path)
	require.EqualError(t, err, "participant program must contain one JSON document")
}
