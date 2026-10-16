package transcript

import (
	"encoding/hex"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

// profileGoldens pins the deterministic profile identity per qualified
// platform. The inventory differs only in its platform field, so every entry
// is checked on every host; the bootstrap frame binds the host profile and is
// checked against the host golden.
var profileGoldens = map[string]struct {
	inventorySHA256      string
	implementationSHA256 string
	frameHex             string
}{
	"darwin/arm64": {
		inventorySHA256:      "sha256:8e5c81235354dbae268d8c333736b25c23c519cb4a5022ce12c1aa598efdb9ed",
		implementationSHA256: "sha256:b6a41f88d2f60e098713d75172d99b025e57ddb285ff54580e2a945bbce02b07",
		frameHex:             "474f4d4144494f01000100018e5c81235354dbae268d8c333736b25c23c519cb4a5022ce12c1aa598efdb9edb6a41f88d2f60e098713d75172d99b025e57ddb285ff54580e2a945bbce02b07aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbf93825e38f9b74b7d0e738f28e7a1530e367538708704c9753280ab650022c78000000000000002afab1d583f18af2a59836467b7aa56476f0821ce8c62ad4206445c06f80c145bf",
	},
	"linux/amd64": {
		inventorySHA256:      "sha256:cc6c31b19bbc4193c917c5eff1f57923f69f21aa9314242dcc40e61342b0fdd4",
		implementationSHA256: "sha256:85d8b71f44b6ab31eb9704adb1ee5d1d545ed35a4ab312284f8e248557c90575",
		frameHex:             "474f4d4144494f0100010001cc6c31b19bbc4193c917c5eff1f57923f69f21aa9314242dcc40e61342b0fdd485d8b71f44b6ab31eb9704adb1ee5d1d545ed35a4ab312284f8e248557c90575aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbf93825e38f9b74b7d0e738f28e7a1530e367538708704c9753280ab650022c78000000000000002adde1b4f54bf3b5b120b09d8dd2ec713150a09f893d043cfbef3cab17e8ca8cd5",
	},
}

const wantInventoryTemplate = `{"boundary_manifest_sha256":"sha256:d1fb1620a54d56775e6fb3d139403524e9ce8999ce8d376af3ba18a62b3b6f55","boundary_manifest_version":"go1.27.1-v1","entries":[{"boundary":"crypto/rand","disposition":"in-memory","operations":["Reader.Read","Read"]},{"boundary":"filesystem","disposition":"in-memory","operations":["open","read","write","stat","rename","remove","mkdir","map"]},{"boundary":"io-transcript","disposition":"shared-memory","operations":["expected-replay","record","terminal"]},{"boundary":"github.com/Masterminds/sprig/v3","disposition":"target-adapter","operations":["host-dns-refusal"]},{"boundary":"github.com/cactus/go-statsd-client/v5","disposition":"target-adapter","operations":["udp-sender-refusal"]},{"boundary":"github.com/cockroachdb/pebble","disposition":"target-adapter","operations":["os-file-construction-refusal","hard-link-refusal"]},{"boundary":"github.com/getsentry/sentry-go","disposition":"target-adapter","operations":["optional-git-release-suppression"]},{"boundary":"github.com/go-playground/validator/v10","disposition":"target-adapter","operations":["address-resolution-refusal"]},{"boundary":"github.com/hashicorp/go-metrics","disposition":"target-adapter","operations":["signal-service-refusal"]},{"boundary":"github.com/hashicorp/go-sockaddr","disposition":"target-adapter","operations":["route-command-denial","interface-discovery-refusal","literal-address-parsing"]},{"boundary":"github.com/hashicorp/memberlist","disposition":"target-adapter","operations":["native-udp-transport-refusal"]},{"boundary":"go.opentelemetry.io/otel/sdk","disposition":"target-adapter","operations":["process-owner-placeholder","uname-placeholder","host-command-denial"]},{"boundary":"go.temporal.io/sdk","disposition":"target-adapter","operations":["interrupt-channel-suppression"]},{"boundary":"go.uber.org/fx","disposition":"target-adapter","operations":["signal-relay-suppression"]},{"boundary":"golang.org/x/net","disposition":"target-adapter","operations":["raw-socket-option-denial"]},{"boundary":"google.golang.org/grpc","disposition":"target-adapter","operations":["virtual-tcp-keepalive-suppression","portable-syscall-removal","host-dns-refusal"]},{"boundary":"modernc.org/libc","disposition":"target-adapter","operations":["filesystem","entropy","time"]},{"boundary":"modernc.org/memory","disposition":"target-adapter","operations":["anonymous-memory"]},{"boundary":"net","disposition":"in-memory","operations":["Dial","DialTCP","Dialer.DialContext","Listen","ListenConfig.Listen","ListenTCP","Resolver.LookupIPAddr(localhost)"]},{"boundary":"os.read-only-mount","disposition":"lazy-in-memory","operations":["open","read","stat","readdir"]}],"platform":"%s","profile":"gomad-deterministic/v1","reserved_fds":["bootstrap","expected-transcript","io-config","io-terminal","stderr","stdout","transcript","world-config","world-record","read-only-mount-request","read-only-mount-response"],"schema":"gomad.io-inventory/v1"}`

func TestDeterministicProfileCompatibilityGolden(t *testing.T) {
	for _, platform := range slices.Sorted(maps.Keys(profileGoldens)) {
		golden := profileGoldens[platform]
		goos, goarch, _ := strings.Cut(platform, "/")
		profile := mustSpec(profileDefinition{
			name:                  Deterministic,
			target:                TargetContract{GoVersion: generatedBoundaryGoVersion, GOOS: goos, GOARCH: goarch},
			implementationFamily:  "gomad.deterministic-io/v1",
			implementationVersion: deterministicImplementationVersion,
			adapters:              deterministicAdapters,
		})
		wantInventory := fmt.Sprintf(wantInventoryTemplate, platform)
		if string(profile.Inventory()) != wantInventory || string(profile.InventorySHA256()) != golden.inventorySHA256 || string(profile.ImplementationSHA256()) != golden.implementationSHA256 {
			t.Fatalf("%s profile identity:\n inventory = %q\n inventory SHA-256 = %q\n implementation SHA-256 = %q", platform, profile.Inventory(), profile.InventorySHA256(), profile.ImplementationSHA256())
		}
	}

	host := runtime.GOOS + "/" + runtime.GOARCH
	golden, qualified := profileGoldens[host]
	if !qualified {
		t.Skipf("no profile golden for %s", host)
	}
	if Default().InventorySHA256() != Digest(golden.inventorySHA256) {
		t.Fatalf("default profile inventory = %s, want the %s golden", Default().InventorySHA256(), host)
	}
	frame, err := Default().BootstrapFrame(target.Prepared{
		SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Argv:   []string{"gomad-target", "argument"},
	}, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 42)
	if err != nil {
		t.Fatal(err)
	}
	if golden.frameHex == "" {
		t.Skipf("record the %s bootstrap frame golden: %s", host, hex.EncodeToString(frame))
	}
	if encoded := hex.EncodeToString(frame); encoded != golden.frameHex {
		t.Fatalf("bootstrap frame = %q", encoded)
	}
}

func TestDefaultProfile(t *testing.T) {
	profile := Default()
	if profile.Name() != Deterministic {
		t.Fatalf("profile name = %q", profile.Name())
	}
	if len(profile.Inventory()) == 0 || profile.InventorySHA256() == "" || profile.ImplementationSHA256() == "" {
		t.Fatalf("profile identity is incomplete: %#v", profile)
	}
}

func TestDefaultReturnsAnImmutableProfileSpecification(t *testing.T) {
	first := Default()
	inventory := first.Inventory()
	inventory[0] ^= 1
	second := Default()
	if first.Name() != Deterministic || second.Name() != Deterministic {
		t.Fatalf("profile names = %q, %q", first.Name(), second.Name())
	}
	if string(second.Inventory()) == string(inventory) {
		t.Fatal("resolved profile inventory was mutable")
	}
	if got, want := first.TargetContract(), (TargetContract{GoVersion: "go1.27.1", GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}); got != want {
		t.Fatalf("target contract = %#v, want %#v", got, want)
	}
}

func TestProfileOwnsIdentityProjectionAndVerification(t *testing.T) {
	profile := Default()
	identity := profile.Identity()
	want := record.IOContract{
		Name:                 profile.Name(),
		ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()),
		InventorySHA256:      record.SHA256(profile.InventorySHA256()),
	}
	if identity != want {
		t.Fatalf("identity = %#v, want %#v", identity, want)
	}
	if !profile.Matches(identity) {
		t.Fatal("profile rejected its identity")
	}
	mutations := map[string]record.IOContract{
		"name":           {Name: "other", ImplementationSHA256: identity.ImplementationSHA256, InventorySHA256: identity.InventorySHA256},
		"implementation": {Name: identity.Name, ImplementationSHA256: record.HashBytes([]byte("other")), InventorySHA256: identity.InventorySHA256},
		"inventory":      {Name: identity.Name, ImplementationSHA256: identity.ImplementationSHA256, InventorySHA256: record.HashBytes([]byte("other"))},
	}
	for name, changed := range mutations {
		t.Run(name, func(t *testing.T) {
			if profile.Matches(changed) {
				t.Fatal("profile accepted changed identity")
			}
		})
	}

	if !profile.MatchesRecorded(identity.Name, string(identity.ImplementationSHA256), string(identity.InventorySHA256), string(profile.Inventory())) {
		t.Fatal("profile rejected its record identity")
	}
	if profile.MatchesRecorded(identity.Name, string(identity.ImplementationSHA256), string(identity.InventorySHA256), "changed") {
		t.Fatal("profile accepted changed record inventory")
	}
}

