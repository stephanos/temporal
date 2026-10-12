package world

import (
	"encoding/json"
	"io"
)

type compressedScanFrame struct {
	kind         byte
	key          string
	previousKey  string
	context      string
	expectKey    bool
	valuePending bool
}

// Bytes reach the JSON decoder only after their token bounds have been checked.
type compressedJSONReader struct {
	source                                      io.Reader
	input                                       [4096]byte
	start, end                                  int
	sourceErr                                   error
	limits                                      Limits
	frames                                      [64]compressedScanFrame
	depth                                       int
	inString, escaped, keyString, payloadString bool
	stringBytes, stringLimit                    uint64
	key                                         [6*1024 + 2]byte
	keyBytes                                    int
	scalarBytes                                 uint64
	payloadBytes                                uint64
	quartet, padding, lastSextet                byte
	payloadEnded                                bool
	configBarrier, configComplete, paused       bool
	envelope                                    bool
}

func newCompressedJSONReader(source io.Reader, limits Limits) *compressedJSONReader {
	return &compressedJSONReader{source: source, limits: limits, configBarrier: true}
}

func newCompressedEnvelopeJSONReader(source io.Reader, limits ProfileLimits) *compressedJSONReader {
	reader := newCompressedJSONReader(source, limits.Limits)
	reader.configBarrier, reader.configComplete, reader.envelope = false, true, true
	reader.limits.MaxPayloadBytes = limits.MaxCompressedBytes
	return reader
}

func (reader *compressedJSONReader) configure(limits Limits) error {
	if !reader.paused || !reader.configComplete {
		return invalidSnapshot("compressed.scan.config")
	}
	reader.limits = limits
	reader.paused = false
	return nil
}

func (reader *compressedJSONReader) Read(output []byte) (int, error) {
	if len(output) == 0 {
		return 0, nil
	}
	if reader.paused {
		return 0, invalidSnapshot("compressed.scan.configuration-required")
	}
	var written int
	for written < len(output) {
		if reader.start == reader.end {
			if reader.sourceErr != nil {
				return written, reader.sourceErr
			}
			//gomad:dynamic reviewed 2026-10-12 compressed model stream wraps memory readers writers and SHA256 only
			n, err := reader.source.Read(reader.input[:])
			reader.start, reader.end, reader.sourceErr = 0, n, err
			if n == 0 {
				if err != nil {
					return written, err
				}
				return written, io.ErrNoProgress
			}
		}
		character := reader.input[reader.start]
		if err := reader.scan(character); err != nil {
			return written, err
		}
		reader.start++
		output[written] = character
		written++
		if reader.paused {
			return written, nil
		}
	}
	return written, nil
}

