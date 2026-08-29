package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
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
	tokens      int
	depth       int
	stringBytes int
}

func measureJSON(encoded []byte) (jsonMetrics, error) {
	var metrics jsonMetrics
	depth := 0
	for position := 0; position < len(encoded); {
		switch character := encoded[position]; character {
		case ' ', '\t', '\r', '\n':
			position++
		case '{', '[':
			metrics.tokens++
			depth++
			metrics.depth = max(metrics.depth, depth)
			position++
		case '}', ']':
			metrics.tokens++
			depth--
			position++
		case ',', ':':
			metrics.tokens++
			position++
		case '"':
			value, next, err := scanJSONString(encoded, position)
			if err != nil {
				return jsonMetrics{}, err
			}
			metrics.tokens++
			metrics.stringBytes = max(metrics.stringBytes, value.decodedBytes)
			position = next
			if nextNonspace(encoded, position) != ':' {
				metrics.depth = max(metrics.depth, depth+1)
			}
		default:
			metrics.tokens++
			metrics.depth = max(metrics.depth, depth+1)
			position++
			for position < len(encoded) && !isJSONSeparator(encoded[position]) {
				position++
			}
		}
	}
	return metrics, nil
}

type jsonString struct {
	start        int
	end          int
	decodedBytes int
}

func scanJSONString(encoded []byte, start int) (jsonString, int, error) {
	decodedBytes := 0
	for position := start + 1; position < len(encoded); position++ {
		switch encoded[position] {
		case '"':
			return jsonString{start: start + 1, end: position, decodedBytes: decodedBytes}, position + 1, nil
		case '\\':
			position++
			if position >= len(encoded) {
				return jsonString{}, 0, errors.New("unterminated JSON escape")
			}
			if encoded[position] != 'u' {
				decodedBytes++
				continue
			}
			codePoint, next, err := scanUnicodeEscape(encoded, position)
			if err != nil {
				return jsonString{}, 0, err
			}
			position = next
			decodedBytes += utf8.RuneLen(rune(codePoint))
		default:
			decodedBytes++
		}
	}
	return jsonString{}, 0, errors.New("unterminated JSON string")
}

func scanUnicodeEscape(encoded []byte, position int) (codePoint int, next int, err error) {
	codePoint, err = scanHexQuad(encoded, position+1)
	if err != nil {
		return 0, 0, err
	}
	position += 4
	if !utf16.IsSurrogate(rune(codePoint)) || codePoint > 0xdbff {
		return codePoint, position, nil
	}
	if position+6 >= len(encoded) || encoded[position+1] != '\\' || encoded[position+2] != 'u' {
		return 0, 0, errors.New("unpaired JSON surrogate")
	}
	low, lowErr := scanHexQuad(encoded, position+3)
	if lowErr != nil || low < 0xdc00 || low > 0xdfff {
		return 0, 0, errors.New("unpaired JSON surrogate")
	}
	return int(utf16.DecodeRune(rune(codePoint), rune(low))), position + 6, nil
}

func scanHexQuad(encoded []byte, start int) (int, error) {
	if start+4 > len(encoded) {
		return 0, errors.New("short JSON unicode escape")
	}
	value := 0
	for _, character := range encoded[start : start+4] {
		value <<= 4
		switch {
		case character >= '0' && character <= '9':
			value += int(character - '0')
		case character >= 'a' && character <= 'f':
			value += int(character-'a') + 10
		case character >= 'A' && character <= 'F':
			value += int(character-'A') + 10
		default:
			return 0, errors.New("invalid JSON unicode escape")
		}
	}
	return value, nil
}

func nextNonspace(encoded []byte, position int) byte {
	for position < len(encoded) {
		switch encoded[position] {
		case ' ', '\t', '\r', '\n':
			position++
		default:
			return encoded[position]
		}
	}
	return 0
}

