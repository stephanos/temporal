# Umpire3 1.3 support policy

`umpire3/v2` Experiment and proof manifests are strict: unknown fields, vocabulary, capabilities,
and claim kinds are rejected. Compatible additions require a new version or an explicitly optional
field policy. Semantic hashes cover every model source named by an exporter; descriptor hashes cover
the selected Temporal API schema.

Umpire3 1.3 supports the checked Nexus cancellation, Workflow Update, Workflow Task acknowledgement,
generated assurance inventory, typed author facade, deterministic sparse compilation, guided
exploration, campaigns, first-class faults, and local, CI, remote, gRPC-only, and canary profile
contracts. The release remains a candidate until the environment-specific qualifications recorded in
`assurance/release/testdata/generated/umpire3-1.3.json` are attached by deployment owners for local in-process, CI test cluster,
remote deployment, public-gRPC-only, and production canary execution. A Deployment profile contract
is not evidence that a particular deployment was tested.

The checked composition artifact has no unresolved metadata obligation. Parity ledger v4 resolves
all 22 retained assurance rows with exact declaration and local-integration evidence. Migration
ledger v3 records 23 exact and 5 semantic-equivalent root behaviors, with no partial or inventory-only
row. A qualified release permits no partial vision goal and separately requires bound external
qualification evidence for all five profiles. The candidate assurance graph has no local vision
omission; it binds the retained native scale benchmark, coverage-guided mutation audit, control-plane
resilience audit, developer UX audit, clock-skew audit, and published documentation. Its remaining
omissions are the five independently signed profile runs.

The supported operator surface is `cmd/umpire3` with `explain`, `run`, `replay`, `mutation`,
`qualify`, and `promote` subcommands. The `umpire3-run` and `umpire3-qualify` compatibility commands
were removed on 2026-08-21; callers must use the corresponding unified subcommands. Replay bundles
use `umpire3/replay-bundle/v3`; qualification receipts use
`umpire3/qualification-receipt/v3`; release manifests use `umpire3/release/v6`. These formats are
strict and require a version change for incompatible fields.

The production canary accepts only v3 Ed25519-signed immutable digests, resource budgets, and
allowlists from its pinned approval authority. Its prepare, execute, wait, observe, and cleanup work
runs behind a killable process boundary with an explicit environment. Deployment credentials,
arbitrary customer data, unapproved faults, and arbitrary Temporal behaviors remain unsupported.

Umpire2 remains independent and supported by its existing owners. Shared extraction is allowed only
after both systems independently expose the same stable responsibility and retain dependency tests.
