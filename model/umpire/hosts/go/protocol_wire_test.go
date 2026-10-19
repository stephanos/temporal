package oracle

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	oraclev1 "umpire.local/oracle/umpire/oracle/v1"
)

func TestProtocolWireCompatibility(t *testing.T) {
	if oraclev1.Method_METHOD_EXPLORE != 1 || oraclev1.Method_METHOD_ROWS != 2 {
		t.Fatal("legacy method numbers changed")
	}
	data, err := os.ReadFile("../../tests/fixtures/shared/protocol-wire.txt")
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
				want = &oraclev1.ExploreRequest{Budget: 150}
			case "rows_request":
				want = &oraclev1.RowsRequest{StateId: 150}
			case "explore_response":
				want = &oraclev1.Response{Outcome: &oraclev1.Response_Explore{Explore: &oraclev1.ExploreResponse{
					Report: "r", Starts: 1, Reached: 2, Emitted: 3, Evaluated: 4, Refused: 5, Depth: 6, Cut: true,
				}}}
			case "rows_response":
				want = &oraclev1.Response{Outcome: &oraclev1.Response_Rows{Rows: &oraclev1.RowsResponse{Rows: []string{"x", "y"}}}}
			default:
				code, err := strconv.Atoi(strings.TrimPrefix(name, "refusal_"))
				if err != nil || code < 0 || code > 7 || name != fmt.Sprintf("refusal_%d", code) {
					t.Fatalf("unknown wire vector %q", name)
				}
				want = &oraclev1.Response{Outcome: &oraclev1.Response_Refusal{Refusal: &oraclev1.Refusal{Code: oraclev1.RefusalCode(code), Message: "x"}}}
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
