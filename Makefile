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
# reducing binary size by 16MB since we only use the REST client (storage.NewClient), not the
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

ifeq ($(shell uname -s),Darwin)
LEAN_SDKROOT := $(shell xcrun --show-sdk-path)
LEAN_CLANG := $(shell xcrun --find clang)
LEAN_CLANG_DIR := $(dir $(LEAN_CLANG))
export CC := $(LEAN_CLANG)
export CXX := $(shell xcrun --find clang++)
export SDKROOT := $(LEAN_SDKROOT)
LEAN_LAKE := SDKROOT="$(LEAN_SDKROOT)" mise exec -- sh -c 'PATH="$(LEAN_CLANG_DIR):$$PATH"; exec lake "$$@"' lean-lake
else
LEAN_LAKE := mise exec -- lake
endif

UMPIRE_GENMODELS := go run -tags test_dep ./cmd/umpire-genmodels
UMPIRE3_ROOT := tools/umpire3
UMPIRE3_MODEL_ROOT := $(UMPIRE3_ROOT)/model
UMPIRE3_LEAN_VERSION := $(shell sed -e 's|leanprover/lean4:v||' $(UMPIRE3_MODEL_ROOT)/lean-toolchain)
UMPIRE3_MANIFEST := $(UMPIRE3_ROOT)/testdata/generated/empty-manifest.json
UMPIRE3_DEV_COMMAND := go run -tags test_dep ./$(UMPIRE3_ROOT)/cmd/umpire3-dev
UMPIRE3_MANIFEST_COMMAND := $(UMPIRE3_DEV_COMMAND) manifest -lean-version $(UMPIRE3_LEAN_VERSION)
UMPIRE3_CATALOG := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/catalog.json
UMPIRE3_IDENTIFIERS := $(UMPIRE3_ROOT)/protocol/catalog/catalog_ids.gen.go
UMPIRE3_AUTHOR_FACADE := $(UMPIRE3_ROOT)/scenario/catalog.gen.go
UMPIRE3_EXPERIMENT_SCHEMA := $(UMPIRE3_ROOT)/protocol/experiment/testdata/generated/experiment.schema.json
UMPIRE3_MONITORS := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/monitor-programs.json
UMPIRE3_OBSERVATIONS := $(UMPIRE3_ROOT)/execution/observation/testdata/generated/programs.json
UMPIRE3_COMPOSITION := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/composition.json
UMPIRE3_PARITY := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/parity-ledger.json
UMPIRE3_COVERAGE := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/coverage-denominator.json
UMPIRE3_FAMILY_DEPENDENCIES := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/family-dependencies.json
UMPIRE3_FINITE_REPLAY := $(UMPIRE3_ROOT)/checker/finite/testdata/generated/finite-replay-catalog.json
UMPIRE3_NEXUS_FIRST_ORDER := $(UMPIRE3_ROOT)/checker/finite/testdata/generated/nexus-cancellation.first-order.json
UMPIRE3_NEXUS_MUTATED_FIRST_ORDER := $(UMPIRE3_ROOT)/checker/finite/testdata/generated/nexus-cancellation-mutated.first-order.json
UMPIRE3_NEXUS_ATTEMPT := $(UMPIRE3_ROOT)/checker/finite/testdata/generated/nexus-cancellation.attempt.json
UMPIRE3_NEXUS_MUTATED_ATTEMPT := $(UMPIRE3_ROOT)/checker/finite/testdata/generated/nexus-cancellation-mutated.attempt.json
UMPIRE3_TASK_DELIVERY_TEMPORAL := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/task-delivery-progress.temporal.json
UMPIRE3_TASK_DELIVERY_MUTATED_TEMPORAL := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/task-delivery-progress-mutated.temporal.json
UMPIRE3_NEXUS_EXPERIMENT := $(UMPIRE3_ROOT)/testdata/generated/nexus-cancellation.json
UMPIRE3_UPDATE_EXPERIMENT := $(UMPIRE3_ROOT)/testdata/generated/update-lifecycle.json
UMPIRE3_RELEASE := $(UMPIRE3_ROOT)/assurance/release/testdata/generated/umpire3-1.3.json
UMPIRE3_MUTATION_AUDIT := $(UMPIRE3_ROOT)/mutation/testdata/retained/cross-layer-mutation.audit.json
UMPIRE3_SEMANTIC_MUTATION_AUDIT := $(UMPIRE3_ROOT)/assurance/audit/mutation/testdata/generated/semantic-mutations.audit.json
UMPIRE3_RESILIENCE_AUDIT := $(UMPIRE3_ROOT)/assurance/audit/resilience/testdata/generated/control-plane.audit.json
UMPIRE3_NEXUS_PROOF_MANIFEST := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/nexus-proof-manifest.json
UMPIRE3_NEXUS_MUTATION_REJECTION_PROOF_MANIFEST := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/nexus-mutation-rejection-proof-manifest.json
UMPIRE3_NEXUS_EXACT_MUTATION_PROOF_MANIFEST := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/nexus-exact-mutation-proof-manifest.json
UMPIRE3_UPDATE_PROOF_MANIFEST := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/update-proof-manifest.json
UMPIRE3_EXPORT_COMMAND := $(UMPIRE3_DEV_COMMAND) export
UMPIRE3_API_COMMAND := $(UMPIRE3_DEV_COMMAND) api
UMPIRE_GEN_LEAN_API_COMMAND := mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api
UMPIRE_GEN_TESTS_COMMAND := mise exec -- lake exe umpire-gen-tests
UMPIRE_GEN_REGRESSION_VIEWS_COMMAND := mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-regression-views
UMPIRE_GEN_LEAN_DYNAMIC_CONFIG_CATALOG_COMMAND := mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog
UMPIRE_EXPORT_PROTO_DESCRIPTORS_COMMAND := mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-export-proto-descriptors
UMPIRE_ARTIFACT_COMMAND := mise exec -- go run ./tools/umpire/cmd/umpire-artifact
UMPIRE_REGRESSION_INSPECTOR := temporal-model-inspect
UMPIRE_REGRESSION_FIXTURES := \
	workflow-nexus.query.exact-action-caller-closure:Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json \
	switch.query.exact-action:Umpire/Examples/testdata/switch-experiment-spec.json
UMPIRE_GEN_LEAN_API_ARGS = \
	--descriptor $(UMPIRE_PUBLIC_BINPB) \
	--descriptor $(API_BINPB) \
	--descriptor $(INTERNAL_BINPB) \
	--descriptor $(CHASM_BINPB) \
	--lean-root Temporal \
	--output-root model
