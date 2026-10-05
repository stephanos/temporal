// Package golden supports migration tests; production packages must not import it.
package golden

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//go:embed config.json
var configBytes []byte

type Substitution struct{ Old, New string }
type Config struct {
	Inventory []string `json:"ir_inventory"`
	// Later are the IR files added after the goldens were captured, such as a lifter fixture of a
	// later construct: each must exist, and none is compared, since no golden froze it.
	Later  []string       `json:"later_inventory"`
	Paths  []Substitution `json:"source_path_substitutions"`
	Labels []Substitution `json:"source_label_substitutions"`
	// Renames are the source files renamed after the mapped goldens were captured, from the path
	// Paths maps them to. The mapped goldens keep that path; the current IR has the renamed one.
	Renames []Substitution `json:"source_path_renames"`
	// Splits are the source files split out of another after the mapped goldens were captured, from
	// the current file to the one its declarations came from: a position in it compares as one there.
	Splits []Substitution `json:"source_path_splits"`
	// RootMoves are the roots whose declarations moved to another owner with them, from the frozen
	// root to the current one: a current Model's source names each by its frozen root.
	RootMoves []Substitution `json:"source_root_moves"`
	// RootRetirements are the frozen roots whose only declarations were Queries a law replacement of
	// the original baseline retires (original.json): the mapped original's source names none of them.
	RootRetirements []string `json:"source_root_retirements"`
	// RootAdditions are the roots added since, each declaring capabilities whose laws generate the
	// claims a law replacement lists: a current Model's source is compared without them.
	RootAdditions []string `json:"source_root_additions"`
	// Merges are the feature directories whose declarations moved between their own files after the
	// mapped goldens were captured, from the directory (ending in "/") to the one name every position
	// under it compares as, in the mapped original and in the current Model alike. A feature
	// reorganized by subject and declaration kind mixes declarations of several frozen files in one
	// file, which Splits cannot name. Which file of its directory a declaration sits in is read by no
	// table, Definition ID, fingerprint, answer or Case, and the original-baseline harness drops
	// positions entirely; a declaration moving to another feature or fixture still fails. An entry
	// may also name one Scala file, which compares as its name alone: the realizations whose shared
	// declarations moved into the Temporal kit (fn-112.9) and the kit compare as one name. The first
	// entry that names a position applies.
	Merges     []Substitution `json:"source_path_merges"`
	Projection Projection     `json:"projection"`
	// PathMoves are the directories moved after the goldens were captured, from the directory (ending
	// in "/") to the one it moved to, such as "model/temporal/nexuscaller/" to
	// "model/temporal/features/nexuscaller/" (fn-114.9): an inventory path under the old directory is
	// read under the new one, and a position or Case source path under the new one compares as the same
	// path under the old one, before Splits and Merges apply. The first entry that names a path applies.
	PathMoves []Substitution `json:"source_path_moves"`
	// PackageMoves are the Scala packages moved after the goldens were captured, from the package
	// (ending in ".") to the one it moved to, such as "temporal.nexuscaller." to
	// "temporal.features.nexuscaller." (fn-114.9): a current Model's source names each root of a moved
	// package under the old one, before RootMoves apply. Definition IDs and type names keep their
	// spelling through the Models' DefinitionScope pins, so only the source's roots, which name Scala
	// declarations by their full names, are read this way; Functions are compared by reference.
	PackageMoves []Substitution `json:"source_package_moves"`
	// Reduced are the IR inventory paths of the original baseline's reduced fixtures, which
	// Configuration reads from original.json (Delta.Reduced) rather than a list of its own: one decision
	// about one file, at the original-baseline key MatchAt already reads the delta at. Its inventory
	// entry stays, naming what the goldens froze; Inputs leaves it out, present or retired, so none of
	// its goldens is compared (Unreduced).
	Reduced []string `json:"-"`
}

// Projection is what Match ignores when the current IR is compared with the mapped original, beyond
// Renames: changes of lifted text that no table, Definition ID, refinement row, fingerprint, Query
// answer or lowered Case byte reads. The goldens themselves are still derived from the frozen inputs.
type Projection struct {
	// PositionsByFile compares a position by its file alone.
	PositionsByFile bool `json:"positions_by_file"`
	// AlphaParameters compares the parameters of every Function and Lambda by their place, not their
	// name: each binder, and every variable that refers to it, gets a name made of its depth and index.
	AlphaParameters bool `json:"alpha_normalized_parameters"`
	// Functions are the Functions renamed after the goldens were captured, from the frozen name. Each
	// renames the Function, and every string of the Model equal to its name, exactly.
	Functions []Substitution `json:"function_name_substitutions"`
	// FunctionsByReference compares a Model without its Functions, and reads each Function reference
	// that is set as one token, as the original baseline does (functionless): fn-112 rewrites Function
	// bodies, retires and adds helpers and renames what steps, Properties, monitors, evidence and
	// refinements call. The IR comparison then holds no step's meaning. What the current IR means is
	// compared with the frozen original's on its readings (TestMigrationProjectionPreservesSemantics:
	// tables, Property answers, receipts, Definition IDs, fingerprints and refined Properties) and on
	// its lowered Cases (TestMigrationProjectionKeepsLoweredCases); each refuses a flipped guard this
	// comparison admits (TestMigrationGoldensAdmitOnlyTheProjection and the lowered Cases' changed
	// step). The original-baseline harness holds the same meaning in the same gate.
	FunctionsByReference bool `json:"functions_by_reference"`
	// Types are exact names of finite declarations moved after the goldens were captured.
	Types []Substitution `json:"type_name_substitutions"`
	// InertFields are IR fields, by full protobuf name, added after the goldens were captured and read
	// by no table, ID, fingerprint, answer or Case: a frozen input never sets one, and the current IR
	// is compared without it. A realization's required settings are read by the Cases lowered through
	// it, which TestMigrationProjectionKeepsLoweredCases compares on their own. Its API behavior hints
	// and server steps are read by no table, ID, fingerprint, answer or Contract. Since fn-118.4 a
	// Case's waits are lowered from them, and the waits they shape are compared through the original
	// baseline's derived waits (Delta.DerivedWaits), which MatchAt and the lowered Cases' comparison read.
	InertFields []string `json:"inert_fields"`
	// CaseIDs are the kinds of lowered Case compared without their ID. An exploration Case's IDs carry
	// the digest of its whole candidate Model, which the changes above alter; its other bytes do not.
	CaseIDs []string `json:"projected_case_ids"`
}

