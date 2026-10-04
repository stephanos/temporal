package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestRuntimeCampaignCollectorProfile(t *testing.T) {
	t.Setenv("GOEXPERIMENT", "greenteagc")
	campaign := runtimeCampaign{config: Config{Go: "/gomad/go"}}
	seeded := campaign.request([]string{"/gomad/go", "test", "-exec", "/gomad/wrapper", "./fixture"}, ".", time.Second, nil)
	if !slices.Contains(seeded.Env, "GOEXPERIMENT=nogreenteagc") || slices.Contains(seeded.Env, "GOEXPERIMENT=greenteagc") {
		t.Fatalf("seeded collector profile = %v", seeded.Env)
	}
	disabled := campaign.request([]string{"/gomad/go", "run", "./fixture"}, ".", time.Second, nil)
	if !slices.Contains(disabled.Env, "GOEXPERIMENT=greenteagc") || slices.Contains(disabled.Env, "GOEXPERIMENT=nogreenteagc") {
		t.Fatalf("disabled collector profile = %v", disabled.Env)
	}
}

func TestRuntimeCampaignSimulationTimeVectors(t *testing.T) {
	for _, test := range []struct {
		name, output string
		wantError    bool
	}{
		{"selected", "--- PASS: TestGomadSimulationTimeGeneratedVectors (0.00s)\nPASS\n", false},
		{"empty selection", "testing: warning: no tests to run\nPASS\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := Report{}
			campaign := runtimeCampaign{
				ctx: context.Background(), config: Config{Go: "/gomad/go"}, goRoot: "/gomad", report: &report,
				run: func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
					want := []string{"/gomad/go", "test", "-count=1", "-tags=test_dep", "-v", "runtime", "-run=^TestGomadSimulationTimeGeneratedVectors$"}
					if !slices.Equal(request.Command, want) || request.Dir != "/gomad/src" {
						t.Fatalf("runtime vector request = %+v, want %v in /gomad/src", request, want)
					}
					result := successfulCommand()
					result.Stdout = hostexec.Output{Bytes: []byte(test.output), RawBytes: []byte(test.output)}
					return result, nil
				},
			}
			if err := campaign.requireSimulationTimeVectors(); (err != nil) != test.wantError {
				t.Fatalf("requireSimulationTimeVectors() error = %v, wantError %t", err, test.wantError)
			}
			if len(report.Cases) != 1 || report.Cases[0].Passed == test.wantError {
				t.Fatalf("runtime vector report = %+v", report)
			}
		})
	}
}

func TestRequireStockCompatibilitySelectsPinnedToolchain(t *testing.T) {
	launcher := filepath.Join(t.TempDir(), "go")
	if err := os.WriteFile(launcher, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	stockRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(stockRoot, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stockRoot, "bin", "go"), []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(launcher))
	t.Setenv("GOMAD3_STOCK_GO", "")
	stop := errors.New("stop after stock Go resolution")
	var requests []hostexec.Request
	report := Report{}
	campaign := runtimeCampaign{
		ctx: context.Background(), testdata: t.TempDir(), report: &report,
		run: func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
			requests = append(requests, request)
			if len(requests) != 1 {
				return hostexec.Result{}, stop
			}
			result := successfulCommand()
			result.Stdout = hostexec.Output{Bytes: []byte(stockRoot + "\n"), RawBytes: []byte(stockRoot + "\n")}
			return result, nil
		},
	}
	if err := campaign.requireStockCompatibility(); !errors.Is(err, stop) {
		t.Fatalf("requireStockCompatibility() error = %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("stock Go resolution requests = %d, want 2", len(requests))
	}
	if !slices.Contains(requests[0].Env, "GOTOOLCHAIN="+gomadversion.GoVersion) {
		t.Fatalf("stock Go resolution environment = %v", requests[0].Env)
	}
	if slices.ContainsFunc(requests[0].Env, func(value string) bool { return strings.HasPrefix(value, "GOPROXY=") }) {
		t.Fatalf("stock Go resolution disabled verified toolchain download: %v", requests[0].Env)
	}
}

