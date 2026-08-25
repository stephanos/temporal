package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

const dynamicConfigPackagePath = "go.temporal.io/server/common/dynamicconfig"

type RegistrationSite struct {
	Key     string `json:"key"`
	Package string `json:"package"`
	File    string `json:"file"`
	Line    int    `json:"line"`
}

func discoverRegistrationSites(ctx context.Context, moduleRoot string) ([]RegistrationSite, error) {
	absoluteRoot, err := filepath.Abs(moduleRoot)
	if err != nil {
		return nil, fmt.Errorf("discovery root %q: %w", moduleRoot, err)
	}
	loaded, err := packages.Load(&packages.Config{
		Context: ctx,
		Dir:     absoluteRoot,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule,
		Tests:      false,
		BuildFlags: []string{"-tags=test_dep"},
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("discovery load %q: %w", absoluteRoot, err)
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("discovery load %q: no packages", absoluteRoot)
	}

	slices.SortFunc(loaded, func(left, right *packages.Package) int {
		return strings.Compare(left.PkgPath, right.PkgPath)
	})
	var sites []RegistrationSite
	for _, loadedPackage := range loaded {
		if err := packageLoadError(loadedPackage, absoluteRoot); err != nil {
			return nil, err
		}
		packageSites, err := discoverPackageSites(loadedPackage, absoluteRoot)
		if err != nil {
			return nil, err
		}
		sites = append(sites, packageSites...)
	}
	if len(sites) == 0 {
		return nil, errors.New("discovery: no production dynamic config initializers found")
	}
	slices.SortFunc(sites, compareRegistrationSites)
	for index := 1; index < len(sites); index++ {
		if sites[index-1].Key == sites[index].Key {
			return nil, fmt.Errorf(
				"discovery key %q: duplicate initializers at %s:%d and %s:%d",
				sites[index].Key,
				sites[index-1].File,
				sites[index-1].Line,
				sites[index].File,
				sites[index].Line,
			)
		}
	}
	return sites, nil
}

func packageLoadError(loadedPackage *packages.Package, moduleRoot string) error {
	if !loadedPackage.IllTyped && len(loadedPackage.Errors) == 0 {
		return nil
	}
	errors := slices.Clone(loadedPackage.Errors)
	slices.SortFunc(errors, func(left, right packages.Error) int {
		if order := strings.Compare(left.Pos, right.Pos); order != 0 {
			return order
		}
		return strings.Compare(left.Msg, right.Msg)
	})
	if len(errors) == 0 {
		return fmt.Errorf("discovery package %q: package is ill typed", loadedPackage.PkgPath)
	}
	first := errors[0]
	position := normalizePosition(first.Pos, moduleRoot)
	if position == "" {
		return fmt.Errorf("discovery package %q: %s", loadedPackage.PkgPath, first.Msg)
	}
	return fmt.Errorf("discovery package %q at %s: %s", loadedPackage.PkgPath, position, first.Msg)
}

func discoverPackageSites(loadedPackage *packages.Package, moduleRoot string) ([]RegistrationSite, error) {
	var sites []RegistrationSite
	for _, file := range loadedPackage.Syntax {
		position := loadedPackage.Fset.Position(file.Pos())
		if strings.HasSuffix(position.Filename, "_test.go") {
			continue
		}
		found, err := discoverFileSites(loadedPackage, moduleRoot, file)
		if err != nil {
			return nil, err
		}
		sites = append(sites, found...)
	}
	return sites, nil
}

func discoverFileSites(
	loadedPackage *packages.Package,
	moduleRoot string,
	file *ast.File,
) ([]RegistrationSite, error) {
	var sites []RegistrationSite
	for _, declaration := range file.Decls {
		found, err := discoverDeclarationSites(loadedPackage, moduleRoot, declaration)
		if err != nil {
			return nil, err
		}
		sites = append(sites, found...)
	}
	return sites, nil
}

func discoverDeclarationSites(
	loadedPackage *packages.Package,
	moduleRoot string,
	declaration ast.Decl,
) ([]RegistrationSite, error) {
	switch typed := declaration.(type) {
	case *ast.GenDecl:
		return discoverVariableSites(loadedPackage, moduleRoot, typed)
	case *ast.FuncDecl:
		if typed.Recv == nil && typed.Name.Name == "init" && typed.Body != nil {
			return registrationCalls(loadedPackage, moduleRoot, typed.Body)
		}
		return nil, nil
	default:
		return nil, nil
	}
}

func discoverVariableSites(
	loadedPackage *packages.Package,
	moduleRoot string,
	declaration *ast.GenDecl,
) ([]RegistrationSite, error) {
	if declaration.Tok.String() != "var" {
		return nil, nil
	}
	var sites []RegistrationSite
	for _, specification := range declaration.Specs {
		valueSpec, ok := specification.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, expression := range valueSpec.Values {
			found, err := registrationCalls(loadedPackage, moduleRoot, expression)
			if err != nil {
				return nil, err
			}
			sites = append(sites, found...)
		}
	}
	return sites, nil
}

func registrationCalls(
	loadedPackage *packages.Package,
	moduleRoot string,
	root ast.Node,
) ([]RegistrationSite, error) {
	var sites []RegistrationSite
	var visitErr error
	ast.Inspect(root, func(node ast.Node) bool {
		if visitErr != nil || node == nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		function := calledFunction(loadedPackage.TypesInfo, call.Fun)
		if function == nil || function.Pkg() == nil ||
			function.Pkg().Path() != dynamicConfigPackagePath || !isSettingConstructor(function.Name()) {
			return true
		}
		position := loadedPackage.Fset.Position(call.Pos())
		if len(call.Args) == 0 {
			visitErr = fmt.Errorf(
				"discovery package %q at %s: constructor %s has no key argument",
				loadedPackage.PkgPath,
				normalizePosition(position.String(), moduleRoot),
				function.Name(),
			)
			return false
		}
		value := loadedPackage.TypesInfo.Types[call.Args[0]].Value
		if value == nil || value.Kind() != constant.String {
			visitErr = fmt.Errorf(
				"discovery package %q at %s: constructor %s key is not a constant string",
				loadedPackage.PkgPath,
				normalizePosition(position.String(), moduleRoot),
				function.Name(),
			)
			return false
		}
		file, err := filepath.Rel(moduleRoot, position.Filename)
		if err != nil {
			visitErr = fmt.Errorf("discovery package %q: relativize %q: %w", loadedPackage.PkgPath, position.Filename, err)
			return false
		}
		sites = append(sites, RegistrationSite{
			Key:     strings.ToLower(constant.StringVal(value)),
			Package: loadedPackage.PkgPath,
			File:    filepath.ToSlash(file),
			Line:    position.Line,
		})
		return true
	})
	return sites, visitErr
}

func calledFunction(info *types.Info, expression ast.Expr) *types.Func {
	switch typed := expression.(type) {
	case *ast.Ident:
		return functionObject(info.Uses[typed])
	case *ast.SelectorExpr:
		return functionObject(info.Uses[typed.Sel])
	case *ast.IndexExpr:
		return calledFunction(info, typed.X)
	case *ast.IndexListExpr:
		return calledFunction(info, typed.X)
	default:
		return nil
	}
}

func functionObject(object types.Object) *types.Func {
	function, ok := object.(*types.Func)
	if !ok {
		return nil
	}
	return function
}

func isSettingConstructor(name string) bool {
	if !strings.HasPrefix(name, "New") {
		return false
	}
	return strings.HasSuffix(name, "Setting") ||
		strings.HasSuffix(name, "SettingWithConverter") ||
		strings.HasSuffix(name, "SettingWithConstrainedDefault")
}

func normalizePosition(position, moduleRoot string) string {
	return strings.ReplaceAll(position, filepath.Clean(moduleRoot)+string(filepath.Separator), "")
}

func compareRegistrationSites(left, right RegistrationSite) int {
	if order := strings.Compare(left.Key, right.Key); order != 0 {
		return order
	}
	if order := strings.Compare(left.Package, right.Package); order != 0 {
		return order
	}
	if order := strings.Compare(left.File, right.File); order != 0 {
		return order
	}
	return left.Line - right.Line
}
