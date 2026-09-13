package protocolmigration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
)

// The R14 step is a relation rather than a function of the baseline: the Case-local names and value
// spellings a Producer derives are read from the regenerated fixture's provenance rows, validated
// against the baseline, and substituted into it. The oracle does not re-derive the naming rule, so
// a regenerated fixture passes exactly when its rows explain every renamed name and value and the
// renaming they declare cannot merge two baseline Definition IDs or two encodings of one definition.

// localizeNames substitutes the regenerated fixture's Case-local names and model value spellings
// into the baseline and adds the rows that declare them. A fixture that carries no rows maps nothing.
func localizeNames(fixture string, tree any, regenerated []byte) (any, error) {
	if regenerated == nil {
		return tree, nil
	}
	switch name := path.Base(fixture); {
	case name == "correlated.json":
		return localizeCorrelatedEntries(tree, regenerated)
	case name == "case.json" || strings.HasSuffix(name, "-case.json"):
		regeneratedCase, err := testpilot.DecodeCaseProtoJSON(regenerated)
		if err != nil {
			return nil, fmt.Errorf("regenerated fixture: %w", err)
		}
		root, isObject := tree.(*Object)
		if !isObject {
			return tree, nil
		}
		renaming, err := newLocalNames(root, regeneratedCase.GetProvenance())
		if err != nil {
			return nil, err
		}
		return root, renaming.localizeCase(root)
	default:
		return tree, nil
	}
}

// localizeCorrelatedEntries localizes each corpus entry's Cases under their own regenerated rows,
// and its events under the rows of the runnable Case that consumes them.
func localizeCorrelatedEntries(tree any, regenerated []byte) (any, error) {
	entries, isList := tree.([]any)
	if !isList {
		return tree, nil
	}
	var regeneratedEntries []map[string]json.RawMessage
	if err := json.Unmarshal(regenerated, &regeneratedEntries); err != nil {
		return nil, fmt.Errorf("regenerated fixture: %w", err)
	}
	if len(regeneratedEntries) != len(entries) {
		return nil, fmt.Errorf("regenerated fixture has %d entries, the baseline %d", len(regeneratedEntries), len(entries))
	}
	for index, element := range entries {
		entry, isObject := element.(*Object)
		if !isObject {
			return nil, fmt.Errorf("entry %d is not an object", index)
		}
		renamings := map[string]*localNames{}
		for _, key := range []string{"case", "runnableCase"} {
			root, isCase := entry.Fields[key].(*Object)
			if !isCase {
				continue
			}
			regeneratedCase, err := testpilot.DecodeCaseProtoJSON(regeneratedEntries[index][key])
			if err != nil {
				return nil, fmt.Errorf("regenerated entry %d %s: %w", index, key, err)
			}
			renaming, err := newLocalNames(root, regeneratedCase.GetProvenance())
			if err != nil {
				return nil, fmt.Errorf("entry %d %s: %w", index, key, err)
			}
			if err := renaming.localizeCase(root); err != nil {
				return nil, fmt.Errorf("entry %d %s: %w", index, key, err)
			}
			renamings[key] = renaming
		}
		events, _ := entry.Fields["events"].([]any)
		if len(events) > 0 && renamings["runnableCase"] == nil {
			return nil, fmt.Errorf("entry %d carries events and no runnable Case", index)
		}
		for _, event := range events {
			renamings["runnableCase"].visitEvidence(event)
		}
	}
	return tree, nil
}

// localNames is one regenerated Case's renaming, validated against its baseline Case.
type localNames struct {
	rows      []*testpilotspb.LocalName
	values    []*testpilotspb.ModelValueFingerprint
	names     map[string]string
	spellings map[[2]string]string
}

func newLocalNames(baseline *Object, provenance *testpilotspb.CaseProvenance) (*localNames, error) {
	renaming := &localNames{
		rows: provenance.GetLocalNames(), values: provenance.GetModelValueFingerprints(),
		names: map[string]string{}, spellings: map[[2]string]string{},
	}
	locals := map[string]bool{}
	for _, row := range renaming.rows {
		if _, repeated := renaming.names[row.GetDefinitionId()]; repeated {
			return nil, fmt.Errorf("local name rows name Definition ID %q twice", row.GetDefinitionId())
		}
		if locals[row.GetLocalName()] {
			return nil, fmt.Errorf("local name %q stands for two Definition IDs", row.GetLocalName())
		}
		renaming.names[row.GetDefinitionId()] = row.GetLocalName()
		locals[row.GetLocalName()] = true
	}
	for _, row := range renaming.values {
		key := [2]string{row.GetLocalName(), row.GetFingerprint()}
		if _, repeated := renaming.spellings[key]; repeated {
			return nil, fmt.Errorf("model value fingerprint %s of %q is recorded twice", row.GetFingerprint(), row.GetLocalName())
		}
		renaming.spellings[key] = row.GetSpelling()
	}
	if len(renaming.rows) == 0 && len(renaming.values) == 0 {
		return renaming, nil
	}
	return renaming, renaming.validate(baseline)
}

