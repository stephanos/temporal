# Modeling Umpire3 behavior

Lean owns product states, actions, transition meaning, executable equivalence, safety properties,
refinement, module contracts, targets, monitors, and the semantic catalog. Go owns transport,
bounded orchestration, evidence normalization, and generated artifact consumption. A modeled
Scenario is compiled into Experiments; each Execution returns a Result whose retained form is a
Replay bundle. Go adapters never redefine the Lean transition relation.

Add a lifecycle through the typed declarations under `model/Umpire3`. State its assumptions and
finite domains, prove executable equivalence, add positive traces and a negative mutation, compose
provider and consumer contracts, and export a monitor only when live observations decide the
property. Run `make umpire3-gen`; never add a parallel Go allowlist.

Selected protobuf structure comes recursively from
`model/Temporal/API/selection.json`. Every selected field receives a generated disposition and fuzz
class. Product meaning belongs in an interpretation module and must be backed by conformance
fixtures. Descriptor presence is not semantic interpretation.
