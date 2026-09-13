package protocolmigration

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	// Registers the WorkflowService descriptors a baseline request assignment's method names.
	_ "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// nameEnumLiterals replaces the number of every enum literal with the value's name. The number is a
// baseline number, so it names a value of the baseline enum the literal's context expects, and that
// value must keep its name in the current protocol. The contexts are the ones the baseline fixtures
// use: a request assignment typed by the request field its target path reaches, and a comparison
// with an instruction status or a Run Event payload path. A literal in any other context fails
// rather than being named by guess.
func nameEnumLiterals(snapshot *Baseline, _ string, tree any) (any, error) {
	if snapshot == nil {
		return nil, errors.New("enum literals are named through the baseline snapshot, and none is supplied")
	}
	if _, err := rewriteTree(tree, func(value any) (any, error) {
		object, ok := value.(*Object)
		if !ok {
			return value, nil
		}
		if compare, ok := object.Fields["compare"].(*Object); ok {
			if err := nameComparedLiterals(snapshot, compare); err != nil {
				return nil, err
			}
		}
		if assignments, ok := object.Fields["requestAssignments"].([]any); ok {
			if err := nameAssignedLiterals(object, assignments); err != nil {
				return nil, err
			}
		}
		return object, nil
	}); err != nil {
		return nil, err
	}
	return rewriteTree(tree, func(value any) (any, error) {
		if object, ok := value.(*Object); ok {
			if literal, ok := object.Fields["enumValue"].(*Object); ok {
				if _, named := literal.Fields["name"]; !named {
					return nil, fmt.Errorf("enum literal %v has no context whose enum the step derives", literal.Fields)
				}
			}
		}
		return value, nil
	})
}

func nameComparedLiterals(snapshot *Baseline, compare *Object) error {
	for _, side := range [][2]string{{"left", "right"}, {"right", "left"}} {
		literal := enumLiteral(compare.Fields[side[0]])
		if literal == nil {
			continue
		}
		enum, err := snapshot.comparedEnum(compare.Fields[side[1]])
		if err != nil {
			return fmt.Errorf("enum literal compared on the %s: %w", side[0], err)
		}
		if err := nameLiteral(literal, enum); err != nil {
			return err
		}
	}
	return nil
}

// nameAssignedLiterals names each enum literal an InvokeRpc assigns, by the enum of the request field
// its target reaches. Request messages belong to the Temporal API, which the migration did not change.
func nameAssignedLiterals(invoke *Object, assignments []any) error {
	for index, element := range assignments {
		assignment, _ := element.(*Object)
		if assignment == nil {
			continue
		}
		value, _ := assignment.Fields["value"].(*Object)
		literal := enumLiteral(value)
		if literal == nil {
			continue
		}
		method, _ := invoke.Fields["method"].(string)
		service, name, found := strings.Cut(strings.TrimPrefix(method, "/"), "/")
		if !found {
			return fmt.Errorf("request assignment %d: method %q is not /package.Service/Method", index, method)
		}
		descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(service))
		if err != nil {
			return fmt.Errorf("request assignment %d: %w", index, err)
		}
		serviceDescriptor, isService := descriptor.(protoreflect.ServiceDescriptor)
		if !isService || serviceDescriptor.Methods().ByName(protoreflect.Name(name)) == nil {
			return fmt.Errorf("request assignment %d: %s declares no method %s", index, service, name)
		}
		enum, err := enumAt(serviceDescriptor.Methods().ByName(protoreflect.Name(name)).Input(), assignment.Fields["target"])
		if err != nil {
			return fmt.Errorf("request assignment %d: %w", index, err)
		}
		if err := nameLiteral(literal, enum); err != nil {
			return fmt.Errorf("request assignment %d: %w", index, err)
		}
	}
	return nil
}

// enumLiteral is the enum literal operand holds, or nil when it holds none.
func enumLiteral(operand any) *Object {
	expression, _ := operand.(*Object)
	if expression == nil {
		return nil
	}
	value, _ := expression.Fields["literal"].(*Object)
	if value == nil {
		return nil
	}
	literal, _ := value.Fields["enumValue"].(*Object)
	return literal
}

