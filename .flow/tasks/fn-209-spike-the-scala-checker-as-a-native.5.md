---
satisfies: [R3a]
---
# fn-209-spike-the-scala-checker-as-a-native.5 WebAssembly build of the checker loaded from a Go host

## Description
Build the checker and one sample with the Scala.js WebAssembly backend and ScalaPB's Scala.js artifact; load `model.wasm` from a Go host (wasmtime-go first; note wazero's GC-proposal status) and answer `rows(state)` for the sample with the JVM's verdicts; record module size, load time and call latency, or the blocking failure (host shim, GC support, rewriter). TeaVM is a fallback probe only.

**Files:** .flow/tmp/fn209/wasm/, model/design/native-checker-spike.md.

**Quick commands:** per the recipe recorded in the note

## Acceptance
- Note section with measurements or the blocker; a Go host program in .flow/tmp that loads the module.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
