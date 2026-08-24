package api

import "fmt"

type schemaEnumValue struct {
	FullName   string `json:"fullName"`
	Name       string `json:"name"`
	LeanName   string `json:"leanName"`
	Number     int32  `json:"number"`
	Deprecated bool   `json:"deprecated"`
}

type schemaEnum struct {
	FullName     string            `json:"fullName"`
	Name         string            `json:"name"`
	Package      string            `json:"package"`
	Parent       string            `json:"parent,omitempty"`
	LeanName     string            `json:"leanName"`
	Source       sourceKind        `json:"source"`
	Values       []schemaEnumValue `json:"values"`
	AllowAliases bool              `json:"allowAliases"`
	Deprecated   bool              `json:"deprecated"`
}

type schemaField struct {
	FullName   string `json:"fullName"`
	Name       string `json:"name"`
	JSONName   string `json:"jsonName"`
	LeanName   string `json:"leanName"`
	Number     int32  `json:"number"`
	Kind       string `json:"kind"`
	TypeName   string `json:"typeName,omitempty"`
	MapKey     string `json:"mapKey,omitempty"`
	MapValue   string `json:"mapValue,omitempty"`
	Presence   bool   `json:"presence"`
	Required   bool   `json:"required"`
	HasDefault bool   `json:"hasDefault"`
	Default    string `json:"defaultValue,omitempty"`
	Oneof      string `json:"oneof,omitempty"`
	Repeated   bool   `json:"repeated"`
	Map        bool   `json:"map"`
	Packed     bool   `json:"packed"`
	Recursive  bool   `json:"recursive"`
	Deprecated bool   `json:"deprecated"`
}

type schemaOneof struct {
	FullName   string   `json:"fullName"`
	Name       string   `json:"name"`
	LeanName   string   `json:"leanName"`
	FieldNames []string `json:"fieldNames"`
}

type schemaMessage struct {
	FullName   string        `json:"fullName"`
	Name       string        `json:"name"`
	Package    string        `json:"package"`
	Parent     string        `json:"parent,omitempty"`
	LeanName   string        `json:"leanName"`
	Source     sourceKind    `json:"source"`
	Fields     []schemaField `json:"fields"`
	Oneofs     []schemaOneof `json:"oneofs"`
	Deprecated bool          `json:"deprecated"`
}

type schemaMethod struct {
	FullName        string `json:"fullName"`
	Name            string `json:"name"`
	LeanName        string `json:"leanName"`
	InputType       string `json:"inputType"`
	InputLeanType   string `json:"inputLeanType"`
	OutputType      string `json:"outputType"`
	OutputLeanType  string `json:"outputLeanType"`
	ClientStreaming bool   `json:"clientStreaming"`
	ServerStreaming bool   `json:"serverStreaming"`
	Deprecated      bool   `json:"deprecated"`
}

type schemaService struct {
	FullName   string         `json:"fullName"`
	Name       string         `json:"name"`
	Package    string         `json:"package"`
	LeanName   string         `json:"leanName"`
	Source     sourceKind     `json:"source"`
	Methods    []schemaMethod `json:"methods"`
	Deprecated bool           `json:"deprecated"`
}

type schemaProjection struct {
	DescriptorDigest string           `json:"descriptorDigest"`
	Files            []fileProjection `json:"files"`
	Enums            []schemaEnum     `json:"enums"`
	Messages         []schemaMessage  `json:"messages"`
	Services         []schemaService  `json:"services"`
}

