package lower

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	runtime "go.temporal.io/server/common/testing/testpilot"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ExpectedClaim is one claim's expected conclusion: its status and, unless it is satisfied, the
// judge's reason id.
type ExpectedClaim struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ExpectedRun is what a Query's expected Run declares, each value the IR enum value's id
// (umpiremodel.ExpectationID): the Contract's Verdict, the Run's disposition and cleanup, the model
// assessment's conformance, and the selected Property's and each monitor's conclusion.
type ExpectedRun struct {
	Contract    string          `json:"contract"`
	Disposition string          `json:"disposition"`
	Cleanup     string          `json:"cleanup"`
	Conformance string          `json:"conformance"`
	Properties  []ExpectedClaim `json:"properties"`
}

type GeneratedCase struct {
	Model       string               `json:"model"`
	Query       umpiremodel.ClaimKey `json:"query"`
	Standing    Standing             `json:"standing"`
	File        string               `json:"file,omitempty"`
	Expected    *ExpectedRun         `json:"expected,omitempty"`
	Unsupported []Unsupported        `json:"unsupported,omitempty"`
}

// GenerateCases accounts for every Query, including those with no executable realization.
func GenerateCases(irDirectory string) (map[string][]byte, error) {
	return generateCases(irDirectory, func(_ string, m *umpirespb.Model) (*Producer, error) { return NewProducer(m) })
}

// generateCases is GenerateCases with the Producer of each loaded IR file made by produce.
func generateCases(irDirectory string, produce func(path string, m *umpirespb.Model) (*Producer, error)) (map[string][]byte, error) {
	paths, err := umpiremodel.IRPaths(irDirectory)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no IR Models in %s", irDirectory)
	}
	files := map[string][]byte{}
	manifest := Manifest{Version: 1}
	for _, path := range paths {
		model, err := umpiremodel.Load(path)
		if err != nil {
			return nil, err
		}
		producer, err := produce(path, model)
		if err != nil {
			return nil, err
		}
		queries := slices.Clone(model.GetQueries())
		slices.SortFunc(queries, func(a, b *umpirespb.Query) int { return strings.Compare(a.GetName(), b.GetName()) })
		for _, query := range queries {
			entry, encoded, err := generateCase(producer, filepath.Base(path), query)
			if err != nil {
				return nil, err
			}
			if entry.Standing == Lowered {
				if _, exists := files[entry.File]; exists {
					return nil, fmt.Errorf("duplicate generated Case filename %s", entry.File)
				}
				files[entry.File] = encoded
			}
			manifest.Queries = append(manifest.Queries, entry)
		}
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	files["manifest.json"] = append(encoded, '\n')
	return files, nil
}

func generateCase(producer *Producer, model string, query *umpirespb.Query) (GeneratedCase, []byte, error) {
	name := strings.TrimSuffix(model, ".json") + "-" + query.GetName() + "-case.json"
	if !bareJSON(name) {
		return GeneratedCase{}, nil, fmt.Errorf("invalid Case filename %q", name)
	}
	lowered, err := producer.Lower(query.GetName(), cp.IdentityFor("temporal.case", "scala."+strings.TrimSuffix(model, ".json"), query.GetName()))
	if err != nil {
		return GeneratedCase{}, nil, err
	}
	entry := GeneratedCase{Model: model, Query: producer.found[query.GetName()].Key, Standing: lowered.Standing, Unsupported: lowered.Unsupported}
	if lowered.Standing != Lowered {
		return entry, nil, nil
	}
	expected := query.GetExpectedRun()
	if expected == nil {
		return GeneratedCase{}, nil, fmt.Errorf("%s:%d: lowerable Query %s declares no expected Run assessment", query.GetPosition().GetFile(), query.GetPosition().GetLine(), query.GetName())
	}
	entry.File = name
	id := umpiremodel.ExpectationID
	entry.Expected = &ExpectedRun{
		Contract:    id(expected.GetContract()),
		Disposition: id(expected.GetDisposition()),
		Cleanup:     id(expected.GetCleanup()),
		Conformance: id(expected.GetConformance()),
		Properties:  []ExpectedClaim{{ID: query.GetProperty().GetName(), Status: id(expected.GetProperty()), Reason: id(expected.GetReason())}},
	}
	for _, monitor := range expected.GetMonitors() {
		entry.Expected.Properties = append(entry.Expected.Properties, ExpectedClaim{ID: monitor.GetName(), Status: id(monitor.GetOutcome()), Reason: id(monitor.GetReason())})
	}
	if err := validateExpectedRun(entry.Expected, fmt.Sprintf("%s:%d: Query %s", query.GetPosition().GetFile(), query.GetPosition().GetLine(), query.GetName())); err != nil {
		return GeneratedCase{}, nil, err
	}
	encoded, err := protojson.Marshal(lowered.Case)
	if err != nil {
		return GeneratedCase{}, nil, err
	}
	var canonical bytes.Buffer
	if err := json.Compact(&canonical, encoded); err != nil {
		return GeneratedCase{}, nil, err
	}
	return entry, append(canonical.Bytes(), '\n'), nil
}

