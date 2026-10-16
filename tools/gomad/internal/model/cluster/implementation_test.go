//go:build !gomad_toolchain

package cluster

import (
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestImplementationIdentityBindsAllSemanticSources(t *testing.T) {
	var names []string
	require.NoError(t, fs.WalkDir(os.DirFS("."), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
		return nil
	}))
	require.Equal(t, names, implementationSourceNames)
	baseline := ImplementationSHA256()
	require.NotEmpty(t, baseline)
	sources := fstest.MapFS{}
	for _, name := range implementationSourceNames {
		source, err := fs.ReadFile(implementationSources, name)
		require.NoError(t, err)
		sources[name] = &fstest.MapFile{Data: source}
	}
	require.Equal(t, baseline, implementationSHA256(sources))
	for _, name := range implementationSourceNames {
		t.Run(name, func(t *testing.T) {
			original := sources[name]
			sources[name] = &fstest.MapFile{Data: append(append([]byte(nil), original.Data...), '\n')}
			require.NotEqual(t, baseline, implementationSHA256(sources))
			delete(sources, name)
			require.Empty(t, implementationSHA256(sources))
			sources[name] = original
		})
	}
}
