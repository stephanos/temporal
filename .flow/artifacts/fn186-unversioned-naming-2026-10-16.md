# Unversioned Gomad naming — fn-186.1

Implementation commit: `3932262536aa7b698b05baac45b53e7e6404f9d6`.
Base: `a5285735ef` (planning checkpoint; implementation source baseline
`0609c11d80`). The owner confirmed a clean rename of all current contracts,
without aliases. Historical artifacts keep their original identities and require
their matching historical build. No push, PR, CI dispatch, native revival or
external settings change occurred.

## Requirements and changed contracts

| Requirement | Current result and evidence |
| --- | --- |
| R1 | Maintained `gomad3`/`GOMAD3` names become `gomad`/`GOMAD`: Make targets, environment variables, integration/toolchain build tags, schema/profile/hash domains, Rust package/helper executable, generated owners, workflows and current documentation. Workflow filenames are `.github/workflows/gomad.yml` and `gomad-smoke.yml`. Independent `/vN` schema versions stay unchanged. |
| R2 | Rename-policy tests use `tools/gomad-legacy` as the distinct old directory. Canonical environment records put `GOMADSEED` before `GOMAD_*`; seed mutation indices and exact environment assertions move with the ordering. Production WASM cluster qualification uses that same ordering. Exact hash and byte assertions remain. The World binary-envelope test now checks `json.Valid` rather than treating a `{` byte inside a binary digest as JSON. |
| R3 | Existing owners regenerate native version/protocol/boundary/qualification outputs, volume/network projections, compatibility requests/packs/reports, namespace/volume/network/clock/World/broker corpora and current Runner characterization fixtures. Rust encoders regenerate all fourteen binary network fixture streams structurally. Both platform adapter/profile pins come from existing owner methods, not old-name aliases. |
| R4 | Git history, archived plans/research/designs, prior Flow evidence/source snapshots and authenticated historical fixtures remain unchanged. Five open Flow specs receive only current operational command/path repairs. AGENTS.md, MILESTONES.md and current README contracts use the unversioned name. Concurrent untracked fn-187 planning files remain outside this task. |
| R5 | Source/model/artifact tests, source compilation, existing generated checks, portable native transcript checks, lint policy tests, architecture checks and 205 Rust conformance/model tests pass. Full current helper execution remains unqualified: the ordinary helper has the same two compile errors as the baseline. Supported native execution remains deferred. See the exact limitations below. |
| R6 | Commands, numeric exits, retained logs, owner harnesses and baseline comparisons are recorded here and in `fn186-unversioned-naming/`. Independent review belongs to the conductor after the implementation commit; this worker claims no review verdict. |

The initial inventory contained 845 maintained files with a case-insensitive old
namespace reference. Regeneration additionally updates dependent current hashes
and encoded fixtures. No public migration command or fallback namespace was added.

## Verification receipts

Unless a row says otherwise, Go commands run from `tools/gomad` with
`GOTOOLCHAIN=local GOWORK=off` and `/tmp/fn182-stock-go/bin/go` (Go 1.27.1,
linux/arm64). Logs are retained in
[`fn186-unversioned-naming/logs`](fn186-unversioned-naming/logs/).
There were no Quick commands in the task/spec: `baseline: none` for that formal
list. Additional pre-change checks were intentionally retained as failing
baselines; they are not green receipts.

Retained `.log` copies normalize indentation tabs, trailing whitespace and blank
lines at EOF; their output text and results are otherwise unchanged. Original
command output remains at the corresponding `/tmp/fn186-*.log` paths.

