## Requirements Extracted

1. Admit execution from committed fn-237.3 implementation and closure; retain its guest reference and required downstream dependency.
2. Move process ownership into `umpire-backend-process`; update TLA+, TLC and Apalache without adding an Apalache-to-TLC dependency.
3. Preserve process refusals, child outcomes, retained output, Java admission and capture/dump contracts.
4. Provide an allocation-free forked supervisor and target path usable by cold tests and direct library consumers.
5. Preserve Linux subreaper ownership and direct/escaped descendant cleanup before return.
6. Preserve Darwin non-root admission and descendant prevention; refuse unsupported platforms.
7. Preserve cancellation, timeout, first-signal, owner EOF, status 125 and signal-status mappings.
8. Preserve inode/substitution, FIFO, UTF-8, bounded-read, output-budget and cleanup controls.
9. Retain workspace unsafe prohibition with a confined, audited process syscall exception.
10. Introduce a host-only development-profile xtask using raw OS arguments, in-process SHA-256 and sorted native JSON.
11. Preserve deterministic, portable dependency-closure identities, checksums, non-dev/build/platform edges, ambiguity and shared-feature behavior.
12. Preserve compiler argument forwarding, macro-collision checks, pipelined sidecars, failure cleanup, previous wrappers, jobserver descriptors and loader environment.
13. Use the approved native TOML grammar, features, limits and errors; document intentional driver changes.
14. Replace embedded guest Python with native cache, launcher, build and artifact-selection responsibilities.
15. Bootstrap before guest flags/wrappers, disable automatic installation before the first probe and fail clearly on bootstrap errors.
16. Validate installed tools, pins, rustup, Rust sources and wasm readiness on cold and cached freshness paths.
17. Distinguish fresh, stale/missing and tool/build failures; preserve useful path handling and never publish in check mode.
18. Justify guest regeneration with old/new hashes, two clean builds and freshness against the committed asset.
19. Install the fixed pinned Apalache archive using archive-hash-before-parse, in-process tar/gzip reading, exact regular-member admission and jar-hash-before-publication.
20. Preserve installer containment, symlink/lock admission, private staging, cleanup, curl safety, concurrency and atomic publication.
21. Replace retired-name filtering with native raw-stream handling while retaining POSIX ERE behavior and failure propagation.
22. Transfer useful Python/process controls to Rust, record retired quirks and execute native controls through the ordinary gate without Python.
23. Remove the checksum-only verifier and manifest while preserving originals, lifecycle comparison and its mutation control.
24. Update prerequisites, ownership, native contracts, platform policy, dependency and validation documentation.
25. Retain approved dependency versions/features, R18 accounting, before/after inventories, guest separation and passing bans checks.
26. Make canonical validation execute one TLC-required complete gate and one benchmark, with Apalache independently selectable.
27. Preserve legacy default/skip modes, conflict resistance, fail-fast behavior, diagnostics, status propagation and soft timing warnings.
28. Record actual joined semantic/backend/host execution, commands, stages, timings and platform limitations.
29. Adopt private `tempfile::Builder` scratch directories with explicit close and the existing symlink-refusing walk.
30. Adopt safe parent-side rustix operations, retain raw child syscalls and demonstrate fewer audited unsafe sites.
31. Join isolated implementation lanes and complete integrated caller, dependency, installation, lifecycle and guest proofs.
32. Preserve Model/checker/protocol/Temporal semantics and frozen originals; avoid unrelated compiler or dependency upgrades.

## Coverage Verification