func buildSchemaProjection(projection projection, plan leanPlan) (schemaProjection, error) {
	result := schemaProjection{
		DescriptorDigest: projection.DescriptorDigest,
		Files:            projection.Files,
		Enums:            []schemaEnum{},
		Messages:         []schemaMessage{},
		Services:         []schemaService{},
	}
	enums := make(map[string]leanEnumPlan, len(plan.Enums))
	for _, enum := range plan.Enums {
		enums[enum.Projection.FullName] = enum
	}
	for _, enum := range projection.Enums {
		planned, exists := enums[enum.FullName]
		if !exists {
			return schemaProjection{}, fmt.Errorf("build schema: enum %q is absent from Lean plan", enum.FullName)
		}
		item := schemaEnum{
			FullName: enum.FullName, Name: enum.Name, Package: enum.Package, Parent: enum.Parent,
			LeanName: planned.Name.String(), Source: enum.Source, Values: []schemaEnumValue{},
			AllowAliases: enum.AllowAliases, Deprecated: enum.Deprecated,
		}
		for _, value := range planned.Values {
			item.Values = append(item.Values, schemaEnumValue{
				FullName: value.Projection.FullName, Name: value.Projection.Name,
				LeanName: value.Name, Number: value.Projection.Number,
				Deprecated: value.Projection.Deprecated,
			})
		}
		result.Enums = append(result.Enums, item)
	}
	messages := make(map[string]leanMessagePlan, len(plan.Messages))
	for _, message := range plan.Messages {
		messages[message.Projection.FullName] = message
	}
	for _, message := range projection.Messages {
		planned, exists := messages[message.FullName]
		if !exists {
			return schemaProjection{}, fmt.Errorf("build schema: message %q is absent from Lean plan", message.FullName)
		}
		item := schemaMessage{
			FullName: message.FullName, Name: message.Name, Package: message.Package, Parent: message.Parent,
			LeanName: planned.Name.String(), Source: message.Source,
			Fields: []schemaField{}, Oneofs: []schemaOneof{}, Deprecated: message.Deprecated,
		}
		for _, field := range message.Fields {
			plannedField, exists := plan.fields[field.FullName]
			if !exists {
				return schemaProjection{}, fmt.Errorf("build schema: field %q is absent from Lean plan", field.FullName)
			}
			item.Fields = append(item.Fields, schemaField{
				FullName: field.FullName, Name: field.Name, JSONName: field.JSONName, LeanName: plannedField.Name,
				Number: field.Number, Kind: field.Kind, TypeName: field.TypeName, MapKey: field.MapKey,
				MapValue: field.MapValue, Presence: field.Presence, Required: field.Required,
				HasDefault: field.HasDefault, Default: field.Default, Oneof: field.Oneof,
				Repeated: field.Repeated, Map: field.Map, Packed: field.Packed,
				Recursive: plannedField.Recursive, Deprecated: field.Deprecated,
			})
		}
		for _, oneof := range message.Oneofs {
			plannedOneof, exists := plan.oneofs[oneof.FullName]
			if !exists {
				return schemaProjection{}, fmt.Errorf("build schema: oneof %q is absent from Lean plan", oneof.FullName)
			}
			item.Oneofs = append(item.Oneofs, schemaOneof{
				FullName: oneof.FullName, Name: oneof.Name, LeanName: plannedOneof.Name.String(),
				FieldNames: oneof.FieldNames,
			})
		}
		result.Messages = append(result.Messages, item)
	}
	for _, service := range projection.Services {
		planned, exists := plan.services[service.FullName]
		if !exists {
			return schemaProjection{}, fmt.Errorf("build schema: service %q is absent from Lean plan", service.FullName)
		}
		item := schemaService{
			FullName: service.FullName, Name: service.Name, Package: service.Package,
			LeanName: planned.Name.String(), Source: service.Source,
			Methods: []schemaMethod{}, Deprecated: service.Deprecated,
		}
		for _, method := range planned.Methods {
			item.Methods = append(item.Methods, schemaMethod{
				FullName: method.Projection.FullName, Name: method.Projection.Name, LeanName: method.Name,
				InputType: method.Projection.InputType, InputLeanType: renderLeanType(method.InputType),
				OutputType: method.Projection.OutputType, OutputLeanType: renderLeanType(method.OutputType),
				ClientStreaming: method.Projection.ClientStreaming,
				ServerStreaming: method.Projection.ServerStreaming,
				Deprecated:      method.Projection.Deprecated,
			})
		}
		result.Services = append(result.Services, item)
	}
	return result, nil
}
