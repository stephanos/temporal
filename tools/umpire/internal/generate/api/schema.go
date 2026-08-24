package api

import "fmt"

type schemaEnumValue struct {
	enumValueProjection
	LeanName string `json:"leanName"`
}

type schemaEnum struct {
	enumProjection
	LeanName string            `json:"leanName"`
	Values   []schemaEnumValue `json:"values"`
}

type schemaField struct {
	fieldProjection
	LeanName  string `json:"leanName"`
	Recursive bool   `json:"recursive"`
}

type schemaOneof struct {
	oneofProjection
	LeanName string `json:"leanName"`
}

type schemaMessage struct {
	messageProjection
	LeanName string        `json:"leanName"`
	Fields   []schemaField `json:"fields"`
	Oneofs   []schemaOneof `json:"oneofs"`
}

type schemaMethod struct {
	methodProjection
	LeanName       string `json:"leanName"`
	InputLeanType  string `json:"inputLeanType"`
	OutputLeanType string `json:"outputLeanType"`
}

type schemaService struct {
	serviceProjection
	LeanName string         `json:"leanName"`
	Methods  []schemaMethod `json:"methods"`
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
		Files:            append([]fileProjection{}, projection.Files...),
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
			enumProjection: enum,
			LeanName:       planned.Name.String(),
			Values:         []schemaEnumValue{},
		}
		for _, value := range planned.Values {
			item.Values = append(item.Values, schemaEnumValue{
				enumValueProjection: value.Projection,
				LeanName:            value.QualifiedName.String(),
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
			messageProjection: message,
			LeanName:          planned.Name.String(),
			Fields:            []schemaField{},
			Oneofs:            []schemaOneof{},
		}
		for _, field := range message.Fields {
			plannedField, exists := plan.fields[field.FullName]
			if !exists {
				return schemaProjection{}, fmt.Errorf("build schema: field %q is absent from Lean plan", field.FullName)
			}
			item.Fields = append(item.Fields, schemaField{
				fieldProjection: field,
				LeanName:        plannedField.QualifiedName.String(),
				Recursive:       plannedField.Recursive,
			})
		}
		for _, oneof := range message.Oneofs {
			plannedOneof, exists := plan.oneofs[oneof.FullName]
			if !exists {
				return schemaProjection{}, fmt.Errorf("build schema: oneof %q is absent from Lean plan", oneof.FullName)
			}
			item.Oneofs = append(item.Oneofs, schemaOneof{
				oneofProjection: oneof,
				LeanName:        plannedOneof.Name.String(),
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
			serviceProjection: service,
			LeanName:          planned.Name.String(),
			Methods:           []schemaMethod{},
		}
		for _, method := range planned.Methods {
			item.Methods = append(item.Methods, schemaMethod{
				methodProjection: method.Projection,
				LeanName:         method.QualifiedName.String(),
				InputLeanType:    renderLeanType(method.InputType),
				OutputLeanType:   renderLeanType(method.OutputType),
			})
		}
		result.Services = append(result.Services, item)
	}
	return result, nil
}
