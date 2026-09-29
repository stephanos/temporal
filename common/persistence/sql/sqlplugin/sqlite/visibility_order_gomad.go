//go:build gomad

package sqlite

// Under Gomad every start in a test shares one virtual instant, so the order
// among equal times is the order clients observe. Run IDs are UUIDv7: a larger
// run ID is the later start, and descending keeps newest-first, as
// Elasticsearch orders. The stock store keeps its ascending tie order.
const (
	visibilityRunIDOrder        = " DESC"
	visibilityRunIDPageOperator = "<"
)
