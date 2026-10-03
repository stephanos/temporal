package upgrade

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
)

const pinFixtureMod = "module example.com/pins\n\ngo 1.27.1\n\nrequire (\n github.com/getsentry/sentry-go v0.46.0\n github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect\n)\n"
const pinFixtureSum = "github.com/getsentry/sentry-go v0.46.0 h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0=\ngithub.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=\n"

func writePinFixture(t *testing.T, module, sums string) string {
	t.Helper()
	directory := t.TempDir()
	for name, contents := range map[string]string{"go.mod": module, "go.sum": sums} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(directory, "go.mod")
}

func changedPins(report PinImpact) []string {
	result := []string{}
	for _, pin := range report.Pins {
		if pin.Status == "invalidated" || pin.Status == "unknown" {
			result = append(result, pin.Class+":"+pin.ID+":"+pin.Status+":"+pin.Reason)
		}
	}
	return result
}

func TestPinImpactMatchesAdapterAndPackRejection(t *testing.T) {
	baseline := writePinFixture(t, pinFixtureMod, pinFixtureSum)
	candidate := writePinFixture(t, strings.ReplaceAll(strings.ReplaceAll(pinFixtureMod, "v0.46.0", "v0.47.0"), "v1.0.3-0.20250322232337-35a7c28c31ee", "v1.0.4"), pinFixtureSum)
	report, err := ReadPinImpact("..", candidate, baseline)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"adapter:github.com/getsentry/sentry-go:invalidated:version_changed", "pack_rule:reflect2-go126:github.com/modern-go/reflect2:invalidated:activation_version_changed"}
	if got := changedPins(report); !reflect.DeepEqual(want, got) {
		t.Fatalf("want %v got %v", want, got)
	}
	_, _, err = deterministicio.Default().PrepareBuildAdapters(target.Spec{Kind: target.KindGoTest, Source: ".", WorkingDir: filepath.Dir(candidate), PreparationRoot: t.TempDir()}, t.TempDir())
	if !deterministicio.IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "unsupported github.com/getsentry/sentry-go version") {
		t.Fatalf("build did not reject same adapter pin: %v", err)
	}
	contents, err := os.ReadFile("../internal/compatibilitypack/packs/reflect2-go126.json")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := compatibility.LoadPack(contents)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := compatibility.PinEvidence()
	if err != nil {
		t.Fatal(err)
	}
	var rule compatibility.PackageRuleEvidence
	for _, entry := range evidence {
		if entry.ID == "reflect2-go126" {
			rule = entry.Rules[0]
		}
	}
	pkg := compatibility.Package{ImportPath: rule.ImportPath, Module: compatibility.Module{Path: rule.Module.Path, Version: rule.Module.Version, Sum: rule.Module.Sum}, SourceSetSHA256: rule.SourceSetSHA256}
	for _, source := range rule.GoSources {
		pkg.GoSources = append(pkg.GoSources, compatibility.Source{Name: source.Name, SHA256: source.SHA256})
	}
	for _, source := range rule.ForeignSources {
		pkg.ForeignSources = append(pkg.ForeignSources, compatibility.ForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256})
	}
	selected, err := compatibility.SelectPacksForPlatform([]compatibility.ValidatedPack{pack}, []compatibility.Package{pkg}, "darwin/arm64")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Identities()) != 1 {
		t.Fatal("positive pack control was not selected")
	}
	pkg.Module.Version = "v1.0.4"
	selected, err = compatibility.SelectPacksForPlatform([]compatibility.ValidatedPack{pack}, []compatibility.Package{pkg}, "darwin/arm64")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Identities()) != 0 {
		t.Fatal("build policy accepted the invalidated pack")
	}
}

