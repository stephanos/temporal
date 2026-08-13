package boundary

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
)

type sourceFingerprint struct {
	source            string
	declarationSHA256 string
	packageSHA256     string
}

type sourcePackage struct {
	fileSet      *token.FileSet
	declarations []*ast.FuncDecl
	files        map[*ast.FuncDecl]string
	sha256       string
}

// Qualify checks every declared target against the standard library used to
// run the command, including its exact type and selected source implementation.
func Qualify(root string) error {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return err
	}
	typedPackages := make(map[string]*types.Package)
	sourcePackages := make(map[string]sourcePackage)
	sourceImporter := importer.ForCompiler(token.NewFileSet(), "source", nil)
	for _, entry := range definition.Intercepts {
		pkg := typedPackages[entry.Package]
		if pkg == nil {
			pkg, err = sourceImporter.Import(entry.Package)
			if err != nil {
				return fmt.Errorf("load boundary package %s: %w", entry.Package, err)
			}
			typedPackages[entry.Package] = pkg
		}
		function, lookupErr := lookupFunction(pkg, entry)
		if lookupErr != nil {
			return lookupErr
		}
		actualSignature := types.TypeString(function.Type(), func(imported *types.Package) string {
			if imported.Path() == pkg.Path() {
				return ""
			}
			return imported.Name()
		})
		if actualSignature != entry.Signature {
			return fmt.Errorf("boundary signature mismatch for %s.%s: got %q, want %q", entry.Package, targetName(entry.Receiver, entry.Symbol), actualSignature, entry.Signature)
		}
		sourcePkg, found := sourcePackages[entry.Package]
		if !found {
			sourcePkg, err = loadSourcePackage(entry.Package)
			if err != nil {
				return err
			}
			sourcePackages[entry.Package] = sourcePkg
		}
		fingerprint, fingerprintErr := sourcePkg.fingerprint(entry)
		if fingerprintErr != nil {
			return fingerprintErr
		}
		if fingerprint.source != entry.Source || fingerprint.declarationSHA256 != entry.DeclarationSHA256 || fingerprint.packageSHA256 != entry.PackageSHA256 {
			return fmt.Errorf("boundary source fingerprint mismatch for %s.%s: run boundarygen -refresh", entry.Package, targetName(entry.Receiver, entry.Symbol))
		}
	}
	return nil
}

// RefreshFingerprints updates only source-derived qualification fields in the
// canonical manifest.
func RefreshFingerprints(root string) error {
	path := filepath.Join(root, filepath.FromSlash(manifestPath))
	definition, err := decode(path)
	if err != nil {
		return err
	}
	validation := definition
	validation.Intercepts = append([]intercept(nil), definition.Intercepts...)
	for index := range validation.Intercepts {
		validation.Intercepts[index].Source = validation.Intercepts[index].Package + "/refresh.go"
		validation.Intercepts[index].DeclarationSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		validation.Intercepts[index].PackageSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	}
	if err := validate(validation); err != nil {
		return err
	}
	packages := make(map[string]sourcePackage)
	for index, entry := range definition.Intercepts {
		pkg, found := packages[entry.Package]
		if !found {
			pkg, err = loadSourcePackage(entry.Package)
			if err != nil {
				return err
			}
			packages[entry.Package] = pkg
		}
		fingerprint, fingerprintErr := pkg.fingerprint(entry)
		if fingerprintErr != nil {
			return fingerprintErr
		}
		definition.Intercepts[index].Source = fingerprint.source
		definition.Intercepts[index].DeclarationSHA256 = fingerprint.declarationSHA256
		definition.Intercepts[index].PackageSHA256 = fingerprint.packageSHA256
	}
	encoded, err := json.MarshalIndent(definition, "", "  ")
	if err != nil {
		return fmt.Errorf("encode refreshed boundary manifest: %w", err)
	}
	return writeAtomic(path, append(encoded, '\n'))
}