// Selected names one Query of one checked IR file.
type Selected struct {
	Model string
	Query string
}

// SelectCases is the part of a complete generated tree that a consumer pins: the Cases of the
// selected Queries, byte for byte, under a manifest that lists only them in the complete tree's
// order. A Query that did not lower has no Case to pin, so it is refused with what it lacks rather
// than left out.
func SelectCases(files map[string][]byte, selected []Selected) (map[string][]byte, error) {
	manifest, err := DecodeManifest(files["manifest.json"])
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, errors.New("no Query selected")
	}
	wanted := map[Selected]bool{}
	for _, query := range selected {
		if wanted[query] {
			return nil, fmt.Errorf("selected Query %s/%s is named twice", query.Model, query.Query)
		}
		wanted[query] = true
	}
	subset := Manifest{Version: manifest.Version}
	result := map[string][]byte{}
	for _, entry := range manifest.Queries {
		query := Selected{Model: entry.Model, Query: entry.Query.Name}
		if !wanted[query] {
			continue
		}
		delete(wanted, query)
		if entry.Standing != Lowered {
			return nil, fmt.Errorf("selected Query %s/%s has no Case: %s %v", query.Model, query.Query, entry.Standing, entry.Unsupported)
		}
		encoded, exists := files[entry.File]
		if !exists {
			return nil, fmt.Errorf("generated Case %s is missing", entry.File)
		}
		result[entry.File] = encoded
		subset.Queries = append(subset.Queries, entry)
	}
	for _, query := range selected {
		if wanted[query] {
			return nil, fmt.Errorf("no Model declares the selected Query %s/%s", query.Model, query.Query)
		}
	}
	encoded, err := json.MarshalIndent(subset, "", "  ")
	if err != nil {
		return nil, err
	}
	result["manifest.json"] = append(encoded, '\n')
	return result, nil
}

type Manifest struct {
	Version int             `json:"version"`
	Queries []GeneratedCase `json:"queries"`
}

func DecodeManifest(encoded []byte) (*Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("manifest has trailing data")
	}
	if manifest.Version != 1 || len(manifest.Queries) == 0 {
		return nil, fmt.Errorf("unsupported or empty Case manifest version %d", manifest.Version)
	}
	queries, files := map[string]bool{}, map[string]bool{}
	for _, entry := range manifest.Queries {
		key := entry.Model + "/" + entry.Query.Name
		if !bareJSON(entry.Model) || entry.Query.Name == "" || entry.Query.Owner == "" || entry.Query.Family == "" || queries[key] {
			return nil, fmt.Errorf("invalid or duplicate manifest Query %s", key)
		}
		queries[key] = true
		switch entry.Standing {
		case Lowered:
			if !bareJSON(entry.File) || entry.File == "manifest.json" || files[entry.File] || entry.Expected == nil || len(entry.Unsupported) != 0 {
				return nil, fmt.Errorf("invalid lowered Case %s", key)
			}
			files[entry.File] = true
			if err := validateExpectedRun(entry.Expected, key); err != nil {
				return nil, err
			}
		case NothingToRealize, NoRealization, NotSupported:
			if entry.File != "" || entry.Expected != nil || (entry.Standing == NotSupported) != (len(entry.Unsupported) > 0) {
				return nil, fmt.Errorf("invalid non-lowered Query %s", key)
			}
		default:
			return nil, fmt.Errorf("unknown standing %q for %s", entry.Standing, key)
		}
	}
	return &manifest, nil
}

