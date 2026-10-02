package checker

// After reads a Monitor's verdict after each step whose result f accepts.
func After(f func(Result) bool) Evaluation { return Evaluation{kind: afterSteps, after: f} }

// NewMonitor declares a Monitor with a finite state type M over a machine whose steps have type
// Step[S, O, F].
func NewMonitor[M, S, O, F any](name string, initial M, next func(M, S, Step[S, O, F]) M, violated func(M) bool,
	at Evaluation) *Monitor {
	decl := "monitor " + name
	states, err := DomainOf[M]()
	if err != nil {
		return &Monitor{Name: name, err: errorf(decl, "state type: %v", err)}
	}
	byKey := make(map[string]M, len(states))
	for _, s := range states {
		byKey[KeyOf(s)] = s
	}
	if _, ok := byKey[KeyOf(initial)]; !ok {
		return &Monitor{Name: name, err: errorf(decl, "the initial state %s is outside the state domain", KeyOf(initial))}
	}
	return &Monitor{Name: name, Initial: KeyOf(initial), At: at,
		Next: func(key string, before any, res Result) (string, error) {
			s, isState := before.(S)
			step, isStep := res.Step.(Step[S, O, F])
			if !isState || !isStep {
				return "", errorf(decl, "the step into %s is not a step of a %T", res.State, s)
			}
			out := KeyOf(next(byKey[key], s, step))
			if _, ok := byKey[out]; !ok {
				return "", errorf(decl, "the step into %s moves it to %s, which is outside the state domain", res.State, out)
			}
			return out, nil
		},
		Violated: func(key string) bool { return violated(byKey[key]) },
	}
}
