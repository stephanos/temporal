package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type descriptorInput struct {
	Name    string `json:"name"`
	Locator string `json:"locator"`
	Digest  string `json:"digest"`
	Encoded []byte `json:"-"`
}

func descriptorFileInput(name, path string) (descriptorInput, error) {
	if path == "" {
		return descriptorInput{}, errors.New("descriptor path is required")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return descriptorInput{}, fmt.Errorf("read %s descriptor %q: %w", name, path, err)
	}
	return newDescriptorInput(name, filepath.ToSlash(path), encoded), nil
}

func newDescriptorInput(name, locator string, encoded []byte) descriptorInput {
	digest := sha256.Sum256(encoded)
	return descriptorInput{
		Name: name, Locator: locator, Digest: "sha256:" + hex.EncodeToString(digest[:]), Encoded: encoded,
	}
}

func mergeDescriptorInputs(inputs []descriptorInput) (*descriptorpb.FileDescriptorSet, error) {
	files := make(map[string]*descriptorpb.FileDescriptorProto)
	encodedFiles := make(map[string][]byte)
	for _, input := range inputs {
		set := &descriptorpb.FileDescriptorSet{}
		if err := proto.Unmarshal(input.Encoded, set); err != nil {
			return nil, fmt.Errorf("decode %s descriptor %q: %w", input.Name, input.Locator, err)
		}
		for _, file := range set.File {
			if file.GetName() == "" {
				return nil, fmt.Errorf("%s descriptor %q contains a file without a path", input.Name, input.Locator)
			}
			encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(file)
			if err != nil {
				return nil, fmt.Errorf("encode descriptor file %q: %w", file.GetName(), err)
			}
			if previous, exists := encodedFiles[file.GetName()]; exists {
				if !bytes.Equal(previous, encoded) {
					return nil, fmt.Errorf("descriptor file %q has conflicting definitions", file.GetName())
				}
				continue
			}
			files[file.GetName()] = proto.CloneOf(file)
			encodedFiles[file.GetName()] = encoded
		}
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	result := &descriptorpb.FileDescriptorSet{}
	for _, path := range paths {
		result.File = append(result.File, files[path])
	}
	return result, nil
}

func exportPublicDescriptors(ctx context.Context, repositoryRoot, module string) (descriptorInput, string, error) {
	version, err := commandOutput(ctx, repositoryRoot, "go", "list", "-m", "-f", "{{.Version}}", module)
	if err != nil {
		return descriptorInput{}, "", fmt.Errorf("resolve public API module version: %w", err)
	}
	packageList, err := commandOutput(ctx, repositoryRoot, "go", "list", "-f", "{{.ImportPath}}\t{{.Dir}}\t{{join .GoFiles \",\"}}", module+"/...")
	if err != nil {
		return descriptorInput{}, "", fmt.Errorf("list public API protobuf packages: %w", err)
	}
	var packages []string
	for _, line := range strings.Split(packageList, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 || strings.Contains(parts[0], "/internal/") {
			continue
		}
		containsTemporalProto, inspectErr := packageHasTemporalProto(parts[1], parts[2])
		if inspectErr != nil {
			return descriptorInput{}, "", inspectErr
		}
		if containsTemporalProto {
			packages = append(packages, parts[0])
		}
	}
	if len(packages) == 0 {
		return descriptorInput{}, "", fmt.Errorf("public API module %q exposed no production protobuf packages", module)
	}
	slices.Sort(packages)
	packages = slices.Compact(packages)

	temporaryRoot, err := os.MkdirTemp("", "umpire-gen-api-public-*")
	if err != nil {
		return descriptorInput{}, "", fmt.Errorf("create public descriptor exporter: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporaryRoot) }()
	helperPath := filepath.Join(temporaryRoot, "main.go")
	if err := os.WriteFile(helperPath, []byte(publicDescriptorHelper(packages)), 0o600); err != nil {
		return descriptorInput{}, "", fmt.Errorf("write public descriptor exporter: %w", err)
	}
	command := exec.CommandContext(ctx, "go", "run", helperPath)
	command.Dir = repositoryRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return descriptorInput{}, "", fmt.Errorf("export public API descriptors: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	version = strings.TrimSpace(version)
	return newDescriptorInput("public", module+"@"+version, stdout.Bytes()), version, nil
}

func packageHasTemporalProto(directory, files string) (bool, error) {
	for _, file := range strings.Split(files, ",") {
		if !strings.HasSuffix(file, ".pb.go") {
			continue
		}
		encoded, err := os.ReadFile(filepath.Join(directory, file))
		if err != nil {
			return false, fmt.Errorf("inspect generated public descriptor %q: %w", filepath.Join(directory, file), err)
		}
		if bytes.Contains(encoded, []byte("// source: temporal/api/")) {
			return true, nil
		}
	}
	return false, nil
}

func commandOutput(ctx context.Context, directory, name string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func publicDescriptorHelper(packages []string) string {
	var result strings.Builder
	result.WriteString("package main\n\nimport (\n")
	for _, packagePath := range packages {
		result.WriteString("\t_ ")
		result.WriteString(strconv.Quote(packagePath))
		result.WriteString("\n")
	}
	result.WriteString(`
	"fmt"
	"os"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	files := make(map[string]protoreflect.FileDescriptor)
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := files[file.Path()]; exists {
			return
		}
		files[file.Path()] = file
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			add(imports.Get(index).FileDescriptor)
		}
	}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(file.Path(), "temporal/api/") {
			add(file)
		}
		return true
	})
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	set := &descriptorpb.FileDescriptorSet{}
	for _, path := range paths {
		set.File = append(set.File, protodesc.ToFileDescriptorProto(files[path]))
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(encoded); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`)
	return result.String()
}
