// Package leannames indexes the qualified names the Lean model tree defines, so a
// document that cites one can be checked against the tree rather than against a
// reader's memory.
//
// The index is deliberately syntactic. It reads declaration heads rather than
// elaborating Lean, which is what lets an ordinary Go test run it. That trades
// exactness for reach: it never misses a name a `def` line spells, and it does not
// know a name only the elaborator produces.
package leannames

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Index answers whether one qualified Lean name exists in the model tree.
type Index struct {
	modules      map[string]struct{}
	namespaces   map[string]struct{}
	declarations map[string]struct{}
}

// declarationHead matches the head of a declaration whose name is an ordinary
// identifier. Modifiers and attributes are consumed first, so `private unsafe def`
// and `@[simp] theorem` reach the same keyword set. A `macro` or `syntax` whose next
// token is a string literal declares no name and is skipped by the identifier group.
var declarationHead = regexp.MustCompile(
	`^(?:@\[[^\]]*\]\s*)*(?:(?:private|protected|partial|unsafe|noncomputable|scoped|local|nonrec)\s+)*` +
		`(?:structure|inductive|class|abbrev|def|theorem|macro|syntax)\s+` +
		`([A-Za-z_\x{00e0}-\x{ffff}][A-Za-z0-9_'!?\x{00e0}-\x{ffff}]*(?:\.[A-Za-z0-9_'!?\x{00e0}-\x{ffff}]+)*)`)

var namespaceHead = regexp.MustCompile(`^namespace\s+([A-Za-z_][A-Za-z0-9_'.]*)`)

// sectionHead matches the other `end`-closed openers. They contribute no name, but
// they must occupy a stack frame or their `end` would close the enclosing namespace
// and every later declaration would be indexed unqualified.
var sectionHead = regexp.MustCompile(`^(?:section|mutual)\b`)

var endHead = regexp.MustCompile(`^end(?:\s+([A-Za-z_][A-Za-z0-9_'.]*))?\s*$`)

// Build indexes every `.lean` file under modelRoot. Lake's build tree is skipped: it
// holds copies of the same sources plus dependency sources that are not this model's
// vocabulary.
func Build(modelRoot string) (*Index, error) {
	index := &Index{
		modules:      map[string]struct{}{},
		namespaces:   map[string]struct{}{},
		declarations: map[string]struct{}{},
	}
	root, err := filepath.Abs(modelRoot)
	if err != nil {
		return nil, err
	}
	if info, err := os.Lstat(root); err != nil {
		return nil, fmt.Errorf("stat %s: %w", modelRoot, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", modelRoot)
	}
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".lake" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".lean" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", relative, err)
		}
		index.addModule(moduleName(relative))
		index.addFile(string(content))
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return index, nil
}

// moduleName turns `Umpire/Model/Check.lean` into `Umpire.Model.Check`.
func moduleName(relativePath string) string {
	trimmed := strings.TrimSuffix(filepath.ToSlash(relativePath), ".lean")
	return strings.ReplaceAll(trimmed, "/", ".")
}

func (index *Index) addModule(module string) {
	index.modules[module] = struct{}{}
	// Every directory on the way to a module is a namespace an author may cite,
	// such as `Temporal.Feature` for `Temporal/Feature/Nexus/Success/Model.lean`.
	segments := strings.Split(module, ".")
	for count := 1; count < len(segments); count++ {
		index.namespaces[strings.Join(segments[:count], ".")] = struct{}{}
	}
}

func (index *Index) addFile(content string) {
	// Each frame is a namespace name, or "" for a `section` or `mutual` block that
	// contributes no name but is still closed by an `end`.
	var open []string
	qualified := func() []string {
		named := make([]string, 0, len(open))
		for _, frame := range open {
			if frame != "" {
				named = append(named, frame)
			}
		}
		return named
	}
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimLeft(line, " \t")
		switch {
		case namespaceHead.MatchString(trimmed):
			name := namespaceHead.FindStringSubmatch(trimmed)[1]
			index.namespaces[qualify(qualified(), name)] = struct{}{}
			open = append(open, name)
		case sectionHead.MatchString(trimmed):
			open = append(open, "")
		case endHead.MatchString(trimmed):
			// A bare `end` closes whichever construct is innermost; a named one closes
			// the namespace it names. Either way the stack shrinks by at most one.
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
		case declarationHead.MatchString(trimmed):
			// Only a declaration at the start of a line opens a name; an indented
			// `def` is a `let`-like body or a field default.
			if line != trimmed {
				continue
			}
			name := declarationHead.FindStringSubmatch(trimmed)[1]
			index.declarations[qualify(qualified(), name)] = struct{}{}
		default:
		}
	}
}

func qualify(open []string, name string) string {
	if len(open) == 0 {
		return name
	}
	return strings.Join(open, ".") + "." + name
}

// Resolve reports whether name is a module, a namespace, a declaration, or the
// trailing field or constructor segment of one. Lean does not spell fields and
// constructors on their own lines in a form worth parsing, and their parent already
// proves the citation points at real code.
func (index *Index) Resolve(name string) bool {
	if index.has(name) {
		return true
	}
	if cut := strings.LastIndex(name, "."); cut > 0 {
		// Only a declaration can own a field or constructor. Falling back to a
		// namespace or module would accept any invented segment under it.
		_, ok := index.declarations[name[:cut]]
		return ok
	}
	return false
}

func (index *Index) has(name string) bool {
	if _, ok := index.modules[name]; ok {
		return true
	}
	if _, ok := index.namespaces[name]; ok {
		return true
	}
	_, ok := index.declarations[name]
	return ok
}

// Size reports how many modules, namespaces and declarations the index holds. It
// exists so a caller can fail loudly on an empty or truncated walk instead of
// reporting that every name resolved.
func (index *Index) Size() (modules, namespaces, declarations int) {
	return len(index.modules), len(index.namespaces), len(index.declarations)
}

// Names returns every indexed name in sorted order, for diagnostics.
func (index *Index) Names() []string {
	all := make([]string, 0, len(index.modules)+len(index.namespaces)+len(index.declarations))
	for _, set := range []map[string]struct{}{index.modules, index.namespaces, index.declarations} {
		for name := range set {
			all = append(all, name)
		}
	}
	slices.Sort(all)
	return all
}
