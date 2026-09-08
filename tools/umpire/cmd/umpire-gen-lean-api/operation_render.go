package main

import (
	"fmt"
	"slices"
	"strings"
)

func renderOperationBindings(generated *strings.Builder, plan leanPlan) {
	if plan.OperationSchemas == nil || len(plan.Services) == 0 {
		return
	}
	methods := []leanMethodPlan{}
	for _, service := range plan.Services {
		methods = append(methods, service.Methods...)
	}
	if len(methods) == 0 {
		return
	}
	root := strings.TrimSuffix(plan.supportNamespace, ".Proto")
	fmt.Fprintf(generated, "namespace %s\n\n", root)
	schemaNames := make(map[string]string)
	for index, node := range plan.OperationSchemas.Nodes {
		name := fmt.Sprintf("schemaNode%d", index)
		schemaNames[node.Name] = name
		fmt.Fprintf(generated, "private def %s : Umpire.Operation.SchemaNode := {\n  name := %q\n  protoSyntax := %q\n  descriptor := %q\n  fileContext := %q\n  references := %s\n  valueShape := some %s\n}\n\n", name, node.Name, node.Syntax, node.Descriptor, node.FileContext, leanStrings(node.References), node.ValueShape)
	}
	generated.WriteString("/-- Complete descriptor input closure, including option and imported schemas. -/\ndef schemaInputs : List Umpire.Operation.SchemaNode := [\n")
	for i, node := range plan.OperationSchemas.Inputs {
		if i != 0 {
			generated.WriteString(",\n")
		}
		fmt.Fprintf(generated, "  ⟨%q, %q, %q, %q, %s, none⟩", node.Name, node.Syntax, node.Descriptor, node.FileContext, leanStrings(node.References))
	}
	generated.WriteString("\n]\n\n")
	slices.SortFunc(methods, func(a, b leanMethodPlan) int { return strings.Compare(a.FullName, b.FullName) })
	for index, method := range methods {
		fmt.Fprintf(generated, "private def methodSchema%d : Umpire.Operation.RpcSchema := {\n  fullName := %q\n", index, method.FullName)
		for _, side := range []struct{ name, root string }{{"request", method.InputSchemaName}, {"response", method.OutputSchemaName}} {
			names := []string{}
			for _, n := range plan.OperationSchemas.Closures[side.root] {
				names = append(names, schemaNames[n])
			}
			fmt.Fprintf(generated, "  %s := ⟨%q, [%s]⟩\n", side.name, side.root, strings.Join(names, ", "))
		}
		fmt.Fprintf(generated, "  schemaInputs := schemaInputs\n  clientStreaming := %t\n  serverStreaming := %t\n}\n\n", method.ClientStreaming, method.ServerStreaming)
	}
	fmt.Fprintf(generated, "/-- Only exact generated methods inhabit this indexed structural provenance witness. -/\ninductive MethodReference : {Request Response : Type} → %s.Method Request Response → Type where\n", plan.supportNamespace)
	for index, method := range methods {
		fmt.Fprintf(generated, "  | method%d : MethodReference %s\n", index, method.QualifiedName.String())
	}
	generated.WriteString("\n/-- The witness selects full request/response closure and streaming metadata, never caller strings. -/\ndef MethodReference.schema {method : " + plan.supportNamespace + ".Method Request Response} :\n    MethodReference method → Umpire.Operation.RpcSchema\n")
	for index := range methods {
		fmt.Fprintf(generated, "  | .method%d => methodSchema%d\n", index, index)
	}
	fmt.Fprintf(generated, `
/-- Every generated witness preserves the referenced method identity. -/
theorem MethodReference.fullName_eq {method : %s.Method Request Response}
    (reference : MethodReference method) : reference.schema.fullName = method.fullName := by
  cases reference <;> rfl
`, plan.supportNamespace)
	fmt.Fprintf(generated, `
/-- This owner's payload-indexed witnesses contain an exact generated method and its reference. -/
def rpcOwner : Umpire.Operation.RpcOwner where
  Witness Request Response := (method : %s.Method Request Response) × MethodReference method
  schema reference := reference.2.schema

/-- Bind a generated method directly; an explicit candidate is checked against its structural owner. -/
def bindUnary (method : %s.Method Request Response)
    (reference : MethodReference method := by constructor)
    (candidate : Umpire.Operation.RpcSchema := reference.schema) :
    Except Umpire.Operation.Error (Umpire.Operation.CheckedRpc rpcOwner ⟨method, reference⟩) :=
  Umpire.Operation.checkRpc rpcOwner ⟨method, reference⟩ candidate

/-- All typed structural fields in this method's complete selected payload schema. -/
def fields (method : %s.Method Request Response) (side : Umpire.Value.Side)
    (reference : MethodReference method := by constructor) :
    List ((containing : String) × Umpire.Value.Field.Reference rpcOwner ⟨method, reference⟩ side containing) :=
  Umpire.Value.Field.references rpcOwner ⟨method, reference⟩ side

/-- Select a field by containing schema and number, retaining the generated method witness. -/
def fieldReference (method : %s.Method Request Response) (side : Umpire.Value.Side)
    (containing : String) (number : Nat) (source : Umpire.SourceLocation)
    (reference : MethodReference method := by constructor) :
    Except Umpire.Value.Field.Error (Umpire.Value.Field.Reference rpcOwner ⟨method, reference⟩ side containing) :=
  Umpire.Value.Field.reference rpcOwner ⟨method, reference⟩ side containing number source

end %s

`, plan.supportNamespace, plan.supportNamespace, plan.supportNamespace, plan.supportNamespace, root)
}

func leanStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
