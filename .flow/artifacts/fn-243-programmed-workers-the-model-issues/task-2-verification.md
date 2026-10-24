# Task 2 verification

Source checkpoint: `fad276dfc399766a28b51d4262829f90be6e73b6`.
Guest SHA256: `b4a4f57f1a64525ae568bd56a73eff85b0b253aeae44d9e4de3e2d88f546cf2a`.
Program SHA256: `518a416709b0ec47f968ff6f97fba0cfa6dc536bce59f233d59906f0497548cd` (unchanged from task 1, re-exported with the scoped guest).

## Final gates

| Command | Actual result | Retained log |
| --- | --- | --- |
| `CI=1 UMPIRE_BACKENDS=require make -C /Users/stephan/Workspace/skunkworks/umpire/temporal model-time` | Exit 0; gate 688.082s, inner edit/test 8.479s | `.flow/tmp/fn2432-model-time-scoped-final.log` |
| `make -C /Users/stephan/Workspace/skunkworks/umpire/temporal umpire-check-live-tests` | Exit 0; 2,427 passing test identities | `.flow/tmp/fn2432-live-scoped-final.log` |
| `GOLANGCI_LINT_BASE_REV=8c82fb4ff3ee4c8f694b9067d7f6652803b50428 make -C /Users/stephan/Workspace/skunkworks/umpire/temporal lint-code-fast` | Exit 0 | `.flow/tmp/fn2432-scoped-final-lint.log` |
| `go test -tags test_dep ./tools/umpire/cmd/umpire/... ./common/testing/testpilot/... ./tools/canary/...` | Exit 0 | `.flow/tmp/fn2432-scope-root-go.log` |
| `go test -tags test_dep ./...` in `model/temporal/hosts/go` | Exit 0 | `.flow/tmp/fn2432-scope-temporal-go-green.log` |
| `go test -tags test_dep ./tools/umpire/temporaldriver -run '^TestUnheldCancellationRefusedBeforeCancelRPC$' -count=1` | Exit 0 for raw and SDK; actual Start1, Cancel0 | `.flow/tmp/fn2432-cancel-rpc-green.log` |
| `python3 model/temporal/feature/activity/compatibility/verify.py` | Exit 0; all 15 original artifacts exact | Command output retained in task handover |

The required TLC mode ran; Apalache was unset and remains deferred. The full gate missed its 60s soft target; the inner loop met its 10s target. Live stages reported slice 51.588s, gaps 394.015s, and the final full Testpilot stage 437.061s. Test identities include negative controls and supported-scope refusals; they are not a count of conformant Runs.

## What the controls establish

- Raw and SDK workers follow the Model-issued table independently of Umpire. Startup validates unknown keys; exact first-match rules are immutable. Unmatched tasks return nonretryable `umpire.unprogrammed`, and the empty-table control produces a named Refusal.
- Composition validates both program namespace and task queue against configured scope. SDK Follow binds the table namespace on an alias of the caller-owned SDK transport, installs its scenario propagator, and preserves the actual poll interceptor. Tests use a real SDK client and fake public gRPC server; payload metadata equality is complete.
- The Driver owns initial poll containment and joins workers before closing connections or deleting resources. Unsupported external/late holds and read timeouts remain DriverFault controls. Model code receives actual server observations rather than callback counts.
- Normal READs judge actual result/failure content, retry history, skipped lifecycle paths and contradictory phase evidence. Wrong result, wrong failure and completion-instead-of-failure are Unexplained for raw and SDK. The full annotation assertion remains, with an independent negative control rejecting absent or unclassified worker transport evidence.
- Admitted terminal cancellation follows an actual normal READ. Its compound event preserves both actual Cancel NotFound and fresh Describe evidence. The response-mutation control distinguishes the captured actual server bytes from intentionally contradictory history; it remains Unexplained with the original ACK retained. Protocol identity/status, publication rollback, timeout, Close and offline replay controls remain.
- Each programmed-worker Run records exact executor kind/version/namespace/task queue through an optional generic recorder. Literal wire controls preserve earlier tags and generic Answer. Fake Driver records its own kind; offline replay preserves metadata without constructing a Driver. Canary fencing stays strict and forwards the recorder.
- A created namespace is publicly probed on the exact pending activity's history target using a valid nonempty synthetic RunID, and on the addressed activity queue partition. Only NamespaceNotFound is retried; actual modeled Start is sent once and its acknowledgment is unchanged. The restored live lifecycle records 21 Runs, requires strict explanations for supported cases plus only exact audited scope refusals, and proves creation while the CLI runs and NamespaceNotFound after deletion. Probes do not guarantee every host/partition or future ownership change.

## Preserved discrepancies and boundaries

The actual SDK seed18 Run at guest `13e91f833a1c3696fa57dc04b551eb6631a9313e896d9a4e259f7dd0e3a0f55d` records successful cancellation followed by FailedPrecondition because cancellation remained pending. Its JSON, canonical binary and source reasoning are protected under `counterexamples/cancel-ack-pending*`. Server-free replay using its original guest reproduces the exact Unexplained Run at event6, with Driver calls0. It has not been normalized or reclassified as conformant.

The conductor-authorized checkpoint now refuses unheld cancellation of an open activity at Propose with exact Unsupported message `unheld cancellation of an open activity is outside the atomic cancellation Model`. Raw and SDK controls prove no cancellation proposal or Cancel RPC. Held-before-start and terminal-cancel cases remain strict. The original unheld-race R2 obligation stays open in fn-243.5 after the independently planned fn-237 cooperative Model/live binding; task2 does not close the spec or claim that behavior passed.

The earlier READ-finality counterexample is protected as `counterexamples/read-finality-run.json`, with its original guest retrievable at the recorded commit. The Runtime fix makes the realization hook own READ continuation, retaining nonempty intermediate progress without consuming the pending request; independent failed-publication and unproposed-observation controls remain.

No activity Model semantics changed. New TimedOut projection is not claimed: start-to-close/heartbeat timeout overwrites LastFailureDetails, whereas schedule timeouts retain prior history. Deadline realization remains fn-237 work. First-attempt history consistency checks apply to the admitted lifecycle, excluding external Reset.

## Evidence honesty

Earlier source passes are labeled by their own source and do not substitute for the final gates above. The pre-scope whole live gate failed on the protected cancellation Run. A lifecycle invocation without the integration build tag selected zero tests and is inconclusive; the corrected actual lifecycle and full gates passed. Default branch-wide lint exposed inherited retired surfaces; the task-base scoped run then found and fixed one new deprecated accessor by comparing the complete request proto. The namespace cache race was retained red before the public-readiness fix. A proposed buffer-lifetime concern was refuted by the pinned gRPC Materialize implementation, and no speculative clone change shipped.

Review and closure metadata are recorded separately after the final source gates. No worktree or branch was force-removed; lane source archives were verified before normal removal.
