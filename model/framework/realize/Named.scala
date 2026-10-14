package framework.realize

final case class Reference(spelling: String, declaration: String = "", location: String = "")

trait Named:
  private var reference = Option.empty[Reference]
  final def ref: Option[Reference] = reference
  final protected def copied[A <: Named](value: A): A =
    value.reference = reference
    value
  def named(spelling: String, declaration: String = "", location: String = ""): Named
  final protected def referenced[A <: Named](
      value: A,
      spelling: String,
      declaration: String,
      location: String
  ): A =
    value.reference = Some(Reference(spelling, declaration, location))
    value
