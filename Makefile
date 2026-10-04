############################# Main targets #############################
# Install all tools and builds binaries.
install: bins

# Rebuild binaries (used by Dockerfile).
bins: temporal-server temporal-cassandra-tool temporal-sql-tool temporal-elasticsearch-tool tdbg

# Install all tools, recompile proto files, run all possible checks and tests (long but comprehensive).
all: clean proto bins check test

# Used in CI.
ci-build-misc: \
	print-go-version \
	clean-tools \
	proto \
	go-generate \
	buf-breaking \
	shell-check \
	goimports \
	gomodtidy \
	ensure-no-changes

# Delete all build artifacts
clean: clean-bins clean-tools clean-test-output

# Recompile proto files.
proto: lint-protos lint-api protoc proto-codegen
########################################################################

.PHONY: proto protoc install bins ci-build-misc clean

##### Arguments ######
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
GOPATH      ?= $(shell go env GOPATH)
# Disable cgo by default.
CGO_ENABLED ?= 0

# --- TEMPORARY: pin the Go toolchain that supports generic methods ---
# This branch uses generic methods (Go 1.27+). go.mod already pins `toolchain go1.27.0`,
# so in-module commands (go build/test/vet/fix) use it. But `go install tool@version`
# (gci, goimports, golangci-lint, the vettool, …) runs OUTSIDE this module and would
# otherwise build those tools with the default toolchain, whose gofmt/parser cannot read
# generic methods. Exporting GOTOOLCHAIN forces every make-invoked `go` — tool installs
# included — to use 1.27.0, so `make fmt` / `make fmt-imports` / `make lint-code` work.
# Remove once Go 1.27 is the default toolchain. If tools were already built under an older
# Go, run `make clean-tools` once to rebuild them under 1.27.0.
export GOTOOLCHAIN := go1.27.0

PERSISTENCE_TYPE ?= nosql
PERSISTENCE_DRIVER ?= cassandra

# Optional args to create multiple keyspaces:
# make install-schema TEMPORAL_DB=temporal2 VISIBILITY_DB=temporal_visibility2
TEMPORAL_DB ?= temporal
VISIBILITY_DB ?= temporal_visibility

# The `disable_grpc_modules` build tag excludes gRPC dependencies from cloud.google.com/go/storage,
# reducing binary size since we only use the REST client (storage.NewClient), not the
# gRPC client (storage.NewGRPCClient). Related issue: https://github.com/googleapis/google-cloud-go/issues/12343
ALL_BUILD_TAGS := disable_grpc_modules,$(BUILD_TAG)
ALL_TEST_TAGS := $(ALL_BUILD_TAGS),test_dep,$(TEST_TAG)
BUILD_TAG_FLAG := -tags $(ALL_BUILD_TAGS)
TEST_TAG_FLAG := -tags $(ALL_TEST_TAGS)

# 20 minutes is the upper bound defined for all tests. (Tests in CI take up to about 14:30 now)
# If you change this, also change .github/workflows/run-tests.yml!
# The timeout in the GH workflow must be larger than this to avoid GH timing out the action,
# which causes the a job run to not produce any logs and hurts the debugging experience.
TEST_TIMEOUT ?= 35m

# Number of retries for *-coverage targets.
MAX_TEST_ATTEMPTS ?= 3
TEST_RUNNER_TIMEOUT_ARG := $(if $(TEST_RUNNER_TIMEOUT),--total-timeout=$(TEST_RUNNER_TIMEOUT),)

# Whether or not to test with the race detector. All of (1 on y yes t true) are true values.
TEST_RACE_FLAG ?= on
# Whether or not to shuffle tests. All of (1 on y yes t true) are true values.
TEST_SHUFFLE_FLAG ?= on
# Common test args used in the various test suite targets.
COMPILED_TEST_ARGS := -timeout=$(TEST_TIMEOUT) \
		     $(if $(filter 1 on y yes t true, $(TEST_RACE_FLAG)),-race,) \
		     $(if $(filter 1 on y yes t true, $(TEST_SHUFFLE_FLAG)),-shuffle on,) \
		     $(TEST_PARALLEL_FLAGS) \
		     $(TEST_ARGS) \
		     $(TEST_TAG_FLAG)

##### Variables ######

ROOT := $(shell git rev-parse --show-toplevel)
LOCALBIN := .bin
STAMPDIR := .stamp
export PATH := $(ROOT)/$(LOCALBIN):$(PATH)
GOINSTALL := GOBIN=$(ROOT)/$(LOCALBIN) go install

OTEL ?= false
ifeq ($(OTEL),true)
	export OTEL_BSP_SCHEDULE_DELAY=100 # in ms
	export OTEL_EXPORTER_OTLP_TRACES_INSECURE=true
	export OTEL_TRACES_EXPORTER=otlp
	export TEMPORAL_OTEL_DEBUG=true
	export TEMPORAL_TEST_DATA_ENCODING=json
endif

MODULE_ROOT := $(lastword $(shell grep -e "^module " go.mod))
COLOR := "\e[1;36m%s\e[0m\n"
RED :=   "\e[1;31m%s\e[0m\n"

define NEWLINE


endef

PROTO_ROOT := proto
PROTO_FILES = $(shell find ./$(PROTO_ROOT)/internal -name "*.proto")
CHASM_PROTO_FILES = $(shell find ./chasm/lib -name "*.proto")
PROTO_DIRS = $(sort $(dir $(PROTO_FILES)))
PROTOC ?= protoc
API_BINPB := $(PROTO_ROOT)/api.binpb
# Note: If you change the value of INTERNAL_BINPB, you'll have to add logic to
# develop/buf-breaking.sh to handle the old and new values at once.
INTERNAL_BINPB := $(PROTO_ROOT)/image.bin
CHASM_BINPB := $(PROTO_ROOT)/chasm.bin
PROTO_OUT := api

NESTED_MODULE_DIRS := $(patsubst %/go.mod,%,$(shell git ls-files --cached --others --exclude-standard -- '*/go.mod'))
SOURCE_FIND = find . -type d \( -name .git -o -name .flow -o -name .build $(foreach dir,$(NESTED_MODULE_DIRS),-o -path ./$(dir)) \) -prune -o
ALL_SRC         := $(shell $(SOURCE_FIND) -name '*.go' -print)
ALL_SRC         += go.mod
ALL_SCRIPTS     := $(shell $(SOURCE_FIND) -name '*.sh' -print)

MAIN_BRANCH    := main

# If you update these dirs, please also update in CategoryDirs find_altered_tests.go
TEST_DIRS       := $(sort $(dir $(filter %_test.go,$(ALL_SRC))))
FUNCTIONAL_TEST_ROOT          := ./tests
FUNCTIONAL_TEST_XDC_ROOT      := ./tests/xdc
FUNCTIONAL_TEST_NDC_ROOT      := ./tests/ndc
MIXED_BRAIN_TEST_ROOT         := ./tests/mixedbrain
DB_INTEGRATION_TEST_ROOT      := ./common/persistence/tests
DB_TOOL_INTEGRATION_TEST_ROOT := ./tools/tests
INTEGRATION_TEST_DIRS := $(DB_INTEGRATION_TEST_ROOT) $(DB_TOOL_INTEGRATION_TEST_ROOT) ./temporaltest
TESTCORE_UNITTESTS := ./tests/testcore
ifeq ($(UNIT_TEST_DIRS),)
UNIT_TEST_DIRS := $(filter-out $(FUNCTIONAL_TEST_ROOT)% $(FUNCTIONAL_TEST_XDC_ROOT)% $(FUNCTIONAL_TEST_NDC_ROOT)% $(MIXED_BRAIN_TEST_ROOT)% $(DB_INTEGRATION_TEST_ROOT)% $(DB_TOOL_INTEGRATION_TEST_ROOT)% ./temporaltest%,$(TEST_DIRS))

# Testcore unit tests are filtered out by the FUNCTIONAL_TEST_ROOT pattern, need to add them back manually.
UNIT_TEST_DIRS += $(TESTCORE_UNITTESTS)
endif
SYSTEM_WORKFLOWS_ROOT := ./service/worker

PINNED_DEPENDENCIES := \

# Code coverage & test report output files.
TEST_OUTPUT_ROOT        := ./.testoutput
NEW_COVER_PROFILE       = $(TEST_OUTPUT_ROOT)/coverage.$(shell xxd -p -l 16 /dev/urandom).out   # generates a new filename each time it's substituted
NEW_REPORT              = $(TEST_OUTPUT_ROOT)/junit.$(shell xxd -p -l 16 /dev/urandom).xml   # generates a new filename each time it's substituted
COVERPKG_FLAG 		    = -coverpkg=./...

# DB
SQL_USER ?= temporal
SQL_PASSWORD ?= temporal

# Only prints output if the exit code is non-zero
define silent_exec
    @output=$$($(1) 2>&1); \
    status=$$?; \
    if [ $$status -ne 0 ]; then \
        echo "$$output"; \
    fi; \
    exit $$status
endef

##### Tools #####
print-go-version:
	@go version

.PHONY: common-formal-test

# tools/common/formal is a nested Go module, so the root go test selectors do
# not reach it and it needs its own invocation.
common-formal-test:
	@cd tools/common/formal && GOWORK=off go test -count=1 -tags test_dep ./...

clean-tools:
	@printf $(COLOR) "Delete tools..."
	@rm -rf $(STAMPDIR)
	@rm -rf $(LOCALBIN)

$(STAMPDIR):
	@mkdir -p $(STAMPDIR)

$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

