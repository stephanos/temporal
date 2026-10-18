package framework.check
package lint

import java.nio.file.{Files, Path}
import scala.annotation.tailrec
import scala.jdk.CollectionConverters.*

// The framework's sugar stays in the files named `Syntax.scala`, each definition documented with the
// core form it stands for, and no core file reaches it. `make lint-model` holds the Models, the
// framework and the lifter to three rules (`gate --check-syntax`):
//
//   - Each definition of a `Syntax.scala` under model/framework, model/temporal or the lifter's
//     model/framework/export that is top-level, a member of an object or an extension method (in
//     the lifter also a member of a top-level class or trait) has a doc comment directly before it,
//     with only blank lines and annotations between, that says `Core form:` and, after it, the core
//     spelling in backticks. Private definitions are helpers and exempt.
//   - No other file of model/framework, model/temporal or the lifter (its own files, not its
//     fixtures or tests) defines a sugar name at the top level, in an object or as an extension
//     method. Members of a class, trait or enum, constructor parameters and locals are not sugar.
//   - A core file, the files of a tree beside its `Syntax.scala` (model/framework with the DSL's,
//     model/framework/realize with the realization sugar's, model/temporal/realize with the
//     Temporal kit's, and the lifter), imports no `Syntax` module and no name that file defines. In
//     the first three it names none of them either, outside comments and strings: top-level
//     sugar of the package needs no import there. A name the core of the
//     tree declares itself, as a member, parameter or local, reads as that declaration and is left
//     alone, and so does a member selection, `flags.is(...)`, of a name the sugar defines only
//     outside an extension.
//
// The tools beside the Models (ModelSources.tool: the lifter, the checks, the migration harness) are
// no Model source: only the lifter's own files are held, as their tree. The one-file framework
// samples are authored Models, not framework core, so they may use the public syntax they teach.
//
// The Models are scalafmt-formatted Scala 3 with significant indentation, so a definition's place is
// read from the indentation of the lines that open its containers, on the source with its comments
// and strings blanked (the literal scanner (Literals.scala)).
object SyntaxRule:
  // The sugar of the framework and of the Temporal kit: no file but a `Syntax.scala` defines them.
  val sugarNames: Set[String] = Set(
    "implies",
    "in",
    "records",
    "enter",
    "Ok",
    "stay",
    "disabled",
    "once",
    "keeps",
    "never",
    "from",
    "stays",
    "unless",
    "sticky",
    "stickyAcross",
    ":=",
    "field",
    "reject",
    "rejects",
    "effect",
    "expect",
    "check",
    "oneOf",
    "branch",
    "before",
    "after",
    "afterAction",
    "across",
    "is",
    "record",
    "on",
    "where",
    "when",
    "always",
    "Case",
    "Firing",
    "Rules",
    "PhasesOf",
    "Phased",
    "proto"
  )

  val remainingExportSugar: Map[String, (String, Set[String])] = Map(
    "Syntax.scala" -> (
      "fn-141.8/.9/.10/.12",
      Set(
        "in",
        "enters",
        "record",
        "reject",
        "rejects",
        "Rules",
        "witness",
        "records",
        "recordsWhen",
        "excludes",
        "ends",
        "limits",
        "live",
        "explore",
        "is",
        "effect",
        "proto",
        "field",
        ":="
      )
    ),
    "Declarations.scala" -> (
      "fn-141.10",
      Set(
        "Rules",
        "Case",
        "Firing",
        "Phased",
        "PhasesOf",
        "on",
        "when",
        "where",
        "always",
        "in",
        "enters",
        "rejects",
        "is",
        "effect",
        "records",
        "disabled",
        "from"
      )
    ),
    "Expressions.scala" -> ("fn-141.10", Set("is", "effect", "Ok")),
    "Claims.scala" -> ("fn-141.12", Set("in", "when", "from", "witness", "records", "excludes")),
    "Capabilities.scala" -> ("fn-141.12/.13", Set("records", "from", "when")),
    "Context.scala" -> ("fn-141.10", Set("Rules", "Phased", "is", "effect")),
    "Compositions.scala" -> ("fn-141.11", Set("field", "records")),
    "Expansion.scala" -> ("fn-141.10", Set("Ok")),
    "Realizations.scala" -> ("fn-141.8", Set("field", ":=", "proto", "records", "from", "when")),
    "Roles.scala" -> ("fn-141.10", Set("Phased", "PhasesOf", "when", "in"))
  )

  private def exportSugar(files: Vector[Source]): Vector[Finding] =
    val names =
      sugarNames ++ Set("Once", "Never", "Stays", "enters", "witness", "recordsWhen", "excludes")
    val exports = files.filter(_.file.startsWith("model/framework/export/"))
    val referenced = names -- exports.flatMap(_.declared)
    val selectable = files
      .filter(_.file == "model/framework/dsl/Syntax.scala")
      .flatMap(sugar)
      .filter(_.isExtension)
      .flatMap(_.name)
      .toSet
    exports.flatMap { file =>
      val allowed = remainingExportSugar
        .get(file.file.stripPrefix("model/framework/export/"))
        .fold(Set.empty[String])(_._2)
      def finding(name: String, line: Int) = Finding(
        file.file,
        line,
        s"syntax rule: the lifter names `$name`, a sugar definition: expand its indexed definition or export its constructed core value"
      )
      val strings = Literals.literals(file.source).values.flatMap { literal =>
        val name = literal.value.split('.').lastOption.getOrElse("").stripSuffix("$")
        // Bare before/after are core parameter identities; qualified names identify DSL symbols.
        val parameterName = Set("before", "after")(literal.value)
        Option.when(names(name) && !allowed(name) && !parameterName)(finding(name, literal.line))
      }
      val code = token
        .findAllMatchIn(file.code)
        .flatMap { matched =>
          val name = unquoted(matched.matched)
          val selected = matched.start > 0 && file.code(matched.start - 1) == '.'
          val prefix = file.code.substring(0, matched.start)
          val qualified = Seq("framework.dsl.", "framework.realize.", "Syntax.", "Syntax$package.")
            .exists(prefix.endsWith)
          Option.when(
            !allowed(name) &&
              ((referenced(name) && (!selected || selectable(name))) || (qualified && names(name)))
          )(
            finding(name, file.lineAt(matched.start))
          )
        }
        .toVector
      strings ++ code
    }

  final case class Finding(file: String, line: Int, reason: String):
    override def toString: String = s"$file:$line: $reason"

  // Where a definition sits.
  enum Place derives CanEqual:
    // Top-level, a member of an object, or an extension method.
    case Sugar

    // A member of a top-level class or trait.
    case TemplateMember

    // Anything deeper: a member of a class, a parameter, a local.
    case Inner

  // A definition: its keyword and name (none for an anonymous given), its line, the offsets its
  // doc comment may end before, whether it is private, its place, and whether it is an extension
  // method, the one kind of sugar a member selection (`x.name`) reaches.
  final case class Definition(
      keyword: String,
      name: Option[String],
      line: Int,
      docAt: Seq[Int],
      isPrivate: Boolean,
      place: Place,
      isExtension: Boolean = false
  ):
    def shown: String = name.fold("an anonymous given")(n => s"`$n`")

  private enum Kind derives CanEqual:
    case Object, Template, Extension, Other

  private val callableViews =
    Set("Across", "Expecting", "RequirementFacts", "WitnessFacts")

  // A line that opens a block: its indentation, what it opens, where it starts, and, for an
  // extension, whether a definition of its block was read yet.
  final private case class Container(indent: Int, kind: Kind, header: Int, defined: Boolean)

  private val identifier = "[A-Za-z_$][A-Za-z0-9_$]*"
  private val operator = "[!#%&*+\\-/:<=>?@\\\\^|~]+"
  private val name = s"(?:$identifier|`[^`]+`|$operator)"
  private val modifier =
    "(?:private(?:\\[[^\\]]*\\])?|protected(?:\\[[^\\]]*\\])?|final|override|inline|infix|" +
      "transparent|implicit|lazy|opaque|sealed|abstract|open|case|erased)"
  private val annotations = "(?:@[A-Za-z_][A-Za-z0-9_.]*(?:\\[[^\\]]*\\])?(?:\\([^)]*\\))*\\s*)*"
  private val definitionLine =
    s"$annotations((?:$modifier\\s+)*)(def|val|var|given|type|class|trait|object|enum)\\b\\s*(.*)".r
  private val extensionLine = s"${annotations}extension\\b(.*)".r
  private val nameAt = name.r
  private val givenName = s"($identifier)\\s*(?:\\[[^\\]]*\\])?\\s*(?:\\([^)]*\\)\\s*)*:(?![:=])".r
  private val docGap = s"\\s*$annotations".r
  private val lineStart = "[ \\t]*".r
  private val adjacent = "[ \\t]*\\n[ \\t]*".r
  private val coreForm = "(?s)Core form:.*?`[^`\\n]+`".r
  private val parameter =
    s"(?m)(?:[(,]|^)\\s*(?:using\\s+)?(?:$modifier\\s+)*(?:val\\s+|var\\s+)?($identifier)\\s*:(?![:=])".r
  private val clause = "(?m)^[ \\t]*(?:import|export)\\s".r
  private val token = s"$identifier|`[^`]+`|$operator".r
  private val declares =
    "\\b(?:def|val|var|given|type|class|trait|object|enum|case)\\s*$".r.unanchored

  private def unquoted(name: String) = name.stripPrefix("`").stripSuffix("`")

  // The offset of the bracket that closes the one `text` starts with.
  private def closing(text: String): Option[Int] =
    val depths = text
      .scanLeft(0): (depth, c) =>
        if "([{".indexOf(c) >= 0 then depth + 1
        else if ")]}".indexOf(c) >= 0 then depth - 1
        else depth
      .tail
    Some(depths.indexWhere(_ == 0))
      .filter(_ >= 0)

  // What follows an extension's type and value parameters on its line; none while they go on.
  @tailrec private def afterParameters(text: String): Option[String] =
    val rest = text.dropWhile(
      _ == ' '
    )
    if rest.startsWith("(") || rest.startsWith("[") then
      closing(rest) match
        case Some(end) => afterParameters(rest.substring(end + 1))
        case None      => None
    else Some(rest)

  private def definition(
      text: String,
      line: Int,
      docAt: Seq[Int],
      place: Place,
      isExtension: Boolean
  ) = text match
    case definitionLine(modifiers, keyword, rest) =>
      val named =
        if keyword == "given" then
          givenName
            .findPrefixMatchOf(rest)
            .map(
              _.group(1)
            )
        else nameAt.findPrefixMatchOf(rest).map(_.matched)
      Some(
        Definition(
          keyword,
          named.map(unquoted),
          line,
          docAt,
          modifiers.contains("private"),
          place,
          isExtension
        )
      )
    case _ => None

  // The source with its comments and strings blanked: offsets and line breaks unchanged.
  def code(source: String): String =
    val blanked = Literals.code(source, Literals.literals(source))
    blanked.indices
      .map(i => if source(i) == '\n' then '\n' else blanked(i))
      .mkString

  // The definitions of a source whose comments and strings are blanked, each at its place.
  def definitions(code: String): Vector[Definition] =
    val texts = code.split("\n", -1).toVector
    val starts = texts.scanLeft(0)((at, text) => at + text.length + 1)
    val (_, found) = texts.indices.foldLeft((List.empty[Container], Vector.empty[Definition])):
      case ((stack, found), index) =>
        val text = texts(index)
        val trimmed = text.trim
        // A closing bracket goes on the line that opened it, and a package clause opens nothing.
        if trimmed.isEmpty || ")]}".indexOf(trimmed.head) >= 0 || trimmed.startsWith("package ")
        then (stack, found)
        else
          val (indent, start, line) = (
            text.indexWhere(_ != ' '),
            starts(index),
            index + 1
          )
          val open = stack.dropWhile(_.indent >= indent)
          val place = open match
            case Nil                                 => Place.Sugar
            case top :: _ if top.kind == Kind.Object =>
              Place.Sugar
            case top :: _ if top.kind == Kind.Extension =>
              Place.Sugar
            case top :: Nil if top.kind == Kind.Template =>
              Place.TemplateMember
            case _ => Place.Inner
          def opened(kind: Kind) = Container(indent, kind, start, defined = false)
          trimmed match
            case extensionLine(rest) =>
              afterParameters(rest).flatMap(
                definition(_, line, Seq(start), Place.Sugar, isExtension = true)
              ) match
                case Some(inline) => (opened(Kind.Other) :: open, found :+ inline)
                case None         => (opened(Kind.Extension) :: open, found)
            case _ =>
              // The first definition of an extension's block may be documented above the extension.
              val docAt = open match
                case top :: _ if top.kind == Kind.Extension && !top.defined =>
                  Seq(start, top.header)
                case _ => Seq(start)
              val inExtension = open.headOption.exists(_.kind == Kind.Extension)
              definition(trimmed, line, docAt, place, inExtension) match
                case Some(defined) =>
                  // An object local to a body is a local too; its members are no sugar.
                  val kind = defined.keyword match
                    case "object" if place != Place.Inner =>
                      Kind.Object
                    case "class" if defined.name.exists(callableViews) => Kind.Extension
                    case "class" | "trait" | "enum"                    => Kind.Template
                    case _                                             => Kind.Other
                  val marked = open match
                    case top :: rest if top.kind == Kind.Extension =>
                      top.copy(defined = true) :: rest
                    case other => other
                  (opened(kind) :: marked, found :+ defined)
                case None => (opened(Kind.Other) :: open, found)
    found

  // A source file, read and blanked once.
  final private case class Source(
      file: String,
      source: String,
      code: String,
      comments: Vector[(Int, Int)]
  ):
    lazy val definitions: Vector[Definition] = SyntaxRule.definitions(code)
    def isSyntax: Boolean = file == "Syntax.scala" || file.endsWith(
      "/Syntax.scala"
    )
    def lineAt(offset: Int): Int = code
      .substring(0, offset)
      .count(
        _ == '\n'
      ) + 1

    // The doc comment that ends right before `at`, with blank lines and annotations between: the run
    // of `//` lines, each on a line of its own, that ends there.
    def doc(at: Int): Option[String] =
      val own = comments.filter: (start, _) =>
        source.startsWith("//", start) && lineStart.matches(code.substring(lineOf(start), start))
      own
        .filter(_._2 <= at)
        .lastOption
        .filter((_, end) => docGap.matches(code.substring(end, at)))
        .map: (last, end) =>
          @tailrec def first(start: Int, earlier: Vector[(Int, Int)]): Int =
            earlier.lastOption match
              case Some((previous, until)) if adjacent.matches(code.substring(until, start)) =>
                first(previous, earlier.init)
              case _ => start
          source.substring(first(last, own.takeWhile(_._1 < last)), end)

    private def lineOf(offset: Int): Int = code.lastIndexOf('\n', offset - 1) + 1

    // The import and export clauses: where each starts and ends.
    def clauses: Vector[(Int, Int)] =
      clause
        .findAllMatchIn(code)
        .map: found =>
          val newline = code.indexOf('\n', found.start)
          val lineEnd = if newline < 0 then code.length else newline
          val brace = code.indexOf('{', found.start)
          val end =
            if brace < 0 || brace > lineEnd then lineEnd
            else closing(code.substring(brace)).fold(code.length)(brace + _ + 1)
          found.start -> end
        .toVector

    // The names this file declares anywhere: definitions, parameters and locals.
    def declared: Set[String] =
      val patterns = "(?m)\\bcase\\s+([^\\n]*?)=>".r
        .findAllMatchIn(code)
        .flatMap: matched =>
          val pattern = matched.group(1).split("\\s+if\\s+", 2).head
          "(?<![\\w.])([a-z_$][A-Za-z0-9_$]*)(?![\\w])(?!\\s*[(.])".r
            .findAllMatchIn(pattern)
            .map(_.group(1))
      definitions
        .flatMap(_.name)
        .toSet ++ parameter.findAllMatchIn(code).map(_.group(1)) ++ patterns

  private def read(root: Path, path: Path): Source =
    val source = Files.readString(path)
    val file = root.relativize(path).iterator.asScala.mkString("/")
    Source(file, source, code(source), Literals.literals(source).comments)

  // The Scala sources of a directory: its own, not its tests, fixtures or build output.
  private def sources(root: Path, directory: String, recursive: Boolean): Vector[Path] =
    val base = root.resolve(directory)
    if !Files.isDirectory(base) then Vector.empty
    else
      val skipped = Set("test", "testdata", "build", ".scala-build", ".bsp")
      val stream = if recursive then Files.walk(base) else Files.list(base)
      try
        stream.iterator.asScala
          .filter: path =>
            val name = path.getFileName.toString
            Files.isRegularFile(path) && name.endsWith(".scala") && !name.endsWith(".test.scala") &&
            !base.relativize(path).iterator.asScala.toSeq.init.exists(p => skipped(p.toString))
          .toVector
          .sortBy(_.toString)
      finally stream.close()

  // A tree of the framework: its directory, and whether its core is held to the names of its sugar.
  final private case class Tree(
      directory: String,
      syntax: String,
      recursive: Boolean,
      references: Boolean
  ):
    def holds(file: String): Boolean =
      file.startsWith(s"$directory/") &&
        !file.startsWith("model/framework/samples/") &&
        (if recursive then !ModelSources.tool(file)
         else !file.stripPrefix(s"$directory/").contains('/'))

  // The lifter's core reaches the hooks of its trait Syntax through its self-type, by name; only an
  // import of it is refused there.
  private val trees = Seq(
    Tree(
      "model/framework",
      "model/framework/dsl/Syntax.scala",
      recursive = true,
      references = true
    ),
    Tree(
      "model/framework/realize",
      "model/framework/realize/Syntax.scala",
      recursive = true,
      references = true
    ),
    Tree(
      "model/temporal/realize",
      "model/temporal/realize/Syntax.scala",
      recursive = true,
      references = true
    ),
    Tree(
      "model/framework/export",
      "model/framework/export/Syntax.scala",
      recursive = false,
      references = false
    )
  )

  private def home(file: String): String =
    if file.startsWith("model/framework/export/") then "model/framework/export/Syntax.scala"
    else if file.startsWith("model/framework/realize/") then "model/framework/realize/Syntax.scala"
    else if file.startsWith("model/temporal/") then "model/temporal/realize/Syntax.scala"
    else "model/framework/dsl/Syntax.scala"

  // The definitions of a `Syntax.scala` that are its sugar: public, at a place the rule reads.
  private def sugar(file: Source): Vector[Definition] =
    val lifter = file.file.startsWith("model/framework/export/")
    file.definitions.filter: d =>
      !d.isPrivate && (d.place == Place.Sugar || (lifter && d.place == Place.TemplateMember))

  private def undocumented(file: Source): Vector[Finding] =
    sugar(file).flatMap: d =>
      d.docAt.flatMap(file.doc).headOption match
        case None =>
          Some(
            Finding(
              file.file,
              d.line,
              s"syntax rule: ${d.shown} in a Syntax.scala has no doc comment right before it: " +
                "document it with `Core form:` and the core spelling in backticks, " +
                "// ... Core form: `...`., or make it private if it is a helper"
            )
          )
        case Some(doc) if coreForm.findFirstIn(doc).isEmpty =>
          Some(
            Finding(
              file.file,
              d.line,
              s"syntax rule: the doc comment of ${d.shown} in a Syntax.scala names no core form: " +
                "add `Core form:` followed by the core spelling in backticks"
            )
          )
        case Some(_) => None

  private def misplaced(file: Source): Vector[Finding] =
    file.definitions
      .filter(d => d.place == Place.Sugar && d.name.exists(sugarNames))
      .map: d =>
        Finding(
          file.file,
          d.line,
          s"syntax rule: ${d.shown} is a sugar name defined outside a Syntax.scala: move it into " +
            s"${home(file.file)} with a `Core form:` doc comment, or name it after what it declares"
        )

  private def reaching(tree: Tree, files: Vector[Source]): Vector[Finding] =
    val inTree = files.filter(f => tree.holds(f.file))
    val (syntax, core) = inTree.partition(
      _.file == tree.syntax
    )
    val names = syntax.flatMap(sugar).flatMap(_.name).toSet
    // A member selection, `flags.is(...)`, names a member of what precedes the dot; it reaches the
    // sugar only when the sugar is an extension method.
    val selectable = syntax.flatMap(sugar).filter(_.isExtension).flatMap(_.name).toSet
    val modules = Set("Syntax", "Syntax$package", "syntax")
    // A name the core declares itself reads as that declaration: `def in` of QueryOn is no sugar.
    val referenced = if tree.references then names -- core.flatMap(_.declared) else Set.empty
    core.flatMap: file =>
      val clauses = file.clauses
      val imports = clauses.flatMap: (start, end) =>
        val named = token
          .findAllIn(file.code.substring(start, end))
          .map(unquoted)
          .filter(n => modules(n) || names(n))
          .toVector
          .distinct
        Option.when(named.nonEmpty)(
          Finding(
            file.file,
            file.lineAt(start),
            s"syntax rule: a core file imports the sugar of ${tree.syntax} " +
              s"(${named.map(n => s"`$n`").mkString(", ")}): remove the import and write the core form"
          )
        )
      val uses = token
        .findAllMatchIn(file.code)
        .filter: found =>
          referenced(unquoted(found.matched)) &&
            (selectable(unquoted(found.matched)) || found.start == 0 ||
              file.code(found.start - 1) != '.') &&
            !clauses.exists((start, end) => found.start >= start && found.start < end) &&
            !declares.matches(file.code.substring(0 max (found.start - 16), found.start))
        .map: found =>
          Finding(
            file.file,
            file.lineAt(found.start),
            s"syntax rule: a core file uses `${unquoted(found.matched)}`, sugar defined in " +
              s"${tree.syntax}: write its core form instead (its doc comment names it after `Core form:`)"
          )
      imports ++ uses

  // What breaks the rules in the repository at `root`, each as `file:line: reason`, in order.
  def findings(root: Path, includes: Path => Boolean = _ => true): Vector[String] =
    val models = Vector("model/framework", "model/temporal")
      .flatMap(sources(root, _, recursive = true))
      .filterNot(path => ModelSources.tool(root.relativize(path).toString))
    val files = (models ++ sources(root, "model/framework/export", recursive = false))
      .filter(includes)
      .map(read(root, _))
    val (syntax, other) = files.partition(_.isSyntax)
    (syntax.flatMap(undocumented) ++ other.flatMap(misplaced) ++ trees.flatMap(
      reaching(_, files)
    ) ++ exportSugar(files))
      .sortBy(f => (f.file, f.line))
      .map(_.toString)
