package model

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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
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

func TestIRSourcePositionsResolveInsideModel(t *testing.T) {
	models, err := filepath.Glob(filepath.Join("..", "..", "..", "model", "ir", "*.json"))
	require.NoError(t, err)
	expected, err := filepath.Glob(filepath.Join("..", "..", "..", "model", "lifter", "testdata", "lifts", "expected", "*.json"))
	require.NoError(t, err)
	models = append(models, expected...)
	require.NotEmpty(t, models)
	for _, path := range models {
		t.Run(filepath.Base(path), func(t *testing.T) {
			m, err := Load(path)
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
	rejects, err := os.ReadFile(filepath.Join("..", "..", "..", "model", "lifter", "testdata", "lifts", "expected", "rejects.txt"))
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
	inside := "model/temporal/nexuscaller/Model.scala"
	require.Empty(t, sourceProblem(repoRoot, &umpirespb.Position{File: inside, Line: 1}))
	for name, p := range map[string]*umpirespb.Position{
		"legacy":       {File: legacyRoot + "/temporal/nexuscaller/Model.scala", Line: 1},
		"escaping":     {File: "model/../model0/scala/temporal/nexuscaller/Model.scala", Line: 1},
		"absolute":     {File: "/" + inside, Line: 1},
		"missing":      {File: "model/temporal/nexuscaller/Missing.scala", Line: 1},
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
	require.Empty(t, legacyReferences("model/lifter/project.scala", "//> using jar "+up+"gen/ir-proto.jar"))
	require.Equal(t, []int{1}, legacyReferences("model/lifter/project.scala", "//> using jar "+up+up+legacyRoot+"/gen/x.jar"))
}

func TestNoInputNamesTheLegacyTree(t *testing.T) {
	scanned := 0
	root := filepath.Join(repoRoot, modelRoot)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Build outputs and tool state, regenerated by the gate and scala-cli.
			switch d.Name() {
			case "gen", ".scala-build", ".bsp", ".bloop", ".metals":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		// Prose may describe the legacy tree; nothing builds or reads it.
		if filepath.Ext(path) == ".md" {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		if lines := legacyReferences(filepath.ToSlash(rel), string(content)); len(lines) > 0 {
			t.Errorf("%s names %s at lines %v", rel, legacyRoot, lines)
		}
		return nil
	})
	require.NoError(t, err)
	require.Greater(t, scanned, 50, "the walk reaches the sources, the lifter, the IR and the gate")
}