.PHONY: golangci-lint
LINT_CODE_TARGETS ?= ./...
GOLANGCI_LINT_BASE_REV ?= $(MAIN_BRANCH)
GOLANGCI_LINT_FIX ?= true
# Bumped from v2.9.0 for Go 1.27 support (generic methods); built under the rc2
# toolchain pinned above. Revisit alongside the temporary GOTOOLCHAIN pin.
GOLANGCI_LINT_VERSION := v2.13.1
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# Don't get confused, there is a single linter called gci, which is a part of the mega linter we use is called golangci-lint.
GCI_VERSION := v0.13.6
GCI := $(LOCALBIN)/gci-$(GCI_VERSION)
$(GCI): $(LOCALBIN)
	$(call go-install-tool,$(GCI),github.com/daixiang0/gci,$(GCI_VERSION))

GOTESTSUM_VER := v1.12.3
GOTESTSUM := $(LOCALBIN)/gotestsum-$(GOTESTSUM_VER)
$(GOTESTSUM): | $(LOCALBIN)
	$(call go-install-tool,$(GOTESTSUM),gotest.tools/gotestsum,$(GOTESTSUM_VER))

API_LINTER_VER := v1.32.3
API_LINTER := $(LOCALBIN)/api-linter-$(API_LINTER_VER)
$(API_LINTER): | $(LOCALBIN)
	$(call go-install-tool,$(API_LINTER),github.com/googleapis/api-linter/cmd/api-linter,$(API_LINTER_VER))

BUF_VER := v1.6.0
BUF := $(LOCALBIN)/buf-$(BUF_VER)
$(BUF): | $(LOCALBIN)
	$(call go-install-tool,$(BUF),github.com/bufbuild/buf/cmd/buf,$(BUF_VER))

GO_API_VER = $(shell go list -m -f '{{.Version}}' go.temporal.io/api \
	|| (echo "failed to fetch version for go.temporal.io/api" >&2))
PROTOGEN := $(LOCALBIN)/protogen-$(GO_API_VER)
$(PROTOGEN): | $(LOCALBIN)
	$(call go-install-tool,$(PROTOGEN),go.temporal.io/api/cmd/protogen,$(GO_API_VER))

ACTIONLINT_VER := v1.7.7
ACTIONLINT := $(LOCALBIN)/actionlint-$(ACTIONLINT_VER)
$(ACTIONLINT): | $(LOCALBIN)
	$(call go-install-tool,$(ACTIONLINT),github.com/rhysd/actionlint/cmd/actionlint,$(ACTIONLINT_VER))

WORKFLOWCHECK_VER := master # TODO: pin this specific version once 0.3.0 follow-up is released
WORKFLOWCHECK := $(LOCALBIN)/workflowcheck-$(WORKFLOWCHECK_VER)
$(WORKFLOWCHECK): | $(LOCALBIN)
	$(call go-install-tool,$(WORKFLOWCHECK),go.temporal.io/sdk/contrib/tools/workflowcheck,$(WORKFLOWCHECK_VER))

# NilAway has no tagged releases; pin the pseudo-version for reproducible CI.
NILAWAY_VER := v0.0.0-20260717164209-b48ebb193579
NILAWAY := $(LOCALBIN)/nilaway-$(NILAWAY_VER)
$(NILAWAY): | $(LOCALBIN)
	$(call go-install-tool,$(NILAWAY),go.uber.org/nilaway/cmd/nilaway,$(NILAWAY_VER))

YAMLFMT_VER := v0.16.0
YAMLFMT := $(LOCALBIN)/yamlfmt-$(YAMLFMT_VER)
$(YAMLFMT): | $(LOCALBIN)
	$(call go-install-tool,$(YAMLFMT),github.com/google/yamlfmt/cmd/yamlfmt,$(YAMLFMT_VER))

GOIMPORTS_VER := v0.49.0
GOIMPORTS := $(LOCALBIN)/goimports-$(GOIMPORTS_VER)
$(STAMPDIR)/goimports-$(GOIMPORTS_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports,$(GOIMPORTS_VER))
	@touch $@
$(GOIMPORTS): $(STAMPDIR)/goimports-$(GOIMPORTS_VER)

GOWRAP_VER := v1.4.3
GOWRAP := $(LOCALBIN)/gowrap
$(STAMPDIR)/gowrap-$(GOWRAP_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(GOWRAP),github.com/hexdigest/gowrap/cmd/gowrap,$(GOWRAP_VER))
	@touch $@
$(GOWRAP): $(STAMPDIR)/gowrap-$(GOWRAP_VER)

GOMAJOR_VER := v0.14.0
GOMAJOR := $(LOCALBIN)/gomajor
$(STAMPDIR)/gomajor-$(GOMAJOR_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(GOMAJOR),github.com/icholy/gomajor,$(GOMAJOR_VER))
	@touch $@
$(GOMAJOR): $(STAMPDIR)/gomajor-$(GOMAJOR_VER)

ERRORTYPE_VER := v0.0.7
ERRORTYPE := $(LOCALBIN)/errortype
$(ERRORTYPE): | $(LOCALBIN)
	$(call go-install-tool,$(ERRORTYPE),fillmore-labs.com/errortype,$(ERRORTYPE_VER))

# Mockgen is called by name throughout the codebase, so we need to keep the binary name consistent
MOCKGEN_VER := v0.6.0
MOCKGEN := $(LOCALBIN)/mockgen
$(STAMPDIR)/mockgen-$(MOCKGEN_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(MOCKGEN),go.uber.org/mock/mockgen,$(MOCKGEN_VER))
	@touch $@
$(MOCKGEN): $(STAMPDIR)/mockgen-$(MOCKGEN_VER)

STRINGER_VER := v0.49.0
STRINGER := $(LOCALBIN)/stringer
$(STAMPDIR)/stringer-$(STRINGER_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(STRINGER),golang.org/x/tools/cmd/stringer,$(STRINGER_VER))
	@touch $@
$(STRINGER): $(STAMPDIR)/stringer-$(STRINGER_VER)

PROTOC_GEN_GO_VER := v1.36.6
PROTOC_GEN_GO := $(LOCALBIN)/protoc-gen-go-$(PROTOC_GEN_GO_VER)
$(STAMPDIR)/protoc-gen-go-$(PROTOC_GEN_GO_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(PROTOC_GEN_GO),google.golang.org/protobuf/cmd/protoc-gen-go,$(PROTOC_GEN_GO_VER))
	@touch $@
$(PROTOC_GEN_GO): $(STAMPDIR)/protoc-gen-go-$(PROTOC_GEN_GO_VER)

PROTOC_GEN_GO_GRPC_VER := v1.3.0
PROTOC_GEN_GO_GRPC := $(LOCALBIN)/protoc-gen-go-grpc-$(PROTOC_GEN_GO_GRPC_VER)
$(STAMPDIR)/protoc-gen-go-grpc-$(PROTOC_GEN_GO_GRPC_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(PROTOC_GEN_GO_GRPC),google.golang.org/grpc/cmd/protoc-gen-go-grpc,$(PROTOC_GEN_GO_GRPC_VER))
	@touch $@
$(PROTOC_GEN_GO_GRPC): $(STAMPDIR)/protoc-gen-go-grpc-$(PROTOC_GEN_GO_GRPC_VER)

PROTOC_GEN_GO_HELPERS := $(LOCALBIN)/protoc-gen-go-helpers-$(GO_API_VER)
$(STAMPDIR)/protoc-gen-go-helpers-$(GO_API_VER): | $(STAMPDIR) $(LOCALBIN)
	$(call go-install-tool,$(PROTOC_GEN_GO_HELPERS),go.temporal.io/api/cmd/protoc-gen-go-helpers,$(GO_API_VER))
	@touch $@
$(PROTOC_GEN_GO_HELPERS): $(STAMPDIR)/protoc-gen-go-helpers-$(GO_API_VER)

$(LOCALBIN)/protoc-gen-go-chasm: $(LOCALBIN) cmd/tools/protoc-gen-go-chasm/main.go go.mod go.sum
	@go build -o $@ ./cmd/tools/protoc-gen-go-chasm

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
# This is courtesy of https://github.com/kubernetes-sigs/kubebuilder/pull/3718
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
printf $(COLOR) "Downloading $${package}" ;\
tmpdir=$$(mktemp -d) ;\
GOBIN=$${tmpdir} go install $${package} ;\
mv $${tmpdir}/$$(basename "$$(echo "$(1)" | sed "s/-$(3)$$//")") $(1) ;\
rm -rf $${tmpdir} ;\
}
endef

##### Proto #####
$(API_BINPB): go.mod go.sum $(PROTO_FILES)
	@printf $(COLOR) "Generating proto dependencies image..."
	@./cmd/tools/getproto/run.sh --out $@

$(INTERNAL_BINPB): $(API_BINPB) $(PROTO_FILES)
	@printf $(COLOR) "Generate proto image..."
	@$(PROTOC) --descriptor_set_in=$(API_BINPB) -I=$(PROTO_ROOT)/internal $(PROTO_FILES) -o $@

$(CHASM_BINPB): $(API_BINPB) $(INTERNAL_BINPB) $(CHASM_PROTO_FILES)
	@printf $(COLOR) "Generate CHASM proto image..."
	@$(PROTOC) --descriptor_set_in=$(API_BINPB):$(INTERNAL_BINPB) -I=. $(CHASM_PROTO_FILES) -o $@