func TestValidateRandomContract(t *testing.T) {
	valid := strings.Repeat("0123456789abcdef 01234567\n", 8)
	if err := validateRandomContract("1", valid); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{
		strings.Repeat("0123456789abcdef 01234567\n", 7),
		strings.Repeat("0123456789abcdef invalid\n", 8),
	} {
		if err := validateRandomContract("1", output); err == nil {
			t.Fatalf("validateRandomContract(%q) succeeded", output)
		}
	}
}

func TestValidateClockRaceOutput(t *testing.T) {
	if err := validateClockRaceOutput("[3 1 0 2]\n", 4); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{"[3 1 0 1]\n", "[3 1 0]\n", "[3 1 nope 2]\n"} {
		if err := validateClockRaceOutput(output, 4); err == nil {
			t.Fatalf("validateClockRaceOutput(%q) succeeded", output)
		}
	}
}

func TestBenchmarkMedianNS(t *testing.T) {
	var output strings.Builder
	for value := 1; value <= 14; value++ {
		output.WriteString("BenchmarkDisabledClockNow-8 100 ")
		output.WriteString(string(rune('0' + value/10)))
		output.WriteString(string(rune('0' + value%10)))
		output.WriteString(" ns/op\n")
	}
	median, err := benchmarkMedianNS(output.String())
	if err != nil {
		t.Fatal(err)
	}
	if median != 7.5 {
		t.Fatalf("benchmarkMedianNS() = %v", median)
	}
	if _, err := benchmarkMedianNS("BenchmarkDisabledClockNow-8 100 1 ns/op\n"); err == nil {
		t.Fatal("benchmarkMedianNS() accepted an incomplete sample")
	}
}

func TestRepeatabilityMismatchRetainsDivergentEvidence(t *testing.T) {
	report := Report{Cases: []CaseResult{{Name: "actual", Passed: true, Stdout: []byte("actual\n")}}}
	campaign := runtimeCampaign{report: &report}
	err := campaign.repeatabilityMismatch("same-seed sync output diverged", "expected", "actual")
	expectedDigest := sha256.Sum256([]byte("expected"))
	actualDigest := sha256.Sum256([]byte("actual"))
	for _, want := range []string{
		"same-seed sync output diverged",
		fmt.Sprintf("expected sha256:%x", expectedDigest),
		fmt.Sprintf("actual sha256:%x", actualDigest),
		`expected-output="expected"`,
		`actual-output="actual"`,
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("repeatabilityMismatch() error = %v, want %q", err, want)
		}
	}
	if report.Cases[0].Passed {
		t.Fatal("divergent case remained passed")
	}
}