| Exact command / scope | Exit | Retained log |
| --- | ---: | --- |
| `env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off PATH=/tmp/fn182-stock-go/bin:$PATH make generate` | 0 | Initial final generation: `/tmp/fn186-generate-final.log`; checked again by validation below. |
| `GOTOOLCHAIN=local GOWORK=off PATH=/tmp/fn182-stock-go/bin:$PATH make validate` | 0 | `fn186-validate-final.log` |
| `go test -p=1 -tags test_dep -count=1 -timeout=10m ./internal/runner/internal/{campaign,corpus,exploration,minimizer,logstore} ./internal/runner/inspect ./internal/evidence/... ./internal/model/... ./internal/backends/native/compatibilitypack/...` with `GOMAD_STOCK_GO=/tmp/fn182-stock-go/bin/go` | 0 | `fn186-source-final.log`: 28 tested packages, plus packages with no tests. |
| `go test -tags test_dep -count=1 -timeout=10m ./internal/backends/wasm/wasi` with `GOMAD_STOCK_GO=/tmp/fn182-stock-go/bin/go` | 0 | `fn186-wasi-final2.log`; helper execution tests skip when the renamed release helper is absent. This is source/oracle coverage, not current-helper execution qualification. |
| `go test -tags test_dep -count=1 -timeout=10m -run '^Test(Portable\|DeterministicProfileCompatibilityGolden\|Semantic\|DecodeSemantic\|SummarizeSemantic)' ./internal/backends/native/transcript` | 0 | `fn186-transcript-portable-final.log` |
| `go test -tags test_dep -count=1 -timeout=10m ./internal/archcheck` | 0 | `fn186-archcheck-final.log` |
| Repository root: `go test -tags test_dep -count=1 ./cmd/tools/lintcode` | 0 | `fn186-lint-policy.log` |
| Repository root: `env PATH=/tmp/fn182-stock-go/bin:$PATH GOMAXPROCS=2 GOFLAGS=-p=1 GOTOOLCHAIN=local GOWORK=off make GOLANGCI_LINT=/tmp/fn186-host-tools/golangci-lint ERRORTYPE=/tmp/fn186-host-tools/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=a5285735ef lint-code-fast` | 0 | `fn186-root-static-committed.log` on committed source `3932262536`; root package, ordinary nested-module source set and `gomad_integration` source set; linter and errortype. Earlier stable pass: `fn186-root-static-post-fixtures.log`. |
| `GOTOOLCHAIN=local GOWORK=off xargs /tmp/fn182-stock-go/bin/go test -p=1 -tags test_dep -exec /bin/true -run '^$' < /tmp/fn186-host-packages.txt` | 0 | `fn186-source-compile-final.log`; compilation only of the 80-package inventory, excluding runtime overlay sources. `/bin/true` deliberately does not execute test binaries or claim tests ran. |
| Same compile-only command with `-tags test_dep,gomad_integration` | 0 | `fn186-source-integration-compile-final.log` |
| `go build -p=1 -o /tmp/fn186-gomad ./cmd/gomad` | 0 | `fn186-cli-build.log`; stock source build, not native runtime qualification. |
| Repository root: `CARGO_TARGET_DIR=/tmp/gomad-fn178-batch-target-20261015 CARGO_BUILD_JOBS=1 cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features conformance --bin gomad-wasmhost --test network_corpus --test world_conformance --test clock_conformance` | 0 | `fn186-rust-corpus-final3.log`: fresh renamed crate compilation, 132 root tests, 5 clock, 62 network and 6 World tests, 205 total. |

The host's existing `.bin` golangci-lint and errortype executables had an
incompatible executable format. The conductor rebuilt the exact pinned versions
(`golangci-lint v2.13.0`, `errortype v0.0.7`) into `/tmp/fn186-host-tools` and
overrode their paths with fixing disabled. The failed discovery attempts and
successful builds remain in `fn186-root-lint-fast.log`,
`fn186-root-lint-fast-host.log`, `fn186-root-lint-build.log` and
`fn186-root-errortype-build.log`. No gate was weakened.

## Regeneration provenance

- `go generate ./internal/model/volume ./internal/model/network` uses the native
  semantic source owners. `make validate` checks the generated results and
  ownership inventories.
- `GOMAD_WORLD_CONFORMANCE_REGENERATE=1 GOMAD_CLOCK_CONFORMANCE_REGENERATE=1 go test -tags test_dep -count=1 -run '^Test(World|Clock)Conformance' ./internal/backends/wasm/cluster`
  regenerated and checked the World and clock corpora, exit 0
  (`fn186-clock-world-regen.log`). An earlier combined exact-name selection ran
  zero cluster tests; its `fn186-corpora-regen.log` observation is not counted as
  World/clock regeneration.
- The exact combined network/broker regeneration was
  `env GOMAD_WORLD_CONFORMANCE_REGENERATE=1 GOMAD_CLOCK_CONFORMANCE_REGENERATE=1 GOMAD_NETWORK_CORPUS_REGENERATE=1 GOMAD_BROKER_ORDERS_REGENERATE=1 GOTOOLCHAIN=local GOWORK=off /tmp/fn182-stock-go/bin/go test -tags test_dep -count=1 -run '^(TestWorldConformance|TestClockConformance|TestNetworkCorpusIsCurrent|TestBrokerOrdersAreCurrent)$' ./internal/backends/wasm/cluster ./internal/model/network ./internal/backends/wasm/wasi`,
  exit 0 (`fn186-corpora-regen.log`); only the network/broker tests ran there.
- `go test -tags test_dep -count=1 -run '^TestNamespaceConformanceCorpus$' ./internal/backends/wasm/wasi -args -update-conformance`
  regenerated namespace fixtures, exit 0 (`fn186-namespace-regen.log`).
- `go test -tags test_dep -count=1 -run '^TestVolume(ConformanceCorpus|CoreVectors)$' ./internal/backends/wasm/wasi -args -volume-conformance-output=/tmp/fn186-volume-regen`
  generated volume fixtures and core vectors into an external directory for
  copying into their current owner, exit 0 (`fn186-volume-regen2.log`).
- Rust's existing `cluster_network_fixture_is_current` test ran with
  `GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE=1`, `--features conformance --test network_corpus`,
  exit 0 (`fn186-network-rust-regen.log`). The ordinary broker owner cannot build
  past the inherited helper errors. Instead, a temporary integration harness
  calls the current Rust `operation_entry`, `quiescence_entry` and `document`
  encoders on newly generated Go documents, checking every operation against
  its JSON and using the Go World encoder for expanded wire bytes. It produces
  twelve binary streams without shortening or splicing old binary payloads,
  exit 0 (`fn186-broker-rust-regen.log`). Current Go decoder tests and Rust corpus
  tests then validate the results.
- [`adapter-profile-owner-harness.go.txt`](fn186-unversioned-naming/adapter-profile-owner-harness.go.txt)
  and [`adapter-profile-pins.json`](fn186-unversioned-naming/adapter-profile-pins.json)
  retain owner-derived pins for all 15 adapters and both supported native
  platforms. Existing per-file dependency bytes and libc templates did not
  change; source-inventory/prepared-source hash domains did. The harness verifies
  rewritten-module preparation with the derived pins. Its run was
  `GOMAD_RENAME_PIN_AUDIT_OUTPUT=/tmp/gomad-rename-namespace-pin-audit-1.json go test -tags test_dep -run '^TestRenameNamespacePinAudit$' -count=1 -timeout=10m ./internal/backends/native/transcript`,
  exit 0. Request source-set digests and previously approved decision digests
  are then rebound through `compatibility.DigestSources`,
  `authoring.ApprovalSHA256` and `authoring.Regenerate`; previously empty
  approvals stay empty. No new native execution approval is inferred.
- Additional temporary source-bound harnesses are retained as `.go.txt` and
  `.rs.txt` files beside that mapping, outside production compilation. They
  capture pack rebinding, broker World wire/stream encoding, entropy vectors
  and World ranks. Temporary live sources and directories were removed.