func TestPinImpactModuleChanges(t *testing.T) {
	baseline := writePinFixture(t, pinFixtureMod, pinFixtureSum)
	for _, test := range []struct {
		name, module, sums string
		want               []string
	}{
		{"unchanged", pinFixtureMod, pinFixtureSum, []string{}},
		{"same-version sum", pinFixtureMod, strings.Replace(pinFixtureSum, "h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0=", "h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=", 1), []string{"adapter:github.com/getsentry/sentry-go:invalidated:sum_changed"}},
		{"removed", strings.Replace(pinFixtureMod, " github.com/getsentry/sentry-go v0.46.0\n", "", 1), pinFixtureSum, []string{"adapter:github.com/getsentry/sentry-go:invalidated:module_removed"}},
		{"replacement", pinFixtureMod + "replace github.com/getsentry/sentry-go => ./local\n", pinFixtureSum, []string{"adapter:github.com/getsentry/sentry-go:invalidated:module_replaced"}},
		{"inactive replacement", pinFixtureMod + "replace github.com/getsentry/sentry-go v0.45.0 => ./local\n", pinFixtureSum, []string{"adapter:github.com/getsentry/sentry-go:invalidated:module_replaced"}},
		{"indirect bump", strings.Replace(pinFixtureMod, "v1.0.3-0.20250322232337-35a7c28c31ee", "v1.0.4", 1), pinFixtureSum, []string{"pack_rule:reflect2-go126:github.com/modern-go/reflect2:invalidated:activation_version_changed"}},
		{"unknown", pinFixtureMod, strings.Split(pinFixtureSum, "\n")[1] + "\n", []string{"adapter:github.com/getsentry/sentry-go:unknown:module_sum_missing"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := writePinFixture(t, test.module, test.sums)
			report, err := ReadPinImpact("..", candidate, baseline)
			if err != nil {
				t.Fatal(err)
			}
			if got := changedPins(report); !reflect.DeepEqual(test.want, got) {
				t.Fatalf("want %v got %v", test.want, got)
			}
			encoded, err := report.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			repeated, err := ReadPinImpact("..", candidate, baseline)
			if err != nil {
				t.Fatal(err)
			}
			encodedAgain, err := repeated.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, encodedAgain) || bytes.Contains(encoded, []byte(filepath.Dir(candidate))) {
				t.Fatal("report is not stable and path-free")
			}
			contents, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if string(contents) != test.module {
				t.Fatal("module mutated")
			}
			contents, err = os.ReadFile(filepath.Join(filepath.Dir(candidate), "go.sum"))
			if err != nil {
				t.Fatal(err)
			}
			if string(contents) != test.sums {
				t.Fatal("sums mutated")
			}
		})
	}
}

func TestPinImpactRejectsGoAndToolchainChanges(t *testing.T) {
	baseline := writePinFixture(t, pinFixtureMod, pinFixtureSum)
	for _, module := range []string{strings.Replace(pinFixtureMod, "go 1.27.1", "go 1.28.0", 1), pinFixtureMod + "toolchain go1.28.0\n"} {
		candidate := writePinFixture(t, module, pinFixtureSum)
		_, err := ReadPinImpact("..", candidate, baseline)
		if _, ok := err.(*InvalidPinImpactInput); !ok {
			t.Fatalf("expected invalid input: %v", err)
		}
	}
}

func TestPinImpactIncludesOtherPlatformFingerprint(t *testing.T) {
	baseline := writePinFixture(t, pinFixtureMod, pinFixtureSum)
	report, err := ReadPinImpact("..", baseline, baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, pin := range report.Pins {
		if pin.Class == "interception_fingerprint" && pin.ID == "os.Pipe@linux/amd64" {
			if pin.SourceSetSHA256 != "sha256:1197aa5233e66c97086931807857c31b6ca7c590f932204cf822c8df73a0b27d" || !reflect.DeepEqual(pin.Platforms, []string{"linux/amd64"}) {
				t.Fatalf("incorrect platform override: %+v", pin)
			}
			return
		}
	}
	t.Fatal("report omitted the Linux os.Pipe declaration fingerprint")
}

func TestPinImpactSkipsUnselectedPackVariants(t *testing.T) {
	module := "module example.com/pins\n\ngo 1.27.1\nrequire (\n golang.org/x/sys v0.47.0\n modernc.org/libc v1.72.3\n modernc.org/memory v1.11.0\n github.com/mattn/go-isatty v0.0.21\n)\n"
	sums := "golang.org/x/sys v0.47.0 h1:dummy\nmodernc.org/libc v1.72.3 h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=\nmodernc.org/memory v1.11.0 h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=\ngithub.com/mattn/go-isatty v0.0.21 h1:dummy\n"
	baseline := writePinFixture(t, module, sums)
	report, err := ReadPinImpact("..", baseline, baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, pin := range report.Pins {
		if pin.ID == "modernc-libc-xsys-v047:github.com/mattn/go-isatty" {
			if pin.Status != "not_selected" {
				t.Fatalf("unselected pack variant was reported affected: %+v", pin)
			}
		}
	}
}
