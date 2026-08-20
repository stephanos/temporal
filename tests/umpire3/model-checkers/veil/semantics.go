package veil

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type generatedActionSemantics struct {
	guard   protocol.FirstOrderFormula
	updates []protocol.FirstOrderUpdate
}

type generatedSemantics struct {
	initial   protocol.FirstOrderFormula
	actions   map[string]generatedActionSemantics
	invariant protocol.FirstOrderFormula
	envelope  protocol.FirstOrderFormula
}

func validateGeneratedSemantics(view protocol.FirstOrderView, source []byte) error {
	names, err := buildNames(view)
	if err != nil {
		return err
	}
	parsed, err := parseGeneratedSemantics(view, names, string(source))
	if err != nil {
		return fmt.Errorf("parse generated Veil semantics: %w", err)
	}
	states, err := enumerateStates(view)
	if err != nil {
		return err
	}
	oracle := make(map[string]struct{}, len(view.Oracle.States))
	for _, state := range view.Oracle.States {
		values := make([]string, len(view.StateFields))
		for fieldIndex, field := range view.StateFields {
			for _, binding := range state.Fields {
				if binding.Field == field.Identifier {
					values[fieldIndex] = binding.Value
					break
				}
			}
		}
		oracle[stateKey(view.StateFields, values)] = struct{}{}
	}
	for _, state := range states {
		key := stateKey(view.StateFields, state.values)
		if evaluateFormula(view, state, parsed.initial) != evaluateFormula(view, state, view.Initial) {
			return fmt.Errorf("generated initial predicate differs at state %s", key)
		}
		if evaluateFormula(view, state, parsed.invariant) != evaluateFormula(view, state, view.Invariant) {
			return fmt.Errorf("generated safety predicate differs at state %s", key)
		}
		_, expectedEnvelope := oracle[key]
		if evaluateFormula(view, state, parsed.envelope) != expectedEnvelope {
			return fmt.Errorf("generated reachable envelope differs at state %s", key)
		}
		for _, action := range view.Actions {
			actual, found := parsed.actions[action.Identifier]
			if !found {
				return fmt.Errorf("generated semantics omit action %q", action.Identifier)
			}
			if evaluateFormula(view, state, actual.guard) != evaluateFormula(view, state, action.Guard) {
				return fmt.Errorf("generated action %q guard differs at state %s", action.Identifier, key)
			}
			expectedSuccessor := applyUpdates(view, state, action.Updates)
			actualSuccessor := applyUpdates(view, state, actual.updates)
			if !slices.Equal(actualSuccessor.values, expectedSuccessor.values) {
				return fmt.Errorf("generated action %q successor differs at state %s", action.Identifier, key)
			}
		}
	}
	return nil
}

