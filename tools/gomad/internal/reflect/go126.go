package reflect

import (
	"iter"
	stdreflect "reflect" //gomad:notranslate
)

func TypeAssert[T any](v Value) (T, bool) {
	return stdreflect.TypeAssert[T](v.inner)
}

func (t *typeImpl) Fields() iter.Seq[StructField] {
	if t.Kind() != Struct {
		panic("reflect: Fields of non-struct type " + t.String())
	}
	return func(yield func(StructField) bool) {
		for i := range t.NumField() {
			if !yield(t.Field(i)) {
				return
			}
		}
	}
}

func (t *typeImpl) Methods() iter.Seq[Method] {
	return func(yield func(Method) bool) {
		for i := range t.NumMethod() {
			if !yield(t.Method(i)) {
				return
			}
		}
	}
}

func (t *typeImpl) Ins() iter.Seq[Type] {
	if t.Kind() != Func {
		panic("reflect: Ins of non-func type " + t.String())
	}
	return func(yield func(Type) bool) {
		for i := range t.NumIn() {
			if !yield(t.In(i)) {
				return
			}
		}
	}
}

func (t *typeImpl) Outs() iter.Seq[Type] {
	if t.Kind() != Func {
		panic("reflect: Outs of non-func type " + t.String())
	}
	return func(yield func(Type) bool) {
		for i := range t.NumOut() {
			if !yield(t.Out(i)) {
				return
			}
		}
	}
}

func (v Value) Fields() iter.Seq2[StructField, Value] {
	t := v.Type()
	if t.Kind() != Struct {
		panic("reflect: Fields of non-struct type " + t.String())
	}
	return func(yield func(StructField, Value) bool) {
		for i := range v.NumField() {
			if !yield(t.Field(i), v.Field(i)) {
				return
			}
		}
	}
}

func (v Value) Methods() iter.Seq2[Method, Value] {
	return func(yield func(Method, Value) bool) {
		t := v.Type()
		for i := range v.NumMethod() {
			if !yield(t.Method(i), v.Method(i)) {
				return
			}
		}
	}
}

func (v Value) Seq() iter.Seq[Value] {
	if v.kind == wrappedMap {
		return func(yield func(Value) bool) {
			iter := v.MapRange()
			for iter.Next() {
				if !yield(iter.Key()) {
					return
				}
			}
		}
	}
	return func(yield func(Value) bool) {
		for value := range v.inner.Seq() {
			if !yield(wrapValue(value)) {
				return
			}
		}
	}
}

func (v Value) Seq2() iter.Seq2[Value, Value] {
	if v.kind == wrappedMap {
		return func(yield func(Value, Value) bool) {
			iter := v.MapRange()
			for iter.Next() {
				if !yield(iter.Key(), iter.Value()) {
					return
				}
			}
		}
	}
	return func(yield func(Value, Value) bool) {
		for key, value := range v.inner.Seq2() {
			if !yield(wrapValue(key), wrapValue(value)) {
				return
			}
		}
	}
}
