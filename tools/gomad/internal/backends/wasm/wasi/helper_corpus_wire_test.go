package wasi

import (
	"bytes"
	"testing"
)

func TestHelperCorpusRawWirePreservesDuplicateKeys(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		frame any
		want  string
	}{
		{"frame", map[string]any{"raw_json": `{"sequence":0,"sequence":1}`}, `{"sequence":0,"sequence":1}`},
		{"nested", map[string]any{"descriptor": map[string]any{"raw_json": `{"chunks":1,"chunks":2}`}}, `{"descriptor":{"chunks":1,"chunks":2}}`},
		{"tagged", map[string]any{"instance": "a", "reply": map[string]any{"raw_json": `{"id":1,"id":2}`}}, `{"instance":"a","reply":{"id":1,"id":2}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := helperCorpusLine(test.frame)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(test.want)) {
				t.Fatalf("wire: got %s, want %s", got, test.want)
			}
		})
	}
}

func TestHelperCorpusRawWireRefusesExtraFrames(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		frame any
	}{
		{"sibling", map[string]any{"raw_json": `{}`, "sequence": 1}},
		{"multiple", map[string]any{"raw_json": `{} {}`}},
		{"newline", map[string]any{"raw_json": "{\n}"}},
		{"carriage-return", map[string]any{"raw_json": "{\r}"}},
		{"scalar", map[string]any{"raw_json": `1`}},
		{"array", map[string]any{"raw_json": `[]`}},
		{"non-string", map[string]any{"raw_json": 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got, err := helperCorpusLine(test.frame); err == nil {
				t.Fatalf("accepted malformed wire fixture as %s", got)
			}
		})
	}
}