// The kinds of lowered Case: one a Query lowers to, and one an exploration's candidate lowers to.
const (
	QueryCase       = "query"
	ExplorationCase = "exploration"
)

const projectedCaseID = "projected-case-id"

// explorationID is a candidate digest: a sha256 in hex, so a shorter string cannot widen the projection.
var explorationID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func Configuration() (Config, error) {
	var c Config
	decoder := json.NewDecoder(bytes.NewReader(configBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return c, err
	}
	delta, err := OriginalDelta()
	if err != nil {
		return c, err
	}
	if c.Reduced, err = c.reduced(delta.Reduced); err != nil {
		return c, err
	}
	return c, c.Projection.caseKinds()
}

// reduced gives the IR inventory path of each reduced fixture's archive key: a lifter fixture of the
// inventory, never a later one, listed once.
func (c Config) reduced(keys []string) ([]string, error) {
	var out []string
	for _, key := range keys {
		i := slices.IndexFunc(c.Inventory, func(path string) bool { return OriginalKey(path) == key })
		if !strings.HasPrefix(key, OriginalLifts) || i < 0 || slices.Contains(out, c.Inventory[i]) {
			return nil, fmt.Errorf("reduced fixture %q is not one lifter fixture of the IR inventory", key)
		}
		out = append(out, c.Inventory[i])
	}
	return out, nil
}

// Unreduced gives the migration goldens without every entry of a reduced fixture: its frozen input,
// its readings and what it lowers to, under whichever directory they are filed.
func (c Config) Unreduced(goldens map[string][]byte) map[string][]byte {
	out := maps.Clone(goldens)
	for _, path := range c.Reduced {
		file := "/" + strings.TrimPrefix(path, "model/scalav2/")
		for key := range goldens {
			if strings.HasSuffix(key, file) || strings.Contains(key, file+"/") {
				delete(out, key)
			}
		}
	}
	return out
}

func (p Projection) caseKinds() error {
	for _, kind := range p.CaseIDs {
		if kind != QueryCase && kind != ExplorationCase {
			return fmt.Errorf("unknown lowered Case kind %q", kind)
		}
	}
	return nil
}

// Case gives the bytes of a lowered Case, or of its Program or Contract, as the goldens compare them.
// For a kind whose ID the projection names, id, the part of the Case's IDs that varies, is replaced
// by a fixed token wherever it occurs. Every other byte, and every byte of another kind, is kept.
func (p Projection) Case(kind string, encoded []byte, id string) ([]byte, error) {
	if err := p.caseKinds(); err != nil {
		return nil, err
	}
	if kind != QueryCase && kind != ExplorationCase {
		return nil, fmt.Errorf("unknown lowered Case kind %q", kind)
	}
	if !slices.Contains(p.CaseIDs, kind) {
		return encoded, nil
	}
	if !explorationID.MatchString(id) || !bytes.Contains(encoded, []byte(id)) {
		return nil, fmt.Errorf("lowered %s Case does not carry the candidate digest %q", kind, id)
	}
	return bytes.ReplaceAll(encoded, []byte(id), []byte(projectedCaseID)), nil
}

// RenameSources applies Renames to the source paths a lowered Case, Program or Contract of the mapped
// goldens names: each is a JSON string equal to a renamed path. Nothing else changes.
func (c Config) RenameSources(encoded []byte) []byte {
	for _, r := range c.Renames {
		encoded = bytes.ReplaceAll(encoded, []byte(strconv.Quote(r.Old)), []byte(strconv.Quote(r.New)))
	}
	return encoded
}

// UnsplitSources applies Splits to the source paths a lowered Case, Program or Contract of the current
// IR names: each is a JSON string equal to a split file, which becomes the file its declarations came
// from. A Case names its Query's file, so a Query split out of a file compares as Unsplit compares it.
func (c Config) UnsplitSources(encoded []byte) []byte {
	for _, s := range c.Splits {
		encoded = bytes.ReplaceAll(encoded, []byte(strconv.Quote(s.Old)), []byte(strconv.Quote(s.New)))
	}
	return encoded
}

// MergeSources applies Merges to the source paths a lowered Case, Program or Contract names: each is a
// JSON string that is a path under a merged directory, which becomes the directory's name.
func (c Config) MergeSources(encoded []byte) []byte {
	for _, m := range c.Merges {
		quoted := regexp.MustCompile(`"` + regexp.QuoteMeta(m.Old) + `[\w./-]*"`)
		encoded = quoted.ReplaceAllLiteral(encoded, []byte(strconv.Quote(m.New)))
	}
	return encoded
}

func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository go.mod not found")
		}
		dir = parent
	}
}

