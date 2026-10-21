# Task 2 verification

Task: fn-231-declarative-realizations-the.2. Task base: e114c16910a388e4dec8b873d0c7476998d17301. Parent/spec base remains b9f26782e39f815d740ea4b89925a723303c861d. The predecessor's committed task closure is 65044c1fda0c520a18036b5c062edef42a53c0d7.

## Scope and baseline

The Temporal-owned coverage consumer renders the actual generated Realization::REALIZATION metadata used by runtime guards. It derives the action universe from the existing API RPC_BINDINGS export. Reports distinguish observed responses, incoming callbacks and outgoing partial SDK requests. They separately name missing observation handlers and missing realization contexts.

The only dependency changes are existing workspace dependencies on the Temporal fixture consumer. The parser/descriptor graph, Model, runtime guards, generic core, protocol, ABI, upstream versions, Go SDK panic behavior and existing goldens remain unchanged. Foreign MILESTONES.md and fn233 source plans/tasks are excluded from staging. No parent completion or successor admission belongs to this worker.

Before edits, actual five-package all-feature native tests, make model-check, make umpire-guest, make umpire-check-model-slice and make umpire-check-model-gaps each returned 0. Logs are .flow/tmp/fn2312-baseline-{native,model-check,guest,slice,gaps}.log. The live slice took 1.118s and gaps 353.350s. Gaps retained 2336 unique RUN/PASS pairs, privacy and byte-identical report checks. The prior task's live proof remains historical evidence and was not relabeled as this run.

The full-gate receipt probe refused reuse because MILESTONES.md was dirty. No reusable green receipt or skipped gate is claimed. Serial commands use CARGO_BUILD_JOBS=4, the leased /home/agent/.cache/codex-build/fn2301-unwind/cargo target and matching go cache. No limits changed.

## Focused evidence

The new realization snapshot seed first failed the actual fixture runner with one collected failure reporting an expected file without a consumer. This ownership RED is recorded in fn2312-coverage-red.log. After registration, the seed lacked snapshot metadata and failed before comparison in fn2312-consumer-red.log; that bootstrap failure is not the added-field acceptance proof. The first new snapshot was generated in fn2312-coverage-update.log. Existing expected payloads and metadata were unchanged.

The new grouped compile fixture pins actual blank-ignore, parent/child overlap and opaque-descendant refusals. The first native run differed only in the hand-authored diagnostic locations; fn2312-quick-native.log retains expected/actual output. The pins were corrected to the compiler's actual locations without changing the checks or source refusal behavior. Existing unknown-path and type-identity controls remain intact.

The physical activity source is 110 lines at the parent base and 197 at the task checkpoint, including blanks/comments. DESIGN records the 87-line increase and measures 998 lines after relevant text replacement. No source-size reduction is claimed.

The installed standalone worker.md is absent. Supplied worker role instructions plus the complete flow-next-work skill and phases govern this installation fallback. There are no glossary matches or feature map. Relevant memory entries about independent completeness universes and proved type identity were read before implementation.

## Actual added-field control

The isolated control changes the real proto/api.binpb descriptor and actual activity api.rs catalog declaration for RequestCancelActivityExecutionResponse. The existing API build, shared descriptor admission and realization macro recompile from that source. The registered fixture then fails its unchanged realization snapshot with exit 101. This is not a manually constructed Context or synthetic renderer failure.

The first control wrapper expected the incorrect snapshot name 'realization.txt' and a textual field diff from an external binary snapshot. Its child reached the real 'realization' snapshot failure, but the wrapper failed. That setup mismatch is retained in fn2312-added-field-control.log and is not claimed as a passing control.

The corrected child uses INSTA_UPDATE=new to retain the actual pending result while failing the assertion and preserving the committed old payload. The control requires the precise realization snapshot assertion, nonzero child exit, unchanged old payload, and the exact sole report delta `(no fields)` to `unsupported coverage_control` in the real Cancel response context. It preserves that pending report outside the temporary worktree and removes its own pending files, restores its own catalog mutation, and normally removes the worktree and integrated branch. The main descriptor/catalog and every existing golden remain untouched.

The control passed once in 19.99s, then passed again in 19.17s after strict clippy required extracting descriptor mutation into a cohesive helper. No lint suppression or check weakening was added. The final command is `mise exec -- cargo test -p umpire-temporal-tests --test fixtures added_catalog_field_fails_the_real_coverage_golden -- --ignored`, recorded in fn2312-added-field-final.log. Both temporary lanes and their branches are absent. The retained actual report is task-2-added-field-control.txt with SHA256 df3aa22be0426960c3e9f1499c25b700eabdd8dcef6ec384c2dc1a36ff0a2e33. The unchanged real golden is SHA256 88a00db7d38337b3ea57a358f938eb4cbcae5120e73b6f4f2ad7470f3ff68c8e.

Main-checkout five-package native tests returned 0 after the first control. Strict clippy initially reported the test's length, then returned 0 after the helper extraction. Ordinary fixture verification returned 0 with seven passes and the explicitly ignored isolated rebuild control. A final owner native/clippy/format/guest sequence runs after the last control so no mutable-control binary can be mistaken for a main-source verification.

## Remaining verification at this checkpoint

Final owner Quick/guest/live controls, two quality axes, formal review and final full gates remain pending. No verdict or task completion is claimed here.
