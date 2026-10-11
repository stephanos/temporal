package world

import (
	"bytes"
	"encoding/json"
	"io"
	"runtime"
	"strings"
	"testing"
)

type compressedScanCountingReader struct {
	reader io.Reader
	read   int
}

func (r *compressedScanCountingReader) Read(data []byte) (int, error) {
	n, err := r.reader.Read(data)
	r.read += n
	return n, err
}

func TestCompressedScannerRejectsOversizeStringsBeforeDecoderGrowth(t *testing.T) {
	for _, field := range []string{"kind", "adapter", "key", "equivalenceClass"} {
		t.Run(field, func(t *testing.T) {
			body := `{"config":{"limits":{"maxEvents":10,"maxPayloadBytes":16,"maxQueuedEvents":10,"maxRequests":10,"maxStringBytes":8,"maxTransitions":20},"seed":"0"},"requests":[{"request":{"resource":{"` + field + `":"` + strings.Repeat("x", 64<<10) + `"}}}]}`
			input := &compressedScanCountingReader{reader: strings.NewReader(body)}
			scanner := newCompressedJSONReader(input, CompressedSnapshotProfile().Limits)
			decoder := json.NewDecoder(scanner)
			if err := compressedJSONDelimiter(decoder, '{'); err != nil {
				t.Fatal(err)
			}
			if err := compressedJSONField(decoder, "config"); err != nil {
				t.Fatal(err)
			}
			var config Config
			if err := decoder.Decode(&config); err != nil {
				t.Fatal(err)
			}
			if err := scanner.configure(config.Limits); err != nil {
				t.Fatal(err)
			}
			if scanner.stringLimit > 6*1024 {
				t.Fatal("scanner lost bounded config parsing")
			}
			if err := compressedJSONField(decoder, "requests"); err != nil {
				t.Fatal(err)
			}
			var requests []RequestSnapshot
			allocated := compressedScanAllocation(func() {
				if err := decoder.Decode(&requests); err == nil {
					t.Fatal("oversized ordinary string admitted")
				}
			})
			if allocated > 16<<10 {
				t.Fatalf("decoder allocated %d bytes for oversized string", allocated)
			}
			if input.read > 8192 {
				t.Fatalf("decoder consumed %d bytes before rejection", input.read)
			}
			t.Logf("consumed=%d bytes, decoder allocated=%d bytes", input.read, allocated)
		})
	}
}

func TestCompressedScannerRejectsPayloadBeforeDecoderGrowth(t *testing.T) {
	for name, payloads := range map[string]string{
		"single payload":    `{"payload":"` + strings.Repeat("AAAA", 16<<10) + `"}`,
		"aggregate payload": `{"payload":"YWJj"},{"payload":"` + strings.Repeat("AAAA", 16<<10) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			body := `{"config":{"limits":{"maxEvents":10,"maxPayloadBytes":4,"maxQueuedEvents":10,"maxRequests":10,"maxStringBytes":8,"maxTransitions":20},"seed":"0"},"events":[` + payloads + `]}`
			input := &compressedScanCountingReader{reader: strings.NewReader(body)}
			scanner := newCompressedJSONReader(input, CompressedSnapshotProfile().Limits)
			decoder := json.NewDecoder(scanner)
			if err := compressedJSONDelimiter(decoder, '{'); err != nil {
				t.Fatal(err)
			}
			if err := compressedJSONField(decoder, "config"); err != nil {
				t.Fatal(err)
			}
			var config Config
			if err := decoder.Decode(&config); err != nil {
				t.Fatal(err)
			}
			if err := scanner.configure(config.Limits); err != nil {
				t.Fatal(err)
			}
			if err := compressedJSONField(decoder, "events"); err != nil {
				t.Fatal(err)
			}
			var events []struct {
				Payload []byte `json:"payload"`
			}
			allocated := compressedScanAllocation(func() {
				if err := decoder.Decode(&events); err == nil {
					t.Fatal("oversized payload admitted")
				}
			})
			if allocated > 16<<10 {
				t.Fatalf("decoder allocated %d bytes for oversized payload", allocated)
			}
			if input.read > 8192 {
				t.Fatalf("decoder consumed %d bytes before rejection", input.read)
			}
			t.Logf("consumed=%d bytes, decoder allocated=%d bytes", input.read, allocated)
		})
	}
}

func TestCompressedEnvelopeRejectsTooSmallGzipDeclarationBeforeAllocation(t *testing.T) {
	data := []byte(`{"compressed_bytes":1,"expanded_bytes":1,"expanded_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","gzip":"` + strings.Repeat("AAAA", 16<<10) + `","schema":"gomad3/world/compressed-snapshot/v1"}`)
	allocated := compressedScanAllocation(func() {
		if _, err := decodeCompressedEnvelope(data, CompressedSnapshotProfile()); err == nil {
			t.Fatal("oversized gzip token admitted")
		}
	})
	if allocated > 64<<10 {
		t.Fatalf("decoder allocated %d bytes before rejecting gzip declaration", allocated)
	}
	t.Logf("envelope input=%d bytes, allocated=%d bytes", len(data), allocated)
}

