package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestArtifactReferenceHoldsNoOpenResource(t *testing.T) {
	reference := reflect.TypeFor[Artifact]()
	for index := range reference.NumField() {
		field := reference.Field(index)
		if !field.IsExported() {
			t.Fatalf("Artifact has unexported field %s; a detached reference must not carry hidden state", field.Name)
		}
	}
	if _, closable := any(Artifact{}).(interface{ Close() error }); closable {
		t.Fatal("Artifact has a Close method; only the opened handle owns a resource")
	}
	if _, closable := any(&Artifact{}).(interface{ Close() error }); closable {
		t.Fatal("*Artifact has a Close method; only the opened handle owns a resource")
	}
}

func TestPublishedReferenceMatchesOpenedHandle(t *testing.T) {
	published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if published.TargetSharing != TargetPrivate {
		t.Fatalf("published target sharing = %q, want %q", published.TargetSharing, TargetPrivate)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	if opened.Path() != published.Path || opened.StoredBytes() != published.StoredBytes || !reflect.DeepEqual(opened.Manifest(), published.Manifest) {
		t.Fatalf("opened handle = %s, %d bytes, %#v; want the published reference %#v", opened.Path(), opened.StoredBytes(), opened.Manifest(), published)
	}
	want := published
	want.TargetSharing = ""
	if snapshot := opened.Snapshot(); !reflect.DeepEqual(snapshot, want) {
		t.Fatalf("Snapshot() = %#v, want %#v", snapshot, want)
	}
}

func TestOpenedArtifactPayloadAccess(t *testing.T) {
	published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	stdout, err := opened.ReadPayload("stdout", 64)
	if err != nil || string(stdout) != "stdout" {
		t.Fatalf("ReadPayload(stdout) = %q, %v", stdout, err)
	}
	file, err := opened.OpenPayload("stderr", 64)
	if err != nil {
		t.Fatal(err)
	}
	stderr := make([]byte, 64)
	count, err := file.Read(stderr)
	if closeErr := file.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil || string(stderr[:count]) != "stderr" {
		t.Fatalf("OpenPayload(stderr) read %q, %v", stderr[:count], err)
	}
	copied := filepath.Join(t.TempDir(), "target")
	if err := opened.CopyPayload("target", copied, 0o500); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(copied)
	if err != nil || info.Mode().Perm() != 0o500 {
		t.Fatalf("copied target = %v, %v", info, err)
	}
	if content, err := os.ReadFile(copied); err != nil || string(content) != "target bytes" {
		t.Fatalf("copied target content = %q, %v", content, err)
	}
	sharing, err := opened.TargetSharing()
	if err != nil {
		t.Fatal(err)
	}
	if _, known := linkCount(info); known && sharing != TargetPrivate {
		t.Fatalf("TargetSharing() = %q, want %q", sharing, TargetPrivate)
	}
}

func TestOpenedArtifactCopyDestinationErrorsPreserveHandle(t *testing.T) {
	published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	directory := t.TempDir()
	existing := filepath.Join(directory, "existing")
	mustDo(t, os.WriteFile(existing, []byte("keep destination"), 0o640))
	mustDo(t, os.Chmod(existing, 0o640))
	missingParent := filepath.Join(directory, "missing")
	tests := []struct {
		name        string
		destination string
		want        error
	}{
		{name: "collision", destination: existing, want: os.ErrExist},
		{name: "missing parent", destination: filepath.Join(missingParent, "stdout"), want: os.ErrNotExist},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := opened.CopyPayload("stdout", test.destination, 0o600)
			pathErr, raw := err.(*os.PathError)
			if !raw || pathErr.Op != "open" || pathErr.Path != test.destination || !errors.Is(err, test.want) {
				t.Fatalf("CopyPayload destination error = %T %v, want raw open PathError for %s with %v", err, err, test.destination, test.want)
			}
			if content, err := os.ReadFile(existing); err != nil || string(content) != "keep destination" {
				t.Fatalf("existing destination = %q, %v", content, err)
			}
			if info, err := os.Stat(existing); err != nil || info.Mode().Perm() != 0o640 {
				t.Fatalf("existing destination mode = %v, %v", info, err)
			}
			if _, err := os.Stat(missingParent); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing parent was created: %v", err)
			}
			if stdout, err := opened.ReadPayload("stdout", 64); err != nil || string(stdout) != "stdout" {
				t.Fatalf("ReadPayload after destination error = %q, %v", stdout, err)
			}
			copied := filepath.Join(t.TempDir(), "stdout")
			if err := opened.CopyPayload("stdout", copied, 0o600); err != nil {
				t.Fatal(err)
			}
			if content, err := os.ReadFile(copied); err != nil || string(content) != "stdout" {
				t.Fatalf("copy after destination error = %q, %v", content, err)
			}
			if info, err := os.Stat(copied); err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("copy after destination error mode = %v, %v", info, err)
			}
			if !reflect.DeepEqual(opened.Manifest(), published.Manifest) {
				t.Fatal("destination error changed the opened manifest")
			}
		})
	}
}

