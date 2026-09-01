package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validIndexJSON = `{
  "format": "umpire-plan-index/v1",
  "documents": [
    {
      "path": ".plans/A.md",
      "lifecycle": "active",
      "authority": "normative-rules",
      "authorityParents": [],
      "supersededBy": null,
      "allowedMissingLinks": []
    }
  ],
  "flowSpecs": [
    {
      "id": "fn-1-example",
      "scope": "umpire-roadmap",
      "disposition": "retained",
      "phase": "p0",
      "status": "open",
      "ready": true,
      "completionReview": "unknown",
      "specDependencies": []
    }
  ]
}`

func TestParseIndex(t *testing.T) {
	got, err := parseIndex([]byte(validIndexJSON))
	require.NoError(t, err)
	require.Equal(t, planIndex{
		Format: "umpire-plan-index/v1",
		Documents: []documentEntry{{
			Path:                ".plans/A.md",
			Lifecycle:           "active",
			Authority:           "normative-rules",
			AuthorityParents:    []string{},
			AllowedMissingLinks: []allowedMissingLink{},
		}},
		FlowSpecs: []flowSpecEntry{{
			ID:               "fn-1-example",
			Scope:            "umpire-roadmap",
			Disposition:      "retained",
			Phase:            "p0",
			Status:           "open",
			Ready:            true,
			CompletionReview: "unknown",
			SpecDependencies: []string{},
		}},
	}, got)
}

func TestParseIndexRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		wantErr string
	}{
		{
			name:    "malformed",
			encoded: `{"format":`,
			wantErr: "decode plan index: unexpected end of JSON input",
		},
		{
			name:    "duplicate key",
			encoded: strings.Replace(validIndexJSON, `"format": "umpire-plan-index/v1"`, `"format": "umpire-plan-index/v1", "format": "umpire-plan-index/v1"`, 1),
			wantErr: `$: duplicate field "format"`,
		},
		{
			name:    "nested duplicate key",
			encoded: strings.Replace(validIndexJSON, `"lifecycle": "active"`, `"lifecycle": "active", "lifecycle": "reference"`, 1),
			wantErr: `$.documents[0]: duplicate field "lifecycle"`,
		},
		{
			name:    "unknown field",
			encoded: strings.Replace(validIndexJSON, `"format": "umpire-plan-index/v1"`, `"format": "umpire-plan-index/v1", "extra": true`, 1),
			wantErr: `$: unknown field "extra"`,
		},
		{
			name:    "unsupported version",
			encoded: strings.Replace(validIndexJSON, "umpire-plan-index/v1", "umpire-plan-index/v2", 1),
			wantErr: `$.format: unsupported value "umpire-plan-index/v2"`,
		},
		{
			name:    "unsupported enum",
			encoded: strings.Replace(validIndexJSON, `"lifecycle": "active"`, `"lifecycle": "retired"`, 1),
			wantErr: `$.documents[0].lifecycle: unsupported value "retired"`,
		},
		{
			name:    "wrong type",
			encoded: `{"format":"umpire-plan-index/v1","documents":"wrong","flowSpecs":[]}`,
			wantErr: `$.documents: expected array, got string`,
		},
		{
			name:    "illegal null",
			encoded: strings.Replace(validIndexJSON, `"path": ".plans/A.md"`, `"path": null`, 1),
			wantErr: `$.documents[0].path: expected string, got null`,
		},
		{
			name: "missing field",
			encoded: strings.Replace(validIndexJSON, `      "ready": true,
`, "", 1),
			wantErr: `$.flowSpecs[0]: missing field "ready"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseIndex([]byte(test.encoded))
			require.EqualError(t, err, test.wantErr)
		})
	}
}
