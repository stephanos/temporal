package wasi

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"math"
	"slices"
)

const debugOutputLimit = 16 << 20

type DebugCoordinates struct {
	Instance         string `json:"instance,omitempty"`
	Incarnation      uint64 `json:"incarnation,omitempty"`
	Decision         uint64 `json:"decision"`
	Trace            uint64 `json:"trace"`
	VirtualTimeNanos uint64 `json:"virtual_time_nanos"`
}

type DebugFrame struct {
	PC       uint64 `json:"pc"`
	Line     uint32 `json:"line"`
	Function string `json:"function"`
	File     string `json:"file"`
}

type DebugStack struct {
	DebugCoordinates
	Goroutine uint64       `json:"goroutine"`
	Status    uint32       `json:"status"`
	Frames    []DebugFrame `json:"frames"`
	Truncated bool         `json:"truncated,omitempty"`
}

type DebugDump struct {
	DebugCoordinates
	ID     uint32 `json:"id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
	Data   []byte `json:"data_base64"`
}

type DebugField struct {
	Type        string `json:"type"`
	Int64       int64  `json:"int64,omitempty"`
	Uint64      uint64 `json:"uint64,omitempty"`
	Bool        bool   `json:"bool,omitempty"`
	Float64Bits uint64 `json:"float64_bits,omitempty"`
	Bytes       []byte `json:"bytes_base64,omitempty"`
}

type DebugProbe struct {
	DebugCoordinates
	ID     uint32       `json:"id"`
	Name   string       `json:"name,omitempty"`
	Fields []DebugField `json:"fields"`
}

type DebugCheck struct {
	DebugCoordinates
	ID     uint32 `json:"id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
}

type DebugDrop struct {
	DebugCoordinates
	Count uint64 `json:"count"`
}

type DebugOutput struct {
	Instance        string           `json:"instance,omitempty"`
	Incarnation     uint64           `json:"incarnation,omitempty"`
	Stacks          []DebugStack     `json:"stacks,omitempty"`
	StacksTruncated bool             `json:"stacks_truncated,omitempty"`
	Dumps           []DebugDump      `json:"dumps,omitempty"`
	Probes          []DebugProbe     `json:"probes,omitempty"`
	Checks          []DebugCheck     `json:"checks,omitempty"`
	Drops           []DebugDrop      `json:"drops,omitempty"`
	HeapProfile     []byte           `json:"heap_profile_base64,omitempty"`
	HeapCoordinates DebugCoordinates `json:"heap_coordinates"`
	HeapSampleRate  uint64           `json:"heap_sample_rate"`
	HeapTruncated   bool             `json:"heap_truncated,omitempty"`
}

type debugHeapSample struct {
	rate, allocatedObjects, freedObjects, allocatedBytes, freedBytes uint64
	pcs                                                              []uint64
	frames                                                           []DebugFrame
}

func (output *DebugOutput) SetTarget(instance string, incarnation uint64) {
	output.Instance, output.Incarnation = instance, incarnation
	set := func(coordinates *DebugCoordinates) {
		coordinates.Instance, coordinates.Incarnation = instance, incarnation
	}
	for index := range output.Stacks {
		set(&output.Stacks[index].DebugCoordinates)
	}
	for index := range output.Dumps {
		set(&output.Dumps[index].DebugCoordinates)
	}
	for index := range output.Probes {
		set(&output.Probes[index].DebugCoordinates)
	}
	for index := range output.Checks {
		set(&output.Checks[index].DebugCoordinates)
	}
	for index := range output.Drops {
		set(&output.Drops[index].DebugCoordinates)
	}
	set(&output.HeapCoordinates)
}