func parseGeneratedSemantics(
	view protocol.FirstOrderView,
	names generatedNames,
	source string,
) (generatedSemantics, error) {
	lines := strings.Split(source, "\n")
	result := generatedSemantics{actions: make(map[string]generatedActionSemantics, len(view.Actions))}
	actions := make(map[string]string, len(names.actions))
	for identifier, name := range names.actions {
		actions[name] = identifier
	}
	foundInitial := false
	foundInvariant := false
	foundEnvelope := false
	for index := 0; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		switch {
		case line == "after_init {":
			if foundInitial {
				return generatedSemantics{}, errors.New("generated module has duplicate initializer")
			}
			block, next, err := generatedBlock(lines, index)
			if err != nil {
				return generatedSemantics{}, err
			}
			initial, err := parseGeneratedInitializer(view, names, block)
			if err != nil {
				return generatedSemantics{}, err
			}
			result.initial = initial
			foundInitial = true
			index = next
		case strings.HasPrefix(line, "action ") && strings.HasSuffix(line, " {"):
			name := strings.TrimSuffix(strings.TrimPrefix(line, "action "), " {")
			identifier, found := actions[name]
			if !found {
				return generatedSemantics{}, fmt.Errorf("generated module has unknown action %q", name)
			}
			if _, duplicate := result.actions[identifier]; duplicate {
				return generatedSemantics{}, fmt.Errorf("generated module has duplicate action %q", identifier)
			}
			block, next, err := generatedBlock(lines, index)
			if err != nil {
				return generatedSemantics{}, err
			}
			action, err := parseGeneratedAction(view, names, identifier, block)
			if err != nil {
				return generatedSemantics{}, err
			}
			result.actions[identifier] = action
			index = next
		case strings.HasPrefix(line, "safety ["):
			if foundInvariant {
				return generatedSemantics{}, errors.New("generated module has duplicate safety predicate")
			}
			formula, err := formulaAfterLabel(line, names)
			if err != nil {
				return generatedSemantics{}, err
			}
			result.invariant = formula
			foundInvariant = true
		case strings.HasPrefix(line, "invariant [CanonicalReachableEnvelope] "):
			if foundEnvelope {
				return generatedSemantics{}, errors.New("generated module has duplicate reachable envelope")
			}
			formula, err := parseGeneratedFormula(
				strings.TrimPrefix(line, "invariant [CanonicalReachableEnvelope] "), names)
			if err != nil {
				return generatedSemantics{}, err
			}
			result.envelope = formula
			foundEnvelope = true
		}
	}
	if !foundInitial || !foundInvariant || !foundEnvelope {
		return generatedSemantics{}, errors.New("generated module omits initializer, safety, or reachable envelope")
	}
	if len(result.actions) != len(view.Actions) {
		return generatedSemantics{}, fmt.Errorf("generated module has %d actions; view requires %d",
			len(result.actions), len(view.Actions))
	}
	return result, nil
}

func generatedBlock(lines []string, start int) ([]string, int, error) {
	for index := start + 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "}" {
			return lines[start+1 : index], index, nil
		}
	}
	return nil, 0, fmt.Errorf("generated block at line %d is not closed", start+1)
}

func parseGeneratedInitializer(
	view protocol.FirstOrderView,
	names generatedNames,
	block []string,
) (protocol.FirstOrderFormula, error) {
	wildcards := make(map[string]struct{}, len(view.StateFields))
	var initial *protocol.FirstOrderFormula
	for _, raw := range block {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "assume "):
			if initial != nil {
				return protocol.FirstOrderFormula{}, errors.New("generated initializer has duplicate assumption")
			}
			formula, err := parseGeneratedFormula(strings.TrimPrefix(line, "assume "), names)
			if err != nil {
				return protocol.FirstOrderFormula{}, err
			}
			initial = &formula
		case strings.HasSuffix(line, " := *"):
			fieldName := strings.TrimSuffix(line, " := *")
			field, found := generatedField(names, fieldName)
			if !found {
				return protocol.FirstOrderFormula{}, fmt.Errorf("generated initializer assigns unknown field %q", fieldName)
			}
			if _, duplicate := wildcards[field]; duplicate {
				return protocol.FirstOrderFormula{}, fmt.Errorf("generated initializer assigns field %q twice", field)
			}
			wildcards[field] = struct{}{}
		case line != "":
			return protocol.FirstOrderFormula{}, fmt.Errorf("unknown generated initializer statement %q", line)
		}
	}
	if initial == nil || len(wildcards) != len(view.StateFields) {
		return protocol.FirstOrderFormula{}, errors.New("generated initializer does not initialize every field exactly once")
	}
	return *initial, nil
}

