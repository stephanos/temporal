# Task 63 investigation

BASE is c58418ee1dc42f0760824f798e5714477cfab0d2. The admission confines product edits to runner.go, new runner_local.go and additive runner_local_test.go. Existing comments stay with their code. No dependency seam, validation policy, storage encoding or existing fixture changes are admitted.

Similar code search: reuse newShardedSeedController and synchronizeCampaignStatistics in runner/campaign.go; reuse completedExecution/assessWorld/assessCompletion in completion.go and the existing retention owner. Extend the local orchestration using a private localCampaign value; the choice/simulation orchestration keeps its existing separate transaction owner. No new interface or execution dependency is required.

The extraction will name request validation, campaign opening, preparation, seed restoration/scheduling, completion assessment, successful/failed publication and final summary phases. The controller retains every Next/Complete/Stop/Finalize call and its counter/failure-policy semantics. State grows only by the existing campaign locals; no selection-sized storage or payload copy is added.

Fixed pre-edit expectations: runLocal must shrink from 780 lines and every extracted orchestration function must remain at most 150 lines. Preparing progress failure and parent cancellation must short-circuit before preparation/execution, retain their host reason/error identity, and publish no campaign. Later phase controls must retain cancellation before supervision/evidence, supervision before prepared integrity, World evidence before outcome, and host failure before final verification/publication. These controls do not authorize changing inherited outcomes.

The unchanged complete ordinary Runner baseline remains RED: 345 passing, 288 failing, 12 skipped cases (82/123/12 at top level). Missing installation/preparation means many preexisting orchestration fixtures do not reach execution. Direct extracted-phase controls will supply bounded evidence, not a full entrypoint or native pass. The initial manifest included present source but emitted errors for sparse tracked paths; retain its receipt and audit consumed inputs separately rather than repeat an unchanged failing suite.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)
Tier: session (jev-unavailable(no_key)); explicitAGENTSimplementermodelretained
