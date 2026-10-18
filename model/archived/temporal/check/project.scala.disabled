// Temporal's gate: the framework's checks (model/framework/check) with the gate over the Temporal
// Models, their coverage registry, the source metrics and the Temporal API proto-name rule (lint/).
// It is built from both halves, the framework's first, whose project.scala is then the build's own:
// `scala-cli <command> model/framework/check model/temporal/check`. Four entry points: `run`, the
// gate; `modelLoop`; `changed`, the change-scoped check (Gate.scala); and `metrics` (Metrics.scala),
// which scala-cli runs with `--main-class temporal.check.metrics`.
//> using mainClass temporal.check.run
