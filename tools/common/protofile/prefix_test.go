package protofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "trim whitespace", value: "  temporal/api  ", want: "temporal/api"},
		{name: "normalize separators", value: `temporal\api\workflow`, want: "temporal/api/workflow"},
		{name: "preserve trailing slash", value: ` temporal\api\ `, want: "temporal/api/"},
		{name: "clean path", value: "temporal//api/./workflow", want: "temporal/api/workflow"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizePrefix(test.value)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestNormalizePrefixRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: "  "},
		{name: "absolute", value: "/temporal/api"},
		{name: "current directory", value: "."},
		{name: "parent", value: ".."},
		{name: "parent traversal", value: "../temporal/api"},
		{name: "cleaned parent traversal", value: "temporal/../../api"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizePrefix(test.value)
			require.Error(t, err)
		})
	}
}
