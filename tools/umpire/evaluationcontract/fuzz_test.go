package evaluationcontract

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzAdmitRejectsSingleByteContractMutations(f *testing.F) {
	canonicalJSON, err := CanonicalProtoJSON(testContract())
	require.NoError(f, err)
	canonical, err := Pack(canonicalJSON)
	require.NoError(f, err)
	f.Add(uint32(0), byte(1))
	f.Add(uint32(len(canonical)/2), byte(0x80))
	f.Add(uint32(len(canonical)-1), byte(0xff))

	f.Fuzz(func(t *testing.T, offset uint32, delta byte) {
		if delta == 0 {
			t.Skip()
		}
		mutated := bytes.Clone(canonical)
		mutated[int(offset)%len(mutated)] ^= delta
		_, err := Admit(mutated)
		require.Error(t, err)
	})
}
