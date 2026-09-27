package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// BindingFingerprint is part of every recorded Run's identity, so its bytes are pinned. The
// unconfigured literal is the bindings identity of the pinned control Run in
// tools/umpire/replay/testdata: the control Case's four bindings under the names its live test binds.
func TestBindingFingerprintGolden(t *testing.T) {
	bindings := []EnvironmentBinding{
		{ID: "temporal.worker.namespace", Value: "umpire-control"},
		{ID: "temporal.task-queue.resource", Value: "umpire-control-queue"},
		{ID: "temporal.handler-task-queue.resource", Value: "umpire-control-queue-handler"},
		{ID: "temporal.nexus-endpoint.resource", Value: "umpire-control-endpoint"},
	}
	for name, test := range map[string]struct {
		configuration []ConfigurationValue
		want          string
	}{
		"bindings": {
			want: "5433bebb9fc7ea78f3bbb112620500a67aa9ee871ce5e552a9ce71a6f4e9c629",
		},
		"bindings and configuration": {
			configuration: []ConfigurationValue{{Key: "history.enablechasm", Value: "true"}, {Key: "frontend.enablenexusapis", Value: "false"}},
			want:          "84ddf6155186493a5d75ff478c527d0779d5b1ca012ed19f91154500265c3179",
		},
	} {
		t.Run(name, func(t *testing.T) {
			profile := ProfileSpec{
				EnvironmentBindings: bindings,
				Configuration:       test.configuration,
				ProgramLimits:       &testpilotspb.ProgramLimits{MaxRequestBytes: 1 << 20},
			}
			fingerprint, err := profile.BindingFingerprint()
			require.NoError(t, err)
			require.Equal(t, test.want, fingerprint)
		})
	}
}
