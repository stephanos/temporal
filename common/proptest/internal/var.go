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

package internal

import (
	"reflect"

	"google.golang.org/protobuf/proto"
)

var (
	protoMessageType = reflect.TypeFor[proto.Message]()
)

type (
	Variable struct {
		TypeOf   VarType
		Versions []VariableVersion
	}
	// TODO: don't export
	VariableVersion struct {
		Value   any
		Version int
	}
	VarType reflect.Type
)

func (v *Variable) Set(val any, version int) {
	if len(v.Versions) > 0 {
		currentVersion := v.Versions[len(v.Versions)-1]
		// Special case for proto messages
		// TODO: type alias doesn't work on newtypes
		//if proto.Equal(currentVersion.Value.(proto.Message), protoVal) {
		//	// no change
		//	return
		//}
		if reflect.DeepEqual(currentVersion.Value, val) {
			// no change
			return
		}
	}
	v.Versions = append(v.Versions, VariableVersion{Value: val, Version: version})
}

func (v *Variable) Get() (any, bool) {
	if len(v.Versions) == 0 {
		return nil, false
	}
	return v.Versions[len(v.Versions)-1].Value, true
}

func (v *Variable) Current() any {
	if len(v.Versions) == 0 {
		return nil
	}
	return v.Versions[len(v.Versions)-1].Value
}

func (v *Variable) CurrentOrDefault() any {
	if len(v.Versions) == 0 {
		return reflect.Zero(v.TypeOf).Interface()
	}
	return v.Versions[len(v.Versions)-1].Value
}

//func (v *Variable) GetChange() (prev any, new any, changed bool) {
//	zero := reflect.Zero(v.TypeOf).Interface()
//	switch len(v.Versions) {
//	case 0:
//		return zero, zero, false
//	case 1:
//		return zero, v.Versions[0].Value, false
//	default:
//		currentVersion := v.Versions[len(v.Versions)-1]
//		if currentVersion.Version != m.getEnv().currentTick {
//			return zero, zero, false
//		}
//		prevVersion := v.Versions[len(v.Versions)-2]
//		return prevVersion.Value, currentVersion.Value, true
//	}
//}
