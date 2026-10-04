# Task 16 independent source audits

Two fresh-context read-only reviewers were explicitly requested as `gpt-6.1-sol` at high effort, following Codex's AGENTS.md routing. Reviewer and writer are from the same family; host metadata does not independently establish actual model identifiers. Both reviewed actual dirty tracked source and untracked lifecycle/tests, not the empty committed range. Neither ran tests, edited files, invoked generation, changed Flow/history or created CLI bridges. These are source audits, not formal implementation-review receipts, native acceptance or SHIP.

## Correctness: lifecycle_source_correctness

Critical, Important and Minor findings: none.

The arbiter lock covers validation before progress mutation, aggregate acknowledgement consumption, response phase changes, wakes and settlement. Admission/forwarding reject duplicate identities before credits change. Wait suspension/resumption, suspended delivery, completion-only reservation, model dispatch/arrival, abandoned late discard and removal own their complete transitions. Multiple blocking operations retain aggregate-credit semantics; coordinator response correlation is independent of model IDs. The old API and coordinator barrier state are removed, not merely wrapped.

## Integration: lifecycle_source_integration

Source integration/preservation findings: none.

The 22 original simulation_unix.go accounting sites and three process_unix.go callback sites, plus runnable/removal boundaries, migrate to the owner. The eight methods, external runnable/remove entrypoints, response-barrier map/helpers and completionPending bookkeeping are gone. Process lifetime/completion/reap, transport pending/abandoned correlation and domain commitment stay separate. Quiescence uses owner validation/consumption while holding the same mutex. Model transport's hash remains unchanged.

All ten final source/test hashes, design and retained preimage match evidence.json. Direct comparison restricts task-15 test-body changes to the two allowed strengthened historical negatives; all nine valid bodies are unchanged. Reversing only the fixture's two migrated callback expressions reproduces its task-15 hash, preserving bounded frame I/O, installed-waiter handshake and cleanup. Both negatives have meaningful old-source RED and candidate GREEN. The namespace case runs coordinator and real model transport request ID 1 concurrently and verifies independent settlement. Unknown/duplicate replies precede callbacks; cancellation retains the committed World snapshot and discards a known late reply.

Task-14's source manifest and task-13 generator/schema/runtime projections remain unchanged by this migration. No public API, schema, wire format or runtime timer owner changed.

## Acceptance and next admission

Conductor reruns and hashes are retained in conductor-verification.md. Required patched runtime/process/native commands and lint remain unavailable, as detailed in native-gates-open.md and worker evidence.json. Task 16 is not complete and R11 remains open. Under MILESTONES immediate-delivery-order item 4, its integrated/reviewed source candidate permits task 17 source implementation; that does not waive either task's native acceptance.
