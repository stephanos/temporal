package execution_test

import (
	"errors"
	"reflect"
	"testing"
)

func TestModelDeclaredDifferencesAreNarrow(t *testing.T) {
	for platform, differences := range modelDeclaredDifferences {
		t.Run(platform, func(t *testing.T) {
			for _, difference := range differences {
				t.Run(difference.Name, func(t *testing.T) {
					if difference.Reason == "" {
						t.Fatal("declared difference requires a reason")
					}
					model := modelOperationResult{"stat-dir workspace/dir", `{"name":"dir","directory":true,"size":0,"mode":448}`, "ok"}
					native := modelOperationResult{"stat-dir workspace/dir", `{"name":"dir","directory":true,"size":96,"mode":448}`, "ok"}
					if err := compareModelLogs(platform, []modelOperationResult{model}, []modelOperationResult{native}); err != nil {
						t.Fatal(err)
					}
					cases := []struct {
						name          string
						model, native modelOperationResult
					}{
						{"directory-mode", model, modelOperationResult{native.Operation, `{"name":"dir","directory":true,"size":96,"mode":493}`, "ok"}},
						{"directory-kind", model, modelOperationResult{native.Operation, `{"name":"dir","directory":false,"size":96,"mode":448}`, "ok"}},
						{"directory-name", model, modelOperationResult{native.Operation, `{"name":"other","directory":true,"size":96,"mode":448}`, "ok"}},
						{"error-class", model, modelOperationResult{native.Operation, native.Result, "permission"}},
						{"operation", model, modelOperationResult{"stat workspace/file", native.Result, "ok"}},
						{"regular-file-size", modelOperationResult{"stat workspace/file", `{"name":"file","directory":false,"size":0,"mode":384}`, "ok"}, modelOperationResult{"stat workspace/file", `{"name":"file","directory":false,"size":96,"mode":384}`, "ok"}},
						{"other-directory-size", modelOperationResult{"stat-dir workspace/other", model.Result, "ok"}, modelOperationResult{"stat-dir workspace/other", native.Result, "ok"}},
						{"payload", modelOperationResult{"read-full client 3", "3:616263", "ok"}, modelOperationResult{"read-full client 3", "3:616264", "ok"}},
						{"byte-count", modelOperationResult{"read-full client 3", "2:6162", "ok"}, modelOperationResult{"read-full client 3", "3:616263", "ok"}},
						{"unknown-metadata", model, modelOperationResult{native.Operation, `{"name":"dir","directory":true,"size":96,"mode":448,"extra":1}`, "ok"}},
						{"negative-size", model, modelOperationResult{native.Operation, `{"name":"dir","directory":true,"size":-1,"mode":448}`, "ok"}},
						{"missing-size", model, modelOperationResult{native.Operation, `{"name":"dir","directory":true,"mode":448}`, "ok"}},
						{"trailing-directory-data", model, modelOperationResult{native.Operation, native.Result + " {}", "ok"}},
					}
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							if err := compareModelLogs(platform, []modelOperationResult{tc.model}, []modelOperationResult{tc.native}); err == nil {
								t.Fatal("semantic difference was hidden")
							}
						})
					}
				})
			}
		})
	}
	if err := compareModelLogs("unqualified/platform", nil, nil); err == nil {
		t.Fatal("unknown platform was accepted")
	}
	if err := compareModelLogs("darwin/arm64", nil, []modelOperationResult{{"close", "", "ok"}}); err == nil {
		t.Fatal("missing operation was hidden")
	}
}

func TestModelShortestDivergenceChecksEveryEarlierPrefix(t *testing.T) {
	failure := errors.New("payload differs")
	var checked []int
	prefix, err := shortestModelDivergence(8, func(length int) error {
		checked = append(checked, length)
		if length == 3 || length == 8 {
			return failure
		}
		return nil
	})
	if prefix != 3 || !errors.Is(err, failure) || !reflect.DeepEqual(checked, []int{1, 2, 3}) {
		t.Fatalf("prefix=%d err=%v checked=%v", prefix, err, checked)
	}
	checked = nil
	prefix, err = shortestModelDivergence(4, func(length int) error { checked = append(checked, length); return nil })
	if prefix != 0 || err != nil || !reflect.DeepEqual(checked, []int{1, 2, 3, 4}) {
		t.Fatalf("non-reproducing result: prefix=%d err=%v checked=%v", prefix, err, checked)
	}
}

func TestModelLogRejectsIncompleteOrMalformedOperations(t *testing.T) {
	for _, data := range []string{"", `{"operation":"read","result":"","error":"ok","extra":1}`, `{"operation":"read","result":""}`, `{"operation":"read","result":"","error":"ok"} {"extra":1}`} {
		if _, err := decodeModelLog([]byte(data), 1); err == nil {
			t.Fatalf("accepted malformed log %q", data)
		}
	}
}
