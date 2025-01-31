// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package telemetry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	maxDepth           = 2
	unserializableNote = `"<serialization error: %v>"`
)

func DebugSnapshot(
	span trace.Span,
	label string,
	val any,
) {
	if span.IsRecording() && DebugMode() {
		var data []byte
		defer func() {
			var payload string
			if r := recover(); r != nil {
				payload = fmt.Sprintf(unserializableNote, r)
			} else {
				payload = string(data)
			}
			span.AddEvent("temporal.snapshot."+label,
				trace.WithAttributes(
					attribute.String("payload", payload),
					attribute.String("label", "snapshot of "+label),
				))
		}()

		v := reflect.ValueOf(val)
		if val == nil || v.IsNil() {
			data = json.RawMessage("{}")
		} else {
			data = valToJSON(v, v, 0)
		}
	}
}

func valToJSON(
	v reflect.Value,
	startV reflect.Value,
	depth int,
) []byte {
	if depth > 0 && v == startV {
		// prevent repeated reference
		// TODO: doesn't work?
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		return json.RawMessage(`"` + v.String() + `"`)

	case reflect.Invalid, reflect.Func, reflect.Chan:
		return nil

	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		if msg, ok := v.Interface().(proto.Message); ok {
			return marshal(msg)
		}
		if v.Kind() == reflect.Interface && depth > 0 {
			// don't go into any struct field or map value that's an interface
			return nil
		}
		return valToJSON(v.Elem(), startV, depth) // don't count pointers towards depth!

	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return nil
		}

		var items []string
		for i := 0; i < v.Len(); i++ {
			item := valToJSON(v.Index(i), startV, depth) // don't count slices towards depth!
			items = append(items, string(item))
		}
		return json.RawMessage("[" + strings.Join(items, ", ") + "]")

	case reflect.Struct:
		t := v.Type()
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return json.RawMessage(`"` + v.Interface().(time.Time).Format(time.RFC3339Nano) + `"`)
		}
		if depth > maxDepth {
			return nil
		}

		data := make(map[string]any)
		for i := 0; i < t.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldName := t.Field(i).Name
			if !fieldVal.CanInterface() && fieldVal.CanAddr() {
				// access unexported fields
				pointer := unsafe.Pointer(fieldVal.UnsafeAddr())
				fieldVal = reflect.NewAt(fieldVal.Type(), pointer).Elem()
			}
			if fieldVal.IsZero() {
				continue
			}
			valJSON := valToJSON(fieldVal, startV, depth+1)
			if len(valJSON) == 0 {
				continue
			}
			data[fieldName] = json.RawMessage(valJSON)
		}
		if depth > 0 && len(data) == 0 {
			return nil
		}
		return marshal(data)

	case reflect.Map:
		if v.Len() == 0 {
			return nil
		}
		if depth > maxDepth {
			return nil
		}

		data := make(map[string]any)
		for _, key := range v.MapKeys() {
			var keyStr string
			keyStr = string(valToJSON(key, startV, depth+1))
			valJSON := json.RawMessage(valToJSON(v.MapIndex(key), startV, depth+1))
			if len(valJSON) == 0 {
				continue
			}
			data[keyStr] = valJSON
		}
		if depth > 0 && len(data) == 0 {
			return nil
		}
		return marshal(data)

	default:
		return marshal(v.Interface())
	}
}

func marshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return captureErr(fmt.Errorf("failed to marshal json message: %w", err))
	}
	return data
}

func captureErr(err error) []byte {
	return json.RawMessage(fmt.Sprintf(unserializableNote, err))
}
