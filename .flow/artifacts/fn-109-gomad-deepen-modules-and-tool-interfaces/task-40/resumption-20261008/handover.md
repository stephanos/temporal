# Task40 readiness repair

The admitted test-only repair is verified on developmental linux/arm64 with stock Go1.27.1. The complete default command gate passes 92 named results with no failures/skips; ten repetitions of acknowledgement, Structured cancellation and the three compatibility cancellation branches pass 60 results. Independent review and task acceptance belong to root.

Base HEAD is `4ce2d847afd8762728f30d153d725fd1c073ecb0`. Final source manifest is [source-1f4c3da2976f69a2ac13ea2ec3218b5594e139fe95cd18de54eae823abc509a1.json](source-1f4c3da2976f69a2ac13ea2ec3218b5594e139fe95cd18de54eae823abc509a1.json); the changed test file hashes to `84364a221242b142729ec1c31d17a8a199ab0df52f39ec7ce6861d6ebe3e4a63`. [evidence.json](evidence.json) retains exact commands, exits, elapsed times, source/control/tool bindings and terminal handles. Each named receipt binds its raw stdout/stderr and environment. No actual model telemetry was available.

Only the two [admitted fixtures](admission.md) changed. Both now publish a positive newline-terminated PID and wait for its complete acknowledgement; an empty, partial, malformed or nonpositive PID cannot trigger the cancellation assertion. Cancellation uses a cancellable context plus an independent five-second readiness bound. The real caller deadline has ten seconds total allowance and must establish readiness within five seconds; it still expires as `context.DeadlineExceeded`. Watchdog retains its one-second trigger and uses no competing caller deadline. A failed readiness monitor cancels the caller and fails a prerequisite assertion. The monitor finishes before the test returns. Complete stdout/stderr byte counts are additional prerequisites, and every original raw-error, SIGKILL, bounded-output, group and ESRCH assertion remains.

The deterministic RED retained an empty-file acknowledgement accepted by the pre-fix predicate (`os.ReadFile` success alone). Its combined selection ran the acknowledgement regression only; watchdog diagnostics were therefore retained separately before the repair. That watchdog run recorded caller error nil, Cancelled false, WatchdogTimeout true, an actual WatchdogError wrapping the raw killed-process error, and complete 11/10-byte streams. The historical task9 watchdog RED winner remains unreconstructed; this packet establishes no production watchdog defect.

| Evidence receipt | Exit | Named results / observation |
| --- | --- | --- |
| `task40-readiness-red` | 1 | 1 expected failure: empty PID acknowledgement |
| `task40-watchdog-before` | 0 | 2 passes; precise original watchdog event |
| `task40-readiness-green` | 0 | 8 passes, including real deadline/watchdog |
| `task40-defaults` | 0 | 92 passes; complete hostexec/gocommand gate |
| `task40-readiness-repeat` | 0 | 60 passes; no deadline/watchdog repetitions |
| `task40-architecture-serialized` | 0 | 14 passes |
| `task40-generated-serialized` | 0 | Check-only `make validate` |
| `task40-source-linux`, `task40-source-darwin` | 0 | Eight packages each; test source included |
| `task40-format`, `task40-errortype` | 0 | Formatting/diff and standalone errortype |
| `task40-lint-scoped` | 0 | hostexec/gocommand/capabilityreview: zero issues |
| `task40-lint-task9-scope` | 1 | target/hostexec: two inherited build-context ST1005 |
| `task40-lint-integrated` | 2 | 24 residuals: 20 errcheck, 1 forbidigo, 3 staticcheck |
| `task40-lint-fast` | 0 | Actual `make lint-code-fast`: zero issues |
| `task40-preservation` | 0 | 15 original function chunks identical; two explicit exceptions |

[Preservation details](task40-preservation.stdout) bind all original function chunks and four other mechanism/test files. `processAlive`, its invalid-PID behavior and its ESRCH-only death predicate are unchanged; original comments are unchanged. The only additions are the acknowledgement reader, bounded waiter and regression. Task9's two fresh upgrade selections each passed 31 results on source `7303aa7e…`; all their source inputs except this test file still match. These are reused same-production controls, not a full-current-candidate qualification; the prior upgrade ENOENT remains unresolved. The two pre-existing untracked Turbo documents were untouched.

Execution bookkeeping retained one accidental read-only overlap: architecture handle 19724 and generated-validation handle 52472 overlapped from 2026-10-09T02:16:13.767Z to 02:16:18.003Z. Their receipts are supplementary. Both terminated before subsequent work; architecture handle 43781 and generated validation in serial driver 15923 supply the required sequential replacements. All 18 receipts bind unchanged inputs. Every execution handle is terminal; the lane is free.

Genuine deferred/simultaneous pipe-close faults remain unexecuted, with no admission waiver. Original predecessor/matched-first-baseline and complete source-owned full/functional/affected-consumer/static requirements remain open wherever unproved; source inventories are not complete platform type-check or native execution proof. Broader unfiltered lint remains red. Native fn128/fn149 remain deferred/unverified, with no patched-runtime, native host-gate or soak claim. Root must attach the independent review and decide source progress/lifecycle. This worker staged, committed, pushed and published nothing and changed no task/goal status.