func isJSONSeparator(character byte) bool {
	switch character {
	case ' ', '\t', '\r', '\n', ',', ']', '}':
		return true
	default:
		return false
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
	cursor := jsonCursor{encoded: encoded}
	var analysis jsonAnalysis
	if err := cursor.inspectValue(schema, "$", false, bounds, limits, &analysis); err != nil {
		return jsonAnalysis{}, err
	}
	cursor.skipSpace()
	if cursor.position != len(encoded) {
		return jsonAnalysis{}, errors.New("unexpected trailing JSON bytes")
	}
	return analysis, nil
}

type jsonCursor struct {
	encoded  []byte
	position int
}

func (c *jsonCursor) inspectValue(
	schema *jsonSchema,
	path JSONPath,
	formatField bool,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	c.skipSpace()
	if c.position >= len(c.encoded) {
		return errors.New("missing JSON value")
	}
	switch c.encoded[c.position] {
	case '{':
		if formatField {
			analysis.formatSeen = true
		}
		return c.inspectObject(schema, path, bounds, limits, analysis)
	case '[':
		if formatField {
			analysis.formatSeen = true
		}
		return c.inspectArray(schema, path, bounds, limits, analysis)
	case '"':
		return c.inspectString(path, formatField, bounds, limits, analysis)
	default:
		if formatField {
			analysis.formatSeen = true
		}
		start := c.position
		for c.position < len(c.encoded) && !isJSONSeparator(c.encoded[c.position]) {
			c.position++
		}
		if isJSONNumber(c.encoded[start:c.position]) && !isCanonicalIntegerBytes(c.encoded[start:c.position]) {
			analysis.noncanonicalValue = true
		}
		return nil
	}
}

func (c *jsonCursor) inspectString(
	path JSONPath,
	formatField bool,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	value, next, err := scanJSONString(c.encoded, c.position)
	if err != nil {
		return err
	}
	c.position = next
	limit := limits.stringBytes
	if bounds.StringLimit != nil {
		limit = tighterLimit(limit, bounds.StringLimit(path))
	}
	if exceeds(value.decodedBytes, limit) {
		analysis.stringLimit = true
	}
	if !formatField {
		return nil
	}
	analysis.formatSeen = true
	analysis.formatString = true
	if exceeds(value.decodedBytes, limits.stringBytes) {
		return nil
	}
	analysis.formatValue, err = decodeJSONString(c.encoded, value)
	return err
}

func isCanonicalIntegerBytes(value []byte) bool {
	if bytes.Equal(value, []byte{'0'}) {
		return true
	}
	value = bytes.TrimPrefix(value, []byte{'-'})
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

func isJSONNumber(value []byte) bool {
	return len(value) > 0 && (value[0] == '-' || value[0] >= '0' && value[0] <= '9')
}

func (c *jsonCursor) inspectObject(
	schema *jsonSchema,
	path JSONPath,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	c.position++
	c.skipSpace()
	limit := limits.objectMembers
	if bounds.CollectionLimit != nil {
		limit = tighterLimit(limit, bounds.CollectionLimit(path, CollectionObject))
	}
	contentStart := c.position
	var keys [MaximumJSONObjectMembers]jsonString
	members := 0
	for c.position < len(c.encoded) && c.encoded[c.position] != '}' {
		if members >= limit {
			analysis.collectionLimit = true
		}
		if err := c.inspectObjectMember(
			schema,
			path,
			bounds,
			limits,
			analysis,
			contentStart,
			members,
			limit,
			keys[:],
		); err != nil {
			return err
		}
		members++
		c.skipComma()
	}
	if c.position >= len(c.encoded) || c.encoded[c.position] != '}' {
		return errors.New("missing JSON object delimiter")
	}
	c.position++
	return nil
}

func (c *jsonCursor) inspectObjectMember(
	schema *jsonSchema,
	path JSONPath,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
	contentStart int,
	members int,
	limit int,
	keys []jsonString,
) error {
	keyPosition := c.position
	key, next, err := scanJSONString(c.encoded, c.position)
	if err != nil {
		return err
	}
	if exceeds(key.decodedBytes, limits.stringBytes) {
		analysis.stringLimit = true
	}
	c.noteObjectKey(contentStart, keyPosition, key, keys[:min(members, len(keys))], analysis)
	if members < len(keys) {
		keys[members] = key
	}
	childSchema := resolveChildSchema(c.encoded, schema, key, analysis)
	formatField := path == "$" && jsonStringEqualsPlain(c.encoded, key, "formatVersion")
	childPath, err := boundedChildPath(c.encoded, key, path, members, limit, limits.stringBytes)
	if err != nil {
		return err
	}
	c.position = next
	c.skipSpace()
	if c.position >= len(c.encoded) || c.encoded[c.position] != ':' {
		return errors.New("missing JSON object colon")
	}
	c.position++
	return c.inspectValue(childSchema, childPath, formatField, bounds, limits, analysis)
}

func boundedChildPath(
	encoded []byte,
	key jsonString,
	path JSONPath,
	members int,
	limit int,
	stringLimit int,
) (JSONPath, error) {
	if members >= limit || exceeds(key.decodedBytes, stringLimit) {
		return path, nil
	}
	decodedKey, err := decodeJSONString(encoded, key)
	if err != nil {
		return "", err
	}
	return appendObjectPath(path, decodedKey), nil
}

func (c *jsonCursor) noteObjectKey(
	contentStart int,
	keyPosition int,
	key jsonString,
	stored []jsonString,
	analysis *jsonAnalysis,
) {
	if len(stored) < MaximumJSONObjectMembers {
		for _, previous := range stored {
			noteKeyCollision(c.encoded, previous, key, analysis)
		}
		return
	}
	probe := jsonCursor{encoded: c.encoded, position: contentStart}
	for probe.position < keyPosition {
		previous, next, err := scanJSONString(probe.encoded, probe.position)
		if err != nil {
			return
		}
		noteKeyCollision(c.encoded, previous, key, analysis)
		probe.position = next
		probe.skipSpace()
		probe.position++
		_ = probe.skipValue()
		probe.skipSpace()
		if probe.position < keyPosition && probe.encoded[probe.position] == ',' {
			probe.position++
			probe.skipSpace()
		}
	}
}

func noteKeyCollision(encoded []byte, previous, key jsonString, analysis *jsonAnalysis) {
	if jsonStringsEqual(encoded, previous, key) {
		analysis.duplicateKey = true
		return
	}
	if jsonStringsFoldEqual(encoded, previous, key) {
		analysis.caseCollision = true
	}
}

func resolveChildSchema(encoded []byte, schema *jsonSchema, key jsonString, analysis *jsonAnalysis) *jsonSchema {
	if schema == nil {
		return &jsonSchema{kind: schemaAny}
	}
	switch schema.kind {
	case schemaObject:
		for name, child := range schema.fields {
			if jsonStringEqualsPlain(encoded, key, name) {
				return child
			}
			if jsonStringFoldEqualsPlain(encoded, key, name) {
				analysis.caseCollision = true
			}
		}
		analysis.unknownField = true
	case schemaMap:
		return schema.element
	default:
	}
	return &jsonSchema{kind: schemaAny}
}

func (c *jsonCursor) inspectArray(
	schema *jsonSchema,
	path JSONPath,
	bounds Bounds,
	limits structuralLimits,
	analysis *jsonAnalysis,
) error {
	c.position++
	c.skipSpace()
	limit := limits.arrayItems
	if bounds.CollectionLimit != nil {
		limit = tighterLimit(limit, bounds.CollectionLimit(path, CollectionArray))
	}
	elementSchema := &jsonSchema{kind: schemaAny}
	if schema != nil && schema.kind == schemaArray {
		elementSchema = schema.element
	}
	elementPath := appendArrayPath(path)
	items := 0
	for c.position < len(c.encoded) && c.encoded[c.position] != ']' {
		if items >= limit {
			analysis.collectionLimit = true
		}
		if err := c.inspectValue(elementSchema, elementPath, false, bounds, limits, analysis); err != nil {
			return err
		}
		items++
		c.skipSpace()
		if c.position < len(c.encoded) && c.encoded[c.position] == ',' {
			c.position++
			c.skipSpace()
		}
	}
	if c.position >= len(c.encoded) || c.encoded[c.position] != ']' {
		return errors.New("missing JSON array delimiter")
	}
	c.position++
	return nil
}

func (c *jsonCursor) skipValue() error {
	c.skipSpace()
	if c.position >= len(c.encoded) {
		return errors.New("missing JSON value")
	}
	switch c.encoded[c.position] {
	case '"':
		_, next, err := scanJSONString(c.encoded, c.position)
		c.position = next
		return err
	case '{':
		return c.skipObject()
	case '[':
		return c.skipArray()
	default:
		for c.position < len(c.encoded) && !isJSONSeparator(c.encoded[c.position]) {
			c.position++
		}
		return nil
	}
}

func (c *jsonCursor) skipObject() error {
	c.position++
	c.skipSpace()
	for c.position < len(c.encoded) && c.encoded[c.position] != '}' {
		_, next, err := scanJSONString(c.encoded, c.position)
		if err != nil {
			return err
		}
		c.position = next
		c.skipSpace()
		c.position++
		if err := c.skipValue(); err != nil {
			return err
		}
		c.skipComma()
	}
	c.position++
	return nil
}

func (c *jsonCursor) skipArray() error {
	c.position++
	c.skipSpace()
	for c.position < len(c.encoded) && c.encoded[c.position] != ']' {
		if err := c.skipValue(); err != nil {
			return err
		}
		c.skipComma()
	}
	c.position++
	return nil
}

func (c *jsonCursor) skipComma() {
	c.skipSpace()
	if c.position < len(c.encoded) && c.encoded[c.position] == ',' {
		c.position++
		c.skipSpace()
	}
}

func (c *jsonCursor) skipSpace() {
	for c.position < len(c.encoded) {
		switch c.encoded[c.position] {
		case ' ', '\t', '\r', '\n':
			c.position++
		default:
			return
		}
	}
}

func decodeJSONString(encoded []byte, value jsonString) (string, error) {
	var decoded strings.Builder
	decoded.Grow(value.decodedBytes)
	iterator := jsonStringIterator{encoded: encoded, value: value, position: value.start}
	for {
		character, ok := iterator.next()
		if !ok {
			return decoded.String(), nil
		}
		if _, err := decoded.WriteRune(character); err != nil {
			return "", err
		}
	}
}

func jsonStringsEqual(encoded []byte, left, right jsonString) bool {
	return compareJSONStrings(encoded, left, right, false)
}

func jsonStringsFoldEqual(encoded []byte, left, right jsonString) bool {
	return compareJSONStrings(encoded, left, right, true)
}

func compareJSONStrings(encoded []byte, left, right jsonString, fold bool) bool {
	leftIterator := jsonStringIterator{encoded: encoded, value: left, position: left.start}
	rightIterator := jsonStringIterator{encoded: encoded, value: right, position: right.start}
	for {
		leftRune, leftOK := leftIterator.next()
		rightRune, rightOK := rightIterator.next()
		if leftOK != rightOK {
			return false
		}
		if !leftOK {
			return true
		}
		if fold {
			leftRune = unicode.ToLower(leftRune)
			rightRune = unicode.ToLower(rightRune)
		}
		if leftRune != rightRune {
			return false
		}
	}
}

func jsonStringEqualsPlain(encoded []byte, value jsonString, plain string) bool {
	return compareJSONStringPlain(encoded, value, plain, false)
}

func jsonStringFoldEqualsPlain(encoded []byte, value jsonString, plain string) bool {
	return compareJSONStringPlain(encoded, value, plain, true)
}

func compareJSONStringPlain(encoded []byte, value jsonString, plain string, fold bool) bool {
	iterator := jsonStringIterator{encoded: encoded, value: value, position: value.start}
	for _, plainRune := range plain {
		valueRune, ok := iterator.next()
		if !ok {
			return false
		}
		if fold {
			valueRune = unicode.ToLower(valueRune)
			plainRune = unicode.ToLower(plainRune)
		}
		if valueRune != plainRune {
			return false
		}
	}
	_, remains := iterator.next()
	return !remains
}

type jsonStringIterator struct {
	encoded  []byte
	value    jsonString
	position int
}

func (i *jsonStringIterator) next() (rune, bool) {
	if i.position >= i.value.end {
		return 0, false
	}
	if i.encoded[i.position] != '\\' {
		character, width := utf8.DecodeRune(i.encoded[i.position:i.value.end])
		i.position += width
		return character, true
	}
	i.position++
	escape := i.encoded[i.position]
	i.position++
	switch escape {
	case '"', '\\', '/':
		return rune(escape), true
	case 'b':
		return '\b', true
	case 'f':
		return '\f', true
	case 'n':
		return '\n', true
	case 'r':
		return '\r', true
	case 't':
		return '\t', true
	default:
		codePoint, _ := scanHexQuad(i.encoded, i.position)
		i.position += 4
		if codePoint >= 0xd800 && codePoint <= 0xdbff {
			i.position += 2
			low, _ := scanHexQuad(i.encoded, i.position)
			i.position += 4
			return utf16.DecodeRune(rune(codePoint), rune(low)), true
		}
		return rune(codePoint), true
	}
}

func appendObjectPath(path JSONPath, key string) JSONPath {
	return JSONPath(string(path) + "." + key)
}

func appendArrayPath(path JSONPath) JSONPath {
	return JSONPath(string(path) + "[*]")
}
