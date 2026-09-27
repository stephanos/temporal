# Temporal integration for Gomad v3

This directory owns Temporal-specific use of the application-neutral Gomad v3
module. It contains the `test_dep` wrapper fixture, the bounded representative
Temporal qualification manifest, and outside-in tests of the root Make targets.

Run the wrapper contract and representative qualification with:

```sh
make gomad3-integration-test
make gomad3-qualification
```

The v3 manifest owns 16 tier 2 package workloads, the tier 3
`frontend-system-info` functional probe (`./tests/gomadfunctional`, guarded
capability mode), and two fixed seeds. Gomad analyzes
the complete corpus first, executes only supported workloads, retains and
replays every successful repetition with bounded choice coverage, and writes a
path-free `gomad3.qualification-set-report/v1` to
`tools/gomad3/.toolchain/temporal-qualification-set.json`. Expected unsupported
boundaries are exact analyzer dispositions, not claims of support; the report
keeps actual supported and unsupported counts separate from expectation
matching.
