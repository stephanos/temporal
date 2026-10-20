package sweep

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ErrEnumerationMismatch reports that the parsed inventory and `go test -list`
// disagree. The sweep does not run when it occurs.
var ErrEnumerationMismatch = errors.New("enumeration mismatch")

// Package is one enumerated tests package.
type Package struct {
	ImportPath string   `json:"import_path"`
	Pattern    string   `json:"pattern"` // ./tests/<rel>, the go-test target
	Tests      []string `json:"tests"`   // sorted top-level Test functions (host file set)
	// GuestExcluded are tests whose defining file is not in the wasip1 guest
	// file set under the slim tags; they cannot run in the guest.
	GuestExcluded map[string]string `json:"guest_excluded,omitempty"`
	ListCount     int               `json:"list_count"`
	ParsedCount   int               `json:"parsed_count"`
	files         map[string]string // test -> defining file (host)
}

type listedPackage struct {
	ImportPath   string
	Dir          string
	TestGoFiles  []string
	XTestGoFiles []string
	Error        *struct{ Err string }
}

// goEnvironment fixes the go command's ambient configuration so the inventory
// does not depend on the operator's GOFLAGS, workspace or toolchain switching.
func goEnvironment(environ []string, goos, goarch string) []string {
	var env []string
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "GOOS", "GOARCH", "CGO_ENABLED", "GOENV", "GOEXPERIMENT":
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOENV=off", "CGO_ENABLED=0")
	if goos != "" {
		env = append(env, "GOOS="+goos, "GOARCH="+goarch)
	}
	if goos == "wasip1" {
		env = append(env, "GOEXPERIMENT=nogreenteagc")
	}
	return env
}

// Enumerate lists the selected packages with the guest build tags, parses
// their test files for top-level Test functions and reconciles the result
// against `go test -list ^Test` for the same packages and tags. Any
// difference is ErrEnumerationMismatch.
func Enumerate(ctx context.Context, executor Executor, options Options, tags []string, log io.Writer) ([]Package, error) {
	host, err := listPackages(ctx, executor, options, tags, "", "")
	if err != nil {
		return nil, err
	}
	guest, err := listPackages(ctx, executor, options, tags, "wasip1", "wasm")
	if err != nil {
		return nil, err
	}
	guestFiles := map[string]map[string]bool{}
	for _, listed := range guest {
		files := map[string]bool{}
		for _, name := range append(slices.Clone(listed.TestGoFiles), listed.XTestGoFiles...) {
			files[name] = true
		}
		guestFiles[listed.ImportPath] = files
	}
	var packages []Package
	var mismatches []string
	for _, listed := range host {
		files := append(slices.Clone(listed.TestGoFiles), listed.XTestGoFiles...)
		if len(files) == 0 {
			continue
		}
		parsed, err := parseTests(listed.Dir, files)
		if err != nil {
			return nil, err
		}
		pattern, err := relativePattern(options.Checkout, listed.Dir)
		if err != nil {
			return nil, err
		}
		listedTests, err := listTests(ctx, executor, options, tags, pattern)
		if err != nil {
			return nil, err
		}
		pkg := Package{ImportPath: listed.ImportPath, Pattern: pattern, ListCount: len(listedTests), ParsedCount: len(parsed), files: parsed}
		for name := range parsed {
			pkg.Tests = append(pkg.Tests, name)
		}
		sort.Strings(pkg.Tests)
		if !slices.Equal(pkg.Tests, listedTests) {
			mismatches = append(mismatches, fmt.Sprintf("%s: parsed %d %v, go test -list %d %v", pattern, len(pkg.Tests), difference(pkg.Tests, listedTests), len(listedTests), difference(listedTests, pkg.Tests)))
		}
		for _, name := range pkg.Tests {
			if file := parsed[name]; !guestFiles[listed.ImportPath][file] {
				if pkg.GuestExcluded == nil {
					pkg.GuestExcluded = map[string]string{}
				}
				pkg.GuestExcluded[name] = file
			}
		}
		if log != nil {
			_, _ = fmt.Fprintf(log, "wasmsweep: enumerated %s parsed=%d list=%d guest-excluded=%d\n", pattern, pkg.ParsedCount, pkg.ListCount, len(pkg.GuestExcluded))
		}
		packages = append(packages, pkg)
	}
	if len(mismatches) != 0 {
		return packages, fmt.Errorf("%w:\n%s", ErrEnumerationMismatch, strings.Join(mismatches, "\n"))
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Pattern < packages[j].Pattern })
	return packages, nil
}

