package dynamicconfig

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	enumsspb "go.temporal.io/server/api/enums/v1"
)

type metadataTestDefault struct {
	Enabled bool
	Names   []string
	Values  map[string]int
}

type metadataOpaqueDefault struct {
	Callback func()
}

func resetMetadataTestRegistry(t *testing.T) {
	t.Helper()
	ResetRegistryForTest()
	t.Cleanup(ResetRegistryForTest)
}

func TestRegisteredSettingMetadataRecordsAllPrecedences(t *testing.T) {
	resetMetadataTestRegistry(t)

	tests := []struct {
		name       string
		precedence Precedence
		newSetting func(string, string) GenericSetting
	}{
		{
			name:       "chasm-task-type",
			precedence: PrecedenceChasmTaskType,
			newSetting: func(key string, description string) GenericSetting {
				return NewChasmTaskTypeBoolSetting(key, false, description)
			},
		},
		{
			name:       "destination",
			precedence: PrecedenceDestination,
			newSetting: func(key string, description string) GenericSetting {
				return NewDestinationBoolSetting(key, false, description)
			},
		},
		{
			name:       "global",
			precedence: PrecedenceGlobal,
			newSetting: func(key string, description string) GenericSetting {
				return NewGlobalBoolSetting(key, false, description)
			},
		},
		{
			name:       "namespace",
			precedence: PrecedenceNamespace,
			newSetting: func(key string, description string) GenericSetting {
				return NewNamespaceBoolSetting(key, false, description)
			},
		},
		{
			name:       "namespace-id",
			precedence: PrecedenceNamespaceID,
			newSetting: func(key string, description string) GenericSetting {
				return NewNamespaceIDBoolSetting(key, false, description)
			},
		},
		{
			name:       "shard-id",
			precedence: PrecedenceShardID,
			newSetting: func(key string, description string) GenericSetting {
				return NewShardIDBoolSetting(key, false, description)
			},
		},
		{
			name:       "task-queue",
			precedence: PrecedenceTaskQueue,
			newSetting: func(key string, description string) GenericSetting {
				return NewTaskQueueBoolSetting(key, false, description)
			},
		},
		{
			name:       "task-type",
			precedence: PrecedenceTaskType,
			newSetting: func(key string, description string) GenericSetting {
				return NewTaskTypeBoolSetting(key, false, description)
			},
		},
	}

	for i := len(tests) - 1; i >= 0; i-- {
		test := tests[i]
		key := "Metadata." + test.name
		description := test.name + " description"
		test.newSetting(key, description)
	}

	want := make([]SettingMetadata, 0, len(tests))
	for _, test := range tests {
		description := test.name + " description"
		want = append(want, SettingMetadata{
			Key:         "metadata." + test.name,
			Description: description,
			Precedence:  test.precedence,
			ResultType:  reflect.TypeFor[bool](),
			Codec:       SettingCodecBool,
			Default: SettingDefaultMetadata{
				Kind:  SettingDefaultConcrete,
				Value: false,
			},
		})
	}

	metadata, err := RegisteredSettingMetadata()
	require.NoError(t, err)
	require.Equal(t, want, metadata)
}

