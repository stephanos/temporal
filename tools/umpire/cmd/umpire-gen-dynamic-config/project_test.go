package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/dynamicconfig"
)

func TestProjectMetadataIsCanonicalAndPreservesOpaqueDefaults(t *testing.T) {
	metadata := []dynamicconfig.SettingMetadata{
		{
			Key:         "Zeta.Structure",
			Description: "documentation is retained but not identity-bearing",
			Precedence:  dynamicconfig.PrecedenceTaskQueue,
			ResultType:  reflect.TypeFor[map[string]any](),
			Codec:       dynamicconfig.SettingCodecMap,
			Default: dynamicconfig.SettingDefaultMetadata{
				Kind: dynamicconfig.SettingDefaultConstrained,
				Constrained: []dynamicconfig.ConstrainedDefaultMetadata{
					{
						Constraints: dynamicconfig.Constraints{},
						Default: dynamicconfig.SettingDefaultMetadata{
							Kind:  dynamicconfig.SettingDefaultConcrete,
							Value: map[string]any{"z": 2, "a": []any{"x", true}},
						},
					},
					{
						Constraints: dynamicconfig.Constraints{TaskQueueName: "queue"},
						Default: dynamicconfig.SettingDefaultMetadata{
							Kind:  dynamicconfig.SettingDefaultConcrete,
							Value: map[string]any{"specific": 1},
						},
					},
				},
			},
		},
		{
			Key:         "alpha.opaque",
			Description: "opaque",
			Precedence:  dynamicconfig.PrecedenceGlobal,
			ResultType:  reflect.TypeFor[func()](),
			Codec:       dynamicconfig.SettingCodecCustom,
			Default: dynamicconfig.SettingDefaultMetadata{
				Kind: dynamicconfig.SettingDefaultOpaque,
				Opaque: dynamicconfig.OpaqueDefaultMetadata{
					ResultType: reflect.TypeFor[func()](),
					Reason:     "default contains unsupported func value at <root>",
				},
			},
		},
	}

	first, err := projectMetadata(metadata)
	require.NoError(t, err)
	reversed := slices.Clone(metadata)
	slices.Reverse(reversed)
	second, err := projectMetadata(reversed)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, []string{"alpha.opaque", "zeta.structure"}, projectedKeys(first.Settings))
	require.Equal(t, []CanonicalField{
		{Name: "a", Value: CanonicalValue{Kind: ValueList, Items: []CanonicalValue{
			{Kind: ValueString, Scalar: "x"},
			{Kind: ValueBool, Scalar: "true"},
		}}},
		{Name: "z", Value: CanonicalValue{Kind: ValueInt, Scalar: "2"}},
	}, first.Settings[1].Default.Constrained[0].Value.Fields)
	require.Equal(t, ProjectedOpaqueDefault{
		GoType: "func()",
		Reason: "default contains unsupported func value at <root>",
	}, *first.Settings[0].Default.Opaque)

	changedDescription := slices.Clone(metadata)
	changedDescription[0].Description = "new prose"
	descriptionOnly, err := projectMetadata(changedDescription)
	require.NoError(t, err)
	require.Equal(t, first.Settings[1].Identity, descriptionOnly.Settings[1].Identity)
}

