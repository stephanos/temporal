package protocolmigration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	testpilotPackagePrefix = "temporal.server.api.testpilot."
	caseMessage            = protoreflect.FullName("temporal.server.api.testpilot.v1.Case")
	evidenceMessage        = protoreflect.FullName("temporal.server.api.testpilot.v1.CorrelatedEvidence")
	anyMessage             = protoreflect.FullName("google.protobuf.Any")
)

// Baseline is the frozen descriptor snapshot the pre-migration fixtures were written against.
type Baseline struct {
	files *protoregistry.Files
	types *dynamicpb.Types
}

// LoadBaseline reads a descriptor set into its own registry. The snapshot declares the same full
// names as the generated protocol, so it never touches protoregistry.GlobalFiles.
func LoadBaseline(descriptorSetPath string) (*Baseline, error) {
	encoded, err := os.ReadFile(descriptorSetPath)
	if err != nil {
		return nil, err
	}
	set := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(encoded, set); err != nil {
		return nil, fmt.Errorf("decode %s: %w", descriptorSetPath, err)
	}
	files, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", descriptorSetPath, err)
	}
	return &Baseline{files: files, types: dynamicpb.NewTypes(files)}, nil
}

// Check maps one baseline fixture under mapping and compares it with its regenerated counterpart.
// fixture is the repository-relative path both trees share and selects the fixture's schema.
func (b *Baseline) Check(fixture string, baseline, regenerated []byte, mapping Mapping) error {
	name := path.Base(fixture)
	switch {
	case name == "expected.json":
		return checkExpected(fixture, baseline, regenerated, mapping)
	case name == "correlated.json":
		return b.checkCorrelated(fixture, baseline, regenerated, mapping)
	case name == "case.json" || strings.HasSuffix(name, "-case.json"):
		return b.checkCase(fixture, baseline, regenerated, mapping)
	default:
		return fmt.Errorf("fixture %s has no declared schema", fixture)
	}
}

func (b *Baseline) checkCase(fixture string, baseline, regenerated []byte, mapping Mapping) error {
	tree, err := b.typedMessage(fixture, baseline, caseMessage)
	if err != nil {
		return err
	}
	mapped, err := mapping.apply(fixture, tree)
	if err != nil {
		return err
	}
	return compareMessage(fixture, "", mapped, regenerated, decodeCase)
}

// checkExpected treats expected.json as the Verdict pin: it must stay byte-identical unless a
// declared step changed its JSON, in which case the regenerated JSON must equal the mapped JSON.
func checkExpected(fixture string, baseline, regenerated []byte, mapping Mapping) error {
	tree, err := decodeJSON(baseline)
	if err != nil {
		return fmt.Errorf("fixture %s: baseline: %w", fixture, err)
	}
	original, err := json.Marshal(tree)
	if err != nil {
		return err
	}
	mapped, err := mapping.apply(fixture, tree)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(mapped)
	if err != nil {
		return fmt.Errorf("fixture %s: encode mapped baseline: %w", fixture, err)
	}
	if !bytes.Equal(original, encoded) {
		return jsonDifference(fixture, "", encoded, regenerated)
	}
	if err := jsonDifference(fixture, "", baseline, regenerated); err != nil {
		return err
	}
	if !bytes.Equal(baseline, regenerated) {
		return fmt.Errorf("fixture %s: bytes differ from the baseline with equal JSON and no declared step", fixture)
	}
	return nil
}

