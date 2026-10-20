---
satisfies: [R2, R6]
---
# fn-217-annotation-catalog-over-the-temporal.1 umpire-annotations: generic kinds, tags, rules, expectations, validate and fill

## Description
New crate `model/umpire/annotations/` (`umpire-annotations`), system-agnostic:
`Kind` with protovalidate-named constraints, `Tag` trait with `Consequence`,
`Rule` (JSON Schema presence algebra: required, any_of, one_of, requires, excludes),
`Expectation`, `Catalog` keyed by field paths, `validate`, seeded `fill`,
`Violation`, and `GapReport` with deterministic readable output. Admission failures
must be typed and must retain the information needed by task .2's compile-time
catalog macro. Preserve the approved spec's full vocabulary and constraints,
including external Secret values, redaction, and expectations for request rules.

Use actual typed field access, not protobuf reflection or another validation engine.
The public surface must be consumable by task .2's exhaustive prost struct patterns.
Borrow protovalidate names and semantics; reuse existing approved dependencies only.
Test valid and invalid values, each presence operator, deterministic filling,
non-fillable and decided fields, tags and admission errors, and secret redaction.
Goldens must reflect actual validation and gap-report output, not hand-built reports.
System-specific tags and catalogs arrive in later tasks.

**Touches:** `model/umpire/annotations/**`, `model/Cargo.toml`, `model/Cargo.lock`
**Quick:** `cd model && mise exec -- cargo test -p umpire-annotations`; `cd model && mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings`
**Full gates:** `make model-check`; `make model-time` (record feedback-loop timings).
## Acceptance
- Generic crate exposes kinds, typed fields, tags/consequences, presence rules,
  expectations, catalog, validation, seeded filling, violations and gap-report types.
- Each operator and implemented kind constraint has meaningful positive and negative
  controls with deterministic report goldens; no silent validation or filling failure.
- Filling respects declared roles and constraints; identical seed/input/catalog gives
  identical output; impossible generation and missing external secrets are refused.
- Secret values cannot appear in validation or gap reports; admission refuses
  conflicting tags, unsupported kind/type pairs, ignored-field rules and missing
  request expectations. Task .2 will pin the corresponding compile-time errors.
- Generic source boundary and focused/full checks pass; no system names or new
  third-party dependencies are introduced.
## Done summary
Implemented the system-independent typed annotation engine with all ten kinds,
tag consequences and admission, presence rules, request expectations, value-free
validation, redaction, transactional seeded fill and deterministic gap reports.
Typed accessors/projections and kind witnesses support task .2's compile-time layer.

The independent Model, application protocol and existing feature goldens are
unchanged by this engine. Native time carriers and explicit generation/search
limits are documented. This task does not claim the later macro, live catalog,
gap probe or reverse coverage requirements complete.

Baseline: existing model gate green; focused commands initially failed because
the new crate did not exist. Worker controls were committed red before fixes.
Five early review regressions, the exact-length anchored-text audit regression
and all three formal review findings were reproduced and corrected. An independent
oracle covers 6,174 small string-domain cases; additional controls pin Unicode,
UTF-8 byte-length generation, typed bounded-search uncertainty, aliases and
17/18-field presence composition. Final worker suite: 23 integration tests and
two doctests; focused clippy and full make model-check pass at 00364f8e0f.
Final model-time: 12.918s full gate and 0.786s annotation edit/test loop.

Conductor full model-check passed at cdf7f3c9e6766aadeb7aab6cadb89b15b40616c6,
using the isolated umpire-integrated target, including source boundary, native
fixtures, wasm and both Go-host parity suites. No optional TLC execution is claimed.
The prior first-review finalize failure was refunded, not presented as a verdict;
its actual findings were retained and fixed. The valid formal fanout recorded
NEEDS_WORK with three findings, all fixed in the resumed SHIP review.

stage: wave-join - ran (isolated engine and disjoint catalog correction commits integrated; full root gate green)
stage: impl-review - ran (fanout f38e74f1081f416ba9334c25d05de26f NEEDS_WORK; single resumed review SHIP; model: gpt-6.1-sol high, same family as writer; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.1.json)
stage: plan-sync - skipped(config: disabled)

