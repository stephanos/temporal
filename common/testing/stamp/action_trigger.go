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

	"github.com/google/uuid"
)

type (
	TriggerActor[A modelWrapper] struct {
		actor A
	}
	TriggerTarget[M modelWrapper] struct {
		mdl M
	}
	trigger[A, M modelWrapper] interface {
		GetActor() A
		getTarget() M
	}
)

func Trigger[M, A modelWrapper](
	actor A,
	trigger trigger[A, M],
) M {
	env := actor.getModel().getEnv()

	// copy trigger to assign actor
	copyVal := reflect.New(reflect.TypeOf(trigger))
	copyVal.Elem().Set(reflect.ValueOf(trigger))
	copyVal.Elem().FieldByName("TriggerActor").Set(reflect.ValueOf(TriggerActor[A]{actor: actor}))
	newTrigger := copyVal.Elem().Interface()

	// setup trigger callback
	var res M
	var matched bool
	var matchPath []string
	triggerID := TriggerID(uuid.NewString())
	env.onMatched(triggerID, func(mdl modelWrapper) {
		if matchedMdl, ok := mdl.(M); ok {
			matched = true
			res = matchedMdl
		}
		if !matched {
			// track path for reporting in case it doesn't match
			matchPath = append(matchPath, mdl.str())
		}
	})

	// convert trigger to action, if possible
	act := NewAction(newTrigger, triggerID, actor)
	getFn := reflect.ValueOf(newTrigger).MethodByName("Get")
	if getFn.IsValid() && getFn.Type().NumIn() == 1 && getFn.Type().NumOut() == 1 {
		generated := getFn.Call([]reflect.Value{reflect.ValueOf(env)})
		act = act.Append(act, generated[0].Interface())
	} else {
		act = act.Append(act, newTrigger)
	}

	env.Info(fmt.Sprintf("Triggering '%v'", boldStr(act)))

	// let model handle the action
	env.Handle(act, triggerID)

	// invoke effect handler, if any
	env.effectHandlers.invokeBestMatch(env.triggerEnv, act, true)

	// verify that the trigger arrived at the target
	if !matched {
		env.Fatal(fmt.Sprintf(`
Action '%s' by actor '%s' did not arrive at target '%s'.

Path: %v

Input: %#v
`,
			boldStr(reflect.TypeOf(trigger).String()),
			boldStr(reflect.TypeOf(actor).String()),
			boldStr(reflect.TypeOf(trigger.getTarget()).String()),
			strings.Join(matchPath, " -> "),
			newTrigger))
	}

	return res
}

func (tt TriggerTarget[M]) getTarget() M {
	return tt.mdl
}

func (ta TriggerActor[A]) GetActor() A {
	return ta.actor
}
