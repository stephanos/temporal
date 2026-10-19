package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type namespaceImageConformanceCase struct {
	Name            string                           `json:"name"`
	Data            []byte                           `json:"data_base64"`
	Files           uint64                           `json:"files"`
	Output          uint64                           `json:"output_bytes"`
	Epoch           uint64                           `json:"epoch_nanos"`
	ImageSHA256     string                           `json:"image_sha256,omitempty"`
	NamespaceSHA256 string                           `json:"namespace_sha256,omitempty"`
	Boundary        string                           `json:"boundary,omitempty"`
	Repeat          *namespaceImageConformanceRepeat `json:"repeat,omitempty"`
}

type namespaceImageConformanceRepeat struct {
	Offset int  `json:"offset"`
	Count  int  `json:"count"`
	Byte   byte `json:"byte"`
}

func TestNamespaceImageConformanceCorpus(t *testing.T) {
	for _, test := range namespaceImageConformanceCases(t) {
		t.Run(test.Name, func(t *testing.T) {
			data, err := json.MarshalIndent(test, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			data = append(data, '\n')
			name := filepath.Join("testdata", "namespace-image-conformance", test.Name+".json")
			if *updateNamespaceConformance {
				if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(name)
			if err != nil {
				t.Fatal("namespace image corpus missing; regenerate with -update-conformance: ", err)
			}
			if !bytes.Equal(data, want) {
				t.Fatalf("Go namespace image drift in %s", name)
			}
		})
	}
}

func namespaceImageConformanceCases(t *testing.T) []namespaceImageConformanceCase {
	t.Helper()
	entry := func(path string, directory, readonly bool, data string) namespaceImageEntry {
		return namespaceImageEntry{path, &node{directory: directory, readonly: readonly, data: []byte(data)}}
	}
	base := func() *namespaceImage {
		return &namespaceImage{
			limits: Limits{Files: 10, Descriptors: 10, FilesystemBytes: 20, OutputBytes: 3}, epoch: 4,
			captured: []string{"/workspace/input"}, writable: []string{"/workspace"}, preopens: []string{"/workspace", "/workspace/input"},
			cwd: "/workspace", stdin: []byte("in"), entries: []namespaceImageEntry{
				entry("/workspace", true, false, ""), entry("/workspace/input", true, true, ""), entry("/workspace/input/a", false, true, "abc"),
			},
		}
	}
	encode := func(image *namespaceImage) []byte {
		var output bytes.Buffer
		if err := image.write(&output); err != nil {
			t.Fatal(err)
		}
		return output.Bytes()
	}
	projection := func(image *namespaceImage) string {
		paths := func(values []string) []string {
			result := make([]string, 0, len(values))
			for _, value := range values {
				result = append(result, base64.StdEncoding.EncodeToString([]byte(value)))
			}
			return result
		}
		entries := make([]map[string]any, 0, len(image.entries))
		for _, e := range image.entries {
			entries = append(entries, map[string]any{"path_base64": base64.StdEncoding.EncodeToString([]byte(e.path)), "directory": e.node.directory, "readonly": e.node.readonly, "data_base64": base64.StdEncoding.EncodeToString(e.node.data)})
		}
		data, err := json.Marshal(map[string]any{
			"entries": entries, "writable_directories": paths(image.writable), "preopens": paths(image.preopens), "working_directory": base64.StdEncoding.EncodeToString([]byte(image.cwd)),
			"stdin_base64": base64.StdEncoding.EncodeToString(image.stdin), "epoch_nanos": image.epoch,
			"limits": map[string]uint64{"files": image.limits.Files, "descriptors": image.limits.Descriptors, "filesystem_bytes": image.limits.FilesystemBytes, "output_bytes": image.limits.OutputBytes},
		})
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	var cases []namespaceImageConformanceCase
	add := func(name string, image *namespaceImage, valid bool) {
		test := namespaceImageConformanceCase{Name: name, Data: encode(image), Files: uint64(len(image.entries)) + 1, Output: image.limits.OutputBytes, Epoch: image.epoch}
		if valid {
			test.ImageSHA256 = projection(image)
		} else {
			test.Boundary = "invalid:namespace image"
		}
		cases = append(cases, test)
	}
	add("captured-readonly-under-writable-ancestor", base(), true)
	root := func() *namespaceImage {
		return &namespaceImage{limits: Limits{Files: 1, Descriptors: 5, FilesystemBytes: 2, OutputBytes: 3}, epoch: 4, cwd: "/", stdin: []byte("x")}
	}
	add("hand-checked-root-with-stdin", root(), true)
	image := root()
	image.limits = Limits{Files: 10, Descriptors: 5, FilesystemBytes: 16, OutputBytes: 32}
	image.epoch = 9
	image.stdin = []byte("hi")
	add("go-literal-root-with-stdin-hi", image, true)
	cases[len(cases)-1].NamespaceSHA256 = "30433144fd02a9eeca8a996fcd1ac435506af1fb83f63013e7315e2b12e722bb"
	for _, test := range []struct {
		name   string
		change func(*namespaceImage)
	}{
		{"zero-files", func(i *namespaceImage) { i.limits.Files = 0 }}, {"files-ceiling", func(i *namespaceImage) { i.limits.Files = 1<<20 + 1 }},
		{"descriptors-below-five", func(i *namespaceImage) { i.limits.Descriptors = 4 }}, {"descriptors-ceiling", func(i *namespaceImage) { i.limits.Descriptors = 1<<20 + 1 }},
		{"zero-filesystem-bytes", func(i *namespaceImage) { i.limits.FilesystemBytes = 0 }}, {"filesystem-bytes-ceiling", func(i *namespaceImage) { i.limits.FilesystemBytes = 1<<30 + 1 }},
		{"zero-output", func(i *namespaceImage) { i.limits.OutputBytes = 0 }}, {"output-ceiling", func(i *namespaceImage) { i.limits.OutputBytes = 1<<30 + 1 }},
		{"epoch-overflow", func(i *namespaceImage) { i.epoch = math.MaxInt64 + 1 }},
		{"directory-has-data", func(i *namespaceImage) { i.entries[0].node.data = []byte("bad") }},
		{"parent-after-child", func(i *namespaceImage) { i.entries[0], i.entries[1] = i.entries[1], i.entries[0] }},
		{"regular-file-parent", func(i *namespaceImage) { i.entries[0].node.directory = false }},
		{"duplicate-entry", func(i *namespaceImage) { i.entries[2].path = i.entries[1].path }},
		{"duplicate-captured-root", func(i *namespaceImage) { i.captured = append(i.captured, "/workspace/input") }},
		{"unsorted-captured-roots", func(i *namespaceImage) { i.captured = append(i.captured, "/a") }},
		{"missing-captured-root", func(i *namespaceImage) { i.captured[0] = "/missing" }},
		{"captured-file-root", func(i *namespaceImage) { i.captured[0] = "/workspace/input/a" }},
		{"captured-directory-not-readonly", func(i *namespaceImage) { i.entries[1].node.readonly = false }},
		{"captured-file-not-readonly", func(i *namespaceImage) { i.entries[2].node.readonly = false }},
		{"readonly-outside-captured-root", func(i *namespaceImage) { i.captured = nil }},
		{"writable-equals-captured-root", func(i *namespaceImage) { i.writable[0] = "/workspace/input" }},
		{"writable-beneath-captured-root", func(i *namespaceImage) {
			i.writable[0] = "/workspace/input/sub"
			i.entries = append(i.entries, entry("/workspace/input/sub", true, false, ""))
		}},
		{"duplicate-writable-root", func(i *namespaceImage) { i.writable = append(i.writable, "/workspace") }},
		{"missing-writable-root", func(i *namespaceImage) { i.writable[0] = "/missing" }},
		{"no-required-preopens", func(i *namespaceImage) { i.preopens = nil }},
		{"duplicate-preopen", func(i *namespaceImage) { i.preopens = append(i.preopens, "/workspace/input") }},
		{"unsorted-preopens", func(i *namespaceImage) { i.preopens[0], i.preopens[1] = i.preopens[1], i.preopens[0] }},
		{"missing-preopen", func(i *namespaceImage) { i.preopens[0] = "/missing" }},
		{"file-preopen", func(i *namespaceImage) { i.preopens[1] = "/workspace/input/a" }},
		{"missing-cwd", func(i *namespaceImage) { i.cwd = "/missing" }},
		{"file-cwd", func(i *namespaceImage) { i.cwd = "/workspace/input/a" }},
		{"trailing-slash-cwd", func(i *namespaceImage) { i.cwd = "/workspace/" }},
		{"files-include-root", func(i *namespaceImage) { i.limits.Files = 3 }},
		{"preopens-plus-standard-fds-over-descriptors", func(i *namespaceImage) {
			i.limits.Descriptors = 5
			i.preopens = append(i.preopens, "/workspace/third")
			i.entries = append(i.entries, entry("/workspace/third", true, false, ""))
		}},
		{"stdin-plus-data-over-budget", func(i *namespaceImage) { i.limits.FilesystemBytes = 4 }},
	} {
		image := base()
		test.change(image)
		add(test.name, image, false)
	}
	for _, test := range []struct{ name, path string }{
		{"root", "/"}, {"relative", "relative"}, {"empty-component", "/workspace//a"}, {"dot-component", "/workspace/./a"}, {"parent-component", "/workspace/../a"}, {"trailing-slash", "/workspace/a/"}, {"nul", "/workspace/a\x00"}, {"non-utf8", "/workspace/\xff"},
	} {
		image := base()
		image.entries[2].path = test.path
		add("entry-path-"+test.name, image, false)
	}
	for _, test := range []struct {
		name   string
		offset int
		value  byte
	}{{"entry-kind", 0, 7}, {"entry-readonly", 1, 2}} {
		image := base()
		data := encode(image)
		at := 80
		for _, path := range append(append(append([]string{}, image.captured...), image.writable...), image.preopens...) {
			at += 8 + len(path)
		}
		at += 8 + len(image.cwd) + 8 + len(image.stdin)
		data[at+test.offset] = test.value
		cases = append(cases, namespaceImageConformanceCase{Name: test.name, Data: data, Files: 4, Output: 3, Epoch: 4, Boundary: "invalid:namespace image"})
	}
	for _, test := range []struct {
		name                 string
		files, output, epoch uint64
	}{{"identity-files", 5, 3, 4}, {"identity-output", 4, 2, 4}, {"identity-epoch", 4, 3, 5}} {
		cases = append(cases, namespaceImageConformanceCase{Name: test.name, Data: encode(base()), Files: test.files, Output: test.output, Epoch: test.epoch, Boundary: "invalid:namespace image"})
	}
	for _, source := range []struct {
		name  string
		image *namespaceImage
	}{{"root", root()}, {"entry", base()}} {
		data := encode(source.image)
		for length := 0; length < len(data); length++ {
			cases = append(cases, namespaceImageConformanceCase{Name: fmt.Sprintf("%s-truncated-%03d", source.name, length), Data: bytes.Clone(data[:length]), Files: uint64(len(source.image.entries)) + 1, Output: source.image.limits.OutputBytes, Epoch: source.image.epoch, Boundary: "invalid:namespace image"})
		}
	}
	for _, test := range []struct {
		name   string
		offset int
	}{{"magic", 0}, {"version", 4}, {"terminator", len(encode(root())) - 1}} {
		data := encode(root())
		data[test.offset] = 9
		cases = append(cases, namespaceImageConformanceCase{Name: "invalid-" + test.name, Data: data, Files: 1, Output: 3, Epoch: 4, Boundary: "invalid:namespace image"})
	}
	cases = append(cases, namespaceImageConformanceCase{Name: "trailing-data", Data: append(encode(root()), 0), Files: 1, Output: 3, Epoch: 4, Boundary: "invalid:namespace image"})
	for _, test := range []struct {
		name   string
		offset int
	}{{"entries", 48}, {"writable", 56}, {"captured", 64}, {"preopens", 72}} {
		data := encode(root())
		binary.LittleEndian.PutUint64(data[test.offset:], math.MaxUint64)
		cases = append(cases, namespaceImageConformanceCase{Name: "overflow-count-" + test.name, Data: data, Files: 1, Output: 3, Epoch: 4, Boundary: "invalid:namespace image"})
	}
	for _, offset := range []int{80, 89} {
		data := bytes.Clone(encode(root())[:offset])
		data = binary.LittleEndian.AppendUint64(data, math.MaxUint64)
		cases = append(cases, namespaceImageConformanceCase{Name: fmt.Sprintf("overflow-blob-at-%d", offset), Data: data, Files: 1, Output: 3, Epoch: 4, Boundary: "invalid:namespace image"})
	}
	image = base()
	image.limits.Files = 4
	image.limits.FilesystemBytes = 5
	add("stdin-plus-data-exact-budget", image, true)
	image.limits.FilesystemBytes = 3
	add("stdin-plus-data-exceeds-budget", image, false)
	image.stdin = nil
	add("cleared-stdin-data-exact-budget", image, true)
	image = base()
	image.captured = nil
	image.writable = []string{"/z", "/a"}
	image.preopens = []string{"/a", "/z"}
	image.cwd = "/z"
	image.entries = []namespaceImageEntry{entry("/z", true, false, ""), entry("/a", true, false, ""), entry("/z/writable", false, false, "bytes")}
	add("unsorted-writable-preserves-entry-order", image, true)
	image = base()
	image.writable = append(image.writable, "/workspace/input2")
	image.preopens = append(image.preopens, "/workspace/input2")
	image.entries = append(image.entries, entry("/workspace/input2", true, false, ""))
	add("captured-prefix-sibling-writable", image, true)
	image.entries = append(image.entries, entry("/workspace/input2/a", false, true, ""))
	add("captured-prefix-sibling-readonly-refused", image, false)
	for _, mode := range []string{"captured", "writable", "both"} {
		image = root()
		image.entries = nil
		image.preopens = []string{"/"}
		image.writable = nil
		image.captured = nil
		if mode != "writable" {
			image.captured = []string{"/"}
		}
		if mode != "captured" {
			image.writable = []string{"/"}
		}
		add("root-mount-"+mode, image, false)
	}
	image = base()
	image.captured = append(image.captured, "/workspace/input/sub")
	image.preopens = append(image.preopens, "/workspace/input/sub")
	image.entries = append(image.entries, entry("/workspace/input/sub", true, true, ""))
	add("overlapping-captured-roots", image, false)
	image.captured = []string{"/workspace/input", "/workspace/input-", "/workspace/input/sub"}
	image.preopens = []string{"/workspace", "/workspace/input", "/workspace/input-", "/workspace/input/sub"}
	image.entries = append(image.entries, entry("/workspace/input-", true, true, ""))
	add("overlap-with-intervening-sibling", image, false)
	image = base()
	image.limits.FilesystemBytes = 2
	image.stdin = nil
	image.entries[2] = entry("/workspace/input/"+strings.Repeat("a", (4<<20)+1), true, true, "")
	add("large-path-metadata-under-tiny-data-budget", image, true)
	last := &cases[len(cases)-1]
	run := bytes.Repeat([]byte{'a'}, (4<<20)+1)
	at := bytes.Index(last.Data, run)
	if at < 0 {
		t.Fatal("missing long path payload")
	}
	last.Data = append(bytes.Clone(last.Data[:at]), last.Data[at+len(run):]...)
	last.Repeat = &namespaceImageConformanceRepeat{Offset: at, Count: len(run), Byte: 'a'}
	return cases
}
