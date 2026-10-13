package framework.dsl

import fixture.facetcontract.*

@main def nativeRows(): Unit =
  val states = Finite[State].values
  assert(Faceted.bindings.map(_.decl) == Whole.bindings.map(_.decl))
  assert(Faceted.init == Whole.init)
  var count = 0
  for (left, right) <- Faceted.bindings.zip(Whole.bindings); state <- states do
    val first = left.function.asInstanceOf[State => List[Step[State, Outcome, Fact]]]
    val second = right.function.asInstanceOf[State => List[Step[State, Outcome, Fact]]]
    assert(first(state) == second(state))
    assert(Faceted.end(state) == Whole.end(state))
    count += 1
  assert(count == 6)
  assert(
    Faceted.bindings.exists(b =>
      b.function
        .asInstanceOf[State => List[Step[State, Outcome, Fact]]](State(Phase.off))
        .exists(_.facts == List(Fact.changed))
    )
  )
  println("PASS: 6 complete native state/action pairs, outcomes, ordered facts, starts and ends")
