package explore

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func TestTraceRenderingIsStableAndSourceLinked(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	plan, err := New(m, "nexusDeadlines")
	require.NoError(t, err)
	c := plan.Candidates[0]
	require.Empty(t, c.Rejection)
	first, err := RenderTrace(c, plan.Query, nil, nil)
	require.NoError(t, err)
	second, err := RenderTrace(c, plan.Query, nil, nil)
	require.NoError(t, err)
	require.Equal(t, first, second)
	for _, section := range []string{"Product expectation", "System execution", "Monitor state", "Fault decisions", "Evidence", "Holes", ".scala#L"} {
		require.True(t, bytes.Contains(first, []byte(section)), section)
	}
}
