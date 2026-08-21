# Temporal integration for Gomad v3

This directory owns Temporal-specific use of the application-neutral Gomad v3
module. It contains the `test_dep` wrapper fixture, the bounded representative
Temporal qualification manifest, and outside-in tests of the root Make targets.

Run the wrapper contract and representative qualification with:

```sh
make gomadv3-integration-test
make gomadv3-qualification
```

The v3 manifest owns 16 Tier 2 workloads and two fixed seeds. Gomad analyzes
the complete corpus first, executes only supported workloads, retains and
replays every successful repetition with bounded choice coverage, and writes a
path-free `gomadv3.qualification-set-report/v3` to
`tools/gomadv3/.toolchain/temporal-qualification-set.json`. Expected unsupported
boundaries are exact analyzer dispositions, not claims of support; the report
keeps actual supported and unsupported counts separate from expectation
matching.
