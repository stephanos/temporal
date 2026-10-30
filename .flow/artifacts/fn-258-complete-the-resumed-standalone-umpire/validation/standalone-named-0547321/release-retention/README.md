This directory retains the terminal standalone release command at
05473210b48c3db862f22051eb7154ce98258f98 (tree46829511b6d582b40f3fa5095f42bdf980c7d466).
Canonical parent release.json/log bytes were copied and hash checked, not changed.
The command exited0 in1668.258310920006seconds with clean source.

count-audit.json segments the raw log by actual Cargo Running/Doc-tests headers.
For each of69 native producers, its first declared count agrees with its final
summary. Nine preceding nested control summaries are excluded. Native totals are
910 discovered,908passed,2ignored; doctests5discovered,4passed,1ignored.
Aggregate915/912/3 has no failures, measured tests or filtered top-level tests.
The three ignored cases and all producer/block counts remain explicit.
Nested UI/trybuild dev compilations do not change the top-level optimized release
profile. The actual target compiler cache records rustc1.99.1/LLVM23.1.1.

recovery-manifest.json references the umbrella .recovery/fn258-standalone-proof
CAS. archive-manifest.json identifies all69 actual named release test executables
with individual SHA256/size/permission modes inside a deterministic compressed
archive. Dynamic dependency observations are retained; there were no additional
target-local shared dependencies. System libraries are not copied. Cargo does not
report retained paths for temporary doctest executables; their raw results remain.
No real Go-host/WASI-guest parity execution or required model-TLC coverage is
inferred from this Cargo-only default-backend command. The required054 gate and
paired gates supply those separately scoped proofs.

Independent recovery requires only this directory and the CAS, no source/target:

    python3 -B verify-recovery.py

It verifies every object and every archived binary's bytes/size/mode and refuses
in-memory digest corruption and missing-member controls. retain.py is reproducible
against the idle original release target; it performs no builds or cleanup.
Only this new artifact directory and CAS objects were written. No active paired
target was read, and no source, refs, Flow state or existing receipt was changed.
