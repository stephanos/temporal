package api

const (
	sourcePublic   sourceGroup = "Public"
	sourceInternal sourceGroup = "Internal"
	sourceCHASM    sourceGroup = "CHASM"
	sourceExternal sourceGroup = "External"
)

var temporalTestConfiguration = testGenerationConfig("Temporal")

func testGenerationConfig(root string) generationConfig {
	layout := newOutputLayout(root)
	return generationConfig{
		Operation: "generate",
		Sources: []sourceRule{
			{Group: sourceInternal, Prefix: "temporal/server/api/"},
			{Group: sourcePublic, Prefix: "temporal/api/"},
			{Group: sourceCHASM, Prefix: "chasm/lib/"},
			{Group: sourceInternal, Prefix: "internal/"},
			{Group: sourcePublic, Prefix: "public/"},
		},
		Groups:        []sourceGroup{sourceCHASM, sourceExternal, sourceInternal, sourcePublic},
		DefaultSource: sourceExternal,
		OutputRoot:    "model",
		Layout:        layout,
	}
}
