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
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sync"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/softassert"
)

var (
	scopeTypePattern = regexp.MustCompile(`Scope\[(.*)]`)
)

type (
	ModelEnv struct {
		log.Logger

		root                Root
		assert              *require.Assertions
		lock                sync.Mutex
		modelSet            ModelSet
		currentTick         int
		currentMdlCounter   modelID
		childrenIdx         map[modelID]map[ID]modelWrapper
		effectHandlers      handlerSet
		triggerEnv          triggerEnv
		callbackLock        sync.RWMutex
		triggerCallbacksIdx map[TriggerID]func(modelWrapper)
	}
	modelID int // internal identifier only - different from public domain ID
)

func NewModelEnv(
	logger log.Logger,
	modelSet ModelSet,
	triggerEnv triggerEnv,
) *ModelEnv {
	res := &ModelEnv{
		Logger:              newLogger(logger),
		modelSet:            modelSet,
		effectHandlers:      newHandlerSet(),
		triggerEnv:          triggerEnv,
		childrenIdx:         make(map[modelID]map[ID]modelWrapper),
		triggerCallbacksIdx: make(map[TriggerID]func(modelWrapper)),
	}
	res.root = Root{env: res}

	triggerEnvValue := reflect.ValueOf(triggerEnv)
	triggerEnvType := triggerEnvValue.Type()
	for i := 0; i < triggerEnvType.NumMethod(); i++ {
		hdlr := newHandler(triggerEnvType, triggerEnvType.Method(i))
		if len(hdlr.inTypes) == 0 {
			logger.Fatal(fmt.Sprintf("method %q must have at least 1 parameter", hdlr.method.Name))
		}
		if len(hdlr.outTypes) != 0 {
			logger.Fatal(fmt.Sprintf("method %q must not have any return values", hdlr.method.Name))
		}
		res.effectHandlers.add(hdlr)
	}

	return res
}

func (e *ModelEnv) Handle(act Action, triggerID TriggerID) {
	defer func() {
		// recover from panics to not disturb the test
		if r := recover(); r != nil {
			softassert.Fail(e, fmt.Sprintf("%v", r))
		}
	}()

	e.lock.Lock()
	defer func() {
		e.currentTick += 1
		e.lock.Unlock()
	}()

	// update models
	e.walk(&e.root, func(parent, child modelWrapper) {
		if child == nil {
			e.createNewChildren(parent, act)
		} else {
			e.updateExistingChild(child, act, triggerID)
		}
	})

	// verify props
	e.walk(&e.root, func(_, child modelWrapper) {
		if child == nil {
			return
		}
		for _, p := range e.modelSet.propertiesOf(child) {
			res, err := p.eval(child.getModel().evalCtx)
			if err != nil {
				e.Error(err.Error())
			}
			if res != true {
				e.Error(propErr{prop: p.String(), err: PropViolated}.Error())
			}
		}
	})
}

func (e *ModelEnv) createNewChildren(
	parent modelWrapper,
	action Action,
) {
	for _, mdlType := range e.modelSet.childTypesOf(parent) {
		// check identity (using an example model to not create a new one)
		exampleMdl := e.modelSet.exampleOf(mdlType)
		id := e.modelSet.identify(exampleMdl, action)
		if id == "" {
			// unable to identify
			continue
		}
		parentModelID := parent.getID()
		if _, ok := e.childrenIdx[parentModelID][id]; ok {
			// already exists
			continue
		}

		// create new model
		newChild := newModel(e, mdlType, parent)
		if id2 := e.modelSet.identify(newChild, action); id2 != id { // have to run it again for side-effects
			e.Fatal(fmt.Sprintf("identity mismatch: %q != %q", id, id2))
		}
		newChild.setDomainID(id)
		e.Info(fmt.Sprintf("%s created by %s", newChild.str(), boldStr(action)))

		// index model
		if _, ok := e.childrenIdx[parentModelID]; !ok {
			e.childrenIdx[parentModelID] = make(map[ID]modelWrapper)
		}
		e.childrenIdx[parentModelID][id] = newChild

		// no need to do anything else here!
		// child will be visited next automatically
	}
	return
}

func (e *ModelEnv) updateExistingChild(
	child modelWrapper,
	action Action,
	triggerID TriggerID,
) {
	// check identity
	id := e.modelSet.identify(child, action)
	if id == "" {
		// unable to identify
		return
	}
	if id != child.getDomainID() {
		// not the same model instance
		return
	}

	e.Info(fmt.Sprintf("%s invoked by '%v'", child.str(), boldStr(action)))

	// invoke action handler
	e.modelSet.trigger(child, action)

	// invoke trigger callback
	if triggerID != "" {
		e.callbackLock.RLock()
		if cb, ok := e.triggerCallbacksIdx[triggerID]; ok {
			cb(child)
		}
		e.callbackLock.RUnlock()
	}
}

func (e *ModelEnv) walk(start modelWrapper, fn func(modelWrapper, modelWrapper)) {
	var walkModelsFn func(modelWrapper)
	walkModelsFn = func(mw modelWrapper) {
		fn(mw, nil) // might create new children

		for _, child := range e.childrenIdx[mw.getID()] {
			fn(mw, child)
			walkModelsFn(child)
		}
	}
	walkModelsFn(start)
}

func (e *ModelEnv) nextID() modelID {
	e.currentMdlCounter++
	return e.currentMdlCounter
}

func (e *ModelEnv) Seed() int {
	// TODO: allow providing a seed
	return 0
}

// TODO: with timeout
func (e *ModelEnv) Context() context.Context {
	return context.Background()
}

func (e *ModelEnv) Root() *Root {
	return &e.root
}

func (e *ModelEnv) onMatched(triggerID TriggerID, f func(modelWrapper)) func() {
	e.callbackLock.Lock()
	defer e.callbackLock.Unlock()

	e.triggerCallbacksIdx[triggerID] = f
	return func() {
		delete(e.triggerCallbacksIdx, triggerID)
	}
}
