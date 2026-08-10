package translate

import (
	"testing"

	simreflect "github.com/temporalio/gomad/internal/reflect"
)

type go126MethodSet struct{}

func (go126MethodSet) Alpha() {}

func (go126MethodSet) Beta() {}

func TestGo126TypeAssert(t *testing.T) {
	value := simreflect.ValueOf(42)
	got, ok := simreflect.TypeAssert[int](value)
	if !ok || got != 42 {
		t.Fatalf("TypeAssert[int](ValueOf(42)) = %v, %v; want 42, true", got, ok)
	}
	if _, ok := simreflect.TypeAssert[string](value); ok {
		t.Fatal("TypeAssert[string](ValueOf(42)) succeeded; want failure")
	}
}

func TestGo126Seq(t *testing.T) {
	value := simreflect.ValueOf([]string{"zero", "one"})
	if !value.Type().CanSeq() {
		t.Fatal("slice type cannot produce Seq")
	}

	var got []int64
	for index := range value.Seq() {
		got = append(got, index.Int())
	}
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("slice Seq indexes = %v; want [0 1]", got)
	}
}

func TestGo126Seq2(t *testing.T) {
	value := simreflect.ValueOf([]string{"zero", "one"})
	if !value.Type().CanSeq2() {
		t.Fatal("slice type cannot produce Seq2")
	}

	var indexes []int64
	var values []string
	for index, element := range value.Seq2() {
		indexes = append(indexes, index.Int())
		values = append(values, element.String())
	}
	if len(indexes) != 2 || indexes[0] != 0 || indexes[1] != 1 {
		t.Fatalf("slice Seq2 indexes = %v; want [0 1]", indexes)
	}
	if len(values) != 2 || values[0] != "zero" || values[1] != "one" {
		t.Fatalf("slice Seq2 values = %v; want [zero one]", values)
	}
}

func TestGo126ReflectTypeIterators(t *testing.T) {
	structType := simreflect.TypeOf(struct {
		First  int
		Second string
	}{})
	var fields []string
	for field := range structType.Fields() {
		fields = append(fields, field.Name)
	}
	if len(fields) != 2 || fields[0] != "First" || fields[1] != "Second" {
		t.Fatalf("Fields names = %v; want [First Second]", fields)
	}

	methodType := simreflect.TypeOf(go126MethodSet{})
	var methods []string
	for method := range methodType.Methods() {
		methods = append(methods, method.Name)
	}
	if len(methods) != 2 || methods[0] != "Alpha" || methods[1] != "Beta" {
		t.Fatalf("Methods names = %v; want [Alpha Beta]", methods)
	}

	funcType := simreflect.TypeOf(func(int, string) (bool, error) { return false, nil })
	var inputs, outputs []simreflect.Kind
	for input := range funcType.Ins() {
		inputs = append(inputs, input.Kind())
	}
	for output := range funcType.Outs() {
		outputs = append(outputs, output.Kind())
	}
	if len(inputs) != 2 || inputs[0] != simreflect.Int || inputs[1] != simreflect.String {
		t.Fatalf("Ins kinds = %v; want [int string]", inputs)
	}
	if len(outputs) != 2 || outputs[0] != simreflect.Bool || outputs[1] != simreflect.Interface {
		t.Fatalf("Outs kinds = %v; want [bool interface]", outputs)
	}
}

func TestGo126ReflectValueIterators(t *testing.T) {
	value := simreflect.ValueOf(struct {
		First  int
		Second string
	}{First: 7, Second: "value"})
	var fields []string
	for field, fieldValue := range value.Fields() {
		fields = append(fields, field.Name+":"+fieldValue.Type().String())
	}
	if len(fields) != 2 || fields[0] != "First:int" || fields[1] != "Second:string" {
		t.Fatalf("Fields = %v; want [First:int Second:string]", fields)
	}

	methodValue := simreflect.ValueOf(go126MethodSet{})
	var methods []string
	for method, bound := range methodValue.Methods() {
		methods = append(methods, method.Name)
		bound.Call(nil)
	}
	if len(methods) != 2 || methods[0] != "Alpha" || methods[1] != "Beta" {
		t.Fatalf("Methods names = %v; want [Alpha Beta]", methods)
	}
}
