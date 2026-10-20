.PHONY: helper conformance-helper test test-helper test-conformance test-go test-model test-determinism qualify-runtime qualify-workloads prototype lint-code lint-rust fmt fmt-check

GOMAD_WASM_STOCK_GO ?= $(GOMAD_STOCK_GO)

fmt:
	cargo +1.99.1 fmt --manifest-path wasmhost/Cargo.toml

fmt-check:
	cargo +1.99.1 fmt --manifest-path wasmhost/Cargo.toml -- --check

lint-rust: fmt-check
	@for features in '' conformance diagnostic-profile diagnostic-fuel-off guest-profile guest-profile,diagnostic-fuel-off stress-jitter; do \
		CARGO_BUILD_WARNINGS=deny cargo +1.99.1 clippy --manifest-path wasmhost/Cargo.toml --release --locked \
			--all-targets --no-deps --features "$$features" -- -D warnings || exit $$?; \
	done

lint-code: lint-rust
	$(MAKE) -C ../../../../.. lint-code-gomad-wasm

helper:
	cargo +1.99.1 build --manifest-path wasmhost/Cargo.toml --release --locked
	@helper_source="$${CARGO_TARGET_DIR:-wasmhost/target}/release/gomad-wasmhost"; \
		mkdir -p wasmhost/target/release && \
		helper_tmp="$$(mktemp wasmhost/target/release/gomad-wasmhost.XXXXXX)" && \
		cp "$$helper_source" "$$helper_tmp" && \
		chmod 755 "$$helper_tmp" && \
		cmp "$$helper_source" "$$helper_tmp" && \
		mv -f "$$helper_tmp" wasmhost/target/release/gomad-wasmhost

test: test-helper test-determinism
	$(MAKE) test-go

# Source inventories and the transport boundary corpus run without long
# runtime qualification or the opt-in stress producer.
test-determinism:
	cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked --test inventory --test helper_corpus
	cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked --test engine determinism_refuses
	@stock_go='$(GOMAD_WASM_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	cd ../../.. && env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -tags test_dep -count=1 -timeout=15m \
		./internal/backends/wasm/determinism/... ./internal/archcheck ./internal/backends/wasm && \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -tags test_dep -count=1 -timeout=15m \
		-run '^TestDeterminism|^TestTransportInputValidation$$' ./internal/backends/wasm/wasi

# The helper transcript corpus also runs on the diagnostic builds (fn-187); they
# change the helper binary, so they build in their own target directory.
test-helper:
	cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked
	cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked \
		--features diagnostic-profile --target-dir '$(CURDIR)/wasmhost/target-diagnostic' --test helper_corpus
	cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked \
		--features diagnostic-fuel-off --target-dir '$(CURDIR)/wasmhost/target-diagnostic' --test helper_corpus
	cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked \
		--features guest-profile --target-dir '$(CURDIR)/wasmhost/target-diagnostic' --test helper_corpus

conformance-helper:
	cargo +1.99.1 build --manifest-path wasmhost/Cargo.toml --release --locked \
		--features conformance --target-dir '$(CURDIR)/wasmhost/target-conformance'

