package wasi

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamespaceWireConformanceCorpus(t *testing.T) {
	type corpus struct {
		Name   string `json:"name"`
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	config := testConfig()
	environment := newTestEnvironment(t, config)
	session := namespaceConformanceSession{"namespace", namespaceConformanceInitialImage(environment, config), nil}
	marshal := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	base := marshal(session)
	boundary := func(value string) string { return marshal(map[string]string{"boundary": value}) + "\n" }
	var cases []corpus
	for _, test := range []struct{ name, input string }{
		{"volume-selector-without-storage", `{"model":"internal/model/volume","image":{}}`},
		{"namespace-selector-with-storage", `{"model":"namespace","image":{},"storage":{}}`},
		{"unknown-model", `{"model":"other","image":{}}`},
	} {
		cases = append(cases, corpus{test.name, test.input + "\n", boundary("invalid:conformance model")})
	}
	for _, field := range []string{"stdin_base64", "data_base64"} {
		var request map[string]json.RawMessage
		if err := json.Unmarshal([]byte(base), &request); err != nil {
			t.Fatal(err)
		}
		var image map[string]json.RawMessage
		if err := json.Unmarshal(request["image"], &image); err != nil {
			t.Fatal(err)
		}
		if field == "data_base64" {
			var entries []map[string]json.RawMessage
			if err := json.Unmarshal(image["entries"], &entries); err != nil {
				t.Fatal(err)
			}
			delete(entries[0], field)
			image["entries"] = json.RawMessage(marshal(entries))
		} else {
			delete(image, field)
		}
		request["image"] = json.RawMessage(marshal(image))
		cases = append(cases, corpus{"missing-required-nullable-" + field, marshal(request) + "\n", boundary("invalid:namespace image")})
	}
	image, declaration, err := declareNamespaceImage(environment)
	if err != nil {
		t.Fatal(err)
	}
	var binaryImage bytes.Buffer
	if err := image.write(&binaryImage); err != nil {
		t.Fatal(err)
	}
	session.BinaryImage = &namespaceConformanceBinaryImage{binaryImage.Bytes(), declaration.Files}
	cases = append(cases, corpus{"binary-image-matches-session", marshal(session) + "\n", "{}\n"})
	session.BinaryImage = &namespaceConformanceBinaryImage{[]byte("GMWN"), declaration.Files}
	cases = append(cases, corpus{"binary-image-truncated", marshal(session) + "\n", boundary("invalid:namespace image")})
	root := &namespaceImage{limits: config.Limits, epoch: config.Clock.EpochNanos, cwd: "/"}
	binaryImage.Reset()
	if err := root.write(&binaryImage); err != nil {
		t.Fatal(err)
	}
	session.BinaryImage = &namespaceConformanceBinaryImage{binaryImage.Bytes(), 1}
	cases = append(cases, corpus{"binary-image-state-mismatch", marshal(session) + "\n", boundary("invalid:binary namespace image mismatch")})
	session.BinaryImage = nil
	for _, test := range []struct {
		name     string
		change   func(*namespaceConformanceImage)
		boundary string
	}{
		{"constructor-malformed-entry", func(i *namespaceConformanceImage) { i.Entries[0].Path = []byte("/tmp/../escape") }, "invalid:namespace image entry"},
		{"constructor-readonly-writable-overlap", func(i *namespaceConformanceImage) { i.Entries[0].Readonly = true }, "invalid:writable mount overlaps captured input"},
		{"constructor-builds-missing-writable-directories", func(i *namespaceConformanceImage) { i.Entries = []namespaceConformanceEntry{} }, ""},
		{"constructor-files-capacity", func(i *namespaceConformanceImage) { i.Limits.Files = 1 }, "capacity:initial-namespace"},
		{"constructor-filesystem-capacity", func(i *namespaceConformanceImage) { i.Stdin = []byte("input"); i.Limits.FilesystemBytes = 4 }, "capacity:initial-namespace"},
	} {
		request := namespaceConformanceSession{"namespace", namespaceConformanceInitialImage(environment, config), nil}
		test.change(&request.Image)
		output := "{}\n"
		if test.boundary != "" {
			output = boundary(test.boundary)
		}
		input := marshal(request) + "\n"
		if test.name == "constructor-builds-missing-writable-directories" {
			input += `{"op":"path_open","input":{"fd":3,"path_base64":"ZmlsZQ==","dirflags":0,"oflags":1,"rights_base":64,"rights_inheriting":0,"fdflags":0}}` + "\n"
			output += `{"errno":0,"output":{"fd":5}}` + "\n"
		}
		cases = append(cases, corpus{test.name, input, output})
	}
	deep := strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33)
	malformedDeep := strings.Repeat("[", 33) + "|" + strings.Repeat("]", 33)
	cases = append(cases, corpus{"nesting-before-malformed-deep-token", `{"z":` + malformedDeep + `}` + "\n", boundary("invalid:JSON nesting")})
	for _, test := range []struct{ name, input, boundary string }{
		{"malformed-before-duplicate", `{"fd":0,"extra":{"\ud800":0},"fd":1}`, "invalid:malformed JSON"},
		{"malformed-value-before-later-duplicate", `{"z":"\ud800","a":{"x":0,"x":1}}`, "invalid:malformed JSON"},
		{"nesting-before-later-surrogate", `{"z":` + deep + `,"a":"\ud800"}`, "invalid:JSON nesting"},
		{"duplicate-before-malformed-value", `{"fd":0,"fd":{"\ud800":0}}`, "invalid:duplicate JSON field"},
		{"escaped-duplicate-key", `{"fd":0,"\u0066d":1}`, "invalid:duplicate JSON field"},
	} {
		cases = append(cases, corpus{test.name, base + "\n" + `{"op":"fd_close","input":` + test.input + "}\n", "{}\n" + boundary(test.boundary)})
	}
	cases = append(cases, corpus{"trailing-before-malformed-followup", base + "\n" + `{"fd":0} {"\ud800":0}` + "\n", "{}\n" + boundary("invalid:trailing JSON")})
	cases = append(cases, corpus{"initial-trailing-before-malformed-followup", `{"fd":0} {"\ud800":0}` + "\n", boundary("invalid:trailing JSON")})
	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			data, err := json.MarshalIndent(test, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			data = append(data, '\n')
			name := filepath.Join("testdata", "namespace-wire-conformance", test.Name+".json")
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
				t.Fatal("namespace wire corpus missing; regenerate with -update-conformance: ", err)
			}
			if !bytes.Equal(data, want) {
				t.Fatalf("namespace wire corpus drift in %s", name)
			}
			var fixture corpus
			decoder := json.NewDecoder(bytes.NewReader(want))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&fixture); err != nil {
				t.Fatal(err)
			}
			command := exec.CommandContext(t.Context(), namespaceConformanceHelper(t))
			command.Stdin = strings.NewReader(fixture.Input)
			var stderr bytes.Buffer
			command.Stderr = &stderr
			output, err := command.Output()
			if err != nil {
				t.Fatalf("conformance driver: %v; stderr=%s", err, stderr.String())
			}
			if err := helperProfileStderr(stderr.Bytes()); err != nil {
				t.Fatal(err)
			}
			if string(output) != fixture.Output {
				t.Fatalf("namespace wire %s\ngot: %s\nwant: %s", fixture.Name, output, fixture.Output)
			}
		})
	}
}
