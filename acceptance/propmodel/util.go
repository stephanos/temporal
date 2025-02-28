// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package propmodel

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TODO: remove again and make variable clone
func protoClone[T proto.Message](msg T) T {
	return proto.Clone(msg).(T)
}

func findProtoValueByNameType[T any](
	msg proto.Message,
	name protoreflect.Name,
	kind protoreflect.Kind,
) T {
	return findProtoValue[T](msg, func(field protoreflect.FieldDescriptor) bool {
		return field.Name() == name && field.Kind() == kind
	})
}

func findProtoValue[T any](
	msg proto.Message,
	matchFn func(protoreflect.FieldDescriptor) bool,
) T {
	var res T
	if v := findProtoValueInternal(msg, matchFn); v != nil {
		res = toValue[T](*v)
	}
	return res
}

func findProtoValueInternal(
	msg proto.Message,
	matchFn func(protoreflect.FieldDescriptor) bool,
) *protoreflect.Value {
	var res *protoreflect.Value
	msgReflect := msg.ProtoReflect()
	msgReflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if matchFn(fd) {
			res = &v
			return false
		}
		if fd.Kind() == protoreflect.MessageKind {
			if fd.IsList() {
				for i := 0; i < v.List().Len(); i++ {
					if res = findProtoValueInternal(v.List().Get(i).Message().Interface(), matchFn); res != nil {
						return false
					}
				}
			} else if !fd.IsMap() {
				if res = findProtoValueInternal(v.Message().Interface(), matchFn); res != nil {
					return false
				}
			}
		}
		return true
	})
	return res
}

func toValue[T any](v protoreflect.Value) T {
	var res T
	switch any(res).(type) {
	case string:
		return any(v.String()).(T)
	case int64:
		return any(v.Int()).(T)
	case float64:
		return any(v.Float()).(T)
	case bool:
		return any(v.Bool()).(T)
	case []uint8:
		return any(v.Bytes()).(T)
	default:
		panic(fmt.Sprintf("unsupported type %T", res))
	}
}