func validateExpectedRun(expected *ExpectedRun, key string) error {
	verdict, known := testpilotValue[testpilotspb.VerdictStatus](testpilotspb.VerdictStatus_value, "VERDICT_STATUS_", expected.Contract)
	if !known || (verdict != testpilotspb.VERDICT_STATUS_SATISFIED && verdict != testpilotspb.VERDICT_STATUS_VIOLATED) {
		return fmt.Errorf("invalid expected Contract for %s", key)
	}
	disposition, known := testpilotValue[testpilotspb.RunDisposition](testpilotspb.RunDisposition_value, "RUN_DISPOSITION_", expected.Disposition)
	if !known {
		return fmt.Errorf("invalid expected disposition for %s", key)
	}
	if !concludable(verdict, disposition) {
		return fmt.Errorf("%s expects a %s Contract on a Run %s, which no Run's rules conclude", key, expected.Contract, expected.Disposition)
	}
	if _, known := testpilotValue[testpilotspb.CleanupStatus](testpilotspb.CleanupStatus_value, "CLEANUP_STATUS_", expected.Cleanup); !known {
		return fmt.Errorf("invalid expected cleanup for %s", key)
	}
	if expected.Conformance != "conformant" && expected.Conformance != "nonconformant" && expected.Conformance != "inconclusive" {
		return fmt.Errorf("invalid expected conformance for %s", key)
	}
	claims := map[string]bool{}
	if len(expected.Properties) == 0 {
		return fmt.Errorf("missing expected claims for %s", key)
	}
	for _, claim := range expected.Properties {
		// A reason is an id exactly as ExpectationID spells one, the one spelling rule.
		reason, named := umpirespb.RunExpectation_Reason_value["REASON_"+strings.ToUpper(claim.Reason)]
		spelled := named && reason != 0 && umpiremodel.ExpectationID(umpirespb.RunExpectation_Reason(reason)) == claim.Reason
		if claim.ID == "" || claims[claim.ID] || (claim.Status != "satisfied" && claim.Status != "violated" && claim.Status != "inconclusive") ||
			(claim.Reason != "" && !spelled) || (claim.Status == "satisfied") != (claim.Reason == "") {
			return fmt.Errorf("invalid expected claim for %s", key)
		}
		claims[claim.ID] = true
	}
	return nil
}

// testpilotValue is the Testpilot enum value an expected Run's id names, by the enum's prefix, and
// whether it names one other than the unspecified.
func testpilotValue[E interface {
	~int32
	protoreflect.Enum
}](values map[string]int32, prefix, id string) (E, bool) {
	n, ok := values[prefix+strings.ToUpper(id)]
	return E(n), ok && n != 0 && umpiremodel.EnumID(E(n), prefix) == id
}

// concludable reports whether testpilot.ConcludeVerdict concludes verdict and leaves a Run in
// disposition for some Run it closes and some rules, so an expectation is held to the judge's own
// aggregation rather than to a copy of it.
func concludable(verdict testpilotspb.VerdictStatus, disposition testpilotspb.RunDisposition) bool {
	rules := [][]testpilotspb.RuleVerdictStatus{nil, {testpilotspb.RULE_VERDICT_STATUS_SATISFIED}, {testpilotspb.RULE_VERDICT_STATUS_VIOLATED}, {testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE}}
	for closed := range testpilotspb.RunDisposition_name {
		for _, statuses := range rules {
			if v, d := runtime.ConcludeVerdict(testpilotspb.RunDisposition(closed), statuses); v == verdict && d == disposition {
				return true
			}
		}
	}
	return false
}

// Check compares a closed Run, its Verdict and its Assessment with what the Query declared, each by
// equality: the Run's disposition and cleanup, the Contract's Verdict, the conformance, and every
// claim's status and reason id. The Assessment's prose is shown, never compared. Conformance is
// compared by its status only: an expected Run declares no conformance reason (fn-124.6).
func (e *ExpectedRun) Check(run *testpilotspb.Run, verdict *testpilotspb.Verdict, assessment *runtime.Assessment) error {
	if run == nil || verdict == nil || assessment == nil {
		return errors.New("a Run, its Verdict and its Assessment are required")
	}
	var problems []error
	differs := func(what, want, got, detail string) {
		switch {
		case want == got:
		case detail == "":
			problems = append(problems, fmt.Errorf("%s is %s, expected %s", what, got, want))
		default:
			problems = append(problems, fmt.Errorf("%s is %s, expected %s: %s", what, got, want, detail))
		}
	}
	diagnostics := ""
	if len(run.GetDiagnostics()) > 0 {
		diagnostics = fmt.Sprint(run.GetDiagnostics())
	}
	differs("the disposition", e.Disposition, testpilotID(run.GetDisposition(), "RUN_DISPOSITION_"), diagnostics)
	differs("the cleanup", e.Cleanup, testpilotID(run.GetCleanup().GetStatus(), "CLEANUP_STATUS_"), diagnostics)
	differs("the Contract's Verdict", e.Contract, testpilotID(verdict.GetStatus(), "VERDICT_STATUS_"), diagnostics)
	if failure := assessment.Failure; failure != nil {
		problems = append(problems, fmt.Errorf("the assessment failed: %s %s", failure.Code, failure.Detail))
	}
	differs("the conformance", e.Conformance, string(assessment.Conformance.Status), assessment.Conformance.Detail)
	if len(assessment.Properties) != len(e.Properties) {
		problems = append(problems, fmt.Errorf("the assessment concludes %d claims, expected %d", len(assessment.Properties), len(e.Properties)))
	}
	for _, expected := range e.Properties {
		at := slices.IndexFunc(assessment.Properties, func(actual runtime.PropertyAssessment) bool { return actual.ID == expected.ID })
		if at < 0 {
			problems = append(problems, fmt.Errorf("the assessment omits %s", expected.ID))
			continue
		}
		actual := assessment.Properties[at]
		differs(expected.ID+"'s status", expected.Status, string(actual.Status), actual.Detail)
		differs(expected.ID+"'s reason", expected.Reason, actual.Reason, actual.Detail)
	}
	return errors.Join(problems...)
}