UMPIRE3_API_DESCRIPTOR := $(UMPIRE3_MODEL_ROOT)/Temporal/API/Generated/descriptor-manifest.json
UMPIRE3_PROTOCOL_DESCRIPTOR := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/descriptor-manifest.json
UMPIRE3_MIGRATION_LEDGER := $(UMPIRE3_ROOT)/assurance/migration/testdata/generated/ledger.json
UMPIRE3_MIGRATION_COMMAND := $(UMPIRE3_DEV_COMMAND) migration
UMPIRE3_FAMILY_COMMAND := $(UMPIRE3_DEV_COMMAND) family
UMPIRE3_COMMAND := go run -tags test_dep ./$(UMPIRE3_ROOT)/cmd/umpire3
UMPIRE3_VEIL_COMMAND := go run -tags test_dep ./$(UMPIRE3_ROOT)/cmd/umpire3-veil
UMPIRE3_NATIVE_COMMAND := go run -tags test_dep ./$(UMPIRE3_ROOT)/cmd/umpire3-native
UMPIRE3_TRACE_REPLAY_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_trace_replay
UMPIRE3_VEIL_SOUND_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_veil_sound
UMPIRE3_VEIL_MUTATED_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_veil_mutated
UMPIRE3_VEIL_SOUND_PROOF_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_veil_sound_proof
UMPIRE3_VEIL_SOUND_TRUSTED_PROOF_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_veil_sound_trusted_proof
UMPIRE3_VEIL_BINDINGS := $(UMPIRE3_ROOT)/checker/veil/testdata/generated
UMPIRE3_VEIL_SOUND_BINDING := $(UMPIRE3_VEIL_BINDINGS)/nexus-cancellation-sound.json
UMPIRE3_VEIL_MUTATED_BINDING := $(UMPIRE3_VEIL_BINDINGS)/nexus-cancellation-mutated.json
UMPIRE3_VEIL_TRUSTED_BINDING := $(UMPIRE3_VEIL_BINDINGS)/nexus-cancellation-sound-trusted.json
UMPIRE3_VEIL_RESULTS := $(UMPIRE3_ROOT)/checker/veil/testdata/retained
UMPIRE3_VEIL_SOUND_RESULT := $(UMPIRE3_VEIL_RESULTS)/nexus-cancellation-sound-concrete.json
UMPIRE3_VEIL_MUTATED_RESULT := $(UMPIRE3_VEIL_RESULTS)/nexus-cancellation-mutated-concrete.json
UMPIRE3_VEIL_SYMBOLIC_RESULT := $(UMPIRE3_VEIL_RESULTS)/nexus-cancellation-sound-symbolic.json
UMPIRE3_VEIL_INVARIANT_RESULT := $(UMPIRE3_VEIL_RESULTS)/nexus-cancellation-sound-invariant.json
UMPIRE3_VEIL_INVARIANT_TRUSTED_RESULT := $(UMPIRE3_VEIL_RESULTS)/nexus-cancellation-sound-invariant-trusted.json
UMPIRE3_TEMPORAL_LASSO_REPLAY_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_temporal_lasso_replay
UMPIRE3_NATIVE_BINDING := $(UMPIRE3_MODEL_ROOT)/Umpire3/Generated/NexusCertificateBinding.lean
UMPIRE3_NATIVE_CERTIFICATE_BIN := $(UMPIRE3_MODEL_ROOT)/.lake/build/bin/umpire3_native_certificate_check
UMPIRE3_NATIVE_GENERATED := $(UMPIRE3_ROOT)/checker/finite/testdata/generated
UMPIRE3_NATIVE_RETAINED := $(UMPIRE3_ROOT)/checker/finite/testdata/retained
UMPIRE3_NATIVE_CERTIFICATE := $(UMPIRE3_NATIVE_GENERATED)/nexus-cancellation-scale.certificate.json
UMPIRE3_NATIVE_RECEIPT := $(UMPIRE3_NATIVE_GENERATED)/nexus-cancellation-scale.receipt.json
UMPIRE3_NATIVE_BENCHMARK := $(UMPIRE3_NATIVE_RETAINED)/nexus-cancellation-scale.benchmark.json
UMPIRE3_CHECKER_COVERAGE := $(UMPIRE3_ROOT)/protocol/internal/generated/testdata/generated/checker-coverage.json

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
GOMAD3_GO := $(ROOT)/tools/gomad3/.toolchain/bin/go
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
UMPIRE_PUBLIC_BINPB := $(PROTO_ROOT)/umpire-public.binpb
# Note: If you change the value of INTERNAL_BINPB, you'll have to add logic to
# develop/buf-breaking.sh to handle the old and new values at once.
INTERNAL_BINPB := $(PROTO_ROOT)/image.bin
CHASM_BINPB := $(PROTO_ROOT)/chasm.bin
UMPIRE_API_FIXTURE_ROOT := tools/umpire/cmd/umpire-gen-lean-api/testdata/basic
UMPIRE_API_FIXTURE_INPUT := $(UMPIRE_API_FIXTURE_ROOT)/input
UMPIRE_API_FIXTURE_DESCRIPTOR := $(UMPIRE_API_FIXTURE_ROOT)/input.pb
UMPIRE_API_FIXTURE_PROTOS := compat/protobuf/v1/options.proto shared/messaging/v1/types.proto public/messaging/v1/message.proto internal/messaging/v1/messaging_service.proto
PROTO_OUT := api

ALL_SRC         := $(shell find . -path "./tools/gomad3/.toolchain" -prune -o -name "*.go" -print)
ALL_SRC         += go.mod
ALL_SCRIPTS     := $(shell find . -path "./tools/gomad3/.toolchain" -prune -o -name "*.sh" -print)

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

.PHONY: agentworkflow-test agentworkflow-race agentworkflow-vet agentworkflow-check

agentworkflow-test:
	@cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./...

agentworkflow-race:
	@cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep -race ./...

agentworkflow-vet:
	@cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./...

agentworkflow-check: agentworkflow-test agentworkflow-race agentworkflow-vet

.PHONY: gomad-prototype gomad-test gomad-formal gomad-formal-veil

gomad-prototype: gomad-test gomad-formal

gomad-test:
	@cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./...
	@cd tools/common/formal && GOWORK=off go test -tags test_dep ./...
	@cd tools/gomad && GOWORK=off go test -tags test_dep ./...

gomad-formal:
	@cd model && $(LEAN_LAKE) build Shared
	@cd tools/gomad/formal && $(LEAN_LAKE) build

gomad-formal-veil:
	@cd tools/gomad/formal/veil && $(LEAN_LAKE) build

.PHONY: gomad3 gomad3-go gomad3-runner gomad3-run gomad3-test gomad3-integration-test gomad3-qualification

gomad3: gomad3-runner

gomad3-go:
	@$(MAKE) -C tools/gomad3 toolchain

gomad3-runner:
	@$(MAKE) -C tools/gomad3 runner

gomad3-run:
	@test "$(origin GOMADSEED)" != undefined || { echo "GOMADSEED is required: make gomad3-run GOMADSEED=<uint64> GOMAD3_RUN=<package>" >&2; exit 1; }
	@test -n "$(GOMAD3_RUN)" || { echo "GOMAD3_RUN is required: make gomad3-run GOMADSEED=<uint64> GOMAD3_RUN=<package>" >&2; exit 1; }
	@$(MAKE) gomad3-go
	@env -u GOMADSEED CGO_ENABLED=0 TZ=UTC GOMAD3_CHILD_SEED="$(GOMADSEED)" \
		$(GOMAD3_GO) run -exec "$(ROOT)/tools/gomad3/exec.sh" $(GOMAD3_RUN) $(GOMAD3_ARGS)

gomad3-test:
	@test "$(origin GOMADSEED)" != undefined || { echo "GOMADSEED is required: make gomad3-test GOMADSEED=<uint64> GOMAD3_PACKAGES=<packages>" >&2; exit 1; }
	@test -n "$(GOMAD3_PACKAGES)" || { echo "GOMAD3_PACKAGES is required: make gomad3-test GOMADSEED=<uint64> GOMAD3_PACKAGES=<packages>" >&2; exit 1; }
	@$(MAKE) gomad3-go
	@env -u GOMADSEED CGO_ENABLED=0 TZ=UTC GOMAD3_CHILD_SEED="$(GOMADSEED)" \
		$(GOMAD3_GO) test -exec "$(ROOT)/tools/gomad3/exec.sh" -count=1 -tags test_dep $(GOMAD3_PACKAGES) $(GOMAD3_ARGS)

gomad3-integration-test: gomad3-runner
	@go test -tags test_dep,gomad3_integration -count=1 ./tools/gomad3integration

gomad3-qualification: gomad3-runner
	@$(MAKE) -C tools/gomad3 compatibility-pack-qualification
	@$(MAKE) -C tools/gomad3 qualification-set \
		GOMAD3_QUALIFICATION_MANIFEST="$(ROOT)/tools/gomad3integration/qualification/temporal.json" \
		GOMAD3_QUALIFICATION_WORKDIR="$(ROOT)" \
		GOMAD3_QUALIFICATION_ARTIFACTS="$(ROOT)/tools/gomad3/.toolchain/temporal-qualification" \
		GOMAD3_QUALIFICATION_OUTPUT="$(ROOT)/tools/gomad3/.toolchain/temporal-qualification-set.json"

