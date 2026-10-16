.PHONY: generate clean prune-cache clean-qualifications clock-audit compatibility-pack-qualification compiler-test-compiler core-qualification core-qualification-set intercept-test test-toolchain test-simulation qualification-set qualification-set-merge runner test-host overlay-test world-test toolchain upgrade-dossier validate validate-toolchain validate-compatibility validate-qualification tests-qualification-generate test-harness test-builder test-live-capability test-runtime test-upstream test

.DEFAULT_GOAL := generate

include version_generated.mk

GOMAD_TEST_TAGS := test_dep
ifneq ($(findstring gomad_native,$(GOFLAGS)),)
GOMAD_TEST_TAGS := test_dep,gomad_native
endif

GOMAD_STOCK_GO ?=

.PHONY: test-wasm wasm-helper wasm-test-helper
test-wasm:
	@$(MAKE) -C internal/backends/wasm test-conformance GOMAD_WASM_STOCK_GO='$(GOMAD_STOCK_GO)'
	@stock_go='$(GOMAD_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -tags test_dep -count=1 -timeout=15m \
		-run '^TestWASMTestsDoNotImportNativeBuilder$$|^TestGeneratedProjectionMatchesNativeSources$$|^TestGeneratedNetworkIsCurrent$$|^TestProjection' \
		./internal/backends/wasm ./internal/model/volume ./internal/model/volume/internal/projection ./internal/model/network/generate && \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -v -tags test_dep -count=1 -timeout=15m -skip '^TestNamespaceConformance' ./internal/backends/wasm/...

wasm-helper:
	$(MAKE) -C internal/backends/wasm helper

wasm-test-helper:
	$(MAKE) -C internal/backends/wasm test-helper

OVERLAY_FILES := $(shell find internal/backends/native/toolchain/runtime/overlay -type f)
VERSION_INPUTS := internal/backends/native/toolchain/version/version.json $(shell find internal/backends/native/toolchain/version -type f -not -name '*_test.go') internal/backends/native/cmd/gomadtool/version.go
BOUNDARY_INPUTS := internal/backends/native/transcript/boundary/manifest.json internal/backends/native/transcript/boundary/compiler-tests.json $(shell find internal/backends/native/transcript/boundary -type f -not -name '*_test.go') $(shell find internal/backends/native/generation/boundary -name '*.go' -not -name '*_test.go')
BOUNDARY_OUTPUTS := $(GOMAD_COMPILER_SPEC) $(GOMAD_EXPECTED_INTERCEPTS) $(GOMAD_BOUNDARY_REPORT) internal/backends/native/transcript/boundary_generated.go
COMPATIBILITY_INPUTS := $(shell find internal/backends/native/compatibilitypack/requests internal/backends/native/compatibilitypack/authoring -type f) internal/backends/native/compatibilitypack/working-directories.json internal/backends/native/compatibilitypack/schema.go internal/backends/native/compatibilitypack/v2_selection.go
COMPILER_TEST_OVERLAY := $(CURDIR)/.toolchain/compiler-test-overlay
COMPILER_TEST_COMPILE := $(CURDIR)/.toolchain/compiler-test/compile
GOMAD_CACHE_MAX_AGE_DAYS ?= 14
GOMAD_QUALIFICATION_ARTIFACTS ?= .toolchain/qualification
GOMAD_QUALIFICATION_OUTPUT ?= .toolchain/qualification-set.json
# GOMAD_QUALIFICATION_PRUNE=1 deletes each qualified seed's retained Campaigns
# once its evidence is in the set report, keeping its qualification report.
GOMAD_QUALIFICATION_PRUNE ?= 0
# GOMAD_QUALIFICATION_SHARD=INDEX/COUNT runs one ordinal-modulo shard of the
# manifest; qualification-set-merge combines the shard reports afterwards.
GOMAD_QUALIFICATION_SHARD ?=
# The ./tests qualification manifest is generated from the package's test list
# and the spec's defaults, overrides, and exclusions; never edit it by hand.
TESTS_QUALIFICATION_GENERATE := qualification-manifest-generate --root=$(CURDIR)/../.. \
	--spec=tools/gomad/internal/backends/native/qualification/tests.generator.json \
	--output=tools/gomad/internal/backends/native/qualification/tests.json

