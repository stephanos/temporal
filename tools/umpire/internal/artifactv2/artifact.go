// Package artifactv2 owns the exact Go reading contract for Lean-generated Umpire v2 artifacts.
package artifactv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	ExperimentFormat = "umpire-experiment/v2"
	DrivePlanFormat  = "umpire-drive-plan/v2"
)

const (
	behaviorFingerprintDomain = "umpire.behavior-fingerprint/v1"
	drivePlanChecksumDomain   = "umpire.drive-plan/v2"
	experimentChecksumDomain  = "umpire.experiment-spec/v2"
)

type Experiment struct {
	FormatVersion                       string     `json:"formatVersion"`
	QueryBehaviorFingerprint            string     `json:"queryBehaviorFingerprint"`
	Plan                                DrivePlan  `json:"plan"`
	Properties                          []Property `json:"properties"`
	ObservationRequirementDefinitionIDs []string   `json:"observationRequirementDefinitionIds"`
	Provenance                          Provenance `json:"provenance"`
	ArtifactChecksum                    string     `json:"artifactChecksum,omitempty"`
}

type DrivePlan struct {
	FormatVersion                      string         `json:"formatVersion"`
	QueryDefinitionID                  string         `json:"queryDefinitionId"`
	QueryBehaviorFingerprint           string         `json:"queryBehaviorFingerprint"`
	BehaviorDefinitionID               string         `json:"behaviorDefinitionId"`
	BehaviorFingerprint                string         `json:"behaviorFingerprint"`
	TargetDefinitionID                 string         `json:"targetDefinitionId"`
	TargetBehaviorFingerprint          string         `json:"targetBehaviorFingerprint"`
	KernelDefinitionID                 string         `json:"kernelDefinitionId"`
	KernelBehaviorFingerprint          string         `json:"kernelBehaviorFingerprint"`
	Bindings                           []Binding      `json:"bindings"`
	SymbolicRoles                      []Role         `json:"symbolicRoles"`
	ModelPreconditions                 []Precondition `json:"modelPreconditions"`
	InitialState                       ModelValue     `json:"initialState"`
	RequestedActions                   []ModelValue   `json:"requestedActions"`
	ModelOutcomes                      []ModelValue   `json:"modelOutcomes"`
	ResultingStates                    []ModelValue   `json:"resultingStates"`
	LinearExtension                    []Occurrence   `json:"linearExtension"`
	SelectedChoices                    []ModelValue   `json:"selectedChoices"`
	SelectedVariants                   []ModelValue   `json:"selectedVariants"`
	RequestedFaults                    []ModelValue   `json:"requestedFaults"`
	CapabilityRequirementDefinitionIDs []string       `json:"capabilityRequirementDefinitionIds"`
	ExpandedLimits                     Limits         `json:"expandedLimits"`
	Checkpoints                        []Checkpoint   `json:"checkpoints"`
	SelectionReason                    string         `json:"selectionReason"`
	Explored                           ExploredCounts `json:"explored"`
	KnownGaps                          []KnownGap     `json:"knownGaps"`
	Provenance                         Provenance     `json:"provenance"`
	ArtifactChecksum                   string         `json:"artifactChecksum,omitempty"`
}

type Property struct {
	DefinitionID             string   `json:"definitionId"`
	BehaviorFingerprint      string   `json:"behaviorFingerprint"`
	RequirementDefinitionIDs []string `json:"requirementDefinitionIds"`
}

type Provenance struct {
	SourceDefinitionIDs []string         `json:"sourceDefinitionIds"`
	SourceLocations     []SourceLocation `json:"sourceLocations"`
}

type SourceLocation struct {
	Path       string `json:"path"`
	Line       uint64 `json:"line"`
	Column     uint64 `json:"column"`
	Provenance string `json:"provenance"`
}

type ModelValue struct {
	DefinitionID string `json:"definitionId"`
	Value        string `json:"value"`
}

type Binding struct {
	RoleDefinitionID string     `json:"roleDefinitionId"`
	Value            ModelValue `json:"value"`
}

type Role struct {
	DefinitionID string `json:"definitionId"`
	ValueKind    string `json:"valueKind"`
}

type Precondition struct {
	DefinitionID string  `json:"definitionId"`
	Relation     string  `json:"relation"`
	Left         Operand `json:"left"`
	Right        Operand `json:"right"`
}