func lookupFunction(pkg *types.Package, entry intercept) (*types.Func, error) {
	var object types.Object
	if entry.Receiver == nil {
		object = pkg.Scope().Lookup(entry.Symbol)
	} else {
		typeName, ok := pkg.Scope().Lookup(entry.Receiver.Name).(*types.TypeName)
		if !ok {
			return nil, fmt.Errorf("boundary receiver is missing: %s.%s", entry.Package, entry.Receiver.Name)
		}
		receiverType := typeName.Type()
		if entry.Receiver.Pointer {
			receiverType = types.NewPointer(receiverType)
		}
		object, _, _ = types.LookupFieldOrMethod(receiverType, true, pkg, entry.Symbol)
	}
	function, ok := object.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("boundary target is missing: %s.%s", entry.Package, targetName(entry.Receiver, entry.Symbol))
	}
	return function, nil
}

func loadSourcePackage(packagePath string) (sourcePackage, error) {
	context := build.Default
	context.CgoEnabled = false
	description, err := context.Import(packagePath, "", 0)
	if err != nil {
		return sourcePackage{}, fmt.Errorf("locate boundary package %s: %w", packagePath, err)
	}
	fileNames := append([]string(nil), description.GoFiles...)
	fileNames = append(fileNames, description.CgoFiles...)
	slices.Sort(fileNames)
	fileSet := token.NewFileSet()
	result := sourcePackage{fileSet: fileSet, files: make(map[*ast.FuncDecl]string)}
	packageHash := sha256.New()
	for _, name := range fileNames {
		if name == "gomad.go" {
			continue
		}
		path := filepath.Join(description.Dir, name)
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return sourcePackage{}, fmt.Errorf("read boundary source %s: %w", path, readErr)
		}
		packageHash.Write([]byte(name))
		packageHash.Write([]byte{0})
		packageHash.Write(contents)
		packageHash.Write([]byte{0})
		parsed, parseErr := parser.ParseFile(fileSet, path, contents, parser.ParseComments)
		if parseErr != nil {
			return sourcePackage{}, fmt.Errorf("parse boundary source %s: %w", path, parseErr)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			result.declarations = append(result.declarations, function)
			result.files[function] = filepath.ToSlash(filepath.Join(packagePath, name))
		}
	}
	for _, name := range description.SFiles {
		path := filepath.Join(description.Dir, name)
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return sourcePackage{}, fmt.Errorf("read boundary assembly %s: %w", path, readErr)
		}
		packageHash.Write([]byte(name))
		packageHash.Write([]byte{0})
		packageHash.Write(contents)
		packageHash.Write([]byte{0})
	}
	result.sha256 = fmt.Sprintf("sha256:%x", packageHash.Sum(nil))
	return result, nil
}

func (pkg sourcePackage) fingerprint(entry intercept) (sourceFingerprint, error) {
	var matches []*ast.FuncDecl
	for _, declaration := range pkg.declarations {
		if declaration.Name.Name == entry.Symbol && receiverMatches(declaration.Recv, entry.Receiver) {
			matches = append(matches, declaration)
		}
	}
	if len(matches) != 1 {
		return sourceFingerprint{}, fmt.Errorf("boundary source target %s.%s has %d definitions", entry.Package, targetName(entry.Receiver, entry.Symbol), len(matches))
	}
	var declaration bytes.Buffer
	if err := format.Node(&declaration, pkg.fileSet, matches[0]); err != nil {
		return sourceFingerprint{}, fmt.Errorf("format boundary declaration %s.%s: %w", entry.Package, targetName(entry.Receiver, entry.Symbol), err)
	}
	digest := sha256.Sum256(declaration.Bytes())
	return sourceFingerprint{
		source: pkg.files[matches[0]], declarationSHA256: fmt.Sprintf("sha256:%x", digest), packageSHA256: pkg.sha256,
	}, nil
}

func receiverMatches(fields *ast.FieldList, want *receiver) bool {
	if want == nil {
		return fields == nil
	}
	if fields == nil || len(fields.List) != 1 {
		return false
	}
	actual := fields.List[0].Type
	pointer := false
	if star, ok := actual.(*ast.StarExpr); ok {
		pointer = true
		actual = star.X
	}
	identifier, ok := actual.(*ast.Ident)
	return ok && identifier.Name == want.Name && pointer == want.Pointer
}