clean-tools:
	@printf $(COLOR) "Delete tools..."
	@rm -rf $(STAMPDIR)
	@rm -rf $(LOCALBIN)

$(STAMPDIR):
	@mkdir -p $(STAMPDIR)

$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

# When updating the version, update the golangci-lint GHA workflow as well.
.PHONY: golangci-lint
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

GOIMPORTS_VER := v0.36.0
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

STRINGER_VER := v0.36.0
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

$(UMPIRE_PUBLIC_BINPB): go.mod go.sum
	@printf $(COLOR) "Generating registered public protobuf descriptors..."
	@$(UMPIRE_EXPORT_PROTO_DESCRIPTORS_COMMAND) \
		--package-pattern go.temporal.io/api/... \
		--file-prefix temporal/api/ \
		--output $@

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
umpire-genmodels:
	@printf $(COLOR) "Generate Umpire verification models..."
	@$(UMPIRE_GENMODELS) -mode generate

umpire-check-genmodels:
	@printf $(COLOR) "Check generated Umpire verification models..."
	@$(UMPIRE_GENMODELS) -mode check-generated

umpire-verify-smoke: umpire-check-genmodels
	@printf $(COLOR) "Run Umpire smoke verification..."
	@$(UMPIRE_GENMODELS) -mode verify -profile smoke

umpire-verify-nightly: umpire-check-genmodels
	@printf $(COLOR) "Run Umpire nightly verification..."
	@$(UMPIRE_GENMODELS) -mode verify -profile nightly

.PHONY: umpire-genmodels umpire-check-genmodels umpire-verify-smoke umpire-verify-nightly

umpire3-gen-manifest:
	@printf $(COLOR) "Generate Umpire3 empty manifest..."
	@$(UMPIRE3_MANIFEST_COMMAND) > $(UMPIRE3_MANIFEST)

umpire3-check-manifest:
	@printf $(COLOR) "Check generated Umpire3 empty manifest..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_MANIFEST_COMMAND) > "$$temporary"; \
		diff -u $(UMPIRE3_MANIFEST) "$$temporary"

umpire3-gen-catalog:
	@printf $(COLOR) "Generate Umpire3 semantic catalog..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact catalog -output $(UMPIRE3_CATALOG)

umpire3-check-catalog:
	@printf $(COLOR) "Check generated Umpire3 semantic catalog..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact catalog > "$$temporary"; \
		diff -u $(UMPIRE3_CATALOG) "$$temporary"

umpire3-gen-identifiers: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 Go identifiers..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact go-identifiers -output $(UMPIRE3_IDENTIFIERS)

umpire3-check-identifiers: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 Go identifiers..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact go-identifiers > "$$temporary"; \
		diff -u $(UMPIRE3_IDENTIFIERS) "$$temporary"

umpire3-gen-author-facade: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 author facade..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact author-facade -output $(UMPIRE3_AUTHOR_FACADE)

umpire3-check-author-facade: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 author facade..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact author-facade > "$$temporary"; \
		diff -u $(UMPIRE3_AUTHOR_FACADE) "$$temporary"

umpire3-gen-schema:
	@printf $(COLOR) "Generate Umpire3 experiment schema..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact experiment-schema -output $(UMPIRE3_EXPERIMENT_SCHEMA)

umpire3-check-schema:
	@printf $(COLOR) "Check generated Umpire3 experiment schema..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact experiment-schema > "$$temporary"; \
		diff -u $(UMPIRE3_EXPERIMENT_SCHEMA) "$$temporary"

umpire3-gen-monitor: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 monitor programs..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact monitor-programs -output $(UMPIRE3_MONITORS)

umpire3-check-monitor: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 monitor programs..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact monitor-programs > "$$temporary"; \
		diff -u $(UMPIRE3_MONITORS) "$$temporary"

umpire3-gen-observation: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 observation programs..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact observation-programs -output $(UMPIRE3_OBSERVATIONS)

umpire3-check-observation: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 observation programs..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact observation-programs > "$$temporary"; \
		diff -u $(UMPIRE3_OBSERVATIONS) "$$temporary"

umpire3-gen-composition: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 model composition..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact composition -output $(UMPIRE3_COMPOSITION)

umpire3-check-composition: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 model composition..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact composition > "$$temporary"; \
		diff -u $(UMPIRE3_COMPOSITION) "$$temporary"

umpire3-gen-parity: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 parity ledger..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact parity-ledger -output $(UMPIRE3_PARITY)

umpire3-check-parity: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 parity ledger..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact parity-ledger > "$$temporary"; \
		diff -u $(UMPIRE3_PARITY) "$$temporary"

umpire3-gen-coverage: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 coverage denominator..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact coverage-denominator -output $(UMPIRE3_COVERAGE)

umpire3-check-coverage: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 coverage denominator..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact coverage-denominator > "$$temporary"; \
		diff -u $(UMPIRE3_COVERAGE) "$$temporary"

umpire3-gen-finite-replay: umpire3-gen-composition
	@printf $(COLOR) "Generate Umpire3 finite replay catalog..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Targets.FiniteReplay
	@$(UMPIRE3_EXPORT_COMMAND) -artifact finite-replay-catalog -output $(UMPIRE3_FINITE_REPLAY)

umpire3-check-finite-replay: umpire3-check-composition
	@printf $(COLOR) "Check generated Umpire3 finite replay catalog..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Targets.FiniteReplay
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact finite-replay-catalog > "$$temporary"; \
		diff -u $(UMPIRE3_FINITE_REPLAY) "$$temporary"

umpire3-gen-first-order: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 Nexus first-order views..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Families.NexusCancellation.Targets.FirstOrder
	@$(UMPIRE3_EXPORT_COMMAND) -artifact first-order-view -variant sound -output $(UMPIRE3_NEXUS_FIRST_ORDER)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact first-order-view -variant mutated -output $(UMPIRE3_NEXUS_MUTATED_FIRST_ORDER)

umpire3-check-first-order: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 Nexus first-order views..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Families.NexusCancellation.Targets.FirstOrder
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact first-order-view -variant sound > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_FIRST_ORDER) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact first-order-view -variant mutated > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_MUTATED_FIRST_ORDER) "$$temporary"

umpire3-gen-attempt: umpire3-gen-first-order
	@printf $(COLOR) "Generate Umpire3 Nexus attempt views..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Families.NexusCancellation.Targets.Attempt
	@$(UMPIRE3_EXPORT_COMMAND) -artifact attempt-view -variant sound -output $(UMPIRE3_NEXUS_ATTEMPT)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact attempt-view -variant mutated -output $(UMPIRE3_NEXUS_MUTATED_ATTEMPT)

umpire3-check-attempt: umpire3-check-first-order
	@printf $(COLOR) "Check generated Umpire3 Nexus attempt views..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Families.NexusCancellation.Targets.Attempt
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact attempt-view -variant sound > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_ATTEMPT) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact attempt-view -variant mutated > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_MUTATED_ATTEMPT) "$$temporary"

umpire3-gen-native-binding: umpire3-gen-first-order
	@printf $(COLOR) "Generate Umpire3 native certificate proof binding..."
	@$(UMPIRE3_NATIVE_COMMAND) -operation bind -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
		-output $(UMPIRE3_NATIVE_BINDING)

umpire3-check-native-binding: umpire3-check-first-order
	@printf $(COLOR) "Check Umpire3 native certificate proof binding..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_NATIVE_COMMAND) -operation bind -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-output "$$temporary"; \
		diff -u $(UMPIRE3_NATIVE_BINDING) "$$temporary"

umpire3-build-native: umpire3-gen-native-binding
	@printf $(COLOR) "Build Umpire3 native certificate checker..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build umpire3_native_certificate_check

