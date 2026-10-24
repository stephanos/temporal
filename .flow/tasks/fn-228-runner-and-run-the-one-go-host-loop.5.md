---
satisfies: [R5]
---
# fn-228-runner-and-run-the-one-go-host-loop.5 Superseded by fn-243: Shim endpoint for non-Go SDK workers (not built)

## Description
Add a local gRPC endpoint on the Runner that forwards Answer for out-of-process workers, carrying a protocol version; a thin forwarder returns result, failure, heartbeat or delay; unknown version is a Refusal; unreachable endpoint and callback timeout are DriverFault events. Prove with one non-Go SDK worker control (the smallest SDK the repository already provisions) and a Go-side fake for the four reply kinds.

### Required integration contract

Route Shim callbacks through the same correlated event queue and serialized Host used by in-process callbacks; return each Answer to its originating worker, enforce version and bounded context, and test cancellation/late/duplicate replies.

**Touches:** model/umpire/hosts/go/umpire/runner/**, common/testing/testpilot/temporal/worker/**, model/umpire/proto/**, model/umpire/hosts/go/go.{mod,sum}, go.{mod,sum}, Makefile, tests/testcore/testpilot/** and the chosen existing non-Go worker fixture. Task .6 follows this task; no parallel shared-manifest writes.

### Deferred (2026-10-23)

The owner deferred the Shim on 2026-10-23 ("skip the Shim for now"). Superseded by fn-243 (programmed workers); at the owner's request the partial Shim start was deleted on 2026-10-23 and no Shim work remains.
## Acceptance
- [ ] Shim endpoint and forwarder; four reply kinds proven
- [ ] Unknown version refused; unreachable and timeout recorded as DriverFault
- [ ] One non-Go worker control passes

## Done summary
Superseded by fn-243 (programmed workers); nothing built. The owner deferred the Shim on 2026-10-23 ("skip the Shim for now"), the partial start was deleted the same day, and the live worker connection R5 asked for is replaced by Model-issued worker programs judged from server history. R5 is not delivered by fn-228.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: