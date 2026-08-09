package reflect

import (
	"iter"
	stdreflect "reflect" //gomad:notranslate
)

func TypeAssert[T any](v Value) (T, bool) {
	return stdreflect.TypeAssert[T](v.inner)
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
