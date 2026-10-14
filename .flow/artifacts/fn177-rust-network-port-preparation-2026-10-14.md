# Rust network owner preparation

Read-only research by `network_port_contract` (gpt-6-astra/high) against frozen `960c966cdc9fba84736a268d8949f30214c16ac4`. This is preparation, not an implementation, review verdict, test pass or task completion. fn154's completion dependency remains mandatory. Root was unchanged during research.

## Current owners and contract

The reference is `tools/gomad3/internal/model/network/{adapter.go,generated.go,corpus_test.go}`. Rust currently has only `wasmhost/src/network/PINNED_GO_NETWORK_SHA256`; no network owner or corpus runner. Existing serde, serde_json, base64 and sha2 dependencies suffice. A binary-only crate needs an explicit source-module inclusion or an intentionally chosen library seam for integration tests; do not use a helper protocol entry point as model transport.

Mirror New/Register/Revoke, Dispatch/Cancel/Ready, Partition/Heal/Delay/Reset, Advance/NextDeadline, Snapshot/Record/Finish. Preserve seams for LimitClockCeiling, AdvanceBefore and atomic ChangeGroup, used by the broker but not comprehensively covered by the corpus. Domains bind run/node/incarnation/address. The broker supplies slot-shifted global u64 request IDs; the owner preserves IDs and domain ownership, not a new slot abstraction.

The corpus schema is `gomad3.network-operation-corpus/v1` (`corpus_test.go:20`). Preserve per-step outputs, new transitions, readiness, deadline, state, snapshot and final Record. Byte fields are base64; Address uses capitalized IP/Port; omitted fields, null, empty arrays and zero are distinct. Record is captured before Finish. Newly appended transitions are not the same representation as canonically lane-sorted final history. History digests use existing binary identity encoding, not JSON hashing.

Source currency is owned by identity.go, TestNetworkCorpusIsCurrent and TestNetworkV2RustOwnerPinIsCurrent. Root pin is `sha256:74ed8c885fbba5314f544436659e4c9f79fc555e7101411ede412c229b4b655c`; rename-derived pins must come from the eventual integration base, not this historical preparation.

## Task boundary requiring care

fn177.2's held FIFO/heal/revoke/topology semantics already expose history_reserved and next_deadline in the corpus. A reservation-free subset would under-deliver the task. Implement correct per-direction interval/due bookkeeping, epoch invalidation, pair reservation count/release and deadline visibility as needed for these semantics. fn177.3 can retain actual expiry processing, pending-dial stalls, exhaustion/overflow and terminal/ordering acceptance. Do not replace missing bookkeeping with fabricated expected values or silently defer held-stream coverage.

Research proposes 90 complete .2 fixtures, 22 explicit .3 fixtures and two mixed broker fixtures, subject to implementation-time verification. This does not amend the task contract. Classification must be exhaustive and reject unclassified new files, not infer exclusions dynamically from observed reservation counts.

The 22 proposed .3 fixtures are equal-time-heal-first-{false,true}; stall-epoch-invalidation; autonomous-expiry-reserved-terminal-slot; simultaneous-pair-expiry; held-reservation-ceiling-{5,6}; partition-reservation-atomic-rejection; stall-overflow-queued-{false,true}; held-timeout-replay-admission-valid; partial-write-stall-timeout; pending-dial-caller-{0,5,100}; pending-dial-heal-{cancel,revoke,refused,capacity,repartition-epoch}; pending-dial-reservation-overflow-{false,true}.

Both broker-strict-topology-graceful-{false,true} scripts introduce disabled-link pending dial at zero-based step 22. Their first 22 steps cover six domains, five connections and several request bases. Deferring whole scripts cannot excuse .2's multi-domain requirement: direct two-pair tests must exercise topology, cancellation, revocation/restart and independently based request IDs. All 64 random scripts, malformed replay creation, held FIFO/heal/close/reset, received-tail repartition and both revoke/restart fixtures remain in the proposed .2 set.

## Behavior-sensitive test seams

Dispatch precedence is finished, special poll ownership/completion, then ordinary ID/limit/domain/pending-capacity validation before handle/operation checks. Poll consumes completion without retry; Ready retries signaled/expired requests in ascending ID order. Completed entries remain until poll. Preserve partial writes, one-chunk reads and short-read tails; empty I/O adds no chunk. Cancellation overwrites even completed results, with separate dial cleanup. Revoke first admits lifecycle history, drops retained bytes, closes listeners and removes domain pending requests; graceful peer EOF differs from crash reset/closed, and old handles stay stale after registration of a newer incarnation.

Partition holds ready-but-unread chunks but not an already received tail. Effective heal resets held readiness to now plus current delay in FIFO order; redundant heal and delay-only changes preserve readiness. Full close drops local unread and disabled-link outbound chunks; CloseWrite permits draining. Local fullness blocks, while global exhaustion rejects the next chunk and retains previously accepted bytes.

Future test-first order: independently validate corpus deserialization/history digests; domain/handle and error precedence; partial writes/readiness/cancel; held FIFO/heal; two concurrent node pairs with distinct slot bases; then direct ChangeGroup, LimitClockCeiling and AdvanceBefore controls. Numeric errno translation, helper ABI, World, guest evidence and execution/performance integration remain fn178 work.
