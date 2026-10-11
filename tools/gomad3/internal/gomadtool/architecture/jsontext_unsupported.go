package architecture

import "go/types"

// These exact native operations are explicitly unsupported, not memory summaries.
// Their source-qualified handling cannot grant even an owned-looking receiver
// permission to read/write/reset a JSONtext stream.
var jsontextUnsupportedSymbols = symbolSet(map[string][]string{
	"encoding/json/jsontext.*Encoder": {"WriteToken", "WriteValue", "Reset"},
	"encoding/json/jsontext.*Decoder": {"ReadToken", "ReadValue", "SkipValue", "Reset"},
})

func jsonUnsupportedResults(results *types.Tuple) *abstractValue {
	value := tupleValue(results)
	var mark func(*abstractValue)
	mark = func(value *abstractValue) {
		if value == nil {
			return
		}
		value.unknown = true
		for _, child := range value.fields {
			mark(child)
		}
	}
	mark(value)
	return value
}
