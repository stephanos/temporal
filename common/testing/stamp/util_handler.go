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

package stamp

import (
	"fmt"
	"reflect"
)

type (
	handler struct {
		ty         reflect.Type
		method     reflect.Method
		inTypes    []reflect.Type
		outTypes   []reflect.Type
		fuzzyMatch bool
	}
	handlerSet struct {
		byType map[reflect.Type][]*handler
	}
)

func newHandlerSet() handlerSet {
	return handlerSet{
		byType: make(map[reflect.Type][]*handler),
	}
}

func newHandler(
	ty reflect.Type,
	method reflect.Method,
) *handler {
	if method.Type.NumIn() == 0 {
		panic(fmt.Sprintf("handler %q on %q must have at least one argument", method.Name, ty))
	}

	knownTypes := map[reflect.Type]struct{}{}
	var inTypes []reflect.Type
	for j := 1; j < method.Type.NumIn(); j++ {
		inType := method.Type.In(j)
		if _, ok := knownTypes[inType]; ok {
			panic(fmt.Sprintf("handler %q on %q has duplicate argument type %q", method.Name, ty, inType))
		}
		inTypes = append(inTypes, inType)
		knownTypes[inType] = struct{}{}
	}

	var outTypes []reflect.Type
	for j := 0; j < method.Type.NumOut(); j++ {
		outType := method.Type.Out(j)
		outTypes = append(outTypes, outType)
	}

	return &handler{
		ty:       ty,
		method:   method,
		inTypes:  inTypes,
		outTypes: outTypes,
	}
}

func (s *handlerSet) add(h *handler) {
	if _, ok := s.byType[h.ty]; !ok {
		s.byType[h.ty] = []*handler{}
	}
	s.byType[h.ty] = append(s.byType[h.ty], h)
}

func (s *handlerSet) invokeBestMatch(
	receiver any,
	action Action,
	includeInterfaceType bool,
) []reflect.Value {
	handlers := s.byType[reflect.TypeOf(receiver)]
	matches := map[*handler][]reflect.Value{}
	for _, h := range handlers {
		if m := h.matches(action, includeInterfaceType); len(m) == len(h.inTypes) {
			matches[h] = m
		}
	}
	if len(matches) == 0 {
		return nil
	}
	var bestMatch *handler
	for cb, numArgs := range matches {
		if bestMatch == nil || len(numArgs) > len(matches[bestMatch]) {
			bestMatch = cb
		}
	}
	if bestMatch == nil {
		return nil
	}
	return bestMatch.invoke(receiver, matches[bestMatch])
}

func (h *handler) matches(
	action Action,
	includeInterfaceType bool,
) []reflect.Value {
	payload := action.getPayload() // TODO: fetch values by type one-by-one instead, like `GetVal(type)`
	var matchingArgs []reflect.Value
	for _, inType := range h.inTypes {
		if _, ok := payload[inType]; ok {
			// exact match
			matchingArgs = append(matchingArgs, payload[inType])
		} else if includeInterfaceType && inType.Kind() == reflect.Interface {
			// interface match
			for _, arg := range payload {
				if arg.Type().Implements(inType) {
					matchingArgs = append(matchingArgs, arg)
					break
				}
			}
		}
	}
	return matchingArgs
}

func (h *handler) invoke(
	receiver any,
	args []reflect.Value,
) []reflect.Value {
	return h.method.Func.Call(append(
		[]reflect.Value{reflect.ValueOf(receiver)},
		args...,
	))
}
