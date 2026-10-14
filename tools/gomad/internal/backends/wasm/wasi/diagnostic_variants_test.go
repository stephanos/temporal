package wasi

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestDiagnosticRunRootPreservesEarlierInvocation(t *testing.T) {
	base := t.TempDir()
	first, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(first, "seed0-guest.firefox.json")
	if err := os.WriteFile(marker, []byte("first profile"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Dir(first) != base || filepath.Dir(second) != base {
		t.Fatalf("invocations share their evidence directory: first=%s second=%s", first, second)
	}
	file, err := os.OpenFile(filepath.Join(second, "seed0-guest.firefox.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("fresh invocation cannot create its profile", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "first profile" {
		t.Fatal("earlier profile changed", err)
	}
}

func TestDiagnosticVariantArtifactSuffixes(t *testing.T) {
	root := t.TempDir()
	_, _, failure := runGuestVariant(t, root, "invalid-request", Request{}, false)
	if failure == "" {
		t.Fatal("invalid request unexpectedly executed")
	}
	files, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, file := range files {
		names = append(names, file.Name())
	}
	want := []string{
		"invalid-request-canonical.bin", "invalid-request-choice.bin",
		"invalid-request-guest-stderr.txt", "invalid-request-guest-stdout.txt",
		"invalid-request-helper-stderr.txt", "invalid-request-network.jsonl",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("artifact names = %v, want %v", names, want)
	}
}

func TestDiagnosticVariantEvidencePreservesDerivedEnvironmentRecords(t *testing.T) {
	config := testConfig()
	config.Environment = append(config.Environment, "GOGC=200")
	baseline, variant := newTestEnvironment(t, testConfig()), newTestEnvironment(t, config)
	for _, environment := range []*Environment{baseline, variant} {
		call(t, environment, "environ_sizes_get", `{}`)
		call(t, environment, "environ_get", `{}`)
		call(t, environment, "args_get", `{}`)
	}
	before := Result{Transcript: baseline.Transcript()}
	after := Result{Transcript: variant.Transcript()}
	unmasked, err := projectVariantEvidence(after, config, "")
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectVariantEvidence(before, testConfig(), "")
	if err != nil || reflect.DeepEqual(want, unmasked) {
		t.Fatal("unmasked derived environment change was lost", err)
	}
	want, err = projectVariantEvidence(before, testConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, config, "200")
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("GC environment was not masked", err)
	}
	for _, replacement := range []struct {
		name   string
		offset int
		value  byte
	}{
		{"environment operation", 8, 3},
		{"environment errno", 10, 1},
		{"argument operation", 40, 2},
		{"call id", 44, 9},
	} {
		changed := after
		changed.Transcript = bytes.Clone(after.Transcript)
		changed.Transcript[replacement.offset] = replacement.value
		projection, err := projectVariantEvidence(changed, config, "200")
		if err != nil || reflect.DeepEqual(want, projection) {
			t.Fatalf("mask hid non-environment change: %+v", replacement)
		}
	}
}

func TestDiagnosticVariantEvidencePreservesClockRuns(t *testing.T) {
	transcript := []byte("GMWT\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00")
	for _, mask := range []bool{false, true} {
		gcVariant := ""
		if mask {
			gcVariant = "200"
		}
		want, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), gcVariant)
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range []struct {
			name   string
			offset int
			value  byte
		}{
			{"id", 12, 2}, {"clock id", 24, 0}, {"precision", 28, 6}, {"position", 36, 1}, {"count", 44, 4},
		} {
			t.Run(change.name+"/mask="+strconv.FormatBool(mask), func(t *testing.T) {
				data := bytes.Clone(transcript)
				data[change.offset] = change.value
				got, err := projectVariantEvidence(Result{Transcript: data}, testConfig(), gcVariant)
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("clock evidence change was hidden", err)
				}
			})
		}
	}
}

