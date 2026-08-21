package umpire3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentationAuditBindsEveryPublishedDocument(t *testing.T) {
	report, err := AuditDocumentation()
	require.NoError(t, err)
	require.Len(t, report.Documents, 9)
	require.NoError(t, report.Validate())

	report.Documents[0].Bytes = 0
	report.ArtifactDigest = report.computedDigest()
	require.ErrorContains(t, report.Validate(), "incomplete")
}
