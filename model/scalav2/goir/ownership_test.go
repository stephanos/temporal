package goir

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/scalav2/goir/internal/golden"
)

func TestCheckerAndProducerHaveOneLiveOwner(t *testing.T) {
	root, err := golden.Root()
	require.NoError(t, err)
	base := filepath.Join(root, "model", "scalav2")
	require.NoError(t, filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil { return err }
		if entry.IsDir() || !strings.HasSuffix(path, ".go") { return nil }
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil { return err }
		for _, imported := range parsed.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil { return err }
			require.NotEqual(t, "go.temporal.io/server/model/go/"+"umpire", name, path)
			require.NotEqual(t, "go.temporal.io/server/model/go/caseproducer", name, path)
			if strings.HasSuffix(name, "/internal/checker") {
				rel, err := filepath.Rel(filepath.Join(base, "goir"), path)
				if err != nil { return err }
				require.True(t, filepath.Dir(rel) == "." || strings.HasPrefix(rel, "internal/checker/"), "%s imports the private checker", path)
			}
			if strings.HasSuffix(name, "/internal/producer") {
				require.True(t, strings.HasPrefix(path, filepath.Join(base, "goir", "testpilot")+string(filepath.Separator)), "%s imports the private producer", path)
			}
		}
		return nil
	}))
}
