.PHONY: generate clock-audit compiler-test-compiler core-qualification intercept-test patch-test qualification-set runner runner-test overlay-test world-test toolchain upgrade-dossier validate test-harness test-builder test-runtime test-upstream test

include version_generated.mk

OVERLAY_FILES := $(shell find toolchain/runtime/overlay -type f)
VERSION_INPUTS := toolchain/version/version.json $(shell find toolchain/version -type f -not -name '*_test.go') toolchain/cmd/gomadtool/version.go
VERSION_OUTPUTS := version_generated.mk toolchain/version/generated.go
IOWIRE_INPUTS := deterministicio/schema/iowire.json deterministicio/schema/iowire.go.tmpl deterministicio/schema/iowire_test.go.tmpl choice/schema/choicewire.json choice/schema/choicewire.go.tmpl choice/schema/choicewire_test.go.tmpl choice/schema/choicewire_runtime.go.tmpl toolchain/runtime/overlay/src/runtime/gomad.go toolchain/runtime/go1.26.4.patch toolchain/internal/generate/protocol.go
IOWIRE_OUTPUTS := deterministicio/internal/wire/wire_generated.go deterministicio/internal/wire/wire_generated_test.go toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go toolchain/runtime/overlay/src/internal/gomadwire/wire_generated_test.go choice/internal/wire/wire_generated.go choice/internal/wire/wire_generated_test.go toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated_test.go toolchain/runtime/overlay/src/runtime/gomad_choicewire_generated.go
BOUNDARY_INPUTS := deterministicio/boundary/manifest.json deterministicio/boundary/compiler-tests.json $(shell find deterministicio/boundary -type f -not -name '*_test.go') $(shell find toolchain/internal/generate -name 'boundary*.go' -not -name '*_test.go')
BOUNDARY_OUTPUTS := $(GOMADV3_COMPILER_SPEC) $(GOMADV3_EXPECTED_INTERCEPTS) $(GOMADV3_BOUNDARY_REPORT) deterministicio/boundary_generated.go
COMPILER_TEST_OVERLAY := $(CURDIR)/.toolchain/compiler-test-overlay
COMPILER_TEST_COMPILE := $(CURDIR)/.toolchain/compiler-test/compile
GOMADV3_QUALIFICATION_ARTIFACTS ?= .toolchain/qualification
GOMADV3_QUALIFICATION_OUTPUT ?= .toolchain/qualification-set.json

generate: $(VERSION_INPUTS) $(IOWIRE_INPUTS) $(BOUNDARY_INPUTS)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool version-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool protocol-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool boundary-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool boundary-generate -check-compiler-tests

patch-test: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./toolchain

compiler-test-compiler: toolchain
	@mkdir -p $(COMPILER_TEST_OVERLAY) $(dir $(COMPILER_TEST_COMPILE))
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./toolchain/cmd/gomadtool boundary-generate \
		-compiler-test-overlay=$(COMPILER_TEST_OVERLAY) -goroot="$$(.toolchain/bin/go env GOROOT)"
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go build -trimpath \
		-overlay=$(COMPILER_TEST_OVERLAY)/overlay.json -o $(COMPILER_TEST_COMPILE) cmd/compile

intercept-test: compiler-test-compiler
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./toolchain/cmd/gomadtool boundary-generate -qualify
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go run ./toolchain/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-interception --go=$(CURDIR)/.toolchain/bin/go \
		--compiler=$(COMPILER_TEST_COMPILE)

clock-audit: toolchain
	bash ./clock_audit_test.sh

toolchain: validate $(GOMADV3_PATCH_FILE) $(OVERLAY_FILES) $(IOWIRE_OUTPUTS) $(BOUNDARY_OUTPUTS)
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOCACHE=$(CURDIR)/.toolchain/generator-cache GOTOOLCHAIN=local GOWORK=off \
		go run ./toolchain/cmd/gomadtool toolchain-build --root=$(CURDIR)

runner: toolchain
	@mkdir -p .bin
	env -u GOMADSEED -u GOMADV3_CHILD_SEED CGO_ENABLED=0 GOTOOLCHAIN=local TZ=UTC \
		.toolchain/bin/go build -trimpath -o .bin/gomad ./cmd/gomad

qualification-set: runner
	@test -n "$(GOMADV3_QUALIFICATION_MANIFEST)" || { echo "GOMADV3_QUALIFICATION_MANIFEST is required" >&2; exit 2; }
	@test -n "$(GOMADV3_QUALIFICATION_WORKDIR)" || { echo "GOMADV3_QUALIFICATION_WORKDIR is required" >&2; exit 2; }
	env -u GOMADSEED -u GOMADV3_CHILD_SEED .bin/gomad qualify-set \
		--manifest="$(GOMADV3_QUALIFICATION_MANIFEST)" --working-dir="$(GOMADV3_QUALIFICATION_WORKDIR)" \
		--artifacts="$(GOMADV3_QUALIFICATION_ARTIFACTS)" --output="$(GOMADV3_QUALIFICATION_OUTPUT)"

core-qualification: GOMADV3_QUALIFICATION_MANIFEST := $(CURDIR)/qualification/core.json
core-qualification: GOMADV3_QUALIFICATION_WORKDIR := $(CURDIR)/qualification/corpus
core-qualification: GOMADV3_QUALIFICATION_ARTIFACTS := $(CURDIR)/.toolchain/core-qualification
core-qualification: GOMADV3_QUALIFICATION_OUTPUT := $(CURDIR)/.toolchain/core-qualification-set.json
core-qualification: qualification-set

upgrade-dossier: core-qualification
	GOWORK=off go run ./toolchain/cmd/gomadtool upgrade-dossier \
		-baseline-ref="$(GOMADV3_BASELINE_REF)" -corpus-report=".toolchain/core-qualification-set.json" $(if $(GOMADV3_APPROVED_BOUNDARY_DIFF_SHA256),-approve-boundary-diff="$(GOMADV3_APPROVED_BOUNDARY_DIFF_SHA256)",)

runner-test: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep \
		. ./cmd/... ./runner/... ./qualification/... ./target/... ./evidence/... ./choice/... \
		./deterministicio/... ./world/... ./toolchain ./toolchain/version ./toolchain/cmd/... \
		./toolchain/internal/... ./internal/...

overlay-test: toolchain
	.toolchain/bin/go test -tags test_dep internal/gomadwire internal/gomadchoicewire internal/gomadio/mount

world-test:
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go test -race -tags test_dep ./world/...

validate:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool version-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool protocol-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool boundary-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool boundary-generate -check-compiler-tests
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool patch-validate --root=$(CURDIR)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./toolchain/cmd/gomadtool script-validate --root=$(CURDIR)

test-harness:
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go test -tags test_dep \
		./internal/hostexec ./toolchain/internal/conformance ./toolchain/cmd/gomadtool

test-builder:
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go run ./toolchain/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-builder --go="$$(command -v go)"

test-runtime: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go run ./toolchain/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-runtime --go=$(CURDIR)/.toolchain/bin/go

test-upstream: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go run ./toolchain/cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-upstream --go=$(CURDIR)/.toolchain/bin/go

test: test-harness patch-test intercept-test runner-test overlay-test world-test test-builder test-runtime test-upstream
	@printf 'gomadv3 all black-box tiers passed\n'
