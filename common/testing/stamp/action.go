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
// THE SOFTMARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package stamp

import (
	"reflect"
	"strings"
)

type (
	Action interface {
		containsRefTo(wrapper modelWrapper) bool
		getPayload() map[reflect.Type]reflect.Value
	}
	action struct {
		payloads     []any
		valuesByType map[reflect.Type]reflect.Value
	}
	TriggerID  ID
	triggerEnv interface{}
)

func NewAction(payloads ...any) *action {
	if len(payloads) == 0 {
		panic("empty action")
	}
	vals := map[reflect.Type]reflect.Value{}
	for _, payload := range payloads {
		if payload == nil {
			continue
		}
		t := reflect.TypeOf(payload)
		v := reflect.ValueOf(payload)
		vals[t] = v
	}
	return &action{
		payloads:     payloads,
		valuesByType: vals,
	}
}

func (a *action) Append(payloads ...any) *action {
	newAction := NewAction(append(a.payloads, payloads...)...)
	return newAction
}

func (a *action) getPayload() map[reflect.Type]reflect.Value {
	return a.valuesByType
}

func (a *action) containsRefTo(mw modelWrapper) bool {
	if mw.getDomainID() == "" {
		// If the model doesn't exist yet, it will never be referenced.
		return false
	}
	for ty, val := range a.valuesByType {
		if ty.Kind() == reflect.Struct {
			for i := 0; i < ty.NumField(); i++ {
				if !ty.Field(i).IsExported() {
					continue
				}
				if fieldModel, ok := val.Field(i).Interface().(modelWrapper); ok {
					checkMdl := fieldModel.getScope()
					for checkMdl.getID() != rootID {
						if fieldModel == mw {
							return true
						}
						checkMdl = checkMdl.getScope()
					}
				}
			}
		}
	}
	return false
}

func (a *action) String() string {
	var s strings.Builder
	s.WriteString("Action[ ")
	for _, payload := range a.payloads {
		if payload == nil {
			continue
		}
		s.WriteString(reflect.TypeOf(payload).String())
		s.WriteString(" ")
	}
	s.WriteString("]")
	return s.String()
}