func parseGeneratedAction(
	view protocol.FirstOrderView,
	names generatedNames,
	identifier string,
	block []string,
) (generatedActionSemantics, error) {
	var guard *protocol.FirstOrderFormula
	updates := make([]protocol.FirstOrderUpdate, 0)
	updated := make(map[string]struct{})
	for _, raw := range block {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "let "):
			parts := strings.Split(strings.TrimPrefix(line, "let "), " := ")
			if len(parts) != 2 {
				return generatedActionSemantics{}, fmt.Errorf("invalid generated snapshot %q", line)
			}
			field, found := generatedField(names, parts[1])
			if !found || names.preFields[field] != parts[0] {
				return generatedActionSemantics{}, fmt.Errorf("invalid generated snapshot %q", line)
			}
		case strings.HasPrefix(line, "require "):
			if guard != nil {
				return generatedActionSemantics{}, fmt.Errorf("generated action %q has duplicate guard", identifier)
			}
			formula, err := parseGeneratedFormula(strings.TrimPrefix(line, "require "), names)
			if err != nil {
				return generatedActionSemantics{}, err
			}
			guard = &formula
		case line == "pure ()":
			if len(updates) != 0 {
				return generatedActionSemantics{}, fmt.Errorf("generated action %q mixes updates and pure", identifier)
			}
		case strings.Contains(line, " := "):
			parts := strings.Split(line, " := ")
			if len(parts) != 2 {
				return generatedActionSemantics{}, fmt.Errorf("invalid generated update %q", line)
			}
			field, found := generatedField(names, parts[0])
			if !found {
				return generatedActionSemantics{}, fmt.Errorf("generated action %q updates unknown field %q",
					identifier, parts[0])
			}
			if _, duplicate := updated[field]; duplicate {
				return generatedActionSemantics{}, fmt.Errorf("generated action %q updates field %q twice",
					identifier, field)
			}
			term, err := parseGeneratedTerm(parts[1], names)
			if err != nil {
				return generatedActionSemantics{}, err
			}
			updates = append(updates, protocol.FirstOrderUpdate{Field: field, Value: term})
			updated[field] = struct{}{}
		case line != "":
			return generatedActionSemantics{}, fmt.Errorf("unknown generated action statement %q", line)
		}
	}
	if guard == nil {
		return generatedActionSemantics{}, fmt.Errorf("generated action %q omits guard", identifier)
	}
	if len(updates) == 0 {
		for _, line := range block {
			if strings.TrimSpace(line) == "pure ()" {
				return generatedActionSemantics{guard: *guard, updates: []protocol.FirstOrderUpdate{}}, nil
			}
		}
		return generatedActionSemantics{}, fmt.Errorf("generated action %q omits update or pure", identifier)
	}
	_ = view
	return generatedActionSemantics{guard: *guard, updates: updates}, nil
}

func formulaAfterLabel(line string, names generatedNames) (protocol.FirstOrderFormula, error) {
	close := strings.Index(line, "] ")
	if close < 0 {
		return protocol.FirstOrderFormula{}, fmt.Errorf("invalid generated safety declaration %q", line)
	}
	return parseGeneratedFormula(line[close+2:], names)
}

func generatedField(names generatedNames, generated string) (string, bool) {
	for field, name := range names.fields {
		if name == generated {
			return field, true
		}
	}
	return "", false
}

type generatedFormulaParser struct {
	tokens []string
	index  int
	names  generatedNames
}

func parseGeneratedFormula(value string, names generatedNames) (protocol.FirstOrderFormula, error) {
	parser := generatedFormulaParser{tokens: generatedFormulaTokens(value), names: names}
	formula, err := parser.formula()
	if err != nil {
		return protocol.FirstOrderFormula{}, err
	}
	if parser.index != len(parser.tokens) {
		return protocol.FirstOrderFormula{}, fmt.Errorf("unexpected generated formula token %q", parser.tokens[parser.index])
	}
	return formula, nil
}

func generatedFormulaTokens(value string) []string {
	tokens := make([]string, 0)
	for runes := []rune(value); len(runes) != 0; {
		if unicode.IsSpace(runes[0]) {
			runes = runes[1:]
			continue
		}
		switch runes[0] {
		case '(', ')', '=', '¬', '∧', '∨':
			tokens = append(tokens, string(runes[0]))
			runes = runes[1:]
			continue
		}
		end := 0
		for end < len(runes) && (unicode.IsLetter(runes[end]) || unicode.IsDigit(runes[end]) ||
			runes[end] == '_') {
			end++
		}
		if end == 0 {
			tokens = append(tokens, string(runes[0]))
			runes = runes[1:]
			continue
		}
		tokens = append(tokens, string(runes[:end]))
		runes = runes[end:]
	}
	return tokens
}

