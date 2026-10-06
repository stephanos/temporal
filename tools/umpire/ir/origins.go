package ir

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// WithoutOrigins is the Model as an identity reads it: with no Property's origin, which only traces a
// generated Property to the capability Property it was expanded from, so no fingerprint, answer,
// lowering or exploration identity changes with it.
func WithoutOrigins(m *umpirespb.Model) *umpirespb.Model {
	out := proto.CloneOf(m)
	for _, p := range out.GetProperties() {
		p.Origin = nil
	}
	return out
}