func TestDiagnosticVariantEvidencePreservesInputsAndFailures(t *testing.T) {
	for _, test := range []struct {
		name         string
		tag          byte
		input        string
		inputOffsets []int
	}{
		{"random_get", 7, "\x02\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00", []int{0, 4}},
		{"fd_write", 8, "\x01\x00\x00\x00\x02\x00\x00\x00ab", []int{0, 9}},
		{"unsupported", 34, "\x03\x00\x00\x00bad\x0b\x00\x00\x00" + `{"value":1}`, []int{2 + 4, 20}},
		{"environ_get", 6, "", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			transcript := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, test.tag, test.input+"\x07\x00\x00\x00refused"))
			transcript[9] = 1
			want, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), "200")
			if err != nil {
				t.Fatal(err)
			}
			for _, offset := range []int{10, len(transcript) - 1} {
				data := bytes.Clone(transcript)
				data[offset] ^= 1
				got, err := projectVariantEvidence(Result{Transcript: data}, testConfig(), "200")
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("failure evidence was hidden", err)
				}
			}
			for _, offset := range test.inputOffsets {
				data := bytes.Clone(transcript)
				data[len(transcriptHeader)+transcriptRecordHeader+offset] ^= 1
				got, err := projectVariantEvidence(Result{Transcript: data}, testConfig(), "200")
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("operation input was hidden", err)
				}
			}
		})
	}
}

func TestDiagnosticVariantEvidencePreservesOutputs(t *testing.T) {
	transcript := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01\x00\x00\x00\x02\x00\x00\x00ab\x02\x00\x00\x00"))
	want, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	transcript[len(transcript)-4] = 1
	got, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), "200")
	if err != nil || reflect.DeepEqual(want, got) {
		t.Fatal("non-environment output change was hidden", err)
	}
}

func TestDiagnosticVariantEvidencePreservesUnsupportedFlag(t *testing.T) {
	before := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 6, ""))
	after := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 6, "\x00\x00\x00\x00"))
	after[9] = 1
	want, err := projectVariantEvidence(Result{Transcript: before}, testConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(Result{Transcript: after}, testConfig(), "200")
	if err != nil || reflect.DeepEqual(want, got) {
		t.Fatal("environment unsupported flag was hidden", err)
	}
}

func TestDiagnosticVariantEvidenceRefusesMalformedBinary(t *testing.T) {
	valid := []byte("GMWT\x01\x00\x00\x00\x06\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	unknown := bytes.Clone(valid)
	unknown[8] = 255
	oversized := bytes.Clone(valid)
	binary.LittleEndian.PutUint32(oversized[20:], frameLimit)
	for _, transcript := range [][]byte{
		[]byte("{\"type\":\"call\",\"id\":1,\"op\":\"environ_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{}}\n"),
		valid[:len(valid)-1], append(bytes.Clone(valid), 0), unknown, oversized,
	} {
		for _, mask := range []bool{false, true} {
			gcVariant := ""
			if mask {
				gcVariant = "200"
			}
			if _, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), gcVariant); err == nil {
				t.Fatalf("malformed binary transcript accepted with mask=%v: %x", mask, transcript)
			}
		}
	}
}

func TestDiagnosticVariantEvidenceSeparatesIdentities(t *testing.T) {
	code := uint32(0)
	before := Result{Termination: "exit", ExitCode: &code, Stdout: []byte("out"), Stderr: []byte("err"), Transcript: []byte{}, NetworkTranscript: []byte("network"), EnvironmentIdentity: "baseline", VirtualTimeElapsedNanos: 12}
	before.ChoiceTrace.Bytes = []byte("choice")
	after := before
	after.EnvironmentIdentity, after.FuelRemaining = "variant", 99
	baseRequest := Request{ModuleSHA256: "base-module", HelperSHA256: "base-helper", Engine: PinnedEngine()}
	variantRequest := Request{ModuleSHA256: "variant-module", HelperSHA256: "variant-helper", Engine: PinnedEngine(), diagnosticConfigurationRevision: PinnedEngine().Configuration + "-fuel-off"}
	want, err := projectVariantEvidence(before, testConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, testConfig(), "")
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("identity or fuel accounting entered behavior equality", err)
	}
	if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(variantRequest, after)) {
		t.Fatal("variant identities were not reported separately")
	}
	for _, field := range []string{"module_sha256", "helper_sha256", "environment_identity", "engine_revision"} {
		t.Run(field, func(t *testing.T) {
			request, result := baseRequest, before
			switch field {
			case "module_sha256":
				request.ModuleSHA256 = "other-module"
			case "helper_sha256":
				request.HelperSHA256 = "other-helper"
			case "environment_identity":
				result.EnvironmentIdentity = "other-environment"
			case "engine_revision":
				request.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-fuel-off"
			}
			if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(request, result)) {
				t.Fatal("changed identity was omitted")
			}
			projection, err := projectVariantEvidence(result, testConfig(), "")
			if err != nil || !reflect.DeepEqual(want, projection) {
				t.Fatal("identity change altered behavior equality", err)
			}
		})
	}
	for name, mutate := range map[string]func(*Result){
		"choice":  func(r *Result) { r.ChoiceTrace.Bytes = []byte("other") },
		"wasi":    func(r *Result) { r.Transcript = []byte("other") },
		"network": func(r *Result) { r.NetworkTranscript = []byte("other") },
		"stdout":  func(r *Result) { r.Stdout = []byte("other") },
		"stderr":  func(r *Result) { r.Stderr = []byte("other") },
		"outcome": func(r *Result) { r.Termination = "trap" },
		"message": func(r *Result) { r.Message = "boundary failure" },
		"exit":    func(r *Result) { other := uint32(1); r.ExitCode = &other },
		"time":    func(r *Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := before
			mutate(&changed)
			projection, err := projectVariantEvidence(changed, testConfig(), "")
			if err == nil && reflect.DeepEqual(want, projection) {
				t.Fatal("behavior change was hidden")
			}
		})
	}
}

