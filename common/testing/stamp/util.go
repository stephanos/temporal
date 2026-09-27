package stamp

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
	playgroundvalidator "github.com/go-playground/validator/v10"
)

var (
	boldStr          = color.New(color.Bold).SprintFunc()
	redStr           = color.New(color.FgRed).SprintFunc()
	underlineStr     = color.New(color.Underline).SprintFunc()
	simpleSpew       = spew.NewDefaultConfig()
	validator        = playgroundvalidator.New()
	genericTypeRegex = regexp.MustCompile(`^[^[]+\[([^\[\]]+)\]$`)
)

func init() {
	color.NoColor = false

	simpleSpew.DisablePointerAddresses = true
	simpleSpew.DisableCapacities = true
	simpleSpew.MaxDepth = 2
}

func qualifiedTypeName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

func mustGetTypeParam(t reflect.Type) string {
	typeStr := t.String()
	if matches := genericTypeRegex.FindStringSubmatch(typeStr); matches != nil {
		return matches[1]
	}
	panic("not a generic type: " + typeStr)
}

func copyToValWithType(src reflect.Value, dstType reflect.Type) reflect.Value {
	dst := reflect.New(dstType)
	if dstType.Kind() == reflect.Pointer {
		dst = reflect.New(dstType.Elem())
	}
	dstElem := dst.Elem()

	srcElem := src
	if src.Kind() == reflect.Pointer {
		srcElem = srcElem.Elem()
	}

	for i := 0; i < srcElem.NumField(); i++ {
		dstField := dstElem.Field(i)
		if !dstField.CanSet() {
			continue
		}
		srcField := srcElem.Field(i)
		switch {
		case srcField.Type().AssignableTo(dstField.Type()):
			dstField.Set(srcField)
		case srcField.Type().ConvertibleTo(dstField.Type()):
			dstField.Set(srcField.Convert(dstField.Type()))
		case srcField.Kind() == reflect.Interface:
			if !srcField.IsNil() {
				dstField.Set(srcField.Elem())
			}
		default:
			panic(fmt.Sprintf("cannot copy field %q from %q to %q",
				dstElem.Type().Field(i).Name, srcElem.Type(), dstField.Type()))
		}
	}

	if dstType.Kind() == reflect.Pointer {
		return dst
	}
	return dst.Elem()
}