func TestClosedArtifactRejectsPayloadAccess(t *testing.T) {
	published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatalf("second Close() = %v, want nil", err)
	}
	for _, handle := range []*Opened{opened, nil} {
		if _, err := handle.ReadPayload("stdout", 64); err == nil || err.Error() != "artifact is not open" {
			t.Fatalf("ReadPayload after close = %v", err)
		}
		if _, err := handle.OpenPayload("stdout", 64); err == nil || err.Error() != "artifact is not open" {
			t.Fatalf("OpenPayload after close = %v", err)
		}
		if err := handle.CopyPayload("target", filepath.Join(t.TempDir(), "target"), 0o500); err == nil || err.Error() != "artifact is not open" {
			t.Fatalf("CopyPayload after close = %v", err)
		}
		if _, err := handle.TargetSharing(); err == nil || err.Error() != "artifact is not open" {
			t.Fatalf("TargetSharing after close = %v", err)
		}
	}
	var unopened *Opened
	if err := unopened.Close(); err != nil {
		t.Fatalf("nil Close() = %v", err)
	}
	if opened.Path() != published.Path || opened.StoredBytes() != published.StoredBytes || opened.Manifest().RecordHash != published.Manifest.RecordHash || opened.Snapshot().Path != published.Path {
		t.Fatal("a closed handle lost its identity")
	}
}

func TestManifestCopiesCannotChangeOpenedHandle(t *testing.T) {
	published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	want := opened.Manifest()
	snapshot := opened.Snapshot()
	copied := opened.Manifest()
	for _, manifest := range []*record.ExecutionRecord{&snapshot.Manifest, &copied} {
		for index := range manifest.Files {
			manifest.Files[index].SHA256 = record.HashBytes([]byte("changed"))
			manifest.Files[index].Size = 1
		}
		manifest.Target.Argv[0] = "changed"
		manifest.Environment[0].Value = "changed"
		*manifest.Outcome.ExitCode = 99
	}
	if got := opened.Manifest(); !reflect.DeepEqual(got, want) {
		t.Fatalf("changing a copy changed the handle's manifest: %#v", got)
	}
	if stdout, err := opened.ReadPayload("stdout", 64); err != nil || string(stdout) != "stdout" {
		t.Fatalf("ReadPayload after changing copies = %q, %v", stdout, err)
	}
}

func TestCloneManifestSharesNoMemory(t *testing.T) {
	var manifest record.ExecutionRecord
	populate(reflect.ValueOf(&manifest).Elem())
	clone := cloneManifest(manifest)
	if !reflect.DeepEqual(clone, manifest) {
		t.Fatalf("cloneManifest() = %#v, want %#v", clone, manifest)
	}
	assertNoSharedMemory(t, "ExecutionRecord", reflect.ValueOf(clone), reflect.ValueOf(manifest))
	if empty := cloneManifest(record.ExecutionRecord{}); !reflect.DeepEqual(empty, record.ExecutionRecord{}) {
		t.Fatalf("cloneManifest(zero) = %#v, want the zero record with nil pointers, slices and maps", empty)
	}
}

// populate fills every pointer, slice and map so a clone has something to share.
func populate(value reflect.Value) {
	switch value.Kind() {
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		populate(value.Elem())
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
		populate(value.Index(0))
	case reflect.Array:
		for index := range value.Len() {
			populate(value.Index(index))
		}
	case reflect.Map:
		value.Set(reflect.MakeMap(value.Type()))
		key := reflect.New(value.Type().Key()).Elem()
		populate(key)
		element := reflect.New(value.Type().Elem()).Elem()
		populate(element)
		value.SetMapIndex(key, element)
	case reflect.Struct:
		for index := range value.NumField() {
			populate(value.Field(index))
		}
	case reflect.String:
		value.SetString("x")
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(1)
	}
}

func assertNoSharedMemory(t *testing.T, path string, clone, original reflect.Value) {
	t.Helper()
	switch original.Kind() {
	case reflect.Pointer:
		if !original.IsNil() && clone.Pointer() == original.Pointer() {
			t.Fatalf("%s is shared", path)
		}
		if !original.IsNil() {
			assertNoSharedMemory(t, path, clone.Elem(), original.Elem())
		}
	case reflect.Slice:
		if original.Len() > 0 && clone.Pointer() == original.Pointer() {
			t.Fatalf("%s is shared", path)
		}
		for index := range original.Len() {
			assertNoSharedMemory(t, path+"[]", clone.Index(index), original.Index(index))
		}
	case reflect.Map:
		if !original.IsNil() && clone.Pointer() == original.Pointer() {
			t.Fatalf("%s is shared", path)
		}
		entries := original.MapRange()
		for entries.Next() {
			assertNoSharedMemory(t, path+"{}", clone.MapIndex(entries.Key()), entries.Value())
		}
	case reflect.Struct:
		for index := range original.NumField() {
			assertNoSharedMemory(t, path+"."+original.Type().Field(index).Name, clone.Field(index), original.Field(index))
		}
	}
}

