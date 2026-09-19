package cli

import "testing"

func TestQuoteArgumentProducesOnePOSIXShellWord(t *testing.T) {
	for input, want := range map[string]string{
		"simple/path": "simple/path",
		"":            "''",
		"two words":   "'two words'",
		"a'b":         "'a'\"'\"'b'",
		"$HOME":       "'$HOME'",
	} {
		if got := quoteArgument(input); got != want {
			t.Fatalf("quoteArgument(%q) = %q, want %q", input, got, want)
		}
	}
}
