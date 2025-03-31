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
)

type (
	logger interface {
		Debug(string)
		Info(string)
		Fatal(string)
	}
	testLogger struct {
		t testing.TB
	}
	noopLogger struct{}
)

func newTestLogger(t testing.TB) *testLogger {
	return &testLogger{t: t}
}

func (l *testLogger) Debug(msg string) {
	l.t.Helper()
	//l.t.Log(msg) TODO?
}

func (l *testLogger) Info(msg string) {
	l.t.Helper()
	l.t.Log(msg)
}

func (l *testLogger) Fatal(msg string) {
	l.t.Helper()
	msg = "💥 " + redStr(msg)
	l.t.Log(msg)
	panic(msg) // `Fatalf` doesn't seem to stop the test execution
}

func (n *noopLogger) Debug(msg string) {
	// noop
}

func (n *noopLogger) Info(msg string) {
	// noop
}

func (n *noopLogger) Error(msg string) {
	// noop
}

func (n *noopLogger) Fatal(msg string) {
	// noop
}
