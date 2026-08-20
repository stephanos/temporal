package protocol

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

type ProtobufField struct {
	FullName    string `json:"fullName"`
	Kind        string `json:"kind"`
	TypeName    string `json:"typeName,omitempty"`
	Presence    bool   `json:"presence"`
	Repeated    bool   `json:"repeated"`
	Map         bool   `json:"map"`
	Recursive   bool   `json:"recursive"`
	Disposition string `json:"disposition"`
}

type ProtobufInventory struct {
	DescriptorDigest string          `json:"descriptorDigest"`
	Roots            []string        `json:"roots"`
	Messages         int             `json:"messages"`
	Enums            int             `json:"enums"`
	Fields           []ProtobufField `json:"fields"`
	FieldClasses     []string        `json:"fieldClasses"`
}

//go:embed generated/descriptor-manifest.json
var descriptorManifestJSON []byte

func DefaultProtobufInventory() (ProtobufInventory, error) {
	var manifest struct {
		DescriptorDigest string   `json:"descriptorDigest"`
		Roots            []string `json:"roots"`
		Enums            []any    `json:"enums"`
		Messages         []struct {
			Fields []ProtobufField `json:"fields"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(descriptorManifestJSON, &manifest); err != nil {
		return ProtobufInventory{}, fmt.Errorf("decode generated descriptor manifest: %w", err)
	}
	inventory := ProtobufInventory{
		DescriptorDigest: manifest.DescriptorDigest, Roots: append([]string(nil), manifest.Roots...),
		Messages: len(manifest.Messages), Enums: len(manifest.Enums),
	}
	classes := make(map[string]struct{})
	for _, message := range manifest.Messages {
		for _, field := range message.Fields {
			inventory.Fields = append(inventory.Fields, field)
			classes["kind:"+field.Kind] = struct{}{}
			classes["disposition:"+field.Disposition] = struct{}{}
			if field.Presence {
				classes["presence"] = struct{}{}
			}
			if field.Repeated {
				classes["repeated"] = struct{}{}
			}
			if field.Map {
				classes["map"] = struct{}{}
			}
			if field.Recursive {
				classes["recursive"] = struct{}{}
			}
		}
	}
	slices.SortFunc(inventory.Fields, func(left, right ProtobufField) int {
		return compareStrings(left.FullName, right.FullName)
	})
	for class := range classes {
		inventory.FieldClasses = append(inventory.FieldClasses, class)
	}
	slices.Sort(inventory.FieldClasses)
	return inventory, nil
}

func compareStrings(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
