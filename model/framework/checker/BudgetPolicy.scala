package framework.checker

final class BudgetPolicy private (
    val default: Int,
    val overrides: Map[String, Int],
    source: String
):
  def forSuite(name: String): Int = overrides.getOrElse(name, default)

  def validate(suites: Set[String]): Unit =
    val unknown = overrides.keySet -- suites
    require(
      unknown.isEmpty,
      s"$source: unknown suite override(s): ${unknown.toVector.sorted.mkString(", ")}"
    )

  def normalized: String =
    (s"default=$default" +: overrides.toVector.sortBy(_._1).map((name, value) => s"$name=$value"))
      .mkString("\n")

object BudgetPolicy:
  val defaultBudget: Int = 1 << 20
  val environmentName: String = "UMPIRE_CHECKER_BUDGETS"

  def fromEnvironment(environment: Map[String, String]): BudgetPolicy =
    parse(environment.get(environmentName))

  def parse(configuration: Option[String], source: String = environmentName): BudgetPolicy =
    val entries = configuration match
      case None       => Vector.empty
      case Some(text) =>
        require(text.trim.nonEmpty, s"$source: an explicit budget policy must contain entries")
        text.linesIterator.zipWithIndex
          .filter(_._1.trim.nonEmpty)
          .map: (line, index) =>
            val parts = line.split("=", -1).map(_.trim)
            require(
              parts.length == 2,
              s"$source:${index + 1}: expected default=budget or suite.name=budget"
            )
            val key = parts(0)
            require(
              key == "default" || key.matches(
                "[A-Za-z_$][A-Za-z0-9_$]*(\\.[A-Za-z_$][A-Za-z0-9_$]*)*"
              ),
              s"$source:${index + 1}: expected default or exact suite name, got '$key'"
            )
            val budget = parts(1).toIntOption
              .filter(_ > 0)
              .getOrElse(
                throw IllegalArgumentException(
                  s"$source:${index + 1}: budget for $key must be a positive Int, got '${parts(1)}'"
                )
              )
            key -> budget
          .toVector
    val duplicates =
      entries.groupMap(_._1)(_._2).collect { case (name, values) if values.size > 1 => name }
    require(
      duplicates.isEmpty,
      s"$source: duplicate budget entry(s): ${duplicates.toVector.sorted.mkString(", ")}"
    )
    val values = entries.toMap
    new BudgetPolicy(values.getOrElse("default", defaultBudget), values - "default", source)