func (reader *compressedJSONReader) scan(character byte) error {
	if reader.inString {
		return reader.scanString(character)
	}
	if reader.depth > 0 && reader.frames[reader.depth-1].valuePending {
		frame := &reader.frames[reader.depth-1]
		if (frame.key == "payload" || frame.key == "gzip") && character != '"' {
			return invalidSnapshot("compressed.scan.base64.token")
		}
		if reader.envelope && reader.depth == 1 && !compressedEnvelopeValue(frame.key, character) {
			return invalidSnapshot("compressed.scan.envelope.value")
		}
		if reader.configBarrier && !reader.configComplete && !compressedConfigValue(frame.context, frame.key, character) {
			return invalidSnapshot("compressed.scan.config.value")
		}
		frame.valuePending = false
	}
	switch character {
	case '"':
		reader.scalarBytes = 0
		reader.inString = true
		reader.stringBytes, reader.keyBytes = 0, 0
		reader.escaped, reader.payloadEnded = false, false
		reader.quartet, reader.padding, reader.lastSextet = 0, 0, 0
		reader.keyString = reader.depth > 0 && reader.frames[reader.depth-1].kind == '{' && reader.frames[reader.depth-1].expectKey
		field, context := reader.valueContext()
		reader.payloadString = !reader.keyString && reader.configComplete && (field == "payload" || field == "gzip")
		reader.stringLimit = uint64(reader.limits.MaxStringBytes) * 6
		if reader.keyString {
			reader.stringLimit = 6 * 1024
			reader.key[0] = '"'
			reader.keyBytes = 1
		} else if reader.payloadString {
			remaining := reader.limits.MaxPayloadBytes - reader.payloadBytes
			reader.stringLimit = (remaining + 2) / 3 * 4
		} else if compressedMetadataString(field, context) {
			reader.stringLimit = 64 * 6
		}
	case '{', '[':
		reader.scalarBytes = 0
		if reader.depth == len(reader.frames) {
			return invalidSnapshot("compressed.scan.depth")
		}
		field, context := reader.valueContext()
		if field != "" {
			context = field
		}
		reader.frames[reader.depth] = compressedScanFrame{kind: character, context: context, expectKey: character == '{'}
		reader.depth++
	case '}', ']':
		reader.scalarBytes = 0
		if reader.depth == 0 {
			return invalidSnapshot("compressed.scan.close")
		}
		frame := reader.frames[reader.depth-1]
		if character == '}' && frame.kind != '{' || character == ']' && frame.kind != '[' {
			return invalidSnapshot("compressed.scan.close")
		}
		reader.depth--
		reader.frames[reader.depth] = compressedScanFrame{}
		if reader.configBarrier && reader.depth == 1 && frame.context == "config" {
			reader.configComplete, reader.paused = true, true
		}
	case ',':
		reader.scalarBytes = 0
		if reader.depth > 0 && reader.frames[reader.depth-1].kind == '{' {
			reader.frames[reader.depth-1].expectKey = true
			reader.frames[reader.depth-1].key = ""
		}
	case ':':
		reader.scalarBytes = 0
		if reader.depth > 0 {
			reader.frames[reader.depth-1].valuePending = true
		}
	case ' ', '\n', '\r', '\t':
		return invalidSnapshot("compressed.scan.whitespace")
	default:
		reader.scalarBytes++
		if reader.scalarBytes > 32 {
			return invalidSnapshot("compressed.scan.scalar")
		}
	}
	return nil
}

func (reader *compressedJSONReader) valueContext() (string, string) {
	if reader.depth == 0 {
		return "", ""
	}
	frame := reader.frames[reader.depth-1]
	return frame.key, frame.context
}

func compressedMetadataString(field, context string) bool {
	switch field {
	case "id", "eventId", "requestId", "nextEventId", "nextRequestId", "nextTransition", "sequence", "now", "at", "before", "after", "seed",
		"digest", "previousDigest", "transcriptDigest", "stateDigest", "expanded_sha256", "schema", "state", "status":
		return true
	case "kind":
		return context == "transitions" || context == "result"
	}
	// Array members of blocked carry decimal request identities.
	return context == "blocked"
}

func (reader *compressedJSONReader) scanString(character byte) error {
	if character == '"' && !reader.escaped {
		if reader.payloadString && reader.quartet != 0 {
			return invalidSnapshot("compressed.scan.base64.length")
		}
		if reader.keyString {
			reader.key[reader.keyBytes] = character
			reader.keyBytes++
			var key string
			if err := json.Unmarshal(reader.key[:reader.keyBytes], &key); err != nil {
				return invalidSnapshot("compressed.scan.key")
			}
			frame := &reader.frames[reader.depth-1]
			if key <= frame.previousKey {
				return invalidSnapshot("compressed.scan.keys.order")
			}
			if reader.configBarrier && !reader.configComplete && !compressedConfigKey(frame.context, key) {
				return invalidSnapshot("compressed.scan.config.key")
			}
			frame.previousKey = key
			frame.key, frame.expectKey = key, false
		}
		reader.inString = false
		return nil
	}
	reader.stringBytes++
	if reader.stringBytes > reader.stringLimit {
		return invalidSnapshot("compressed.scan.string")
	}
	if reader.payloadString {
		return reader.scanBase64(character)
	}
	if reader.keyString {
		reader.key[reader.keyBytes] = character
		reader.keyBytes++
	}
	if reader.escaped {
		reader.escaped = false
	} else if character == '\\' {
		reader.escaped = true
	}
	return nil
}

