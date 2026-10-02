package main

import (
	"fmt"
	"strings"
)

const (
	apiFacadeModuleDoc = "/-!\n" +
		"Generated gRPC method descriptors projected from the source Protobuf API.\n\n" +
		"Each service namespace contains explicitly typed method descriptor values. These declarations describe\n" +
		"transport structure only; handwritten model modules assign behavioral meaning.\n" +
		"-/"
	apiProtoModuleDoc = "/-!\n" +
		"Common structural types used by the generated Temporal API projection.\n\n" +
		"`Bytes` and `MessageRef` retain opaque descriptor data, while `Method` records the request,\n" +
		"response, streaming, and deprecation shape of one gRPC method.\n" +
		"-/"
	apiTypesModuleDoc = "/-!\n" +
		"Generated Lean representations of source Protobuf messages, enumerations, and oneofs.\n\n" +
		"The declarations preserve descriptor structure for handwritten consumers. Recursive Protobuf\n" +
		"references remain explicit through `MessageRef` and carry no behavioral meaning.\n" +
		"-/"
)

func renderArtifacts(plan leanPlan) map[string][]byte {
	return map[string][]byte{
		plan.ProtoModule.Path: renderProto(plan),
		plan.TypesModule.Path: renderTypes(plan),
		plan.APIModule.Path:   renderAPI(plan),
	}
}

func renderProto(plan leanPlan) []byte {
	var generated strings.Builder
	writeGeneratedHeader(&generated)
	writeModuleDoc(&generated, apiProtoModuleDoc)
	generated.WriteString("set_option linter.missingDocs false\n")
	fmt.Fprintf(&generated, "\nnamespace %s\n\n", plan.supportNamespace)
	generated.WriteString(`structure Bytes where
  digest : String
  size : Nat
  deriving DecidableEq, Repr

structure MessageRef where
  descriptor : String
  remainingDepth : Nat
  deriving DecidableEq, Repr

structure Method (Request Response : Type) where
  fullName : String
  clientStreaming : Bool
  serverStreaming : Bool
  deprecated : Bool

-- Written out because deriving would require instances of the phantom payload types.
instance : DecidableEq (Method Request Response)
  | ⟨fullName, clientStreaming, serverStreaming, deprecated⟩,
    ⟨fullName', clientStreaming', serverStreaming', deprecated'⟩ =>
    decidable_of_iff (fullName = fullName' ∧ clientStreaming = clientStreaming' ∧
      serverStreaming = serverStreaming' ∧ deprecated = deprecated') (by simp)

instance : Repr (Method Request Response) where
  reprPrec method _ := Std.Format.bracket "{ "
    (f!"fullName := {repr method.fullName}," ++ Std.Format.line ++
      f!"clientStreaming := {method.clientStreaming}," ++ Std.Format.line ++
      f!"serverStreaming := {method.serverStreaming}," ++ Std.Format.line ++
      f!"deprecated := {method.deprecated}") " }"

`)
	fmt.Fprintf(&generated, "end %s\n", plan.supportNamespace)
	return []byte(generated.String())
}

