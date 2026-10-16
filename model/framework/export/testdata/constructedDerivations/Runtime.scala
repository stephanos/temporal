package framework.derivationsnative

import framework.dsl.*

object Runtime:
  // Native state has no catalog to walk (fn-174): each fixture names its example states, in the
  // order the frozen callback observations list them. Facet fixtures declare theirs beside their
  // state, which this observer does not compile against.
  private def examples(root: String): Vector[Any] =
    if root.startsWith("fixture.derivationfacets.") then
      val examples = java.lang.Class.forName("fixture.derivationfacets.Examples$")
      examples
        .getMethod("states")
        .invoke(examples.getField("MODULE$").get(null))
        .asInstanceOf[Vector[Any]]
    else Vector(false, true)
  private def observe[S, O, F](module: Machine[S, O, F], states: Vector[Any]): Unit =
    module.table.zipWithIndex.foreach { case ((decl, bound), index) =>
      val run = effectOf[S, O, F](decl, stepFunction(decl, bound))
      states.map(_.asInstanceOf[S]).foreach { state =>
        classesOf(decl).foreach { inputs =>
          println(s"$index:$state:${inputs.mkString(",")}:${run(state, inputs)}")
        }
      }
    }
  def main(args: Array[String]): Unit =
    observe(
      java.lang.Class
        .forName(args(0) + "$")
        .getField("MODULE$")
        .get(null)
        .asInstanceOf[Machine[?, ?, ?]],
      examples(args(0))
    )
