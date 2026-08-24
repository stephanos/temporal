package documentationaudit

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestAuditRejectsIncompletePublishedDocumentEvidence(t *testing.T) {
	files := fstest.MapFS{
		"README.md":       {Data: []byte("# Umpire3\n")},
		"docs/support.md": {Data: []byte("# Support\n")},
	}
	report, err := Audit(files)
	require.NoError(t, err)
	require.Equal(t, []Document{
		{Name: "README.md", Bytes: 10, Digest: documentationDigest([]byte("# Umpire3\n"))},
		{Name: "docs/support.md", Bytes: 10, Digest: documentationDigest([]byte("# Support\n"))},
	}, report.Documents)

	report.Documents[0].Bytes = 0
	report.ArtifactDigest = report.computedDigest()
	require.ErrorContains(t, report.Validate(), "incomplete")
}
