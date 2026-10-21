I’ll read the spec and task requirements, trace them to the implementation, and check the changed files for untraced scope.

The current spec replaces the earlier validator-source walk with black-box probes and replaces `annotate!` with derive mirrors. I’m checking those amended requirements, including exhaustive admission, actual SDK transport validation, and how the probe accounts for untested mutations.

The live probe golden accounts for every strict expectation and records generic mutations that were not sent, with dated dispositions for the reviewed `Unannotated` findings. I’m checking admission controls and scope tracing next. Cargo tests cannot start in this read-only sandbox because Cargo cannot open its build lock.

The activity Model, its tests, and its report, rows, states, coverage, and expansion goldens are byte-for-byte preserved across the rename. The gap golden has 79 strict cases, all confirmed, and 2,238 generic cases; all 217 `Unannotated` rows have dated dispositions. Formatting passes. The range also contains interleaved work from other specs, which I’ll list separately in reverse coverage.

## Requirements Extracted

1. Expose system-independent `Catalog`, kinds, tags, consequences, rules, expectations, validation, seeded filling, violations and gap reports.
2. Implement the specified kind vocabulary and native constraints using protovalidate names and semantics.
3. Support compatible tags, concrete consequences and Temporal-owned custom tags.
4. Implement all five presence operators: `required`, `any_of`, `one_of`, `requires`, `excludes`.
5. Reject incompatible kinds, contradictory roles, ignored-field rules, unexplained ignores and missing request expectations at compile time.
6. Fill deterministically, preserve supplied/decided values, and report impossible generation or missing external secrets.
7. Provide secret redaction and value-free validation, generation and probe diagnostics.
8. Use readable derive mirrors with exact protobuf field names, kind types and the two `rule`/`mirror` namespaces.
9. Share one parser between derive expansion and descriptor scanning; track catalog inputs.
10. Supply typed projections and fallible setters that preserve fields on malformed encoded mutations.
11. Enforce exhaustive fields, nested messages and oneof alternatives.
12. Enforce reviewed enum inventories and reject unknown runtime integers.
13. Generate RPC identities and require exhaustive service classifications and Model action bindings.
14. Enforce protobuf declaration order and test added fields, values, alternatives, methods and actions.
15. Attach faithful observable expectations to every request rule and kind constraint.
16. Construct isolated strict probes in Rust using valid baselines and real server prerequisites.
17. Check rejection status, message pattern and requested detail type; classify intentional acceptance correctly.
18. Render deterministic gap goldens and fail unacknowledged `Mismatch`, `Accepted` and `Unannotated` outcomes.
19. Preserve descriptions, inherited patterns and counted loose expectations without exposing values.
20. Generate independent black-box mutations for every request field; remove validator-source mappings and tooling.
21. Account explicitly for inapplicable, refused and unchanged mutations; reject incomplete or crossed observations.
22. Cover the required activity messages, consumed nested fields and actual slice RPC traffic.
23. Validate actual SDK requests before transport and successful responses afterward; retain background failures and report RPC coverage.
24. Preserve supplied identifiers, retry stability, empty polls, response bytes and independent Model state.
25. Consume Visibility/Immutable/Monotone consequences when the activity catalog declares them.
26. Provide shared foundation catalogs, explicit limits, supplemental raw-response checks and the documented crate-splitting seam.
27. Preserve Model behavior, existing protocol numbers, ABI contracts and independent goldens.
28. Integrate live gaps into Testpilot, retain native/external golden ownership, and record verification and feedback timings.
29. Document layering, admission, authoring, API updates, gap findings and system-free tags.
30. Complete the singular `feature/` migration without stale operational references.
31. Preserve upstream protos and Testpilot schemas; introduce no external validation engine, new third-party dependency or automatic Model transitions.

## Coverage Verification

