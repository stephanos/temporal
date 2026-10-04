package golden

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// The original baseline is the Scala Model's output frozen before fn-112 changed how the Model is
// written: every IR, every positive lifter fixture, the lifter's refusals and every checked-in Case.
// Every later change of the author surface is compared with it, never with the change before.
//
// What may differ from it is the closed delta of original.json (Delta): inert fields, entity
// attachments and, since fn-122, the claims the lifter generates from the laws a Model's
// capabilities bring. A law replacement names one generated claim of one IR file; the expected Model
// is the baseline with each replacement's retired Queries removed and its renamed Property under the
// generated name, every Query that read it reading it so; the current Model is compared without the
// listed generated claims (Ungenerated). What is compared after that is compared exactly: tables,
// answers, receipts, Definition IDs, canonical forms, fingerprints, refined Properties and Cases are
// all derived again from the transformed baseline, never waived.

//go:embed original.json
var originalBytes []byte

// OriginalDir is the archive, relative to the repository root. Its .gz files are the frozen inputs;
// the JSON files beside them are what the Go tooling derived from those inputs when they were frozen.
const OriginalDir = "tools/umpire/internal/golden/testdata/original"

// Archived inputs, by archive key.
const (
	OriginalIR      = "ir/"
	OriginalLifts   = "lifts/"
	OriginalRejects = "lifts/rejects.txt"
	OriginalCases   = "cases/"
)

// The current trees the archive freezes, by archive key prefix.
var originalTrees = map[string]string{
	OriginalIR:    "model/ir",
	OriginalLifts: "model/lifter/testdata/lifts/expected",
	OriginalCases: "model/cases",
}

// Delta is the closed list of differences from the original baseline that R1 of fn-112 permits
// beyond source positions and Functions: nothing else may differ. original.json spells it, and
// refuses a field it does not name.
type Delta struct {
	// InertFields are IR fields, by full protobuf name, that the baseline never sets and that carry
	// metadata no table, ID, fingerprint, answer or Case reads: Query.total, named-choice names. A
	// realization's required settings are listed too: no table, ID, fingerprint or answer reads them,
	// and the Cases that carry them are compared on their own as new Cases.
	InertFields []string `json:"inert_fields"`
	// Attachments are the entity attachments of fn-112's R20 task-queue entity. Each sets one field the
	// baseline left empty, so whatever reads it is derived again from the baseline plus the attachment.
	Attachments []Attachment `json:"entity_attachments"`
	// Replacements are the claims the lifter generates from laws (fn-122), each with what it replaces
	// in the baseline: "law_replacements": [{"model": "ir/activity.json", "machine": "activityProduct",
	// "law": "terminalStatesAreFinal", "verdict": "verified-within-limits", "renames":
	// "terminalIsFinal", "retires": ["terminalHolds"]}]. Every generated claim of an archived IR file
	// is listed, so the list is closed: one left out stays in the compared Model and fails.
	Replacements []Replacement `json:"law_replacements"`
	// NewCases are the Cases a generated find Query lowers to, which the baseline has no file for:
	// "new_cases": [{"model": "ir/activity.json", "query": "activityProtocol.terminateSettles"}]. The
	// Case file is the one `make umpire-gen-cases` names, `<model without .json>-<query>-case.json`.
	NewCases []NewCase `json:"new_cases"`
	// NewIRFiles are IR files, by archive key, that the baseline has no Model for: "new_ir_files":
	// ["ir/nexus-operation.json"]. Each must be produced, and nothing compares it with the baseline.
	NewIRFiles []string `json:"new_ir_files"`
	// Reduced are the archived lifter fixtures, by archive key, that fn-114.10 replaced:
	// "reduced_fixtures": ["lifts/admission.json"]. They copied live Model text; each is reduced to a
	// fixture-local Model, a new Model rather than a converted one, or retired outright, and neither
	// harness compares it with its frozen original (Compared, ComparedOutputs, Config.Inputs). What its
	// copied text exercised is covered by the reduced fixture's own lifter test or by the gate's
	// live-Model lift, mapped in .flow/tmp/fn114-10/. A reduced fixture is still produced, lifted by the
	// lifter's own tests against its expected file; a retired one is not, so the inventory does not
	// require it. Each stays archived, and the archive stays frozen.
	Reduced []string `json:"reduced_fixtures"`
}

// Attachment attaches the machine of a name, or the action of an ID, to an entity.
type Attachment struct {
	Machine string `json:"machine,omitempty"`
	Action  string `json:"action,omitempty"`
	// Field is the attachment: "entity" of a machine, "on" or "creates" of an action.
	Field  string `json:"field"`
	Entity string `json:"entity"`
}

