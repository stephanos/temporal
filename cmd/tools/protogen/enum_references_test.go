package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCrossFileEnumReferences(t *testing.T) {
	root := t.TempDir()
	write := func(name, source string) string {
		path := filepath.Join(root, "v1", name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(source), 0644))
		return path
	}
	write("event.pb.go", `package v1

type Kind int32

const (
	KIND_UNSPECIFIED Kind = 0
)

type Parent_Nested int32

const (
	Parent_NESTED_UNSPECIFIED Parent_Nested = 0
)
`)
	run := write("run.pb.go", `package v1

func (x *Event) GetKind() Kind {
	return Kind_KIND_UNSPECIFIED
}

func (x *Event) GetNested() Parent_Nested {
	return Parent_NESTED_UNSPECIFIED
}
`)

	require.NoError(t, resolveCrossFileEnumReferences(root))

	rewritten, err := os.ReadFile(run)
	require.NoError(t, err)
	require.Equal(t, `package v1

func (x *Event) GetKind() Kind {
	return KIND_UNSPECIFIED
}

func (x *Event) GetNested() Parent_Nested {
	return Parent_NESTED_UNSPECIFIED
}
`, string(rewritten))
}