// nameOnly is the local name the rows give a Definition ID, or the ID itself when no row maps it.
func (r *localNames) nameOnly(id string) string {
	if local, renamed := r.names[id]; renamed {
		return local
	}
	return id
}

// valueOf is the spelling the rows give an encoding of a definition: a recorded fingerprint's
// spelling, else the local name of an encoding that is a Definition ID, else the encoding itself.
func (r *localNames) valueOf(definitionID, encoding string) string {
	if spelling, recorded := r.spellings[[2]string{r.nameOnly(definitionID), fingerprint(encoding)}]; recorded {
		return spelling
	}
	return r.nameOnly(encoding)
}

func fingerprint(encoding string) string {
	sum := sha256.Sum256([]byte(encoding))
	return hex.EncodeToString(sum[:])
}

// validate requires every row to describe the baseline: each LocalName row's Definition ID and each
// fingerprint's encoding occur in it, and the renaming is injective over its Definition IDs and over
// each definition's encodings.
func (r *localNames) validate(baseline *Object) error {
	var ids []string
	encodings := map[string][]string{}
	var definitions []string
	r.visitCase(baseline, func(id string) string {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
		return id
	}, func(definitionID, encoding string) string {
		if !slices.Contains(definitions, definitionID) {
			definitions = append(definitions, definitionID)
		}
		if !slices.Contains(encodings[definitionID], encoding) {
			encodings[definitionID] = append(encodings[definitionID], encoding)
		}
		if _, renamed := r.names[encoding]; renamed && !slices.Contains(ids, encoding) {
			ids = append(ids, encoding)
		}
		return encoding
	})
	for _, row := range r.rows {
		if !slices.Contains(ids, row.GetDefinitionId()) {
			return fmt.Errorf("local name %q maps Definition ID %q, which the baseline does not name", row.GetLocalName(), row.GetDefinitionId())
		}
	}
	renamedIDs := map[string]string{}
	for _, id := range ids {
		local := r.nameOnly(id)
		if other, merged := renamedIDs[local]; merged {
			return fmt.Errorf("local name %q stands for baseline Definition IDs %q and %q", local, other, id)
		}
		renamedIDs[local] = id
	}
	for _, row := range r.values {
		if !slices.ContainsFunc(definitions, func(definitionID string) bool {
			return r.nameOnly(definitionID) == row.GetLocalName() && slices.ContainsFunc(encodings[definitionID], func(encoding string) bool {
				return fingerprint(encoding) == row.GetFingerprint()
			})
		}) {
			return fmt.Errorf("model value fingerprint %s of %q matches no baseline encoding of that definition", row.GetFingerprint(), row.GetLocalName())
		}
	}
	for _, definitionID := range definitions {
		spelled := map[string]string{}
		for _, encoding := range encodings[definitionID] {
			spelling := r.valueOf(definitionID, encoding)
			if other, merged := spelled[spelling]; merged {
				return fmt.Errorf("spelling %q of %q stands for two baseline encodings of fingerprints %s and %s", spelling, definitionID, fingerprint(other), fingerprint(encoding))
			}
			spelled[spelling] = encoding
		}
	}
	return nil
}

// localizeCase renames the baseline Case in place and adds the rows that declare the renaming.
func (r *localNames) localizeCase(root *Object) error {
	if len(r.rows) == 0 && len(r.values) == 0 {
		return nil
	}
	r.visitCase(root, r.nameOnly, r.valueOf)
	provenance, isObject := root.Fields["provenance"].(*Object)
	if !isObject {
		return errors.New("a Case whose regenerated fixture carries local name rows has no provenance")
	}
	encoded, err := protojson.Marshal(&testpilotspb.CaseProvenance{LocalNames: r.rows, ModelValueFingerprints: r.values})
	if err != nil {
		return err
	}
	rows, err := decodeJSON(encoded)
	if err != nil {
		return err
	}
	for key, list := range rows.(*Object).Fields {
		if _, clashes := provenance.Fields[key]; clashes {
			return fmt.Errorf("provenance already carries %q", key)
		}
		provenance.Fields[key] = list
	}
	return nil
}

