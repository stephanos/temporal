// runtimeinventory derives compiler-overlay source sites; the receiving native
// inventory gate decides whether those sites have reviewed classifications.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
)

const platform = "wasip1/wasm"

var (
	clockIdentifier      = regexp.MustCompile(`\b(cputicks|libc_gettimeofday|libc_gettimeofday_trampoline|nanotime1|walltime|walltime1|time_now|vdsoClockgettimeSym|vdsoGettimeofdaySym|clock_time_get)\b`)
	glueClockIdentifier  = regexp.MustCompile(`\b(nanotime)\b`)
	drawPatchIdentifier  = regexp.MustCompile(`\b(cheaprand|cheaprandn|cheaprand64|rand|randn|mrandinit|randinit|startupRand|randomizeScheduler|gomadSeed|gomadWasmInit|gomadChoiceRunnextSeeded|gomadChoiceRunqIndex|gomadChoiceSelectSeeded|gomadRuntimeRand|gomadRuntimeCheapRand|gomadTimerRand)\b`)
	clockPatchIdentifier = regexp.MustCompile(`\b(nanotime|nanotime1|walltime|walltime1|usleep|notetsleepg|beforeIdle|time_now|cputicks|gomadWasmLastTime)\b`)
)

type overlayPatch struct {
	file, from, to string
	count          int
}

type inventory struct {
	Schema               string                    `json:"schema"`
	Platform             string                    `json:"platform"`
	GoVersion            string                    `json:"go_version"`
	OverlayIdentity      string                    `json:"overlay_identity"`
	ImplementationSHA256 string                    `json:"implementation_sha256"`
	OverlayInputs        map[string]string         `json:"overlay_inputs"`
	Sources              map[string]string         `json:"sources"`
	Sites                map[string]map[string]int `json:"sites"`
}

func read(path string, sources map[string]string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	sources[absolute] = hex.EncodeToString(sum[:])
	return data, nil
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: runtimeinventory STOCK_ROOT SEMANTICS_ROOT OUTPUT_ROOT")
		os.Exit(2)
	}
	result, err := derive(os.Args[1], os.Args[2], os.Args[3])
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(result)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func derive(stock, semantics, output string) (inventory, error) {
	result := inventory{Schema: "gomad.runtime-source-sites/v1", Platform: platform, Sources: map[string]string{}}
	var err error
	stock, err = filepath.EvalSymlinks(stock)
	if err != nil {
		return result, err
	}
	stock, err = filepath.Abs(stock)
	if err != nil {
		return result, err
	}
	version, err := read(filepath.Join(stock, "VERSION"), result.Sources)
	if err != nil {
		return result, err
	}
	result.GoVersion = strings.SplitN(string(version), "\n", 2)[0]
	path, identity, err := toolchain.BuildOverlay(stock, semantics, output)
	if err != nil {
		return result, err
	}
	result.OverlayIdentity = identity
	result.ImplementationSHA256 = toolchain.ImplementationSHA256()
	manifest, err := read(path, result.Sources)
	if err != nil {
		return result, err
	}
	var overlay struct{ Replace map[string]string }
	if err := json.Unmarshal(manifest, &overlay); err != nil {
		return result, err
	}
	// The compiler pin, rewrite program, embedded glue and shared semantics are
	// inputs as well as the selected effective source files.
	for _, root := range []string{"internal/backends/wasm/toolchain", semantics} {
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".txt") {
				_, err = read(path, result.Sources)
			}
			return err
		})
		if err != nil {
			return result, err
		}
	}
	source, err := read("internal/backends/wasm/toolchain/overlay.go", result.Sources)
	if err != nil {
		return result, err
	}
	patches, err := overlayPatches(source)
	if err != nil {
		return result, err
	}
	draws, others, err := patchReferences(stock, patches, result.Sources)
	if err != nil {
		return result, err
	}
	clocks, err := clockReferences(stock, overlay.Replace, result.Sources)
	if err != nil {
		return result, err
	}
	result.OverlayInputs = map[string]string{"VERSION": filepath.Join(stock, "VERSION"), "wasm-glue": "internal/backends/wasm/toolchain/runtime_wasm.go.txt", "wasm-inspect-glue": "internal/backends/wasm/toolchain/runtime_inspect_wasm.go.txt", "wasm-net-stream-glue": "internal/backends/wasm/toolchain/net_stream_wasm.go.txt", "wasm-net-stream-status": "internal/backends/wasm/toolchain/net_stream_status_generated.go.txt"}
	for _, patch := range patches {
		result.OverlayInputs["stock/"+patch.file] = filepath.Join(stock, "src", stockPath(patch.file))
	}
	for original, replacement := range overlay.Replace {
		relative, err := filepath.Rel(filepath.Join(stock, "src"), original)
		if err != nil {
			return result, err
		}
		name := filepath.ToSlash(relative)
		if strings.HasPrefix(name, "runtime/") {
			name = strings.TrimPrefix(name, "runtime/")
		}
		result.OverlayInputs["output/"+name] = replacement
	}
	for _, name := range []string{"gomad_semantics.go", "gomad_choicewire_generated.go"} {
		result.OverlayInputs["semantics/"+name] = filepath.Join(semantics, name+".txt")
	}
	for name, path := range result.OverlayInputs {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return result, err
		}
		result.OverlayInputs[name] = absolute
		if _, err := read(absolute, result.Sources); err != nil {
			return result, err
		}
	}
	result.Sites = map[string]map[string]int{"runtime-draw": draws, "runtime-clock": clocks, "runtime-other": others}
	return result, nil
}

