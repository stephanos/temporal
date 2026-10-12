package record

import (
	"fmt"
	"strings"
)

const SemanticCoverageSchema = "gomad3.semantic-coverage/v1"

const (
	// DefaultIOTranscriptBytes is the I/O transcript a campaign gets unless it
	// asks for more; a cluster-sized target can exceed it.
	DefaultIOTranscriptBytes = 64 << 20
	// MaximumIOTranscriptBytes bounds any requested transcript; the runtime
	// refuses a larger backing.
	MaximumIOTranscriptBytes = 1 << 30
	// IOTranscriptBytesGranularity keeps requested limits on whole mebibytes.
	IOTranscriptBytesGranularity = 1 << 20
)

// ValidateIOTranscriptLimit checks a campaign's requested I/O transcript bound.
func ValidateIOTranscriptLimit(limit uint64) error {
	if limit < DefaultIOTranscriptBytes || limit > MaximumIOTranscriptBytes || limit%IOTranscriptBytesGranularity != 0 {
		return fmt.Errorf("I/O transcript limit %d must be a whole number of MiB between %d and %d", limit, uint64(DefaultIOTranscriptBytes), uint64(MaximumIOTranscriptBytes))
	}
	return nil
}

type IOContract struct {
	ImplementationSHA256 SHA256 `json:"implementation_sha256"`
	InventorySHA256      SHA256 `json:"inventory_sha256"`
	Name                 string `json:"name"`
}

type MissingSemanticProbesError struct {
	Probes []string
}

type SemanticCoverage struct {
	Schema string   `json:"schema"`
	Digest SHA256   `json:"digest"`
	Probes []string `json:"probes"`
}

func (err *MissingSemanticProbesError) Error() string {
	return "required semantic probes were not observed: " + strings.Join(err.Probes, ", ")
}
