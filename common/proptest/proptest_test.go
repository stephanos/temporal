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
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/proptest/expect"
)

type (
	Foo struct {
		Model[Foo]
		Root Scope[Root]
	}
	FooBar struct {
		Model[FooBar]
		Root Scope[Foo]
	}
	request[T any] struct {
		id   string
		data T
	}
	response[T any] struct {
		data T
	}
	LastReqData           string
	LastRespData          string
	CompletedConversation bool
)

func TestSend(t *testing.T) {
	env := NewEnv(t)
	RegisterModel[Foo](env)
	RegisterModel[FooBar](env)

	errs := env.Send(request[string]{id: "a", data: "hello"})
	require.Len(t, errs, 2)
	require.ErrorContains(t, errs[0], "check for 'proptest.LastRespData' failed: 'Should not be zero, but was '")
	require.ErrorContains(t, errs[1], "check for 'proptest.CompletedConversation' failed: 'Should be true'")

	models := 0
	env.walk(func(_, mw modelWrapper) bool {
		switch m := mw.(type) {
		case *Foo:
			require.Equal(t, "a", string(m.getDomainID()))
			require.Equal(t, "hello", string(MustGet[LastReqData](m)))
			require.False(t, bool(MustGet[CompletedConversation](m)))
			models++
		case *FooBar:
			require.Equal(t, "a", string(m.getDomainID()))
			require.Equal(t, "hello", string(MustGet[LastReqData](m)))
			require.False(t, bool(MustGet[CompletedConversation](m)))
			models++
		}
		return true
	})
	require.Equal(t, 2, models)

	errs = env.Send(request[string]{id: "a", data: "hello"}) // should not change anything
	require.Len(t, errs, 2)

	models = 0
	env.walk(func(_, mw modelWrapper) bool {
		switch m := mw.(type) {
		case *Foo:
			require.Equal(t, "a", string(m.getDomainID()))
			require.Equal(t, "hello", string(MustGet[LastReqData](m)))
			require.False(t, bool(MustGet[CompletedConversation](m)))
			models++
		case *FooBar:
			require.Equal(t, "a", string(m.getDomainID()))
			require.Equal(t, "hello", string(MustGet[LastReqData](m)))
			require.False(t, bool(MustGet[CompletedConversation](m)))
			models++
		}
		return true
	})
	require.Equal(t, 2, models)

	errs = env.Send(request[string]{id: "a", data: "hello"}, response[string]{data: "hola"})
	require.Empty(t, errs)

	models = 0
	env.walk(func(_, mw modelWrapper) bool {
		switch m := mw.(type) {
		case *Foo:
			require.Equal(t, "a", string(m.getDomainID()))             // updated
			require.Equal(t, "hello", string(MustGet[LastReqData](m))) // same
			require.Equal(t, "hola", string(MustGet[LastRespData](m)))
			require.True(t, bool(MustGet[CompletedConversation](m)))
			models++
		case *FooBar:
			require.Equal(t, "a", string(m.getDomainID()))             // updated
			require.Equal(t, "hello", string(MustGet[LastReqData](m))) // same
			require.Equal(t, "hola", string(MustGet[LastRespData](m)))
			require.True(t, bool(MustGet[CompletedConversation](m)))
			models++
		}
		return true
	})
	require.Equal(t, 2, models)
}

func (f *Foo) IdRequest(req request[string]) ID { // TODO: make this a var like all the others?
	return ID(req.id)
}

func (f *Foo) OnRequest(req request[string]) LastReqData {
	return LastReqData(req.data)
}

func (f *Foo) OnResponse(req request[string], resp response[string]) LastRespData {
	return LastRespData(resp.data)
}

func (f *Foo) GetCompletedConversation(reqData LastReqData, respData LastRespData) CompletedConversation {
	return reqData != "" && respData != ""
}

func (fb *FooBar) IdRequest(req request[string]) ID {
	return ID(req.id)
}

func (fb *FooBar) OnRequest(req request[string]) LastReqData {
	return LastReqData(req.data)
}

func (fb *FooBar) Verify() expect.Rule {
	return All(
		expect.NotZero[LastReqData](),
		expect.NotZero[LastRespData](),
		expect.True[CompletedConversation](),
	)
}

func (fb *FooBar) OnResponse(req request[string], resp response[string]) LastRespData {
	return LastRespData(resp.data)
}

func (fb *FooBar) GetCompletedConversation(reqData LastReqData, respData LastRespData) CompletedConversation {
	return reqData != "" && respData != ""
}