type Operand struct {
	Kind         string      `json:"kind"`
	DefinitionID string      `json:"definitionId,omitempty"`
	Value        *ModelValue `json:"value,omitempty"`
}

type Occurrence struct {
	DefinitionID         string  `json:"definitionId"`
	ActionDefinitionID   string  `json:"actionDefinitionId"`
	Position             uint64  `json:"position"`
	AuthoredDefinitionID *string `json:"authoredDefinitionId"`
}

type Limits struct {
	Behavior BehaviorLimits `json:"behavior"`
	Search   Limit          `json:"search"`
}

type BehaviorLimits struct {
	Transitions     Limit `json:"transitions"`
	SelectedActions Limit `json:"selectedActions"`
}

type Limit struct {
	Value uint64 `json:"value"`
	Unit  string `json:"unit"`
}

type Checkpoint struct {
	Transition   uint64       `json:"transition"`
	Observations []ModelValue `json:"observations"`
}

type ExploredCounts struct {
	Setups              uint64 `json:"setups"`
	Traces              uint64 `json:"traces"`
	Transitions         uint64 `json:"transitions"`
	PropertyEvaluations uint64 `json:"propertyEvaluations"`
}

type KnownGap struct {
	Kind    string  `json:"kind"`
	Code    string  `json:"code"`
	Subject *string `json:"subject"`
	Detail  *string `json:"detail"`
}

var canonicalKeys = map[string]string{
	"actiondefinitionid":                  "actionDefinitionId",
	"artifactchecksum":                    "artifactChecksum",
	"authoreddefinitionid":                "authoredDefinitionId",
	"behavior":                            "behavior",
	"behaviordefinitionid":                "behaviorDefinitionId",
	"behaviorfingerprint":                 "behaviorFingerprint",
	"bindings":                            "bindings",
	"capabilityrequirementdefinitionids":  "capabilityRequirementDefinitionIds",
	"checkpoints":                         "checkpoints",
	"code":                                "code",
	"column":                              "column",
	"definitionid":                        "definitionId",
	"detail":                              "detail",
	"expandedlimits":                      "expandedLimits",
	"explored":                            "explored",
	"formatversion":                       "formatVersion",
	"initialstate":                        "initialState",
	"kernelbehaviorfingerprint":           "kernelBehaviorFingerprint",
	"kerneldefinitionid":                  "kernelDefinitionId",
	"kind":                                "kind",
	"knowngaps":                           "knownGaps",
	"left":                                "left",
	"line":                                "line",
	"linearextension":                     "linearExtension",
	"modeloutcomes":                       "modelOutcomes",
	"modelpreconditions":                  "modelPreconditions",
	"observationrequirementdefinitionids": "observationRequirementDefinitionIds",
	"observations":                        "observations",
	"path":                                "path",
	"plan":                                "plan",
	"position":                            "position",
	"properties":                          "properties",
	"propertyevaluations":                 "propertyEvaluations",
	"provenance":                          "provenance",
	"querybehaviorfingerprint":            "queryBehaviorFingerprint",
	"querydefinitionid":                   "queryDefinitionId",
	"relation":                            "relation",
	"requestactions":                      "requestedActions",
	"requestedactions":                    "requestedActions",
	"requestedfaults":                     "requestedFaults",
	"requirementdefinitionids":            "requirementDefinitionIds",
	"resultingstates":                     "resultingStates",
	"right":                               "right",
	"roledefinitionid":                    "roleDefinitionId",
	"search":                              "search",
	"selectedactions":                     "selectedActions",
	"selectedchoices":                     "selectedChoices",
	"selectedvariants":                    "selectedVariants",
	"selectionreason":                     "selectionReason",
	"setups":                              "setups",
	"sourcedefinitionids":                 "sourceDefinitionIds",
	"sourcelocations":                     "sourceLocations",
	"subject":                             "subject",
	"symbolicroles":                       "symbolicRoles",
	"targetbehaviorfingerprint":           "targetBehaviorFingerprint",
	"targetdefinitionid":                  "targetDefinitionId",
	"traces":                              "traces",
	"transition":                          "transition",
	"transitions":                         "transitions",
	"unit":                                "unit",
	"value":                               "value",
	"valuekind":                           "valueKind",
}

