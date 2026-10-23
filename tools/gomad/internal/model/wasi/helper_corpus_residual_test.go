package wasi

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const residualRemoteYield = "residual_remote_yield"
const residualStorageContext = "residual_storage_context_declared_overlimit"

func helperCorpusResidualInput(name string, value any) error {
	if name != residualRemoteYield && name != residualStorageContext {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var c struct {
		Guest struct {
			WAT string `json:"wat"`
		} `json:"guest"`
		Request struct {
			Storage struct {
				CrashContext struct {
					Bytes, Chunks uint64
					SHA256        string
				} `json:"crash_context"`
			}
		} `json:"request"`
		Inputs  []json.RawMessage `json:"inputs"`
		Replies []struct {
			After json.RawMessage
			Send  []struct {
				Type      string
				ID, Errno uint64
				Output    json.RawMessage
				Data      string `json:"data_base64"`
			}
		} `json:"replies"`
		CloseInput string `json:"close_input"`
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}
	var request map[string]json.RawMessage
	if err := json.Unmarshal(raw["request"], &request); err != nil {
		return err
	}
	if name == residualRemoteYield {
		if c.Guest.WAT != `(module (import "wasi_snapshot_preview1" "sched_yield" (func $yield (result i32))) (memory (export "memory") 1) (func (export "_start") call $yield if unreachable end))` || len(request["local_model"]) != 0 || len(request["namespace"]) != 0 || len(c.Inputs) != 0 || len(c.Replies) != 2 || len(c.Replies[1].Send) != 1 {
			return fmt.Errorf("remote yield must be exact unmodeled input")
		}
		var after struct {
			Type, Op string
			ID       uint64
		}
		if err := json.Unmarshal(c.Replies[1].After, &after); err != nil {
			return err
		}
		r := c.Replies[1].Send[0]
		if after.Type != "call" || after.Op != "sched_yield" || after.ID != 1 || r.Type != "reply" || r.ID != 1 || r.Errno != 0 || string(r.Output) != "{}" {
			return fmt.Errorf("remote yield reply tuple differs")
		}
	} else {
		if len(c.Replies) != 1 || len(c.Replies[0].Send) != 3 || c.CloseInput != "after_replies" {
			return fmt.Errorf("storage setup must contain no context payload")
		}
		s := c.Replies[0].Send
		if s[0].Type != "authorize" || s[1].Type != "namespace_input" || s[2].Type != "namespace_inputs_end" {
			return fmt.Errorf("storage setup order differs")
		}
		image, err := base64.StdEncoding.DecodeString(s[1].Data)
		if err != nil {
			return err
		}
		if len(image) < 32 || binary.LittleEndian.Uint64(image[24:32]) != 1024 {
			return fmt.Errorf("namespace filesystem limit differs")
		}
		d := c.Request.Storage.CrashContext
		if d.Bytes != 18892801 || d.Chunks != 289 || d.SHA256 != "0000000000000000000000000000000000000000000000000000000000000000" || d.Bytes != 18*(binary.LittleEndian.Uint64(image[24:32])+(1<<20))+1 {
			return fmt.Errorf("context must declare exact computed maximum+1 with valid syntax")
		}
	}
	return nil
}

func helperCorpusResidualActual(name string, value any) error {
	if name != residualRemoteYield && name != residualStorageContext {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var p struct {
		Frames []struct {
			Type, Op, Termination, Message string
			ID                             uint64
			Input                          json.RawMessage
			ExitCode                       *uint32         `json:"exit_code"`
			Peak                           *uint64         `json:"peak_memory_bytes"`
			Fuel                           *uint64         `json:"fuel_remaining"`
			LocalModel                     json.RawMessage `json:"local_model"`
			Metrics                        json.RawMessage
		}
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	if len(p.Frames) < 2 || p.Frames[0].Type != "started" || p.Frames[len(p.Frames)-1].Type != "result" {
		return fmt.Errorf("missing started/terminal result")
	}
	for _, f := range p.Frames[1:] {
		if f.Type == "started" {
			return fmt.Errorf("duplicate started")
		}
	}
	r := p.Frames[len(p.Frames)-1]
	if name == residualRemoteYield {
		if len(p.Frames) != 3 || p.Frames[1].Type != "call" || p.Frames[1].Op != "sched_yield" || p.Frames[1].ID != 1 || string(p.Frames[1].Input) != "{}" || r.Termination != "exit" || r.ExitCode == nil || *r.ExitCode != 0 || r.Message != "" || len(r.LocalModel) != 0 {
			return fmt.Errorf("remote yield callback/exit literal differs")
		}
	} else {
		var metrics struct {
			Instantiate *uint64 `json:"instantiate_nanos"`
			Execute     *uint64 `json:"execute_nanos"`
		}
		if err := json.Unmarshal(r.Metrics, &metrics); err != nil {
			return err
		}
		var model struct{ Calls *uint64 }
		if err := json.Unmarshal(r.LocalModel, &model); err != nil {
			return err
		}
		zero := func(value *uint64) bool { return value != nil && *value == 0 }
		if len(p.Frames) != 2 || r.Termination != "invalid" || r.Message != "WASI invalid: storage-context-bytes" || !zero(r.Peak) || !zero(r.Fuel) || !zero(metrics.Instantiate) || !zero(metrics.Execute) || !zero(model.Calls) {
			return fmt.Errorf("storage context refused at wrong stage")
		}
	}
	return nil
}

func TestHelperCorpusResidualLiteralGuards(t *testing.T) {
	for _, value := range []struct{ name, actual string }{
		{residualRemoteYield, `{"frames":[{"type":"started"},{"type":"call","op":"sched_yield","id":1,"input":{}},{"type":"result","termination":"exit","exit_code":0,"message":"","metrics":{"compile_nanos":"nonzero","instantiate_nanos":"nonzero","execute_nanos":"nonzero"}}]}`},
		{residualStorageContext, `{"frames":[{"type":"started"},{"type":"result","termination":"invalid","message":"WASI invalid: storage-context-bytes","peak_memory_bytes":0,"fuel_remaining":0,"metrics":{"instantiate_nanos":0,"execute_nanos":0},"local_model":{"calls":0}}]}`},
	} {
		actual, err := decodeCorpusJSON([]byte(value.actual))
		if err != nil {
			t.Fatal(err)
		}
		if err := helperCorpusResidualActual(value.name, actual); err != nil {
			t.Fatal(err)
		}
		if value.name == residualStorageContext {
			for _, mutation := range []string{"nonzero", "missing", "null", "projected-nonzero"} {
				changed, err := decodeCorpusJSON([]byte(value.actual))
				if err != nil {
					t.Fatal(err)
				}
				frames := changed.(map[string]any)["frames"].([]any)
				metrics := frames[1].(map[string]any)["metrics"].(map[string]any)
				switch mutation {
				case "nonzero":
					metrics["execute_nanos"] = json.Number("1")
				case "missing":
					delete(metrics, "execute_nanos")
				case "null":
					metrics["execute_nanos"] = nil
				case "projected-nonzero":
					metrics["execute_nanos"] = "nonzero"
				}
				if helperCorpusResidualActual(value.name, changed) == nil {
					t.Fatalf("storage %s execution metric accepted", mutation)
				}
			}
		}
	}
	for _, name := range []string{residualRemoteYield, residualStorageContext} {
		data, err := os.ReadFile(filepath.Join(helperCorpusDirectory, name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		value, err := decodeCorpusJSON(data)
		if err != nil {
			t.Fatal(err)
		}
		if err := helperCorpusResidualInput(name, value); err != nil {
			t.Fatal(err)
		}
		fields := value.(map[string]any)
		if name == residualRemoteYield {
			fields["replies"].([]any)[1].(map[string]any)["send"].([]any)[0].(map[string]any)["errno"] = json.Number("1")
		} else {
			fields["request"].(map[string]any)["storage"].(map[string]any)["crash_context"].(map[string]any)["bytes"] = json.Number("18892800")
		}
		if helperCorpusResidualInput(name, value) == nil {
			t.Fatalf("%s mutated setup accepted", name)
		}
	}
	for _, value := range []struct{ name, actual string }{
		{residualRemoteYield, `{"frames":[{"type":"started"},{"type":"call","op":"clock_time_get","id":1,"input":{}},{"type":"result","termination":"exit","exit_code":0,"message":""}]}`},
		{residualStorageContext, `{"frames":[{"type":"started"},{"type":"result","termination":"invalid","message":"invalid storage crash context descriptor","local_model":{"calls":0}}]}`},
	} {
		actual, err := decodeCorpusJSON([]byte(value.actual))
		if err != nil {
			t.Fatal(err)
		}
		if helperCorpusResidualActual(value.name, actual) == nil {
			t.Fatalf("%s wrong-stage projection accepted", value.name)
		}
	}
}
