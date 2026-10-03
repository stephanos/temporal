package deterministicio

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// sourceEdit replaces one string literal of an adapter source file.
type sourceEdit struct {
	start, end int
	text       string
}

// literalAnchor is one pinned identity the adapter declares as a string
// literal, with the value that replaces it.
type literalAnchor struct {
	name, previous, proposed string
	platform                 string
	found                    int
}

// SourceEdits returns the deterministic I/O package files under root, the
// Gomad source module root, rewritten from the previous anchors to the
// proposed ones, keyed by slash path relative to root. Only string literals
// change:
//
//   - each pinned digest and the module sum, wherever the exact literal occurs;
//   - each platform's prepared source-set pin, where a map keys it by platform;
//   - the version, in the adapter's own file and in records that name the
//     module (Module or Path fields);
//   - "module version", "module@version", and module-cache directory names
//     inside longer literals, such as test go.mod and go.sum fixtures.
//
// Every anchor the adapter declares must be replaced, so an anchor that is
// computed rather than written as a literal fails instead of staying stale.
func (regeneration AdapterRegeneration) SourceEdits(root string) (map[string][]byte, error) {
	spec, err := registeredRewrittenModule(regeneration.Module)
	if err != nil {
		return nil, err
	}
	anchors, err := regeneration.literalAnchors()
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(root, "deterministicio")
	names, err := filepath.Glob(filepath.Join(directory, "*.go"))
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	adapterFile := ""
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		if bytes.Contains(contents, []byte(strconv.Quote(spec.sum))) {
			if adapterFile != "" {
				return nil, fmt.Errorf("%s sum is declared in both %s and %s", spec.module, filepath.Base(adapterFile), filepath.Base(name))
			}
			adapterFile = name
		}
	}
	if adapterFile == "" {
		return nil, fmt.Errorf("no deterministic I/O source declares the %s sum", spec.module)
	}
	substrings := regeneration.versionSubstrings()
	versionFound := 0
	edited := map[string][]byte{}
	for _, name := range names {
		contents, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		edits, versions, err := regeneration.fileEdits(name, contents, anchors, substrings, name == adapterFile)
		if err != nil {
			return nil, err
		}
		versionFound += versions
		if len(edits) == 0 {
			continue
		}
		sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
		result := append([]byte(nil), contents...)
		for _, edit := range edits {
			result = append(result[:edit.start:edit.start], append([]byte(edit.text), result[edit.end:]...)...)
		}
		formatted, err := format.Source(result)
		if err != nil {
			return nil, fmt.Errorf("format regenerated %s: %w", filepath.Base(name), err)
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return nil, err
		}
		edited[filepath.ToSlash(relative)] = formatted
	}
	if versionFound == 0 {
		return nil, fmt.Errorf("%s version %s is not declared as a literal in %s", spec.module, spec.version, filepath.Base(adapterFile))
	}
	for _, anchor := range anchors {
		if anchor.found == 0 {
			return nil, fmt.Errorf("%s anchor %s (%s) is not declared as a literal in the deterministic I/O package; update it by hand", spec.module, anchor.name, anchor.previous)
		}
	}
	return edited, nil
}

// literalAnchors pairs every changed pinned value with its replacement. A
// previous value that would need two different replacements outside a
// platform-keyed map is ambiguous and fails.
func (regeneration AdapterRegeneration) literalAnchors() ([]*literalAnchor, error) {
	previous, proposed := regeneration.Previous, regeneration.Proposed
	if len(previous.Rewrites) != len(proposed.Rewrites) {
		return nil, fmt.Errorf("%s regeneration rewrites do not correspond", regeneration.Module)
	}
	anchors := []*literalAnchor{
		{name: "sum", previous: previous.Sum, proposed: proposed.Sum},
		{name: "original source inventory", previous: previous.OriginalSourceInventorySHA256, proposed: proposed.OriginalSourceInventorySHA256},
		{name: "replacement source inventory", previous: previous.ReplacementSourceInventorySHA256, proposed: proposed.ReplacementSourceInventorySHA256},
	}
	for index, rewrite := range previous.Rewrites {
		next := proposed.Rewrites[index]
		if next.Path != rewrite.Path || next.Base != rewrite.Base {
			return nil, fmt.Errorf("%s regeneration rewrite %d does not correspond", regeneration.Module, index)
		}
		anchors = append(anchors,
			&literalAnchor{name: rewrite.Path + " source", previous: rewrite.SourceSHA256, proposed: next.SourceSHA256},
			&literalAnchor{name: rewrite.Path + " replacement", previous: rewrite.ReplacementSHA256, proposed: next.ReplacementSHA256},
		)
		if rewrite.Base != "" {
			anchors = append(anchors, &literalAnchor{name: rewrite.Base + " source", previous: rewrite.BaseSHA256, proposed: next.BaseSHA256})
		}
	}
	for _, platform := range sortedKeys(previous.PreparedSourceSetSHA256) {
		next, found := proposed.PreparedSourceSetSHA256[platform]
		if !found {
			return nil, fmt.Errorf("%s regeneration has no %s prepared source set", regeneration.Module, platform)
		}
		anchors = append(anchors, &literalAnchor{name: platform + " prepared source set", previous: previous.PreparedSourceSetSHA256[platform], proposed: next, platform: platform})
	}
	changed := anchors[:0]
	for _, anchor := range anchors {
		if anchor.previous == "" || anchor.proposed == "" {
			return nil, fmt.Errorf("%s regeneration anchor %s is empty", regeneration.Module, anchor.name)
		}
		if anchor.previous != anchor.proposed {
			changed = append(changed, anchor)
		}
	}
	return changed, nil
}

