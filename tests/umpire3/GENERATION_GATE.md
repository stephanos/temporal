# M11 generated Go interface decision

Decision: skip generated Go semantic types in Umpire3 1.0.

The formal negative-control loop has succeeded and two Lean models use the versioned experiment
schema. The remaining entry condition is not met: there is one handwritten Go protocol model, and
the Nexus and Update adapters contain domain behavior rather than repeated schema declarations.
A Lean-to-Go generator would add a second compiler and differential-test surface without removing
measured repeated boilerplate. The ordinary protocol structs remain a non-authoritative data seam.

Re-evaluate only when at least two additional models repeat the same Go schema declarations and a
change log demonstrates recurring maintenance cost. Generated code must then remain deterministic,
carry source/proof hashes, reject unsupported Lean, and pass differential tests against Lean.
