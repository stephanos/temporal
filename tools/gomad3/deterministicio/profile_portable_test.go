package deterministicio

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestPortableSelectedAdapterIdentities(t *testing.T) {
	selected := []Adapter{
		{Module: "google.golang.org/grpc", Version: "v1.83.2", Sum: "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="},
		{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="},
	}
	if err := deterministicAdapters.verify(selected); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		adapters []Adapter
		want     string
	}{
		{"missing", nil, "selected adapter identities are missing"},
		{"empty", []Adapter{}, ""},
		{"modified version", []Adapter{{Module: selected[0].Module, Version: "v1.80.1", Sum: selected[0].Sum}}, "selected adapter google.golang.org/grpc is unavailable or modified"},
		{"modified sum", []Adapter{{Module: selected[0].Module, Version: selected[0].Version, Sum: "h1:changed"}}, "selected adapter google.golang.org/grpc is unavailable or modified"},
		{"unavailable", []Adapter{{Module: "example.test/unavailable"}}, "selected adapter example.test/unavailable is unavailable or modified"},
		{"duplicate", []Adapter{selected[0], selected[0]}, "selected adapter identities are not sorted and unique"},
		{"unsorted", []Adapter{selected[1], selected[0]}, "selected adapter identities are not sorted and unique"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := deterministicAdapters.verify(test.adapters)
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != test.want {
				t.Fatalf("verify() = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPortableRequirementsProjection(t *testing.T) {
	closure := target.CapabilityClosure{Packages: []target.CapabilityPackage{
		{ImportPath: "example.com/dependency", Name: "dependency", Imports: []string{"path/filepath"}},
		{ImportPath: "example.com/target", Name: "main", Root: true, Imports: []string{"crypto/rand", "net", "os", "time"}},
		{ImportPath: "google.golang.org/grpc/internal", Name: "internal", Module: &target.CapabilityModule{Path: "google.golang.org/grpc", Version: "v1.83.2"}},
		{ImportPath: "modernc.org/libc", Name: "libc", Module: &target.CapabilityModule{Path: "modernc.org/libc", Version: "v1.72.3"}},
	}}
	adapters := []Adapter{
		{Module: "google.golang.org/grpc", Version: "v1.83.2", Sum: "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="},
		{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="},
	}
	if err := deterministicAdapters.verify(adapters); err != nil {
		t.Fatal(err)
	}
	requirements, err := projectAdapterRequirements(closure, adapters)
	requireTestNoError(t, err)
	requireTestEqual(t, []string{"adapter:google.golang.org/grpc", "adapter:modernc.org/libc", "entropy", "filesystem", "loopback_tcp", "time"}, requirementNames(requirements))
	for _, requirement := range requirements {
		if !requirement.Modeled || len(requirement.Packages) == 0 {
			t.Fatalf("requirement = %#v", requirement)
		}
	}
	requireTestEqual(t, []target.CapabilityPackageReference{{ImportPath: "example.com/dependency", Name: "dependency"}, {ImportPath: "example.com/target", Name: "main"}}, requirements[3].Packages)
	if err := deterministicAdapters.verify([]Adapter{{Module: "modernc.org/libc", Version: "changed", Sum: "h1:changed"}}); err == nil || !strings.Contains(err.Error(), "unavailable or modified") {
		t.Fatalf("verify() = %v", err)
	}
	outside := target.CapabilityClosure{Packages: []target.CapabilityPackage{{ImportPath: "example.com/target", Name: "target", Root: true, Imports: []string{"time"}}}}
	requirements, err = projectAdapterRequirements(outside, adapters[1:])
	requireTestNoError(t, err)
	requireTestEqual(t, []string{"time"}, requirementNames(requirements))
	requirements, err = projectAdapterRequirements(target.CapabilityClosure{}, []Adapter{})
	requireTestNoError(t, err)
	if requirements == nil || len(requirements) != 0 {
		t.Fatalf("empty projection shape = %#v", requirements)
	}
}

func TestPortablePreparedTargetShape(t *testing.T) {
	profile := Default()
	spec := target.Spec{Kind: target.KindGoTest, Source: "./pkg", Args: []string{"-test.run=^TestUnrelatedSuite$"}}
	prepared := target.Prepared{Kind: spec.Kind, Source: spec.Source, Argv: []string{"gomad3-target", spec.Args[0]}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, BuildInfo: record.BuildInfo{Path: "example.test/project/pkg.test"}, GoVersion: "go1.27.1", TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH}
	adapters, err := validatePreparedTargetShape(profile.definition, spec, prepared)
	if err != nil || adapters == nil || len(adapters) != 0 {
		t.Fatalf("valid arbitrary argv = %#v, %v", adapters, err)
	}
	identity := "deterministic I/O target identity does not match its build specification"
	arguments := "deterministic I/O target arguments do not match their build specification"
	for _, test := range []struct {
		name     string
		spec     target.Spec
		prepared target.Prepared
		want     string
	}{
		{"kind", withKind(spec, target.KindGoRun), prepared, identity},
		{"source", withSource(spec, "example.test/project/other"), prepared, identity},
		{"argument", withArgs(spec, spec.Args[0], "-test.count=1"), prepared, arguments},
		{"platform", spec, withPlatform(prepared, "darwin", "amd64"), fmt.Sprintf("deterministic I/O requires Go %s on %s/%s; target was built with %s for darwin/amd64", profile.definition.target.GoVersion, runtime.GOOS, runtime.GOARCH, prepared.GoVersion)},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := validatePreparedTargetShape(profile.definition, test.spec, test.prepared)
			if got != nil || err == nil || err.Error() != test.want {
				t.Fatalf("shape() = %#v, %v; want %q", got, err, test.want)
			}
		})
	}
	prepared.Adapters = []record.TargetAdapter{{Module: grpcModulePath, Version: grpcVersion, Sum: grpcSum}}
	adapters, err = validatePreparedTargetShape(profile.definition, spec, prepared)
	if err != nil || !reflect.DeepEqual(adapters, []Adapter{{Module: grpcModulePath, Version: grpcVersion, Sum: grpcSum}}) {
		t.Fatalf("projected adapters = %#v, %v", adapters, err)
	}
}

func TestPortableBootstrapFrameIdentity(t *testing.T) {
	profile := Default()
	prepared := target.Prepared{SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Argv: []string{"gomad3-target", "-test.run=^TestScenario$"}}
	runner := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	frame, err := encodeBootstrapFrame(profile, prepared, runner, 42)
	requireTestNoError(t, err)
	decoded, err := DecodeBootstrapFrame(frame)
	requireTestNoError(t, err)
	argv, err := canonicalJSON(prepared.Argv)
	requireTestNoError(t, err)
	want := Bootstrap{Profile: profile.Name(), InventorySHA256: profile.InventorySHA256(), ImplementationSHA256: profile.ImplementationSHA256(), TargetSHA256: prepared.SHA256, RunnerSHA256: runner, ArgvSHA256: hashBytes(argv), Seed: 42}
	requireTestEqual(t, want, decoded)
	changed := append([]byte(nil), frame...)
	changed[len(changed)-1] ^= 1
	if _, err := DecodeBootstrapFrame(changed); err == nil {
		t.Fatal("DecodeBootstrapFrame accepted changed frame")
	}
	prepared.SHA256 = "changed"
	if got, err := encodeBootstrapFrame(profile, prepared, runner, 42); got != nil || err == nil || !strings.Contains(err.Error(), "invalid SHA-256 identity \"changed\"") {
		t.Fatalf("invalid target digest = %x, %v", got, err)
	}
}

func TestPortableBootstrapPlatformGoldens(t *testing.T) {
	for _, platform := range []string{"darwin/arm64", "linux/amd64"} {
		t.Run(platform, func(t *testing.T) {
			goos, goarch, _ := strings.Cut(platform, "/")
			profile := mustSpec(profileDefinition{
				name:                  Deterministic,
				target:                TargetContract{GoVersion: generatedBoundaryGoVersion, GOOS: goos, GOARCH: goarch},
				implementationFamily:  "gomad3.deterministic-io/v1",
				implementationVersion: deterministicImplementationVersion,
				adapters:              deterministicAdapters,
			})
			frame, err := encodeBootstrapFrame(profile, target.Prepared{SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Argv: []string{"gomad3-target", "argument"}}, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 42)
			requireTestNoError(t, err)
			requireTestEqual(t, profileGoldens[platform].frameHex, hex.EncodeToString(frame))
		})
	}
}

func TestPortableProfilePublicGuardsRemainFirst(t *testing.T) {
	profile := Default()
	_, guard := profile.validated()
	if guard == nil {
		t.Skip("qualified public host; unsupported-host ordering control does not apply")
	}
	checks := []struct {
		name string
		run  func() error
	}{
		{"adapters", func() error { return profile.VerifyAdapters(nil) }},
		{"requirements", func() error { _, err := profile.Requirements(target.CapabilityClosure{}, nil); return err }},
		{"bootstrap", func() error {
			_, err := profile.BootstrapFrame(target.Prepared{SHA256: "invalid"}, "invalid", 0)
			return err
		}},
		{"prepared", func() error { return profile.ValidatePreparedTarget(target.Spec{}, target.Prepared{}, nil) }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.run(); err == nil || err.Error() != guard.Error() {
				t.Fatalf("public guard = %v; want %v", err, guard)
			}
		})
	}
}
