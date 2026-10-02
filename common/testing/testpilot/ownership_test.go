package testpilot_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeHelpersDoNotImportModelTooling(t *testing.T) {
	for _, root := range []string{".", "../../../tools/canary"} {
		forbiddenImports := []string{
			"go.temporal.io/server/tools/umpire",
			"go.temporal.io/server/model",
			"go.temporal.io/server/api/umpire",
		}
		if root == "." {
			forbiddenImports = append(forbiddenImports, "go.temporal.io/server/tests")
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			source, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range source.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				require.NoError(t, err)
				for _, forbidden := range forbiddenImports {
					require.False(t, imported == forbidden || strings.HasPrefix(imported, forbidden+"/") || strings.HasPrefix(imported, forbidden+"0/"), "%s imports %s", path, imported)
				}
			}
			return nil
		})
		require.NoError(t, err)
	}
}
