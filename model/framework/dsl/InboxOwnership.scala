package framework.dsl

private[framework] object InboxOwnership:
  final case class Field(owner: String, caseName: String, name: String) derives CanEqual:
    def context: String =
      (Seq(owner) ++ Option.when(caseName.nonEmpty)(caseName) :+ name).mkString(".")

  final class Entry(
      val field: Field,
      val message: String,
      val declaration: String,
      supplier: () => Channel[?]
  ):
    lazy val channel: Channel[?] =
      val value = supplier()
      require(value ne null, "Channel supplier is not initialized")
      value
    def agrees(other: Entry): Boolean =
      field == other.field && message == other.message && declaration == other.declaration