// movedTo checks the moves, and that each path move names a directory the checkout under root has.
// That the old one is gone is TestRetiredModelPathsStayRetired's.
func (c Config) movedTo(root string) error {
	if err := c.movesApply(); err != nil {
		return err
	}
	for _, m := range c.PathMoves {
		if _, err := os.Stat(filepath.Join(root, m.New)); err != nil {
			return fmt.Errorf("source path move to %s: %w", m.New, err)
		}
	}
	return nil
}

// movedInventory gives the inventory paths of underBase under the directories PathMoves moved them to.
func (c Config) movedInventory(paths map[string]string) map[string]string {
	out := make(map[string]string, len(paths))
	for path, entry := range paths {
		out[c.MovedPath(path)] = entry
	}
	return out
}

// moved gives s with the prefix of the first move that names it replaced: from Old to New forward,
// from New to Old backward. s is kept when no move names it.
func moved(moves []Substitution, s string, forward bool) string {
	for _, m := range moves {
		from, to := m.New, m.Old
		if forward {
			from, to = m.Old, m.New
		}
		if rest, ok := strings.CutPrefix(s, from); ok {
			return to + rest
		}
	}
	return s
}

// MovedPath gives path under the directory PathMoves moved it to, or path when none did.
func (c Config) MovedPath(path string) string { return moved(c.PathMoves, path, true) }

// unmovedPath gives a current path under the directory it had before PathMoves moved it.
func (c Config) unmovedPath(path string) string { return moved(c.PathMoves, path, false) }

// MovedPackage gives a Scala full name of a moved package's declaration under the package PackageMoves
// moved it to, such as a Function name the substitutions spell as it was before the move.
func (c Config) MovedPackage(name string) string { return moved(c.PackageMoves, name, true) }

// unmovedRoot gives a current root under the package it had before PackageMoves moved it.
func (c Config) unmovedRoot(name string) string { return moved(c.PackageMoves, name, false) }

// movesApply refuses a path move that is not of one directory to another, a package move that is not
// of one package to another, and an entry listed twice.
func (c Config) movesApply() error {
	for _, moves := range []struct {
		list []Substitution
		end  string
		kind string
	}{{c.PathMoves, "/", "source path move"}, {c.PackageMoves, ".", "source package move"}} {
		seen := map[string]bool{}
		for _, m := range moves.list {
			if !strings.HasSuffix(m.Old, moves.end) || !strings.HasSuffix(m.New, moves.end) || m.Old == m.New ||
				seen[m.Old] || seen[m.New] {
				return fmt.Errorf("%s of %q to %q is not a move of one %q-terminated name to another", moves.kind, m.Old, m.New, moves.end)
			}
			seen[m.Old], seen[m.New] = true, true
		}
	}
	return nil
}

// Unmoved gives a current Model as it was spelled before PathMoves and PackageMoves: every position
// under a moved directory names the old one, and its source names each root of a moved package under
// the old package, in the sorted order the lifter lists roots in.
func (c Config) Unmoved(current *umpirespb.Model) (*umpirespb.Model, error) {
	if err := c.movesApply(); err != nil {
		return nil, err
	}
	if len(c.PathMoves) == 0 && len(c.PackageMoves) == 0 {
		return current, nil
	}
	m := proto.CloneOf(current)
	if roots, ok := strings.CutPrefix(m.GetSource(), "model: "); ok && len(c.PackageMoves) > 0 {
		names := strings.Split(roots, ", ")
		for i, name := range names {
			names[i] = c.unmovedRoot(name)
		}
		slices.Sort(names)
		m.Source = "model: " + strings.Join(names, ", ")
	}
	err := positions(m.ProtoReflect(), func(p protoreflect.Message) error {
		field := p.Descriptor().Fields().ByName("file")
		p.Set(field, protoreflect.ValueOfString(c.unmovedPath(p.Get(field).String())))
		return nil
	})
	return m, err
}

// UnmoveSources applies PathMoves backwards to the source paths a lowered Case, Program or Contract of
// the current IR names: each is a JSON string that is a path under a moved directory, which becomes the
// same path under the old one. Nothing else changes.
func (c Config) UnmoveSources(encoded []byte) []byte {
	for _, m := range c.PathMoves {
		quoted := regexp.MustCompile(`"` + regexp.QuoteMeta(m.New) + `[\w./-]*"`)
		encoded = quoted.ReplaceAllFunc(encoded, func(path []byte) []byte {
			return append([]byte(`"`+m.Old), path[len(m.New)+1:]...)
		})
	}
	return encoded
}

