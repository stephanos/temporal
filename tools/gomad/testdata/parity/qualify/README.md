# Qualification contract checkpoint

The actual public Go WASM grammar is `qualify --spec FILE`. Backend command
interception runs before the generic host CLI. Both frozen Go references reject
`qualify --json` with status 2, `flag provided but not defined: -json`, and usage
listing only `-spec`. Generic host workload flags cannot establish public parity.

`../qualify-corpus.json` records intended fixture, fixture canary, file-backed
SQLite (`setup WAL`) and onebox (`TestFrontendSystemInfo`) inputs. Actual
multi-instance cluster execution is pending. No corpus cell has a positive
qualification result or reviewed golden in this checkpoint.

`../qualify-check.py --grammar-only` runs cheap Go probes for rejected generic
flags, absent specs and unreadable spec files, plus the three set-command
refusals on both binaries. It then probes Rust's actual `qualify --spec` command.
The Rust verb now exists. The grammar mode checks that its missing-spec refusal
matches Go, and `qualify-lane.py` owns workload parity, Result/receipt
authentication and the reviewed goldens (see `CLI.md#rust-qualification`).
Neither check mode compiles or executes a workload.

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
A resumed implementation must authenticate original result, report, receipt,
campaign and payload bytes before projecting semantic verdicts. Rust needs its
own reviewed schema rather than Go storage identities.

Preparation passed under
`/home/agent/wt/evidence/fn232-qualify-prepare-02`. Existing Rust missing-verb
and Go generic-grammar refusals were observed on 2026-10-21. The final
`--grammar-only` mode retained all twelve command receipts under
`/home/agent/wt/evidence/fn232-qualify/grammar-01` and exited 1 at the expected
`$.rust.qualify.availability` difference. All five Go grammar probes and the
three unavailable set-command probes on each binary returned status 2.
That checkpoint predates the Rust port. `qualify-lane.py` now covers the fixture,
fixture-canary and SQLite cells, raw-integrity and tamper negatives, repeated
agreement and golden regeneration. The onebox and cluster cells and acceptance
on the integrated candidate remain open under fn-232.10.