1. **COVERED** — Generic API and exports: `model/umpire/annotations/src/lib.rs:1`, `model/umpire/annotations/src/catalog.rs:564`.
2. **COVERED** — Kinds and constraints: `model/umpire/annotations/src/kind.rs:105`; positive/negative controls: `model/umpire/annotations/tests/catalog.rs:87`.
3. **COVERED** — Consequences and tag contracts: `model/umpire/annotations/src/kind.rs:252`; custom-tag controls: `model/temporal/annotate/tests/controls.rs:1`.
4. **COVERED** — Presence algebra and evaluation: `model/umpire/annotations/src/catalog.rs:82`; controls: `model/umpire/annotations/tests/catalog.rs:203`.
5. **COVERED** — Shared admission: `model/temporal/annotate/syntax/src/derive.rs:397`; compile-fail suite: `model/temporal/annotate/tests/ui.rs:4`.
6. **COVERED** — Transactional seeded fill: `model/umpire/annotations/src/catalog.rs:851`; role/secret controls: `model/umpire/annotations/tests/catalog.rs:535`.
7. **COVERED** — Redaction: `model/umpire/annotations/src/catalog.rs:601`; report-copy controls: `model/temporal/realization/tests/catalog.rs:331`; privacy canary: `tests/testpilot_model_gaps_test.go:104`.
8. **COVERED** — Exactly two derive helpers: `model/temporal/annotate/macros/src/lib.rs:4`; actual mirrors: `model/temporal/feature/activity/api.rs:26`.
9. **COVERED** — Shared parser: `model/temporal/annotate/syntax/src/lib.rs:17`; recursive tracked inputs: `model/temporal/api/build.rs:80`.
10. **COVERED** — Typed setter contract: `model/temporal/annotate/src/views.rs:98`; malformed mutation and propagation controls: `model/temporal/annotate/src/lib.rs:99`.
11. **COVERED** — Exhaustive generated patterns/matches: `model/temporal/annotate/syntax/src/derive_expand.rs:17`; nested descriptor coverage: `model/temporal/annotate/syntax/src/descriptor.rs:140`.
12. **COVERED** — Reviewed inventory: `model/temporal/foundation/enum_inventory.rs:1`; runtime validity bridge: `model/temporal/annotate/src/views.rs:108`.
13. **COVERED** — Generated service identities/bindings: `model/temporal/api/services.rs:11`; exhaustive classification and action binding: `model/temporal/feature/activity/api.rs:497`, `model/temporal/feature/activity/api.rs:766`.
14. **COVERED** — Declaration-order checks: `model/temporal/annotate/syntax/src/descriptor.rs:200`; descriptor mutations: `model/temporal/annotate/syntax/tests/descriptor.rs:194`; generated compiler controls: `model/temporal/api/tests/services.rs:124`.
15. **COVERED** — Actual request expectations: `model/temporal/feature/activity/api.rs:294`, `model/temporal/foundation/api.rs:67`; missing-expectation controls: `model/temporal/annotate/syntax/tests/admission.rs:147`.
16. **COVERED** — Strict inventory, baseline construction and witness checks: `model/temporal/realization/src/probes.rs:67`; real prerequisite dispatch: `tests/testpilot_model_gaps_test.go:364`.
17. **COVERED** — Status/pattern/detail matching: `model/temporal/hosts/go/probes.go:165`; outcome classification: `model/umpire/annotations/src/gap.rs:155`.
18. **COVERED** — Failure policy: `model/umpire/annotations/src/gap.rs:204`; dated exact-identity dispositions: `model/temporal/realization/src/probes/dispositions.rs:13`; golden comparison: `tests/testpilot_model_gaps_test.go:482`.
19. **COVERED** — Safe metadata rendering: `model/umpire/annotations/src/gap.rs:228`; aggregate loose count: `model/temporal/realization/src/probes/report.rs:109`.
20. **COVERED** — Descriptor-driven mutations: `model/temporal/api/wire_schema.rs:49`; generic inventory: `model/temporal/realization/src/probes/generic.rs:25`; retired syntax rejection: `model/temporal/annotate/syntax/tests/admission.rs:125`. No `reverse_coverage*.go` remains.
21. **COVERED** — Explicit applicability and canonical correlation: `model/temporal/realization/src/probes/generic.rs:145`, `model/temporal/realization/src/probes/generic.rs:235`; controls: `model/temporal/realization/tests/probes.rs:65`.
22. **COVERED** — Activity mirrors: `model/temporal/feature/activity/api.rs:26`; foundation messages: `model/temporal/foundation/api.rs:16`; recursive inspection: `model/temporal/annotate/syntax/src/derive_expand.rs:355`.
23. **COVERED** — Actual unary interceptor, retained failures and required RPC coverage: `tests/testpilot_model_annotations_test.go:40`, `tests/testpilot_model_annotations_test.go:103`; SDK installation: `tests/testpilot_model_slice_test.go:143`.
24. **COVERED** — Transport preservation: `model/temporal/feature/activity/api_transport.rs:71`; retry and response controls: `tests/testpilot_model_annotations_test.go:163`, `tests/testpilot_model_annotations_test.go:225`; frozen-state control: `model/temporal/realization/tests/annotations.rs:54`.
25. **COVERED** — These consequences are available, but the activity catalog declares none requiring additional judgment behavior. This limitation is explicit in `model/DESIGN.md:656`.
26. **COVERED** — Foundation catalogs: `model/temporal/foundation/api.rs:16`; limits: `model/temporal/foundation/dc/mod.rs:26`; raw-response link: `model/temporal/foundation/api.rs:76`; orphan-rule guidance: `model/temporal/annotate/README.md:109`.
27. **COVERED** — Annotate uses method 7 while reserving 6: `model/umpire/proto/umpire/model/v1/model.proto:15`; stateless dispatch: `model/temporal/realization/src/annotations.rs:25`. Direct base/head comparisons confirm identical Model, tests, coverage, report, rows, states and expansion golden bytes after relocation.
28. **COVERED** — Live gate: `Makefile:575`; external golden ownership: `model/temporal/testing/fixtures.rs:28`, `model/umpire/testing/src/runner.rs:88`; retained gate/timing evidence: `.flow/artifacts/fn-217-annotation-catalog-over-the-temporal/task-6-post-review-root-verification.md:11`.
29. **COVERED** — `model/DESIGN.md:572`, `model/README.md:210`, `model/temporal/annotate/README.md:1`.
30. **COVERED** — New paths and fixture discovery: `model/temporal/realization/src/lib.rs:5`, `model/temporal/testing/build.rs:8`. The scoped check found no old-root references in changed active sources or documentation.
31. **COVERED** — No upstream/Testpilot schema changes appear in the diff. Lockfile additions contain only four internal crates. Generic annotation sources contain no Temporal, CHASM, WorkflowService or Testpilot names; the existing boundary gate remains at `Makefile:1472`.

