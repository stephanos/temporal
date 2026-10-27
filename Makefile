ifneq ($(filter check-framework,$(MAKECMDGOALS)),)
ifneq ($(words $(MAKECMDGOALS)),1)
$(error check-framework runs alone, without system build targets)
endif
.PHONY: check-framework
check-framework:
	@mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/archived/framework/check -M framework.check.checkFramework
else
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
model_quote = '$(subst ','"'"',$(1))'
SOURCE_FIND = find . -type d \( -name .git -o -name .flow -o -name .build -o -name .worktrees -o -name .scala-build -o -name .bsp -o -name .venv -o -name .ruff_cache -o -name .agents $(foreach dir,$(NESTED_MODULE_DIRS),-o -path $(call model_quote,./$(dir))) \) -prune -o
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
	proto/internal/temporal/server/api/testpilot/v1/evidence.proto \
	proto/internal/temporal/server/api/testpilot/v1/expression.proto \
	proto/internal/temporal/server/api/testpilot/v1/instruction.proto \
	proto/internal/temporal/server/api/testpilot/v1/program.proto \
	proto/internal/temporal/server/api/testpilot/v1/run.proto \
	proto/internal/temporal/server/api/testpilot/v1/source.proto \
	proto/internal/temporal/server/api/testpilot/v1/value.proto

.PHONY: canary-build umpire-check-testpilot-protocol umpire umpire-check-live-tests umpire-check-backends fmt-model lint-model fix-model umpire-check-model umpire-gen-model

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

# The one host loop: umpire <question|explore|reduce|replay|repeat> --model <guest.wasm> ...
umpire:
	@printf $(COLOR) "Build the Umpire Runner command..."
	@mise exec -- go build -o ./.build/umpire ./tools/umpire/cmd/umpire
	@printf 'Built ./.build/umpire\n'

.PHONY: umpire-guest umpire-check-model-slice umpire-check-model-gaps
umpire-guest: $(API_BINPB)
	@/bin/bash "$(CURDIR)/model/scripts/guest.sh" umpire-temporal-guest "$(CURDIR)/model/temporal/hosts/go/umpire_temporal_guest.wasm"

