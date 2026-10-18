// What a change to a path of the repository puts in scope, and what every Scala suite of the model
// covers (fn-166.13): the vocabulary of a coverage registry, which `make model-check-changed`
// (ChangeCheck.scala) reads to turn a diff into the features to compile, lift and compare, and the
// suites to run. A system's registry is data, kept in one place (Temporal's:
// model/temporal/check/Coverage.scala), so a move of the tree updates its path rules and modules
// and nothing else.
package framework.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// What a changed path is part of.
enum Area derives CanEqual:
  // The DSL and the generic realization core.
  case Framework

  // The lifter (the IR generator) and its fixtures.
  case Lifter

  // The gate and the checks, their tests and fixtures.
  case Check

  // The migration harness, its rules and their mini proofs.
  case Migrate

  // Code every feature compiles against: foundations, actors, capabilities, realize, `Bounds`.
  case Shared

  // The build settings every Scala module reads.
  case Build

  // The model targets of the Makefile.
  case Make

  // The recorded rename maps of the equivalence baselines below.
  case Baselines

  // A feature's own sources or outputs; the feature is resolved from the path (ModelSources).
  case Feature

  // Prose: it selects nothing, unless a suite names the file among its paths (GuideSourcesTest).
  case Docs

  // Build output, scratch and everything outside the model's Scala: selects nothing.
  case Outside

// A suite's declared reach: the areas it reads, the features whose Models it compiles, lifts or
// reads (a feature that is behind skips it), and single files it reads outside those areas.
// `anyFeature` marks a suite that reads every compiled feature's sources or tables, whichever they
// are; a change to any feature selects it, and a behind feature never skips it.
final case class Covers(
    areas: Set[Area],
    features: Set[String] = Set.empty,
    paths: Set[String] = Set.empty,
    anyFeature: Boolean = false
):
  def and(more: Area*): Covers = copy(areas = areas ++ more)
  def reading(feature: String*): Covers = copy(features = features ++ feature)
  def files(path: String*): Covers = copy(paths = paths ++ path)
  def everyFeature: Covers = copy(anyFeature = true)

// A Scala module whose suites run in one JVM: where its suites are, the directories under them that
// hold another module's (`skipped`), and what scala-cli builds it from (the Models' sources come
// from ModelSources, for the features in scope).
final case class Module(
    name: String,
    suiteRoots: Seq[String],
    inputs: Seq[String],
    skipped: Seq[String] = Nil
) derives CanEqual:
  def models: Boolean = inputs.isEmpty

// The step-table baseline a current feature's meaning is compared with, and the recorded rename map
// from that baseline to the tree. A task that renames appends its entries to the map.
final case class Baseline(directory: String, renames: String)

// A suite as the sources declare it: its module, its file and its class name.
final case class Declared(module: Module, file: String, name: String)

// What a system supplies to the change-scoped check: its current features and their baselines, the
// area of each path, the modules and what every suite covers. A suite missing from `suites` fails the
// check (the command refuses to run), and so does a registered suite no source declares.
trait Registry:
  // Where the registry is written, for the check's messages.
  def source: String

  // The features that are not behind: a DSL change keeps them current.
  def current: Set[String]

  def baselines: Map[String, Baseline]

  // Whether a fresh lift that differs from a feature's checked-in IR is held rather than failed.
  def committedIrHeld: Boolean

  // A path's area: the first rule whose glob matches it (`*` within a segment, `**` across them).
  def rules: Seq[(String, Area)]

  // The areas whose change selects every current feature.
  def selectsEveryFeature: Set[Area]

  def modules: Seq[Module]

  def suites: Map[String, Covers]

  // A path's area by the rules above.
  def area(path: String): Area =
    rules
      .collectFirst { case (glob, area) if matches(glob, path) => area }
      .getOrElse(Area.Outside)

  // `**/` also matches no directory at all, so `**/*.md` matches `AGENTS.md`.
  private def matches(glob: String, path: String): Boolean =
    def segment(part: String) =
      part.split("\\*", -1).map(java.util.regex.Pattern.quote).mkString("[^/]*")
    val pattern = glob
      .split("\\*\\*/", -1)
      .map(_.split("\\*\\*", -1).map(segment).mkString(".*"))
      .mkString("(?:.*/)?")
    path.matches(pattern)

  def declared(root: Path): Vector[Declared] =
    modules.toVector.flatMap: module =>
      SuiteSources.declared(root, module)

