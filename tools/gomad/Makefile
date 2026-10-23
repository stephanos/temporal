.PHONY: generate validate validate-protocol validate-replay-plan validate-wasm-determinism build lint-code test-harness world-test test

.DEFAULT_GOAL := generate

.NOTPARALLEL: validate

GOMAD_TEST_TAGS := test_dep
GOMAD_STOCK_GO ?=

.PHONY: test-wasm test-wasm-slim-refusals wasm-test-helper
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

test-wasm: test-wasm-slim-refusals validate-model-corpora test-rust-engine
	cargo +1.99.1 test --manifest-path rust/Cargo.toml --release --locked -p gomad

# The standalone protocol transport is available only to retained oracle fixtures.
wasm-test-helper:
	cargo +1.99.1 build --manifest-path rust/Cargo.toml --release --locked -p gomad-wasmhost --features reference-helper
	@target_root="$${CARGO_TARGET_DIR:-$(CURDIR)/rust/target}"; \
	sh rust/publish-executable.sh "$$target_root/release/gomad-wasmhost" "$(CURDIR)/rust/target/release/gomad-wasmhost"

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
	diff -ru rust/wasmhost/tests/testdata/replay_plan "$$corpus_tmp/first" && \
		diff -ru "$$corpus_tmp/first" "$$corpus_tmp/second"
	cargo +1.99.1 test --manifest-path rust/Cargo.toml -p gomad-wasmhost --release --locked --lib replay_plan

.PHONY: validate-model-corpora
validate-model-corpora:
	cargo +1.99.1 build --manifest-path rust/Cargo.toml -p gomad-wasmhost --release --locked --features conformance
	@target_root="$${CARGO_TARGET_DIR:-$(CURDIR)/rust/target}"; \
	cp "$$target_root/release/gomad-wasmhost" "$$target_root/release/gomad-model-conformance"
	$(MAKE) wasm-test-helper
	@target_root="$${CARGO_TARGET_DIR:-$(CURDIR)/rust/target}"; \
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off \
		GOMAD_WASM_TEST_HELPER="$$target_root/release/gomad-wasmhost" \
		GOMAD_WASM_CONFORMANCE_HELPER="$$target_root/release/gomad-model-conformance" GOMAD_WASM_CONFORMANCE_REQUIRED=1 \
		go test -tags $(GOMAD_TEST_TAGS) -count=2 \
		./internal/model/... ./internal/backends/wasm/cluster \
		./internal/backends/wasm/toolchain/... ./internal/support/...
	@for pass in first second; do \
		cargo +1.99.1 test --manifest-path rust/Cargo.toml -p gomad-wasmhost --release --locked --lib || exit $$?; \
	done

.PHONY: lint-code
lint-code:
	$(MAKE) -C ../.. lint-code-gomad

build:
	cargo +1.99.1 build --manifest-path rust/Cargo.toml --release --locked -p gomad
	@target_root="$${CARGO_TARGET_DIR:-$(CURDIR)/rust/target}"; \
	sh rust/publish-executable.sh "$$target_root/release/gomad" "$(CURDIR)/rust/target/release/gomad"

world-test:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go test -race -tags test_dep ./internal/model/world/...

.PHONY: validate-wasm-determinism
validate-wasm-determinism:
	cargo +1.99.1 test --manifest-path rust/Cargo.toml --release --locked -p gomad-wasmhost --test inventory --test determinism_observed --test diagnostic_roles
	go test -tags $(GOMAD_TEST_TAGS) -count=1 ./internal/archcheck

test-harness:
	env -u GOMADSEED -u GOMAD_CHILD_SEED GOWORK=off go test -tags $(GOMAD_TEST_TAGS) ./internal/archcheck ./internal/support/...
	cargo +1.99.1 test --manifest-path rust/Cargo.toml --release --locked -p gomad

test: test-harness world-test test-wasm

.PHONY: fmt-rust lint-rust qualify
fmt-rust:
	cargo +1.99.1 fmt --manifest-path rust/Cargo.toml --all -- --check

lint-rust: fmt-rust
	@for features in '' determinism-fixture helper-stress-jitter helper-determinism-fixture determinism-fixture,helper-stress-jitter determinism-fixture,helper-determinism-fixture helper-diagnostic-profile helper-guest-profile helper-diagnostic-fuel-off determinism-fixture,helper-diagnostic-profile; do \
		cargo +1.99.1 clippy --manifest-path rust/Cargo.toml --release --locked -p gomad --all-targets --no-deps \
			--features "$$features" -- -D warnings || exit $$?; \
	done
	@for features in reference-helper conformance reference-helper,diagnostic-profile reference-helper,diagnostic-fuel-off reference-helper,guest-profile reference-helper,guest-profile,diagnostic-fuel-off reference-helper,stress-jitter; do \
		cargo +1.99.1 clippy --manifest-path rust/Cargo.toml --release --locked -p gomad-wasmhost --all-targets --no-deps \
			--features "$$features" -- -D warnings || exit $$?; \
	done

qualify:
	@test -n "$(GOMAD_QUALIFICATION_CONFIG)" && test -n "$(GOMAD_QUALIFICATION_CONFIG_SHA256)"
	python3 rust/driver/driver.py serial --config '$(GOMAD_QUALIFICATION_CONFIG)' --config-sha256 '$(GOMAD_QUALIFICATION_CONFIG_SHA256)'

.PHONY: test-rust-engine
test-rust-engine:
	cargo +1.99.1 test --manifest-path rust/Cargo.toml --release --locked -p gomad-wasmhost --features reference-helper
	@for features in diagnostic-profile diagnostic-fuel-off guest-profile; do \
		cargo +1.99.1 test --manifest-path rust/Cargo.toml --release --locked -p gomad-wasmhost \
			--features "reference-helper,$$features" --test helper_corpus || exit $$?; \
	done

.PHONY: parity
parity:
	@test -n "$(GOMAD_PARITY_BINDINGS)" && test -n "$(GOMAD_PARITY_BINDINGS_SHA256)" && test -n "$(GOMAD_PARITY_WORK)"
	python3 scripts/all-verb-parity --bindings '$(GOMAD_PARITY_BINDINGS)' --bindings-sha256 '$(GOMAD_PARITY_BINDINGS_SHA256)' --work '$(GOMAD_PARITY_WORK)'
