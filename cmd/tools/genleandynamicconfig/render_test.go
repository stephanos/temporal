package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderCatalogIsCompleteAndIndependentOfInputOrder(t *testing.T) {
	t.Parallel()
	catalog := renderFixtureCatalog()
	shuffled := renderFixtureCatalog()
	slices.Reverse(shuffled.Settings)
	slices.Reverse(shuffled.Settings[1].Provenance)
	slices.Reverse(shuffled.Settings[1].Schema.Fields)
	slices.Reverse(shuffled.Settings[1].Default.Value.Fields)
	slices.Reverse(shuffled.Fixtures)
	slices.Reverse(shuffled.Fixtures[1].Overrides)

	first, err := renderCatalog(catalog)
	require.NoError(t, err)
	second, err := renderCatalog(shuffled)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, []string{
		"Temporal/DynamicConfig.lean",
		"Temporal/DynamicConfig/Settings.lean",
		"Temporal/DynamicConfig/Types.lean",
	}, sortedArtifactPaths(first))
	require.Contains(t, string(first["Temporal/DynamicConfig.lean"]), "import Temporal.DynamicConfig.Settings")
	require.Contains(t, string(first["Temporal/DynamicConfig/Types.lean"]), "inductive CanonicalValue")
	settings := string(first["Temporal/DynamicConfig/Settings.lean"])
	require.Contains(t, settings, "def a_b : Setting")
	require.Contains(t, settings, "def z_setting : Setting")
	require.Contains(t, settings, "def fixtures : List ResolutionFixture")
	require.Less(t, strings.Index(settings, "def a_b : Setting"), strings.Index(settings, "def z_setting : Setting"))
}

func TestRenderCatalogRejectsInvalidLeanEncodings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Catalog)
		want   string
	}{
		{
			name: "invalid boolean",
			mutate: func(catalog *Catalog) {
				catalog.Settings[1].Default.Value.Scalar = "TRUE"
			},
			want: "invalid bool",
		},
		{
			name: "identifier collision",
			mutate: func(catalog *Catalog) {
				duplicate := catalog.Settings[0]
				duplicate.Key = "a-b"
				catalog.Settings = append(catalog.Settings, duplicate)
			},
			want: "Lean identifier",
		},
		{
			name: "unknown value kind",
			mutate: func(catalog *Catalog) {
				catalog.Fixtures[0].Result.Kind = "unknown"
			},
			want: "unknown canonical value kind",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := renderFixtureCatalog()
			tt.mutate(&catalog)
			_, err := renderCatalog(catalog)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestRenderedProductionCatalogIsStableAndLeanElaborates(t *testing.T) {
	moduleRoot, err := findModuleRoot()
	require.NoError(t, err)
	catalog, err := run(context.Background(), moduleRoot)
	require.NoError(t, err)
	require.NotEmpty(t, catalog.Settings)
	artifacts, err := renderCatalog(catalog)
	require.NoError(t, err)
	repeated, err := renderCatalog(catalog)
	require.NoError(t, err)
	require.Equal(t, artifacts, repeated)
	candidateRoot := t.TempDir()
	for path, encoded := range artifacts {
		absolute := filepath.Join(candidateRoot, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0o700))
		require.NoError(t, os.WriteFile(absolute, encoded, 0o600))
	}
	require.NoError(t, validateLeanCandidate(context.Background(), moduleRoot, candidateRoot))
}

func renderFixtureCatalog() Catalog {
	shard := int32(7)
	return Catalog{
		Identity: "sha256:catalog",
		Settings: []ProjectedSetting{
			{
				Key:         "z.setting",
				Description: "structured\nsetting",
				Policy:      PolicyShardID,
				Schema: ValueSchema{
					Kind:   SchemaStruct,
					GoType: "fixture.Struct",
					Fields: []SchemaField{
						{Name: "z", Schema: ValueSchema{Kind: SchemaString, GoType: "string"}},
						{Name: "a", Schema: ValueSchema{Kind: SchemaInt, GoType: "int"}},
					},
				},
				Codec: CodecClass("structure"),
				Default: ProjectedDefault{
					Kind: DefaultConcrete,
					Value: &CanonicalValue{Kind: ValueObject, Fields: []CanonicalField{
						{Name: "z", Value: CanonicalValue{Kind: ValueString, Scalar: "last"}},
						{Name: "a", Value: CanonicalValue{Kind: ValueInt, Scalar: "1"}},
					}},
				},
				Provenance: []RegistrationSite{
					{Key: "z.setting", Package: "z/package", File: "z.go", Line: 9},
					{Key: "z.setting", Package: "a/package", File: "a.go", Line: 3},
				},
				Identity: "sha256:z",
			},
			{
				Key:         "a_b",
				Description: "boolean setting",
				Policy:      PolicyGlobal,
				Schema:      ValueSchema{Kind: SchemaBool, GoType: "bool"},
				Codec:       CodecClass("bool"),
				Default: ProjectedDefault{
					Kind:  DefaultConcrete,
					Value: &CanonicalValue{Kind: ValueBool, Scalar: "true"},
				},
				Provenance: []RegistrationSite{{Key: "a_b", Package: "a/package", File: "a.go", Line: 2}},
				Identity:   "sha256:a",
			},
		},
		Fixtures: []ResolutionFixture{
			{
				Name:       "z-fixture",
				Policy:     PolicyShardID,
				SettingKey: "z.setting",
				Context:    ExactConstraints{ShardID: &shard},
				Overrides: []FixtureOverride{
					{Constraints: ExactConstraints{}, Value: CanonicalValue{Kind: ValueObject, Fields: []CanonicalField{
						{Name: "a", Value: CanonicalValue{Kind: ValueInt, Scalar: "2"}},
						{Name: "z", Value: CanonicalValue{Kind: ValueString, Scalar: "fallback"}},
					}}},
					{Constraints: ExactConstraints{ShardID: &shard}, Value: CanonicalValue{Kind: ValueObject, Fields: []CanonicalField{
						{Name: "a", Value: CanonicalValue{Kind: ValueInt, Scalar: "3"}},
						{Name: "z", Value: CanonicalValue{Kind: ValueString, Scalar: "specific"}},
					}}},
				},
				SelectedSource:     SourceOverride,
				SelectedConstraint: ExactConstraints{ShardID: &shard},
				Result: CanonicalValue{Kind: ValueObject, Fields: []CanonicalField{
					{Name: "z", Value: CanonicalValue{Kind: ValueString, Scalar: "specific"}},
					{Name: "a", Value: CanonicalValue{Kind: ValueInt, Scalar: "3"}},
				}},
			},
			{
				Name:               "a-fixture",
				Policy:             PolicyGlobal,
				SettingKey:         "a_b",
				Context:            ExactConstraints{},
				SelectedSource:     SourceSimpleDefault,
				SelectedConstraint: ExactConstraints{},
				Result:             CanonicalValue{Kind: ValueBool, Scalar: "true"},
			},
		},
	}
}

func sortedArtifactPaths(artifacts map[string][]byte) []string {
	paths := make([]string, 0, len(artifacts))
	for path := range artifacts {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}