Correctness/standards audit reports are preserved verbatim under
.flow/artifacts/fn-217-annotation-catalog-over-the-temporal/task-1-quality-audit.md,
with separate conductor fix dispositions. No Critical findings remain; the sole
Should Fix was corrected. The uniqueness-cost Consider remains nonblocking.
## Evidence
- Commits: 8baf7c6ef32bad11bccdda9a53c99f2b1d6ea831, d3a845f413167bba4742339eeadcd6577a2a9437, 4b04d0a3a7f0f877526be013d9e8b2311f3b2861, 665e8d533483d34289b45a535e643e4c04e567ea, dc071f05c1b216119274a96ad5937d479e56dc6d, 8afe887b9f05d3288e71cef98448678c0f5903b0, 02bfd95d48f83ac14ad2a870a6a62ddb28960a4e, 40240a5c554f7f6ca739984cdf25b17824cd92fe, 00364f8e0f3fbd42ead1fd600989b655bdb005f0, cdf7f3c9e6766aadeb7aab6cadb89b15b40616c6
- Tests: baseline: focused Quick test and clippy exited 101 before edits because umpire-annotations did not exist; existing make model-check exited 0; final isolated-target gates below replace shared-target baseline as authoritative lane proof, cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations (exit 0; 9 integration tests and 2 doctests; /tmp/fn2171-final-focused.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings (exit 0; /tmp/fn2171-final-clippy.log; final source additionally checked by model-time and postcommit model-check), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target make model-check (exit 0; committed source; /tmp/fn2171-postcommit-model-check.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target UMPIRE_INNER_CRATE=umpire-annotations UMPIRE_INNER_SOURCE=umpire/annotations/src/lib.rs make model-time (exit 0; full gate 16.653s; inner loop 0.940s; /tmp/fn2171-final-model-time.log), red-to-green: missing crate/API test, overlapping string generation, zero-length bytes, unconstrained enums, and canonical gap ordering (exit 101 controls in /tmp/fn2171-red.log, /tmp/fn2171-regression-red.log and /tmp/fn2171-boundary-red*.log; exit 0 final focused suite), flowctl gate classify --base 32d63cceeaeba286d6bb843e1fb415b82ab82452 (FULL; exit 1), git diff --check (exit 0), correction baseline: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations (exit 0; /tmp/fn2171-fixes-baseline.log), correction baseline: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings (exit 0; /tmp/fn2171-fixes-baseline-clippy.log), red reproduction on d3a845f413167bba4742339eeadcd6577a2a9437: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations --test generation (exit 101; all five regressions fail; /tmp/fn2171-five-findings-red.log), red sibling-pattern control: unique repeated text, bytes and unrestricted enums all refuse valid domains before the shared correction (exit 101; /tmp/fn2171-sibling-pattern-red.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations (exit 0; corrected source has 14 integration tests and 2 doctests; /tmp/fn2171-fixes-final-focused.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings (exit 0; /tmp/fn2171-fixes-final-clippy.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target make model-check (exit 0; frozen correction commit 4b04d0a3a7f0f877526be013d9e8b2311f3b2861; /tmp/fn2171-fixes-model-check.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target UMPIRE_INNER_CRATE=umpire-annotations UMPIRE_INNER_SOURCE=umpire/annotations/src/lib.rs make model-time (exit 0; gate 15.496s; inner loop 1.271s; /tmp/fn2171-fixes-model-time.log), flowctl gate classify --base 8baf7c6ef32bad11bccdda9a53c99f2b1d6ea831 (FULL; exit 1; Rust source changed), anchored-text baseline: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations (exit 0; /tmp/fn2171-anchor-baseline.log), anchored-text baseline: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings (exit 0; /tmp/fn2171-anchor-baseline-clippy.log), red reproduction committed as 665e8d533483d34289b45a535e643e4c04e567ea: cargo test -p umpire-annotations --test generation (exit 101; exact ab/bc length4 and exhaustive-domain controls fail, prior five controls pass; /tmp/fn2171-anchor-red.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations (exit 0; 16 integration tests, 2 doctests; /tmp/fn2171-anchor-final-focused.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings (exit 0; /tmp/fn2171-anchor-final-clippy.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target make model-check (exit 0; frozen commit dc071f05c1b216119274a96ad5937d479e56dc6d; /tmp/fn2171-anchor-model-check.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target UMPIRE_INNER_CRATE=umpire-annotations UMPIRE_INNER_SOURCE=umpire/annotations/src/lib.rs make model-time (exit 0; gate 11.653s; inner loop 0.701s; /tmp/fn2171-anchor-model-time.log), flowctl gate classify --base 4b04d0a3a7f0f877526be013d9e8b2311f3b2861 (FULL; exit 1; Rust source changed), formal-fanout correction baseline: cargo test -p umpire-annotations and cargo clippy -p umpire-annotations --all-targets -- -D warnings with CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target (both exit 0; /tmp/fn2171-fanout-baseline.log and /tmp/fn2171-fanout-baseline-clippy.log), catalog helper red 8afe887b9f05d3288e71cef98448678c0f5903b0: 3 failing controls for aliased operands and unrelated-field presence starvation; helper green 02bfd95d48f83ac14ad2a870a6a62ddb28960a4e; logs /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2171-presence/{red-test,green-test,green-clippy,green-fmt}.log, generation red 40240a5c554f7f6ca739984cdf25b17824cd92fe: cargo test -p umpire-annotations --test generation (exit 101; 2 failed and 7 passed; /tmp/fn2171-unique-domain-red.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo test -p umpire-annotations (exit 0; combined 23 integration tests and 2 doctests; /tmp/fn2171-fanout-combined-focused.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target mise exec -- cargo clippy -p umpire-annotations --all-targets -- -D warnings (exit 0; /tmp/fn2171-fanout-combined-clippy.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target make model-check (exit 0; frozen combined commit 00364f8e0f3fbd42ead1fd600989b655bdb005f0; /tmp/fn2171-fanout-combined-model-check.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2171-target UMPIRE_INNER_CRATE=umpire-annotations UMPIRE_INNER_SOURCE=umpire/annotations/src/lib.rs make model-time (exit 0; gate 12.918s; inner loop 0.786s; /tmp/fn2171-fanout-combined-model-time.log), flowctl gate classify --base dc071f05c1b216119274a96ad5937d479e56dc6d (FULL; exit 1; Rust source changed), Conductor integrated cdf7f3c9e6766aadeb7aab6cadb89b15b40616c6: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check (exit 0; .flow/tmp/fn2171-review-fixed-integrated-model-check.log), Formal resumed implementation review: SHIP; all three introduced findings fixed; /tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.1.json
- PRs: