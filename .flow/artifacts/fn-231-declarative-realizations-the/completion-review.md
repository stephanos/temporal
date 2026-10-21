I’ll read the spec and task requirements, then check each against the diff and focused verification evidence.

The activity rewrite uses explicit phase projection and separates observed responses, incoming callbacks, and outgoing requests. I’m checking the coverage report’s catalog source, field-presence guards, and negative controls next.

The coverage report uses the same field classifications as the runtime guards, and its added-field control rebuilds from a mutated descriptor and catalog. A fresh focused Cargo run was blocked by the read-only build-cache lock; I’m checking retained binaries and preservation evidence, with that limit noted.

The retained binaries passed 32 focused checks across projection, reply packing, field presence, enum admission, refusal ordering, and the generic boundary. The diff leaves Model and existing golden files unchanged. I found no completion gaps; the fn-230 retirement note is the one unrelated change to acknowledge.

## Requirements Extracted

1. Rewrite activity as `#[realization(activity)]` with `#[observe]` and `#[answer]` handlers.
2. Provide Temporal-owned macros and `Step<A>` / `Reply<A>` vocabulary.
3. Support Projection, Step and explicitly mapped state returns, preserving Current, terminal steps and unexplained phases.
4. Generate method dispatch, decoding and Unsupported refusals while preserving refusal ordering and runtime responsibilities.
5. Keep callbacks and outgoing SDK replies partial, packing each reply envelope exactly once.
6. Validate field paths through existing catalog declarations and protobuf edges, with tracked inputs and acyclic dependencies.
7. Check unsupported fields using decoded optional/oneof presence, including opaque fields; document scalar-default limits.
8. Stop traversal at opaque or classified parents and retain ignore reasons.
9. Generate a stable activity coverage golden covering all five contexts, classifications, reasons and unhandled Model-action RPCs.
10. Prove an added catalog field fails the actual coverage golden.
11. Reject duplicate handlers, unclassifiable returns, unknown paths and generated-enum wildcard arms with pinned compiler diagnostics.
12. Reject empty ignore reasons and overlapping or opaque-descendant paths.
13. Make helper enum conversions exhaustive while preserving existing mappings and byte-policy fallbacks.
14. Pin the formatted activity macro expansion.
15. Preserve activity, wire, slice and fn-218 cover behavior byte-identically.
16. Leave Model and generic core unchanged; pass the source boundary check without ABI/protocol changes.
17. Document the surface, return rules, coverage format and complete-file before/after counts in DESIGN.md.
18. Reserve `#[propose]` without implementing proposal behavior.
19. Retain budgets and dependency constraints, refresh guest bytes and document declaration-analysis limits.

## Coverage Verification

1. **COVERED** — `model/temporal/feature/activity/temporal.rs:38`, `:58`, `:123`.
2. **COVERED** — `model/temporal/realization/macros/src/lib.rs:22`; `model/temporal/realization/src/lib.rs:7`, `:16`.
3. **COVERED** — `model/temporal/annotate/syntax/src/realize.rs:217`, `:382`; explicit activity mapping at `model/temporal/feature/activity/temporal.rs:111`; Step control at `model/temporal/realization/tests/declarative.rs:34`.
4. **COVERED** — `model/temporal/annotate/syntax/src/realize.rs:346`, `:367`, `:402`, `:878`. Existing runtime ownership remains unchanged.
5. **COVERED** — `model/temporal/annotate/syntax/src/realize.rs:458`, `:489`; decoded-byte controls at `model/temporal/realization/tests/declarative.rs:145`.
6. **COVERED** — `model/temporal/annotate/syntax/src/realization.rs:162`, `:250`, `:267`; freshness tracking at `model/temporal/realization/macros/build.rs:13`. Dependencies reuse the API-independent syntax crate.
7. **COVERED** — `model/temporal/annotate/syntax/src/realization.rs:377`, `:409`, `:443`; controls at `model/temporal/realization/tests/declarative.rs:82`, `:114`; limits documented in `model/temporal/realization/README.md:29`.
8. **COVERED** — `model/temporal/annotate/syntax/src/realization.rs:328`, `:365`; coverage contract at `model/temporal/testing/realization.rs:20`.
9. **COVERED** — `model/temporal/testing/realization.rs:19`, `:46`; registered consumer at `model/temporal/testing/fixtures.rs:73`; golden at `model/temporal/feature/activity/expected/realization.snap.txt:1`.
10. **COVERED** — actual descriptor/catalog mutation and golden-failure assertions at `model/temporal/testing/realization_control.rs:18`, `:99`, `:109`; retained output in `.flow/artifacts/fn-231-declarative-realizations-the/task-2-added-field-control.txt:1`.
11. **COVERED** — `model/temporal/annotate/syntax/src/realize.rs:223`, `:736`, `:827`; compiler fixtures registered by `model/temporal/realization/tests/realization_admission.rs:4`, with corresponding `.stderr` pins.
12. **COVERED** — `model/temporal/annotate/syntax/src/realize.rs:91`; `model/temporal/annotate/syntax/src/realization.rs:312`; grouped compile controls at `model/temporal/realization/tests/ui/realization_fields.rs:1`.
13. **COVERED** — exhaustive conversion at `model/temporal/feature/activity/phase.rs:17`, `:39`; upstream-addition control at `model/temporal/realization/tests/ui/phase_enum_addition.rs:1`; byte fallback retained at `model/temporal/feature/activity/temporal.rs:193`.
14. **COVERED** — prettyplease/insta assertion at `model/temporal/realization/macros/src/lib.rs:156`; committed activity expansion snapshot.
15. **COVERED** — existing golden files and controls are unchanged in the reviewed range. Retained realization/helper binaries passed; recorded slice, complete gaps/privacy and report-byte evidence appears in `.flow/artifacts/fn-231-declarative-realizations-the/task-2-verification.md:43`.
16. **COVERED** — base-to-head comparison confirms no Model or generic-core changes. Retained `generic_source_boundary` passed; no envelope or ABI source changes landed.
17. **COVERED** — `model/DESIGN.md:571`. Measured counts agree: activity 110→197 lines; DESIGN remains 998 lines.
18. **COVERED** — `model/temporal/realization/macros/src/lib.rs:79`; reservation check at `model/temporal/annotate/syntax/src/realize.rs:811`.
19. **COVERED** — one added proc-macro crate, existing dependency versions reused, unchanged limits, refreshed guest matching the recorded SHA-256, and bounded-analysis limitations in `model/temporal/realization/README.md:36`.

Verification limitation: fresh Cargo execution stopped at the read-only build-cache lock. Retained binaries passed 32 focused checks; this is not a fresh-build pass. Full gates were not rerun during this review.

## Reverse Coverage (untraced changes)

`.flow/artifacts/fn-230-reproducible-guest-builds-a-system/retirement-checkpoint.md` — **UNRELATED_CHANGE** — records predecessor cache retirement; it does not implement an fn-231 requirement.

All remaining changed files trace to the requirements above, including build wiring, compiler controls, snapshots, documentation and verification artifacts.

## Gaps Found

None.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Activity attributes, generated dispatch, compiler pins, unchanged goldens and recorded live verification |
| R2 | met | Registered coverage consumer, five-context golden and real added-field failure control |
| R3 | met | Unchanged Model/generic core and passing retained source-boundary check |
| R4 | met | Formatted expansion snapshot and DESIGN section with verified counts |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - All requirements implemented (R-IDs all met or deferred)
