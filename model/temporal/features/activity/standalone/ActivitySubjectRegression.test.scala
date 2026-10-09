package framework

import temporal.features.activity.standalone.system.{ActivitySystem, Completion}
import temporal.features.activity.standalone.system
import framework.outcomes.Outcome

class ActivitySubjectRegression extends munit.FunSuite:
  test("ordinary zero rebind preserves every lifecycle state and ordered transition result") {
    val subject = Completion
    assertEquals(subject.fs.values, ActivitySystem.fs.values)
    assertEquals(subject.fo.values, ActivitySystem.fo.values)
    assertEquals(subject.ff.values, ActivitySystem.ff.values)
    assertEquals(subject.init, ActivitySystem.init)
    assertEquals(subject.table, ActivitySystem.table)
    val original = ActivitySystem.bindings
    val rebound = subject.bindings
    assertEquals(original.size, 21)
    assertEquals(rebound.map(_.decl), original.map(_.decl))
    val catalog = original.zip(rebound).flatMap { (before, after) =>
      val inputs = classesOf(before.decl)
      assertEquals(classesOf(after.decl), inputs)
      val oldStep = effectOf[system.State, Outcome, system.Fact](before.decl, before.function)
      val newStep = effectOf[system.State, Outcome, system.Fact](after.decl, after.function)
      inputs.map(values => (values, oldStep, newStep))
    }
    assertEquals(catalog.size, 119)
    assertEquals(subject.fs.values.size, 5616)
    val (visited, enabled) = subject.fs.values.foldLeft(0 -> 0) { case (counts, state) =>
      assertEquals(subject.end(state), ActivitySystem.end(state))
      assertEquals(subject.sourcePhasing.phase(state), ActivitySystem.phased.phase(state))
      catalog.foldLeft(counts) { case ((seen, enabled), (inputs, oldStep, newStep)) =>
        val expected = oldStep(state, inputs)
        assertEquals(newStep(state, inputs), expected)
        (seen + 1) -> (enabled + (if expected.nonEmpty then 1 else 0))
      }
    }
    assertEquals(visited, 668304)
    assert(enabled > 0 && enabled < visited)
  }
