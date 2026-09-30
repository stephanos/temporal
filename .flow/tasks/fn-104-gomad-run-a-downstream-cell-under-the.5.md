---
satisfies: [R3]
---
# fn-104-gomad-run-a-downstream-cell-under-the.5 Record a disposition for every run-time boundary operation the downstream services perform

## Description
C4. For datagram sockets, advisory file locks, statfs, concrete listener-type assertions, all-interface binds, port probing, process metrics, and long readiness waits: decide modeled (with the COMPAT-5 evidence set), target-injectable (injection point named), or denied (exact finding), using the linked-mode baseline. Where Gomad behavior is undocumented (unspecified bind addresses, *net.TCPListener assertions), make it deterministic and documented with a test.

## Acceptance
- GOMAD_CLOUD.md lists each C4 item with its disposition and evidence; none unknown
- unspecified-address bind and listener-type behavior are tested and documented


## Done summary
Every C4 operation has a recorded disposition with evidence in GOMAD_CLOUD.md: all-interface binds, concrete listener types, and port probing are modeled (net_bind under the deterministic profile for two seeds: an unspecified host binds the in-memory 127.0.0.1 listener, net.Listen returns *net.TCPListener with a *net.TCPAddr, a closed port rebinds, a second bind fails, the transcript is seed-independent); datagram sockets, advisory locks and the other filesystem calls, statfs, interface enumeration and address resolution, and process signals are denied with their exact findings and a named target injection point or gomad-tag seam; DNS is modeled for localhost only; process metrics are admitted on linux and a target seam on darwin; long readiness waits follow the virtual clock. The direct GOMADSEED mode was found to use the host network, so the check runs at the Runner level.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5421fcafb
- Tests: go test ./runner/internal/execution -run TestProfileNetworkBindContract
- PRs: