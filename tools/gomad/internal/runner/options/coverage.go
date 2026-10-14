package options

import (
	"errors"
	"fmt"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
)

type CoverageMode string

const (
	CoverageNone           CoverageMode = "none"
	CoverageSemantic       CoverageMode = "semantic"
	CoverageChoice         CoverageMode = "choice"
	CoverageSemanticChoice CoverageMode = "semantic+choice"
)

type SemanticCoverageRequiredError struct{}

func (*SemanticCoverageRequiredError) Error() string {
	return "required semantic probes require semantic coverage"
}

func NormalizeCoverage(mode CoverageMode, guided bool) CoverageMode {
	if mode == "" {
		if guided {
			return CoverageSemantic
		}
		return CoverageNone
	}
	return mode
}

func ValidateCoverage(mode CoverageMode, required []string) error {
	switch mode {
	case "", CoverageNone, CoverageChoice:
		if len(required) != 0 {
			return &SemanticCoverageRequiredError{}
		}
	case CoverageSemantic, CoverageSemanticChoice:
	default:
		return fmt.Errorf("unknown coverage mode %q", mode)
	}
	return nil
}

func ValidateChoiceTraceLimit(limit uint64) error {
	if limit != 0 && (limit < backend.MinimumChoiceTraceBytes || limit > backend.MaximumChoiceTraceBytes) {
		return fmt.Errorf("choice trace capacity must be between %d bytes and 64 MiB", backend.MinimumChoiceTraceBytes)
	}
	return nil
}

func ValidateChoiceCoverage(mode CoverageMode, choiceTraceLimit uint64) error {
	if (mode == CoverageChoice || mode == CoverageSemanticChoice) && choiceTraceLimit == 0 {
		return errors.New("choice coverage requires an enabled choice trace")
	}
	return nil
}

// SupportedCoverageModes lists the coverage modes a backend with the given evidence can satisfy.
func SupportedCoverageModes(semantic, choices bool) []CoverageMode {
	modes := []CoverageMode{CoverageNone}
	if semantic {
		modes = append(modes, CoverageSemantic)
	}
	if choices {
		modes = append(modes, CoverageChoice)
	}
	if semantic && choices {
		modes = append(modes, CoverageSemanticChoice)
	}
	return modes
}
