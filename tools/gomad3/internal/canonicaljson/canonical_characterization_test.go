package canonicaljson

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestCanonicalJSONCharacterizesKinds(t *testing.T) {
	text := "<>&雪"
	var nested any = &text
	for _, test := range []struct {
		name  string
		value any
		want  string
	}{
		{"nil", nil, `null`},
		{"nil-interface-field", struct{ Value any }{}, `{"Value":null}`},
		{"nil-pointer", (*string)(nil), `null`},
		{"nil-map", map[string]string(nil), `null`},
		{"nil-slice", []string(nil), `null`},
		{"string", text, `"<>&雪"`},
		{"pointer", &text, `"<>&雪"`},
		{"nested-interface", struct{ Value any }{Value: &nested}, `{"Value":"<>&雪"}`},
		{"empty-map", map[string]string{}, `{}`},
		{"map", map[string]string{"z": "last", "a": text}, `{"a":"<>&雪","z":"last"}`},
		{"empty-slice", []string{}, `[]`},
		{"slice", []string{"first", text}, `["first","<>&雪"]`},
		{"empty-array", [0]string{}, `[]`},
		{"array", [2]string{"first", text}, `["first","<>&雪"]`},
		{"struct", struct {
			Zulu   string `json:"z"`
			Alpha  string `json:"a"`
			hidden string
			Ignore string `json:"-"`
		}{"last", text, "hidden", "ignored"}, `{"a":"<>&雪","z":"last"}`},
		{"bool", [2]bool{false, true}, `[false,true]`},
		{"int", [2]int{math.MinInt, math.MaxInt}, `[-9223372036854775808,9223372036854775807]`},
		{"int8", [2]int8{math.MinInt8, math.MaxInt8}, `[-128,127]`},
		{"int16", [2]int16{math.MinInt16, math.MaxInt16}, `[-32768,32767]`},
		{"int32", [2]int32{math.MinInt32, math.MaxInt32}, `[-2147483648,2147483647]`},
		{"int64", [2]int64{math.MinInt64, math.MaxInt64}, `[-9223372036854775808,9223372036854775807]`},
		{"uint", [2]uint{0, math.MaxUint}, `[0,18446744073709551615]`},
		{"uint8", [2]uint8{0, math.MaxUint8}, `[0,255]`},
		{"uint16", [2]uint16{0, math.MaxUint16}, `[0,65535]`},
		{"uint32", [2]uint32{0, math.MaxUint32}, `[0,4294967295]`},
		{"uint64", [2]uint64{0, math.MaxUint64}, `[0,18446744073709551615]`},
		{"uintptr", [2]uintptr{0, ^uintptr(0)}, `[0,18446744073709551615]`},
		{"float32-integral", float32(2), `2`},
		{"float64-integral", float64(2), `2`},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CanonicalJSON(test.value)
			t.Logf("bytes=%q error=%v", got, err)
			if err != nil || string(got) != test.want {
				t.Fatalf("CanonicalJSON() = %q, %v; want %q, nil", got, err, test.want)
			}
		})
	}
}

func TestCanonicalJSONCharacterizesInvalidStrings(t *testing.T) {
	invalid := "\xff"
	var nested any = &invalid
	for _, test := range []struct {
		name  string
		value any
	}{
		{"string", invalid},
		{"pointer", &invalid},
		{"interface", struct{ Value any }{Value: &nested}},
		{"map-key", map[string]string{invalid: "ok"}},
		{"map-value", map[string]string{"ok": invalid}},
		{"slice", []string{"ok", invalid}},
		{"array", [2]string{"ok", invalid}},
		{"struct-exported", struct{ Value string }{invalid}},
		{"struct-unexported", struct{ hidden string }{invalid}},
		{"struct-ignored", struct {
			Value string `json:"-"`
		}{invalid}},
		{"before-unsupported-field", struct {
			Unsupported chan int
			Value       string
		}{make(chan int), invalid}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CanonicalJSON(test.value)
			t.Logf("bytes=%q error=%v", got, err)
			if got != nil || err == nil || err.Error() != "JSON string is not valid UTF-8" {
				t.Fatalf("CanonicalJSON() = %q, %v; want nil, invalid UTF-8 error", got, err)
			}
		})
	}
}

func TestCanonicalJSONCharacterizesEncoderErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		want  string
	}{
		{"float32-fractional", float32(1.25), "floating-point JSON values are forbidden"},
		{"float64-fractional", float64(1.25), "floating-point JSON values are forbidden"},
		{"float32-nan", float32(math.NaN()), "encode JSON: json: unsupported value: NaN"},
		{"float64-nan", math.NaN(), "encode JSON: json: unsupported value: NaN"},
		{"float32-positive-inf", float32(math.Inf(1)), "encode JSON: json: unsupported value: +Inf"},
		{"float64-positive-inf", math.Inf(1), "encode JSON: json: unsupported value: +Inf"},
		{"float32-negative-inf", float32(math.Inf(-1)), "encode JSON: json: unsupported value: -Inf"},
		{"float64-negative-inf", math.Inf(-1), "encode JSON: json: unsupported value: -Inf"},
		{"complex64", complex64(1 + 2i), "encode JSON: json: unsupported type: complex64"},
		{"complex128", complex128(1 + 2i), "encode JSON: json: unsupported type: complex128"},
		{"channel", make(chan int), "encode JSON: json: unsupported type: chan int"},
		{"nil-channel", (chan int)(nil), "encode JSON: json: unsupported type: chan int"},
		{"function", func() {}, "encode JSON: json: unsupported type: func()"},
		{"nil-function", (func())(nil), "encode JSON: json: unsupported type: func()"},
		{"unsafe-pointer", unsafe.Pointer(new(int)), "encode JSON: json: unsupported type: unsafe.Pointer"},
		{"nil-unsafe-pointer", unsafe.Pointer(nil), "encode JSON: json: unsupported type: unsafe.Pointer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CanonicalJSON(test.value)
			t.Logf("bytes=%q error=%v", got, err)
			if got != nil || err == nil || err.Error() != test.want {
				t.Fatalf("CanonicalJSON() = %q, %v; want nil, %q", got, err, test.want)
			}
			if test.want == "floating-point JSON values are forbidden" {
				return
			}
			var valueError *json.UnsupportedValueError
			var typeError *json.UnsupportedTypeError
			if errors.As(err, &valueError) {
				if valueError.Error() != strings.TrimPrefix(test.want, "encode JSON: ") {
					t.Fatalf("encoder value error = %#v", valueError)
				}
			} else if errors.As(err, &typeError) {
				if typeError.Type != reflect.TypeOf(test.value) {
					t.Fatalf("encoder type error = %#v", typeError)
				}
			} else {
				t.Fatalf("encoder error lost its typed cause: %v", err)
			}
		})
	}
}

