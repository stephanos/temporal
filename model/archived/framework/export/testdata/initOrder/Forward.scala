// (a): a val read while its object initializes, before the object declares it. Beside the reads
// the lint refuses, those that initialize nothing yet: in a def, a lambda, a by-name argument, a
// lazy val and an object declared but not read, and a constant the compiler writes in place.
package fixture.features.initorder

object Forward:
  val early = late + 1
  def inDef = late
  val inLambda = () => late
  val byName = Option(1).getOrElse(late)
  lazy val inLazy = late
  object unread:
    val held = late
  val constant = fixed + 1
  final val fixed = 2
  // A context function is applied where it is written, as a DSL block that takes one is.
  val inContext = applied(late)
  val late = 1

def applied(body: Int ?=> Int): Int = body(using 0)

// (d): a Model declaration beside the feature file.
def stray = Switch.scenario.actions(flip)

// (d): a Model declaration in a type's companion beside the feature file.
final case class Spare(lit: Boolean)

object Spare:
  def strayInCompanion = Switch.scenario.actions(flip)
