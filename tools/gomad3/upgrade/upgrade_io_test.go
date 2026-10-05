package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

type failingDossierWriter struct {
	failAt int
	writes int
	err    error
}

func (writer *failingDossierWriter) Write(contents []byte) (int, error) {
	writer.writes++
	if writer.writes == writer.failAt {
		return 0, writer.err
	}
	return len(contents), nil
}

func TestRunRenderingFailurePreservesGateEvidence(t *testing.T) {
	for _, scenario := range []string{"qualified", "gate", "host", "publication", "corpus", "baseline", "boundary"} {
		t.Run(scenario, func(t *testing.T) {
			root := writeUpgradeFixture(t, false)
			if scenario == "host" || scenario == "gate" || scenario == "publication" {
				platform := "darwin/arm64"
				if runtime.GOOS+"/"+runtime.GOARCH == platform {
					platform = "linux/amd64"
				}
				root = writeUpgradeFixtureForPlatforms(t, false, []string{platform})
			}
			baseline, err := os.ReadFile(filepath.Join(root, "deterministicio/boundary/manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			spec := Spec{Root: root, Output: filepath.Join(root, "dossier.json"), BaselineManifest: baseline, CorpusReport: writeQualifiedCorpus(t, root, "gomad3-core"), Gates: []Gate{
				{Name: "first", Command: []string{"/usr/bin/printf", "first output"}},
				{Name: "later", Command: []string{"/usr/bin/printf", "later output"}},
			}}
			switch scenario {
			case "gate", "publication":
				spec.Gates[0].Command = []string{"/bin/sh", "-c", "printf first-output; exit 23"}
			case "corpus":
				spec.CorpusReport = ""
			case "baseline":
				spec.BaselineManifest = nil
			case "boundary":
				spec.BaselineManifest = []byte(`{"manifest_version":"old","intercepts":[]}`)
			}
			if scenario == "publication" {
				if err := os.WriteFile(spec.Output, []byte("prior"), 0o600); err != nil {
					t.Fatal(err)
				}
				spec.Output = filepath.Join(spec.Output, "blocked.json")
			}
			primary := Run(context.Background(), spec)
			wantPrimary := map[string]string{
				"gate": "qualification gate first:", "host": "qualification host ", "publication": "publish upgrade dossier:",
				"corpus": "a checked core qualification corpus is required", "baseline": "a baseline boundary manifest is required",
				"boundary": "boundary changes require explicit approval",
			}[scenario]
			if (wantPrimary == "" && primary != nil) || (wantPrimary != "" && (primary == nil || !strings.HasPrefix(primary.Error(), wantPrimary))) {
				t.Fatalf("control primary = %v, want %q", primary, wantPrimary)
			}
			var control []byte
			if scenario != "publication" {
				control, err = os.ReadFile(spec.Output)
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, failAt := range []int{1, 2} {
				t.Run(fmt.Sprintf("write%d", failAt), func(t *testing.T) {
					renderErr := errors.New("render failure")
					writer := &failingDossierWriter{failAt: failAt, err: renderErr}
					spec.Writer = writer
					err := Run(context.Background(), spec)
					if !errors.Is(err, renderErr) {
						t.Fatalf("Run error = %v, missing rendering failure", err)
					}
					if primary != nil {
						joined, ok := err.(interface{ Unwrap() []error })
						if !ok || joined.Unwrap()[0].Error() != primary.Error() {
							t.Fatalf("primary error changed: %v, want %v first", err, primary)
						}
					}
					if scenario == "publication" {
						return
					}
					actual, readErr := os.ReadFile(spec.Output)
					if readErr != nil || !bytes.Equal(actual, control) {
						t.Fatalf("rendering changed published dossier: %s, %v", actual, readErr)
					}
					if scenario == "qualified" {
						var dossier Dossier
						if err := json.Unmarshal(actual, &dossier); err != nil {
							t.Fatal(err)
						}
						if !dossier.Qualified || len(dossier.Gates) != 2 || dossier.Gates[1].Status != "passed" || dossier.Gates[1].Output != "later output" || writer.writes != 4 {
							t.Fatalf("later gate evidence = %+v, writes=%d", dossier.Gates, writer.writes)
						}
					}
				})
			}
		})
	}
}

func TestOverlayArchiveCleanupFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		archive []byte
		primary bool
	}{
		{name: "read and close fail", archive: []byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 0, 7}, primary: true},
		{name: "tar ends before stored deflate error", archive: append(append([]byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0xff, 0xfb}, make([]byte, 1024)...), 7)},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := writeUpgradeFixture(t, false)
			descriptor := replaceFixtureArchive(t, root, test.archive)
			result, err := inspectOverlayCollision(root, descriptor)
			var corrupt flate.CorruptInputError
			if !errors.As(err, &corrupt) || result.Checked || result.ArchiveSHA256 != "" || result.OverlayPaths != nil || result.Collisions != nil {
				t.Fatalf("collision evidence = %+v, error=%v", result, err)
			}
			if test.primary {
				joined, ok := err.(interface{ Unwrap() []error })
				if !ok || len(joined.Unwrap()) != 2 || !strings.HasPrefix(joined.Unwrap()[0].Error(), "read qualified Go archive:") || !errors.As(joined.Unwrap()[1], &corrupt) {
					t.Fatalf("read/close errors = %v", err)
				}
			} else if _, ok := err.(flate.CorruptInputError); !ok {
				t.Fatalf("sole close error wrapped: %T %v", err, err)
			}
		})
	}
}