umpire3-gen-native-results: umpire3-gen-native-binding
	@$(MAKE) umpire3-build-native
	@printf $(COLOR) "Generate Umpire3 parallel native certificate and checked receipt..."
	@set -eu; \
		$(UMPIRE3_NATIVE_COMMAND) -operation produce -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-workers 8 -replicas 10 -output $(UMPIRE3_NATIVE_CERTIFICATE); \
		$(UMPIRE3_NATIVE_COMMAND) -operation check -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-certificate $(UMPIRE3_NATIVE_CERTIFICATE) -checker-command $(UMPIRE3_NATIVE_CERTIFICATE_BIN) \
			-output $(UMPIRE3_NATIVE_RECEIPT)

umpire3-check-native-results: umpire3-check-native-binding
	@printf $(COLOR) "Check Umpire3 parallel native certificate and checked receipt..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build umpire3_native_certificate_check
	@set -eu; temporary=$$(mktemp -d); \
		trap 'rm -rf "$$temporary"' EXIT; \
		$(UMPIRE3_NATIVE_COMMAND) -operation produce -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-workers 8 -replicas 10 -output "$$temporary/certificate.json"; \
		diff -u $(UMPIRE3_NATIVE_CERTIFICATE) "$$temporary/certificate.json"; \
		$(UMPIRE3_NATIVE_COMMAND) -operation check -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-certificate "$$temporary/certificate.json" -checker-command $(UMPIRE3_NATIVE_CERTIFICATE_BIN) \
			-output "$$temporary/receipt.json"; \
		diff -u $(UMPIRE3_NATIVE_RECEIPT) "$$temporary/receipt.json"

umpire3-record-native-benchmark: umpire3-check-native-results
	@printf $(COLOR) "Record Umpire3 10x native-search benchmark..."
	@$(UMPIRE3_NATIVE_COMMAND) -operation benchmark -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
		-certificate $(UMPIRE3_NATIVE_CERTIFICATE) -receipt $(UMPIRE3_NATIVE_RECEIPT) \
		-checker-command $(UMPIRE3_NATIVE_CERTIFICATE_BIN) -workers 8 -replicas 10 \
		-output $(UMPIRE3_NATIVE_BENCHMARK)

umpire3-check-native-benchmark: umpire3-check-native-results
	@printf $(COLOR) "Check retained and fresh Umpire3 10x native-search benchmarks..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_NATIVE_COMMAND) -operation validate-benchmark -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-certificate $(UMPIRE3_NATIVE_CERTIFICATE) -receipt $(UMPIRE3_NATIVE_RECEIPT) \
			-benchmark $(UMPIRE3_NATIVE_BENCHMARK); \
		$(UMPIRE3_NATIVE_COMMAND) -operation benchmark -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-certificate $(UMPIRE3_NATIVE_CERTIFICATE) -receipt $(UMPIRE3_NATIVE_RECEIPT) \
			-checker-command $(UMPIRE3_NATIVE_CERTIFICATE_BIN) -workers 8 -replicas 10 \
			-output "$$temporary"

umpire3-gen-checker-coverage: umpire3-gen-finite-replay umpire3-gen-native-results umpire3-record-veil-results umpire3-check-native-benchmark
	@printf $(COLOR) "Generate Umpire3 checker coverage..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact checker-coverage -output $(UMPIRE3_CHECKER_COVERAGE)

umpire3-check-checker-coverage: umpire3-check-finite-replay umpire3-check-native-benchmark umpire3-check-veil-results
	@printf $(COLOR) "Check Umpire3 checker coverage..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact checker-coverage > "$$temporary"; \
		diff -u $(UMPIRE3_CHECKER_COVERAGE) "$$temporary"

umpire3-gen-family-dependencies: umpire3-gen-checker-coverage
	@printf $(COLOR) "Generate Umpire3 family dependency graph..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact family-dependencies -output $(UMPIRE3_FAMILY_DEPENDENCIES)

umpire3-check-family-dependencies: umpire3-check-checker-coverage
	@printf $(COLOR) "Check Umpire3 family dependency graph..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact family-dependencies > "$$temporary"; \
		diff -u $(UMPIRE3_FAMILY_DEPENDENCIES) "$$temporary"

umpire3-gen-temporal: umpire3-gen-catalog
	@printf $(COLOR) "Generate Umpire3 task-delivery temporal views..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Families.WorkflowProgress.Targets.Temporal
	@$(UMPIRE3_EXPORT_COMMAND) -artifact temporal-view -variant sound -output $(UMPIRE3_TASK_DELIVERY_TEMPORAL)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact temporal-view -variant delivery-fairness-removed -output $(UMPIRE3_TASK_DELIVERY_MUTATED_TEMPORAL)

umpire3-check-temporal: umpire3-check-catalog
	@printf $(COLOR) "Check generated Umpire3 task-delivery temporal views..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build Temporal.Families.WorkflowProgress.Targets.Temporal
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact temporal-view -variant sound > "$$temporary"; \
		diff -u $(UMPIRE3_TASK_DELIVERY_TEMPORAL) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact temporal-view -variant delivery-fairness-removed > "$$temporary"; \
		diff -u $(UMPIRE3_TASK_DELIVERY_MUTATED_TEMPORAL) "$$temporary"

umpire3-build-temporal:
	@printf $(COLOR) "Build Umpire3 canonical temporal lasso replay..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build umpire3_temporal_lasso_replay

umpire3-build-veil:
	@printf $(COLOR) "Build Umpire3's embedded Veil checks..."
	@cd $(UMPIRE3_MODEL_ROOT) && $(LEAN_LAKE) build \
		Temporal.Families.NexusCancellation.Targets.Veil.Binding \
		Temporal.Families.NexusCancellation.Targets.Veil.MutatedBinding \
		Temporal.Families.NexusCancellation.Targets.Veil.TrustedBinding \
		umpire3_trace_replay \
		umpire3_veil_sound umpire3_veil_mutated \
		umpire3_veil_sound_proof umpire3_veil_sound_trusted_proof

umpire3-export-veil-bindings: umpire3-gen-first-order
	@printf $(COLOR) "Export checked Umpire3 Veil bindings..."
	@set -eu; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact veil-binding -variant sound \
			-output $(UMPIRE3_VEIL_SOUND_BINDING); \
		$(UMPIRE3_EXPORT_COMMAND) -artifact veil-binding -variant mutated \
			-output $(UMPIRE3_VEIL_MUTATED_BINDING); \
		$(UMPIRE3_EXPORT_COMMAND) -artifact veil-binding -variant trusted \
			-output $(UMPIRE3_VEIL_TRUSTED_BINDING)

umpire3-check-veil-bindings: umpire3-check-first-order
	@printf $(COLOR) "Check Umpire3 Veil bindings..."
	@set -eu; temporary=$$(mktemp -d); \
		trap 'rm -rf "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact veil-binding -variant sound \
			-output "$$temporary/sound.json"; \
		diff -u $(UMPIRE3_VEIL_SOUND_BINDING) "$$temporary/sound.json"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact veil-binding -variant mutated \
			-output "$$temporary/mutated.json"; \
		diff -u $(UMPIRE3_VEIL_MUTATED_BINDING) "$$temporary/mutated.json"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact veil-binding -variant trusted \
			-output "$$temporary/trusted.json"; \
		diff -u $(UMPIRE3_VEIL_TRUSTED_BINDING) "$$temporary/trusted.json"

