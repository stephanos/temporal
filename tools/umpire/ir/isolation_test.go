package ir

// model owns its Scala sources, tools and IR; the legacy Scala tree they were adopted from is
// an independent baseline, never an input. Every source position the IR carries resolves to a line
// of a file inside model, and no file here names a path in the legacy tree.

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	repoRoot  = "../../.."
	modelRoot = "model"
	// Spelled in two parts so that this file does not name the tree it checks for.
	legacyRoot = "model" + "0"
)

var (
	legacyPath   = regexp.MustCompile(regexp.QuoteMeta(legacyRoot) + `\b`)
	relativePath = regexp.MustCompile(`(?:\.\./)+[\w.-]+(?:/[\w.-]+)*`)
	rejectedAt   = regexp.MustCompile(`^lift: (\S+):(\d+): `)
)

// retiredFrontEnd matches a mention of the front end the model was ported from: its name and the
// suffix of its sources, its build tool, its version manager and its checkers, and its module names.
// The model is described on its own terms; the archive is where its history lives. The words are
// spelled here, outside model, so that the check itself is no mention.
var retiredFrontEnd = regexp.MustCompile(
	`(?i:\blean(?:4|v2)?\b|\blake(?:file)?\b|\belan\b|leanprover|\bveil\b|\bstainless\b)|\bUmpire\.[A-Z]|\bTemporal\.(?:Feature|Case)\b`)

// retiredFrontEndMentions returns where the file at path, with this content, mentions the retired
// front end: "path" for its own name, and "path:line" for each line that does.
func retiredFrontEndMentions(path, content string) []string {
	var found []string
	if retiredFrontEnd.MatchString(path) {
		found = append(found, path)
	}
	for i, line := range strings.Split(content, "\n") {
		if retiredFrontEnd.MatchString(line) {
			found = append(found, fmt.Sprintf("%s:%d", path, i+1))
		}
	}
	return found
}

// sourceProblem says why p does not name a line of a file inside model under root, or "" when
// it does.
func sourceProblem(root string, p *umpirespb.Position) string {
	file := p.GetFile()
	if filepath.IsAbs(file) || filepath.Clean(file) != file || !strings.HasPrefix(file, modelRoot+"/") {
		return fmt.Sprintf("%s:%d is not inside %s", file, p.GetLine(), modelRoot)
	}
	f, err := os.Open(filepath.Join(root, file))
	if err != nil {
		return err.Error()
	}
	defer func() { _ = f.Close() }()
	lines := 0
	for s := bufio.NewScanner(f); s.Scan(); {
		lines++
	}
	if p.GetLine() < 1 || int(p.GetLine()) > lines {
		return fmt.Sprintf("%s has %d lines, not a line %d", file, lines, p.GetLine())
	}
	return ""
}

// legacyReferences returns the lines of content, the text of the file at path, that name the legacy
// tree: by its repository path, or by a relative path that resolves into it from path's directory.
// Each line is read whole, so a line that also names model is still reported.
func legacyReferences(path string, content string) []int {
	var found []int
	inLegacy := func(p string) bool { return p == legacyRoot || strings.HasPrefix(p, legacyRoot+"/") }
	for i, line := range strings.Split(content, "\n") {
		hit := legacyPath.MatchString(line)
		for _, rel := range relativePath.FindAllString(line, -1) {
			hit = hit || inLegacy(filepath.ToSlash(filepath.Join(filepath.Dir(path), rel)))
		}
		if hit {
			found = append(found, i+1)
		}
	}
	return found
}

// positions returns every source position in m.
func positions(m proto.Message) []*umpirespb.Position {
	var out []*umpirespb.Position
	var walk func(protoreflect.Message)
	walk = func(r protoreflect.Message) {
		if p, ok := r.Interface().(*umpirespb.Position); ok {
			out = append(out, p)
			return
		}
		r.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			switch {
			case fd.IsList() && fd.Message() != nil:
				for i := range v.List().Len() {
					walk(v.List().Get(i).Message())
				}
			case fd.IsMap() && fd.MapValue().Message() != nil:
				v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
					walk(mv.Message())
					return true
				})
			case !fd.IsList() && !fd.IsMap() && fd.Message() != nil:
				walk(v.Message())
			default:
				// A scalar holds no position.
			}
			return true
		})
	}
	walk(m.ProtoReflect())
	return out
}