protoc: $(PROTOGEN) $(MOCKGEN) $(GOIMPORTS) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(PROTOC_GEN_GO_HELPERS) $(API_BINPB) $(LOCALBIN)/protoc-gen-go-chasm
	@go run ./cmd/tools/protogen \
		-root=$(ROOT) \
		-proto-out=$(PROTO_OUT) \
		-proto-root=$(PROTO_ROOT) \
		-api-binpb=$(API_BINPB) \
		-protogen-bin=$(PROTOGEN) \
		-goimports-bin=$(GOIMPORTS) \
		-mockgen-bin=$(MOCKGEN) \
		-protoc-gen-go-chasm-bin=$(LOCALBIN)/protoc-gen-go-chasm \
		-protoc-gen-go-bin=$(PROTOC_GEN_GO) \
		-protoc-gen-go-grpc-bin=$(PROTOC_GEN_GO_GRPC) \
		-protoc-gen-go-helpers-bin=$(PROTOC_GEN_GO_HELPERS) \
		$(PROTO_DIRS)

proto-codegen:
	@printf $(COLOR) "Generate service clients..."
	@go generate -run genrpcwrappers ./client/...
	@printf $(COLOR) "Generate server interceptors..."
	@go generate ./common/rpc/interceptor/logtags/...
	@printf $(COLOR) "Generate routing key extractor..."
	@go generate -run genroutingkeyextractor ./common/rpc/interceptor/...
	@printf $(COLOR) "Generate search attributes helpers..."
	@go generate -run gensearchattributehelpers ./common/searchattribute/...

update-go-api:
	@printf $(COLOR) "Update go.temporal.io/api@master..."
	@go get -u go.temporal.io/api@master

##### Binaries #####
clean-bins:
	@printf $(COLOR) "Delete old binaries..."
	@rm -f temporal-server
	@rm -f temporal-server-debug
	@rm -f temporal-cassandra-tool
	@rm -f tdbg
	@rm -f fairsim
	@rm -f temporal-sql-tool
	@rm -f temporal-elasticsearch-tool

temporal-server: $(ALL_SRC)
	@printf $(COLOR) "Build temporal-server with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG) -o temporal-server ./cmd/server

tdbg: $(ALL_SRC)
	@printf $(COLOR) "Build tdbg with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG) -o tdbg ./cmd/tools/tdbg

fairsim: $(ALL_SRC)
	@printf $(COLOR) "Build fairsim with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG) -o fairsim ./cmd/tools/fairsim

temporal-cassandra-tool: $(ALL_SRC)
	@printf $(COLOR) "Build temporal-cassandra-tool with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG) -o temporal-cassandra-tool ./cmd/tools/cassandra

temporal-sql-tool: $(ALL_SRC)
	@printf $(COLOR) "Build temporal-sql-tool with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG) -o temporal-sql-tool ./cmd/tools/sql

temporal-elasticsearch-tool: $(ALL_SRC)
	@printf $(COLOR) "Build temporal-elasticsearch-tool with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG) -o temporal-elasticsearch-tool ./cmd/tools/elasticsearch

temporal-server-debug: $(ALL_SRC)
	@printf $(COLOR) "Build temporal-server-debug with CGO_ENABLED=$(CGO_ENABLED) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_TAG_FLAG),TEMPORAL_DEBUG -o temporal-server-debug ./cmd/server

##### Checks #####
TESTPILOT_PROTOCOL_PROTOS := \
	proto/internal/temporal/server/api/testpilot/v1/case.proto \
	proto/internal/temporal/server/api/testpilot/v1/contract.proto \
	proto/internal/temporal/server/api/testpilot/v1/correlated.proto \
	proto/internal/temporal/server/api/testpilot/v1/event.proto \
	proto/internal/temporal/server/api/testpilot/v1/expression.proto \
	proto/internal/temporal/server/api/testpilot/v1/instruction.proto \
	proto/internal/temporal/server/api/testpilot/v1/program.proto \
	proto/internal/temporal/server/api/testpilot/v1/run.proto \
	proto/internal/temporal/server/api/testpilot/v1/value.proto

.PHONY: canary-build umpire-check-testpilot-protocol umpire-run umpire-fuzz umpire-fuzz-run umpire-repeat umpire-repeat-run umpire-replay umpire-replay-run umpire-assess umpire-assess-run umpire-check-live-tests umpire-rerecord-pinned-runs umpire-ir-bridge umpire-gen-cases umpire-check-cases umpire-gen-fixtures umpire-check-fixtures canary-gen-case canary-check-case umpire-check-backends umpire-check-exploration-bridge umpire-check-replay-bridge fmt-model lint-model fix-model umpire-check-model umpire-gen-model

canary-build:
	@printf $(COLOR) "Build the production canary..."
	@mise exec -- go build -o ./.build/umpire-canary ./tools/canary/cmd/umpire-canary
	@printf 'Built ./.build/umpire-canary\n'

umpire-check-testpilot-protocol: $(API_BINPB) $(TESTPILOT_PROTOCOL_PROTOS)
	@printf $(COLOR) "Check Testpilot protocol..."
	@set -eu; protoc=$$(mise exec -- which protoc); \
		test "$$($$protoc --version)" = "libprotoc 29.5"; \
		temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT HUP INT TERM; \
		"$$protoc" --proto_path=proto/internal --descriptor_set_in=$(API_BINPB) --include_source_info \
			--descriptor_set_out="$$temporary" $(TESTPILOT_PROTOCOL_PROTOS:proto/internal/%=%); \
		TESTPILOT_PROTOCOL_DESCRIPTOR_SET="$$temporary" \
			mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot \
			-run '^TestProtocolMessagesCarryLeadingComments$$'

umpire-run:
	@printf $(COLOR) "Build the Umpire Case runner..."
	@mise exec -- go build -o ./.build/umpire-run ./tools/umpire/cmd/umpire-run
	@printf 'Built ./.build/umpire-run\n'

umpire-fuzz:
	@printf $(COLOR) "Build the Umpire exploration campaign runner..."
	@mise exec -- go build -o ./.build/umpire-fuzz ./tools/umpire/cmd/umpire-fuzz
	@printf 'Built ./.build/umpire-fuzz\n'

# One exploration campaign against a deployment named by environment: SET names the exploratory
# set, UMPIRE_FUZZ_GRPC/UMPIRE_FUZZ_HTTP the frontend, UMPIRE_FUZZ_NAMESPACE/UMPIRE_FUZZ_TASK_QUEUE/
# UMPIRE_FUZZ_NEXUS_ENDPOINT the resources it binds to (created and removed when UMPIRE_FUZZ_CREATE
# is set), UMPIRE_FUZZ_FLAGS any further flags such as caps. The bridge is built first.
umpire-fuzz-run:
	@test -n "$(SET)" || { printf 'SET=<exploratory set> is required\n'; exit 3; }
	@for required in UMPIRE_FUZZ_GRPC UMPIRE_FUZZ_HTTP UMPIRE_FUZZ_NAMESPACE UMPIRE_FUZZ_TASK_QUEUE; do \
		eval "value=\$$$$required"; test -n "$$value" || { printf '%s is required\n' "$$required"; exit 3; }; \
	done
	@$(MAKE) --no-print-directory umpire-fuzz
	@$(MAKE) --no-print-directory umpire-ir-bridge
	@./.build/umpire-fuzz run --set "$(SET)" \
		--grpc "$$UMPIRE_FUZZ_GRPC" --http "$$UMPIRE_FUZZ_HTTP" \
		--namespace "$$UMPIRE_FUZZ_NAMESPACE" --task-queue "$$UMPIRE_FUZZ_TASK_QUEUE" \
		$${UMPIRE_FUZZ_NEXUS_ENDPOINT:+--nexus-endpoint "$$UMPIRE_FUZZ_NEXUS_ENDPOINT"} \
		$${UMPIRE_FUZZ_CREATE:+--create} \
		$(UMPIRE_FUZZ_FLAGS)

umpire-repeat:
	@printf $(COLOR) "Build the Umpire repeat harness..."
	@mise exec -- go build -o ./.build/umpire-repeat ./tools/umpire/cmd/umpire-repeat
	@printf 'Built ./.build/umpire-repeat\n'

# One reproduction loop over the live Testpilot tests: SELECT names the -test.run selection, COUNT
# the iterations, MODE process (one process each) or in-process (one process with -test.count),
# RECORD the record file (a new one under ./.build/umpire-repeat when empty), UMPIRE_REPEAT_FLAGS
# any further flags such as --timeout. It builds the IR bridge the gate uses and runs under the
# physical temporary directory the gate uses; it is an explicit operator action. SELECT is
# read unexpanded, so a trailing `$` anchor survives.
umpire-repeat-run:
	@test -n '$(value SELECT)' || { printf 'SELECT=<test regex> is required\n'; exit 3; }
	@test -n "$(COUNT)" || { printf 'COUNT=<iterations> is required\n'; exit 3; }
	@test -n "$(MODE)" || { printf 'MODE=process|in-process is required\n'; exit 3; }
	@$(MAKE) --no-print-directory umpire-repeat
	@$(MAKE) --no-print-directory umpire-ir-bridge
	@set -eu; \
		physical_tmpdir=$$(cd "$${TMPDIR:-/tmp}" && pwd -P); \
		record='$(RECORD)'; \
		if [ -z "$$record" ]; then mkdir -p ./.build/umpire-repeat; record=./.build/umpire-repeat/record-$$(date +%Y%m%dT%H%M%S).jsonl; fi; \
		TMPDIR="$$physical_tmpdir" mise exec -- ./.build/umpire-repeat run --select '$(value SELECT)' \
			--count "$(COUNT)" --mode "$(MODE)" --record "$$record" $(UMPIRE_REPEAT_FLAGS)

umpire-replay:
	@printf $(COLOR) "Build the Umpire replay runner..."
	@mise exec -- go build -o ./.build/umpire-replay ./tools/umpire/cmd/umpire-replay
	@printf 'Built ./.build/umpire-replay\n'

