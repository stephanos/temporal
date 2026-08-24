package api

import (
	"fmt"
	"slices"
)

type messageGraph struct {
	dependencies map[string][]string
	component    map[string]int
	cyclic       map[int]bool
	order        []string
}

func buildMessageGraph(messages []messageProjection) (messageGraph, error) {
	graph := messageGraph{
		dependencies: make(map[string][]string, len(messages)),
		component:    make(map[string]int, len(messages)),
		cyclic:       make(map[int]bool),
	}
	known := make(map[string]bool, len(messages))
	for _, message := range messages {
		if message.FullName == "" {
			return messageGraph{}, fmt.Errorf("build message graph: message has empty full name")
		}
		if known[message.FullName] {
			return messageGraph{}, fmt.Errorf("build message graph: duplicate message %q", message.FullName)
		}
		known[message.FullName] = true
	}
	for _, message := range messages {
		seen := make(map[string]bool)
		dependencies := []string{}
		for _, field := range message.Fields {
			if known[field.TypeName] && !seen[field.TypeName] {
				dependencies = append(dependencies, field.TypeName)
				seen[field.TypeName] = true
			}
		}
		slices.Sort(dependencies)
		graph.dependencies[message.FullName] = dependencies
	}

	components := graph.stronglyConnectedComponents()
	for component, members := range components {
		for _, member := range members {
			graph.component[member] = component
		}
		if len(members) > 1 || slices.Contains(graph.dependencies[members[0]], members[0]) {
			graph.cyclic[component] = true
		}
	}

	componentDependencies := make(map[int][]int, len(components))
	componentKeys := make([]string, len(components))
	for component, members := range components {
		componentKeys[component] = members[0]
		seen := make(map[int]bool)
		for _, member := range members {
			for _, dependency := range graph.dependencies[member] {
				dependencyComponent := graph.component[dependency]
				if dependencyComponent != component && !seen[dependencyComponent] {
					componentDependencies[component] = append(componentDependencies[component], dependencyComponent)
					seen[dependencyComponent] = true
				}
			}
		}
		slices.SortFunc(componentDependencies[component], func(left, right int) int {
			return compareStrings(componentKeys[left], componentKeys[right])
		})
	}
	componentOrder := make([]int, len(components))
	for index := range componentOrder {
		componentOrder[index] = index
	}
	slices.SortFunc(componentOrder, func(left, right int) int {
		return compareStrings(componentKeys[left], componentKeys[right])
	})
	visited := make(map[int]bool, len(components))
	var visit func(int)
	visit = func(component int) {
		if visited[component] {
			return
		}
		visited[component] = true
		for _, dependency := range componentDependencies[component] {
			visit(dependency)
		}
		graph.order = append(graph.order, components[component]...)
	}
	for _, component := range componentOrder {
		visit(component)
	}
	if len(graph.order) != len(messages) {
		return messageGraph{}, fmt.Errorf("build message graph: ordered %d of %d messages", len(graph.order), len(messages))
	}
	return graph, nil
}

func (g messageGraph) stronglyConnectedComponents() [][]string {
	nodes := make([]string, 0, len(g.dependencies))
	for node := range g.dependencies {
		nodes = append(nodes, node)
	}
	slices.Sort(nodes)
	indices := make(map[string]int, len(nodes))
	lowLinks := make(map[string]int, len(nodes))
	onStack := make(map[string]bool, len(nodes))
	stack := make([]string, 0, len(nodes))
	index := 0
	var components [][]string
	var visit func(string)
	visit = func(node string) {
		indices[node] = index
		lowLinks[node] = index
		index++
		stack = append(stack, node)
		onStack[node] = true
		for _, dependency := range g.dependencies[node] {
			dependencyIndex, visited := indices[dependency]
			if !visited {
				visit(dependency)
				lowLinks[node] = min(lowLinks[node], lowLinks[dependency])
			} else if onStack[dependency] {
				lowLinks[node] = min(lowLinks[node], dependencyIndex)
			}
		}
		if lowLinks[node] != indices[node] {
			return
		}
		var component []string
		for {
			last := len(stack) - 1
			member := stack[last]
			stack = stack[:last]
			onStack[member] = false
			component = append(component, member)
			if member == node {
				break
			}
		}
		slices.Sort(component)
		components = append(components, component)
	}
	for _, node := range nodes {
		if _, visited := indices[node]; !visited {
			visit(node)
		}
	}
	return components
}

func (g messageGraph) recursive(from, to string) bool {
	component, known := g.component[from]
	return known && g.component[to] == component && g.cyclic[component]
}

func compareStrings(left, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