func overlayPatches(source []byte) ([]overlayPatch, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "overlay.go", source, 0)
	if err != nil {
		return nil, err
	}
	constants := map[string]string{}
	var rewrites *ast.CompositeLit
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok && declaration.Tok == token.CONST && len(value.Values) == len(value.Names) {
					for index, name := range value.Names {
						if literal, ok := value.Values[index].(*ast.BasicLit); ok && literal.Kind == token.STRING {
							constants[name.Name], err = strconv.Unquote(literal.Value)
							if err != nil {
								return nil, err
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if declaration.Name.Name == "stockRewrites" {
				ast.Inspect(declaration.Body, func(node ast.Node) bool {
					if composite, ok := node.(*ast.CompositeLit); ok && rewrites == nil {
						rewrites = composite
						return false
					}
					return true
				})
			}
		}
	}
	if rewrites == nil {
		return nil, errors.New("overlay.go has no stockRewrites map")
	}
	text := func(expression ast.Expr) (string, error) {
		switch expression := expression.(type) {
		case *ast.BasicLit:
			if expression.Kind == token.STRING {
				return strconv.Unquote(expression.Value)
			}
		case *ast.Ident:
			if value, ok := constants[expression.Name]; ok {
				return value, nil
			}
		}
		return "", fmt.Errorf("overlay rewrite operand %T not understood", expression)
	}
	var patches []overlayPatch
	for _, element := range rewrites.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return nil, errors.New("overlay rewrite entry is not a file map")
		}
		name, err := text(pair.Key)
		if err != nil {
			return nil, err
		}
		rules, ok := pair.Value.(*ast.CompositeLit)
		if !ok {
			return nil, errors.New("overlay rewrite rules are not a literal")
		}
		for _, rule := range rules.Elts {
			fields, ok := rule.(*ast.CompositeLit)
			if !ok || len(fields.Elts) != 3 {
				return nil, fmt.Errorf("overlay rewrite for %s requires three fields", name)
			}
			from, err := text(fields.Elts[0])
			if err != nil {
				return nil, err
			}
			to, err := text(fields.Elts[1])
			if err != nil {
				return nil, err
			}
			literal, ok := fields.Elts[2].(*ast.BasicLit)
			if !ok || literal.Kind != token.INT {
				return nil, errors.New("overlay rewrite count is not an integer")
			}
			count, err := strconv.Atoi(literal.Value)
			if err != nil || count <= 0 {
				return nil, errors.New("overlay rewrite count must be positive")
			}
			patches = append(patches, overlayPatch{name, from, to, count})
		}
	}
	if len(patches) == 0 {
		return nil, errors.New("overlay has no rewrites")
	}
	return patches, nil
}

