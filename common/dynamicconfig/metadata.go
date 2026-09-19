package dynamicconfig

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
)

type (
	// SettingCodec identifies the conversion contract used by a setting constructor.
	SettingCodec string

	// SettingDefaultKind identifies the shape of a registered setting default.
	SettingDefaultKind string

	// OpaqueDefaultMetadata preserves the type and reason for a default that cannot be copied safely.
	OpaqueDefaultMetadata struct {
		ResultType reflect.Type
		Reason     string
	}

	// ConstrainedDefaultMetadata preserves one registration-time constrained default.
	ConstrainedDefaultMetadata struct {
		Constraints Constraints
		Default     SettingDefaultMetadata
	}

	// SettingDefaultMetadata preserves a concrete, constrained, or opaque registration-time default.
	SettingDefaultMetadata struct {
		Kind        SettingDefaultKind
		Value       any
		Constrained []ConstrainedDefaultMetadata
		Opaque      OpaqueDefaultMetadata
	}

	// SettingMetadata describes one registered dynamic config setting without exposing its converter.
	SettingMetadata struct {
		Key         string
		Description string
		Precedence  Precedence
		ResultType  reflect.Type
		Codec       SettingCodec
		Default     SettingDefaultMetadata
	}
)

const (
	// SettingCodecBool identifies the built-in bool converter.
	SettingCodecBool SettingCodec = "bool"
	// SettingCodecInt identifies the built-in int converter.
	SettingCodecInt SettingCodec = "int"
	// SettingCodecFloat identifies the built-in float64 converter.
	SettingCodecFloat SettingCodec = "float"
	// SettingCodecString identifies the built-in string converter.
	SettingCodecString SettingCodec = "string"
	// SettingCodecDuration identifies the built-in time.Duration converter.
	SettingCodecDuration SettingCodec = "duration"
	// SettingCodecMap identifies the built-in map converter.
	SettingCodecMap SettingCodec = "map"
	// SettingCodecStructure identifies the mapstructure-based converter.
	SettingCodecStructure SettingCodec = "structure"
	// SettingCodecCustom identifies a caller-provided converter.
	SettingCodecCustom SettingCodec = "custom"
)

const (
	// SettingDefaultConcrete identifies a copied concrete default.
	SettingDefaultConcrete SettingDefaultKind = "concrete"
	// SettingDefaultConstrained identifies an ordered set of constrained defaults.
	SettingDefaultConstrained SettingDefaultKind = "constrained"
	// SettingDefaultOpaque identifies a default whose mutable value cannot be copied safely.
	SettingDefaultOpaque SettingDefaultKind = "opaque"
)

// RegisteredSettingMetadata returns a deterministic, deeply copied snapshot of the registry.
// Calling it freezes the registry in the same way as querying a setting.
func RegisteredSettingMetadata() ([]SettingMetadata, error) {
	globalRegistry.queried.Store(true)
	if len(globalRegistry.settings) == 0 {
		return nil, errorsNewMetadata("registry is empty")
	}

	result := make([]SettingMetadata, 0, len(globalRegistry.settings))
	seen := make(map[string]struct{}, len(globalRegistry.settings))
	for registryKey, setting := range globalRegistry.settings {
		if setting == nil {
			return nil, errorsNewMetadata("setting %q is nil", registryKey.String())
		}
		metadata := setting.registrationMetadata()
		if metadata == nil {
			return nil, errorsNewMetadata("setting %q is missing metadata", registryKey.String())
		}

		copy, err := cloneSettingMetadata(*metadata)
		if err != nil {
			return nil, errorsNewMetadata("setting %q: %v", registryKey.String(), err)
		}
		copy.Key = MakeKey(copy.Key).String()
		if err := validateSettingMetadata(registryKey, copy); err != nil {
			return nil, err
		}
		if _, exists := seen[copy.Key]; exists {
			return nil, errorsNewMetadata("duplicate normalized key %q", copy.Key)
		}
		seen[copy.Key] = struct{}{}
		result = append(result, copy)
	}

	slices.SortFunc(result, func(a, b SettingMetadata) int {
		return strings.Compare(a.Key, b.Key)
	})
	return result, nil
}

