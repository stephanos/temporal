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

package proptest1

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.temporal.io/server/common/debug"
	"go.temporal.io/server/common/rpc"
)

type (
	ID     = string // TOOD: remove `=`
	TypeOf interface {
		typeOf() string
	}
)

// TODO: remove
func NewContext() context.Context {
	ctx, _ := rpc.NewContextWithTimeoutAndVersionHeaders(90 * time.Second * debug.TimeoutMultiplier)
	return ctx
}

func getTypeName[T any]() string {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	return typeToStr(ty, ty)
}

func typeToStr(val any, ty reflect.Type) string {
	if ty == nil {
		panic(fmt.Sprintf("type of %T is nil", ty))
	}
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	name := ty.String()
	if name == "" {
		panic(fmt.Sprintf("type of %T has empty name", val))
	}
	return name
}