umpire-check-model-slice:
	@set -eu; \
		temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT HUP INT TERM; \
		status=0; \
		CGO_ENABLED=0 \
			mise exec -- go test -v -count=1 -timeout 10m -tags 'test_dep integration' \
			./tests -run '^TestTestpilotModelSlice(ThroughUmpire)?$$' > "$$temporary" 2>&1 || status=$$?; \
		cat "$$temporary"; \
		if [ "$$status" -ne 0 ]; then exit "$$status"; fi; \
		awk '/^[[:space:]]*--- PASS:/ { passed[$$3] = 1 } \
			END { count = split("TestTestpilotModelSlice TestTestpilotModelSlice/Annotations TestTestpilotModelSlice/Transport TestTestpilotModelSlice/Reconciliation TestTestpilotModelSlice/Reconciliation/raw/Complete TestTestpilotModelSlice/Reconciliation/raw/Fail TestTestpilotModelSlice/Reconciliation/raw/wrong-result TestTestpilotModelSlice/Reconciliation/raw/wrong-failure TestTestpilotModelSlice/Reconciliation/raw/complete-instead-of-fail TestTestpilotModelSlice/Reconciliation/raw/Terminal/Complete TestTestpilotModelSlice/Reconciliation/raw/Terminal/Fail TestTestpilotModelSlice/Reconciliation/raw/Terminal/wrong-result TestTestpilotModelSlice/Reconciliation/raw/Terminal/wrong-failure TestTestpilotModelSlice/Reconciliation/raw/Terminal/complete-instead-of-fail TestTestpilotModelSlice/Reconciliation/raw/Terminal/response-mutated-history TestTestpilotModelSlice/Reconciliation/sdk/Complete TestTestpilotModelSlice/Reconciliation/sdk/Fail TestTestpilotModelSlice/Reconciliation/sdk/wrong-result TestTestpilotModelSlice/Reconciliation/sdk/wrong-failure TestTestpilotModelSlice/Reconciliation/sdk/complete-instead-of-fail TestTestpilotModelSlice/Reconciliation/sdk/Terminal/Complete TestTestpilotModelSlice/Reconciliation/sdk/Terminal/Fail TestTestpilotModelSlice/Reconciliation/sdk/Terminal/wrong-result TestTestpilotModelSlice/Reconciliation/sdk/Terminal/wrong-failure TestTestpilotModelSlice/Reconciliation/sdk/Terminal/complete-instead-of-fail TestTestpilotModelSlice/Reconciliation/sdk/Terminal/response-mutated-history TestTestpilotModelSliceThroughUmpire TestTestpilotModelSliceThroughUmpire/raw TestTestpilotModelSliceThroughUmpire/raw/UnheldCancelScope TestTestpilotModelSliceThroughUmpire/raw/Complete TestTestpilotModelSliceThroughUmpire/raw/Fail TestTestpilotModelSliceThroughUmpire/raw/RetryThenComplete TestTestpilotModelSliceThroughUmpire/raw/CancelBeforeWorkerStart TestTestpilotModelSliceThroughUmpire/raw/CancelAfterCompletion TestTestpilotModelSliceThroughUmpire/raw/HostRefusal TestTestpilotModelSliceThroughUmpire/sdk TestTestpilotModelSliceThroughUmpire/sdk/UnheldCancelScope TestTestpilotModelSliceThroughUmpire/sdk/Complete TestTestpilotModelSliceThroughUmpire/sdk/Fail TestTestpilotModelSliceThroughUmpire/sdk/RetryThenComplete TestTestpilotModelSliceThroughUmpire/sdk/CancelBeforeWorkerStart TestTestpilotModelSliceThroughUmpire/sdk/CancelAfterCompletion TestTestpilotModelSliceThroughUmpire/sdk/HostRefusal TestTestpilotModelSliceThroughUmpire/Controls TestTestpilotModelSliceThroughUmpire/Controls/complete-instead-of-fail TestTestpilotModelSliceThroughUmpire/Controls/wrong-result TestTestpilotModelSliceThroughUmpire/Controls/empty TestTestpilotModelSliceThroughUmpire/ProvisionLifecycle TestTestpilotModelSliceThroughUmpire/UnreachableEndpoint TestTestpilotModelSliceThroughUmpire/ReplayWithoutServer", required, " "); \
				for (i = 1; i <= count; i++) if (!passed[required[i]]) { \
					printf "Required Model slice test did not pass: %s\n", required[i]; failed = 1 \
				} exit failed }' "$$temporary"

umpire-check-model-gaps:
	@set -eu; \
		temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT HUP INT TERM; \
		status=0; \
		CGO_ENABLED=0 \
			mise exec -- go test -v -count=1 -timeout 20m -tags 'test_dep integration' \
			./tests -run '^TestTestpilotModelGap(s|LoggerPreservesFailuresWithoutPrivateOutput)$$' > "$$temporary" 2>&1 || status=$$?; \
		cat "$$temporary"; \
		if [ "$$status" -ne 0 ]; then exit "$$status"; fi; \
		awk '/^--- PASS:/ { passed[$$3] = 1 } \
			END { count = split("TestTestpilotModelGaps TestTestpilotModelGapLoggerPreservesFailuresWithoutPrivateOutput", required, " "); \
				for (i = 1; i <= count; i++) if (!passed[required[i]]) { \
					printf "Required Model gap test did not pass: %s\n", required[i]; failed = 1 \
				} exit failed }' "$$temporary"

umpire-check-live-tests: umpire-check-model-slice umpire-check-model-gaps
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

# The backend agreement: Quint reads the lifted IR and is held to Go's reading of it. The tests
# pin their tools and fail when one is missing; the pins are in tools/umpire/export/tools_test.go.
# UMPIRE_BACKENDS_OUT names a directory that
# keeps every export, dump and report. Run from the package's directory, the receipts are printed.
# Warm the pinned Quint package in npm's cache. Apalache is downloaded by `quint verify` on first use.
.PHONY: umpire-install-backends
umpire-install-backends:
	@mise exec -- tools/umpire/export/quint.sh --version

umpire-check-backends:
	@UMPIRE_BACKENDS=require CC="$${CC:-/usr/bin/clang}" mise exec -- go test -C ./tools/umpire/export -count=1 -timeout 30m -tags test_dep

