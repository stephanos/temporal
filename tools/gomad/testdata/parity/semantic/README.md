# Behavioral parity foundation

`../corpus.json` is the shared input index for the differential driver and its
goldens. The initial `fixture` case is a semantic example, not a guest execution
or a qualification claim. Its repeated `a` digests are deliberately illustrative;
raw integrity tests use independently computed real SHA-256 values and payload
bytes. The frozen Go inputs remain owned by fn-232 at the `baseline_inventory`
path. This corpus does not replace that inventory or execute its historical commands.

Run from `tools/gomad`:

```sh
python3 scripts/behavioral-parity --corpus testdata/parity --selftest
python3 testdata/parity/semantic/selftest.py
GOMAD_GOLDEN_REGENERATE=1 python3 scripts/behavioral-parity --corpus testdata/parity --selftest
```

The command's `gomad.parity-observation/v1` adapter declares every field and type
in `project`. Unknown fields fail. The registered future verbs use that adapter;
each verb task supplies its real CLI adapter and fixed corpus entries. This
foundation does not invoke or require Rust `inspect`.

The full per-lineage golden keeps failure signatures, record hashes and execution, implementation,
payload and trace identities unchanged. Cross-lineage comparison checks the
behavioral view after independent raw authentication: ordering, seeds, outcomes,
termination, failure partition, replay disposition/ordinal/reason, reductions,
limits, virtual time and qualification cells. Record and implementation identities
belong to different lineages and are retained in their separate goldens rather
than required to equal one another. Failure signature labels compare by partition.

Diagnostics are limited to the adapter's seven explicit diagnostic fields.
Campaign/run IDs receive consistent first-occurrence tokens; timestamp and host
elapsed reporting are diagnostic. Declared output/scratch prefixes preserve
relative suffixes. A replay argv keeps every argument exactly except its final
artifact path operand. No recursive field masking is performed.

Each observation's `payload_file` and `trace_file` name exact manifest payloads;
their reported digests must match independently authenticated bytes. The trace
must additionally be a named stream in the canonical execution record.

Future fresh execution mode:

```sh
python3 scripts/behavioral-parity --verb replay --corpus testdata/parity \
  --go-driver /absolute/go-adapter --rust-driver /absolute/rust-adapter \
  --output-root /absolute/new-comparison
```

Each executable adapter receives `--verb`, `--corpus`, `--case` and
`--artifact-root`, executes its binary with its own fresh artifact lineage, and
emits one observation on stdout. A successful adapter returns zero even when the
observed CLI command returns an expected failure; the observation retains that
execution's status. Raw stdout, stderr, argv and adapter exit are retained.
Every adapter must also write `foundation-result.json` referencing canonical
records, manifests, payloads and a hash-linked ledger inside its artifact root.
The foundation bundle schema is `gomad.foundation-result/v1`; the Rust foundation
fixture producer owns its record and storage schemas. A Go adapter must authenticate
original Go evidence before exporting this fixture representation; conversion
alone does not authenticate historical evidence.

Each lineage runs twice into different empty directories. Raw records and the
declared views must agree before comparison or regeneration. `GOMAD_GOLDEN_REGENERATE=1`
writes separate Go/Rust goldens only after agreement. Running regeneration a
second time must leave golden bytes unchanged. Normal mode refuses absent or
stale goldens and missing corpus entries. Every new real verb adds its fixed
inputs and reviewed goldens; none is claimed by these semantic examples.

To authenticate a foundation producer directly:

```sh
python3 scripts/behavioral-parity --corpus testdata/parity \
  --foundation-root /absolute/bundle --foundation-result /absolute/result.json
```

Raw record canonical bytes and hashes, derived failure signatures, exact payload
sizes/digests and original ledger links/order/counts are checked before projection.
Ledger topology is then represented by ordinal references. The harness does not
replace the production authenticated store opener or make a qualification claim.
