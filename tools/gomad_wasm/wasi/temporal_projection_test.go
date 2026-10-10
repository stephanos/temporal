package wasi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
)

func TestTemporalPrivateModule(t *testing.T) {
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "sub"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "sub", "source.go"), []byte("package fixture\n"), 0440); err != nil {
		t.Fatal(err)
	}
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: source, Target: "/module"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "module")
	if err := temporalMaterializeModule(destination, captured); err != nil {
		t.Fatal(err)
	}
	copy, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: destination, Target: "/module"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(captured.Descriptor, copy.Descriptor) {
		t.Fatal("private module changed complete bytes or modes")
	}
	if err := temporalMaterializeModule(destination, captured); err == nil {
		t.Fatal("materializer must reject an existing root")
	}
	captured.Descriptor = append(captured.Descriptor, '\n')
	if err := temporalMaterializeModule(filepath.Join(t.TempDir(), "module"), captured); err == nil {
		t.Fatal("materializer accepted unvalidated snapshot")
	}
}

func TestTemporalFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "sqlite_contention")
	original, err := os.ReadFile(filepath.Join(source, "contention_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if temporalHash(original) != "a9bfaf87df31ad4eb18abfccb4920ae0911c77851f34be02ab7f2ea4683a52ae" {
		t.Fatal("original exclusive fixture bytes changed")
	}
	destinationAnchor := t.TempDir()
	destination := filepath.Join(destinationAnchor, "parent", "fixture")
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"contention_test.go", "controls_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatal("staged fixture body changed")
		}
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, "contention_test.go")
	if err := os.WriteFile(target, append(bytes.Clone(original), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err == nil {
		t.Fatal("staging accepted changed fixture hash")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(original, data) {
		t.Fatal("staging silently replaced changed source")
	}
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "extra_test.go"), []byte("package contention\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err == nil {
		t.Fatal("staging accepted an extra test body")
	}
}

func TestTemporalGRPCDiagnosticFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "environment")
	destination := t.TempDir()
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destination, "grpc-progress"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", "grpc_progress_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "main.go" && temporalHash(expected) != "fde93d6955cafae1d1696d405eaa46987534c05cfef5dc2fe2a14483bacc98f3" {
			t.Fatal("original environment fixture bytes changed")
		}
		actual, err := os.ReadFile(filepath.Join(destination, "grpc-progress", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("staged gRPC diagnostic changed %s", name)
		}
	}
}

func TestTemporalNamespaceStackFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "frontend_namespace")
	destination := t.TempDir()
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destination, "namespace-stack"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"namespace_test.go", "namespace_stack_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(destination, "namespace-stack", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("staged namespace stack diagnostic changed %s", name)
		}
	}
}

func TestTemporalFixtureStagingRejectsLinks(t *testing.T) {
	for _, name := range []string{"source-root", "source-parent", "destination-root", "destination-parent"} {
		t.Run(name, func(t *testing.T) {
			anchor, outside := t.TempDir(), t.TempDir()
			source, destination := filepath.Join(anchor, "source"), filepath.Join(anchor, "destination")
			owner := filepath.Join("..", "testdata", "sqlite_contention")
			if err := temporalStageFixture(filepath.Dir(owner), filepath.Base(owner), anchor, "source"); err != nil {
				t.Fatal(err)
			}
			outsideFixture := filepath.Join(outside, "fixture")
			if err := temporalStageFixture(filepath.Dir(owner), filepath.Base(owner), outside, "fixture"); err != nil {
				t.Fatal(err)
			}
			before, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: outside, Target: "/outside"}}, readonlymount.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "source-root":
				source = filepath.Join(anchor, "source-link")
				if err := os.Symlink(outsideFixture, source); err != nil {
					t.Fatal(err)
				}
			case "source-parent":
				parent := filepath.Join(anchor, "source-parent")
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				source = filepath.Join(parent, "fixture")
			case "destination-root":
				if err := os.Symlink(outsideFixture, destination); err != nil {
					t.Fatal(err)
				}
			case "destination-parent":
				parent := filepath.Join(anchor, "destination-parent")
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				destination = filepath.Join(parent, "new-fixture")
			}
			sourceRelative, err := filepath.Rel(anchor, source)
			if err != nil {
				t.Fatal(err)
			}
			destinationRelative, err := filepath.Rel(anchor, destination)
			if err != nil {
				t.Fatal(err)
			}
			stageErr := temporalStageFixture(anchor, sourceRelative, anchor, destinationRelative)
			after, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: outside, Target: "/outside"}}, readonlymount.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before.Descriptor, after.Descriptor) {
				t.Error("staging changed bytes or entries outside the trusted anchor")
			}
			if stageErr == nil {
				t.Error("staging accepted a linked fixture path")
			}
		})
	}
}

