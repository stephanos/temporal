package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type volumeWireSchema struct {
	Version uint16             `json:"version"`
	Magic   map[string]string  `json:"magic"`
	Limits  map[string]uint64  `json:"limits"`
	Layouts []volumeWireLayout `json:"layouts"`
}

type volumeWireLayout struct {
	Name   string            `json:"name"`
	Fields []volumeWireField `json:"fields"`
}

type volumeWireField struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Type  string `json:"type,omitempty"`
	Limit string `json:"limit,omitempty"`
	Empty bool   `json:"empty,omitempty"`
}

type volumeWireConsumer struct {
	Package   string
	Prefix    string
	Host      bool
	Schema    volumeWireSchema
	Types     map[string]string
	Functions map[string]string
	Codecs    []volumeWireCodec
}

type volumeWireCodec struct {
	Type     string
	Function string
	Encode   string
	Decode   string
}

func volumeWireData(definition volumeWireSchema, host bool) (volumeWireConsumer, error) {
	consumer := volumeWireMapping(definition, host)
	for _, layout := range definition.Layouts {
		codec, err := volumeWireLayoutCodec(consumer, layout)
		if err != nil {
			return volumeWireConsumer{}, err
		}
		consumer.Codecs = append(consumer.Codecs, codec)
	}
	return consumer, nil
}

