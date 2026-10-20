---
satisfies: [R2, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.2 Rust evidence layer: record schema, canonical encoding, identities

Touches: tools/gomad/rust/gomad/src/evidence/{record.rs,identity.rs,canonical.rs}

## Description
Define the new versioned record envelope (`gomad.execution-record/v2`) with serde structs, canonical encoding via serde_json with sorted keys and native integers, record hash over canonical bytes, failure signature excluding the seed, execution identity binding target, toolchain, platform and implementation digests. Exclude diagnostic paths and host timestamps. Property test: two executions of the same fixture target, seed and profile produce byte-equal records and hashes.

## Acceptance
Use the foundation's fixture-only driver for the execution/store behavior below; no later public verb is a prerequisite. Actual public-verb goldens land with their owning tasks.
Golden corpus: two executions of each fixture target, seed and profile produce byte-equal records and the pinned record hash and failure signature in the goldens; a fault fixture that injects a host timestamp or path into a record field fails the repeated-execution check; identities are stable across hosts (digests pinned in goldens). No separate unit suite for codecs beyond one round-trip test that names its invariant.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
