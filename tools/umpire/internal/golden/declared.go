package golden

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Since fn-124.3 a Case lowered from a realization that declares its API behavior carries what that
// behavior says of the system under test (.plans/API_BEHAVIOR_HINTS.md, tools/umpire/lower/realization.go):
// how it numbers an activity's attempts, the limits an instruction that writes none runs under, and
// that the order of a run is causal. A baseline has no behavior, so no baseline Case carries them. The
// delta lists each such Case member (Delta.DeclaredMembers); both harnesses compare a current Case
// without them, and refuse a baseline Case that carries one. Every other byte of the Case, the whole
// Contract included, is compared, and a Case identity derived from the bytes is derived from the bytes
// so compared.

// Declared are the Case members a realization's API behavior declares, each a path of JSON members
// from the Case: "program.instructionDefaults", or, through each element of an array,
// "program.entrypoints[*].activity.attemptNumbering".
type Declared []string

// Declared gives the Case members the delta lists as declared by a realization's API behavior.
func (d Delta) Declared() Declared { return d.DeclaredMembers }

// declaredMember is a path of lowerCamel JSON members under the Case's Program, any of them but the
// last read through each element of the array it is ("[*]").
var declaredMember = regexp.MustCompile(`^program(\.[a-z][A-Za-z0-9]*(\[\*\])?)*\.[a-z][A-Za-z0-9]*$`)

// checkDeclared checks each declared member: a member of the Program, listed once. A Contract member
// is never one: no Contract reads the API behavior.
func (d Delta) checkDeclared() error {
	seen := map[string]bool{}
	for _, path := range d.DeclaredMembers {
		switch {
		case !declaredMember.MatchString(path):
			return fmt.Errorf("declared Case member %q is no member of the Program", path)
		case seen[path]:
			return fmt.Errorf("declared Case member %q is listed twice", path)
		}
		seen[path] = true
	}
	return nil
}

// segment is one member of a declared member's path: its key, and whether it is read through each
// element of the array it is.
type segment struct {
	key  string
	each bool
}

func segments(path string) []segment {
	var out []segment
	for key := range strings.SplitSeq(path, ".") {
		key, each := strings.CutSuffix(key, "[*]")
		out = append(out, segment{key: key, each: each})
	}
	return out
}

// Carried gives the declared members a compact Case carries, each once, in the order listed.
func (ds Declared) Carried(encoded []byte) ([]string, error) {
	var out []string
	_, err := ds.edit(encoded, false, func(path string) error {
		if !slices.Contains(out, path) {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

// Current gives the bytes of a current Case as both harnesses compare them: without each declared
// member. Every other byte is kept. The Case must be compact JSON, as the lowering writes it, so that
// it is rewritten member by member; with no members listed it is kept as it is.
func (ds Declared) Current(encoded []byte) ([]byte, error) {
	return ds.edit(encoded, false, nil)
}

// CurrentProgram gives the bytes of a current Case's Program as Current gives them within the Case.
func (ds Declared) CurrentProgram(encoded []byte) ([]byte, error) {
	return ds.edit(encoded, true, nil)
}

// Baseline gives the bytes of a baseline Case as both harnesses compare them: as they are, since a
// baseline has no API behavior to declare a member from. One that carries a declared member is
// refused.
func (ds Declared) Baseline(encoded []byte) ([]byte, error) {
	return ds.edit(encoded, false, refuseDeclared)
}

// BaselineProgram gives the bytes of a baseline Case's Program as Baseline gives them within the Case.
func (ds Declared) BaselineProgram(encoded []byte) ([]byte, error) {
	return ds.edit(encoded, true, refuseDeclared)
}

func refuseDeclared(path string) error {
	return fmt.Errorf("the baseline Case carries the declared member %s, which only an API behavior declares", path)
}

// edit gives a Case, or its Program alone, without each declared member it carries, calling found, when
// given, with each it carries; an error of found refuses the Case. With found the bytes are kept as
// they are.
func (ds Declared) edit(encoded []byte, program bool, found func(path string) error) ([]byte, error) {
	if len(ds) == 0 {
		return encoded, nil
	}
	return trailing(encoded, func(c []byte) ([]byte, error) {
		edited := c
		for _, path := range ds {
			at := segments(path)
			if program {
				at = at[1:]
			}
			var err error
			if edited, err = strip(edited, at, func() error {
				if found == nil {
					return nil
				}
				return found(path)
			}); err != nil {
				return nil, err
			}
		}
		if found != nil {
			return c, nil
		}
		return edited, nil
	})
}

// strip gives a compact JSON object without the member the path names, calling found where it has it.
// An object without a member on the path is kept as it is.
func strip(encoded []byte, path []segment, found func() error) ([]byte, error) {
	return editObject(encoded, func(members []member) ([]member, error) {
		s := path[0]
		if len(path) == 1 {
			if !slices.ContainsFunc(members, func(m member) bool { return m.key == s.key }) {
				return members, nil
			}
			if err := found(); err != nil {
				return nil, err
			}
			return without(members, s.key), nil
		}
		return editMember(members, s.key, func(value []byte) ([]byte, error) {
			next := func(value []byte) ([]byte, error) { return strip(value, path[1:], found) }
			if s.each {
				return editArray(value, next)
			}
			return next(value)
		})
	})
}
