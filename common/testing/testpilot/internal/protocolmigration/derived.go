package protocolmigration

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
)

// The oracle spells the derivation rules again rather than calling preparation, so a regenerated
// fixture that drops a declaration preparation derives differently still fails here.

// profileInstructionDefaults is the instruction timeout and attempts each fixture's Profile gives an
// instruction that writes none, keyed by the Profile names of profileCeilings.
var profileInstructionDefaults = map[string]struct{ timeoutMilliseconds, maxAttempts int64 }{
	// temporal.DefaultInstructionLimits.
	"temporal": {timeoutMilliseconds: 10000, maxAttempts: 1},
	// The synthetic Case's and the correlated corpus's own instruction bounds, which their test
	// Profiles take as defaults.
	"synthetic":  {timeoutMilliseconds: 1000, maxAttempts: 1},
	"correlated": {timeoutMilliseconds: 1000, maxAttempts: 1},
}

// startWorkflowExecution is the one reservation carrier every fixture Profile authorizes.
const startWorkflowExecution = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"

// statusType and textType are the ProtoJSON spellings of the two outcome field types preparation
// derives.
const (
	statusType = `{"singular":{"enumeration":{"protobufType":"temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}`
	textType   = `{"singular":{"scalar":{"kind":"SCALAR_KIND_TEXT"}}}`
)

// outcomeFieldNames spells InstructionOutcomeField by number, so a literal written either way reads
// as its name.
var outcomeFieldNames = map[string]string{
	"1": "INSTRUCTION_OUTCOME_FIELD_STATUS",
	"2": "INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE",
	"3": "INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE",
	"4": "INSTRUCTION_OUTCOME_FIELD_DETAIL",
	"5": "INSTRUCTION_OUTCOME_FIELD_VALUE",
}

// deriveProgramDeclarations drops the Program declarations preparation now derives, after checking
// each equals what it derives: the environment binding list, each instruction's activation
// reservations and outcome fields, and each instruction limit equal to the fixture Profile's default.
func deriveProgramDeclarations(fixture string, program *Object) (any, error) {
	if err := dropEnvironment(program); err != nil {
		return nil, err
	}
	profile, err := fixtureProfile(fixture)
	if err != nil {
		return nil, err
	}
	defaults := profileInstructionDefaults[profile]
	graphs, err := instructionGraphs(program)
	if err != nil {
		return nil, err
	}
	reservable := reservableEntrypoints(program)
	for _, graph := range graphs {
		for index, node := range graph.instructions {
			if err := dropReservations(graph, node, reservable); err != nil {
				return nil, fmt.Errorf("%s instruction %d: %w", graph.id, index, err)
			}
			if err := dropOutcome(program, graph, node); err != nil {
				return nil, fmt.Errorf("%s instruction %d: %w", graph.id, index, err)
			}
			if err := dropDefaultLimits(node, defaults.timeoutMilliseconds, defaults.maxAttempts); err != nil {
				return nil, fmt.Errorf("%s instruction %d: %w", graph.id, index, err)
			}
		}
	}
	return program, nil
}

// dropEnvironment checks the declared binding list is the binding graph preparation derives: every
// binding a role names, namespace before resource in role order, then every other binding an
// expression references, each once.
func dropEnvironment(program *Object) error {
	value, declared := program.Fields["environment"]
	if !declared {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return errors.New("environment is not a list")
	}
	written := make([]string, len(list))
	for index, element := range list {
		definition, isObject := element.(*Object)
		if !isObject {
			return fmt.Errorf("environment %d is not an object", index)
		}
		written[index] = literalText(definition.Fields["bindingId"])
	}
	var derived []string
	roles, _ := program.Fields["roles"].([]any)
	for _, element := range roles {
		role, _ := element.(*Object)
		if role == nil {
			continue
		}
		for _, key := range []string{"namespaceBindingId", "resourceBindingId"} {
			if id := literalText(role.Fields[key]); id != "" && !slices.Contains(derived, id) {
				derived = append(derived, id)
			}
		}
	}
	var referenced []string
	collectEnvironmentReferences(program.Fields["entrypoints"], &referenced)
	collectEnvironmentReferences(program.Fields["cleanup"], &referenced)
	// A mapped object's fields are an unordered map, so declaration order is not recoverable here;
	// the extra references compare as a sorted set. Every checked-in reference is a role binding.
	slices.Sort(referenced)
	for _, id := range slices.Compact(referenced) {
		if !slices.Contains(derived, id) {
			derived = append(derived, id)
		}
	}
	if !slices.Equal(written, derived) {
		return fmt.Errorf("environment %v is not the derived binding graph %v", written, derived)
	}
	delete(program.Fields, "environment")
	return nil
}