type versionSubstring struct {
	previous, proposed string
}

func (regeneration AdapterRegeneration) versionSubstrings() []versionSubstring {
	module := regeneration.Module
	previous, proposed := regeneration.Previous, regeneration.Proposed
	substrings := []versionSubstring{
		{module + " " + previous.Version, module + " " + proposed.Version},
		{module + "@" + previous.Version, module + "@" + proposed.Version},
		{path.Base(module) + "@" + previous.Version, path.Base(module) + "@" + proposed.Version},
	}
	if previous.GoModSum != "" && proposed.GoModSum != "" {
		substrings = append(substrings, versionSubstring{previous.GoModSum, proposed.GoModSum})
	}
	return substrings
}

func (regeneration AdapterRegeneration) fileEdits(name string, contents []byte, anchors []*literalAnchor, substrings []versionSubstring, adapterFile bool) ([]sourceEdit, int, error) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, name, contents, parser.SkipObjectResolution)
	if err != nil {
		return nil, 0, err
	}
	handled := map[*ast.BasicLit]bool{}
	var edits []sourceEdit
	versions := 0
	replace := func(literal *ast.BasicLit, value string) error {
		text, err := quoteLike(literal.Value, value)
		if err != nil {
			return fmt.Errorf("%s: %w", files.Position(literal.Pos()), err)
		}
		edits = append(edits, sourceEdit{start: files.Position(literal.Pos()).Offset, end: files.Position(literal.End()).Offset, text: text})
		handled[literal] = true
		return nil
	}
	var walkErr error
	ast.Inspect(file, func(node ast.Node) bool {
		composite, ok := node.(*ast.CompositeLit)
		if !ok || walkErr != nil {
			return walkErr == nil
		}
		namesModule := false
		for _, element := range composite.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if key, ok := pair.Key.(*ast.Ident); ok && (key.Name == "Module" || key.Name == "Path") && stringValue(pair.Value) == regeneration.Module {
				namesModule = true
			}
		}
		for _, element := range composite.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			value, ok := pair.Value.(*ast.BasicLit)
			if !ok || value.Kind != token.STRING {
				continue
			}
			if key, ok := pair.Key.(*ast.Ident); ok && key.Name == "Version" && namesModule && stringValue(value) == regeneration.Previous.Version {
				walkErr = replace(value, regeneration.Proposed.Version)
				versions++
				continue
			}
			platform := stringValue(pair.Key)
			for _, anchor := range anchors {
				if anchor.platform != "" && anchor.platform == platform && stringValue(value) == anchor.previous {
					walkErr = replace(value, anchor.proposed)
					anchor.found++
				}
			}
		}
		return walkErr == nil
	})
	if walkErr != nil {
		return nil, 0, walkErr
	}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING || handled[literal] || walkErr != nil {
			return walkErr == nil
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			walkErr = err
			return false
		}
		if adapterFile && value == regeneration.Previous.Version {
			walkErr = replace(literal, regeneration.Proposed.Version)
			versions++
			return false
		}
		var matched []*literalAnchor
		for _, anchor := range anchors {
			if value == anchor.previous {
				matched = append(matched, anchor)
			}
		}
		if len(matched) > 0 {
			for _, anchor := range matched[1:] {
				if anchor.proposed != matched[0].proposed {
					walkErr = fmt.Errorf("%s: %s is both %s and %s, which regenerate differently; update it by hand", files.Position(literal.Pos()), value, matched[0].name, anchor.name)
					return false
				}
			}
			for _, anchor := range matched {
				anchor.found++
			}
			walkErr = replace(literal, matched[0].proposed)
			return false
		}
		next := value
		for _, substring := range substrings {
			next = replaceVersionSubstring(next, substring.previous, substring.proposed)
		}
		if strings.Contains(next, regeneration.Previous.Sum) {
			next = strings.ReplaceAll(next, regeneration.Previous.Sum, regeneration.Proposed.Sum)
		}
		if next != value {
			walkErr = replace(literal, next)
		}
		return false
	})
	return edits, versions, walkErr
}

// replaceVersionSubstring replaces previous where it is not followed by more
// version characters, so v1.2.3 never rewrites v1.2.30.
func replaceVersionSubstring(value, previous, proposed string) string {
	var result strings.Builder
	for {
		index := strings.Index(value, previous)
		if index < 0 {
			result.WriteString(value)
			return result.String()
		}
		end := index + len(previous)
		result.WriteString(value[:index])
		if end < len(value) && strings.ContainsRune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ.+-", rune(value[end])) {
			result.WriteString(previous)
		} else {
			result.WriteString(proposed)
		}
		value = value[end:]
	}
}

func stringValue(expression ast.Expr) string {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return value
}

func quoteLike(original, value string) (string, error) {
	if strings.HasPrefix(original, "`") {
		if strings.Contains(value, "`") {
			return "", fmt.Errorf("regenerated raw string literal contains a backquote")
		}
		return "`" + value + "`", nil
	}
	return strconv.Quote(value), nil
}
