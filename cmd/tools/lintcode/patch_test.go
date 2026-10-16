package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopePatch(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, patch, want string
		invalid           bool
	}{
		{"edited rename", "diff --git a/tools/gomad-legacy/a.go b/tools/gomad/a.go\nsimilarity index 90%\nrename from tools/gomad-legacy/a.go\nrename to tools/gomad/a.go\n--- a/tools/gomad-legacy/a.go\n+++ b/tools/gomad/a.go\n@@ -10 +10 @@\n-old\n+new\n", "+++ b/a.go\n@@ -10 +10 @@\n-\n+\n", false},
		{"header shaped source line", "+++ b/tools/gomad/a.go\n@@ -0,0 +1,2 @@\n+++ b/tools/gomad/other.go\n+last\n", "+++ b/a.go\n@@ -0,0 +1,2 @@\n+\n+\n", false},
		{"quoted filename", "+++ \"b/tools/gomad/a\\\"b.go\"\n@@ -0,0 +1 @@\n+new\n", "+++ b/a\"b.go\n@@ -0,0 +1 @@\n+\n", false},
		{"space filename", "+++ b/tools/gomad/a b.go\n@@ -0,0 +1 @@\n+new\n", "+++ b/a b.go\n@@ -0,0 +1 @@\n+\n", false},
		{"Git space filename terminator", "+++ b/tools/gomad/a b.go\t\n@@ -0,0 +1 @@\n+new\n", "+++ b/a b.go\n@@ -0,0 +1 @@\n+\n", false},
		{"old file lacked final newline", "+++ b/tools/gomad/a.go\n@@ -1 +1 @@\n-old\n\\ No newline at end of file\n+new\n", "+++ b/a.go\n@@ -1 +1 @@\n-\n+\n", false},
		{"other owner", "+++ b/tests/mixedbrain/a.go\n@@ -0,0 +1 @@\n+new\n", "", false},
		{"pure rename", "diff --git a/tools/gomad-legacy/a.go b/tools/gomad/a.go\nsimilarity index 100%\nrename from tools/gomad-legacy/a.go\nrename to tools/gomad/a.go\n", "", false},
		{"deletion", "+++ /dev/null\n@@ -1 +0,0 @@\n-old\n", "", false},
		{"truncated", "+++ b/tools/gomad/a.go\n@@ -0,0 +1,2 @@\n+new\n", "", true},
		{"extra source line", "+++ b/tools/gomad/a.go\n@@ -0,0 +1 @@\n+new\n+extra\n", "", true},
		{"invalid header", "+++ b/tools/gomad/a.go\n@@ broken\n", "", true},
		{"invalid quote", "+++ \"b/tools/gomad/a.go\n", "", true},
		{"newline filename", "+++ \"b/tools/gomad/a\\nb.go\"\n", "", true},
		{"carriage return filename", "+++ \"b/tools/gomad/a\\rb.go\"\n", "", true},
		{"unsupported patch", "invalid patch\n", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := scopePatch(tc.patch, "tools/gomad")
			if tc.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
