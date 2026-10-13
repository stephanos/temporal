package artifact

import (
	"bytes"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

func TestPublishArtifactPreservesVerifiedCapabilityPayload(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*ArtifactInput)
		want   string
	}{
		{name: "linked"},
		{name: "guarded", change: func(input *ArtifactInput) {
			input.Manifest.Target.CapabilityMode = string(target.CapabilityModeGuarded)
		}},
		{name: "missing payload", change: func(input *ArtifactInput) { input.CapabilityManifest = nil }, want: "manifest payload is required"},
		{name: "changed hash", change: func(input *ArtifactInput) { input.CapabilityManifest.Payload[0] ^= 1 }, want: "payload changed during publication"},
		{name: "changed size", change: func(input *ArtifactInput) {
			input.CapabilityManifest.Payload = append(input.CapabilityManifest.Payload, ' ')
		}, want: "payload changed during publication"},
		{name: "changed record", change: func(input *ArtifactInput) { input.CapabilityManifest.Facts++ }, want: "identity changed during publication"},
		{name: "unexpected closure payload", change: func(input *ArtifactInput) {
			input.Manifest.Target.CapabilityMode = string(target.CapabilityModeClosure)
			input.Manifest.Target.CapabilityManifest = nil
		}, want: "capability mode is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := executionArtifactInput(t)
			payload := []byte("verified capability payload")
			input.CapabilityManifest = &target.CapabilityManifest{
				Schema: "gomad3.live-capability-manifest/v2", SHA256: record.HashBytes(payload), Bytes: uint64(len(payload)),
				ProducerImplementationSHA256: string(record.HashBytes([]byte("producer"))), GuardImplementationSHA256: string(record.HashBytes([]byte("guard"))),
				CapabilityUniverseSHA256: string(record.HashBytes([]byte("universe"))), Payload: payload,
			}
			input.Manifest.Target.CapabilityMode = string(target.CapabilityModeLinked)
			input.Manifest.Target.CapabilityManifest = input.CapabilityManifest.Record()
			if test.change != nil {
				test.change(&input)
			}
			published, err := PublishArtifact(Store{Root: t.TempDir()}, input)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("PublishArtifact() = %v, want %q", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			opened, err := OpenArtifact(published.Path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := opened.Close(); err != nil {
					t.Error(err)
				}
			}()
			actual, err := opened.ReadPayload("target-capabilities.json", 1024)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, input.CapabilityManifest.Payload) {
				t.Fatalf("capability payload = %q, want %q", actual, input.CapabilityManifest.Payload)
			}
		})
	}
}
