package veil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type Mode string

const (
	Interactive Mode = "interactive"
	Concrete    Mode = "concrete"
	Mutation    Mode = "mutation"
)

type SMTTrustMode string

const (
	ReconstructedSMT SMTTrustMode = "reconstructed"
	TrustedSMT       SMTTrustMode = "trusted"
	SMTNotUsed       SMTTrustMode = "not-used"
)

type GeneratedModule struct {
	Module              string
	Source              []byte
	ModelHash           string
	ActionLabels        map[string]string
	ExportsModelChecker bool
	TrustMode           SMTTrustMode
}

func Generate(view protocol.FirstOrderView, mode Mode) (GeneratedModule, error) {
	return GenerateWithTrust(view, mode, ReconstructedSMT)
}

func GenerateWithTrust(
	view protocol.FirstOrderView,
	mode Mode,
	trustMode SMTTrustMode,
) (GeneratedModule, error) {
	if err := view.Validate(); err != nil {
		return GeneratedModule{}, fmt.Errorf("validate first-order view: %w", err)
	}
	if err := CompareReachableStates(view); err != nil {
		return GeneratedModule{}, err
	}
	if mode != Interactive && mode != Concrete && mode != Mutation {
		return GeneratedModule{}, fmt.Errorf("unknown Veil generation mode %q", mode)
	}
	if trustMode != ReconstructedSMT && trustMode != TrustedSMT {
		return GeneratedModule{}, fmt.Errorf("unknown Veil SMT trust mode %q", trustMode)
	}
	names, err := buildNames(view)
	if err != nil {
		return GeneratedModule{}, err
	}
	module := exportedIdentifier(string(view.Target)) + exportedIdentifier(view.Variant)
	if mode == Concrete {
		module += "Concrete"
	}

	var source bytes.Buffer
	source.WriteString("import Veil\n")
	if mode == Interactive {
		source.WriteString("import Umpire3Veil.JobReceipt\n")
	}
	source.WriteString("\n")
	if mode == Concrete {
		source.WriteString("set_option veil.__modelCheckCompileMode true\n\n")
	} else {
		source.WriteString("set_option veil.solver \"grind+smt\"\n")
		fmt.Fprintf(&source, "set_option veil.smt.trust %t\n\n", trustMode == TrustedSMT)
	}
	fmt.Fprintf(&source, "veil module %s\n\n", module)
	for _, sort := range view.Sorts {
		sortName := names.sorts[sort.Identifier]
		if sort.Kind == protocol.FirstOrderSortEnum {
			values := make([]string, len(sort.Values))
			for index, value := range sort.Values {
				values[index] = names.values[sort.Identifier][value]
			}
			fmt.Fprintf(&source, "enum %s = {%s}\n", sortName, strings.Join(values, ", "))
		} else {
			fmt.Fprintf(&source, "type %s\n", sortName)
		}
	}
	source.WriteString("\n")
	for _, field := range view.StateFields {
		fmt.Fprintf(&source, "individual %s : %s\n", names.fields[field.Identifier], names.sorts[field.Sort])
	}
	source.WriteString("\n#gen_state\n\nafter_init {\n")
	for _, field := range view.StateFields {
		fmt.Fprintf(&source, "  %s := *\n", names.fields[field.Identifier])
	}
	initial, err := renderFormula(view.Initial, names, false)
	if err != nil {
		return GeneratedModule{}, err
	}
	fmt.Fprintf(&source, "  assume %s\n", initial)
	source.WriteString("}\n\n")
	for _, action := range view.Actions {
		fmt.Fprintf(&source, "action %s {\n", names.actions[action.Identifier])
		snapshots := actionSnapshots(action)
		for _, field := range view.StateFields {
			if snapshots[field.Identifier] {
				fmt.Fprintf(&source, "  let %s := %s\n", names.preFields[field.Identifier], names.fields[field.Identifier])
			}
		}
		guard, err := renderFormula(action.Guard, names, false)
		if err != nil {
			return GeneratedModule{}, err
		}
		fmt.Fprintf(&source, "  require %s\n", guard)
		for _, update := range action.Updates {
			value, err := renderTerm(update.Value, names, true)
			if err != nil {
				return GeneratedModule{}, err
			}
			fmt.Fprintf(&source, "  %s := %s\n", names.fields[update.Field], value)
		}
		if len(action.Updates) == 0 {
			source.WriteString("  pure ()\n")
		}
		source.WriteString("}\n\n")
	}
	invariant, err := renderFormula(view.Invariant, names, false)
	if err != nil {
		return GeneratedModule{}, err
	}
	fmt.Fprintf(&source, "safety [%s] %s\n\n", exportedIdentifier(string(view.Property)), invariant)
	fmt.Fprintf(&source, "invariant [CanonicalReachableEnvelope] %s\n\n",
		renderOracleEnvelope(view, names))
	source.WriteString("#gen_spec\n\n")
	instantiation := renderInstantiation(view, names)
	switch mode {
	case Concrete:
		fmt.Fprintf(&source, "#model_check %s { } (sequential := true)\n", instantiation)
	case Interactive:
		fmt.Fprintf(&source, "#model_check interpreted %s { } (sequential := true)\n\n", instantiation)
		fmt.Fprintf(&source, "unsat trace [bounded_safety] {\n  any %d actions\n  assert ¬ (%s)\n}\n\n",
			view.Bounds.SymbolicDepth, invariant)
		source.WriteString("#check_invariants\n\n")
		source.WriteString("#gen_theorems\n\n")
		fmt.Fprintf(&source, "end %s\n", module)
	case Mutation:
		source.WriteString("set_option veil.violationIsError false in\n")
		fmt.Fprintf(&source, "#model_check interpreted %s { } (sequential := true)\n\n", instantiation)
		fmt.Fprintf(&source, "sat trace [counterexample] {\n  any %d actions\n  assert ¬ (%s)\n}\n\n",
			view.Bounds.SymbolicDepth, invariant)
		fmt.Fprintf(&source, "end %s\n", module)
	}
	modelHash := sourceDigest(source.Bytes())
	if mode == Interactive {
		fmt.Fprintf(&source, "\nnamespace Umpire3Veil.Generated\n\ndef %sEvidence : Umpire3Veil.JobReceipt.Evidence where\n", module)
		fmt.Fprintf(&source, "  semanticHash := %q\n", view.SemanticHash)
		fmt.Fprintf(&source, "  generatedModelHash := %q\n", modelHash)
		fmt.Fprintf(&source, "  trustMode := .%s\n", trustMode)
		source.WriteString("  invariantAxioms := resolved_veil_axioms% [\n")
		theorems := proofTheoremNames(module, names, view)
		for index, theorem := range theorems {
			separator := ","
			if index == len(theorems)-1 {
				separator = ""
			}
			fmt.Fprintf(&source, "    %s%s\n", theorem, separator)
		}
		source.WriteString("  ]\n\nend Umpire3Veil.Generated\n")
	}
	actionLabels := make(map[string]string, len(names.actions))
	for identifier, label := range names.actions {
		actionLabels[identifier] = label
	}
	return GeneratedModule{
		Module:              module,
		Source:              source.Bytes(),
		ModelHash:           modelHash,
		ActionLabels:        actionLabels,
		ExportsModelChecker: mode == Concrete,
		TrustMode:           trustModeForMode(mode, trustMode),
	}, nil
}

