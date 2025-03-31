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

package proptest1

import (
	"fmt"
	"testing"
)

type (
	Logger interface {
		Debugf(string, ...any)
		Infof(string, ...any)
		Errorf(string, ...any)
		Fatalf(string, ...any)
	}
	logger struct {
		t testing.TB
	}
	noopLogger struct{}
)

func newLogger(t testing.TB) *logger {
	return &logger{t: t}
}

func (l *logger) Debug(msg string, args ...any) {
	l.t.Logf("%v", fmt.Sprintf(msg, args...))
}

func (l *logger) Info(msg string, args ...any) {
	l.t.Logf("%v", fmt.Sprintf(msg, args...))
}

func (l *logger) Error(msg string, args ...any) {
	l.t.Errorf("🔴 %v", fmt.Sprintf(msg, args...))
}

func (l *logger) Fatal(msg string, args ...any) {
	l.t.Fatalf("💥 %v", fmt.Sprintf(msg, args...))
}

func (n *noopLogger) Debug(msg string, args ...any) {
	// noop
}

func (n *noopLogger) Info(msg string, args ...any) {
	// noop
}

func (n *noopLogger) Error(msg string, args ...any) {
	// noop
}

func (n *noopLogger) Fatal(msg string, args ...any) {
	// noop
}
