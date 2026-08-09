package reflect

import "reflect" //gomad:notranslate

func Swapper(slice any) func(i, j int) {
	return reflect.Swapper(slice)
}
