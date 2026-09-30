# Why Gomad v3 is retained

Gomad v3 runs ordinary Go target source under a reviewed patched toolchain and
records bounded identities, choices, I/O, and replay evidence. This keeps Go
language/runtime behavior in the upstream toolchain while Gomad owns deterministic
execution and explicit external models.

V1's AST rewriting and replacement-library matrix required growing exceptions to
preserve types and dependency behavior. V2's typed translation and custom runtime
provided strong multi-machine simulation, but also required maintaining translated
Go behavior and an operating-system model. Those trees were retired by `fn-81`.
Their implementation comparisons remain in Git history.

The useful lifecycle, network, durability, fault, and isolation contracts belong
to the current [simulation architecture](../tools/gomad3/ARCHITECTURE.md#in-process-cluster-simulation).
The process backend supplies fresh globals and hard isolation; the in-process
backend retains its documented limits. Simulation parity does not imply race-mode
or complete historical product-feature parity.

Future investment follows [GOMAD_MILESTONES.md](GOMAD_MILESTONES.md) and the
[capability roadmap](GOMAD3_NEXT.md). The dated
[Loom and simulation-testing assessment](../docs/research/gomad/GOMAD_CMPv2.md)
records ideas for improving the retained implementation.
