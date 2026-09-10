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
		`(?:class\s+inductive|structure|inductive|class|abbrev|def|theorem|macro|syntax)\s+` +
		`([A-Za-z_\x{00e0}-\x{ffff}][A-Za-z0-9_'!?\x{00e0}-\x{ffff}]*(?:\.[A-Za-z0-9_'!?\x{00e0}-\x{ffff}]+)*)`)

// memberHead splits a declaration head into its keyword and name, so the walker knows
// whether the indented block that follows holds fields or constructors.
var memberHead = regexp.MustCompile(
	`^(?:@\[[^\]]*\]\s*)*(?:(?:private|protected|partial|unsafe|noncomputable|scoped|local|nonrec)\s+)*` +
		`(?:class\s+)?(structure|class|inductive)\s`)

// structureField matches one field line inside a `structure` or `class` block. A field
// name is an ordinary identifier followed by a colon, which excludes `deriving`,
// `extends`, comments, and the continuation lines of a multi-line type.
var structureField = regexp.MustCompile(`^\s+([A-Za-z_][A-Za-z0-9_'!?]*)\s*:[^=]`)

// inductiveConstructor matches every `| name` on one line, so both the one-per-line and
// the packed spellings are indexed.
var inductiveConstructor = regexp.MustCompile(`\|\s*([A-Za-z_][A-Za-z0-9_'!?]*)`)

var stringLiteral = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

var lineComment = regexp.MustCompile(`--.*$`)

var namespaceHead = regexp.MustCompile(`^namespace\s+([A-Za-z_][A-Za-z0-9_'.]*)`)

// sectionHead matches the other `end`-closed openers. They contribute no name, but
// they must occupy a stack frame or their `end` would close the enclosing namespace
// and every later declaration would be indexed unqualified.
var sectionHead = regexp.MustCompile(
	`^(?:(?:private|protected|partial|unsafe|noncomputable|scoped|local|nonrec)\s+)*(?:section|mutual)\b`)

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
		// WalkDir never descends a symlink, and never reports one as a directory, so
		// skipping the entry is the whole of it.
		if entry.Type()&os.ModeSymlink != 0 {
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

// commentDelta reports how many block comments one line opens minus how many it closes.
// String literals and `--` comments are removed first: a description holding `+/-10%`
// would otherwise open a comment that never closes, and every later declaration in the
// file would vanish from the index.
func commentDelta(line string) int {
	stripped := lineComment.ReplaceAllString(stringLiteral.ReplaceAllString(line, `""`), "")
	return strings.Count(stripped, "/-") - strings.Count(stripped, "-/")
}

// memberBlock is the indented body of a `structure`, `class` or `inductive`, whose
// lines name the owner's fields or constructors. The block ends at the first line that
// starts in column zero.
type memberBlock struct {
	owner   string
	keyword string
}

// consume indexes one line of the block and reports whether the block is still open.
func (block memberBlock) consume(index *Index, line, trimmed string) bool {
	if trimmed != "" && line == trimmed {
		return false
	}
	if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/-") {
		return true
	}
	if block.keyword == "inductive" {
		for _, match := range inductiveConstructor.FindAllStringSubmatch(trimmed, -1) {
			index.declarations[block.owner+"."+match[1]] = struct{}{}
		}
		return true
	}
	if match := structureField.FindStringSubmatch(line); match != nil {
		index.declarations[block.owner+"."+match[1]] = struct{}{}
	}
	return true
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
	var members memberBlock
	// Lean block comments nest, and a docstring is one. Lines inside a comment name
	// nothing, so a fenced example holding column-zero Lean cannot add a phantom name or
	// close a namespace the code did not close.
	commentDepth := 0
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimLeft(line, " \t")
		inComment := commentDepth > 0
		commentDepth += commentDelta(line)
		if commentDepth < 0 {
			commentDepth = 0
		}
		if inComment {
			continue
		}
		if members.owner != "" && !members.consume(index, line, trimmed) {
			members = memberBlock{}
		}
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
			qualifiedName := qualify(qualified(), name)
			index.declarations[qualifiedName] = struct{}{}
			members = memberBlock{}
			if match := memberHead.FindStringSubmatch(trimmed); match != nil {
				members = memberBlock{owner: qualifiedName, keyword: match[1]}
			}
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

// Resolve reports whether name is a module, a namespace, or a declaration. Fields and
// constructors are indexed as declarations under their owner, so an invented segment
// under a real structure does not resolve just because its parent does.
func (index *Index) Resolve(name string) bool {
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
