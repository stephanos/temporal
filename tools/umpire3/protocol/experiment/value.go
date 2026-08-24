package experiment

import (
	"errors"
	"fmt"
)

type ValueType string

const (
	ValueString      ValueType = "string"
	ValueInteger     ValueType = "integer"
	ValueBoolean     ValueType = "boolean"
	ValueDuration    ValueType = "duration"
	ValueEnum        ValueType = "enum"
	ValueBytesDigest ValueType = "bytes-digest"
	ValueSymbol      ValueType = "symbol"
	ValueList        ValueType = "list"
	ValueRecord      ValueType = "record"
)

type Value struct {
	Type     ValueType    `json:"type"`
	Text     *string      `json:"text,omitempty"`
	Integer  *int64       `json:"integer,omitempty"`
	Boolean  *bool        `json:"boolean,omitempty"`
	Elements []Value      `json:"elements,omitempty"`
	Fields   []NamedValue `json:"fields,omitempty"`
}

type NamedValue struct {
	Name  string `json:"name"`
	Value Value  `json:"value"`
}

type Binding struct {
	Symbol     string `json:"symbol"`
	Type       string `json:"type"`
	Projection string `json:"projection"`
}

func (v Value) semanticType() string {
	if v.Type == ValueSymbol {
		return "identity"
	}
	return string(v.Type)
}

func (v Value) validate(depth int) error {
	if depth > 32 {
		return errors.New("semantic value nesting exceeds 32 levels")
	}
	scalarFields := func(text, integer, boolean bool) bool {
		return (v.Text != nil) == text && (v.Integer != nil) == integer &&
			(v.Boolean != nil) == boolean && len(v.Elements) == 0 && len(v.Fields) == 0
	}
	switch v.Type {
	case ValueString:
		if !scalarFields(true, false, false) {
			return errors.New("string value requires only text")
		}
	case ValueInteger:
		if !scalarFields(false, true, false) {
			return errors.New("integer value requires only integer")
		}
	case ValueBoolean:
		if !scalarFields(false, false, true) {
			return errors.New("boolean value requires only boolean")
		}
	case ValueDuration:
		if !scalarFields(false, true, false) {
			return errors.New("duration value requires only integer nanoseconds")
		}
	case ValueEnum:
		if !scalarFields(true, true, false) {
			return errors.New("enum value requires only text and integer number")
		}
	case ValueBytesDigest:
		if !scalarFields(true, false, false) || !validHash(*v.Text) {
			return errors.New("bytes-digest value requires only a sha256 text digest")
		}
	case ValueSymbol:
		if !scalarFields(true, false, false) || *v.Text == "" {
			return errors.New("symbol value requires only non-empty text")
		}
	case ValueList:
		if v.Text != nil || v.Integer != nil || v.Boolean != nil || len(v.Fields) != 0 {
			return errors.New("list value requires only elements")
		}
		for index, element := range v.Elements {
			if err := element.validate(depth + 1); err != nil {
				return fmt.Errorf("list element %d: %w", index, err)
			}
		}
	case ValueRecord:
		if v.Text != nil || v.Integer != nil || v.Boolean != nil || len(v.Elements) != 0 {
			return errors.New("record value requires only fields")
		}
		if err := validateNamedValues(v.Fields, depth+1); err != nil {
			return fmt.Errorf("record: %w", err)
		}
	default:
		return fmt.Errorf("unknown semantic value type %q", v.Type)
	}
	return nil
}

func validateNamedValues(values []NamedValue, depth int) error {
	names := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.Name == "" {
			return errors.New("typed value name is required")
		}
		if _, duplicate := names[value.Name]; duplicate {
			return fmt.Errorf("duplicate typed value %q", value.Name)
		}
		names[value.Name] = struct{}{}
		if sensitiveField(value.Name) {
			return fmt.Errorf("typed value %q is sensitive", value.Name)
		}
		if err := value.Value.validate(depth); err != nil {
			return fmt.Errorf("typed value %q: %w", value.Name, err)
		}
	}
	return nil
}

func referencedSymbols(values []NamedValue) []string {
	var result []string
	for _, value := range values {
		result = append(result, value.Value.referencedSymbols()...)
	}
	return result
}

func (v Value) referencedSymbols() []string {
	if v.Type == ValueSymbol && v.Text != nil {
		return []string{*v.Text}
	}
	var result []string
	for _, element := range v.Elements {
		result = append(result, element.referencedSymbols()...)
	}
	for _, field := range v.Fields {
		result = append(result, field.Value.referencedSymbols()...)
	}
	return result
}
