---
satisfies: [R3a]
---
# fn-209-spike-the-scala-checker-as-a-native.5 WebAssembly build of the checker loaded from a Go host

## Description
**Owner 2026-10-18: the simulation itself is WebAssembly on wasmtime; this is the required target and runs first after .1.** Load the module through wasmtime-go with the GC and exception-handling proposals enabled, beside a second module in the same store; snapshot and restore the store and show identical answers after restore; measure per-call latency over 10^5 calls (R3b in addition to R3a). TeaVM remains a fallback probe only if the Scala.js backend's output cannot be made to load.

---
satisfies: [R3a]
---
# fn-209-spike-the-scala-checker-as-a-native.5 WebAssembly build of the checker loaded from a Go host

### Description
Build the checker and one sample with the Scala.js WebAssembly backend and ScalaPB's Scala.js artifact; load `model.wasm` from a Go host (wasmtime-go first; note wazero's GC-proposal status) and answer `rows(state)` for the sample with the JVM's verdicts; record module size, load time and call latency, or the blocking failure (host shim, GC support, rewriter). TeaVM is a fallback probe only.

**Files:** .flow/tmp/fn209/wasm/, model/design/native-checker-spike.md.

**Quick commands:** per the recipe recorded in the note

### Acceptance
- Note section with measurements or the blocker; a Go host program in .flow/tmp that loads the module.

### Done summary
TBD

### Evidence
- Commits:
- Tests:
- PRs:
## Acceptance
- Note section with measurements or the blocker; a Go host program in .flow/tmp that loads the module.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