1. **COVERED** — admission and predecessor ancestry: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task1/admission.json:2`.
2. **COVERED** — ownership and consumers: `model/umpire/backends/process/src/lib.rs:1`, `model/umpire/backends/tla/src/lib.rs:8`, `model/umpire/backends/tlc/src/runner.rs:17`, `model/umpire/backends/apalache/src/lib.rs:13`.
3. **COVERED** — typed process results and Java/capture admission: `model/umpire/backends/process/src/support.rs:109`, `model/umpire/backends/process/src/support.rs:150`.
4. **COVERED** — prepared child context and non-returning child execution: `model/umpire/backends/process/src/syscall.rs:156`, `model/umpire/backends/process/src/syscall.rs:404`; cold-consumer and stress evidence: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task4/README.md:1`.
5. **COVERED** — subreaper setup and adopted-child cleanup: `model/umpire/backends/process/src/syscall.rs:404`, `model/umpire/backends/process/src/syscall.rs:736`; real lifecycle controls: `model/umpire/backends/process/tests/process.rs:101`, `model/umpire/backends/process/tests/process.rs:438`.
6. **COVERED** — platform admission and Darwin restriction: `model/umpire/backends/process/src/syscall.rs:20`, `model/umpire/backends/process/src/syscall.rs:649`. Native Darwin execution is explicitly unclaimed.
7. **COVERED** — cancellation/timeout and supervisor outcomes: `model/umpire/backends/process/src/support.rs:176`, `model/umpire/backends/process/src/syscall.rs:404`; owner-EOF and first-signal controls: `model/umpire/backends/process/src/syscall.rs:918`.
8. **COVERED** — held capture handles, bounded output and dump checks: `model/umpire/backends/process/src/support.rs:176`, `model/umpire/backends/process/src/support.rs:250`, `model/umpire/backends/process/src/support.rs:295`; negative controls: `model/umpire/backends/process/tests/process.rs:293`.
9. **COVERED** — workspace forbid and confined exception: `model/Cargo.toml:112`, `model/umpire/backends/process/Cargo.toml:21`, `model/umpire/backends/process/src/support.rs:1`, `model/umpire/backends/process/src/syscall.rs:7`.
10. **COVERED** — host tool, raw arguments, hashing and canonical JSON: `model/dev/xtask/Cargo.toml:1`, `model/dev/xtask/src/main.rs:10`, `model/dev/xtask/src/support.rs:15`, `model/dev/xtask/src/support.rs:31`.
11. **COVERED** — native closure graph and portable coordinates: `model/dev/xtask/src/lock.rs:104`, `model/dev/xtask/src/lock.rs:135`; topology/checksum/ambiguity controls: `model/dev/xtask/tests/identity/lock.rs:73`.
12. **COVERED** — compiler identities, sidecars and direct execution: `model/dev/xtask/src/compiler.rs:267`, `model/dev/xtask/src/compiler.rs:364`, `model/dev/xtask/src/compiler.rs:435`; inheritance/failure controls: `model/dev/xtask/tests/identity/compiler.rs:367`.
13. **COVERED** — approved parser configuration: `model/Cargo.toml:92`; native contracts and intentional breaks: `model/dev/xtask/README.md:17`; current-document/malformed-input control: `model/dev/xtask/tests/identity/current_inputs.rs:4`.
14. **COVERED** — native guest orchestration and shell switch: `model/dev/xtask/src/guest.rs:157`, `model/dev/xtask/src/guest.rs:214`, `model/scripts/guest.sh:13`.
15. **COVERED** — clean native bootstrap and disabled auto-install: `model/scripts/xtask.sh:3`; actual cold/failure controls: `model/dev/xtask/tests/guest_pipeline.rs:202`, `model/dev/xtask/tests/guest_pipeline.rs:230`.
16. **COVERED** — native prerequisite admission: `model/dev/xtask/src/guest.rs:74`; cached/missing/probe/rustup controls: `model/dev/xtask/tests/guest_pipeline.rs:202`, `model/dev/xtask/tests/guest_pipeline.rs:256`, `model/dev/xtask/tests/guest_pipeline.rs:361`.
17. **COVERED** — artifact schema/discovery and read-only comparison: `model/dev/xtask/src/guest.rs:189`, `model/dev/xtask/src/guest.rs:299`; stale/path/Make controls: `model/dev/xtask/tests/guest_pipeline.rs:171`, `model/dev/xtask/tests/guest_pipeline.rs:288`, `model/dev/xtask/tests/guest_pipeline.rs:330`.
18. **COVERED** — two-clean-build proof and subsequent canonical publication: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task5/joined/proof-manifest.json:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task5/joined/canonical-publication.json:1`.
19. **COVERED** — one pin authority and hash/member ordering: `model/dev/xtask/src/apalache_pins.rs:3`, `model/dev/xtask/src/fetch_apalache.rs:108`; backend shares the authority: `model/umpire/backends/apalache/src/lib.rs:14`.
20. **COVERED** — lock/path/staging/publication controls: `model/dev/xtask/src/fetch_apalache.rs:42`, `model/dev/xtask/src/fetch_apalache.rs:55`, `model/dev/xtask/src/fetch_apalache.rs:133`; concurrency and failure tests: `model/dev/xtask/tests/apalache/cases.rs:47`.
21. **COVERED** — raw records, exemption boundaries and POSIX grep: `model/dev/xtask/src/retired_names.rs:33`, `model/dev/xtask/src/retired_names.rs:58`; pipeline error precedence: `model/scripts/retired-names.sh:16`, `model/dev/xtask/tests/retired_names.rs:270`.
22. **COVERED** — complete responsibility mapping and census: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/control-mapping.md:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/census.json:1`; ordinary native test stage: `Makefile:1289`.
23. **COVERED** — verifier/manifest deletions and retained semantic/mutation controls: `model/temporal/feature/activity/compatibility/checkpoint.rs:146`, `model/temporal/feature/activity/compatibility/checkpoint.rs:154`. No retained-original path changed.
24. **COVERED** — current ownership, contracts and safety documentation: `model/README.md:34`, `model/GUEST_BUILDS.md:1`, `model/dev/xtask/README.md:1`, `model/DESIGN.md:245`, `.plans/UMPIRE_MODULES.md:25`.
25. **COVERED** — manifests plus package/feature/R18 evidence: `model/Cargo.toml:86`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task2/dependency-inventory.json:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task6/dependency-inventory.json:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task9/audit.md:25`.
26. **COVERED** — canonical target, existing complete gate and benchmark: `Makefile:1274`, `Makefile:1308`, `model/scripts/time-gate.sh:39`; Apalache remains independently selected at `Makefile:1277`.
27. **COVERED** — actual Make/timer conflict, propagation, failure and warning controls: `model/scripts/test-model-validate.sh:36`, `model/scripts/test-model-validate.sh:69`; wired through `model/scripts/test-time-gate.sh`.
28. **COVERED** — final actual combined-backend run: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/model-validate.log:1`. It records 18 stages, one benchmark and both timing overruns.
29. **COVERED** — private TempDir creation, unchanged walk and explicit cleanup: `model/umpire/backends/process/src/support.rs:55`, `model/umpire/backends/process/src/support.rs:94`; ownership/result controls: `model/umpire/backends/process/tests/process.rs:545`.
30. **COVERED** — safe parent identity/descriptor/wait/resource operations: `model/umpire/backends/process/src/syscall.rs:62`, `model/umpire/backends/process/src/syscall.rs:102`, `model/umpire/backends/process/src/syscall.rs:137`; documented raw-call exceptions and unsafe reduction 77→67: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task9/audit.md:13`.
31. **COVERED** — joined dispatcher/bootstrap/lifecycle checks, real installation and guest proofs: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/integration/joined-validation.json:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task6/joined/execution-receipt.json:1`, `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task5/review/guest-proof-continuity.json:1`.
32. **COVERED** — production changes are tooling/process ownership, mechanical consumers and the authorized checkpoint retirement. The retained originals and frozen Model sources have no diff; the guest regeneration has an explicit cause and current-input proof.

