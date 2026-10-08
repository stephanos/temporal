package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type replacement struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type origin struct {
	Path            string        `json:"path"`
	Function        string        `json:"function"`
	Portable        string        `json:"portable"`
	SourceSHA256    string        `json:"source_sha256"`
	FunctionSHA256  string        `json:"function_sha256"`
	GeneratedSHA256 string        `json:"generated_sha256"`
	Replacements    []replacement `json:"replacements"`
}

func digest(source []byte) string { return fmt.Sprintf("%x", sha256.Sum256(source)) }

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	out := filepath.Join(root, ".flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008")
	selected := map[string][]string{
		"adapter_rewrite_test.go":   {"TestRewriteAdapterSourceRejectsDrift"},
		"grpc_adapter_test.go":      {"TestPinnedAdapterModuleInventories", "TestRewriteGRPCKeepalivePreservesDialerWithoutHostControl", "TestRewriteGRPCLinuxSourcesCompileTheNonLinuxImplementations", "TestRewriteGRPCKeepaliveRejectsSourceIdentityDrift", "TestRewriteGRPCKeepaliveSourceRejectsChangedAnchor", "TestRewriteGRPCKeepaliveSourceRejectsDuplicateAnchor", "TestPrepareGRPCRecordsExactPrivateReplacement", "TestPrepareGRPCRejectsChangedIdentity", "readPinnedGRPCKeepalive"},
		"grpc_dns_test.go":          {"TestGRPCDNSRewrite", "TestGRPCDNSConsumer"},
		"memory_adapter_test.go":    {"TestRewriteModerncMemoryModelsOnlyAnonymousAllocatorMappings", "TestRewriteModerncMemoryRejectsSourceIdentityDrift", "TestModerncMemoryRewriteRejectsAnchorDrift", "TestPrepareModerncMemoryRejectsModuleDrift", "TestModerncMemoryRejectsChangedReplacementInventory", "TestPrepareModerncMemoryRecordsExactPrivateReplacement", "TestPrepareModerncMemoryRejectsChangedIdentity", "readPinnedModerncMemorySource"},
		"xnet_adapter_test.go":      {"TestRewriteXNetSocketDeniesRawSocketOptions", "TestRewriteXNetSocketRejectsSourceIdentityDrift", "TestPrepareXNetRecordsExactPrivateReplacement", "TestPrepareXNetRejectsChangedIdentity", "readPinnedXNetSocketSources"},
		"sockaddr_boundary_test.go": {"TestSockaddrBoundaryConsumer", "TestSockaddrBoundaryRewritesRejectDrift", "TestSockaddrBoundaryPreservesOriginalComments"},
	}
	renames := map[string]string{"pinnedModuleCache": "portableRetainedModuleCache", "downloadPinnedModule": "portableRetainedDownloadPinnedModule", "readPinnedGRPCKeepalive": "portableRetainedReadPinnedGRPCKeepalive", "readPinnedModerncMemorySource": "portableRetainedReadPinnedModerncMemorySource", "readPinnedXNetSocketSources": "portableRetainedReadPinnedXNetSocketSources"}
	var fileNames []string
	for name := range selected {
		fileNames = append(fileNames, name)
	}
	slices.Sort(fileNames)
	imports := map[string]string{"testing": "\"testing\""}
	var bodies []string
	var origins []origin
	for _, name := range fileNames {
		relative := "tools/gomad3/deterministicio/" + name
		source, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, relative, source, parser.ParseComments)
		if err != nil {
			return err
		}
		found := map[string]bool{}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !slices.Contains(selected[name], function.Name.Name) {
				continue
			}
			found[function.Name.Name] = true
			start, end := fset.Position(function.Pos()).Offset, fset.Position(function.End()).Offset
			if function.Doc != nil {
				start = fset.Position(function.Doc.Pos()).Offset
			}
			original := string(source[start:end])
			portable := renames[function.Name.Name]
			if strings.HasPrefix(function.Name.Name, "Test") {
				portable = "TestPortableRetained" + strings.TrimPrefix(function.Name.Name, "Test")
			}
			var changes []replacement
			add := func(pos, end token.Pos, to string) {
				a, b := fset.Position(pos).Offset-start, fset.Position(end).Offset-start
				changes = append(changes, replacement{Start: a, End: b, From: original[a:b], To: to})
			}
			add(function.Name.Pos(), function.Name.End(), portable)
			used := map[string]bool{}
			ast.Inspect(function, func(node ast.Node) bool {
				if selector, ok := node.(*ast.SelectorExpr); ok {
					if identifier, ok := selector.X.(*ast.Ident); ok {
						used[identifier.Name] = true
					}
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if identifier, ok := call.Fun.(*ast.Ident); ok {
					if to, ok := renames[identifier.Name]; ok {
						add(identifier.Pos(), identifier.End(), to)
					}
				}
				if function.Name.Name == "TestGRPCDNSConsumer" || function.Name.Name == "TestSockaddrBoundaryConsumer" {
					for _, argument := range call.Args {
						if literal, ok := argument.(*ast.BasicLit); ok && literal.Value == `"test"` {
							add(literal.Pos(), literal.End(), `"test", "-tags=test_dep"`)
						}
					}
				}
				return true
			})
			for _, imported := range file.Imports {
				value, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				alias := filepath.Base(value)
				text := imported.Path.Value
				if imported.Name != nil {
					alias = imported.Name.Name
					text = alias + " " + text
				}
				if used[alias] {
					imports[alias] = text
				}
			}
			slices.SortFunc(changes, func(a, b replacement) int { return b.Start - a.Start })
			generated := original
			for _, change := range changes {
				generated = generated[:change.Start] + change.To + generated[change.End:]
			}
			bodies = append(bodies, generated)
			origins = append(origins, origin{Path: relative, Function: function.Name.Name, Portable: portable, SourceSHA256: digest(source), FunctionSHA256: digest([]byte(original)), GeneratedSHA256: digest([]byte(generated)), Replacements: changes})
		}
		for _, function := range selected[name] {
			if !found[function] {
				return fmt.Errorf("selected origin %s/%s missing", name, function)
			}
		}
	}
	var importLines []string
	for _, text := range imports {
		importLines = append(importLines, text)
	}
	slices.Sort(importLines)
	generated := "package deterministicio\n\nimport (\n" + strings.Join(importLines, "\n") + "\n)\n\n" + strings.Join(bodies, "\n\n") + ` 

func portableRetainedModuleCache(t *testing.T) string {
	t.Helper()
	_, cache := portableAdapterGo(t)
	return cache
}

func portableRetainedDownloadPinnedModule(t *testing.T, name, version string) {
	t.Helper()
	portableAdapterModule(t, portableRetainedModuleCache(t), name, version)
}
`
	actual := filepath.Join(out, "retained-assertions-generated.go")
	logical := filepath.Join(root, "tools/gomad3/deterministicio/retained_assertions_portable_generated_test.go")
	if _, err := os.Stat(logical); !os.IsNotExist(err) {
		return fmt.Errorf("additive overlay logical path already exists or is inaccessible: %v", err)
	}
	if err := os.WriteFile(actual, []byte(generated), 0o600); err != nil {
		return err
	}
	mapping, err := json.MarshalIndent(map[string]any{"Replace": map[string]string{logical: actual}}, "", "  ")
	if err != nil {
		return err
	}
	mapping = append(mapping, '\n')
	if err := os.WriteFile(filepath.Join(out, "retained-assertions-overlay.json"), mapping, 0o600); err != nil {
		return err
	}
	recipe, err := os.ReadFile(filepath.Join(out, "retained_overlay.go"))
	if err != nil {
		return err
	}
	manifest, err := json.MarshalIndent(map[string]any{"kind": "additive portable retained assertions; originals and guards unchanged; no native credit", "origins": origins, "recipe_sha256": digest(recipe), "generated_sha256": digest([]byte(generated)), "mapping_sha256": digest(mapping), "logical": logical, "actual": actual, "transformations": "AST identifier replacements and the two explicitly admitted test_dep argument additions; every other origin byte preserved"}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "retained-assertions-binding.json"), append(manifest, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Printf("generated %d bound original functions including three helper clones\n", len(origins))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
