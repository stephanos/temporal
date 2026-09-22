package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlattenRendersEachLeafInOrder(t *testing.T) {
	require.Equal(t, []string{"one"}, Flatten(errors.New("one")))
	joined := errors.Join(errors.New("one"), errors.Join(errors.New("two"), errors.New("three")))
	require.Equal(t, []string{"one", "two", "three"}, Flatten(joined))
}
