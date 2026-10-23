---
satisfies: [R2]
---
# fn-211-apalache-backend-for-umpire-in-rust.2 umpire-apalache: pinned release, bounded check per invariant, deterministic lifecycle

## Description
Finalize the existing umpire-apalache foundation from task .1 under the parent R2 bound/agreement and runner-lifecycle contracts. The crate, official release 0.62.4 pins, Makefile integration, actual safety controls and reused TLC process helper already exist; evolve those owners rather than create parallel tool or subprocess frameworks.

**Size:** L. **Dependencies:** task .1 must be done and its closure committed on umpire before admission. **Touches:** [model/umpire/backends/apalache/**, model/umpire/backends/tlc/**, model/umpire/testing/**, model/scripts/fetch-apalache.py, model/scripts/fetch-tlc.sh, model/scripts/tests/**, model/Cargo.toml, model/Cargo.lock, Makefile]
No Model authoring expansion, canonical production Model change, protocol/Go runtime change, third-party libraries beyond those already admitted to the workspace or publication. Task .3 owns general typed ITF conversion and native replay, .4 owns the fixture consumer/goldens, and .5 owns symbolic Worlds. Preserve the reviewed safety projection, arithmetic/history controls and existing TLC answers.

### Approach

1. Reanchor actual .1 source and final receipts before edits. Baseline focused native/fake-tool and existing real controls, retain artifact/source/guest hashes. Reuse Apalache::discover/check, fetch-apalache.py, and umpire_tlc::process::{Scratch,run_process}; inspect their actual current contracts instead of reimplementing them.
2. Finalize the parent bounded-result API with explicit NoViolation, Violation and NotRun distinctions and retained bound/raw typed-wire witness evidence. Admission failures and tool failures must never become a false safety pass. General ITF-to-Model decoding remains .3. Validate declared properties and length conversion/overflow, preserve transition-depth-zero semantics and the pinned release's verified zero/one controls. Do not treat every successful-looking output substring as sufficient proof; pin selected release outcomes, including empty-source initialization/vacuity, with actual controls and refuse inconclusive or malformed execution.
3. Obtain the bound from a completed native BFS maximum discovery depth plus one using the narrow existing count/metadata route; reject cut/truncated exploration and overflow. Selected attribute Kettle currently has reached=283, depth=62, cut=false, so its required check is at length 63 rather than the short length 3 smoke control. Run one actual pinned-tool check per declared safety property, including postconditions and arithmetic metadata, with no liveness claims. Preserve the later consumer/replay ownership while exposing enough explicit bound/result information for .3/.4. Do not substitute source graphs with native explored edges.
4. Harden pin admission and installation under the canonical approved model/build root: reject existing/missing-component symlink escapes, mismatched archive/jar/version/JDK, partial installs and wrong release before checking any Model. Use a private staging directory, checksum validation, lock and atomic publication; concurrent installers converge without partial visibility. Avoid shell HOME mutation and external installation caches. Keep default gate skip behavior: no Apalache download/launch unless required or explicitly requested.
5. Complete deterministic subprocess lifecycle in shared process support. Every invocation owns a private scratch directory, bounded wall/output/trace limits, stdin closure and output draining. Timeout, cancellation, failed launch/exit and post-launch errors must terminate/reap descendants as well as the direct child, with no leftover process or scratch leak. Current Running(Child) directly kills/waits the child only; add the smallest supported cancellation/process-tree mechanism and preserve TLC call compatibility. Do not duplicate orchestration in the Apalache crate.
6. Emit fixture/property/bound/start/finish/failure progress on stderr on every terminal path, including early refusal or timeout; deterministic result rendering excludes timing, absolute paths and raw diagnostics. Retain raw tool output separately as evidence. Collect traces deterministically and bound aggregate output/trace resource use, without following tool-output symlinks. Required NotRun/timeout/refusal/malformed output fails selected checks and cannot be blessed by golden update.
7. Add focused fake-tool/installer controls for missing/wrong version, digest mismatch, failed launch/exit, inconclusive/malformed success, timeout, cancellation, child cleanup, output limits, symlink escape, partial publication and concurrent installation. Retain actual Kettle/property and length-zero/one controls against the selected release. Preserve all existing source/native/TLC artifacts; only owning task artifacts change deliberately. Run strict changed-crate lint and focused tests before source review. Coordinate root's two-axis audit before the one final full gate.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-apalache`, then `UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache --test controls`. Extend focused runner/installer tests as implemented; use existing Python test runner conventions. The final actual Kettle checks use the completed native-derived bound 63 and every selected safety property, not just the existing length 3 smoke proof.

Final from root: ONE `UMPIRE_BACKENDS=require make model-time` after corrections and reviews settle, with outer timeout at least 2400 seconds (3600 where needed). It subsumes default/full/required checks and timing. Preserve all 17 stages and soft-target misses. Actual source/guest changes require appropriate refreshed guest/live proof; exact unchanged production/guest proof permits explicit artifact reuse, never a synthetic receipt. Preserve the fresh .1 source-scoped live proof and cross-OS boundaries. Do not rerun duplicate full gates solely for Flow evidence commits.
### Narrow review correction scope

The all-Darwin no-descendant policy exposed an existing required helper: the TLC unsafe-cache control executes fetch-tlc.sh and requires its exact containment diagnostic. The conductor authorizes only its root/path/cache containment preflight to use Bash builtins before any fork, preserving the established test and all assertions. Keep post-admission download/checksum/atomic-install behavior intact; do not bypass shared supervision or introduce a second test process framework. Verify the negative control still reaches the intended containment refusal before creating files or attempting a download, including under the simulated Darwin launch policy. Actual native macOS execution remains explicitly unverified.
## Acceptance
- [ ] The existing backend exposes explicit bounded NoViolation/Violation/NotRun behavior with the checked transition bound and retained witness/diagnostic data; missing/mis-versioned tools and inconclusive/malformed runs never pass.
- [ ] Completed native depth derives the bound with checked +1, cut exploration is refused, zero/one semantics remain pinned, and actual Kettle safety properties are checked at native-derived bound 63 (currently depth 62), including postconditions/arithmetic and without liveness claims.
- [ ] Canonical build-root containment rejects symlink escapes, wrong checksums/version/JDK, partial installs and unsafe outputs; private staging, lock and atomic publication pass concurrent-installer controls.
- [ ] Shared subprocess support closes stdin, bounds resources, drains output and terminates/reaps the entire owned process tree on timeout/cancellation/error; focused fake-tool tests prove child/scratch cleanup and preserve TLC behavior.
- [ ] Property/bound progress includes start and every terminal finish/failure on stderr; deterministic result rendering excludes paths, timings and raw diagnostics. Default tests neither launch nor download Apalache; required failures remain failures.
- [ ] Focused fake-tool/installer and actual pinned-tool controls, strict lint, review/audits and one final required model-time pass; exact artifact/source/guest evidence supports every output change and any explicit live-proof reuse.
## Done summary
Finalized bounded Apalache checking with explicit NoViolation/Violation/NotRun outcomes, pinned installation admission, and the completed native Kettle exploration bound (depth62 + 1 = 63). Shared TLC process supervision retains bounded diagnostic handles and cleans up owned processes; focused controls cover installer containment, concurrent publication, parser admission, output limits, cancellation and signal/owner death.

Tier: session (jev-unavailable(no_key)); AGENTS implementer override gpt-6.1-sol medium
stage: impl-review - ran (original fanout NEEDS_WORK; single resume SHIP, explicit codex:gpt-6.1-sol:high; same model family as requested implementer)

Baseline: green focused native/default and actual pinned backend controls. Initial wrong-directory attempt was inconclusive and corrected; no inherited failing baseline was accepted. Meaningful red/green controls and all unchanged audit reports are retained in `.flow/artifacts/fn-211-apalache-backend-for-umpire-in-rust/task2/`.

Verification: `timeout 3600 env UMPIRE_BACKENDS=require make model-time` exited0, all17 stages green, gate903.818s/60s soft target missed, inner7.229s/10s soft target met. Current raw output confirms all five Kettle safety properties NoViolation63. Final gate exercised Python installer/supervisor controls through maintained guest-build-controls discovery.

GATE_SKIPPED:unittest:green-receipt 64d47616 — final Verify after Flow-only evidence commits honored the completed exact-command run; no second full gate was executed.

Runtime/live reuse: conductor verified all2181 runtime/live entries, all558 prior fixture artifacts and guest SHA f832a33c3024a8752976d146be839b62a95afde4790fa0db60f1dac9004b4e7e unchanged at source88803cad1c. Reuse task1 retained fresh live slice/gaps evidence explicitly; no new live execution is claimed.

Platform limit: actual native macOS Temurin27 execution remains unverified. Darwin ownership prevents descendants using child-only hard/soft NPROC0 and refuses privileged execution. Linux simulated-Darwin policy controls passed; Linux mise NPROC0 thread abort is inconclusive for native Darwin. No cross-OS runtime pass is claimed.

Task .3 still owns general typed ITF conversion/replay; .4 owns selected consumer/goldens; .5 owns symbolic Worlds. No feature-entry route changed. No native graph edge emission was added.

Source scope: finalized existing Apalache backend, shared TLC process support, backend tests, and fetch-apalache containment. The only extra authorized path was fetch-tlc.sh for builtin-only preflight before descendant restriction; its post-admission download behavior remains unchanged.

Exact green receipt: `.flow/tmp/green-receipts/64d47616-unittest.json` (durable copy `.flow/artifacts/fn-211-apalache-backend-for-umpire-in-rust/task2/final-green-receipt.json`). Honor command: `python3 /home/agent/.codex/scripts/flowctl.py gate check --gate unittest --command 'timeout 3600 env UMPIRE_BACKENDS=require make model-time'`; exit0 HONORED64d47616 after evidence commitb4853923a2.

| Stage | Seconds |
|---|---:|
| temporal-guest-freshness | 32.245 |
| timing-controls | 0.017 |
| guest-build-controls | 3.572 |
| capability-controls | 0.076 |
| source-boundary | 4.681 |
| protobuf-freshness | 0.079 |
| kettle-protobuf-freshness | 0.076 |
| temporal-protobuf-freshness | 0.055 |
| fmt | 0.496 |
| clippy | 5.363 |
| docs | 64.125 |
| tests | 688.492 |
| wasm32-wasip1 | 0.215 |
| core-wasm-clippy | 0.152 |
| wasm-guest | 53.539 |
| go-host | 38.846 |
| temporal-go-host | 11.411 |

Durable proof directory: `.flow/artifacts/fn-211-apalache-backend-for-umpire-in-rust/task2/`. `final-model-time.log` is the raw gate timing log; `final-gate-report.json` links it to source88803cad1c, verified gateHEAD64d476165b, native bound and all five property outputs/source hashes. `final-backend-controls.tar.gz` retains178 raw Apalache/TLC files; `final-backend-controls-inventory.jsonl` pins every archived byte stream. `source-inventory.jsonl` pins38 source files, all verified exact after the gate. `review-ship-receipt.json` and `review-resume.log` retain same-session SHIP; all prior reports and separate correction addenda remain unchanged. The green receipt is valid across subsequent Flow-only closure commits.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1a3dfc05492dbe0f978b46217c7b51c9bbd027f5, 5f4d62e9d4df25041655bae5ef9eb96c1f2d56b3, 8e7e015f6406151d1d6a66932f635e1fb0ec097d, 0eda2a2669b6649882d6e7978809226da56d6717, 2e1c2680de3b49b8bb7965e784650b863713bb40, c6a4b839c7a9e64b66dbb1d1af6dfd069f50395d, c9d72c6ef6ee26274adab9aff0992ab95c4b656e, 48bfe5c95dc4c1758f90bbb85b1484d821fe78d3, 88803cad1c09ecb6ca831bf8e18aa2ae604ead22, 94f093f756dc9eadd310ec3484d09c9d39950f3c, 64d476165b41f0f673b381449a20c3fca8666820, b4853923a266fd762ac203eb5391ce8bda2cd75f
- Tests: timeout 3600 env UMPIRE_BACKENDS=require make model-time, GATE_SKIPPED:unittest:green-receipt 64d47616
- PRs: