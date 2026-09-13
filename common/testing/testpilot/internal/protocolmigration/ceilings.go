package protocolmigration

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// profileCeilings is the resource ceiling set of each Profile a fixture's bounds moved into when
// ceilings left the Case, keyed by the baseline limits message and its ProtoJSON field. An
// InstructionLimits entry is the Profile's per-instruction ceiling the dropped instruction bound
// moved to (max_instruction_emitted_events and max_instruction_response_bytes). A dropped bound is
// admitted only when its value lies within its Profile's ceiling, so a bound tighter than the
// ceiling is a declared loosening and one above it fails the step.
var profileCeilings = map[string]map[protoreflect.FullName]map[string]int64{
	// temporal.DefaultCeilings, the Profile every Temporal Case and the facade conformance corpus
	// run under.
	"temporal": {
		protocol + "ProgramLimits": {
			"maxEntrypoints": 4, "maxNodes": 16, "maxEdges": 24, "maxActivations": 8, "maxAttempts": 16,
			"maxRunEvents": 512, "maxExpressionDepth": 12, "maxPathFanout": 32,
			"maxRequestBytes": 32768, "maxResponseBytes": 8192,
			"maxTotalDurationMilliseconds": 30000, "maxCleanupDurationMilliseconds": 20000,
		},
		protocol + "InstructionLimits": {"maxEmittedEvents": 128, "maxResponseBytes": 8192},
		protocol + "ContractLimits": {
			"maxRules": 4, "maxStates": 16, "maxTransitions": 64, "maxExpressionDepth": 12,
			"maxWorkPerEvent": 4000000, "maxTotalWork": 1000000000, "maxCaptures": 64, "maxCaptureBytes": 65536,
		},
		protocol + "CorrelatedLimits": {
			"maxEvents": 64, "maxBuffered": 32, "maxKeys": 8, "maxSupport": 256, "maxProjectionWork": 1000000000,
			"maxEventBytes": 512, "maxSemanticTransitions": 32, "maxObligations": 16, "maxObligationWork": 100000000,
			"maxCaptures": 16, "maxCorrelationDepth": 2,
		},
	},
	// The synthetic Case's own former bounds, which its no-I/O admission test keeps as its Profile.
	"synthetic": {
		protocol + "ProgramLimits": {
			"maxEntrypoints": 1, "maxNodes": 1, "maxEdges": 1, "maxActivations": 1, "maxAttempts": 1,
			"maxRunEvents": 8, "maxExpressionDepth": 8, "maxPathFanout": 4,
			"maxRequestBytes": 1024, "maxResponseBytes": 1024,
			"maxTotalDurationMilliseconds": 1000, "maxCleanupDurationMilliseconds": 1000,
		},
		protocol + "InstructionLimits": {"maxEmittedEvents": 1, "maxResponseBytes": 1024},
		protocol + "ContractLimits": {
			"maxRules": 1, "maxStates": 2, "maxTransitions": 1, "maxExpressionDepth": 8,
			"maxWorkPerEvent": 32, "maxTotalWork": 64, "maxCaptures": 1, "maxCaptureBytes": 1024,
		},
	},
	// The correlated corpus's runnable Cases read one evidence value per instruction, so their Profile
	// admits more nodes and Run Events than any Temporal Case.
	"correlated": {
		protocol + "ProgramLimits": {
			"maxEntrypoints": 4, "maxNodes": 256, "maxEdges": 256, "maxActivations": 256, "maxAttempts": 256,
			"maxRunEvents": 2048, "maxExpressionDepth": 8, "maxPathFanout": 256,
			"maxRequestBytes": 4096, "maxResponseBytes": 4096,
			"maxTotalDurationMilliseconds": 10000, "maxCleanupDurationMilliseconds": 1000,
		},
		protocol + "InstructionLimits": {"maxEmittedEvents": 1, "maxResponseBytes": 4096},
		protocol + "ContractLimits": {
			"maxRules": 16, "maxStates": 32, "maxTransitions": 64, "maxExpressionDepth": 16,
			"maxWorkPerEvent": 100000, "maxTotalWork": 1000000000, "maxCaptures": 32, "maxCaptureBytes": 65536,
		},
		protocol + "CorrelatedLimits": {
			"maxEvents": 16, "maxBuffered": 8, "maxKeys": 8, "maxSupport": 256, "maxProjectionWork": 1000000000,
			"maxEventBytes": 512, "maxSemanticTransitions": 32, "maxObligations": 16, "maxObligationWork": 1000000000,
		},
	},
}

// fixtureProfile names the Profile of profileCeilings each fixture's dropped bounds moved into.
func fixtureProfile(fixture string) (string, error) {
	switch {
	case fixture == functionalFixtureRoot+"/synthetic-case.json":
		return "synthetic", nil
	case fixture == conformanceFixtureRoot+"/correlated.json":
		return "correlated", nil
	case strings.HasPrefix(fixture, functionalFixtureRoot+"/"), strings.HasPrefix(fixture, conformanceFixtureRoot+"/"):
		return "temporal", nil
	default:
		return "", fmt.Errorf("fixture %s declares no Profile its bounds move to", fixture)
	}
}

// checkMovedLimits admits dropping a Program, Contract or correlated limits message when every bound
// it declared is within the ceiling its fixture's Profile now supplies.
func checkMovedLimits(fixture string, object *Object) error {
	limits, ok := object.Fields["limits"].(*Object)
	if !ok {
		return errors.New("limits is not a message")
	}
	for field, value := range limits.Fields {
		if err := checkCeiling(fixture, limits.Message, field, value); err != nil {
			return err
		}
	}
	return nil
}

// checkMovedBound admits dropping one InstructionLimits resource bound under the same rule.
func checkMovedBound(field string) func(string, *Object) error {
	return func(fixture string, object *Object) error {
		return checkCeiling(fixture, object.Message, field, object.Fields[field])
	}
}

func checkCeiling(fixture string, message protoreflect.FullName, field string, value any) error {
	profile, err := fixtureProfile(fixture)
	if err != nil {
		return err
	}
	ceiling, declared := profileCeilings[profile][message][field]
	if !declared {
		return fmt.Errorf("%s.%s has no ceiling in the %s Profile", message, field, profile)
	}
	bound, err := strconv.ParseInt(literalText(value), 10, 64)
	if err != nil || bound < 0 || bound > ceiling {
		return fmt.Errorf("%s.%s is %v, outside the %s Profile ceiling %d", message, field, value, profile, ceiling)
	}
	return nil
}
