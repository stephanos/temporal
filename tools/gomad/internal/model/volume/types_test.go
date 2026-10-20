package volume

import (
	"reflect"
	"testing"

	wire "go.temporal.io/server/tools/gomad/internal/model/iowire"
)

// TestNamedMirrorsMatchAuthoritativeWireTypes keeps the owned mount mirrors in
// types.go structurally equal to the generated I/O wire codec, which replaced
// the deleted projection generator's mirrorTypes derivation.
func TestNamedMirrorsMatchAuthoritativeWireTypes(t *testing.T) {
	mirrors := map[reflect.Type]reflect.Type{
		reflect.TypeFor[Kind]():        reflect.TypeFor[wire.MountKind](),
		reflect.TypeFor[MountStatus](): reflect.TypeFor[wire.MountStatus](),
		reflect.TypeFor[Child]():       reflect.TypeFor[wire.MountChild](),
		reflect.TypeFor[LoadEntry]():   reflect.TypeFor[wire.MountEntry](),
	}
	for mirror, authoritative := range mirrors {
		if !sameShape(mirror, authoritative, mirrors) {
			t.Errorf("%s does not mirror authoritative %s", mirror, authoritative)
		}
	}
	type driftedChild struct {
		Name string
		Kind MountStatus
		Mode uint32
	}
	if sameShape(reflect.TypeFor[driftedChild](), reflect.TypeFor[wire.MountChild](), mirrors) {
		t.Fatal("a mirror with a drifted field type was accepted")
	}
}

func sameShape(mirror, authoritative reflect.Type, mirrors map[reflect.Type]reflect.Type) bool {
	if expected, named := mirrors[mirror]; named && mirror.Kind() != reflect.Struct {
		return expected == authoritative && mirror.Kind() == authoritative.Kind()
	}
	if mirror.Kind() != authoritative.Kind() {
		return false
	}
	switch mirror.Kind() {
	case reflect.Struct:
		if mirror.NumField() != authoritative.NumField() {
			return false
		}
		for index := range mirror.NumField() {
			left, right := mirror.Field(index), authoritative.Field(index)
			if left.Name != right.Name || !sameShape(left.Type, right.Type, mirrors) {
				return false
			}
		}
		return true
	case reflect.Slice:
		element := mirror.Elem()
		if expected, named := mirrors[element]; named {
			return expected == authoritative.Elem()
		}
		return sameShape(element, authoritative.Elem(), mirrors)
	default:
		return mirror == authoritative
	}
}
