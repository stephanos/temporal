package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
)

func TestDoctorOutput(t *testing.T) {
	root, _, artifacts := writeDoctorFixture(t, runtime.GOOS, runtime.GOARCH)
	toolchain := filepath.Join(root, ".toolchain")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	runner := fmt.Sprintf("sha256:%x", sha256.Sum256(binary))
	host := runtime.GOOS + "/" + runtime.GOARCH
	available, status := false, 1
	hostStatus, hostDetail := "error", host+" is unsupported; supported=darwin/arm64,linux/amd64"
	hostRow := "host       error " + hostDetail + "\n"
	if host == "darwin/arm64" || host == "linux/amd64" {
		available, status, hostRow = true, 0, "host       ok    "+host+"\n"
		hostStatus, hostDetail = "ok", host
	}
	key := strings.Repeat("a", 64)
	repair := "install the Gomad toolchain at " + toolchain
	text := []string{
		fmt.Sprintf("gomad doctor: available=%t host=%s go=go1.27.1 toolchain=%s runner=%s boundary=go1.27.1-v1\n", available, host, key, runner),
		hostRow,
		"toolchain  ok    go1.27.1 build=" + key + "\n",
		"runner     ok    " + runner + "\n",
	}
	var adapters, checks []string
	checks = append(checks,
		fmt.Sprintf(`{"name":"host","status":%q,"detail":%q}`, hostStatus, hostDetail),
		fmt.Sprintf(`{"name":"toolchain","status":"ok","detail":%q}`, "go1.27.1 build="+key),
		fmt.Sprintf(`{"name":"runner","status":"ok","detail":%q}`, runner),
	)
	for _, adapter := range []struct{ module, version, sum string }{
		{"github.com/Masterminds/sprig/v3", "v3.3.0", "h1:mQh0Yrg1XPo6vjYXgtf5OtijNAKJRNcTdOOGZe3tPhs="},
		{"github.com/cactus/go-statsd-client/v5", "v5.1.0", "h1:sbbdfIl9PgisjEoXzvXI1lwUKWElngsjJKaZeC021P4="},
		{"github.com/cockroachdb/pebble", "v0.0.0-20260703021901-41f35d3cb7df", "h1:p7vkumDcPw0de7t8pYA95HPC4cYQZGDG6b57d4Om5cA="},
		{"github.com/getsentry/sentry-go", "v0.46.0", "h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0="},
		{"github.com/go-playground/validator/v10", "v10.30.1", "h1:f3zDSN/zOma+w6+1Wswgd9fLkdwy06ntQJp0BBvFG0w="},
		{"github.com/hashicorp/go-metrics", "v0.5.4", "h1:8mmPiIJkTPPEbAiV97IxdAGNdRdaWwVap1BU6elejKY="},
		{"github.com/hashicorp/go-sockaddr", "v1.0.7", "h1:G+pTkSO01HpR5qCxg7lxfsFEZaG+C0VssTy/9dbT+Fw="},
		{"github.com/hashicorp/memberlist", "v0.5.4", "h1:40YY+3qq2tAUhZIMEK8kqusKZBBjdwJ3NUjvYkcxh74="},
		{"go.opentelemetry.io/otel/sdk", "v1.44.0", "h1:nHYwb9lK+fJPU/dnT6s7W7Z8itMWyqrnVfbheVYrZ58="},
		{"go.temporal.io/sdk", "v1.48.0", "h1:WDctKDVuh0Z8Nf7euAyqs/EwcPg1JTIIq1Fut8Tq118="},
		{"go.uber.org/fx", "v1.24.0", "h1:wE8mruvpg2kiiL1Vqd0CC+tr0/24XIB10Iwp2lLWzkg="},
		{"golang.org/x/net", "v0.58.0", "h1:ynWG7rqYi4ccpTEuPZ2QGWHktVEM9DMCj9yzDE0Q7To="},
		{"google.golang.org/grpc", "v1.83.2", "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="},
		{"modernc.org/libc", "v1.72.3", "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="},
		{"modernc.org/memory", "v1.11.0", "h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI="},
	} {
		name := "adapter:" + adapter.module
		detail := adapter.module + "@" + adapter.version + " " + adapter.sum
		text = append(text, name+" ok    "+detail+"\n")
		adapters = append(adapters, fmt.Sprintf(`{"module":%q,"version":%q,"sum":%q,"status":"available"}`, adapter.module, adapter.version, adapter.sum))
		checks = append(checks, fmt.Sprintf(`{"name":%q,"status":"ok","detail":%q}`, name, detail))
	}
	text = append(text, "artifacts  ok    "+artifacts+"\n", "installation: source=cli toolchain="+toolchain+"\nrepair: "+repair+"\n")
	checks = append(checks, fmt.Sprintf(`{"name":"artifacts","status":"ok","detail":%q}`, artifacts))
	identity := deterministicio.Default().Identity()
	jsonReport := fmt.Sprintf(`{"schema":"gomad3.doctor/v3","available":%t,"host":%q,"supported_platforms":["darwin/arm64","linux/amd64"],"go_version":"go1.27.1","toolchain_build":%q,"runner_build":%q,"boundary_manifest_version":"go1.27.1-v1","io_inventory_sha256":%q,"io_implementation_sha256":%q,"adapters":[%s],"installation_source":"cli","toolchain_root":%q,"artifact_directory":%q,"repair_instruction":%q,"checks":[%s]}`+"\n", available, host, key, runner, identity.InventorySHA256, identity.ImplementationSHA256, strings.Join(adapters, ","), toolchain, artifacts, repair, strings.Join(checks, ","))
	for _, test := range []struct {
		name    string
		json    bool
		failure int
	}{
		{"text healthy", false, 0},
		{"JSON healthy", true, 0},
		{"JSON failure", true, 1},
		{"headline failure", false, 1},
		{"first row failure", false, 2},
		{"interior row failure", false, 11},
		{"final row failure", false, 20},
		{"footer failure", false, 21},
	} {
		t.Run(test.name, func(t *testing.T) {
			arguments := []string{"doctor", "--toolchain-root=" + toolchain, "--artifacts=" + artifacts}
			attempts := text
			if test.json {
				arguments = append(arguments, "--json")
				attempts = []string{jsonReport}
			}
			wantStatus := status
			var failures map[int]bool
			if test.failure != 0 {
				wantStatus = 3
				failures = map[int]bool{test.failure: true}
				attempts = attempts[:test.failure]
			}
			stdout := terminalDiagnostics(t, failures)
			var stderr bytes.Buffer
			if got := Run(arguments, stdout, &stderr); got != wantStatus {
				t.Errorf("status = %d, want %d", got, wantStatus)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			checkTerminalDiagnostics(t, stdout, attempts)
			entries, err := os.ReadDir(artifacts)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("artifact probe left entries: %v", entries)
			}
		})
	}
}

func TestDoctorOutputEarlierErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		arguments  []string
		diagnostic string
	}{
		{"flag", []string{"--unknown"}, "flag provided but not defined: -unknown\nUsage of gomad doctor:\n  -artifacts string\n    \tartifact root to verify (default \".gomad/artifacts\")\n  -json\n    \temit stable JSON\n  -toolchain-root string\n    \tabsolute pinned toolchain root\n"},
		{"argument", []string{"extra"}, usage},
		{"root", []string{"--toolchain-root=relative"}, "resolve Gomad installation: CLI --toolchain-root toolchain root must be an absolute non-root clean path: \"relative\"\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifacts := filepath.Join(t.TempDir(), "unprobed")
			arguments := append([]string{"doctor", "--artifacts=" + artifacts}, test.arguments...)
			stdout := terminalDiagnostics(t, map[int]bool{1: true})
			var stderr bytes.Buffer
			if got := Run(arguments, stdout, &stderr); got != 2 {
				t.Fatalf("status = %d, want 2", got)
			}
			if stderr.String() != test.diagnostic {
				t.Fatalf("stderr = %q, want %q", stderr.String(), test.diagnostic)
			}
			checkTerminalDiagnostics(t, stdout, nil)
			if _, err := os.Stat(artifacts); !os.IsNotExist(err) {
				t.Fatalf("artifact path probed before input rejection: %v", err)
			}
		})
	}
}