// UnmovedText applies PathMoves backwards to free text that names current source paths, such as a
// located diagnostic: each path under a moved directory names the same path under the old one.
func (c Config) UnmovedText(text string) string {
	for _, m := range c.PathMoves {
		text = strings.ReplaceAll(text, m.New, m.Old)
	}
	return text
}

// underBase maps each inventory path, spelled under model/scalav2, to its path under base.
func underBase(paths []string, base string) map[string]string {
	out := make(map[string]string, len(paths))
	for _, path := range paths {
		out[base+strings.TrimPrefix(path, "model/scalav2")] = path
	}
	return out
}

// LawSidecarSuffix ends the law sidecar the lifter writes beside an IR file whose Models declare
// capabilities, `<file>.laws.json`: JSON that records the generated claims, and no Model. It is
// tools/umpire/model.LawSidecarSuffix, which this test-only package does not import.
const LawSidecarSuffix = ".laws.json"

// IsLawSidecar reports whether a path or name is a law sidecar rather than an IR file.
func IsLawSidecar(path string) bool { return strings.HasSuffix(path, LawSidecarSuffix) }

// AcceptedSuffix ends the file of lint's accepted findings an author writes beside an IR file,
// `<file>.lint.json`. It is lint.AcceptedSuffix, which this test-only package does not import.
const AcceptedSuffix = ".lint.json"

// IsAccepted reports whether a path or name is a file of accepted lint findings rather than an IR
// file. The lifter produces none, so no archive freezes one.
func IsAccepted(path string) bool { return strings.HasSuffix(path, AcceptedSuffix) }

// IRFiles lists a directory's IR files: its JSON files, apart from the law sidecars and the accepted
// lint findings beside them.
func IRFiles(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(paths, func(p string) bool { return IsLawSidecar(p) || IsAccepted(p) }), nil
}

// inventoried checks that each reduced fixture is an entry of the IR inventory.
func (c Config) inventoried() error {
	for _, path := range c.Reduced {
		if !slices.Contains(c.Inventory, path) {
			return fmt.Errorf("reduced IR inventory entry %s is not inventoried", path)
		}
	}
	return nil
}

func (c Config) Inputs(root string) (map[string]*umpirespb.Model, error) {
	if err := errors.Join(c.inventoried(), c.movedTo(root)); err != nil {
		return nil, err
	}
	base := "model/scalav2"
	if _, err := os.Stat(filepath.Join(root, base)); errors.Is(err, fs.ErrNotExist) {
		base = "model"
	} else if err != nil {
		return nil, err
	}
	expected, later := c.movedInventory(underBase(c.Inventory, base)), c.movedInventory(underBase(c.Later, base))
	found := map[string]bool{}
	for _, dir := range []string{base + "/ir/", base + "/lifter/testdata/lifts/expected/"} {
		paths, err := IRFiles(filepath.Join(root, c.MovedPath(dir)))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil, err
			}
			found[rel] = true
			_, listed := expected[rel]
			if _, added := later[rel]; !listed && !added {
				return nil, fmt.Errorf("unknown IR inventory entry %s", rel)
			}
		}
	}
	for _, path := range slices.Sorted(maps.Keys(later)) {
		if !found[path] {
			return nil, fmt.Errorf("missing later IR inventory entry %s", path)
		}
	}
	// A reduced fixture is a new Model, or retired: it is compared with no frozen input.
	maps.DeleteFunc(expected, func(_, entry string) bool { return slices.Contains(c.Reduced, entry) })
	out := map[string]*umpirespb.Model{}
	for _, path := range slices.Sorted(maps.Keys(expected)) {
		if !found[path] {
			return nil, fmt.Errorf("missing IR inventory entry %s", path)
		}
		encoded, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return nil, err
		}
		m := new(umpirespb.Model)
		if err := protojson.Unmarshal(encoded, m); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[expected[path]] = m
	}
	return out, nil
}

func (c Config) Migrate(original *umpirespb.Model) (*umpirespb.Model, error) {
	m := proto.CloneOf(original)
	var err error
	m.Source, err = substitute(m.GetSource(), c.Labels)
	if err != nil {
		return nil, err
	}
	err = positions(m.ProtoReflect(), func(p protoreflect.Message) error {
		field := p.Descriptor().Fields().ByName("file")
		mapped, err := substitute(p.Get(field).String(), c.Paths)
		if err != nil {
			return err
		}
		p.Set(field, protoreflect.ValueOfString(mapped))
		return nil
	})
	return m, err
}

// Rename applies Renames to a Model Migrate mapped. A path no rename names is kept as it is.
func (c Config) Rename(mapped *umpirespb.Model) (*umpirespb.Model, error) {
	for _, r := range c.Renames {
		if !slices.ContainsFunc(c.Paths, func(p Substitution) bool { return p.New == r.Old }) {
			return nil, fmt.Errorf("rename of %q, which no source path substitution produces", r.Old)
		}
	}
	m := proto.CloneOf(mapped)
	err := positions(m.ProtoReflect(), func(p protoreflect.Message) error {
		field := p.Descriptor().Fields().ByName("file")
		for _, r := range c.Renames {
			if p.Get(field).String() == r.Old {
				p.Set(field, protoreflect.ValueOfString(r.New))
				break
			}
		}
		return nil
	})
	return m, err
}

