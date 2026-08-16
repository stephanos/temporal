package deterministicio

import (
	"encoding/hex"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target"
)

func TestDeterministicProfileCompatibilityGolden(t *testing.T) {
	profile := Default()
	const wantInventory = `{"boundary_manifest_sha256":"sha256:febc956871f3bd6885b4af2daed7653705eb6f03d5df2784decea946a849aa14","boundary_manifest_version":"go1.26.4-darwin-arm64-v1","entries":[{"boundary":"crypto/rand","disposition":"in-memory","operations":["Reader.Read","Read"]},{"boundary":"filesystem","disposition":"in-memory","operations":["open","read","write","stat","rename","remove","mkdir"]},{"boundary":"io-transcript","disposition":"shared-memory","operations":["expected-replay","record","terminal"]},{"boundary":"google.golang.org/grpc","disposition":"target-adapter","operations":["virtual-tcp-keepalive-suppression"]},{"boundary":"modernc.org/libc","disposition":"target-adapter","operations":["filesystem","entropy","time"]},{"boundary":"net","disposition":"in-memory","operations":["Dial","DialTCP","Dialer.DialContext","Listen","ListenConfig.Listen","ListenTCP"]},{"boundary":"os.read-only-mount","disposition":"lazy-in-memory","operations":["open","read","stat","readdir"]}],"platform":"darwin/arm64","profile":"gomadv3-deterministic/v1","reserved_fds":["bootstrap","expected-transcript","io-config","io-terminal","stderr","stdout","transcript","world-config","world-record","read-only-mount-request","read-only-mount-response"],"schema":"gomadv3.io-inventory/v1"}`
	const wantInventorySHA256 = "sha256:57f3f2a3f87edac15f8723765e61a909efcca28bb9e8feb7a67261b8e9ae8528"
	const wantImplementationSHA256 = "sha256:034755da63de6446baa5c7fefaaecaeb03c1e18c753ed18fcedbf17a76813610"
	if string(profile.Inventory()) != wantInventory || string(profile.InventorySHA256()) != wantInventorySHA256 || string(profile.ImplementationSHA256()) != wantImplementationSHA256 {
		t.Fatalf("profile identity:\n inventory = %q\n inventory SHA-256 = %q\n implementation SHA-256 = %q", profile.Inventory(), profile.InventorySHA256(), profile.ImplementationSHA256())
	}
	frame, err := profile.BootstrapFrame(target.Prepared{
		SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Argv:   []string{"gomadv3-target", "argument"},
	}, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 42)
	if err != nil {
		t.Fatal(err)
	}
	const wantFrameHex = "474f4d4144494f010001000157f3f2a3f87edac15f8723765e61a909efcca28bb9e8feb7a67261b8e9ae8528034755da63de6446baa5c7fefaaecaeb03c1e18c753ed18fcedbf17a76813610aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb26ba1df2e711e0add254aeb48b2779e9aa32a01c3d07c2ee506cadf29cfec8ff000000000000002a011d42d5d049c180d0b0b8d09b2b6f3f31c929bea9b53f29bf3ce9ba147d0a03"
	if encoded := hex.EncodeToString(frame); encoded != wantFrameHex {
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
	if got, want := first.TargetContract(), (TargetContract{GoVersion: "go1.26.4", GOOS: "darwin", GOARCH: "arm64"}); got != want {
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
		"implementation": {Name: identity.Name, ImplementationSHA256: Digest(evidence.HashBytes([]byte("other"))), InventorySHA256: identity.InventorySHA256},
		"inventory":      {Name: identity.Name, ImplementationSHA256: identity.ImplementationSHA256, InventorySHA256: Digest(evidence.HashBytes([]byte("other")))},
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
		Kind: target.KindGoTest, Source: "./pkg", Argv: []string{"gomadv3-target", argument}, BuildTags: []string{"gomad_fixture"},
		Adapters: []evidence.TargetAdapter{}, BuildInfo: evidence.BuildInfo{Path: "example.test/project/pkg.test"}, GoVersion: "go1.26.4", TargetGOOS: "darwin", TargetGOARCH: "arm64",
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
		Kind: target.KindGoTest, Source: "./pkg", Argv: []string{"gomadv3-target", "-test.run=^TestScenario$"},
		BuildTags: []string{"gomad_fixture"}, Adapters: []evidence.TargetAdapter{}, BuildInfo: evidence.BuildInfo{Path: "example.test/project/pkg.test"},
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
		Argv:   []string{"gomadv3-target", "-test.run=^TestScenario$"},
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
