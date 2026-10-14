package explore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/encoding/protojson"
)

type traceSource struct {
	Name, File string
	Line       int32
}
type traceStep struct {
	Action, Before, After, Outcome, Product string
	Facts                                   []interp.Atom
}
type traceView struct {
	Query, Digest, Identity                               string
	Steps                                                 []traceStep
	Sources                                               []traceSource
	Monitors, Events, Faults, Evidence, Holes, Assessment string
}

// RenderTrace keeps recorded events whole beside the model's expectation; a witness is never
// presented as proof that hidden system steps occurred. sourceRoot resolves links in a local file.
func RenderTrace(c *Candidate, query string, run *testpilotspb.Run, assessment *testpilot.Assessment, sourceRoot ...string) ([]byte, error) {
	if c == nil || c.Case == nil {
		return nil, errors.New("a lowered candidate is required")
	}
	if run != nil && (run.GetCaseId() != c.Case.GetCaseId() || run.GetProgramId() != c.Case.GetProgram().GetProgramId()) {
		return nil, errors.New("trace Run belongs to another Case")
	}
	realizer, err := check.NewRealizer(c.Model, check.DefaultScope)
	if err != nil {
		return nil, err
	}
	var receipt *check.Receipt
	for _, r := range check.Check(c.Model, check.DefaultScope).Receipts {
		if r.Subject == check.QuerySubject && r.Key.Name == query {
			receipt = &r
			break
		}
	}
	if receipt == nil || receipt.Witness == nil {
		return nil, fmt.Errorf("query %s has no witness", query)
	}
	machine := realizer.Machine(receipt.Key.Owner)
	view := traceView{Query: query, Digest: c.Digest, Identity: c.Identity}
	if err := view.witness(c.Model, machine, receipt); err != nil {
		return nil, err
	}
	add := func(name string, at *umpirespb.Position) {
		if at == nil {
			return
		}
		file := at.GetFile()
		if len(sourceRoot) > 0 {
			file = filepath.Join(sourceRoot[0], file)
		}
		view.Sources = append(view.Sources, traceSource{Name: name, File: file, Line: at.GetLine()})
	}
	add("machine "+machine.Decl.GetName(), machine.Decl.GetPosition())
	declared, err := realizer.Declared(receipt.Key)
	if err != nil {
		return nil, err
	}
	add("Query "+query, declared.Query.GetPosition())
	add("Property "+declared.Property.GetName(), declared.Property.GetPosition())
	for _, a := range c.Model.GetActions() {
		add("action "+a.GetName(), a.GetPosition())
	}
	for _, r := range realizer.Realizations() {
		if r.GetMachine() == machine.Decl.GetName() {
			add("realization "+r.GetName(), r.GetPosition())
		}
	}
	holes, err := json.MarshalIndent(struct {
		Reached   any `json:"reached"`
		Declared  any `json:"declared"`
		KnownGaps any `json:"knownGaps"`
	}{receipt.Holes, c.Model.GetHoles(), c.Case.GetProvenance().GetKnownGaps()}, "", "  ")
	if err != nil {
		return nil, err
	}
	view.Holes = string(holes)
	monitors, err := json.MarshalIndent(receipt.Monitors, "", "  ")
	if err != nil {
		return nil, err
	}
	view.Monitors = string(monitors)
	if err := view.record(run, assessment); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = traceTemplate.Execute(&out, view)
	return out.Bytes(), err
}

func (view *traceView) witness(model *umpirespb.Model, machine *interp.Machine, receipt *check.Receipt) error {
	before := receipt.Witness.Initial.Value
	interpreter := interp.NewInterpreter(model)
	for _, step := range receipt.Witness.Steps {
		row := traceStep{Action: step.Action.Value, Before: before, After: step.State.Value, Outcome: step.Outcome.Value, Facts: step.Facts}
		if ref := machine.Decl.GetRefines(); ref != nil {
			state, ok := machine.State(row.After)
			if !ok {
				return errors.New("witness state missing")
			}
			projected, err := interpreter.Call(ref.GetMap(), []interp.Value{state}, machine.Decl.GetPosition())
			if err != nil {
				return err
			}
			row.Product = ref.GetProduct() + ": " + projected.Key()
		}
		view.Steps = append(view.Steps, row)
		before = row.After
	}
	return nil
}

func (view *traceView) record(run *testpilotspb.Run, assessment *testpilot.Assessment) error {
	events := []json.RawMessage{}
	faults := []json.RawMessage{}
	evidence := []json.RawMessage{}
	for _, event := range run.GetEvents() {
		data, err := protojson.Marshal(event)
		if err != nil {
			return err
		}
		events = append(events, data)
		if event.GetKind() == testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED {
			faults = append(faults, data)
		}
		if len(event.GetObservations()) > 0 {
			evidence = append(evidence, data)
		}
	}
	fields := []struct {
		value any
		into  *string
	}{{events, &view.Events}, {faults, &view.Faults}, {evidence, &view.Evidence}, {assessment, &view.Assessment}}
	for _, field := range fields {
		data, err := json.MarshalIndent(field.value, "", "  ")
		if err != nil {
			return err
		}
		*field.into = string(data)
	}
	if run != nil {
		data, err := protojson.MarshalOptions{Indent: "  "}.Marshal(run.GetVerdict())
		if err != nil {
			return err
		}
		view.Monitors += "\nRecorded Contract monitor:\n" + string(data)
	}
	return nil
}

var traceTemplate = template.Must(template.New("trace").Parse(`<!doctype html>
<html lang="en"><meta charset="utf-8"><title>Umpire trace — {{.Query}}</title>
<style>body{font:16px system-ui;max-width:1100px;margin:2rem auto;padding:0 1rem}table{border-collapse:collapse;width:100%}td,th{padding:.5rem;border:1px solid #aaa;text-align:left}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f5f5f5;padding:1rem}details{margin:1rem 0}</style>
<h1>{{.Query}}</h1><p>Model candidate <code>{{.Digest}}</code><br>Case <code>{{.Identity}}</code></p>
<h2>Product expectation</h2><p>The checked bounded witness predicts these steps. Recorded evidence below determines what the runtime supports.</p>
<table><tr><th>Action</th><th>Before → after</th><th>Outcome / facts</th><th>Abstract product projection</th></tr>{{range .Steps}}<tr><td>{{.Action}}</td><td>{{.Before}} → {{.After}}</td><td>{{.Outcome}} {{.Facts}}</td><td>{{.Product}}</td></tr>{{end}}</table>
<h2>System execution</h2><p>Recorded Testpilot events, including instruction coordinates and causal parents. Hidden server commitments are known only where evidence declares them.</p><pre>{{.Events}}</pre>
<h2>Monitor state</h2><pre>{{.Monitors}}</pre><h2>Assessment</h2><pre>{{.Assessment}}</pre>
<h2>Fault decisions</h2><pre>{{.Faults}}</pre><h2>Evidence</h2><pre>{{.Evidence}}</pre><h2>Holes and known gaps</h2><pre>{{.Holes}}</pre>
<h2>Source definitions</h2><ul>{{range .Sources}}<li><a href="{{.File}}#L{{.Line}}">{{.Name}} — {{.File}}:{{.Line}}</a></li>{{end}}</ul></html>
`))
