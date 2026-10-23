package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"
)

var transcriptHeader = []byte{'G', 'M', 'W', 'T', 1, 0, 0, 0}

const transcriptRecordHeader = 16
const transcriptMessageLimit = 64 << 10

type TranscriptRecord struct {
	ID            uint64
	Op            string
	Errno         uint16
	Unsupported   bool
	Message       string
	Input, Output json.RawMessage
	Count         uint64
	Now           uint64
}

type transcriptOperation struct {
	name          string
	input, output string
}

var transcriptOperations = [...]transcriptOperation{
	{},
	{"clock_time_get", "clock_id:u32 precision:u64", "timestamp:u64"},
	{"sched_yield", "", ""},
	{"args_sizes_get", "", ""},
	{"args_get", "", ""},
	{"environ_sizes_get", "", ""},
	{"environ_get", "", ""},
	{"random_get", "length:u32 offset:u64", ""},
	{"fd_write", "fd:u32 data_base64:bytes", "written:u32"},
	{"fd_pwrite", "fd:u32 data_base64:bytes offset:u64", "written:u32"},
	{"fd_read", "fd:u32 length:u32", "data_base64:bytes"},
	{"fd_pread", "fd:u32 length:u32 offset:u64", "data_base64:bytes"},
	{"fd_close", "fd:u32", ""},
	{"fd_sync", "fd:u32", ""},
	{"fd_filestat_get", "fd:u32", "dev:u64 ino:u64 filetype:u8 nlink:u64 size:u64 atim:u64 mtim:u64 ctim:u64"},
	{"fd_fdstat_get", "fd:u32", "filetype:u8 flags:u16 rights_base:u64 rights_inheriting:u64"},
	{"fd_prestat_get", "fd:u32", "name_length:u32"},
	{"fd_prestat_dir_name", "fd:u32 length:u32", "data_base64:bytes"},
	{"fd_fdstat_set_flags", "fd:u32 flags:u16", ""},
	{"fd_filestat_set_size", "fd:u32 size:u64", ""},
	{"fd_seek", "fd:u32 offset:i64 whence:u8", "offset:u64"},
	{"fd_readdir", "fd:u32 length:u32 cookie:u64", "data_base64:bytes"},
	{"path_create_directory", "fd:u32 path_base64:bytes", ""},
	{"path_remove_directory", "fd:u32 path_base64:bytes", ""},
	{"path_unlink_file", "fd:u32 path_base64:bytes", ""},
	{"path_filestat_get", "fd:u32 flags:u32 path_base64:bytes", "dev:u64 ino:u64 filetype:u8 nlink:u64 size:u64 atim:u64 mtim:u64 ctim:u64"},
	{"path_open", "fd:u32 dirflags:u32 path_base64:bytes oflags:u16 rights_base:u64 rights_inheriting:u64 fdflags:u16", "fd:u32"},
	{"path_readlink", "fd:u32 path_base64:bytes length:u32", "data_base64:bytes"},
	{"path_rename", "fd:u32 path_base64:bytes new_fd:u32 new_path_base64:bytes", ""},
	{"path_symlink", "old_path_base64:bytes fd:u32 path_base64:bytes", ""},
	{"sock_accept", "fd:u32 flags:u16", "fd:u32"},
	{"sock_shutdown", "fd:u32 how:u8", ""},
	{"poll_oneoff", "subscriptions:subscriptions", "events:events"},
	{},
	{"unsupported_operation", "", ""},
}

func transcriptTag(op string) (byte, error) {
	for tag, operation := range transcriptOperations {
		if operation.name != "" && operation.name == op {
			return byte(tag), nil
		}
	}
	return 34, nil
}