# Archived Scala targets are unmaintained and are not expected to build. Their explicit source
# roots keep authoring, the IR generator (export) and the check separate under model/archived.
MODEL_ROOT := model/archived
MODEL_CLI := mise exec -- scala-cli
MODEL_GATE_ARGS ?=
# Scalafix 0.14.9 reads JDK25's URLClassPath.path; JDK27 renamed that field to searchPath.
MODEL_SCALAFIX = $(MODEL_CLI) --power fix --jvm 25 --enable-built-in=false \
	--scalafix-conf "$(CURDIR)/$(MODEL_ROOT)/.scalafix.conf" \
	--scalac-option -Wunused:all --suppress-outdated-dependency-warning
MODEL_FEATURE_ROOT := $(MODEL_ROOT)/temporal/features
MODEL_DISCOVERED_FEATURES := $(shell find $(call model_quote,$(MODEL_FEATURE_ROOT)) -type d \( -name '.*' -o -iname shared -o -iname product -o -iname protocol -o -iname generated \) -prune -o -type f -name '*.scala' ! -name '*.test.scala' -print | awk -v root=$(call model_quote,$(MODEL_FEATURE_ROOT)/) 'function normalized(name) { gsub(/_/,"",name); return tolower(name) } { relative=substr($$0,length(root)+1); n=split(relative,parts,"/"); file=parts[n]; sub(/\.scala$$/,"",file); file=normalized(file); folder=normalized(parts[n-1]); kind=normalized(parts[1]); separator=index(parts[n-1],"_"); reversed=normalized(substr(parts[n-1],separator+1) substr(parts[n-1],1,separator-1)); if ((n==2 && (file==folder || (separator && file==reversed))) || (n==3 && (file==folder || file==folder kind))) { feature=parts[1]; if (n==3) { forms[feature]=1; feature=feature "/" parts[2] } candidates[feature]=1 } } END { for (feature in candidates) if (!(feature in forms)) print feature }' | sort -u)
FEATURES ?= $(MODEL_DISCOVERED_FEATURES)
export FEATURES
MODEL_UNKNOWN_FEATURES := $(filter-out $(MODEL_DISCOVERED_FEATURES),$(FEATURES))
ifneq ($(strip $(MODEL_UNKNOWN_FEATURES)),)
$(error feature folder not found: $(MODEL_UNKNOWN_FEATURES))
endif
MODEL_FEATURE_PATHS := $(addprefix $(MODEL_FEATURE_ROOT)/,$(FEATURES))
MODEL_FLAT_KIND_PARENTS := $(foreach feature,$(filter-out %/%,$(FEATURES)),$(if $(findstring _,$(feature)),$(foreach kind,$(wildcard $(MODEL_FEATURE_ROOT)/$(firstword $(subst _, ,$(feature)))),$(kind)/)))
MODEL_FEATURE_PARENTS := $(filter-out $(MODEL_FEATURE_ROOT)/,$(sort $(dir $(MODEL_FEATURE_PATHS)) $(MODEL_FLAT_KIND_PARENTS)))
MODEL_FEATURE_SHARED := $(filter-out $(addprefix $(MODEL_FEATURE_ROOT)/,$(MODEL_DISCOVERED_FEATURES)),$(patsubst %/,%,$(wildcard $(addsuffix */,$(MODEL_FEATURE_PARENTS)))))
# The framework's sources the Models compile against: the DSL and the generic realization core, not
# the tooling beside them.
MODEL_FRAMEWORK_SOURCES := $(addprefix $(MODEL_ROOT)/,project.scala framework/dsl framework/realize)
MODEL_SOURCES := $(MODEL_FRAMEWORK_SOURCES) \
	$(wildcard $(MODEL_ROOT)/temporal/*.scala) \
	$(addprefix $(MODEL_ROOT)/temporal/,foundations actors capabilities realize) \
	$(wildcard $(addsuffix *.scala,$(MODEL_FEATURE_PARENTS))) $(MODEL_FEATURE_SHARED) $(MODEL_FEATURE_PATHS)
MODEL_UNLISTED_PACKAGES := $(subst /,.,$(addprefix temporal.features.,$(filter-out $(FEATURES),$(MODEL_DISCOVERED_FEATURES))))
MODEL_SOURCE_ARGS := $(foreach source,$(MODEL_SOURCES),$(call model_quote,$(source)))
MODEL_EXCLUDED_TESTS := $(if $(MODEL_UNLISTED_PACKAGES),$(shell find $(foreach source,$(filter-out $(MODEL_FRAMEWORK_SOURCES),$(MODEL_SOURCES)),$(call model_quote,$(source))) -name '*.test.scala' -exec grep -l -F $(foreach pkg,$(MODEL_UNLISTED_PACKAGES),-e $(call model_quote,$(pkg))) {} +))
MODEL_SOURCE_OPTIONS := $(foreach file,$(MODEL_EXCLUDED_TESTS),--exclude $(call model_quote,$(CURDIR)/$(file)))
# Temporal Model sources have package-only kind headers. Bloop rejects them under -Werror even
# when scala-cli exits 0, so their package, compile, test and scalafix calls keep --server=false.
# Direct DSL/export package, compile and run calls also keep it: Bloop can return 0 for -Werror
# failures, and Make reads exit codes alone. The checks have no -Werror; samples and suite runners
# read compiler diagnostics.
model_scalafix_files = $(foreach source,$(1),--scalafix-arg=--files --scalafix-arg="$(CURDIR)/$(source)")
MODEL_SCALAFIX_FILES = $(call model_scalafix_files,$(MODEL_SOURCES))
# The lifter's and the checks' Temporal builds: each tool's framework half, whose project.scala is
# the build's own, and its Temporal half.
MODEL_EXPORT := $(MODEL_ROOT)/framework/export $(MODEL_ROOT)/temporal/export
MODEL_CHECK := $(MODEL_ROOT)/framework/check $(MODEL_ROOT)/temporal/check
# scala-cli hands scalafix every file under a project's directory, so the IR generator names its own
# sources to leave out the fixtures under testdata, which its build excludes.
MODEL_IRGEN_SOURCES := $(wildcard $(addsuffix /*.scala,$(MODEL_EXPORT)))
# The IR generator's fixtures that lift are a build of their own against the packaged Models. The
# refusal fixtures stay outside scalafix: unsupported's `var` and `while` are what it must refuse,
# and werror and crossed must not compile, so RemoveUnused has nothing to read. The fixtures' build
# makes warnings errors, and -Wunused:all also warns of the parameters RemoveUnused keeps
# (params = false), so the lint keeps warnings as warnings.
MODEL_LIFTS := $(MODEL_ROOT)/framework/export/testdata/lifts
# The tools' directories scalafmt formats beside the Models.
MODEL_TOOL_DIRS := $(foreach tool,export check,$(MODEL_ROOT)/framework/$(tool) $(MODEL_ROOT)/temporal/$(tool))
MODEL_LIFTS_SCALAFIX = $(MODEL_SCALAFIX) --scalac-option -Werror:false \
	$(call model_scalafix_files,$(wildcard $(MODEL_LIFTS)/*.scala))
MODEL_BUILD := $(MODEL_ROOT)/build
MODEL_PROTO_JARS := $(MODEL_BUILD)/ir-scalapb.jar $(MODEL_BUILD)/api-scalapb.jar
MODEL_REIFY_PLUGIN := $(MODEL_BUILD)/reify.jar
MODEL_REIFY_INPUTS := $(wildcard $(MODEL_ROOT)/framework/reify/plugin/*.scala) \
	$(wildcard $(MODEL_ROOT)/framework/reify/body/*.scala) \
	$(MODEL_ROOT)/framework/inspect/LiftableSubset.scala \
	$(MODEL_ROOT)/framework/inspect/LiftError.scala \
	$(MODEL_ROOT)/framework/reify/plugin/resources/plugin.properties
MODEL_REIFY_DIGEST = $(shell shasum -a 256 $(call model_quote,$(MODEL_REIFY_PLUGIN)) | awk '{print $$1}')
MODEL_REIFY_OPTION = --scalac-option $(call model_quote,-Xplugin:$(CURDIR)/$(MODEL_REIFY_PLUGIN)) \
	--scalac-option -P:umpire-reify:mode=realization \
	--scalac-option $(call model_quote,-P:umpire-reify:cache-key=$(MODEL_REIFY_DIGEST))
MODEL_JAR := $(MODEL_BUILD)/model-scala.jar
MODEL_CLASSPATH := $(MODEL_BUILD)/model-scala.classpath
# The gate is one Scala program, model/temporal/check; its own arguments follow.
MODEL_GATE = $(MODEL_CLI) run --server=false --suppress-outdated-dependency-warning $(MODEL_CHECK) -M temporal.check.run --
# The gate's own tests run it against stand-in tools, so they need no jar and no real tool.
MODEL_GATE_TEST = $(MODEL_CLI) test --server=false --suppress-outdated-dependency-warning $(MODEL_CHECK)

# The IR and linked-API classes the Models compile against. Packaging them runs protoc and the
# linked-API descriptors through Go (`make protoc` territory, owner-held), so make never does it on
# its own: a missing jar fails naming the opt-in target, and the model targets read the cached jars.
# gen-model-api packages both when their inputs changed; the canonical gate (umpire-check-model,
# umpire-gen-model) still regenerates them itself.
$(MODEL_PROTO_JARS):
	@printf '%s is missing; run make gen-model-api once (the held linked-API generation)\n' $@ >&2; exit 1

.PHONY: gen-model-api
gen-model-api:
	@printf $(COLOR) "Package model IR and API classes..."
	@$(MODEL_GATE) --generate-api --if-stale

# The Models retain their authoring identities through this compiler plugin. It is a Model build
# input, not a framework dependency: the framework-alone boundary below deliberately omits it.
$(MODEL_REIFY_PLUGIN): $(MODEL_BUILD)/ir-scalapb.jar $(MODEL_REIFY_INPUTS)
	@printf $(COLOR) "Package the Model reify compiler plugin..."
	@set -eu; \
		plugin_tmp=$$(mktemp $(call model_quote,$(MODEL_BUILD)/reify.XXXXXX.jar)); \
		trap 'rm -f "$$plugin_tmp"' EXIT; \
		$(MODEL_CLI) --power package --server=false --assembly --preamble=false \
			--main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 \
			$(MODEL_ROOT)/framework/reify/plugin -f -o "$$plugin_tmp"; \
		test -s "$$plugin_tmp" || { printf '%s\n' '$@ is missing: scala-cli did not package the reify compiler plugin' >&2; exit 1; }; \
		mv "$$plugin_tmp" $(call model_quote,$@)

# The Models' TASTy, packaged as the gate's "package the Models' TASTy" step packages it.
.PHONY: model-source-selection
$(MODEL_BUILD)/model-sources: model-source-selection $(MODEL_REIFY_PLUGIN)
	@mkdir -p $(MODEL_BUILD)
	@printf '%s\n' $(call model_quote,$(MODEL_SOURCES) $(MODEL_SOURCE_OPTIONS) $(MODEL_REIFY_OPTION)) > $@.tmp
	@cmp -s $@.tmp $@ && rm $@.tmp || mv $@.tmp $@

$(MODEL_JAR): $(MODEL_PROTO_JARS) $(MODEL_REIFY_PLUGIN) $(MODEL_BUILD)/model-sources $(shell find $(MODEL_SOURCE_ARGS) -name '*.scala')
	@printf $(COLOR) "Package the Models' TASTy..."
	@set -eu; \
		jar_tmp=$(call model_quote,$@.tmp.jar); classpath_tmp=$(call model_quote,$(MODEL_CLASSPATH).tmp); \
		trap 'rm -f "$$jar_tmp" "$$classpath_tmp"' EXIT; \
		$(MODEL_CLI) compile --server=false --suppress-outdated-dependency-warning --print-class-path --compilation-output "$$jar_tmp" $(MODEL_REIFY_OPTION) $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS) > "$$classpath_tmp"; \
		test -s "$$jar_tmp"; test -s "$$classpath_tmp"; \
		mv "$$jar_tmp" $(call model_quote,$@); \
		mv "$$classpath_tmp" $(call model_quote,$(MODEL_CLASSPATH))

# Lifts the selected features from the packaged Models into LIFT_OUT (default: a fresh directory
# under model/build/history) and says, per feature, whether the lift equals its checked-in IR.
MODEL_LIFT_NAMES = $(subst _,-,$(subst /,-,$(FEATURES)))
.PHONY: lift-model
lift-model: $(MODEL_JAR)
	@printf $(COLOR) "Lift the selected features' IR..."
	@set -eu; out=$${LIFT_OUT:-}; \
		if [ -z "$$out" ]; then mkdir -p $(MODEL_BUILD)/history; out=$$(mktemp -d $(MODEL_BUILD)/history/lift.XXXXXX); fi; \
		mkdir -p "$$out"; \
		$(MODEL_CLI) run --server=false --suppress-outdated-dependency-warning $(MODEL_EXPORT) --main-class framework.export.lift -- \
			--ir $(call model_quote,$(CURDIR)/$(MODEL_JAR))=model/archived/ $(MODEL_CLASSPATH) "$$out" $(MODEL_LIFT_NAMES); \
		for feature in $(FEATURES); do \
			name=$$(printf '%s' "$$feature" | tr '/_' '--'); \
			if cmp -s "$$out/$$name.json" $(MODEL_FEATURE_ROOT)/"$$feature"/generated/ir.json; then state=current; else state='differs from its checked-in IR'; fi; \
			printf 'feature %s: %s/%s.json %s\n' "$$feature" "$$out" "$$name" "$$state"; \
		done

.PHONY: compile-model
compile-model: $(MODEL_PROTO_JARS) $(MODEL_REIFY_PLUGIN)
	@printf $(COLOR) "Compiling model files..."
	@$(MODEL_CLI) compile --server=false $(MODEL_REIFY_OPTION) $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS)

.PHONY: test-model model model-mini
test-model: $(MODEL_JAR)
	@$(MODEL_CLI) run --server=false $(MODEL_CHECK) -M framework.check.nativeModelTests
	@$(MODEL_CLI) test --server=false $(MODEL_ROOT)/framework/dsl $(MODEL_ROOT)/framework/realize \
		--test-only framework.realize.RealizationConversionTest

# The loop reads the cached jars (gen-model-api); Go checks the held protobuf generation leaves
# stale are skipped and named.
model: $(MODEL_PROTO_JARS)
	@$(MODEL_CLI) run --server=false --suppress-outdated-dependency-warning $(MODEL_CHECK) -M temporal.check.modelLoop -- $(MODEL_GATE_ARGS)

model-mini:
	@$(MAKE) --no-print-directory model FEATURES=activity_mini

fmt-model:
	@printf $(COLOR) "Formatting model files..."
	@$(MODEL_CLI) fmt --scalafmt-conf $(MODEL_ROOT)/.scalafmt.conf $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS) $(MODEL_TOOL_DIRS)

# The four scalafix runs build separate projects, so they run side by side; output-sync prints each
# run's output whole, after its title. The sugar check builds the gate, which lint-model-check builds
# with other options, so it runs after them.
MODEL_LINTS := lint-model-models lint-model-irgen lint-model-irgen-lifts lint-model-check
.PHONY: $(MODEL_LINTS) lint-model-syntax lint-model-conventions

lint-model: $(MODEL_PROTO_JARS) $(MODEL_JAR)
	@printf $(COLOR) "Checking model formatting..."
	@$(MODEL_CLI) fmt --scalafmt-conf $(MODEL_ROOT)/.scalafmt.conf --check $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS) $(MODEL_TOOL_DIRS)
	@$(MAKE) --no-print-directory -j4 --output-sync=target $(MODEL_LINTS)
	@$(MAKE) --no-print-directory lint-model-syntax
	@$(MAKE) --no-print-directory lint-model-conventions

# Unlike semantic lifting, this lint reads framework and test TASTy as well as production Models.
lint-model-conventions: $(MODEL_PROTO_JARS) $(MODEL_JAR)
	@printf $(COLOR) "Checking typed model conventions and imports..."
	@$(MODEL_CLI) --power package --server=false --suppress-outdated-dependency-warning --library --test $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS) -f -o $(MODEL_BUILD)/model-conventions.jar
	@$(MODEL_CLI) compile --server=false --suppress-outdated-dependency-warning --test --print-classpath $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS) > $(MODEL_BUILD)/model-conventions.classpath
	@$(MODEL_CLI) --power package --server=false --suppress-outdated-dependency-warning --library --test $(MODEL_EXPORT) -f -o $(MODEL_BUILD)/irgen-conventions.jar
	@$(MODEL_CLI) compile --server=false --suppress-outdated-dependency-warning --test --print-classpath $(MODEL_EXPORT) > $(MODEL_BUILD)/irgen-conventions.classpath
	@$(MODEL_CLI) --power package --suppress-outdated-dependency-warning --library --test $(MODEL_CHECK) -f -o $(MODEL_BUILD)/check-conventions.jar
	@$(MODEL_CLI) compile --suppress-outdated-dependency-warning --test --print-classpath $(MODEL_CHECK) > $(MODEL_BUILD)/check-conventions.classpath
	@$(MODEL_CLI) run --server=false --suppress-outdated-dependency-warning $(MODEL_EXPORT) -M framework.export.lintConventions -- $(MODEL_BUILD)/model-conventions.jar=model/archived/,$(MODEL_BUILD)/irgen-conventions.jar=model/archived/framework/export/,$(MODEL_BUILD)/check-conventions.jar=model/archived/framework/check/ $(MODEL_BUILD)/model-conventions.classpath $(MODEL_BUILD)/irgen-conventions.classpath $(MODEL_BUILD)/check-conventions.classpath

lint-model-models: $(MODEL_PROTO_JARS)
	@printf $(COLOR) "Linting model files..."
	@$(MODEL_SCALAFIX) --server=false $(MODEL_SCALAFIX_FILES) --check $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS)

lint-model-irgen:
	@printf $(COLOR) "Linting the IR generator..."
	@$(MODEL_SCALAFIX) $(call model_scalafix_files,$(MODEL_IRGEN_SOURCES)) --check $(MODEL_EXPORT)

lint-model-irgen-lifts:
	@printf $(COLOR) "Linting the IR generator's lifting fixtures..."
	@cd $(MODEL_LIFTS) && $(MODEL_LIFTS_SCALAFIX) --check .

lint-model-check:
	@printf $(COLOR) "Linting the check..."
	@$(MODEL_SCALAFIX) --server=false --check $(MODEL_CHECK)

# The framework's boundary (fn-169.1): model/framework names no Temporal package and takes no
# Temporal jar or source (model/framework/check/lint/Boundary.scala), and its core and each of its
# tools compile on their own, with no Temporal API, Models or tools on their classpath.
.PHONY: check-model-boundary
check-model-boundary: $(MODEL_BUILD)/ir-scalapb.jar
	@printf $(COLOR) "Checking the framework's boundary..."
	@$(MODEL_GATE) --check-boundary
	@$(MODEL_CLI) compile --server=false --test $(MODEL_ROOT)/framework/dsl $(MODEL_ROOT)/framework/realize
	@$(MODEL_CLI) compile --server=false --test $(MODEL_ROOT)/framework/export
	@$(MODEL_CLI) compile --server=false --test $(MODEL_ROOT)/framework/check

# Sugar is defined only in a Syntax.scala, each definition documented with its `Core form:`, and no
# core file of the framework or the IR generator imports or names it. Comments are `//` lines in
# every Scala file of model/, its tests and fixtures too. Temporal rule blocks use canonical braces.
lint-model-syntax:
	@printf $(COLOR) "Checking the model's sugar and comments..."
	@$(MODEL_GATE) --check-syntax
	@$(MODEL_GATE) --check-comments
	@$(MODEL_GATE) --check-block-form

# Applies the scalafix rewrites; findings without a rewrite (e.g. DisableSyntax) still fail.
fix-model: $(MODEL_PROTO_JARS) $(MODEL_JAR)
	@printf $(COLOR) "Applying model lint fixes..."
	@$(MODEL_SCALAFIX) --server=false $(MODEL_SCALAFIX_FILES) $(MODEL_SOURCE_ARGS) $(MODEL_SOURCE_OPTIONS)
	@$(MODEL_SCALAFIX) $(call model_scalafix_files,$(MODEL_IRGEN_SOURCES)) $(MODEL_EXPORT)
	@cd $(MODEL_LIFTS) && $(MODEL_LIFTS_SCALAFIX) .
	@$(MODEL_SCALAFIX) --server=false $(MODEL_CHECK)

# The gate packages the IR classes itself when the schema changed.
umpire-check-model:
	@printf $(COLOR) "Check model IR..."
	@$(MODEL_GATE_TEST)
	@$(MODEL_GATE) $(MODEL_GATE_ARGS)

umpire-gen-model:
	@printf $(COLOR) "Generate model IR..."
	@$(MODEL_GATE_TEST)
	@$(MODEL_GATE) --update $(MODEL_GATE_ARGS)

# Seals a batch that keeps the Models' meaning but changes identities (model/README.md, "Sealing a
# batch"): capture a baseline into a new directory before the batch's first edit, then compare the
# batch's end with it under the batch's rename map. STEPS_ONLY=1 compares the step tables alone,
# while the checked-in IR is stale (Batch 1d). DUMPER names the table pins' interpreter dumper
# (model/testdata/table-pins/README.md) to capture and compare the IR's interpreter tables too.
.PHONY: umpire-capture-baseline umpire-check-equivalence
umpire-capture-baseline: $(MODEL_PROTO_JARS) $(MODEL_REIFY_PLUGIN)
	@test -n "$(OUT)" || { echo "usage: make umpire-capture-baseline OUT=<new directory> [DUMPER=<path>]"; exit 2; }
	@printf $(COLOR) "Capture a model baseline into $(OUT)..."
	@$(MODEL_GATE) --capture-baseline "$(OUT)" $(if $(DUMPER),--dumper "$(DUMPER)")

umpire-check-equivalence: $(MODEL_PROTO_JARS) $(MODEL_REIFY_PLUGIN)
	@test -n "$(BASE)" && test -n "$(MAP)" || { echo "usage: make umpire-check-equivalence BASE=<baseline> MAP=<rename map> [STEPS_ONLY=1] [DUMPER=<path>]"; exit 2; }
	@printf $(COLOR) "Check the model against the baseline $(BASE)..."
	@$(MODEL_GATE) --check-equivalence "$(BASE)" "$(MAP)" $(if $(STEPS_ONLY),--steps-only) $(if $(DUMPER),--dumper "$(DUMPER)")

# Removes scala-cli build state (.bsp/, .scala-build/) anywhere in the checkout and the given
# scratch directories under .flow/tmp (UMPIRE_SCRATCH="fn112-3 fn114-1"). Agents use this target
# instead of running rm directly; model/build and evidence they did not name are left alone.
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
# Flow evidence, testdata and nested modules, tracked and untracked, against the merge base.
# $(1) is the patch file.
define go-only-patch
base=$$(git merge-base HEAD "$(GOLANGCI_LINT_BASE_REV)"); \
excludes=$$(git ls-files --cached --others --exclude-standard -- '*/go.mod' \
  | sed 's|/go.mod$$|/**|; s|^|:(exclude,glob)|'); \
git diff --no-ext-diff --no-renames "$$base" -- '*.go' ':(exclude,glob).flow/**' ':(exclude,glob)**/testdata/**' $$excludes > "$(1)"; \
git ls-files --others --exclude-standard -- '*.go' ':(exclude,glob).flow/**' ':(exclude,glob)**/testdata/**' $$excludes | \
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
		git diff --no-renames --name-only "$$base" -- '*.go' ':(exclude,glob).flow/**' ':(exclude,glob)**/testdata/**' $$excludes; \
		git ls-files --others --exclude-standard -- '*.go' ':(exclude,glob).flow/**' ':(exclude,glob)**/testdata/**' $$excludes; \
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
endif

##### model: Umpire in Rust (fn-210) #####
RUST_MODEL_CARGO := MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo
RUST_MODEL_FRAMEWORK = helper=$$(/bin/bash "$(CURDIR)/model/umpire/scripts/xtask.sh") && "$$helper"

.PHONY: model-check model-quick model-fmt model-update model-time model-validate
model-quick:
	@$(RUST_MODEL_CARGO) run --manifest-path "$(CURDIR)/model/Cargo.toml" --package temporal-xtask --bin temporal-xtask --profile dev -- quick

model-check: override export UMPIRE_QUICK_CACHE = 0
model-check: override export UMPIRE_CACHE_DISABLE = 1
model-check: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(API_BINPB)
	@$(RUST_MODEL_CARGO) run --manifest-path "$(CURDIR)/model/Cargo.toml" --package temporal-xtask --bin temporal-xtask --profile dev -- gate

model-fmt:
	@$(RUST_MODEL_FRAMEWORK) fmt
	@$(RUST_MODEL_CARGO) fmt --manifest-path "$(CURDIR)/model/Cargo.toml" --all

model-update:
	@$(RUST_MODEL_FRAMEWORK) update
	@INSTA_UPDATE=always TRYBUILD=overwrite $(RUST_MODEL_CARGO) run --manifest-path "$(CURDIR)/model/Cargo.toml" --package temporal-xtask --bin temporal-xtask --profile dev -- test-run

model-time:
	@printf $(COLOR) "Time the model feedback loop (fn-210 R30)..."
	@$(CURDIR)/model/scripts/time-gate.sh

model-validate:
	@$(MAKE) UMPIRE_BACKENDS=require model-time