func newSettingMetadata[T any](
	key Key,
	description string,
	precedence Precedence,
	codec SettingCodec,
	def T,
) *SettingMetadata {
	resultType := reflect.TypeFor[T]()
	return &SettingMetadata{
		Key:         key.String(),
		Description: description,
		Precedence:  precedence,
		ResultType:  resultType,
		Codec:       codec,
		Default:     captureSettingDefault(resultType, def),
	}
}

func newConstrainedSettingMetadata[T any](
	key Key,
	description string,
	precedence Precedence,
	codec SettingCodec,
	cdef []TypedConstrainedValue[T],
) *SettingMetadata {
	resultType := reflect.TypeFor[T]()
	defaults := make([]ConstrainedDefaultMetadata, len(cdef))
	for i, value := range cdef {
		defaults[i] = ConstrainedDefaultMetadata{
			Constraints: value.Constraints,
			Default:     captureSettingDefault(resultType, value.Value),
		}
	}
	return &SettingMetadata{
		Key:         key.String(),
		Description: description,
		Precedence:  precedence,
		ResultType:  resultType,
		Codec:       codec,
		Default: SettingDefaultMetadata{
			Kind:        SettingDefaultConstrained,
			Constrained: defaults,
		},
	}
}

func captureSettingDefault(resultType reflect.Type, value any) SettingDefaultMetadata {
	copy, err := cloneMetadataValue(reflect.ValueOf(value), "")
	if err != nil {
		return SettingDefaultMetadata{
			Kind: SettingDefaultOpaque,
			Opaque: OpaqueDefaultMetadata{
				ResultType: resultType,
				Reason:     "default " + err.Error(),
			},
		}
	}
	if !copy.IsValid() {
		return SettingDefaultMetadata{Kind: SettingDefaultConcrete}
	}
	return SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: copy.Interface()}
}

func cloneSettingMetadata(metadata SettingMetadata) (SettingMetadata, error) {
	copy := metadata
	defaultCopy, err := cloneSettingDefaultMetadata(metadata.Default)
	if err != nil {
		return SettingMetadata{}, err
	}
	copy.Default = defaultCopy
	return copy, nil
}

func cloneSettingDefaultMetadata(metadata SettingDefaultMetadata) (SettingDefaultMetadata, error) {
	copy := metadata
	switch metadata.Kind {
	case SettingDefaultConcrete:
		value, err := cloneMetadataValue(reflect.ValueOf(metadata.Value), "")
		if err != nil {
			return SettingDefaultMetadata{}, err
		}
		if value.IsValid() {
			copy.Value = value.Interface()
		}
	case SettingDefaultConstrained:
		copy.Constrained = make([]ConstrainedDefaultMetadata, len(metadata.Constrained))
		for i, constrained := range metadata.Constrained {
			defaultCopy, err := cloneSettingDefaultMetadata(constrained.Default)
			if err != nil {
				return SettingDefaultMetadata{}, err
			}
			copy.Constrained[i] = ConstrainedDefaultMetadata{
				Constraints: constrained.Constraints,
				Default:     defaultCopy,
			}
		}
	case SettingDefaultOpaque:
	default:
		return SettingDefaultMetadata{}, fmt.Errorf("unknown default kind %q", metadata.Kind)
	}
	return copy, nil
}

func cloneMetadataValue(value reflect.Value, path string) (reflect.Value, error) {
	return cloneMetadataValueAt(value, path, make(map[metadataCloneVisit]struct{}))
}

type metadataCloneVisit struct {
	typeID  reflect.Type
	pointer uintptr
}

func cloneMetadataValueAt(
	value reflect.Value,
	path string,
	active map[metadataCloneVisit]struct{},
) (reflect.Value, error) {
	if !value.IsValid() {
		return reflect.Value{}, nil
	}
	if value.Kind() == reflect.Map || value.Kind() == reflect.Pointer || value.Kind() == reflect.Slice {
		if !value.IsNil() {
			visit := metadataCloneVisit{typeID: value.Type(), pointer: uintptr(value.UnsafePointer())}
			if _, exists := active[visit]; exists {
				return reflect.Value{}, fmt.Errorf("contains unsupported cycle at %s", metadataPath(path))
			}
			active[visit] = struct{}{}
			defer delete(active, visit)
		}
	}

	switch value.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return value, nil
	case reflect.Array:
		copy := reflect.New(value.Type()).Elem()
		for i := range value.Len() {
			item, err := cloneMetadataValueAt(value.Index(i), fmt.Sprintf("%s[%d]", path, i), active)
			if err != nil {
				return reflect.Value{}, err
			}
			copy.Index(i).Set(item)
		}
		return copy, nil
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		item, err := cloneMetadataValueAt(value.Elem(), path, active)
		if err != nil {
			return reflect.Value{}, err
		}
		copy := reflect.New(value.Type()).Elem()
		copy.Set(item)
		return copy, nil
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		copy := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			key, err := cloneMetadataValueAt(iterator.Key(), path+"{key}", active)
			if err != nil {
				return reflect.Value{}, err
			}
			item, err := cloneMetadataValueAt(iterator.Value(), path+"[value]", active)
			if err != nil {
				return reflect.Value{}, err
			}
			copy.SetMapIndex(key, item)
		}
		return copy, nil
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		item, err := cloneMetadataValueAt(value.Elem(), path+"*", active)
		if err != nil {
			return reflect.Value{}, err
		}
		copy := reflect.New(value.Type().Elem())
		copy.Elem().Set(item)
		return copy, nil
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := range value.Len() {
			item, err := cloneMetadataValueAt(value.Index(i), fmt.Sprintf("%s[%d]", path, i), active)
			if err != nil {
				return reflect.Value{}, err
			}
			copy.Index(i).Set(item)
		}
		return copy, nil
	case reflect.Struct:
		copy := reflect.New(value.Type()).Elem()
		copy.Set(value)
		for i := range value.NumField() {
			fieldInfo := value.Type().Field(i)
			fieldPath := path + "." + fieldInfo.Name
			if fieldInfo.PkgPath != "" {
				if metadataValueContainsMutableReference(value.Field(i)) {
					return reflect.Value{}, fmt.Errorf("contains unsupported unexported mutable value at %s", fieldPath)
				}
				continue
			}
			field, err := cloneMetadataValueAt(value.Field(i), fieldPath, active)
			if err != nil {
				return reflect.Value{}, err
			}
			copy.Field(i).Set(field)
		}
		return copy, nil
	case reflect.Chan, reflect.Func:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		return reflect.Value{}, fmt.Errorf("contains unsupported %s value at %s", value.Kind(), metadataPath(path))
	case reflect.UnsafePointer:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		return reflect.Value{}, fmt.Errorf("contains unsupported unsafe pointer at %s", metadataPath(path))
	default:
		return reflect.Value{}, fmt.Errorf("contains unsupported %s value at %s", value.Kind(), metadataPath(path))
	}
}

func metadataValueContainsMutableReference(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return !value.IsNil()
	case reflect.Array:
		for i := range value.Len() {
			if metadataValueContainsMutableReference(value.Index(i)) {
				return true
			}
		}
	case reflect.Struct:
		for i := range value.NumField() {
			if metadataValueContainsMutableReference(value.Field(i)) {
				return true
			}
		}
	}
	return false
}

func metadataPath(path string) string {
	if path == "" {
		return "<root>"
	}
	return path
}

