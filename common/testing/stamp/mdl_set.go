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
	"strings"
)

var (
	embeddableMdlTypes = []reflect.Type{
		reflect.TypeOf(&Model[Root]{}).Elem(),
		reflect.TypeOf(&Scope[Root]{}).Elem(),
	}
)

type (
	ModelSet struct {
		typeIdx        map[modelType]struct{}
		typeByName     map[string]modelType
		exampleByType  map[modelType]modelWrapper
		childTypes     map[modelType][]modelType
		ruleIdx        map[modelType][]prop
		identHandlers  handlerSet
		actionHandlers handlerSet
	}
)

func NewModelSet() *ModelSet {
	return &ModelSet{
		typeIdx: make(map[modelType]struct{}),
		typeByName: map[string]modelType{
			rootTypeName: rootType,
		},
		exampleByType:  make(map[modelType]modelWrapper),
		childTypes:     make(map[modelType][]modelType),
		ruleIdx:        make(map[modelType][]prop),
		identHandlers:  newHandlerSet(),
		actionHandlers: newHandlerSet(),
	}
}

// TODO: type check parent matches model somehow?
// TODO: fail when 2 params of a handler have the same type
// TODO: check if handler contains unexpected parameter types (ie from outside of this context)
func RegisterModel[M modelWrapper](set *ModelSet) {
	mdlType := modelType(reflect.TypeFor[M]())
	mdlStructType := mdlType.Elem()
	mdlName := mdlStructType.String()

	if _, ok := set.typeIdx[mdlStructType]; ok {
		panic(fmt.Sprintf("%q already registered", mdlName))
	}
	set.typeIdx[mdlStructType] = struct{}{}
	set.typeByName[qualifiedTypeName(mdlStructType)] = mdlType

	for i := 0; i < mdlStructType.NumField(); i++ {
		field := mdlStructType.Field(i)
		parentMatch := scopeTypePattern.FindStringSubmatch(field.Type.String())
		if len(parentMatch) == 2 {
			fieldMdlTypeName := strings.TrimPrefix(parentMatch[1], "*")
			fieldMdlType, ok := set.typeByName[fieldMdlTypeName]
			if !ok {
				panic(fmt.Sprintf("%q field %q must be a registered model", mdlName, field.Name))
			}
			set.childTypes[fieldMdlType] = append(set.childTypes[fieldMdlType], mdlType)
		}
	}

	// create example of the model (used for identity checks)
	mdlInstance := reflect.New(mdlStructType).Interface().(modelWrapper)
	mdlInstance.setModel(&internalModel{typeOf: mdlType})
	set.exampleByType[mdlType] = mdlInstance

	// register model handlers
	var identityHandlerCount int
	mdlInstVal := reflect.ValueOf(mdlInstance)
	for i := 0; i < mdlType.NumMethod(); i++ {
		method := mdlType.Method(i)
		if isFromEmbedded(method) {
			continue
		}
		hdlr := newHandler(mdlType, method)

		switch {
		case strings.HasPrefix(method.Name, "Id"):
			if len(hdlr.inTypes) == 0 {
				panic(fmt.Sprintf("identity handler %q on %q must have parameters", method.Name, mdlName))
			}
			if len(hdlr.outTypes) != 1 {
				panic(fmt.Sprintf("identity handler %q on %q must return `ID`", method.Name, mdlName))
			}
			// TODO: check returns `ID`
			identityHandlerCount += 1
			set.identHandlers.add(hdlr)

		case strings.HasPrefix(method.Name, "On"):
			if len(hdlr.inTypes) == 0 {
				panic(fmt.Sprintf("action handler %q on %q must have parameters", method.Name, mdlName))
			}
			if len(hdlr.outTypes) != 0 {
				panic(fmt.Sprintf("action handler %q on %q must not return anything", method.Name, mdlName))
			}
			set.actionHandlers.add(newHandler(mdlType, method))

		case method.Type.Out(0).Implements(propType):
			if len(hdlr.inTypes) != 0 {
				panic(fmt.Sprintf("property %q on %q must not have any parameters", method.Name, mdlName))
			}
			if len(hdlr.outTypes) != 1 {
				panic(fmt.Sprintf("property %q on %q must return `Prop` or `Rule`", method.Name, mdlName))
			}
			newProp := hdlr.method.Func.Call([]reflect.Value{mdlInstVal})[0].Interface().(prop)
			newProp.SetName(fmt.Sprintf("%s.%s", mdlName, method.Name))
			if err := newProp.Validate(); err != nil {
				panic(err.Error())
			}

			// index properties that are rules
			if method.Type.Out(0) == ruleType {
				set.ruleIdx[mdlType] = append(set.ruleIdx[mdlType], newProp)
			}

		default:
			// TODO: elaborate
			panic(fmt.Sprintf("unsupported method %q on %q", method.Name, mdlName))
		}
	}

	if identityHandlerCount == 0 {
		panic(fmt.Sprintf("%q has no identity handlers", mdlName))
	}
}

func (s ModelSet) trigger(mw modelWrapper, action Action) {
	s.actionHandlers.invokeBestMatch(mw, action, false)
}

func (s ModelSet) exampleOf(t modelType) modelWrapper {
	return s.exampleByType[t]
}

func (s ModelSet) identify(mw modelWrapper, action Action) ID {
	// if it's an internal action and an existing model, try to see if a model reference exists in the action
	if action.containsRefTo(mw) {
		return mw.getDomainID()
	}
	// TODO: special case: if action actor is the same as action target, try to match mw here
	// if it's an external action ...
	if res := s.identHandlers.invokeBestMatch(mw, action, true); len(res) == 1 {
		return res[0].Interface().(ID)
	}
	return "" // no identity handler was found
}

func (s ModelSet) childTypesOf(m modelWrapper) []modelType {
	return s.childTypes[m.getType()]
}

func (s ModelSet) propertiesOf(m modelWrapper) []prop {
	return s.ruleIdx[m.getType()]
}

func isFromEmbedded(method reflect.Method) bool {
	for _, typ := range embeddableMdlTypes {
		if _, ok := typ.MethodByName(method.Name); ok {
			return true
		}
	}
	return false
}