# One replay of a recorded violated Run: CASE and RUN name the subject's files, SET and QUERY (or
# TARGET) the Query the bridge recovers it by, UMPIRE_REPLAY_GRPC/UMPIRE_REPLAY_HTTP the frontend,
# UMPIRE_REPLAY_NAMESPACE/UMPIRE_REPLAY_TASK_QUEUE/UMPIRE_REPLAY_NEXUS_ENDPOINT the resources the
# Run was recorded against, UMPIRE_REPLAY_FLAGS any further flags such as --promotion-root. The
# replay bridge is built first.
umpire-replay-run:
	@test -n "$(CASE)" || { printf 'CASE=<case.json> is required\n'; exit 3; }
	@test -n "$(RUN)" || { printf 'RUN=<recorded run.json> is required\n'; exit 3; }
	@test -n "$(SET)" || { printf 'SET=<set> is required\n'; exit 3; }
	@test -n "$(QUERY)$(TARGET)" || { printf 'QUERY=<query> or TARGET=<target key> is required\n'; exit 3; }
	@for required in UMPIRE_REPLAY_GRPC UMPIRE_REPLAY_HTTP UMPIRE_REPLAY_NAMESPACE UMPIRE_REPLAY_TASK_QUEUE; do \
		eval "value=\$$$$required"; test -n "$$value" || { printf '%s is required\n' "$$required"; exit 3; }; \
	done
	@$(MAKE) --no-print-directory umpire-replay
	@$(MAKE) --no-print-directory umpire-ir-bridge
	@./.build/umpire-replay run --case "$(CASE)" --run "$(RUN)" --set "$(SET)" \
		$(if $(QUERY),--query "$(QUERY)") $(if $(TARGET),--target "$(TARGET)") \
		--grpc "$$UMPIRE_REPLAY_GRPC" --http "$$UMPIRE_REPLAY_HTTP" \
		--namespace "$$UMPIRE_REPLAY_NAMESPACE" --task-queue "$$UMPIRE_REPLAY_TASK_QUEUE" \
		$${UMPIRE_REPLAY_NEXUS_ENDPOINT:+--nexus-endpoint "$$UMPIRE_REPLAY_NEXUS_ENDPOINT"} \
		$(UMPIRE_REPLAY_FLAGS)

umpire-assess:
	@printf $(COLOR) "Build the Umpire assessment command..."
	@mise exec -- go build -o ./.build/umpire-assess ./tools/umpire/cmd/umpire-assess
	@printf 'Built ./.build/umpire-assess\n'

# One offline assessment of a recorded Run: CASE and RUN name the subject's files, PROFILE an
# Evaluation Profile by its exact name, RECEIPT_ROOT an existing directory outside the model. It
# creates and replays no Run.
umpire-assess-run:
	@test -n "$(CASE)" || { printf 'CASE=<case.json> is required\n'; exit 3; }
	@test -n "$(RUN)" || { printf 'RUN=<recorded run.json> is required\n'; exit 3; }
	@test -n "$(PROFILE)" || { printf 'PROFILE=<name> is required\n'; exit 3; }
	@test -n "$(RECEIPT_ROOT)" || { printf 'RECEIPT_ROOT=<dir> is required\n'; exit 3; }
	@$(MAKE) --no-print-directory umpire-assess
	@./.build/umpire-assess run --case "$(CASE)" --run "$(RUN)" --profile "$(PROFILE)" --receipt-root "$(RECEIPT_ROOT)"

umpire-check-live-tests:
	@$(MAKE) --no-print-directory umpire-ir-bridge
	@set -eu; \
		physical_tmpdir=$$(cd "$${TMPDIR:-/tmp}" && pwd -P); \
		temporary=$$(TMPDIR="$$physical_tmpdir" mktemp); \
		expected=$$(TMPDIR="$$physical_tmpdir" mktemp); \
		actual=$$(TMPDIR="$$physical_tmpdir" mktemp); \
		trap 'rm -f "$$temporary" "$$expected" "$$actual"' EXIT HUP INT TERM; \
		status=0; \
		TMPDIR="$$physical_tmpdir" mise exec -- go test -v -count=1 -timeout 30m -tags 'test_dep integration' \
			./tests -run '^TestTestpilot' > "$$temporary" 2>&1 || status=$$?; \
		cat "$$temporary"; \
		sed -n -E 's/^[[:space:]]*--- FAIL: ([^ ]+).*/\1/p' "$$temporary" | LC_ALL=C sort -u > "$$actual"; \
		: > "$$expected"; \
		if [ "$$status" -ne 0 ] && [ ! -s "$$actual" ]; then \
			printf 'The live suite failed without reporting a test identity.\n'; \
			exit "$$status"; \
		fi; \
		if ! diff -u "$$expected" "$$actual"; then \
			printf 'Live Testpilot failure identities differ from the empty expected set.\n'; \
			exit 1; \
		fi; \
		passing=$$(sed -n -E 's/^[[:space:]]*--- PASS: ([^ ]+).*/\1/p' "$$temporary" | LC_ALL=C sort -u | grep -c . || true); \
		if [ "$$passing" -lt 1 ]; then \
			printf 'The live Testpilot selector matched no passing test identity.\n'; \
			exit 1; \
		fi; \
		printf 'Live Testpilot failure identities match the empty expected set across %s passing identities.\n' "$$passing"

# The recorded Runs the offline tests pin, each as live-test:variable:record:package:probe -- the
# live test that records it into the file the variable names, and the offline test that admits
# it. A Testpilot protocol change moves the Driver catalog, and a Case change the Case identity;
# either leaves the record stale or crossed until it is recorded again live. The control record is
# of the Case kept beside it, not of the generated fixture the live test runs: recording it again
# makes it a Run of the generated Case, and the Case beside it and the Profile name its probe
# expects must follow in the same change.
UMPIRE_PINNED_RUNS := \
	TestTestpilotNexusControlForgedCompletionIsViolated:UMPIRE_CONTROL_RECORD:common/testing/testpilot/replay/testdata/nexusCallerControl-forgedCompletion-run.json:./common/testing/testpilot/replay:TestControlRecordPinsTheCorrelatedKey \
	TestTestpilotCanaryLifecycle:UMPIRE_CANARY_RECORD:tools/canary/assessment/testdata/nexus-caller-syncCompletion-run.json:./tools/canary/assessment:TestAdmitRecordsAClosedRunAndAdmitsIt

# Re-records every pinned Run its probe rejects as stale or crossed, leaves a current one alone, and
# renders the receipt goldens from the control record again, so an unchanged protocol changes
# nothing. A probe failing for any other reason stops the target: re-recording would hide it.
umpire-rerecord-pinned-runs:
	@printf $(COLOR) "Re-record stale pinned Testpilot Runs..."
	@set -eu; \
		physical_tmpdir=$$(cd "$${TMPDIR:-/tmp}" && pwd -P); \
		log=$$(TMPDIR="$$physical_tmpdir" mktemp); \
		temporary=; \
		trap 'rm -f "$$log"; [ -z "$$temporary" ] || rm -rf "$$temporary"' EXIT HUP INT TERM; \
		for pinned in $(UMPIRE_PINNED_RUNS); do \
			set -- $$(printf '%s' "$$pinned" | tr ':' ' '); \
			live=$$1; variable=$$2; record=$$3; package=$$4; probe=$$5; \
			status=0; \
			TMPDIR="$$physical_tmpdir" mise exec -- go test -v -count=1 -tags test_dep "$$package" \
				-run "^$$probe"'$$' > "$$log" 2>&1 || status=$$?; \
			if [ "$$status" -eq 0 ]; then \
				grep -q "^--- PASS: $$probe " "$$log" || { printf '%s matched no test in %s.\n' "$$probe" "$$package"; exit 1; }; \
				printf '%s is current.\n' "$$record"; \
				continue; \
			fi; \
			if ! grep -Eq '(stale|crossed): ' "$$log"; then \
				cat "$$log"; \
				printf '%s fails %s for another reason than a stale or crossed record.\n' "$$record" "$$probe"; \
				exit 1; \
			fi; \
			printf 'Re-recording %s through %s...\n' "$$record" "$$live"; \
			temporary=$$(mktemp -d "$$(dirname "$$record")/.rerecord.XXXXXX"); \
			if ! env "$$variable=$$PWD/$$temporary/run.json" TMPDIR="$$physical_tmpdir" mise exec -- go test -count=1 -timeout 30m \
				-tags 'test_dep integration' ./tests -run "^$$live"'$$' > "$$log" 2>&1; then \
				cat "$$log"; \
				printf '%s failed; %s is unchanged.\n' "$$live" "$$record"; \
				exit 1; \
			fi; \
			test -s "$$temporary/run.json" || { printf '%s wrote no record.\n' "$$live"; exit 1; }; \
			mv -f "$$temporary/run.json" "$$record"; \
			rm -rf "$$temporary"; \
			temporary=; \
		done; \
		UMPIRE_RECEIPT_GOLDENS=write TMPDIR="$$physical_tmpdir" mise exec -- go test -count=1 -tags test_dep \
			./common/testing/testpilot/evaluation -run '^TestReceiptGoldens$$' > "$$log" 2>&1 || { cat "$$log"; exit 1; }
	@temporary_root=$$(cd "$${TMPDIR:-/tmp}" && pwd -P); \
		TMPDIR="$$temporary_root" mise exec -- go test -count=1 -tags test_dep \
			./common/testing/testpilot/replay ./common/testing/testpilot/evaluation ./tools/umpire/cmd/umpire-assess ./tools/canary/...

