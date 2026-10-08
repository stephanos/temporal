# R12 review fix 1

The six guides now define the reducer boundary as seven proven shapes with two polled non-nil cases. Unknown readiness, unlisted shapes and selects with three or more polled cases remain expanded. Nil-channel source clauses do not count as polled cases.

The [nil-channel fixture](../../../../../../tools/gomad3/internal/gomadtool/conformance/testdata/select_readiness/main.go) has three source clauses, one for a nil channel and two for non-nil channels. The [shape list](../../../../../../tools/gomad3/choice/no_op_select.go) declares all seven shapes with `PolledCases: 2`, including the nil-channel shape. The [reducer predicate](../../../../../../tools/gomad3/runner/internal/exploration/choice/select_readiness.go) derives that count from the final poll's alternatives. This source proof addresses the introduced P3/R12 in `.flow/review-fanout/f32f5cc32c5747aab7c57fb40ac3ae94/correctness.review.md`; no runtime change or native execution is needed.

`final-docs.json` and `checks.json` in this directory bind the current fix. The parent `final-docs.json`, baseline/final/portable/freeze/conductor command receipts retain their checkpoint `1fcef5ea4bfbf3ec5f7279a8a6c67070d5159a9d` meanings and original exits. `parent-evidence.json` preserves the original committed evidence bytes. The parent `evidence.json` binds the corrected handover/current doc hashes and explicitly links both snapshots.

The fresh doc-only audit and whitespace commands are recorded in `checks.json`. The audit reruns the original full documentation/source/qualification guard with only its output destination routed here, checks the unchanged precise Go input closure against the pre-fix report, and checks the nil-channel fixture and corrected wording. Prior Go tests supply only their unchanged exact scoped-input credit. No whole-module lint, native qualification, measurement or soak credit is added.

Tier: explicit implementer gpt-6.1-sol at high; executed host model metadata unavailable.

stage: impl-review - skipped(policy: conductor resumes the existing canonical review)

Task remains `in_progress`; root owns Git, Flow, milestones and re-review. All worker commands are terminal and the Go/cache lane is released. The root-owned milestone correction is in `milestone-recommendations.md`.
