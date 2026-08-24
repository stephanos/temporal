package configuration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"go.temporal.io/server/tools/agentworkflow/internal/process"
)

func Digest(command []string, settings any) (string, error) {
	resolved := append([]string(nil), command...)
	executable, err := exec.LookPath(resolved[0])
	if err != nil {
		return "", fmt.Errorf("resolve backend executable: %w", err)
	}
	if absolute, err := filepath.Abs(executable); err == nil {
		executable = absolute
	}
	if evaluated, err := filepath.EvalSymlinks(executable); err == nil {
		executable = evaluated
	}
	resolved[0] = executable
	encoded, err := json.Marshal(struct {
		Command  []string `json:"command"`
		Settings any      `json:"settings"`
	}{Command: resolved, Settings: settings})
	if err != nil {
		return "", fmt.Errorf("encode backend configuration identity: %w", err)
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("agentworkflow.backend-configuration/v1\x00"))
	_, _ = hasher.Write(encoded)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func Environment(credentials ...string) []string {
	names := []string{
		"PATH", "HOME", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "SYSTEMROOT",
		"SSL_CERT_FILE", "SSL_CERT_DIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
		"GOCOVERDIR",
	}
	return process.SelectEnvironment(append(names, credentials...)...)
}
