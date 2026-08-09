package reflect

import "reflect" //gomad:notranslate

func VisibleFields(t Type) []StructField {
	return wrapStructFields(reflect.VisibleFields(t.(*typeImpl).inner))
}
