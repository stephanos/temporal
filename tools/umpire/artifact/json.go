package artifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func CanonicalPretty(value any) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}
	return encoded.Bytes(), nil
}

type jsonMetrics struct {
	tokens int
	depth  int
}

type metricFrame struct {
	kind      json.Delim
	count     int
	expectKey bool
}

type metricState struct {
	metrics jsonMetrics
	stack   []metricFrame
}

func measureJSON(encoded []byte) (jsonMetrics, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var state metricState
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return state.metrics, nil
		}
		if err != nil {
			return jsonMetrics{}, err
		}
		if err := state.consume(token); err != nil {
			return jsonMetrics{}, err
		}
	}
}

func (s *metricState) consume(token json.Token) error {
	delimiter, isDelimiter := token.(json.Delim)
	if isDelimiter && (delimiter == '}' || delimiter == ']') {
		return s.consumeClosing(delimiter)
	}
	if s.expectsObjectKey() {
		return s.consumeObjectKey(token)
	}
	startMetricValue(&s.metrics, s.stack)
	s.metrics.tokens++
	depth := len(s.stack) + 1
	if depth > s.metrics.depth {
		s.metrics.depth = depth
	}
	if !isDelimiter {
		completeMetricValue(s.stack)
		return nil
	}
	switch delimiter {
	case '{':
		s.stack = append(s.stack, metricFrame{kind: delimiter, expectKey: true})
	case '[':
		s.stack = append(s.stack, metricFrame{kind: delimiter})
	default:
		return fmt.Errorf("unexpected opening delimiter %q", delimiter)
	}
	return nil
}

func (s *metricState) consumeClosing(delimiter json.Delim) error {
	s.metrics.tokens++
	if len(s.stack) == 0 || (delimiter == '}' && s.stack[len(s.stack)-1].kind != '{') ||
		(delimiter == ']' && s.stack[len(s.stack)-1].kind != '[') {
		return fmt.Errorf("unexpected closing delimiter %q", delimiter)
	}
	s.stack = s.stack[:len(s.stack)-1]
	completeMetricValue(s.stack)
	return nil
}

func (s *metricState) expectsObjectKey() bool {
	return len(s.stack) > 0 && s.stack[len(s.stack)-1].kind == '{' && s.stack[len(s.stack)-1].expectKey
}

func (s *metricState) consumeObjectKey(token json.Token) error {
	if _, ok := token.(string); !ok {
		return fmt.Errorf("JSON object key has type %T", token)
	}
	frame := &s.stack[len(s.stack)-1]
	if frame.count > 0 {
		s.metrics.tokens++
	}
	s.metrics.tokens += 2
	frame.count++
	frame.expectKey = false
	return nil
}

func startMetricValue(metrics *jsonMetrics, stack []metricFrame) {
	if len(stack) == 0 {
		return
	}
	frame := &stack[len(stack)-1]
	if frame.kind == '[' {
		if frame.count > 0 {
			metrics.tokens++
		}
		frame.count++
	}
}

func completeMetricValue(stack []metricFrame) {
	if len(stack) == 0 {
		return
	}
	frame := &stack[len(stack)-1]
	if frame.kind == '{' {
		frame.expectKey = true
	}
}

type schemaKind uint8

const (
	schemaAny schemaKind = iota
	schemaObject
	schemaArray
	schemaMap
)

type jsonSchema struct {
	kind    schemaKind
	fields  map[string]*jsonSchema
	folded  map[string]string
	element *jsonSchema
}

func schemaFor[T any]() *jsonSchema {
	return buildSchema(reflect.TypeFor[T](), make(map[reflect.Type]*jsonSchema))
}

