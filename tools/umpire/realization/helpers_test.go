package realization

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func admWritten(kind any) *umpirespb.Operand {
	value := &umpirespb.ProtoValue{}
	switch k := kind.(type) {
	case *umpirespb.ProtoValue_Text:
		value.Kind = k
	case *umpirespb.ProtoValue_Number:
		value.Kind = k
	default:
	}
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: value}}
}

// admPayload reads a path of the value a poll is looking at, or of a Run Event's payload.
func admPayload(path string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: path,
		Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &emptypb.Empty{}}}}}}
}

func admGreater(left, right *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Greater{Greater: &umpirespb.Greater{Left: left, Right: right}}}
}

func admNot(of *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Not{Not: &umpirespb.Not{Of: of}}}
}
