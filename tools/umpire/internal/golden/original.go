package golden

import (
	"bytes"
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
// beyond source positions and Functions: nothing else may differ.
type Delta struct {
	// InertFields are IR fields, by full protobuf name, that the baseline never sets and that carry
	// metadata no table, ID, fingerprint, answer or Case reads: Query.total, named-choice names.
	InertFields []string `json:"inert_fields"`
	// Attachments are the entity attachments of fn-112's R20 task-queue entity. Each sets one field the
	// baseline left empty, so whatever reads it is derived again from the baseline plus the attachment.
	Attachments []Attachment `json:"entity_attachments"`
}

// Attachment attaches the machine of a name, or the action of an ID, to an entity.
type Attachment struct {
	Machine string `json:"machine,omitempty"`
	Action  string `json:"action,omitempty"`
	// Field is the attachment: "entity" of a machine, "on" or "creates" of an action.
	Field  string `json:"field"`
	Entity string `json:"entity"`
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

func (d Delta) check() error {
	for _, name := range d.InertFields {
		if _, err := inertField(name); err != nil {
			return err
		}
	}
	for _, a := range d.Attachments {
		valid := a.Entity != "" && (a.Machine == "") != (a.Action == "") &&
			(a.Machine == "" || a.Field == "entity") && (a.Action == "" || a.Field == "on" || a.Field == "creates")
		if !valid {
			return fmt.Errorf("entity attachment %+v is not one machine's entity or one action's on or creates", a)
		}
	}
	return nil
}

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

// Expected is the baseline Model with the attachments applied: what the current Model must equal.
// An attachment must find its declaration in some baseline Model, so applied reports each it applied.
func (d Delta) Expected(baseline *umpirespb.Model, applied map[int]bool) (*umpirespb.Model, error) {
	m := proto.CloneOf(baseline)
	for i, a := range d.Attachments {
		for _, machine := range m.GetMachines() {
			if a.Machine != "" && machine.GetName() == a.Machine {
				if machine.GetEntity() != "" {
					return nil, fmt.Errorf("machine %s already has entity %s", a.Machine, machine.GetEntity())
				}
				machine.Entity = a.Entity
				applied[i] = true
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
				return nil, fmt.Errorf("action %s already has %s %s", a.Action, a.Field, *target)
			}
			*target = a.Entity
			applied[i] = true
		}
	}
	return m, nil
}

// Unapplied names the attachments no baseline Model had a declaration for.
func (d Delta) Unapplied(applied map[int]bool) error {
	var errs []error
	for i, a := range d.Attachments {
		if !applied[i] {
			errs = append(errs, fmt.Errorf("entity attachment %+v names no declaration of the baseline", a))
		}
	}
	return errors.Join(errs...)
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
		if !strings.HasSuffix(key, ".json") || !modelKey {
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

// OriginalInventory checks the current files against the archived ones: the same IR Models and
// Cases, every archived lifter fixture, and every archived refusal at some line. Later fixtures and
// refusals, for constructs added after the baseline, may be added.
func OriginalInventory(archived, current map[string][]byte) error {
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(archived)) {
		if _, ok := current[key]; !ok {
			errs = append(errs, fmt.Errorf("%s is archived and no longer produced", key))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(current)) {
		closed := strings.HasPrefix(key, OriginalIR) || strings.HasPrefix(key, OriginalCases)
		if _, ok := archived[key]; !ok && closed {
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
