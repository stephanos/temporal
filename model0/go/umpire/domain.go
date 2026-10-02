// Package umpire is a Go implementation of the Umpire model layer, built to be compared with the
// Lean one under model/lean/. A Model is ordinary Go: domains are Go types, actions and machines are
// package-level values, step functions are plain funcs, and the finite table, the refinement and
// the Queries are computed and checked when a test asks for them.
//
// The orders and key spellings here follow the Lean implementation exactly, because Definition IDs,
// witnesses and exploration targets are compared against Lean's output byte for byte. Where a rule
// is non-obvious, the comment names the Lean source it mirrors.
package umpire

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Finite is implemented by a value type that lists its members in catalog order: a string enum
// lists its constants, a bounded counter its values. The framework calls Values on the zero value.
type Finite[T any] interface {
	Values() []T
}

// Keyed overrides the key a value is spelled with in state, action, row and Definition keys.
type Keyed interface {
	Key() string
}

var (
	sumsMu    sync.RWMutex
	sums      = map[reflect.Type][]reflect.Value{}
	sumErrors = map[reflect.Type]error{}
)

// Sum registers the variants of a sealed interface, in catalog order. A variant is a struct; one
// with fields contributes one member per assignment of them, the way a Lean constructor with
// finite fields contributes one class per assignment. Call it from a package-level var so it runs
// at init:
//
//	var _ = umpire.Sum[Reply](SyncSuccess{}, Async{}, HandlerError{})
func Sum[I any](variants ...I) struct{} {
	t := reflect.TypeFor[I]()
	values := make([]reflect.Value, 0, len(variants))
	for _, v := range variants {
		values = append(values, reflect.ValueOf(v))
	}
	sumsMu.Lock()
	defer sumsMu.Unlock()
	if t.Kind() != reflect.Interface {
		// Reported when a domain of this type is enumerated.
		sumErrors[t] = fmt.Errorf("umpire.Sum: %s is not an interface", t)
		return struct{}{}
	}
	sums[t] = values
	return struct{}{}
}

// DomainOf returns every member of T in catalog order.
func DomainOf[T any]() ([]T, error) {
	values, err := domainOf(reflect.TypeFor[T]())
	if err != nil {
		return nil, err
	}
	out := make([]T, len(values))
	for i, v := range values {
		m, ok := v.Interface().(T)
		if !ok {
			return nil, fmt.Errorf("member %v is not a %s", v.Interface(), reflect.TypeFor[T]())
		}
		out[i] = m
	}
	return out, nil
}

// KeyOf spells a value the way Lean keys it: a string enum by its value, a Bool as true or false,
// a counter in decimal, a variant by its lowerCamel type name followed by its fields, and a state
// structure by its fields in declaration order, joined by "-".
func KeyOf[T any](v T) string {
	return keyOf(reflect.ValueOf(&v).Elem(), reflect.TypeFor[T]())
}

var finiteMethod = "Values"

func domainOf(t reflect.Type) ([]reflect.Value, error) {
	sumsMu.RLock()
	sumErr := sumErrors[t]
	sumsMu.RUnlock()
	if sumErr != nil {
		return nil, sumErr
	}
	if m, ok := t.MethodByName(finiteMethod); ok && t.Kind() != reflect.Interface {
		out := m.Func.Call([]reflect.Value{reflect.Zero(t)})
		if len(out) != 1 || out[0].Kind() != reflect.Slice || out[0].Type().Elem() != t {
			return nil, fmt.Errorf("%s.Values must return []%s", t, t)
		}
		values := make([]reflect.Value, out[0].Len())
		for i := range values {
			values[i] = out[0].Index(i)
		}
		if len(values) == 0 {
			return nil, fmt.Errorf("%s.Values is empty", t)
		}
		return values, nil
	}
	switch t.Kind() {
	case reflect.Bool:
		// Lean's `Bool` instance lists false before true.
		return []reflect.Value{reflect.ValueOf(false).Convert(t), reflect.ValueOf(true).Convert(t)}, nil
	case reflect.Interface:
		sumsMu.RLock()
		variants, ok := sums[t]
		sumsMu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("%s is an interface with no umpire.Sum registration", t)
		}
		var out []reflect.Value
		for _, variant := range variants {
			members, err := structProduct(variant.Type())
			if err != nil {
				return nil, fmt.Errorf("variant %s of %s: %w", variant.Type(), t, err)
			}
			for _, m := range members {
				boxed := reflect.New(t).Elem()
				boxed.Set(m)
				out = append(out, boxed)
			}
		}
		return out, nil
	case reflect.Struct:
		return structProduct(t)
	default:
		return nil, fmt.Errorf("%s is not finite: give it a Values() []%s method", t, t.Name())
	}
}

// structProduct enumerates a structure as the product of its fields, with the last field varying
// fastest, which is the order the Lean `Finite` derivation produces.
func structProduct(t reflect.Type) ([]reflect.Value, error) {
	out := []reflect.Value{reflect.New(t).Elem()}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		members, err := domainOf(f.Type)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}
		next := make([]reflect.Value, 0, len(out)*len(members))
		for _, prefix := range out {
			for _, m := range members {
				v := reflect.New(t).Elem()
				v.Set(prefix)
				v.Field(i).Set(m)
				next = append(next, v)
			}
		}
		out = next
	}
	return out, nil
}

func keyOf(v reflect.Value, static reflect.Type) string {
	if static.Kind() == reflect.Interface {
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		return variantKey(v)
	}
	if k, ok := v.Interface().(Keyed); ok {
		return k.Key()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Struct:
		return strings.Join(fieldKeys(v), "-")
	default:
		return fmt.Sprint(v.Interface())
	}
}

func variantKey(v reflect.Value) string {
	if k, ok := v.Interface().(Keyed); ok {
		return k.Key()
	}
	parts := append([]string{lowerFirst(v.Type().Name())}, fieldKeys(v)...)
	return strings.Join(parts, "-")
}

func fieldKeys(v reflect.Value) []string {
	var parts []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		parts = append(parts, keyOf(v.Field(i), f.Type))
	}
	return parts
}

// fieldName is the name a state field has in Definition IDs: its `umpire` tag, or its Go name with
// the first letter lowered.
func fieldName(f reflect.StructField) string {
	if tag := f.Tag.Get("umpire"); tag != "" {
		return tag
	}
	return lowerFirst(f.Name)
}

func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}