umpire3-record-veil-results: umpire3-export-veil-bindings
	@$(MAKE) umpire3-build-veil
	@printf $(COLOR) "Record normalized Umpire3 Veil results..."
	@set -eu; \
		$(UMPIRE3_VEIL_COMMAND) -operation check-concrete -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_SOUND_BINDING) -backend-command $(UMPIRE3_VEIL_SOUND_BIN) \
			-output $(UMPIRE3_VEIL_SOUND_RESULT); \
		$(UMPIRE3_VEIL_COMMAND) -operation check-concrete -input $(UMPIRE3_NEXUS_MUTATED_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_MUTATED_BINDING) -backend-command $(UMPIRE3_VEIL_MUTATED_BIN) \
			-replay-command $(UMPIRE3_TRACE_REPLAY_BIN) \
			-output $(UMPIRE3_VEIL_MUTATED_RESULT); \
		$(UMPIRE3_VEIL_COMMAND) -operation check-job -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_SOUND_BINDING) \
			-job symbolic-trace -job-command $(UMPIRE3_VEIL_SOUND_PROOF_BIN) \
			-output $(UMPIRE3_VEIL_SYMBOLIC_RESULT); \
		$(UMPIRE3_VEIL_COMMAND) -operation check-job -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_SOUND_BINDING) \
			-job invariant -job-command $(UMPIRE3_VEIL_SOUND_PROOF_BIN) \
			-output $(UMPIRE3_VEIL_INVARIANT_RESULT); \
		$(UMPIRE3_VEIL_COMMAND) -operation check-job -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_TRUSTED_BINDING) \
			-job invariant -job-command $(UMPIRE3_VEIL_SOUND_TRUSTED_PROOF_BIN) \
			-output $(UMPIRE3_VEIL_INVARIANT_TRUSTED_RESULT)

umpire3-check-veil-results: umpire3-check-veil-bindings
	@$(MAKE) umpire3-build-veil
	@printf $(COLOR) "Check normalized Umpire3 Veil results..."
	@set -eu; temporary=$$(mktemp -d); \
		trap 'rm -rf "$$temporary"' EXIT; \
		$(UMPIRE3_VEIL_COMMAND) -operation check-concrete -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_SOUND_BINDING) -backend-command $(UMPIRE3_VEIL_SOUND_BIN) \
			-output "$$temporary/sound.json"; \
		diff -u $(UMPIRE3_VEIL_SOUND_RESULT) "$$temporary/sound.json"; \
		$(UMPIRE3_VEIL_COMMAND) -operation check-concrete -input $(UMPIRE3_NEXUS_MUTATED_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_MUTATED_BINDING) -backend-command $(UMPIRE3_VEIL_MUTATED_BIN) \
			-replay-command $(UMPIRE3_TRACE_REPLAY_BIN) \
			-output "$$temporary/mutated.json"; \
		diff -u $(UMPIRE3_VEIL_MUTATED_RESULT) "$$temporary/mutated.json"; \
		$(UMPIRE3_VEIL_COMMAND) -operation check-job -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_SOUND_BINDING) \
			-job symbolic-trace -job-command $(UMPIRE3_VEIL_SOUND_PROOF_BIN) \
			-output "$$temporary/symbolic.json"; \
		diff -u $(UMPIRE3_VEIL_SYMBOLIC_RESULT) "$$temporary/symbolic.json"; \
		$(UMPIRE3_VEIL_COMMAND) -operation check-job -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_SOUND_BINDING) \
			-job invariant -job-command $(UMPIRE3_VEIL_SOUND_PROOF_BIN) \
			-output "$$temporary/invariant.json"; \
		diff -u $(UMPIRE3_VEIL_INVARIANT_RESULT) "$$temporary/invariant.json"; \
		$(UMPIRE3_VEIL_COMMAND) -operation check-job -input $(UMPIRE3_NEXUS_FIRST_ORDER) \
			-binding $(UMPIRE3_VEIL_TRUSTED_BINDING) \
			-job invariant -job-command $(UMPIRE3_VEIL_SOUND_TRUSTED_PROOF_BIN) \
			-output "$$temporary/invariant-trusted.json"; \
		diff -u $(UMPIRE3_VEIL_INVARIANT_TRUSTED_RESULT) "$$temporary/invariant-trusted.json"

umpire3-gen-proof: umpire3-gen-api umpire3-gen-composition
	@printf $(COLOR) "Generate Umpire3 proof manifests..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment nexus -output $(UMPIRE3_NEXUS_PROOF_MANIFEST)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment nexus-mutation-refinement -output $(UMPIRE3_NEXUS_MUTATION_REJECTION_PROOF_MANIFEST)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment nexus-mutation-exact -output $(UMPIRE3_NEXUS_EXACT_MUTATION_PROOF_MANIFEST)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment update -output $(UMPIRE3_UPDATE_PROOF_MANIFEST)

umpire3-check-proof: umpire3-check-api umpire3-check-composition
	@printf $(COLOR) "Check generated Umpire3 proof manifests..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment nexus > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_PROOF_MANIFEST) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment nexus-mutation-refinement > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_MUTATION_REJECTION_PROOF_MANIFEST) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment nexus-mutation-exact > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_EXACT_MUTATION_PROOF_MANIFEST) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact proof-manifest -experiment update > "$$temporary"; \
		diff -u $(UMPIRE3_UPDATE_PROOF_MANIFEST) "$$temporary"

umpire3-gen-experiment: umpire3-gen-catalog umpire3-gen-api umpire3-gen-composition
	@printf $(COLOR) "Generate Umpire3 experiments..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact experiment -experiment nexus -output $(UMPIRE3_NEXUS_EXPERIMENT)
	@$(UMPIRE3_EXPORT_COMMAND) -artifact experiment -experiment update -output $(UMPIRE3_UPDATE_EXPERIMENT)

umpire3-check-experiment: umpire3-check-catalog umpire3-check-api umpire3-check-composition
	@printf $(COLOR) "Check generated Umpire3 Nexus experiment..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact experiment -experiment nexus > "$$temporary"; \
		diff -u $(UMPIRE3_NEXUS_EXPERIMENT) "$$temporary"; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact experiment -experiment update > "$$temporary"; \
		diff -u $(UMPIRE3_UPDATE_EXPERIMENT) "$$temporary"

umpire3-gen-api:
	@printf $(COLOR) "Generate Umpire3 Nexus API model..."
	@$(UMPIRE3_API_COMMAND) -mode generate
	@cp $(UMPIRE3_API_DESCRIPTOR) $(UMPIRE3_PROTOCOL_DESCRIPTOR)

umpire3-check-api:
	@printf $(COLOR) "Check generated Umpire3 Nexus API model..."
	@$(UMPIRE3_API_COMMAND) -mode check
	@cmp $(UMPIRE3_API_DESCRIPTOR) $(UMPIRE3_PROTOCOL_DESCRIPTOR)

umpire-build-model:
	@printf $(COLOR) "Build Temporal Umpire Lean model..."
	@cd model && $(LEAN_LAKE) build

umpire-check-plan-index:
	@printf $(COLOR) "Check Umpire plan authority index..."
	@go run ./tools/planindex

umpire-check-artifact:
	@test -n "$(FAMILY)" || { echo "FAMILY is required" >&2; exit 2; }
	@test -n "$(ARTIFACT)" || { echo "ARTIFACT is required" >&2; exit 2; }
	@$(UMPIRE_ARTIFACT_COMMAND) check --family "$(FAMILY)" --artifact "$(ARTIFACT)"

umpire-check-artifact-set:
	@test -n "$(SET)" || { echo "SET is required" >&2; exit 2; }
	@$(UMPIRE_ARTIFACT_COMMAND) check-set --set "$(SET)"

umpire-check-local-run-evaluation:
	@test -n "$(SET)" || { echo "SET is required" >&2; exit 2; }
	@test -n "$(OUTPUT_ROOT)" || { echo "OUTPUT_ROOT is required" >&2; exit 2; }
	@test -d "$(SET)" || { echo "SET must be a directory" >&2; exit 2; }
	@test -d "$(OUTPUT_ROOT)" || { echo "OUTPUT_ROOT must be a directory" >&2; exit 2; }
	@set -eu; installation=$$(mktemp -d); \
		trap 'rm -rf "$$installation"' EXIT; \
		cd model && $(LEAN_LAKE) build temporal-run-evaluation-checker >/dev/null; \
		cd ..; \
		cp "model/.lake/build/bin/temporal-run-evaluation-checker" \
			"$$installation/temporal-run-evaluation-checker"; \
		chmod 0700 "$$installation/temporal-run-evaluation-checker"; \
		checker_sha=$$(shasum -a 256 "$$installation/temporal-run-evaluation-checker" | awk '{print $$1}'); \
		go build -ldflags "-X go.temporal.io/server/tools/umpire/runevaluation.installedCheckerSHA256=sha256:$$checker_sha" \
			-o "$$installation/umpire-local-run-evaluation" \
			./tools/umpire/cmd/umpire-local-run-evaluation; \
		"$$installation/umpire-local-run-evaluation" \
			--set "$(SET)" --output-root "$(OUTPUT_ROOT)"