// DecodeDebugOutput decodes the bounded diagnostic stream, not replay evidence.
// The caller binds Instance/Incarnation from the verified enclosing guest stream.
func DecodeDebugOutput(raw []byte) (DebugOutput, error) {
	var output DebugOutput
	if len(raw) > debugOutputLimit {
		return output, invalid("debug output byte limit")
	}
	stacks := make(map[uint64]int)
	var samples []debugHeapSample
	var heapSeen, heapDone, stackSeen, stackDone bool
	var rateSeen bool
	for len(raw) > 0 {
		if len(raw) < 32 {
			return DebugOutput{}, invalid("debug envelope header")
		}
		op, size := binary.LittleEndian.Uint32(raw), binary.LittleEndian.Uint32(raw[4:])
		coordinates := DebugCoordinates{Decision: binary.LittleEndian.Uint64(raw[8:]), Trace: binary.LittleEndian.Uint64(raw[16:]), VirtualTimeNanos: binary.LittleEndian.Uint64(raw[24:])}
		raw = raw[32:]
		if size > 64*1024+16 || uint64(size) > uint64(len(raw)) {
			return DebugOutput{}, invalid("debug envelope length")
		}
		payload := raw[:size]
		raw = raw[size:]
		switch op {
		case 4:
			if err := output.decodeLibrary(payload, coordinates); err != nil {
				return DebugOutput{}, err
			}
		case 8, 11:
			identity, status, index, frame, err := decodeDebugFrame(payload)
			if err != nil {
				return DebugOutput{}, err
			}
			if op == 8 {
				stackSeen = true
				if stackDone {
					return DebugOutput{}, invalid("debug stack after end")
				}
				if index == math.MaxUint32 {
					stackDone = true
					output.StacksTruncated = status>>31 != 0
					continue
				}
				if identity == 0 {
					return DebugOutput{}, invalid("debug goroutine identity")
				}
				position, ok := stacks[identity]
				if !ok {
					if len(stacks) >= 4096 {
						return DebugOutput{}, invalid("debug goroutine capacity")
					}
					position = len(output.Stacks)
					stacks[identity] = position
					output.Stacks = append(output.Stacks, DebugStack{DebugCoordinates: coordinates, Goroutine: identity, Status: status &^ (1 << 31)})
				}
				stack := &output.Stacks[position]
				if index != uint32(len(stack.Frames)) || len(stack.Frames) >= 1024 || stack.DebugCoordinates != coordinates || stack.Status != status&^(1<<31) {
					return DebugOutput{}, invalid("debug stack frame order")
				}
				stack.Frames = append(stack.Frames, frame)
				stack.Truncated = stack.Truncated || status>>31 != 0
			} else {
				heapSeen = true
				output.HeapCoordinates = coordinates
				if heapDone {
					return DebugOutput{}, invalid("debug heap frame after end")
				}
				if index == math.MaxUint32 {
					heapDone = true
					output.HeapTruncated = output.HeapTruncated || status>>31 != 0
					continue
				}
				if identity >= uint64(len(samples)) {
					return DebugOutput{}, invalid("debug heap sample identity")
				}
				sample := &samples[identity]
				if index != uint32(len(sample.frames)) || len(sample.frames) >= 1024 {
					return DebugOutput{}, invalid("debug heap frame order")
				}
				sample.frames = append(sample.frames, frame)
				output.HeapTruncated = output.HeapTruncated || status>>31 != 0
			}
		case 9:
			heapSeen = true
			output.HeapCoordinates = coordinates
			if heapDone || len(payload) < 48 || len(samples) >= 4096 {
				return DebugOutput{}, invalid("debug heap sample header")
			}
			var numbers [5]uint64
			for index := range numbers {
				numbers[index] = binary.LittleEndian.Uint64(payload[8*index:])
				if numbers[index] > math.MaxInt64 {
					return DebugOutput{}, invalid("debug heap sample overflow")
				}
			}
			id, count := binary.LittleEndian.Uint32(payload[40:]), binary.LittleEndian.Uint32(payload[44:])
			if rateSeen && output.HeapSampleRate != numbers[0] {
				return DebugOutput{}, invalid("debug heap sampling rate changed")
			}
			output.HeapSampleRate = numbers[0]
			rateSeen = true
			if id == math.MaxUint32 {
				if len(payload) != 48 || count != 0 || numbers[1] != 0 || numbers[2] != 0 || numbers[3] != 0 || numbers[4] != 0 || len(samples) != 0 {
					return DebugOutput{}, invalid("debug heap rate metadata")
				}
				continue
			}
			if id != uint32(len(samples)) || count > 32 || uint64(len(payload)) != 48+8*uint64(count) || numbers[2] > numbers[1] || numbers[4] > numbers[3] || numbers[1] == 0 && numbers[3] != 0 {
				return DebugOutput{}, invalid("debug heap sample layout")
			}
			sample := debugHeapSample{rate: numbers[0], allocatedObjects: numbers[1], freedObjects: numbers[2], allocatedBytes: numbers[3], freedBytes: numbers[4]}
			for index := uint32(0); index < count; index++ {
				sample.pcs = append(sample.pcs, binary.LittleEndian.Uint64(payload[48+8*index:]))
			}
			samples = append(samples, sample)
		default:
			return DebugOutput{}, invalid("debug output operation")
		}
	}
	output.StacksTruncated = output.StacksTruncated || stackSeen && !stackDone
	output.HeapTruncated = output.HeapTruncated || heapSeen && !heapDone
	if heapSeen {
		var err error
		output.HeapProfile, err = encodeDebugHeap(samples, output.HeapSampleRate)
		if err != nil {
			return DebugOutput{}, err
		}
	}
	return output, nil
}

