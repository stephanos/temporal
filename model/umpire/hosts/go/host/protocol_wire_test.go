package host

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func TestExplanationOutcomeWireCompatibility(t *testing.T) {
	for _, tc := range []struct {
		outcomes []string
		wire     []byte
	}{
		{nil, []byte{10, 1, 'x'}},
		{[]string{"()"}, []byte{10, 1, 'x', 18, 2, '(', ')'}},
	} {
		want := &modelv1.Explanation{Steps: []string{"x"}, Outcomes: tc.outcomes}
		encoded, err := proto.Marshal(want)
		if err != nil || !bytes.Equal(encoded, tc.wire) {
			t.Fatalf("encoded %x, %v; want %x", encoded, err, tc.wire)
		}
		var decoded modelv1.Explanation
		if err := proto.Unmarshal(tc.wire, &decoded); err != nil || !proto.Equal(want, &decoded) {
			t.Fatalf("decoded %v, %v; want %v", &decoded, err, want)
		}
	}
}

func TestProtocolWireCompatibility(t *testing.T) {
	if modelv1.Method_METHOD_EXPLORE != 1 || modelv1.Method_METHOD_ROWS != 2 {
		t.Fatal("legacy method numbers changed")
	}
	for name, code := range map[string]modelv1.RefusalCode{
		"REFUSAL_CODE_UNSPECIFIED": 0, "REFUSAL_CODE_UNKNOWN_METHOD": 1,
		"REFUSAL_CODE_MALFORMED_REQUEST": 2, "REFUSAL_CODE_BUDGET": 3,
		"REFUSAL_CODE_STATE": 4, "REFUSAL_CODE_NOT_EXPLORED": 5,
		"REFUSAL_CODE_ALLOCATION": 6, "REFUSAL_CODE_SCOPE_ACTION": 7,
		"REFUSAL_CODE_UNSUPPORTED": 8,
	} {
		if modelv1.RefusalCode_value[name] != int32(code) || modelv1.RefusalCode_name[int32(code)] != name {
			t.Fatalf("refusal number changed: %s = %d", name, code)
		}
	}
	data, err := os.ReadFile("../../../proto/tests/protocol-wire.txt")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || seen[fields[0]] {
			t.Fatalf("invalid or duplicate wire vector: %q", line)
		}
		name := fields[0]
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			wire, err := hex.DecodeString(fields[1])
			if err != nil {
				t.Fatal(err)
			}
			var want proto.Message
			switch name {
			case "explore_request":
				want = &modelv1.ExploreRequest{Budget: 150}
			case "rows_request":
				want = &modelv1.RowsRequest{StateId: 150}
			case "explore_response":
				want = &modelv1.Response{Outcome: &modelv1.Response_Explore{Explore: &modelv1.ExploreResponse{
					Report: "r", Starts: 1, Reached: 2, Emitted: 3, Evaluated: 4, Refused: 5, Depth: 6, Cut: true,
				}}}
			case "rows_response":
				want = &modelv1.Response{Outcome: &modelv1.Response_Rows{Rows: &modelv1.RowsResponse{Rows: []string{"x", "y"}}}}
			default:
				code, err := strconv.Atoi(strings.TrimPrefix(name, "refusal_"))
				if err != nil || code < 0 || code > 7 || name != fmt.Sprintf("refusal_%d", code) {
					t.Fatalf("unknown wire vector %q", name)
				}
				want = &modelv1.Response{Outcome: &modelv1.Response_Refusal{Refusal: &modelv1.Refusal{Code: modelv1.RefusalCode(code), Message: "x"}}}
			}
			got := want.ProtoReflect().New().Interface()
			if err := proto.Unmarshal(wire, got); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, want) {
				t.Fatalf("decoded %v, want %v", got, want)
			}
			encoded, err := proto.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(encoded) != fields[1] {
				t.Fatalf("reencoded %x, want %x", encoded, wire)
			}
		})
	}
	if len(seen) != 12 {
		t.Fatalf("got %d legacy vectors, want 12", len(seen))
	}
}

func TestCoreProtocolWireCompatibility(t *testing.T) {
	if modelv1.Method_METHOD_RESET != 3 || modelv1.Method_METHOD_JUDGE != 4 || modelv1.Method_METHOD_ANSWER != 5 {
		t.Fatal("core method numbers changed")
	}
	opaque := &anypb.Any{TypeUrl: "x", Value: []byte{1, 2}}
	wants := map[string]proto.Message{
		"reset_response":    &modelv1.Response{Outcome: &modelv1.Response_Reset_{Reset_: &modelv1.ResetResponse{}}},
		"judge_request":     &modelv1.JudgeRequest{Observation: opaque},
		"answer_request":    &modelv1.AnswerRequest{Callback: opaque},
		"judge_explained":   &modelv1.Response{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{}}}}},
		"judge_unexplained": &modelv1.Response{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Unexplained{Unexplained: &modelv1.Unexplained{}}}}},
		"answer_response":   &modelv1.Response{Outcome: &modelv1.Response_Answer{Answer: &modelv1.AnswerResponse{Reply: opaque}}},
	}
	data, err := os.ReadFile("../../../proto/tests/protocol-core-wire.txt")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || wants[fields[0]] == nil || seen[fields[0]] {
			t.Fatalf("invalid or duplicate core wire vector: %q", line)
		}
		name := fields[0]
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			wire, err := hex.DecodeString(fields[1])
			if err != nil {
				t.Fatal(err)
			}
			got := wants[name].ProtoReflect().New().Interface()
			if err := proto.Unmarshal(wire, got); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, wants[name]) {
				t.Fatalf("decoded %v, want %v", got, wants[name])
			}
			encoded, err := proto.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(encoded) != fields[1] {
				t.Fatalf("reencoded %x, want %x", encoded, wire)
			}
		})
	}
	if len(seen) != len(wants) {
		t.Fatalf("got %d core vectors, want %d", len(seen), len(wants))
	}
}

func TestProtocolNamespace(t *testing.T) {
	request := &modelv1.ExploreRequest{Budget: 150}
	if name := request.ProtoReflect().Descriptor().FullName(); name != "umpire.model.v1.ExploreRequest" {
		t.Fatalf("wrong message namespace: %s", name)
	}
	packed, err := anypb.New(request)
	if err != nil {
		t.Fatal(err)
	}
	if packed.TypeUrl != "type.googleapis.com/umpire.model.v1.ExploreRequest" || !packed.MessageIs(request) {
		t.Fatalf("wrong Any identity: %v", packed)
	}
	packed.TypeUrl = "type.googleapis.com/umpire.oracle.v1.ExploreRequest"
	if packed.MessageIs(request) {
		t.Fatal("old Any namespace matched renamed message")
	}
	if err := packed.UnmarshalTo(new(modelv1.ExploreRequest)); err == nil {
		t.Fatal("old Any namespace decoded as renamed message")
	}
}
