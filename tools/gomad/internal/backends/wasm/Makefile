.PHONY: test test-conformance test-model lint-rust fmt-check

test:
	$(MAKE) -C ../../.. test-wasm

test-conformance:
	$(MAKE) -C ../../.. validate-model-corpora

test-model:
	go test -tags test_dep -count=1 ../../../internal/model/wasi ./cluster

lint-rust:
	$(MAKE) -C ../../.. lint-rust

fmt-check:
	$(MAKE) -C ../../.. fmt-rust