func TestOverlayArchiveNilCleanupPreservesReadError(t *testing.T) {
	var compressed bytes.Buffer
	zipper := gzip.NewWriter(&compressed)
	invalidHeader := make([]byte, 512)
	invalidHeader[0] = 1
	if _, err := zipper.Write(invalidHeader); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	root := writeUpgradeFixture(t, false)
	descriptor := replaceFixtureArchive(t, root, compressed.Bytes())
	result, err := inspectOverlayCollision(root, descriptor)
	if result.Checked || !errors.Is(err, tar.ErrHeader) {
		t.Fatalf("invalid tar result = %+v, %v", result, err)
	}
	wrapped, ok := err.(interface{ Unwrap() error })
	if !ok || wrapped.Unwrap() != tar.ErrHeader {
		t.Fatalf("nil cleanup changed primary error identity: %T %v", err, err)
	}
}

func TestLegacyRegularOverlayArchive(t *testing.T) {
	for _, test := range []struct {
		name      string
		collision bool
	}{
		{name: "go/src/os/gomad.go", collision: true},
		{name: "go/src/runtime/proc.go"},
		{name: "go/src/os/gomad.go/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var raw bytes.Buffer
			archive := tar.NewWriter(&raw)
			typeflag := byte(tar.TypeReg)
			if strings.HasSuffix(test.name, "/") {
				typeflag = tar.TypeDir
			}
			if err := archive.WriteHeader(&tar.Header{Name: test.name, Mode: 0o644, Typeflag: typeflag}); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			header := raw.Bytes()[:512]
			header[156] = 0
			for index := 148; index < 156; index++ {
				header[index] = ' '
			}
			checksum := 0
			for _, value := range header {
				checksum += int(value)
			}
			copy(header[148:156], fmt.Sprintf("%06o\x00 ", checksum))
			if raw.Bytes()[156] != 0 {
				t.Fatal("legacy type flag was normalized before gzip")
			}
			var compressed bytes.Buffer
			zipper := gzip.NewWriter(&compressed)
			if _, err := zipper.Write(raw.Bytes()); err != nil {
				t.Fatal(err)
			}
			if err := zipper.Close(); err != nil {
				t.Fatal(err)
			}
			root := writeUpgradeFixture(t, false)
			descriptor := replaceFixtureArchive(t, root, compressed.Bytes())
			result, err := inspectOverlayCollision(root, descriptor)
			if err != nil || !result.Checked || (len(result.Collisions) != 0) != test.collision {
				t.Fatalf("legacy regular collision = %+v, %v", result, err)
			}
			if test.collision {
				if err := Run(context.Background(), Spec{Root: root, Output: filepath.Join(root, "dossier.json")}); err == nil || !strings.Contains(err.Error(), "overlay collides") {
					t.Fatalf("collision refusal = %v", err)
				}
			}
		})
	}
}

func replaceFixtureArchive(t *testing.T, root string, contents []byte) gomadversion.Descriptor {
	t.Helper()
	descriptor, err := gomadversion.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	descriptor.Archive.SHA256 = strings.TrimPrefix(digest(contents), "sha256:")
	if err := os.WriteFile(filepath.Join(root, ".toolchain/downloads", descriptor.Archive.Name), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "toolchain/version/version.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return descriptor
}