func TestRegisteredSettingMetadataRecordsConstructorFamilies(t *testing.T) {
	resetMetadataTestRegistry(t)

	mapDefault := map[string]any{"nested": []string{"original"}}
	structuredDefault := metadataTestDefault{
		Enabled: true,
		Names:   []string{"first"},
		Values:  map[string]int{"one": 1},
	}
	constraints := Constraints{
		Namespace:     "namespace",
		NamespaceID:   "namespace-id",
		TaskQueueName: "task-queue",
		Destination:   "destination",
		ChasmTaskType: "chasm-task-type",
		TaskQueueType: enumspb.TASK_QUEUE_TYPE_WORKFLOW,
		ShardID:       12,
		TaskType:      enumsspb.TASK_TYPE_TRANSFER_ACTIVITY_TASK,
	}

	NewGlobalIntSetting("metadata.int", 7, "int")
	NewGlobalFloatSetting("metadata.float", 1.5, "float")
	NewGlobalStringSetting("metadata.string", "value", "string")
	NewGlobalDurationSetting("metadata.duration", 3*time.Second, "duration")
	NewGlobalMapSetting("metadata.map", mapDefault, "map")
	NewGlobalTypedSetting("metadata.structure", structuredDefault, "structure")
	NewGlobalTypedSettingWithConverter(
		"metadata.custom",
		func(value any) (metadataTestDefault, error) {
			converted, ok := value.(metadataTestDefault)
			if !ok {
				return metadataTestDefault{}, errors.New("not metadataTestDefault")
			}
			return converted, nil
		},
		metadataTestDefault{Enabled: true},
		"custom",
	)
	NewTaskQueueIntSettingWithConstrainedDefault(
		"metadata.constrained",
		[]TypedConstrainedValue[int]{{Constraints: constraints, Value: 11}},
		"constrained",
	)
	NewGlobalTypedSettingWithConverter(
		"metadata.opaque",
		func(value any) (metadataOpaqueDefault, error) {
			converted, ok := value.(metadataOpaqueDefault)
			if !ok {
				return metadataOpaqueDefault{}, errors.New("not metadataOpaqueDefault")
			}
			return converted, nil
		},
		metadataOpaqueDefault{Callback: func() {}},
		"opaque",
	)

	mapDefault["nested"].([]string)[0] = "mutated"
	structuredDefault.Names[0] = "mutated"
	structuredDefault.Values["one"] = 2
	constraints.Namespace = "mutated"

	metadata, err := RegisteredSettingMetadata()
	require.NoError(t, err)
	require.Equal(t, []SettingMetadata{
		{
			Key:         "metadata.constrained",
			Description: "constrained",
			Precedence:  PrecedenceTaskQueue,
			ResultType:  reflect.TypeFor[int](),
			Codec:       SettingCodecInt,
			Default: SettingDefaultMetadata{
				Kind: SettingDefaultConstrained,
				Constrained: []ConstrainedDefaultMetadata{{
					Constraints: Constraints{
						Namespace:     "namespace",
						NamespaceID:   "namespace-id",
						TaskQueueName: "task-queue",
						Destination:   "destination",
						ChasmTaskType: "chasm-task-type",
						TaskQueueType: enumspb.TASK_QUEUE_TYPE_WORKFLOW,
						ShardID:       12,
						TaskType:      enumsspb.TASK_TYPE_TRANSFER_ACTIVITY_TASK,
					},
					Default: SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: 11},
				}},
			},
		},
		{
			Key:         "metadata.custom",
			Description: "custom",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[metadataTestDefault](),
			Codec:       SettingCodecCustom,
			Default: SettingDefaultMetadata{
				Kind:  SettingDefaultConcrete,
				Value: metadataTestDefault{Enabled: true},
			},
		},
		{
			Key:         "metadata.duration",
			Description: "duration",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[time.Duration](),
			Codec:       SettingCodecDuration,
			Default:     SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: 3 * time.Second},
		},
		{
			Key:         "metadata.float",
			Description: "float",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[float64](),
			Codec:       SettingCodecFloat,
			Default:     SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: 1.5},
		},
		{
			Key:         "metadata.int",
			Description: "int",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[int](),
			Codec:       SettingCodecInt,
			Default:     SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: 7},
		},
		{
			Key:         "metadata.map",
			Description: "map",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[map[string]any](),
			Codec:       SettingCodecMap,
			Default: SettingDefaultMetadata{
				Kind: SettingDefaultConcrete,
				Value: map[string]any{
					"nested": []string{"original"},
				},
			},
		},
		{
			Key:         "metadata.opaque",
			Description: "opaque",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[metadataOpaqueDefault](),
			Codec:       SettingCodecCustom,
			Default: SettingDefaultMetadata{
				Kind: SettingDefaultOpaque,
				Opaque: OpaqueDefaultMetadata{
					ResultType: reflect.TypeFor[metadataOpaqueDefault](),
					Reason:     "default contains unsupported func value at .Callback",
				},
			},
		},
		{
			Key:         "metadata.string",
			Description: "string",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[string](),
			Codec:       SettingCodecString,
			Default:     SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: "value"},
		},
		{
			Key:         "metadata.structure",
			Description: "structure",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[metadataTestDefault](),
			Codec:       SettingCodecStructure,
			Default: SettingDefaultMetadata{
				Kind: SettingDefaultConcrete,
				Value: metadataTestDefault{
					Enabled: true,
					Names:   []string{"first"},
					Values:  map[string]int{"one": 1},
				},
			},
		},
	}, metadata)
}