func readVolumeWireSchema(path string) (volumeWireSchema, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return volumeWireSchema{}, fmt.Errorf("read simulation volume wire schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var definition volumeWireSchema
	if err := decoder.Decode(&definition); err != nil {
		return volumeWireSchema{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return volumeWireSchema{}, errors.New("simulation volume wire schema has trailing data")
	}
	if definition.Version != 1 {
		return volumeWireSchema{}, errors.New("simulation volume wire schema version is unsupported")
	}
	seenMagic := make(map[string]bool)
	for _, name := range []string{"config", "finish", "error", "enumeration", "frontier"} {
		magic := definition.Magic[name]
		if len(magic) != 8 || seenMagic[magic] {
			return volumeWireSchema{}, fmt.Errorf("simulation volume wire %s magic is invalid", name)
		}
		seenMagic[magic] = true
	}
	for _, name := range []string{"bytes", "string_bytes", "nodes", "volumes", "transitions", "operations", "states", "entries", "file_bytes"} {
		if definition.Limits[name] == 0 {
			return volumeWireSchema{}, fmt.Errorf("simulation volume wire %s limit is invalid", name)
		}
	}
	seen := make(map[string]bool)
	mapping := volumeWireMapping(definition, true)
	for _, layout := range definition.Layouts {
		if mapping.Types[layout.Name] == "" || seen[layout.Name] {
			return volumeWireSchema{}, fmt.Errorf("simulation volume wire layout %q is invalid", layout.Name)
		}
		seen[layout.Name] = true
		fields := make(map[string]bool)
		if len(layout.Fields) == 0 {
			return volumeWireSchema{}, fmt.Errorf("simulation volume wire layout %q is empty", layout.Name)
		}
		for _, field := range layout.Fields {
			if field.Name == "" || fields[field.Name] {
				return volumeWireSchema{}, fmt.Errorf("simulation volume wire field %q is invalid", field.Name)
			}
			fields[field.Name] = true
		}
		if _, err := volumeWireLayoutCodec(mapping, layout); err != nil {
			return volumeWireSchema{}, err
		}
	}
	if len(seen) != len(mapping.Types) {
		return volumeWireSchema{}, errors.New("simulation volume wire layouts are incomplete")
	}
	return definition, nil
}

func volumeWireMapping(definition volumeWireSchema, host bool) volumeWireConsumer {
	result := volumeWireConsumer{Package: "gomadfs", Prefix: "simulationVolume", Schema: definition, Host: host}
	result.Types = map[string]string{
		"Config": "simulationVolumeConfig", "Node": "simulationVolumeNodeConfig", "Mount": "VolumeConfig",
		"Record": "simulationVolumeRecord", "Transition": "simulationVolumeTransition", "Snapshot": "simulationVolumeSnapshot",
		"StateSnapshot": "simulationVolumeStateSnapshot", "SnapshotEntry": "SnapshotEntry", "ErrorBody": "simulationVolumeBridgeError",
		"FrontierBody": "CrashExploration", "CrashEntry": "CrashEntry", "CrashState": "CrashState", "EnumerationBody": "CrashEnumeration",
	}
	result.Functions = map[string]string{"Config": "SimulationVolumeConfigBody", "FrontierBody": "SimulationCrashExplorationBody", "SnapshotEntry": "SimulationSnapshotEntry", "EnumerationBody": "SimulationCrashEnumerationBody", "ErrorBody": "SimulationVolumeErrorBody"}
	if host {
		result.Package, result.Prefix = "wire", "runtimeVolume"
		result.Types = map[string]string{
			"Config": "runtimeVolumeWireConfig", "Node": "runtimeVolumeWireNode", "Mount": "runtimeVolumeWireMount",
			"Record": "VolumeRecord", "Transition": "VolumeTransition", "Snapshot": "VolumeSnapshot",
			"StateSnapshot": "VolumeStateSnapshot", "SnapshotEntry": "VolumeEntrySnapshot", "ErrorBody": "runtimeVolumeError",
			"FrontierBody": "VolumeCrashFrontier", "CrashEntry": "VolumeCrashEntry", "CrashState": "VolumeCrashState", "EnumerationBody": "VolumeCrashEnumeration",
		}
		result.Functions = map[string]string{"Config": "RuntimeVolumeConfigBody", "FrontierBody": "RuntimeVolumeFrontierBody", "EnumerationBody": "RuntimeVolumeEnumerationBody", "ErrorBody": "RuntimeVolumeErrorBody"}
	}
	for name := range result.Types {
		if result.Functions[name] == "" {
			result.Functions[name] = exportedIdentifier(result.Prefix) + name
		}
	}
	return result
}

func volumeWireLayoutCodec(consumer volumeWireConsumer, layout volumeWireLayout) (volumeWireCodec, error) {
	codec := volumeWireCodec{Type: consumer.Types[layout.Name], Function: consumer.Functions[layout.Name]}
	var encode, decode strings.Builder
	for index, field := range layout.Fields {
		name := field.Name
		if !consumer.Host && layout.Name == "EnumerationBody" && name == "Frontier" {
			name = "ChoiceExploration"
		}
		access := "value." + name
		cast := ""
		if consumer.Host {
			switch field.Name {
			case "Volume":
				cast = "VolumeID"
			case "Node", "Handle.Node":
				if layout.Name != "Node" {
					cast = "NodeID"
				}
			case "Kind":
				if layout.Name == "Transition" {
					cast = "VolumeTransitionKind"
				}
			case "Outcome":
				cast = "VolumeOutcome"
			case "Capacity":
				cast = "VolumeCrashCapacity"
			}
		} else if field.Name == "Capacity" {
			cast = "CrashCapacity"
		}
		limit := consumer.Schema.Limits[field.Limit]
		if field.Limit != "" && limit == 0 {
			return volumeWireCodec{}, fmt.Errorf("simulation volume wire field %s has unknown limit", field.Name)
		}
		read := func(expression string) string {
			if cast != "" {
				return cast + "(" + expression + ")"
			}
			return expression
		}
		switch field.Kind {
		case "u64", "u32", "i64", "bool", "string", "crashkind":
			method, input, output := "uint64", access, "decoder.uint64()"
			switch field.Kind {
			case "u32":
				input, output = "uint64("+access+")", "uint32(decoder.uint64())"
			case "i64":
				input, output = "uint64("+access+")", "int64(decoder.uint64())"
			case "bool":
				method, output = "boolean", "decoder.boolean()"
			case "string":
				method, input, output = "string", "string("+access+")", read("decoder.string()")
			case "crashkind":
				if consumer.Host {
					input, output = "runtimeVolumeCrashKind("+access+")", "volumeCrashKind(decoder.uint64())"
				} else {
					input, output = "uint64("+access+")", "Kind(decoder.uint64())"
				}
			}
			fmt.Fprintf(&encode, "encoder.%s(%s)\n", method, input)
			fmt.Fprintf(&decode, "%s = %s\n", access, output)
		case "bytes":
			fmt.Fprintf(&encode, "encoder.bytes(%s, %d)\n", access, limit)
			fmt.Fprintf(&decode, "%s = decoder.bytes(%d)\n", access, limit)
		case "record", "optional":
			function := consumer.Functions[field.Type]
			if function == "" {
				return volumeWireCodec{}, fmt.Errorf("simulation volume wire field %s has unknown type", field.Name)
			}
			if field.Kind == "record" {
				fmt.Fprintf(&encode, "encode%s(encoder, %s)\n", function, access)
				fmt.Fprintf(&decode, "%s = decode%s(decoder)\n", access, function)
			} else {
				fmt.Fprintf(&encode, "encoder.boolean(%s != nil)\nif %s != nil { encode%s(encoder, *%s) }\n", access, access, function, access)
				fmt.Fprintf(&decode, "if decoder.boolean() { decoded := decode%s(decoder); %s = &decoded }\n", function, access)
			}
		case "list", "integers", "strings":
			itemType, expression, statement := consumer.Types[field.Type], "", ""
			switch field.Kind {
			case "list":
				if itemType == "" {
					return volumeWireCodec{}, fmt.Errorf("simulation volume wire field %s has unknown type", field.Name)
				}
				expression = "decode" + consumer.Functions[field.Type] + "(decoder)"
				statement = "encode" + consumer.Functions[field.Type] + "(encoder, item)"
			case "integers":
				itemType, expression, statement = "uint64", "decoder.uint64()", "encoder.uint64(item)"
			case "strings":
				itemType, expression, statement = "string", "decoder.string()", "encoder.string(item)"
			}
			fmt.Fprintf(&encode, "encoder.uint64(uint64(len(%s)))\nfor _, item := range %s { %s }\n", access, access, statement)
			fmt.Fprintf(&decode, "count%d := decoder.count(%d)\n", index, limit)
			if field.Empty {
				fmt.Fprintf(&decode, "%s = make([]%s, 0, count%d)\n", access, itemType, index)
			}
			fmt.Fprintf(&decode, "for range count%d { if decoder.err != nil { break }; %s = append(%s, %s) }\n", index, access, access, expression)
		default:
			return volumeWireCodec{}, fmt.Errorf("simulation volume wire field %s has unknown kind %q", field.Name, field.Kind)
		}
	}
	codec.Encode, codec.Decode = encode.String(), decode.String()
	return codec, nil
}