umpire-ir-bridge:
	@printf $(COLOR) "Build the Umpire IR bridge..."
	@mise exec -- go build -o ./.build/umpire-ir-bridge ./tools/umpire/cmd/umpire-ir-bridge

umpire-gen-cases:
	@mise exec -- go run ./tools/umpire/cmd/umpire-gen-cases --update

umpire-check-cases:
	@mise exec -- go run ./tools/umpire/cmd/umpire-gen-cases

umpire-gen-fixtures:
	@mise exec -- go run ./tools/umpire/cmd/umpire-gen-cases --kind functional --update

umpire-check-fixtures:
	@mise exec -- go run ./tools/umpire/cmd/umpire-gen-cases --kind functional

canary-gen-case:
	@mise exec -- go run ./tools/umpire/cmd/umpire-gen-cases --kind canary --update

canary-check-case:
	@mise exec -- go run ./tools/umpire/cmd/umpire-gen-cases --kind canary

# The backend agreement: Quint and P read the lifted IR and are held to Go's reading of it. The tests
# pin their tools and fail when one is missing; the pins are in tools/umpire/export/tools_test.go.
# UMPIRE_BACKEND_TOOLS names where P and .NET are installed, UMPIRE_BACKENDS_OUT a directory that
# keeps every export, dump and report. Run from the package's directory, the receipts are printed.
umpire-check-backends:
	@UMPIRE_BACKENDS=require CC="$${CC:-/usr/bin/clang}" mise exec -- go test -C ./tools/umpire/export -count=1 -timeout 30m -tags test_dep

umpire-check-exploration-bridge:
	@mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-ir-bridge -run '^TestIRBridgeProtocol$$/(campaign|sequence)$$'

umpire-check-replay-bridge:
	@mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-ir-bridge -run '^TestIRBridgeProtocol$$/replay$$'

# Scala model tooling uses explicit source roots so authoring, the lifter and the gate stay separate projects.
MODEL_ROOT := model
MODEL_CLI := mise exec -- scala-cli
MODEL_GATE_ARGS ?=
MODEL_SCALAFIX = $(MODEL_CLI) --power fix --enable-built-in=false \
	--scalafix-conf "$(CURDIR)/$(MODEL_ROOT)/.scalafix.conf" \
	--scalac-option -Wunused:all --suppress-outdated-dependency-warning
MODEL_SOURCES := $(MODEL_ROOT)/project.scala $(MODEL_ROOT)/umpire $(MODEL_ROOT)/temporal
model_scalafix_files = $(foreach source,$(1),--scalafix-arg=--files --scalafix-arg="$(CURDIR)/$(source)")
MODEL_SCALAFIX_FILES = $(call model_scalafix_files,$(MODEL_SOURCES))
# scala-cli hands scalafix every file under a project's directory, so the lifter names its own sources
# to leave out the fixtures under testdata, which its build excludes.
MODEL_LIFTER_SOURCES := $(wildcard $(MODEL_ROOT)/lifter/*.scala) $(MODEL_ROOT)/lifter/test
# The lifter's fixtures that lift are a build of their own against the packaged Models. The refusal
# fixtures stay outside scalafix: unsupported's `var` and `while` are what the lifter must refuse,
# and werror and crossed must not compile, so RemoveUnused has nothing to read. The fixtures' build
# makes warnings errors, and -Wunused:all also warns of the parameters RemoveUnused keeps
# (params = false), so the lint keeps warnings as warnings.
MODEL_LIFTS := $(MODEL_ROOT)/lifter/testdata/lifts
MODEL_LIFTS_SCALAFIX = $(MODEL_SCALAFIX) --scalac-option -Werror:false \
	$(call model_scalafix_files,$(wildcard $(MODEL_LIFTS)/*.scala))
# The metrics project compiles gate sources beside its own; lint-model-gate lints those.
MODEL_METRICS_SOURCES := $(wildcard $(MODEL_ROOT)/metrics/*.scala)
MODEL_PROTO_JARS := $(MODEL_ROOT)/gen/ir-scalapb.jar $(MODEL_ROOT)/gen/api-scalapb.jar
MODEL_JAR := $(MODEL_ROOT)/gen/model-scala.jar
# The gate is one Scala program; its own arguments follow.
MODEL_GATE = $(MODEL_CLI) run --suppress-outdated-dependency-warning $(MODEL_ROOT)/gate --
# The gate's own tests run it against stand-in tools, so they need no jar and no real tool.
MODEL_GATE_TEST = $(MODEL_CLI) test --suppress-outdated-dependency-warning $(MODEL_ROOT)/gate

# make sees only the schema; after the generator's version changes in the gate, the gate's own run
# repackages the jar.
$(MODEL_ROOT)/gen/ir-scalapb.jar: proto/internal/temporal/server/api/umpire/v1/ir.proto
	@printf $(COLOR) "Package model IR classes..."
	@$(MODEL_GATE) --generate-ir

$(MODEL_ROOT)/gen/api-scalapb.jar: $(MODEL_ROOT)/gen/ir-scalapb.jar proto/api.binpb cmd/tools/getproto/main.go cmd/tools/getproto/files.go model/gate/Gate.scala model/gate/project.scala go.mod go.sum mise.toml
	@printf $(COLOR) "Package model API classes..."
	@$(MODEL_GATE) --generate-api --if-stale

# The Models' TASTy, packaged as the gate's "package the Models' TASTy" step packages it.
$(MODEL_JAR): $(MODEL_PROTO_JARS) $(MODEL_ROOT)/project.scala $(shell find $(MODEL_ROOT)/umpire $(MODEL_ROOT)/temporal -name '*.scala')
	@printf $(COLOR) "Package the Models' TASTy..."
	@$(MODEL_CLI) --power package --suppress-outdated-dependency-warning --library $(MODEL_SOURCES) -f -o $@

fmt-model:
	@printf $(COLOR) "Formatting model files..."
	@$(MODEL_CLI) fmt --scalafmt-conf $(MODEL_ROOT)/.scalafmt.conf $(MODEL_SOURCES) $(MODEL_ROOT)/lifter $(MODEL_ROOT)/gate $(MODEL_ROOT)/metrics

# The five scalafix runs build separate projects, so they run side by side; output-sync prints each
# run's output whole, after its title. The sugar check builds the gate, which lint-model-gate builds
# with other options, so it runs after them.
MODEL_LINTS := lint-model-models lint-model-lifter lint-model-lifts lint-model-gate lint-model-metrics
.PHONY: $(MODEL_LINTS) lint-model-syntax

lint-model: $(MODEL_PROTO_JARS) $(MODEL_JAR)
	@printf $(COLOR) "Checking model formatting..."
	@$(MODEL_CLI) fmt --scalafmt-conf $(MODEL_ROOT)/.scalafmt.conf --check $(MODEL_SOURCES) $(MODEL_ROOT)/lifter $(MODEL_ROOT)/gate $(MODEL_ROOT)/metrics
	@$(MAKE) --no-print-directory -j5 --output-sync=target $(MODEL_LINTS)
	@$(MAKE) --no-print-directory lint-model-syntax

lint-model-models:
	@printf $(COLOR) "Linting model files..."
	@$(MODEL_SCALAFIX) $(MODEL_SCALAFIX_FILES) --check $(MODEL_SOURCES)

lint-model-lifter:
	@printf $(COLOR) "Linting the lifter..."
	@cd $(MODEL_ROOT)/lifter && $(MODEL_SCALAFIX) $(call model_scalafix_files,$(MODEL_LIFTER_SOURCES)) --check .

lint-model-lifts:
	@printf $(COLOR) "Linting the lifter's lifting fixtures..."
	@cd $(MODEL_LIFTS) && $(MODEL_LIFTS_SCALAFIX) --check .

lint-model-gate:
	@printf $(COLOR) "Linting the gate..."
	@cd $(MODEL_ROOT)/gate && $(MODEL_SCALAFIX) --check .

lint-model-metrics:
	@printf $(COLOR) "Linting the source metrics..."
	@cd $(MODEL_ROOT)/metrics && $(MODEL_SCALAFIX) $(call model_scalafix_files,$(MODEL_METRICS_SOURCES)) --check .

# Sugar is defined only in a Syntax.scala, each definition documented with its `Core form:`, and no
# core file of the framework or the lifter imports or names it.
lint-model-syntax:
	@printf $(COLOR) "Checking the model's sugar..."
	@$(MODEL_GATE) --check-syntax

# Applies the scalafix rewrites; findings without a rewrite (e.g. DisableSyntax) still fail.
fix-model: $(MODEL_PROTO_JARS) $(MODEL_JAR)
	@printf $(COLOR) "Applying model lint fixes..."
	@$(MODEL_SCALAFIX) $(MODEL_SCALAFIX_FILES) $(MODEL_SOURCES)
	@cd $(MODEL_ROOT)/lifter && $(MODEL_SCALAFIX) $(call model_scalafix_files,$(MODEL_LIFTER_SOURCES)) .
	@cd $(MODEL_LIFTS) && $(MODEL_LIFTS_SCALAFIX) .
	@cd $(MODEL_ROOT)/gate && $(MODEL_SCALAFIX) .
	@cd $(MODEL_ROOT)/metrics && $(MODEL_SCALAFIX) $(call model_scalafix_files,$(MODEL_METRICS_SOURCES)) .

# The gate packages the IR classes itself when the schema changed.
umpire-check-model:
	@printf $(COLOR) "Check model IR..."
	@$(MODEL_GATE_TEST)
	@$(MODEL_GATE) $(MODEL_GATE_ARGS)

umpire-gen-model:
	@printf $(COLOR) "Generate model IR..."
	@$(MODEL_GATE_TEST)
	@$(MODEL_GATE) --update $(MODEL_GATE_ARGS)

# Removes scala-cli build state (.bsp/, .scala-build/) anywhere in the checkout and the given
# scratch directories under .flow/tmp (UMPIRE_SCRATCH="fn112-3 fn114-1"). Agents use this target
# instead of running rm directly; model/gen and evidence they did not name are left alone.
UMPIRE_SCRATCH ?=
.PHONY: umpire-clean-scratch
umpire-clean-scratch:
	@printf $(COLOR) "Remove scala-cli build state and named .flow/tmp scratch..."
	@find . -path ./.git -prune -o -path ./.claude -prune -o \( -name .bsp -o -name .scala-build \) -type d -prune -exec rm -rf {} +
	@for dir in $(UMPIRE_SCRATCH); do \
		case "$$dir" in */*|..*|"") echo "refusing scratch name '$$dir'"; exit 1;; esac; \
		rm -rf ".flow/tmp/$$dir"; \
	done