func (p *generatedFormulaParser) formula() (protocol.FirstOrderFormula, error) {
	if p.consume("True") {
		return protocol.FirstOrderFormula{Kind: protocol.FirstOrderFormulaTrue}, nil
	}
	if !p.consume("(") {
		return protocol.FirstOrderFormula{}, fmt.Errorf("expected generated formula, found %q", p.peek())
	}
	if p.consume("¬") {
		operand, err := p.formula()
		if err != nil {
			return protocol.FirstOrderFormula{}, err
		}
		if !p.consume(")") {
			return protocol.FirstOrderFormula{}, errors.New("generated negation is not closed")
		}
		return protocol.FirstOrderFormula{Kind: protocol.FirstOrderFormulaNot, Operand: &operand}, nil
	}
	if p.termAhead() {
		left, err := p.term()
		if err != nil {
			return protocol.FirstOrderFormula{}, err
		}
		if !p.consume("=") {
			return protocol.FirstOrderFormula{}, errors.New("generated equality omits operator")
		}
		right, err := p.term()
		if err != nil {
			return protocol.FirstOrderFormula{}, err
		}
		if !p.consume(")") {
			return protocol.FirstOrderFormula{}, errors.New("generated equality is not closed")
		}
		return protocol.FirstOrderFormula{Kind: protocol.FirstOrderFormulaEqual, Left: &left, Right: &right}, nil
	}
	first, err := p.formula()
	if err != nil {
		return protocol.FirstOrderFormula{}, err
	}
	if p.consume(")") {
		return first, nil
	}
	operator := p.peek()
	kind := protocol.FirstOrderFormulaAll
	if operator == "∨" {
		kind = protocol.FirstOrderFormulaAny
	} else if operator != "∧" {
		return protocol.FirstOrderFormula{}, fmt.Errorf("expected generated boolean operator, found %q", operator)
	}
	operands := []protocol.FirstOrderFormula{first}
	for p.consume(operator) {
		operand, err := p.formula()
		if err != nil {
			return protocol.FirstOrderFormula{}, err
		}
		operands = append(operands, operand)
	}
	if !p.consume(")") {
		return protocol.FirstOrderFormula{}, errors.New("generated boolean formula is not closed")
	}
	return protocol.FirstOrderFormula{Kind: kind, Operands: operands}, nil
}

func (p *generatedFormulaParser) termAhead() bool {
	return p.index+1 < len(p.tokens) && p.tokens[p.index+1] == "="
}

func (p *generatedFormulaParser) term() (protocol.FirstOrderTerm, error) {
	if p.index >= len(p.tokens) {
		return protocol.FirstOrderTerm{}, errors.New("generated term is missing")
	}
	value := p.tokens[p.index]
	p.index++
	return parseGeneratedTerm(value, p.names)
}

func parseGeneratedTerm(value string, names generatedNames) (protocol.FirstOrderTerm, error) {
	for field, name := range names.fields {
		if value == name || value == names.preFields[field] {
			return protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermField, Field: field}, nil
		}
	}
	for sort, values := range names.values {
		for canonical, name := range values {
			if value == name {
				return protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermValue,
					Sort: sort, Value: canonical}, nil
			}
		}
	}
	return protocol.FirstOrderTerm{}, fmt.Errorf("unknown generated term %q", value)
}

func (p *generatedFormulaParser) consume(value string) bool {
	if p.peek() != value {
		return false
	}
	p.index++
	return true
}

func (p *generatedFormulaParser) peek() string {
	if p.index >= len(p.tokens) {
		return "<end>"
	}
	return p.tokens[p.index]
}
