# Umpire3 1.0 support policy

`umpire3/v1` experiment and proof manifests are strict: unknown fields, vocabulary, capabilities,
and claim kinds are rejected. Compatible additions require a new version or an explicitly optional
field policy. Semantic hashes cover every model source named by an exporter; descriptor hashes cover
the selected Temporal API schema.

Umpire3 1.0 supports the checked Nexus cancellation and Workflow Update slices, the controlled local
profile, and the CI profile. Deployment authority, customer data, production canaries, and arbitrary
Temporal behaviors are unsupported. Cooperative adapter cancellation is required; hard termination
of non-cooperative adapters requires a future process-isolated profile.

Umpire2 remains independent and supported by its existing owners. Shared extraction is allowed only
after both systems independently expose the same stable responsibility and retain dependency tests.