func substitute(value string, substitutions []Substitution) (string, error) {
	for _, s := range substitutions {
		if value == s.Old {
			return s.New, nil
		}
	}
	return "", fmt.Errorf("unlisted source %q", value)
}

func positions(m protoreflect.Message, visit func(protoreflect.Message) error) error {
	position := (&umpirespb.Position{}).ProtoReflect().Descriptor().FullName()
	return messages(m, func(child protoreflect.Message) (bool, error) {
		if child.Descriptor().FullName() == position {
			return false, visit(child)
		}
		return true, nil
	})
}

// messages visits every message under m, in field order, and descends into one when visit says so.
func messages(m protoreflect.Message, visit func(protoreflect.Message) (bool, error)) error {
	var result error
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if f.Message() == nil || f.IsMap() {
			return true
		}
		walk := func(child protoreflect.Message) error {
			descend, err := visit(child)
			if err != nil || !descend {
				return err
			}
			return messages(child, visit)
		}
		if f.IsList() {
			for i := range v.List().Len() {
				if result = walk(v.List().Get(i).Message()); result != nil {
					return false
				}
			}
		} else {
			result = walk(v.Message())
		}
		return result == nil
	})
	return result
}

// OriginalKey is the original baseline's archive key of an IR inventory path: "ir/activity.json" for
// model/scalav2/ir/activity.json, "lifts/admission.json" for a lifter fixture.
func OriginalKey(path string) string {
	path = strings.TrimPrefix(path, "model/scalav2/")
	if name, ok := strings.CutPrefix(path, "lifter/testdata/lifts/expected/"); ok {
		return OriginalLifts + name
	}
	return path
}

// Match is MatchAt of no original-baseline key: what no law replacement or derived wait names.
func (c Config) Match(original, current *umpirespb.Model) (bool, error) {
	return c.MatchAt("", original, current)
}

// MatchAt checks a current Model against its frozen input, the Model of an original-baseline key
// (OriginalKey): the frozen input with the original baseline's delta applied (Attached), against the
// current Model without the generated claims the delta lists (Ungenerated), each read without the
// waits the delta lists as derived (Underived).
func (c Config) MatchAt(key string, original, current *umpirespb.Model) (bool, error) {
	if proto.Equal(original, current) {
		return false, nil
	}
	current, err := c.Unmoved(current)
	if err != nil {
		return false, err
	}
	mapped, err := c.Migrate(original)
	if err != nil {
		return false, err
	}
	if mapped, err = c.Rename(mapped); err != nil {
		return false, err
	}
	// The original baseline's delta, the entity metadata fn-112's R20 adds and the claims fn-122's laws
	// replace, applies to the frozen input too, and a declaration split out of a file compares as one
	// of the file it left.
	if mapped, err = c.Retired(mapped); err != nil {
		return false, err
	}
	if mapped, err = Attached(key, mapped); err != nil {
		return false, err
	}
	if current, err = Ungenerated(key, current); err != nil {
		return false, err
	}
	if mapped, current, err = Underived(key, mapped, current); err != nil {
		return false, err
	}
	if current, err = c.Unsplit(current); err != nil {
		return false, err
	}
	if mapped, err = c.Merged(mapped); err != nil {
		return false, err
	}
	if current, err = c.Merged(current); err != nil {
		return false, err
	}
	if proto.Equal(mapped, current) {
		return true, nil
	}
	if mapped, err = c.Projection.project(mapped, true); err != nil {
		return false, err
	}
	projected, err := c.Projection.project(current, false)
	if err != nil {
		return false, err
	}
	if !proto.Equal(mapped, projected) {
		return false, errors.New("IR differs outside the closed source migration")
	}
	return true, nil
}

// Attached gives the frozen input of an original-baseline key with the original baseline's delta
// applied (original.json, Delta.Expected): the entity metadata fn-112's R20 adds to existing
// declarations, and the key's law replacements.
func Attached(key string, original *umpirespb.Model) (*umpirespb.Model, error) {
	delta, err := OriginalDelta()
	if err != nil {
		return nil, err
	}
	return delta.Expected(key, original, Applied{})
}

// Ungenerated gives a current Model of an original-baseline key without the generated claims the
// original baseline's delta lists for it (Delta.Ungenerated).
func Ungenerated(key string, current *umpirespb.Model) (*umpirespb.Model, error) {
	delta, err := OriginalDelta()
	if err != nil {
		return nil, err
	}
	return delta.Ungenerated(key, current)
}

// Underived gives a frozen input and a current Model of an original-baseline key without the waits
// the original baseline's delta lists as derived for it: each listed command the frozen input writes a
// wait of, and the current Model writes none of (Waits.Baseline, Waits.Current).
func Underived(key string, frozen, current *umpirespb.Model) (*umpirespb.Model, *umpirespb.Model, error) {
	delta, err := OriginalDelta()
	if err != nil {
		return nil, nil, err
	}
	waits := delta.Waits(key)
	if frozen, err = waits.Baseline(frozen); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", key, err)
	}
	if current, err = waits.Current(current); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", key, err)
	}
	return frozen, current, nil
}