func difference(a, b []string) []string {
	var only []string
	for _, value := range a {
		if !slices.Contains(b, value) {
			only = append(only, value)
		}
	}
	return only
}

func relativePattern(checkout, dir string) (string, error) {
	relative, err := filepath.Rel(checkout, dir)
	if err != nil || relative == ".." || strings.HasPrefix(relative, "../") {
		return "", fmt.Errorf("package directory %s is outside the checkout", dir)
	}
	return "./" + filepath.ToSlash(relative), nil
}

func listPackages(ctx context.Context, executor Executor, options Options, tags []string, goos, goarch string) ([]listedPackage, error) {
	args := []string{"list", "-e", "-json=ImportPath,Dir,TestGoFiles,XTestGoFiles,Error", "-tags", strings.Join(tags, ",")}
	args = append(args, options.Packages...)
	completed := executor.Exec(ctx, Command{Path: options.Compiler, Args: args, Dir: options.Checkout, Env: goEnvironment(options.Environ, goos, goarch), Timeout: 10 * time.Minute})
	if completed.Err != nil || completed.Exit != 0 {
		return nil, fmt.Errorf("go list %s (GOOS=%s): exit %d %v\n%s", strings.Join(options.Packages, " "), goos, completed.Exit, completed.Err, completed.Stderr)
	}
	decoder := json.NewDecoder(bytes.NewReader(completed.Stdout))
	var packages []listedPackage
	for decoder.More() {
		var listed listedPackage
		if err := decoder.Decode(&listed); err != nil {
			return nil, fmt.Errorf("decode go list: %w", err)
		}
		if listed.Error != nil {
			return nil, fmt.Errorf("go list %s (GOOS=%s): %s", listed.ImportPath, goos, listed.Error.Err)
		}
		packages = append(packages, listed)
	}
	if len(packages) == 0 {
		return nil, fmt.Errorf("go list matched no packages for %v", options.Packages)
	}
	return packages, nil
}

// listTests runs `go test -list ^Test` on the host for one package.
func listTests(ctx context.Context, executor Executor, options Options, tags []string, pattern string) ([]string, error) {
	completed := executor.Exec(ctx, Command{Path: options.Compiler, Args: []string{"test", "-count=1", "-tags", strings.Join(tags, ","), "-list", "^Test", pattern}, Dir: options.Checkout, Env: goEnvironment(options.Environ, "", ""), Timeout: 30 * time.Minute})
	if completed.Err != nil || completed.Exit != 0 {
		return nil, fmt.Errorf("go test -list %s: exit %d %v\n%s%s", pattern, completed.Exit, completed.Err, completed.Stdout, completed.Stderr)
	}
	return parseTestList(completed.Stdout), nil
}

// parseTestList keeps the identifier lines of `go test -list` output and
// drops the trailing "ok <package> <time>" summary.
func parseTestList(output []byte) []string {
	var names []string
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.ContainsAny(line, " \t") || !isIdentifier(line) {
			continue
		}
		names = append(names, line)
	}
	sort.Strings(names)
	return slices.Compact(names)
}

func isIdentifier(value string) bool {
	for index, r := range value {
		if !unicode.IsLetter(r) && r != '_' && (index == 0 || !unicode.IsDigit(r)) {
			return false
		}
	}
	return value != ""
}

// parseTests returns the top-level Test functions of the named files, keyed
// by name, with the file that defines each. It applies the go test naming
// rule (Test, then end or a non-lowercase rune) and skips TestMain and
// methods.
func parseTests(dir string, files []string) (map[string]string, error) {
	tests := map[string]string{}
	set := token.NewFileSet()
	for _, name := range files {
		file, err := parser.ParseFile(set, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Join(dir, name), err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Type.TypeParams != nil || !isTestName(function.Name.Name) || function.Name.Name == "TestMain" {
				continue
			}
			if function.Type.Params == nil || len(function.Type.Params.List) != 1 || len(function.Type.Params.List[0].Names) > 1 {
				continue
			}
			if previous, ok := tests[function.Name.Name]; ok {
				return nil, fmt.Errorf("test %s defined in %s and %s", function.Name.Name, previous, name)
			}
			tests[function.Name.Name] = name
		}
	}
	return tests, nil
}

func isTestName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	if len(name) == len("Test") {
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len("Test"):])
	return !unicode.IsLower(r)
}
