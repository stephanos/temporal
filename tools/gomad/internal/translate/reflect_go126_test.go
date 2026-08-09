package translate

import (
	"testing"

	simreflect "github.com/jellevandenhooff/gosim/internal/reflect"
)

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
