# Temporal integration for Gomad v3

This directory owns Temporal-specific use of the application-neutral Gomad v3
module. It contains the `test_dep` wrapper fixture, the bounded representative
Temporal qualification manifest, and outside-in tests of the root Make targets.

Run the wrapper contract and representative qualification with:

```sh
make gomadv3-integration-test
make gomadv3-qualification
```

The qualification command runs every manifest entry through Gomad's public
CLI and retains the aggregate at
`tools/gomadv3/.toolchain/temporal-qualification-set.json`. Expected unsupported
boundaries are exact downstream dispositions, not claims of support.