// Replacement is one claim the lifter generates from a law in one IR file: the Property, the Scenario
// and the Query it names `<machine>.<law>` (Generated).
type Replacement struct {
	// Model is the IR file, by archive key: "ir/activity.json".
	Model   string `json:"model"`
	Machine string `json:"machine"`
	Law     string `json:"law"`
	// Verdict is the receipt kind the checker gives the generated Query, a model.ReceiptKind such as
	// "verified-within-limits", "counterexample" or "found".
	Verdict string `json:"verdict"`
	// Renames is the baseline Property of the machine the generated Property is, under its old name;
	// every Query that read it reads the generated one. Empty when the generated Property is new.
	Renames string `json:"renames,omitempty"`
	// Retires are the baseline Queries the generated Query takes the place of, which are gone.
	Retires []string `json:"retires,omitempty"`
}

// Generated is the name of the Property, Scenario and Query the replacement generates.
func (r Replacement) Generated() string { return r.Machine + "." + r.Law }

// NewCase is the Case a generated Query of an IR file lowers to.
type NewCase struct {
	Model string `json:"model"`
	Query string `json:"query"`
}

// File is the Case's key, as `make umpire-gen-cases` names its file (tools/umpire/lower/generated.go).
func (c NewCase) File() string {
	return OriginalCases + strings.TrimSuffix(strings.TrimPrefix(c.Model, OriginalIR), ".json") + "-" + c.Query + "-case.json"
}

// ReceiptKinds are the verdicts a replacement may record: tools/umpire/model.ReceiptKind's values,
// which this test-only package does not import (TestOriginalVerdictsAreReceiptKinds holds them equal).
var ReceiptKinds = []string{
	"admission-error", "declaration-error", "resource-limit", "limit-reached", "unresolved",
	"refinement-rejected", "counterexample", "verified-within-limits", "found", "not-found",
	"incomplete", "unsupported", "replay-failed",
}

