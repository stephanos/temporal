package framework

import scala.collection.mutable
import java.nio.file.{Files, Path}
import java.util.jar.JarFile
import scala.jdk.CollectionConverters.*
import framework.realize.Realization

// A checked-in IR file, `model/ir/<name>.json`, and the declarations that are its roots, named by
// value beside the Models it holds:
//
// {{{
// val orders =
//   irFile("orders")(OrdersSystem.queries.delayed, OrdersRealization.delayed)
// }}}
//
// The IR generator (model/irgen) reads every such val and writes each file, in one run, from its
// roots and everything they reach, as it lifts the roots named on its command line. A root is a
// machine or composition object, a Query, a list of Queries, a progress claim, a realization, a
// `capabilities` section, which declares its object's capabilities, or a `queries`
// section, which roots every Query it declares; the generator refuses anything else. A declaration
// may be a root of several files and is lifted into each; one no file names stays out of model/ir,
// so a design can be kept out of the checked files. The file's `source` lists its roots' fully
// qualified names.
final class IrFile private[framework] (val name: String, val roots: Seq[IrRoot]):
  // Every reachable declaration, retaining the typed instances and their finite witnesses.
  def subjects(): Vector[Model] = traverse()._2

  // Every Model constructing the roots reaches, each once, in first-reached order: the declarations
  // `subjects` gives, and the machines and compositions derivations make in place, such as
  // `m.restrict(...)` passed as a member (the step tables, model/framework/StepTables.scala).
  private[framework] def reached(): Vector[Model] = traverse()._1

  // The Models the roots reach, in first-reached order, and the declarations among them.
  private def traverse(): (Vector[Model], Vector[Model]) =
    val seen = java.util.IdentityHashMap[Model, java.lang.Boolean]()
    val reached = mutable.ArrayBuffer.empty[Model]
    val declarations = mutable.LinkedHashMap.empty[String, Model]
    def model(m: Model): Unit =
      if !seen.containsKey(m) then
        seen.put(m, java.lang.Boolean.TRUE): Unit
        reached += m: Unit
        val declared = m match
          case _: Built[?, ?, ?, ?] => false // format: off
          case c: Composition[?]   => !c.getClass.getName.contains("$anon")
          case _                   => true // format: on
        if declared then
          val id = IrFile.identity(m)
          require(
            declarations.get(id).forall(_ eq m),
            s"$name repeats declaration identity $id with distinct instances"
          )
          declarations.update(id, m)
        m match
          case machine: Machine[?, ?, ?] =>
            machine.reaches.foreach(model)
          case c: Composition[?] =>
            c.members.foreach(model)
            c.shape match
              case Composition.Shape.Of(source)      => model(source)
              case Composition.Shape.With(source, _) => model(source)
              case Composition.Shape.Members(_)      => ()
          case _ => ()
    def query(q: Query): Unit = Seq(q.scenario.machine, q.property.machine).foreach(model)
    def root(r: Any): Unit = r match
      case m: Machine[?, ?, ?]      => model(m)
      case c: Composition[?]        => model(c)
      case q: Query                 => query(q)
      case qs: Seq[?]               => qs.foreach { case q: Query => query(q); case _ => () }
      case p: Progress[?]           => model(p.machine)
      case r: Realization           => model(r.machine)
      case c: Capabilities[?, ?, ?] => model(c.model)
      case section: AnyRef          => IrFile.queriesOf(section).foreach(root)
    roots.foreach(root)
    val subjects = declarations.values.toVector
    val collisions = subjects.groupBy(_.name).filter(_._2.sizeIs > 1)
    require(
      collisions.isEmpty,
      s"$name has colliding model names: " + collisions.toSeq
        .sortBy(_._1)
        .map((n, ms) => s"$n: ${ms.map(IrFile.identity).mkString(", ")}")
        .mkString("; ")
    )
    (reached.toVector, subjects)

  // Constructs each reachable machine's rules, refusing overlaps as the model gate does.
  def construct(): Set[String] =
    val declarations = subjects()
    declarations.foreach:
      case machine: Machine[?, ?, ?] => machine.bindings: Unit
      case _                         => ()
    declarations.map(_.name).toSet

