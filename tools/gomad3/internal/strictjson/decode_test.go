package strictjson

import (
	"encoding/json"
	"strings"
	"testing"
)

type decodeDocument struct {
	Name   string `json:"name"`
	Nested struct {
		Value string `json:"value"`
	} `json:"nested"`
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
	Fields map[string]int  `json:"fields"`
	Raw    json.RawMessage `json:"raw"`
	Limit  uint8           `json:"limit"`
}

func TestDecodeRejectsStrictSyntaxAndTypedUnknownMembers(t *testing.T) {
	invalidUTF8 := append([]byte(`{"name":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	for _, test := range []struct {
		name  string
		input []byte
	}{
		{name: "malformed", input: []byte(`{"name":`)},
		{name: "unknown top-level member", input: []byte(`{"unknown":true}`)},
		{name: "unknown nested member", input: []byte(`{"nested":{"value":"ok","unknown":true}}`)},
		{name: "duplicate top-level member", input: []byte(`{"name":"first","name":"second"}`)},
		{name: "duplicate member in array", input: []byte(`{"items":[{"name":"first","name":"second"}]}`)},
		{name: "duplicate member in map", input: []byte(`{"fields":{"value":1,"value":2}}`)},
		{name: "duplicate member in raw projection", input: []byte(`{"raw":{"value":1,"value":2}}`)},
		{name: "inserted dash in typed member", input: []byte(`{"li-mit":1}`)},
		{name: "inserted underscore in typed member", input: []byte(`{"li_mit":1}`)},
		{name: "trailing value", input: []byte(`{"name":"ok"} {}`)},
		{name: "raw invalid UTF-8", input: invalidUTF8},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got decodeDocument
			if err := Decode(test.input, &got); err == nil {
				t.Fatalf("Decode(%q) succeeded: %#v", test.input, got)
			}
		})
	}
}

func TestDecodeAllowsMapAndRawProjectionMembers(t *testing.T) {
	input := []byte(`{"fields":{"producer_owned":7},"raw":{"producer_value" : true}}`)
	var got decodeDocument
	if err := Decode(input, &got); err != nil {
		t.Fatal(err)
	}
	if got.Fields["producer_owned"] != 7 {
		t.Fatalf("Decode() fields = %#v", got.Fields)
	}
	if string(got.Raw) != `{"producer_value" : true}` {
		t.Fatalf("Decode() raw projection = %s", got.Raw)
	}
}

func TestDecodeAcceptsOrdinaryJSONSpellingAndCaseAliases(t *testing.T) {
	input := []byte("{\n  \"LIMIT\": 255, \"NAME\": \"\\u0063af\\u00e9\"\n}")
	var got decodeDocument
	if err := Decode(input, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "café" || got.Limit != 255 {
		t.Fatalf("Decode() = %#v", got)
	}
}

func TestDecodeRetainsTypedNumericBounds(t *testing.T) {
	var got decodeDocument
	if err := Decode([]byte(`{"limit":256}`), &got); err == nil {
		t.Fatalf("Decode() accepted uint8 overflow: %#v", got)
	}
}

func TestDecodeAcceptsPairedAndRejectsUnpairedUTF16Surrogates(t *testing.T) {
	var got decodeDocument
	if err := Decode([]byte(`{"name":"\ud83d\ude00"}`), &got); err != nil {
		t.Fatalf("Decode() rejected a paired surrogate: %v", err)
	}
	if got.Name != "😀" {
		t.Fatalf("Decode() name = %q", got.Name)
	}
	if err := Decode([]byte(`{"name":"\ud800"}`), &got); err == nil || !strings.Contains(err.Error(), "surrogate") {
		t.Fatalf("Decode() unpaired-surrogate error = %v", err)
	}
}

func TestDecodeReportsLegacySyntaxErrors(t *testing.T) {
	var got decodeDocument
	err := Decode([]byte(`{"name":"value"|}`), &got)
	const want = "invalid character '|' after object key:value pair"
	if err == nil || err.Error() != want {
		t.Fatalf("Decode() error = %v, want %q", err, want)
	}
}
