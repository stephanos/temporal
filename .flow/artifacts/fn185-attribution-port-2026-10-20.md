# fn185.1 current-source attribution evidence

Scope: port the historical fn185 attribution tooling onto the current helper
module ownership at base `e09f1b10174f10d67465c7fc83e96ba2240dcdd8`.
Historical measurements are not evidence for this candidate. The full matched
single-/multi-guest measurements remain fn185.2/.3 after the fn221 sweep.

## Source

- `229d2a9f9d`: eight reviewed fn188 strict-boundary refusal expectations;
  `invalid` becomes `unsupported`, preserving the specific diagnostic messages
  except the intentional named unknown-schema refusal. Parent approved these
  semantic changes separately from later identity regeneration.
- `beeb80a460`: exact import/owner attribution and current collector/producers.
- `d20d64c465`: detached retained cluster stderr accessor regression.
- `88355e51ec`: bounded completed-thread aggregation, full-width incarnation
  identities, owner/thread reconciliation refusals, canonical release/diagnostic
  cluster comparison, actual broker World spans and adversarial tests.
- `db0df2e27a`: regenerated feature attribution expectations. The guest-profile
  unknown-field refusal additionally follows fn188's `unsupported` contract.
- `f172c17499`: combined guest-profile/fuel-off sidecars, using the same v2
  extension and preserved explicit cluster refusal.

Timing-only diagnostic builds support cluster attribution. Fuel-off and sampled
guest-profile helpers retain explicit cluster refusal; no sampled cluster path
is claimed. Release feature selection and Cargo manifests/lockfile are unchanged.

## Verification

Raw logs: `/home/agent/wt/evidence/fn185/`.

| Current-source check | Exit/result |
| --- | --- |
| Go measure, measurement and wasmmeasure suites | 0 |
| Changed Go module lint/vet, base e09f1b1017 | 0 (retry; initial global lint lock collision) |
| Diagnostic-profile all-target release Clippy, warnings denied | 0 |
| Diagnostic Rust unit filter | 0; 13 passed, one overhead-calibration test intentionally ignored |
| Diagnostic-profile helper corpus regeneration | 0; 25 passed |
| Guest-profile helper corpus regeneration | 0; 25 passed |
| Diagnostic-fuel-off helper corpus check | 0; 27 passed |
| Empty-feature release helper corpus | 0; 26 passed |
| Repeat regeneration of those three modes | 0; complete sidecar SHA-256 inventory byte-identical |
| Combined guest-profile/fuel-off corpus and repeat regeneration | 0; 27 passed, sidecar SHA-256 inventory byte-identical |
| Cargo.toml / Cargo.lock diff against base | empty |

The ninth semantic compatibility case is explicitly
`feature_guest_invalid_unknown`: its termination becomes `unsupported`, while
the existing unknown-field message remains unchanged. Unknown-import and
stock-profile diagnostics are preserved. These refusal changes are not described
as identity-only regeneration.

The Rust unit run includes 80 sequential completed incarnations retaining only
64 compact rows plus an omission counter, and real World broker calls proving
nested exclusive subtraction. Go adversarial cases reject missing served owners,
duplicate/oversized/incomplete thread rows and non-choice canonical divergence.

## Acceptance status

- Independent source re-review approved `88355e51ec`: no remaining blocking
  source findings. Reviewer checked `db0df2e27a` JSON changes are limited to the
  diagnostic extension, intended timing-cluster support and strict refusals.
- Fresh empty-feature helper build and actual runtime-cost proof exited 0.
- Fresh seed-zero activity release/diagnostic proof exited 0; no historical
  helper or module proof is represented as current qualification.
- Root owns integrated full qualification and task-state updates.

Fresh proof inputs (source `f172c17499`; helpers have the same source as the
approved source tip, with only generated expectation changes afterwards):

- Empty features: `cargo +1.99.1 build --release --locked -j2
  --no-default-features`; helper SHA-256
  `d939435b43422578b5f3b637779c7f8cde310971b7d1859c446cb47d41900a58`.
- Diagnostic features: `diagnostic-profile`; helper SHA-256
  `0b4201d298d8fd16b8556a28e9ae005ddc1d7b5355bef26cfb1a247698f8d8a0`.
- Stock Go 1.27.1, current runtime-source overlay, fresh slim Temporal guest
  SHA-256 `4e19470a4f6f68083f6bea28d3ff61d7ae7317d2799fdde7185e37e42506da78`.
- Producer scope is explicitly `diagnostic_only` with the existing
  `prototype-legacy-v1` compatibility profile. It proves attribution and
  canonical execution equality, not a qualified fn185.2 performance baseline.

## Fresh activity proof

Report:
`/home/agent/wt/evidence/fn185/current-proof/activity-seed0/.gomad-measure/host-owner-profile/run-FNUCAN6FQCWD2R2ORJ2Q6QTXLL/host-owner-profile.json`.
Command log: `/home/agent/wt/evidence/fn185/current-proof/activity-seed0.log`.
The report retains the complete rerun command, identities and owner tables.

Seed 0, activity selection, one warmup and one measured repetition; both have
216633 calls. Both match release choice witness
`437443c1e34bf18941038243b71d8de69fdf20f96a87f6917921926e7f0ec698`.
The producer also requires identical canonical WASI, network, stdout/stderr,
terminal, call/time-advance/environment and remaining-fuel evidence.

