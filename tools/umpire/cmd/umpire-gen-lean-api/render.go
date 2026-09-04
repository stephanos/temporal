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
  deriving DecidableEq, Repr

`)
	fmt.Fprintf(&generated, "end %s\n", plan.supportNamespace)
	return []byte(generated.String())
}

func renderTypes(plan leanPlan) []byte {
	var generated strings.Builder
	writeModuleHeader(&generated, plan.TypesModule, apiTypesModuleDoc)
	generated.WriteString("set_option linter.extra.dupNamespace false\n\n")
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
					fmt.Fprintf(&generated, "  | %s (value : %s)\n",
						constructor.Field.Name, renderLeanType(constructor.Field.BaseType))
				}
				generated.WriteString("  deriving Repr\n\n")
			}
			fmt.Fprintf(&generated, "structure %s where\n", message.RelativeName)
			for _, field := range message.StructureFields {
				fmt.Fprintf(&generated, "  %s : %s\n", field.Name, renderLeanType(field.Type))
			}
			if len(message.StructureFields) == 0 {
				generated.WriteString("  unit : Unit := ()\n")
			}
			generated.WriteString("  deriving Repr\n\n")
		}
		fmt.Fprintf(&generated, "end %s\n\n", namespace.Name.String())
	}
	return []byte(strings.TrimRight(generated.String(), "\n") + "\n")
}

func renderAPI(plan leanPlan) []byte {
	var generated strings.Builder
	writeModuleHeader(&generated, plan.APIModule, apiFacadeModuleDoc)
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
