package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	mode := flag.String("mode", "generate", "generate or check")
	output := flag.String("output", "tests/umpire3/model/Temporal/API/Generated/Nexus.lean", "generated Lean output")
	flag.Parse()
	if err := run(*mode, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode, output string) error {
	generated, err := generateNexusAPI()
	if err != nil {
		return err
	}
	switch mode {
	case "generate":
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return fmt.Errorf("create generated Nexus API directory: %w", err)
		}
		if err := os.WriteFile(output, generated, 0o600); err != nil {
			return fmt.Errorf("write generated Nexus API: %w", err)
		}
	case "check":
		current, err := os.ReadFile(output)
		if err != nil {
			return fmt.Errorf("read generated Nexus API: %w", err)
		}
		if !bytes.Equal(current, generated) {
			return errors.New("generated Nexus API is stale; run make umpire3-gen-api")
		}
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
	return nil
}

func generateNexusAPI() ([]byte, error) {
	descriptor := (&workflowservice.RequestCancelNexusOperationExecutionRequest{}).ProtoReflect().Descriptor()
	fileDescriptor := protodesc.ToFileDescriptorProto(descriptor.ParentFile())
	encodedDescriptor, err := proto.MarshalOptions{Deterministic: true}.Marshal(fileDescriptor)
	if err != nil {
		return nil, fmt.Errorf("marshal Temporal API descriptor: %w", err)
	}
	digest := sha256.Sum256(encodedDescriptor)

	var generated strings.Builder
	generated.WriteString("namespace Umpire3.Temporal.API.Generated\n\n")
	fmt.Fprintf(&generated, "def descriptorFullName : String := %q\n", descriptor.FullName())
	fmt.Fprintf(&generated, "def descriptorHash : String := %q\n\n", "sha256:"+hex.EncodeToString(digest[:]))
	generated.WriteString("structure RequestCancelNexusOperationExecutionRequest where\n")
	fields := descriptor.Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		if field.Cardinality() == protoreflect.Repeated || field.Kind() != protoreflect.StringKind {
			return nil, fmt.Errorf("unsupported selected Nexus field %s", field.FullName())
		}
		fmt.Fprintf(&generated, "  %s : String\n", leanFieldName(string(field.Name())))
	}
	generated.WriteString("  deriving DecidableEq, Repr\n\n")
	generated.WriteString("end Umpire3.Temporal.API.Generated\n")
	return []byte(generated.String()), nil
}

func leanFieldName(protoName string) string {
	parts := strings.Split(protoName, "_")
	for index := 1; index < len(parts); index++ {
		runes := []rune(parts[index])
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		parts[index] = string(runes)
	}
	name := strings.Join(parts, "")
	if name == "namespace" {
		return "namespaceName"
	}
	return name
}
