# Qualification parity

The actual public Go WASM grammar is `qualify --spec FILE`. Backend command
interception runs before the generic host CLI. Both frozen Go references reject
`qualify --json` with status 2, `flag provided but not defined: -json`, and usage
listing only `-spec`. Generic host workload flags cannot establish public parity.

`../qualify-corpus.json` records fixture, fixture canary, file-backed
SQLite (`setup WAL`) and onebox (`TestFrontendSystemInfo`) inputs. The cluster
case uses two resident nodes, crash and durable restart. Reviewed goldens cover
every cell; retained results qualify only their named candidate.

`../qualify-check.py --grammar-only` probes both binaries for rejected generic
flags, absent specs and unreadable spec files, plus the three set-command
refusals on both binaries. Each refused verb is tested bare, with help and with
its former generic flags, using both default and missing installations. All
18 cases run twice on each binary, compare status 2, empty stdout and the
unsupported-command classification, and match `goldens-qualify/refusals.json`.
Raw stdout/stderr and command hashes are retained separately; diagnostic wording
is not normalized into execution evidence. The lane supports the same
`GOMAD_GOLDEN_REGENERATE=1` repeated-agreement and byte-identical second
regeneration rule as the other qualification goldens. The grammar probes also
check a final `--` terminator and refusal of positional arguments after it.
They compile and execute no workload.

`qualify-lane.py` owns workload parity, Result/receipt authentication and the
reviewed goldens (see `CLI.md#rust-qualification`).
Fixture qualification invocations include a final `--`; other cells retain the
original spelling. Report runs compare both forms byte-for-byte and refuse
arguments after the terminator.

Preparation remains available separately, reusing public Go installation, the
existing Rust fixture identity bootstrap and the existing namespace-image
conversion. It writes six lineage-specific manifests without compiling a guest:

```sh
python3 testdata/parity/qualify-check.py --prepare-only \
  --binary /absolute/gomad-fixture --go-binary /absolute/go/gomad \
  --go-source /absolute/frozen-go-source --go-helper /absolute/go/gomad-wasmhost \
  --go-compiler /absolute/go1.27.1/bin/go --input /absolute/replay-input.json \
  --work /absolute/new-preparation
```

The actual owner spec contains `Manifest`, `ArtifactRoot` and `Campaign`.
`Campaign.Target.Backend` must be `wasm` before Freeze; omitting it returns
status 3, `unsupported WASM source preparation request`. The Go response names
`result_path` and `operator_receipt`; its Result retains `Qualification`,
base64 `QualificationBytes`, candidate identity Manifest and measured costs.
The workload lane authenticates raw result, report, receipt, campaign and payload
bytes before comparing semantic verdicts and stable golden projections. Rust
uses its own Result schema and storage identities. The lane covers all five
cells, tamper negatives, repeated agreement and golden regeneration.

The v17 retained lanes under
`/home/agent/wt/evidence/fn232-qualify/d/v17` passed all five cells, including
canonical budgets, two regenerations and strict comparison. Those results retain
their original candidate scope; current-source checks and their limitations are
recorded in the fn-232.10 Flow handover. Final whole-candidate qualification
belongs to fn-232.13.
