.PHONY: generate clock-audit intercept-test patch-test qualification-set runner runner-test overlay-test world-test toolchain upgrade-dossier validate test-harness test-builder test-runtime test-upstream test

include version_generated.mk

OVERLAY_FILES := $(shell find overlay -type f)
VERSION_INPUTS := version.json $(shell find internal/version internal/versiongen -type f -not -name '*_test.go')
VERSION_OUTPUTS := toolchain-version.sh version_generated.mk internal/version/generated.go
IOWIRE_INPUTS := protocol/iowire.json protocol/iowire.go.tmpl protocol/iowire_test.go.tmpl $(shell find internal/protocolgen -type f)
IOWIRE_OUTPUTS := internal/iowire/wire_generated.go internal/iowire/wire_generated_test.go overlay/src/internal/gomadwire/wire_generated.go overlay/src/internal/gomadwire/wire_generated_test.go
BOUNDARY_INPUTS := boundary/manifest.json $(shell find internal/boundary internal/boundarygen -type f -not -name '*_test.go')
BOUNDARY_OUTPUTS := $(GOMADV3_COMPILER_SPEC) $(GOMADV3_EXPECTED_INTERCEPTS) $(GOMADV3_BOUNDARY_REPORT) internal/ioprofile/boundary_generated.go

generate: $(VERSION_INPUTS) $(IOWIRE_INPUTS) $(BOUNDARY_INPUTS)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/versiongen
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/protocolgen
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/boundarygen

patch-test: toolchain
	bash ./materialize_patch_test.sh
	bash ./regenerate_patch_test.sh

intercept-test: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./internal/boundarygen -qualify
	bash ./intercept_report_test.sh
	bash ./intercept_failure_test.sh
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go -C testdata test -tags test_dep ./intercept -count=1
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go -C testdata test -tags test_dep -gcflags=all=-l ./intercept -count=1

clock-audit: toolchain
	bash ./clock_audit_test.sh

toolchain: validate build.sh materialize-patch.sh toolchain-version.sh $(GOMADV3_PATCH_FILE) $(OVERLAY_FILES) $(IOWIRE_OUTPUTS) $(BOUNDARY_OUTPUTS)
	./build.sh

runner: toolchain
	@mkdir -p .bin
	env -u GOMADSEED -u GOMADV3_CHILD_SEED CGO_ENABLED=0 GOTOOLCHAIN=local TZ=UTC \
		.toolchain/bin/go build -trimpath -o .bin/gomad ./cmd/gomad

qualification-set: runner
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go run ./internal/qualificationgen

upgrade-dossier:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache GOWORK=off go run ./internal/upgradegen \
		-baseline-ref="$(GOMADV3_BASELINE_REF)" -corpus-report="$(GOMADV3_CORPUS_REPORT)"

runner-test: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./cmd/... ./internal/... ./world/...

overlay-test: toolchain
	.toolchain/bin/go test -tags test_dep internal/gomadwire internal/gomadio/mount

world-test:
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go test -race -tags test_dep ./world/...

validate:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/versiongen -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/protocolgen -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/boundarygen -check
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/qualificationgen -check
	./test.sh validate

test-harness:
	bash ./testlib_test.sh
	bash ./test_modes_test.sh

test-builder: toolchain
	./test.sh test-builder

test-runtime: toolchain
	./test.sh test-runtime

test-upstream: toolchain
	./test.sh test-upstream

test: test-harness patch-test intercept-test runner-test overlay-test world-test
	./test.sh
