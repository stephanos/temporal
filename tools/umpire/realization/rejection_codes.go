package realization

import (
	"sort"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// RejectionCodes returns the rejection-to-gRPC-code table declared by a realization. When a
// realization declares this optional projection, Admit checks that it is a complete function over
// the generated Rejection enum.
func RejectionCodes(r *umpirespb.Realization) map[umpirespb.RejectionCode_Rejection]string {
	codes := make(map[umpirespb.RejectionCode_Rejection]string, len(r.GetRejectionCodes()))
	for _, entry := range r.GetRejectionCodes() {
		codes[entry.GetRejection()] = entry.GetGrpcCode()
	}
	return codes
}

func knownRejections() []umpirespb.RejectionCode_Rejection {
	values := make([]umpirespb.RejectionCode_Rejection, 0, len(umpirespb.RejectionCode_Rejection_name)-1)
	for number := range umpirespb.RejectionCode_Rejection_name {
		rejection := umpirespb.RejectionCode_Rejection(number)
		if rejection != umpirespb.RejectionCode_REJECTION_UNSPECIFIED {
			values = append(values, rejection)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

func (a *realizing) rejectionCodes() {
	// The framework also lifts generic, non-Temporal realizations whose outcome vocabulary has no
	// shared rejection variants. Temporal's realization kit always supplies the table; an explicit
	// partial table is invalid below.
	if len(a.r.GetRejectionCodes()) == 0 {
		return
	}
	declared := make(map[umpirespb.RejectionCode_Rejection]bool, len(a.r.GetRejectionCodes()))
	for _, entry := range a.r.GetRejectionCodes() {
		rejection := entry.GetRejection()
		_, known := umpirespb.RejectionCode_Rejection_name[int32(rejection)]
		switch {
		case !known:
			a.report(a.r.GetPosition(), "a rejection code names unknown rejection %d", rejection)
		case rejection == umpirespb.RejectionCode_REJECTION_UNSPECIFIED:
			a.report(a.r.GetPosition(), "a rejection code names the unspecified rejection")
		case declared[rejection]:
			a.report(a.r.GetPosition(), "rejection %s has more than one code", rejection)
		default:
			declared[rejection] = true
		}
		if entry.GetGrpcCode() == "" {
			a.report(a.r.GetPosition(), "rejection %s names no gRPC code", rejection)
		}
	}
	for _, rejection := range knownRejections() {
		if !declared[rejection] {
			a.report(a.r.GetPosition(), "rejection %s has no gRPC code", rejection)
		}
	}
}