// Retired gives a mapped original whose source names no retired root.
func (c Config) Retired(mapped *umpirespb.Model) (*umpirespb.Model, error) {
	roots, ok := strings.CutPrefix(mapped.GetSource(), "model: ")
	if !ok || len(c.RootRetirements) == 0 {
		return mapped, nil
	}
	names := strings.Split(roots, ", ")
	kept := slices.DeleteFunc(slices.Clone(names), func(name string) bool { return slices.Contains(c.RootRetirements, name) })
	if len(kept) == len(names) {
		return mapped, nil
	}
	m := proto.CloneOf(mapped)
	m.Source = "model: " + strings.Join(kept, ", ")
	return m, nil
}

// RootsApply checks that every root retirement names a root of some mapped frozen input, and every
// root addition and moved root one of some current Model, so the lists stay closed.
func (c Config) RootsApply(originals, currents map[string]*umpirespb.Model) error {
	frozen, err := c.sourceRoots(originals, true)
	if err != nil {
		return err
	}
	moved, err := c.sourceRoots(currents, false)
	if err != nil {
		return err
	}
	// A root of a moved package is read under the package it had, as Unmoved reads it.
	current := map[string]bool{}
	for name := range moved {
		current[c.unmovedRoot(name)] = true
	}
	var errs []error
	for _, name := range c.RootRetirements {
		if !frozen[name] {
			errs = append(errs, fmt.Errorf("source root retirement of %q, which no frozen input names", name))
		}
	}
	for _, name := range c.RootAdditions {
		if !current[name] {
			errs = append(errs, fmt.Errorf("source root addition of %q, which no current Model names", name))
		}
	}
	for _, move := range c.RootMoves {
		if !current[move.New] {
			errs = append(errs, fmt.Errorf("source root move to %q, which no current Model names", move.New))
		}
	}
	for _, move := range c.PackageMoves {
		if !slices.ContainsFunc(slices.Collect(maps.Keys(moved)), func(name string) bool { return strings.HasPrefix(name, move.New) }) {
			errs = append(errs, fmt.Errorf("source package move to %q, which no root of a current Model is in", move.New))
		}
	}
	return errors.Join(errs...)
}

// sourceRoots are the roots the sources of models name. A frozen input is migrated first, so its
// roots are read under the names the closed source migration gives them.
func (c Config) sourceRoots(models map[string]*umpirespb.Model, migrate bool) (map[string]bool, error) {
	out := map[string]bool{}
	for _, m := range models {
		if migrate {
			var err error
			if m, err = c.Migrate(m); err != nil {
				return nil, err
			}
		}
		if names, ok := strings.CutPrefix(m.GetSource(), "model: "); ok {
			for _, name := range strings.Split(names, ", ") {
				out[name] = true
			}
		}
	}
	return out, nil
}

// Unsplit gives a current Model with every position in a split file naming the file its
// declaration came from, and its source naming each moved root by its frozen name, in the sorted
// order the lifter lists roots in, and no added root.
func (c Config) Unsplit(current *umpirespb.Model) (*umpirespb.Model, error) {
	if len(c.Splits) == 0 && len(c.RootMoves) == 0 && len(c.RootAdditions) == 0 {
		return current, nil
	}
	m := proto.CloneOf(current)
	if roots, ok := strings.CutPrefix(m.GetSource(), "model: "); ok && len(c.RootMoves)+len(c.RootAdditions) > 0 {
		names := slices.DeleteFunc(strings.Split(roots, ", "), func(name string) bool { return slices.Contains(c.RootAdditions, name) })
		for i, name := range names {
			for _, move := range c.RootMoves {
				if name == move.New {
					names[i] = move.Old
				}
			}
		}
		slices.Sort(names)
		m.Source = "model: " + strings.Join(names, ", ")
	}
	err := positions(m.ProtoReflect(), func(p protoreflect.Message) error {
		field := p.Descriptor().Fields().ByName("file")
		for _, s := range c.Splits {
			if p.Get(field).String() == s.Old {
				p.Set(field, protoreflect.ValueOfString(s.New))
				break
			}
		}
		return nil
	})
	return m, err
}

// Merged gives m with every position in a merged directory naming that directory.
func (c Config) Merged(m *umpirespb.Model) (*umpirespb.Model, error) {
	for _, merge := range c.Merges {
		if !strings.HasSuffix(merge.Old, "/") && !strings.HasSuffix(merge.Old, ".scala") {
			return nil, fmt.Errorf("source path merge of %q, which is not a directory or a Scala file", merge.Old)
		}
	}
	if len(c.Merges) == 0 {
		return m, nil
	}
	m = proto.CloneOf(m)
	err := positions(m.ProtoReflect(), func(p protoreflect.Message) error {
		field := p.Descriptor().Fields().ByName("file")
		for _, merge := range c.Merges {
			if strings.HasPrefix(p.Get(field).String(), merge.Old) {
				p.Set(field, protoreflect.ValueOfString(merge.New))
				break
			}
		}
		return nil
	})
	return m, err
}

