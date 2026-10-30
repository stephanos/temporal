package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func TestCurrentContractNativeWasmPublishedResponsesAgree(t *testing.T) {
	directory := os.Getenv("UMPIRE_CURRENT_CONTRACT_NATIVE")
	if directory == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../../target"
		}
		directory = filepath.Join(target, "current-contract-native")
	}
	for _, name := range []string{"ordinary", "callback", "configured", "frontier"} {
		t.Run(name, func(t *testing.T) {
			host := statefulHost(t)
			methods, err := os.ReadFile(filepath.Join(directory, name+".methods"))
			if err != nil {
				t.Fatal(err)
			}
			rows := strings.Fields(string(methods))
			if len(rows) < 7 {
				t.Fatalf("native transcript has %d calls, want a nonempty observation prefix", len(rows))
			}
			for index, row := range rows {
				method, err := strconv.ParseInt(row, 10, 32)
				if err != nil {
					t.Fatal(err)
				}
				prefix := filepath.Join(directory, fmt.Sprintf("%s-%d", name, index))
				request, err := os.ReadFile(prefix + ".request.pb")
				if err != nil {
					t.Fatal(err)
				}
				native, err := os.ReadFile(prefix + ".response.pb")
				if err != nil {
					t.Fatal(err)
				}
				want := new(modelv1.Response)
				if err := proto.Unmarshal(native, want); err != nil {
					t.Fatal(err)
				}
				got, err := host.Call(context.Background(), modelv1.Method(method), request)
				if err != nil {
					t.Fatalf("call %d: %v", index, err)
				}
				if !proto.Equal(got, want) {
					t.Fatalf("call %d differs from native published response: got %v, want %v", index, got, want)
				}
				if index == 0 && got.GetReset_().GetProofContract() != modelv1.ProofContract_PROOF_CONTRACT_WHOLE_PREFIX_V1 {
					t.Fatalf("ordinary Reset did not admit the current contract: %v", got)
				}
			}
		})
	}
}