umpire-inspect:
	@test -n "$(SCENARIO)" || (echo "SCENARIO is required" >&2; exit 1)
	@cd model && $(LEAN_LAKE) exe $(UMPIRE_REGRESSION_INSPECTOR) "$(SCENARIO)"

umpire-gen-lean-api: PROTOC = mise exec -- protoc
umpire-gen-lean-api: $(UMPIRE_PUBLIC_BINPB) $(API_BINPB) $(INTERNAL_BINPB) $(CHASM_BINPB)
	@printf $(COLOR) "Generate Temporal API Lean modules..."
	@$(UMPIRE_GEN_LEAN_API_COMMAND) $(UMPIRE_GEN_LEAN_API_ARGS)

umpire-gen-lean-dynamic-config-catalog:
	@printf $(COLOR) "Generate Temporal dynamic configuration Lean modules..."
	@$(UMPIRE_GEN_LEAN_DYNAMIC_CONFIG_CATALOG_COMMAND) --output-root model

$(UMPIRE_API_FIXTURE_DESCRIPTOR): $(addprefix $(UMPIRE_API_FIXTURE_INPUT)/,$(UMPIRE_API_FIXTURE_PROTOS))
	@mise exec -- protoc \
		--proto_path=$(UMPIRE_API_FIXTURE_INPUT) \
		--include_imports \
		--descriptor_set_out=$@ \
		$(UMPIRE_API_FIXTURE_PROTOS)

umpire-gen-lean-api-fixture: $(UMPIRE_API_FIXTURE_DESCRIPTOR)
	@go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api -run '^TestBasicFixture$$' -rewrite

umpire-gen-tests:
	@printf $(COLOR) "Generate complete model-selected Umpire tests..."
	@cd model && $(UMPIRE_GEN_TESTS_COMMAND) $(ARGS)

umpire-gen-regression-views:
	@cd model && $(LEAN_LAKE) build $(UMPIRE_REGRESSION_INSPECTOR) >/dev/null
	@$(UMPIRE_GEN_REGRESSION_VIEWS_COMMAND) --repository-root . --output-root .

umpire-check-regression-views:
	@printf $(COLOR) "Check generated Umpire regression views..."
	@cd model && $(LEAN_LAKE) build $(UMPIRE_REGRESSION_INSPECTOR)
	@set -eu; temporary_root=$$(cd "$${TMPDIR:-/tmp}" && pwd -P); \
		temporary=$$(mktemp -d "$$temporary_root/umpire-regression.XXXXXX"); \
		trap 'rm -rf "$$temporary"' EXIT; \
		$(UMPIRE_GEN_REGRESSION_VIEWS_COMMAND) --repository-root . --output-root "$$temporary"; \
		diff -u tools/umpire/regression/catalog_generated_test.go \
			"$$temporary/tools/umpire/regression/catalog_generated_test.go"; \
		diff -u model/Temporal/Tool/Generated/Regressions.md \
			"$$temporary/model/Temporal/Tool/Generated/Regressions.md"; \
		diff -u tools/umpire/regression/switch_generated_view_test.go \
			"$$temporary/tools/umpire/regression/switch_generated_view_test.go"; \
		diff -u model/Umpire/Examples/Generated/Switch.md \
			"$$temporary/model/Umpire/Examples/Generated/Switch.md"
	@temporary_root=$$(cd "$${TMPDIR:-/tmp}" && pwd -P); \
		TMPDIR="$$temporary_root" go test -count=1 -tags test_dep \
			./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression

umpire-check-legacy-vocabulary:
	@printf $(COLOR) "Check active Umpire vocabulary..."
	@mise exec -- go run ./tools/umpire/cmd/umpire-check-legacy-vocabulary

