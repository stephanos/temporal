package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

type resumeProfileProvider struct {
	backend.Provider
	description backend.Description
	options     string
	validations int
}

type resumeProfileProvenance struct {
	Options      string
	TargetSHA256 record.SHA256
	Args         []string
}

func (provider *resumeProfileProvider) Describe(_ context.Context, config backend.Configuration) (backend.Description, error) {
	if config.IOProfile != provider.description.IOProfile.Name {
		return backend.Description{}, errors.New("configured provider profile changed")
	}
	return provider.description, nil
}

func (provider *resumeProfileProvider) ValidatePrepared(spec target.Spec, prepared target.Prepared, environment []string) error {
	provider.validations++
	if err := prepared.ValidateBackendPayloads(); err != nil {
		return err
	}
	var provenance resumeProfileProvenance
	if err := json.Unmarshal(prepared.BackendPayloads[0].Data, &provenance); err != nil {
		return err
	}
	if provenance.Options != provider.options {
		return errors.New("retained provider options changed")
	}
	if spec.Backend != prepared.Backend.Name || spec.Kind != prepared.Kind || spec.Source != prepared.Source || provenance.TargetSHA256 != record.SHA256(prepared.SHA256) || !reflect.DeepEqual(prepared.Argv, append([]string{"gomad3-target"}, provenance.Args...)) || !reflect.DeepEqual(spec.Args, provenance.Args) || !reflect.DeepEqual(environment, []string{"MODE=retained"}) {
		return errors.New("retained target provenance changed")
	}
	return prepared.Verify()
}

func TestResumeProfileUsesValidatedBackendProvenance(t *testing.T) {
	for _, test := range []struct {
		name        string
		change      func(*resumeProfileProvider, *campaign.CampaignPlan, string)
		wantError   string
		validations int
	}{
		{name: "retained provenance", validations: 1},
		{name: "changed current options", change: func(provider *resumeProfileProvider, _ *campaign.CampaignPlan, _ string) {
			provider.options = "changed"
		}, wantError: "retained provider options changed", validations: 1},
		{name: "changed provenance bytes", change: func(_ *resumeProfileProvider, _ *campaign.CampaignPlan, path string) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
		}, wantError: "external backend payload bytes do not match their reference"},
		{name: "changed recorded profile", change: func(_ *resumeProfileProvider, plan *campaign.CampaignPlan, _ string) {
			plan.IOProfile.ImplementationSHA256 = record.HashBytes([]byte("changed"))
		}, wantError: "recorded I/O profile identity does not match the external backend"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			module := []byte("retained executable")
			provenance, err := json.Marshal(resumeProfileProvenance{Options: "retained", TargetSHA256: record.HashBytes(module), Args: []string{"fixed"}})
			if err != nil {
				t.Fatal(err)
			}
			provenancePath := filepath.Join(root, "prepared", "backend", "provenance.json")
			if err := os.MkdirAll(filepath.Dir(provenancePath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(provenancePath, provenance, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "prepared", "target"), module, 0o500); err != nil {
				t.Fatal(err)
			}
			metadata := &record.BackendMetadata{Name: "fixture-resume", ReplayMode: record.ReplayObserved, Provenance: record.BackendPayload{Schema: "fixture.resume-provenance/v1", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance))}}
			profile := record.BackendIOProfile(*metadata)
			plan := campaign.CampaignPlan{
				RunnerBuild: "runner", Selection: "7", SelectionCount: 1,
				IOProfile:   record.IOContract{Name: profile.Name, ImplementationSHA256: profile.ImplementationSHA256, InventorySHA256: profile.InventorySHA256},
				Prepared:    campaign.PreparedTargetPlan{Path: "prepared/target", Target: record.Target{Backend: metadata, Kind: "go-run", Source: "example.com/workload", SHA256: record.HashBytes(module), Size: record.Uint64String(len(module)), Argv: []string{"gomad3-target", "fixed"}}},
				Toolchain:   record.Toolchain{GoVersion: "go1.27.1", BuildKey: "retained-build", TargetGOOS: "wasip1", TargetGOARCH: "wasm"},
				Environment: []record.Environment{{Name: "MODE", Value: "retained"}}, IOROMountLimits: record.ReadOnlyMountLimits{PathBytes: 4096, Requests: 1, Files: 1, DirectoryEntries: 1, SingleFileBytes: 1024, TotalBytes: 1024},
			}
			provider := &resumeProfileProvider{options: "retained", description: backend.Description{IOProfile: record.BackendIOProfile(record.BackendMetadata{Name: metadata.Name, Provenance: record.BackendPayload{SHA256: record.HashBytes([]byte("provider model"))}})}}
			if provider.description.IOProfile.ImplementationSHA256 == profile.ImplementationSHA256 {
				t.Fatal("fixture lacks the material provenance and static description distinction")
			}
			if test.change != nil {
				test.change(provider, &plan, provenancePath)
			}
			config, selection, environment, _, prepared, err := resumeConfiguration(campaignRequest{campaignOptions: campaignOptions{campaignIdentityOptions: campaignIdentityOptions{ResumeCampaign: root}}, campaignRuntime: campaignRuntime{Backend: provider, RunnerBuild: "runner"}}, plan)
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError || provider.validations != test.validations {
					t.Fatalf("resume error = %v, validations = %d; want %q, %d", err, provider.validations, test.wantError, test.validations)
				}
				return
			}
			if err != nil || config.Backend != provider || config.Target.Backend != metadata.Name || selection.String() != "7" || !reflect.DeepEqual(environment, plan.Environment) || len(prepared.BackendPayloads) != 1 || !bytes.Equal(prepared.BackendPayloads[0].Data, provenance) || provider.validations != 1 {
				t.Fatalf("resume = %#v, selection = %s, prepared = %#v, error = %v, validations = %d", config, selection.String(), prepared, err, provider.validations)
			}
		})
	}
}

func TestResumeNativeStaticProfileFailsBeforeOtherPreflight(t *testing.T) {
	provider := &resumeProfileProvider{description: backend.Description{IOProfile: record.IOProfile{Name: "native-profile", ImplementationSHA256: record.HashBytes([]byte("current")), InventorySHA256: record.HashBytes([]byte("inventory"))}}}
	plan := campaign.CampaignPlan{RunnerBuild: "runner", Selection: "invalid", IOProfile: record.IOContract{Name: "native-profile", ImplementationSHA256: record.HashBytes([]byte("changed")), InventorySHA256: provider.description.IOProfile.InventorySHA256}}
	_, _, _, _, _, err := resumeConfiguration(campaignRequest{campaignRuntime: campaignRuntime{Backend: provider, RunnerBuild: "runner"}}, plan)
	if err == nil || err.Error() != "recorded I/O profile identity does not match the backend provider" || provider.validations != 0 {
		t.Fatalf("native profile error precedence = %v, validations = %d", err, provider.validations)
	}
}
