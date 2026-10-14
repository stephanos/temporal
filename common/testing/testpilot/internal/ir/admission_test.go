package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

func TestInvalidTruncatesPath(t *testing.T) {
	require.Equal(t, &Error{Category: Malformed, Path: "p", Detail: "d"}, Invalid(Malformed, "p", "d"))
	require.Equal(t, &Error{Category: Unknown, Path: strings.Repeat("a", 256), Detail: "d"}, Invalid(Unknown, strings.Repeat("a", 300), "d"))
}

func TestValidID(t *testing.T) {
	for id, valid := range map[string]bool{
		"a": true, "Case_1-v.2": true, strings.Repeat("a", 256): true,
		"": false, strings.Repeat("a", 257): false, "a b": false, "a/b": false, "é": false,
	} {
		require.Equal(t, valid, ValidID(id), id)
	}
}

func TestIsNil(t *testing.T) {
	var pointer *testpilotspb.Case
	var slice []int
	var mapping map[string]int
	var function func()
	var channel chan int
	var nested any = pointer
	for _, value := range []any{nil, pointer, slice, mapping, function, channel, nested} {
		require.True(t, IsNil(value), "%T", value)
	}
	for _, value := range []any{0, "", struct{}{}, &testpilotspb.Case{}, []int{}, map[string]int{}} {
		require.False(t, IsNil(value), "%T", value)
	}
}

// TestCheckCeilings pins the three admission call shapes: execution locates the field in the path,
// verification names it in the detail, and the correlated check skips its optional fields and has
// no ceiling.
func TestCheckCeilings(t *testing.T) {
	inPath := func(field string) error {
		return Invalid(LimitExceeded, field, "limit is outside the positive Driver ceiling")
	}
	inDetail := func(field string) error {
		return Invalid(LimitExceeded, "contract", "limit outside positive Driver ceiling: "+field)
	}
	unnamed := func(string) error { return Invalid(LimitExceeded, "contract", "correlated limits must be positive") }
	contract := func(edit func(*testpilotspb.ContractLimits)) *testpilotspb.ContractLimits {
		limits := &testpilotspb.ContractLimits{MaxRules: 2, MaxStates: 2, MaxTransitions: 2, MaxExpressionDepth: 2, MaxWorkPerEvent: 2, MaxTotalWork: 2, MaxCaptures: 2, MaxCaptureBytes: 2}
		if edit != nil {
			edit(limits)
		}
		return limits
	}
	correlated := func(edit func(*testpilotspb.CorrelatedLimits)) *testpilotspb.CorrelatedLimits {
		limits := &testpilotspb.CorrelatedLimits{MaxEvents: 1, MaxBuffered: 1, MaxKeys: 1, MaxSupport: 1, MaxEventBytes: 1, MaxProjectionWork: 1, MaxObligationWork: 1, MaxObligations: 1, MaxSemanticTransitions: 1, MaxCaptures: 1, MaxCorrelationDepth: 1}
		if edit != nil {
			edit(limits)
		}
		return limits
	}
	ceiling := contract(nil)
	for _, tc := range []struct {
		name string
		err  func() error
		want error
	}{
		{"field in path within ceiling", func() error { return CheckCeilings(contract(nil), ceiling, inPath) }, nil},
		{"field in path above ceiling", func() error {
			return CheckCeilings(contract(func(l *testpilotspb.ContractLimits) { l.MaxStates = 3 }), ceiling, inPath)
		}, &Error{Category: LimitExceeded, Path: "max_states", Detail: "limit is outside the positive Driver ceiling"}},
		{"field in path first of several", func() error {
			return CheckCeilings(contract(func(l *testpilotspb.ContractLimits) { l.MaxCaptureBytes = 0; l.MaxTransitions = -1 }), ceiling, inPath)
		}, &Error{Category: LimitExceeded, Path: "max_transitions", Detail: "limit is outside the positive Driver ceiling"}},
		{"field in detail not positive", func() error {
			return CheckCeilings(contract(func(l *testpilotspb.ContractLimits) { l.MaxRules = 0 }), ceiling, inDetail)
		}, &Error{Category: LimitExceeded, Path: "contract", Detail: "limit outside positive Driver ceiling: max_rules"}},
		{"skipped fields may be unset without a ceiling", func() error {
			return CheckCeilings(correlated(func(l *testpilotspb.CorrelatedLimits) {
				l.MaxCaptures = 0
				l.MaxCorrelationDepth = -1
				l.MaxEvents = 1 << 40
			}), nil, unnamed, "max_captures", "max_correlation_depth")
		}, nil},
		{"unskipped field not positive", func() error {
			return CheckCeilings(correlated(func(l *testpilotspb.CorrelatedLimits) { l.MaxKeys = 0 }), nil, unnamed, "max_captures", "max_correlation_depth")
		}, &Error{Category: LimitExceeded, Path: "contract", Detail: "correlated limits must be positive"}},
		{"skip is not a ceiling exemption for other fields", func() error {
			return CheckCeilings(contract(func(l *testpilotspb.ContractLimits) { l.MaxCaptures = 0; l.MaxRules = 3 }), ceiling, inPath, "max_captures")
		}, &Error{Category: LimitExceeded, Path: "max_rules", Detail: "limit is outside the positive Driver ceiling"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.err())
		})
	}
}
