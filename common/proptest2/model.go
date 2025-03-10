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
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pborman/uuid"
	"github.com/qmuntal/stateless"
)

const (
	provisionalPrefix = "provisional-"
	initialized       = iota
)

var (
	_ envProvider      = (*Root)(nil)
	_ parentableEntity = (*Root)(nil)
)

type (
	// TODO: help with printing model string exactly once
	modelErr struct {
		model fmt.Stringer
		err   error
	}
	Model[M entity] struct {
		id           relativeID
		alias        relativeID
		entity       M
		initialised  bool
		name         string
		handlers     map[reflect.Type]func(any, any) error
		stateMachine *stateless.StateMachine
	}
	Parent[M entity] struct {
		envProvider
		entity M
	}
	parentableEntity interface {
		envProvider
	}
	modelOrParent interface {
		envProvider
		getParent() idEntity
	}
	Root struct {
		env *Env
	}
	idEntity interface {
		envProvider
	}
	identifiableEntity interface {
		envProvider   // implemented by Parent or Root
		modelOrParent // implemented by Parent or Root

		getModelName() string
		getID() relativeID
		getAlias() relativeID
	}
	entity interface {
		identifiableEntity
		parentableEntity

		getParent() idEntity
		SetAlias(string)
		MustSetID(string)
		GetID() string
		handleEvent(any) error
		sm() *stateless.StateMachine
		addEventHandler(reflect.Type, func(any, any) error)
		setEntity(entity)
		setModelName(string)
	}
	ResponseHandlerProvider interface {
		ResponseHandler() func(any, error)
	}
	// TODO: remove M and check at runtime
	modelOption[M entity] interface {
		apply(M)
	}
	modelFuncOpt[M entity] struct {
		applyFunc func(M)
	}
)

func (m *modelFuncOpt[M]) apply(entity M) {
	m.applyFunc(entity)
}

func GetOrCreateModel[M entity](
	initEvt any,
	entity M,
	opts ...modelOption[M],
) M {
	entity.setEntity(entity)
	entity.MustSetID(provisionalPrefix + uuid.New()) // TODO: use generator for deterministic IDs
	entity.setModelName(getTypeName[M]())
	for _, opt := range opts {
		opt.apply(entity)
	}
	if initEvt != nil { // TODO: remove and require init event
		Must1(entity.handleEvent(initEvt))
	}
	if existing, err := getByID[M](entity.getParent(), entity.getID()); err == nil {
		return existing
	}
	mustAdd(entity)
	Must1(entity.handleEvent(initialized))
	return entity
}

func NewParent[M entity](entity M) Parent[M] {
	return Parent[M]{
		entity:      entity,
		envProvider: entity,
	}
}

func NewRoot(env *Env) *Root {
	return &Root{env: env}
}

func WithInputFilter[M entity, E any](
	filter func(M, E) bool,
) modelOption[M] {
	return &modelFuncOpt[M]{
		applyFunc: func(m M) {
			// TODO
		},
	}
}

func WithInputHandler[M entity, E any](
	handler func(M, E) error,
) modelOption[M] {
	return &modelFuncOpt[M]{
		applyFunc: func(m M) {
			eventType := reflect.TypeOf((*E)(nil)).Elem()
			m.addEventHandler(eventType, func(entity any, input any) error {
				return handler(entity.(M), input.(E))
			})
		},
	}
}

func (m *Model[M]) Fire(trigger Trigger) error {
	env := m.entity.getEnv()
	env.testingT.Helper()

	return m.propagateChanges(func() error {
		if err := m.sm().Fire(trigger); err != nil {
			return fmt.Errorf("%v: %w", m, err)
		}
		return nil
	})
}

func (m *Model[M]) propagateChanges(fn func() error) error {
	env := m.entity.getEnv()
	env.testingT.Helper()

	previousID := m.getID()
	previousAlias := m.alias

	if err := fn(); err != nil {
		return fmt.Errorf("%v: %w", m, err)
	}

	// HACK so that the init event doesn't register its IDs before the entity is deduped
	if !m.initialised {
		return nil
	}

	if currentID := m.getID(); previousID != currentID {
		updateID(getTypeName[M](), m.entity, previousID, currentID)
	}
	if currentAlias := m.alias; previousAlias != currentAlias {
		updateID(getTypeName[M](), m.entity, previousAlias, currentAlias)
	}

	return nil
}

