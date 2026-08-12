# Activity batch-terminate I/O profile

Add `temporal-activity-api-batch-terminate/v1` as a second exact Gomad I/O
profile. It reuses the existing deterministic network, filesystem, entropy,
SQLite, and transcript implementation, while its inventory and implementation
identity remain distinct from the batch-cancel profile.

Each profile accepts only its unchanged Temporal functional-test suite selector.
The existing batch-cancel profile and its replay identity remain compatible.
No Temporal production or test source is changed.

Qualification builds and runs
`TestActivityAPIBatchTerminateClientTestSuite` twice with the same seed and
requires successful exits and byte-identical I/O transcripts. Unit tests cover
resolution, exact target acceptance, and rejection of cross-profile selectors.