object IrFile:
  private[framework] val made = mutable.ArrayBuffer.empty[IrFile]

  // Every IR file declared so far: each `irFile` val of an object that has initialized.
  def declared: Seq[IrFile] = made.synchronized(made.toSeq)

  def identity(model: Model): String = model.getClass.getName.stripSuffix("$").replace('$', '.')

  // The compiled declaration universe initializes owners, independently of any generated files.
  def discover(packagePrefix: String): Vector[IrFile] =
    val loader = getClass.getClassLoader
    val prefix = packagePrefix.replace('.', '/') + "/"
    val classes = mutable.TreeSet.empty[String]
    def directory(path: Path): Unit =
      if Files.isDirectory(path) then
        val stream = Files.walk(path)
        try
          stream.iterator.asScala
            .filter(p => p.toString.endsWith("$.class"))
            .foreach: p =>
              classes.add(
                packagePrefix + "." + path
                  .relativize(p)
                  .toString
                  .stripSuffix(".class")
                  .replace(java.io.File.separatorChar, '.')
              ): Unit
        finally stream.close()
    loader
      .getResources(prefix.stripSuffix("/"))
      .asScala
      .foreach: url =>
        if url.getProtocol.equals("file") then directory(Path.of(url.toURI))
    System
      .getProperty("java.class.path")
      .split(java.io.File.pathSeparator)
      .foreach: entry =>
        val path = Path.of(entry)
        if Files.isDirectory(path) then directory(path.resolve(prefix))
        else if entry.endsWith(".jar") && Files.isRegularFile(path) then
          val jar = JarFile(path.toFile)
          try
            jar.entries.asScala
              .filter(e => e.getName.startsWith(prefix) && e.getName.endsWith("$.class"))
              .foreach(e => classes.add(e.getName.stripSuffix(".class").replace('/', '.')): Unit)
          finally jar.close()
    val files = classes.toVector.flatMap: name =>
      val cls = java.lang.Class.forName(name, false, loader)
      val getters = cls.getDeclaredMethods.toVector
        .filter(m =>
          m.getParameterCount.equals(0) && java.lang.reflect.Modifier.isPublic(m.getModifiers) &&
            classOf[IrFile].isAssignableFrom(m.getReturnType)
        )
        .sortBy(_.getName)
      if getters.isEmpty then Vector.empty
      else
        val owner = cls.getField("MODULE$").get(null) // scalafix:ok DisableSyntax.null
        getters.map(m =>
          m.invoke(owner) match
            case file: IrFile => file
            case other        => throw IllegalStateException(s"$name.${m.getName} returned $other")
        )
    val duplicates = files.groupBy(_.name).filter(_._2.sizeIs > 1)
    require(
      duplicates.isEmpty,
      s"duplicate IR export names: ${duplicates.keys.toSeq.sorted.mkString(", ")}"
    )
    require(files.nonEmpty, s"no compiled IR exports found in $packagePrefix")
    files.sortBy(_.name)

  // The Queries and lists of Queries a `queries` section declares, read as the members it declares
  // with no parameter: the section is no class of the framework, so its members are found as the
  // object's own, as `Refinement.declaredBy` finds a machine's refinement.
  private def queriesOf(section: AnyRef): Seq[Any] =
    section.getClass.getDeclaredMethods.toSeq
      .filter(m => m.getParameterCount == 0 && java.lang.reflect.Modifier.isPublic(m.getModifiers))
      .filter(m =>
        classOf[Query].isAssignableFrom(m.getReturnType) ||
          classOf[scala.collection.Seq[?]].isAssignableFrom(m.getReturnType)
      )
      .sortBy(_.getName)
      .map(_.invoke(section))

// What an IR file names as a root: a machine, a composition, a Query, a list of Queries, a progress
// claim, a realization, a `capabilities` section, or a `queries` section. A section is no class of
// the framework, so the
// type admits any object, and the IR generator refuses one that is none of these.
type IrRoot = AnyRef

// Declares the IR file `model/ir/<name>.json` and its roots.
def irFile(name: String)(roots: IrRoot*): IrFile =
  val file = IrFile(name, roots)
  IrFile.made.synchronized(IrFile.made += file): Unit
  file
