package dynamicconfig

import (
	"errors"
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

		cloned, err := cloneSettingMetadata(*metadata)
		if err != nil {
			return nil, errorsNewMetadata("setting %q: %v", registryKey.String(), err)
		}
		cloned.Key = MakeKey(cloned.Key).String()
		if err := validateSettingMetadata(registryKey, cloned); err != nil {
			return nil, err
		}
		if _, exists := seen[cloned.Key]; exists {
			return nil, errorsNewMetadata("duplicate normalized key %q", cloned.Key)
		}
		seen[cloned.Key] = struct{}{}
		result = append(result, cloned)
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
	cloned, err := cloneMetadataValue(reflect.ValueOf(value), "")
	if err != nil {
		return SettingDefaultMetadata{
			Kind: SettingDefaultOpaque,
			Opaque: OpaqueDefaultMetadata{
				ResultType: resultType,
				Reason:     "default " + err.Error(),
			},
		}
	}
	if !cloned.IsValid() {
		return SettingDefaultMetadata{Kind: SettingDefaultConcrete}
	}
	return SettingDefaultMetadata{Kind: SettingDefaultConcrete, Value: cloned.Interface()}
}

func cloneSettingMetadata(metadata SettingMetadata) (SettingMetadata, error) {
	cloned := metadata
	defaultCopy, err := cloneSettingDefaultMetadata(metadata.Default)
	if err != nil {
		return SettingMetadata{}, err
	}
	cloned.Default = defaultCopy
	return cloned, nil
}

func cloneSettingDefaultMetadata(metadata SettingDefaultMetadata) (SettingDefaultMetadata, error) {
	cloned := metadata
	switch metadata.Kind {
	case SettingDefaultConcrete:
		value, err := cloneMetadataValue(reflect.ValueOf(metadata.Value), "")
		if err != nil {
			return SettingDefaultMetadata{}, err
		}
		if value.IsValid() {
			cloned.Value = value.Interface()
		}
	case SettingDefaultConstrained:
		cloned.Constrained = make([]ConstrainedDefaultMetadata, len(metadata.Constrained))
		for i, constrained := range metadata.Constrained {
			defaultCopy, err := cloneSettingDefaultMetadata(constrained.Default)
			if err != nil {
				return SettingDefaultMetadata{}, err
			}
			cloned.Constrained[i] = ConstrainedDefaultMetadata{
				Constraints: constrained.Constraints,
				Default:     defaultCopy,
			}
		}
	case SettingDefaultOpaque:
	default:
		return SettingDefaultMetadata{}, fmt.Errorf("unknown default kind %q", metadata.Kind)
	}
	return cloned, nil
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
	leave, err := enterMetadataCloneVisit(value, path, active)
	if err != nil {
		return reflect.Value{}, err
	}
	defer leave()

	switch value.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return value, nil
	case reflect.Array:
		return cloneMetadataElements(reflect.New(value.Type()).Elem(), value, path, active)
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		item, err := cloneMetadataValueAt(value.Elem(), path, active)
		if err != nil {
			return reflect.Value{}, err
		}
		cloned := reflect.New(value.Type()).Elem()
		cloned.Set(item)
		return cloned, nil
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		return cloneMetadataMap(value, path, active)
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		item, err := cloneMetadataValueAt(value.Elem(), path+"*", active)
		if err != nil {
			return reflect.Value{}, err
		}
		cloned := reflect.New(value.Type().Elem())
		cloned.Elem().Set(item)
		return cloned, nil
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		return cloneMetadataElements(reflect.MakeSlice(value.Type(), value.Len(), value.Len()), value, path, active)
	case reflect.Struct:
		return cloneMetadataStruct(value, path, active)
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

// enterMetadataCloneVisit records a non-nil reference value as being cloned and returns the func that
// removes it again, so a reference reached from itself is rejected as a cycle.
func enterMetadataCloneVisit(
	value reflect.Value,
	path string,
	active map[metadataCloneVisit]struct{},
) (func(), error) {
	switch value.Kind() {
	case reflect.Map, reflect.Pointer, reflect.Slice:
	default:
		return func() {}, nil
	}
	if value.IsNil() {
		return func() {}, nil
	}
	visit := metadataCloneVisit{typeID: value.Type(), pointer: uintptr(value.UnsafePointer())}
	if _, exists := active[visit]; exists {
		return nil, fmt.Errorf("contains unsupported cycle at %s", metadataPath(path))
	}
	active[visit] = struct{}{}
	return func() { delete(active, visit) }, nil
}

func cloneMetadataElements(
	cloned reflect.Value,
	value reflect.Value,
	path string,
	active map[metadataCloneVisit]struct{},
) (reflect.Value, error) {
	for i := range value.Len() {
		item, err := cloneMetadataValueAt(value.Index(i), fmt.Sprintf("%s[%d]", path, i), active)
		if err != nil {
			return reflect.Value{}, err
		}
		cloned.Index(i).Set(item)
	}
	return cloned, nil
}

func cloneMetadataMap(
	value reflect.Value,
	path string,
	active map[metadataCloneVisit]struct{},
) (reflect.Value, error) {
	cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
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
		cloned.SetMapIndex(key, item)
	}
	return cloned, nil
}

func cloneMetadataStruct(
	value reflect.Value,
	path string,
	active map[metadataCloneVisit]struct{},
) (reflect.Value, error) {
	cloned := reflect.New(value.Type()).Elem()
	cloned.Set(value)
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
		cloned.Field(i).Set(field)
	}
	return cloned, nil
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
	default:
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
			return errors.New("concrete default has conflicting metadata")
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
			return errors.New("nested constrained default")
		}
		if metadata.Value != nil || len(metadata.Constrained) == 0 ||
			metadata.Opaque.ResultType != nil || metadata.Opaque.Reason != "" {
			return errors.New("constrained default has conflicting or empty metadata")
		}
		for _, constrained := range metadata.Constrained {
			if err := validateSettingDefaultMetadata(constrained.Default, resultType, false); err != nil {
				return err
			}
		}
	case SettingDefaultOpaque:
		if metadata.Value != nil || len(metadata.Constrained) != 0 ||
			metadata.Opaque.ResultType == nil || metadata.Opaque.Reason == "" {
			return errors.New("opaque default has incomplete or conflicting metadata")
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
