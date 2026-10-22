# fn-222.1 Rust one-box gates: checkpoint 2026-10-22

Status: **pending**. Gate 1 (unchanged source preparation) has no Rust entry
point, so no measured gate was run. The task requires reporting a gate-1 failure
before any measured gate. The QUALIFICATION.md one-box row is unchanged.

## Candidate

| Identity | Value |
| --- | --- |
| Integration commit | `gomad` 7aacd7cb21 (includes fn-232.10 partial merge 1c804fd576) |
| Rust release binary (`cargo +1.99.1 build --release --locked`, no features) | sha256 `ebb8eedf1fa908a53a0b009cd70dbf37b9755548558860a04a59411e6607b678` |
| Linked helper implementation (`helper_sha256`) | `54773ee79492c60e23585b9dfe3f71dd4e9e3aada19d06a5b7af045768b3c2a3` (per `.flow/artifacts/fn232.8-identity-handoff-2026-10-22.txt`) |
| Compiler | stock Go 1.27.1, `.build-cache/fn232-stock-go1.27.1/bin/go`, sha256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64` |
| Host | linux/arm64 |

Host, runtime-overlay, module, engine and model identities are bound only by a
Rust `qualify` freeze (`gomad.rust-qualification-manifest/v1`). No freeze ran,
so none is recorded for this candidate.

## Gate mapping on the Rust binary

The gates in `QUALIFICATION.md#one-box-cell-gates` are Go test-API commands
(`go test ./qualification`, `./backend`). The Rust binary has no test API. Its
public verbs are `explore`, `resume`, `execute-shard`, `merge`, `minimize`,
`qualify`, `report`, `replay` and `inspect`.

| Gate | Rust equivalent | Numeric exit |
| --- | --- | --- |
| `unchanged-source-preparation` (prepare-only, retained module receipt) | none: `qualify --spec` always runs the measured repetitions after freeze; no prepare-only mode or spec field exists | not run (no command) |
| `measured-pass-replay-success` | `gomad --installation INST qualify --spec SPEC` with `Manifest.Cell=success`, then `gomad report --result R` | not run (gate 1 first) |
| `measured-pass-replay-canary` | same with `Cell=canary` and `-test.run=[` | not run (gate 1 first) |
| `storage-transport-inspection` | Result/receipt validation is `report` (fn-232.10); the inspection cell is the Go test `TestTemporalInspectionRetainedFailure/onebox` in `./backend`, with no Rust command | not run (no command) |
| R2 support list (`gomad.wasm-support-list/v1`) | none: `BuildSupportList` is Go and accepts only Go-lineage Results | not produced |

Probes on the candidate binary
(`/home/agent/wt/evidence/fn222-onebox/probe-verbs.txt`):

| Command | Exit | stderr |
| --- | --- | --- |
| `gomad install` | 2 | `gomad: unsupported command` |
| `gomad support-list` | 2 | `gomad: unsupported command` |
| `gomad compare-support` | 2 | `gomad: unsupported command` |
| `gomad qualify-set` | 2 | `gomad: unsupported command` |
| `gomad qualify --spec /nonexistent.json` | 2 | `gomad: open /nonexistent.json: no such file or directory` |

## Further gaps for the measured gates

- **No Rust installation verb.** The fn-232.10 lanes derive Rust installations
  from a Go `install` manifest through test tooling
  (`testdata/parity/qualify-check.py --prepare-only`, `fixture-replay-input`).
  That tooling is hard-coded to the `./tests/gomadfunctional` namespace
  (`frontend-system-info`). The mandatory smoke cell needs the `./tests`
  namespace (working directory `/workspace/tests`).
- **Frontend-system-info on the Rust binary.** The fn-232.10 lanes already ran
  it as a final success cell with the owner budgets on the fn232-qualify
  candidate (evidence `/home/agent/wt/evidence/fn232-qualify/d/v17/`, exit 0).
  That candidate is not this one, and no canary was run, so it supplies no row.

## Decisions needed (not taken here)

Each needs host code or test-tooling changes, which this task's stop rule
forbids. They are recorded rather than started:

1. A Rust prepare-only freeze (gate 1) that retains the module receipt.
2. A Rust installation path for the `./tests` smoke namespace.
3. A Rust storage/transport inspection cell equivalent to
   `TestTemporalInspectionRetainedFailure/onebox`.
4. A Rust port of `BuildSupportList`/`ValidateSupportList` over Rust Results
   and receipts. The task's Touches already name `rust/gomad/src/qualification/**`.

No capacity limit was changed.