// comparedEnum is the baseline enum of the operand an enum literal is compared with: an instruction
// outcome status, or a field a path reads from a Run Event payload.
func (b *Baseline) comparedEnum(operand any) (protoreflect.EnumDescriptor, error) {
	expression, _ := operand.(*Object)
	if expression == nil {
		return nil, errors.New("the other operand is not an expression")
	}
	if reference, ok := expression.Fields["reference"].(*Object); ok {
		if outcome, ok := reference.Fields["outcome"].(*Object); ok {
			if field := literalText(outcome.Fields["field"]); field != "INSTRUCTION_OUTCOME_FIELD_STATUS" && field != "1" {
				return nil, fmt.Errorf("outcome field %s is not the status", field)
			}
			return b.snapshotEnum("InstructionOutcomeStatus")
		}
	}
	if read, ok := expression.Fields["path"].(*Object); ok {
		operand, _ := read.Fields["operand"].(*Object)
		reference, _ := operand.fields()["reference"].(*Object)
		runEvent, _ := reference.fields()["runEvent"].(*Object)
		if _, payload := runEvent.fields()["payload"]; payload {
			descriptor, err := b.files.FindDescriptorByName(protocol + "RunEvent")
			if err != nil {
				return nil, err
			}
			return enumAt(descriptor.(protoreflect.MessageDescriptor), read.Fields["path"])
		}
	}
	return nil, errors.New("the other operand's enum is not one the step derives")
}

func (o *Object) fields() map[string]any {
	if o == nil {
		return nil
	}
	return o.Fields
}

func (b *Baseline) snapshotEnum(name protoreflect.Name) (protoreflect.EnumDescriptor, error) {
	descriptor, err := b.files.FindDescriptorByName(protocol + protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	enum, ok := descriptor.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not an enum", name)
	}
	return enum, nil
}

// enumAt walks a baseline FieldPath object from message to the enum field it reaches.
func enumAt(message protoreflect.MessageDescriptor, path any) (protoreflect.EnumDescriptor, error) {
	object, _ := path.(*Object)
	segments, _ := object.fields()["segments"].([]any)
	var field protoreflect.FieldDescriptor
	for index, element := range segments {
		if message == nil {
			return nil, fmt.Errorf("path segment %d reads into a non-message", index)
		}
		segment, _ := element.(*Object)
		name, _ := segment.fields()["field"].(string)
		if oneof, ok := segment.fields()["oneof"].(*Object); ok {
			selected, _ := oneof.Fields["selectedField"].(string)
			if message.Oneofs().ByName(protoreflect.Name(name)) == nil {
				return nil, fmt.Errorf("%s has no oneof %s", message.FullName(), name)
			}
			name = selected
		}
		field = message.Fields().ByName(protoreflect.Name(name))
		if field == nil {
			return nil, fmt.Errorf("%s has no field %s", message.FullName(), name)
		}
		message = field.Message()
	}
	if field == nil || field.Enum() == nil {
		return nil, errors.New("the path does not reach an enum field")
	}
	return field.Enum(), nil
}

// nameLiteral replaces literal's baseline number with its name in enum, which the current enum of the
// same full name must still declare.
func nameLiteral(literal *Object, enum protoreflect.EnumDescriptor) error {
	number := int64(0)
	for key, value := range literal.Fields {
		if key != "number" {
			return fmt.Errorf("enum literal carries %q", key)
		}
		parsed, err := strconv.ParseInt(literalText(value), 10, 32)
		if err != nil {
			return fmt.Errorf("enum literal number %v: %w", value, err)
		}
		number = parsed
	}
	value := enum.Values().ByNumber(protoreflect.EnumNumber(number))
	if value == nil {
		return fmt.Errorf("%s declares no number %d", enum.FullName(), number)
	}
	current, err := protoregistry.GlobalFiles.FindDescriptorByName(enum.FullName())
	if err != nil {
		return fmt.Errorf("the current protocol lacks %s: %w", enum.FullName(), err)
	}
	if currentEnum, ok := current.(protoreflect.EnumDescriptor); !ok || currentEnum.Values().ByName(value.Name()) == nil {
		return fmt.Errorf("the current %s no longer declares %s", enum.FullName(), value.Name())
	}
	literal.Fields = map[string]any{"name": string(value.Name())}
	return nil
}

