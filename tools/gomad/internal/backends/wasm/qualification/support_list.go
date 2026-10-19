package qualification

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

const SupportListSchema = "gomad.wasm-support-list/v1"

// SupportListStatement is part of every list: the list is per test and bound
// to one candidate, never a package, suite or cross-host claim.
const SupportListStatement = "Per-test support on exactly this candidate. Only the listed test selections passed every WASM qualification cell gate with the candidate identities recorded here. Every test that is not listed is unsupported on this candidate. This is not a functional-suite, package-level or cross-host claim, and a new candidate starts from an empty list."

// Gate names a support row must carry, in this order, each with exit 0.
const (
	SupportGatePreparation      = "unchanged-source-preparation"
	SupportGateMeasuredSuccess  = "measured-pass-replay-success"
	SupportGateMeasuredCanary   = "measured-pass-replay-canary"
	SupportGateStorageTransport = "storage-transport-inspection"
)

var supportGateOrder = []string{SupportGatePreparation, SupportGateMeasuredSuccess, SupportGateMeasuredCanary, SupportGateStorageTransport}

const supportCanaryRun = "["

type supportSelection struct{ Package, Function, Run string }

// supportCells is the closed set of qualification cells that can produce a
// support row, keyed by Manifest.Workload/Manifest.ID. Adding a test is a
// reviewed source change that adds a cell; a sweep pass cannot add a row.
var supportCells = map[string]supportSelection{
	"onebox/unchanged-workflow-type-smoke": {Package: "./tests", Function: "TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds", Run: "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
	"onebox/frontend-system-info":          {Package: "./tests/gomadfunctional", Function: "TestFrontendSystemInfo", Run: "^TestFrontendSystemInfo$"},
}

// SupportCellInput is one retained measured cell: a content-addressed Result
// and its terminal publication receipt. It has no field for sweep or triage
// outcomes; only qualification Results can become support evidence.
type SupportCellInput struct {
	ResultPath string
	Result     Result
	Receipt    PublicationReceipt
}

// SupportGate is the recorded numeric exit of one cell gate command.
type SupportGate struct {
	Cell    string // Manifest.Workload + "/" + Manifest.ID
	Name    string
	Command string
	Exit    int
	Log     string
}

type SupportCandidate struct {
	IntegratedSourceSHA256, CompilerSHA256, RuntimeSHA256 record.SHA256
	HelperSHA256, EngineSHA256, ModelSHA256               record.SHA256
	HostOS, HostArch                                      string
}

type SupportResultReference struct {
	Path           string
	ResultSHA256   record.SHA256
	Executions     uint64
	MatchedReplays uint64
	ReplayableBugs uint64
}

type SupportGateExit struct {
	Name    string
	Command string
	Exit    int
	Log     string
}

type SupportRow struct {
	Package, Function, Run                 string
	Workload, CellID, Profile              string
	BuildTags                              []string
	ModuleSHA256                           record.SHA256
	ModuleBytes                            uint64
	WorkloadSourceSHA256, HostConfigSHA256 record.SHA256
	Seed, Repeat, Fuel, MemoryBytes        uint64
	Success, Canary                        SupportResultReference
	Gates                                  []SupportGateExit
}

type SupportList struct {
	Schema    string
	Statement string
	Candidate SupportCandidate
	Rows      []SupportRow
}

type supportPair struct{ success, canary *SupportCellInput }

// BuildSupportList accepts only validated, final, measured success and canary
// cells of known support cells, all from one candidate, plus a zero exit for
// every required gate of every row. Anything else is refused, not dropped.
func BuildSupportList(cells []SupportCellInput, gates []SupportGate) (SupportList, error) {
	if len(cells) == 0 {
		return SupportList{}, errors.New("support list requires qualified cells; an empty candidate list is not produced")
	}
	pairs := map[string]*supportPair{}
	processes := map[uint64]bool{}
	var candidate *SupportCandidate
	for index := range cells {
		cell := &cells[index]
		m := cell.Result.Manifest
		key := m.Workload + "/" + m.ID
		selection, known := supportCells[key]
		if !known {
			return SupportList{}, fmt.Errorf("unknown support cell %s", key)
		}
		if m.Provisional {
			return SupportList{}, fmt.Errorf("support list cannot accept provisional source binding for %s", key)
		}
		if !filepath.IsAbs(cell.ResultPath) || filepath.Clean(cell.ResultPath) != cell.ResultPath {
			return SupportList{}, errors.New("support cell requires an absolute retained Result path")
		}
		if err := ValidateResult(cell.Result); err != nil {
			return SupportList{}, fmt.Errorf("support cell %s/%s: %w", key, m.Cell, err)
		}
		if _, err := ValidatePublicationReceipt(cell.ResultPath, cell.Result, cell.Receipt); err != nil {
			return SupportList{}, fmt.Errorf("support cell %s/%s: %w", key, m.Cell, err)
		}
		if processes[cell.Result.OperatorProcessID] {
			return SupportList{}, errors.New("support cells require distinct fresh operator processes")
		}
		processes[cell.Result.OperatorProcessID] = true
		if cell.Result.PreparationReusedWithinProducer {
			return SupportList{}, errors.New("support cells cannot import another cell's preparation metrics")
		}
		if err := validateMeasuredCell(cell.Result, key+"/"+m.Profile+"/"+m.Cell); err != nil {
			return SupportList{}, err
		}
		if uint64(len(cell.Result.Qualification.Executions)) != m.Repeat {
			return SupportList{}, fmt.Errorf("support cell %s/%s retains fewer executions than its repeat", key, m.Cell)
		}
		run := selection.Run
		if m.Cell == "canary" {
			run = supportCanaryRun
		}
		target, err := supportResultTarget(cell.Result)
		if err != nil {
			return SupportList{}, fmt.Errorf("support cell %s/%s: %w", key, m.Cell, err)
		}
		if target.Source != selection.Package || !slices.Contains(target.Argv, "-test.run="+run) {
			return SupportList{}, fmt.Errorf("support cell %s/%s did not run %s -test.run=%s", key, m.Cell, selection.Package, run)
		}
		identity := SupportCandidate{IntegratedSourceSHA256: m.IntegratedSourceSHA256, CompilerSHA256: m.CompilerSHA256, RuntimeSHA256: m.RuntimeSHA256, HelperSHA256: m.HelperSHA256, EngineSHA256: m.EngineSHA256, ModelSHA256: m.ModelSHA256, HostOS: m.HostOS, HostArch: m.HostArch}
		if candidate == nil {
			candidate = &identity
		} else if *candidate != identity {
			return SupportList{}, errors.New("support cells are not from one candidate source/compiler/runtime/helper/engine/model/host")
		}
		pair := pairs[key]
		if pair == nil {
			pair = &supportPair{}
			pairs[key] = pair
		}
		slot := &pair.success
		if m.Cell == "canary" {
			slot = &pair.canary
		}
		if *slot != nil {
			return SupportList{}, fmt.Errorf("repeated support cell %s/%s", key, m.Cell)
		}
		*slot = cell
	}
	for key, pair := range pairs {
		if pair.success == nil || pair.canary == nil {
			return SupportList{}, fmt.Errorf("support cell %s requires both a success and a canary Result", key)
		}
	}
	gateExits, err := supportGateExits(pairs, gates)
	if err != nil {
		return SupportList{}, err
	}
	list := SupportList{Schema: SupportListSchema, Statement: SupportListStatement, Candidate: *candidate, Rows: []SupportRow{}}
	for key, pair := range pairs {
		success, canary := pair.success.Result, pair.canary.Result
		if err := Matched(success.Manifest, canary.Manifest); err != nil {
			return SupportList{}, fmt.Errorf("support cell %s: %w", key, err)
		}
		if success.Manifest.Profile != canary.Manifest.Profile || success.Manifest.HostConfigSHA256 != canary.Manifest.HostConfigSHA256 {
			return SupportList{}, fmt.Errorf("support cell %s success and canary profile or host config differ", key)
		}
		successTarget, _ := supportResultTarget(success)
		canaryTarget, _ := supportResultTarget(canary)
		if successTarget.SHA256 != canaryTarget.SHA256 || successTarget.Size != canaryTarget.Size || !slices.Equal(successTarget.BuildTags, canaryTarget.BuildTags) {
			return SupportList{}, fmt.Errorf("support cell %s success and canary ran different modules", key)
		}
		selection := supportCells[key]
		m := success.Manifest
		list.Rows = append(list.Rows, SupportRow{
			Package: selection.Package, Function: selection.Function, Run: selection.Run,
			Workload: m.Workload, CellID: m.ID, Profile: m.Profile,
			BuildTags:    append([]string{}, successTarget.BuildTags...),
			ModuleSHA256: successTarget.SHA256, ModuleBytes: uint64(successTarget.Size),
			WorkloadSourceSHA256: m.WorkloadSourceSHA256, HostConfigSHA256: m.HostConfigSHA256,
			Seed: m.Seed, Repeat: m.Repeat, Fuel: m.Fuel, MemoryBytes: m.MemoryBytes,
			Success: supportReference(*pair.success), Canary: supportReference(*pair.canary),
			Gates: gateExits[key],
		})
	}
	slices.SortFunc(list.Rows, func(a, b SupportRow) int {
		return strings.Compare(a.Package+"\x00"+a.Function, b.Package+"\x00"+b.Function)
	})
	return list, ValidateSupportList(list)
}

func supportGateExits(pairs map[string]*supportPair, gates []SupportGate) (map[string][]SupportGateExit, error) {
	recorded := map[string]map[string]SupportGate{}
	for _, gate := range gates {
		if pairs[gate.Cell] == nil {
			return nil, fmt.Errorf("gate %s names no qualified support cell %q", gate.Name, gate.Cell)
		}
		if !slices.Contains(supportGateOrder, gate.Name) {
			return nil, fmt.Errorf("unknown support gate %q", gate.Name)
		}
		if recorded[gate.Cell] == nil {
			recorded[gate.Cell] = map[string]SupportGate{}
		}
		if _, repeated := recorded[gate.Cell][gate.Name]; repeated {
			return nil, fmt.Errorf("repeated support gate %s for %s", gate.Name, gate.Cell)
		}
		recorded[gate.Cell][gate.Name] = gate
	}
	exits := map[string][]SupportGateExit{}
	for key := range pairs {
		for _, name := range supportGateOrder {
			gate, ok := recorded[key][name]
			if !ok {
				return nil, fmt.Errorf("support cell %s has no recorded %s gate", key, name)
			}
			exits[key] = append(exits[key], SupportGateExit{Name: gate.Name, Command: gate.Command, Exit: gate.Exit, Log: gate.Log})
		}
	}
	return exits, nil
}

// supportResultTarget returns the prepared target bound by every retained
// execution of the Result; executions disagreeing on the module are refused.
func supportResultTarget(result Result) (record.Target, error) {
	var target *record.Target
	for _, execution := range result.Qualification.Executions {
		evidence := execution.Evidence
		if evidence == nil && result.Qualification.EvidenceDigest != "" && execution.EvidenceDigest == result.Qualification.EvidenceDigest {
			evidence = result.Qualification.Evidence
		}
		if evidence == nil {
			return record.Target{}, errors.New("retained execution evidence is required")
		}
		if target == nil {
			target = &evidence.Target
		} else if target.SHA256 != evidence.Target.SHA256 || target.Source != evidence.Target.Source || !slices.Equal(target.Argv, evidence.Target.Argv) {
			return record.Target{}, errors.New("retained executions ran different targets")
		}
	}
	if target == nil {
		return record.Target{}, errors.New("retained execution evidence is required")
	}
	if _, err := record.ParseSHA256(string(target.SHA256)); err != nil {
		return record.Target{}, fmt.Errorf("prepared module identity: %w", err)
	}
	return *target, nil
}

func supportReference(cell SupportCellInput) SupportResultReference {
	reference := SupportResultReference{Path: cell.ResultPath, ResultSHA256: cell.Receipt.ResultSHA256, Executions: uint64(len(cell.Result.Qualification.Executions)), ReplayableBugs: cell.Result.Cost.ReplayableBugs}
	for _, execution := range cell.Result.Qualification.Executions {
		if execution.Replay != nil && execution.Replay.Match {
			reference.MatchedReplays++
		}
	}
	return reference
}

// ValidateSupportList checks a list's shape independently of its producer, so
// a retained list read back from disk carries the same guarantees.
func ValidateSupportList(list SupportList) error {
	if list.Schema != SupportListSchema || list.Statement != SupportListStatement {
		return errors.New("support list schema or per-test candidate-bound statement changed")
	}
	c := list.Candidate
	for _, digest := range []record.SHA256{c.IntegratedSourceSHA256, c.CompilerSHA256, c.RuntimeSHA256, c.HelperSHA256, c.EngineSHA256, c.ModelSHA256} {
		if _, err := record.ParseSHA256(string(digest)); err != nil {
			return fmt.Errorf("support list candidate identity: %w", err)
		}
	}
	if c.HostOS == "" || c.HostArch == "" || len(list.Rows) == 0 {
		return errors.New("support list requires a candidate host and at least one row")
	}
	for index, row := range list.Rows {
		if index > 0 && strings.Compare(list.Rows[index-1].Package+"\x00"+list.Rows[index-1].Function, row.Package+"\x00"+row.Function) >= 0 {
			return errors.New("support list rows must be unique and sorted by package and function")
		}
		if supportCells[row.Workload+"/"+row.CellID] != (supportSelection{Package: row.Package, Function: row.Function, Run: row.Run}) {
			return fmt.Errorf("support row %s %s is not a known support cell", row.Package, row.Function)
		}
		for _, digest := range []record.SHA256{row.ModuleSHA256, row.WorkloadSourceSHA256, row.HostConfigSHA256, row.Success.ResultSHA256, row.Canary.ResultSHA256} {
			if _, err := record.ParseSHA256(string(digest)); err != nil {
				return fmt.Errorf("support row %s identity: %w", row.Function, err)
			}
		}
		if row.Success.Executions != row.Repeat || row.Success.MatchedReplays != row.Repeat || row.Canary.Executions != row.Repeat || row.Canary.MatchedReplays != row.Repeat || row.Canary.ReplayableBugs != 1 || row.Success.ReplayableBugs != 0 {
			return fmt.Errorf("support row %s lacks complete matched replays or its replayable canary", row.Function)
		}
		if len(row.Gates) != len(supportGateOrder) {
			return fmt.Errorf("support row %s requires every cell gate", row.Function)
		}
		for gateIndex, gate := range row.Gates {
			if gate.Name != supportGateOrder[gateIndex] || gate.Command == "" || gate.Log == "" || gate.Exit != 0 {
				return fmt.Errorf("support row %s gate %s has no passing recorded exit", row.Function, gate.Name)
			}
		}
	}
	return nil
}

// MarshalSupportList renders the canonical JSON bytes.
func MarshalSupportList(list SupportList) ([]byte, error) {
	if err := ValidateSupportList(list); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// RenderSupportListMarkdown is a pure projection of the JSON list for review.
func RenderSupportListMarkdown(list SupportList) ([]byte, error) {
	if err := ValidateSupportList(list); err != nil {
		return nil, err
	}
	var b strings.Builder
	c := list.Candidate
	fmt.Fprintf(&b, "# WASM per-test support list (%s)\n\n", list.Schema)
	fmt.Fprintf(&b, "%s\n\n", list.Statement)
	b.WriteString("## Candidate\n\n| Identity | Value |\n| --- | --- |\n")
	for _, entry := range [][2]string{{"Integrated source", string(c.IntegratedSourceSHA256)}, {"Compiler", string(c.CompilerSHA256)}, {"Runtime overlay", string(c.RuntimeSHA256)}, {"Helper", string(c.HelperSHA256)}, {"Engine", string(c.EngineSHA256)}, {"Model", string(c.ModelSHA256)}, {"Host", c.HostOS + "/" + c.HostArch}} {
		fmt.Fprintf(&b, "| %s | `%s` |\n", entry[0], entry[1])
	}
	b.WriteString("\n## Supported tests\n\n| Package | Test | Cell | Module | Success Result | Canary Result |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, row := range list.Rows {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s/%s` (%s, seed %d, %d repeats) | `%s` (%d B) | `%s` (%d/%d replays matched) | `%s` (%d replayable bug) |\n", row.Package, row.Function, row.Workload, row.CellID, row.Profile, row.Seed, row.Repeat, row.ModuleSHA256, row.ModuleBytes, row.Success.ResultSHA256, row.Success.MatchedReplays, row.Success.Executions, row.Canary.ResultSHA256, row.Canary.ReplayableBugs)
	}
	for _, row := range list.Rows {
		fmt.Fprintf(&b, "\n### `%s` `%s`\n\n", row.Package, row.Function)
		fmt.Fprintf(&b, "Selection `-test.run=%s`; build tags `%s`; workload source `%s`; host config `%s`.\n\n", row.Run, strings.Join(row.BuildTags, ","), row.WorkloadSourceSHA256, row.HostConfigSHA256)
		fmt.Fprintf(&b, "Retained Results: success `%s`, canary `%s`.\n\n", row.Success.Path, row.Canary.Path)
		b.WriteString("| Gate | Exit | Command | Log |\n| --- | --- | --- | --- |\n")
		for _, gate := range row.Gates {
			fmt.Fprintf(&b, "| %s | %d | `%s` | `%s` |\n", gate.Name, gate.Exit, strings.ReplaceAll(gate.Command, "|", `\|`), gate.Log)
		}
	}
	return []byte(b.String()), nil
}

// WriteSupportList publishes base+".json" and base+".md" without replacing
// existing files; historical lists keep their identities.
func WriteSupportList(base string, list SupportList) (string, string, error) {
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || strings.HasSuffix(base, ".json") || strings.HasSuffix(base, ".md") {
		return "", "", errors.New("support list output requires an absolute base path without extension")
	}
	encoded, err := MarshalSupportList(list)
	if err != nil {
		return "", "", err
	}
	rendered, err := RenderSupportListMarkdown(list)
	if err != nil {
		return "", "", err
	}
	jsonPath, mdPath := base+".json", base+".md"
	for _, output := range []struct {
		path string
		data []byte
	}{{jsonPath, encoded}, {mdPath, rendered}} {
		if err := writeNewFile(output.path, output.data); err != nil {
			return "", "", err
		}
	}
	return jsonPath, mdPath, nil
}

func writeNewFile(path string, data []byte) (retErr error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}

// OpenSupportList reads a retained list strictly and revalidates its shape.
func OpenSupportList(path string) (SupportList, error) {
	encoded, err := hostfs.ReadBounded(path, 4<<20)
	if err != nil {
		return SupportList{}, err
	}
	var list SupportList
	if err := strictDecode(encoded, &list); err != nil {
		return SupportList{}, err
	}
	return list, ValidateSupportList(list)
}
