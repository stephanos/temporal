package authoring

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

const generationSchema = "gomad3.compatibility-pack-generation/v1"

type generationState struct {
	Schema  string                  `json:"schema"`
	Outputs []generationStateOutput `json:"outputs"`
}

type generationStateOutput struct {
	Path   string        `json:"path"`
	SHA256 record.SHA256 `json:"sha256"`
}

type renderedGeneration struct {
	files map[string][]byte
	state []byte
}

func Generate(root string, request Request, approval string) error {
	wantApproval, err := ApprovalSHA256(request)
	if err != nil {
		return err
	}
	if approval != wantApproval || request.ApprovalSHA256 != "" && request.ApprovalSHA256 != approval {
		return errors.New("compatibility-pack generation approval does not match the canonical review")
	}
	request.ApprovalSHA256 = approval
	requests, err := loadRequests(root, true)
	if err != nil {
		return err
	}
	requests[request.ID] = request
	rendered, err := renderGeneration(requests)
	if err != nil {
		return err
	}
	return publishGeneration(root, rendered)
}

func Regenerate(root string) error {
	requests, err := loadRequests(root, false)
	if err != nil {
		return err
	}
	rendered, err := renderGeneration(requests)
	if err != nil {
		return err
	}
	return publishGeneration(root, rendered)
}

func publishGeneration(root string, rendered renderedGeneration) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create compatibility-pack root: %w", err)
	}
	paths := make([]string, 0, len(rendered.files))
	for relative := range rendered.files {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		mode := os.FileMode(0o644)
		if strings.HasPrefix(relative, "requests/") {
			mode = 0o600
		}
		if err := hostfs.Replace(filepath.Join(root, filepath.FromSlash(relative)), rendered.files[relative], mode); err != nil {
			return fmt.Errorf("write generated compatibility-pack artifact %s: %w", relative, err)
		}
	}
	if err := hostfs.Replace(filepath.Join(root, "generation.json"), rendered.state, 0o644); err != nil {
		return fmt.Errorf("write compatibility-pack generation state: %w", err)
	}
	return nil
}

func Check(root string) error {
	requests, err := loadRequests(root, false)
	if err != nil {
		return err
	}
	rendered, err := renderGeneration(requests)
	if err != nil {
		return err
	}
	for relative, expected := range rendered.files {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || !bytes.Equal(actual, expected) {
			return fmt.Errorf("generated compatibility-pack artifact is stale: %s", relative)
		}
	}
	actualState, err := os.ReadFile(filepath.Join(root, "generation.json"))
	if err != nil || !bytes.Equal(actualState, rendered.state) {
		return errors.New("generated compatibility-pack artifact is stale: generation.json")
	}
	if err := rejectExtraGeneratedFiles(root, rendered.files); err != nil {
		return err
	}
	return nil
}

func loadRequests(root string, missingAllowed bool) (map[string]Request, error) {
	directory := filepath.Join(root, "requests")
	entries, err := os.ReadDir(directory)
	if err != nil {
		if missingAllowed && errors.Is(err, os.ErrNotExist) {
			return map[string]Request{}, nil
		}
		return nil, fmt.Errorf("read compatibility-pack requests: %w", err)
	}
	if len(entries) > 4096 {
		return nil, errors.New("compatibility-pack request count exceeds its bound")
	}
	requests := make(map[string]Request, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("compatibility-pack request entry is invalid: %s", entry.Name())
		}
		path := filepath.Join(directory, entry.Name())
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > MaximumRequestBytes {
			return nil, fmt.Errorf("compatibility-pack request entry is invalid: %s", entry.Name())
		}
		encoded, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read compatibility-pack request %s: %w", entry.Name(), err)
		}
		request, err := DecodeRequest(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode compatibility-pack request %s: %w", entry.Name(), err)
		}
		if entry.Name() != request.ID+".json" {
			return nil, fmt.Errorf("compatibility-pack request filename does not match ID: %s", entry.Name())
		}
		if _, duplicate := requests[request.ID]; duplicate {
			return nil, fmt.Errorf("compatibility-pack request ID is duplicated: %s", request.ID)
		}
		requests[request.ID] = request
	}
	return requests, nil
}