func appendTranscriptObject(data []byte, raw json.RawMessage, schema string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, invalid("operation JSON types")
	}
	names := strings.Fields(schema)
	if len(fields) != len(names) {
		return nil, invalid("operation fields")
	}
	for _, field := range names {
		name, kind, _ := strings.Cut(field, ":")
		value, ok := fields[name]
		if !ok || bytes.Equal(value, []byte("null")) {
			return nil, invalid("missing operation field " + name)
		}
		var err error
		data, err = appendTranscriptValue(data, value, kind)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func appendTranscriptValue(data []byte, raw json.RawMessage, kind string) ([]byte, error) {
	switch kind {
	case "bytes":
		var value []byte
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, invalid("operation JSON types")
		}
		if len(value) > bufferLimit {
			return nil, capacity("buffer-bytes")
		}
		data = binary.LittleEndian.AppendUint32(data, uint32(len(value)))
		return append(data, value...), nil
	case "subscriptions", "events":
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, invalid("operation JSON types")
		}
		if len(values) > 4096 {
			return nil, capacity("pending-events")
		}
		data = binary.LittleEndian.AppendUint32(data, uint32(len(values)))
		for _, value := range values {
			schema := "userdata:u64 errno:u16 type:u8 nbytes:u64 flags:u16"
			if kind == "subscriptions" {
				var header struct {
					Type uint8 `json:"type"`
				}
				if err := json.Unmarshal(value, &header); err != nil || header.Type > 2 {
					return nil, invalid("subscription type")
				}
				schema = "userdata:u64 type:u8 fd:u32"
				if header.Type == 0 {
					schema = "userdata:u64 type:u8 clock_id:u32 timeout:u64 precision:u64 flags:u16"
				}
			}
			var err error
			data, err = appendTranscriptObject(data, value, schema)
			if err != nil {
				return nil, err
			}
		}
		return data, nil
	case "i64":
		var value int64
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, invalid("operation JSON types")
		}
		return binary.LittleEndian.AppendUint64(data, uint64(value)), nil
	default:
		var value uint64
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, invalid("operation JSON types")
		}
		switch kind {
		case "u8":
			if value > math.MaxUint8 {
				return nil, invalid("operation JSON types")
			}
			return append(data, byte(value)), nil
		case "u16":
			if value > math.MaxUint16 {
				return nil, invalid("operation JSON types")
			}
			return binary.LittleEndian.AppendUint16(data, uint16(value)), nil
		case "u32":
			if value > math.MaxUint32 {
				return nil, invalid("operation JSON types")
			}
			return binary.LittleEndian.AppendUint32(data, uint32(value)), nil
		case "u64":
			return binary.LittleEndian.AppendUint64(data, value), nil
		}
	}
	return nil, invalid("transcript field")
}

func transcriptInput(call Call, entropyOffset uint64) (byte, []byte, error) {
	tag, err := transcriptTag(call.Op)
	if err != nil {
		return 0, nil, err
	}
	input := call.Input
	if tag == 34 {
		if len(call.Op) > 64<<10 || !utf8.ValidString(call.Op) || len(input) > frameLimit {
			return 0, nil, capacity("transcript-operation")
		}
		data := binary.LittleEndian.AppendUint32(nil, uint32(len(call.Op)))
		data = append(data, call.Op...)
		data = binary.LittleEndian.AppendUint32(data, uint32(len(input)))
		return tag, append(data, input...), nil
	}
	if tag == 7 {
		var value struct {
			Length uint32 `json:"length"`
		}
		if err := decodeInput(input, &value, "length"); err != nil {
			return 0, nil, err
		}
		return tag, binary.LittleEndian.AppendUint64(binary.LittleEndian.AppendUint32(nil, value.Length), entropyOffset), nil
	}
	data, err := appendTranscriptObject(nil, input, transcriptOperations[tag].input)
	return tag, data, err
}

