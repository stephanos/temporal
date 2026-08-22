package catalog

import (
	"errors"
	"fmt"
	"slices"
)

// ResolvedDeclaration is evidence captured by Lean while elaborating a named declaration.
type ResolvedDeclaration struct {
	Declaration string     `json:"declaration"`
	TypeHash    string     `json:"typeHash"`
	Type        string     `json:"type"`
	Axioms      []string   `json:"axioms"`
	TrustBadge  TrustBadge `json:"trustBadge"`
}

func (d *ResolvedDeclaration) derive() {
	d.Derive()
}

func (d *ResolvedDeclaration) Derive() {
	if d.TypeHash == "derived" {
		d.TypeHash = statementDigest(d.Type)
	}
	slices.Sort(d.Axioms)
}

func (d ResolvedDeclaration) Validate() error {
	if d.Declaration == "" || d.Type == "" {
		return errors.New("resolved declaration and type are required")
	}
	if d.TypeHash != statementDigest(d.Type) {
		return errors.New("resolved declaration type hash does not match its elaborated type")
	}
	if !slices.IsSorted(d.Axioms) || len(slices.Compact(append([]string(nil), d.Axioms...))) != len(d.Axioms) {
		return errors.New("resolved declaration axioms must be sorted and unique")
	}
	for _, axiom := range d.Axioms {
		if axiom == "" || axiom == "sorryAx" || axiom == "Lean.ofReduceBool" {
			return fmt.Errorf("resolved declaration has invalid axiom %q", axiom)
		}
	}
	if d.TrustBadge != trustBadgeForAxioms(d.Axioms) {
		return errors.New("resolved declaration trust badge does not match its axiom inventory")
	}
	return nil
}

func trustBadgeForAxioms(axioms []string) TrustBadge {
	if len(axioms) == 0 {
		return TrustBadgeKernel
	}
	return TrustBadgeKernelWithDeclaredAxioms
}

func aggregateTrustBadge(declarations ...ResolvedDeclaration) TrustBadge {
	for _, declaration := range declarations {
		if len(declaration.Axioms) != 0 {
			return TrustBadgeKernelWithDeclaredAxioms
		}
	}
	return TrustBadgeKernel
}