// FrozenModels reads the frozen inputs whose keys begin with prefix, such as "ir/", as Models by key:
// the inputs FunctionsRenamed checks the substitutions against.
func FrozenModels(files map[string][]byte, prefix string) (map[string]*umpirespb.Model, error) {
	models := map[string]*umpirespb.Model{}
	for key, encoded := range files {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		m := new(umpirespb.Model)
		if err := protojson.Unmarshal(encoded, m); err != nil {
			return nil, fmt.Errorf("frozen input %s: %w", key, err)
		}
		models[key] = m
	}
	return models, nil
}

// FunctionsRenamed checks that every function-name substitution renames a Function of some frozen
// input, so the list stays closed.
func (c Config) FunctionsRenamed(originals map[string]*umpirespb.Model) error {
	declared := map[string]bool{}
	for _, m := range originals {
		for _, f := range m.GetFunctions() {
			declared[f.GetName()] = true
		}
	}
	for _, s := range c.Projection.Functions {
		if !declared[s.Old] {
			return fmt.Errorf("function-name substitution of %q, which no frozen input declares", s.Old)
		}
	}
	return nil
}

// TypesRenamed checks each listed source against the frozen declarations and rejects ambiguous targets.
func (c Config) TypesRenamed(originals map[string]*umpirespb.Model) error {
	declared := map[string]bool{}
	for _, m := range originals {
		for _, t := range m.GetTypes() {
			declared[t.GetName()] = true
		}
	}
	sources := map[string]bool{}
	targets := map[string]bool{}
	for _, s := range c.Projection.Types {
		if !declared[s.Old] {
			return fmt.Errorf("type-name substitution of %q, which no frozen input declares", s.Old)
		}
		if s.Old == s.New || sources[s.Old] || targets[s.New] || declared[s.New] {
			return fmt.Errorf("type-name substitution of %q is not a rename to one new name", s.Old)
		}
		sources[s.Old] = true
		targets[s.New] = true
	}
	return nil
}

// project gives m as Match compares it. Only the mapped original's Function names are substituted:
// the current IR already has the new ones.
func (p Projection) project(m *umpirespb.Model, original bool) (*umpirespb.Model, error) {
	m = proto.CloneOf(m)
	if original {
		if err := p.rename(m); err != nil {
			return nil, err
		}
	}
	for _, name := range p.InertFields {
		field, err := inertField(name)
		if err != nil {
			return nil, err
		}
		if err := messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
			if child.Descriptor() != field.ContainingMessage() {
				return true, nil
			}
			if original && child.Has(field) {
				return false, fmt.Errorf("inert field %s is set in a frozen input", name)
			}
			child.Clear(field)
			return true, nil
		}); err != nil {
			return nil, err
		}
	}
	if p.FunctionsByReference {
		if err := functionless(m); err != nil {
			return nil, err
		}
	}
	if p.PositionsByFile {
		if err := positions(m.ProtoReflect(), func(at protoreflect.Message) error {
			at.Clear(at.Descriptor().Fields().ByName("line"))
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if p.AlphaParameters {
		if err := messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
			switch v := child.Interface().(type) {
			case *umpirespb.Function:
				scope := parameters(v.GetParams(), nil, 0)
				alpha(v.GetBody(), scope, 1)
				alpha(v.GetRequires(), scope, 1)
				return false, nil
			case *umpirespb.Expr:
				alpha(v, nil, 0)
				return false, nil
			}
			return true, nil
		}); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (p Projection) rename(m *umpirespb.Model) error {
	declared := map[string]bool{}
	for _, f := range m.GetFunctions() {
		declared[f.GetName()] = true
	}
	renamed := map[string]string{}
	for _, s := range p.Functions {
		if s.Old == s.New || renamed[s.Old] != "" {
			return fmt.Errorf("function-name substitution of %q is not a rename to one new name", s.Old)
		}
		if declared[s.Old] {
			renamed[s.Old] = s.New
		}
	}
	renamedType, err := p.addTypeNames(m, renamed)
	if err != nil {
		return err
	}
	if len(renamed) == 0 {
		return nil
	}
	// Every string equal to an old name is renamed, not only the fields that reference a Function:
	// the names are fully qualified, so nothing else spells one, and no reference field is missed.
	if err := messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
		var names []protoreflect.FieldDescriptor
		child.Range(func(f protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			if f.Kind() == protoreflect.StringKind && !f.IsMap() {
				names = append(names, f)
			}
			return true
		})
		for _, f := range names {
			if !f.IsList() {
				if name, ok := renamed[child.Get(f).String()]; ok {
					child.Set(f, protoreflect.ValueOfString(name))
				}
				continue
			}
			list := child.Mutable(f).List()
			for i := range list.Len() {
				if name, ok := renamed[list.Get(i).String()]; ok {
					list.Set(i, protoreflect.ValueOfString(name))
				}
			}
		}
		return true, nil
	}); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, f := range m.GetFunctions() {
		if seen[f.GetName()] {
			return fmt.Errorf("function-name substitution gives two Functions the name %q", f.GetName())
		}
		seen[f.GetName()] = true
	}
	seen = map[string]bool{}
	for _, t := range m.GetTypes() {
		if seen[t.GetName()] {
			return fmt.Errorf("type-name substitution gives two Types the name %q", t.GetName())
		}
		seen[t.GetName()] = true
	}
	if renamedType {
		// Lift sorts declarations by name, so a moved type takes its new place in that order.
		slices.SortFunc(m.Types, func(a, b *umpirespb.Type) int {
			return strings.Compare(a.GetName(), b.GetName())
		})
	}
	return nil
}

