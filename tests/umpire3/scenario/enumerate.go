package scenario

import (
	"context"
	"fmt"
	"slices"
)

type enumerationResult struct {
	paths       [][]string
	states      int
	memoryBytes int64
}

func enumerate(ctx context.Context, actions []*normalizedAction, edges []normalizedEdge, allPaths bool, limits Limits) (enumerationResult, error) {
	indegree := make(map[string]int, len(actions))
	next := make(map[string][]string, len(actions))
	sources := make(map[string]Source, len(actions))
	for _, action := range actions {
		indegree[action.identifier] = 0
		sources[action.identifier] = action.source
	}
	for _, edge := range edges {
		next[edge.before] = append(next[edge.before], edge.after)
		indegree[edge.after]++
	}
	for identifier := range next {
		slices.Sort(next[identifier])
	}

	result := enumerationResult{}
	var visit func([]string, map[string]int) error
	visit = func(path []string, degrees map[string]int) error {
		if err := ctx.Err(); err != nil {
			return compileError(ErrorLimitExceeded, Source{}, "compiler time budget exhausted: "+err.Error())
		}
		result.states++
		if result.states > limits.MaxStates {
			return compileError(ErrorLimitExceeded, Source{}, fmt.Sprintf("state limit %d exceeded", limits.MaxStates))
		}
		if len(path) == len(actions) {
			result.memoryBytes += int64(len(path) * 32)
			if result.memoryBytes > limits.MaxMemoryBytes {
				return compileError(ErrorLimitExceeded, Source{}, fmt.Sprintf("memory limit %d exceeded", limits.MaxMemoryBytes))
			}
			result.paths = append(result.paths, append([]string(nil), path...))
			if !allPaths {
				return nil
			}
			if len(result.paths) > limits.MaxPaths {
				return compileError(ErrorIncompleteEnumeration, Source{}, fmt.Sprintf("all-path enumeration exceeds path limit %d", limits.MaxPaths))
			}
			return nil
		}

		ready := make([]string, 0, len(actions)-len(path))
		selected := make(map[string]struct{}, len(path))
		for _, identifier := range path {
			selected[identifier] = struct{}{}
		}
		for identifier, degree := range degrees {
			if degree == 0 {
				if _, exists := selected[identifier]; !exists {
					ready = append(ready, identifier)
				}
			}
		}
		slices.Sort(ready)
		if len(ready) == 0 {
			remaining := make([]string, 0, len(actions)-len(path))
			for identifier := range degrees {
				if _, exists := selected[identifier]; !exists {
					remaining = append(remaining, identifier)
				}
			}
			slices.Sort(remaining)
			source := Source{}
			for _, identifier := range remaining {
				if candidate := sources[identifier]; candidate.File != "" {
					source = candidate
					break
				}
			}
			return compileError(ErrorCycle, source, "scenario ordering contains a cycle")
		}
		for _, identifier := range ready {
			updated := cloneDegrees(degrees)
			updated[identifier] = -1
			for _, dependent := range next[identifier] {
				updated[dependent]--
			}
			if err := visit(append(path, identifier), updated); err != nil {
				return err
			}
			if !allPaths && len(result.paths) != 0 {
				return nil
			}
		}
		return nil
	}
	if err := visit(nil, indegree); err != nil {
		return enumerationResult{}, err
	}
	slices.SortFunc(result.paths, func(left, right []string) int {
		for index := 0; index < len(left) && index < len(right); index++ {
			if result := stringCompare(left[index], right[index]); result != 0 {
				return result
			}
		}
		return len(left) - len(right)
	})
	return result, nil
}

func cloneDegrees(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for identifier, degree := range source {
		result[identifier] = degree
	}
	return result
}
