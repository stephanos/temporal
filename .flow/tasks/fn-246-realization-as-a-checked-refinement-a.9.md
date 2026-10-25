---
satisfies: [R1, R5]
---
# fn-246-realization-as-a-checked-refinement-a.9 Current-format whole-prefix Run recording, replay and presentation

## Description

Implement current-format Run recording, replay/question/reduce and generic presentation for the default whole-prefix witness contract after .8's current guest/Host integration. Historical-format loading and legacy delta readers are removed from this task's scope; Go continues to transport generic proof data without Temporal meaning.

**Size:** M
**Files:** Runner session/admission/replay/question/reduce, current protocol/bindings, report consumers and focused current-format fixtures/documentation.
**Touches:** [model/umpire/hosts/go/**, model/umpire/protocol/proto/**, model/umpire/protocol/runtime/tests/**]

## Approach

1. Record the actual validated current contract from .8 in the new Run format, together with the loaded guest identity. Layouts, tags and generated APIs may change; migrate all current consumers together. No old guest/recording relabeling or backward reader is required.
2. Admit only the supported current record/guest contract and matching model_hash. Replay/question/reduce reject incompatible or malformed input before Driver effects, without supporting a legacy fallback or invoking an unrecorded probe. Every actual Model call, including Reset, remains represented in the recording.
3. Treat each successful Explanation as a canonical complete witness from Reset for the entire actual observation prefix and callback/proposal/evidence ordering. Later witnesses may choose a different compatible hidden history while satisfying every prior observation. Earlier reports remain immutable existential-prefix claims; presentation never concatenates them as deltas.
4. Preserve deterministic exact event/response replay within the current format on the actual .1/.7/.8 singleton runtime. Add multi-observation and callback-delivery fixtures, including Current after a nonempty history and a latest-observation-only mutant; test question prefixes/reduce, contract/hash mismatch, malformed/crossed/redacted evidence, failed publication/retry and coherent event numbering. Remove legacy refusal normalization instead of maintaining parallel old semantics. The generic changed-canonical-past fixture requires .10's compatible frontier and is implemented with its runtime/record/replay tests there, then rerun in .6; no changed-past/ambiguity proof is claimed at .9 closure.

## Investigation targets

**Required:**

- Current Run/Explanation schema and .8's validated ordinary Reset contract.
- Runner `session.go`, `replay.go`, question/reduce and model-hash admission.
- Current runner mode tests, report/progress/render consumers and generic record fixtures.
- .1/.7/.8's actual singleton whole-prefix runtime and transaction controls; .10 owns changed-canonical-past/frontier integration tests after this task.

## Verification

Focused generic Go runner/Host tests with test_dep, current-format record/render comparisons and runtime/guest fixture parity. Regenerate current schemas and fixtures as needed. No old recording, old reader or old rendering compatibility gate remains.

## Acceptance

- [ ] Current Run records the actual admitted contract and exact guest identity; malformed/incompatible records and contract/hash mismatches fail without invented evidence or Driver effects.
- [ ] Replay/question/reduce deterministically reproduce current recordings, including recorded Reset and all subsequent calls. Current publication/retry, redaction/crossed evidence and event controls pass.
- [ ] On the actual singleton runtime, every latest witness validates the whole prior observation prefix, including Current after a nonempty history and callback delivery; previous reports remain immutable existential claims. Latest-only, wrong-earlier-prefix and delta-concatenation mutants fail. Changed-canonical-past runtime/record/replay controls remain explicitly owned by .10 and the integrated .6 gate.
- [ ] Current schema/bindings/presentation migrate coherently and the format break is documented. Go contains no Temporal policy; legacy readers, delta mode and old refusal normalization are unnecessary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
