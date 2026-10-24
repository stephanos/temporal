---
satisfies: [R1, R5]
---
# fn-246-realization-as-a-checked-refinement-a.9 Versioned whole-prefix Run recording, replay and generic presentation

## Description
Implement generic Run version recording, admission/replay and presentation for the selected whole-prefix witness contract after immutable guest version discovery. This is protocol mechanics for R1/R5, with no Temporal meaning in Go.

**Size:** M
**Files:** Runner session/admission/replay/question/reduce and version-aware report consumers, focused legacy/prefix/unknown-version tests and protocol/host documentation.
**Touches:** [model/umpire/hosts/go/runner/**, model/umpire/hosts/go/README.md, model/umpire/protocol/proto/README.md, model/umpire/protocol/runtime/tests/**]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors, golden hashes and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Preserve the meaning of Run.protocol_version as the admitted Model protocol semantic version; tag2 and all message layouts stay fixed. Session records the loaded Host's discovered value, not a global constant/caller request. Preserve existing exported legacy constants where compatibility requires; explicit internal tables distinguish delta1 and prefix2.
2. Resolve replay/question/reduce admission ordering so the loaded guest's discovered semantics and exact model_hash are checked before the first application/Driver call. A fresh Host/session may be constructed to inspect metadata, but no extra Model probe RPC is introduced. Known v1 and v2 each replay only matching guest/version; unknown recorded/guest values and mismatches refuse explicitly. No relabeling old Runs or hashing a new guest as an old one.
3. Define v2 Explanation steps/outcomes as one canonical whole actual observation-prefix witness from Reset, including required callback/proposal ordering. Each later successful report must validate all earlier immutable observations. Prior reports remain immutable existential prefix claims and need not be prefixes of later reports. Generic rendering/consumers choose semantics from the version and never concatenate v2 reports as deltas. v1/Exact/Kettle behavior and old event bytes remain unchanged.
4. Keep exact proto.Equal event/response replay. Pin retained old v1 guest bytes/recordings by hash; add a generic prefix2 fixture with deliberate changed canonical past and an invalid latest-observation-only witness mutant. Keep existing legacyRefusalVerdict presentation compatibility explicitly confined to its version1 contract rather than using normalization to hide new-prefix/version mismatches. Test question prefixes/reduce, unknown/mismatched versions, redacted/crossed evidence, failed publication/retry and unchanged numbering. Go transports versions and opaque evidence only.

### Investigation targets

**Required:**

- `model/umpire/protocol/proto/umpire/run/v1/run.proto:14` and `model/v1/model.proto::Explanation` for unchanged carriers.
- `model/umpire/hosts/go/runner/runner.go:21-22`, `session.go:38` for current constant stamping.
- `model/umpire/hosts/go/runner/replay.go:19-20` and question/replay/reduce admission paths for version/hash ordering.
- `model/umpire/hosts/go/runner/modes_test.go:161,406` and existing literal/legacy record tests.
- `model/umpire/hosts/go/runner/{progress,command}.go` and store/render consumers for avoiding prefix concatenation or changed v1 output.
- Version-admission task's immutable Host semantic-version getter and production prefix fixture.

### Verification

From generic Go host module run focused runner/host tests with test_dep, including retained v1 guest+Run and new prefix2/unknown-version controls; run exact event/record rendering comparisons. Native/guest source proof fixture parity accompanies the runtime task. No schema generation, Driver call or Temporal policy implementation is needed.

## Acceptance
- [ ] Run.protocol_version comes from actual admitted guest metadata, retaining tag2/layout/model_hash meaning. No caller/global version stamp can label an old guest2 or unknown guest as known.
- [ ] Matching retained v1 guest/recordings replay unchanged; prefix2 replays whole actual prefixes; unknown/malformed/mismatched guest/Run versions reject before application/Driver I/O. Question/reduce share the same admission controls and event numbering.
- [ ] Latest v2 witness satisfies every prior actual observation from Reset, while prior reports remain immutable existential claims. Prefix lists are never concatenated as deltas; latest-only/rewritten-prefix and mismatched-version mutants fail.
- [ ] Publication/retry, redaction/crossed evidence, record/hash and exact proto.Equal event controls pass. Go code contains no Temporal evidence meaning; old v1/Exact/Kettle output bytes remain pinned.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