goimports: fmt-imports $(GOIMPORTS)
	@printf $(COLOR) "Run goimports for all files..."
	@UNGENERATED_FILES=$$($(SOURCE_FIND) -type f -name '*.go' -print0 | xargs -0 grep -L -e "Code generated by .* DO NOT EDIT." || true) && \
		$(GOIMPORTS) -w $$UNGENERATED_FILES

lint: lint-code lint-actions lint-api lint-protos lint-yaml
	@printf $(COLOR) "Run linters..."

lint-actions: $(ACTIONLINT)
	@printf $(COLOR) "Linting GitHub actions..."
	@$(ACTIONLINT)

.PHONY: lint-code lint-code-fast
# golangci-lint's --new-from-rev reads `git diff <rev>` of the whole tree, and its diff reader
# (revgrep) parses the continuation of any line over 4 KiB as a fresh line: a tracked log that quotes
# source on one line makes it panic. Both targets therefore filter by a patch of the Go files outside
# testdata and nested modules, tracked and untracked, against the merge base. $(1) is the patch file.
define go-only-patch
base=$$(git merge-base HEAD "$(GOLANGCI_LINT_BASE_REV)"); \
excludes=$$(git ls-files --cached --others --exclude-standard -- '*/go.mod' \
  | sed 's|/go.mod$$|/**|; s|^|:(exclude,glob)|'); \
git diff --no-ext-diff --no-renames "$$base" -- '*.go' ':(exclude,glob)**/testdata/**' $$excludes > "$(1)"; \
git ls-files --others --exclude-standard -- '*.go' ':(exclude,glob)**/testdata/**' $$excludes | \
	while IFS= read -r file; do \
		status=0; git diff --no-ext-diff --no-index -- /dev/null "$$file" >> "$(1)" || status=$$?; \
		[ "$$status" -le 1 ] || exit "$$status"; \
	done
endef

# The patch filters reported issues _after_ analysis; this target also reduces package inputs _before_ analysis.
# testdata and nested modules are skipped like `./...` skips them: listing such a package explicitly would lint it.
lint-code-fast:
	@if ! git rev-parse --verify --quiet "$(GOLANGCI_LINT_BASE_REV)^{commit}" >/dev/null; then \
		printf $(RED) "GOLANGCI_LINT_BASE_REV=$(GOLANGCI_LINT_BASE_REV) is not a known commit; fetch it or override GOLANGCI_LINT_BASE_REV"; \
		exit 1; \
	fi
	@set -eu; base=$$(git merge-base HEAD "$(GOLANGCI_LINT_BASE_REV)"); \
	excludes=$$(git ls-files --cached --others --exclude-standard -- '*/go.mod' \
	  | sed 's|/go.mod$$|/**|; s|^|:(exclude,glob)|'); \
	targets=$$({ \
		git diff --no-renames --name-only "$$base" -- '*.go' ':(exclude,glob)**/testdata/**' $$excludes; \
		git ls-files --others --exclude-standard -- '*.go' ':(exclude,glob)**/testdata/**' $$excludes; \
	} | sed 's|^|./|; s|/[^/]*$$||' | sort -u \
	  | while read -r dir; do [ -d "$$dir" ] && printf '%s ' "$$dir"; done); \
	if [ -z "$$targets" ]; then \
		printf $(COLOR) "No changed Go packages to lint."; \
	else \
		patch=$$(mktemp); \
		trap 'rm -f "$$patch"' EXIT HUP INT TERM; \
		$(call go-only-patch,$$patch); \
		$(MAKE) GOLANGCI_LINT_BASE_REV="$$base" GOLANGCI_LINT_PATCH="$$patch" LINT_CODE_TARGETS="$$targets" lint-code; \
	fi

lint-code: $(GOLANGCI_LINT) $(ERRORTYPE)
	@printf $(COLOR) "Linting code..."
	@set -eu; patch="$(GOLANGCI_LINT_PATCH)"; \
	if [ -z "$$patch" ]; then \
		git rev-parse --verify --quiet "$(GOLANGCI_LINT_BASE_REV)^{commit}" >/dev/null \
		  || { printf $(RED) "GOLANGCI_LINT_BASE_REV=$(GOLANGCI_LINT_BASE_REV) is not a known commit; fetch it or override GOLANGCI_LINT_BASE_REV"; exit 1; }; \
		patch=$$(mktemp); trap 'rm -f "$$patch"' EXIT HUP INT TERM; \
		$(call go-only-patch,$$patch); \
	fi; \
	$(GOLANGCI_LINT) run \
		--verbose \
		--build-tags $(ALL_TEST_TAGS) \
		--timeout 10m \
		--fix=$(GOLANGCI_LINT_FIX) \
		--new-from-patch="$$patch" \
		--config=.github/.golangci.yml \
		$(LINT_CODE_TARGETS)
	@go vet -tags $(ALL_TEST_TAGS) -vettool="$(ERRORTYPE)" -style-check=false $(LINT_CODE_TARGETS)

lint-yaml: $(YAMLFMT)
	@printf $(COLOR) "Checking YAML formatting..."
	@$(YAMLFMT) -conf .github/.yamlfmt -lint .

# Nil-safety analysis. Override NILAWAY_SCOPE to widen coverage as more packages
# are made nil-clean; every path below is derived from it. -include-pkgs restricts
# the expensive inference to our own packages; without it nilaway analyzes the
# entire transitive dependency graph and OOMs CI runners.
.PHONY: lint-nilaway
NILAWAY_SCOPE ?= chasm/lib/scheduler
lint-nilaway: $(NILAWAY)
	@printf $(COLOR) "Running nilaway..."
	@$(NILAWAY) \
		-include-pkgs $(MODULE_ROOT)/$(NILAWAY_SCOPE) \
		-include-errors-in-files $(ROOT)/$(NILAWAY_SCOPE) \
		-exclude-test-files \
		-exclude-file-docstrings "Code generated by" \
		./$(NILAWAY_SCOPE)/...

lint-api: $(API_LINTER) $(API_BINPB)
	@printf $(COLOR) "Linting proto API..."
	$(call silent_exec, $(API_LINTER) --set-exit-status -I=$(PROTO_ROOT)/internal --descriptor-set-in $(API_BINPB) --config=$(PROTO_ROOT)/api-linter.yaml $(PROTO_FILES))

lint-protos: $(BUF) $(INTERNAL_BINPB) $(CHASM_BINPB)
	@printf $(COLOR) "Linting proto definitions..."
	@$(BUF) lint --config $(PROTO_ROOT)/internal/buf.yaml $(INTERNAL_BINPB)
	@$(BUF) lint --config chasm/lib/buf.yaml $(CHASM_BINPB)

fmt: fmt-gofix fmt-imports fmt-protos fmt-yaml

# Some fixes enable others (e.g. rangeint may expose minmax opportunities),
# so - as recommended by the Go team - we run go fix in a loop until it reaches
# a fixed point. We check for "files updated" in the output rather than relying
# on the exit code alone, since go fix can exit non-zero without actually
# modifying any files (see https://github.com/golang/go/issues/77482).
#
# Note: go fix automatically skips generated files.
#
# embedlit is disabled because it rewrites large parts of the codebase.
GOFIX_FLAGS ?= -embedlit=false
GOFIX_MAX_ITERATIONS ?= 5
fmt-gofix:
	@printf $(COLOR) "Run go fix..."
	@n=0; while [ $$n -lt $(GOFIX_MAX_ITERATIONS) ]; do \
		output=$$(go fix $(GOFIX_FLAGS) ./... 2>&1); \
		echo "$$output"; \
		if ! echo "$$output" | grep -q "files updated"; then break; fi; \
		n=$$((n + 1)); \
		printf $(COLOR) "Re-running go fix..."; \
	done; \
	if [ $$n -ge $(GOFIX_MAX_ITERATIONS) ]; then echo "ERROR: go fix did not converge after $(GOFIX_MAX_ITERATIONS) iterations"; exit 1; fi

fmt-imports: $(GCI) # Don't get confused, there is a single linter called gci, which is a part of the mega linter we use is called golangci-lint.
	@printf $(COLOR) "Formatting imports..."
	@$(SOURCE_FIND) -type f -name '*.go' -print0 | \
		xargs -0 $(GCI) write --skip-generated -s standard -s default

parallelize-tests:
	@printf $(COLOR) "Add t.Parallel() to tests..."
	@go run ./cmd/tools/parallelize $(INTEGRATION_TEST_DIRS)

fmt-protos: $(BUF)
	@printf $(COLOR) "Formatting proto files..."
	@$(BUF) format -w $(PROTO_ROOT)/internal
	@$(BUF) format -w --config chasm/lib/buf.yaml chasm/lib

