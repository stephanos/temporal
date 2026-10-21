# Task .4 external consumer integration authorization

The first full `make model-check` after creating the explicitly reviewed live golden failed with `expected file has no consumer: .../features/activity/expected/gaps.txt`. The native fixture runner owns a closed set of rendered artifacts. The approved task requires a direct live-server `gaps.txt` at that same expected directory, consumed by the actual Go `TestTestpilotModelGaps` gate.

The conductor explicitly authorized this exception before edits outside the original Touches.

> Explicit scope exception approved for this task: model/umpire/testing/** solely for the minimal additive ExternalConsumer registration seam and negative controls you proposed. This overrides the dispatch Touches prohibition for that exact integration need, not other generic changes.

The additive declaration assigns one exact fixture/artifact to its named live consumer. Native verification reports that it supplies no live execution or comparison evidence. The actual dedicated Go gate remains responsible for real server probes and golden comparison. Unknown artifacts, missing declarations, collisions, unsafe paths and ambiguous ownership remain failures. Native tests never render, copy into actual results, or update the external golden.

Original full-gate failure is captured in `.flow/tmp/fn2174-final-model.log`. Generic integration RED/GREEN evidence is recorded in `.flow/tmp/fn2174-external-consumer-lane.md`.

Three externally owned planning files appeared during execution. The owner confirmed another session edits them. The conductor approved `git add -A` followed by unstaging exactly `MILESTONES.md` and `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.6.{md,json}` without changing their working-tree contents. Task .4 excludes that planning work from its source commit and review.

The implementation retained the original rule that every Model fixture selects a native consumer. A negative control first demonstrated that an external-only fixture could otherwise bypass that rule; restoring the unconditional native selection check made it fail correctly. Generic runner regression tests and Clippy then passed. The actual live gate had already completed all 62 contextual probes and compared the reviewed golden.

The externally owned planning exclusion was extended to `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.7.{md,json}` when those appeared. All five external paths remain untouched and excluded from staging.

The standards audit requested splitting the additive Probe messages into a dedicated schema. The conductor explicitly approved this exact additional scope exception before edits:

> Approved exact additional scope exception: model/scripts/proto-temporal-go.sh solely to add probe.proto to existing generator inputs (and matching freshness input list only if samefile requires it). Preserve pinned tooling/options/package/fieldnumbers and freshness enforcement. Record this authorization alongside task-4-scope-exception.md, include audits/reviewcontext. No broader scripts changes authorized.

The conductor subsequently approved exactly one `cargo:rerun-if-changed` entry for `probe.proto` in `model/temporal/api/build.rs`, and reuse of `model/temporal/api/tests/build_inputs.rs` to assert that both schema files are actual tracked, existing inputs. No other API source or test behavior changes were authorized. This prevents a later probe-only schema edit from reusing stale Rust generation.