// refusedFixture is the lifter fixture whose realizations the reader refuses, each at its line.
const refusedFixture = "hintsRefused.json"

func TestIRSourcePositionsResolveInsideModel(t *testing.T) {
	models, err := IRPaths(filepath.Join("..", "..", "..", "model", "ir"))
	require.NoError(t, err)
	expected, err := IRPaths(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected"))
	require.NoError(t, err)
	models = append(models, expected...)
	require.NotEmpty(t, models)
	for _, path := range models {
		t.Run(filepath.Base(path), func(t *testing.T) {
			m, err := Load(path)
			if filepath.Base(path) == refusedFixture {
				// The reader refuses it by design (hints_fixture_test.go); its positions resolve all the same.
				require.Error(t, err)
				encoded, readErr := os.ReadFile(path)
				require.NoError(t, readErr)
				m = &umpirespb.Model{}
				err = protojson.Unmarshal(encoded, m)
			}
			require.NoError(t, err)
			ps := positions(m)
			require.NotEmpty(t, ps)
			var problems []string
			for _, p := range ps {
				if problem := sourceProblem(repoRoot, p); problem != "" {
					problems = append(problems, problem)
				}
			}
			require.Empty(t, problems)
		})
	}

	// The declarations the lifter refuses are reported at their positions, as text; a root that names
	// nothing has none.
	rejects, err := os.ReadFile(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "rejects.txt"))
	require.NoError(t, err)
	positioned := 0
	for _, l := range strings.Split(strings.TrimSpace(string(rejects)), "\n") {
		at := rejectedAt.FindStringSubmatch(l)
		if at == nil {
			continue
		}
		positioned++
		line, err := strconv.ParseInt(at[2], 10, 32)
		require.NoError(t, err)
		require.Empty(t, sourceProblem(repoRoot, &umpirespb.Position{File: at[1], Line: int32(line)}), l)
	}
	require.Positive(t, positioned)
}

func TestSourceProblemRejectsPositionsOutsideModel(t *testing.T) {
	inside := "model/temporal/features/nexus/workflow/Workflow.scala"
	require.Empty(t, sourceProblem(repoRoot, &umpirespb.Position{File: inside, Line: 1}))
	for name, p := range map[string]*umpirespb.Position{
		"legacy":       {File: legacyRoot + "/temporal/nexuscaller/NexusCaller.scala", Line: 1},
		"escaping":     {File: "model/../MILESTONES.md", Line: 1},
		"absolute":     {File: "/" + inside, Line: 1},
		"missing":      {File: "model/temporal/features/nexus/workflow/Missing.scala", Line: 1},
		"past the end": {File: inside, Line: 1 << 20},
		"no line":      {File: inside, Line: 0},
		"unnamed":      {Line: 1},
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, sourceProblem(repoRoot, p))
		})
	}
}

func TestLegacyReferencesReadEveryLine(t *testing.T) {
	// Relative paths are resolved from the file they are read in, so they are spelled in parts here too.
	up := "../"
	content := strings.Join([]string{legacyRoot + "/scala/run.sh", "model/temporal", up + legacyRoot + "/scala/gen/model.jar"}, "\n")
	require.Equal(t, []int{1, 3}, legacyReferences("model/project.scala", content))
	require.Empty(t, legacyReferences("model/irgen/project.scala", "//> using jar "+up+"build/ir-proto.jar"))
	require.Equal(t, []int{1}, legacyReferences("model/irgen/project.scala", "//> using jar "+up+up+legacyRoot+"/gen/x.jar"))
}

