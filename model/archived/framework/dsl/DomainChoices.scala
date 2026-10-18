package framework.dsl

@scala.annotation.publicInBinary
private[framework] object DomainChoices:
  def create[A](values: Seq[A])(using structural: Data.Structural[A]): Finite[A] =
    require(values.nonEmpty, "an explicit input domain must not be empty")
    require(
      !values.indices.exists(i => values.take(i).exists(structural.equivalent(_, values(i)))),
      "an explicit input domain must not contain duplicate values"
    )
    Finite.of(values*)

  def range(values: Range): Finite[Int] = create(values)
