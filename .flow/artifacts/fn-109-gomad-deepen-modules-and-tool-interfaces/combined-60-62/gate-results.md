# Joined source verification

The startup correction lets the ordinary Runner suite finish and exposes 46 previously unexecuted failing top-level tests. The joined candidate removes seven lint findings. Required Runner and lint gates remain red, so tasks60-62 and fn112.10 retain open acceptance.

Root ran the frozen batch at95354f677162df9cd76383569b11fe1f902870cd in the isolated conductor worktree. Fingerprint fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297 binds1,070 source inputs, including parent and mixedbrain module files. All11 commands terminated without source or orchestration changes. The pinned stock Go host is linux/arm64; these results establish no patched-runtime or native qualification.

| Gate | Exit | Seconds | Observed scope |
| --- | ---: | ---: | --- |
| Focused combined | 0 | 7.067 | 15 top-level and47 total passing tests across Runner, execution and toolchain |
| Full ordinary Runner | 1 | 7.603 | 82 pass,123 fail,12 skip top-level;345 pass,288 test failures,12 skips including subtests |
| Affected vet | 0 | 0.983 | Runner, execution, toolchain |
| Standalone errortype | 0 | 1.091 | Same three packages |
| Architecture source sets | 0 | 56.588 | Six selected boundary tests, including HostPackageVet |
| Private execution ownership | 0 | 0.786 | Existing ownership test |
| Generated validation | 0 | 19.481 | Existing check-only validate target |
| Formatting and diff | 0 | 0.216 | Five product files and checkout diff |
| Affected configured lint | 1 | 6.619 | 17 findings |
| Canonical fast lint | 0 | 15.843 | Admission-base diff filter; full lint remains red |
| Original-base nested lint | 2 | 3.585 | 53 findings; integrated errortype unreached |

[Receipts and raw logs](full-ordinary-runner.json) retain exact commands, cwd, environment, tool identities, terminal exits and output hashes. [verify_packet.py](verify_packet.py) checks the11 receipts,22 raw-stream hashes, source manifest, both original orchestration scripts, five reviewed product hashes and lint residuals. Its final read-only invocation exited0. The initial verifier rejected duplicate lint identities; the corrected verifier includes source line numbers and maps actual unchanged lines from7b75ae3 to the candidate. No gate receipt or raw log changed.

Actual original-base lint falls60 to53. Exactly six child-output errcheck blocks and the competing-build sleep disappear; zero findings appear. Every remaining three-line diagnostic block matches after the one affected location maps process_test.go1419 to1432. The53 residuals are one errcheck, eight forbidigo and44 staticcheck. Affected configured lint retains one errcheck, four forbidigo and12 staticcheck. Fast exit0 is diff-filtered and does not replace either red gate.

The Runner emits its normal terminal package failure after all217 top-level tests. There is no diagnostic timeout or manual abort. Its extra package-level fail event explains289 raw fail events versus288 test failures. Completion improves observed coverage; it does not establish a passing package or baseline equality for newly reached cases. Healthy native progress/heartbeat execution remains unverified.

## Updated fixture boundary research

Root reused the research worker at requested gpt-6-astra/high for read-only classification. Actual execution-model telemetry is unobserved. The researcher made no writes and ran no Go gates.

All77 historical failed top-level tests remain;46 newly reached failures include42 runner_test cases, two seed-completion cases and two watchdog-replay cases. First-blocker presentations are80 explicit unsupported linux/arm64 errors,34 missing .toolchain/bin/go errors, seven validation-stage errors whose formatting hides their cause and two downstream consequences. The last nine are not demonstrated independent defects.

Fourteen failing tests exercise real boundaries and stay outside blanket fake-operation migration. They are the three isolated coordinator cases, two environment cases, coverage-binary replay rejection, preparation-owner bundle ordering, three replay-I/O integration cases, two real replay-operation transport cases, pinned-outcome sampling and captured-input watchdog replay. Portable assertions inside these tests keep their current owners; the label alone transfers nothing.

The remaining109 failures are fixture-correction candidates, including five mixed public/private characterizations. An explicit per-test inventory is still required. The existing four-operation private dependency recommendation remains preparation, bootstrap, recorded-adapter verification and toolchain-identity reading. Zero dependencies must retain real production validation. No global testConfig change, executor-type inference, public hook, skip or fake-host substitution follows.

Current primary fn109.63 orders decomposition before further Runner changes. Root will re-anchor any private-fixture correction after that task lands. Current fn152/fn153 retire journal, corpus and canonical-JSON implementation; root will not harden their retiring code. The owner's format amendment supersedes encoded-byte equality only and preserves behavior, error precedence, lifetimes and crash guarantees.

The remaining watchdog ready-marker errcheck requires its own admitted failed-write decision. Research recommends immediate fixture exit3 without secondary output, with genuine read-only fd1 and valid terminal descriptor controls. The current failed write still enters its deliberate wait. No readiness change or lint exception is authorized by this report.

## Continuation and acceptance

The previous report-only turn changed no authoritative milestone state. This continuation revalidated the retained execution handle, received its terminal batch, verified the complete packet and used the newly reached failures to refine the next action. Root will retain an independent joined source-progress review before primary integration. Formal implementation review stays deferred while required source gates are red.

Native fn149/fn128 stay deferred and unverified. No Done, native pass, soak bound, push, PR, CI dispatch or native-owner revival follows. The broader milestone objective remains active.
