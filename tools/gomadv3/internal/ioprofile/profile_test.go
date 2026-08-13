package ioprofile

import (
	"encoding/hex"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestDeterministicProfileCompatibilityGolden(t *testing.T) {
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	const wantInventory = `{"boundary_manifest_sha256":"sha256:ae8153017485b5277251a541c79b24b894da6774c202f4a86b56e66617e115a2","boundary_manifest_version":"go1.26.4-darwin-arm64-v1","entries":[{"boundary":"crypto/rand","disposition":"in-memory","operations":["Reader.Read","Read"]},{"boundary":"filesystem","disposition":"in-memory","operations":["open","read","write","stat","rename","remove","mkdir"]},{"boundary":"io-transcript","disposition":"shared-memory","operations":["expected-replay","record","terminal"]},{"boundary":"modernc.org/libc","disposition":"target-adapter","operations":["filesystem","entropy","time"]},{"boundary":"net","disposition":"in-memory","operations":["Dial","DialTCP","Dialer.DialContext","Listen","ListenConfig.Listen","ListenTCP"]},{"boundary":"os.read-only-mount","disposition":"lazy-in-memory","operations":["open","read","stat","readdir"]}],"platform":"darwin/arm64","profile":"gomadv3-deterministic/v1","reserved_fds":["bootstrap","expected-transcript","io-config","io-terminal","stderr","stdout","transcript","world-config","world-record","read-only-mount-request","read-only-mount-response"],"schema":"gomadv3.io-inventory/v1"}`
	const wantInventorySHA256 = "sha256:a93863f00737eee971feb0547eed55387c0cbe34b6ec8def3c2e5b6566c4686f"
	const wantImplementationSHA256 = "sha256:70402c9a81972264dcdda0d7c26a4661c9b7370f69a50ec94cc85432f15e4b5f"
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
	const wantFrameHex = "474f4d4144494f0100010001a93863f00737eee971feb0547eed55387c0cbe34b6ec8def3c2e5b6566c4686f70402c9a81972264dcdda0d7c26a4661c9b7370f69a50ec94cc85432f15e4b5faaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb26ba1df2e711e0add254aeb48b2779e9aa32a01c3d07c2ee506cadf29cfec8ff000000000000002ac3b482890a5cc9fc06162d532091c4ac5d87f557123adb4c56088644116437fe"
	if encoded := hex.EncodeToString(frame); encoded != wantFrameHex {
		t.Fatalf("bootstrap frame = %q", encoded)
	}
}

func TestResolvePublicProfile(t *testing.T) {
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name() != Deterministic {
		t.Fatalf("profile name = %q", profile.Name())
	}
	if len(profile.Inventory()) == 0 || profile.InventorySHA256() == "" || profile.ImplementationSHA256() == "" {
		t.Fatalf("profile identity is incomplete: %#v", profile)
	}
	if _, err := Resolve("unknown/v1"); err == nil || !strings.Contains(err.Error(), "unknown I/O profile") {
		t.Fatalf("Resolve(unknown) error = %v", err)
	}
}

func TestResolveReturnsAnImmutableProfileSpecification(t *testing.T) {
	first, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	inventory := first.Inventory()
	inventory[0] ^= 1
	second, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
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

func TestDeterministicProfileAcceptsArbitraryTargetArguments(t *testing.T) {
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	argument := "-test.run=^TestUnrelatedSuite$"
	err = profile.ValidatePreparedTarget(target.Spec{Kind: target.KindGoTest, Source: "./tests", Args: []string{argument}}, target.Prepared{
		Kind: target.KindGoTest, Source: "./tests", Argv: []string{"gomadv3-target", argument}, BuildTags: []string{"test_dep"},
		BuildInfo: record.BuildInfo{Path: "go.temporal.io/server/tests.test"}, GoVersion: "go1.26.4", TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidatePreparedTargetRejectsIdentityMismatch(t *testing.T) {
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	validSpec := target.Spec{
		Kind: target.KindGoTest, Source: "./tests", Args: []string{"-test.run=^TestActivityAPIBatchCancelClientTestSuite$"},
	}
	validPrepared := target.Prepared{
		Kind: target.KindGoTest, Source: "./tests", Argv: []string{"gomadv3-target", "-test.run=^TestActivityAPIBatchCancelClientTestSuite$"},
		BuildTags: []string{"test_dep"}, BuildInfo: record.BuildInfo{Path: "go.temporal.io/server/tests.test"},
		GoVersion: "go1.26.4", TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}
	tests := map[string]struct {
		spec        target.Spec
		prepared    target.Prepared
		environment []string
	}{
		"kind":     {spec: withKind(validSpec, target.KindGoRun), prepared: validPrepared},
		"source":   {spec: withSource(validSpec, "go.temporal.io/server/tests"), prepared: validPrepared},
		"argument": {spec: withArgs(validSpec, "-test.run=^TestActivityAPIBatchCancelClientTestSuite$", "-test.count=1"), prepared: validPrepared},
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
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	prepared := target.Prepared{
		SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Argv:   []string{"gomadv3-target", "-test.run=^TestActivityAPIBatchCancelClientTestSuite$"},
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
