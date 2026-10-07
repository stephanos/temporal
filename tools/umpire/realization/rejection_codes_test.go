package realization

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

type rejectionAdmitter struct{ problems []string }

func (a *rejectionAdmitter) Once(*umpirespb.Position, ...string) bool { return false }
func (a *rejectionAdmitter) Report(_ *umpirespb.Position, format string, args ...any) {
	a.problems = append(a.problems, fmt.Sprintf(format, args...))
}
func (a *rejectionAdmitter) Errors() int { return len(a.problems) }
func (a *rejectionAdmitter) ActionClass(string, *umpirespb.Machine, *umpirespb.ActionClass, *umpirespb.Position) {
}
func (a *rejectionAdmitter) ClassKey(*umpirespb.ActionClass) string { return "" }
func (a *rejectionAdmitter) Machine(string) (*umpirespb.Machine, bool) {
	return &umpirespb.Machine{}, true
}
func (a *rejectionAdmitter) Channel(string) bool { return true }

func validRejectionRealization() *umpirespb.Realization {
	return &umpirespb.Realization{
		Id: "test.realization", Name: "test", Machine: "machine", Producer: "producer",
		Observations: []*umpirespb.Observed{{Id: "observation", Message: "test.Message"}},
		Correlation: &umpirespb.Correlation{
			Projection: "projection", Run: "run", Operation: "operation", Observation: "observation",
			Events: 1, Buffered: 1, Keys: 1, Support: 1, Work: 1, EventSize: 1,
		},
		RejectionCodes: []*umpirespb.RejectionCode{
			{Rejection: umpirespb.RejectionCode_REJECTION_NOT_FOUND, GrpcCode: "NOT_FOUND"},
			{Rejection: umpirespb.RejectionCode_REJECTION_ALREADY_EXISTS, GrpcCode: "ALREADY_EXISTS"},
			{Rejection: umpirespb.RejectionCode_REJECTION_FAILED_PRECONDITION, GrpcCode: "FAILED_PRECONDITION"},
			{Rejection: umpirespb.RejectionCode_REJECTION_INVALID_ARGUMENT, GrpcCode: "INVALID_ARGUMENT"},
		},
	}
}

func TestRejectionCodesAreReadAndAdmittedAsAUniqueCompleteTable(t *testing.T) {
	r := validRejectionRealization()
	require.Equal(t, map[umpirespb.RejectionCode_Rejection]string{
		umpirespb.RejectionCode_REJECTION_NOT_FOUND:           "NOT_FOUND",
		umpirespb.RejectionCode_REJECTION_ALREADY_EXISTS:      "ALREADY_EXISTS",
		umpirespb.RejectionCode_REJECTION_FAILED_PRECONDITION: "FAILED_PRECONDITION",
		umpirespb.RejectionCode_REJECTION_INVALID_ARGUMENT:    "INVALID_ARGUMENT",
	}, RejectionCodes(r))
	valid := &rejectionAdmitter{}
	(&realizing{d: valid, r: r}).rejectionCodes()
	require.Empty(t, valid.problems)

	for name, mutate := range map[string]func(*umpirespb.Realization){
		"unspecified": func(r *umpirespb.Realization) {
			r.RejectionCodes[0].Rejection = umpirespb.RejectionCode_REJECTION_UNSPECIFIED
		},
		"unknown":    func(r *umpirespb.Realization) { r.RejectionCodes[0].Rejection = umpirespb.RejectionCode_Rejection(99) },
		"duplicate":  func(r *umpirespb.Realization) { r.RejectionCodes[1].Rejection = r.RejectionCodes[0].Rejection },
		"missing":    func(r *umpirespb.Realization) { r.RejectionCodes = r.RejectionCodes[:len(r.RejectionCodes)-1] },
		"empty code": func(r *umpirespb.Realization) { r.RejectionCodes[0].GrpcCode = "" },
	} {
		t.Run(name, func(t *testing.T) {
			r := validRejectionRealization()
			mutate(r)
			a := &rejectionAdmitter{}
			(&realizing{d: a, r: r}).rejectionCodes()
			require.NotEmpty(t, a.problems)
			require.Contains(t, strings.Join(a.problems, "\n"), "rejection")
		})
	}
}