type characterizationMarshaler struct {
	Text  string
	Calls *int
	Err   error
}

func (value *characterizationMarshaler) MarshalJSON() ([]byte, error) {
	*value.Calls++
	return nil, value.Err
}

func TestCanonicalJSONCharacterizesMarshalerPrecedence(t *testing.T) {
	sentinel := errors.New("sentinel")
	for _, test := range []struct {
		name, text, want string
		calls            int
	}{
		{"valid", "ok", "encode JSON: json: error calling MarshalJSON for type *canonicaljson.characterizationMarshaler: sentinel", 1},
		{"invalid", "\xff", "JSON string is not valid UTF-8", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			value := &characterizationMarshaler{Text: test.text, Calls: &calls, Err: sentinel}
			got, err := CanonicalJSON(value)
			t.Logf("bytes=%q error=%v callbacks=%d", got, err, calls)
			if got != nil || err == nil || err.Error() != test.want || calls != test.calls {
				t.Fatalf("CanonicalJSON() = %q, %v, calls=%d; want nil, %q, calls=%d", got, err, calls, test.want, test.calls)
			}
			var marshalerError *json.MarshalerError
			if test.calls == 1 {
				if !errors.Is(err, sentinel) || !errors.As(err, &marshalerError) || marshalerError.Err != sentinel || marshalerError.Type != reflect.TypeOf(value) {
					t.Fatalf("marshaler cause lost: %v", err)
				}
			} else if errors.Is(err, sentinel) || errors.As(err, &marshalerError) {
				t.Fatalf("validation error acquired an encoder cause: %v", err)
			}
		})
	}
}

type characterizationNode struct {
	Next *characterizationNode
}

func TestCanonicalJSONCharacterizesCyclesAndSharedAliases(t *testing.T) {
	pointer := &characterizationNode{}
	pointer.Next = pointer
	object := map[string]any{}
	object["self"] = object
	sequence := make([]any, 1)
	sequence[0] = sequence
	sharedText := "ok"
	sharedMap := map[string]string{"value": "ok"}
	sharedSlice := []string{"ok"}
	for _, test := range []struct {
		name  string
		value any
		want  string
		err   string
	}{
		{"pointer-cycle", pointer, "", "encode JSON: json: unsupported value: encountered a cycle via *canonicaljson.characterizationNode"},
		{"map-cycle", object, "", "encode JSON: json: unsupported value: encountered a cycle via map[string]interface {}"},
		{"slice-cycle", sequence, "", "encode JSON: json: unsupported value: encountered a cycle via []interface {}"},
		{"shared-pointer", [2]*string{&sharedText, &sharedText}, `["ok","ok"]`, ""},
		{"shared-map", [2]map[string]string{sharedMap, sharedMap}, `[{"value":"ok"},{"value":"ok"}]`, ""},
		{"shared-slice", [2][]string{sharedSlice, sharedSlice}, `[["ok"],["ok"]]`, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CanonicalJSON(test.value)
			t.Logf("bytes=%q error=%v", got, err)
			if test.err == "" {
				if err != nil || string(got) != test.want {
					t.Fatalf("CanonicalJSON() = %q, %v; want %q, nil", got, err, test.want)
				}
				return
			}
			var cause *json.UnsupportedValueError
			if got != nil || err == nil || err.Error() != test.err || !errors.As(err, &cause) || cause.Error() != strings.TrimPrefix(test.err, "encode JSON: ") {
				t.Fatalf("CanonicalJSON() = %q, %v; want nil, %q with typed cycle cause", got, err, test.err)
			}
		})
	}
}

func TestCanonicalJSONCharacterizesSliceVisitWithoutLength(t *testing.T) {
	type shortStrings []string
	type longStrings []string
	storage := []string{"ok", "\xff"}
	for _, test := range []struct {
		name  string
		value any
		want  string
		err   string
	}{
		{"short-before-long", [2][]string{storage[:1], storage}, `[["ok"],["ok","�"]]`, ""},
		{"long-before-short", [2][]string{storage, storage[:1]}, "", "JSON string is not valid UTF-8"},
		{"different-named-types", struct {
			Short shortStrings
			Long  longStrings
		}{shortStrings(storage[:1]), longStrings(storage)}, "", "JSON string is not valid UTF-8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CanonicalJSON(test.value)
			t.Logf("bytes=%q error=%v", got, err)
			if test.err == "" {
				if err != nil || string(got) != test.want {
					t.Fatalf("CanonicalJSON() = %q, %v; want %q, nil", got, err, test.want)
				}
			} else if got != nil || err == nil || err.Error() != test.err {
				t.Fatalf("CanonicalJSON() = %q, %v; want nil, %q", got, err, test.err)
			}
		})
	}
}
