package framework.derivationsnative

import framework.dsl.*

object Runtime:
  private def observe[S, O, F](module: Machine[S, O, F]): Unit =
    module.table.zipWithIndex.foreach { case ((decl, bound), index) =>
      val run = effectOf[S, O, F](decl, stepFunction(decl, bound))
      module.fs.values.foreach { state =>
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
        .asInstanceOf[Machine[?, ?, ?]]
    )
