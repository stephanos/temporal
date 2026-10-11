package architecture

import (
	"strings"
	"testing"
)

func TestJSONCallbacksFlatEmbedUniqueNames(t *testing.T) {
	for _, decoding := range []bool{false, true} {
		program := effectResolverProgram(t, "package record\ntype Inner struct{Type string `json:\"type\"`; ID uint64 `json:\"id\"`}\ntype Dirty struct{Inner; Local string `json:\"local_state\"`; Transcript []byte `json:\"transcript_base64\"`}\nfunc Root(){}\n")
		root := program.Functions["example.invalid/resolver/record.Root"]
		analysis := effectResolverAnalysis(program)
		analysis.jsonCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), decoding, root.declaration.Pos())
		if len(analysis.findings) != 0 {
			t.Fatalf("finite disjoint flat embed decoding=%v rejected: %v", decoding, analysis.findings)
		}
	}
}

func TestJSONCallbacksFlatEmbedRejectsUnsupportedLayouts(t *testing.T) {
	for _, tc := range []struct{ name, fields, inner, extra string }{
		{"exact-collision", "Inner; Other string `json:\"same\"`", "Value string `json:\"same\"`", ""},
		{"folded-collision", "Inner; Other string `json:\"K\"`", "Value string `json:\"K\"`", ""},
		{"tagged-embed", "Inner `json:\"named\"`", "Value string", ""},
		{"pointer-embed", "*Inner", "Value string", ""},
		{"nested-embed", "Inner", "Nested", "type Nested struct{Value string}\n"},
		{"unknown-option", "Inner", "Value string `json:\"value,custom\"`", ""},
		{"malformed-tag", "Inner", "Value string `json:\"unterminated`", ""},
		{"ambiguous-codec", "Inner; Other", "Value string", "type Other struct{}\nfunc(Inner)MarshalJSON()([]byte,error){return nil,nil}\nfunc(Other)MarshalJSON()([]byte,error){return nil,nil}\n"},
		{"codec-field-shadow", "Inner; MarshalJSON string", "Value string", "func(Inner)MarshalJSON()([]byte,error){return nil,nil}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\ntype Inner struct{"+tc.inner+"}\ntype Dirty struct{"+tc.fields+"}\nfunc Root(){}\n"+tc.extra)
			root := program.Functions["example.invalid/resolver/record.Root"]
			analysis := effectResolverAnalysis(program)
			analysis.jsonCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), false, root.declaration.Pos())
			found := false
			for _, finding := range analysis.findings {
				found = found || finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "JSON")
			}
			if !found {
				t.Fatalf("unsupported embedded layout was accepted: %v", analysis.findings)
			}
		})
	}
}