func OriginalDelta() (Delta, error) {
	var d Delta
	decoder := json.NewDecoder(bytes.NewReader(originalBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		return d, err
	}
	return d, d.check()
}

// irKey reports whether key is an IR file's archive key: under ir/, a .json file and no law sidecar.
func irKey(key string) bool {
	name, ok := strings.CutPrefix(key, OriginalIR)
	return ok && name != ".json" && strings.HasSuffix(name, ".json") && !strings.Contains(name, "/") && !IsLawSidecar(name)
}

// liftKey reports whether key is a positive lifter fixture's archive key: a .json file under lifts/,
// so never the refusals.
func liftKey(key string) bool {
	name, ok := strings.CutPrefix(key, OriginalLifts)
	return ok && name != ".json" && strings.HasSuffix(name, ".json") && !strings.Contains(name, "/")
}

func (d Delta) check() error {
	for _, name := range d.InertFields {
		if _, err := inertField(name); err != nil {
			return err
		}
	}
	for _, a := range d.Attachments {
		if !a.valid() {
			return fmt.Errorf("entity attachment %+v is not one machine's entity or one action's on or creates", a)
		}
	}
	newFiles := map[string]bool{}
	for _, key := range d.NewIRFiles {
		if !irKey(key) || newFiles[key] {
			return fmt.Errorf("new IR file %q is not one IR file under %s", key, OriginalIR)
		}
		newFiles[key] = true
	}
	generated, err := d.checkReplacements(newFiles)
	if err != nil {
		return err
	}
	cases := map[NewCase]bool{}
	for _, c := range d.NewCases {
		// A new Case is a generated Query's, or any named Query's of an IR file the baseline lacks.
		lowered := generated[c] || newFiles[c.Model] && c.Query != ""
		if cases[c] || !lowered {
			return fmt.Errorf("new Case %+v is not one generated Query's, or one Query's of a new IR file", c)
		}
		cases[c] = true
	}
	reduced := map[string]bool{}
	for _, key := range d.Reduced {
		if !liftKey(key) || reduced[key] {
			return fmt.Errorf("reduced fixture %q is not one lifter fixture under %s", key, OriginalLifts)
		}
		reduced[key] = true
	}
	return nil
}

// valid reports whether the attachment sets exactly one field the baseline may leave empty: a
// machine's entity, or an action's on or creates.
func (a Attachment) valid() bool {
	return a.Entity != "" && (a.Machine == "") != (a.Action == "") &&
		(a.Machine == "" || a.Field == "entity") && (a.Action == "" || a.Field == "on" || a.Field == "creates")
}

// checkReplacements checks each law replacement and gives the claims they generate. A Query may be
// retired once and generated once, never both, so the expected Model is the same whatever order the
// replacements of a file are applied in.
func (d Delta) checkReplacements(newFiles map[string]bool) (map[NewCase]bool, error) {
	generated, retired := map[NewCase]bool{}, map[NewCase]bool{}
	for _, r := range d.Replacements {
		claim := NewCase{Model: r.Model, Query: r.Generated()}
		if err := r.check(newFiles); err != nil {
			return nil, err
		}
		if generated[claim] {
			return nil, fmt.Errorf("law replacement %+v is listed twice", r)
		}
		generated[claim] = true
		for _, query := range r.Retires {
			retirement := NewCase{Model: r.Model, Query: query}
			if query == "" || retired[retirement] {
				return nil, fmt.Errorf("law replacement %+v retires Query %q twice or by no name", r, query)
			}
			retired[retirement] = true
		}
	}
	for c := range retired {
		if generated[c] {
			return nil, fmt.Errorf("law replacements of %s retire and generate Query %s", c.Model, c.Query)
		}
	}
	return generated, nil
}

// check checks what one law replacement names on its own: an archived IR file, one machine's law, a
// receipt kind and a rename to a different name. That it is listed once is checkReplacements'.
func (r Replacement) check(newFiles map[string]bool) error {
	switch {
	case !irKey(r.Model) || newFiles[r.Model]:
		return fmt.Errorf("law replacement %+v names no archived IR file", r)
	case r.Machine == "" || strings.Contains(r.Machine, ".") || r.Law == "" || strings.Contains(r.Law, "."):
		return fmt.Errorf("law replacement %+v is not one machine's law", r)
	case !slices.Contains(ReceiptKinds, r.Verdict):
		return fmt.Errorf("law replacement %+v records no receipt kind", r)
	case r.Renames == r.Generated():
		return fmt.Errorf("law replacement %+v renames the Property to its own name", r)
	default:
		return nil
	}
}

// replacements are the law replacements of the IR file of a key.
func (d Delta) replacements(key string) []int {
	var out []int
	for i, r := range d.Replacements {
		if r.Model == key {
			out = append(out, i)
		}
	}
	return out
}

// Compared gives files by archive key without the reduced fixtures, which nothing compares with
// their baselines. Inventory still reads every file.
func (d Delta) Compared(files map[string][]byte) map[string][]byte {
	out := maps.Clone(files)
	for _, key := range d.Reduced {
		delete(out, key)
	}
	return out
}

// ComparedOutputs gives derived outputs, keyed `<output>/<archive key>`, without the reduced
// fixtures' outputs.
func (d Delta) ComparedOutputs(outputs Derived) Derived {
	out := maps.Clone(outputs)
	for key := range outputs {
		if _, model, _ := strings.Cut(key, "/"); slices.Contains(d.Reduced, model) {
			delete(out, key)
		}
	}
	return out
}

// unproduced reports each archived file no longer produced, which only a reduced fixture may be, and
// each reduced fixture that is not archived.
func (d Delta) unproduced(archived, current map[string][]byte) []error {
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(archived)) {
		if _, ok := current[key]; !ok && !slices.Contains(d.Reduced, key) {
			errs = append(errs, fmt.Errorf("%s is archived and no longer produced", key))
		}
	}
	for _, key := range d.Reduced {
		if _, ok := archived[key]; !ok {
			errs = append(errs, fmt.Errorf("reduced fixture %s is not archived", key))
		}
	}
	return errs
}

// Rederives reports whether the delta changes a baseline Model, so whatever the archive derived from
// it is derived again from the expected Model.
func (d Delta) Rederives() bool { return len(d.Attachments) > 0 || len(d.Replacements) > 0 }

func inertField(name string) (protoreflect.FieldDescriptor, error) {
	i := strings.LastIndex(name, ".")
	if i < 0 {
		return nil, fmt.Errorf("inert field %q is not a full protobuf field name", name)
	}
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name[:i]))
	if err != nil {
		return nil, fmt.Errorf("inert field %q: %w", name, err)
	}
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok || message.ParentFile().Package() != (&umpirespb.Model{}).ProtoReflect().Descriptor().ParentFile().Package() {
		return nil, fmt.Errorf("inert field %q is not a field of the Umpire IR", name)
	}
	field := message.Fields().ByName(protoreflect.Name(name[i+1:]))
	if field == nil {
		return nil, fmt.Errorf("inert field %q: %s has no such field", name, message.FullName())
	}
	return field, nil
}

// Applied records which entries of the delta found their declarations in some baseline Model.
type Applied map[string]bool