func buildSchema(typ reflect.Type, cache map[reflect.Type]*jsonSchema) *jsonSchema {
	if typ == nil {
		return &jsonSchema{kind: schemaAny}
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if cached, ok := cache[typ]; ok {
		return cached
	}
	schema := &jsonSchema{kind: schemaAny}
	cache[typ] = schema
	switch typ.Kind() {
	case reflect.Struct:
		populateObjectSchema(schema, typ, cache)
	case reflect.Slice, reflect.Array:
		schema.kind = schemaArray
		schema.element = buildSchema(typ.Elem(), cache)
	case reflect.Map:
		schema.kind = schemaMap
		schema.element = buildSchema(typ.Elem(), cache)
	default:
	}
	return schema
}

func populateObjectSchema(schema *jsonSchema, typ reflect.Type, cache map[reflect.Type]*jsonSchema) {
	schema.kind = schemaObject
	schema.fields = make(map[string]*jsonSchema)
	schema.folded = make(map[string]string)
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.PkgPath != "" {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		if addEmbeddedSchema(schema, field, name, cache) {
			continue
		}
		if name == "" {
			name = field.Name
		}
		schema.fields[name] = buildSchema(field.Type, cache)
		schema.folded[strings.ToLower(name)] = name
	}
}

func addEmbeddedSchema(schema *jsonSchema, field reflect.StructField, name string, cache map[reflect.Type]*jsonSchema) bool {
	if !field.Anonymous || name != "" {
		return false
	}
	embedded := buildSchema(field.Type, cache)
	if embedded.kind != schemaObject {
		return false
	}
	for embeddedName, embeddedField := range embedded.fields {
		schema.fields[embeddedName] = embeddedField
		schema.folded[strings.ToLower(embeddedName)] = embeddedName
	}
	return true
}

type jsonAnalysis struct {
	duplicateKey      bool
	caseCollision     bool
	unknownField      bool
	collectionLimit   bool
	stringLimit       bool
	noncanonicalValue bool
	formatSeen        bool
	formatString      bool
	formatValue       string
}

func (a jsonAnalysis) format() (string, bool) {
	return a.formatValue, a.formatSeen && a.formatString
}

func inspectJSON(encoded []byte, schema *jsonSchema, bounds Bounds, limits structuralLimits) (jsonAnalysis, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var analysis jsonAnalysis
	if err := inspectJSONValue(decoder, schema, "$", bounds, limits, &analysis); err != nil {
		return jsonAnalysis{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return jsonAnalysis{}, err
	}
	return analysis, nil
}

func inspectJSONValue(
	decoder *json.Decoder,
	schema *jsonSchema,
	path JSONPath,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if path == "$.formatVersion" {
		analysis.formatSeen = true
		analysis.formatValue, analysis.formatString = token.(string)
	}
	delimiter, structured := token.(json.Delim)
	if !structured {
		if value, ok := token.(string); ok {
			limit := limits.stringBytes
			if bounds.StringLimit != nil {
				limit = tighterLimit(limit, bounds.StringLimit(path))
			}
			if exceeds(len(value), limit) {
				analysis.stringLimit = true
			}
		}
		if number, ok := token.(json.Number); ok && !isCanonicalInteger(string(number)) {
			analysis.noncanonicalValue = true
		}
		return nil
	}
	switch delimiter {
	case '{':
		return inspectJSONObject(decoder, schema, path, bounds, limits, analysis)
	case '[':
		return inspectJSONArray(decoder, schema, path, bounds, limits, analysis)
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func isCanonicalInteger(value string) bool {
	if value == "0" {
		return true
	}
	value = strings.TrimPrefix(value, "-")
	if len(value) == 0 || value[0] == '0' {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func inspectJSONObject(
	decoder *json.Decoder,
	schema *jsonSchema,
	path JSONPath,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	limit := limits.objectMembers
	if bounds.CollectionLimit != nil {
		limit = tighterLimit(limit, bounds.CollectionLimit(path, CollectionObject))
	}
	seen := make(map[string]struct{})
	folded := make(map[string]string)
	members := 0
	for decoder.More() {
		if members >= limit {
			analysis.collectionLimit = true
		}
		members++
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("JSON object key has type %T", keyToken)
		}
		if exceeds(len(key), limits.stringBytes) {
			analysis.stringLimit = true
		}
		foldedKey := noteObjectKey(key, seen, folded, analysis)
		childSchema := resolveChildSchema(schema, key, foldedKey, analysis)
		if err := inspectJSONValue(decoder, childSchema, appendObjectPath(path, key), bounds, limits, analysis); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	if closing != json.Delim('}') {
		return fmt.Errorf("unexpected JSON object delimiter %q", closing)
	}
	return nil
}

func noteObjectKey(key string, seen map[string]struct{}, folded map[string]string, analysis *jsonAnalysis) string {
	if _, exists := seen[key]; exists {
		analysis.duplicateKey = true
	}
	seen[key] = struct{}{}
	foldedKey := strings.ToLower(key)
	previous, exists := folded[foldedKey]
	if exists && previous != key {
		analysis.caseCollision = true
	}
	if !exists {
		folded[foldedKey] = key
	}
	return foldedKey
}

func resolveChildSchema(schema *jsonSchema, key, foldedKey string, analysis *jsonAnalysis) *jsonSchema {
	if schema == nil {
		return &jsonSchema{kind: schemaAny}
	}
	switch schema.kind {
	case schemaObject:
		if canonical, exists := schema.folded[foldedKey]; exists && canonical != key {
			analysis.caseCollision = true
		}
		child, known := schema.fields[key]
		if known {
			return child
		}
		analysis.unknownField = true
	case schemaMap:
		return schema.element
	default:
	}
	return &jsonSchema{kind: schemaAny}
}

func inspectJSONArray(
	decoder *json.Decoder,
	schema *jsonSchema,
	path JSONPath,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	limit := limits.arrayItems
	if bounds.CollectionLimit != nil {
		limit = tighterLimit(limit, bounds.CollectionLimit(path, CollectionArray))
	}
	elementSchema := &jsonSchema{kind: schemaAny}
	if schema != nil && schema.kind == schemaArray {
		elementSchema = schema.element
	}
	items := 0
	for decoder.More() {
		if items >= limit {
			analysis.collectionLimit = true
		}
		items++
		if err := inspectJSONValue(decoder, elementSchema, appendArrayPath(path), bounds, limits, analysis); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	if closing != json.Delim(']') {
		return fmt.Errorf("unexpected JSON array delimiter %q", closing)
	}
	return nil
}

func appendObjectPath(path JSONPath, key string) JSONPath {
	return JSONPath(string(path) + "." + key)
}

func appendArrayPath(path JSONPath) JSONPath {
	return JSONPath(string(path) + "[*]")
}
