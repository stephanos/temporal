package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type nameRequest struct {
	identity  string
	base      string
	number    int32
	hasNumber bool
}

func allocateScopedLeanNames(
	requestsByScope map[string][]nameRequest,
	reservedByScope map[string][]string,
) (map[string]string, error) {
	localNames := make(map[string]string)
	scopes := make([]string, 0, len(requestsByScope))
	for scope := range requestsByScope {
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	for _, scope := range scopes {
		allocated, err := allocateNames(requestsByScope[scope], reservedByScope[scope])
		if err != nil {
			return nil, fmt.Errorf("build Lean declaration scope %q: %w", scope, err)
		}
		for identity, name := range allocated {
			localNames[identity] = name
		}
	}
	return localNames, nil
}

func allocateNames(requests []nameRequest, reserved []string) (map[string]string, error) {
	allocator, err := newNameAllocator(requests, reserved)
	if err != nil {
		return nil, err
	}
	unresolved := allocator.allocateUniqueBases()
	remaining := allocator.allocateUniqueNumbers(unresolved)
	if err := allocator.allocateDigests(remaining); err != nil {
		return nil, err
	}
	return allocator.result, nil
}

type nameAllocator struct {
	result   map[string]string
	groups   map[string][]nameRequest
	original map[string]bool
	used     map[string]bool
}

func newNameAllocator(requests []nameRequest, reserved []string) (nameAllocator, error) {
	result := nameAllocator{
		result:   make(map[string]string, len(requests)),
		groups:   make(map[string][]nameRequest),
		original: make(map[string]bool, len(requests)+len(reserved)),
		used:     make(map[string]bool, len(requests)+len(reserved)),
	}
	identities := make(map[string]bool, len(requests))
	for _, name := range reserved {
		result.original[name] = true
		result.used[name] = true
	}
	for _, request := range requests {
		if request.identity == "" || request.base == "" {
			return nameAllocator{}, errors.New("name identity and base are required")
		}
		if identities[request.identity] {
			return nameAllocator{}, fmt.Errorf("duplicate name identity %q", request.identity)
		}
		identities[request.identity] = true
		result.groups[request.base] = append(result.groups[request.base], request)
		result.original[request.base] = true
	}
	return result, nil
}

func (a *nameAllocator) allocateUniqueBases() []nameRequest {
	var unresolved []nameRequest
	bases := make([]string, 0, len(a.groups))
	for base := range a.groups {
		bases = append(bases, base)
	}
	slices.Sort(bases)
	for _, base := range bases {
		group := a.groups[base]
		slices.SortFunc(group, func(left, right nameRequest) int {
			return strings.Compare(left.identity, right.identity)
		})
		if len(group) == 1 && !a.used[base] {
			a.result[group[0].identity] = base
			a.used[base] = true
			continue
		}
		unresolved = append(unresolved, group...)
	}
	return unresolved
}

func (a *nameAllocator) allocateUniqueNumbers(unresolved []nameRequest) []nameRequest {
	proposals := make(map[string][]nameRequest)
	for _, request := range unresolved {
		if !request.hasNumber {
			continue
		}
		candidate := request.base + leanNumberSuffix(request.number)
		if !a.original[candidate] && !a.used[candidate] {
			proposals[candidate] = append(proposals[candidate], request)
		}
	}
	var remaining []nameRequest
	for _, request := range unresolved {
		candidate := ""
		if request.hasNumber {
			candidate = request.base + leanNumberSuffix(request.number)
		}
		if candidate != "" && len(proposals[candidate]) == 1 {
			a.result[request.identity] = candidate
			a.used[candidate] = true
			continue
		}
		remaining = append(remaining, request)
	}
	return remaining
}

func (a *nameAllocator) allocateDigests(remaining []nameRequest) error {
	slices.SortFunc(remaining, func(left, right nameRequest) int {
		return strings.Compare(left.identity, right.identity)
	})
	for _, request := range remaining {
		digest := sha256.Sum256([]byte(request.identity))
		hexDigest := hex.EncodeToString(digest[:])
		allocated := false
		for length := 8; length <= len(hexDigest); length += 2 {
			candidate := request.base + "_" + hexDigest[:length]
			if !a.original[candidate] && !a.used[candidate] {
				a.result[request.identity] = candidate
				a.used[candidate] = true
				allocated = true
				break
			}
		}
		if !allocated {
			return fmt.Errorf("cannot disambiguate name %q", request.identity)
		}
	}
	return nil
}

func leanNumberSuffix(value int32) string {
	if value < 0 {
		return fmt.Sprintf("Neg%d", -int64(value))
	}
	return fmt.Sprint(value)
}

func upperIdentifier(value string) string {
	parts := identifierParts(value)
	var result strings.Builder
	for _, part := range parts {
		if part == strings.ToUpper(part) {
			part = strings.ToLower(part)
		}
		characters := []rune(part)
		if len(characters) == 0 {
			continue
		}
		result.WriteRune(unicode.ToUpper(characters[0]))
		result.WriteString(string(characters[1:]))
	}
	if result.Len() == 0 {
		return "Unnamed"
	}
	return result.String()
}

func lowerIdentifier(value string) string {
	name := upperIdentifier(value)
	characters := []rune(name)
	characters[0] = unicode.ToLower(characters[0])
	name = string(characters)
	if leanReserved[name] {
		return name + "Value"
	}
	return name
}

func identifierParts(value string) []string {
	return strings.FieldsFunc(value, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}

var leanReserved = map[string]bool{
	"abbrev": true, "attribute": true, "axiom": true, "by": true, "class": true, "def": true,
	"deriving": true, "do": true, "elab": true, "else": true, "end": true, "example": true,
	"export": true, "extends": true, "for": true, "fun": true, "if": true, "import": true,
	"in": true, "include": true, "inductive": true, "infix": true, "infixl": true, "infixr": true,
	"instance": true, "let": true, "macro": true, "match": true, "meta": true, "mutual": true,
	"namespace": true, "omit": true, "opaque": true, "open": true, "partial": true, "postfix": true,
	"prefix": true, "private": true, "protected": true, "scoped": true, "structure": true,
	"syntax": true, "theorem": true, "universe": true, "variable": true, "where": true, "with": true,
}