// Expected is the baseline Model of a key with the attachments and the key's law replacements
// applied: what the current Model, without its generated claims (Ungenerated), must equal. An
// attachment must find its declaration in some baseline Model, so applied records each it applied. A
// replacement finds its renamed Property and its retired Queries in its own file, or fails.
func (d Delta) Expected(key string, baseline *umpirespb.Model, applied Applied) (*umpirespb.Model, error) {
	m := proto.CloneOf(baseline)
	for i, a := range d.Attachments {
		attached, err := a.attach(m)
		if err != nil {
			return nil, err
		}
		if attached {
			applied[fmt.Sprint("attachment ", i)] = true
		}
	}
	replacements := d.replacements(key)
	for _, i := range replacements {
		if err := d.Replacements[i].replace(m); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		applied[fmt.Sprint("replacement ", i)] = true
	}
	if len(replacements) == 0 {
		return m, nil
	}
	// Lift sorts Properties by machine and name, so a renamed one takes its new place in that order.
	slices.SortStableFunc(m.Properties, func(a, b *umpirespb.Property) int {
		return cmp.Or(strings.Compare(a.GetMachine(), b.GetMachine()), strings.Compare(a.GetName(), b.GetName()))
	})
	return m, readsDeclaredProperties(m)
}

// attach sets the attachment's entity on each declaration of m it names, and reports whether m had
// one. A field the baseline already sets is no attachment but a change, so it fails.
func (a Attachment) attach(m *umpirespb.Model) (bool, error) {
	attached := false
	for _, machine := range m.GetMachines() {
		if a.Machine != "" && machine.GetName() == a.Machine {
			if machine.GetEntity() != "" {
				return false, fmt.Errorf("machine %s already has entity %s", a.Machine, machine.GetEntity())
			}
			machine.Entity = a.Entity
			attached = true
		}
	}
	for _, action := range m.GetActions() {
		if a.Action == "" || action.GetId() != a.Action {
			continue
		}
		target := &action.On
		if a.Field == "creates" {
			target = &action.Creates
		}
		if *target != "" {
			return false, fmt.Errorf("action %s already has %s %s", a.Action, a.Field, *target)
		}
		*target = a.Entity
		attached = true
	}
	return attached, nil
}

// replace renames the replacement's Property of m to the generated name, with every Query that reads
// it, and removes each Query it retires.
func (r Replacement) replace(m *umpirespb.Model) error {
	if r.Renames != "" {
		var renamed []*umpirespb.Property
		for _, p := range m.GetProperties() {
			if p.GetMachine() == r.Machine && p.GetName() == r.Generated() {
				return fmt.Errorf("law replacement %s renames a Property to the name the baseline gives another", r.Generated())
			}
			if p.GetMachine() == r.Machine && p.GetName() == r.Renames {
				renamed = append(renamed, p)
			}
		}
		if len(renamed) != 1 {
			return fmt.Errorf("law replacement %s renames Property %s of %s, which the baseline declares %d times", r.Generated(), r.Renames, r.Machine, len(renamed))
		}
		renamed[0].Name = r.Generated()
		for _, q := range m.GetQueries() {
			if q.GetProperty().GetMachine() == r.Machine && q.GetProperty().GetName() == r.Renames {
				q.Property.Name = r.Generated()
			}
		}
	}
	for _, retired := range r.Retires {
		n := len(m.GetQueries())
		m.Queries = slices.DeleteFunc(m.Queries, func(q *umpirespb.Query) bool { return q.GetName() == retired })
		if n-len(m.GetQueries()) != 1 {
			return fmt.Errorf("law replacement %s retires Query %s, which the baseline declares %d times", r.Generated(), retired, n-len(m.GetQueries()))
		}
	}
	return nil
}

// readsDeclaredProperties checks that every Query of m reads a Property m declares: a retired or
// renamed Property no Query is left reading.
func readsDeclaredProperties(m *umpirespb.Model) error {
	declared := map[[2]string]bool{}
	for _, p := range m.GetProperties() {
		declared[[2]string{p.GetMachine(), p.GetName()}] = true
	}
	for _, q := range m.GetQueries() {
		if !declared[[2]string{q.GetProperty().GetMachine(), q.GetProperty().GetName()}] {
			return fmt.Errorf("the expected Model does not declare Property %[2]s of %[3]s, which Query %[1]s reads", q.GetName(), q.GetProperty().GetName(), q.GetProperty().GetMachine())
		}
	}
	return nil
}

