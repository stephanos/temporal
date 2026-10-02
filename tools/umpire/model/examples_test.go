package model

// An Abstraction Claim is on one class of a one-input action: admission rejects an example anywhere
// else, where a reader of claims would have no input to name.

import (
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/proto"
)

func TestAnExampleIsOfAClassOfAOneInputAction(t *testing.T) {
	action := func(m *modelirspb.Model, name string) *modelirspb.Action {
		for _, a := range m.GetActions() {
			if a.GetName() == name {
				return a
			}
		}
		return nil
	}
	flag := &modelirspb.Value{Kind: &modelirspb.Value_Bool{Bool: true}}
	for _, c := range []struct {
		name   string
		mutate func(m *modelirspb.Model)
		want   string
	}{
		{"an example on an action with no input", func(m *modelirspb.Model) {
			a := action(m, "backoff")
			a.Examples = append(a.Examples, &modelirspb.Example{Value: flag, Example: "soon"})
		}, "gives an example and takes 0 inputs; an example is of one class of a one-input action"},
		{"an example on an action with three inputs", func(m *modelirspb.Model) {
			a := action(m, "schedule")
			a.Examples = append(a.Examples, &modelirspb.Example{Value: flag, Example: "soon"})
		}, "gives an example and takes 3 inputs; an example is of one class of a one-input action"},
		{"an example of another type than the input", func(m *modelirspb.Model) {
			action(m, "handlerReply").GetExamples()[0].Value = flag
		}, "gives an example of true, which is no temporal.nexuscaller.kernel.Reply"},
		{"an example of no value", func(m *modelirspb.Model) { action(m, "handlerReply").GetExamples()[0].Value = nil },
			"gives an example of no value"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := proto.Clone(load(t)).(*modelirspb.Model)
			c.mutate(m)
			err := Validate(m)
			require.ErrorContains(t, err, c.want)
			require.ErrorContains(t, err, "model/")
		})
	}
}
