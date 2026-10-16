package framework.dsl

import scala.quoted.*
import scala.annotation.unused

private[framework] object RuleSource:
  inline def spelling[A](inline target: A, inline index: Int = 0): String =
    ${ spellingImpl('target, 'index) }

  private def spellingImpl[A: Type](@unused target: Expr[A], index: Expr[Int])(using
      Quotes
  ): Expr[String] =
    val source = quotes.reflect.Position.ofMacroExpansion.sourceCode.getOrElse("")
    Expr(arguments(source).lift(index.valueOrAbort).map(_._1).getOrElse(source))

  inline def qualified[A](
      inline target: A,
      form: String,
      declarer: String,
      inline index: Int = 0
  ): Unit =
    ${ qualifiedImpl('target, 'form, 'declarer, 'index) }

  inline def disabledLabels(inline targets: Seq[Action[?]]): Vector[String] =
    ${ disabledImpl('targets) }

  inline def imports(inline body: Unit): String = ${ importsImpl('body) }

  private def position(using q: Quotes)(at: q.reflect.Position): String =
    s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}"

  private def qualifiedImpl[A: Type](
      target: Expr[A],
      form: Expr[String],
      declarer: Expr[String],
      index: Expr[Int]
  )(using
      Quotes
  ): Expr[Unit] =
    import quotes.reflect.*
    val at = Position.ofMacroExpansion
    val call =
      at.sourceCode.getOrElse(report.errorAndAbort("rule target has no source spelling", target))
    val (source, offset) = arguments(call)
      .lift(index.valueOrAbort)
      .getOrElse(report.errorAndAbort("rule target has no source argument", target))
    val start = at.start + offset
    val located = Position(at.sourceFile, start, start + source.length)
    val location = position(located)
    def actionOwner(term: Term): Option[String] = term match
      case Inlined(_, _, body) => actionOwner(body)
      case Typed(body, _)      => actionOwner(body)
      case ref: Ref if ref.tpe.widen.baseClasses.exists(_.fullName == "framework.dsl.Action") =>
        val owner = ref.symbol.owner
        Option.when(owner.flags.is(Flags.Module))(owner.name.stripSuffix("$"))
      case Select(body, "apply") => actionOwner(body)
      case TypeApply(body, _)    => actionOwner(body)
      case Apply(body, args)     => actionOwner(body).orElse(args.headOption.flatMap(actionOwner))
      case _                     => None
    val owner = actionOwner(target.asTerm).fold(declarer)(Expr(_))
    '{ validate(${ Expr(source) }, ${ Expr(location) }, $form, $owner) }

  private def disabledImpl(targets: Expr[Seq[Action[?]]])(using Quotes): Expr[Vector[String]] =
    targets match
      case Varargs(values) =>
        val spelling = quotes.reflect.Position.ofMacroExpansion.sourceCode.getOrElse("")
        val labels = arguments(spelling).map((text, _) => writtenAction(text))
        values.zipWithIndex.foldRight('{ Vector.empty[String] }) { (entry, rest) =>
          val (target, index) = entry
          val checked = qualifiedImpl(target, Expr("disabled"), Expr("<declarer>"), Expr(index))
          '{ $checked; ${ Expr(labels(index)) } +: $rest }
        }
      case _ => quotes.reflect.report.errorAndAbort("disabled targets must be written at the call")

  private def arguments(call: String): Vector[(String, Int)] =
    val result = Vector.newBuilder[(String, Int)]
    var start = call.indexOf('(') + 1
    var index = start
    var depth = 0
    var quoted = false
    var escaped = false
    var lineComment = false
    var blockComment = false
    var done = false
    while index < call.length && !done do
      val c = call(index)
      val next = if index + 1 < call.length then call(index + 1) else ' '
      if lineComment then lineComment = c != '\n'
      else if blockComment then
        if c == '*' && next == '/' then
          blockComment = false
          index += 1
      else if quoted then
        if escaped then escaped = false
        else if c == '\\' then escaped = true
        else if c == '"' then quoted = false
      else if c == '/' && next == '/' then
        lineComment = true
        index += 1
      else if c == '/' && next == '*' then
        blockComment = true
        index += 1
      else if c == '"' then quoted = true
      else if c == '(' || c == '[' || c == '{' then depth += 1
      else if c == ')' && depth == 0 || c == ',' && depth == 0 then
        val raw = call.substring(start, index)
        val leading = raw.takeWhile(_.isWhitespace).length
        result += raw.trim -> (start + leading)
        start = index + 1
        done = c == ')'
      else if c == ')' || c == ']' || c == '}' then depth -= 1
      index += 1
    result.result()

  private def importsImpl(body: Expr[Unit])(using Quotes): Expr[String] =
    import quotes.reflect.*
    val found = scala.collection.mutable.ArrayBuffer.empty[String]
    val scan = new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case i: Import => found += position(i.pos): Unit
        case _         => super.traverseTree(tree)(owner)
    scan.traverseTree(body.asTerm)(Symbol.spliceOwner)
    Expr(found.headOption.getOrElse(""))

  private[framework] def validate(
      source: String,
      location: String,
      form: String,
      declarer: String
  ): Unit =
    val spelling = source.replaceAll("(?s)/\\*.*?\\*/|//[^\\n]*", " ")
    val action = writtenAction(spelling)
    require(
      writtenPath(spelling).contains('.'),
      s"$form($action) names its action bare at $location: name it through the object that " +
        s"declares it, `$form($declarer.$action)`"
    )