func (m *Model[M]) sm() *stateless.StateMachine {
	if m.stateMachine == nil {
		m.stateMachine = stateless.NewStateMachine(StateDraft)
		m.stateMachine.OnTransitioning(func(_ context.Context, t stateless.Transition) {
			m.entity.getEnv().Info(fmt.Sprintf("%v transition to %s\n", m, t.Destination))
		})
		m.sm().OnUnhandledTrigger(func(ctx context.Context, state stateless.State, trigger stateless.Trigger, unmetGuards []string) error {
			permitted, _ := m.sm().PermittedTriggers()
			return fmt.Errorf("cannot fire %q in state %s (permitted triggers: %v)",
				trigger, state, permitted)
		})
	}
	return m.stateMachine
}

func (m *Model[M]) addEventHandler(t reflect.Type, handler func(any, any) error) {
	if m.handlers == nil {
		m.handlers = make(map[reflect.Type]func(any, any) error)
	}
	if _, ok := m.handlers[t]; ok {
		m.entity.getEnv().Fatal(fmt.Sprintf("%v already has handler for %q", m, t))
	}
	m.handlers[t] = handler
}

func (m *Model[M]) ResponseHandler() func(any, error) {
	return func(resp any, err error) {
		var handleErr error
		if err != nil {
			handleErr = m.handleEvent(err)
		} else {
			handleErr = m.handleEvent(resp)
		}
		if handleErr != nil {
			m.entity.getEnv().Fatal(fmt.Sprintf("error handling response: %v", handleErr))
		}
	}
}

// TODO: support interfaces or not?
func (m *Model[M]) handleEvent(event any) error {
	env := m.entity.getEnv()
	env.testingT.Helper()

	// HACK so that the init event doesn't register its IDs before the entity is deduped
	if event == initialized {
		m.initialised = true
		return nil
	}

	return m.propagateChanges(func() error {
		eventType := reflect.TypeOf(event)
		if handler, ok := m.handlers[eventType]; ok {
			if err := handler(m.entity, event); err != nil {
				return fmt.Errorf("error handling %q: %w", eventType, err)
			}
			return nil
		}
		return fmt.Errorf("%v no handler for %q", m, eventType)
	})
}

func (m *Model[M]) String() string {
	env := m.entity.getEnv()
	env.testingT.Helper()

	params := []string{
		string(m.getID()),
		m.CurrentState().String(),
	}
	if alias := m.entity.getAlias(); alias != "" {
		params = append(params, string(alias))
	}
	return boldStr(fmt.Sprintf("[%s(%s)]", m.name, strings.Join(params, ",")))
}

func (m *Model[M]) setEntity(entity entity) {
	m.entity = entity.(M)
}

func (m *Model[M]) SetAlias(alias string) {
	m.alias = relativeID(alias)
}

func (m *Model[M]) getAlias() relativeID {
	return m.alias
}

func (m *Model[M]) getModelName() string {
	return m.name
}

func (m *Model[M]) getID() relativeID {
	if m.id == "" {
		m.entity.getEnv().Fatal(fmt.Sprintf("%v has empty ID", m.getModelName()))
	}
	return m.id
}

func (m *Model[M]) GetID() string {
	return string(m.entity.getID())
}

func (m *Model[M]) MustSetID(newID string) {
	currentID := m.id
	if newID == "" {
		m.entity.getEnv().Fatal(fmt.Sprintf("%v cannot set ID to empty string", m))
	}
	if currentID == relativeID(newID) {
		return
	}
	if currentID != "" && !currentID.isProvisional() {
		m.entity.getEnv().Fatal(fmt.Sprintf("%v cannot set ID to %v, ID is already set", m, newID))
	}
	m.id = relativeID(newID)
}

func (m *Model[M]) setModelName(name string) {
	m.name = name
}

func (m *Model[M]) CurrentState() State {
	state, _ := m.sm().State(context.Background())
	s, ok := state.(State)
	if !ok {
		m.entity.getEnv().Fatal(fmt.Sprintf("%v unexpected state type %T, expected type 'State'", m, state))
	}
	return s
}

func (m *Model[M]) Errorf(format string, args ...any) {
	m.entity.getEnv().testingT.Error(fmt.Sprintf(format, args...))
}

func (m *Model[M]) FailNow() {
	m.entity.getEnv().testingT.FailNow()
}

func (p *Parent[M]) getParent() idEntity {
	return p.entity
}

func (r *Root) getEnv() *Env {
	return r.env
}

func (r *Root) getParent() idEntity {
	return nil
}
