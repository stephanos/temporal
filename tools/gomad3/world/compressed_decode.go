package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

func DecodeCompressedSnapshot(data []byte, limits ProfileLimits) (CompressedSnapshotSummary, error) {
	if err := validateCompressedProfile(limits); err != nil {
		return CompressedSnapshotSummary{}, err
	}
	maximumEnvelope := (limits.MaxCompressedBytes+2)/3*4 + 1024
	if uint64(len(data)) > maximumEnvelope {
		return CompressedSnapshotSummary{}, invalidSnapshot("compressed.envelope.bytes")
	}
	envelope, err := decodeCompressedEnvelope(data, limits)
	if err != nil {
		return CompressedSnapshotSummary{}, err
	}
	if envelope.Schema != CompressedSnapshotSchema || envelope.ExpandedBytes == 0 || envelope.ExpandedBytes > limits.MaxExpandedBytes ||
		envelope.CompressedBytes == 0 || envelope.CompressedBytes > limits.MaxCompressedBytes ||
		envelope.CompressedBytes != uint64(len(envelope.Gzip)) || !validDigest(envelope.ExpandedSHA256) {
		return CompressedSnapshotSummary{}, invalidSnapshot("compressed.envelope.identity")
	}
	canonicalDigest, canonicalSize, err := compressedCanonicalDigest(envelope, maximumEnvelope)
	if err != nil {
		return CompressedSnapshotSummary{}, err
	}
	actualDigest := sha256.Sum256(data)
	if canonicalSize != uint64(len(data)) || canonicalDigest != Digest(hex.EncodeToString(actualDigest[:])) {
		return CompressedSnapshotSummary{}, invalidSnapshot("compressed.envelope.canonical")
	}
	if err := compressedGzipPass(envelope, func(reader io.Reader) error {
		return preflightCompressedSnapshot(reader, limits, envelope.ExpandedBytes)
	}); err != nil {
		return CompressedSnapshotSummary{}, err
	}
	var snapshot Snapshot
	var requests, events, transitions uint64
	if err := compressedGzipPass(envelope, func(reader io.Reader) error {
		var decodeErr error
		snapshot, requests, events, transitions, decodeErr = decodeCompressedExpanded(reader, limits)
		return decodeErr
	}); err != nil {
		return CompressedSnapshotSummary{}, err
	}
	canonicalDigest, canonicalSize, err = compressedCanonicalDigest(snapshot, limits.MaxExpandedBytes)
	if err != nil {
		return CompressedSnapshotSummary{}, err
	}
	if canonicalDigest != envelope.ExpandedSHA256 || canonicalSize != envelope.ExpandedBytes {
		return CompressedSnapshotSummary{}, invalidSnapshot("compressed.snapshot.canonical-or-state")
	}
	return CompressedSnapshotSummary{
		Config: snapshot.Config, Now: snapshot.Now, RequestCount: requests, EventCount: events, TransitionCount: transitions,
		PayloadBytes: snapshot.PayloadBytes, TranscriptDigest: snapshot.TranscriptDigest, StateDigest: snapshot.StateDigest,
		Replay: snapshot.Replay, ExpandedBytes: envelope.ExpandedBytes, CompressedBytes: envelope.CompressedBytes, ExpandedSHA256: envelope.ExpandedSHA256,
	}, nil
}