func decodeDebugFrame(b []byte) (uint64, uint32, uint32, DebugFrame, error) {
	if len(b) < 36 {
		return 0, 0, 0, DebugFrame{}, invalid("debug frame header")
	}
	functionSize, fileSize := binary.LittleEndian.Uint32(b[28:]), binary.LittleEndian.Uint32(b[32:])
	if functionSize > 2048 || fileSize > 2048 || uint64(len(b)) != 36+uint64(functionSize)+uint64(fileSize) {
		return 0, 0, 0, DebugFrame{}, invalid("debug frame strings")
	}
	id, status, index := binary.LittleEndian.Uint64(b), binary.LittleEndian.Uint32(b[8:]), binary.LittleEndian.Uint32(b[12:])
	frame := DebugFrame{PC: binary.LittleEndian.Uint64(b[16:]), Line: binary.LittleEndian.Uint32(b[24:]), Function: string(b[36 : 36+functionSize]), File: string(b[36+functionSize:])}
	if index == math.MaxUint32 && (id != 0 || status & ^uint32(1<<31) != 0 || frame.PC != 0 || frame.Line != 0 || functionSize != 0 || fileSize != 0) {
		return 0, 0, 0, DebugFrame{}, invalid("debug frame end marker")
	}
	return id, status, index, frame, nil
}

func (output *DebugOutput) decodeLibrary(b []byte, coordinates DebugCoordinates) error {
	if len(b) < 16 {
		return invalid("debug library header")
	}
	kind, id, status, size := binary.LittleEndian.Uint32(b), binary.LittleEndian.Uint32(b[4:]), binary.LittleEndian.Uint32(b[8:]), binary.LittleEndian.Uint32(b[12:])
	if id >= 256 || uint64(len(b)) != 16+uint64(size) {
		return invalid("debug library record length or ID")
	}
	b = b[16:]
	switch kind {
	case 1:
		if len(b) > 16*1024-16 || status > 1024 {
			return invalid("debug probe capacity")
		}
		probe := DebugProbe{DebugCoordinates: coordinates, ID: id}
		for index := uint32(0); index < status; index++ {
			if len(b) < 5 {
				return invalid("debug probe field header")
			}
			tag, n := b[0], binary.LittleEndian.Uint32(b[1:])
			b = b[5:]
			if uint64(n) > uint64(len(b)) {
				return invalid("debug probe field length")
			}
			value := b[:n]
			b = b[n:]
			var field DebugField
			switch tag {
			case 1, 2, 4:
				if n != 8 {
					return invalid("debug numeric probe field")
				}
				bits := binary.LittleEndian.Uint64(value)
				switch tag {
				case 1:
					field.Type = "int64"
					field.Int64 = int64(bits)
				case 2:
					field.Type = "uint64"
					field.Uint64 = bits
				case 4:
					field.Type = "float64"
					field.Float64Bits = bits
				}
			case 3:
				if n != 1 || value[0] > 1 {
					return invalid("debug boolean probe field")
				}
				field.Type = "bool"
				field.Bool = value[0] != 0
			case 5:
				field.Type = "string"
				field.Bytes = slices.Clone(value)
			default:
				return invalid("debug probe field type")
			}
			probe.Fields = append(probe.Fields, field)
		}
		if len(b) != 0 {
			return invalid("debug probe field count")
		}
		output.Probes = append(output.Probes, probe)
	case 2:
		if status > 2 || len(b) > 64*1024 {
			return invalid("debug dump status or length")
		}
		output.Dumps = append(output.Dumps, DebugDump{DebugCoordinates: coordinates, ID: id, Status: [3]string{"ok", "panic", "truncated"}[status], Data: slices.Clone(b)})
	case 3:
		if status > 2 || len(b) != 0 {
			return invalid("debug invariant result")
		}
		output.Checks = append(output.Checks, DebugCheck{DebugCoordinates: coordinates, ID: id, Status: [3]string{"true", "false", "panic"}[status]})
	case 4:
		if id != 0 || status != 0 || len(b) != 8 {
			return invalid("debug drop count")
		}
		count := binary.LittleEndian.Uint64(b)
		if len(output.Drops) > 0 && count < output.Drops[len(output.Drops)-1].Count {
			return invalid("debug drop count regressed")
		}
		output.Drops = append(output.Drops, DebugDrop{DebugCoordinates: coordinates, Count: count})
	default:
		return invalid("debug library record kind")
	}
	return nil
}

