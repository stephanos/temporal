package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"
)

func TestRunPinImpactMissingPackSumIsUnknown(t *testing.T) {
	fixture := newPinImpactFixture(t)
	packs, err := compatibility.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	var modules []pinImpactModule
	for _, validated := range packs {
		pack := validated.Pack()
		if pack.ID == "temporal-leaf-xxhash-darwin-arm64" {
			for _, activation := range pack.Activation {
				modules = append(modules, pinImpactModule{path: activation.Path, version: activation.Version, sum: activation.Sum})
			}
		}
	}
	if len(modules) != 2 {
		t.Fatal("checked xxhash pack must activate on two modules")
	}
	proxy := t.TempDir()
	var sums strings.Builder
	for _, served := range modules {
		escaped, err := module.EscapePath(served.path)
		if err != nil {
			t.Fatal(err)
		}
		directory := filepath.Join(proxy, filepath.FromSlash(escaped), "@v")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		goMod := "module " + served.path + "\n\ngo 1.21\n"
		for name, contents := range map[string]string{
			served.version + ".mod":  goMod,
			served.version + ".info": fmt.Sprintf(`{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, served.version),
		} {
			if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		modSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(goMod)), nil })
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&sums, "%s %s/go.mod %s\n", served.path, served.version, modSum)
		if served.path != "github.com/klauspost/compress" {
			fmt.Fprintf(&sums, "%s %s %s\n", served.path, served.version, served.sum)
		}
	}
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	baseline, candidate := t.TempDir(), t.TempDir()
	for _, directory := range []string{baseline, candidate} {
		writePinImpactModule(t, directory, modules)
		if err := os.WriteFile(filepath.Join(directory, "go.sum"), []byte(sums.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	beforeBaseline, beforeCandidate := readModuleSnapshot(t, baseline), readModuleSnapshot(t, candidate)
	for _, jsonOutput := range []bool{false, true} {
		arguments := []string{"pin-impact", "--root", fixture.root, "--module", candidate, "--baseline-module", baseline, "--go", fixture.goCommand}
		if jsonOutput {
			arguments = append(arguments, "--json")
		}
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 1 {
			t.Fatalf("json=%t status=%d, want1:\n%s%s", jsonOutput, status, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "module_sum_missing") || !strings.Contains(stdout.String(), "github.com/klauspost/compress@v1.18.5") {
			t.Fatalf("missing activation diagnostic:\n%s", stdout.String())
		}
		if jsonOutput {
			var report pinimpact.Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if !report.Invalidated || len(report.Pins) != 2 {
				t.Fatalf("report = %+v, want two unknown pack rules", report)
			}
			for _, pin := range report.Pins {
				if pin.Class != pinimpact.ClassPackRule || pin.Status != pinimpact.StatusUnknown {
					t.Fatalf("pin = %+v, want unknown pack rule", pin)
				}
			}
		}
	}
	if readModuleSnapshot(t, baseline) != beforeBaseline || readModuleSnapshot(t, candidate) != beforeCandidate {
		t.Fatal("pin impact changed a target module")
	}
}

type pinImpactModule struct {
	path, version, sum string
}

type pinImpactFixture struct {
	root, goCommand string
	sentry          pinImpactModule
	baseline        []pinImpactModule
	candidate       []pinImpactModule
}

// newPinImpactFixture serves a baseline that selects the sentry adapter and
// a candidate that bumps it from a file-based module proxy.
func newPinImpactFixture(t *testing.T) pinImpactFixture {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command is unavailable")
	}
	fixture := pinImpactFixture{root: root, goCommand: goCommand}
	for _, identity := range deterministicio.Default().Adapters() {
		if identity.Module == "github.com/getsentry/sentry-go" {
			fixture.sentry = pinImpactModule{path: identity.Module, version: identity.Version, sum: identity.Sum}
		}
	}
	if fixture.sentry.path == "" {
		t.Fatal("adapter registry has no sentry adapter")
	}
	digest := sha256.Sum256([]byte("sentry v0.99.0"))
	bumped := pinImpactModule{path: fixture.sentry.path, version: "v0.99.0", sum: "h1:" + base64.StdEncoding.EncodeToString(digest[:])}
	fixture.baseline = []pinImpactModule{fixture.sentry}
	fixture.candidate = []pinImpactModule{bumped}
	proxy := t.TempDir()
	for _, served := range []pinImpactModule{fixture.sentry, bumped} {
		escaped, err := module.EscapePath(served.path)
		if err != nil {
			t.Fatal(err)
		}
		directory := filepath.Join(proxy, filepath.FromSlash(escaped), "@v")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		for name, contents := range map[string]string{
			served.version + ".mod":  "module " + served.path + "\n\ngo 1.21\n",
			served.version + ".info": fmt.Sprintf(`{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, served.version),
		} {
			if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	t.Setenv("GOSUMDB", "off")
	return fixture
}

func writePinImpactModule(t *testing.T, directory string, modules []pinImpactModule) {
	t.Helper()
	var goMod, goSum strings.Builder
	goMod.WriteString("module example.test/pinimpact\n\ngo 1.27.0\n")
	for _, required := range modules {
		fmt.Fprintf(&goMod, "\nrequire %s %s\n", required.path, required.version)
		fmt.Fprintf(&goSum, "%s %s %s\n", required.path, required.version, required.sum)
	}
	for name, contents := range map[string]string{"go.mod": goMod.String(), "go.sum": goSum.String()} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func readModuleSnapshot(t *testing.T, directory string) string {
	t.Helper()
	var snapshot strings.Builder
	for _, name := range []string{"go.mod", "go.sum"} {
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		snapshot.Write(contents)
	}
	return snapshot.String()
}

func TestRunPinImpactReportsInvalidatedPinsWithoutTouchingTheModule(t *testing.T) {
	fixture := newPinImpactFixture(t)
	baseline, candidate := t.TempDir(), t.TempDir()
	writePinImpactModule(t, baseline, fixture.baseline)
	writePinImpactModule(t, candidate, fixture.candidate)
	before := readModuleSnapshot(t, candidate)
	output := filepath.Join(t.TempDir(), "pin-impact.json")
	arguments := []string{"pin-impact", "--root", fixture.root, "--module", candidate, "--baseline-module", baseline, "--go", fixture.goCommand}

	var stdout, stderr bytes.Buffer
	if status := run(append(arguments, "--json", "--output", output), &stdout, &stderr); status != 1 {
		t.Fatalf("status = %d, want 1; stderr:\n%s", status, stderr.String())
	}
	var report pinimpact.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Invalidated || len(report.Pins) != 1 || report.Pins[0].ID != fixture.sentry.path+"@"+fixture.sentry.version ||
		report.Pins[0].Status != pinimpact.StatusInvalidated || report.Pins[0].CandidateVersion != "v0.99.0" {
		t.Fatalf("report pins = %+v, want only the invalidated sentry adapter", report.Pins)
	}
	written, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(written, stdout.Bytes()) {
		t.Fatalf("--output = %q, %v; want the canonical report", written, err)
	}
	if bytes.Contains(stdout.Bytes(), []byte(candidate)) || bytes.Contains(stdout.Bytes(), []byte(fixture.root)) {
		t.Fatal("canonical report contains a host path")
	}
	if after := readModuleSnapshot(t, candidate); after != before {
		t.Fatalf("candidate module changed:\n%s\nwant:\n%s", after, before)
	}

	stdout.Reset()
	if status := run(arguments, &stdout, &stderr); status != 1 || !strings.Contains(stdout.String(), "invalidated adapter "+fixture.sentry.path+"@") {
		t.Fatalf("human report status = %d:\n%s", status, stdout.String())
	}
	stdout.Reset()
	if status := run([]string{"pin-impact", "--root", fixture.root, "--module", baseline, "--baseline-module", baseline, "--go", fixture.goCommand}, &stdout, &stderr); status != 0 {
		t.Fatalf("unchanged module status = %d, want 0:\n%s%s", status, stdout.String(), stderr.String())
	}
}

// TestRunPinImpactReportsANewerGoDirectiveAsUnknownPins runs a candidate whose
// go directive is newer than the resolving go command through the real
// resolver: the report still renders, with the toolchain-bound pins unknown.
func TestRunPinImpactReportsANewerGoDirectiveAsUnknownPins(t *testing.T) {
	fixture := newPinImpactFixture(t)
	baseline, candidate := t.TempDir(), t.TempDir()
	writePinImpactModule(t, baseline, fixture.baseline)
	writePinImpactModule(t, candidate, fixture.baseline)
	goMod := filepath.Join(candidate, "go.mod")
	contents, err := os.ReadFile(goMod)
	if err != nil {
		t.Fatal(err)
	}
	newer := strings.Replace(string(contents), "\ngo 1.27.0\n", "\ngo 1.99.0\n\ntoolchain go1.99.0\n", 1)
	if newer == string(contents) {
		t.Fatal("fixture go.mod has no go directive to raise")
	}
	if err := os.WriteFile(goMod, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	before := readModuleSnapshot(t, candidate)
	var stdout, stderr bytes.Buffer
	status := run([]string{"pin-impact", "--root", fixture.root, "--module", candidate, "--baseline-module", baseline, "--go", fixture.goCommand, "--json"}, &stdout, &stderr)
	if status != 1 {
		t.Fatalf("status = %d, want 1; stderr:\n%s", status, stderr.String())
	}
	var report pinimpact.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	unknown := map[pinimpact.Class]int{}
	for _, pin := range report.Pins {
		if pin.Status == pinimpact.StatusUnknown {
			unknown[pin.Class]++
			if !strings.Contains(pin.Reason, "requires go 1.99.0") {
				t.Fatalf("unknown pin reason = %q", pin.Reason)
			}
		} else if pin.Class == pinimpact.ClassAdapter {
			t.Fatalf("adapter pin changed status without a version change: %+v", pin)
		}
	}
	if !report.Invalidated || unknown[pinimpact.ClassInterception] == 0 || unknown[pinimpact.ClassClockReference] == 0 {
		t.Fatalf("unknown pins = %v, want interception and clock-reference pins unknown", unknown)
	}
	if after := readModuleSnapshot(t, candidate); after != before {
		t.Fatalf("candidate module changed:\n%s\nwant:\n%s", after, before)
	}
}

// TestRunPinImpactReportsADependencyRequiringNewerGoAsUnknownPins runs a
// candidate whose own go directive is supported but whose dependency requires
// a newer Go, which keeps the go command from resolving the module graph
// under GOTOOLCHAIN=local. Every pin that depends on the graph is unknown.
func TestRunPinImpactReportsADependencyRequiringNewerGoAsUnknownPins(t *testing.T) {
	fixture := newPinImpactFixture(t)
	baseline, candidate := t.TempDir(), t.TempDir()
	writePinImpactModule(t, baseline, fixture.baseline)
	writePinImpactModule(t, candidate, fixture.baseline)
	dependency := filepath.Join(candidate, "dep")
	if err := os.Mkdir(dependency, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dependency, "go.mod"), []byte("module example.test/dep\n\ngo 1.99.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	goMod := filepath.Join(candidate, "go.mod")
	contents, err := os.ReadFile(goMod)
	if err != nil {
		t.Fatal(err)
	}
	contents = append(contents, "\nrequire example.test/dep v0.0.0\n\nreplace example.test/dep => ./dep\n"...)
	if err := os.WriteFile(goMod, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	before := readModuleSnapshot(t, candidate)
	var stdout, stderr bytes.Buffer
	status := run([]string{"pin-impact", "--root", fixture.root, "--module", candidate, "--baseline-module", baseline, "--go", fixture.goCommand, "--json"}, &stdout, &stderr)
	if status != 1 {
		t.Fatalf("status = %d, want 1; stderr:\n%s", status, stderr.String())
	}
	var report pinimpact.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	unknown := map[pinimpact.Class]int{}
	for _, pin := range report.Pins {
		if pin.Status != pinimpact.StatusUnknown {
			t.Fatalf("pin %s is %s, want unknown", pin.ID, pin.Status)
		}
		if !strings.Contains(pin.Reason, "module graph requires go 1.99.0") {
			t.Fatalf("unknown pin reason = %q", pin.Reason)
		}
		unknown[pin.Class]++
	}
	if !report.Invalidated || unknown[pinimpact.ClassAdapter] != 1 || unknown[pinimpact.ClassInterception] == 0 || unknown[pinimpact.ClassClockReference] == 0 {
		t.Fatalf("unknown pins = %v, want the sentry adapter and the toolchain-bound pins unknown", unknown)
	}
	if after := readModuleSnapshot(t, candidate); after != before {
		t.Fatalf("candidate module changed:\n%s\nwant:\n%s", after, before)
	}
}

func TestRunPinImpactReadsTheBaselineFromGit(t *testing.T) {
	fixture := newPinImpactFixture(t)
	repository := t.TempDir()
	writePinImpactModule(t, repository, fixture.baseline)
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"add", "go.mod", "go.sum"},
		{"-c", "user.name=Gomad Test", "-c", "user.email=gomad@example.test", "commit", "-q", "-m", "baseline"},
	} {
		command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	writePinImpactModule(t, repository, fixture.candidate)
	var stdout, stderr bytes.Buffer
	if status := run([]string{"pin-impact", "--root", fixture.root, "--module", repository, "--go", fixture.goCommand}, &stdout, &stderr); status != 1 {
		t.Fatalf("status = %d, want 1:\n%s%s", status, stdout.String(), stderr.String())
	}
	if status := run([]string{"pin-impact", "--root", fixture.root, "--module", repository, "--baseline-ref", "missing-revision", "--go", fixture.goCommand}, &stdout, &stderr); status != 2 {
		t.Fatalf("missing baseline revision status = %d, want 2", status)
	}
}

func TestRunPinImpactStatuses(t *testing.T) {
	fixture := newPinImpactFixture(t)
	baseline, malformed := t.TempDir(), t.TempDir()
	writePinImpactModule(t, baseline, fixture.baseline)
	if err := os.WriteFile(filepath.Join(malformed, "go.mod"), []byte("module\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	common := []string{"--root", fixture.root, "--go", fixture.goCommand}
	for name, test := range map[string]struct {
		arguments []string
		proxy     string
		want      int
	}{
		"both baselines":   {arguments: []string{"--module", baseline, "--baseline-module", baseline, "--baseline-ref", "HEAD"}, want: 2},
		"malformed module": {arguments: []string{"--module", malformed, "--baseline-module", baseline}, want: 2},
		"missing module":   {arguments: []string{"--module", filepath.Join(baseline, "missing"), "--baseline-module", baseline}, want: 2},
		"unreachable proxy": {
			arguments: []string{"--module", baseline, "--baseline-module", baseline}, proxy: "off", want: 3,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if test.proxy != "" {
				t.Setenv("GOPROXY", test.proxy)
			}
			var stdout, stderr bytes.Buffer
			if status := run(append(append([]string{"pin-impact"}, common...), test.arguments...), &stdout, &stderr); status != test.want {
				t.Fatalf("status = %d, want %d:\n%s%s", status, test.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunPinImpactFileFlagsKeepTheV1Report(t *testing.T) {
	fixture := newPinImpactFixture(t)
	baseline, candidate := t.TempDir(), t.TempDir()
	writePinImpactModule(t, baseline, fixture.baseline)
	writePinImpactModule(t, candidate, fixture.candidate)
	t.Setenv("GOMAD3_BOOTSTRAP_GO", fixture.goCommand)
	var stdout, stderr bytes.Buffer
	status := run([]string{
		"pin-impact", "--root", fixture.root,
		"--baseline", filepath.Join(baseline, "go.mod"),
		"--candidate", filepath.Join(candidate, "go.mod"), "--format=json",
	}, &stdout, &stderr)
	if status != 1 || !strings.Contains(stdout.String(), `"schema":"gomad3.pin-impact/v1"`) ||
		!strings.Contains(stdout.String(), `"class":"adapter"`) ||
		!strings.Contains(stdout.String(), `"class":"interception_fingerprint"`) ||
		!strings.Contains(stdout.String(), `"class":"clock_inventory_reference"`) ||
		!strings.Contains(stdout.String(), `"source_set_sha256"`) {
		t.Fatalf("file report status = %d, output = %s, error = %s", status, stdout.String(), stderr.String())
	}
}
