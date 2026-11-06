package activityrelations_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"testing"

	"github.com/temporalio/umpire/hosts/go/relation"
	_ "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"umpire.local/temporal/activityrelations"
)

type vectors struct {
	Format  string `json:"format"`
	Vectors []struct {
		Name         string `json:"name"`
		Method       string `json:"method"`
		ReceiptRunID string `json:"receipt_run_id"`
		Outcome      string `json:"outcome"`
		RequestHex   string `json:"request_hex"`
		ResponseHex  string `json:"response_hex"`
	} `json:"vectors"`
}

func decode(t *testing.T, name, data string) proto.Message {
	t.Helper()
	kind, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name))
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}
	message := kind.New().Interface()
	if err := proto.Unmarshal(bytes, message); err != nil {
		t.Fatal(err)
	}
	return message
}

// The Rust bindings decided these invocations; the generated Go plans must decide them alike.
func TestGoPlansReproduceRustRelationVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var loaded vectors
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Format != "umpire.temporal.activity-relations.vectors.v1" || len(loaded.Vectors) == 0 {
		t.Fatalf("unexpected vectors %q", loaded.Format)
	}
	bindings, err := activityrelations.NewBindings()
	if err != nil {
		t.Fatal(err)
	}
	for _, vector := range loaded.Vectors {
		novelty, ok := bindings.For(vector.Method)
		if !ok {
			t.Fatalf("%s: no binding for %s", vector.Name, vector.Method)
		}
		plan := novelty.Evaluator().Plan()
		response, err := hex.DecodeString(vector.ResponseHex)
		if err != nil {
			t.Fatal(err)
		}
		ctx := relation.Context{
			Request:  decode(t, plan.Input, vector.RequestHex),
			Response: decode(t, plan.Output, vector.ResponseHex),
			Receipt:  activityrelations.Receipt(vector.ReceiptRunID),
		}
		_, key, err := novelty.CheckAndProject(ctx, response)
		var got string
		var check *relation.CheckError
		var refusal *relation.NoveltyRefusal
		switch {
		case err == nil:
			got = "key:" + key.Hex()
		case errors.As(err, &check):
			got = fmt.Sprintf("correlation:%s:%s:%s", check.Class, check.Reason, check.Declaration)
		case errors.As(err, &refusal):
			got = "projection:" + string(refusal.Reason)
		default:
			t.Fatalf("%s: %v", vector.Name, err)
		}
		if got != vector.Outcome {
			t.Errorf("%s: got %s, want %s", vector.Name, got, vector.Outcome)
		}
	}
}

func TestEveryPlanAdmitsWithItsEmbeddedDescriptors(t *testing.T) {
	bindings, err := activityrelations.NewBindings()
	if err != nil {
		t.Fatal(err)
	}
	for _, plan := range activityrelations.Plans() {
		if plan.BindingSHA256 == "" || len(plan.Relations) == 0 {
			t.Fatalf("%s: empty plan", plan.Method)
		}
		if _, ok := bindings.For("/" + plan.Method); !ok {
			t.Fatalf("%s: not bound", plan.Method)
		}
	}
}

// The generator emits gofmt-stable Go, so the committed plans need no formatter.
func TestGeneratedPlansAreGofmtClean(t *testing.T) {
	files, err := filepath.Glob("*_plan.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(activityrelations.Plans()) {
		t.Fatalf("found %d plan files for %d plans", len(files), len(activityrelations.Plans()))
	}
	for _, name := range files {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		formatted, err := format.Source(source)
		if err != nil {
			t.Fatal(err)
		}
		if string(formatted) != string(source) {
			t.Errorf("%s is not gofmt-clean", name)
		}
	}
}
