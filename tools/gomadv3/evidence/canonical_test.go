package evidence

import (
	"bytes"
	"testing"
)

func TestCanonicalJSONSortsKeysAndPreservesJSONText(t *testing.T) {
	value := struct {
		Zulu  []any  `json:"z"`
		Alpha string `json:"a"`
		Beta  uint32 `json:"b"`
	}{
		Zulu:  []any{true, nil},
		Alpha: "<>&",
		Beta:  2,
	}

	got, err := CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"a":"<>&","b":2,"z":[true,null]}`)
	if !bytes.Equal(got, want) {
		t.Fatalf("CanonicalJSON() = %s, want %s", got, want)
	}
}

func TestCanonicalJSONRejectsFloatingPointAndInvalidUTF8(t *testing.T) {
	for name, value := range map[string]any{
		"float":         map[string]any{"value": 1.25},
		"invalid UTF-8": map[string]any{"value": string([]byte{0xff})},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := CanonicalJSON(value); err == nil {
				t.Fatal("CanonicalJSON() succeeded")
			}
		})
	}
}

func TestUint64StringJSONUsesStrictDecimalStrings(t *testing.T) {
	type document struct {
		Value Uint64String `json:"value"`
	}
	encoded, err := CanonicalJSON(document{Value: Uint64String(^uint64(0))})
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
		if err := StrictDecode([]byte(input), &decoded); err == nil {
			t.Fatalf("StrictDecode(%s) succeeded", input)
		}
	}
}

func TestStrictDecodeRejectsDuplicateUnknownAndTrailingData(t *testing.T) {
	type document struct {
		Name string `json:"name"`
	}
	for _, input := range []string{
		`{"name":"first","name":"second"}`,
		`{"name":"value","unknown":true}`,
		`{"name":"value"} {}`,
	} {
		var decoded document
		if err := StrictDecode([]byte(input), &decoded); err == nil {
			t.Fatalf("StrictDecode(%s) succeeded", input)
		}
	}
}

func TestDecodeCanonicalJSONRejectsNonCanonicalEncoding(t *testing.T) {
	type document struct {
		Alpha string `json:"alpha"`
		Zulu  string `json:"zulu"`
	}
	var decoded document
	if err := DecodeCanonicalJSON([]byte(`{"zulu":"last","alpha":"first"}`), &decoded); err == nil {
		t.Fatal("DecodeCanonicalJSON() accepted non-canonical key order")
	}
}

func TestDecodeCanonicalJSONAcceptsCanonicalEncoding(t *testing.T) {
	type document struct {
		Alpha string `json:"alpha"`
		Zulu  string `json:"zulu"`
	}
	var decoded document
	if err := DecodeCanonicalJSON([]byte(`{"alpha":"first","zulu":"last"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Alpha != "first" || decoded.Zulu != "last" {
		t.Fatalf("DecodeCanonicalJSON() = %#v", decoded)
	}
}

func TestCanonicalJSONLinesRoundTripsAndRejectsTruncation(t *testing.T) {
	type entry struct {
		Seed Uint64String `json:"seed"`
	}
	encoded, err := CanonicalJSONLines([]any{entry{Seed: 7}, entry{Seed: 11}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), "{\"seed\":\"7\"}\n{\"seed\":\"11\"}\n"; got != want {
		t.Fatalf("CanonicalJSONLines() = %q, want %q", got, want)
	}
	decoded, err := StrictDecodeJSONLines[entry](encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[0].Seed != 7 || decoded[1].Seed != 11 {
		t.Fatalf("StrictDecodeJSONLines() = %#v", decoded)
	}
	if _, err := StrictDecodeJSONLines[entry](bytes.TrimSuffix(encoded, []byte{'\n'})); err == nil {
		t.Fatal("StrictDecodeJSONLines accepted a truncated final line")
	}
	if decoded, err := StrictDecodeJSONLines[entry](nil); err != nil || len(decoded) != 0 {
		t.Fatalf("StrictDecodeJSONLines(nil) = %#v, %v", decoded, err)
	}
}
