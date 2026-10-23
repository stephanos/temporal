# fn-232.13 receiving reader source handover

Source implementation only. No frozen runtime, mandatory workload, SQLite selection,
Temporal recovery/reset/inspection, or final qualification run is claimed.

`tools/gomad/rust/driver/receiving.py --config /ABS/driver.json --root /ABS/ATTEMPT`
implements the 13 IDs in `receiving.OWNED`. The external FINAL section contract is
`gomad.rust-driver-final-section/v1`, with exact source, identity and owned rows.
Exit 0 requires all 13 proofs; absent/unbound evidence exits 4. Configure the FINAL
section with a pinned Python interpreter and this reader, followed by `--config`
and the frozen configuration path. The composition owner appends `--root`.

The reader reopens retained observations; it runs no producer and writes no report.
It uses private copies of adapter function namespaces to reuse original runtime,
cell/budget and corpus assertions with read-only command and write operations.
It authenticates actual command argv/cwd/PID/process registration/time/exit,
release discovery and fixture-control refusal, raw canonical Result/receipt,
publication CPU/RSS tail, private single-link Result/receipt publication,
selected marker presence/kind/mode/time/size/content, current source/helper/host/toolchain/choice source,
provenance, native module and every payload/stream, and complete journal links.
It recomputes all 32x25 runtime shards and 32 forced replays, exact ten distinct
mandatory producers and immutable before/after pairs, four SQLite states and
contexts/selections plus independent native oracle, replication histories,
Temporal committed full-state coverage and exact interrupted prefix/images,
named-reset actual stream/reconnection/in-flight RPC history, and all three
source-detached useful stack/heap/probe/snapshot inspection owners.

Workload command metadata now records truthful Popen PID, cwd, start/end,
process registration, argv hash, raw stdout/stderr hashes, selected detached
lookup environment fields and a full environment digest. No environment secrets
are copied. Each producer retains its absolute owned root and actual process
argv/PID/time. Historical receipts lacking these observations remain historical.
`temporal-inspection.py --owner all` is an internal coordinator for the exact
three existing owners; each keeps its original limits and selection.

Remaining concrete actual bindings before a candidate can pass:

- All installation/spec/compiler/runtime/source/build-feature/reader/oracle pins,
  actual release binary, fresh roots and sent epoch must be frozen by the owner.
  The shipped example bindings remain deliberately unbound.
- `storage` must bind the actual `temporal-recovery.py --primary /ABS/PRIMARY
  --root /ABS/NEW_ROOT` argv after the independent recovery primary producer.
- `network-streams` must bind actual `producer.py --workload retry --mode success`
  argv with its original captured configuration and source manifest.
- `temporal-inspection` must bind actual `temporal-inspection.py --owner all`
  argv and frozen captured configuration. All three nested roots must exist.
- Mandatory domain checkers and the SQLite independent native Go oracle still
  need reviewed pinned argv and actual executions. No retained PASS-only checker
  can replace this reader's raw proofs.
- Terminal, retained20, operator, stress41x4, ringpop120s, R4 and public verbs
  remain separate unbound owners. No receiving row claims those IDs.
- The complete original test-wasm qualification predicate mapping is a separate
  source review extension; these 13 IDs do not automatically cover its 72 roots.

Portable evidence: `.flow/tmp/fn232-receiving-red.log` records the initial missing
reader failure; `.flow/tmp/fn232-receiving-all.log` retains all 43 passing driver
framework tests, including 16 receiving tests. Tests execute only bounded Python
fixtures, never Go, Cargo, guests, builds or actual qualification producers.
`git diff --check` passes. Logs are local scratch evidence, not qualification.

Owned worktree `/home/agent/wt/fn232-receiving-reader` remains for root integration
and the qualification mapping extension. No other worktree or cache was created.
