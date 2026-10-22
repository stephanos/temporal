---
satisfies: [R2]
---
# fn-233-kettle-service-a-framework-owned-end-to.2 Selected-start runtime and Kettle Judge/Answer live reference system

## Description
Create `model/umpire/reference/kettle/` with `proto/kettle/v1/kettle.proto` (`umpire.reference.kettle.v1.KettleService`: Fill, Heat, Pour, Describe, PollHeating, RespondHeating, ListKettles, ResetKettle) and the Go reference server (`server/`: gRPC on a loopback listener, `validators.go` with one named function per rule and distinctive messages, `limits.go` with a static and a per-kettle scoped limit, `mutants.go` with the enumerated mutant switch, no mutants wired yet). Add `realization.rs` (`Realize<kettle::World>` over the included `sample_kettle/model.rs`), the `export_guest!` guest crate, and `umpire/hosts/go/kettle_test.go`: start the server in-process, load the guest, run a scripted client and heater, route Describe through Judge and polls through Answer. No annotations yet.

**Size:** L
**Touches:** `model/umpire/reference/kettle/**`, `model/umpire/hosts/go/kettle_test.go`, `model/Cargo.toml`, `model/scripts/guest.sh` (second guest), `Makefile`

### Approach
The schema is awkward on purpose: enum, oneof, map, nested message, repeated, duration, bytes, a deprecated field, an idempotency key the client may omit, a cross-field rule. The server is written from the proto, never from `model.rs`. The transcript golden is the deliverable; the mutant switch exists but every mutant is a no-op until .4.

### Selected start and exact live projection

Implement the parent's additive Runtime::with_initial contract and transactional Reset; add `model/umpire/wasm/src/runtime.rs` and its existing runtime transaction tests to Touches (or the renamed runtime path if fn-228.1 has landed). The existing canonical Model is unchanged and Explore retains 81 starts. Pin invalid initial-state refusal, legacy multi-start behavior and publication rollback.

Implement the parent's explicit RPC mapping and healthy trace. Describe is current-only; Poll/Answer does not commit heat; only the RespondHeating receipt commits its exact action. Fill is initialization-only and Pour projection-neutral. Place server/generated Go API inside the generic host module's reference/kettle packages, with reproducible protobuf/gRPC generation and dependency freshness. Host tests use bounded contexts, dynamic loopback ports and cleanup. Missing guest fails an explicitly selected test. Quick: focused runtime and Go Kettle tests; final: make model-check with unchanged canonical Kettle goldens.

## Acceptance
- Proto, Go server with named validators and both limit kinds, realization, guest and host driver exist under `umpire/` (R2).
- `expected/transcript.snap.txt` golden from the Judge-only live run; `report.snap.txt` exploration golden unchanged from `sample_kettle` (R2).
- Server shares no code with the Model; reviewer checks the import graph (R2).
- Guest build goes through the existing pinned build step; `make model-check` and the Go host test target green.


## Done summary
Kettle now runs its complete healthy client/heater trace through the generic Go host, Judge and Answer, with exact receipt commits and a pinned transcript. Runtime::with_initial validates complete declared-start membership and transactional Reset preserves the selected start; Explore retains all 81 canonical starts.

The independent Go service lives at model/umpire/hosts/go/reference/kettle, with generated API at reference/kettle/v1. Its schema exercises nested messages, enum, oneof, map, repeated, duration, bytes, a deprecated field, and an idempotency key. Named validators enforce static ID and per-kettle tag limits. Mutant names exist as no-op options for task .4; the catalog and Annotate remain task .3.

baseline: green focused runtime (10 stateful tests) and configured Go host suite before edits. Initial wrong-cwd and missing-profile-env invocations were inconclusive/red harness invocations and were corrected before implementation.

Verification covers invalid selected starts, legacy zero/multiple starts, Reset after freeze, failed and panicking response publication, healthy Heat(false), 80 Heat receipts, Failover, neutral Pour, current-only Describe, Answer without commit, premature boil, malformed callbacks/receipts and server rejection without mutation. Focused runtime tests (13), native Kettle tests (6), Go live tests and server race tests pass. make model-check passes, including both Go host modules. Canonical Kettle Model and goldens are unchanged; the reference report matches the canonical report byte for byte.

The full gate took 426.52 seconds, above the R30 60-second soft target while rebuilding existing compile-failure fixtures and guest profiles. Added live tests took 0.525 seconds; the first Kettle guest build took 25.81 seconds and a warm rebuild 2.38 seconds. Measurements and verification metadata live in .flow/artifacts/fn-233-kettle-service-a-framework-owned-end-to/task2-verification.json.

Task-base root lint with GOLANGCI_LINT_FIX=false and strict changed-line submodule lint pass. The initial default root lint selected inherited changes and applied unrelated autofixes; all ten induced edits were restored after preserving their patch in .flow/tmp/fn-233/task2-unrelated-lint-autofixes.patch. No such edit shipped.

Review returned SHIP on correctness, contracts and integration, with zero findings. The reviewer ran gpt-6.1-sol at high in fresh read-only contexts; writer and reviewer are from the same family by explicit repository routing. Review evidence is .flow/artifacts/fn-233-kettle-service-a-framework-owned-end-to/task2-review.json and task2-review-<axis>.md.

GATE_SKIPPED:unittest:green-receipt b276db05 - verification reused prior post-gate pass

Tier: session (jev-unavailable(no_key)); repository implementer override gpt-6.1-sol at medium.
stage: impl-review - ran (model: gpt-6.1-sol at high; same-family; SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 683f85f0fb983a73a59d918c30de0968cbdc140a, b276db05e3205d743f88144f1eba7f0c772bc083
- Tests: cd model && mise exec -- cargo test -p umpire-wasm --test selected_start --test stateful, cd model && mise exec -- cargo test -p umpire-reference-kettle, cd model/umpire/hosts/go && mise exec -- go test -race -tags test_dep ./reference/kettle/..., cd model/umpire/hosts/go && mise exec -- go test -tags test_dep -count=1 -run ^TestKettle ., bash model/scripts/proto-kettle-go.sh --check, make model-check, make lint-code-fast GOLANGCI_LINT_BASE_REV=82ce151d4c8f634012a527f6001ca7f84cab04ed GOLANGCI_LINT_FIX=false, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/model/umpire/hosts/go && timeout 600 /Users/stephan/Workspace/skunkworks/umpire/temporal/.bin/golangci-lint-v2.13.1 run --build-tags test_dep --timeout 10m --config /Users/stephan/Workspace/skunkworks/umpire/temporal/.github/.golangci.yml --new-from-patch /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-233/task2-go.patch ./..., GATE_SKIPPED:unittest:green-receipt b276db05 - verification reused prior post-gate pass
- PRs: