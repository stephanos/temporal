package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validFirstOrderView = `{
  "formatVersion": "umpire3/first-order-view/v1",
  "target": "nexus-cancellation",
  "property": "nexus.cancellation.won-excludes-success",
  "world": "smoke",
  "variant": "sound",
  "semanticHash": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
  "canonicalModel": "Umpire3.Temporal.System.NexusCancellationFencing.behavior",
  "relation": {
    "declaration": "Umpire3.Temporal.Targets.NexusCancellationFencing.firstOrderView",
    "axioms": [],
    "trustBadge": "kernel"
  },
  "bounds": {"symbolicDepth": 4, "concreteStateLimit": 16},
  "sorts": [
    {"identifier": "bit", "kind": "enum", "values": ["zero", "one"]}
  ],
  "stateFields": [
    {"identifier": "flag", "sort": "bit"}
  ],
  "initial": {
    "kind": "equal",
    "left": {"kind": "field", "field": "flag"},
    "right": {"kind": "value", "sort": "bit", "value": "zero"}
  },
  "actions": [
    {
      "identifier": "flip",
      "guard": {
        "kind": "equal",
        "left": {"kind": "field", "field": "flag"},
        "right": {"kind": "value", "sort": "bit", "value": "zero"}
      },
      "updates": [
        {"field": "flag", "value": {"kind": "value", "sort": "bit", "value": "one"}}
      ]
    }
  ],
  "invariant": {"kind": "true"},
  "oracle": {
    "resultClass": "finite-exhaustive",
    "trustBadge": "checked-certificate",
    "states": [
      {"fields": [{"field": "flag", "value": "zero"}]},
      {"fields": [{"field": "flag", "value": "one"}]}
    ]
  }
}`