generate: $(VERSION_INPUTS) $(BOUNDARY_INPUTS) $(COMPATIBILITY_INPUTS)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool version-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool protocol-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool boundary-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool boundary-generate -check-compiler-tests
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool compatibility-pack generate --root=$(CURDIR)
	$(MAKE) tests-qualification-generate

.PHONY: lint-code
lint-code:
	$(MAKE) -C ../.. lint-code-gomad

tests-qualification-generate:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool $(TESTS_QUALIFICATION_GENERATE)

clean:
	rm -rf -- "$(CURDIR)/.bin" "$(COMPILER_TEST_OVERLAY)" "$(CURDIR)/.toolchain/compiler-test"

prune-cache:
	@age='$(GOMAD_CACHE_MAX_AGE_DAYS)'; case "$$age" in ''|*[!0-9]*) echo "GOMAD_CACHE_MAX_AGE_DAYS must be a non-negative integer" >&2; exit 2;; esac; \
	current_key="$$(tr -d '\n' < "$(CURDIR)/.toolchain/build-key")"; \
	test "$${#current_key}" -eq 64 || { echo "invalid active toolchain build key" >&2; exit 2; }; \
	case "$$current_key" in *[!0-9a-f]*) echo "invalid active toolchain build key" >&2; exit 2;; esac; \
	find "$(CURDIR)/.toolchain/builds" ! -path "$(CURDIR)/.toolchain/builds" -type d -prune -mtime "+$$age" ! -name "$$current_key" -exec rm -rf -- {} \;; \
	find "$(CURDIR)/.toolchain/generator-cache" -type f -mtime "+$$age" -delete; \
	find "$(CURDIR)/.toolchain/generator-cache" -depth -type d -empty -delete; \
	find "$(CURDIR)/.toolchain/downloads" -type f -mtime "+$$age" -delete

clean-qualifications:
	find "$(CURDIR)/.toolchain" ! -path "$(CURDIR)/.toolchain" -type d -name '*qualification*' -prune -exec rm -rf -- {} \;
	find "$(CURDIR)/.toolchain" -type f -name '*qualification*.json' -delete

test-toolchain: toolchain
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep,gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/toolchain

compiler-test-compiler: toolchain
	@mkdir -p $(COMPILER_TEST_OVERLAY) $(dir $(COMPILER_TEST_COMPILE))
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool boundary-generate \
		-compiler-test-overlay=$(COMPILER_TEST_OVERLAY) -goroot="$$(.toolchain/bin/go env GOROOT)"
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go build -trimpath \
		-overlay=$(COMPILER_TEST_OVERLAY)/overlay.json -o $(COMPILER_TEST_COMPILE) cmd/compile

intercept-test: compiler-test-compiler
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool boundary-generate -qualify
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-interception --go=$(CURDIR)/.toolchain/bin/go \
		--compiler=$(COMPILER_TEST_COMPILE)

clock-audit: toolchain
	bash internal/backends/native/conformance/scripts/clock_audit_test.sh

toolchain: validate-toolchain $(GOMAD_PATCH_FILE) $(OVERLAY_FILES) $(BOUNDARY_OUTPUTS)
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOCACHE=$(CURDIR)/.toolchain/generator-cache GOTOOLCHAIN=local GOWORK=off \
		go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool toolchain-build --root=$(CURDIR)

runner: toolchain
	@mkdir -p .bin
	env -u GOMADSEED -u GOMAD_CHILD_SEED CGO_ENABLED=0 GOTOOLCHAIN=local TZ=UTC \
		.toolchain/bin/go build -tags gomad_native -trimpath -o .bin/gomad ./cmd/gomad

.PHONY: build
build:
	go build ./...

qualification-set: runner
	@test -n "$(GOMAD_QUALIFICATION_MANIFEST)" || { echo "GOMAD_QUALIFICATION_MANIFEST is required" >&2; exit 2; }
	@test -n "$(GOMAD_QUALIFICATION_WORKDIR)" || { echo "GOMAD_QUALIFICATION_WORKDIR is required" >&2; exit 2; }
	@case '$(GOMAD_QUALIFICATION_PRUNE)' in 0|1) ;; *) echo "GOMAD_QUALIFICATION_PRUNE must be 0 or 1" >&2; exit 2;; esac
	env -u GOMADSEED -u GOMAD_CHILD_SEED .bin/gomad qualify-set \
		--manifest="$(GOMAD_QUALIFICATION_MANIFEST)" --working-dir="$(GOMAD_QUALIFICATION_WORKDIR)" \
		--artifacts="$(GOMAD_QUALIFICATION_ARTIFACTS)" --output="$(GOMAD_QUALIFICATION_OUTPUT)" \
		$(if $(filter 1,$(GOMAD_QUALIFICATION_PRUNE)),--prune-qualified-artifacts) \
		$(if $(GOMAD_QUALIFICATION_SHARD),--shard="$(GOMAD_QUALIFICATION_SHARD)")

