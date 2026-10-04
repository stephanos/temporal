//go:build ignore

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func bodies(path string) (map[string][]byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, source, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]byte)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		result[function.Name.Name] = source[set.Position(function.Body.Pos()).Offset:set.Position(function.Body.End()).Offset]
	}
	return result, nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: compare-test-bodies before after")
		os.Exit(2)
	}
	before, err := bodies(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	after, err := bodies(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	allowed := map[string]bool{
		"TestSimulationTimeProgressPreexistingMalformedWaitWakesQuiescence": true,
		"TestSimulationTimeProgressPreexistingDuplicateAdmissionConsumesArrival": true,
	}
	type comparison struct {
		Before string `json:"before_sha256"`
		After string `json:"after_sha256"`
		Unchanged bool `json:"body_byte_identical"`
		StrengthenedNegative bool `json:"strengthened_negative"`
	}
	comparisons := make(map[string]comparison)
	valid := true
	for name, body := range before {
		other, exists := after[name]
		unchanged := exists && bytes.Equal(body, other)
		if !exists || !unchanged && !allowed[name] {
			valid = false
		}
		comparisons[name] = comparison{
			Before: fmt.Sprintf("%x", sha256.Sum256(body)), After: fmt.Sprintf("%x", sha256.Sum256(other)),
			Unchanged: unchanged, StrengthenedNegative: allowed[name],
		}
	}
	if len(after) != len(before) {
		valid = false
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(comparisons); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if !valid {
		os.Exit(1)
	}
}