fmt-yaml: $(YAMLFMT)
	@printf $(COLOR) "Formatting YAML files..."
	@$(YAMLFMT) -conf .github/.yamlfmt .

# Edit proto/internal/buf.yaml to exclude specific files from this check.
# TODO: buf breaking check for CHASM protos.
buf-breaking: $(BUF) $(API_BINPB) $(INTERNAL_BINPB)
	@printf $(COLOR) "Run buf breaking proto changes check..."
	@env BUF=$(BUF) API_BINPB=$(API_BINPB) INTERNAL_BINPB=$(INTERNAL_BINPB) CHASM_BINPB=$(CHASM_BINPB) MAIN_BRANCH=$(MAIN_BRANCH) \
		./develop/buf-breaking.sh

shell-check:
	@printf $(COLOR) "Run shellcheck for script files..."
	@shellcheck $(ALL_SCRIPTS)

workflowcheck: $(WORKFLOWCHECK)
	@printf $(COLOR) "Run workflowcheck for system workflows..."
	for dir in $(SYSTEM_WORKFLOWS_ROOT)/*/ ; do \
		echo "Running workflowcheck on $$dir" ; \
		$(WORKFLOWCHECK) "$$dir" ; \
	done

check: lint shell-check

##### Tests #####
clean-test-output:
	@printf $(COLOR) "Delete test output..."
	@rm -rf $(TEST_OUTPUT_ROOT)
	@go clean -testcache

build-tests:
	@printf $(COLOR) "Build tests..."
	@CGO_ENABLED=$(CGO_ENABLED) go test $(TEST_TAG_FLAG) -exec="true" -count=0 $(TEST_DIRS)

unit-test: clean-test-output
	@printf $(COLOR) "Run unit tests..."
	@CGO_ENABLED=$(CGO_ENABLED) go test $(UNIT_TEST_DIRS) $(COMPILED_TEST_ARGS) 2>&1 | tee -a test.log
	@$(MAKE) verify-test-log

integration-test: clean-test-output
	@printf $(COLOR) "Run integration tests..."
	@CGO_ENABLED=$(CGO_ENABLED) go test $(INTEGRATION_TEST_DIRS) $(COMPILED_TEST_ARGS) 2>&1 | tee -a test.log
	@$(MAKE) verify-test-log

functional-test: clean-test-output
	@printf $(COLOR) "Run functional tests..."
	@CGO_ENABLED=$(CGO_ENABLED) go test $(FUNCTIONAL_TEST_ROOT) $(COMPILED_TEST_ARGS) -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER) 2>&1 | tee -a test.log
	@CGO_ENABLED=$(CGO_ENABLED) go test $(FUNCTIONAL_TEST_NDC_ROOT) $(COMPILED_TEST_ARGS) -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER) 2>&1 | tee -a test.log
	@CGO_ENABLED=$(CGO_ENABLED) go test $(FUNCTIONAL_TEST_XDC_ROOT) $(COMPILED_TEST_ARGS) -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER) 2>&1 | tee -a test.log
	@$(MAKE) verify-test-log

functional-with-fault-injection-test: clean-test-output
	@printf $(COLOR) "Run integration tests with fault injection..."
	@CGO_ENABLED=$(CGO_ENABLED) go test $(FUNCTIONAL_TEST_ROOT) $(COMPILED_TEST_ARGS) -enableFaultInjection=true -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER) 2>&1 | tee -a test.log
	@CGO_ENABLED=$(CGO_ENABLED) go test $(FUNCTIONAL_TEST_NDC_ROOT) $(COMPILED_TEST_ARGS) -enableFaultInjection=true -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER) 2>&1 | tee -a test.log
	@CGO_ENABLED=$(CGO_ENABLED) go test $(FUNCTIONAL_TEST_XDC_ROOT) $(COMPILED_TEST_ARGS) -enableFaultInjection=true -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER) 2>&1 | tee -a test.log
	@$(MAKE) verify-test-log

mixed-brain-test: clean-test-output
	@printf $(COLOR) "Run mixed brain tests..."
	@cd $(MIXED_BRAIN_TEST_ROOT) && CGO_ENABLED=1 TEST_OUTPUT_ROOT=$(CURDIR)/$(TEST_OUTPUT_ROOT) go test -v ./... $(COMPILED_TEST_ARGS) 2>&1 | tee -a $(CURDIR)/test.log
	@$(MAKE) verify-test-log

LEAK_OUTPUT_DIR        ?= $(TEST_OUTPUT_ROOT)/leakcheck
LEAK_ITERS             ?= 15
LEAK_ITERS_WARMUP      ?= 3
LEAK_GC_SETTLE_TIMEOUT ?= 10s
LEAK_TIMEOUT           ?= 5m
leak-test:
	@printf $(COLOR) "Run goroutine-leak regression test..."
	@mkdir -p $(LEAK_OUTPUT_DIR)
	LEAK_ITERS=$(LEAK_ITERS) \
		LEAK_ITERS_WARMUP=$(LEAK_ITERS_WARMUP) \
		LEAK_OUTPUT_DIR=$(LEAK_OUTPUT_DIR) \
		LEAK_GC_SETTLE_TIMEOUT=$(LEAK_GC_SETTLE_TIMEOUT) \
		go test -run TestClusterShutdownLeak -count=1 -v \
			-timeout $(LEAK_TIMEOUT) $(TEST_TAG_FLAG) \
			./tests/leakcheck/ -args -persistenceType=sql -persistenceDriver=sqlite

verify-test-log:
	@test -s test.log || (echo "TEST FAILURE: test.log is missing or empty" && exit 1)
	@grep -q "^ok" test.log || (echo "TEST FAILURE: no passing test found in test.log" && exit 1)
	@! grep -q "^--- FAIL" test.log || (echo "TEST FAILURE: failing test found in test.log" && exit 1)

test: unit-test integration-test functional-test

##### Coverage & Reporting #####
$(TEST_OUTPUT_ROOT):
	@mkdir -p $(TEST_OUTPUT_ROOT)

prepare-coverage-test: $(GOTESTSUM) $(TEST_OUTPUT_ROOT)

unit-test-coverage: prepare-coverage-test
	@printf $(COLOR) "Run unit tests with coverage..."
	go run ./cmd/tools/test-runner test --gotestsum-path=$(GOTESTSUM) --max-attempts=$(MAX_TEST_ATTEMPTS) $(TEST_RUNNER_TIMEOUT_ARG) --junitfile=$(NEW_REPORT) -- \
		$(COMPILED_TEST_ARGS) -coverprofile=$(NEW_COVER_PROFILE) $(UNIT_TEST_DIRS)

integration-test-coverage: prepare-coverage-test
	@printf $(COLOR) "Run integration tests with coverage..."
	go run ./cmd/tools/test-runner test --gotestsum-path=$(GOTESTSUM) --max-attempts=$(MAX_TEST_ATTEMPTS) $(TEST_RUNNER_TIMEOUT_ARG) --junitfile=$(NEW_REPORT) -- \
		$(COMPILED_TEST_ARGS) -coverprofile=$(NEW_COVER_PROFILE) $(INTEGRATION_TEST_DIRS)

# MUST use the same build flags as functional-test-coverage and functional-test-{xdc,ndc}-coverage for best build caching.
pre-build-functional-test-coverage: prepare-coverage-test
	go test -c -cover -o /dev/null $(COMPILED_TEST_ARGS) $(COVERPKG_FLAG) $(FUNCTIONAL_TEST_ROOT)

functional-test-coverage: prepare-coverage-test
	@printf $(COLOR) "Run functional tests with coverage with $(PERSISTENCE_DRIVER) driver..."
	go run ./cmd/tools/test-runner test --gotestsum-path=$(GOTESTSUM) --max-attempts=$(MAX_TEST_ATTEMPTS) $(TEST_RUNNER_TIMEOUT_ARG) --junitfile=$(NEW_REPORT) -- \
		$(COMPILED_TEST_ARGS) -coverprofile=$(NEW_COVER_PROFILE) $(COVERPKG_FLAG) $(FUNCTIONAL_TEST_ROOT) \
		-args -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER)

functional-test-xdc-coverage: prepare-coverage-test
	@printf $(COLOR) "Run functional test for cross DC with coverage with $(PERSISTENCE_DRIVER) driver..."
	go run ./cmd/tools/test-runner test --gotestsum-path=$(GOTESTSUM) --max-attempts=$(MAX_TEST_ATTEMPTS) $(TEST_RUNNER_TIMEOUT_ARG) --junitfile=$(NEW_REPORT) -- \
		$(COMPILED_TEST_ARGS) -coverprofile=$(NEW_COVER_PROFILE) $(COVERPKG_FLAG) $(FUNCTIONAL_TEST_XDC_ROOT) \
		-args -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER)

functional-test-ndc-coverage: prepare-coverage-test
	@printf $(COLOR) "Run functional test for NDC with coverage with $(PERSISTENCE_DRIVER) driver..."
	go run ./cmd/tools/test-runner test --gotestsum-path=$(GOTESTSUM) --max-attempts=$(MAX_TEST_ATTEMPTS) $(TEST_RUNNER_TIMEOUT_ARG) --junitfile=$(NEW_REPORT) -- \
		$(COMPILED_TEST_ARGS) -coverprofile=$(NEW_COVER_PROFILE) $(COVERPKG_FLAG) $(FUNCTIONAL_TEST_NDC_ROOT) \
		-args -persistenceType=$(PERSISTENCE_TYPE) -persistenceDriver=$(PERSISTENCE_DRIVER)

report-test-crash: $(TEST_OUTPUT_ROOT)
	@printf $(COLOR) "Generate test crash junit report..."
	@go run ./cmd/tools/test-runner report-crash --gotestsum=report-crash \
		--junitfile=$(TEST_OUTPUT_ROOT)/junit.crash.xml \
		--crashreportname=$(CRASH_REPORT_NAME)

generate-test-summary: $(TEST_OUTPUT_ROOT)
	@go run ./cmd/tools/test-runner generate-summary \
		--junit-glob=$(TEST_OUTPUT_ROOT)/junit.*.xml \
		--summary-output-dir=$(TEST_OUTPUT_ROOT)

##### Schema #####
install-schema-cass-es: temporal-cassandra-tool install-schema-es
	@printf $(COLOR) "Install Cassandra schema..."
	./temporal-cassandra-tool drop -k $(TEMPORAL_DB) -f
	./temporal-cassandra-tool create -k $(TEMPORAL_DB) --rf 1
	./temporal-cassandra-tool -k $(TEMPORAL_DB) setup-schema -v 0.0
	./temporal-cassandra-tool -k $(TEMPORAL_DB) update-schema -d ./schema/cassandra/temporal/versioned

install-schema-mysql: install-schema-mysql8

install-schema-mysql8: temporal-sql-tool
	@printf $(COLOR) "Install MySQL schema..."
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(TEMPORAL_DB) drop -f
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(TEMPORAL_DB) create
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(TEMPORAL_DB) setup-schema -v 0.0
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(TEMPORAL_DB) update-schema -d ./schema/mysql/v8/temporal/versioned
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(VISIBILITY_DB) drop  -f
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(VISIBILITY_DB) create
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(VISIBILITY_DB) setup-schema -v 0.0
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) --pl mysql8 --db $(VISIBILITY_DB) update-schema -d ./schema/mysql/v8/visibility/versioned

install-schema-postgresql: install-schema-postgresql12

install-schema-postgresql12: temporal-sql-tool
	@printf $(COLOR) "Install Postgres schema..."
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(TEMPORAL_DB) drop -f
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(TEMPORAL_DB) create
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(TEMPORAL_DB) setup -v 0.0
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(TEMPORAL_DB) update-schema -d ./schema/postgresql/v12/temporal/versioned
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(VISIBILITY_DB) drop -f
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(VISIBILITY_DB) create
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(VISIBILITY_DB) setup-schema -v 0.0
	./temporal-sql-tool -u $(SQL_USER) --pw $(SQL_PASSWORD) -p 5432 --pl postgres12 --db $(VISIBILITY_DB) update-schema -d ./schema/postgresql/v12/visibility/versioned

install-schema-es: temporal-elasticsearch-tool
	@printf $(COLOR) "Install Elasticsearch schema..."
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 setup-schema
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 create-index --index temporal_visibility_v1_dev

install-schema-es-secondary: temporal-elasticsearch-tool
	@printf $(COLOR) "Install Elasticsearch schema..."
	./temporal-elasticsearch-tool -ep http://127.0.0.1:8200 setup-schema
	./temporal-elasticsearch-tool -ep http://127.0.0.1:8200 create-index --index temporal_visibility_v1_secondary

install-schema-xdc: temporal-cassandra-tool temporal-elasticsearch-tool
	@printf $(COLOR)  "Install Cassandra schema (active)..."
	./temporal-cassandra-tool drop -k temporal_cluster_a -f
	./temporal-cassandra-tool create -k temporal_cluster_a --rf 1
	./temporal-cassandra-tool -k temporal_cluster_a setup-schema -v 0.0
	./temporal-cassandra-tool -k temporal_cluster_a update-schema -d ./schema/cassandra/temporal/versioned

	@printf $(COLOR)  "Install Cassandra schema (standby)..."
	./temporal-cassandra-tool drop -k temporal_cluster_b -f
	./temporal-cassandra-tool create -k temporal_cluster_b --rf 1
	./temporal-cassandra-tool -k temporal_cluster_b setup-schema -v 0.0
	./temporal-cassandra-tool -k temporal_cluster_b update-schema -d ./schema/cassandra/temporal/versioned

	@printf $(COLOR)  "Install Cassandra schema (other)..."
	./temporal-cassandra-tool drop -k temporal_cluster_c -f
	./temporal-cassandra-tool create -k temporal_cluster_c --rf 1
	./temporal-cassandra-tool -k temporal_cluster_c setup-schema -v 0.0
	./temporal-cassandra-tool -k temporal_cluster_c update-schema -d ./schema/cassandra/temporal/versioned

	@printf $(COLOR) "Install Elasticsearch schemas..."
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 setup-schema
# Delete indices if they exist (drop-index fails silently if index doesn't exist)
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 drop-index --index temporal_visibility_v1_dev_cluster_a --fail
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 drop-index --index temporal_visibility_v1_dev_cluster_b --fail
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 drop-index --index temporal_visibility_v1_dev_cluster_c --fail
# Create indices
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 create-index --index temporal_visibility_v1_dev_cluster_a
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 create-index --index temporal_visibility_v1_dev_cluster_b
	./temporal-elasticsearch-tool -ep http://127.0.0.1:9200 create-index --index temporal_visibility_v1_dev_cluster_c

##### Run server #####
DOCKER_COMPOSE_FILES     := -f ./develop/docker-compose/docker-compose.yml -f ./develop/docker-compose/docker-compose.$(GOOS).yml
DOCKER_COMPOSE_CDC_FILES := -f ./develop/docker-compose/docker-compose.cdc.yml -f ./develop/docker-compose/docker-compose.cdc.$(GOOS).yml
start-dependencies:
	docker compose $(DOCKER_COMPOSE_FILES) up

stop-dependencies:
	docker compose $(DOCKER_COMPOSE_FILES) down

start-dependencies-dual:
	docker compose $(DOCKER_COMPOSE_FILES) -f ./develop/docker-compose/docker-compose.secondary-es.yml up

stop-dependencies-dual:
	docker compose $(DOCKER_COMPOSE_FILES) -f ./develop/docker-compose/docker-compose.secondary-es.yml down

start-dependencies-cdc:
	docker compose $(DOCKER_COMPOSE_FILES) $(DOCKER_COMPOSE_CDC_FILES) up

stop-dependencies-cdc:
	docker compose $(DOCKER_COMPOSE_FILES) $(DOCKER_COMPOSE_CDC_FILES) down

start: start-sqlite

start-cass-es: temporal-server
	./temporal-server --config-file config/development-cass-es.yaml --allow-no-auth start

start-cass-archival: temporal-server
	./temporal-server --config-file config/development-cass-archival.yaml --allow-no-auth start

start-cass-es-dual: temporal-server
	./temporal-server --config-file config/development-cass-es-dual.yaml --allow-no-auth start

start-cass-es-custom: temporal-server
	./temporal-server --config-file config/development-cass-es-custom.yaml --allow-no-auth start

start-es-fi: temporal-server
	./temporal-server --config-file config/development-cass-es-fi.yaml --allow-no-auth start

start-mysql: start-mysql8

start-mysql8: temporal-server
	./temporal-server --config-file config/development-mysql8.yaml --allow-no-auth start

start-mysql-es: temporal-server
	./temporal-server --config-file config/development-mysql-es.yaml --allow-no-auth start

start-postgres: start-postgres12

start-postgres12: temporal-server
	./temporal-server --config-file config/development-postgres12.yaml --allow-no-auth start

start-sqlite: temporal-server
	./temporal-server --config-file config/development-sqlite.yaml --allow-no-auth start

start-sqlite-file: temporal-server
	./temporal-server --config-file config/development-sqlite-file.yaml --allow-no-auth start

start-xdc-cluster-a: temporal-server
	./temporal-server --config-file config/development-cluster-a.yaml --allow-no-auth start

start-xdc-cluster-b: temporal-server
	./temporal-server --config-file config/development-cluster-b.yaml --allow-no-auth start

start-xdc-cluster-c: temporal-server
	./temporal-server --config-file config/development-cluster-c.yaml --allow-no-auth start

start-jwt: temporal-server
	@./config/jwt/setup-keys.sh
	./temporal-server --config-file config/development-jwt.yaml start --service frontend --service internal-frontend --service history --service matching --service worker

##### Grafana #####
update-dashboards:
	@printf $(COLOR) "Update dashboards submodule from remote..."
	git submodule update --force --init --remote develop/docker-compose/grafana/provisioning/temporalio-dashboards

##### Auxiliary #####
gomodtidy:
	@printf $(COLOR) "go mod tidy..."
	@go mod tidy

update-dependencies:
	@printf $(COLOR) "Update dependencies (minor versions only) ..."
	@go get -u -t $(PINNED_DEPENDENCIES) ./...
	@go mod tidy

update-dependencies-major: $(GOMAJOR)
	@printf $(COLOR) "Major version upgrades available:"
	@$(GOMAJOR) list -major
	@echo ""
	@printf $(COLOR) "Update dependencies (major versions only) ..."
	@$(GOMAJOR) get -major all
	@go mod tidy

go-generate: $(MOCKGEN) $(GOIMPORTS) $(STRINGER) $(GOWRAP)
	@printf $(COLOR) "Process go:generate directives..."
	@PATH="$(ROOT)/$(LOCALBIN):$(PATH)" go generate ./...

ensure-no-changes:
	@printf $(COLOR) "Check for local changes..."
	@printf $(COLOR) "========================================================================"
	@git status --porcelain
	@test -z "`git status --porcelain`" || (printf $(COLOR) "========================================================================"; printf $(RED) "Above files are not regenerated properly. Regenerate them and try again."; git diff HEAD ; exit 1)
