package umpire3

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentationAuditBindsEveryPublishedDocument(t *testing.T) {
	report, err := AuditDocumentation()
	require.NoError(t, err)
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var published []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			published = append(published, entry.Name())
		}
	}
	audited := make([]string, len(report.Documents))
	for index, document := range report.Documents {
		audited[index] = document.Name
	}
	require.Equal(t, published, audited)
	require.NoError(t, report.Validate())

	report.Documents[0].Bytes = 0
	report.ArtifactDigest = report.computedDigest()
	require.ErrorContains(t, report.Validate(), "incomplete")
}
