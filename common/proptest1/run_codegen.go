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
	"os"
	"text/template"
)

var (
	tmpl = template.Must(template.New("").Parse(`package {{.outputPkg}}
	
import (
	"testing"

	. "go.temporal.io/server/proptests/internal"
	. "go.temporal.io/server/proptests/model"
)

{{ range .testCases }}
func Test{{$.specName}}_{{.Name}}(t *testing.T) {
	t.Parallel()
	// TODO
}
{{ end }}
`,
	))
)

type (
	codegenRun struct {
		*run
		outputPath string
		outputPkg  string
	}
	testCase struct {
		Name string
	}
)

func NewCodegenRun(outputPath, outputPkg string) *codegenRun {
	return &codegenRun{
		run: &run{
			logger:     &noopLogger{},
			modelTypes: make(map[string]ModelType[any]),
			models:     make(map[string]map[string]*model),
		},
		outputPath: outputPath,
		outputPkg:  outputPkg,
	}
}

func (r *codegenRun) WaitForState(m modelType, desiredStates ...*State) {
	// nothing to do
}

func (r *codegenRun) registerModel(m modelType) {
	// TODO
}

func (r *codegenRun) Finish(specName string) {
	w, err := os.OpenFile(r.outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	var testCases []testCase
	testCases = append(testCases, testCase{
		Name: "01",
	})

	err = tmpl.Execute(w, map[string]any{
		"outputPkg": r.outputPkg,
		"specName":  specName,
		"testCases": testCases,
	})
	if err != nil {
		panic(err)
	}
}