// checkCorrelated compares the correlated conformance corpus entry by entry. Its Cases and events
// are protocol messages; expected and incomplete are the corpus's Verdict pins and must not move.
func (b *Baseline) checkCorrelated(fixture string, baseline, regenerated []byte, mapping Mapping) error {
	var baselineEntries []map[string]json.RawMessage
	if err := json.Unmarshal(baseline, &baselineEntries); err != nil {
		return fmt.Errorf("fixture %s: baseline: %w", fixture, err)
	}
	tree := make([]any, len(baselineEntries))
	for index, entry := range baselineEntries {
		typed, err := b.typedCorrelatedEntry(fmt.Sprintf("%s[%d]", fixture, index), entry)
		if err != nil {
			return err
		}
		tree[index] = typed
	}
	mapped, err := mapping.apply(fixture, tree)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(mapped)
	if err != nil {
		return fmt.Errorf("fixture %s: encode mapped baseline: %w", fixture, err)
	}
	var mappedEntries, regeneratedEntries []map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &mappedEntries); err != nil {
		return fmt.Errorf("fixture %s: mapped baseline is not a list of entries: %w", fixture, err)
	}
	if err := json.Unmarshal(regenerated, &regeneratedEntries); err != nil {
		return fmt.Errorf("fixture %s: regenerated fixture is not a list of entries: %w", fixture, err)
	}
	if len(mappedEntries) != len(regeneratedEntries) {
		return fmt.Errorf("fixture %s: mapped baseline has %d entries and the regenerated fixture %d",
			fixture, len(mappedEntries), len(regeneratedEntries))
	}
	for index := range mappedEntries {
		if err := compareCorrelatedEntry(fixture, index, baselineEntries[index], mappedEntries[index], regeneratedEntries[index]); err != nil {
			return err
		}
	}
	return nil
}

func (b *Baseline) typedCorrelatedEntry(location string, entry map[string]json.RawMessage) (*Object, error) {
	object := &Object{Fields: make(map[string]any, len(entry))}
	for key, raw := range entry {
		var (
			typed any
			err   error
		)
		switch key {
		case "case", "runnableCase":
			typed, err = b.typedMessage(location+"."+key, raw, caseMessage)
		case "events":
			typed, err = b.typedEvents(location+"."+key, raw)
		case "expected", "incomplete", "name":
			typed, err = decodeJSON(raw)
		default:
			return nil, fmt.Errorf("fixture %s: undeclared entry field %q", location, key)
		}
		if err != nil {
			return nil, err
		}
		object.Fields[key] = typed
	}
	return object, nil
}

func (b *Baseline) typedEvents(location string, raw json.RawMessage) ([]any, error) {
	var events []json.RawMessage
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, fmt.Errorf("fixture %s: %w", location, err)
	}
	typed := make([]any, len(events))
	for index, event := range events {
		value, err := b.typedMessage(fmt.Sprintf("%s[%d]", location, index), event, evidenceMessage)
		if err != nil {
			return nil, err
		}
		typed[index] = value
	}
	return typed, nil
}

