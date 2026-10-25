# Private preparation capture

The existing driver `command` and workload `producer.run` accept an optional
`private_observations` frozen driver configuration. The default is `None` and
omits the witness field. A requested capture must succeed before the command
witness is published or the caller receives success. The generic driver retains
its existing child PID, wait4 accounting, timeout and process-group cleanup.
Capture refusal retains the actual child exit; it does not fabricate a failed
native execution.

The private Python wrappers accept `--private-observation-config PATH`. This
loads the existing driver configuration, authenticates its complete frozen
source/tool inventory, and checks the wrapper's actual binary, checkout, source
manifest and owning root. Each wrapper reuses the original host admission before
any candidate command: an authenticated initial admission is a predecessor, and
both evidence/scratch must still have 12 GiB available. It retains a fresh host
fact/refusal receipt. No public Rust CLI grammar is added.

Only actual native `plan`, `explore` and `qualify` commands opt in. Driver entries
for plan, runtime recording and qualification select this through the boolean
`private_observations: true`. Install, replay, reporting and wrapper subprocesses
do not masquerade as native preparation owners. Nested workload/inspection
wrappers receive the same private configuration path.

The command witness gains this field:

```json
{"private_observations":{"source_sha256":"<frozen source manifest SHA>","binary_sha256":"<actual executable SHA>","campaigns":[{"path":"<absolute campaign>","receipt_path":"<campaign>/preparation-observations/receipt.json","receipt_sha256":"<actual receipt SHA>","files":{"<absolute metadata or raw payload path>":"<actual SHA>"}}]}}
```

`files` is the exact native receipt payload table, including phase metadata and
excluding the separately sealed receipt. Campaigns are selected from actual
stdout and authoritative argv: plan authenticates its output/preparation receipt
and plan/module/provenance commitments; explore selects its actual result's
campaign under `--artifacts`; qualify opens its actual Result under the spec's
ArtifactRoot, binds source/operator PID, and requires all ordered iterations and
distinct campaign paths for the declared repeat. There is no discovery glob.

Capture hashes the receipt and every listed raw payload immediately after the
child exits. It binds current candidate identities, prepared Record/module/
provenance/build key/compiler/sources, current owner PID, both phase metadata and
current admission command ownership. Paths, private modes, bounds, content names
and the closed sidecar tree are checked. Compile observations may retain an older
owner on a genuine cache hit. Admission observations belong to this command.

The receipt schema is `gomad.native-preparation-observation-receipt/v1`.
Receipt/metadata limits are 16 MiB, individual raw blobs 128 MiB, with at most
32 raw blobs and checked uint64 byte accumulation. These capture limits do not
replace the independent reader's command-specific bounds or full original Go
projection. No volatile observation enters canonical Record, Store or plan
identity. `verify_snapshot` checks an already authenticated outer seal; it never
adopts a replacement native receipt as authority or grants qualification credit.

The independent receiving reader and native sidecar publisher are separately
owned. Missing sidecars or a missing opt-in cannot satisfy final proof. Their
reviewed integration, actual native two-publication test, frozen candidate,
original admission and full onebox gates remain required. Bounded Python fixture
processes test this capture boundary without providing runtime evidence.