- Current Runner characterization and diagnostic-identity fixtures are rebound
  through their existing observation/finalization owners. Portable fixture
  validation, exact expected hashes and vet pass (`fn186-runner-fixtures-focused.log`,
  `fn186-runner-fixtures-vet.log`). The Darwin-only diagnostic producer test
  skipped on Linux/arm64; these historically derived fixtures retain their
  older campaign-plan/journal lineage and do not establish current Darwin
  native qualification.

## Baseline failures and limits

The first pre-edit Runner command was
`env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAD3_STOCK_GO=/tmp/fn182-stock-go/bin/go /tmp/fn182-stock-go/bin/go test -tags test_dep -count=1 -timeout=10m ./internal/runner/...`,
exit 1 (`fn186-go-baseline.log`). It includes absent native
`.toolchain/bin/go`, the two existing replay compatibility-pack fixture failures,
and unfinished scenario transport behavior. The first attempt mistakenly used
the nonexistent `/tmp/fn182-stock-go/bin/go1.27.1`, exit 127; it was corrected
before implementation and is not a source baseline.

A sparse, detached baseline worktree at `a5285735ef` supplied independent numeric
comparisons after the rename. It was cleanly removed afterward:

| Baseline/current command | Baseline | Candidate | Meaning |
| --- | ---: | ---: | --- |
| `CARGO_TARGET_DIR=/tmp/gomad-fn178-batch-target-20261015 CARGO_BUILD_JOBS=1 cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --no-run` | 101 | 101 | Same missing `Namespace.stdout()` and `.stderr()` methods at lines 1478/1479. `fn186-rust-baseline-verified.log` and `fn186-rust-current-build.log`. |
| `go test -p=1 -tags test_dep -count=1 -run '^TestCharacterizeHelpAndUsage$' ./cmd/gomad/internal/cli` | 1 | 1 in broad candidate run | Existing usage/replay snapshots lag unfinished work. `fn186-cli-help-baseline.log`; no unrelated snapshot update. |
| `go test -tags test_dep -count=1 -run '^TestSpecCloneDetachesScenarioDeclaration$' ./internal/backends/target` | 1 | 1 in broad candidate run | Existing scenario slice aliasing; `fn186-target-baseline.log`. No unrelated fix. |

Broad exploratory attempts additionally reached unsupported native
Linux/arm64 tests, missing current-helper tests and native overlay packages that
stock Go cannot compile as standalone packages. Those are recorded as unavailable
or an incorrect suite surface, not as green gates. An early Runner/CLI build
overlapped gofmt and reported inconsistent source lengths; that attempt was
inconclusive. Subsequent stable source compilation, shared tests, architecture
checks and final lint passed. A prior architecture check saw temporary generator
directories and exceeded its five-second checker budget under load; after
removing the scratch directories, its complete unchanged gate passed.

WASM operator/runtime/replay, native runtime, performance and native platform
qualification are not claimed. The ordinary helper's inherited compile failure
prevents current executable qualification. Deferred fn-128/fn-149 and unfinished
fn-180 through fn-185 remain with their owners.

## Residual old namespace inventory

`git grep -l -i gomad3 -- ':!.flow' ':!.turbo' ':!.plans' ':!docs/research' ':!docs/superpowers'`
returns exactly these immutable historical fixture manifests (exit 0, retained
`fn186-residuals.txt`):

- `tools/gomad/internal/backends/wasm/backend/testdata/premove-stock-artifact/manifest.json`
- `tools/gomad/internal/backends/wasm/backend/testdata/premove-stock-artifact/backend/provenance.json`

The authenticated historical SQLite `recorded-storage-sqlite/sample.json` also
retains old identifiers inside base64 bytes. Its derived current fixture first
asserts that the historical profile is refused, then explicitly selects the
current cooperative profile; stored sample bytes and hashes are untouched.
`git diff --numstat` for both historical fixture roots is empty. Historical Flow,
archived `.plans`/`.turbo`, dated research and design material remain under their
original identities. Current supported readers have no old-name aliases.

Tier: session judge unavailable(no_key).
stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)
