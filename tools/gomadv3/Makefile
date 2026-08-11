.PHONY: toolchain validate test-harness test

OVERLAY_FILES := $(shell find overlay -type f)

toolchain: validate build.sh go1.26.4.patch $(OVERLAY_FILES)
	./build.sh

validate:
	./test.sh validate

test-harness:
	bash ./testlib_test.sh

test: test-harness toolchain
	./test.sh
