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

package propcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

var (
	_ Run = &run{}
	_ Run = &FakeRun{}
)

type (
	Run interface {
		require.TestingT
		Logf(string, ...any)
		Fatalf(string, ...any)
		Helper()
	}
	run struct {
		testingT testing.TB
		rapidT   *rapid.T // only present when running PBT
	}
	FakeRun struct { // TODO: unexport
	}
	Runnable interface {
		Run() Run
	}
)

func NewExampleRun(t testing.TB) Run {
	return &run{
		testingT: t,
	}
}

func NewFuzzRun(t *rapid.T) Run {
	return &run{
		rapidT: t,
	}
}

func (r run) Helper() {
}

func (r run) Logf(s string, a ...any) {
	r.testingT.Logf(s, a...)
}

func (r run) Fatalf(s string, a ...any) {
	r.testingT.Fatalf(s, a...)
}

func NewGenerateRun() Run {
	return &FakeRun{
		// TODO
	}
}

func (r run) Errorf(format string, args ...interface{}) {
	r.testingT.Errorf(format, args...)
}

func (r run) FailNow() {
	r.testingT.FailNow()
}

func (f FakeRun) Errorf(format string, args ...interface{}) {
	// TODO
}

func (f FakeRun) FailNow() {
	// TODO
}

func (f FakeRun) Helper() {
}

func (f FakeRun) Logf(s string, a ...any) {
	// TODO
}

func (f FakeRun) Fatalf(s string, a ...any) {
	// TODO
}

func (f FakeRun) TestingT() *testing.T {
	return nil
}
