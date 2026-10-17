package samples.kettle

import plain.*

class TopLevelKettleTest extends munit.FunSuite:
  test("the top-level form has the nested form's schema, reachable graph and verdicts"):
    val top = Runtime(Kettle)
    val nested = Runtime(proto.Kettle.Model)
    assertEquals(top.schema.map(_.toString), nested.schema.map(_.toString))
    val (topStarts, topStates, topEdges) = top.explore()
    val (starts, states, edges) = nested.explore()
    assertEquals((topStarts, topStates, topEdges.size), (starts, states, edges.size))
    assertEquals(topStates.size, 283)
    assertEquals(top.verifyAll(), nested.verifyAll())
    assertEquals(
      top.progress(Kettle.progress.boils),
      nested.progress(proto.Kettle.Model.progress.boils)
    )
