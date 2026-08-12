.PHONY: generate intercept-test patch-test runner runner-test overlay-test world-test toolchain validate test-harness test-builder test-runtime test-upstream test

OVERLAY_FILES := $(shell find overlay -type f)
IOWIRE_INPUTS := protocol/iowire.json protocol/iowire.go.tmpl protocol/iowire_test.go.tmpl $(shell find internal/protocolgen -type f)
IOWIRE_OUTPUTS := internal/iowire/wire_generated.go internal/iowire/wire_generated_test.go overlay/src/internal/gomadwire/wire_generated.go overlay/src/internal/gomadwire/wire_generated_test.go

generate: $(IOWIRE_INPUTS)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/protocolgen

patch-test: toolchain
	bash ./materialize_patch_test.sh
	bash ./regenerate_patch_test.sh

intercept-test: toolchain
	bash ./intercept_report_test.sh
	bash ./intercept_failure_test.sh
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go -C testdata test -tags test_dep ./intercept -count=1
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go -C testdata test -tags test_dep -gcflags=all=-l ./intercept -count=1

toolchain: validate build.sh materialize-patch.sh toolchain-version.sh go1.26.4.patch $(OVERLAY_FILES) $(IOWIRE_OUTPUTS)
	./build.sh

runner: toolchain
	@mkdir -p .bin
	env -u GOMADSEED -u GOMADV3_CHILD_SEED CGO_ENABLED=0 GOTOOLCHAIN=local TZ=UTC \
		.toolchain/bin/go build -trimpath -o .bin/gomad ./cmd/gomad

runner-test: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./cmd/... ./internal/... ./world/...

overlay-test: toolchain
	.toolchain/bin/go test -tags test_dep internal/gomadwire internal/gomadio/mount

world-test:
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go test -race -tags test_dep ./world/...

validate:
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/protocolgen -check
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
