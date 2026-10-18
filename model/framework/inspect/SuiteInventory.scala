package framework.inspect

import java.nio.file.{Files, Path}
import java.security.MessageDigest
import java.util.zip.{ZipFile, ZipInputStream}
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.Quotes
import scala.tasty.inspector.{Inspector, Tasty, TastyInspector}
import scala.util.control.NonFatal

private[framework] object SuiteInventory:
  // Directories hash sorted entries: big-endian name-byte length (Int), content length (Long), UTF-8 name, content.
  case class Artifact(path: Path, sha256: String)
  case class Source(path: Path, sha256: String)
  case class Toolchain(scalaVersion: String, javaVersion: String)
  case class Request(
      module: String,
      candidates: Vector[Artifact],
      hierarchy: Vector[Artifact],
      trusted: Vector[Artifact],
      sources: Vector[Source],
      skippedRoots: Vector[Path],
      compilerFlags: Vector[String],
      toolchain: Toolchain
  )
  case class Diagnostic(message: String)
  case class Location(
      artifact: Artifact,
      recordedPath: String,
      source: Option[Source],
      symbol: String,
      start: Int,
      end: Int,
      startLine: Int,
      endLine: Int
  )
  trait Declaration:
    val quotes: Quotes
    val tree: quotes.reflect.ClassDef
    val location: Location
    val binaryName: String
    val parents: Vector[String]

  case class ClassFacts(
      binaryName: String,
      flags: Int,
      publicNoArg: Boolean,
      parents: Vector[String]
  )
  case class Suite(facts: ClassFacts, declaration: Declaration, ancestors: Vector[Declaration])
  case class Exclusion(facts: ClassFacts, reason: String)
  case class Result(
      input: Request,
      fingerprint: String,
      suites: Vector[Suite],
      exclusions: Vector[Exclusion]
  )

  private case class Snapshot(
      identity: Artifact,
      entries: Map[String, Array[Byte]],
      trusted: Boolean
  )
  private case class Owned(snapshot: Snapshot, facts: ClassFacts, tastyId: Option[String])

  private def fail(context: String, reason: String): Nothing =
    throw IllegalArgumentException(s"$context: $reason")

  private def digest(bytes: Array[Byte]): String =
    MessageDigest.getInstance("SHA-256").digest(bytes).map(b => f"${b & 0xff}%02x").mkString

  private def readArtifact(path: Path): (Map[String, Array[Byte]], String) =
    val contents = if Files.isDirectory(path) then
      val stream = Files.walk(path)
      try
        stream.iterator.asScala
          .map: file =>
            require(!Files.isSymbolicLink(file), s"$file: symbolic artifact entry")
            file
          .filter(Files.isRegularFile(_))
          .map: file =>
            require(file.toRealPath().startsWith(path), s"$file: artifact entry escapes $path")
            path.relativize(file).toString.replace('\\', '/') -> Files.readAllBytes(file)
          .toMap
      finally stream.close()
    else
      val bytes = Files.readAllBytes(path)
      val zip = new ZipInputStream(new java.io.ByteArrayInputStream(bytes))
      val decoded = mutable.LinkedHashMap.empty[String, Array[Byte]]
      try
        var next = Option(zip.getNextEntry())
        while next.nonEmpty do
          val entry = next.get
          if !entry.isDirectory then
            require(
              !decoded.contains(entry.getName),
              s"$path: duplicate archive entry ${entry.getName}"
            )
            decoded(entry.getName) = zip.readAllBytes()
          next = Option(zip.getNextEntry())
      finally zip.close()
      val directory = new ZipFile(path.toFile)
      try
        require(
          directory
            .entries()
            .asScala
            .filterNot(_.isDirectory)
            .map(_.getName)
            .toVector
            .sorted == decoded.keys.toVector.sorted,
          s"$path: inconsistent archive directory"
        )
      finally directory.close()
      return decoded.toMap -> digest(bytes)
    val hash = MessageDigest.getInstance("SHA-256")
    contents.toVector
      .sortBy(_._1)
      .foreach: (name, bytes) =>
        val encoded = name.getBytes(java.nio.charset.StandardCharsets.UTF_8)
        hash.update(
          java.nio.ByteBuffer
            .allocate(12)
            .putInt(encoded.length)
            .putLong(bytes.length.toLong)
            .array()
        )
        hash.update(encoded)
        hash.update(bytes)
    contents -> hash.digest().map(b => f"${b & 0xff}%02x").mkString

  private def api(owner: String): java.lang.Class[?] =
    java.lang.Class.forName(s"java.lang.classfile.$owner")

  // Scala 3.9 cannot complete the JDK 27 class-file API's sealed implementation graph.
  private def call(owner: String, receiver: Option[AnyRef], name: String, args: AnyRef*): AnyRef =
    val method = api(owner).getMethods
      .find(m =>
        m.getName == name && m.getParameterCount == args.size &&
          m.getParameterTypes.zip(args).forall((tpe, arg) => tpe.isInstance(arg))
      )
      .getOrElse(throw NoSuchMethodException(s"$owner.$name"))
    try method.invoke(receiver.orNull, args*) // scalafix:ok DisableSyntax.null
    catch case wrapped: java.lang.reflect.InvocationTargetException => throw wrapped.getCause

  private def list(value: AnyRef): Vector[AnyRef] =
    value
      .asInstanceOf[java.util.List[AnyRef]]
      .asScala
      .toVector // scalafix:ok DisableSyntax.asInstanceOf

  private def className(value: AnyRef): String =
    call("constantpool.ClassEntry", Some(value), "asInternalName").toString.replace('/', '.')

  private def utf8(value: AnyRef): String =
    call("constantpool.Utf8Entry", Some(value), "stringValue").toString

  private def parse(snapshot: Snapshot, entry: String, bytes: Array[Byte]): Owned =
    try
      val parser = call("ClassFile", None, "of")
      val model = call("ClassFile", Some(parser), "parse", bytes)
      val major = call("ClassModel", Some(model), "majorVersion").toString.toInt
      require(
        major <= call("ClassFile", None, "latestMajorVersion").toString.toInt,
        s"incompatible classfile version $major"
      )
      val name = className(call("ClassModel", Some(model), "thisClass"))
      require(
        entry == name.replace('.', '/') + ".class",
        s"binary name disagrees with entry: $name"
      )
      def flags(owner: String, value: AnyRef): Int =
        call("AccessFlags", Some(call(owner, Some(value), "flags")), "flagsMask").toString.toInt
      val superclass = call("ClassModel", Some(model), "superclass")
        .asInstanceOf[java.util.Optional[AnyRef]] // scalafix:ok DisableSyntax.asInstanceOf
      val parents = superclass.stream().iterator().asScala.map(className).toVector ++
        list(call("ClassModel", Some(model), "interfaces")).map(className)
      val constructor = list(call("ClassModel", Some(model), "methods")).exists: method =>
        utf8(call("MethodModel", Some(method), "methodName")) == "<init>" &&
          utf8(call("MethodModel", Some(method), "methodType")) == "()V" &&
          (flags("MethodModel", method) & 1) != 0
      val id = list(call("AttributedElement", Some(model), "attributes"))
        .find(attribute => utf8(call("Attribute", Some(attribute), "attributeName")) == "TASTY")
        .map(attribute =>
          val contents = call("attribute.UnknownAttribute", Some(attribute), "contents")
            .asInstanceOf[Array[Byte]] // scalafix:ok DisableSyntax.asInstanceOf
          require(contents.length == 16, "invalid TASTY identifier")
          java.util.HexFormat.of().formatHex(contents)
        )
      Owned(snapshot, ClassFacts(name, flags("ClassModel", model), constructor, parents), id)
    catch
      case NonFatal(error) =>
        fail(s"${snapshot.identity.path}!$entry", s"malformed classfile: ${error.getMessage}")

  private def fingerprint(input: Request): String =
    val fields = Vector(input.module, input.toolchain.scalaVersion, input.toolchain.javaVersion) ++
      Vector(input.candidates, input.hierarchy, input.trusted).flatMap(artifacts =>
        Vector(artifacts.size.toString) ++ artifacts.flatMap(a => Vector(a.path.toString, a.sha256))
      ) ++ Vector(input.sources.size.toString) ++ input.sources.flatMap(s =>
        Vector(s.path.toString, s.sha256)
      ) ++
      Vector(input.skippedRoots.size.toString) ++ input.skippedRoots.map(
        _.toString
      ) ++ input.compilerFlags
    digest(
      fields
        .map(value => s"${value.length}:$value")
        .mkString
        .getBytes(java.nio.charset.StandardCharsets.UTF_8)
    )

  def inspect(request: Request): Either[Diagnostic, Result] =
    try Right(checked(request))
    catch
      case NonFatal(error) =>
        Left(Diagnostic(Option(error.getMessage).getOrElse(error.getClass.getName)))

  private def checked(request: Request): Result =
    require(request.module.nonEmpty, "module identity is empty")
    require(request.candidates.nonEmpty, s"${request.module}: missing candidate outputs")
    require(request.compilerFlags.nonEmpty, s"${request.module}: missing compiler flags")
    require(
      request.toolchain.scalaVersion == "3.9.0",
      s"${request.module}: incompatible Scala toolchain"
    )
    require(
      request.toolchain.javaVersion == System.getProperty("java.runtime.version"),
      s"${request.module}: incompatible Java toolchain"
    )

    val snapshots = mutable.LinkedHashMap.empty[Path, Snapshot]
    def snapshot(artifact: Artifact, trusted: Boolean): Artifact =
      val path = artifact.path.toRealPath()
      snapshots.get(path) match
        case Some(previous) =>
          require(
            previous.identity.sha256 == artifact.sha256 && previous.trusted == trusted,
            s"$path: conflicting artifact identity or trust"
          )
          previous.identity
        case None =>
          val (contents, hash) = readArtifact(path)
          require(hash == artifact.sha256, s"$path: changed artifact bytes")
          val identity = artifact.copy(path = path)
          snapshots(path) = Snapshot(identity, contents, trusted)
          identity
    val candidates = request.candidates.map(snapshot(_, false)).distinct
    val hierarchy = request.hierarchy.map(snapshot(_, false)).distinct
    val trusted = request.trusted.map(snapshot(_, true)).distinct
    val sources = request.sources.map: source =>
      val path = source.path.toRealPath()
      require(digest(Files.readAllBytes(path)) == source.sha256, s"$path: changed source bytes")
      source.copy(path = path)
    require(
      sources.map(_.path).distinct.size == sources.size,
      s"${request.module}: conflicting source ownership"
    )
    val input = request.copy(
      candidates = candidates,
      hierarchy = hierarchy,
      trusted = trusted,
      sources = sources,
      skippedRoots = request.skippedRoots.map(_.toRealPath()).distinct
    )
    val candidatePaths = candidates.map(_.path).toSet
    val owners = snapshots.valuesIterator
      .flatMap: artifact =>
        artifact.entries.toVector
          .sortBy(_._1)
          .collect:
            case (name, bytes) if name.endsWith(".class") && !name.startsWith("META-INF/") =>
              val owned = parse(artifact, name, bytes)
              owned.facts.binaryName -> owned
      .toVector
      .groupMap(_._1)(_._2)

    def owner(name: String): Owned = owners.get(name) match
      case Some(Vector(single)) => single
      case Some(conflicts)      =>
        fail(
          name,
          "conflicting binary owners: " + conflicts.map(_.snapshot.identity.path).mkString(", ")
        )
      case None => fail(name, "missing hierarchy classfile in explicit artifacts")

    def runtime(name: String): Boolean =
      Option(ClassLoader.getPlatformClassLoader.getResource(name.replace('.', '/') + ".class"))
        .exists(_.getProtocol == "jrt")

    val suiteMemo = mutable.Map.empty[String, Boolean]
    def isSuite(name: String, active: Set[String]): Boolean =
      require(!active(name), s"$name: cyclic classfile hierarchy")
      suiteMemo.getOrElseUpdate(
        name,
        if !owners.contains(name) && runtime(name) then false
        else
          val entry = owner(name)
          if name == "munit.FunSuite" then
            require(entry.snapshot.trusted, s"$name: suite root is not in a trusted artifact")
            true
          else entry.facts.parents.exists(parent => isSuite(parent, active + name))
      )

    val candidateNames = owners.toVector
      .collect:
        case (name, entries)
            if entries.exists(entry => candidatePaths(entry.snapshot.identity.path)) =>
          name
      .sorted
    require(candidateNames.nonEmpty, s"${request.module}: missing candidate classfiles")
    val suiteClasses = candidateNames.filter(name => isSuite(name, Set.empty)).map(owner)
    val required = mutable.LinkedHashMap.empty[String, Owned]
    def closure(name: String, active: Set[String]): Unit =
      require(!active(name), s"$name: cyclic author hierarchy")
      if !owners.contains(name) && runtime(name) then ()
      else
        val entry = owner(name)
        if !entry.snapshot.trusted && !required.contains(name) then
          required(name) = entry
          entry.facts.parents.foreach(parent => closure(parent, active + name))
    suiteClasses.foreach(entry => closure(entry.facts.binaryName, Set.empty))

    val declarations = mutable.Map.empty[String, Declaration]
    val typedEdges = mutable.Map.empty[String, Set[String]]
    val classConstraints = mutable.Map.empty[String, Vector[String]]
    val classpath = snapshots.keysIterator.map(_.toString).toList
    val authorArtifacts = (hierarchy ++ candidates).distinct.map(a => snapshots(a.path))
    authorArtifacts.foreach: artifact =>
      val needed = required.valuesIterator
        .filter(_.snapshot.identity.path.equals(artifact.identity.path))
        .toVector
      if needed.nonEmpty then
        val tastys = artifact.entries.toVector.filter(_._1.endsWith(".tasty")).sortBy(_._1)
        needed.foreach: entry =>
          val name = entry.facts.binaryName
          val containers = owners.valuesIterator.flatten
            .filter(other =>
              other.snapshot.identity.path
                .equals(artifact.identity.path) && other.tastyId.nonEmpty &&
                (name == other.facts.binaryName || name.startsWith(other.facts.binaryName + "$"))
            )
            .toVector
          containers.foreach: container =>
            require(
              artifact.entries.contains(container.facts.binaryName.replace('.', '/') + ".tasty"),
              s"${artifact.identity.path}!$name: missing typed declaration container ${container.facts.binaryName}"
            )
        tastys.foreach: (entry, bytes) =>
          val header = try new dotty.tools.tasty.TastyHeaderUnpickler(bytes).readFullHeader()
          catch
            case NonFatal(error) =>
              fail(s"${artifact.identity.path}!$entry", s"incompatible TASTy: ${error.getMessage}")
          val id = header.uuid.toString.replace("-", "")
          val rootName = entry.stripSuffix(".tasty").replace('/', '.')
          val container = owners
            .getOrElse(rootName, Vector.empty)
            .find(_.snapshot.identity.path.equals(artifact.identity.path))
            .getOrElse(fail(s"${artifact.identity.path}!$entry", "missing classfile container"))
          require(
            container.tastyId.contains(id),
            s"${artifact.identity.path}!$rootName: mixed classfile/TASTy generation"
          )
        val inspector = new Inspector:
          def inspect(using q: Quotes)(tastys: List[Tasty[q.type]]): Unit =
            val index = Index(tastys, Map.empty)
            import index.quotes.reflect.*
            def binary(symbol: Symbol): String =
              if symbol.flags.is(Flags.Package) then symbol.fullName
              else
                val parent = symbol.owner
                val name = scala.reflect.NameTransformer.encode(symbol.name)
                if parent.flags.is(Flags.Package) then
                  val prefix = binary(parent)
                  (if prefix == "<empty>" then "" else prefix + ".") + name
                else
                  val prefix = binary(parent)
                  (if parent.flags.is(Flags.Module) then prefix.stripSuffix("$")
                   else prefix) + "$" + name
            object visit extends TreeTraverser:
              override def traverseTree(tree: Tree)(parent: Symbol): Unit =
                tree match
                  case cls: ClassDef =>
                    val name = binary(cls.symbol)
                    if required.contains(name) then
                      val physical = owner(name)
                      require(
                        physical.snapshot.identity.path.equals(artifact.identity.path),
                        s"${artifact.identity.path}!$name: typed declaration belongs to another artifact"
                      )
                      require(
                        !declarations.contains(name),
                        s"$name: conflicting typed declarations"
                      )
                      val pos = cls.pos
                      val recorded = Path.of(pos.sourceFile.path).normalize().toString
                      val source = if candidatePaths(artifact.identity.path) then
                        val path = Path.of(recorded).toRealPath()
                        Some(
                          sources
                            .find(_.path.equals(path))
                            .getOrElse(fail(name, s"source outside explicit manifest: $path"))
                        )
                      else None
                      require(
                        pos.start >= 0 && pos.end > pos.start && pos.startLine >= 0,
                        s"${artifact.identity.path}!$name: unavailable retained source position"
                      )
                      val sourceLocation = Location(
                        artifact.identity,
                        recorded,
                        source,
                        cls.symbol.fullName,
                        pos.start,
                        pos.end,
                        pos.startLine,
                        pos.endLine
                      )
                      val parentSymbols = cls.parents.map:
                        case term: Term    => term.tpe.dealias.typeSymbol
                        case tpe: TypeTree => tpe.tpe.dealias.typeSymbol
                        case other         => fail(name, s"unavailable typed parent: ${other.show}")
                      val direct = parentSymbols.map(binary)
                      if cls.symbol.flags.is(Flags.Trait) then
                        typedEdges(name) = parentSymbols
                          .filter(_.flags.is(Flags.Trait))
                          .map(binary)
                          .toSet + "java.lang.Object"
                        classConstraints(name) =
                          parentSymbols.filterNot(_.flags.is(Flags.Trait)).map(binary).toVector
                      else
                        typedEdges(name) = direct.toSet ++
                          (if cls.symbol.flags.is(Flags.Module) then Set("java.io.Serializable")
                           else Set.empty[String])
                      val declaration = new Declaration:
                        val quotes: index.quotes.type = index.quotes
                        val tree: index.quotes.reflect.ClassDef = cls
                        val location: Location = sourceLocation
                        val binaryName: String = name
                        val parents: Vector[String] = direct.toVector
                      declarations(name) = declaration
                  case _ => ()
                super.traverseTree(tree)(parent)
            index.trees.foreach(tree => visit.traverseTree(tree)(Symbol.spliceOwner))
        val inspected = if Files.isDirectory(artifact.identity.path) then
          TastyInspector.inspectAllTastyFiles(
            tastys.map((entry, _) => artifact.identity.path.resolve(entry).toString).toList,
            Nil,
            classpath
          )(inspector)
        else
          TastyInspector.inspectAllTastyFiles(
            Nil,
            List(artifact.identity.path.toString),
            classpath
          )(inspector)
        require(
          inspected,
          s"${artifact.identity.path}!${needed.map(_.facts.binaryName).mkString(",")}: incompatible or partial typed metadata"
        )
        needed.foreach: entry =>
          val name = entry.facts.binaryName
          val declaration = declarations.getOrElse(
            name,
            fail(s"${artifact.identity.path}!$name", "missing typed declaration")
          )
          require(
            typedEdges(name) == entry.facts.parents.toSet,
            s"${artifact.identity.path}!$name: typed/classfile parent edges disagree: ${declaration.parents} / ${entry.facts.parents}"
          )

    def ancestors(name: String): Vector[Declaration] =
      val seen = mutable.LinkedHashSet.empty[String]
      def walk(current: String): Unit =
        if required.contains(current) && seen.add(current) then
          required(current).facts.parents.foreach(walk)
      walk(name)
      seen.toVector.map(declarations)
    suiteClasses.foreach: entry =>
      val names = ancestors(entry.facts.binaryName).map(_.binaryName).toSet
      val boundary = names.flatMap(name => required(name).facts.parents).toSet + "java.lang.Object"
      names.foreach: name =>
        classConstraints
          .getOrElse(name, Vector.empty)
          .foreach: parent =>
            require(
              names(parent) || boundary(parent),
              s"$name: typed trait superclass constraint $parent is absent from independent classfile closure"
            )
    val exclusions = mutable.ArrayBuffer.empty[Exclusion]
    val suites = suiteClasses.flatMap: entry =>
      val facts = entry.facts
      val reason = if (facts.flags & 1) == 0 then Some("JVM class is not public")
      else if (facts.flags & (0x400 | 0x200)) != 0 then
        Some("JVM class is abstract or an interface")
      else if !facts.publicNoArg then Some("no public JVM ()V constructor")
      else if declarations(facts.binaryName).location.source
          .exists(source => input.skippedRoots.exists(source.path.startsWith(_)))
      then Some("skipped source")
      else None
      reason match
        case Some(message) =>
          exclusions += Exclusion(facts, message)
          None
        case None => Some(Suite(facts, declarations(facts.binaryName), ancestors(facts.binaryName)))
    snapshots.valuesIterator.foreach: artifact =>
      require(
        readArtifact(artifact.identity.path)._2 == artifact.identity.sha256,
        s"${artifact.identity.path}: artifact changed during inspection"
      )
    sources.foreach(source =>
      require(
        digest(Files.readAllBytes(source.path)) == source.sha256,
        s"${source.path}: source changed during inspection"
      )
    )
    val nonSuites = candidateNames
      .filterNot(name => suiteClasses.exists(_.facts.binaryName == name))
      .map(name => Exclusion(owner(name).facts, "not a JVM FunSuite descendant"))
    Result(
      input,
      fingerprint(input),
      suites,
      (exclusions.toVector ++ nonSuites).sortBy(_.facts.binaryName)
    )
