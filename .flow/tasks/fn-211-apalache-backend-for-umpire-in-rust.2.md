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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
