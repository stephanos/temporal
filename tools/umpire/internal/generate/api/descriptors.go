package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func descriptorFileInput(name, path, locator string) (descriptorInput, error) {
	if path == "" {
		return descriptorInput{}, errors.New("descriptor path is required")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return descriptorInput{}, fmt.Errorf("read %s descriptor %q: %w", name, path, err)
	}
	return newDescriptorInput(name, filepath.ToSlash(locator), encoded)
}

func newDescriptorInput(name, locator string, encoded []byte) (descriptorInput, error) {
	set := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(encoded, set); err != nil {
		return descriptorInput{}, fmt.Errorf("decode %s descriptor %q: %w", name, locator, err)
	}
	slices.SortFunc(set.File, func(left, right *descriptorpb.FileDescriptorProto) int {
		return strings.Compare(left.GetName(), right.GetName())
	})
	digestInput, err := (proto.MarshalOptions{Deterministic: true}).Marshal(set)
	if err != nil {
		return descriptorInput{}, fmt.Errorf("normalize %s descriptor %q: %w", name, locator, err)
	}
	digest := sha256.Sum256(digestInput)
	return descriptorInput{
		Name: name, Locator: locator, Digest: "sha256:" + hex.EncodeToString(digest[:]), Encoded: encoded,
	}, nil
}

func mergeDescriptorInputs(inputs []descriptorInput) (*descriptorpb.FileDescriptorSet, error) {
	files := make(map[string]*descriptorpb.FileDescriptorProto)
	owners := make(map[string]string)
	for _, input := range inputs {
		set := &descriptorpb.FileDescriptorSet{}
		if err := proto.Unmarshal(input.Encoded, set); err != nil {
			return nil, fmt.Errorf("decode %s descriptor %q: %w", input.Name, input.Locator, err)
		}
		for _, file := range set.File {
			if file.GetName() == "" {
				return nil, fmt.Errorf("%s descriptor %q contains a file without a path", input.Name, input.Locator)
			}
			if previous, exists := files[file.GetName()]; exists {
				if !proto.Equal(previous, file) {
					return nil, fmt.Errorf(
						"descriptor file %q has conflicting definitions in inputs %q and %q",
						file.GetName(), owners[file.GetName()], input.Name,
					)
				}
				continue
			}
			files[file.GetName()] = proto.CloneOf(file)
			owners[file.GetName()] = input.Name
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