func TestProjectMetadataRejectsInvalidCatalogInputs(t *testing.T) {
	valid := func() dynamicconfig.SettingMetadata {
		return dynamicconfig.SettingMetadata{
			Key:         "valid.key",
			Description: "valid",
			Precedence:  dynamicconfig.PrecedenceGlobal,
			ResultType:  reflect.TypeFor[int](),
			Codec:       dynamicconfig.SettingCodecInt,
			Default: dynamicconfig.SettingDefaultMetadata{
				Kind:  dynamicconfig.SettingDefaultConcrete,
				Value: 1,
			},
		}
	}
	tests := []struct {
		name     string
		metadata func() []dynamicconfig.SettingMetadata
		contains string
	}{
		{
			name: "duplicate normalized key",
			metadata: func() []dynamicconfig.SettingMetadata {
				first := valid()
				second := valid()
				second.Key = "VALID.KEY"
				return []dynamicconfig.SettingMetadata{first, second}
			},
			contains: `projection setting "valid.key": duplicate normalized key`,
		},
		{
			name: "unknown policy",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.Precedence = dynamicconfig.Precedence(99)
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: `projection setting "valid.key": unknown precedence 99`,
		},
		{
			name: "illegal constraint shape",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.Precedence = dynamicconfig.PrecedenceNamespace
				value.Default = dynamicconfig.SettingDefaultMetadata{
					Kind: dynamicconfig.SettingDefaultConstrained,
					Constrained: []dynamicconfig.ConstrainedDefaultMetadata{
						{
							Constraints: dynamicconfig.Constraints{ShardID: 1},
							Default:     dynamicconfig.SettingDefaultMetadata{Kind: dynamicconfig.SettingDefaultConcrete, Value: 1},
						},
						{
							Default: dynamicconfig.SettingDefaultMetadata{Kind: dynamicconfig.SettingDefaultConcrete, Value: 2},
						},
					},
				}
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: `constraint {shard_id=1} is illegal for namespace precedence`,
		},
		{
			name: "duplicate exact constraint",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.Default = dynamicconfig.SettingDefaultMetadata{
					Kind: dynamicconfig.SettingDefaultConstrained,
					Constrained: []dynamicconfig.ConstrainedDefaultMetadata{
						{Default: dynamicconfig.SettingDefaultMetadata{Kind: dynamicconfig.SettingDefaultConcrete, Value: 1}},
						{Default: dynamicconfig.SettingDefaultMetadata{Kind: dynamicconfig.SettingDefaultConcrete, Value: 2}},
					},
				}
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: "duplicate exact constraint {}",
		},
		{
			name: "incoherent constrained default",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.Default = dynamicconfig.SettingDefaultMetadata{
					Kind: dynamicconfig.SettingDefaultConstrained,
					Constrained: []dynamicconfig.ConstrainedDefaultMetadata{{
						Constraints: dynamicconfig.Constraints{Namespace: "only-specific"},
						Default:     dynamicconfig.SettingDefaultMetadata{Kind: dynamicconfig.SettingDefaultConcrete, Value: 1},
					}},
				}
				value.Precedence = dynamicconfig.PrecedenceNamespace
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: "constrained default has no unconstrained fallback",
		},
		{
			name: "unsupported canonical value",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.ResultType = reflect.TypeFor[func()]()
				value.Codec = dynamicconfig.SettingCodecCustom
				value.Default.Value = func() {}
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: "unsupported concrete func value",
		},
		{
			name: "nondeterministic float",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.ResultType = reflect.TypeFor[float64]()
				value.Codec = dynamicconfig.SettingCodecFloat
				value.Default.Value = math.NaN()
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: "non-finite float",
		},
		{
			name: "nondeterministic map key",
			metadata: func() []dynamicconfig.SettingMetadata {
				value := valid()
				value.ResultType = reflect.TypeFor[map[int]string]()
				value.Codec = dynamicconfig.SettingCodecStructure
				value.Default.Value = map[int]string{1: "one"}
				return []dynamicconfig.SettingMetadata{value}
			},
			contains: "map key type int is not canonical",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := projectMetadata(test.metadata())
			require.ErrorContains(t, err, test.contains)
			require.Equal(t, Catalog{}, catalog)
		})
	}
}

func TestReconcileDiscoveryRejectsIncompleteCatalogs(t *testing.T) {
	setting := ProjectedSetting{Key: "registered.key"}
	tests := []struct {
		name     string
		catalog  Catalog
		sites    []RegistrationSite
		contains string
	}{
		{
			name:     "zero catalog",
			contains: "reconcile: initialized registry catalog is empty",
		},
		{
			name:    "discovered but unregistered",
			catalog: Catalog{Settings: []ProjectedSetting{setting}},
			sites: []RegistrationSite{
				{Key: "registered.key", Package: "example/registered", File: "registered.go", Line: 10},
				{Key: "missing.key", Package: "example/missing", File: "missing.go", Line: 20},
			},
			contains: `reconcile package "example/missing" key "missing.key": discovered initializer was not registered`,
		},
		{
			name:     "registry setting missing discovery",
			catalog:  Catalog{Settings: []ProjectedSetting{setting}},
			contains: `reconcile setting "registered.key": initialized registry entry has no production initializer discovery`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := reconcileDiscovery(test.catalog, test.sites)
			require.ErrorContains(t, err, test.contains)
			require.Equal(t, Catalog{}, catalog)
		})
	}
}

