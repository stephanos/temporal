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

package proptest

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

const (
	idSeparator = "/"
)

type (
	Env struct {
		logger
		testingT           *testing.T
		indexLock          sync.RWMutex
		entitiesByTypeByID map[string]map[absoluteID]indexableEntity
	}
	envProvider interface {
		getEnv() *Env
	}
	indexableEntity interface {
		envProvider
		modelOrParent
		getID() relativeID
		getAlias() relativeID
		getModelName() string
	}
	relativeID string
	absoluteID string
)

func NewEnv(t *testing.T) *Env {
	return &Env{
		testingT:           t,
		logger:             newTestLogger(t),
		entitiesByTypeByID: make(map[string]map[absoluteID]indexableEntity),
	}
}

func mustAdd[M indexableEntity](entity M) {
	env := entity.getEnv()
	env.indexLock.Lock()
	defer env.indexLock.Unlock()

	internalAdd(entity.getModelName(), entity, entity.getID())
	if alias := entity.getAlias(); alias != "" {
		internalAdd(entity.getModelName(), entity, alias)
	}
}

func Get[M indexableEntity](
	parent modelOrParent, // TODO: this can be a Model or a Parent!
	id string,
) M {
	entity, err := getByID[M](parent, relativeID(id))
	if err != nil {
		parent.getEnv().Fatal(err.Error())
	}
	return entity
}

func getByID[M indexableEntity](
	parent idEntity,
	id relativeID,
) (M, error) {
	var zero M
	if parent == nil {
		return zero, fmt.Errorf("no parent found")
	}

	env := parent.getEnv()
	env.indexLock.RLock()
	defer env.indexLock.RUnlock()

	typeOf := getTypeName[M]()
	if _, ok := env.entitiesByTypeByID[typeOf]; !ok {

		return zero, fmt.Errorf("no entity of type %q was found", typeOf)
	}

	absID := makeAbsoluteID(parent, id)
	existingModel, ok := env.entitiesByTypeByID[typeOf][absID]
	if !ok {
		return zero, fmt.Errorf("%q with ID %q not found", typeOf, absID)
	}

	return existingModel.(M), nil
}

func updateID(
	typeOf string,
	entity indexableEntity,
	removeID, addID relativeID,
) {
	env := entity.getEnv()
	env.indexLock.Lock()
	defer env.indexLock.Unlock()

	if addID != "" {
		internalAdd(typeOf, entity, addID)
		if !addID.isProvisional() {
			env.Debug(fmt.Sprintf("%v registered %q\n", entity, addID))
		}
	}

	if removeID != "" {
		internalDel(typeOf, entity, removeID)
		if !removeID.isProvisional() {
			env.Info(fmt.Sprintf("%v unregistered %q\n", entity, removeID))
		}
	}
}

// expects lock to be held
func internalAdd[M indexableEntity](
	typeOf string,
	entity M,
	id relativeID,
) {
	env := entity.getEnv()
	if _, ok := env.entitiesByTypeByID[typeOf]; !ok {
		env.entitiesByTypeByID[typeOf] = make(map[absoluteID]indexableEntity)
	}

	absID := makeAbsoluteID(entity.getParent(), id)
	if _, ok := env.entitiesByTypeByID[typeOf][absID]; ok {
		env.Fatal(fmt.Sprintf("%q with ID %q already exists", typeOf, absID))
	}

	env.entitiesByTypeByID[typeOf][absID] = entity
}

func internalDel[M indexableEntity](
	typeOf string,
	entity M,
	id relativeID,
) {
	env := entity.getEnv()
	absID := makeAbsoluteID(entity.getParent(), id)
	delete(env.entitiesByTypeByID[typeOf], absID)
}

func makeAbsoluteID(
	parent idEntity, // TODO
	id relativeID,
) absoluteID {
	return absoluteID(idSeparator + string(id))
}

func (id relativeID) isProvisional() bool {
	return strings.HasPrefix(string(id), provisionalPrefix)
}
