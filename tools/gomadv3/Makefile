.PHONY: runner runner-test world-test toolchain validate test-harness test

OVERLAY_FILES := $(shell find overlay -type f)

toolchain: validate build.sh go1.26.4.patch $(OVERLAY_FILES)
	./build.sh

runner: toolchain
	@mkdir -p .bin
	env -u GOMADSEED -u GOMADV3_CHILD_SEED CGO_ENABLED=0 GOTOOLCHAIN=local TZ=UTC \
		.toolchain/bin/go build -trimpath -o .bin/gomad ./cmd/gomad

runner-test: toolchain
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./cmd/... ./internal/... ./world/...

world-test:
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOWORK=off go test -race -tags test_dep ./world/...

validate:
	./test.sh validate

test-harness:
	bash ./testlib_test.sh

test: test-harness runner-test world-test
	./test.sh