func TestDiagnosticVariantEvidencePreservesDerivedInputs(t *testing.T) {
	for _, gcVariant := range []string{"", "200", "off"} {
		t.Run("gc="+gcVariant, func(t *testing.T) {
			want, err := projectVariantEvidence(Result{}, testConfig(), gcVariant)
			if err != nil {
				t.Fatal(err)
			}
			for name, mutate := range map[string]func(*Config){
				"argv":        func(c *Config) { c.Args = append(c.Args, "other") },
				"environment": func(c *Config) { c.Environment = append(c.Environment, "USER_INPUT=other") },
				"entropy":     func(c *Config) { c.EntropyKey[0]++ },
				"clock step":  func(c *Config) { c.Clock.ReadStepNanos++ },
				"clock epoch": func(c *Config) { c.Clock.EpochNanos++ },
			} {
				t.Run(name, func(t *testing.T) {
					config := testConfig()
					mutate(&config)
					got, err := projectVariantEvidence(Result{}, config, gcVariant)
					if err != nil || reflect.DeepEqual(want, got) {
						t.Fatal("derived input change was hidden", err)
					}
				})
			}
		})
	}
}

func TestDiagnosticVariantEvidenceMasksOnlyDeclaredGCInputs(t *testing.T) {
	for _, gcVariant := range []string{"200", "400", "off"} {
		t.Run(gcVariant, func(t *testing.T) {
			want, err := projectVariantEvidence(Result{}, testConfig(), gcVariant)
			if err != nil {
				t.Fatal(err)
			}
			config := testConfig()
			config.Environment = append(config.Environment, "GOGC="+gcVariant)
			if gcVariant == "off" {
				config.Environment = append(config.Environment, "GOMEMLIMIT=100B")
			}
			got, err := projectVariantEvidence(Result{}, config, gcVariant)
			if err != nil || !reflect.DeepEqual(want, got) {
				t.Fatal("declared GC policy change was not masked", err)
			}
			if gcVariant != "off" {
				config.Environment = append(config.Environment, "GOMEMLIMIT=100B")
				got, err := projectVariantEvidence(Result{}, config, gcVariant)
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("unexpected memory policy change was hidden", err)
				}
			}
		})
	}
}

func TestDiagnosticVariantRecommendationRequiresReliableGain(t *testing.T) {
	for _, test := range []struct {
		name                string
		baseline, candidate []uint64
		failed, identical   bool
		want                string
	}{
		{"gain", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, true, "adopt"},
		{"spread", []uint64{90, 99, 100, 101, 110}, []uint64{80, 89, 90, 91, 100}, false, true, "investigate"},
		{"slower", []uint64{98, 99, 100, 101, 102}, []uint64{108, 109, 110, 111, 112}, false, true, "reject"},
		{"mismatch", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, false, "reject"},
		{"failed", []uint64{98, 99, 100, 101, 102}, nil, true, false, "reject"},
		{"missing", []uint64{100}, nil, false, true, "investigate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := variantRecommendation(test.baseline, test.candidate, test.failed, test.identical); got != test.want {
				t.Fatalf("recommendation=%s want=%s", got, test.want)
			}
		})
	}
}
