Implemented the public stdlib-only common/gomaddebug helper and Temporal shard/mutable-state diagnostic snapshots for fn-180.3/.5. Commit 309f1a367f8d9a277fbac6a2a5a2575dc34c54e9 in /tmp/gomad-fn180-library; root owns integrated review, qualification and Flow completion.

Similar-code search found complementary gomad seams in common/config and service packages, but no public registered-diagnostic/probe helper to reuse. The implementation follows those build seams and extends shard write-unlock and mutable-state created/loaded/history-event/transaction-close mutation points. Required anchor/spec/source instructions were read before editing; no explicit parent Quick commands were defined, and heavy integrated gates are conductor-owned under MILESTONES batch-first policy.

The fixed 256-entry registry retains registration-order IDs, names and monotonic flags while dormant. Invalid, duplicate, late and over-capacity registration poisons closure, including recovered panic. The four-word enablement mask, primitive/string TLV encoder, counted overflow marker, per-dump bounded writer and panic-isolated checks/dumps use the runtime seam agreed with the runtime worker. The WASM compiler output shows one mask load and branch before field construction, no bounds check/call; escape analysis reports Emit fields do not escape. Real guest fixture covers an application-held lock, 32MiB post-stop allocation+GC, panic/truncation/timeout, early/duplicate/capacity/late registration modes and a witness-only allocating probe.

Temporal views contain bounded byte/scalar copies, no live application pointers. Shard snapshots retain 32 completed write-unlock views; workflow snapshots retain 64 completed mutation views, with sequence/seam and explicit stale-view/identity-truncation labels. Ordinary/native no-op hooks inline away and create no debug globals. Active tests hold the real shard RWMutex and workflow ContextImpl priority semaphore while dumping.

Verification observed:
- New core tests first failed exit 1 with missing registry/typed API, /tmp/fn180-library-red.log.
- Ordinary public helper/core tests exit 0, /tmp/fn180-library-green.log.
- Native gomad no-op public helper/core tests exit 0, /tmp/fn180-library-native-noop.log.
- Encoder with prebuilt and per-call primitive/string fields, plus overflow path: AllocsPerRun = 0, asserted in the real production encoder tests.
- Active WASM helper archive compile exit 0, /tmp/fn180-library-active.log.
- WASM assembly/escape builds exit 0, /tmp/fn180-library-disassembly.log and /tmp/fn180-library-escape.log.
- Real guest call-site archive compile exit 0, /tmp/fn180-library-callsite-disassembly.log.
- Helper go vet exit 0, /tmp/fn180-library-vet.log.
- Native no-op fixture build exit 0, /tmp/fn180-library-noop-fixture.log.
- Native/active WASM/ordinary WASM source-set selection and imports checked exit 0; imports are stdlib only (encoding/binary, math, active unsafe).
- git diff --check exit 0.
- A mistakenly started native history package test was promptly cancelled with SIGINT, exit 130 (/tmp/fn180-temporal-baseline.log); it is inconclusive and no pass is claimed.

Pending integrated execution: active Temporal package tests/guest fixture, retained Temporal failure demonstration, target isolation/prefix and compulsory-witness perturbation proof, post-stop GC/freeze/timeout, fuel pressure and full WASM/runtime qualification. History closure compile is queued for the shared heavy slot with root/smaller_guests. No active history test result, retained Temporal demonstration, independent review verdict, or full qualification is claimed by this handover.

stage: impl-review - skipped(policy: parallel surface - conductor owns integrated review)
