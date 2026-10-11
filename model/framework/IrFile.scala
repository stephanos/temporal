package framework

import scala.collection.mutable
import java.nio.file.{Files, Path}
import java.util.jar.JarFile
import scala.jdk.CollectionConverters.*
import framework.realize.Realization

// A checked-in IR file, `model/ir/<name>.json`: one per feature, collected from the feature's
// package (fn-160), as the IR generator (model/irgen) collects it: its machines, compositions,
// realizations and their `queries` and `capabilities` sections, and everything they reach. Nothing
// lists them, so no declaration is left out by omission. The step tables and the model tests read
// the same collection (`discover`).
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
      case section: AnyRef          =>
        scala.util
          .Try(section.getClass.getMethod("machine").invoke(section))
          .toOption
          .collect { case m: Model => m }
          .fold(IrFile.queriesOf(section).foreach(root))(model)
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
    // fn-160 R2, R10: one file per feature, collected from its package (model/irgen/Lifting.scala,
    // `irFiles`): its machine, composition and realization objects and their `queries` and
    // `capabilities` sections. A facet's `queries`, which no static object holds, reach only its
    // machine, which the machine object roots already.
    def module(name: String): Option[AnyRef] =
      scala.util
        .Try(
          java.lang.Class.forName(name, false, loader).getField("MODULE$").get(null)
        ) // scalafix:ok DisableSyntax.null
        .toOption
        .collect { case o: AnyRef => o }
    def machineOf(o: AnyRef): Option[Model] =
      scala.util
        .Try(o.getClass.getMethod("machine").invoke(o))
        .toOption
        .collect { case m: Model => m }
    val roots = classes.toVector.flatMap: name =>
      val section = name.split('$').filter(_.nonEmpty)
      module(name).toVector.flatMap {
        case m: Model                                        => Vector(name -> (m, false))
        case c: Capabilities[?, ?, ?] if section.sizeIs == 2 => Vector(name -> (c, false))
        case o if section.lastOption.contains("queries") && section.sizeIs == 2 =>
          Vector(name -> (o, true))
        case o if machineOf(o).nonEmpty => Vector(name -> (o, true))
        case _                          => Vector.empty
      }
    val files = roots
      .groupBy((name, _) => IrFile.featureOf(name.take(name.lastIndexOf('.'))))
      .toVector
      .collect {
        case (Some(file), rs) if file.contains('-') || rs.exists(_._2._2) =>
          IrFile(file, rs.sortBy(_._1).map(_._2._1))
      }
    require(files.nonEmpty, s"no compiled features found in $packagePrefix")
    files.sortBy(_.name)

  // The IR file of a package: a form under a kind, `<kind>-<form>`; a kind's own package, `<kind>`;
  // a package under `foundations` or `actors`, its name; none for any other package. A form's file
  // holds whatever it declares; another's only where it declares Queries or realizations.
  def featureOf(pkg: String): Option[String] =
    val parts = pkg.split('.').toList
    val at = parts.indexWhere(Set("features", "foundations", "actors"))
    if at < 0 then None
    else
      parts.drop(at) match
        case "features" :: kind :: form :: _ if Set("workflow", "standalone")(form) =>
          Some(s"$kind-$form")
        case _ :: feature :: _ => Some(feature)
        case _                 => None

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

// What an IR file collects as a root: a machine, a composition, a Query, a list of Queries, a
// progress claim, a realization, a `capabilities` section, or a `queries` section. A section is no
// class of the framework, so the type admits any object.
type IrRoot = AnyRef
