---
satisfies: [R1, R3, R4]
---
# fn-231-declarative-realizations-the.1 Realization macro crate: #[realization], #[observe], #[answer]; Step and Reply; generated Realize impl; activity rewritten with expansion snapshot

## Description
Add a proc-macro sub-crate under model/temporal providing #[realization(<module>)], #[observe(Service::Method, reads(..), ignores(..))] and #[answer(Task, reads(..), ignores(..))] plus the reserved #[propose]; return-type classification (state-valued, Step<A>, Projection); generated Realize impl dispatching on method path through the fn-229 helpers with Unsupported for unlisted methods. Add Step and Reply to the realization crate. Rewrite the activity Realization on the attributes; all activity goldens, wire goldens, slice and cover tests byte-identical. trybuild pins duplicate handler, unclassifiable return, wildcard enum arm; insta expansion snapshot via prettyplease. Boundary scan passes. Gate and model-time pasted.

### Execution scope and verification

**Touches:** `model/Cargo.toml`, `model/Cargo.lock`, `model/temporal/realization/**`, `model/temporal/feature/activity/temporal.rs`, `model/temporal/feature/activity/phase.rs`, `model/temporal/annotate/syntax/**`, `model/temporal/api/**`, `model/temporal/testing/**`, `model/temporal/hosts/go/umpire_temporal_guest.wasm`, `model/temporal/realization/README.md`, `model/README.md`
**Quick:** focused macro parser/expansion and trybuild controls; actual realization/helper tests with refusal-order, unknown Cancel code, Current/one-step/unexplained projection, callback and partial-reply controls; native activity fixtures and existing fn-218 cover controls; generic source boundary; `make umpire-guest` followed by actual `make umpire-check-model-slice` and `make umpire-check-model-gaps`, retaining all cases/privacy/goldens and existing operation/aggregate limits.
**Full gates:** `make model-check`; `UMPIRE_BACKENDS=require make model-check`; `make model-time`.

Use the integrated source, not the old `features/activity` spelling or five-message
examples. The read-only seam research is `.flow/tmp/fn231-realization-seam-research.md`.
It was re-anchored before fn-230.2; recheck current source rather than assuming its
line numbers or the former test-build exclusion still apply. The current catalog
has sixteen WorkflowService request/response messages plus nested declarations.
Reuse the admitted shared parser and descriptor graph with an acyclic dependency
boundary. No second parser for Catalog or generated API imports in its syntax crate.

Preserve the exact existing observable behavior. Select method before decoding;
Cancel's unknown raw i32 code yields empty candidates, and Cancel/Describe retain
their distinct decode refusal mappings. State conversion is explicitly owned by
the Realization; Started from Describe must not invent Poll. Exhaustiveness checks
target admitted generated-enum matches, preserving legitimate byte-policy fallbacks.
Retain existing helper, atomicity, rollback, external-payload and cover controls.
The existing `phase.rs` protobuf-enum conversion must also become wildcard-free,
as the parent requires for helper conversions. Preserve status-first/run-second
decoding, every current status/run-state mapping, and exact refusal precedence
and messages. Retain the independent full pair/unknown-enum control unchanged;
add a compile control proving a new declared enum variant is non-exhaustive.
This admits only exhaustive spelling of existing behavior, not a phase change.
Partial callbacks and outgoing SDK replies are not complete catalog-valid requests;
do not newly require SDK-filled task tokens or double-wrap the Reply envelope.
Document the `Reply.request` packing contract and pin actual decoded outgoing bytes.

Unknown/overlapping path and empty-ignore admission must follow the parent contract.
Keep incoming callback, observed response and outgoing request contexts distinct.
Reasoned ignores preserve currently accepted unmodeled fields; unsupported checks
use decoded protobuf optional/oneof presence, never opaque IgnoredField presence.
Do not defer a runtime safety/admission obligation merely because the full coverage
golden belongs to task .2. Any shared metadata seam required now must support that
successor without a duplicate declaration list. The runtime owns state freezing,
transition judgment and transactional publication; do not invent new quiescent or
pending semantics absent from its existing trait. Propose remains reserved only.

No Model edits, phase-mapping changes, fitted judgments, report/rows/states/wire or
live-gap golden updates, ABI/protocol changes, dependency/version additions, Go/SDK
panic changes, timeout extensions or unrelated refactors. Existing workspace test
dependencies may be reused. Refresh committed guest bytes after Rust changes with
the admitted generic builder. The separate fn-232 macOS proof remains open and
does not block this task. Task .2 owns realization.snap.txt and DESIGN.md.
## Acceptance
- [ ] Macro crate with the three attributes and reserved propose; Step and Reply
- [ ] Activity Realization on attributes; goldens, slice and cover tests byte-identical
- [ ] trybuild refusals and expansion snapshot pinned; boundary scan passes

