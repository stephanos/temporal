module go.temporal.io/umpire/model

go 1.24

require (
	github.com/stretchr/testify v1.10.0
	go.temporal.io/api v1.52.0
	google.golang.org/protobuf v1.36.6
)

// The two linters that stand in for the exhaustiveness Lean gets from `match`. Both also ship in
// golangci-lint (`exhaustive`, `gochecksumtype`), which the server's `make lint-code` already runs.
tool (
	github.com/alecthomas/go-check-sumtype/cmd/go-check-sumtype
	github.com/nishanths/exhaustive/cmd/exhaustive
)
