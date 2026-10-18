package framework.`export`

class FacetAdmissionTest extends munit.FunSuite:
  test(
    "both default facet guards hold every independently selected declared machine without overrides"
  ):
    Switches.within(Set.empty) {
      for owner <- List(
          "temporal.actors.worker.Polling",
          "temporal.foundations.taskqueue.product.TaskQueueProduct",
          "temporal.foundations.taskqueue.protocol.TaskQueueProtocol",
          "temporal.features.activity.mini.product.ActivityProduct",
          "temporal.features.activity.mini.protocol.ActivityProtocol",
          "temporal.features.activity.mini.protocol.AnswerDispatch"
        )
      do
        assert(Switches.facetsOnly(owner), owner)
        assert(Switches.facetVocabulary(owner), owner)
        assert(!Switches.layout(owner), owner)
      assert(!Switches.facetsOnly("fixture.design.Probe"))
      assert(!Switches.facetVocabulary("fixture.design.Probe"))
    }
