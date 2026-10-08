package deterministicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestPortableAdapterConfigurationRefusals(t *testing.T) {
	for _, adapter := range []struct{ name, module, version, otherVersion string }{
		{name: "sentry", module: sentryModulePath, version: sentryVersion, otherVersion: "v0.45.0"},
		{name: "hashicorp-metrics", module: hashicorpMetricsModulePath, version: hashicorpMetricsVersion, otherVersion: "v0.5.3"},
		{name: "grpc", module: grpcModulePath, version: grpcVersion, otherVersion: "v1.80.1"},
	} {
		for _, test := range []struct {
			name         string
			version      string
			replace      string
			sum          string
			buildModFile bool
			want         string
		}{
			{name: "version", version: adapter.otherVersion, want: "unsupported " + adapter.module + " version"},
			{name: "replacement", replace: "replace " + adapter.module + " => ./adapted\n", want: "already replaces " + adapter.module},
			{name: "version-replacement", replace: "replace " + adapter.module + " " + adapter.version + " => ./adapted\n", want: "already replaces " + adapter.module},
			{name: "replacement-block", replace: "replace (\n\t" + adapter.module + " => ./adapted\n)\n", want: "already replaces " + adapter.module},
			{name: "sum", sum: "h1:changed", want: "module sum"},
			{name: "missing-sum", want: "module sum"},
			{name: "build-modfile", buildModFile: true, want: "existing build modfile"},
		} {
			t.Run(adapter.name+"/"+test.name, func(t *testing.T) {
				workingDirectory := t.TempDir()
				version := test.version
				if version == "" {
					version = adapter.version
				}
				modFile := "module example.test\n\ngo 1.27.1\n\nrequire " + adapter.module + " " + version + "\n" + test.replace
				if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte(modFile), 0o600); err != nil {
					t.Fatal(err)
				}
				if test.sum != "" {
					if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), []byte(adapter.module+" "+version+" "+test.sum+"\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				spec := target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir()}
				if test.buildModFile {
					spec.BuildModFile = filepath.Join(workingDirectory, "existing.mod")
				}
				if _, _, err := deterministicAdapters.prepare(spec, t.TempDir()); !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("prepare configuration: %v, want %s", err, test.want)
				}
			})
		}
	}
}

func TestPortableLibcSelectionConfiguration(t *testing.T) {
	for _, test := range []struct{ name, moduleFile, want string }{
		{"unsupported", "module example.test\n\ngo 1.26.4\n\nrequire modernc.org/libc v1.72.2\n", "unsupported modernc.org/libc version"},
		{"absent", "module example.test\n\ngo 1.26.4\n", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			working := t.TempDir()
			if err := os.WriteFile(filepath.Join(working, "go.mod"), []byte(test.moduleFile), 0o600); err != nil {
				t.Fatal(err)
			}
			spec, adapters, err := deterministicAdapters.prepare(target.Spec{WorkingDir: working, PreparationRoot: t.TempDir()}, "")
			if test.want != "" {
				if !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("prepare() = %v", err)
				}
			} else if err != nil || spec.BuildModFile != "" || adapters == nil || len(adapters) != 0 {
				t.Fatalf("empty selection = %#v, %#v, %v", spec, adapters, err)
			}
		})
	}
}