// visitCase applies name to every name position and value to every model value position of a
// Case's JSON tree: each Contract rule id; the correlated contract's projection, operation and scope
// fields, sources, evidence kinds, field policies, captures, rule ids, step references and model
// values; and the Program's evidence lift rules. A value is visited with its definition's Definition
// ID before that ID is renamed, and a step condition's text is visited as a value of its step's
// definition. The positions mirror Umpire.Case.LocalNames' traversal in Lean; a new Definition ID
// position is added to both.
func (r *localNames) visitCase(root *Object, name func(string) string, value func(definitionID, encoding string) string) {
	program := child(root, "program")
	for _, entrypoint := range append(childObjects(program, "entrypoints"), child(program, "cleanup")) {
		for _, node := range childObjects(entrypoint, "instructions") {
			for _, read := range childObjects(child(child(node, "instruction"), "invokeRpc"), "responseReads") {
				for _, target := range childObjects(read, "targets") {
					for _, rule := range childObjects(child(target, "correlatedEvidence"), "rules") {
						renameKey(rule, "evidenceSource", name)
						renameKey(rule, "kind", name)
						for _, named := range append(childObjects(rule, "scope"), childObjects(rule, "fields")...) {
							renameKey(named, "fieldId", name)
						}
					}
				}
			}
		}
	}
	contract := child(root, "contract")
	for _, rule := range childObjects(contract, "rules") {
		renameKey(rule, "ruleId", name)
	}
	correlated := child(contract, "correlated")
	if correlated == nil {
		return
	}
	renameKey(correlated, "projectionId", name)
	renameList(correlated, "scopeFields", name)
	renameKey(correlated, "operationField", name)
	renameList(correlated, "sources", name)
	visitModelValue(child(correlated, "initialState"), name, value)
	transitions := childObjects(correlated, "transitions")
	for _, rule := range childObjects(correlated, "projectionRules") {
		renameKey(rule, "kind", name)
		visitModelValue(child(rule, "submission"), name, value)
		transitions = append(transitions, childObjects(rule, "outputs")...)
		for _, field := range childObjects(rule, "fields") {
			renameKey(field, "fieldId", name)
		}
	}
	for _, transition := range transitions {
		for _, key := range []string{"priorState", "action", "state", "outcome"} {
			visitModelValue(child(transition, key), name, value)
		}
		for _, fact := range childObjects(transition, "facts") {
			visitModelValue(fact, name, value)
		}
	}
	for _, rule := range childObjects(correlated, "rules") {
		renameKey(rule, "ruleId", name)
		for _, capture := range childObjects(rule, "captures") {
			renameKey(capture, "captureId", name)
			renameKey(capture, "fieldId", name)
		}
		for _, key := range []string{"trigger", "response", "correlation"} {
			visitExpression(child(rule, key), name, value)
		}
	}
}

// visitEvidence renames one correlated evidence event's scope fields, source, kind and fields.
func (r *localNames) visitEvidence(event any) {
	object, _ := event.(*Object)
	for _, identity := range append(childObjects(object, "parents"), child(object, "identity")) {
		renameKey(identity, "evidenceSource", r.nameOnly)
		for _, scope := range childObjects(identity, "scope") {
			renameKey(scope, "fieldId", r.nameOnly)
		}
	}
	renameKey(object, "kind", r.nameOnly)
	for _, field := range childObjects(object, "fields") {
		renameKey(field, "fieldId", r.nameOnly)
	}
}

func visitModelValue(object *Object, name func(string) string, value func(definitionID, encoding string) string) {
	if object == nil {
		return
	}
	definitionID, _ := object.Fields["definitionId"].(string)
	if encoding, present := object.Fields["value"].(string); present {
		object.Fields["value"] = value(definitionID, encoding)
	}
	renameKey(object, "definitionId", name)
}

func visitExpression(expression *Object, name func(string) string, value func(definitionID, encoding string) string) {
	if expression == nil {
		return
	}
	reference := child(expression, "reference")
	renameKey(child(reference, "correlatedStep"), "definitionId", name)
	renameKey(reference, "evidenceFieldId", name)
	renameKey(child(reference, "correlatedCapture"), "captureId", name)
	visitModelValue(child(reference, "modelValue"), name, value)
	for _, key := range []string{"present", "not", "path"} {
		visitExpression(child(child(expression, key), "operand"), name, value)
	}
	for _, key := range []string{"all", "any"} {
		for _, operand := range childObjects(child(expression, key), "operands") {
			visitExpression(operand, name, value)
		}
	}
	compare := child(expression, "compare")
	if compare == nil {
		return
	}
	step := child(child(child(compare, "left"), "reference"), "correlatedStep")
	literal := child(child(compare, "right"), "literal")
	if text, isText := literal.fieldString("textValue"); step != nil && isText {
		definitionID, _ := step.Fields["definitionId"].(string)
		literal.Fields["textValue"] = value(definitionID, text)
	} else {
		visitExpression(child(compare, "right"), name, value)
	}
	visitExpression(child(compare, "left"), name, value)
}

func child(object *Object, key string) *Object {
	if object == nil {
		return nil
	}
	found, _ := object.Fields[key].(*Object)
	return found
}

func childObjects(object *Object, key string) []*Object {
	if object == nil {
		return nil
	}
	list, _ := object.Fields[key].([]any)
	found := make([]*Object, 0, len(list))
	for _, element := range list {
		if element, isObject := element.(*Object); isObject {
			found = append(found, element)
		}
	}
	return found
}

func (o *Object) fieldString(key string) (string, bool) {
	if o == nil {
		return "", false
	}
	text, isText := o.Fields[key].(string)
	return text, isText
}

func renameKey(object *Object, key string, name func(string) string) {
	if text, isText := object.fieldString(key); isText {
		object.Fields[key] = name(text)
	}
}

func renameList(object *Object, key string, name func(string) string) {
	list, _ := object.Fields[key].([]any)
	for index, element := range list {
		if text, isText := element.(string); isText {
			list[index] = name(text)
		}
	}
}
