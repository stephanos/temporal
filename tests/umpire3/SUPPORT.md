# Umpire3 1.2 support policy

`umpire3/v2` experiment and proof manifests are strict: unknown fields, vocabulary, capabilities,
and claim kinds are rejected. Compatible additions require a new version or an explicitly optional
field policy. Semantic hashes cover every model source named by an exporter; descriptor hashes cover
the selected Temporal API schema.

Umpire3 1.2 supports the checked Nexus cancellation, Workflow Update, Workflow Task acknowledgement,
generated assurance inventory, typed author facade, deterministic sparse compilation, guided
exploration, campaigns, first-class faults, and local, CI, remote, gRPC-only, and canary profile
contracts. The release remains a candidate until the environment-specific qualifications recorded in
`testdata/umpire3-1.2.json` are attached by deployment owners. A profile contract is not evidence that
a particular deployment was tested.

The candidate may contain explicit pending composition obligations, partial migration fidelity, and
not-yet-implemented parity entries. The checked candidate currently has one exact/equivalent parity
target and 19 partial rows, plus 14 exact, 4 semantic-equivalent, and 10 partial root behavior
migrations. A qualified release permits none of those partial states and requires profile-qualified
parity evidence.

The supported operator surface is `cmd/umpire3` with `explain`, `run`, `replay`, `campaign`, and
`qualify` subcommands. Existing single-purpose commands remain buildable compatibility entry points.
Replay bundles use `umpire3/replay-bundle/v1`; qualification receipts use
`umpire3/qualification-receipt/v1`. Both formats are strict and require a version change for
incompatible fields.

The production canary accepts only immutable approved digests and allowlists. Its prepare, execute,
wait, observe, and cleanup work runs behind a killable process boundary. Deployment credentials,
arbitrary customer data, unapproved faults, and arbitrary Temporal behaviors remain unsupported.

Umpire2 remains independent and supported by its existing owners. Shared extraction is allowed only
after both systems independently expose the same stable responsibility and retain dependency tests.
