//go:build !gomad

package sql

// The stock SQLite store lists executions with equal close and start times by
// run_id ascending and pages past a run_id with >.
const (
	sqliteRunIDOrder        = ""
	sqliteRunIDPageOperator = ">"
)