// DecodeExperiment accepts only the exact canonical bytes emitted by Lean.
func DecodeExperiment(encoded []byte) (Experiment, error) {
	if len(bytes.TrimSpace(encoded)) == 0 {
		return Experiment{}, errors.New("canonical ExperimentSpec JSON is empty")
	}
	format, err := preflightFormat(encoded)
	if err != nil {
		return Experiment{}, err
	}
	if format != ExperimentFormat {
		return Experiment{}, fmt.Errorf("unsupported format %q", format)
	}
	if err := validateJSONStructure(encoded); err != nil {
		return Experiment{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var document Experiment
	if err := decoder.Decode(&document); err != nil {
		return Experiment{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return Experiment{}, err
	}
	if err := validateExperiment(document); err != nil {
		return Experiment{}, err
	}
	if err := verifyChecksums(document); err != nil {
		return Experiment{}, err
	}
	canonical, err := CanonicalExperimentBytes(document)
	if err != nil {
		return Experiment{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return Experiment{}, errors.New("ExperimentSpec is not canonical v2 bytes")
	}
	return document, nil
}

func CanonicalExperimentBytes(document Experiment) ([]byte, error) {
	return encodeJSONLine(document)
}

func ExpectedDrivePlanChecksum(plan DrivePlan) (string, error) {
	plan.ArtifactChecksum = ""
	encoded, err := encodeJSONObject(plan)
	if err != nil {
		return "", err
	}
	return derive(drivePlanChecksumDomain, encoded), nil
}

func ExpectedExperimentChecksum(document Experiment) (string, error) {
	document.ArtifactChecksum = ""
	encoded, err := encodeJSONObject(document)
	if err != nil {
		return "", err
	}
	return derive(experimentChecksumDomain, encoded), nil
}

func BehaviorFingerprint(canonical []byte) string {
	return derive(behaviorFingerprintDomain, canonical)
}

func SealExperiment(document Experiment) (Experiment, error) {
	planChecksum, err := ExpectedDrivePlanChecksum(document.Plan)
	if err != nil {
		return Experiment{}, err
	}
	document.Plan.ArtifactChecksum = planChecksum
	experimentChecksum, err := ExpectedExperimentChecksum(document)
	if err != nil {
		return Experiment{}, err
	}
	document.ArtifactChecksum = experimentChecksum
	return document, nil
}

func preflightFormat(encoded []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	first, err := decoder.Token()
	if err != nil {
		return "", err
	}
	if first != json.Delim('{') {
		return "", errors.New("ExperimentSpec must be a JSON object")
	}
	seen := make(map[string]string)
	format := ""
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return "", err
		}
		key, ok := keyToken.(string)
		if !ok {
			return "", errors.New("ExperimentSpec object key is not a string")
		}
		folded := strings.ToLower(key)
		if previous, duplicate := seen[folded]; duplicate {
			return "", fmt.Errorf("duplicate or case-colliding top-level key %q and %q", previous, key)
		}
		seen[folded] = key
		if folded == "formatversion" && key != "formatVersion" {
			return "", fmt.Errorf("JSON object key %q must be spelled %q", key, "formatVersion")
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return "", err
		}
		if key == "formatVersion" {
			if err := json.Unmarshal(raw, &format); err != nil {
				return "", errors.New("formatVersion must be a string")
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return "", err
	}
	if err := requireEOF(decoder); err != nil {
		return "", err
	}
	if format == "" {
		return "", errors.New("formatVersion is required")
	}
	return format, nil
}

func validateJSONStructure(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return err
	}
	if err := validateJSONValue(decoder, first); err != nil {
		return err
	}
	return requireEOF(decoder)
}

func validateJSONValue(decoder *json.Decoder, token json.Token) error {
	delimiter, structured := token.(json.Delim)
	if !structured {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]string)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key has type %T", keyToken)
			}
			folded := strings.ToLower(key)
			if previous, duplicate := seen[folded]; duplicate {
				return fmt.Errorf("duplicate or case-colliding JSON object key %q and %q", previous, key)
			}
			seen[folded] = key
			if canonical, known := canonicalKeys[folded]; known && key != canonical {
				return fmt.Errorf("JSON object key %q must be spelled %q", key, canonical)
			}
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := validateJSONValue(decoder, value); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("unexpected JSON object delimiter %q", closing)
		}
	case '[':
		for decoder.More() {
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := validateJSONValue(decoder, value); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("unexpected JSON array delimiter %q", closing)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err == nil {
		return errors.New("trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func validateExperiment(document Experiment) error {
	if document.FormatVersion != ExperimentFormat {
		return fmt.Errorf("unsupported format %q", document.FormatVersion)
	}
	if document.Plan.FormatVersion != DrivePlanFormat {
		return fmt.Errorf("unsupported nested plan format %q", document.Plan.FormatVersion)
	}
	for label, value := range map[string]string{
		"query behavior fingerprint":       document.QueryBehaviorFingerprint,
		"plan query definition ID":         document.Plan.QueryDefinitionID,
		"plan query behavior fingerprint":  document.Plan.QueryBehaviorFingerprint,
		"plan behavior definition ID":      document.Plan.BehaviorDefinitionID,
		"plan behavior fingerprint":        document.Plan.BehaviorFingerprint,
		"plan target definition ID":        document.Plan.TargetDefinitionID,
		"plan target behavior fingerprint": document.Plan.TargetBehaviorFingerprint,
		"plan kernel definition ID":        document.Plan.KernelDefinitionID,
		"plan kernel behavior fingerprint": document.Plan.KernelBehaviorFingerprint,
		"plan selection reason":            document.Plan.SelectionReason,
		"plan initial-state definition ID": document.Plan.InitialState.DefinitionID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	for label, value := range map[string]string{
		"query behavior fingerprint":       document.QueryBehaviorFingerprint,
		"plan query behavior fingerprint":  document.Plan.QueryBehaviorFingerprint,
		"plan behavior fingerprint":        document.Plan.BehaviorFingerprint,
		"plan target behavior fingerprint": document.Plan.TargetBehaviorFingerprint,
		"plan kernel behavior fingerprint": document.Plan.KernelBehaviorFingerprint,
	} {
		if !ValidDigest(value) {
			return fmt.Errorf("%s %q is invalid", label, value)
		}
	}
	if !ValidDigest(document.Plan.ArtifactChecksum) {
		return fmt.Errorf("nested plan artifact checksum %q is invalid", document.Plan.ArtifactChecksum)
	}
	if !ValidDigest(document.ArtifactChecksum) {
		return fmt.Errorf("artifact checksum %q is invalid", document.ArtifactChecksum)
	}
	if document.QueryBehaviorFingerprint != document.Plan.QueryBehaviorFingerprint {
		return errors.New("query behavior fingerprint differs from nested plan")
	}
	if document.Properties == nil || document.ObservationRequirementDefinitionIDs == nil ||
		document.Provenance.SourceDefinitionIDs == nil || document.Provenance.SourceLocations == nil {
		return errors.New("ExperimentSpec arrays must not be null")
	}
	if err := validateStringSet("observation requirement definition ID", document.ObservationRequirementDefinitionIDs); err != nil {
		return err
	}
	if err := validateProvenance(document.Provenance); err != nil {
		return err
	}
	for _, property := range document.Properties {
		if strings.TrimSpace(property.DefinitionID) == "" || !ValidDigest(property.BehaviorFingerprint) {
			return errors.New("property has malformed definition ID or behavior fingerprint")
		}
		if err := validateStringSet("property requirement definition ID", property.RequirementDefinitionIDs); err != nil {
			return err
		}
	}
	if !slices.IsSortedFunc(document.Properties, func(left, right Property) int {
		return strings.Compare(left.DefinitionID, right.DefinitionID)
	}) {
		return errors.New("properties are not in canonical order")
	}
	if err := validateDrivePlan(document.Plan); err != nil {
		return err
	}
	return nil
}

func validateDrivePlan(plan DrivePlan) error {
	if plan.Bindings == nil || plan.SymbolicRoles == nil || plan.ModelPreconditions == nil ||
		plan.RequestedActions == nil || plan.ModelOutcomes == nil || plan.ResultingStates == nil ||
		plan.LinearExtension == nil || plan.SelectedChoices == nil || plan.SelectedVariants == nil ||
		plan.RequestedFaults == nil || plan.CapabilityRequirementDefinitionIDs == nil ||
		plan.Checkpoints == nil || plan.KnownGaps == nil || plan.Provenance.SourceDefinitionIDs == nil ||
		plan.Provenance.SourceLocations == nil {
		return errors.New("DrivePlan arrays must not be null")
	}
	if err := validateStringSet("capability requirement definition ID", plan.CapabilityRequirementDefinitionIDs); err != nil {
		return err
	}
	if err := validateProvenance(plan.Provenance); err != nil {
		return err
	}
	for _, gap := range plan.KnownGaps {
		switch gap.Kind {
		case "capability-contract", "input", "interpretation", "claim":
		default:
			return fmt.Errorf("known gap kind %q is invalid", gap.Kind)
		}
		if strings.TrimSpace(gap.Code) == "" {
			return errors.New("known gap code is required")
		}
	}
	if !slices.IsSortedFunc(plan.KnownGaps, compareKnownGap) {
		return errors.New("known gaps are not in canonical order")
	}
	for index := 1; index < len(plan.KnownGaps); index++ {
		previous, current := plan.KnownGaps[index-1], plan.KnownGaps[index]
		if previous.Kind == current.Kind && previous.Code == current.Code && pointerValue(previous.Subject) == pointerValue(current.Subject) {
			return fmt.Errorf("duplicate or conflicting known gap %q", current.Code)
		}
	}
	return nil
}

func validateProvenance(provenance Provenance) error {
	if err := validateStringSet("source definition ID", provenance.SourceDefinitionIDs); err != nil {
		return err
	}
	if len(provenance.SourceLocations) == 0 {
		return errors.New("at least one source location is required")
	}
	if !slices.IsSortedFunc(provenance.SourceLocations, compareSourceLocation) {
		return errors.New("source locations are not in canonical order")
	}
	for index, source := range provenance.SourceLocations {
		if strings.TrimSpace(source.Path) == "" || strings.TrimSpace(source.Provenance) == "" || source.Line == 0 || source.Column == 0 {
			return errors.New("source location is malformed")
		}
		if index > 0 && source == provenance.SourceLocations[index-1] {
			return fmt.Errorf("duplicate source location %q", source.Path)
		}
	}
	return nil
}

func validateStringSet(label string, values []string) error {
	if !slices.IsSorted(values) {
		return fmt.Errorf("%ss are not in canonical order", label)
	}
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is empty", label)
		}
		if index > 0 && value == values[index-1] {
			return fmt.Errorf("duplicate %s %q", label, value)
		}
	}
	return nil
}

func compareKnownGap(left, right KnownGap) int {
	for _, comparison := range []int{
		compareInt(knownGapKindRank(left.Kind), knownGapKindRank(right.Kind)),
		strings.Compare(left.Code, right.Code),
		strings.Compare(pointerValue(left.Subject), pointerValue(right.Subject)),
		strings.Compare(pointerValue(left.Detail), pointerValue(right.Detail)),
	} {
		if comparison != 0 {
			return comparison
		}
	}
	return 0
}

func knownGapKindRank(kind string) int {
	switch kind {
	case "capability-contract":
		return 0
	case "input":
		return 1
	case "interpretation":
		return 2
	case "claim":
		return 3
	default:
		return 4
	}
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareSourceLocation(left, right SourceLocation) int {
	if comparison := strings.Compare(left.Path, right.Path); comparison != 0 {
		return comparison
	}
	if left.Line != right.Line {
		if left.Line < right.Line {
			return -1
		}
		return 1
	}
	if left.Column != right.Column {
		if left.Column < right.Column {
			return -1
		}
		return 1
	}
	return strings.Compare(left.Provenance, right.Provenance)
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func verifyChecksums(document Experiment) error {
	nested, err := ExpectedDrivePlanChecksum(document.Plan)
	if err != nil {
		return err
	}
	if nested != document.Plan.ArtifactChecksum {
		return fmt.Errorf("nested plan artifact checksum mismatch: got %q, want %q", document.Plan.ArtifactChecksum, nested)
	}
	outer, err := ExpectedExperimentChecksum(document)
	if err != nil {
		return err
	}
	if outer != document.ArtifactChecksum {
		return fmt.Errorf("ExperimentSpec artifact checksum mismatch: got %q, want %q", document.ArtifactChecksum, outer)
	}
	return nil
}

func ValidDigest(value string) bool {
	const prefix = "sha256:"
	if len(value) != len(prefix)+sha256.Size*2 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, character := range value[len(prefix):] {
		if !('0' <= character && character <= '9') && !('a' <= character && character <= 'f') {
			return false
		}
	}
	return true
}

func encodeJSONObject(value any) ([]byte, error) {
	line, err := encodeJSONLine(value)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(line, []byte{'\n'}), nil
}

func encodeJSONLine(value any) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func derive(domain string, canonical []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write(canonical)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}
