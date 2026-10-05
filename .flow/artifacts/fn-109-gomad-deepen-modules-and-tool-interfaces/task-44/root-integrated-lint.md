# Root integrated lint and qualification handback

The actual original gate removes exactly the four admitted findings:323→319.
All319 remaining complete diagnostic blocks are byte-identical to task43's
retained original gate. This establishes bounded source progress, not a green
gate or completed acceptance.

[Raw capture](root-integrated.json) records the exact argv, cwd, HEAD, source/tool
hashes, timestamps, complete stdout/stderr, elapsed132.949 seconds and actual
Make exit2. The command used the original comparison base
`951c5516e9e7b3066e7e069adda9565cfd68844c`, pinned golangci/errortype executables,
the original repository config and fix=false. Root serialized the run after all
worker Go/cache handles were terminal; session35098 was polled through exit2.

The actual module router selected55 ordinary host packages and retained tags
`disable_grpc_modules,,test_dep,`. golangci returned1; recursive Make returned2,
the module router reported that failure, and top-level Make returned2. The
integrated errortype stage was not reached. Standalone affected-package
errortype success remains separate evidence and does not repair that gate.

[Diagnostic comparison](root-integrated-comparison.json), produced by
[compare-integrated.mjs](compare-integrated.mjs), verifies complete blocks and
header multiplicities, not just counts. It removes exactly the three S1016
source-record findings and one test-import gci finding. No finding was added or
moved. Residuals:252 errcheck,3 exhaustive,11 forbidigo,53 staticcheck. The three
ST1005 compatibility-pack error strings remain unchanged.

[Root worker verification](root-worker-final-verification.json), produced by
[verify-evidence.mjs](verify-evidence.mjs), independently reads raw capture
events/statuses, checks current selected hashes and the worker's exact-source
proof. All1264 selected file hashes and five executable hashes still match the
worker freeze after the integrated command;1261 noncandidate inputs and51
generated/pin paths retain their before/final bytes. These selected inventories
are bounded evidence, not a full-repository/toolchain qualification closure.

The initial broader target BASE failure remains retained: the missing patched
toolchain prevented TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests.
The later bounded37-test consumer controls do not qualify that failed command.
The ordinary package profile skip likewise supplies no native platform proof.

Task11/R17, task42 exhaustive correction and task43/R21 ownership remain intact.
Task21 directly consumes task44. Original matched-first-baseline, predecessor,
fixed-identity preservation, full/default/functional/affected-consumer/native-
Darwin/static-both-source-set and formal requirements remain required and open
where unproved under task11/21; use their original commands and unchanged
dispositions for qualification. Native Linux remains unverified and nonblocking
under fn128. Fresh bounded source/evidence reviews permit only a separate source
checkpoint, not formal SHIP or DONE. The worker's pending-root handover is an
earlier snapshot superseded only by this actual root gate and current Flow state.
