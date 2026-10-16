# Recorded SQLite operation prefix

`sample.json` retains the original producer result, artifact manifest,
provenance and simulation bytes as base64. The Go test verifies their exact
SHA-256 values before deriving a script. Producer result:
`017d5d9a84f3c37d3b6f93fd1570b388d9d63120c5f4a23d17f18f763f712b66`;
artifact record: `sha256-e6b432a5fc6d733ac8f0536be864f1ad`.

The actual guest is `testdata/storage_sqlite/main.go`, source SHA-256
`b1cc7c67c5e46d5fa6a1644f5dc033788674259d38909e6ede493bcd696e8980`,
module SHA-256
`5b6eade386726065e5e57fd2b67c399104a084563ea1b88c1fc1192902ca7bc3`.
The producer is the cooperative SQLite durable success qualification, seed 7,
Linux/aarch64, integrated source SHA-256
`0bd4d29c8b074decac6b489544a75b923e54778642f9b21ff658855348ca70b5`.
The retained producer manifest identifies compiler, runtime, helper, engine,
model and workload source hashes. This is historical evidence from that
producer, not acceptance for this provisional source branch.

Original retained location:
`/Users/stephan/Workspace/skunkworks/gomad-postrename-retry-20261014.TMcAas/workloads/`.
The producer references the artifact; the artifact manifest authenticates
`backend/provenance.json` and `backend/simulation.json`. The latter contains the
recorded volume transitions in its base64 `record` JSON.

The replay sample projects exactly transitions 0–3: allocation and directory
entry creation for `/db`, then `/db-journal`, in incarnation 1. The Go test
compares every projected `volume.Operation` field, including IDs, dependencies,
inodes, paths and current effect hashes, with the derived prefix. Transition kind
`allocate` maps to projection kind `alloc`; recorded ordinal, node/incarnation,
volume and outcome remain in the authenticated source bytes.

Derived inputs are two `path_open` calls on preopen fd 4 with create flag 1,
`rightsAll`, zero dirflags/inherited rights/fdflags, and virtual `now=1`.
The historical profile is explicitly refused before the test selects the current
cooperative profile for these derived inputs. Effect hashes bind the current
unversioned namespace; the retained sample keeps its historical hashes.
These inputs are an equivalent
operation projection, not captured guest syscall requests. The producer did
not retain original WASI requests, per-call clock values, or write payloads;
the sample therefore makes no full syscall or write-payload replay claim.
