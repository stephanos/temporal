package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

func formatted(set *token.FileSet, node ast.Node) string {
	var output bytes.Buffer
	if err := format.Node(&output, set, node); err != nil {
		panic(err)
	}
	return output.String()
}

func parse(path string, input any) (*token.FileSet, *ast.File) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, input, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	return set, file
}

func calls(set *token.FileSet, node ast.Node, counts map[string]int, literals map[string]int) {
	wrappers := map[string]bool{
		"reportProgress": true, "recordCompletion": true, "publishRunnerFailure": true,
		"completePartial": true, "launch": true, "launchSeed": true,
		"validateRequest": true, "openCampaign": true, "failCampaign": true,
		"prepareTarget": true, "runSeeds": true, "scheduleSeeds": true,
		"handleCompletion": true, "recordAssessedCompletion": true,
		"recordSuccessfulExecution": true, "recordFailedExecution": true, "finishSeedCampaign": true,
	}
	ast.Inspect(node, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			literals[literal.Value]++
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if _, wrapper := call.Fun.(*ast.FuncLit); wrapper {
			return true
		}
		name := strings.ReplaceAll(formatted(set, call.Fun), "local.", "")
		if !wrappers[name] {
			expression := strings.ReplaceAll(formatted(set, call), "local.", "")
			expression = strings.ReplaceAll(expression, "*retErr", "retErr")
			counts[expression]++
		}
		return true
	})
}

func main() {
	base, err := os.ReadFile(".flow/tmp/base_commit")
	if err != nil {
		panic(err)
	}
	before, err := exec.Command("git", "show", strings.TrimSpace(string(base))+":tools/gomad3/runner/runner.go").Output()
	if err != nil {
		panic(err)
	}
	oldSet, oldFile := parse("BASE/runner.go", before)
	newSet, newFile := parse("tools/gomad3/runner/runner.go", nil)
	localSet, localFile := parse("tools/gomad3/runner/runner_local.go", nil)
	oldFunctions, newFunctions := map[string]string{}, map[string]string{}
	oldCalls, newCalls := map[string]int{}, map[string]int{}
	oldLiterals, newLiterals := map[string]int{}, map[string]int{}
	for _, declaration := range oldFile.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Name.Name == "runLocal" {
			calls(oldSet, function, oldCalls, oldLiterals)
		} else {
			oldFunctions[function.Name.Name] = formatted(oldSet, function)
		}
	}
	for _, declaration := range newFile.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			newFunctions[function.Name.Name] = formatted(newSet, function)
		}
	}
	for _, declaration := range localFile.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			calls(localSet, function, newCalls, newLiterals)
		}
	}
	if !reflect.DeepEqual(oldFunctions, newFunctions) {
		panic("a function outside runLocal changed")
	}
	if !reflect.DeepEqual(oldCalls, newCalls) {
		fmt.Printf("BASE calls: %#v\nFINAL calls: %#v\n", oldCalls, newCalls)
		panic("primitive call inventory changed")
	}
	if !reflect.DeepEqual(oldLiterals, newLiterals) {
		fmt.Printf("BASE literals: %#v\nFINAL literals: %#v\n", oldLiterals, newLiterals)
		panic("local literal inventory changed")
	}
	comments := func(file *ast.File) []string {
		var result []string
		for _, group := range file.Comments {
			result = append(result, group.Text())
		}
		return result
	}
	if !reflect.DeepEqual(comments(oldFile), comments(newFile)) || len(localFile.Comments) != 0 {
		panic("original comments changed or moved without owning code")
	}
	fmt.Printf("%d unchanged nonlocal functions; %d primitive call expressions; %d literal values; %d comment groups preserved\n", len(oldFunctions), len(oldCalls), len(oldLiterals), len(oldFile.Comments))
	fmt.Println("Inventory proof does not establish branch ordering, OS-fault execution, or a native pass.")
}