// pathName is a protobuf field name the path grammar spells as a segment name.
var (
	pathName   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	integerKey = regexp.MustCompile(`^-?[0-9]+$`)
)

// spellFieldPath replaces a baseline FieldPath object with its string in the path grammar. It spells
// a segment and its one selector exactly as the grammar does, independently of the runtime parser,
// and fails on a segment the grammar cannot spell.
func spellFieldPath(_ string, object *Object) (any, error) {
	var spelled strings.Builder
	for key, value := range object.Fields {
		if key != "segments" {
			return nil, fmt.Errorf("field path carries %q", key)
		}
		segments, ok := value.([]any)
		if !ok {
			return nil, errors.New("field path segments is not a list")
		}
		for index, element := range segments {
			segment, err := spellSegment(element)
			if err != nil {
				return nil, fmt.Errorf("segment %d: %w", index, err)
			}
			if index > 0 {
				spelled.WriteByte('.')
			}
			spelled.WriteString(segment)
		}
	}
	return spelled.String(), nil
}

func spellSegment(element any) (string, error) {
	segment, _ := element.(*Object)
	if segment == nil {
		return "", errors.New("segment is not an object")
	}
	name, _ := segment.Fields["field"].(string)
	if !pathName.MatchString(name) {
		return "", fmt.Errorf("field name %q is not a path segment name", name)
	}
	if len(segment.Fields) > 2 {
		return "", fmt.Errorf("segment carries %d keys, want a field and at most one selector", len(segment.Fields))
	}
	for key, value := range segment.Fields {
		selector, _ := value.(*Object)
		switch key {
		case "field":
		case "repeated":
			return name + "[*]", nil
		case "presence":
			return name + "?", nil
		case "oneof":
			member, _ := selector.fields()["selectedField"].(string)
			if !pathName.MatchString(member) || len(selector.fields()) != 1 {
				return "", fmt.Errorf("oneof selector %v names no member", selector.fields())
			}
			return name + "<" + member + ">", nil
		case "mapKey":
			key, err := spellMapKey(selector.fields()["key"])
			if err != nil {
				return "", err
			}
			return name + "[" + key + "]", nil
		default:
			return "", fmt.Errorf("segment carries %q", key)
		}
	}
	return name, nil
}

// spellMapKey spells a map key Value: a text key as a JSON string escaping only the quote, the
// backslash and control characters, and an integer or boolean key bare.
func spellMapKey(value any) (string, error) {
	key, _ := value.(*Object)
	if len(key.fields()) != 1 {
		return "", fmt.Errorf("map key %v is not one value", key.fields())
	}
	for arm, literal := range key.Fields {
		switch arm {
		case "textValue":
			text, _ := literal.(string)
			var spelled strings.Builder
			spelled.WriteByte('"')
			for _, r := range text {
				switch {
				case r == '"' || r == '\\':
					spelled.WriteRune('\\')
					spelled.WriteRune(r)
				case r == '\n':
					spelled.WriteString(`\n`)
				case r == '\r':
					spelled.WriteString(`\r`)
				case r < 0x20:
					fmt.Fprintf(&spelled, "\\u%04x", r)
				default:
					spelled.WriteRune(r)
				}
			}
			spelled.WriteByte('"')
			return spelled.String(), nil
		case "boolValue":
			if literal != true && literal != false {
				return "", fmt.Errorf("boolean map key %v", literal)
			}
			return fmt.Sprint(literal), nil
		case "signedIntegerValue", "unsignedIntegerValue":
			text, _ := literal.(string)
			if !integerKey.MatchString(text) {
				return "", fmt.Errorf("integer map key %q", text)
			}
			return text, nil
		default:
			return "", fmt.Errorf("map key arm %q", arm)
		}
	}
	return "", errors.New("unreachable")
}