func compressedScanAllocation(run func()) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	run()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func TestCompressedScannerBase64RejectsInvalidLengthAndPadding(t *testing.T) {
	for _, token := range []string{"A", "AA", "AAA", "=AAA", "A=AA", "AA=A", "AAAA=", "AB==", "AAB=", "AA==AAAA", `AA\u003d=`} {
		t.Run(token, func(t *testing.T) {
			body := `{"payload":"` + token + `"}`
			scanner := newCompressedJSONReader(strings.NewReader(body), Limits{MaxStringBytes: 8, MaxPayloadBytes: 16})
			scanner.configBarrier, scanner.configComplete = false, true
			var value struct {
				Payload []byte `json:"payload"`
			}
			if err := json.NewDecoder(scanner).Decode(&value); err == nil {
				t.Fatal("invalid base64 admitted")
			}
		})
	}
	for _, payload := range [][]byte{{}, {0}, {0, 1}, {0, 1, 255}, {0, 1, 255, 255}} {
		body, err := json.Marshal(struct {
			Payload []byte `json:"payload"`
		}{Payload: payload})
		if err != nil {
			t.Fatal(err)
		}
		scanner := newCompressedJSONReader(bytes.NewReader(body), Limits{MaxStringBytes: 8, MaxPayloadBytes: 16})
		scanner.configBarrier, scanner.configComplete = false, true
		var value struct {
			Payload []byte `json:"payload"`
		}
		if err := json.NewDecoder(scanner).Decode(&value); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(value.Payload, payload) || scanner.payloadBytes != uint64(len(payload)) {
			t.Fatalf("base64 decoded %v, count=%d", value.Payload, scanner.payloadBytes)
		}
	}
}

func TestCompressedScannerRejectsNonStringPayloadBeforeDecoderGrowth(t *testing.T) {
	for _, value := range []string{"[" + strings.Repeat("0,", 32<<10) + "0]", "null"} {
		input := &compressedScanCountingReader{reader: strings.NewReader(`{"payload":` + value + `}`)}
		scanner := newCompressedJSONReader(input, Limits{MaxStringBytes: 8, MaxPayloadBytes: 4})
		scanner.configBarrier, scanner.configComplete = false, true
		var decoded struct {
			Payload []byte `json:"payload"`
		}
		if err := json.NewDecoder(scanner).Decode(&decoded); err == nil {
			t.Fatal("non-string payload admitted")
		}
		if input.read > 4096 {
			t.Fatalf("consumed %d bytes before rejecting non-string payload", input.read)
		}
	}
}

func TestCompressedScannerRejectsMalformedConfigBeforeDecoderGrowth(t *testing.T) {
	large := "[" + strings.Repeat("0,", 32<<10) + "0]"
	for _, body := range []string{
		`{"config":{"limits":` + large + `}}`,
		`{"config":{"seed":` + large + `}}`,
		`{"config":{"unexpected":` + large + `}}`,
		`{"config":{"limits":{"maxEvents":` + large + `}}}`,
	} {
		input := &compressedScanCountingReader{reader: strings.NewReader(body)}
		scanner := newCompressedJSONReader(input, CompressedSnapshotProfile().Limits)
		var decoded struct {
			Config Config `json:"config"`
		}
		if err := json.NewDecoder(scanner).Decode(&decoded); err == nil {
			t.Fatal("malformed config admitted")
		}
		if input.read > 4096 {
			t.Fatalf("consumed %d bytes before rejecting malformed config", input.read)
		}
	}
}

func TestCompressedScannerRejectsMalformedEnvelopeHeaderBeforeDecoderGrowth(t *testing.T) {
	for _, field := range []string{"compressed_bytes", "expanded_bytes", "expanded_sha256", "schema"} {
		for _, value := range []string{"[" + strings.Repeat("0,", 32<<10) + "0]", `{"unexpected":[` + strings.Repeat("0,", 32<<10) + `0]}`} {
			t.Run(field+value[:1], func(t *testing.T) {
				body := `{"` + field + `":` + value + `}`
				input := &compressedScanCountingReader{reader: strings.NewReader(body)}
				scanner := newCompressedEnvelopeJSONReader(input, CompressedSnapshotProfile())
				decoder := json.NewDecoder(scanner)
				var envelope compressedSnapshotEnvelope
				allocated := compressedScanAllocation(func() {
					if err := decoder.Decode(&envelope); err == nil {
						t.Fatal("malformed envelope header admitted")
					}
				})
				if input.read > 4096 {
					t.Fatalf("consumed %d bytes before rejecting header", input.read)
				}
				if allocated > 16<<10 {
					t.Fatalf("decoder allocated %d bytes before rejecting header", allocated)
				}
				t.Logf("consumed=%d bytes, allocated=%d bytes", input.read, allocated)
			})
		}
	}
}
