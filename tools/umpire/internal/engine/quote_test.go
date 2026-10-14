package engine_test

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
)

// encoded is how Quote wrote every string before it wrote plain ones itself: one JSON encoder for
// each, with HTML escaping off.
func encoded(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}

// adversarial is texts whose JSON spelling differs from their bytes, or nearly does: every byte on
// its own, the escapes JSON writes short and long, the HTML characters left as they are, the line
// and paragraph separators JSON escapes, invalid and truncated UTF-8, and those mixed with plain text.
func adversarial() []string {
	out := []string{"", "temporal.activity.state.idle", "a b", "~", " ", "<script>&amp;</script>", `"`, `\`, `\"`,
		"\u2028", "\u2029", "x\u2028y\u2029z", "\u007f", "\u0080", "\u00e9", "e\u0301", "\u65e5\u672c\u8a9e", "\U0001f600", "\ufeff", "\ufffd",
		"\xff", "\xc3", "\xc3\x28", "\xe2\x80", "\xe2\x80\xa8", "\xed\xa0\x80", "\xf0\x9f\x98", "\xf4\x90\x80\x80",
		"\x00", "\b\f\n\r\t", "\x1b[31m", "a\x00b", "plain\xffafter", "<>&'", "\u0000\u001f "}
	for c := range 256 {
		out = append(out, string([]byte{byte(c)}), "id."+string([]byte{byte(c)})+".tail")
	}
	return out
}

func TestQuoteWritesWhatTheEncoderWrote(t *testing.T) {
	for _, s := range adversarial() {
		require.Equal(t, encoded(s), umpire.Quote(s), "%q", s)
	}
	// Every pair of bytes, so each byte is also read after each other one.
	pair := make([]byte, 2)
	for a := range 256 {
		for b := range 256 {
			pair[0], pair[1] = byte(a), byte(b)
			require.Equal(t, encoded(string(pair)), umpire.Quote(string(pair)), "%q", pair)
		}
	}
	// Longer texts drawn mostly from the bytes and runes that are escaped, with a fixed seed.
	alphabet := []string{"a", "Z", "0", ".", "-", " ", "<", ">", "&", `"`, `\`, "/", "\x00", "\x1f", "\n", "\x7f",
		"\xff", "\xc3", "\u2028", "\u2029", "\u00e9", "\U0001f600", "\ufffd"}
	rng := rand.New(rand.NewPCG(2026, 10))
	for range 20000 {
		var b strings.Builder
		for range rng.IntN(24) {
			b.WriteString(alphabet[rng.IntN(len(alphabet))])
		}
		require.Equal(t, encoded(b.String()), umpire.Quote(b.String()), "%q", b.String())
	}
}

func FuzzQuote(f *testing.F) {
	for _, s := range adversarial() {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		require.Equal(t, encoded(s), umpire.Quote(s), "%q", s)
	})
}
