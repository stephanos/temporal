VERDICT: SHIP

1. **P2 — `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:837`:** Finalization marks diagnostics complete even when choice-trace overflow truncated them. With space for two choice records and 100 diagnostic records, choice overflow suppresses subsequent diagnostic appends while execution continues; the diagnostic header nevertheless becomes `Complete`. Propagate choice overflow into the diagnostic state and add an unequal-capacity regression test. The paired choice terminal still reports overflow, so the existing Runner fails closed.

Acceptance criteria not verified:

- Cross-build behavioral equivalence and correct recording of new identities; retained claims were not independently reproduced.
- Actual Runner qualification with diagnostics enabled; the worker reports direct-launch comparisons only.
- Executed proof of allocation/draw noninterference and passing Darwin gates; no tests were run in this sandbox. Native Linux validation was explicitly not performed.