func TestOpenedArtifactReadsPinnedDirectoryAfterReplacement(t *testing.T) {
	store := Store{Root: t.TempDir()}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	input := artifactInput(t)
	replacement := []byte("replacement stdout")
	input.Payloads[1] = Payload{Path: "stdout", Mode: 0o600, Data: replacement, SHA256: record.HashBytes(replacement), Size: record.Uint64String(len(replacement))}
	stream := &input.Record.Streams.Stdout
	stream.FullSHA256, stream.RetainedSHA256 = record.HashBytes(replacement), record.HashBytes(replacement)
	stream.TotalBytes, stream.RetainedBytes = record.Uint64String(len(replacement)), record.Uint64String(len(replacement))
	other, err := (Store{Root: t.TempDir()}).PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if other.Manifest.RecordHash == published.Manifest.RecordHash {
		t.Fatal("the replacement artifact must differ from the opened one")
	}
	if err := os.Rename(published.Path, published.Path+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(other.Path, published.Path); err != nil {
		t.Fatal(err)
	}
	if stdout, err := opened.ReadPayload("stdout", 64); err != nil || string(stdout) != "stdout" {
		t.Fatalf("ReadPayload after replacement = %q, %v", stdout, err)
	}
	copied := filepath.Join(t.TempDir(), "stdout")
	if err := opened.CopyPayload("stdout", copied, 0o600); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(copied); err != nil || string(content) != "stdout" {
		t.Fatalf("CopyPayload after replacement = %q, %v", content, err)
	}
	if info, err := os.Stat(copied); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("copied stdout after replacement mode = %v, %v", info, err)
	}
	if opened.Manifest().RecordHash != published.Manifest.RecordHash || opened.Snapshot().Manifest.RecordHash != published.Manifest.RecordHash {
		t.Fatal("the handle follows the replacement directory instead of its pinned one")
	}
}

func TestOpenedArtifactRejectsChangedPayloads(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		maximum uint64
		change  func(t *testing.T, directory string)
		want    string
	}{
		{name: "unlisted", payload: "extra", maximum: 64, want: `artifact payload "extra" is not listed`},
		{name: "over bound", payload: "stdout", maximum: 3, want: `artifact payload "stdout" exceeds its bound`},
		{name: "mode", payload: "stdout", maximum: 64, change: func(t *testing.T, directory string) {
			mustDo(t, os.Chmod(filepath.Join(directory, "stdout"), 0o644))
		}, want: "stdout metadata does not match its manifest"},
		{name: "size", payload: "stdout", maximum: 64, change: func(t *testing.T, directory string) {
			mustDo(t, os.WriteFile(filepath.Join(directory, "stdout"), []byte("longer stdout"), 0o600))
		}, want: "stdout metadata does not match its manifest"},
		{name: "hash", payload: "stdout", maximum: 64, change: func(t *testing.T, directory string) {
			mustDo(t, os.WriteFile(filepath.Join(directory, "stdout"), []byte("STDOUT"), 0o600))
		}, want: `artifact payload "stdout" identity mismatch`},
		{name: "symbolic link", payload: "stderr", maximum: 64, change: func(t *testing.T, directory string) {
			mustDo(t, os.Remove(filepath.Join(directory, "stderr")))
			mustDo(t, os.Symlink("stdout", filepath.Join(directory, "stderr")))
		}, want: "stderr is a symbolic link"},
		{name: "escaping directory", payload: "world/snapshot.json", maximum: 1 << 20, change: func(t *testing.T, directory string) {
			outside := t.TempDir()
			mustDo(t, os.Rename(filepath.Join(directory, "world"), filepath.Join(outside, "world")))
			mustDo(t, os.Symlink(filepath.Join(outside, "world"), filepath.Join(directory, "world")))
		}, want: "path escapes from parent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
			if err != nil {
				t.Fatal(err)
			}
			opened, err := OpenArtifact(published.Path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := opened.Close(); err != nil {
					t.Error(err)
				}
			}()
			if test.change != nil {
				test.change(t, published.Path)
			}
			if _, err := opened.ReadPayload(test.payload, test.maximum); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ReadPayload(%s) error = %v, want %q", test.payload, err, test.want)
			}
			if _, err := opened.OpenPayload(test.payload, test.maximum); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("OpenPayload(%s) error = %v, want %q", test.payload, err, test.want)
			}
			if test.name == "over bound" {
				return
			}
			directory := t.TempDir()
			existing := filepath.Join(directory, "existing")
			mustDo(t, os.WriteFile(existing, []byte("keep destination"), 0o600))
			fresh := filepath.Join(directory, "fresh")
			for _, destination := range []string{existing, fresh} {
				if err := opened.CopyPayload(test.payload, destination, 0o600); err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("CopyPayload(%s) error = %v, want source error %q before destination creation", test.payload, err, test.want)
				}
			}
			if content, err := os.ReadFile(existing); err != nil || string(content) != "keep destination" {
				t.Fatalf("destination after source validation error = %q, %v", content, err)
			}
			if info, err := os.Stat(existing); err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("destination after source validation error mode = %v, %v", info, err)
			}
			if _, err := os.Stat(fresh); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("CopyPayload created a destination before validating source: %v", err)
			}
		})
	}
}

func mustDo(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
