package main

var fixtureTestConfiguration = testGenerationConfig("Fixture")

func testGenerationConfig(root string) generationConfig {
	return generationConfig{
		OutputRoot: "model",
		Layout:     newOutputLayout(root),
	}
}
