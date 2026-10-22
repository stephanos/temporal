.PHONY: generate validate validate-protocol validate-replay-plan validate-wasm-determinism build lint-code test-harness world-test test

.DEFAULT_GOAL := generate

.NOTPARALLEL: validate

GOMAD_TEST_TAGS := test_dep
GOMAD_STOCK_GO ?=

.PHONY: test-wasm test-wasm-slim-refusals wasm-helper wasm-test-helper
# Excluded providers in the gomad_wasm_slim guest selection must refuse clearly
# without falling back (fn-182 R2), and ringpop membership must stay linked
# (fn-224). Every named test must pass; a rename must not silently run nothing.
WASM_SLIM_REFUSAL_TESTS := TestCloudArchiversRefuseConfiguredSchemesWithoutFallback TestDefaultMembershipLinksRingpop TestSlimGuestRefusesUnlinkedSQLDriversWithoutFallback
test-wasm-slim-refusals:
	@stock_go='$(GOMAD_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	log="$$(mktemp)" || exit 1; \
	trap 'rm -f "$$log"' EXIT; \
	pattern="$$(printf '^%s$$|' $(WASM_SLIM_REFUSAL_TESTS))"; \
	(cd ../.. && env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -v -tags gomad_wasm_slim,test_dep -count=1 -timeout=15m \
		-run "$${pattern%|}" \
		./common/archiver/provider ./temporal ./tests/testcore) > "$$log" 2>&1; \
	status=$$?; cat "$$log"; test "$$status" -eq 0 || exit "$$status"; \
	for name in $(WASM_SLIM_REFUSAL_TESTS); do \
		grep -q -- "^--- PASS: $$name " "$$log" || { echo "slim refusal test did not pass: $$name" >&2; exit 1; }; \
	done

test-wasm: test-wasm-slim-refusals
	@$(MAKE) -C internal/backends/wasm test-conformance GOMAD_WASM_STOCK_GO='$(GOMAD_STOCK_GO)'
	@stock_go='$(GOMAD_STOCK_GO)'; \
	if test -z "$$stock_go"; then \
		stock_root="$$(env -u GOROOT GOENV=off GOTOOLCHAIN=go1.27.1 go env GOROOT)" || exit 1; \
		stock_go="$$stock_root/bin/go"; \
	fi; \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -tags test_dep -count=1 -timeout=15m \
		-run '^TestWASMTestsDoNotImportNativeBuilder$$' \
		./internal/backends/wasm && \
	env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off \
		GOMAD_STOCK_GO="$$stock_go" PATH="$$(dirname "$$stock_go"):$$PATH" \
		"$$stock_go" test -v -tags test_dep -count=1 -timeout=15m -skip '^TestNamespaceConformance' ./internal/backends/wasm/...

wasm-helper:
	$(MAKE) -C internal/backends/wasm helper

wasm-test-helper:
	$(MAKE) -C internal/backends/wasm test-helper

# The shared wire codecs are written by protocolgen with stock Go and no tags.
generate:
	go run ./internal/model/cmd/protocolgen

validate: validate-protocol validate-replay-plan validate-model-corpora validate-wasm-determinism

validate-protocol:
	go run ./internal/model/cmd/protocolgen -check
	go test -tags $(GOMAD_TEST_TAGS) -count=1 ./internal/model/generation/protocol

.PHONY: validate-replay-plan
validate-replay-plan:
	@corpus_tmp="$$(mktemp -d)" || exit 1; \
	trap 'rm -rf -- "$${corpus_tmp:?}"' EXIT; \
	for pass in first second; do \
		GOMAD_REPLAY_PLAN_CORPUS_REGENERATE=1 GOMAD_REPLAY_PLAN_CORPUS_DIR="$$corpus_tmp/$$pass" \
			go test -tags $(GOMAD_TEST_TAGS) -count=1 -run '^TestReplayPlanConformance$$' ./internal/model/choice || exit $$?; \
	done; \
	diff -ru internal/backends/wasm/wasmhost/tests/testdata/replay_plan "$$corpus_tmp/first" && \
		diff -ru "$$corpus_tmp/first" "$$corpus_tmp/second"
	cargo +1.99.1 test --manifest-path internal/backends/wasm/wasmhost/Cargo.toml --release --locked --lib replay_plan

.PHONY: validate-model-corpora
validate-model-corpora:
	cargo +1.99.1 build --manifest-path internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features conformance
	@target_root="$${CARGO_TARGET_DIR:-$(CURDIR)/internal/backends/wasm/wasmhost/target}"; \
	cp "$$target_root/release/gomad-wasmhost" "$$target_root/release/gomad-model-conformance"
	$(MAKE) wasm-helper
	@target_root="$${CARGO_TARGET_DIR:-$(CURDIR)/internal/backends/wasm/wasmhost/target}"; \
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off \
		GOMAD_WASM_TEST_HELPER='$(CURDIR)/internal/backends/wasm/wasmhost/target/release/gomad-wasmhost' \
		GOMAD_WASM_CONFORMANCE_HELPER="$$target_root/release/gomad-model-conformance" GOMAD_WASM_CONFORMANCE_REQUIRED=1 \
		go test -tags $(GOMAD_TEST_TAGS) -count=2 \
		-run '^(TestNetworkCorpus|TestClockConformance|TestWorldConformance|TestNamespace.*Conformance|TestVolume|TestHelperCorpus$$)' \
		./internal/model/network ./internal/backends/wasm/cluster ./internal/backends/wasm/wasi
	@for pass in first second; do \
		cargo +1.99.1 test --manifest-path internal/backends/wasm/wasmhost/Cargo.toml --release --locked --lib || exit $$?; \
	done

.PHONY: lint-code
lint-code:
	$(MAKE) -C ../.. lint-code-gomad

build:
	go build ./...

world-test:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go test -race -tags test_dep ./internal/model/world/...

.PHONY: validate-wasm-determinism
validate-wasm-determinism:
	$(MAKE) -C internal/backends/wasm test-determinism GOMAD_WASM_STOCK_GO='$(GOMAD_STOCK_GO)'

test-harness:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go test -tags $(GOMAD_TEST_TAGS) \
		./cmd/gomad/... ./internal/host/... ./internal/runner/... ./internal/evidence/... ./internal/backends/target/... ./internal/archcheck

test: test-harness world-test test-wasm
