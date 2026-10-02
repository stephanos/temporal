package goir

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReaderAndLowererDoNotImportHandwrittenOracles(t *testing.T) {
	for _, directory := range []string{".", "testpilot"} {
		files, err := filepath.Glob(filepath.Join(directory, "*.go"))
		require.NoError(t, err)
		require.NotEmpty(t, files)
		for _, file := range files {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			require.NoError(t, err)
			for _, imported := range parsed.Imports {
				path, err := strconv.Unquote(imported.Path.Value)
				require.NoError(t, err)
				for _, oracle := range []string{"standaloneactivity", "nexuscaller", "worker"} {
					require.NotEqual(t, "go.temporal.io/server/model/go/"+oracle, path, "%s imports a retired oracle", file)
				}
			}
		}
	}
}
