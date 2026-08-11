package ioprofile

import (
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestResolvePublicProfile(t *testing.T) {
	profile, err := Resolve(TemporalActivityAPIBatchCancel)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != TemporalActivityAPIBatchCancel {
		t.Fatalf("profile name = %q", profile.Name)
	}
	if len(profile.Inventory) == 0 || profile.InventorySHA256 == "" || profile.ImplementationSHA256 == "" {
		t.Fatalf("profile identity is incomplete: %#v", profile)
	}
	if _, err := Resolve("unknown/v1"); err == nil || !strings.Contains(err.Error(), "unknown I/O profile") {
		t.Fatalf("Resolve(unknown) error = %v", err)
	}
}

func TestValidatePreparedTargetAcceptsExactSuite(t *testing.T) {
	profile, err := Resolve(TemporalActivityAPIBatchCancel)
	if err != nil {
		t.Fatal(err)
	}
	err = profile.ValidatePreparedTarget(target.Spec{
		Kind: target.KindGoTest, Source: "./tests", Args: []string{"-test.run=^TestActivityAPIBatchCancelClientTestSuite$"},
	}, target.Prepared{
		Kind: target.KindGoTest, Source: "./tests", Argv: []string{"gomadv3-target", "-test.run=^TestActivityAPIBatchCancelClientTestSuite$"},
		BuildTags: []string{"test_dep"}, BuildInfo: record.BuildInfo{Path: "go.temporal.io/server/tests.test"},
		GoVersion: "go1.26.4", TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidatePreparedTargetRejectsProfileExpansion(t *testing.T) {
	profile, err := Resolve(TemporalActivityAPIBatchCancel)
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
		"kind":          {spec: withKind(validSpec, target.KindGoRun), prepared: validPrepared},
		"source":        {spec: withSource(validSpec, "go.temporal.io/server/tests"), prepared: validPrepared},
		"argument":      {spec: withArgs(validSpec, "-test.run=^TestActivityAPIBatchCancelClientTestSuite$", "-test.count=1"), prepared: validPrepared},
		"built package": {spec: validSpec, prepared: withBuildPath(validPrepared, "go.temporal.io/server/tests")},
		"build tags":    {spec: validSpec, prepared: withBuildTags(validPrepared, "custom", "test_dep")},
		"platform":      {spec: validSpec, prepared: withPlatform(validPrepared, "linux", "arm64")},
		"environment":   {spec: validSpec, prepared: validPrepared, environment: []string{"MODE=test"}},
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
	profile, err := Resolve(TemporalActivityAPIBatchCancel)
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
	if decoded.Profile != profile.Name || decoded.TargetSHA256 != prepared.SHA256 || decoded.RunnerSHA256 == "" || decoded.ArgvSHA256 == "" || decoded.InventorySHA256 != profile.InventorySHA256 || decoded.ImplementationSHA256 != profile.ImplementationSHA256 || decoded.Seed != 42 {
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

func withBuildPath(prepared target.Prepared, path string) target.Prepared {
	prepared.BuildInfo.Path = path
	return prepared
}

func withBuildTags(prepared target.Prepared, tags ...string) target.Prepared {
	prepared.BuildTags = tags
	return prepared
}

func withPlatform(prepared target.Prepared, goos, goarch string) target.Prepared {
	prepared.TargetGOOS = goos
	prepared.TargetGOARCH = goarch
	return prepared
}