func TestTemporalFixtureStagingPolicy(t *testing.T) {
	owner := filepath.Join("..", "testdata")
	for _, relative := range []string{"../escape", "/absolute", "."} {
		t.Run(relative, func(t *testing.T) {
			anchor := t.TempDir()
			if err := temporalStageFixture(owner, "sqlite_contention", anchor, relative); err == nil {
				t.Fatal("accepted invalid destination")
			}
			if err := temporalStageFixture(owner, relative, anchor, "fixture"); err == nil {
				t.Fatal("accepted invalid source")
			}
			entries, err := os.ReadDir(anchor)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatal("invalid path admission changed the destination anchor")
			}
		})
	}
	anchor := t.TempDir()
	if err := os.MkdirAll(filepath.Join(anchor, "source", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(anchor, "source", anchor, "destination"); err == nil {
		t.Fatal("accepted a nonflat fixture owner")
	}
	if _, err := os.Lstat(filepath.Join(anchor, "destination")); !os.IsNotExist(err) {
		t.Fatal("rejected source created a destination")
	}
}

func TestTemporalModuleGraph(t *testing.T) {
	original := []byte(`{"Path":"main","Main":true,"GoMod":"/root/go.mod"}
{"Path":"github.com/ncruces/go-sqlite3","Version":"v0.35.6","Dir":"/cache/sqlite","GoMod":"/cache/sqlite/go.mod","Sum":"pinned","GoModSum":"pinned-mod"}
{"Path":"unrelated","Version":"v1.2.3","Sum":"unchanged"}`)
	effective := []byte(`{"Path":"main","Main":true,"GoMod":"/stage/temporal.mod"}
{"Path":"github.com/ncruces/go-sqlite3","Version":"v0.35.6","Dir":"/private/sqlite","GoMod":"/private/sqlite/go.mod","Replace":{"Path":"/private/sqlite","Dir":"/private/sqlite","GoMod":"/private/sqlite/go.mod"}}
{"Path":"unrelated","Version":"v1.2.3","Sum":"unchanged"}`)
	for _, test := range []struct {
		name                string
		original, effective []byte
		valid               bool
	}{
		{"owned", original, effective, true},
		{"extra-module", original, append(bytes.Clone(effective), []byte(`{"Path":"extra","Version":"v1"}`)...), false},
		{"version", original, bytes.Replace(effective, []byte("v1.2.3"), []byte("v1.2.4"), 1), false},
		{"preexisting", bytes.Replace(original, []byte(`"Sum":"pinned"`), []byte(`"Sum":"pinned","Replace":{"Path":"/user"}`), 1), effective, false},
		{"wrong-copy", original, bytes.ReplaceAll(effective, []byte("/private/sqlite"), []byte("/elsewhere")), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := temporalCompareModuleGraphs(test.original, test.effective, "/cache/sqlite", "/private/sqlite", "/stage/temporal.mod"); (err == nil) != test.valid {
				t.Fatalf("expected admission %v: %v", test.valid, err)
			}
		})
	}
}