func TestRegisteredSettingMetadataReturnsDeepCopies(t *testing.T) {
	resetMetadataTestRegistry(t)

	NewGlobalMapSetting(
		"Metadata.Mutable",
		map[string]any{"slice": []string{"original"}},
		"original description",
	)
	NewNamespaceStringSettingWithConstrainedDefault(
		"Metadata.Constrained",
		[]TypedConstrainedValue[string]{{
			Constraints: Constraints{Namespace: "original namespace"},
			Value:       "original value",
		}},
		"constrained description",
	)

	first, err := RegisteredSettingMetadata()
	require.NoError(t, err)
	first[0].Description = "mutated description"
	first[0].Default.Constrained[0].Constraints.Namespace = "mutated namespace"
	first[0].Default.Constrained[0].Default.Value = "mutated value"
	first[1].Default.Value.(map[string]any)["slice"].([]string)[0] = "mutated"
	first = append(first, SettingMetadata{Key: "injected"})

	second, err := RegisteredSettingMetadata()
	require.NoError(t, err)
	require.Equal(t, []SettingMetadata{
		{
			Key:         "metadata.constrained",
			Description: "constrained description",
			Precedence:  PrecedenceNamespace,
			ResultType:  reflect.TypeFor[string](),
			Codec:       SettingCodecString,
			Default: SettingDefaultMetadata{
				Kind: SettingDefaultConstrained,
				Constrained: []ConstrainedDefaultMetadata{{
					Constraints: Constraints{Namespace: "original namespace"},
					Default:     SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: "original value"},
				}},
			},
		},
		{
			Key:         "metadata.mutable",
			Description: "original description",
			Precedence:  PrecedenceGlobal,
			ResultType:  reflect.TypeFor[map[string]any](),
			Codec:       SettingCodecMap,
			Default: SettingDefaultMetadata{
				Kind: SettingDefaultConcrete,
				Value: map[string]any{
					"slice": []string{"original"},
				},
			},
		},
	}, second)
}

func TestRegisteredSettingMetadataRejectsIncompleteRegistry(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
	}{
		{
			name:  "empty",
			setup: func() {},
		},
		{
			name: "missing metadata",
			setup: func() {
				setting := GlobalTypedSetting[int]{key: MakeKey("metadata.missing")}
				globalRegistry.settings = map[Key]GenericSetting{setting.Key(): setting}
			},
		},
		{
			name: "missing key",
			setup: func() {
				setting := GlobalTypedSetting[int]{
					key: MakeKey("metadata.missing-key"),
					metadata: &SettingMetadata{
						Precedence: PrecedenceGlobal,
						ResultType: reflect.TypeFor[int](),
						Codec:      SettingCodecInt,
						Default:    SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: 1},
					},
				}
				globalRegistry.settings = map[Key]GenericSetting{setting.Key(): setting}
			},
		},
		{
			name: "missing result type",
			setup: func() {
				setting := GlobalTypedSetting[int]{
					key: MakeKey("metadata.missing-type"),
					metadata: &SettingMetadata{
						Key:        "metadata.missing-type",
						Precedence: PrecedenceGlobal,
						Codec:      SettingCodecInt,
						Default:    SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: 1},
					},
				}
				globalRegistry.settings = map[Key]GenericSetting{setting.Key(): setting}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetMetadataTestRegistry(t)
			test.setup()

			metadata, err := RegisteredSettingMetadata()
			require.Error(t, err)
			require.Nil(t, metadata)
		})
	}
}

func TestRegisteredSettingMetadataFreezesRegistry(t *testing.T) {
	resetMetadataTestRegistry(t)

	NewGlobalBoolSetting("metadata.first", true, "first")
	metadata, err := RegisteredSettingMetadata()
	require.NoError(t, err)
	require.Len(t, metadata, 1)

	require.PanicsWithValue(t,
		"dynamicconfig.New*Setting must only be called from static initializers",
		func() { NewGlobalBoolSetting("metadata.second", true, "second") },
	)
}

func TestRegisteredSettingMetadataRejectsNormalizedKeyCollision(t *testing.T) {
	resetMetadataTestRegistry(t)

	NewGlobalBoolSetting("Metadata.Collision", true, "first")
	require.PanicsWithValue(t,
		"duplicate registration of dynamic config key: \"metadata.collision\"",
		func() { NewGlobalBoolSetting("metadata.collision", false, "second") },
	)
}
