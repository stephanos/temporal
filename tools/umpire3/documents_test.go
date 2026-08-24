package umpire3

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/assurance/audit/documentationaudit"
)

func TestDocumentationAuditBindsEveryPublishedDocument(t *testing.T) {
	report, err := documentationaudit.Audit(PublishedDocumentation)
	require.NoError(t, err)
	entries, err := os.ReadDir("docs")
	require.NoError(t, err)
	published := []string{"README.md"}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			published = append(published, filepath.ToSlash(filepath.Join("docs", entry.Name())))
		}
	}
	audited := make([]string, len(report.Documents))
	for index, document := range report.Documents {
		audited[index] = document.Name
	}
	require.Equal(t, published, audited)
	require.NoError(t, report.Validate())
}