func compareCorrelatedEntry(fixture string, index int, baseline, mapped, regenerated map[string]json.RawMessage) error {
	location := fmt.Sprintf("[%d]", index)
	mappedKeys, regeneratedKeys := sortedKeys(mapped), sortedKeys(regenerated)
	if !slices.Equal(mappedKeys, regeneratedKeys) {
		return fmt.Errorf("fixture %s: entry %s has fields %v mapped and %v regenerated", fixture, location, mappedKeys, regeneratedKeys)
	}
	for _, key := range mappedKeys {
		fieldLocation := location + "." + key
		var err error
		switch key {
		case "case", "runnableCase":
			err = compareMessage(fixture, fieldLocation, mapped[key], regenerated[key], decodeCase)
		case "events":
			err = compareEvents(fixture, fieldLocation, mapped[key], regenerated[key])
		case "expected", "incomplete":
			err = jsonDifference(fixture, fieldLocation, baseline[key], regenerated[key])
		default:
			err = jsonDifference(fixture, fieldLocation, mapped[key], regenerated[key])
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func compareEvents(fixture, location string, mapped, regenerated json.RawMessage) error {
	var mappedEvents, regeneratedEvents []json.RawMessage
	if err := json.Unmarshal(mapped, &mappedEvents); err != nil {
		return fmt.Errorf("fixture %s: mapped %s: %w", fixture, location, err)
	}
	if err := json.Unmarshal(regenerated, &regeneratedEvents); err != nil {
		return fmt.Errorf("fixture %s: regenerated %s: %w", fixture, location, err)
	}
	if len(mappedEvents) != len(regeneratedEvents) {
		return fmt.Errorf("fixture %s: %s has %d events mapped and %d regenerated", fixture, location, len(mappedEvents), len(regeneratedEvents))
	}
	for index := range mappedEvents {
		if err := compareMessage(fixture, fmt.Sprintf("%s[%d]", location, index), mappedEvents[index], regeneratedEvents[index], decodeEvidence); err != nil {
			return err
		}
	}
	return nil
}

func decodeCase(encoded []byte) (proto.Message, error) {
	return testpilot.DecodeCaseProtoJSON(encoded)
}

func decodeEvidence(encoded []byte) (proto.Message, error) {
	decoded := new(testpilotspb.CorrelatedEvidence)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// compareMessage strictly decodes the mapped baseline and the regenerated value into the current
// generated types and names the first field where they differ.
func compareMessage(fixture, location string, mapped any, regenerated []byte, decode func([]byte) (proto.Message, error)) error {
	encoded, ok := mapped.(json.RawMessage)
	if !ok {
		var err error
		if encoded, err = json.Marshal(mapped); err != nil {
			return fmt.Errorf("fixture %s: encode mapped baseline%s: %w", fixture, describe(location), err)
		}
	}
	want, err := decode(encoded)
	if err != nil {
		return fmt.Errorf("fixture %s: mapped baseline%s does not decode under the current protocol: %w", fixture, describe(location), err)
	}
	got, err := decode(regenerated)
	if err != nil {
		return fmt.Errorf("fixture %s: regenerated fixture%s does not decode under the current protocol: %w", fixture, describe(location), err)
	}
	return difference(fixture, location, want, got, protocmp.Transform())
}

func jsonDifference(fixture, location string, want, got []byte) error {
	wantTree, err := decodePlainJSON(want)
	if err != nil {
		return fmt.Errorf("fixture %s: mapped baseline%s: %w", fixture, describe(location), err)
	}
	gotTree, err := decodePlainJSON(got)
	if err != nil {
		return fmt.Errorf("fixture %s: regenerated fixture%s: %w", fixture, describe(location), err)
	}
	return difference(fixture, location, wantTree, gotTree)
}

func difference(fixture, location string, want, got any, options ...cmp.Option) error {
	first := &firstDifference{}
	if cmp.Equal(want, got, append(slices.Clone(options), cmp.Reporter(first))...) {
		return nil
	}
	return fmt.Errorf("fixture %s differs from its mapped baseline at %s (-mapped +regenerated):\n%s",
		fixture, joinPath(location, first.path), cmp.Diff(want, got, options...))
}

func describe(location string) string {
	if location == "" {
		return ""
	}
	return " " + location
}

func joinPath(location, field string) string {
	switch {
	case location == "" && field == "":
		return "the root"
	case location == "":
		return field
	case field == "" || strings.HasPrefix(field, "["):
		return location + field
	default:
		return location + "." + field
	}
}

// firstDifference records the path of the first unequal node cmp reports, rendered as the
// field and index path a reader finds in the fixture.
type firstDifference struct {
	steps cmp.Path
	found bool
	path  string
}

func (r *firstDifference) PushStep(step cmp.PathStep) { r.steps = append(r.steps, step) }

func (r *firstDifference) PopStep() { r.steps = r.steps[:len(r.steps)-1] }

func (r *firstDifference) Report(result cmp.Result) {
	if r.found || result.Equal() {
		return
	}
	r.found = true
	var rendered strings.Builder
	for _, step := range r.steps {
		switch typed := step.(type) {
		case cmp.MapIndex:
			if rendered.Len() > 0 {
				rendered.WriteByte('.')
			}
			fmt.Fprint(&rendered, typed.Key())
		case cmp.SliceIndex:
			index, other := typed.SplitKeys()
			if index < 0 {
				index = other
			}
			fmt.Fprintf(&rendered, "[%d]", index)
		default:
			// Transforms, type assertions and indirections do not appear in a fixture path.
		}
	}
	r.path = rendered.String()
}

// typedMessage validates encoded strictly against the snapshot message and returns its JSON tree
// with every snapshot message object annotated.
func (b *Baseline) typedMessage(location string, encoded []byte, message protoreflect.FullName) (any, error) {
	messageType, err := b.types.FindMessageByName(message)
	if err != nil {
		return nil, fmt.Errorf("snapshot lacks %s: %w", message, err)
	}
	options := protojson.UnmarshalOptions{DiscardUnknown: false, Resolver: snapshotResolver{snapshot: b.types}}
	if err := options.Unmarshal(encoded, messageType.New().Interface()); err != nil {
		return nil, fmt.Errorf("fixture %s: baseline does not decode through the snapshot: %w", location, err)
	}
	tree, err := decodeJSON(encoded)
	if err != nil {
		return nil, fmt.Errorf("fixture %s: baseline: %w", location, err)
	}
	return b.annotate(location, tree, messageType.Descriptor())
}

func (b *Baseline) annotate(location string, value any, message protoreflect.MessageDescriptor) (any, error) {
	object, ok := value.(*Object)
	if !ok {
		// Well-known types such as Duration encode a message as a JSON scalar.
		return value, nil
	}
	object.Message = message.FullName()
	fields := message
	if message.FullName() == anyMessage {
		payload, err := b.anyPayload(object)
		if err != nil {
			return nil, fmt.Errorf("fixture %s: %w", location, err)
		}
		if payload == nil {
			return object, nil
		}
		fields = payload
	}
	for key, field := range object.Fields {
		if fields != message && key == "@type" {
			continue
		}
		descriptor := fields.Fields().ByJSONName(key)
		if descriptor == nil {
			descriptor = fields.Fields().ByTextName(key)
		}
		if descriptor == nil {
			return nil, fmt.Errorf("fixture %s: %s has no field %q", location, fields.FullName(), key)
		}
		annotated, err := b.annotateField(location+"."+key, field, descriptor)
		if err != nil {
			return nil, err
		}
		object.Fields[key] = annotated
	}
	return object, nil
}

func (b *Baseline) annotateField(location string, value any, field protoreflect.FieldDescriptor) (any, error) {
	switch {
	case field.IsMap():
		entries, ok := value.(*Object)
		if !ok || field.MapValue().Message() == nil {
			return value, nil
		}
		for key, entry := range entries.Fields {
			annotated, err := b.annotate(location+"."+key, entry, field.MapValue().Message())
			if err != nil {
				return nil, err
			}
			entries.Fields[key] = annotated
		}
		return entries, nil
	case field.Message() == nil:
		return value, nil
	case field.IsList():
		elements, ok := value.([]any)
		if !ok {
			return value, nil
		}
		for index, element := range elements {
			annotated, err := b.annotate(fmt.Sprintf("%s[%d]", location, index), element, field.Message())
			if err != nil {
				return nil, err
			}
			elements[index] = annotated
		}
		return elements, nil
	default:
		return b.annotate(location, value, field.Message())
	}
}

// anyPayload returns the snapshot descriptor of an Any payload, or nil when the payload belongs to
// another package and so carries no baseline name.
func (b *Baseline) anyPayload(object *Object) (protoreflect.MessageDescriptor, error) {
	url, ok := object.Fields["@type"].(string)
	if !ok {
		return nil, errors.New("google.protobuf.Any has no @type")
	}
	name := protoreflect.FullName(url[strings.LastIndex(url, "/")+1:])
	descriptor, err := b.files.FindDescriptorByName(name)
	if errors.Is(err, protoregistry.NotFound) {
		if strings.HasPrefix(string(name), testpilotPackagePrefix) {
			return nil, fmt.Errorf("snapshot lacks Any payload %s", name)
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("payload %s of google.protobuf.Any is not a message", name)
	}
	return message, nil
}

// snapshotResolver resolves Any payloads and extensions from the snapshot first. Only names
// outside the Testpilot package fall back to the generated registry: a Testpilot name the
// snapshot lacks is a baseline error, never a lookup of the current protocol.
type snapshotResolver struct {
	snapshot *dynamicpb.Types
}

func (r snapshotResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	found, err := r.snapshot.FindMessageByName(name)
	if !errors.Is(err, protoregistry.NotFound) || strings.HasPrefix(string(name), testpilotPackagePrefix) {
		return found, err
	}
	return protoregistry.GlobalTypes.FindMessageByName(name)
}

func (r snapshotResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return r.FindMessageByName(protoreflect.FullName(url[strings.LastIndex(url, "/")+1:]))
}

func (r snapshotResolver) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	found, err := r.snapshot.FindExtensionByName(name)
	if !errors.Is(err, protoregistry.NotFound) || strings.HasPrefix(string(name), testpilotPackagePrefix) {
		return found, err
	}
	return protoregistry.GlobalTypes.FindExtensionByName(name)
}

func (r snapshotResolver) FindExtensionByNumber(message protoreflect.FullName, number protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	found, err := r.snapshot.FindExtensionByNumber(message, number)
	if !errors.Is(err, protoregistry.NotFound) || strings.HasPrefix(string(message), testpilotPackagePrefix) {
		return found, err
	}
	return protoregistry.GlobalTypes.FindExtensionByNumber(message, number)
}

// decodeJSON decodes one JSON document into the tree steps operate on.
func decodeJSON(encoded []byte) (any, error) {
	value, err := decodePlainJSON(encoded)
	if err != nil {
		return nil, err
	}
	return objects(value), nil
}

func decodePlainJSON(encoded []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON document")
	}
	return value, nil
}

func objects(value any) any {
	switch node := value.(type) {
	case map[string]any:
		for key, field := range node {
			node[key] = objects(field)
		}
		return &Object{Fields: node}
	case []any:
		for index, element := range node {
			node[index] = objects(element)
		}
		return node
	default:
		return value
	}
}

func sortedKeys(entry map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(entry))
	for key := range entry {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// Fixture trees the generators write. Every regular file under them must have a baseline.
const (
	functionalFixtureRoot  = "tests/testcore/testpilot/testdata"
	conformanceFixtureRoot = "common/testing/testpilot/testdata/case-runtime-conformance"
)

// PairedFixtures lists the repository-relative fixture paths present in both the baseline copy and
// the regenerated trees, and fails when either side holds a fixture the other lacks, so a deleted
// or added fixture is never silently skipped.
func PairedFixtures(baselineFixtureRoot, repositoryRoot string) ([]string, error) {
	baseline, err := regularFiles(baselineFixtureRoot, ".")
	if err != nil {
		return nil, err
	}
	regenerated, err := regularFiles(repositoryRoot, conformanceFixtureRoot)
	if err != nil {
		return nil, err
	}
	functional, err := filepath.Glob(filepath.Join(repositoryRoot, filepath.FromSlash(functionalFixtureRoot), "*-case.json"))
	if err != nil {
		return nil, err
	}
	for _, match := range functional {
		relative, err := filepath.Rel(repositoryRoot, match)
		if err != nil {
			return nil, err
		}
		regenerated = append(regenerated, filepath.ToSlash(relative))
	}
	slices.Sort(regenerated)

	var unpaired []string
	for _, fixture := range baseline {
		if _, found := slices.BinarySearch(regenerated, fixture); !found {
			unpaired = append(unpaired, "baseline fixture "+fixture+" has no regenerated counterpart")
		}
	}
	for _, fixture := range regenerated {
		if _, found := slices.BinarySearch(baseline, fixture); !found {
			unpaired = append(unpaired, "regenerated fixture "+fixture+" has no baseline")
		}
	}
	if len(unpaired) > 0 {
		return nil, errors.New(strings.Join(unpaired, "; "))
	}
	if len(baseline) == 0 {
		return nil, fmt.Errorf("no baseline fixtures under %s", baselineFixtureRoot)
	}
	return baseline, nil
}

func regularFiles(root, relativeRoot string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(relativeRoot)), func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("fixture %s is not a regular file", filepath.ToSlash(relative))
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}
