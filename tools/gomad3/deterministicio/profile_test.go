package deterministicio

import (
	"encoding/hex"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
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
		inventorySHA256:      "sha256:0d34edfc0ab13c693b78d514ca1a72a9adc196fe61e2142dba108c0ff47358f7",
		implementationSHA256: "sha256:9cd0cff9595bb7f79ec3de247031165cc96d7b40c0e8133051ec17afbf6ac7c0",
		frameHex:             "474f4d4144494f01000100010d34edfc0ab13c693b78d514ca1a72a9adc196fe61e2142dba108c0ff47358f79cd0cff9595bb7f79ec3de247031165cc96d7b40c0e8133051ec17afbf6ac7c0aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb454c9d4564fd6ff285e8b3392ac6eee3fa398987997bcbc9ce24b643068bc72d000000000000002ab4c04364a4ed76089d71bfdb3dbb48f8b0366cdc4203510477e0e79f4ea7a90a",
	},
	"linux/amd64": {
		inventorySHA256:      "sha256:b556d860816c57c032f82e68dfe120ed6df6978be23424ce3ecd68acf2a7907f",
		implementationSHA256: "sha256:84b27e6227508fda72c8be6b0ae588e4f4f052a07bc35a27d6feff729b22d50b",
		frameHex:             "474f4d4144494f0100010001b556d860816c57c032f82e68dfe120ed6df6978be23424ce3ecd68acf2a7907f84b27e6227508fda72c8be6b0ae588e4f4f052a07bc35a27d6feff729b22d50baaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb454c9d4564fd6ff285e8b3392ac6eee3fa398987997bcbc9ce24b643068bc72d000000000000002ad096cdd84074cb0a907390076ea66811cc76d2ea7fbcb7fb44bf406bdd53fbd8",
	},
}

const wantInventoryTemplate = `{"boundary_manifest_sha256":"sha256:ca18b6934d906b95235e04f83dfc2eef0a94d17d5417eb032cd086f7425ebbd0","boundary_manifest_version":"go1.27.1-v1","entries":[{"boundary":"crypto/rand","disposition":"in-memory","operations":["Reader.Read","Read"]},{"boundary":"filesystem","disposition":"in-memory","operations":["open","read","write","stat","rename","remove","mkdir","map"]},{"boundary":"io-transcript","disposition":"shared-memory","operations":["expected-replay","record","terminal"]},{"boundary":"github.com/Masterminds/sprig/v3","disposition":"target-adapter","operations":["host-dns-refusal"]},{"boundary":"github.com/cactus/go-statsd-client/v5","disposition":"target-adapter","operations":["udp-sender-refusal"]},{"boundary":"github.com/cockroachdb/pebble","disposition":"target-adapter","operations":["os-file-construction-refusal","hard-link-refusal"]},{"boundary":"github.com/getsentry/sentry-go","disposition":"target-adapter","operations":["optional-git-release-suppression"]},{"boundary":"github.com/go-playground/validator/v10","disposition":"target-adapter","operations":["address-resolution-refusal"]},{"boundary":"github.com/hashicorp/go-metrics","disposition":"target-adapter","operations":["signal-service-refusal"]},{"boundary":"github.com/hashicorp/go-sockaddr","disposition":"target-adapter","operations":["route-command-denial","interface-discovery-refusal","literal-address-parsing"]},{"boundary":"github.com/hashicorp/memberlist","disposition":"target-adapter","operations":["native-udp-transport-refusal"]},{"boundary":"go.opentelemetry.io/otel/sdk","disposition":"target-adapter","operations":["process-owner-placeholder","uname-placeholder","host-command-denial"]},{"boundary":"go.temporal.io/sdk","disposition":"target-adapter","operations":["interrupt-channel-suppression"]},{"boundary":"go.uber.org/fx","disposition":"target-adapter","operations":["signal-relay-suppression"]},{"boundary":"golang.org/x/net","disposition":"target-adapter","operations":["raw-socket-option-denial"]},{"boundary":"google.golang.org/grpc","disposition":"target-adapter","operations":["virtual-tcp-keepalive-suppression","portable-syscall-removal","host-dns-refusal"]},{"boundary":"modernc.org/libc","disposition":"target-adapter","operations":["filesystem","entropy","time"]},{"boundary":"modernc.org/memory","disposition":"target-adapter","operations":["anonymous-memory"]},{"boundary":"net","disposition":"in-memory","operations":["Dial","DialTCP","Dialer.DialContext","Listen","ListenConfig.Listen","ListenTCP","Resolver.LookupIPAddr(localhost)"]},{"boundary":"os.read-only-mount","disposition":"lazy-in-memory","operations":["open","read","stat","readdir"]}],"platform":"%s","profile":"gomad3-deterministic/v1","reserved_fds":["bootstrap","expected-transcript","io-config","io-terminal","stderr","stdout","transcript","world-config","world-record","read-only-mount-request","read-only-mount-response"],"schema":"gomad3.io-inventory/v1"}`

func TestDeterministicProfileCompatibilityGolden(t *testing.T) {
	for _, platform := range slices.Sorted(maps.Keys(profileGoldens)) {
		golden := profileGoldens[platform]
		goos, goarch, _ := strings.Cut(platform, "/")
		profile := mustSpec(profileDefinition{
			name:                  Deterministic,
			target:                TargetContract{GoVersion: generatedBoundaryGoVersion, GOOS: goos, GOARCH: goarch},
			implementationFamily:  "gomad3.deterministic-io/v1",
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
		Argv:   []string{"gomad3-target", "argument"},
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
	want := Contract{
		Name:                 profile.Name(),
		ImplementationSHA256: profile.ImplementationSHA256(),
		InventorySHA256:      profile.InventorySHA256(),
	}
	if identity != want {
		t.Fatalf("identity = %#v, want %#v", identity, want)
	}
	if !profile.Matches(identity) {
		t.Fatal("profile rejected its identity")
	}
	mutations := map[string]Contract{
		"name":           {Name: "other", ImplementationSHA256: identity.ImplementationSHA256, InventorySHA256: identity.InventorySHA256},
		"implementation": {Name: identity.Name, ImplementationSHA256: Digest(record.HashBytes([]byte("other"))), InventorySHA256: identity.InventorySHA256},
		"inventory":      {Name: identity.Name, ImplementationSHA256: identity.ImplementationSHA256, InventorySHA256: Digest(record.HashBytes([]byte("other")))},
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
		Kind: target.KindGoTest, Source: "./pkg", Argv: []string{"gomad3-target", argument}, BuildTags: []string{"gomad_fixture"},
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
		Kind: target.KindGoTest, Source: "./pkg", Argv: []string{"gomad3-target", "-test.run=^TestScenario$"},
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
		Argv:   []string{"gomad3-target", "-test.run=^TestScenario$"},
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