// Unapplied names the attachments no baseline Model had a declaration for, and the law replacements
// whose IR file no baseline Model was.
func (d Delta) Unapplied(applied Applied) error {
	var errs []error
	for i, a := range d.Attachments {
		if !applied[fmt.Sprint("attachment ", i)] {
			errs = append(errs, fmt.Errorf("entity attachment %+v names no declaration of the baseline", a))
		}
	}
	for i, r := range d.Replacements {
		if !applied[fmt.Sprint("replacement ", i)] {
			errs = append(errs, fmt.Errorf("law replacement %+v names no baseline Model", r))
		}
	}
	return errors.Join(errs...)
}

// Ungenerated is a current Model of a key without the generated claims its law replacements list:
// each generated Query and its Scenario, and the generated Property where it renames none, with the
// Function it holds by and every Function named under that. Each must be there, read as the lifter
// generates it: the Query reads the generated Property and Scenario. A generated claim the delta does
// not list stays, so the comparison with the baseline fails on it.
func (d Delta) Ungenerated(key string, current *umpirespb.Model) (*umpirespb.Model, error) {
	replacements := d.replacements(key)
	if len(replacements) == 0 {
		return current, nil
	}
	m := proto.CloneOf(current)
	for _, i := range replacements {
		if err := d.Replacements[i].ungenerate(m); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
	}
	return m, nil
}

// ungenerate removes the replacement's generated claim from m: the Query and its Scenario, and the
// Property with the Functions it holds by where it renames none. A renamed Property stays, since the
// expected Model declares it under the generated name. Each must be there as the lifter generates
// it, so a claim the lifter stopped generating fails rather than passing unnoticed.
func (r Replacement) ungenerate(m *umpirespb.Model) error {
	name := r.Generated()
	claim := func(ref *umpirespb.ClaimRef) bool { return ref.GetMachine() == r.Machine && ref.GetName() == name }
	var queries []*umpirespb.Query
	for _, q := range m.GetQueries() {
		if q.GetName() == name {
			queries = append(queries, q)
		}
	}
	if len(queries) != 1 || !claim(queries[0].GetProperty()) || !claim(queries[0].GetScenario()) {
		return fmt.Errorf("generated Query %s is not declared once, reading Property and Scenario %s of %s", name, name, r.Machine)
	}
	m.Queries = slices.DeleteFunc(m.Queries, func(q *umpirespb.Query) bool { return q.GetName() == name })
	n := len(m.GetScenarios())
	m.Scenarios = slices.DeleteFunc(m.Scenarios, func(s *umpirespb.Scenario) bool { return s.GetMachine() == r.Machine && s.GetName() == name })
	if n-len(m.GetScenarios()) != 1 {
		return fmt.Errorf("generated Scenario %s of %s is declared %d times", name, r.Machine, n-len(m.GetScenarios()))
	}
	holds := ""
	n = len(m.GetProperties())
	m.Properties = slices.DeleteFunc(m.Properties, func(p *umpirespb.Property) bool {
		generated := p.GetMachine() == r.Machine && p.GetName() == name
		if generated {
			holds = p.GetHolds()
		}
		return generated && r.Renames == ""
	})
	if holds == "" {
		return fmt.Errorf("generated Property %s of %s is not declared", name, r.Machine)
	}
	if r.Renames == "" {
		if n-len(m.GetProperties()) != 1 {
			return fmt.Errorf("generated Property %s of %s is declared %d times", name, r.Machine, n-len(m.GetProperties()))
		}
		m.Functions = slices.DeleteFunc(m.Functions, func(f *umpirespb.Function) bool {
			return f.GetName() == holds || strings.HasPrefix(f.GetName(), holds+".")
		})
	}
	return nil
}

// ProjectBaseline gives an expected Model as the comparison reads it: without source positions or
// Functions.
func (d Delta) ProjectBaseline(expected *umpirespb.Model) (*umpirespb.Model, error) {
	m := proto.CloneOf(expected)
	for _, name := range d.InertFields {
		field, err := inertField(name)
		if err != nil {
			return nil, err
		}
		if err := messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
			if child.Descriptor() == field.ContainingMessage() && child.Has(field) {
				return false, fmt.Errorf("inert field %s is set in the baseline", name)
			}
			return true, nil
		}); err != nil {
			return nil, err
		}
	}
	return m, sourceless(m)
}

// ProjectCurrent gives a current Model as the comparison reads it: without source positions,
// Functions or the inert fields.
func (d Delta) ProjectCurrent(current *umpirespb.Model) (*umpirespb.Model, error) {
	m := proto.CloneOf(current)
	for _, name := range d.InertFields {
		field, err := inertField(name)
		if err != nil {
			return nil, err
		}
		if err := messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
			if child.Descriptor() == field.ContainingMessage() {
				child.Clear(field)
			}
			return true, nil
		}); err != nil {
			return nil, err
		}
	}
	return m, sourceless(m)
}

