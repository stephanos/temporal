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
The evidence layer meets its fixture-only R2/R14 acceptance on integrated source d045958da2 with audited goldens integrated at 4623917d3b. Each of six targets (exit, trap, fuel, watchdog, memory and modeled calls) executes twice for seeds 7/8 with fresh engines/stores. Canonical records, record hashes, seed-excluded failure signatures, execution identities and authenticated raw evidence agree. Injected host timestamps and paths fail the repeat check.

Two independent regenerations and the following strict run passed. Root reran the full foundation gate on the same frozen fixture binary against current tracked goldens; exit 0, with all 15 manifest hashes verified. Raw receipts remain under /home/agent/wt/evidence/fn232-foundation-closeout and workspace .build-cache/fn232-foundation-closeout. The typed golden audit changes only implementation identity and its derived execution identity/hash links.

Fresh independent foundation_acceptance_review returned SHIP for this task's bounded R2/R14 scope. The reviewer is a GPT-6 Codex session, the same family as the writer; exact variant was not exposed. Its separate R3 durability finding belongs to .3 and remains open. This task explicitly requires no later public verb. No fn232.1 compiler-deadline, public-verb parity or final qualification pass is claimed.

stage: impl-review - ran (fresh independent foundation acceptance review, SHIP for .2)
stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 4623917d3b, d045958da2
- Tests: python3 tools/gomad/testdata/parity/foundation-check.py --binary /Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-foundation-closeout/candidate/gomad-fixture --work /Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-foundation-closeout/root-closure-strict (exit 0), all 15 tracked foundation golden SHA256 values checked against retained golden-manifest.json (exit 0), retained foundation-regeneration1, foundation-regeneration2 and foundation-strict-after receipts each exit 0; independent review SHIP scoped R2/R14
- PRs: