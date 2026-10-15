---
satisfies: [R2, R3, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.3 Go probe library: registered dumps and dormant probes

## Description
Implement the public standard-library-only server package `common/gomaddebug`, with active `gomad && wasip1 && wasm` and complementary no-op variants for ordinary builds and native Gomad. Do not add a nested-module root library directory or import a nested module's internal package; the Gomad host does not import the server helper. Provide RegisterDump, DeclareProbe and invariant registration (including retained monotonic declarations) through a narrow inventoried guest-runtime init link seam. Runtime never imports runtime/pprof or the server package.

Create identical dormant registry, fixed emit buffer, bitmask and debug goroutine in every recording/replay incarnation before application initialization. Retain closed bounded registry descriptors and stable registration-order IDs while disabled, including deterministic duplicate/late/capacity failures and incomplete-registry refusal after early failure. Scope enablement to the chosen target and coordinate it with .1's preflight and .2's runtime handoff.

The disabled call site tests Enabled before computing fields, costing one load/branch. Enabled typed emit accepts only primitive fields/existing strings and copies directly to reserved storage: no fmt, interfaces, boxing, variadic formatting, concatenation or unguarded pre-call allocation. Dedicated imports never yield or alter clock folding/evidence; host formats records with already-held choice/time/instance/incarnation coordinates. Buffer exhaustion/oversized records drop with counted markers. Panicking/oversized dumps are isolated; nonyielding timeout ends inspection. Probes consume fuel; report recorded-limit pressure separately. A deliberately allocating test probe must diverge through the compulsory witness even without GC, changed output or optional diagnostics.
## Acceptance
- [ ] Registered dumps selectable by target-scoped name; panic/oversize isolation tested
- [ ] Probes enabled by pattern produce decoded records with choice position, virtual time and cluster instance/incarnation; disabled cost one load+branch; enabled path allocation-free
- [ ] Same probe names/IDs on different nodes or incarnations remain distinguishable and enablement does not leak between them
- [ ] Buffer exhaustion marker, fuel-limit report and allocating-probe divergence tests
- [ ] Public helper compiles across active WASM, ordinary server and native no-op source sets without host-to-server or reverse/internal module imports
- [ ] Identical dormant initialization and retained registry metadata available while disabled; duplicate/late/overflow and early incomplete registration are deterministic refusals
- [ ] Call-site guard prevents formatting/evaluation allocations while disabled; enabled primitive/string emit and overflow path allocate zero Go heap bytes
- [ ] Allocating-probe fixture is detected with diagnostics off and without GC/output changes by the compulsory runtime-state witness
- [ ] Registered callbacks run on the debug goroutine with frozen application goroutines, isolate panic/oversize and never resume simulation after inspection
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