func proofTheoremNames(
	module string,
	names generatedNames,
	view protocol.FirstOrderView,
) []string {
	procedures := []string{"initializer"}
	for _, action := range view.Actions {
		procedures = append(procedures, names.actions[action.Identifier])
	}
	assertions := []string{exportedIdentifier(string(view.Property)), "CanonicalReachableEnvelope"}
	result := make([]string, 0, len(procedures)*(len(assertions)+1))
	for _, procedure := range procedures {
		result = append(result, module+"."+procedure+"_doesNotThrow")
		for _, assertion := range assertions {
			result = append(result, module+"."+procedure+"_"+assertion)
		}
	}
	return result
}

func sourceDigest(source []byte) string {
	sum := sha256.Sum256(source)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func actionSnapshots(action protocol.FirstOrderAction) map[string]bool {
	result := make(map[string]bool)
	for _, update := range action.Updates {
		if update.Value.Kind == protocol.FirstOrderTermField {
			result[update.Value.Field] = true
		}
	}
	return result
}

func renderOracleEnvelope(view protocol.FirstOrderView, names generatedNames) string {
	states := make([]string, len(view.Oracle.States))
	for stateIndex, state := range view.Oracle.States {
		bindings := make(map[string]string, len(state.Fields))
		for _, binding := range state.Fields {
			bindings[binding.Field] = binding.Value
		}
		fields := make([]string, len(view.StateFields))
		for fieldIndex, field := range view.StateFields {
			fields[fieldIndex] = fmt.Sprintf("(%s = %s)", names.fields[field.Identifier],
				names.values[field.Sort][bindings[field.Identifier]])
		}
		states[stateIndex] = "(" + strings.Join(fields, " ∧ ") + ")"
	}
	return "(" + strings.Join(states, " ∨ ") + ")"
}

func trustModeForMode(mode Mode, trustMode SMTTrustMode) SMTTrustMode {
	if mode == Concrete {
		return SMTNotUsed
	}
	return trustMode
}

type generatedNames struct {
	sorts     map[string]string
	values    map[string]map[string]string
	fields    map[string]string
	preFields map[string]string
	actions   map[string]string
}

func buildNames(view protocol.FirstOrderView) (generatedNames, error) {
	names := generatedNames{
		sorts:     make(map[string]string, len(view.Sorts)),
		values:    make(map[string]map[string]string, len(view.Sorts)),
		fields:    make(map[string]string, len(view.StateFields)),
		preFields: make(map[string]string, len(view.StateFields)),
		actions:   make(map[string]string, len(view.Actions)),
	}
	used := make(map[string]string)
	register := func(kind, source, generated string) error {
		if previous, duplicate := used[generated]; duplicate {
			return fmt.Errorf("Veil identifiers %q and %q both generate %q", previous, kind+" "+source, generated)
		}
		used[generated] = kind + " " + source
		return nil
	}
	for _, sort := range view.Sorts {
		generated := exportedIdentifier(sort.Identifier)
		if sort.Kind == protocol.FirstOrderSortUninterpreted {
			generated = localIdentifier(sort.Identifier)
		}
		if err := register("sort", sort.Identifier, generated); err != nil {
			return generatedNames{}, err
		}
		names.sorts[sort.Identifier] = generated
		names.values[sort.Identifier] = make(map[string]string, len(sort.Values))
		for _, value := range sort.Values {
			generated := exportedIdentifier(value)
			if err := register("sort value", sort.Identifier+"/"+value, generated); err != nil {
				return generatedNames{}, err
			}
			names.values[sort.Identifier][value] = generated
		}
	}
	for _, field := range view.StateFields {
		generated := localIdentifier(field.Identifier)
		if err := register("field", field.Identifier, generated); err != nil {
			return generatedNames{}, err
		}
		names.fields[field.Identifier] = generated
		pre := "pre" + exportedIdentifier(field.Identifier)
		if err := register("field snapshot", field.Identifier, pre); err != nil {
			return generatedNames{}, err
		}
		names.preFields[field.Identifier] = pre
	}
	for _, action := range view.Actions {
		generated := exportedIdentifier(action.Identifier)
		if err := register("action", action.Identifier, generated); err != nil {
			return generatedNames{}, err
		}
		names.actions[action.Identifier] = generated
	}
	return names, nil
}

func renderInstantiation(view protocol.FirstOrderView, names generatedNames) string {
	values := make([]string, 0)
	for _, sort := range view.Sorts {
		if sort.Kind == protocol.FirstOrderSortUninterpreted {
			values = append(values, fmt.Sprintf("%s := Fin %d", names.sorts[sort.Identifier], sort.Cardinality))
		}
	}
	return "{ " + strings.Join(values, ", ") + " }"
}

func renderFormula(formula protocol.FirstOrderFormula, names generatedNames, snapshots bool) (string, error) {
	switch formula.Kind {
	case protocol.FirstOrderFormulaTrue:
		return "True", nil
	case protocol.FirstOrderFormulaEqual:
		left, err := renderTerm(*formula.Left, names, snapshots)
		if err != nil {
			return "", err
		}
		right, err := renderTerm(*formula.Right, names, snapshots)
		if err != nil {
			return "", err
		}
		return "(" + left + " = " + right + ")", nil
	case protocol.FirstOrderFormulaNot:
		operand, err := renderFormula(*formula.Operand, names, snapshots)
		if err != nil {
			return "", err
		}
		return "(¬ " + operand + ")", nil
	case protocol.FirstOrderFormulaAll, protocol.FirstOrderFormulaAny:
		operator := " ∧ "
		if formula.Kind == protocol.FirstOrderFormulaAny {
			operator = " ∨ "
		}
		operands := make([]string, len(formula.Operands))
		for index, operand := range formula.Operands {
			rendered, err := renderFormula(operand, names, snapshots)
			if err != nil {
				return "", err
			}
			operands[index] = rendered
		}
		return "(" + strings.Join(operands, operator) + ")", nil
	default:
		return "", fmt.Errorf("cannot render first-order formula kind %q", formula.Kind)
	}
}

func renderTerm(term protocol.FirstOrderTerm, names generatedNames, snapshots bool) (string, error) {
	switch term.Kind {
	case protocol.FirstOrderTermField:
		if snapshots {
			return names.preFields[term.Field], nil
		}
		return names.fields[term.Field], nil
	case protocol.FirstOrderTermValue:
		value, found := names.values[term.Sort][term.Value]
		if !found {
			return "", fmt.Errorf("cannot render first-order value %q/%q", term.Sort, term.Value)
		}
		return value, nil
	default:
		return "", fmt.Errorf("cannot render first-order term kind %q", term.Kind)
	}
}

func exportedIdentifier(value string) string {
	parts := identifierParts(value)
	var result strings.Builder
	for _, part := range parts {
		runes := []rune(part)
		result.WriteRune(unicode.ToUpper(runes[0]))
		result.WriteString(string(runes[1:]))
	}
	if result.Len() == 0 {
		return "UmpireIdentifier"
	}
	if first := []rune(result.String())[0]; unicode.IsDigit(first) {
		return "Umpire" + result.String()
	}
	return result.String()
}

func localIdentifier(value string) string {
	exported := exportedIdentifier(value)
	if exported == "" {
		return "umpireIdentifier"
	}
	runes := []rune(exported)
	runes[0] = unicode.ToLower(runes[0])
	result := string(runes)
	switch result {
	case "action", "after", "assume", "end", "enum", "if", "invariant", "let", "require", "safety", "type":
		return "umpire" + exported
	default:
		return result
	}
}

func identifierParts(value string) []string {
	return strings.FieldsFunc(value, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}
