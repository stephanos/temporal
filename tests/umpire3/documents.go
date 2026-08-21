package umpire3

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const DocumentationFormatVersion = "umpire3/documentation-audit/v1"

var documentationNames = []string{
	"AUTHORING.md",
	"CONTEXT.md",
	"IMPLEMENTATION_VERIFICATION.md",
	"INCIDENT_RECOVERY.md",
	"MODELING.md",
	"OPERATIONS.md",
	"README.md",
	"SECURITY.md",
	"SUPPORT.md",
}

//go:embed AUTHORING.md CONTEXT.md IMPLEMENTATION_VERIFICATION.md INCIDENT_RECOVERY.md MODELING.md OPERATIONS.md README.md SECURITY.md SUPPORT.md
var documentationFiles embed.FS

type DocumentationDocument struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	Digest string `json:"digest"`
}

type DocumentationReport struct {
	FormatVersion  string                  `json:"formatVersion"`
	Documents      []DocumentationDocument `json:"documents"`
	ArtifactDigest string                  `json:"artifactDigest"`
}

func AuditDocumentation() (DocumentationReport, error) {
	report := DocumentationReport{
		FormatVersion: DocumentationFormatVersion,
		Documents:     make([]DocumentationDocument, 0, len(documentationNames)),
	}
	for _, name := range documentationNames {
		contents, err := documentationFiles.ReadFile(name)
		if err != nil {
			return DocumentationReport{}, fmt.Errorf("read Umpire3 documentation %q: %w", name, err)
		}
		if len(strings.TrimSpace(string(contents))) == 0 {
			return DocumentationReport{}, fmt.Errorf("Umpire3 documentation %q is empty", name)
		}
		report.Documents = append(report.Documents, DocumentationDocument{
			Name: name, Bytes: len(contents), Digest: documentationDigest(contents),
		})
	}
	report.ArtifactDigest = report.computedDigest()
	if err := report.Validate(); err != nil {
		return DocumentationReport{}, err
	}
	return report, nil
}

func (r DocumentationReport) Validate() error {
	if r.FormatVersion != DocumentationFormatVersion || len(r.Documents) != len(documentationNames) ||
		!slices.IsSortedFunc(r.Documents, func(left, right DocumentationDocument) int {
			return strings.Compare(left.Name, right.Name)
		}) {
		return errors.New("documentation audit requires the complete sorted document set")
	}
	for index, document := range r.Documents {
		if document.Name != documentationNames[index] || document.Bytes <= 0 || !validDocumentationDigest(document.Digest) {
			return fmt.Errorf("documentation audit entry %q is incomplete", document.Name)
		}
	}
	if r.ArtifactDigest != r.computedDigest() {
		return errors.New("documentation audit digest does not match its contents")
	}
	return nil
}

func (r DocumentationReport) computedDigest() string {
	r.ArtifactDigest = ""
	encoded, _ := json.Marshal(r)
	return documentationDigest(encoded)
}

func documentationDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validDocumentationDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