func TestRuntimeSearchFixtures(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(root, ".toolchain", "bin", "go")
	if _, err := os.Stat(goCommand); errors.Is(err, os.ErrNotExist) {
		t.Skip("patched toolchain is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if retained := os.Getenv("GOMAD3_RUNTIME_REPRODUCTION_DIR"); retained != "" {
		workspace = retained
		if err := os.MkdirAll(workspace, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	report := Report{Mode: "test-runtime-search-reproduction"}
	var commands []map[string]any
	run := func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
		controls := []string{}
		for _, value := range request.Env {
			if strings.HasPrefix(value, "GOMADSEED=") || strings.HasPrefix(value, "GOMAD3_CHOICE_") {
				controls = append(controls, value)
			}
		}
		result, err := hostexec.Run(ctx, request)
		commands = append(commands, map[string]any{"command": request.Command, "directory": request.Dir, "controls": controls, "timeout": request.Timeout.String(), "exit": result.ExitCode, "timed_out": result.WatchdogTimeout})
		return result, err
	}
	campaign := runtimeCampaign{ctx: context.Background(), config: Config{Root: root, Go: goCommand}, testdata: filepath.Join(root, "internal", "gomadtool", "conformance", "testdata"), workspace: workspace, run: run, report: &report}
	defer func() {
		for name, value := range map[string]any{"cases.json": report, "commands.json": commands} {
			data, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workspace, name), append(data, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}()
	binaries := map[string]string{}
	for _, fixture := range schedulingSearchFixtures {
		binary, err := campaign.build(fixture.name, fixture.packageName, false)
		if err != nil {
			t.Fatal(err)
		}
		binaries[fixture.name] = binary
	}
	if err := campaign.requireSearchReproduction(binaries); err != nil {
		t.Fatal(err)
	}
	report.Passed = true
}

func TestRuntimeChannelFixtures(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(root, ".toolchain", "bin", "go")
	if _, err := os.Stat(goCommand); errors.Is(err, os.ErrNotExist) {
		t.Skip("patched toolchain is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	report := Report{Mode: "test-runtime-channel-fixtures"}
	campaign := runtimeCampaign{
		ctx: context.Background(), config: Config{Root: root, Go: goCommand},
		testdata:    filepath.Join(root, "internal", "gomadtool", "conformance", "testdata"),
		execWrapper: filepath.Join(root, "internal", "gomadtool", "conformance", "scripts", "exec.sh"),
		workspace:   t.TempDir(), run: hostexec.Run, report: &report,
	}
	for _, packageName := range []string{"./timer_ties", "./runq_shuffle"} {
		for _, seed := range []string{"0", "1", "18446744073709551615"} {
			if err := campaign.requireRepeatable(packageName, seed, 10); err != nil {
				t.Fatal(err)
			}
		}
		if err := campaign.requireDiverse(packageName); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTimerCallbackAssociationRejectsUnidentifiedHandoff(t *testing.T) {
	identity := sha256.Sum256([]byte("callback"))
	run := choiceRun{transcript: "A B", trace: choice.Trace{Records: []choice.Record{
		{Kind: choice.KindRunnable, Alternatives: 2, SelectedIdentity: identity},
		{Kind: choice.KindSelectResult},
		{Kind: choice.KindSelectPoll, SiteOffset: 1},
		{Kind: choice.KindSelectResult},
		{Kind: choice.KindSelectPoll, SiteOffset: 2},
	}}}
	association, err := timerCallbackAssociation("6", run)
	if err != nil {
		t.Fatal(err)
	}
	if association.FirstIdentity != "" || association.AlternativeSet != "" {
		t.Fatalf("unidentified callback handoff associated identities: %+v", association)
	}
	if _, err := goroutineHandoffEvidence("none", "6", run, []string{"A", "B"}); err == nil {
		t.Fatal("unidentified hand-off was accepted as evidence")
	}
	run.transcript = "A A"
	if _, err := timerCallbackAssociation("6", run); err == nil {
		t.Fatal("duplicate callback markers were accepted")
	}
}

func TestStableHandoffsRequireOneAlternativeSetAndConsistentLeaders(t *testing.T) {
	first, second := sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))
	set, err := choice.AlternativeSetDigest([][sha256.Size]byte{first, second})
	if err != nil {
		t.Fatal(err)
	}
	other, err := choice.AlternativeSetDigest([][sha256.Size]byte{first, sha256.Sum256([]byte("third"))})
	if err != nil {
		t.Fatal(err)
	}
	handoff := func(label string, identity, set [sha256.Size]byte) goroutineHandoff {
		return goroutineHandoff{Seed: label, FirstLabel: label, FirstIdentity: fmt.Sprintf("%x", identity), AlternativeSet: fmt.Sprintf("%x", set)}
	}
	for _, test := range []struct {
		name string
		runs []goroutineHandoff
		led  int
		want string
	}{
		{name: "both lead", runs: []goroutineHandoff{handoff("A", first, set), handoff("B", second, set), handoff("A", first, set)}, led: 2},
		{name: "one leads", runs: []goroutineHandoff{handoff("A", first, set), handoff("A", first, set)}, led: 1},
		{name: "set changed", runs: []goroutineHandoff{handoff("A", first, set), handoff("B", second, other)}, want: "decided among alternative set"},
		{name: "leader changed", runs: []goroutineHandoff{handoff("A", first, set), handoff("A", second, set)}, want: "led A with identity"},
		{name: "leaders outside set", runs: []goroutineHandoff{handoff("A", first, set), handoff("B", sha256.Sum256([]byte("third")), set)}, want: "do not make up the alternative set"},
		{name: "unidentified", runs: []goroutineHandoff{{Seed: "1"}}, want: "did not identify its hand-off"},
		{name: "empty", want: "no hand-offs to compare"},
	} {
		led, err := requireStableHandoffs(test.runs)
		if test.want == "" {
			if err != nil || led != test.led {
				t.Fatalf("%s: requireStableHandoffs() = %d, %v", test.name, led, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%s: requireStableHandoffs() error = %v, want %q", test.name, err, test.want)
		}
	}
}
