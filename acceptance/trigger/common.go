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

package trigger

import (
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/server/common/testing/stamp"
)

type Payloads struct{}

// TODO: make random
func (g Payloads) Get(s stamp.Seeder) *commonpb.Payloads {
	return &commonpb.Payloads{
		Payloads: []*commonpb.Payload{Payload{}.Get(s)},
	}
}

type Payload struct{}

// TODO: make random
func (g Payload) Get(s stamp.Seeder) *commonpb.Payload {
	return &commonpb.Payload{
		Metadata: map[string][]byte{
			"meta_data_key": []byte(`meta_data_val`),
		},
		Data: []byte(`test data`),
	}
}

type Header struct{}

// TODO: make random
func (g Header) Get(s stamp.Seeder) *commonpb.Header {
	return &commonpb.Header{
		Fields: map[string]*commonpb.Payload{
			"header_field": Payload{}.Get(s),
		},
	}
}
