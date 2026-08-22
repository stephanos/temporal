package documentationaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

const FormatVersion = "umpire3/documentation-audit/v1"

type Document struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	Digest string `json:"digest"`
}

type Report struct {
	FormatVersion  string     `json:"formatVersion"`
	Documents      []Document `json:"documents"`
	ArtifactDigest string     `json:"artifactDigest"`

	expectedNames []string
}

func Audit(files fs.FS) (Report, error) {
	names, err := publishedNames(files)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		FormatVersion: FormatVersion,
		Documents:     make([]Document, 0, len(names)),
		expectedNames: names,
	}
	for _, name := range names {
		contents, err := fs.ReadFile(files, name)
		if err != nil {
			return Report{}, fmt.Errorf("read Umpire3 documentation %q: %w", name, err)
		}
		if len(strings.TrimSpace(string(contents))) == 0 {
			return Report{}, fmt.Errorf("Umpire3 documentation %q is empty", name)
		}
		report.Documents = append(report.Documents, Document{
			Name: name, Bytes: len(contents), Digest: documentationDigest(contents),
		})
	}
	report.ArtifactDigest = report.computedDigest()
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r Report) Validate() error {
	if r.FormatVersion != FormatVersion || len(r.expectedNames) == 0 ||
		len(r.Documents) != len(r.expectedNames) ||
		!slices.IsSortedFunc(r.Documents, func(left, right Document) int {
			return strings.Compare(left.Name, right.Name)
		}) {
		return errors.New("documentation audit requires the complete sorted document set")
	}
	for index, document := range r.Documents {
		if document.Name != r.expectedNames[index] || document.Bytes <= 0 || !validDocumentationDigest(document.Digest) {
			return fmt.Errorf("documentation audit entry %q is incomplete", document.Name)
		}
	}
	if r.ArtifactDigest != r.computedDigest() {
		return errors.New("documentation audit digest does not match its contents")
	}
	return nil
}

func publishedNames(files fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(files, "docs")
	if err != nil {
		return nil, fmt.Errorf("read Umpire3 documentation set: %w", err)
	}
	names := make([]string, 0, len(entries)+1)
	if _, err := fs.Stat(files, "README.md"); err != nil {
		return nil, fmt.Errorf("read Umpire3 README: %w", err)
	}
	names = append(names, "README.md")
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, "docs/"+entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

func (r Report) computedDigest() string {
	canonical := r
	canonical.ArtifactDigest = ""
	encoded, _ := json.Marshal(canonical)
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
