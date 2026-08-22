package execution

import (
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
)

var testChoiceImplementationSHA256 = sha256.Sum256([]byte("choice implementation"))

func TestValidateRequestChoiceCapability(t *testing.T) {
	request := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		ExecutionTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World:  WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
		Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes},
	}
	if err := validateSpec(request); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		profile string
		limit   uint64
	}{
		{name: "missing profile", limit: MinimumChoiceTraceBytes},
		{name: "unknown profile", profile: "unknown", limit: MinimumChoiceTraceBytes},
		{name: "no record capacity", profile: choice.Profile, limit: MinimumChoiceTraceBytes - 1},
		{name: "over maximum", profile: choice.Profile, limit: MaximumChoiceTraceBytes + 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request.Choice = &ChoiceCapability{Mode: choice.ModeRecord, Profile: test.profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: test.limit}
			if err := validateSpec(request); err == nil {
				t.Fatal("validateSpec() succeeded")
			}
		})
	}
}

func TestValidateRequestRejectsChoiceEnvironmentInjection(t *testing.T) {
	base := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		ExecutionTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	for _, choice := range []*ChoiceCapability{nil, {Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes}} {
		for _, name := range []string{choiceProfileEnvironmentName, choiceModeEnvironmentName, choiceTraceFDEnvironmentName, choiceTerminalFDEnvironmentName, choiceTraceBytesEnvironmentName, choiceTapeFDEnvironmentName, choiceTapeBytesEnvironmentName, ioReadOnlyMountsEnvironmentName} {
			request := base
			request.Choice = choice
			request.Env = []string{name + "=injected"}
			if err := validateSpec(request); err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("validateSpec() accepted %s with choice %#v: %v", name, choice, err)
			}
		}
	}
}

func TestValidateRequestChoiceControllerModeMatrix(t *testing.T) {
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: testChoiceImplementationSHA256,
	}
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 1, false, [][sha256.Size]byte{sha256.Sum256([]byte("one"))}, sha256.Sum256([]byte("one")), 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	base := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		ExecutionTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	for _, mode := range []choice.Mode{choice.ModeRecord, choice.ModeReplay, choice.ModePrefix} {
		request := base
		request.Choice = &ChoiceCapability{Mode: mode, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: MinimumChoiceTraceBytes}
		if mode == choice.ModeReplay || mode == choice.ModePrefix {
			request.Choice.ReplayPlan = &tape
		}
		if err := validateSpec(request); err != nil {
			t.Fatalf("validateSpec(%d) error = %v", mode, err)
		}
	}
	for _, capability := range []*ChoiceCapability{
		{Mode: choice.ModeSeed, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes},
		{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes, ReplayPlan: &tape},
		{Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes},
	} {
		request := base
		request.Choice = capability
		if err := validateSpec(request); err == nil {
			t.Fatalf("validateSpec() accepted %#v", capability)
		}
	}
	oversized := tape
	oversized.Bytes = make([]byte, MaximumChoiceReplayPlanBytes+1)
	request := base
	request.Choice = &ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: MinimumChoiceTraceBytes, ReplayPlan: &oversized,
	}
	if err := validateSpec(request); err == nil || !strings.Contains(err.Error(), "exceeds its bound") {
		t.Fatalf("validateSpec() oversized tape error = %v", err)
	}
}

func TestValidateRequestAcceptsRankOverrideOnlyInPrefixMode(t *testing.T) {
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: testChoiceImplementationSHA256,
	}
	first := sha256.Sum256([]byte("first"))
	second := sha256.Sum256([]byte("second"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 1, false, [][sha256.Size]byte{first, second}, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := choice.BuildRankPrefix(tape, 0, (decision.Selected+1)%decision.Alternatives)
	if err != nil {
		t.Fatal(err)
	}
	request := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		ExecutionTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
		Choice: &ChoiceCapability{
			Mode: choice.ModePrefix, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
			ExecutionIdentity: identity, Limit: MinimumChoiceTraceBytes, ReplayPlan: &prefix,
		},
	}
	if err := validateSpec(request); err != nil {
		t.Fatal(err)
	}
	session, err := choice.NewSession(choice.SessionSpec{Limit: request.Choice.Limit, Mode: request.Choice.Mode, ReplayPlan: request.Choice.ReplayPlan})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	request.Choice.Mode = choice.ModeReplay
	if err := validateSpec(request); err == nil {
		t.Fatal("validateSpec() accepted a rank override in exact replay mode")
	}
}
