package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/common/artifactio"
	"go.temporal.io/server/tools/common/protofile"
)

var removeDescriptorHelper = os.RemoveAll

type repeatedStrings []string

func (values *repeatedStrings) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedStrings) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value cannot be empty")
	}
	*values = append(*values, value)
	return nil
}

func Run(ctx context.Context, arguments []string) error {
	var patterns repeatedStrings
	var rawPrefixes repeatedStrings
	flags := flag.NewFlagSet("genleanmodeldescriptors", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Var(&patterns, "package-pattern", "Go package pattern containing registered descriptors (repeatable)")
	flags.Var(&rawPrefixes, "file-prefix", "protobuf file prefix to export (repeatable)")
	output := flags.String("output", "", "output FileDescriptorSet path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if len(patterns) == 0 {
		return errors.New("at least one --package-pattern is required")
	}
	if len(rawPrefixes) == 0 {
		return errors.New("at least one --file-prefix is required")
	}
	if strings.TrimSpace(*output) == "" {
		return errors.New("--output is required")
	}
	if filepath.Clean(*output) == "." || strings.HasSuffix(*output, string(filepath.Separator)) {
		return fmt.Errorf("invalid --output path %q", *output)
	}

	prefixes := make([]string, 0, len(rawPrefixes))
	for _, prefix := range rawPrefixes {
		normalized, err := protofile.NormalizePrefix(prefix)
		if err != nil {
			return fmt.Errorf("invalid file prefix %q: %w", prefix, err)
		}
		prefixes = append(prefixes, normalized)
	}
	slices.Sort(prefixes)
	prefixes = slices.Compact(prefixes)
	packages, err := listDescriptorPackages(ctx, patterns, prefixes)
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		return fmt.Errorf(
			"no registered protobuf descriptors matched the configured file prefixes in package patterns %q",
			strings.Join(patterns, ", "),
		)
	}
	encoded, err := exportDescriptors(ctx, packages, prefixes)
	if err != nil {
		return err
	}
	if err := artifactio.Publish(*output, encoded); err != nil {
		return fmt.Errorf("write descriptor set %q: %w", *output, err)
	}
	return nil
}

func listDescriptorPackages(ctx context.Context, patterns, prefixes []string) ([]string, error) {
	seen := make(map[string]bool)
	var packages []string
	for _, pattern := range patterns {
		command := exec.CommandContext(
			ctx,
			"go", "list", "-f", "{{.ImportPath}}\t{{.Dir}}\t{{join .GoFiles \",\"}}", pattern,
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			return nil, fmt.Errorf("go list package pattern %q: %w: %s", pattern, err, strings.TrimSpace(stderr.String()))
		}
		for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 || parts[0] == "" || strings.Contains(parts[0], "/internal/") {
				continue
			}
			matches, err := packageContainsPrefix(parts[1], parts[2], prefixes)
			if err != nil {
				return nil, fmt.Errorf("inspect generated protobuf package %q: %w", parts[0], err)
			}
			if !matches {
				continue
			}
			if !seen[parts[0]] {
				seen[parts[0]] = true
				packages = append(packages, parts[0])
			}
		}
	}
	slices.Sort(packages)
	return packages, nil
}

func packageContainsPrefix(directory, files string, prefixes []string) (bool, error) {
	for _, file := range strings.Split(files, ",") {
		if !strings.HasSuffix(file, ".pb.go") {
			continue
		}
		encoded, err := os.ReadFile(filepath.Join(directory, file))
		if err != nil {
			return false, err
		}
		for _, prefix := range prefixes {
			if bytes.Contains(encoded, []byte("// source: "+prefix)) {
				return true, nil
			}
		}
	}
	return false, nil
}

func exportDescriptors(ctx context.Context, packages, prefixes []string) (encoded []byte, resultErr error) {
	temporaryRoot, err := os.MkdirTemp("", "genleanmodeldescriptors-*")
	if err != nil {
		return nil, fmt.Errorf("create descriptor helper: %w", err)
	}
	defer func() {
		if err := removeDescriptorHelper(temporaryRoot); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove descriptor helper: %w", err))
		}
	}()
	source, err := helperSource(packages, prefixes)
	if err != nil {
		return nil, err
	}
	helperPath := filepath.Join(temporaryRoot, "main.go")
	if err := os.WriteFile(helperPath, source, 0o600); err != nil {
		return nil, fmt.Errorf("write descriptor helper: %w", err)
	}
	command := exec.CommandContext(ctx, "go", "run", helperPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		diagnostic := strings.TrimSpace(stderr.String())
		if strings.Contains(diagnostic, "no registered protobuf descriptors matched") {
			return nil, errors.New("no registered protobuf descriptors matched the configured file prefixes")
		}
		return nil, fmt.Errorf("run descriptor helper: %w: %s", err, diagnostic)
	}
	return stdout.Bytes(), nil
}

func helperSource(packages, prefixes []string) ([]byte, error) {
	var source strings.Builder
	source.WriteString("package main\n\nimport (\n")
	for _, packagePath := range packages {
		fmt.Fprintf(&source, "\t_ %s\n", strconv.Quote(packagePath))
	}
	source.WriteString(`
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
	prefixes := []string{`)
	for index, prefix := range prefixes {
		if index != 0 {
			source.WriteString(", ")
		}
		source.WriteString(strconv.Quote(prefix))
	}
	source.WriteString(`}
	files := make(map[string]protoreflect.FileDescriptor)
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := files[file.Path()]; exists {
			return
		}
		files[file.Path()] = file
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			add(imports.Get(index))
		}
	}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(file.Path(), prefix) {
				add(file)
				break
			}
		}
		return true
	})
	if len(files) == 0 {
		if _, err := fmt.Fprintln(os.Stderr, "no registered protobuf descriptors matched"); err != nil {
			panic(err)
		}
		os.Exit(1)
	}
	paths := make([]string, 0, len(files))
	for filePath := range files {
		paths = append(paths, filePath)
	}
	slices.Sort(paths)
	set := &descriptorpb.FileDescriptorSet{}
	for _, filePath := range paths {
		set.File = append(set.File, protodesc.ToFileDescriptorProto(files[filePath]))
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(encoded); err != nil {
		panic(err)
	}
}
`)
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("format descriptor helper: %w", err)
	}
	return formatted, nil
}
