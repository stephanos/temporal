package fixture.realizationRefusals

import framework.realize.{Activation, Realization, Script}

val unknownConstructor: Realization = Realization(
  name = "unknownConstructor",
  scripts = Vector(Script("script", Activation.Unknown))
)

val unknownField: Realization = Realization(
  name = "unknownField",
  invented = "value"
)
