---
satisfies: [R5]
---
# fn-228-runner-and-run-the-one-go-host-loop.5 After .4: Shim endpoint for non-Go SDK workers with protocol versioning

## Description
Add a local gRPC endpoint on the Runner that forwards Answer for out-of-process workers, carrying a protocol version; a thin forwarder returns result, failure, heartbeat or delay; unknown version is a Refusal; unreachable endpoint and callback timeout are DriverFault events. Prove with one non-Go SDK worker control (the smallest SDK the repository already provisions) and a Go-side fake for the four reply kinds.

## Acceptance
- [ ] Shim endpoint and forwarder; four reply kinds proven
- [ ] Unknown version refused; unreachable and timeout recorded as DriverFault
- [ ] One non-Go worker control passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
