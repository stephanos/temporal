package artifact

import (
	"fmt"
	"reflect"

	"go.temporal.io/server/tools/gomad3/record"
)

// cloneManifest returns a manifest that shares no memory with manifest. An
// opened artifact hands out only such copies, so a caller that changes a
// returned manifest cannot change what the handle validates payloads against.
// The copy walks the type, so a field added to the record is copied without a
// change here; nil pointers, slices and maps stay nil.
func cloneManifest(manifest record.ExecutionRecord) record.ExecutionRecord {
	var clone record.ExecutionRecord
	deepCopy(reflect.ValueOf(&clone).Elem(), reflect.ValueOf(manifest))
	return clone
}

func deepCopy(destination, source reflect.Value) {
	switch source.Kind() {
	case reflect.Pointer:
		if source.IsNil() {
			return
		}
		destination.Set(reflect.New(source.Type().Elem()))
		deepCopy(destination.Elem(), source.Elem())
	case reflect.Slice:
		if source.IsNil() {
			return
		}
		destination.Set(reflect.MakeSlice(source.Type(), source.Len(), source.Len()))
		for index := range source.Len() {
			deepCopy(destination.Index(index), source.Index(index))
		}
	case reflect.Array:
		for index := range source.Len() {
			deepCopy(destination.Index(index), source.Index(index))
		}
	case reflect.Map:
		if source.IsNil() {
			return
		}
		destination.Set(reflect.MakeMapWithSize(source.Type(), source.Len()))
		entries := source.MapRange()
		for entries.Next() {
			key := reflect.New(source.Type().Key()).Elem()
			deepCopy(key, entries.Key())
			value := reflect.New(source.Type().Elem()).Elem()
			deepCopy(value, entries.Value())
			destination.SetMapIndex(key, value)
		}
	case reflect.Struct:
		for index := range source.NumField() {
			deepCopy(destination.Field(index), source.Field(index))
		}
	case reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		panic(fmt.Sprintf("artifact manifest cannot copy a %s field", source.Kind()))
	default:
		destination.Set(source)
	}
}