Verification performed here: formatting passed; preservation, boundary and golden-content checks passed. The golden contains 79 strict confirmations, 2,238 generic identities, 754 dispatched generic observations, 1,484 explicit local outcomes and 217 dated `Unannotated` dispositions. Its SHA-256 matches the approved artifact.

Focused Cargo tests could not start because the read-only sandbox prevents opening `.cargo-build-lock`. Runtime pass evidence above is retained implementation evidence, not a fresh test run in this review.

## Reverse Coverage (untraced changes)

The following changes belong to interleaved work outside fn-217. These are scope acknowledgments, not completion blockers. Remaining changed files trace to the requirements above or their implementation evidence.

| File | Classification | Reason |
| --- | --- | --- |
| `.flow/artifacts/fn-229-realization-ergonomics-shared-temporal/completion-review.md` | UNRELATED_CHANGE | Separate realization-ergonomics review. |
| `.flow/artifacts/fn-229-realization-ergonomics-shared-temporal/quality-audit.md` | UNRELATED_CHANGE | Separate realization-ergonomics audit. |
| `.flow/artifacts/fn-230-reproducible-guest-builds-a-system/determinism-checkpoint.md` | UNRELATED_CHANGE | Separate reproducible-build proof. |
| `.flow/artifacts/fn-230-reproducible-guest-builds-a-system/task-1-quality-audit.md` | UNRELATED_CHANGE | Separate reproducible-build audit. |
| `.flow/specs/fn-218-test-space-patterns-cover-and-the-full.json` | UNRELATED_CHANGE | Other-spec review metadata. |
| `.flow/specs/fn-227-parameterized-models-declared-worlds.json` | UNRELATED_CHANGE | Parameterization planning metadata. |
| `.flow/specs/fn-227-parameterized-models-declared-worlds.md` | UNRELATED_CHANGE | Parameterization design clarification. |
| `.flow/specs/fn-229-realization-ergonomics-shared-temporal.json` | UNRELATED_CHANGE | Other-spec execution metadata. |
| `.flow/specs/fn-230-reproducible-guest-builds-a-system.json` | UNRELATED_CHANGE | Reproducible-build planning metadata. |
| `.flow/specs/fn-230-reproducible-guest-builds-a-system.md` | UNRELATED_CHANGE | Cross-platform proof scope adjustment. |
| `.flow/specs/fn-231-declarative-realizations-the.json` | UNRELATED_CHANGE | Future realization planning metadata. |
| `.flow/specs/fn-231-declarative-realizations-the.md` | UNRELATED_CHANGE | Future realization requirements clarification. |
| `.flow/specs/fn-232-linux-macos-guest-byte-identity.json` | UNRELATED_CHANGE | Separate cross-platform proof plan. |
| `.flow/specs/fn-232-linux-macos-guest-byte-identity.md` | UNRELATED_CHANGE | Separate cross-platform proof requirements. |
| `.flow/tasks/fn-218-test-space-patterns-cover-and-the-full.1.json` | UNRELATED_CHANGE | Other-spec task metadata. |
| `.flow/tasks/fn-218-test-space-patterns-cover-and-the-full.2.json` | UNRELATED_CHANGE | Other-spec task metadata. |
| `.flow/tasks/fn-218-test-space-patterns-cover-and-the-full.3.json` | UNRELATED_CHANGE | Other-spec task metadata. |
| `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.1.json` | UNRELATED_CHANGE | Other-spec task metadata. |
| `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.1.md` | UNRELATED_CHANGE | Runtime-constructor implementation evidence. |
| `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.2.json` | UNRELATED_CHANGE | Other-spec task metadata. |
| `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.2.md` | UNRELATED_CHANGE | Shared-helper implementation evidence. |
| `.flow/tasks/fn-230-reproducible-guest-builds-a-system.1.json` | UNRELATED_CHANGE | Other-spec task metadata. |
| `.flow/tasks/fn-230-reproducible-guest-builds-a-system.1.md` | UNRELATED_CHANGE | Reproducible-build implementation evidence. |
| `.flow/tasks/fn-232-linux-macos-guest-byte-identity.1.json` | UNRELATED_CHANGE | Separate proof task metadata. |
| `.flow/tasks/fn-232-linux-macos-guest-byte-identity.1.md` | UNRELATED_CHANGE | Separate proof task requirements. |
| `AGENTS.md` | UNRELATED_CHANGE | Workspace cleanup and disk-management policy. |
| `model/GUEST_BUILDS.md` | UNRELATED_CHANGE | Reproducible-build documentation. |
| `model/scripts/guest.sh` | UNDOCUMENTED_ADDITION | General guest build/freshness command from fn-230. |
| `model/scripts/tests/.gitignore` | LEGITIMATE_SUPPORT | Ignores generated guest-script test output. |
| `model/scripts/tests/guest-parity.sh` | LEGITIMATE_SUPPORT | Tests the separate guest build profile. |
| `model/scripts/tests/guest_profile_test.go.fixture` | LEGITIMATE_SUPPORT | Fixture for guest-profile parity. |
| `model/scripts/tests/test_guest.py` | LEGITIMATE_SUPPORT | Controls the separate guest build command. |
| `model/temporal/realization/src/support.rs` | UNDOCUMENTED_ADDITION | Shared realization authoring helpers from fn-229. |
| `model/temporal/realization/tests/support.rs` | LEGITIMATE_SUPPORT | Controls those shared helpers and refusal precedence. |
| `model/umpire/wasm/src/runtime.rs` | UNDOCUMENTED_ADDITION | Typed candidates and realization API rewrite from fn-229. |
| `model/umpire/wasm/src/lib.rs` | LEGITIMATE_SUPPORT | Exports the new candidate type. |
| `model/umpire/wasm/src/stock.rs` | LEGITIMATE_SUPPORT | Adapts the stock realization to that API. |
| `model/umpire/wasm/tests/constructors.rs` | LEGITIMATE_SUPPORT | Tests the new runtime constructors. |
| `model/umpire/wasm/tests/rejecting.rs` | LEGITIMATE_SUPPORT | Adapts rejection controls to typed candidates. |
| `model/umpire/wasm/tests/stateful.rs` | LEGITIMATE_SUPPORT | Adapts stateful controls to typed candidates. |
| `model/umpire/wasm/tests/support/mod.rs` | LEGITIMATE_SUPPORT | Adapts test realizations to the new API. |

Mixed-purpose files such as `model/Cargo.toml`, `model/DESIGN.md`, `model/README.md` and the activity realization also contain fn-229/fn-230 changes, while independently serving fn-217 requirements.

## Gaps Found

None. No introduced requirement-completion gap survives review.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Exhaustive expansion, reviewed enum inventory, descriptor checker and generated RPC/action compiler controls. |
| R2 | met | Generic vocabulary, shared admission, typed views and trybuild rejection fixtures. |
| R3 | met | Complete strict inventory, live transport, status/pattern/detail classification and 79-case golden. |
| R4 | met | Independent descriptor mutations, explicit local outcomes and dated exact-identity `Unannotated` dispositions. |
| R5 | met | Complete activity/foundation mirrors and actual SDK interception; Visibility condition is inactive. |
| R6 | met | System-independent generic crate, Temporal-owned declarations and retained source-boundary gate. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - All requirements implemented (R-IDs all met or deferred)