func TestDecodeFirstOrderViewValidatesVersionedTypedView(t *testing.T) {
	view, err := DecodeFirstOrderView(strings.NewReader(validFirstOrderView), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, FirstOrderView{
		FormatVersion:  FirstOrderViewFormatVersion,
		Target:         TargetIDNexusCancellation,
		Property:       PropertyIDNexusCancellationWonExcludesSuccess,
		World:          "smoke",
		Variant:        "sound",
		SemanticHash:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		CanonicalModel: "Umpire3.Temporal.System.NexusCancellationFencing.behavior",
		Relation: FirstOrderRelation{
			Declaration: "Umpire3.Temporal.Targets.NexusCancellationFencing.firstOrderView",
			Axioms:      []string{},
			TrustBadge:  TrustBadgeKernel,
		},
		Bounds:      FirstOrderBounds{SymbolicDepth: 4, ConcreteStateLimit: 16},
		Sorts:       []FirstOrderSort{{Identifier: "bit", Kind: FirstOrderSortEnum, Values: []string{"zero", "one"}}},
		StateFields: []FirstOrderField{{Identifier: "flag", Sort: "bit"}},
		Initial: FirstOrderFormula{
			Kind:  FirstOrderFormulaEqual,
			Left:  &FirstOrderTerm{Kind: FirstOrderTermField, Field: "flag"},
			Right: &FirstOrderTerm{Kind: FirstOrderTermValue, Sort: "bit", Value: "zero"},
		},
		Actions: []FirstOrderAction{{
			Identifier: "flip",
			Guard: FirstOrderFormula{
				Kind:  FirstOrderFormulaEqual,
				Left:  &FirstOrderTerm{Kind: FirstOrderTermField, Field: "flag"},
				Right: &FirstOrderTerm{Kind: FirstOrderTermValue, Sort: "bit", Value: "zero"},
			},
			Updates: []FirstOrderUpdate{{
				Field: "flag",
				Value: FirstOrderTerm{Kind: FirstOrderTermValue, Sort: "bit", Value: "one"},
			}},
		}},
		Invariant: FirstOrderFormula{Kind: FirstOrderFormulaTrue},
		Oracle: FirstOrderOracle{
			ResultClass: ResultClassFiniteExhaustive,
			TrustBadge:  TrustBadgeCheckedCertificate,
			States: []FirstOrderState{
				{Fields: []FirstOrderBinding{{Field: "flag", Value: "zero"}}},
				{Fields: []FirstOrderBinding{{Field: "flag", Value: "one"}}},
			},
		},
	}, view)

	first, err := view.CanonicalJSON()
	require.NoError(t, err)
	second, err := view.CanonicalJSON()
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestDecodeFirstOrderViewRejectsUnknownAndTrailingInput(t *testing.T) {
	unknown := strings.Replace(validFirstOrderView, `"world": "smoke",`, `"world": "smoke", "unknown": true,`, 1)
	_, err := DecodeFirstOrderView(strings.NewReader(unknown), DefaultDecodeLimit)
	require.ErrorContains(t, err, "unknown field")

	_, err = DecodeFirstOrderView(strings.NewReader(validFirstOrderView+` {}`), DefaultDecodeLimit)
	require.ErrorContains(t, err, "multiple JSON values")
}

func TestFirstOrderViewRejectsBrokenTypedReferencesAndTrustClaims(t *testing.T) {
	tests := map[string]struct {
		replace string
		with    string
		want    string
	}{
		"unknown field": {
			replace: `"field": "flag"`,
			with:    `"field": "missing"`,
			want:    `unknown state field "missing"`,
		},
		"unknown enum value": {
			replace: `"value": "zero"`,
			with:    `"value": "missing"`,
			want:    `unknown value "missing" for sort "bit"`,
		},
		"assignment type mismatch": {
			replace: `{"field": "flag", "value": {"kind": "value", "sort": "bit", "value": "one"}}`,
			with:    `{"field": "flag", "value": {"kind": "field", "field": "missing"}}`,
			want:    `unknown state field "missing"`,
		},
		"unearned relation trust": {
			replace: "\"axioms\": [],\n    \"trustBadge\": \"kernel\"",
			with:    "\"axioms\": [\"propext\"],\n    \"trustBadge\": \"kernel\"",
			want:    `relation with axioms requires kernel-with-declared-axioms`,
		},
		"unearned oracle class": {
			replace: "\"resultClass\": \"finite-exhaustive\",\n    \"trustBadge\": \"checked-certificate\"",
			with:    "\"resultClass\": \"external-no-counterexample\",\n    \"trustBadge\": \"checked-certificate\"",
			want:    `oracle result class must be finite-exhaustive`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			encoded := strings.Replace(validFirstOrderView, test.replace, test.with, 1)
			_, err := DecodeFirstOrderView(strings.NewReader(encoded), DefaultDecodeLimit)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestDecodeFirstOrderViewEnforcesByteLimit(t *testing.T) {
	_, err := DecodeFirstOrderView(bytes.NewReader([]byte(validFirstOrderView)), 8)
	require.ErrorContains(t, err, "exceeds 8-byte decode limit")
}

func TestFirstOrderViewSupportsCanonicalUninterpretedMembers(t *testing.T) {
	view, err := DecodeFirstOrderView(strings.NewReader(validFirstOrderView), DefaultDecodeLimit)
	require.NoError(t, err)
	view.Sorts = []FirstOrderSort{{
		Identifier: "node", Kind: FirstOrderSortUninterpreted, Values: []string{}, Cardinality: 2,
	}}
	view.StateFields = []FirstOrderField{{Identifier: "flag", Sort: "node"}}
	view.Initial.Right = &FirstOrderTerm{
		Kind: FirstOrderTermValue, Sort: "node", Value: "member-0",
	}
	view.Actions[0].Guard.Right = &FirstOrderTerm{
		Kind: FirstOrderTermValue, Sort: "node", Value: "member-0",
	}
	view.Actions[0].Updates[0].Value = FirstOrderTerm{
		Kind: FirstOrderTermValue, Sort: "node", Value: "member-1",
	}
	view.Oracle.States = []FirstOrderState{
		{Fields: []FirstOrderBinding{{Field: "flag", Value: "member-0"}}},
		{Fields: []FirstOrderBinding{{Field: "flag", Value: "member-1"}}},
	}
	require.NoError(t, view.Validate())

	view.Oracle.States[1].Fields[0].Value = "member-2"
	require.ErrorContains(t, view.Validate(), `unknown value "member-2" for sort "node"`)
}

func TestFirstOrderViewCapsHostileStateDomainsBeforeAllocation(t *testing.T) {
	view, err := DecodeFirstOrderView(strings.NewReader(validFirstOrderView), DefaultDecodeLimit)
	require.NoError(t, err)
	view.Bounds.ConcreteStateLimit = MaxFirstOrderConcreteStateLimit + 1
	require.ErrorContains(t, view.Validate(), "bounded first-order")

	view.Bounds.ConcreteStateLimit = 16
	view.StateFields = append(view.StateFields, FirstOrderField{Identifier: "second", Sort: "bit"},
		FirstOrderField{Identifier: "third", Sort: "bit"},
		FirstOrderField{Identifier: "fourth", Sort: "bit"},
		FirstOrderField{Identifier: "fifth", Sort: "bit"})
	require.ErrorContains(t, view.Validate(), "state domain exceeds concrete state limit 16")
}
