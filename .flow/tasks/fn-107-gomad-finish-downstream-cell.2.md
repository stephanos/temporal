---
satisfies: [R3, R5, R6, R11]
---
# fn-107-gomad-finish-downstream-cell.2 Inject an in-process Walker profile and TCP-only membership

## Description
Work in ../saas-temporal. Reuse localcluster Spec/InProcessExecutionEnvironment and GossipNetwork. Add cluster-scoped volume options/replication provider propagation across bootstrap/datanode/bimmer paths, and TCP-only membership with loopback IP configuration. Preserve default UDP/native behavior. Profile validates missing providers, capacity and addresses before startup. Files/Touches: walker/localcluster/**, walker/testutil/memberlist*.go, walker/serviceinstance/**, walker/bimmer/server lifecycle seams as needed. Quick: mise run test -t 3m ./walker/localcluster; focused testutil/serviceinstance tests and lint. Red/green isolated tests cover membership discovery/update, bindings/rebind, profile failures and lifecycle.

## Acceptance
In-process profile propagates one injected storage tree and local replication provider to every component; membership uses TCP without UDP while preserving discovery/update/failure handling. Logical readiness/shutdown and wall watchdogs expose failures. Native defaults and relevant tests remain unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