private object SuiteSources:
  private case class Token(text: String, offset: Int, line: Int, indent: Int)
  private case class Scope(
      start: Int,
      end: Int,
      name: String,
      packaged: Boolean = false,
      runnable: Boolean = true
  )
  private case class Binding(
      start: Int,
      end: Int,
      from: String,
      to: String,
      excluded: Set[String] = Set.empty
  )
  private case class Definition(
      name: String,
      binaryName: String,
      owner: String,
      file: Path,
      offset: Int,
      parents: Vector[String],
      bindings: Vector[Binding],
      runnable: Boolean,
      section: Boolean = false,
      erasedMember: Boolean = false,
      line: Int = 0,
      sectionOffset: Int = -1,
      local: Boolean = false
  )
  private val identifiers =
    "'(?:\\\\.|[^'\\\\])'|`[^`]+`|[A-Za-z_$][A-Za-z0-9_$]*|=>|[^\\s]".r
  private val modifiers = Set(
    "abstract",
    "final",
    "private",
    "protected",
    "sealed",
    "open",
    "case",
    "implicit",
    "override",
    "inline",
    "transparent"
  )
  private val roots = Set("framework.dsl.ModelSuite", "munit.FunSuite")
  private val sections = Set("examples", "explorations", "conformance", "unit")

  private def files(root: Path, module: Module): Vector[Path] =
    (module.inputs ++ module.suiteRoots).toVector.distinct
      .flatMap: input =>
        val base = root.resolve(input)
        if Files.isRegularFile(base) then Vector(base)
        else if !Files.isDirectory(base) then Vector.empty
        else
          val stream = Files.walk(base)
          try stream.iterator.asScala.filter(Files.isRegularFile(_)).toVector
          finally stream.close()
      .filter: path =>
        val relative = root.relativize(path)
        path.toString.endsWith(".scala") &&
        !relative.iterator.asScala.exists(p =>
          Set("testdata", "build", ".scala-build", ".bsp")(p.toString)
        )
      .distinct
      .sortBy(_.toString)

  private def definitions(path: Path): Vector[Definition] =
    val source = Files.readString(path)
    val masked = lint.Literals.code(source, lint.Literals.literals(source))
    val code = masked.indices
      .map(i => if source(i) == '\n' then '\n' else masked(i))
      .mkString
    var line = 0
    var lineStart = 0
    val tokens = identifiers
      .findAllMatchIn(code)
      .map: found =>
        while code.indexOf('\n', lineStart) >= 0 && code.indexOf(
            '\n',
            lineStart
          ) < found.start
        do
          lineStart = code.indexOf('\n', lineStart) + 1
          line += 1
        Token(
          found.matched.replace("`", ""),
          found.start,
          line,
          code
            .substring(lineStart, found.start)
            .takeWhile(_.isWhitespace)
            .length
        )
      .toVector
    def text(at: Int): String = tokens.lift(at).map(_.text).getOrElse("")
    def preceding(at: Int): Set[String] =
      var previous = at - 1
      var qualifier = 0
      var found = Set.empty[String]
      while previous >= 0 && (qualifier > 0 || text(
          previous
        ) == "]" || modifiers(text(previous)))
      do
        if text(previous) == "]" then qualifier += 1
        if text(previous) == "[" then qualifier -= 1
        if qualifier == 0 then found += text(previous)
        previous -= 1
      found
    def qualified(at: Int): (String, Int) =
      var next = at + 1
      var name = text(at)
      while text(next) == "." && next + 1 < tokens.size do
        name += "." + text(next + 1)
        next += 2
      (name, next)
    def scopeEnd(at: Int, body: Int): Int =
      if text(body) == "{" then
        var depth = 1
        var next = body + 1
        while next < tokens.size && depth > 0 do
          if text(next) == "{" then depth += 1
          if text(next) == "}" then depth -= 1
          next += 1
        tokens.lift(next - 1).map(_.offset).getOrElse(code.length)
      else
        tokens
          .drop(body + 1)
          .find(t => t.line > tokens(body).line && t.indent <= tokens(at).indent)
          .map(_.offset)
          .getOrElse(code.length)
    val scopes = scala.collection.mutable.ArrayBuffer.empty[Scope]
    val bodies = scala.collection.mutable.Set.empty[Int]
    val bindings = scala.collection.mutable.ArrayBuffer.empty[Binding]
    val found = scala.collection.mutable.ArrayBuffer.empty[Definition]
    var packageName = ""
    def owner(at: Int): String =
      (Vector(packageName).filter(_.nonEmpty) ++ scopes
        .filter(s => s.start < at && at < s.end)
        .map(_.name)
        .filter(_.nonEmpty)).mkString(".")
    def erasedMember(at: Int): Unit =
      val token = tokens(at)
      val active =
        scopes.filter(s => s.start < token.offset && token.offset < s.end)
      if preceding(at)("inline") && active.lastOption.exists(s => sections(s.name))
      then
        val enclosing = owner(token.offset)
        val name = s"$enclosing.${text(at + 1)}"
        found += Definition(
          name,
          name,
          enclosing,
          path,
          token.offset,
          Vector.empty,
          Vector.empty,
          runnable = false,
          erasedMember = true,
          line = token.line + 1,
          sectionOffset = active.last.start
        )
    var i = 0
    while i < tokens.size do
      val token = tokens(i)
      if token.text == "package" && text(i + 1) != "object" then
        val (name, next) = qualified(i + 1)
        if Set("{", ":")(text(next)) then
          scopes += Scope(
            token.offset,
            scopeEnd(i, next),
            name,
            packaged = true
          )
          bodies += tokens(next).offset
        else packageName = Vector(packageName, name).filter(_.nonEmpty).mkString(".")
        i = next
      else if token.text == "import" then
        val end = scopes
          .filter(s => s.start < token.offset && token.offset < s.end)
          .map(_.end)
          .minOption
          .getOrElse(code.length)
        var at = i + 1
        var more = true
        while more do
          val (prefix, next) = qualified(at)
          if prefix.endsWith(".{") then
            val base = prefix.stripSuffix(".{")
            val selected = scala.collection.mutable.ArrayBuffer.empty[Binding]
            var excluded = Set.empty[String]
            at = next
            while at < tokens.size && text(at) != "}" do
              if text(at) != "," then
                val name = text(at)
                val renamed = Set("as", "=>")(text(at + 1))
                val alias = if renamed then text(at + 2) else name
                if renamed && alias == "_" then excluded += name
                else selected += Binding(token.offset, end, s"$base.$name", alias)
                at += (if renamed then 3 else 1)
              else at += 1
            bindings ++= selected.map(_.copy(excluded = excluded))
            at += 1
          else
            val renamed = Set("as", "=>")(text(next))
            val alias =
              if renamed then text(next + 1) else prefix.split('.').last
            if !(renamed && alias == "_") then bindings += Binding(token.offset, end, prefix, alias)
            at = next + (if renamed then 2 else 0)
          more = text(at) == ","
          if more then at += 1
        i = at
      else if token.text == "def" then
        erasedMember(i)
        var at = i + 1
        var depth = 0
        while at < tokens.size && !(depth == 0 && text(at) == "=") &&
          !(tokens(at).line > token.line && tokens(at).indent <= token.indent)
        do
          if Set("(", "[")(text(at)) then depth += 1
          if Set(")", "]")(text(at)) then depth -= 1
          at += 1
        if text(at) == "=" then
          val body = if text(at + 1) == "{" then at + 1 else at
          scopes += Scope(token.offset, scopeEnd(i, body), "", runnable = false)
          bodies += tokens(body).offset
        i += 1
      else if token.text == "val" then
        erasedMember(i)
        i += 1
      else if Set("class", "trait", "object", "type")(token.text) then
        val name = text(i + 1)
        val enclosing = owner(token.offset)
        val active =
          scopes.filter(s => s.start < token.offset && token.offset < s.end)
        val packages =
          (Vector(packageName)
            .filter(_.nonEmpty) ++ active.filter(_.packaged).map(_.name))
            .mkString(".")
        val binary =
          (active.filterNot(_.packaged).map(_.name).filter(_.nonEmpty) :+ name)
            .map(scala.reflect.NameTransformer.encode)
            .mkString("$")
        var at = i + 2
        var depth = 0
        var done = false
        var parameters = false
        val parents = scala.collection.mutable.ArrayBuffer.empty[String]
        while at < tokens.size && !done do
          val current = text(at)
          val newStatement =
            tokens(at).line > token.line && tokens(at).indent <= token.indent &&
              !Set("extends", "with", ",")(current)
          if depth == 0 && (Set(":", "{", ";")(current) || newStatement) then done = true
          else if depth == 0 && (Set("extends", "with")(
              current
            ) || (token.text == "type" && current == "="))
          then
            val (parent, next) = qualified(at + 1)
            parents += parent
            at = next
          else
            if current == "(" && depth == 0 && parents.isEmpty && text(
                at + 1
              ) != ")"
            then parameters = true
            if Set("(", "[")(current) then depth += 1
            if Set(")", "]")(current) then depth -= 1
            at += 1
        found += Definition(
          Vector(enclosing, name).filter(_.nonEmpty).mkString("."),
          Vector(packages, binary).filter(_.nonEmpty).mkString("."),
          enclosing,
          path,
          token.offset,
          parents.toVector,
          bindings.toVector,
          token.text == "class" && !preceding(i)(
            "abstract"
          ) && !parameters && active.forall(
            _.runnable
          ),
          section = token.text == "object" && sections(
            name
          ) && active.lastOption.exists(_.name.nonEmpty),
          local = active.exists(_.name.isEmpty)
        )
        if Set(":", "{")(text(at)) then
          scopes += Scope(
            token.offset,
            scopeEnd(i, at),
            name,
            runnable = token.text == "object"
          )
          bodies += tokens(at).offset
        i = at
      else if Set("{", ":")(token.text) && !bodies(token.offset) then
        scopes += Scope(token.offset, scopeEnd(i, i), "", runnable = false)
        i += 1
      else i += 1
    found.toVector

  def declared(root: Path, module: Module): Vector[Declared] =
    val all = files(root, module).flatMap(definitions)
    val index = all.filterNot(d => d.erasedMember || d.local).groupBy(_.name)
    def resolve(name: String, definition: Definition): Vector[String] =
      val parts = name.split('.').toVector
      val bindings =
        definition.bindings.filter(b => b.start < definition.offset && definition.offset < b.end)
      def relative(value: String): Vector[String] =
        if value.startsWith("_root_.") then Vector(value.stripPrefix("_root_."))
        else
          definition.owner
            .split('.')
            .toVector
            .inits
            .map(p => (p :+ value).filter(_.nonEmpty).mkString("."))
            .toVector
      def known(name: String): Boolean = index.contains(name) || roots(name)
      val imports = bindings.filter(_.to == parts.head)
      val explicit =
        imports.flatMap(b => relative((b.from +: parts.tail).mkString(".")).find(known))
      val owners = relative(name).find(known).toVector
      val wildcard = bindings
        .filter(b => Set("*", "_")(b.to) && !b.excluded(parts.head))
        .flatMap(b => relative(b.from.dropRight(1) + name).find(known))
      val candidates =
        (if imports.nonEmpty then explicit
         else if owners.nonEmpty then owners
         else wildcard) .distinct
      if candidates.size > 1 then
        throw GateError(
          s"${root.relativize(definition.file)}: ambiguous suite ancestor $name: ${candidates.mkString(", ")}"
        )
      candidates
    def suite(
        definition: Definition,
        seen: Set[String],
        knownRoots: Set[String] = roots
    ): Boolean =
      if seen(definition.name) then false
      else
        definition.parents.exists: parent =>
          resolve(parent, definition).exists: name =>
            knownRoots(name) || index
              .getOrElse(name, Vector.empty)
              .exists(suite(_, seen + definition.name, knownRoots))
    val candidates = all
      .filter: definition =>
        definition.runnable && definition.file.toString.endsWith(
          ".test.scala"
        ) &&
          !module.skipped.exists(s => definition.file.startsWith(root.resolve(s))) &&
          module.suiteRoots.exists(s => definition.file.startsWith(root.resolve(s))) && suite(
            definition,
            Set.empty
          )
    def ancestors(
        definition: Definition,
        seen: Set[String]
    ): Vector[Definition] =
      if seen(definition.name) then Vector.empty
      else
        definition +: definition.parents
          .flatMap(resolve(_, definition))
          .flatMap(index.getOrElse(_, Vector.empty))
          .flatMap(ancestors(_, seen + definition.name))
    val admitted = candidates
      .filter(suite(_, Set.empty, Set("framework.dsl.ModelSuite")))
      .flatMap(ancestors(_, Set.empty))
      .map(d => d.file.toString -> d.offset)
      .toSet
    for
      member <- all if member.erasedMember
      section <- index.getOrElse(member.owner, Vector.empty)
      if section.section && section.file.equals(
        member.file
      ) && section.offset == member.sectionOffset
      declaring <- index.getOrElse(section.owner, Vector.empty)
      if admitted(declaring.file.toString -> declaring.offset)
    do
      throw GateError(
        s"${root.relativize(member.file)}:${member.line}: inline section member ${member.name} is not an admitted test; move shared helpers outside sections"
      )
    candidates
      .map(d => Declared(module, root.relativize(d.file).toString, d.binaryName))
      .distinct
