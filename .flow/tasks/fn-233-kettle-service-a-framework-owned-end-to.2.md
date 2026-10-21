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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