func (p Projection) addTypeNames(m *umpirespb.Model, renamed map[string]string) (bool, error) {
	declared := map[string]bool{}
	for _, t := range m.GetTypes() {
		declared[t.GetName()] = true
	}
	changed := false
	for _, s := range p.Types {
		if s.Old == s.New || renamed[s.Old] != "" || (declared[s.Old] && declared[s.New]) {
			return false, fmt.Errorf("type-name substitution of %q is not a rename to one new name", s.Old)
		}
		if declared[s.Old] {
			renamed[s.Old] = s.New
			changed = true
		}
	}
	return changed, nil
}

// parameters names each parameter by its depth and index, in a scope that extends outer. The names
// cannot be written in Scala, so none of them collides with a lifted one.
func parameters(params []*umpirespb.Param, outer map[string]string, depth int) map[string]string {
	scope := maps.Clone(outer)
	if scope == nil {
		scope = map[string]string{}
	}
	for i, p := range params {
		name := fmt.Sprintf("#%d.%d", depth, i)
		scope[p.GetName()] = name
		p.Name = name
	}
	return scope
}

// alpha renames the variables of e that refer to a parameter in scope. A `let` or a pattern that
// binds the same name hides the parameter in what it scopes over.
func alpha(e *umpirespb.Expr, scope map[string]string, depth int) {
	if e == nil {
		return
	}
	hide := func(names ...string) map[string]string {
		inner := maps.Clone(scope)
		for _, name := range names {
			delete(inner, name)
		}
		return inner
	}
	switch k := e.GetKind().(type) {
	case *umpirespb.Expr_Var:
		if name, ok := scope[k.Var]; ok {
			k.Var = name
		}
	case *umpirespb.Expr_Lambda:
		alpha(k.Lambda.GetBody(), parameters(k.Lambda.GetParams(), scope, depth), depth+1)
	case *umpirespb.Expr_Let:
		alpha(k.Let.GetValue(), scope, depth)
		alpha(k.Let.GetBody(), hide(k.Let.GetName()), depth)
	case *umpirespb.Expr_Match:
		alpha(k.Match.GetScrutinee(), scope, depth)
		for _, c := range k.Match.GetCases() {
			inner := hide(bound(c.GetPattern())...)
			alpha(c.GetGuard(), inner, depth)
			alpha(c.GetBody(), inner, depth)
		}
	default:
		// The error is always nil: the visit returns none.
		_ = messages(e.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
			if x, ok := child.Interface().(*umpirespb.Expr); ok {
				alpha(x, scope, depth)
				return false, nil
			}
			return true, nil
		})
	}
}

func bound(p *umpirespb.Pattern) []string {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Bind:
		return append([]string{k.Bind.GetName()}, bound(k.Bind.GetPattern())...)
	case *umpirespb.Pattern_Case:
		var names []string
		for _, field := range k.Case.GetFields() {
			names = append(names, bound(field)...)
		}
		return names
	case *umpirespb.Pattern_Alternatives:
		var names []string
		for _, alternative := range k.Alternatives.GetPatterns() {
			names = append(names, bound(alternative)...)
		}
		return names
	}
	return nil
}

func JSON(v any) ([]byte, error) {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func Proto(m proto.Message) ([]byte, error) {
	encoded, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Compact(&out, encoded); err != nil {
		return nil, err
	}
	return append(out.Bytes(), '\n'), nil
}

func Digest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func Capture(dir string, files map[string][]byte) error {
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if !filepath.IsLocal(name) {
			return fmt.Errorf("unsafe capture path %q", name)
		}
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(path+".gz", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		compressed := gzip.NewWriter(file)
		_, writeErr := compressed.Write(files[name])
		if err := errors.Join(writeErr, compressed.Close(), file.Close()); err != nil {
			return err
		}
	}
	return nil
}

func Read(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(path, ".gz") {
			return fmt.Errorf("unexpected baseline file %s", path)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			return errors.Join(err, file.Close())
		}
		data, readErr := io.ReadAll(reader)
		if err := errors.Join(readErr, reader.Close(), file.Close()); err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out[strings.TrimSuffix(rel, ".gz")] = data
		return nil
	})
	return out, err
}

func Compare(expected, actual map[string][]byte) error {
	for _, key := range slices.Sorted(maps.Keys(expected)) {
		got, exists := actual[key]
		if !exists {
			return fmt.Errorf("missing golden entry %s", key)
		}
		if !bytes.Equal(expected[key], got) {
			return fmt.Errorf("golden difference %s: expected sha256 %s, got %s", key, Digest(expected[key]), Digest(got))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(actual)) {
		if _, exists := expected[key]; !exists {
			return fmt.Errorf("unknown golden entry %s", key)
		}
	}
	return nil
}
