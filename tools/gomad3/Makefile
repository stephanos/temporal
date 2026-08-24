.PHONY: generate clean prune-cache clean-qualifications clock-audit compatibility-pack-qualification compiler-test-compiler core-qualification intercept-test test-toolchain qualification-set runner test-host overlay-test world-test toolchain upgrade-dossier validate validate-toolchain validate-compatibility test-harness test-builder test-live-capability test-runtime test-upstream test

include version_generated.mk

OVERLAY_FILES := $(shell find toolchain/runtime/overlay -type f)
VERSION_INPUTS := toolchain/version/version.json $(shell find toolchain/version -type f -not -name '*_test.go') cmd/gomadtool/version.go
BOUNDARY_INPUTS := deterministicio/boundary/manifest.json deterministicio/boundary/compiler-tests.json $(shell find deterministicio/boundary -type f -not -name '*_test.go') $(shell find internal/gomadtool/generation/boundary -name '*.go' -not -name '*_test.go')
BOUNDARY_OUTPUTS := $(GOMAD3_COMPILER_SPEC) $(GOMAD3_EXPECTED_INTERCEPTS) $(GOMAD3_BOUNDARY_REPORT) deterministicio/boundary_generated.go
COMPATIBILITY_INPUTS := $(shell find internal/compatibilitypack/requests internal/compatibilitypack/authoring -type f) internal/compatibilitypack/schema.go internal/compatibilitypack/v2_selection.go
COMPILER_TEST_OVERLAY := $(CURDIR)/.toolchain/compiler-test-overlay
COMPILER_TEST_COMPILE := $(CURDIR)/.toolchain/compiler-test/compile
GOMAD3_CACHE_MAX_AGE_DAYS ?= 14
GOMAD3_QUALIFICATION_ARTIFACTS ?= .toolchain/qualification
GOMAD3_QUALIFICATION_OUTPUT ?= .toolchain/qualification-set.json

generate: $(VERSION_INPUTS) $(BOUNDARY_INPUTS) $(COMPATIBILITY_INPUTS)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool version-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool protocol-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool boundary-generate
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool boundary-generate -check-compiler-tests
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool compatibility-pack generate --root=$(CURDIR)

clean:
	rm -rf -- "$(CURDIR)/.bin" "$(COMPILER_TEST_OVERLAY)" "$(CURDIR)/.toolchain/compiler-test"

prune-cache:
	@age='$(GOMAD3_CACHE_MAX_AGE_DAYS)'; case "$$age" in ''|*[!0-9]*) echo "GOMAD3_CACHE_MAX_AGE_DAYS must be a non-negative integer" >&2; exit 2;; esac; \
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
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./toolchain

compiler-test-compiler: toolchain
	@mkdir -p $(COMPILER_TEST_OVERLAY) $(dir $(COMPILER_TEST_COMPILE))
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./cmd/gomadtool boundary-generate \
		-compiler-test-overlay=$(COMPILER_TEST_OVERLAY) -goroot="$$(.toolchain/bin/go env GOROOT)"
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go build -trimpath \
		-overlay=$(COMPILER_TEST_OVERLAY)/overlay.json -o $(COMPILER_TEST_COMPILE) cmd/compile

intercept-test: compiler-test-compiler
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./cmd/gomadtool boundary-generate -qualify
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ./cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-interception --go=$(CURDIR)/.toolchain/bin/go \
		--compiler=$(COMPILER_TEST_COMPILE)

clock-audit: toolchain
	bash ./internal/gomadtool/conformance/scripts/clock_audit_test.sh

toolchain: validate-toolchain $(GOMAD3_PATCH_FILE) $(OVERLAY_FILES) $(BOUNDARY_OUTPUTS)
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOCACHE=$(CURDIR)/.toolchain/generator-cache GOTOOLCHAIN=local GOWORK=off \
		go run ./cmd/gomadtool toolchain-build --root=$(CURDIR)

runner: toolchain
	@mkdir -p .bin
	env -u GOMADSEED -u GOMAD3_CHILD_SEED CGO_ENABLED=0 GOTOOLCHAIN=local TZ=UTC \
		.toolchain/bin/go build -trimpath -o .bin/gomad ./cmd/gomad