func collectEnvironmentReferences(value any, ids *[]string) {
	switch node := value.(type) {
	case *Object:
		if id, ok := node.Fields["environmentBindingId"]; ok {
			*ids = append(*ids, literalText(id))
		}
		for _, field := range node.Fields {
			collectEnvironmentReferences(field, ids)
		}
	case []any:
		for _, element := range node {
			collectEnvironmentReferences(element, ids)
		}
	default:
	}
}

// instructionGraph is one entrypoint's or cleanup's instruction list, with whether a controller runs
// it and whether it is the cleanup graph.
type instructionGraph struct {
	id                  string
	controller, cleanup bool
	instructions        []*Object
}

func instructionGraphs(program *Object) ([]instructionGraph, error) {
	var graphs []instructionGraph
	entrypoints, _ := program.Fields["entrypoints"].([]any)
	for index, element := range entrypoints {
		entrypoint, ok := element.(*Object)
		if !ok {
			return nil, fmt.Errorf("entrypoint %d is not an object", index)
		}
		_, controller := entrypoint.Fields["controller"]
		graph, err := graphOf(entrypoint, controller, false)
		if err != nil {
			return nil, err
		}
		graphs = append(graphs, graph)
	}
	if cleanup, ok := program.Fields["cleanup"].(*Object); ok {
		graph, err := graphOf(cleanup, true, true)
		if err != nil {
			return nil, err
		}
		graphs = append(graphs, graph)
	}
	return graphs, nil
}

func graphOf(object *Object, controller, cleanup bool) (instructionGraph, error) {
	graph := instructionGraph{id: literalText(object.Fields["entrypointId"]), controller: controller, cleanup: cleanup}
	list, _ := object.Fields["instructions"].([]any)
	for index, element := range list {
		node, ok := element.(*Object)
		if !ok {
			return instructionGraph{}, fmt.Errorf("%s instruction %d is not an object", graph.id, index)
		}
		graph.instructions = append(graph.instructions, node)
	}
	return graph, nil
}

// reservableEntrypoints are the workflow and Nexus-handler entrypoints, in declaration order: the
// ones a carrier reserves one activation of.
func reservableEntrypoints(program *Object) []string {
	var ids []string
	entrypoints, _ := program.Fields["entrypoints"].([]any)
	for _, element := range entrypoints {
		entrypoint, _ := element.(*Object)
		if entrypoint == nil {
			continue
		}
		_, workflow := entrypoint.Fields["workflow"]
		_, handler := entrypoint.Fields["nexusHandler"]
		if workflow || handler {
			ids = append(ids, literalText(entrypoint.Fields["entrypointId"]))
		}
	}
	return ids
}

// dropReservations checks an instruction's reservations are the ones preparation derives: an ordinary
// controller's StartWorkflowExecution reserves one activation of every reservable entrypoint, and no
// other instruction reserves any.
func dropReservations(graph instructionGraph, node *Object, reservable []string) error {
	key := "activation" + "Reservations"
	var derived []string
	if graph.controller && !graph.cleanup && invokedMethod(node) == startWorkflowExecution {
		derived = reservable
	}
	value, declared := node.Fields[key]
	list, _ := value.([]any)
	written := make([]string, len(list))
	for index, element := range list {
		reservation, ok := element.(*Object)
		if !ok {
			return fmt.Errorf("reservation %d is not an object", index)
		}
		if count, err := strconv.ParseInt(literalText(reservation.Fields["count"]), 10, 64); err != nil || count != 1 {
			return fmt.Errorf("reservation %d reserves %v activations, not the one preparation derives", index, reservation.Fields["count"])
		}
		written[index] = literalText(reservation.Fields["entrypointId"])
	}
	if !slices.Equal(written, derived) {
		return fmt.Errorf("reservations %v are not the derived reservations %v", written, derived)
	}
	if declared {
		delete(node.Fields, key)
	}
	return nil
}

func invokedMethod(node *Object) string {
	instruction, _ := node.Fields["instruction"].(*Object)
	if instruction == nil {
		return ""
	}
	rpc, _ := instruction.Fields["invokeRpc"].(*Object)
	if rpc == nil {
		return ""
	}
	return literalText(rpc.Fields["method"])
}

func instructionKind(node *Object) string {
	instruction, _ := node.Fields["instruction"].(*Object)
	if instruction == nil {
		return ""
	}
	for kind := range instruction.Fields {
		return kind
	}
	return ""
}

