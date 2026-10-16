package command

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func TestDiscoverAdvertisesOnlyExecutableScenarioSupport(t *testing.T) {
	for _, profile := range []string{wasi.StockProfile, wasi.CooperativeProfile} {
		t.Run(profile, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status, handled := Run([]string{"discover"}, &stdout, &stderr, func() (installation.Description, error) {
				return installation.Description{Path: "/installed/gomad-wasm.json", Manifest: installation.Manifest{Options: backend.Options{Config: installation.DefaultConfig(profile)}}}, nil
			})
			var response struct {
				installation.Description
				Capabilities *backends.Capabilities `json:"capabilities"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if !handled || status != 0 || response.Path != "/installed/gomad-wasm.json" || response.Capabilities == nil || response.Capabilities.Name != "wasm" || response.Capabilities.Simulation || response.Capabilities.TypedScenarios != (profile == wasi.CooperativeProfile) {
				t.Fatalf("discovery lost or overstated capabilities: status=%d stdout=%s stderr=%s", status, stdout.String(), stderr.String())
			}
		})
	}
}
