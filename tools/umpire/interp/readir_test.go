package interp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// readIR decodes ProtoJSON without admission, which depends on the interpreter.
func readIR(t *testing.T, path string) *umpirespb.Model {
	t.Helper()
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	m := &umpirespb.Model{}
	require.NoError(t, protojson.UnmarshalOptions{}.Unmarshal(encoded, m))
	return m
}

func readLifted(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	return readIR(t, filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", name+".json"))
}
