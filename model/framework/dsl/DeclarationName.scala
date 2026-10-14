package framework.dsl

final private[framework] class DeclarationName:
  private var explicit = Option.empty[Boolean]
  private var source = Option.empty[String]
  private var spelling = Option.empty[String]
  private var anonymous = false

  def mark(value: Boolean, context: String): Unit =
    require(explicit.forall(_ == value), s"$context has conflicting name provenance")
    explicit = Some(value)

  def rename(): Unit = explicit = Some(true)

  def retain(id: String, spelling: String, context: String): Unit =
    if source.forall(_ == id) then
      if source.nonEmpty then
        require(this.spelling.forall(_ == spelling), s"$context has conflicting declaration names")
      if this.spelling.forall(_ == spelling) then
        source = Some(id)
        this.spelling = Some(spelling)

  def spelledAt(value: String): Unit =
    if spelling.isEmpty then spelling = Some(value)

  def identity(context: String): String =
    source
      .getOrElse(throw IllegalArgumentException(s"$context: no retained source identity"))

  def anonymousAt(id: String): Unit =
    if source.contains(id) then anonymous = true

  def effective(raw: String, context: String): String =
    if explicit.contains(true) || raw.nonEmpty then raw
    else if explicit.contains(false) then
      val spelling = this.spelling
        .getOrElse(
          throw IllegalArgumentException(s"$context: no retained declaration name")
        )
      require(
        !spelling.contains('$') && !anonymous,
        s"$context: declaration name $spelling is a name the compiler made up"
      )
      spelling
    else throw IllegalArgumentException(s"$context: no retained name provenance")

  def copyTo(value: DeclarationName): Unit =
    value.explicit = explicit
    value.source = source
    value.spelling = spelling
    value.anonymous = anonymous

private[framework] trait RetainedDeclarationName:
  def name: String
  private[framework] val declarationName = new DeclarationName
  private def context: String = s"${getClass.getSimpleName.toLowerCase} $name"
  @scala.annotation.publicInBinary
  private[framework] def retainName(id: String, spelling: String): this.type =
    declarationName.retain(id, spelling, context)
    this
  @scala.annotation.publicInBinary
  private[framework] def retainExplicitName(explicit: Boolean): this.type =
    declarationName.mark(explicit, context)
    this
  @scala.annotation.publicInBinary
  private[framework] def retainAnonymousName(id: String): this.type =
    declarationName.anonymousAt(id)
    this
  @scala.annotation.publicInBinary
  private[framework] def retainCopyName(explicit: Boolean): this.type =
    if explicit then declarationName.rename()
    this
  @scala.annotation.publicInBinary
  private[framework] def sourceName: String =
    declarationName.effective(name, context)
