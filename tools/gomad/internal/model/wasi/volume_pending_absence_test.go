package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"testing"
)

type volumePendingAbsenceCorpus struct {
	Schema string                  `json:"schema"`
	Path   string                  `json:"probe_path"`
	Absent []bool                  `json:"absent_after_operation"`
	Corpus volumeConformanceCorpus `json:"corpus"`
}

func runVolumePendingAbsence(t *testing.T) volumePendingAbsenceCorpus {
	t.Helper()
	fixture := volumePendingAbsenceCorpus{Schema: "gomad.volume-pending-namespace-absence/v1", Path: "/workspace/db"}
	config := storageConfig()
	config.Storage.Limits.PendingOperations = 1
	script := volumeConformanceScript{
		name:       "pending-open-namespace-absence",
		origin:     "wasi/volume_test.go:TestStorageCrashFaultAndModelLimits",
		config:     config,
		operations: []volumeConformanceOperation{volumeConformanceOpen("db", 1, rightsAll)},
		observe: func(environment *Environment, step int) {
			if step != len(fixture.Absent) {
				t.Fatal("namespace observer operation order differs")
			}
			fixture.Absent = append(fixture.Absent, environment.namespace.lookup(fixture.Path) == nil)
		},
	}
	fixture.Corpus = volumeConformanceRun(t, script)
	return fixture
}

func checkVolumePendingAbsenceLiterals(fixture volumePendingAbsenceCorpus) error {
	if fixture.Schema != "gomad.volume-pending-namespace-absence/v1" || fixture.Path != "/workspace/db" || !reflect.DeepEqual(fixture.Absent, []bool{true}) {
		return fmt.Errorf("pending-open namespace probe differs")
	}
	corpus := fixture.Corpus
	config := storageConfig()
	config.Storage.Limits.PendingOperations = 1
	if corpus.Schema != "gomad.volume-conformance/v1" || corpus.Name != "pending-open-namespace-absence" || corpus.Origin != "wasi/volume_test.go:TestStorageCrashFaultAndModelLimits" || corpus.Seed != 0 || !reflect.DeepEqual(corpus.Session.Storage, config.Storage) || len(corpus.Operations) != 1 || len(corpus.Observations) != 1 {
		return fmt.Errorf("pending-open setup or operation count differs")
	}
	op := corpus.Operations[0]
	var input openInput
	decoder := json.NewDecoder(bytes.NewReader(op.Input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("pending-open trailing input: %v", err)
	}
	if op.Op != "path_open" || op.Now != 0 || input.FD != 4 || string(input.Path) != "db" || input.Oflags != 1 || input.Base != 536870911 || input.Inheriting != 0 || input.Dirflags != 0 || input.Flags != 0 {
		return fmt.Errorf("pending-open exact call differs")
	}
	if !bytes.Equal(corpus.InitialReply, []byte(`{}`)) || !bytes.Equal(corpus.Observations[0].Reply, []byte(`{"boundary":"capacity:storage-pending_operations"}`)) {
		return fmt.Errorf("pending-open exact capacity reply differs")
	}
	return nil
}

func TestVolumePendingOpenNamespaceAbsenceLiteralGuards(t *testing.T) {
	mutations := map[string]func(*volumePendingAbsenceCorpus){
		"false-absence":   func(c *volumePendingAbsenceCorpus) { c.Absent[0] = false },
		"missing-absence": func(c *volumePendingAbsenceCorpus) { c.Absent = nil },
		"wrong-probe":     func(c *volumePendingAbsenceCorpus) { c.Path = "/workspace/other" },
		"wrong-operation": func(c *volumePendingAbsenceCorpus) { c.Corpus.Operations[0].Op = "fd_close" },
		"wrong-rights": func(c *volumePendingAbsenceCorpus) {
			c.Corpus.Operations[0] = volumeConformanceOpen("db", 1, rightRead)
		},
		"wrong-path": func(c *volumePendingAbsenceCorpus) {
			c.Corpus.Operations[0] = volumeConformanceOpen("other", 1, rightsAll)
		},
		"wrong-limit": func(c *volumePendingAbsenceCorpus) { c.Corpus.Session.Storage.Limits.PendingOperations = 2 },
		"unknown-input-field": func(c *volumePendingAbsenceCorpus) {
			input := c.Corpus.Operations[0].Input
			c.Corpus.Operations[0].Input = append(append([]byte{}, input[:len(input)-1]...), []byte(`,"unknown":1}`)...)
		},
		"trailing-input": func(c *volumePendingAbsenceCorpus) {
			c.Corpus.Operations[0].Input = append(c.Corpus.Operations[0].Input, []byte(` {}`)...)
		},
		"wrong-reply": func(c *volumePendingAbsenceCorpus) {
			c.Corpus.Observations[0].Reply = []byte(`{"boundary":"infrastructure:other"}`)
		},
		"dropped-operation": func(c *volumePendingAbsenceCorpus) { c.Corpus.Operations = nil },
		"duplicated-operation": func(c *volumePendingAbsenceCorpus) {
			c.Corpus.Operations = append(c.Corpus.Operations, c.Corpus.Operations[0])
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			fixture := runVolumePendingAbsence(t)
			if err := checkVolumePendingAbsenceLiterals(fixture); err != nil {
				t.Fatal(err)
			}
			mutate(&fixture)
			if err := checkVolumePendingAbsenceLiterals(fixture); err == nil {
				t.Fatal("changed pending-open absence contract accepted")
			}
		})
	}
}

func TestVolumePendingOpenNamespaceAbsenceCorpus(t *testing.T) {
	actual := runVolumePendingAbsence(t)
	if err := checkVolumePendingAbsenceLiterals(actual); err != nil {
		t.Fatal(err)
	}
	if *volumeConformanceOutput != "" {
		volumeBoundaryWrite(t, "pending-open-namespace-absence.json", actual)
		return
	}
	if *updateNamespaceConformance {
		t.Fatal("use external -volume-conformance-output for this Go-only vector")
	}
	var fixture volumePendingAbsenceCorpus
	path := filepath.Join("testdata", "volume-conformance", "vectors", "pending-open-namespace-absence.json")
	namespacePublicFixture(t, path, func() any {
		actual := runVolumePendingAbsence(t)
		if err := checkVolumePendingAbsenceLiterals(actual); err != nil {
			t.Fatal(err)
		}
		return actual
	}, &fixture)
	if err := checkVolumePendingAbsenceLiterals(fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != actual.Schema || fixture.Path != actual.Path || !reflect.DeepEqual(fixture.Absent, actual.Absent) {
		t.Fatal("pending-open namespace probe differs")
	}
	if difference := volumeConformanceDifference(fixture.Corpus, actual.Corpus); difference != "" {
		t.Fatal(difference)
	}
}
