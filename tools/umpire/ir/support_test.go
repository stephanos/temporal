package ir

// What only the reader's tests ask of its results: no consumer reads a report or a machine this way.

import umpirespb "go.temporal.io/server/api/umpire/v1"

// queryTotal is the static combination count of one Query of a Model Validate admits.
func queryTotal(m *umpirespb.Model, q *umpirespb.Query) (Total, error) {
	return newValidator(m).total(m, q)
}