func stockPath(name string) string {
	if strings.Contains(name, "/") {
		return name
	}
	return "runtime/" + name
}

func enclosingFunction(source []byte, offset int) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "", source, parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || int(function.Pos())-1 > offset || offset >= int(function.End())-1 {
			continue
		}
		if function.Recv != nil {
			return "method." + function.Name.Name, nil
		}
		return function.Name.Name, nil
	}
	return "package", nil
}

func patchReferences(stock string, patches []overlayPatch, sources map[string]string) (map[string]int, map[string]int, error) {
	draws, other := map[string]int{}, map[string]int{}
	for _, patch := range patches {
		var symbol string
		for _, pattern := range []*regexp.Regexp{drawPatchIdentifier, clockPatchIdentifier} {
			if symbol != "" {
				break
			}
			for _, candidate := range pattern.FindAllString(patch.to, -1) {
				if !regexp.MustCompile(`\b`+candidate+`\b`).MatchString(patch.from) && (symbol == "" || !strings.HasPrefix(symbol, "gomad") && strings.HasPrefix(candidate, "gomad")) {
					symbol = candidate
				}
			}
			if symbol == "" && pattern.MatchString(patch.from) {
				symbol = pattern.FindString(patch.from)
			}
		}
		source, err := read(filepath.Join(stock, "src", filepath.FromSlash(stockPath(patch.file))), sources)
		if err != nil {
			return nil, nil, err
		}
		offset := bytes.Index(source, []byte(patch.from))
		if offset < 0 {
			return nil, nil, fmt.Errorf("overlay anchor for %s not in stock source: %q", patch.file, patch.from)
		}
		function, err := enclosingFunction(source, offset)
		if err != nil {
			return nil, nil, err
		}
		if symbol == "" {
			other[strings.Join([]string{platform, stockPath(patch.file), function, "patch"}, " ")] += patch.count
		} else {
			draws[strings.Join([]string{platform, stockPath(patch.file), function, symbol, "patch"}, " ")] += patch.count
		}
	}
	return draws, other, nil
}

func clockReferences(stock string, replace map[string]string, sources map[string]string) (map[string]int, error) {
	root := filepath.Join(stock, "src")
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" || entry.Name() == "vendor" || path == filepath.Join(root, "cmd") {
				return filepath.SkipDir
			}
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for path := range replace {
		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	context := build.Default
	context.GOROOT, context.GOOS, context.GOARCH, context.CgoEnabled = stock, "wasip1", "wasm", false
	context.ToolTags = slices.DeleteFunc(slices.Clone(build.Default.ToolTags), func(tag string) bool { return tag == "goexperiment.greenteagc" })
	context.OpenFile = func(path string) (io.ReadCloser, error) {
		if replacement, ok := replace[path]; ok {
			path = replacement
		}
		data, err := read(path, sources)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	got := map[string]int{}
	for _, path := range paths {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") || (!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s")) {
			continue
		}
		match, err := context.MatchFile(filepath.Dir(path), name)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}
		selected := path
		if replacement, ok := replace[path]; ok {
			selected = replacement
		}
		contents, err := read(selected, sources)
		if err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		relative = filepath.ToSlash(relative)
		patterns := []*regexp.Regexp{clockIdentifier}
		if relative == "runtime/gomad_wasm.go" {
			patterns = append(patterns, glueClockIdentifier)
		}
		for line := range strings.Lines(string(contents)) {
			if comment := strings.Index(line, "//"); comment >= 0 && !strings.HasPrefix(strings.TrimSpace(line), "//go:") {
				line = line[:comment]
			}
			for _, pattern := range patterns {
				for _, identifier := range pattern.FindAllString(line, -1) {
					got[strings.Join([]string{platform, relative, identifier}, " ")]++
				}
			}
		}
	}
	return got, nil
}
