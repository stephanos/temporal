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
	"sync"
)

type (
	modelRecorder struct {
		records []any
		lock    sync.RWMutex
	}
	OperationID      string
	OperationContext interface {
		OperationID() string
	}
)

func newRecorder(records ...any) modelRecorder {
	return modelRecorder{records: records}
}

func (l *modelRecorder) Add(records ...any) *modelRecorder {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.records = append(l.records, records...)
	return l
}

func (l *modelRecorder) Get() []any {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.records
}

func (o OperationID) OperationID() string {
	return string(o)
}

// TODO: emit as YAML instead! more compact and prob no overrides in proto for it
//func (l modelRecorder) Format() []string {
//	var res []string
//	for _, entry := range l.records {
//		mypp := pp.New()
//		mypp.SetColoringEnabled(false)
//		mypp.SetExportedOnly(true)
//		mypp.SetOmitEmpty(true)
//		res = append(res, fmt.Sprintf("```\n%v\n```", mypp.Sprint(entry)))
//	}
//	return res
//}