qualification-set-merge: runner
	@test -n "$(GOMAD_QUALIFICATION_MANIFEST)" || { echo "GOMAD_QUALIFICATION_MANIFEST is required" >&2; exit 2; }
	@test -n "$(GOMAD_QUALIFICATION_SHARD_REPORTS)" || { echo "GOMAD_QUALIFICATION_SHARD_REPORTS is required" >&2; exit 2; }
	env -u GOMADSEED -u GOMAD_CHILD_SEED .bin/gomad merge-set \
		--manifest="$(GOMAD_QUALIFICATION_MANIFEST)" --output="$(GOMAD_QUALIFICATION_OUTPUT)" \
		$(GOMAD_QUALIFICATION_SHARD_REPORTS)

# Packs are scoped to one platform, so each host qualifies only the requests
# that name it, each in the working directory that
# internal/backends/native/compatibilitypack/working-directories.json maps it to.
compatibility-pack-qualification: toolchain
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool compatibility-pack qualify --root=$(CURDIR) --all

core-qualification-set: GOMAD_QUALIFICATION_MANIFEST := $(CURDIR)/internal/evidence/qualification/core.json
core-qualification-set: GOMAD_QUALIFICATION_WORKDIR := $(CURDIR)/internal/evidence/qualification/corpus
core-qualification-set: GOMAD_QUALIFICATION_ARTIFACTS := $(CURDIR)/.toolchain/core-qualification
core-qualification-set: GOMAD_QUALIFICATION_OUTPUT := $(CURDIR)/.toolchain/core-qualification-set.json
core-qualification-set: qualification-set

core-qualification: compatibility-pack-qualification core-qualification-set

upgrade-dossier: core-qualification
	GOWORK=off go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool upgrade-dossier \
		-baseline-ref="$(GOMAD_BASELINE_REF)" -corpus-report=".toolchain/core-qualification-set.json" $(if $(GOMAD_APPROVED_BOUNDARY_DIFF_SHA256),-approve-boundary-diff="$(GOMAD_APPROVED_BOUNDARY_DIFF_SHA256)",)

# Resolve stock Go before the patched test driver prepends its GOROOT/bin to PATH.
# Several tests run their own test binary as a seeded target, and the seeded
# runtime refuses the Green Tea collector, so the test binaries use the classic one.
test-host: toolchain
	@stock_go="$${GOMAD_STOCK_GO:-}"; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOFLAGS= GOWORK=off GOTOOLCHAIN=$(GOMAD_GO_VERSION) go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOMAD_STOCK_GO="$$stock_go" GOEXPERIMENT=nogreenteagc GOWORK=off .toolchain/bin/go test -tags test_dep,gomad_native \
		./cmd/... ./internal/runner/... go.temporal.io/server/tools/gomad/internal/backends/native/execution/... ./internal/model/clock/... \
		./internal/evidence/qualification/... ./internal/backends/target/... ./internal/evidence/record/... ./internal/evidence/artifact/... ./internal/model/choice/... \
		./internal/evidence/artifact/capture/... ./internal/model/volume/... go.temporal.io/server/tools/gomad/internal/backends/native/transcript/... ./internal/model/world/... \
		go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/installation \
		go.temporal.io/server/tools/gomad/internal/backends/native/upgrade/... go.temporal.io/server/tools/gomad/internal/backends/native/preparation/... go.temporal.io/server/tools/gomad/internal/backends/native/qualification/... \
		go.temporal.io/server/tools/gomad/internal/backends/native/generation/... go.temporal.io/server/tools/gomad/internal/backends/native/build/... go.temporal.io/server/tools/gomad/internal/backends/native/cmd/... \
		go.temporal.io/server/tools/gomad/internal/backends/native/compatibilitypack/... go.temporal.io/server/tools/gomad/internal/backends/native/sourceinventory \
		go.temporal.io/server/tools/gomad/internal/backends/native/conformance ./internal/host/... ./internal/archcheck ./internal/model/iowire ./internal/model/strictjson

