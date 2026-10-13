module go.temporal.io/server/tools/gomad3

go 1.27.1

require (
	github.com/go-json-experiment/json v0.0.0-20260820222146-c27c302e5fc3
	github.com/stretchr/testify v1.11.1
	go.temporal.io/server v0.0.0
	golang.org/x/mod v0.41.0
	golang.org/x/sys v0.48.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.temporal.io/server => ../..
