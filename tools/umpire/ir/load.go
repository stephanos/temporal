// Package ir loads the Umpire IR and admits it: Load reads a Model's ProtoJSON and Validate checks
// that it is well formed, its keys, totals, realizations and explorations included, before anything
// interprets or checks it.
package ir

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// AcceptedSuffix ends the file of lint's accepted findings an author writes beside an IR file,
// `<file>.lint.json`: JSON and no Model. Lint, which imports the reader, names it by this constant.
const AcceptedSuffix = ".lint.json"

// IRPaths lists the IR files of a directory: its JSON files, apart from the accepted lint findings
// beside them. Any other JSON file is loaded and refused if it is not an IR Model.
func IRPaths(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(paths, func(p string) bool {
		return strings.HasSuffix(p, AcceptedSuffix)
	}), nil
}

// Load reads a Model in ProtoJSON, rejects fields the schema does not have, and validates it. Fields
// a Model may leave unset, a Property's origin among them, are admitted unset.
func Load(path string) (*umpirespb.Model, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &umpirespb.Model{}
	if err := (protojson.UnmarshalOptions{}).Unmarshal(encoded, m); err != nil {
		return nil, &interp.Error{Position: path, Message: err.Error()}
	}
	if err := Validate(m); err != nil {
		return nil, err
	}
	return m, nil
}

// choiceField is Construct.choice, the name of a named choice's alternative.
var choiceField = (&umpirespb.Construct{}).ProtoReflect().Descriptor().Fields().ByName("choice")

// WithoutChoiceNames is a copy of a Model with every named choice's names cleared: what an identity
// hashed from a Model's content reads, since the names are inert (model/SEMANTICS.md, Named choices).
func WithoutChoiceNames(m *umpirespb.Model) *umpirespb.Model {
	out := proto.CloneOf(m)
	clearChoices(out.ProtoReflect())
	return out
}

func clearChoices(m protoreflect.Message) {
	if m.Descriptor() == choiceField.ContainingMessage() {
		m.Clear(choiceField)
	}
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case f.Message() == nil || f.IsMap():
		case f.IsList():
			for i := range v.List().Len() {
				clearChoices(v.List().Get(i).Message())
			}
		default:
			clearChoices(v.Message())
		}
		return true
	})
}
