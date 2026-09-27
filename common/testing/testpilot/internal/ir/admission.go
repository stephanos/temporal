package ir

import (
	"reflect"
	"slices"

	"google.golang.org/protobuf/proto"
)

// Invalid is the admission rejection every Testpilot core package returns, its path truncated to
// the bound every located path keeps.
func Invalid(category ErrorCategory, path, detail string) error {
	if len(path) > 256 {
		path = path[:256]
	}
	return &Error{Category: category, Path: path, Detail: detail}
}

// ValidID admits an identity: 1 to 256 ASCII letters, digits, '_', '-' or '.'.
func ValidID(id string) bool {
	if len(id) == 0 || len(id) > 256 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-', c == '.':
		default:
			return false
		}
	}
	return true
}

// IsNil reports a nil interface or an interface holding a nil value of a nilable kind.
func IsNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}

// CheckCeilings bounds limits' surface, then admits each int64 field in declaration order: it must
// be positive and, unless ceiling is nil, within ceiling's same field. Fields named in skip are not
// checked. The first field out of bounds is rejected with reject(field name), so each caller keeps
// its own path and detail. limits must be non-nil; callers reject a missing one themselves.
func CheckCeilings[M proto.Message](limits, ceiling M, reject func(field string) error, skip ...string) error {
	if err := CheckSurface(limits, DefaultLimits()); err != nil {
		return err
	}
	message := limits.ProtoReflect()
	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		if slices.Contains(skip, name) {
			continue
		}
		value := message.Get(field).Int()
		if value <= 0 || !IsNil(ceiling) && value > ceiling.ProtoReflect().Get(field).Int() {
			return reject(name)
		}
	}
	return nil
}