func decodeCompressedEnvelope(data []byte, limits ProfileLimits) (compressedSnapshotEnvelope, error) {
	scanner := newCompressedEnvelopeJSONReader(bytes.NewReader(data), limits)
	//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
	decoder := json.NewDecoder(scanner)
	decoder.DisallowUnknownFields()
	if err := compressedJSONDelimiter(decoder, '{'); err != nil {
		return compressedSnapshotEnvelope{}, err
	}
	var envelope compressedSnapshotEnvelope
	for _, field := range []struct {
		name  string
		value any
	}{
		{"compressed_bytes", &envelope.CompressedBytes}, {"expanded_bytes", &envelope.ExpandedBytes},
		{"expanded_sha256", &envelope.ExpandedSHA256}, {"gzip", &envelope.Gzip}, {"schema", &envelope.Schema},
	} {
		if err := compressedJSONField(decoder, field.name); err != nil {
			return compressedSnapshotEnvelope{}, err
		}
		if field.name == "gzip" {
			if err := scanCompressedGzipToken(data, decoder.InputOffset(), envelope.CompressedBytes); err != nil {
				return compressedSnapshotEnvelope{}, err
			}
		}
		//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
		if err := decoder.Decode(field.value); err != nil {
			return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope." + field.name)
		}
		if field.name == "compressed_bytes" && (envelope.CompressedBytes == 0 || envelope.CompressedBytes > limits.MaxCompressedBytes) {
			return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope.compressed_bytes")
		}
		if field.name == "expanded_bytes" && (envelope.ExpandedBytes == 0 || envelope.ExpandedBytes > limits.MaxExpandedBytes) {
			return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope.expanded_bytes")
		}
	}
	if err := compressedJSONDelimiter(decoder, '}'); err != nil {
		return compressedSnapshotEnvelope{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return compressedSnapshotEnvelope{}, err
	}
	return envelope, nil
}

func compressedJSONDelimiter(decoder *json.Decoder, expected json.Delim) error {
	actual, err := decoder.Token()
	if err != nil || actual != expected {
		return invalidSnapshot(fmt.Sprintf("compressed.json.delimiter.%c", expected))
	}
	return nil
}

func compressedJSONField(decoder *json.Decoder, expected string) error {
	actual, err := decoder.Token()
	if err != nil || actual != expected {
		return invalidSnapshot("compressed.json.field." + expected)
	}
	return nil
}

func preflightCompressedSnapshot(reader io.Reader, limits ProfileLimits, budget uint64) error {
	scanner := newCompressedJSONReader(reader, limits.Limits)
	//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
	decoder := json.NewDecoder(scanner)
	decoder.UseNumber()
	if err := compressedJSONDelimiter(decoder, '{'); err != nil {
		return err
	}
	if err := compressedJSONField(decoder, "config"); err != nil {
		return err
	}
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return err
	}
	if err := validateCompressedConfig(config, limits); err != nil {
		return err
	}
	if err := scanner.configure(config.Limits); err != nil {
		return err
	}
	for _, name := range compressedSnapshotFields {
		if err := compressedJSONField(decoder, name); err != nil {
			return err
		}
		var countLimit uint64
		switch name {
		case "events":
			countLimit = config.Limits.MaxEvents
		case "requests":
			countLimit = config.Limits.MaxRequests
		case "transitions":
			countLimit = config.Limits.MaxTransitions
		}
		nested := max(config.Limits.MaxRequests, config.Limits.MaxEvents)
		if countLimit > 0 {
			if err := consumeBoundedJSONArray(decoder, countLimit, nested, &budget, 1); err != nil {
				return err
			}
		} else if err := consumeBoundedJSONValue(decoder, nested, &budget, 1); err != nil {
			return err
		}
	}
	if err := compressedJSONDelimiter(decoder, '}'); err != nil {
		return err
	}
	return requireJSONEOF(decoder)
}

var compressedSnapshotFields = [...]string{
	"events", "nextEventId", "nextRequestId", "nextTransition", "now", "payloadBytes", "replay",
	"requests", "schemaVersion", "stateDigest", "transcriptDigest", "transitions",
}