func TestDiscoverRegistrationSitesUsesTypedProductionFiles(t *testing.T) {
	root := filepath.Join("testdata", "discovery", "valid")
	sites, err := discoverRegistrationSites(context.Background(), root)
	require.NoError(t, err)
	require.Equal(t, []RegistrationSite{
		{Key: "alpha.key", Package: "go.temporal.io/server/production", File: "production/config.go", Line: 5},
		{Key: "beta.key", Package: "go.temporal.io/server/production", File: "production/config.go", Line: 8},
	}, sites)
}

func TestDiscoverRegistrationSitesRejectsLoadAndTypeErrors(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		contains string
	}{
		{
			name:     "unloadable",
			root:     filepath.Join("testdata", "discovery", "missing"),
			contains: "discovery load",
		},
		{
			name:     "ill typed",
			root:     filepath.Join("testdata", "discovery", "illtyped"),
			contains: `discovery package "go.temporal.io/server/production"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sites, err := discoverRegistrationSites(context.Background(), test.root)
			require.ErrorContains(t, err, test.contains)
			require.Empty(t, sites)
		})
	}
}

func TestRegistryHelperCleansSourceOnEveryPath(t *testing.T) {
	validCatalog, err := json.Marshal(Catalog{})
	require.NoError(t, err)
	tests := []struct {
		name     string
		stdout   []byte
		stderr   []byte
		runErr   error
		contains string
	}{
		{name: "success", stdout: validCatalog},
		{name: "build or run failure", stderr: []byte("compile failed"), runErr: errors.New("exit 1"), contains: "helper run: exit 1: compile failed"},
		{name: "initialization panic", stderr: []byte("panic: duplicate registration"), runErr: errors.New("exit 2"), contains: "helper initialization panic: duplicate registration"},
		{name: "malformed output", stdout: []byte("{"), contains: "helper decode catalog"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helperDirectory := t.TempDir()
			runner := func(_ context.Context, moduleRoot string, arguments ...string) ([]byte, []byte, error) {
				require.Equal(t, helperDirectory, moduleRoot)
				require.Equal(t, []string{
					"run",
					"-tags=test_dep",
					"./tools/umpire/cmd/umpire-gen-dynamic-config",
					registryHelperArgument,
				}, arguments)
				return test.stdout, test.stderr, test.runErr
			}
			catalog, err := runRegistryHelper(
				context.Background(),
				helperDirectory,
				helperDirectory,
				[]string{"go.temporal.io/server/production"},
				runner,
			)
			if test.contains == "" {
				require.NoError(t, err)
				require.Equal(t, Catalog{}, catalog)
			} else {
				require.ErrorContains(t, err, test.contains)
				require.Equal(t, Catalog{}, catalog)
			}
			matches, globErr := filepath.Glob(filepath.Join(helperDirectory, "zz_umpire_gen_dynamic_config_helper_*.go"))
			require.NoError(t, globErr)
			require.Empty(t, matches)
		})
	}
}

func TestFixtureSelectionRejectsIncorrectSourceAndConstraint(t *testing.T) {
	namespaceName := "fixture-namespace"
	global := ExactConstraints{}
	namespaceOnly := ExactConstraints{Namespace: &namespaceName}
	defaultValue := boolValue(true)
	setting := ProjectedSetting{
		Key: "fixture.key",
		Default: ProjectedDefault{
			Kind:  DefaultConcrete,
			Value: &defaultValue,
		},
	}
	fixture := newFixture(
		"selection",
		PolicyNamespace,
		setting.Key,
		namespaceOnly,
		[]FixtureOverride{{Constraints: global, Value: boolValue(false)}},
		SourceOverride,
		global,
		boolValue(false),
	)

	t.Run("source", func(t *testing.T) {
		fixture := fixture
		fixture.SelectedSource = SourceSimpleDefault
		err := validateFixtureSelection(fixture, boolValue(false), setting)
		require.ErrorContains(t, err, "selected source")
	})

	t.Run("constraint", func(t *testing.T) {
		fixture := fixture
		fixture.SelectedConstraint = namespaceOnly
		err := validateFixtureSelection(fixture, boolValue(false), setting)
		require.ErrorContains(t, err, "selected constraint")
	})
}

func TestFixtureSelectionRejectsIndistinguishableCandidates(t *testing.T) {
	global := ExactConstraints{}
	defaultValue := boolValue(true)
	setting := ProjectedSetting{
		Key: "fixture.key",
		Default: ProjectedDefault{
			Kind:  DefaultConcrete,
			Value: &defaultValue,
		},
	}
	fixture := newFixture(
		"ambiguous",
		PolicyGlobal,
		setting.Key,
		global,
		[]FixtureOverride{{Constraints: global, Value: boolValue(true)}},
		SourceOverride,
		global,
		boolValue(true),
	)

	err := validateFixtureSelection(fixture, boolValue(true), setting)
	require.ErrorContains(t, err, "indistinguishable candidates")
}

func TestProductionFixturesCoverEveryPolicyAndResolutionBoundary(t *testing.T) {
	catalog := Catalog{Fixtures: productionFixtureShape()}
	require.NoError(t, validateFixtures(catalog.Fixtures))

	policies := make([]PrecedencePolicy, 0, len(catalog.Fixtures))
	for _, fixture := range catalog.Fixtures {
		policies = append(policies, fixture.Policy)
		require.NotEmpty(t, fixture.SettingKey)
		require.NotEqual(t, CanonicalValue{}, fixture.Result)
	}
	slices.Sort(policies)
	policies = slices.Compact(policies)
	require.Equal(t, allPrecedencePolicies(), policies)
	require.True(t, hasFixture(catalog.Fixtures, "task-queue-constrained-default-before-namespace-override"))
	require.True(t, hasFixture(catalog.Fixtures, "task-queue-specific-override-before-constrained-default"))
	require.True(t, hasFixture(catalog.Fixtures, "namespace-unconstrained-fallback"))
	require.True(t, hasFixture(catalog.Fixtures, "destination-specific"))

	for _, fixture := range catalog.Fixtures {
		encoded, err := json.Marshal(fixture.Context)
		require.NoError(t, err)
		var fields map[string]any
		require.NoError(t, json.Unmarshal(encoded, &fields))
		require.Len(t, fields, 8)
	}
}

func TestRunProjectsInitializedProductionRegistry(t *testing.T) {
	moduleRoot, err := findModuleRoot()
	require.NoError(t, err)
	catalog, err := run(context.Background(), moduleRoot)
	require.NoError(t, err)
	require.NotEmpty(t, catalog.Identity)
	require.NotEmpty(t, catalog.Settings)
	require.Equal(t, productionFixtureShape(), catalog.Fixtures)
	for _, setting := range catalog.Settings {
		require.NotEmpty(t, setting.Provenance)
	}
}

func TestCanonicalDurationUsesExactNanoseconds(t *testing.T) {
	value, err := canonicalValue(reflect.ValueOf(1500*time.Millisecond), reflect.TypeFor[time.Duration]())
	require.NoError(t, err)
	require.Equal(t, CanonicalValue{Kind: ValueDuration, Scalar: "1500000000"}, value)
}

func projectedKeys(settings []ProjectedSetting) []string {
	result := make([]string, len(settings))
	for index, setting := range settings {
		result[index] = setting.Key
	}
	return result
}

func hasFixture(fixtures []ResolutionFixture, name string) bool {
	return slices.ContainsFunc(fixtures, func(fixture ResolutionFixture) bool {
		return fixture.Name == name
	})
}
