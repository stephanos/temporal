package artifactv2

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// Lean owns these vocabularies and writes them into every artifact. Reading the Lean source is
// the only way this package learns that a rename happened: a stale list here decodes a valid
// artifact as invalid, which no fixture catches until a producer emits the renamed value.
func TestKindVocabulariesMatchLean(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		relative string
		function string
		want     []string
	}{
		{
			name:     "known gap kinds",
			relative: "model/Umpire/KnownGap.lean",
			function: "KnownGapKind",
			want:     knownGapKinds,
		},
		{
			name:     "definition kinds",
			relative: "model/Umpire/Core.lean",
			function: "DefinitionKind",
			want:     definitionKinds,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, leanNameArms(t, test.relative, test.function))
		})
	}
}

// leanNameArms returns the string literals of `def <function>.name : <function> -> String`, in
// declaration order, which is also the canonical rank order both languages sort by.
func leanNameArms(t *testing.T, relative, function string) []string {
	t.Helper()

	source := string(readRepositoryFile(t, relative))
	head := regexp.MustCompile(`(?m)^def ` + function + `\.name : ` + function + ` → String\n`)
	start := head.FindStringIndex(source)
	require.NotNil(t, start, "no name function for %s in %s", function, relative)

	arms := regexp.MustCompile(`(?m)\A(?:  \| \.\w+ => "([^"]+)"\n)+`).
		FindString(source[start[1]:])
	require.NotEmpty(t, arms, "no arms after the %s name function head", function)

	var names []string
	for _, match := range regexp.MustCompile(`"([^"]+)"`).FindAllStringSubmatch(arms, -1) {
		names = append(names, match[1])
	}
	return names
}
