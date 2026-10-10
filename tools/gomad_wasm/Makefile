.PHONY: helper test test-helper test-go test-model qualify-runtime lint-code

GOMAD_WASM_STOCK_GO ?=

lint-code:
	$(MAKE) -C ../.. lint-code-gomad-wasm

helper:
	cargo +1.94.1 build --manifest-path wasmhost/Cargo.toml --release --locked
	@if test -n "$$CARGO_TARGET_DIR"; then \
		mkdir -p wasmhost/target/release && \
		if ! cmp -s "$$CARGO_TARGET_DIR/release/gomad3-wasmhost" wasmhost/target/release/gomad3-wasmhost; then \
			cp "$$CARGO_TARGET_DIR/release/gomad3-wasmhost" wasmhost/target/release/gomad3-wasmhost && \
			cmp "$$CARGO_TARGET_DIR/release/gomad3-wasmhost" wasmhost/target/release/gomad3-wasmhost; \
		fi; \
	fi

test: helper test-helper test-go

test-helper:
	cargo +1.94.1 test --manifest-path wasmhost/Cargo.toml --release --locked

test-go:
	@stock_go='$(GOMAD_WASM_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD3_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -v -tags test_dep -count=1 -timeout=15m ./...

test-model:
	@stock_go='$(GOMAD_WASM_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		PATH="$$(dirname "$$stock_go"):$$PATH" "$$stock_go" test -tags test_dep -count=1 \
		-run 'TestCaptured|TestEnvironment|TestNamespace|TestExploratory|TestCapacity|TestPathOpen|TestOffered' ./wasi

qualify-runtime: helper
	@test -n "$(GOMAD_WASM_STOCK_GO)" && test -n "$$GOMAD_WASM_QUALIFICATION_ROOT" && test -n "$$GOMAD_WASM_QUALIFICATION_MANIFEST"
	@for index in $$(seq 0 31); do \
		env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
			GOMAD3_STOCK_GO='$(GOMAD_WASM_STOCK_GO)' GOMAD_WASM_QUALIFICATION_INDEX="$$index" \
			'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 -timeout=10m \
			-run '^TestCooperativeRuntimeQualification$$' ./wasi || exit $$?; \
	done
	@env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD3_STOCK_GO='$(GOMAD_WASM_STOCK_GO)' \
		'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 \
		-run '^TestCooperativeRuntimeQualificationAggregate$$' ./wasi
