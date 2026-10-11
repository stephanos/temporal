---
satisfies: [R1, R6]
---
# fn-140-one-sentence-witness-queries-with.7 Validate exact composed witness selectors across deferred Go consumers

## Description
**Size:** M
**Files:** existing `tools/umpire/{ir,check,lint,export}` selector consumers and tests.
**Touches:** [tools/umpire/ir/validate_keys.go, tools/umpire/ir/*selector*.go, tools/umpire/ir/admission_test.go, tools/umpire/check/claims.go, tools/umpire/check/*composition*test.go, tools/umpire/check/transition_test.go, tools/umpire/lint/holes.go, tools/umpire/lint/holes_test.go, tools/umpire/export/composed.go, tools/umpire/export/slice.go, tools/umpire/export/quint.go, tools/umpire/export/*test.go]

Owner-deferred non-Scala half split from the original fn-140.2. This preserves, rather than waives, its common exact-selector proof. Do not admit while MILESTONES limits work to Scala/model. Consume the committed encoding and source fixture handoff from .2. Shared production regeneration/full gates/review/live remain at fn-123.8.

### Approach
- Read the source lifter's exact-composed tagged convention in existing Property.when_action. Confirm it cannot collide with admitted legacy action names before using it. No new schema fields/kinds, synthetic Action declarations, general interpreter ClassKey change or widening to all inputs.
- Use one source-aware selector-decoding contract across composition admission, checking, hole/coverage lint and composed/slice/Quint export. The tagged form is legal only on a composition Property. Admission requires exact membership in its independently derived composed class catalog, including member/sync ownership, inputs and binding identity; malformed, blank, wrong-owner, wrong-input and absent keys refuse at the Property source.
- Runtime compares the complete admitted class key by equality, never an action-only prefix. Export exact input guards faithfully or return a source-located UnsupportedError without widening/exporting around it. Preserve every legacy admission diagnostic and obligation; no equality/lint exemptions.
- Prove a witness/core twin whose input-bearing tail follows a different input of the same composed action producing indistinguishable state/facts. Cover own-member and renamed-sync keys, two same-typed members, wrong-owner/input refusals and repeated exact-tail rejection. A seeded all-input selector substitution must fail. Exercise full Go admission/checking, legacy byte/semantics pins, hole/coverage accounting and faithful export/located refusal; lift-only or state/fact-only equivalence is insufficient.
- Keep the ordinary hand-authored core twin and witness on the same encoding, preserving untagged action and machine when_class bytes/meaning. If the committed Scala contract cannot satisfy consumer admission, stop before migration and reconcile that contract without broadening the claim.

### Investigation targets
**Required:** `proto/internal/temporal/server/api/umpire/v1/claim.proto:22-27`; `tools/umpire/ir/validate_keys.go:63-103`; `tools/umpire/check/claims.go:690-709`; `tools/umpire/lint/holes.go:260-295`; `tools/umpire/export/{composed,slice,quint}.go`; fn-140.2 source proof/fixtures.
**Optional:** `tools/umpire/ir/admission_test.go`, composition/check and export tests.

### Quick commands (only once owner lifts deferral)
```sh
mise exec -- go test -tags test_dep ./tools/umpire/ir ./tools/umpire/check ./tools/umpire/lint ./tools/umpire/export
```
## Acceptance
- [ ] All exact-composed selector obligations of the original fn-140.2 are met across admission, checking, lint/coverage, composed/slice/Quint export and source-attributed refusals (R1/R6).
- [ ] Complete input-bearing tails, renamed sync/member ownership, wrong owner/input, legacy-byte/semantics pins and seeded all-input-selection rejection are proved end to end; lift-only equivalence is not accepted.
- [ ] The actual Scala encoding contract from fn-140.2 matches every consumer, and ordinary core twins have identical meaning with no schema/kind/ClassKey widening.
- [ ] This task is admitted only after the owner reopens the deferred non-Scala lane; full shared gate/review/live obligations remain tracked at fn-123.8.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