## Done summary
Implemented declarative realization attributes, Step/Reply vocabulary, shared descriptor/input metadata and the actual activity rewrite while preserving Model, existing goldens, wire behavior and raw Cancel refusal ordering. Generated-enum admission and the phase helper have actual RED/GREEN and ordinary compile controls; the rebuilt guest exactly matches the captured successful live slice, 2336-case gaps/privacy run and report.

Tier: session (jev-unavailable(no_key)), explicit project pin retained.
stage: impl-review - ran [2026-10-21T11:34:03Z..2026-10-21T11:59:33.310803Z] (SHIP on round 2; actual reviewer model gpt-6.1-sol at high, same-family).

Baseline: green before edits at b9f26782e39f815d740ea4b89925a723303c861d; first wrong-cwd native invocation was inconclusive and corrected before edits. Installed standalone worker.md was missing; supplied worker instructions plus complete work skill/phases governed the installation fallback.

Final owner native/UI/snapshot/fixtures/boundary tests, strict clippy, formatting and generic guest rebuild returned 0. Actual post-SHIP make model-check, UMPIRE_BACKENDS=require make model-check and make model-time each returned 0. Timing total 85.714s exceeds the unchanged 30s soft target (predecessor 85.295s); inner 1.634s is below 10s. No timeout or budget changed. All receipt attempts were refused because foreign MILESTONES.md is dirty; no reusable receipt or gate skip is claimed.

Authoritative context: .flow/artifacts/fn-231-declarative-realizations-the/task-1-verification.md, task-1-quality-audits.md and task-1-formal-review.md. Exact actual SHIP receipt: task-1-formal-ship-receipt.json; live receipt /tmp/impl-review-receipt-8f37faba39e2-fn-231-declarative-realizations-the.1.json. Reviewer retained-binary checks and read-only-blocked builds are not fresh-build passes. Nontrivial fix captured in bug/integration/prove-result-shape-before-propagating-2026-10-21.

Full base range includes the root-owned fn230 retirement artifact-only commit 1b71756; it is not fn231 source work. Foreign MILESTONES/fn233 files remain untouched and excluded. Task .2 owns realization.snap.txt and DESIGN; fn232 cross-OS proof remains open/nonblocking; seven inherited broad umbrella failures are not claimed green. No parent completion or successor admission performed.

All started commands are terminal. Integrated temporary worktrees were removed normally. Source/index and serial fn2301-unwind Cargo/Go cache leases return to the conductor after closure.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fbee9856c4586f8216eb64a3bb3140efcfe43fad, 1b71756bcdd5e80d533ec329b464d064f14b9bea, d59eb1d5b0f072fade2471cecacfeb37dcc643bc, 263f3cdaa13960f680464c354a7b0b8587cff692, 1598f342aef1098b8413136f164f4b929f21f1f7, fe98aede9c4995d8d021000bc914109044f00ba7, a65b4e4623c51296a8a537927252f8c6a5746fd6, 3067e81ec78b8efe4aa28ca13a1e1f1378efa0ef, cd782674e0de1a645abc29cf7aa5fe4174512375, eddbacd8fbb19611c6ed549030d5ea3c5ac22f05, 05f448d36d20e5ad4720e2c131209b0cb4e2c71c, 22482ab2cc5a4cc592d5bbb52cb483b37668add4, 653fd40331fb5740940d6f91df8ad3c234ab9c4e, f8df139c1cb23586972f4841ffb9358942cf74a1, 8e97b484a3d38bc194d8b583fc43150bfe5deb52
- Tests: mise exec -- cargo test --manifest-path model/Cargo.toml -p umpire-temporal-api -p umpire-temporal-annotate-syntax -p umpire-temporal-realization-macros -p umpire-temporal -p umpire-temporal-tests --all-features, mise exec -- cargo clippy --manifest-path model/Cargo.toml -p umpire-temporal-api -p umpire-temporal-annotate-syntax -p umpire-temporal-realization-macros -p umpire-temporal -p umpire-temporal-tests --all-targets --all-features -- -D warnings, mise exec -- cargo fmt --manifest-path model/Cargo.toml --all --check, make umpire-guest, make umpire-check-model-slice, make umpire-check-model-gaps, make model-check, UMPIRE_BACKENDS=require make model-check, make model-time
- PRs: