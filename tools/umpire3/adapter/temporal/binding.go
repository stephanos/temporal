package temporal

import (
	"errors"
	"fmt"

	environment "go.temporal.io/server/tools/umpire3/execution"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func groundActionBindings(action protocolexperiment.Action, projections map[string]string) (map[string]string, error) {
	if len(action.Bindings) == 0 {
		return nil, nil
	}
	grounded := make(map[string]string, len(action.Bindings))
	for _, binding := range action.Bindings {
		value := projections[binding.Projection]
		if value == "" {
			return nil, fmt.Errorf("projection %q has no concrete identity", binding.Projection)
		}
		grounded[binding.Symbol] = value
	}
	return grounded, nil
}

func validateIdentityArgument(
	action protocolexperiment.Action,
	name string,
	expected string,
	bindings environment.Bindings,
) error {
	for _, argument := range action.Arguments {
		if argument.Name != name {
			continue
		}
		if argument.Value.Type != protocolexperiment.ValueSymbol || argument.Value.Text == nil {
			return fmt.Errorf("identity argument %q is not symbolic", name)
		}
		actual, grounded := bindings[*argument.Value.Text]
		if !grounded {
			return fmt.Errorf("identity argument %q is not grounded", name)
		}
		if actual != expected {
			return fmt.Errorf("identity argument %q does not match realized lineage", name)
		}
		return nil
	}
	if expected == "" {
		return errors.New("expected identity is missing")
	}
	return nil
}