// functionReference is what every Function reference that is set reads as.
const functionReference = "<function>"

// functionReferences are the IR fields that name a Function of the Model.
var functionReferences = map[protoreflect.FullName]bool{
	"temporal.server.api.umpire.v1.Call.function":               true,
	"temporal.server.api.umpire.v1.StepBinding.function":        true,
	"temporal.server.api.umpire.v1.Machine.evidence":            true,
	"temporal.server.api.umpire.v1.Refinement.map":              true,
	"temporal.server.api.umpire.v1.Refinement.visible":          true,
	"temporal.server.api.umpire.v1.Refinement.visible_outcomes": true,
	"temporal.server.api.umpire.v1.Monitor.next":                true,
	"temporal.server.api.umpire.v1.Monitor.violated":            true,
	"temporal.server.api.umpire.v1.Monitor.after":               true,
	"temporal.server.api.umpire.v1.Property.holds":              true,
	"temporal.server.api.umpire.v1.Progress.from":               true,
	"temporal.server.api.umpire.v1.Progress.to":                 true,
}

// sourceless removes where m was lifted from and every position, and what functionless removes.
func sourceless(m *umpirespb.Model) error {
	m.Source = ""
	if err := functionless(m); err != nil {
		return err
	}
	position := (&umpirespb.Position{}).ProtoReflect().Descriptor()
	return messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
		child.Range(func(f protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			if f.Message() == position {
				child.Clear(f)
			}
			return true
		})
		return true, nil
	})
}

// functionless removes every Function and the name of each Function a declaration refers to. fn-112
// gives constructs other bodies and retires or adds helper Functions without changing what they mean,
// and what they mean is compared on the tables, answers, receipts, fingerprints and Cases derived from
// the Model. An empty reference stays empty, so whether a declaration names a Function, such as a
// refinement's visibility projection or the point a monitor evaluates at, is still compared.
func functionless(m *umpirespb.Model) error {
	m.Functions = nil
	return messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
		child.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if functionReferences[f.FullName()] && v.String() != "" {
				child.Set(f, protoreflect.ValueOfString(functionReference))
			}
			return true
		})
		return true, nil
	})
}

// MatchOriginal checks a current Model against its baseline under the delta.
func (d Delta) MatchOriginal(expected, current *umpirespb.Model) error {
	want, err := d.ProjectBaseline(expected)
	if err != nil {
		return err
	}
	got, err := d.ProjectCurrent(current)
	if err != nil {
		return err
	}
	if !proto.Equal(want, got) {
		return errors.New("IR differs from the original baseline outside source positions and the recorded delta: " + firstDifference(want, got))
	}
	return nil
}

func firstDifference(want, got *umpirespb.Model) string {
	w, err := Proto(want)
	if err != nil {
		return err.Error()
	}
	g, err := Proto(got)
	if err != nil {
		return err.Error()
	}
	return FirstDifference(w, g)
}

// FirstDifference shows where two encodings first differ.
func FirstDifference(want, got []byte) string {
	at := 0
	for at < len(want) && at < len(got) && want[at] == got[at] {
		at++
	}
	excerpt := func(b []byte) string {
		return string(b[max(0, at-120):min(len(b), at+120)])
	}
	return fmt.Sprintf("at byte %d, expected …%s… got …%s…", at, excerpt(want), excerpt(got))
}

// ProjectedDigest is the digest of a Model as the comparison reads it, for a derived Model such as an
// exploration candidate whose own digest covers positions.
func ProjectedDigest(projected *umpirespb.Model) (string, error) {
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(projected)
	if err != nil {
		return "", err
	}
	return Digest(encoded), nil
}

// located is a path of a Scala source the IR names, with the line and column a diagnostic adds.
var located = regexp.MustCompile(`model/[A-Za-z0-9_./-]*\.scala(?::[0-9]+)*`)

// Located replaces each source position in derived text, and each of the labels, by one token.
func Located(text []byte, labels ...string) []byte {
	for _, label := range labels {
		if label != "" {
			text = bytes.ReplaceAll(text, []byte(label), []byte("<source>"))
		}
	}
	return located.ReplaceAll(text, []byte("<source>"))
}

// OriginalArchive reads the archive's frozen inputs.
func OriginalArchive(root string) (map[string][]byte, error) {
	return Read(filepath.Join(root, OriginalDir, "archive"))
}