// testpilotID is a Testpilot enum value as an expected Run names it (umpiremodel.EnumID).
func testpilotID(value protoreflect.Enum, prefix string) string {
	return umpiremodel.EnumID(value, prefix)
}

func bareJSON(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.ContainsAny(name, `/\\`) && strings.HasSuffix(name, ".json")
}

func validateCaseFiles(manifest *Manifest, files map[string][]byte) error {
	expected := map[string]bool{"manifest.json": true}
	for _, entry := range manifest.Queries {
		if entry.Standing != Lowered {
			continue
		}
		expected[entry.File] = true
	}
	if len(expected) != len(files) {
		return errors.New("case files differ from manifest inventory")
	}
	for name := range files {
		if !expected[name] {
			return fmt.Errorf("unexpected generated file %s", name)
		}
	}
	return nil
}

// SyncCases validates a complete temporary tree before checking or replacing the managed directory.
func SyncCases(directory string, files map[string][]byte, update bool) (result error) {
	manifest, err := DecodeManifest(files["manifest.json"])
	if err != nil {
		return err
	}
	if err := validateCaseFiles(manifest, files); err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(directory))
	if err != nil {
		return err
	}
	directory = filepath.Join(parent, filepath.Base(directory))
	lock := directory + ".lock"
	if err := os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("case publication lock: %w", err)
	}
	defer func() { result = errors.Join(result, os.Remove(lock)) }()
	if info, err := os.Lstat(directory); err == nil && !info.IsDir() {
		return fmt.Errorf("case destination must be a directory, not %s", info.Mode())
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".cases-*")
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, os.RemoveAll(temporary)) }()
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := os.WriteFile(filepath.Join(temporary, name), files[name], 0644); err != nil {
			return err
		}
	}
	if err := os.Chmod(temporary, 0755); err != nil {
		return err
	}
	if err := validateCaseTree(temporary, manifest); err != nil {
		return err
	}
	if !update {
		return checkCaseTree(directory, files)
	}
	return publishCaseTree(directory, temporary)
}

func validateCaseTree(directory string, manifest *Manifest) error {
	staged, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}
	if _, err := DecodeManifest(staged); err != nil {
		return err
	}
	for _, entry := range manifest.Queries {
		if entry.Standing != Lowered {
			continue
		}
		encoded, err := os.ReadFile(filepath.Join(directory, entry.File))
		if err != nil {
			return err
		}
		if _, err := runtime.DecodeCaseProtoJSON(encoded); err != nil {
			return fmt.Errorf("%s: %w", entry.File, err)
		}
	}
	return nil
}

func checkCaseTree(directory string, files map[string][]byte) error {
	existing, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	if len(existing) != len(files) {
		return errors.New("generated Case inventory is stale")
	}
	for _, file := range existing {
		if !file.Type().IsRegular() || files[file.Name()] == nil {
			return fmt.Errorf("unexpected generated Case entry %s", file.Name())
		}
		actual, err := os.ReadFile(filepath.Join(directory, file.Name()))
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, files[file.Name()]) {
			return fmt.Errorf("%s is stale", file.Name())
		}
	}
	return nil
}

func publishCaseTree(directory, temporary string) error {
	backup := directory + ".previous"
	if _, err := os.Lstat(backup); err == nil {
		return fmt.Errorf("previous Case publication requires recovery: %s", backup)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	moved := false
	if err := os.Rename(directory, backup); err == nil {
		moved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(temporary, directory); err != nil {
		if moved {
			err = errors.Join(err, os.Rename(backup, directory))
		}
		return err
	}
	if moved {
		return os.RemoveAll(backup)
	}
	return nil
}
