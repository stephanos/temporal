package reflect

import "reflect" //gomad:notranslate

func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	wrapped := func(args []reflect.Value) (results []reflect.Value) {
		return unwrapValues(fn(wrapValues(args)))
	}
	return Value{inner: reflect.MakeFunc(typ.(*typeImpl).inner, wrapped)}
}