// OriginalCurrent reads the current files the archive freezes, under the archive's keys.
func OriginalCurrent(root string) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, prefix := range slices.Sorted(maps.Keys(originalTrees)) {
		paths, err := filepath.Glob(filepath.Join(root, originalTrees[prefix], "*"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			name := filepath.Base(path)
			if !strings.HasSuffix(name, ".json") && prefix+name != OriginalRejects {
				continue
			}
			encoded, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			out[prefix+name] = encoded
		}
	}
	return out, nil
}

// OriginalModels decodes the IR Models of files, by key: every model/ir file and lifter fixture.
func OriginalModels(files map[string][]byte) (map[string]*umpirespb.Model, error) {
	out := map[string]*umpirespb.Model{}
	for key, encoded := range files {
		modelKey := strings.HasPrefix(key, OriginalIR) || strings.HasPrefix(key, OriginalLifts)
		if !strings.HasSuffix(key, ".json") || IsLawSidecar(key) || !modelKey {
			continue
		}
		m := new(umpirespb.Model)
		if err := protojson.Unmarshal(encoded, m); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = m
	}
	return out, nil
}

// Inventory checks the current files against the archived ones: the same IR Models and Cases, every
// archived lifter fixture, and every archived refusal at some line. Later fixtures and refusals, for
// constructs added after the baseline, may be added. Of IR files and Cases only the delta's may be
// added: its new IR files, its new Cases, and the law sidecar of an IR file that is new or has law
// replacements, which lists exactly the claims the delta lists for it. Each must be produced. A
// reduced fixture must be an archived one, and may be retired: only it may no longer be produced.
func (d Delta) Inventory(archived, current map[string][]byte) error {
	errs := d.unproduced(archived, current)
	// added are the files the delta adds, each of which must be produced; optional are the law
	// sidecars of new IR files, which a new IR file without capabilities does not have.
	added, optional := map[string]bool{}, map[string]bool{}
	for _, key := range d.NewIRFiles {
		added[key], optional[sidecarKey(key)] = true, true
	}
	for _, c := range d.NewCases {
		added[c.File()] = true
	}
	generated := map[string][]string{}
	for _, r := range d.Replacements {
		generated[sidecarKey(r.Model)] = append(generated[sidecarKey(r.Model)], r.Generated())
	}
	for _, key := range slices.Sorted(maps.Keys(generated)) {
		added[key] = true
		if encoded, ok := current[key]; ok {
			if err := sidecarLists(encoded, generated[key]); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", key, err))
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(added)) {
		if _, ok := archived[key]; ok {
			errs = append(errs, fmt.Errorf("%s is listed as new and archived", key))
		} else if _, ok := current[key]; !ok {
			errs = append(errs, fmt.Errorf("%s is listed as new and not produced", key))
		}
	}
	maps.Copy(added, optional)
	for _, key := range slices.Sorted(maps.Keys(current)) {
		closed := strings.HasPrefix(key, OriginalIR) || strings.HasPrefix(key, OriginalCases)
		if _, ok := archived[key]; !ok && closed && !added[key] {
			errs = append(errs, fmt.Errorf("%s is produced and not archived", key))
		}
	}
	lines := map[string]bool{}
	for line := range strings.Lines(string(Located(current[OriginalRejects]))) {
		lines[line] = true
	}
	for line := range strings.Lines(string(Located(archived[OriginalRejects]))) {
		if !lines[line] {
			errs = append(errs, fmt.Errorf("the lifter no longer refuses %q", strings.TrimSpace(line)))
		}
	}
	return errors.Join(errs...)
}

// sidecarKey is the law sidecar beside an IR file.
func sidecarKey(key string) string { return strings.TrimSuffix(key, ".json") + LawSidecarSuffix }

// sidecarLists checks that a law sidecar lists exactly the named generated claims.
func sidecarLists(encoded []byte, names []string) error {
	var sidecar struct {
		Claims []struct {
			Name string `json:"name"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(encoded, &sidecar); err != nil {
		return err
	}
	var listed []string
	for _, c := range sidecar.Claims {
		listed = append(listed, c.Name)
	}
	slices.Sort(listed)
	names = slices.Sorted(slices.Values(names))
	if !slices.Equal(listed, names) {
		return fmt.Errorf("the sidecar lists generated claims %v, and the delta %v", listed, names)
	}
	return nil
}

// CaptureOriginal archives the current files exclusively in dir.
func CaptureOriginal(root, dir string) error {
	files, err := OriginalCurrent(root)
	if err != nil {
		return err
	}
	return Capture(filepath.Join(dir, "archive"), files)
}

// Derived is what the Go tooling derived from the archived inputs: the digest of each output, by
// key, as the comparison reads it. The outputs themselves are derived again from the archive, and
// only to explain a difference: the largest is hundreds of megabytes of JSON.
type Derived map[string]string

// Stream digests one derived output part by part, without holding its encoding: each part, such
// as one Property's rows, is encoded, projected by Located and hashed on its own.
type Stream struct {
	// Keep records each part's name and digest.
	Keep  bool
	Parts []Part
	// Only names the one part whose projected encoding Captured keeps.
	Only     string
	Captured []byte
	// Verbatim hashes each part as encoded, with its positions.
	Verbatim bool
	hash     hash.Hash
}

type Part struct{ Name, Digest string }

// Add appends one part: a value encoded as JSON, or bytes as they are. Labels are the Model sources
// Located replaces.
func (s *Stream) Add(name string, v any, labels ...string) error {
	encoded, ok := v.([]byte)
	if !ok {
		var err error
		if encoded, err = json.Marshal(v); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if !s.Verbatim {
		encoded = Located(encoded, labels...)
	}
	if s.hash == nil {
		s.hash = sha256.New()
	}
	// The name and the length frame each part, so no two sequences of parts hash alike.
	_, _ = fmt.Fprintf(s.hash, "%d:%s\n%d:", len(name), name, len(encoded))
	_, _ = s.hash.Write(encoded)
	if s.Keep {
		s.Parts = append(s.Parts, Part{Name: name, Digest: Digest(encoded)})
	}
	if s.Only == name {
		s.Captured = encoded
	}
	return nil
}

func (s *Stream) Digest() string {
	if s.hash == nil {
		s.hash = sha256.New()
	}
	return hex.EncodeToString(s.hash.Sum(nil))
}

// ReadDerived reads a derived-output file of the archive.
func ReadDerived(root, name string) (Derived, error) {
	encoded, err := os.ReadFile(filepath.Join(root, OriginalDir, name))
	if err != nil {
		return nil, err
	}
	var d Derived
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return d, decoder.Decode(&d)
}

// WriteDerived writes a derived-output file exclusively into dir.
func WriteDerived(dir, name string, outputs Derived) error {
	encoded, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(encoded, '\n'))
	return errors.Join(writeErr, file.Close())
}

// CompareDerived compares the digests of the current outputs with the expected ones. explain, when
// given, says where an output that differs first differs.
func CompareDerived(expected, current Derived, explain func(key string) string) error {
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(expected)) {
		got, ok := current[key]
		if !ok {
			errs = append(errs, fmt.Errorf("missing derived output %s", key))
			continue
		}
		if got == expected[key] {
			continue
		}
		detail := ""
		if explain != nil {
			detail = ": " + explain(key)
		}
		errs = append(errs, fmt.Errorf("derived output %s differs from the original baseline%s", key, detail))
	}
	for _, key := range slices.Sorted(maps.Keys(current)) {
		if _, ok := expected[key]; !ok {
			errs = append(errs, fmt.Errorf("unknown derived output %s", key))
		}
	}
	return errors.Join(errs...)
}

// Explain derives one output on each side again, part by part, and shows the first part that
// differs. derive streams the expected side's output when expected is set, the current side's
// otherwise; archived is the expected digest the archive holds.
func Explain(archived string, derive func(expected bool, s *Stream) error) string {
	want, got := &Stream{Keep: true}, &Stream{Keep: true}
	if err := errors.Join(derive(true, want), derive(false, got)); err != nil {
		return err.Error()
	}
	prefix := ""
	if want.Digest() != archived {
		prefix = "the archive itself derives differently now, so the Go tooling changed what it derives; "
	}
	for i := range max(len(want.Parts), len(got.Parts)) {
		if i < len(want.Parts) && i < len(got.Parts) && want.Parts[i] == got.Parts[i] {
			continue
		}
		if i >= len(want.Parts) {
			return prefix + "an extra part " + got.Parts[i].Name
		}
		if i >= len(got.Parts) {
			return prefix + "a missing part " + want.Parts[i].Name
		}
		if want.Parts[i].Name != got.Parts[i].Name {
			return fmt.Sprintf("%spart %d is %s, expected %s", prefix, i, got.Parts[i].Name, want.Parts[i].Name)
		}
		name := want.Parts[i].Name
		want, got = &Stream{Only: name}, &Stream{Only: name}
		if err := errors.Join(derive(true, want), derive(false, got)); err != nil {
			return err.Error()
		}
		return prefix + "part " + name + " " + FirstDifference(want.Captured, got.Captured)
	}
	return prefix + "every part is the same"
}