func TestNoInputNamesTheLegacyTree(t *testing.T) {
	scanned := 0
	require.NoError(t, modelFiles(func(rel, content string) {
		// Prose may describe the legacy tree; nothing builds or reads it.
		if filepath.Ext(rel) == ".md" {
			return
		}
		scanned++
		if lines := legacyReferences(rel, content); len(lines) > 0 {
			t.Errorf("%s names %s at lines %v", rel, legacyRoot, lines)
		}
	}))
	require.Greater(t, scanned, 50, "the walk reaches the sources, the lifter, the IR and the gate")
}

// modelFiles visits every file of the model a build did not write, by its path from the repository's
// root and with its content.
func modelFiles(visit func(rel, content string)) error { return modelFilesUnder(repoRoot, visit) }

// modelBuildOutputs are the gate's build caches under model/: build, and gen, the name it had before
// fn-114.9, which a checkout may still hold until it is cleaned.
var modelBuildOutputs = []string{"build", "gen"}

// modelFilesUnder is modelFiles of the repository at root.
func modelFilesUnder(root string, visit func(rel, content string)) error {
	return filepath.WalkDir(filepath.Join(root, modelRoot), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Build outputs and tool state, regenerated by the gate and scala-cli.
			if filepath.Dir(path) == filepath.Join(root, modelRoot) && slices.Contains(modelBuildOutputs, d.Name()) {
				return filepath.SkipDir
			}
			switch d.Name() {
			case ".scala-build", ".bsp", ".bloop", ".metals":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		visit(filepath.ToSlash(rel), string(content))
		return nil
	})
}

// TestModelNamesNoRetiredFrontEnd is the model gate's vocabulary check: the gate runs it by name on
// every check and update, and fails when it did not run.
func TestModelNamesNoRetiredFrontEnd(t *testing.T) {
	scanned := 0
	var mentions []string
	require.NoError(t, modelFiles(func(rel, content string) {
		scanned++
		mentions = append(mentions, retiredFrontEndMentions(rel, content)...)
	}))
	require.Greater(t, scanned, 50, "the walk reaches the sources, the documents, the IR and the Cases")
	require.Empty(t, mentions, "the model is described on its own terms: reword these, keeping the rule each explains")
}

func TestRetiredFrontEndMentionsAreFound(t *testing.T) {
	const path = "model/umpire/Table.scala"
	for name, test := range map[string]struct {
		path, content string
		mentions      []string
	}{
		"the name":              {path, "// as Lean spells it", []string{path + ":1"}},
		"the name in lowercase": {path, "ok\n// the lean table", []string{path + ":2"}},
		"a source path":         {path, "// Ported from Caller/Model.lean.", []string{path + ":1"}},
		"a versioned tree":      {path, "// see leanv2 and lean4", []string{path + ":1"}},
		"the build tool":        {path, "// run `lake build`", []string{path + ":1"}},
		"the build file":        {path, "// declared in the lakefile", []string{path + ":1"}},
		"the version manager":   {path, "// installed by elan", []string{path + ":1"}},
		"the organization":      {path, "// github.com/leanprover/lean4", []string{path + ":1"}},
		"the checker":           {path, "// Veil visits 171 states", []string{path + ":1"}},
		"the proof checker":     {path, "// Stainless verifies this", []string{path + ":1"}},
		"a module":              {path, "// `Umpire.Command.Compose`", []string{path + ":1"}},
		"a model namespace":     {path, "// below `Temporal.Feature`", []string{path + ":1"}},
		"every line":            {path, "Lean\nfine\nlake", []string{path + ":1", path + ":3"}},
		"a file name":           {"model/umpire/Lean.scala", "package umpire", []string{"model/umpire/Lean.scala"}},
		"a directory":           {"model/lean/Table.scala", "// Lean", []string{"model/lean/Table.scala", "model/lean/Table.scala:1"}},
		"other words":           {path, "a clean Boolean; cleanup; Leander; umpire.Table; Temporal.Features; unveiled; flake", nil},
		"the Scala package":     {path, "import umpire.realize.Operand.*\nval f = umpire.Family(\"temporal.case\")", nil},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.mentions, retiredFrontEndMentions(test.path, test.content))
		})
	}
}
