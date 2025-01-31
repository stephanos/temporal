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

package telemetry_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/telemetry"
)

type testStruct struct {
	boolField      bool
	strField       string
	intField       int
	timeField      time.Time
	sliceField     []testStruct
	funcField      func()
	structField    *testStruct
	protoField     *workflowservice.DeleteWorkflowExecutionRequest
	mapStructField map[int64]*testStruct
	mapProtoField  map[int64]*workflowservice.DeleteWorkflowExecutionRequest
}

func TestDebugSnapshot(t *testing.T) {
	var selfRef = &testStruct{}
	selfRef.structField = selfRef

	tests := []struct {
		name   string
		input  any
		output map[string]any
	}{
		{
			name:   "nil",
			input:  nil,
			output: map[string]any{},
		},
		// TODO
		//{
		//	name:   "string",
		//	input:  "test",
		//	output: "test",
		//},
		{
			name:   "to empty struct",
			input:  &testStruct{},
			output: map[string]any{},
		},
		{
			name: "bool field",
			input: &testStruct{
				boolField: true,
			},
			output: map[string]any{
				"boolField": true,
			},
		},
		{
			name: "string field",
			input: &testStruct{
				strField: "test",
			},
			output: map[string]any{
				"strField": "test",
			},
		},
		{
			name: "int field",
			input: &testStruct{
				intField: 42,
			},
			output: map[string]any{
				"intField": 42.0,
			},
		},
		{
			name: "time field",
			input: &testStruct{
				timeField: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			output: map[string]any{
				"timeField": "2021-01-01T00:00:00Z",
			},
		},
		{
			name:   "struct self-reference",
			input:  selfRef,
			output: map[string]any{}, // ignore!
		},
		{
			name: "slice field",
			input: &testStruct{
				sliceField: []testStruct{
					{intField: 42},
				},
			},
			output: map[string]any{
				"sliceField": []any{
					map[string]any{
						"intField": 42.0,
					},
				},
			},
		},
		{
			name: "func field",
			input: &testStruct{
				funcField: func() {},
			},
			output: map[string]any{}, // ignored!
		},
		{
			name: "proto field",
			input: &testStruct{
				protoField: &workflowservice.DeleteWorkflowExecutionRequest{
					Namespace: "test",
				},
			},
			output: map[string]any{
				"protoField": map[string]any{
					"namespace": "test",
				},
			},
		},
		{
			name: "map struct field",
			input: &testStruct{
				mapStructField: map[int64]*testStruct{
					1: {boolField: true},
				},
			},
			output: map[string]any{
				"mapStructField": map[string]any{
					"1": map[string]any{
						"boolField": true,
					},
				},
			},
		},
		{
			name: "map proto field",
			input: &testStruct{
				mapProtoField: map[int64]*workflowservice.DeleteWorkflowExecutionRequest{
					1: {Namespace: "test"},
				},
			},
			output: map[string]any{
				"mapProtoField": map[string]any{
					"1": map[string]any{
						"namespace": "test",
					},
				},
			},
		},
	}

	os.Setenv(telemetry.DebugModeEnvVar, "true")
	defer os.Unsetenv(telemetry.DebugModeEnvVar)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			exporter := tracetest.NewInMemoryExporter()
			processor := trace.NewSimpleSpanProcessor(exporter)
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(processor))
			tracer := tp.Tracer("test-tracer")

			_, span := tracer.Start(context.Background(), "test-span")
			telemetry.DebugSnapshot(span, "test", tt.input)
			span.End()

			exportedSpans := exporter.GetSpans()
			exportedSpan := exportedSpans[0]
			exportedEvent := exportedSpan.Events[0]
			exportedAttr := exportedEvent.Attributes[0]
			snapshot := exportedAttr.Value.AsString()

			t.Log("JSON:", snapshot)

			var result map[string]any
			err := json.Unmarshal([]byte(snapshot), &result)
			require.NoError(t, err)
			require.Equal(t, tt.output, result)
		})
	}
}
