# Accepted60f baseline identity census

This census reads accepted task16 evidence and current source at `9babff6c815beb7cff92d7001f647ddca19663bf`. No Cargo, Flow, test binary or proof ran in this lane. The new task15 full runtime inventory is pending; this report makes no claim about its cardinality or acceptance.

`provenance.json` records the tracked raw archive SHA, recorded member-manifest SHA and exact extracted member hashes. All four members match their recorded hashes. Accepted validation and source-before both name `60f1c762a618d32478b58ae4e79c151f4355aeb3`. The derived inventory exactly matches raw nextest JSON: 909 baseline framework-child identities, 2 ignored. Baseline doctests separately report 4 passed and 1 ignored. These are baseline observations only. No exact accepted60f Temporal runtime inventory was located in that archive.

`anticipated-identity-map.json` records four framework/driver renames. Two guest pipeline names change from shell to retained-bootstrap/native terminology. Framework retired-name tests replace POSIX ERE/external-grep assumptions with owner-approved Rust regex and no-grep behavior while retaining invalid-pattern and actual I/O refusal checks. All four new declarations exist in current source.

Temporal's old integration `retired_names::active_spellings_and_historical_exclusions` and `retired_names::tool_failures_never_report_clean` move into binary-unit `retired_name_policy_controls`. The shell SIGPIPE race test is deliberately obsolete because the Git/grep pipeline is gone; its map points to native raw-byte/I/O coverage plus the separate Git-failure test. The new ignored `scan_child` is a private exact child invoked by those tests. These are source-derived anticipated identities, not reconstructed runtime results; confirm exact binary kind and flags in a fresh Temporal inventory.

The tracked source test-function delta has only five removed/renamed function declarations: those four framework/driver renames and the obsolete Temporal pipeline test. A broad basename comparison of all 909 framework baseline rows has no unexplained absence: 24 generated fixture cases come from unchanged fixture/build-discovery source, and the other four absences are the declared renames. A basename match alone cannot prove module registration, cfg/filter status or runtime execution.

After the running proof produces its fresh framework inventory, root can perform the full tuple comparison with:

```
python3 .flow/tmp/fn25415-validation/test-identity-census/compare_inventory.py <fresh-child-inventory-identities.json-or-raw-nextest-JSON> --output .flow/tmp/fn25415-validation/test-identity-census/fresh-comparison.json
```

This helper accepts only actual fresh input, translates exactly the four declared baseline rows, reports every missing translated identity and separately lists actual additions. It has been syntax-parsed only; root owns its execution and final comparison. Count/doctest/stage/PID proof authority remains with the unchanged standalone harness. All writes in this lane are confined to this ignored scratch directory; task/source/HEAD remain frozen.