func TestDeterministicProfileAcceptsArbitraryTargetArguments(t *testing.T) {
	profile := Default()
	argument := "-test.run=^TestUnrelatedSuite$"
	err := profile.ValidatePreparedTarget(target.Spec{Kind: target.KindGoTest, Source: "./pkg", Args: []string{argument}}, target.Prepared{
		Kind: target.KindGoTest, Source: "./pkg", Argv: []string{"gomad-target", argument}, BuildTags: []string{"gomad_fixture"},
		Adapters: []record.TargetAdapter{}, BuildInfo: record.BuildInfo{Path: "example.test/project/pkg.test"}, GoVersion: "go1.27.1", TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidatePreparedTargetRejectsIdentityMismatch(t *testing.T) {
	profile := Default()
	validSpec := target.Spec{
		Kind: target.KindGoTest, Source: "./pkg", Args: []string{"-test.run=^TestScenario$"},
	}
	validPrepared := target.Prepared{
		Kind: target.KindGoTest, Source: "./pkg", Argv: []string{"gomad-target", "-test.run=^TestScenario$"},
		BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, BuildInfo: record.BuildInfo{Path: "example.test/project/pkg.test"},
		GoVersion: "go1.26.4", TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}
	tests := map[string]struct {
		spec        target.Spec
		prepared    target.Prepared
		environment []string
	}{
		"kind":     {spec: withKind(validSpec, target.KindGoRun), prepared: validPrepared},
		"source":   {spec: withSource(validSpec, "example.test/project/other"), prepared: validPrepared},
		"argument": {spec: withArgs(validSpec, "-test.run=^TestScenario$", "-test.count=1"), prepared: validPrepared},
		"platform": {spec: validSpec, prepared: withPlatform(validPrepared, "linux", "arm64")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := profile.ValidatePreparedTarget(test.spec, test.prepared, test.environment); err == nil {
				t.Fatal("ValidatePreparedTarget succeeded")
			}
		})
	}
}

func TestBootstrapFrameBindsLaunchIdentity(t *testing.T) {
	profile := Default()
	prepared := target.Prepared{
		SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Argv:   []string{"gomad-target", "-test.run=^TestScenario$"},
	}
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 42)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBootstrapFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Profile != profile.Name() || decoded.TargetSHA256 != prepared.SHA256 || decoded.RunnerSHA256 == "" || decoded.ArgvSHA256 == "" || decoded.InventorySHA256 != profile.InventorySHA256() || decoded.ImplementationSHA256 != profile.ImplementationSHA256() || decoded.Seed != 42 {
		t.Fatalf("decoded frame = %#v", decoded)
	}

	changed := append([]byte(nil), frame...)
	changed[len(changed)-1] ^= 1
	if _, err := DecodeBootstrapFrame(changed); err == nil {
		t.Fatal("DecodeBootstrapFrame accepted changed frame")
	}
}

func withKind(spec target.Spec, kind target.Kind) target.Spec {
	spec.Kind = kind
	return spec
}

func withSource(spec target.Spec, source string) target.Spec {
	spec.Source = source
	return spec
}

func withArgs(spec target.Spec, arguments ...string) target.Spec {
	spec.Args = arguments
	return spec
}

func withPlatform(prepared target.Prepared, goos, goarch string) target.Prepared {
	prepared.TargetGOOS = goos
	prepared.TargetGOARCH = goarch
	return prepared
}