func renderGeneration(requests map[string]Request) (renderedGeneration, error) {
	ids := make([]string, 0, len(requests))
	for id := range requests {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	files := make(map[string][]byte, len(ids)*3+1)
	active := make([]Request, 0, len(ids))
	for _, id := range ids {
		request := requests[id]
		if err := ValidateRequest(request); err != nil {
			return renderedGeneration{}, fmt.Errorf("validate compatibility-pack request %s: %w", id, err)
		}
		requestBytes, err := canonicaljson.CanonicalJSON(request)
		if err != nil {
			return renderedGeneration{}, fmt.Errorf("encode compatibility-pack request %s: %w", id, err)
		}
		requestBytes = append(requestBytes, '\n')
		files["requests/"+id+".json"] = requestBytes
		report, approval, err := RenderReview(request)
		if err != nil {
			return renderedGeneration{}, err
		}
		files["reports/"+id+".md"] = report
		if request.ApprovalSHA256 == "" {
			continue
		}
		if request.ApprovalSHA256 != approval {
			return renderedGeneration{}, fmt.Errorf("compatibility-pack request %s approval is stale", id)
		}
		requestSHA256 := string(record.DomainHash("gomad3.compatibility-pack-request/v1", requestBytes))
		pack, err := projectPack(request, requestSHA256, approval, false)
		if err != nil {
			return renderedGeneration{}, err
		}
		packBytes, err := canonicaljson.CanonicalJSON(pack)
		if err != nil {
			return renderedGeneration{}, fmt.Errorf("encode compatibility pack %s: %w", id, err)
		}
		files["packs/"+id+".json"] = append(packBytes, '\n')
		active = append(active, request)
	}
	generatedTests, err := renderGeneratedTests(active)
	if err != nil {
		return renderedGeneration{}, err
	}
	files["packs_generated_test.go"] = generatedTests
	outputs := make([]generationStateOutput, 0, len(files))
	for relative, contents := range files {
		outputs = append(outputs, generationStateOutput{Path: relative, SHA256: record.HashBytes(contents)})
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	state, err := canonicaljson.CanonicalJSON(generationState{Schema: generationSchema, Outputs: outputs})
	if err != nil {
		return renderedGeneration{}, fmt.Errorf("encode compatibility-pack generation state: %w", err)
	}
	return renderedGeneration{files: files, state: append(state, '\n')}, nil
}

func renderGeneratedTests(requests []Request) ([]byte, error) {
	var source strings.Builder
	source.WriteString("// Code generated by gomadtool compatibility-pack generate. DO NOT EDIT.\n\n")
	source.WriteString("package compatibility\n\n")
	source.WriteString("var generatedPackMutationInventory = []generatedPackMutation{\n")
	for _, request := range requests {
		mutations := generatedMutations(request)
		for _, mutation := range mutations {
			fmt.Fprintf(&source, "\t{PackID: %s, Mutation: %s},\n", strconv.Quote(request.ID), strconv.Quote(mutation))
		}
	}
	source.WriteString("}\n")
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated compatibility-pack tests: %w", err)
	}
	return formatted, nil
}

func generatedMutations(request Request) []string {
	mutations := []string{
		"approval", "arbitrary_local_replacement", "availability", "go_source", "justification",
		"module_sum", "module_version", "owner", "pack_digest", "platform", "positive",
		"request_identity", "review_time", "source_set", "workload",
	}
	hasAdapter := false
	hasDirective := false
	hasForeignSource := false
	for _, activation := range request.Activation {
		hasAdapter = hasAdapter || activation.Evidence.Replacement.Adapter != nil
	}
	for _, pkg := range request.Packages {
		hasAdapter = hasAdapter || pkg.Evidence.Module.Replacement.Adapter != nil
		hasForeignSource = hasForeignSource || len(pkg.Evidence.ForeignSources) != 0
		for _, fact := range pkg.Facts {
			hasDirective = hasDirective || fact.Kind == FactLinkname && fact.Disposition == DispositionAllow
		}
	}
	if hasAdapter {
		mutations = append(mutations, "adapter_identity", "original_source_inventory", "prepared_source_set", "replacement_source_inventory")
	}
	if hasDirective {
		mutations = append(mutations, "directive")
	}
	if hasForeignSource {
		mutations = append(mutations, "foreign_source")
	}
	sort.Strings(mutations)
	return mutations
}

func rejectExtraGeneratedFiles(root string, expected map[string][]byte) error {
	for _, directory := range []string{"packs", "reports", "requests"} {
		entries, err := os.ReadDir(filepath.Join(root, directory))
		if err != nil {
			return fmt.Errorf("read generated compatibility-pack directory %s: %w", directory, err)
		}
		for _, entry := range entries {
			relative := directory + "/" + entry.Name()
			if _, found := expected[relative]; !found {
				return fmt.Errorf("generated compatibility-pack artifact is unexpected: %s", relative)
			}
		}
	}
	return nil
}