func temporalModuleSnapshot(captured readonlymount.CapturedInputs) (readonlymount.Snapshot, error) {
	mappings, _, snapshot, err := readonlymount.DecodeCapturedInputs(captured.Manifest, captured.Descriptor, func(name string, size uint64) ([]byte, error) {
		data, found := captured.Payloads[name]
		if !found || uint64(len(data)) != size {
			return nil, fmt.Errorf("missing private module payload %s", name)
		}
		return data, nil
	})
	if err != nil {
		return readonlymount.Snapshot{}, err
	}
	if len(mappings) != 1 || mappings[0].Target != "/module" || len(snapshot.NotExist) != 0 {
		return readonlymount.Snapshot{}, fmt.Errorf("private module requires one complete /module snapshot")
	}
	return snapshot, nil
}

func temporalMaterializeModule(destination string, captured readonlymount.CapturedInputs) (retErr error) {
	snapshot, err := temporalModuleSnapshot(captured)
	if err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	return temporalWriteModule(root, snapshot)
}

func temporalWriteModule(root *os.Root, snapshot readonlymount.Snapshot) error {
	for _, entry := range snapshot.Entries {
		relative, err := filepath.Rel("/module", entry.Path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, "../") || filepath.IsAbs(relative) {
			return fmt.Errorf("invalid private module entry %s", entry.Path)
		}
		if entry.Kind == readonlymount.KindDirectory {
			if relative != "." {
				if err := root.Mkdir(relative, 0700); err != nil {
					return err
				}
			}
			continue
		}
		file, err := root.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(entry.Data)
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return err
		}
		if err := root.Chmod(relative, entry.Mode); err != nil {
			return err
		}
	}
	for _, entry := range slices.Backward(snapshot.Entries) {
		if entry.Kind == readonlymount.KindDirectory {
			relative, err := filepath.Rel("/module", entry.Path)
			if err != nil {
				return err
			}
			if err := root.Chmod(relative, entry.Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

func temporalStageFixture(sourceAnchor, source, destinationAnchor, destination string) (retErr error) {
	if !filepath.IsLocal(source) || !filepath.IsLocal(destination) || filepath.Clean(source) == "." || filepath.Clean(destination) == "." {
		return fmt.Errorf("fixture paths must remain beneath their trusted anchors")
	}
	limits := readonlymount.DefaultLimits()
	limits.Files, limits.Requests, limits.DirectoryEntries = 16, 64, 16
	limits.SingleFileBytes, limits.TotalBytes = 64<<10, 128<<10
	capture := func(anchor, relative string) (_ readonlymount.CapturedInputs, captureErr error) {
		mapping := readonlymount.Mapping{Source: anchor, Target: "/module"}
		broker, err := readonlymount.Prepare([]readonlymount.Mapping{mapping}, limits)
		if err != nil {
			return readonlymount.CapturedInputs{}, err
		}
		defer func() { captureErr = errors.Join(captureErr, broker.Close()) }()
		prefix := "/module/" + filepath.ToSlash(filepath.Clean(relative))
		entry, err := broker.Lookup(prefix)
		if err != nil {
			return readonlymount.CapturedInputs{}, err
		}
		if entry.Kind != readonlymount.KindDirectory {
			return readonlymount.CapturedInputs{}, fmt.Errorf("fixture owner must be a directory")
		}
		for _, child := range entry.Children {
			if child.Kind != readonlymount.KindFile {
				return readonlymount.CapturedInputs{}, fmt.Errorf("fixture owner must contain only flat regular files")
			}
			if _, err := broker.Lookup(prefix + "/" + child.Name); err != nil {
				return readonlymount.CapturedInputs{}, err
			}
		}
		snapshot := broker.Captured()
		for index := range snapshot.Entries {
			snapshot.Entries[index].Path = "/module" + strings.TrimPrefix(snapshot.Entries[index].Path, prefix)
		}
		return readonlymount.EncodeCapturedInputs([]readonlymount.Mapping{mapping}, limits, snapshot)
	}
	captured, err := capture(sourceAnchor, source)
	if err != nil {
		return err
	}
	snapshot, err := temporalModuleSnapshot(captured)
	if err != nil {
		return err
	}
	anchor, err := os.OpenRoot(destinationAnchor)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, anchor.Close()) }()
	parent := anchor
	components := strings.Split(filepath.Clean(destination), string(filepath.Separator))
	for _, component := range components[:len(components)-1] {
		info, err := parent.Lstat(component)
		if os.IsNotExist(err) {
			if err := parent.Mkdir(component, 0700); err != nil {
				return err
			}
			info, err = parent.Lstat(component)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture destination parent is not a regular directory")
		}
		next, err := parent.OpenRoot(component)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, next.Close()) }()
		parent = next
	}
	name := components[len(components)-1]
	info, err := parent.Lstat(name)
	created := os.IsNotExist(err)
	if created {
		if err := parent.Mkdir(name, 0700); err != nil {
			return err
		}
		info, err = parent.Lstat(name)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fixture destination is not a regular directory")
	}
	if created {
		root, err := parent.OpenRoot(name)
		if err != nil {
			return err
		}
		writeErr := temporalWriteModule(root, snapshot)
		if err := errors.Join(writeErr, root.Close()); err != nil {
			return err
		}
	}
	staged, err := capture(destinationAnchor, destination)
	if err != nil {
		return err
	}
	if !bytes.Equal(captured.Descriptor, staged.Descriptor) {
		return fmt.Errorf("staged diagnostic fixture differs from its source owner: %s", destination)
	}
	return nil
}

