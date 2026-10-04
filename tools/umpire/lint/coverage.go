package lint

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// Coverage is one line of the coverage summary: the population of one kind in one owner, and the
// part of it that satisfies the kind. Its difference is that kind's findings in that owner, accepted
// or not: an acceptance hides nothing from a count.
type Coverage struct {
	Owner      string
	Kind       Kind
	Population int
	Satisfied  int
	count      count
}

// Coverage is the result's coverage lines, by owner and then in the order kinds() prints them. A
// kind with no count, and a population of none, print nothing.
func (r *Result) Coverage() []Coverage {
	var out []Coverage
	for _, k := range kinds() {
		if k.count == nil {
			continue
		}
		for _, t := range r.Tallies {
			if t.Kind != k.kind || t.Population == 0 {
				continue
			}
			out = append(out, Coverage{Owner: t.Owner, Kind: t.Kind, Population: t.Population,
				Satisfied: t.Population - len(t.Findings), count: *k.count})
		}
	}
	slices.SortStableFunc(out, func(x, y Coverage) int { return strings.Compare(x.Owner, y.Owner) })
	return out
}

// WriteCoverage writes the coverage summary of one IR file: a block per owner, one line per count. It
// is informational and deterministic, and compared with nothing but its own fixture golden.
func WriteCoverage(w io.Writer, r *Result) error {
	owner := ""
	for _, c := range r.Coverage() {
		if c.Owner != owner {
			owner = c.Owner
			if _, err := fmt.Fprintf(w, "coverage %s %s\n", r.File, owner); err != nil {
				return err
			}
		}
		population := fmt.Sprint(c.Population)
		if c.count.population != "" {
			population += " " + c.count.population
		}
		if _, err := fmt.Fprintf(w, "  %-20s %s, %d %s\n", c.count.name, population, c.Satisfied, c.count.satisfied); err != nil {
			return err
		}
	}
	return nil
}