func decodeCompressedExpanded(reader io.Reader, limits ProfileLimits) (Snapshot, uint64, uint64, uint64, error) {
	scanner := newCompressedJSONReader(reader, limits.Limits)
	//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
	decoder := json.NewDecoder(scanner)
	decoder.DisallowUnknownFields()
	if err := compressedJSONDelimiter(decoder, '{'); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	if err := compressedJSONField(decoder, "config"); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	var declared Snapshot
	if err := decoder.Decode(&declared.Config); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	if err := validateCompressedConfig(declared.Config, limits); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	if err := scanner.configure(declared.Config.Limits); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	model := newModel(declared.Config)
	var requests, events, transitions, payloadBytes uint64
	addPayload := func(size uint64) error {
		if size > declared.Config.Limits.MaxPayloadBytes-payloadBytes {
			return invalidSnapshot("compressed.payloadBytes")
		}
		payloadBytes += size
		return nil
	}
	for _, name := range compressedSnapshotFields {
		if err := compressedJSONField(decoder, name); err != nil {
			return Snapshot{}, 0, 0, 0, err
		}
		var destination any
		var err error
		switch name {
		case "events":
			var previous EventID
			events, err = decodeCompressedArray(decoder, declared.Config.Limits.MaxEvents, func() error {
				var event EventSnapshot
				if err := decoder.Decode(&event); err != nil {
					return err
				}
				if event.ID == 0 || event.ID <= previous {
					return invalidSnapshot("events.order")
				}
				previous = event.ID
				if err := validateReadiness(declared.Config.Limits, event.Readiness); err != nil {
					return err
				}
				//gomad:dynamic reviewed 2026-10-12 local model closure or reviewed caller-supplied callback performs memory-only work
				return addPayload(uint64(len(event.Readiness.Payload)))
			})
		case "requests":
			var previous RequestID
			requests, err = decodeCompressedArray(decoder, declared.Config.Limits.MaxRequests, func() error {
				var request RequestSnapshot
				if err := decoder.Decode(&request); err != nil {
					return err
				}
				if request.ID == 0 || request.ID <= previous {
					return invalidSnapshot("requests.order")
				}
				previous = request.ID
				if err := validateRequest(declared.Config.Limits, request.Request); err != nil {
					return err
				}
				//gomad:dynamic reviewed 2026-10-12 local model closure or reviewed caller-supplied callback performs memory-only work
				return addPayload(uint64(len(request.Request.Payload)))
			})
		case "transitions":
			transitions, err = decodeCompressedArray(decoder, declared.Config.Limits.MaxTransitions, func() error {
				var expected Transition
				if err := decoder.Decode(&expected); err != nil {
					return err
				}
				if err := replaySnapshotTransition(model, expected); err != nil {
					return err
				}
				size, ok := checkedTransitionPayloadSize(expected)
				if !ok {
					return invalidSnapshot("payloadBytes")
				}
				//gomad:dynamic reviewed 2026-10-12 local model closure or reviewed caller-supplied callback performs memory-only work
				if err := addPayload(size); err != nil {
					return err
				}
				actual := model.history[len(model.history)-1]
				if !bytes.Equal(transitionBytes(actual, true), transitionBytes(expected, true)) {
					return invalidSnapshot("transitions.result")
				}
				return nil
			})
		case "nextEventId":
			destination = &declared.NextEventID
		case "nextRequestId":
			destination = &declared.NextRequestID
		case "nextTransition":
			destination = &declared.NextTransition
		case "now":
			destination = &declared.Now
		case "payloadBytes":
			destination = &declared.PayloadBytes
		case "replay":
			destination = &declared.Replay
		case "schemaVersion":
			destination = &declared.SchemaVersion
		case "stateDigest":
			destination = &declared.StateDigest
		case "transcriptDigest":
			destination = &declared.TranscriptDigest
		}
		if err != nil {
			return Snapshot{}, 0, 0, 0, err
		}
		if destination != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
			if err := decoder.Decode(destination); err != nil {
				return Snapshot{}, 0, 0, 0, err
			}
		}
	}
	if err := compressedJSONDelimiter(decoder, '}'); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Snapshot{}, 0, 0, 0, err
	}
	if declared.SchemaVersion != SchemaVersion || declared.Replay.Cursor > declared.Replay.Expected || declared.PayloadBytes != payloadBytes {
		return Snapshot{}, 0, 0, 0, invalidSnapshot("compressed.metadata")
	}
	actual := ownedSnapshot(model)
	actual.Replay = declared.Replay
	if declared.NextEventID != actual.NextEventID || declared.NextRequestID != actual.NextRequestID || declared.NextTransition != actual.NextTransition ||
		declared.Now != actual.Now || declared.PayloadBytes != actual.PayloadBytes || declared.StateDigest != actual.StateDigest || declared.TranscriptDigest != actual.TranscriptDigest ||
		requests != uint64(len(actual.Requests)) || events != uint64(len(actual.Events)) || transitions != uint64(len(actual.Transitions)) {
		return Snapshot{}, 0, 0, 0, invalidSnapshot("compressed.state")
	}
	return actual, requests, events, transitions, nil
}

func decodeCompressedArray(decoder *json.Decoder, limit uint64, decode func() error) (uint64, error) {
	if err := compressedJSONDelimiter(decoder, '['); err != nil {
		return 0, err
	}
	var count uint64
	for decoder.More() {
		if count >= limit {
			return 0, invalidSnapshot("compressed.array.count")
		}
		//gomad:dynamic reviewed 2026-10-12 local model closure or reviewed caller-supplied callback performs memory-only work
		if err := decode(); err != nil {
			return 0, err
		}
		count++
	}
	if err := compressedJSONDelimiter(decoder, ']'); err != nil {
		return 0, err
	}
	return count, nil
}