test-conformance:
	@if test -z "$${GOMAD_WASM_CONFORMANCE_HELPER:-}"; then $(MAKE) conformance-helper; fi
	@helper="$${GOMAD_WASM_CONFORMANCE_HELPER:-}"; \
	if test -n "$$helper"; then \
		case "$$helper" in /*) ;; *) echo "GOMAD_WASM_CONFORMANCE_HELPER must be absolute" >&2; exit 2;; esac; \
		test -x "$$helper" || { echo "GOMAD_WASM_CONFORMANCE_HELPER is missing or not executable: $$helper" >&2; exit 2; }; \
	else \
		helper='$(CURDIR)/wasmhost/target-conformance/release/gomad-wasmhost'; \
	fi; \
	stock_go='$(GOMAD_WASM_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_WASM_CONFORMANCE_HELPER="$$helper" GOMAD_WASM_CONFORMANCE_REQUIRED=1 \
		PATH="$$(dirname "$$stock_go"):$$PATH" "$$stock_go" test -v -tags test_dep -count=1 -timeout=10m \
		-run '^(TestNamespaceConformance|TestVolumeConformance)' ./wasi

test-go:
	$(MAKE) helper
	$(MAKE) -C ../../.. test-wasm GOMAD_STOCK_GO='$(GOMAD_WASM_STOCK_GO)'

test-model:
	@stock_go='$(GOMAD_WASM_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		PATH="$$(dirname "$$stock_go"):$$PATH" "$$stock_go" test -tags test_dep -count=1 \
		-run 'TestCaptured|TestEnvironment|TestNamespace|TestExploratory|TestCapacity|TestPathOpen|TestOffered' ./wasi

qualify-runtime: helper
	@test -n "$(GOMAD_WASM_STOCK_GO)" && test -n "$$GOMAD_WASM_QUALIFICATION_ROOT" && test -n "$$GOMAD_WASM_QUALIFICATION_MANIFEST"
	@for index in $$(seq 0 31); do \
		env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
			GOMAD_STOCK_GO='$(GOMAD_WASM_STOCK_GO)' GOMAD_WASM_QUALIFICATION_INDEX="$$index" \
			'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 -timeout=10m \
			-run '^TestCooperativeRuntimeQualification$$' ./wasi || exit $$?; \
	done
	@env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO='$(GOMAD_WASM_STOCK_GO)' \
		'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 \
		-run '^TestCooperativeRuntimeQualificationAggregate$$' ./wasi

# Mandatory Phase 9 workload gate: each measured cell runs in a fresh process.
# Missing durable/Temporal cells fail before starting expensive preparation;
# the final aggregate rejects partial, provisional or budget-unmeasured reports.
qualify-workloads: helper
	@test -n "$(GOMAD_WASM_STOCK_GO)" && test -n "$$GOMAD_WASM_QUALIFICATION_ROOT" && \
		test -n "$$GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST" && test -n "$$GOMAD_WASM_INTEGRATED_SOURCE_SHA256" && \
		test -n "$$GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS" && test -n "$$GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE"
	@for name in TestActualMatchedFixtureQualification TestActualSQLiteDurableQualification TestActualOneboxQualification TestActualTemporalClusterQualification; do \
		env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
			'$(GOMAD_WASM_STOCK_GO)' test -tags test_dep -list "^$$name$$" ./qualification | rg -qx "$$name" || exit 1; \
	done
	@for entry in 'fixture controlled TestActualMatchedFixtureQualification' 'fixture cooperative TestActualMatchedFixtureQualification' \
		'sqlite cooperative TestActualSQLiteDurableQualification' 'onebox cooperative TestActualOneboxQualification' \
		'multi-instance cooperative TestActualTemporalClusterQualification'; do \
		set -- $$entry; profile=$$2; name=$$3; \
		for cell in success canary; do \
			env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
				GOMAD_STOCK_GO='$(GOMAD_WASM_STOCK_GO)' GOMAD_WASM_QUALIFY_WORKLOADS=1 \
				GOMAD_WASM_QUALIFICATION_PROFILE="$$profile" GOMAD_WASM_QUALIFICATION_CELL="$$cell" \
				'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 -timeout=90m -run "^$$name$$" ./qualification || exit $$?; \
		done; \
	done
	@env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_WASM_QUALIFY_WORKLOADS=1 GOMAD_WASM_QUALIFICATION_CELL=success \
		'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 -run '^TestActualMandatoryWorkloadCorpus$$' ./qualification

# The fn-151 determinism prototype gates. GOMAD_WASM_PROTOTYPE_ROOT names an
# absolute evidence directory; GOMAD_WASM_PROTOTYPE_ONLY narrows the Temporal
# gates (long-timers, network, smoke, smoke-<suite>).
prototype: helper
	@test -n "$(GOMAD_WASM_STOCK_GO)" && test -n "$$GOMAD_WASM_PROTOTYPE_ROOT"
	env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO='$(GOMAD_WASM_STOCK_GO)' \
		'$(GOMAD_WASM_STOCK_GO)' test -v -tags test_dep -count=1 -timeout=600m \
		-run '^TestPrototype' ./wasi
