package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

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

func TestTimerCallbackAssociationRejectsUnidentifiedHandoff(t *testing.T) {
	input := append([]byte("gomad3-choice-goroutine-runtime/v1"), make([]byte, 8)...)
	binary.BigEndian.PutUint64(input[len(input)-8:], 2)
	identity := sha256.Sum256(input)
	run := choiceRun{transcript: "A B", trace: choice.Trace{Records: []choice.Record{
		{Kind: choice.KindRunnable, SelectedIdentity: identity},
		{Kind: choice.KindSelectResult},
		{Kind: choice.KindSelectPoll, SiteOffset: 1},
		{Kind: choice.KindSelectResult},
		{Kind: choice.KindSelectPoll, SiteOffset: 2},
	}}}
	association, err := timerCallbackAssociation("6", run)
	if err != nil {
		t.Fatal(err)
	}
	if len(association.CallbackIdentities) != 0 {
		t.Fatalf("unidentified callback handoff associated identities: %+v", association)
	}
	run.transcript = "A A"
	if _, err := timerCallbackAssociation("6", run); err == nil {
		t.Fatal("duplicate callback markers were accepted")
	}
}
