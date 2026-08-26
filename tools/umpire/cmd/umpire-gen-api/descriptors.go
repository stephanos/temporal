package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type descriptorInput struct {
	Locator string
	Encoded []byte
}

func descriptorFileInput(path, locator string) (descriptorInput, error) {
	if path == "" {
		return descriptorInput{}, errors.New("descriptor path is required")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return descriptorInput{}, fmt.Errorf("read descriptor %q: %w", path, err)
	}
	return newDescriptorInput(locator, encoded)
}

func newDescriptorInput(locator string, encoded []byte) (descriptorInput, error) {
	set := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(encoded, set); err != nil {
		return descriptorInput{}, fmt.Errorf("decode descriptor %q: %w", locator, err)
	}
	return descriptorInput{Locator: locator, Encoded: encoded}, nil
}

func mergeDescriptorInputs(inputs []descriptorInput) (*descriptorpb.FileDescriptorSet, error) {
	files := make(map[string]*descriptorpb.FileDescriptorProto)
	owners := make(map[string]string)
	for _, input := range inputs {
		set := &descriptorpb.FileDescriptorSet{}
		if err := proto.Unmarshal(input.Encoded, set); err != nil {
			return nil, fmt.Errorf("decode descriptor %q: %w", input.Locator, err)
		}
		for _, file := range set.File {
			if file.GetName() == "" {
				return nil, fmt.Errorf("descriptor %q contains a file without a path", input.Locator)
			}
			if previous, exists := files[file.GetName()]; exists {
				if !proto.Equal(previous, file) {
					return nil, fmt.Errorf(
						"descriptor file %q has conflicting definitions in descriptor paths %q and %q",
						file.GetName(), owners[file.GetName()], input.Locator,
					)
				}
				continue
			}
			files[file.GetName()] = proto.CloneOf(file)
			owners[file.GetName()] = input.Locator
		}
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	slices.SortFunc(paths, strings.Compare)
	result := &descriptorpb.FileDescriptorSet{}
	for _, path := range paths {
		result.File = append(result.File, files[path])
	}
	return result, nil
}
