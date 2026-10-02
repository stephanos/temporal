package umpire.realize

import umpire.ClassRef

final case class Alternative(name: String, priority: Int, actions: Vector[ClassRef])
final case class Variation(index: Int, choices: Vector[Alternative])
final case class Exploration(
    name: String,
    variations: Vector[Variation],
    runs: Int,
    edits: Int,
    dropPrefix: Boolean
)
