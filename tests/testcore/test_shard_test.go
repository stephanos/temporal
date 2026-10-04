package testcore

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShardKey(t *testing.T) {
	require.Equal(t, "A", shardKey("A"))
	require.Equal(t, "A/B", shardKey("A/B"))
	require.Equal(t, "A/B", shardKey("A/B/C"))
	require.Equal(t, "A/B", shardKey("A/B/C/D"))
}

// A depth-3 subtest must be skipped or run exactly as its depth-2 parent is.
func TestCheckTestShardFollowsDepth2Parent(t *testing.T) {
	const total = 5
	t.Setenv("TEST_TOTAL_SHARDS", strconv.Itoa(total))

	t.Run("parent", func(t *testing.T) {
		expected := shardIndex(t.Name(), total)
		for index := range total {
			t.Setenv("TEST_SHARD_INDEX", strconv.Itoa(index))
			// Repeated runs get distinct depth-3 names (child, child#01, ...),
			// which must all land in the parent's shard.
			var child *testing.T
			t.Run("child", func(t *testing.T) {
				child = t
				CheckTestShard(t)
			})
			require.NotNil(t, child)
			require.Equal(t, index != expected, child.Skipped(), "shard index %d, parent runs in %d", index, expected)
		}
	})
}