// derivedOutcomeTypes are the outcome fields preparation derives for an instruction, with their
// types: every instruction a status and a detail, a controller protocol effect a protocol code, a
// worker instruction an SDK failure code, and an awaited Nexus operation its text value.
func derivedOutcomeTypes(graph instructionGraph, kind string) map[string]string {
	fields := map[string]string{"INSTRUCTION_OUTCOME_FIELD_STATUS": statusType, "INSTRUCTION_OUTCOME_FIELD_DETAIL": textType}
	switch {
	case kind == "invokeRpc" || kind == "completeNexusOperation":
		fields["INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE"] = textType
	case !graph.controller:
		fields["INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE"] = textType
	default:
		// Other controller instructions have neither code.
	}
	if kind == "awaitInstruction" {
		fields["INSTRUCTION_OUTCOME_FIELD_VALUE"] = textType
	}
	return fields
}

// dropOutcome checks every declared outcome field is one preparation derives, with the derived type.
// A Finish or RespondNexus value ends its activation, so preparation derives none; a declaration of
// one is dropped only when no expression of the Program reads it.
func dropOutcome(program *Object, graph instructionGraph, node *Object) error {
	value, declared := node.Fields["outcome"]
	if !declared {
		return nil
	}
	outcome, ok := value.(*Object)
	if !ok {
		return errors.New("outcome is not an object")
	}
	kind := instructionKind(node)
	derived := derivedOutcomeTypes(graph, kind)
	fields, _ := outcome.Fields["fields"].([]any)
	for index, element := range fields {
		declaration, isObject := element.(*Object)
		if !isObject {
			return fmt.Errorf("outcome field %d is not an object", index)
		}
		name := literalText(declaration.Fields["field"])
		if spelled, numbered := outcomeFieldNames[name]; numbered {
			name = spelled
		}
		written, err := json.Marshal(declaration.Fields["type"])
		if err != nil {
			return err
		}
		if want, derives := derived[name]; derives {
			if !sameJSON(written, want) {
				return fmt.Errorf("outcome field %s has type %s, not the derived %s", name, written, want)
			}
			continue
		}
		terminal := kind == "finish" || kind == "respondNexus"
		if name != "INSTRUCTION_OUTCOME_FIELD_VALUE" || !terminal {
			return fmt.Errorf("outcome field %s is not derived for %s", name, kind)
		}
		if readsOutcome(program, graph.id, literalText(node.Fields["instructionId"]), name) {
			return fmt.Errorf("the %s value an expression reads is not derived", kind)
		}
	}
	delete(node.Fields, "outcome")
	return nil
}

// readsOutcome reports whether any expression of the Program reads field of the instruction.
func readsOutcome(value any, entrypointID, instructionID, field string) bool {
	switch node := value.(type) {
	case *Object:
		if reference, ok := node.Fields["outcome"].(*Object); ok {
			if instruction, isReference := reference.Fields["instruction"].(*Object); isReference {
				name := literalText(reference.Fields["field"])
				if spelled, numbered := outcomeFieldNames[name]; numbered {
					name = spelled
				}
				if literalText(instruction.Fields["entrypointId"]) == entrypointID && literalText(instruction.Fields["instructionId"]) == instructionID && name == field {
					return true
				}
			}
		}
		for _, child := range node.Fields {
			if readsOutcome(child, entrypointID, instructionID, field) {
				return true
			}
		}
	case []any:
		for _, element := range node {
			if readsOutcome(element, entrypointID, instructionID, field) {
				return true
			}
		}
	default:
	}
	return false
}

// dropDefaultLimits drops each instruction limit equal to the Profile's default, and the limits when
// none remains; a limit with another value stays.
func dropDefaultLimits(node *Object, timeoutMilliseconds, maxAttempts int64) error {
	limits, ok := node.Fields["limits"].(*Object)
	if !ok {
		return nil
	}
	for field, fallback := range map[string]int64{"timeoutMilliseconds": timeoutMilliseconds, "maxAttempts": maxAttempts} {
		value, written := limits.Fields[field]
		if !written {
			continue
		}
		bound, err := strconv.ParseInt(literalText(value), 10, 64)
		if err != nil {
			return fmt.Errorf("limits.%s is %v, not an integer", field, value)
		}
		if bound == fallback {
			delete(limits.Fields, field)
		}
	}
	if len(limits.Fields) == 0 {
		delete(node.Fields, "limits")
	}
	return nil
}

func sameJSON(encoded []byte, want string) bool {
	var left, right any
	if json.Unmarshal(encoded, &left) != nil || json.Unmarshal([]byte(want), &right) != nil {
		return false
	}
	leftEncoded, leftErr := json.Marshal(left)
	rightEncoded, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftEncoded) == string(rightEncoded)
}