func renderTypes(plan leanPlan) []byte {
	var generated strings.Builder
	writeModuleHeader(&generated, plan.TypesModule, apiTypesModuleDoc)
	generated.WriteString("set_option linter.extra.dupNamespace false\n\n")
	fieldless := fieldlessMessages(plan)
	for _, namespace := range plan.Namespaces {
		fmt.Fprintf(&generated, "namespace %s\n\n", namespace.Name.String())
		for _, enum := range namespace.Enums {
			fmt.Fprintf(&generated, "structure %s where\n  number : Int\n  deriving DecidableEq, Repr\n\n", enum.RelativeName)
			fmt.Fprintf(&generated, "namespace %s\n", enum.RelativeName)
			for _, value := range enum.Values {
				fmt.Fprintf(&generated, "def %s : %s := { number := %d }\n",
					value.Name, enum.RelativeName, value.Number)
			}
			fmt.Fprintf(&generated, "end %s\n\n", enum.RelativeName)
		}
		for _, message := range namespace.Messages {
			for _, oneof := range message.Oneofs {
				fmt.Fprintf(&generated, "inductive %s where\n  | notSet\n", oneof.RelativeName)
				for _, constructor := range oneof.Constructors {
					// A fieldless payload carries no data, and its constructor injectivity lemma is
					// one simp already proves by structure eta, which simpNF rejects as redundant.
					if !constructor.Field.Recursive && fieldless[constructor.Field.Projection.TypeName] {
						fmt.Fprintf(&generated, "  | %s\n", constructor.Field.Name)
						continue
					}
					fmt.Fprintf(&generated, "  | %s (value : %s)\n",
						constructor.Field.Name, renderLeanType(constructor.Field.BaseType))
				}
				generated.WriteString("  deriving Repr\n\n")
			}
			fmt.Fprintf(&generated, "structure %s where\n", message.RelativeName)
			for _, field := range message.StructureFields {
				fmt.Fprintf(&generated, "  %s : %s\n", field.Name, renderLeanType(field.Type))
			}
			generated.WriteString("  deriving Repr\n\n")
		}
		fmt.Fprintf(&generated, "end %s\n\n", namespace.Name.String())
	}
	return []byte(strings.TrimRight(generated.String(), "\n") + "\n")
}

func fieldlessMessages(plan leanPlan) map[string]bool {
	result := make(map[string]bool)
	for _, namespace := range plan.Namespaces {
		for _, message := range namespace.Messages {
			if len(message.StructureFields) == 0 {
				result[message.Projection.FullName] = true
			}
		}
	}
	return result
}

func renderAPI(plan leanPlan) []byte {
	var generated strings.Builder
	module := cloneLeanModulePlan(plan.APIModule)
	if plan.OperationSchemas != nil && len(plan.Services) > 0 {
		module.Imports = append(module.Imports, "Umpire.Operation", "Umpire.Value.Field")
	}
	writeModuleHeader(&generated, module, apiFacadeModuleDoc)
	for _, service := range plan.Services {
		fmt.Fprintf(&generated, "namespace %s\n", service.Name.String())
		for _, method := range service.Methods {
			fmt.Fprintf(&generated, "def %s : %s.Method %s %s :=\n",
				method.Name, plan.supportNamespace, renderLeanType(method.InputType), renderLeanType(method.OutputType))
			fmt.Fprintf(&generated, "  { fullName := %q, clientStreaming := %t, serverStreaming := %t, deprecated := %t }\n",
				method.FullName, method.ClientStreaming, method.ServerStreaming, method.Deprecated)
		}
		fmt.Fprintf(&generated, "end %s\n\n", service.Name.String())
	}
	renderOperationBindings(&generated, plan)
	return []byte(strings.TrimRight(generated.String(), "\n") + "\n")
}

func renderLeanType(value leanType) string {
	switch value.Kind {
	case leanTypeNamed:
		return value.Name
	case leanTypeOption, leanTypeList:
		argument := renderLeanType(value.Arguments[0])
		if value.Arguments[0].Kind != leanTypeNamed {
			argument = "(" + argument + ")"
		}
		constructor := "Option "
		if value.Kind == leanTypeList {
			constructor = "List "
		}
		return constructor + argument
	case leanTypeProduct:
		return renderLeanType(value.Arguments[0]) + " × " + renderLeanType(value.Arguments[1])
	default:
		return ""
	}
}

func writeGeneratedHeader(generated *strings.Builder) {
	generated.WriteString("-- Code generated by umpire-gen-lean-api. DO NOT EDIT.\n")
	generated.WriteString("-- This is a structural descriptor projection, not behavioral semantics.\n")
}

func writeModuleHeader(generated *strings.Builder, module leanModulePlan, moduleDoc string) {
	writeGeneratedHeader(generated)
	for _, imported := range module.Imports {
		fmt.Fprintf(generated, "import %s\n", imported)
	}
	writeModuleDoc(generated, moduleDoc)
	generated.WriteString("set_option linter.missingDocs false\n")
	generated.WriteString("set_option maxRecDepth 100000\n\n")
}

func writeModuleDoc(generated *strings.Builder, moduleDoc string) {
	fmt.Fprintf(generated, "\n%s\n\n", moduleDoc)
}
