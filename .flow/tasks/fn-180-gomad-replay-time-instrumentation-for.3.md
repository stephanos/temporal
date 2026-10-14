---
satisfies: [R2, R3, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.3 Go probe library: registered dumps and dormant probes

## Description
Importable Go package (no-op outside Gomad and on native): RegisterDump(name, fn), DeclareProbe(name) with stable registration-order IDs, enabled check as one load+branch on a host-set bitmask, allocation-free encoding of primitives/string bytes into a buffer reserved at init in every run, flushed through the debug import with choice index and virtual time. Buffer exhaustion drops records with a counted marker. Panicking/oversized dumps reported per dump while others run. Enabled probes consume fuel; approaching the recorded fuel limit is reported. A deliberately allocating test probe is reported as instrumentation divergence.

## Acceptance
- [ ] Registered dumps selectable by name; panic/oversize isolation tested
- [ ] Probes enabled by pattern produce decoded records with choice index and virtual time; disabled cost is one load+branch; enabled path allocation-free
- [ ] Buffer exhaustion marker, fuel-limit report and allocating-probe divergence tests

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
