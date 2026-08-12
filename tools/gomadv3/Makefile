.PHONY: generate runner runner-test overlay-test world-test toolchain validate test-harness test

OVERLAY_FILES := $(shell find overlay -type f)
IOWIRE_INPUTS := protocol/iowire.json protocol/iowire.go.tmpl protocol/iowire_test.go.tmpl $(shell find internal/protocolgen -type f)
IOWIRE_OUTPUTS := internal/iowire/wire_generated.go internal/iowire/wire_generated_test.go overlay/src/internal/gomadwire/wire_generated.go overlay/src/internal/gomadwire/wire_generated_test.go

generate: $(IOWIRE_INPUTS)
	GOCACHE=$(CURDIR)/.toolchain/generator-cache go run ./internal/protocolgen

toolchain: validate build.sh go1.26.4.patch $(OVERLAY_FILES) $(IOWIRE_OUTPUTS)
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

test: test-harness runner-test overlay-test world-test
	./test.sh