func encodeTranscriptRecord(tag byte, id uint64, input []byte, reply Reply) ([]byte, error) {
	body := append([]byte(nil), input...)
	flags := byte(0)
	if reply.Unsupported {
		flags = 1
		if len(reply.Message) > transcriptMessageLimit || !utf8.ValidString(reply.Message) {
			return nil, invalid("transcript message")
		}
		body = binary.LittleEndian.AppendUint32(body, uint32(len(reply.Message)))
		body = append(body, reply.Message...)
	} else if reply.Errno == 0 {
		var err error
		// Argument and entropy bytes follow from the identity-bound request.
		if tag != 3 && tag != 4 && tag != 5 && tag != 6 && tag != 7 {
			body, err = appendTranscriptObject(body, reply.Output, transcriptOperations[tag].output)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(body) > frameLimit-transcriptRecordHeader {
		return nil, capacity("transcript-record")
	}
	data := []byte{tag, flags}
	data = binary.LittleEndian.AppendUint16(data, reply.Errno)
	data = binary.LittleEndian.AppendUint64(data, id)
	data = binary.LittleEndian.AppendUint32(data, uint32(len(body)))
	return append(data, body...), nil
}

type transcriptEncoder struct{ clock *transcriptClockRun }
type transcriptClockRun struct {
	id                    uint64
	clockID               uint32
	precision, now, count uint64
}

const transcriptClockRunBytes = 44

func (run *transcriptClockRun) folds(id uint64, clockID uint32, precision, now, step uint64) bool {
	return run != nil && run.clockID == clockID && run.precision == precision &&
		run.count < maxCooperativeCalls && run.id <= math.MaxUint64-run.count && id == run.id+run.count &&
		run.count <= math.MaxUint64/max(uint64(1), step) && run.now <= math.MaxUint64-run.count*step && now == run.now+run.count*step
}
func (run *transcriptClockRun) bytes() []byte {
	data := []byte{33, 0, 0, 0}
	data = binary.LittleEndian.AppendUint64(data, run.id)
	data = binary.LittleEndian.AppendUint32(data, 28)
	data = binary.LittleEndian.AppendUint32(data, run.clockID)
	data = binary.LittleEndian.AppendUint64(data, run.precision)
	data = binary.LittleEndian.AppendUint64(data, run.now)
	return binary.LittleEndian.AppendUint64(data, run.count)
}
func (e *Environment) appendClockRead(id uint64, clockID uint32, precision, now uint64) {
	run := e.transcriptWriter.clock
	if run.folds(id, clockID, precision, now, e.config.Clock.ReadStepNanos) {
		run.count++
		return
	}
	e.closeClockRun()
	if e.transcriptLength == 0 {
		e.appendTranscriptPart(transcriptHeader)
	}
	e.transcriptWriter.clock = &transcriptClockRun{id: id, clockID: clockID, precision: precision, now: now, count: 1}
	e.transcriptLength += transcriptClockRunBytes
}
func (e *Environment) closeClockRun() {
	if run := e.transcriptWriter.clock; run != nil {
		e.transcriptWriter.clock = nil
		e.transcriptLength -= transcriptClockRunBytes
		e.appendTranscriptPart(run.bytes())
	}
}

func (e *Environment) appendTranscript(data []byte) {
	e.closeClockRun()
	if e.transcriptLength == 0 {
		e.appendTranscriptPart(transcriptHeader)
	}
	e.appendTranscriptPart(data)
}

func (e *Environment) appendTranscriptPart(data []byte) {
	if e.digestOnly && !e.helperOwned && !e.splitOwned {
		e.transcriptHash.Write(data)
	} else {
		e.transcript = append(e.transcript, data...)
	}
	e.transcriptLength += uint64(len(data))
}

type transcriptReader struct{ data []byte }

func (r *transcriptReader) take(n int) ([]byte, error) {
	if n < 0 || n > len(r.data) {
		return nil, invalid("truncated transcript")
	}
	data := r.data[:n]
	r.data = r.data[n:]
	return data, nil
}
func (r *transcriptReader) number(width int) (uint64, error) {
	data, err := r.take(width)
	if err != nil {
		return 0, err
	}
	switch width {
	case 1:
		return uint64(data[0]), nil
	case 2:
		return uint64(binary.LittleEndian.Uint16(data)), nil
	case 4:
		return uint64(binary.LittleEndian.Uint32(data)), nil
	case 8:
		return binary.LittleEndian.Uint64(data), nil
	}
	return 0, invalid("transcript field")
}
func (r *transcriptReader) object(schema string) (map[string]any, error) {
	result := map[string]any{}
	for _, field := range strings.Fields(schema) {
		name, kind, _ := strings.Cut(field, ":")
		value, err := r.value(kind)
		if err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}
func (r *transcriptReader) value(kind string) (any, error) {
	switch kind {
	case "bytes":
		length, err := r.number(4)
		if err != nil {
			return nil, err
		}
		if length > bufferLimit {
			return nil, capacity("transcript-buffer")
		}
		data, err := r.take(int(length))
		if err != nil {
			return nil, err
		}
		return append([]byte{}, data...), nil
	case "subscriptions", "events":
		count, err := r.number(4)
		if err != nil {
			return nil, err
		}
		if count > 4096 {
			return nil, capacity("transcript-count")
		}
		result := make([]any, 0, count)
		for range count {
			schema := "userdata:u64 errno:u16 type:u8 nbytes:u64 flags:u16"
			if kind == "subscriptions" {
				if len(r.data) < 9 {
					return nil, invalid("truncated transcript")
				}
				schema = "userdata:u64 type:u8 fd:u32"
				switch r.data[8] {
				case 0:
					schema = "userdata:u64 type:u8 clock_id:u32 timeout:u64 precision:u64 flags:u16"
				case 1, 2:
				default:
					return nil, invalid("transcript subscription")
				}
			}
			value, err := r.object(schema)
			if err != nil {
				return nil, err
			}
			if kind == "events" && value["type"].(uint64) > 2 {
				return nil, invalid("transcript event")
			}
			result = append(result, value)
		}
		return result, nil
	case "u8":
		return r.number(1)
	case "u16":
		return r.number(2)
	case "u32":
		return r.number(4)
	case "u64":
		return r.number(8)
	case "i64":
		value, err := r.number(8)
		return int64(value), err
	}
	return nil, invalid("transcript field")
}

func DecodeTranscript(data []byte) ([]TranscriptRecord, error) {
	if len(data) == 0 {
		return []TranscriptRecord{}, nil
	}
	if len(data) > 1<<30 {
		return nil, capacity("transcript-bytes")
	}
	if len(data) < len(transcriptHeader) || string(data[:len(transcriptHeader)]) != string(transcriptHeader) {
		return nil, invalid("transcript header")
	}
	reader := transcriptReader{data[len(transcriptHeader):]}
	records := []TranscriptRecord{}
	var previous uint64
	for len(reader.data) != 0 {
		header, err := reader.take(transcriptRecordHeader)
		if err != nil {
			return nil, err
		}
		tag, flags := header[0], header[1]
		if tag == 0 || int(tag) >= len(transcriptOperations) || flags > 1 {
			return nil, invalid("transcript record")
		}
		length := binary.LittleEndian.Uint32(header[12:])
		if length > frameLimit-transcriptRecordHeader {
			return nil, capacity("transcript-record")
		}
		body, err := reader.take(int(length))
		if err != nil {
			return nil, err
		}
		record := TranscriptRecord{ID: binary.LittleEndian.Uint64(header[4:]), Op: transcriptOperations[tag].name, Errno: binary.LittleEndian.Uint16(header[2:]), Unsupported: flags == 1, Count: 1}
		if record.ID == 0 || record.ID <= previous {
			return nil, invalid("transcript sequence")
		}
		previous = record.ID
		payload := transcriptReader{body}
		if tag == 33 {
			if flags != 0 || record.Errno != 0 || len(body) != 28 {
				return nil, invalid("transcript clock run")
			}
			clockID := binary.LittleEndian.Uint32(body)
			count := binary.LittleEndian.Uint64(body[20:])
			if count > maxCooperativeCalls {
				return nil, capacity("transcript-count")
			}
			if clockID > 1 || count == 0 || binary.LittleEndian.Uint64(body[12:]) > math.MaxInt64 || record.ID > math.MaxUint64-count+1 || record.ID+count-1 > maxCooperativeCalls {
				return nil, invalid("transcript clock run")
			}
			record.Op, record.Count, record.Now = "clock_time_get", count, binary.LittleEndian.Uint64(body[12:])
			record.Input, err = json.Marshal(struct {
				ClockID   uint32 `json:"clock_id"`
				Precision uint64 `json:"precision"`
			}{clockID, binary.LittleEndian.Uint64(body[4:])})
			if err != nil {
				return nil, err
			}
			record.Output = json.RawMessage(`{}`)
			previous = record.ID + count - 1
			records = append(records, record)
			continue
		}
		if tag == 34 {
			if !record.Unsupported {
				return nil, invalid("transcript rejected operation")
			}
			nameLength, err := payload.number(4)
			if err != nil {
				return nil, err
			}
			if nameLength > 64<<10 {
				return nil, capacity("transcript-operation")
			}
			name, err := payload.take(int(nameLength))
			if err != nil {
				return nil, err
			}
			if !utf8.Valid(name) {
				return nil, invalid("transcript operation")
			}
			if known, _ := transcriptTag(string(name)); known != 34 {
				return nil, invalid("transcript known operation")
			}
			record.Op = string(name)
			rawLength, err := payload.number(4)
			if err != nil {
				return nil, err
			}
			if rawLength > frameLimit {
				return nil, capacity("transcript-input")
			}
			raw, err := payload.take(int(rawLength))
			if err != nil {
				return nil, err
			}
			if err := validateJSON(raw); err != nil {
				return nil, err
			}
			record.Input = bytes.Clone(raw)
		}
		input, err := payload.object(transcriptOperations[tag].input)
		if err != nil {
			return nil, err
		}
		if tag == 1 && (!record.Unsupported || input["clock_id"].(uint64) <= 1) {
			return nil, invalid("transcript clock record")
		}
		output := map[string]any{}
		if record.Unsupported {
			size, err := payload.number(4)
			if err != nil {
				return nil, err
			}
			if size > transcriptMessageLimit {
				return nil, capacity("transcript-message")
			}
			message, err := payload.take(int(size))
			if err != nil {
				return nil, err
			}
			if !utf8.Valid(message) {
				return nil, invalid("transcript message")
			}
			record.Message = string(message)
		} else if record.Errno == 0 && tag != 3 && tag != 4 && tag != 5 && tag != 6 && tag != 7 {
			output, err = payload.object(transcriptOperations[tag].output)
			if err != nil {
				return nil, err
			}
		}
		if len(payload.data) != 0 {
			return nil, invalid("trailing transcript data")
		}
		if tag != 34 {
			record.Input, err = json.Marshal(input)
			if err != nil {
				return nil, err
			}
		}
		record.Output, err = json.Marshal(output)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (e *Environment) responseReservation(tag byte, input []byte, call Call) (uint64, error) {
	output := uint64(0)
	switch tag {
	case 1:
		clockID, precision := binary.LittleEndian.Uint32(input), binary.LittleEndian.Uint64(input[4:])
		if clockID <= 1 {
			if e.transcriptWriter.clock.folds(call.ID, clockID, precision, e.now, e.config.Clock.ReadStepNanos) {
				return 0, nil
			}
			header := uint64(0)
			if e.transcriptLength == 0 {
				header = uint64(len(transcriptHeader))
			}
			return header + transcriptClockRunBytes, nil
		}
		output = 4 + uint64(len("CPU clocks outside stock WASI profile"))
	case 10, 11, 17, 21, 27:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(call.Input, &fields); err != nil {
			return 0, invalid("operation JSON types")
		}
		var length uint32
		if err := json.Unmarshal(fields["length"], &length); err != nil {
			return 0, invalid("operation length")
		}
		output = uint64(length) + 4
		if tag == 27 {
			output = max(output, 4+uint64(len("operation outside stock WASI profile: "+call.Op)))
		}
	case 8, 9, 16:
		output = 4
	case 18:
		output = 4 + uint64(len(descriptorFlagsUnsupported))
	case 26:
		output = max(4, 4+uint64(len(openFlagsUnsupported)))
	case 14, 25:
		output = 57
	case 15:
		output = 19
	case 20:
		output = 8
	case 28, 29, 30, 31:
		output = 4 + uint64(len("operation outside stock WASI profile: "+call.Op))
	case 32:
		var value struct {
			Subscriptions []json.RawMessage `json:"subscriptions"`
		}
		if err := json.Unmarshal(call.Input, &value); err != nil {
			return 0, invalid("operation JSON types")
		}
		output = max(4+uint64(len(value.Subscriptions))*23, 4+uint64(len("CPU poll clock outside stock WASI profile")))
	case 34:
		output = 4 + uint64(len("operation outside stock WASI profile: "+call.Op))
	}
	header := uint64(0)
	if e.transcriptLength == 0 {
		header = uint64(len(transcriptHeader))
	}
	return header + transcriptRecordHeader + uint64(len(input)) + output, nil
}
