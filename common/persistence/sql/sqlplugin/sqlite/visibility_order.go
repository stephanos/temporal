//go:build !gomad

package sqlite

// The stock SQLite store lists executions with equal close and start times by
// run_id ascending and pages past a run_id with >.
const (
	visibilityRunIDOrder        = ""
	visibilityRunIDPageOperator = ">"
)