qualification-set: runner
	@test -n "$(GOMAD3_QUALIFICATION_MANIFEST)" || { echo "GOMAD3_QUALIFICATION_MANIFEST is required" >&2; exit 2; }
	@test -n "$(GOMAD3_QUALIFICATION_WORKDIR)" || { echo "GOMAD3_QUALIFICATION_WORKDIR is required" >&2; exit 2; }
	env -u GOMADSEED -u GOMAD3_CHILD_SEED .bin/gomad qualify-set \
		--manifest="$(GOMAD3_QUALIFICATION_MANIFEST)" --working-dir="$(GOMAD3_QUALIFICATION_WORKDIR)" \
		--artifacts="$(GOMAD3_QUALIFICATION_ARTIFACTS)" --output="$(GOMAD3_QUALIFICATION_OUTPUT)"

compatibility-pack-qualification: toolchain
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./cmd/gomadtool compatibility-pack qualify --root=$(CURDIR) --request=internal/compatibilitypack/requests/reflect2-go126.json --working-dir=$(CURDIR)/../..
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./cmd/gomadtool compatibility-pack qualify --root=$(CURDIR) --request=internal/compatibilitypack/requests/modernc-libc-xsys-v041.json --working-dir=$(CURDIR)/internal/compatibilitypack/testdata/v041
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./cmd/gomadtool compatibility-pack qualify --root=$(CURDIR) --request=internal/compatibilitypack/requests/modernc-libc-xsys-v047.json --working-dir=$(CURDIR)/qualification/corpus
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./cmd/gomadtool compatibility-pack qualify --root=$(CURDIR) --request=internal/compatibilitypack/requests/modernc-libc-xsys-v047-isatty-v021.json --working-dir=$(CURDIR)/../..

core-qualification: compatibility-pack-qualification
core-qualification: GOMAD3_QUALIFICATION_MANIFEST := $(CURDIR)/qualification/core.json
core-qualification: GOMAD3_QUALIFICATION_WORKDIR := $(CURDIR)/qualification/corpus
core-qualification: GOMAD3_QUALIFICATION_ARTIFACTS := $(CURDIR)/.toolchain/core-qualification
core-qualification: GOMAD3_QUALIFICATION_OUTPUT := $(CURDIR)/.toolchain/core-qualification-set.json
core-qualification: qualification-set

upgrade-dossier: core-qualification
	GOWORK=off go run ./cmd/gomadtool upgrade-dossier \
		-baseline-ref="$(GOMAD3_BASELINE_REF)" -corpus-report=".toolchain/core-qualification-set.json" $(if $(GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256),-approve-boundary-diff="$(GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256)",)

test-host: toolchain
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep \
		. ./cmd/... ./runner/... ./qualification/... ./target/... ./record/... ./artifact/... ./choice/... \
		./deterministicio/... ./world/... ./simulation/... ./toolchain ./toolchain/version ./upgrade/... ./internal/...

overlay-test: toolchain
	.toolchain/bin/go test -tags test_dep internal/gomadwire internal/gomadchoicewire internal/gomadio/mount

world-test:
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go test -race -tags test_dep ./world/...

validate: validate-toolchain validate-compatibility

validate-toolchain:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool version-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool protocol-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool boundary-generate -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool boundary-generate -check-compiler-tests
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool patch-validate --root=$(CURDIR)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool script-validate --root=$(CURDIR)

validate-compatibility:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./cmd/gomadtool compatibility-pack check --root=$(CURDIR)

test-harness:
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go test -tags test_dep \
		./internal/hostexec ./internal/gomadtool/conformance ./cmd/gomadtool

test-builder:
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ./cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-builder --go="$$(command -v go)"

test-live-capability: toolchain
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ./cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-live-capability --go=$(CURDIR)/.toolchain/bin/go

test-runtime: toolchain
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ./cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-runtime --go=$(CURDIR)/.toolchain/bin/go

test-upstream: toolchain
	env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ./cmd/gomadtool test \
		--root=$(CURDIR) --mode=test-upstream --go=$(CURDIR)/.toolchain/bin/go

test: test-harness test-toolchain intercept-test test-host overlay-test world-test test-builder test-live-capability test-runtime test-upstream
	@printf 'gomad3 all black-box tiers passed\n'