func temporalCompareModuleGraphs(original, effective []byte, originalRoot, privateRoot, modfile string) error {
	decode := func(data []byte) (map[string]map[string]any, error) {
		modules := map[string]map[string]any{}
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var module map[string]any
			if err := decoder.Decode(&module); err == io.EOF {
				return modules, nil
			} else if err != nil {
				return nil, err
			}
			name, valid := module["Path"].(string)
			if !valid || modules[name] != nil {
				return nil, fmt.Errorf("invalid or duplicate module graph identity")
			}
			modules[name] = module
		}
	}
	before, err := decode(original)
	if err != nil {
		return err
	}
	after, err := decode(effective)
	if err != nil {
		return err
	}
	if len(before) != len(after) {
		return fmt.Errorf("owned projection changed module graph size")
	}
	for name, module := range before {
		projected := after[name]
		if projected == nil {
			return fmt.Errorf("owned projection removed module %s", name)
		}
		if module["Main"] == true {
			if projected["GoMod"] != modfile {
				return fmt.Errorf("owned projection has wrong main modfile")
			}
			projected["GoMod"] = module["GoMod"]
		}
		if name == "github.com/ncruces/go-sqlite3" {
			if module["Replace"] != nil || module["Version"] != "v0.35.6" || module["Dir"] != originalRoot || !filepath.IsAbs(fmt.Sprint(module["GoMod"])) {
				return fmt.Errorf("owned projection requires original unreplaced pinned SQLite graph")
			}
			replacement, valid := projected["Replace"].(map[string]any)
			expected := map[string]any{"Path": privateRoot, "Dir": privateRoot, "GoMod": filepath.Join(privateRoot, "go.mod")}
			if module["GoVersion"] != nil {
				expected["GoVersion"] = module["GoVersion"]
			}
			if !valid || !reflect.DeepEqual(replacement, expected) || projected["Dir"] != privateRoot || projected["GoMod"] != filepath.Join(privateRoot, "go.mod") {
				return fmt.Errorf("SQLite replacement is not the exact owned projection")
			}
			delete(projected, "Replace")
			for _, key := range []string{"Dir", "GoMod", "Sum", "GoModSum", "Time"} {
				if key == "Sum" || key == "GoModSum" || key == "Time" {
					if projected[key] != nil {
						return fmt.Errorf("unexpected projected SQLite checksum metadata")
					}
				}
				if module[key] != nil {
					projected[key] = module[key]
				} else {
					delete(projected, key)
				}
			}
		}
		if !reflect.DeepEqual(module, projected) {
			return fmt.Errorf("owned projection changed unrelated graph metadata for %s: original=%v effective=%v", name, module, projected)
		}
	}
	if before["github.com/ncruces/go-sqlite3"] == nil {
		return fmt.Errorf("owned projection missing SQLite module")
	}
	return nil
}