func validateSettingMetadata(registryKey Key, metadata SettingMetadata) error {
	if metadata.Key == "" {
		return errorsNewMetadata("setting %q has an empty key", registryKey.String())
	}
	if metadata.Key != MakeKey(registryKey.String()).String() {
		return errorsNewMetadata("setting %q metadata has mismatched key %q", registryKey.String(), metadata.Key)
	}
	if metadata.ResultType == nil {
		return errorsNewMetadata("setting %q is missing result type metadata", metadata.Key)
	}
	if metadata.Precedence < PrecedenceGlobal || metadata.Precedence > PrecedenceChasmTaskType {
		return errorsNewMetadata("setting %q has unknown precedence %d", metadata.Key, metadata.Precedence)
	}
	switch metadata.Codec {
	case SettingCodecBool:
		if metadata.ResultType != reflect.TypeFor[bool]() {
			return errorsNewMetadata("setting %q bool codec has result type %s", metadata.Key, metadata.ResultType)
		}
	case SettingCodecInt:
		if metadata.ResultType != reflect.TypeFor[int]() {
			return errorsNewMetadata("setting %q int codec has result type %s", metadata.Key, metadata.ResultType)
		}
	case SettingCodecFloat:
		if metadata.ResultType != reflect.TypeFor[float64]() {
			return errorsNewMetadata("setting %q float codec has result type %s", metadata.Key, metadata.ResultType)
		}
	case SettingCodecString:
		if metadata.ResultType != reflect.TypeFor[string]() {
			return errorsNewMetadata("setting %q string codec has result type %s", metadata.Key, metadata.ResultType)
		}
	case SettingCodecDuration:
		if metadata.ResultType != reflect.TypeFor[time.Duration]() {
			return errorsNewMetadata("setting %q duration codec has result type %s", metadata.Key, metadata.ResultType)
		}
	case SettingCodecMap:
		if metadata.ResultType != reflect.TypeFor[map[string]any]() {
			return errorsNewMetadata("setting %q map codec has result type %s", metadata.Key, metadata.ResultType)
		}
	case SettingCodecStructure, SettingCodecCustom:
	default:
		return errorsNewMetadata("setting %q has unknown codec %q", metadata.Key, metadata.Codec)
	}
	if err := validateSettingDefaultMetadata(metadata.Default, metadata.ResultType, true); err != nil {
		return errorsNewMetadata("setting %q: %v", metadata.Key, err)
	}
	return nil
}

func validateSettingDefaultMetadata(
	metadata SettingDefaultMetadata,
	resultType reflect.Type,
	allowConstrained bool,
) error {
	switch metadata.Kind {
	case SettingDefaultConcrete:
		if len(metadata.Constrained) != 0 || metadata.Opaque.ResultType != nil || metadata.Opaque.Reason != "" {
			return fmt.Errorf("concrete default has conflicting metadata")
		}
		if metadata.Value == nil {
			switch resultType.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			default:
				return fmt.Errorf("nil concrete default is not assignable to %s", resultType)
			}
		} else if !reflect.TypeOf(metadata.Value).AssignableTo(resultType) {
			return fmt.Errorf("concrete default type %s is not assignable to %s", reflect.TypeOf(metadata.Value), resultType)
		}
	case SettingDefaultConstrained:
		if !allowConstrained {
			return fmt.Errorf("nested constrained default")
		}
		if metadata.Value != nil || len(metadata.Constrained) == 0 ||
			metadata.Opaque.ResultType != nil || metadata.Opaque.Reason != "" {
			return fmt.Errorf("constrained default has conflicting or empty metadata")
		}
		for _, constrained := range metadata.Constrained {
			if err := validateSettingDefaultMetadata(constrained.Default, resultType, false); err != nil {
				return err
			}
		}
	case SettingDefaultOpaque:
		if metadata.Value != nil || len(metadata.Constrained) != 0 ||
			metadata.Opaque.ResultType == nil || metadata.Opaque.Reason == "" {
			return fmt.Errorf("opaque default has incomplete or conflicting metadata")
		}
		if metadata.Opaque.ResultType != resultType {
			return fmt.Errorf("opaque default result type %s does not match %s", metadata.Opaque.ResultType, resultType)
		}
	default:
		return fmt.Errorf("unknown default kind %q", metadata.Kind)
	}
	return nil
}

func errorsNewMetadata(format string, args ...any) error {
	return fmt.Errorf("dynamic config metadata: "+format, args...)
}