| Run | Execute ns | Attributed host ns | Guest residual ns | Difference ns | Tolerance ns |
| --- | ---: | ---: | ---: | ---: | ---: |
| warmup | 5561958170 | 1694710714 | 3867247456 | 0 | 27809790 |
| repetition 0 | 7952102837 | 2194022623 | 5758080214 | 0 | 39760514 |

Both raw helper profiles report zero ABI overlap errors, span errors,
unattributed imports, unframed imports and omitted thread rows. These two runs
establish attribution correctness; their timings do not establish a performance
ranking or optimization decision.

## Fresh empty-feature runtime-cost proof

Exit 0. Log:
`/home/agent/wt/evidence/fn185/current-proof/runtime-cost.log`.
Metrics, parent CPU profiles, guest module and exact choice bytes:
`/home/agent/wt/evidence/fn185/current-proof/runtime-cost/.gomad-measure/runtime-cost-profile/run-WWYGKSQUW2A2SARAORPZRVLFDZ/`.

The moved `runtime-cost-profile` producer runs the actual cooperative runtime
with Go-owned then helper-owned model execution. Both runs used the fresh
empty-feature helper above and runtime module SHA-256
`8006536a3db89822a6595c039e7a24c595ccf16fd5cb476a6b4fa2b11a96b10a`.
Both completed successfully with 100000 select polls, 100000 select results and
one terminal result. Both exact choice streams have SHA-256
`c25c65d470200b04222c4ec569867309db8ab39b3ea0b038401208dc69625c63`.
The producer also compared canonical stdout/stderr, WASI, network and terminal
evidence; it would fail rather than publish success on any difference.

Reproduce from `/home/agent/wt/fn185-next/tools/gomad` with the current stock Go
binary at `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`:

```sh
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  go run ./internal/backends/wasm/cmd/wasmmeasure runtime-cost-profile \
  --repo /home/agent/wt/fn185-next --stock-go /absolute/stock/go1.27.1/bin/go \
  --profile-root /absolute/new/evidence/root \
  --test-helper /home/agent/wt/evidence/fn185/current-proof/helpers/release-gomad-wasmhost
```

These receipts establish the attribution source and early-proof checks on
`f172c17499`, not all fn185.1 acceptance. R4's qualified-workload preservation
remains open. The conductor owns integrated qualification, Flow state and
milestone updates; fn185.2/.3 measurements are not claimed complete here.

## Integration acceptance review context

The attribution proof above retains its exact `f172c17499` candidate and helper
identities. It is not relabeled onto integration source `d3c6382aa3`: later replay,
model and cluster changes mean complete helper-input equivalence is false. The
per-op/owner instrumentation and collector themselves remain unchanged. Final
changed-candidate runtime/workload/replay qualification and the later matched
profiling window remain pending; no new Go-only profiling cycle was run.

Fresh source admission on integration `d3c6382aa3`, with Go1.27.1 and Rust1.99.1:

- `go test -p=1 -tags test_dep ./internal/backends/wasm/internal/measure ./internal/backends/wasm/internal/measurement ./internal/backends/wasm/cmd/wasmmeasure`: exit0.
- `go run ./internal/backends/wasm/cmd/wasmmeasure --help`: exit0, no producer execution.
- `cargo +1.99.1 clippy --release --locked --offline --all-targets --features diagnostic-profile --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml -- -D warnings`: exit0, shared release target/jobs1/incremental0.

Raw logs are `/home/agent/wt/evidence/fn185/root-{source-admission-corrected,command-help,diagnostic-clippy}-d3c6382aa3.log`.
The initial source command mistakenly named nonexistent
`internal/backends/wasm/measurement` instead of `internal/backends/wasm/internal/measurement`;
that setup failure remains in `root-source-admission-d3c6382aa3.log`, exit1.
It is not a failing existing package or a successful aggregate command.
The spec's older Quick toolchain is superseded by the repository's pinned1.99.1.

Independent implementation review must assess task .1's exact R1/R4 scope and
the limits of the retained early proof; this note is not a verdict or a waiver
of any remaining qualified-workload obligation. Tasks .2/.3 remain open.

Independent three-draw review finalized NEEDS_WORK, RID
`4b18246eda3f4ea0c41fb874140b65ec`, receipt
`.flow/tmp/impl-review-receipt-fn185-attribution-current.json`:

- R1: new choice-stream hashing in `model/reply.rs` lacks a transcript/evidence
  owner span, so reconciliation passes while its cost is assigned to scheduler.
  A narrow Rust-only repair and actual-helper regression are in progress.
- R4: the original proofs do not establish current qualified-workload
  preservation. Task .1 remains in progress. Its pending dependency now names
  fn232.13's final frozen Rust runtime-cost-equivalent behavior and qualified
  choice/WASI/network/output comparison. This is not a waiver or a completed
  transfer: fn185.1/R4 cannot close until that actual receiving evidence exists.

No new Go-only profiling or full qualification is authorized. The reviewer and
writers are from the same model family (`gpt-6.1-sol`, reviewer effort high).
