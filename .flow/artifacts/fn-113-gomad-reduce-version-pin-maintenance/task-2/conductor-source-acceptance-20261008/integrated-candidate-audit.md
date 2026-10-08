# R3 integrated candidate audit

The root independently checked the worker's terminal handover against the
integrated candidate before committing it for formal review. This is an evidence
audit, not a review verdict or task-completion claim.

- Base: `ca6fd855868fac364b69cb31394c87ad2912e623`; task remains `in_progress`.
- Frozen source: `1ea646fc66ac63aa0d2dde38e434f8b5529c28ac9040b55b7a04c56c4ebc0f3f`.
  All 1,028 manifest paths match their current bytes; hashing the compact JSON
  manifest reproduces the source digest.
- All 17 final receipt hashes, raw-log hashes, command exits, test-event counts,
  before/after source/tool/overlay freeze fields and actual tool hashes were
  checked. All 15 indexed artifact hashes also match. The expected lint and
  injected-counterfactual RED receipts remain RED, not passing gates.
- Recomputed all four production-tail extraction proofs, public signature
  preservation, both original test-bootstrap bodies, 12 stderr argument/order
  expressions, unchanged tracked source and root pin preservation. The retained
  proof reproduces exactly: nine selected roots, six absent, zero moved.
- Compared actual unfiltered lint rows: 100 baseline findings minus exactly
  14 owned sites equals the final 86 byte-identical OTHER diagnostic rows.
  Separate added-version lint and diff-filtered fast lint remain distinct gates.
- Read the assertion mapping and all 57 excluded-name classifications; verified
  original source-file hashes for the 59 selected and 57 excluded entries.
  Retained overlay origin/transformation/effective-source proof was independently
  checked earlier and its final indexed bindings remain identical.
- The four unchanged cache controls and 55-name workspace complement form the
  exact, disjoint 59-name selection. The root separately replayed the four
  originals: 17 passing events, no failures or skips, with five real permission
  probes. See [independent-cache-controls.md](independent-cache-controls.md).
  This rerun adds verification, not unique coverage or a full single-environment
  package-pass claim.
- Inspected the unchanged `TestMemberlistSuppliedTCPConsumer`: its early skip
  requires an externally supplied checkout; its later `mise` command runs the
  actual TCP membership lifecycle. That absent-checkout workload and native
  wrapper execution remain unclaimed. Actual private stock consumers and source
  identities do not replace that workload or fn-105's SDK Git checkout.
- Verified the corrected preserved-failure attribution: extraction ENOSPC is
  `portable-preservation-consumers`; telemetry SIGBUS is
  `portable-consumers-private-cache`; `portable-consumers-private-tool-state`
  is an intermediate GREEN, not a failure or final-candidate receipt. Original
  logs and receipts were not rewritten.
- Both unrelated user files retain their original hashes and remain excluded
  from staging. No native qualification, CI, PR or push action is authorized.

[Worker handover](../source-acceptance-20261008/handover.md) and
[evidence index](../source-acceptance-20261008/evidence.json) retain exact commands,
timings, failure provenance and remaining owners. Independent review must still
assess the non-empty committed range, especially coverage equivalence,
preservation, release ordering and limits of portable proof.
