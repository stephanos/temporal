package api

const (
	sourcePublic    sourceGroup = "Public"
	sourceInternal  sourceGroup = "Internal"
	sourceExtension sourceGroup = "Extension"
	sourceExternal  sourceGroup = "External"
)

var fixtureTestConfiguration = testGenerationConfig("Fixture")

func testGenerationConfig(root string) generationConfig {
	layout := newOutputLayout(root)
	return generationConfig{
		Operation: "generate",
		Sources: []sourceRule{
			{Group: sourceInternal, Prefix: "internal/"},
			{Group: sourcePublic, Prefix: "public/"},
			{Group: sourceExtension, Prefix: "extensions/"},
		},
		Groups:        []sourceGroup{sourceExtension, sourceExternal, sourceInternal, sourcePublic},
		DefaultSource: sourceExternal,
		OutputRoot:    "model",
		Layout:        layout,
	}
}

func testCatalogPath(configuration generationConfig, group sourceGroup) string {
	return newSourceModuleSpec(configuration.Layout, group).catalogModule.Path
}

func testGRPCPath(configuration generationConfig, group sourceGroup) string {
	return newSourceModuleSpec(configuration.Layout, group).grpcModule.Path
}