func compressedConfigKey(context, key string) bool {
	switch context {
	case "":
		return key == "config"
	case "config":
		return key == "limits" || key == "seed"
	case "limits":
		switch key {
		case "maxEvents", "maxPayloadBytes", "maxQueuedEvents", "maxRequests", "maxStringBytes", "maxTransitions":
			return true
		}
	}
	return false
}

func compressedConfigValue(context, key string, character byte) bool {
	if context == "" && key == "config" || context == "config" && key == "limits" {
		return character == '{'
	}
	if context == "config" && key == "seed" {
		return character == '"'
	}
	return context == "limits" && character >= '0' && character <= '9'
}

func compressedEnvelopeValue(key string, character byte) bool {
	switch key {
	case "compressed_bytes", "expanded_bytes":
		return character >= '0' && character <= '9'
	case "expanded_sha256", "gzip", "schema":
		return character == '"'
	}
	return false
}

func (reader *compressedJSONReader) scanBase64(character byte) error {
	if reader.payloadEnded {
		return invalidSnapshot("compressed.scan.base64.padding")
	}
	sextet, valid := compressedBase64Sextet(character)
	if character == '=' {
		if reader.quartet < 2 || reader.quartet == 2 && reader.lastSextet&15 != 0 || reader.quartet == 3 && reader.padding == 0 && reader.lastSextet&3 != 0 {
			return invalidSnapshot("compressed.scan.base64.padding")
		}
		reader.padding++
	} else if !valid || reader.padding != 0 {
		return invalidSnapshot("compressed.scan.base64.character")
	}
	reader.lastSextet = sextet
	reader.quartet++
	if reader.quartet == 4 {
		decoded := uint64(3 - reader.padding)
		if decoded > reader.limits.MaxPayloadBytes-reader.payloadBytes {
			return invalidSnapshot("compressed.scan.payloadBytes")
		}
		reader.payloadBytes += decoded
		reader.payloadEnded = reader.padding != 0
		reader.quartet = 0
	}
	return nil
}

func compressedBase64Sextet(character byte) (byte, bool) {
	switch {
	case character >= 'A' && character <= 'Z':
		return character - 'A', true
	case character >= 'a' && character <= 'z':
		return character - 'a' + 26, true
	case character >= '0' && character <= '9':
		return character - '0' + 52, true
	case character == '+':
		return 62, true
	case character == '/':
		return 63, true
	}
	return 0, false
}

func scanCompressedGzipToken(data []byte, offset int64, compressedBytes uint64) error {
	start := int(offset)
	if start+2 > len(data) || data[start] != ':' || data[start+1] != '"' {
		return invalidSnapshot("compressed.envelope.gzip.token")
	}
	start += 2
	expected := (compressedBytes + 2) / 3 * 4
	if expected > uint64(len(data)-start) {
		return invalidSnapshot("compressed.envelope.gzip.length")
	}
	end := start + int(expected)
	if end >= len(data) || data[end] != '"' {
		return invalidSnapshot("compressed.envelope.gzip.length")
	}
	scanner := compressedJSONReader{limits: Limits{MaxPayloadBytes: compressedBytes}}
	for _, character := range data[start:end] {
		if err := scanner.scanBase64(character); err != nil {
			return err
		}
	}
	if scanner.quartet != 0 || scanner.payloadBytes != compressedBytes {
		return invalidSnapshot("compressed.envelope.gzip.length")
	}
	return nil
}