Independent review checks confirmed all **3,337** final validation input hashes match, the committed guest matches SHA-256 `bc27af7d5b6b1ed8fa5c524015046f3a8b7c12d2d488e9ce7cb3df3ea9217872`, tracked Model Python inventory is empty, and shell syntax/diff checks pass. Both Quick Cargo commands stopped before executing tests because the sandbox cannot open Cargo’s build-lock file. Executed-suite coverage therefore relies on the retained, matching-source evidence; the full gate was not repeated.

## Reverse Coverage (untraced changes)

The Model implementation, removed helpers/tests, native controls, ownership documentation and fn-247 evidence trace to the requirements above. The consumer dependency-version correction is legitimate support for the required bans check. The following files contain separately owned work; these scope flags do not establish a completion gap.

- `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/planning/downstream-fn2374-amendment-pending.md` - UNRELATED_CHANGE - downstream realization planning.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/parallel-readiness-amendment.json` - UNRELATED_CHANGE - fn-248 planning.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/parallel-readiness-amendment.md` - UNRELATED_CHANGE - fn-248 planning.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/provenance.md` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/receipt.json` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/reviewed-input-hashes.json` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/round1-receipt.json` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/round1.log` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/round1.md` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/round2.log` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/round2.md` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/status.json` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/plan-review/validation.json` - UNRELATED_CHANGE - fn-248 review evidence.
- `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/planning/readiness-audit.md` - UNRELATED_CHANGE - fn-248 planning.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/.gitattributes` - UNRELATED_CHANGE - fn-249 artifact conversion policy.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/implementation/cleanup.json` - UNRELATED_CHANGE - fn-249 implementation evidence.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/implementation/impl-review-receipt.json` - UNRELATED_CHANGE - fn-249 review evidence.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/implementation/review-round-1.json` - UNRELATED_CHANGE - fn-249 review evidence.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/implementation/validation-evidence.json` - UNRELATED_CHANGE - fn-249 validation evidence.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/planning/open-plan-policy-and-dependencies.json` - UNRELATED_CHANGE - fn-249 planning.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/planning/plan-review-receipt.json` - UNRELATED_CHANGE - fn-249 review evidence.
- `.flow/artifacts/fn-249-durable-validation-cost-review-at-spec/validation-cost-review-ef5dff8cfba6956e04158de363192217cfaa473a2d6ae8a1608c1d3e051c8e2c.json` - UNRELATED_CHANGE - fn-249 closure evidence.
- `.flow/config.json` - UNRELATED_CHANGE - enables fn-249 validation-review policy.
- `.flow/memory/bug/data/preserve-git-conversion-policy-for-2026-10-24.md` - UNRELATED_CHANGE - fn-249 conversion-policy lesson.
- `.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.json` - UNRELATED_CHANGE - other-spec planning.
- `.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md` - UNRELATED_CHANGE - other-spec compatibility/consolidation plan.
- `.flow/specs/fn-237-activity-timeouts-heartbeat-and.json` - UNRELATED_CHANGE - other-spec planning.
- `.flow/specs/fn-237-activity-timeouts-heartbeat-and.md` - UNRELATED_CHANGE - timeout/realization planning amendments.
- `.flow/specs/fn-240-symbolic-back-end-models-as-btor2.json` - UNRELATED_CHANGE - deferred symbolic-backend planning.
- `.flow/specs/fn-240-symbolic-back-end-models-as-btor2.md` - UNRELATED_CHANGE - deferred symbolic-backend planning.
- `.flow/specs/fn-243-programmed-workers-the-model-issues.json` - UNRELATED_CHANGE - worker-spec planning.
- `.flow/specs/fn-243-programmed-workers-the-model-issues.md` - UNRELATED_CHANGE - worker-spec compatibility amendments.
- `.flow/specs/fn-245-finite-collections-sets-maps-and.json` - UNRELATED_CHANGE - collection-spec planning.
- `.flow/specs/fn-245-finite-collections-sets-maps-and.md` - UNRELATED_CHANGE - collection/consolidation planning.
- `.flow/specs/fn-246-realization-as-a-checked-refinement-a.json` - UNRELATED_CHANGE - realization-spec planning.
- `.flow/specs/fn-246-realization-as-a-checked-refinement-a.md` - UNRELATED_CHANGE - realization architecture/scheduling amendments.
- `.flow/specs/fn-248-off-the-shelf-crates-and-consolidation.json` - UNRELATED_CHANGE - separate consolidation spec.
- `.flow/specs/fn-248-off-the-shelf-crates-and-consolidation.md` - UNRELATED_CHANGE - separate consolidation spec.
- `.flow/specs/fn-249-durable-validation-cost-review-at-spec.json` - UNRELATED_CHANGE - separate validation-review spec.
- `.flow/specs/fn-249-durable-validation-cost-review-at-spec.md` - UNRELATED_CHANGE - separate validation-review spec.
- `.flow/specs/fn-250-shared-value-type-and-trace-layer-for.json` - UNRELATED_CHANGE - future shared-value spec.
- `.flow/specs/fn-250-shared-value-type-and-trace-layer-for.md` - UNRELATED_CHANGE - future shared-value spec.
- `.flow/specs/fn-251-one-probe-engine-and-one-catalog-build.json` - UNRELATED_CHANGE - future probe-engine spec.
- `.flow/specs/fn-251-one-probe-engine-and-one-catalog-build.md` - UNRELATED_CHANGE - future probe-engine spec.
- `.flow/specs/fn-252-quick-consolidation-wins-in-model-code.json` - UNRELATED_CHANGE - future consolidation spec.
- `.flow/specs/fn-252-quick-consolidation-wins-in-model-code.md` - UNRELATED_CHANGE - future consolidation spec.
- `.flow/tasks/fn-211-apalache-backend-for-umpire-in-rust.4.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-211-apalache-backend-for-umpire-in-rust.4.md` - UNRELATED_CHANGE - deferred consumer plan.
- `.flow/tasks/fn-211-apalache-backend-for-umpire-in-rust.5.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-211-apalache-backend-for-umpire-in-rust.5.md` - UNRELATED_CHANGE - symbolic-World consumer plan.
- `.flow/tasks/fn-237-activity-timeouts-heartbeat-and.4.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-237-activity-timeouts-heartbeat-and.4.md` - UNRELATED_CHANGE - live timeout/realization plan.
- `.flow/tasks/fn-237-activity-timeouts-heartbeat-and.5.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-237-activity-timeouts-heartbeat-and.5.md` - UNRELATED_CHANGE - downstream validation plan.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.1.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.1.md` - UNRELATED_CHANGE - symbolic emitter plan.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.2.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.2.md` - UNRELATED_CHANGE - solver-admission plan.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.3.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.3.md` - UNRELATED_CHANGE - symbolic replay plan.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.4.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.4.md` - UNRELATED_CHANGE - symbolic fixture plan.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.6.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-240-symbolic-back-end-models-as-btor2.6.md` - UNRELATED_CHANGE - symbolic documentation plan.
- `.flow/tasks/fn-243-programmed-workers-the-model-issues.5.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-243-programmed-workers-the-model-issues.5.md` - UNRELATED_CHANGE - worker live-proof plan.
- `.flow/tasks/fn-245-finite-collections-sets-maps-and.1.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-245-finite-collections-sets-maps-and.1.md` - UNRELATED_CHANGE - collection IR plan.
- `.flow/tasks/fn-245-finite-collections-sets-maps-and.3.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-245-finite-collections-sets-maps-and.3.md` - UNRELATED_CHANGE - collection backend plan.
- `.flow/tasks/fn-245-finite-collections-sets-maps-and.4.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-245-finite-collections-sets-maps-and.4.md` - UNRELATED_CHANGE - collection validation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.1.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.1.md` - UNRELATED_CHANGE - realization design plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.10.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.10.md` - UNRELATED_CHANGE - realization strategy plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.11.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.11.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.12.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.12.md` - UNRELATED_CHANGE - realization validation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.13.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.13.md` - UNRELATED_CHANGE - realization follow-up plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.14.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.14.md` - UNRELATED_CHANGE - worker continuation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.2.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.2.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.3.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.3.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.4.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.4.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.5.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.5.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.6.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.6.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.7.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.7.md` - UNRELATED_CHANGE - realization implementation plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.8.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.8.md` - UNRELATED_CHANGE - whole-prefix admission plan.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.9.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.9.md` - UNRELATED_CHANGE - whole-prefix recording plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.1.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.1.md` - UNRELATED_CHANGE - snapshot consolidation plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.2.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.2.md` - UNRELATED_CHANGE - helper consolidation plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.3.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.3.md` - UNRELATED_CHANGE - IR traversal plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.4.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.4.md` - UNRELATED_CHANGE - macro consolidation plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.5.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.5.md` - UNRELATED_CHANGE - dependency consolidation plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.6.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.6.md` - UNRELATED_CHANGE - ITF decoder plan.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.7.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-248-off-the-shelf-crates-and-consolidation.7.md` - UNRELATED_CHANGE - consolidation integration plan.
- `.flow/tasks/fn-249-durable-validation-cost-review-at-spec.1.json` - UNRELATED_CHANGE - other-spec task metadata.
- `.flow/tasks/fn-249-durable-validation-cost-review-at-spec.1.md` - UNRELATED_CHANGE - validation-review hook plan.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.1.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.1.md` - UNRELATED_CHANGE - shared Literal plan.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.2.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.2.md` - UNRELATED_CHANGE - shared Schema plan.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.3.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.3.md` - UNRELATED_CHANGE - shared replay plan.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.4.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.4.md` - UNRELATED_CHANGE - shared lowering plan.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.5.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-250-shared-value-type-and-trace-layer-for.5.md` - UNRELATED_CHANGE - shared jar-admission plan.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.1.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.1.md` - UNRELATED_CHANGE - shared descriptor plan.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.2.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.2.md` - UNRELATED_CHANGE - shared generation plan.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.3.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.3.md` - UNRELATED_CHANGE - generated request-finding plan.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.4.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.4.md` - UNRELATED_CHANGE - shared probe-engine plan.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.5.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.5.md` - UNRELATED_CHANGE - annotation dispatch plan.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.6.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-251-one-probe-engine-and-one-catalog-build.6.md` - UNRELATED_CHANGE - facade consolidation plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.1.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.1.md` - UNRELATED_CHANGE - shared search plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.2.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.2.md` - UNRELATED_CHANGE - path representation plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.3.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.3.md` - UNRELATED_CHANGE - spike consolidation plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.4.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.4.md` - UNRELATED_CHANGE - fixture consolidation plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.5.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.5.md` - UNRELATED_CHANGE - test harness consolidation plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.6.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.6.md` - UNRELATED_CHANGE - shared activity replay plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.7.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.7.md` - UNRELATED_CHANGE - tooling consolidation plan.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.8.json` - UNRELATED_CHANGE - future-spec task metadata.
- `.flow/tasks/fn-252-quick-consolidation-wins-in-model-code.8.md` - UNRELATED_CHANGE - runtime fixture consolidation plan.
- `AGENTS.md` - UNRELATED_CHANGE - mixed file: canonical command serves R5; added closure-review policy belongs to fn-249.
- `MILESTONES.md` - UNRELATED_CHANGE - mixed fn-247 progress and separately owned roadmap/scheduling changes.
- `tools/flow/README.md` - UNRELATED_CHANGE - fn-249 tooling documentation.
- `tools/flow/install_validation_review.py` - UNRELATED_CHANGE - fn-249 installer.
- `tools/flow/test_validation_preflight.py` - UNRELATED_CHANGE - fn-249 tests.
- `tools/flow/test_validation_review.py` - UNRELATED_CHANGE - fn-249 tests.
- `tools/flow/validation-review.patch` - UNRELATED_CHANGE - fn-249 closure-review implementation.

## Gaps Found

None. No qualifying completion gaps were found.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Process ownership, audited supervisor and native lifecycle/capture controls; coverage items 2–9. |
| R2 | met | Native identity, guest, installer and stream commands with negative controls and reproduction evidence; items 10–21. |
| R3 | met | Native responsibility mapping, empty tracked Python inventory, ordinary gate ownership and retained lifecycle mutation control; items 22–23. |
| R4 | met | Current contracts, ownership/safety documentation, dependency accounting and actual validation evidence; items 24–25, 28. |
| R5 | met | Forced required-TLC target, independent Apalache selection, legacy-mode/failure controls and one final gate/benchmark; items 26–28. |
| R6 | met | TempDir scratch, safe parent rustix operations, unchanged raw child boundary and unsafe reduction; items 29–30. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

## Verdict

<verdict>SHIP</verdict> - All requirements implemented (R-IDs all met or deferred)
