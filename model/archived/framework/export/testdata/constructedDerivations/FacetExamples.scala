package fixture.derivationfacets

// The example states the native callback observer (Runtime.scala) runs each facet machine's
// bindings in, in the frozen observations' order: native state has no catalog to walk (fn-174).
// A file of its own, so the combined fixture keeps every frozen position.
object Examples:
  def states: Vector[Any] =
    for left <- Vector(Phase.open, Phase.done); right <- Vector(Phase.open, Phase.done)
    yield State(left, right)
