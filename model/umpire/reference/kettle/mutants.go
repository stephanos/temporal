package kettle

import "fmt"

type Options struct{ Mutant string }

func Mutants() []string {
	return []string{"overlong-id", "premature-boil", "removed-validator", "wrong-status"}
}

func (o Options) Validate() error {
	if o.Mutant == "" {
		return nil
	}
	for _, name := range Mutants() {
		if name == o.Mutant {
			return nil
		}
	}
	return fmt.Errorf("unknown kettle mutant %q", o.Mutant)
}