# internal/gomadsim keeps its tests external because testing reaches the
# package through os.
overlay-test: toolchain
	.toolchain/bin/go test -tags test_dep,gomad_native internal/gomadfs internal/gomadwire internal/gomadchoicewire internal/gomadmodelwire \
		internal/gomadsim internal/gomadio internal/gomadio/mount internal/gomadvfd syscall os cmd/internal/gomadcap
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOEXPERIMENT=nogreenteagc GOWORK=off \
		.toolchain/bin/go test -tags test_dep,gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/runtime/testdata/vfdpointer
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOEXPERIMENT=nogreenteagc GOWORK=off \
		.toolchain/bin/go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/runtime/testdata/vfdpointer -root=$(CURDIR) -go=$(CURDIR)/.toolchain/bin/go
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOEXPERIMENT=nogreenteagc GOWORK=off \
		.toolchain/bin/go test -tags test_dep,gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/runtime/testdata/vfdnative
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOEXPERIMENT=nogreenteagc GOWORK=off \
		.toolchain/bin/go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/runtime/testdata/vfdnative -root=$(CURDIR) -go=$(CURDIR)/.toolchain/bin/go

# The gomad_toolchain tests of internal/model/cluster need a seeded
# runtime, so they run through the exec wrapper. The tests that need the
# Runner's simulation transport skip there; the integration-tagged Runner test
# is their executor. The strict node-clock delay test has a recorded watchdog
# failure and remains a finding; its forward-mode regression runs separately.
test-simulation: toolchain
	env -u GOMADSEED CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off TZ=UTC GOMAD_CHILD_SEED=89 \
		$(CURDIR)/.toolchain/bin/go test -exec $(CURDIR)/internal/backends/native/conformance/scripts/exec.sh \
		-count=1 -tags test_dep,gomad_native,gomad_toolchain ./internal/model/cluster
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep,gomad_native,integration \
		-run '^TestRootProcessSimulationUsesRunnerTransport$$/^Test(Scenario|ProcessExploration|ProcessAndInProcess|ProcessNetwork(Handle|HeldFIFO|StallExpiry|StallOrdering|PartitionedDialStall|FramedPartition)|ProcessFilesystemHandle|ProcessBackend(Carries|Resets|Routes|Preserves|Crash|Model))' ./internal/runner/internal/exploration
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep,gomad_native,integration \
		-run '^TestRootProcessSimulationUsesRunnerTransport$$/^TestProcessBackendSynchronizesNodeClockWithModelDelay$$/^forward$$' ./internal/runner/internal/exploration

world-test:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go test -race -tags test_dep ./internal/model/world/...

validate: validate-toolchain validate-compatibility validate-qualification

validate-toolchain:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool version-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool protocol-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool boundary-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool boundary-generate -check-compiler-tests
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool patch-validate --root=$(CURDIR)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool script-validate --root=$(CURDIR)

validate-compatibility:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool compatibility-pack check --root=$(CURDIR)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go test -count=1 -tags test_dep,gomad_native -run '^TestHostPacksBindCurrentProfile$$' go.temporal.io/server/tools/gomad/internal/backends/native/compatibilitypack

validate-qualification:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool $(TESTS_QUALIFICATION_GENERATE) -check

ifneq ($(findstring gomad_native,$(GOFLAGS)),)
test-harness: toolchain
endif
test-harness:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go test -tags $(GOMAD_TEST_TAGS) \
		./cmd/gomad/... ./internal/host/... ./internal/runner/... ./internal/evidence/... ./internal/backends/target/... ./internal/archcheck

test-builder:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-builder --go="$$(command -v go)"

test-live-capability: toolchain
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-live-capability --go=$(CURDIR)/.toolchain/bin/go

test-runtime: toolchain
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-runtime --go=$(CURDIR)/.toolchain/bin/go

test-upstream: toolchain
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-upstream --go=$(CURDIR)/.toolchain/bin/go

ifneq ($(findstring gomad_native,$(GOFLAGS)),)
test: test-harness test-toolchain intercept-test test-host overlay-test test-simulation world-test test-builder test-live-capability test-runtime test-upstream
	@printf 'gomad all black-box tiers passed\n'
else
test: test-harness world-test test-wasm
endif
