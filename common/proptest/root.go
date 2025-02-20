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
	"reflect"
)

var (
	root                 = Root{env: nil}
	rootID       modelID = -1
	rootScope            = scope{id: "", typeOf: reflect.TypeFor[Root]()}
	rootType             = reflect.TypeFor[Root]()
	rootTypeName         = qualifiedTypeName(rootType)
)

type Root struct {
	env *Env
}

func (r Root) getDomainID() ID {
	return "/"
}

func (r Root) getEnv() *Env {
	return r.env
}

func (r Root) getScope() scope {
	return rootScope
}

func (r Root) setModel(*internalModel) {
	panic("not supported")
}

func (r Root) str() string {
	return "Root()"
}

func (r Root) getModel() *internalModel {
	return nil
}

func (r Root) getType() wrapperType {
	return reflect.TypeOf(r)
}

func (r Root) getID() modelID {
	return rootID
}

func (r Root) GetID() string {
	panic("not supported")
}
