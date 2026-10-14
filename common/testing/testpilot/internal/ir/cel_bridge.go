package ir

import (
	celpb "cel.dev/expr"
	legacy "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// CEL-Go's pinned engine still loads google.api.expr. The stored artifact uses cel.expr.
func bridgeCEL(source *celpb.ParsedExpr, limits Limits) (*legacy.ParsedExpr, error) {
	limits.Depth = DefaultLimits().Depth
	limits.Work = DefaultLimits().Work
	if err := CheckSurface(source, limits); err != nil {
		return nil, err
	}
	if _, err := CanonicalCEL(source); err != nil {
		return nil, err
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(source)
	if err != nil {
		return nil, err
	}
	result := &legacy.ParsedExpr{}
	if err = (proto.UnmarshalOptions{RecursionLimit: int(limits.Depth*3 + 8)}).Unmarshal(encoded, result); err != nil {
		return nil, err
	}
	return result, nil
}
