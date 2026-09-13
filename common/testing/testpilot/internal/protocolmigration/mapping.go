package protocolmigration

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Object is one JSON object of a fixture while the Mapping runs. Message is the snapshot message
// the object encoded when the baseline was validated, and it stays that name after steps rename
// the object's fields or the message itself, so every step addresses messages by their baseline
// names. Objects that encode no snapshot message (map fields, Any payloads of other packages,
// correlated fixture entries, objects a step builds) carry an empty Message.
type Object struct {
	Message protoreflect.FullName
	Fields  map[string]any
}

func (o *Object) MarshalJSON() ([]byte, error) {
	if o.Fields == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(o.Fields)
}

// ApplyFunc transforms one fixture's JSON tree. fixture is the repository-relative path the
// baseline and regenerated trees share, so a step can confine itself to the fixtures it names.
// Scalars are json.Number, string, bool or nil; arrays are []any; objects are *Object.
type ApplyFunc func(fixture string, tree any) (any, error)

// Step is one declared difference between the baseline protocol and the current one.
type Step struct {
	Name string
	// Requirement is the fn-87 R-ID the step implements.
	Requirement string
	Apply       ApplyFunc
}

// Mapping is the ordered list of Steps from the frozen baseline to the current protocol.
type Mapping []Step

// Declared is the mapping from the baseline snapshot to the current protocol. Every structural
// task appends the steps its change declares; a difference no step declares fails the
// equivalence test.
var Declared = Mapping{}

func (m Mapping) apply(fixture string, tree any) (any, error) {
	for _, step := range m {
		mapped, err := step.Apply(fixture, tree)
		if err != nil {
			return nil, fmt.Errorf("fixture %s: step %q (%s): %w", fixture, step.Name, step.Requirement, err)
		}
		tree = mapped
	}
	return tree, nil
}

// RewriteMessages replaces every object that encoded message with rewrite's result. Nested
// messages are rewritten before the objects that contain them.
func RewriteMessages(message protoreflect.FullName, rewrite func(fixture string, object *Object) (any, error)) ApplyFunc {
	return func(fixture string, tree any) (any, error) {
		return rewriteTree(tree, func(value any) (any, error) {
			object, ok := value.(*Object)
			if !ok || object.Message != message {
				return value, nil
			}
			return rewrite(fixture, object)
		})
	}
}

// RenameField moves field from to field to in every object that encoded message.
func RenameField(message protoreflect.FullName, from, to string) ApplyFunc {
	return RewriteMessages(message, func(_ string, object *Object) (any, error) {
		value, ok := object.Fields[from]
		if !ok {
			return object, nil
		}
		if _, exists := object.Fields[to]; exists {
			return nil, fmt.Errorf("%s carries both %q and %q", message, from, to)
		}
		delete(object.Fields, from)
		object.Fields[to] = value
		return object, nil
	})
}

// RenameEnumLiteral replaces the enum literal from with to in field of every object that encoded
// message, whether the field holds one literal or a list. A literal is its number or its value
// name; to is written as a JSON number when it is one.
func RenameEnumLiteral(message protoreflect.FullName, field, from, to string) ApplyFunc {
	return RewriteMessages(message, func(_ string, object *Object) (any, error) {
		value, ok := object.Fields[field]
		if !ok {
			return object, nil
		}
		if list, isList := value.([]any); isList {
			for index, element := range list {
				list[index] = renameLiteral(element, from, to)
			}
			return object, nil
		}
		object.Fields[field] = renameLiteral(value, from, to)
		return object, nil
	})
}

func renameLiteral(value any, from, to string) any {
	var text string
	switch literal := value.(type) {
	case json.Number:
		text = string(literal)
	case string:
		text = literal
	default:
		return value
	}
	if text != from {
		return value
	}
	if _, err := strconv.ParseInt(to, 10, 32); err == nil {
		return json.Number(to)
	}
	return to
}

// DropField removes field from every object that encoded message. check runs on each object
// that still carries the field and must prove the value is what the current protocol derives or
// defaults, so a drop never discards data silently.
func DropField(message protoreflect.FullName, field string, check func(fixture string, object *Object) error) ApplyFunc {
	return RewriteMessages(message, func(fixture string, object *Object) (any, error) {
		if _, ok := object.Fields[field]; !ok {
			return object, nil
		}
		if check == nil {
			return nil, errors.New("dropping " + string(message) + "." + field + " declares no check")
		}
		if err := check(fixture, object); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", message, field, err)
		}
		delete(object.Fields, field)
		return object, nil
	})
}

func rewriteTree(value any, visit func(any) (any, error)) (any, error) {
	switch node := value.(type) {
	case *Object:
		for key, field := range node.Fields {
			rewritten, err := rewriteTree(field, visit)
			if err != nil {
				return nil, err
			}
			node.Fields[key] = rewritten
		}
	case []any:
		for index, element := range node {
			rewritten, err := rewriteTree(element, visit)
			if err != nil {
				return nil, err
			}
			node[index] = rewritten
		}
	default:
		// Scalars have no children.
	}
	return visit(value)
}