umpire-check-regression: umpire-check-regression-views umpire-check-legacy-vocabulary
	@mise exec -- go test -count=1 -tags test_dep \
		./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$$'
	@set -eu; \
		old_namespace='Temporal''[.](Experiment|Umpire)'; \
		old_path='Temporal/''(Experiment|Umpire)'; \
		old_targets='Experiment''Tests|temporal-experiment''-inspect|Temporal''UmpireTests|temporal-umpire''-inspect|Nexus''AutoClose'; \
		old_experiment_tree=model/Temporal/''Experiment; \
		old_umpire_tree=model/Temporal/''Umpire; \
		old_temporal_tests=model/Temporal''UmpireTests.lean; \
		old_auto_close_root=model/Temporal/Feature/Nexus/AutoClose.lean; \
		old_caller_closure_root=model/Temporal/Feature/Nexus/CallerClosure.lean; \
		old_examples_tree=model/Temporal/Feature/Nexus/Examples; \
		test ! -e "$$old_experiment_tree"; \
		test ! -e "$${old_experiment_tree}Tests.lean"; \
		test ! -e "$$old_umpire_tree"; \
		test ! -e "$$old_temporal_tests"; \
		test ! -e "$$old_auto_close_root"; \
		test ! -e "$$old_caller_closure_root"; \
		test ! -e "$$old_examples_tree"; \
		live_sources=$$(find model/Umpire model/Temporal -type f -name '*.lean' -print); \
		test -n "$$live_sources"; \
		if grep -nE "$$old_namespace|$$old_path" $$live_sources \
			model/Umpire.lean model/UmpireTests.lean model/Temporal.lean model/TemporalModelTests.lean; then \
			echo "found obsolete Temporal interface in live Lean sources" >&2; \
			exit 1; \
		else \
			scan_status=$$?; \
			test "$$scan_status" -eq 1; \
		fi; \
		if grep -nE "$$old_namespace|$$old_path|$$old_targets" Makefile model/lakefile.toml model/README.md; then \
			echo "found obsolete Temporal interface in live build or model documentation" >&2; \
			exit 1; \
		else \
			scan_status=$$?; \
			test "$$scan_status" -eq 1; \
		fi; \
		if git grep -nE '^[[:space:]]*(import|namespace)[[:space:]]+(Temporal|Nexus)([.]|[[:space:]]|$$)|(^|[^[:alnum:]_-])(Temporal|Nexus)([.]|/)|(^|[^[:alnum:]_-])(nexus|workflow|workflow-nexus)[.]' -- \
			model/Umpire model/Umpire.lean model/UmpireTests.lean; then \
			echo "found Temporal-owned dependency, namespace, or semantic prefix in reusable Umpire artifacts" >&2; \
			exit 1; \
		else \
			scan_status=$$?; \
			test "$$scan_status" -eq 1; \
		fi; \
		configuration_sources="model/Temporal/System/Configuration.lean $$(find model/Temporal/System/Configuration -type f -name '*.lean' ! -name '*Tests.lean' -print)"; \
		if grep -nE '^[[:space:]]*import[[:space:]]+Temporal[.]System[.](Callback|Matching)([.]|[[:space:]]|$$)' $$configuration_sources; then \
			echo "found forbidden shared Configuration dependency on Callback or Matching" >&2; \
			exit 1; \
		else \
			scan_status=$$?; \
			test "$$scan_status" -eq 1; \
		fi; \
		for package in Target Property Behavior Query; do \
			test -f "model/Umpire/$$package/Language.lean" || { \
				echo "missing physical Umpire $$package package" >&2; \
				exit 1; \
			}; \
			grep -qx "import Umpire.$$package.Language" "model/Umpire/$$package.lean" || { \
				echo "Umpire $$package facade does not expose its package" >&2; \
				exit 1; \
			}; \
		done; \
		test -f model/Umpire/Planning/Engine.lean || { \
			echo "missing physical Umpire Planning package" >&2; \
			exit 1; \
		}; \
		grep -qx 'import Umpire.Planning.Engine' model/Umpire/Planning.lean || { \
			echo "Umpire Planning facade does not expose its package" >&2; \
			exit 1; \
		}
	@cd model && $(LEAN_LAKE) build Temporal UmpireTests TemporalModelTests TemporalExperimentalTests $(UMPIRE_REGRESSION_INSPECTOR)
	@cd model && $(LEAN_LAKE) exe umpire-gen-tests-tests
	@set -eu; temporary=$$(mktemp -d); \
		trap 'rm -rf "$$temporary"' EXIT; \
		cd model; \
		for scenario_fixture in $(UMPIRE_REGRESSION_FIXTURES); do \
			scenario=$${scenario_fixture%%:*}; \
			fixture=$${scenario_fixture#*:}; \
			$(LEAN_LAKE) exe $(UMPIRE_REGRESSION_INSPECTOR) "$$scenario" > "$$temporary/first.json"; \
			$(LEAN_LAKE) exe $(UMPIRE_REGRESSION_INSPECTOR) "$$scenario" > "$$temporary/second.json"; \
			cmp -s "$$temporary/first.json" "$$temporary/second.json"; \
			cmp -s "$$fixture" "$$temporary/first.json"; \
		done; \
		if $(LEAN_LAKE) exe $(UMPIRE_REGRESSION_INSPECTOR) missing-scenario \
			> "$$temporary/negative.stdout" 2> "$$temporary/negative.stderr"; then \
			echo "expected the inspector to reject an unknown scenario" >&2; \
			exit 1; \
		fi; \
		test ! -s "$$temporary/negative.stdout"; \
		printf '%s\n' '{"kind":"unknown-scenario","subject":"missing-scenario","context":"scenario registry"}' \
			> "$$temporary/expected-negative.stderr"; \
		cmp -s "$$temporary/expected-negative.stderr" "$$temporary/negative.stderr"; \
		if $(LEAN_LAKE) exe $(UMPIRE_REGRESSION_INSPECTOR) \
			> "$$temporary/invalid.stdout" 2> "$$temporary/invalid.stderr"; then \
			echo "expected the inspector to reject invalid arguments" >&2; \
			exit 1; \
		fi; \
		test ! -s "$$temporary/invalid.stdout"; \
		printf '%s\n' '{"kind":"invalid-arguments","subject":"inspect","context":"expected exactly one scenario identity"}' \
			> "$$temporary/expected-invalid.stderr"; \
		cmp -s "$$temporary/expected-invalid.stderr" "$$temporary/invalid.stderr"

umpire3-gen-migration:
	@printf $(COLOR) "Generate Umpire3 root-test migration ledger..."
	@$(UMPIRE3_MIGRATION_COMMAND) -output $(UMPIRE3_MIGRATION_LEDGER)

umpire3-check-migration:
	@printf $(COLOR) "Check Umpire3 root-test migration ledger..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_MIGRATION_COMMAND) -output "$$temporary"; \
		diff -u $(UMPIRE3_MIGRATION_LEDGER) "$$temporary"

umpire3-gen-release: umpire3-gen-experiment umpire3-gen-proof umpire3-gen-migration umpire3-gen-checker-coverage umpire3-record-mutation-audit umpire3-record-semantic-mutation-audit umpire3-record-resilience-audit
	@printf $(COLOR) "Generate Umpire3 candidate release bindings..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact release-candidate -release-template $(UMPIRE3_RELEASE) \
		-release-experiments $(UMPIRE3_NEXUS_EXPERIMENT),$(UMPIRE3_UPDATE_EXPERIMENT) \
		-migration-ledger $(UMPIRE3_MIGRATION_LEDGER) -output $(UMPIRE3_RELEASE)

umpire3-check-release: umpire3-check-experiment umpire3-check-proof umpire3-check-migration umpire3-check-checker-coverage umpire3-check-mutation-audit umpire3-check-semantic-mutation-audit umpire3-check-resilience-audit
	@printf $(COLOR) "Check Umpire3 candidate release bindings..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact release-candidate -release-template $(UMPIRE3_RELEASE) \
			-release-experiments $(UMPIRE3_NEXUS_EXPERIMENT),$(UMPIRE3_UPDATE_EXPERIMENT) \
			-migration-ledger $(UMPIRE3_MIGRATION_LEDGER) > "$$temporary"; \
		diff -u $(UMPIRE3_RELEASE) "$$temporary"

umpire3-gen: umpire3-gen-manifest umpire3-gen-identifiers umpire3-gen-author-facade umpire3-gen-schema umpire3-gen-monitor umpire3-gen-observation umpire3-gen-composition umpire3-gen-parity umpire3-gen-coverage umpire3-gen-finite-replay umpire3-gen-attempt umpire3-record-veil-results umpire3-gen-native-results umpire3-gen-checker-coverage umpire3-gen-family-dependencies umpire3-gen-temporal umpire3-gen-proof umpire3-gen-experiment umpire3-gen-api umpire3-gen-migration umpire3-gen-release

umpire3-check-generated: umpire3-check-manifest umpire3-check-identifiers umpire3-check-author-facade umpire3-check-schema umpire3-check-monitor umpire3-check-observation umpire3-check-composition umpire3-check-parity umpire3-check-coverage umpire3-check-finite-replay umpire3-check-attempt umpire3-check-veil-results umpire3-check-native-results umpire3-check-checker-coverage umpire3-check-family-dependencies umpire3-check-temporal umpire3-check-proof umpire3-check-experiment umpire3-check-api umpire3-check-migration umpire3-check-release

umpire3-check: umpire3-check-generated umpire3-check-native-benchmark
	@printf $(COLOR) "Check Umpire3 Lean model..."
	@$(MAKE) -C $(UMPIRE3_MODEL_ROOT) check
	@printf $(COLOR) "Test Umpire3 Go packages..."
	@go test -count=1 -tags test_dep ./$(UMPIRE3_ROOT)/...

umpire3-check-family:
	@test -n "$(FAMILY)" || (echo "FAMILY is required" >&2; exit 1)
	@$(UMPIRE3_FAMILY_COMMAND) -family $(FAMILY) -repository-root $(CURDIR)

umpire3-integration:
	@printf $(COLOR) "Run Umpire3 real-cluster integration..."
	@go test -count=1 -tags test_dep,integration ./$(UMPIRE3_ROOT)/adapter/temporal \
		-run '^TestLean(Nexus|TaskAck)ExperimentRunsWithRealTemporal' -timeout 10m

umpire3-explain:
	@test -n "$(EXPERIMENT)" || (echo "EXPERIMENT is required" >&2; exit 1)
	@$(UMPIRE3_COMMAND) explain -experiment $(EXPERIMENT)

umpire3-record-mutation-audit:
	@printf $(COLOR) "Record Umpire3 seeded cross-layer mutation audit..."
	@$(UMPIRE3_COMMAND) audit-mutation -experiment $(UMPIRE3_NEXUS_EXPERIMENT) \
		-output $(UMPIRE3_MUTATION_AUDIT) >/dev/null

umpire3-check-mutation-audit:
	@printf $(COLOR) "Check retained and fresh Umpire3 seeded cross-layer mutation audit..."
	@$(UMPIRE3_COMMAND) audit-mutation -experiment $(UMPIRE3_NEXUS_EXPERIMENT) \
		-output $(UMPIRE3_MUTATION_AUDIT) -check >/dev/null

umpire3-record-semantic-mutation-audit: umpire3-gen-proof umpire3-record-veil-results umpire3-build-temporal
	@printf $(COLOR) "Record Umpire3 cross-checker semantic mutation audit..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact semantic-mutation-audit \
		-mutation-experiment $(UMPIRE3_NEXUS_EXPERIMENT) \
		-mutation-finite-replay-command $(UMPIRE3_TRACE_REPLAY_BIN) \
		-mutation-temporal-replay-command $(UMPIRE3_TEMPORAL_LASSO_REPLAY_BIN) \
		-output $(UMPIRE3_SEMANTIC_MUTATION_AUDIT)

umpire3-check-semantic-mutation-audit: umpire3-check-proof umpire3-check-veil-results umpire3-build-temporal
	@printf $(COLOR) "Check retained and fresh Umpire3 cross-checker semantic mutation audits..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact semantic-mutation-audit \
			-mutation-experiment $(UMPIRE3_NEXUS_EXPERIMENT) \
			-mutation-finite-replay-command $(UMPIRE3_TRACE_REPLAY_BIN) \
			-mutation-temporal-replay-command $(UMPIRE3_TEMPORAL_LASSO_REPLAY_BIN) > "$$temporary"; \
		diff -u $(UMPIRE3_SEMANTIC_MUTATION_AUDIT) "$$temporary"

umpire3-mutation-gate: umpire3-check-mutation-audit umpire3-check-semantic-mutation-audit

umpire3-record-resilience-audit:
	@printf $(COLOR) "Record Umpire3 hostile-input, isolation, and recovery audit..."
	@$(UMPIRE3_EXPORT_COMMAND) -artifact resilience-audit -output $(UMPIRE3_RESILIENCE_AUDIT)

umpire3-check-resilience-audit:
	@printf $(COLOR) "Check retained and fresh Umpire3 hostile-input, isolation, and recovery audits..."
	@set -eu; temporary=$$(mktemp); \
		trap 'rm -f "$$temporary"' EXIT; \
		$(UMPIRE3_EXPORT_COMMAND) -artifact resilience-audit > "$$temporary"; \
		diff -u $(UMPIRE3_RESILIENCE_AUDIT) "$$temporary"

umpire3-resilience-gate: umpire3-check-resilience-audit
	@printf $(COLOR) "Run Umpire3 hostile-input, isolation, redaction, and recovery gates..."
	@go test -count=1 -tags test_dep \
		./$(UMPIRE3_ROOT)/internal/artifactio ./$(UMPIRE3_ROOT)/internal/subprocess \
		./$(UMPIRE3_ROOT)/protocol/... ./$(UMPIRE3_ROOT)/replay ./$(UMPIRE3_ROOT)/deployment \
		./$(UMPIRE3_ROOT)/deployment/canary ./$(UMPIRE3_ROOT)/assurance/release

umpire3-root:
	@printf $(COLOR) "Run retained Umpire2 and independent Umpire3 root tests..."
	@status=0; \
		go test -count=1 -tags test_dep ./tests -run '^TestUmpire2' -timeout 20m || status=$$?; \
		go test -count=1 -tags test_dep ./tests -run '^TestUmpire3' -timeout 20m || status=$$?; \
		exit $$status

umpire3-clean:
	@printf $(COLOR) "Remove resolved Umpire3 tool caches..."
	@sh $(UMPIRE3_ROOT)/clean.sh

.PHONY: umpire-build-model umpire-check-plan-index umpire-check-artifact umpire-check-artifact-set umpire-inspect umpire-gen-lean-api umpire-gen-lean-api-fixture umpire-gen-lean-dynamic-config-catalog umpire-gen-tests umpire-gen-regression-views umpire-check-regression-views umpire-check-legacy-vocabulary umpire-check-regression

.PHONY: umpire3-gen-manifest umpire3-check-manifest umpire3-gen-catalog umpire3-check-catalog umpire3-gen-identifiers umpire3-check-identifiers umpire3-gen-author-facade umpire3-check-author-facade umpire3-gen-schema umpire3-check-schema umpire3-gen-monitor umpire3-check-monitor umpire3-gen-observation umpire3-check-observation umpire3-gen-composition umpire3-check-composition umpire3-gen-parity umpire3-check-parity umpire3-gen-coverage umpire3-check-coverage umpire3-gen-finite-replay umpire3-check-finite-replay umpire3-gen-first-order umpire3-check-first-order umpire3-gen-attempt umpire3-check-attempt umpire3-gen-native-binding umpire3-check-native-binding umpire3-build-native umpire3-gen-native-results umpire3-check-native-results umpire3-record-native-benchmark umpire3-check-native-benchmark umpire3-gen-checker-coverage umpire3-check-checker-coverage umpire3-gen-family-dependencies umpire3-check-family-dependencies umpire3-gen-temporal umpire3-check-temporal umpire3-build-temporal-results umpire3-build-veil umpire3-export-veil-bindings umpire3-check-veil-bindings umpire3-record-veil-results umpire3-check-veil-results umpire3-gen-proof umpire3-check-proof umpire3-gen-experiment umpire3-check-experiment umpire3-gen-api umpire3-check-api umpire3-gen-migration umpire3-check-migration umpire3-record-mutation-audit umpire3-check-mutation-audit umpire3-record-semantic-mutation-audit umpire3-check-semantic-mutation-audit umpire3-record-resilience-audit umpire3-check-resilience-audit umpire3-gen-release umpire3-check-release umpire3-gen umpire3-check-generated umpire3-check umpire3-check-family umpire3-integration umpire3-explain umpire3-mutation-gate umpire3-resilience-gate umpire3-root umpire3-clean

goimports: fmt-imports $(GOIMPORTS)
	@printf $(COLOR) "Run goimports for all files..."
	@UNGENERATED_FILES=$$(find . -path './tools/gomad3/.toolchain' -prune -o -type f -name '*.go' -print0 | xargs -0 grep -L -e "Code generated by .* DO NOT EDIT." || true) && \
		$(GOIMPORTS) -w $$UNGENERATED_FILES

lint: lint-code lint-model lint-actions lint-api lint-protos lint-yaml
	@printf $(COLOR) "Run linters..."

lint-actions: $(ACTIONLINT)
	@printf $(COLOR) "Linting GitHub actions..."
	@$(ACTIONLINT)

lint-code: $(GOLANGCI_LINT) $(ERRORTYPE)
	@printf $(COLOR) "Linting code..."
	@$(GOLANGCI_LINT) run --verbose --build-tags $(ALL_TEST_TAGS) --timeout 10m --fix=$(GOLANGCI_LINT_FIX) --new-from-rev=$(GOLANGCI_LINT_BASE_REV) --config=.github/.golangci.yml
	@go vet -tags $(ALL_TEST_TAGS) -vettool="$(ERRORTYPE)" -style-check=false ./...

.PHONY: lint-model
lint-model:
	@printf $(COLOR) "Linting Lean model..."
	@cd model && $(LEAN_LAKE) build modelLintTests modelLint
	@cd model && $(LEAN_LAKE) exe modelLintTests
	@diagnostics=$$(mktemp); \
		trap 'rm -f "$$diagnostics"' EXIT; \
		status=0; \
		cd model && $(LEAN_LAKE) exe modelLintTests --controlled-violation 2>"$$diagnostics" || status=$$?; \
		test "$$status" -eq 1; \
		expected='[model-import-graph/shared-independence] forbidden qualified import path: Shared.Root -> ModelLint.Bridge -> Umpire.Core'; \
		test "$$(cat "$$diagnostics")" = "$$expected"
	@cd model && $(LEAN_LAKE) exe modelLint
	@cd model && $(LEAN_LAKE) --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs

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
	@$(BUF) lint $(INTERNAL_BINPB)
	@$(BUF) lint --config chasm/lib/buf.yaml $(CHASM_BINPB)

fmt: fmt-gofix fmt-imports fmt-protos fmt-yaml

# Some fixes enable others (e.g. rangeint may expose minmax opportunities),
# so - as recommended by the Go team - we run go fix in a loop until it reaches
# a fixed point. We check for "files updated" in the output rather than relying
# on the exit code alone, since go fix can exit non-zero without actually
# modifying any files (see https://github.com/golang/go/issues/77482).
# Note: go fix automatically skips generated files.
GOFIX_FLAGS ?=
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
	@find . -path './.gomad' -prune -o -path './tools/gomad3/.toolchain' -prune -o -type f -name '*.go' -print0 | \
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
