---
satisfies: [R2]
---
# fn-124-shrink-and-simplify-the-umpire-go.2 Remove the duplicate refinement implementation and production APIs only tests call

## Description
Implements R2. Remove model/machine.go:690-851 (Interpreter.refinement, readsAs, seen, noStutter, carrierOf, sameNamedKey, allIn) and the RefineTables-vs-Build comparison in checking.go:569-607, plus Machine.Refinement/Rejected/Transitions plumbing only that comparison reads; keep one behaviour test of the refinement rule against checker/refine.go. Remove production APIs with only test callers: checker QueryCanonical, ComposedStep.Moves/MemberMove, Table.Stuck, Realizer.ClassKey, QueryTotal, explore.RenderTrace, and testpilot PackCaseProtoJSON, Bundle.Handles, replay NewBridge/EvidenceCore/OutsideCore, campaign RunCandidate, evaluation ProfileNames, except where an open task will consume one (model/laws.go ReadLawSidecar/LawViolations is planned for fn-122.5's law lint: keep it and say so). Also drop the stale 'Package testpilot' comment in tools/umpire/lower/lower.go:1 and ownership_test's reference to the missing export/export.go. Gates as in the spec.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
