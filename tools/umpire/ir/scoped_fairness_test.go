package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func TestAdmissionChecksStructuredFairnessBeforeGraphChecking(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*umpirespb.ComposedActionRef)
		why    string
	}{
		{"unknown owner", func(r *umpirespb.ComposedActionRef) { r.Composition = "absent" }, "no composition absent"},
		{"unknown member", func(r *umpirespb.ComposedActionRef) { r.Member = "absent" }, "no member absent"},
		{"unknown action", func(r *umpirespb.ComposedActionRef) {
			r.Selection = &umpirespb.ComposedActionRef_Action{Action: "absent"}
		}, "no action absent"},
		{"unbound action", func(r *umpirespb.ComposedActionRef) { r.Member = "front" }, "binds no action"},
		{"ambiguous member", func(r *umpirespb.ComposedActionRef) {
			r.Composition, r.Member = "pair", ""
			r.Selection = &umpirespb.ComposedActionRef_Action{Action: admDeclaredID + "put"}
		}, "member field"},
		{"unknown sync", func(r *umpirespb.ComposedActionRef) {
			r.Member, r.Selection = "", &umpirespb.ComposedActionRef_Sync{Sync: "absent"}
		}, "no sync absent"},
		{"unsupported selector", func(r *umpirespb.ComposedActionRef) { r.Selection = nil }, "no known fairness selection"},
		{"sync with member", func(r *umpirespb.ComposedActionRef) { r.Selection = &umpirespb.ComposedActionRef_Sync{Sync: "putBoth"} }, "sync selection names member"},
		{"invalid exact class", func(r *umpirespb.ComposedActionRef) {
			r.Selection = &umpirespb.ComposedActionRef_ActionClass{ActionClass: &umpirespb.ActionClass{Action: admDeclaredID + "flush", Inputs: []*umpirespb.Value{admIntValue(1)}}}
		}, "takes 0 inputs, not 1"},
		{"invalid exact sync class", func(r *umpirespb.ComposedActionRef) {
			r.Member = ""
			r.Selection = &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "putBoth", Inputs: []*umpirespb.Value{admIntValue(1)}}}
		}, "takes 0 inputs, not 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := admFixture(t, "declarations")
			r := &umpirespb.ComposedActionRef{Composition: "detailedPair", Member: "back",
				Selection: &umpirespb.ComposedActionRef_Action{Action: admDeclaredID + "flush"}}
			a := &umpirespb.Assumption{Id: "scoped", Name: "scoped", Position: &umpirespb.Position{File: "fairness.scala", Line: 12}, FairRefs: []*umpirespb.ComposedActionRef{r}}
			m.Assumptions = append(m.Assumptions, a)
			require.NoError(t, Validate(m))
			tc.change(r)
			err := Validate(m)
			require.ErrorContains(t, err, tc.why)
			require.ErrorContains(t, err, "fairness.scala:12")
		})
	}
}
