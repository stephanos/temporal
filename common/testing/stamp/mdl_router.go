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
	"sync"
)

type (
	Router struct {
		env        modelEnv
		routerVal  reflect.Value
		routeIndex map[string]reflect.Method

		routeContextLock sync.RWMutex
		routeContexts    map[ActID]*routerContext
	}
	OnComplete    func(action)
	routerWrapper interface {
		getRouter() *Router
	}
	routerContext struct {
		onMatchedFunc  func(routableAction, modelWrapper)
		modelCallbacks []func(reflect.Value)
	}
)

// TODO: type-check parent
func Consume[T, P modelWrapper](
	r routerWrapper,
	parent Model[P],
	id ID,
	act routableAction,
) Model[T] {
	return Model[T]{
		internalModel: r.getRouter().consume(reflect.TypeFor[T](), parent.getModel(), id, act),
	}
}

func initRouter(rw routerWrapper, env modelEnv) {
	routeIndex := make(map[string]reflect.Method)

	routerType := reflect.TypeOf(rw)
	for i := 0; i < routerType.NumMethod(); i++ {
		method := routerType.Method(i)
		methodName := method.Name

		if !method.IsExported() || !strings.HasPrefix(methodName, "On") {
			panic(fmt.Sprintf("Invalid route %q: does not start with 'On'\n", methodName))
		}

		methodType := method.Type
		if methodType.NumIn() != 2 {
			panic(fmt.Sprintf("Route %q has %d parameters, expected 1\n", methodName, methodType.NumIn()-1))
		}
		// TODO: callback is optional
		//if methodType.NumOut() != 1 {
		//	panic(fmt.Sprintf("Route %q has %d return values, expected 1\n", methodName, methodType.NumOut()))
		//}
		//if methodType.Out(0).Kind() != reflect.Func {
		//	panic(fmt.Sprintf("Route %q has return type %q, expected func\n", methodName, methodType.Out(0)))
		//}

		incomingActionType := methodType.In(1)
		routeIndex[mustGetTypeParam(incomingActionType)] = method
	}

	if len(routeIndex) == 0 {
		panic("No routes were found")
	}

	r := rw.getRouter()
	r.routeContexts = make(map[ActID]*routerContext)
	r.routerVal = reflect.ValueOf(rw)
	r.routeIndex = routeIndex
	r.env = env
}

func (r *Router) route(
	incomingAct routableAction,
	onMatchedFunc func(routableAction, modelWrapper),
) OnComplete {
	ctx := routerContext{onMatchedFunc: onMatchedFunc}
	ctxID := incomingAct.ID()
	r.routeContextLock.Lock()
	r.routeContexts[ctxID] = &ctx
	r.routeContextLock.Unlock()

	// identify which route to invoke
	route := incomingAct.Route()
	method, ok := r.routeIndex[route]
	if !ok {
		return nil
	}

	// invoke route
	incActType := method.Type.In(1)
	castIncAct := mustCastVal(reflect.ValueOf(incomingAct), incActType)
	routeCallback := method.Func.Call([]reflect.Value{
		r.routerVal,
		castIncAct,
	})

	// return callback for outgoing action
	return func(outgoingAct action) {
		outFuncType := method.Type.Out(0)
		outActType := outFuncType.In(0)
		typedOutAct := mustCastVal(reflect.ValueOf(outgoingAct), outActType)

		// invoke the router callback
		if routeCallback != nil && !routeCallback[0].IsNil() {
			routeCallback[0].Call([]reflect.Value{typedOutAct})
		}

		// invoke model callbacks
		r.routeContextLock.Lock()
		for _, mdlCallback := range ctx.modelCallbacks {
			mdlCallback(typedOutAct)
		}
		delete(r.routeContexts, ctxID)
		r.routeContextLock.Unlock()
	}
}

func (r *Router) consume(
	mdlType reflect.Type,
	parent *internalModel,
	id ID,
	act routableAction,
) *internalModel {
	if id == "" {
		panic(fmt.Sprintf("empty ID provided for %v", mdlType))
	}

	router := r.getRouter()
	mdl, cb := router.env.consume(mdlType, parent.getModel(), id, act)

	r.routeContextLock.Lock()
	ctx := r.routeContexts[act.ID()]
	if ctx.onMatchedFunc != nil {
		ctx.onMatchedFunc(act, mdl)
	}
	if cb != nil {
		ctx.modelCallbacks = append(ctx.modelCallbacks, cb)
	}
	r.routeContextLock.Unlock()

	return mdl.getModel()
}

func (r *Router) getRouter() *Router {
	return r
}
