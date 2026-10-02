package execution

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestDiagnosticLauncherLocalizesInjectedDraw(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{Kind: target.KindGoRun, Source: "./diagnostic_fault", WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"), PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot})
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	run := func(diagnostics bool, perturb bool) Result {
		t.Helper()
		environment := []string{"GOMADSEED=7", "TZ=UTC", "GOMAD3_IO_PROFILE=" + profile.Name()}
		if perturb {
			environment = append(environment, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW=5")
		}
		result, err := Run(context.Background(), Spec{SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"}, Command: prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: environment, ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1024, World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7}, IO: &IOCapability{Config: frame, Transcript: &IOTranscriptCapability{Limit: 64 << 20}}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: implementation, Limit: 1 << 20}, Diagnostics: diagnostics})
		if err != nil {
			t.Fatal(err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("target status %d: %s", result.ExitCode, result.Stderr.Bytes)
		}
		return result
	}
	plain := run(true, false)
	repeated := run(true, false)
	if !bytes.Equal(plain.DiagnosticTrace.Bytes, repeated.DiagnosticTrace.Bytes) {
		t.Fatal("same-seed diagnostic traces differ")
	}
	off := run(false, false)
	if !bytes.Equal(off.ChoiceTrace.Trace.Bytes, plain.ChoiceTrace.Trace.Bytes) || !bytes.Equal(off.Stdout.Bytes, plain.Stdout.Bytes) || off.IOTranscript.SHA256 != plain.IOTranscript.SHA256 {
		t.Fatal("diagnostics changed behavior")
	}
	perturbed := run(true, true)
	difference, err := choice.DiffDiagnostics(plain.DiagnosticTrace.Bytes, perturbed.DiagnosticTrace.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if difference == nil || difference.Ordinal != 5 || !reflect.DeepEqual(difference.Fields, []string{"runtime_cheap_rand_draws"}) {
		t.Fatalf("injected draw difference = %+v", difference)
	}
}
