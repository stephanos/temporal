package framework.realize

final case class Realization(
    name: String,
    scripts: Vector[Script] = Vector.empty,
    invented: String = ""
)

final case class Script(id: String, activation: Activation)

enum Activation:
  case Unknown
