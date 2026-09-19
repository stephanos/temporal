package record

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

func TestUint64StringJSONUsesStrictDecimalStrings(t *testing.T) {
	type document struct {
		Value Uint64String `json:"value"`
	}
	encoded, err := canonicaljson.CanonicalJSON(document{Value: Uint64String(^uint64(0))})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), `{"value":"18446744073709551615"}`; got != want {
		t.Fatalf("CanonicalJSON() = %s, want %s", got, want)
	}

	for _, input := range []string{
		`{"value":1}`,
		`{"value":""}`,
		`{"value":"+1"}`,
		`{"value":"01"}`,
		`{"value":"18446744073709551616"}`,
	} {
		var decoded document
		if err := canonicaljson.StrictDecode([]byte(input), &decoded); err == nil {
			t.Fatalf("StrictDecode(%s) succeeded", input)
		}
	}
}
