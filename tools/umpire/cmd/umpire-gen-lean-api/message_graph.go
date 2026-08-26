package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type messageGraph struct {
	dependencies map[string][]string
	component    map[string]int
	cyclic       map[int]bool
	order        []string
}

func buildMessageGraph(messages []messageProjection) (messageGraph, error) {
	dependencies, err := buildMessageDependencies(messages)
	if err != nil {
		return messageGraph{}, err
	}
	graph := messageGraph{
		dependencies: dependencies,
		component:    make(map[string]int, len(messages)),
		cyclic:       make(map[int]bool),
	}
	components := graph.stronglyConnectedComponents()
	graph.indexComponents(components)
	componentDependencies, componentKeys := graph.buildComponentGraph(components)
	graph.order = orderMessageComponents(components, componentDependencies, componentKeys)
	if len(graph.order) != len(messages) {
		return messageGraph{}, fmt.Errorf("build message graph: ordered %d of %d messages", len(graph.order), len(messages))
	}
	return graph, nil
}

func buildMessageDependencies(messages []messageProjection) (map[string][]string, error) {
	known := make(map[string]bool, len(messages))
	for _, message := range messages {
		if message.FullName == "" {
			return nil, errors.New("build message graph: message has empty full name")
		}
		if known[message.FullName] {
			return nil, fmt.Errorf("build message graph: duplicate message %q", message.FullName)
		}
		known[message.FullName] = true
	}
	result := make(map[string][]string, len(messages))
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
		result[message.FullName] = dependencies
	}
	return result, nil
}

func (g *messageGraph) indexComponents(components [][]string) {
	for component, members := range components {
		for _, member := range members {
			g.component[member] = component
		}
		if len(members) > 1 || slices.Contains(g.dependencies[members[0]], members[0]) {
			g.cyclic[component] = true
		}
	}
}

func (g messageGraph) buildComponentGraph(components [][]string) (map[int][]int, []string) {
	componentDependencies := make(map[int][]int, len(components))
	componentKeys := make([]string, len(components))
	for component, members := range components {
		componentKeys[component] = members[0]
		seen := make(map[int]bool)
		for _, member := range members {
			for _, dependency := range g.dependencies[member] {
				dependencyComponent := g.component[dependency]
				if dependencyComponent != component && !seen[dependencyComponent] {
					componentDependencies[component] = append(componentDependencies[component], dependencyComponent)
					seen[dependencyComponent] = true
				}
			}
		}
		slices.SortFunc(componentDependencies[component], func(left, right int) int {
			return strings.Compare(componentKeys[left], componentKeys[right])
		})
	}
	return componentDependencies, componentKeys
}

func orderMessageComponents(
	components [][]string,
	componentDependencies map[int][]int,
	componentKeys []string,
) []string {
	componentOrder := make([]int, len(components))
	for index := range componentOrder {
		componentOrder[index] = index
	}
	slices.SortFunc(componentOrder, func(left, right int) int {
		return strings.Compare(componentKeys[left], componentKeys[right])
	})
	visited := make(map[int]bool, len(components))
	result := make([]string, 0, len(componentKeys))
	var visit func(int)
	visit = func(component int) {
		if visited[component] {
			return
		}
		visited[component] = true
		for _, dependency := range componentDependencies[component] {
			visit(dependency)
		}
		result = append(result, components[component]...)
	}
	for _, component := range componentOrder {
		visit(component)
	}
	return result
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
	fromComponent, fromKnown := g.component[from]
	toComponent, toKnown := g.component[to]
	return fromKnown && toKnown && fromComponent == toComponent && g.cyclic[fromComponent]
}
