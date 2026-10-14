---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.3 The signature constructs its messages: actions, inputs, entities, observations, channels, assumptions, holes, Limits, types

## Description
Activate signature collection from constructed declarations after full .2 and the .10/.11/.12 framework source checkpoints. .10 retains native finite schemas; .11 constructs Action/Input/Channel contributions in isolation; .12 retains and converts composed action references without complete composition export. Reuse those capabilities, complete entities/observations, assumptions, holes, Limits and machine-held type/Inbox ownership contributions, then make the driver consume the single native signature producer. Complete Assumption conversion must retain explicit-empty versus default-name provenance and consume lossless ordinary/composed fairness chains. Retire the corresponding Declarations reader arms only at this integration boundary. Full R3/R4/R5 acceptance, Cases/receipts/backend verification and all original dependency obligations remain here. Do not depend on full .5 for composed references, since that would cycle through .4.
The final name-provenance audit must include explicitly empty action names. The old reader at model/framework/export/Declarations.scala:88 preserves action("", actor), while the staged native converter at model/framework/dsl/Signature.scala:83 selects declaration spelling whenever decl.name is empty. Both action factory overloads currently store the same empty string, so conversion needs retained omitted-versus-explicit provenance rather than name.nonEmpty. This source-read finding needs a compiled old/native regression before a fix or parity claim. It remains part of full R4 integration, not .12's composed-reference implementation surface.
## Acceptance
- Every signature kind in the IR comes from constructed values; the corresponding lifter arms are deleted (R3).
- Names and derived IDs equal today's (R4).
- Types equal today's as messages (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
