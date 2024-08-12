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

//go:build with_assertions

package assert

import (
	"fmt"
)

var WithAssertions = true

// Should asserts a condition is true, or panics if not.
// In development/testing, it will panic. In production, it will not do anything.
// See package documentation for more details.
func Should(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

func Must(condition bool, logger *log.Logger, msg string) {
	if !WithAssertions {
		return
	}
	if !condition {
		message := fmt.Sprintf(format, args...)
		if logger == nil {
			logger = DefaultLogger
		}
		logger.Printf("Assertion failed: %s", message)
		// Optionally, you could also call panic(message) here if you want to halt execution.
	}
}