func debugProtoUint(b []byte, field uint64, value uint64) []byte {
	b = binary.AppendUvarint(b, field<<3)
	return binary.AppendUvarint(b, value)
}

func debugProtoBytes(b []byte, field uint64, value []byte) []byte {
	b = binary.AppendUvarint(b, field<<3|2)
	b = binary.AppendUvarint(b, uint64(len(value)))
	return append(b, value...)
}

func encodeDebugHeap(samples []debugHeapSample, rate uint64) ([]byte, error) {
	strings := []string{"", "alloc_objects", "count", "alloc_space", "bytes", "inuse_objects", "inuse_space", "space"}
	stringIndexes := make(map[string]uint64, len(strings))
	for position, value := range strings {
		stringIndexes[value] = uint64(position)
	}
	index := func(value string) uint64 {
		if position, ok := stringIndexes[value]; ok {
			return position
		}
		strings = append(strings, value)
		position := uint64(len(strings) - 1)
		stringIndexes[value] = position
		return position
	}
	var profile []byte
	for _, kind := range []uint64{1, 3, 5, 6} {
		unit := uint64(2)
		if kind == 3 || kind == 6 {
			unit = 4
		}
		profile = debugProtoBytes(profile, 1, debugProtoUint(debugProtoUint(nil, 1, kind), 2, unit))
	}
	profile = debugProtoBytes(profile, 11, debugProtoUint(debugProtoUint(nil, 1, 7), 2, 4))
	profile = debugProtoUint(profile, 14, 6)
	profile = debugProtoUint(profile, 12, rate)
	var next uint64
	for _, sample := range samples {
		var locations []byte
		frames := sample.frames
		if len(frames) == 0 {
			for _, pc := range sample.pcs {
				frames = append(frames, DebugFrame{PC: pc})
			}
		}
		for _, frame := range frames {
			next++
			function := debugProtoUint(nil, 1, next)
			function = debugProtoUint(function, 2, index(frame.Function))
			function = debugProtoUint(function, 3, index(frame.Function))
			function = debugProtoUint(function, 4, index(frame.File))
			profile = debugProtoBytes(profile, 5, function)
			line := debugProtoUint(debugProtoUint(nil, 1, next), 2, uint64(frame.Line))
			location := debugProtoUint(debugProtoUint(nil, 1, next), 3, frame.PC)
			location = debugProtoBytes(location, 4, line)
			profile = debugProtoBytes(profile, 4, location)
			locations = binary.AppendUvarint(locations, next)
		}
		allocatedObjects, allocatedBytes, err := debugScaleHeap(sample.allocatedObjects, sample.allocatedBytes, sample.rate)
		if err != nil {
			return nil, err
		}
		inuseObjects, inuseBytes, err := debugScaleHeap(sample.allocatedObjects-sample.freedObjects, sample.allocatedBytes-sample.freedBytes, sample.rate)
		if err != nil {
			return nil, err
		}
		var values []byte
		for _, value := range []uint64{allocatedObjects, allocatedBytes, inuseObjects, inuseBytes} {
			values = binary.AppendUvarint(values, value)
		}
		profile = debugProtoBytes(profile, 2, debugProtoBytes(debugProtoBytes(nil, 1, locations), 2, values))
	}
	for _, value := range strings {
		profile = debugProtoBytes(profile, 6, []byte(value))
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(profile); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func debugScaleHeap(objects, size, rate uint64) (uint64, uint64, error) {
	if objects == 0 || size == 0 {
		return 0, 0, nil
	}
	if rate <= 1 {
		return objects, size, nil
	}
	scale := 1 / -math.Expm1(-float64(size)/float64(objects)/float64(rate))
	count, bytes := float64(objects)*scale, float64(size)*scale
	if count >= math.MaxInt64 || bytes >= math.MaxInt64 {
		return 0, 0, invalid("debug scaled heap sample overflow")
	}
	return uint64(count), uint64(bytes), nil
}
