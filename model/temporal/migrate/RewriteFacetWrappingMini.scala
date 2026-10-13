package temporal.migrate

import java.nio.file.{Files, LinkOption, Path, StandardCopyOption, StandardOpenOption}
import scala.meta.*
import scala.util.control.NonFatal
import java.net.URLClassLoader
import java.util.zip.ZipFile
import scala.jdk.CollectionConverters.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteFacetWrappingMini:
  def verifyInventory(names: List[String]): Unit =
    val expected = Set(
      "temporal.actors.worker.Polling$",
      "temporal.foundations.taskqueue.product.TaskQueueProduct$",
      "temporal.foundations.taskqueue.protocol.TaskQueueProtocol$",
      "temporal.features.activity.mini.product.ActivityProduct$",
      "temporal.features.activity.mini.protocol.ActivityProtocol$",
      "temporal.features.activity.mini.protocol.AnswerDispatch$"
    )
    require(
      names.toSet == expected && names.size == expected.size,
      s"selected declared-machine inventory differs: ${names.sorted.mkString(", ")}"
    )

  private def additionalImports(base: List[Import], proposed: List[Import]): List[String] =
    val existing = base
      .flatMap(_.importers)
      .groupBy(_.ref.syntax)
      .view
      .mapValues(_.flatMap(_.importees.map(_.syntax)).toSet)
      .toMap
    proposed
      .filterNot(i => base.exists(_.syntax == i.syntax))
      .map { i =>
        require(i.importers.size == 1, "the seed inventory supports one importer per statement")
        val importer = i.importers.head
        require(
          importer.importees.forall(_.isInstanceOf[Importee.Name]),
          "new seed imports must name explicit selected members"
        )
        val missing = importer.importees
          .map(_.syntax)
          .filterNot(existing.getOrElse(importer.ref.syntax, Set.empty).contains)
        val selected = if missing.size == 1 then missing.head else missing.mkString("{", ", ", "}")
        if missing.isEmpty then "" else s"import ${importer.ref.syntax}.$selected"
      }
      .filter(_.nonEmpty)

  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val protocol = Path.of(args(0)).toRealPath()
    val feature = protocol.getParent
    val campaign = feature.getParent.getParent.getParent.getParent
    val repository = Path.of("").toRealPath()
    require(
      campaign.startsWith(repository.resolve(".flow/tmp").toRealPath()),
      "the campaign must be an isolated copied mini"
    )
    val archive = Files.createDirectory(campaign.resolve("facet-wrapping-evidence"))
    val beforeJar = campaign.resolve("before.jar")
    val beforeClasspath = Files.readString(campaign.resolve("before.classpath")).trim
    val loader = URLClassLoader(
      beforeClasspath
        .split(java.io.File.pathSeparator)
        .map(p => Path.of(p).toUri.toURL),
      ClassLoader.getPlatformClassLoader
    )
    val zip = ZipFile(beforeJar.toFile)
    try
      def parents(c: Class[?]): List[String] =
        if c == null then Nil else c.getName :: parents(c.getSuperclass)
      val inventory = zip
        .entries()
        .asScala
        .filter(e => e.getName.startsWith("temporal/") && e.getName.endsWith(".class"))
        .map { entry =>
          val name = entry.getName.stripSuffix(".class").replace('/', '.')
          name -> parents(Class.forName(name, false, loader))
        }
        .filter { (_, chain) =>
          chain.contains("framework.Machine") &&
          !chain.exists(Set("framework.Derived", "framework.Built"))
        }
        .map(_._1)
        .toList
        .sorted
      verifyInventory(inventory)
      Files.writeString(
        archive.resolve("selected-declared-machines.txt"),
        inventory.mkString("", "\n", "\n")
      ): Unit
      val expectedFacets = Map(
        "temporal.actors.worker.Polling$" -> Set("Serving"),
        "temporal.foundations.taskqueue.product.TaskQueueProduct$" -> Set("Queueing"),
        "temporal.foundations.taskqueue.protocol.TaskQueueProtocol$" -> Set("Queueing"),
        "temporal.features.activity.mini.product.ActivityProduct$" -> Set(
          "Dispatching",
          "Completion"
        ),
        "temporal.features.activity.mini.protocol.ActivityProtocol$" -> Set(
          "Dispatching",
          "Completion"
        ),
        "temporal.features.activity.mini.protocol.AnswerDispatch$" -> Set("Answering")
      )
      val selected = inventory.map { name =>
        val interfaces = Class
          .forName(name, false, loader)
          .getInterfaces
          .toList
          .filter(_.getInterfaces.exists(_.getName == "framework.Facet"))
          .map(_.getSimpleName)
          .toSet
        require(
          interfaces == expectedFacets(name),
          s"actual compiled facet mixins differ for $name: $interfaces"
        )
        s"$name: ${interfaces.toList.sorted.mkString(", ")}"
      }
      val vocabularies = List(
        "temporal.actors.worker.FacetName$" -> Set("serving"),
        "temporal.foundations.taskqueue.FacetName$" -> Set("queueing"),
        "temporal.features.activity.mini.FacetName$" -> Set(
          "dispatching",
          "completion",
          "answering"
        )
      ).map { (name, expected) =>
        val companion = Class.forName(name, true, loader).getField("MODULE$").get(null)
        val values = companion.getClass
          .getMethod("values")
          .invoke(companion)
          .asInstanceOf[Array[Object]]
          .map(_.toString)
          .toSet
        require(values == expected, s"actual compiled facet vocabulary differs for $name: $values")
        s"$name: ${values.toList.sorted.mkString(", ")}"
      }
      Files.writeString(
        archive.resolve("selected-facets-vocabulary.txt"),
        (selected ++ vocabularies).mkString("", "\n", "\n")
      ): Unit
    finally
      zip.close()
      loader.close()
    val rules = Rules(List(new Fn161Task4FacetWrapping))
    def rewrite(source: String) =
      rules.syntacticPatch(SyntacticDocument.fromTree(parsed(source)), suppress = false)
    val targets = List(
      ("protocol", "AnswerDispatch.scala", "AnswerDispatch", "Answering"),
      ("protocol", "Protocol.scala", "ActivityProtocol", "Completion"),
      ("product", "Product.scala", "ActivityProduct", "Completion")
    )
    val prepared = targets.map { (level, file, name, kind) =>
      val directory = feature.resolve(level)
      val path = directory.resolve(file)
      val companion = directory.resolve(kind + ".scala")
      val accepted = Files.readString(path)
      val acceptedFacet = Files.readString(companion)
      val original = machine(parsed(accepted), name)
      val contributor = facet(parsed(acceptedFacet), kind)
      val namespace = contributor.templ.body.stats.collect { case o: Defn.Object => o }
      require(namespace.size == 1, "the independently selected facet has one namespace")
      val start = acceptedFacet.lastIndexOf('\n', namespace.head.pos.start - 1) + 1
      val body = acceptedFacet.substring(start, namespace.head.pos.end)
      val parents = original.templ.inits
      require(
        parents.last.tpe.syntax == kind && parents.size >= 2,
        "the captured machine's final mixin is the selected facet"
      )
      val parentStart = parents.head.pos.start
      val keptEnd = parents.init.last.pos.end
      val stripped = accepted.substring(original.pos.start, parentStart) +
        accepted.substring(parentStart, keptEnd) +
        accepted.substring(parents.last.pos.end, original.pos.end)
      def headerImports(tree: Source, before: Int) = tree.collect {
        case i: Import if i.pos.end <= before => i
      }
      val machineImports = headerImports(parsed(accepted), original.pos.start)
      val extra = additionalImports(
        machineImports,
        headerImports(parsed(acceptedFacet), contributor.pos.start)
      )
      val legacy = accepted.take(original.pos.start) + extra.map(_ + "\n").mkString +
        stripped + "\n" + body + accepted.drop(original.pos.end)
      val label = level + "-" + kind
      Files.writeString(archive.resolve(label + ".machine.accepted.txt"), accepted): Unit
      Files.writeString(archive.resolve(label + ".facet.accepted.txt"), acceptedFacet): Unit
      Files.writeString(archive.resolve(label + ".legacy.txt"), legacy): Unit
      Files.move(companion, archive.resolve(label + ".facet.original.scala.txt")): Unit
      Files.writeString(path, legacy): Unit
      (directory, file, kind, accepted, acceptedFacet, legacy, label)
    }
    val model = campaign.resolve("model")
    val sources = Seq(
      "project.scala",
      "framework",
      "temporal/Bounds.scala",
      "temporal/foundations",
      "temporal/actors",
      "temporal/capabilities",
      "temporal/realize",
      "temporal/features/activity/Activity.scala",
      "temporal/features/activity_mini"
    ).map(model.resolve(_).toString)
    def run(label: String, command: Seq[String], output: Path): Int =
      val process = ProcessBuilder(command*)
        .directory(repository.toFile)
        .redirectOutput(output.toFile)
        .redirectError(archive.resolve(label + ".stderr.log").toFile)
        .start()
      process.waitFor()
    val scala = Seq("mise", "exec", "--", "scala-cli")
    val legacyJar = archive.resolve("legacy.jar")
    val packaged = run(
      "legacy-compile",
      scala ++ Seq(
        "--power",
        "package",
        "--server=false",
        "--suppress-outdated-dependency-warning",
        "--library"
      ) ++ sources ++ Seq("-f", "-o", legacyJar.toString),
      archive.resolve("legacy-compile.stdout.log")
    )
    require(packaged == 0, s"actual legacy API compilation failed; inspect $archive")
    val mutableClasspath = archive.resolve("legacy.mutable-classpath")
    require(
      run(
        "legacy-classpath",
        scala ++ Seq(
          "compile",
          "--server=false",
          "--suppress-outdated-dependency-warning",
          "--print-class-path"
        ) ++ sources,
        mutableClasspath
      ) == 0
    )
    val cp = Files.readString(mutableClasspath).trim
    require(cp.contains(":"), "the compiled legacy classpath must be nonempty")
    val frozenClasspath = archive.resolve("legacy.classpath")
    Files.writeString(frozenClasspath, s"$legacyJar:${cp.substring(cp.indexOf(':') + 1)}\n"): Unit
    val refusedOutput = Files.createDirectory(archive.resolve("legacy-refused-lift"))
    val lifted = run(
      "legacy-lift",
      scala ++ Seq(
        "run",
        "--server=false",
        "--suppress-outdated-dependency-warning",
        "model/framework/export",
        "model/temporal/export",
        "-M",
        "framework.export.lift",
        "--",
        "--ir",
        s"$legacyJar=$model/",
        frozenClasspath.toString,
        refusedOutput.toString,
        "activity-mini"
      ),
      archive.resolve("legacy-lift.stdout.log")
    )
    require(lifted != 0, "current facet admission must refuse the compiled legacy declarations")
    val refusal = Files.readString(archive.resolve("legacy-lift.stderr.log")) +
      Files.readString(archive.resolve("legacy-lift.stdout.log"))
    for (namespace, owner) <- List(
        "answering" -> "AnswerDispatch",
        "completion" -> "ActivityProduct",
        "completion" -> "ActivityProtocol"
      )
    do
      require(
        refusal.contains(s"$namespace is an object in $owner and none of its sections"),
        s"legacy admission must refuse the actual inline namespace in $owner"
      )
    require(
      !refusal.contains("R4.dotted-in"),
      "legacy diagnostics must read the actual copied sources"
    )
    require(
      !Files.exists(refusedOutput.resolve("activity-mini.json")),
      "refusal must not publish partial IR"
    )
    for (directory, file, kind, accepted, acceptedFacet, legacy, label) <- prepared do
      val result = rewrite(legacy)
      Files.writeString(archive.resolve(label + ".fixed.txt"), result.fixed): Unit
      require(
        result.diagnostics.isEmpty && result.fixed != legacy,
        result.diagnostics.map(_.message).mkString("; ")
      )
      val again = rewrite(result.fixed)
      require(
        again.diagnostics.isEmpty && again.fixed == result.fixed,
        "actual compiled migration must converge"
      )
      val ambiguous = legacy + s"\ntrait $kind\n"
      val refused = rewrite(ambiguous)
      require(
        refused.diagnostics.nonEmpty && refused.fixed == ambiguous,
        "actual ambiguity refusal must be atomic"
      )
      Files.writeString(
        archive.resolve(label + ".refusal.txt"),
        refused.diagnostics.map(_.message).mkString("\n")
      ): Unit
      writeCompanion(
        directory,
        file,
        legacy,
        result.fixed,
        kind + ".scala",
        accepted,
        acceptedFacet
      )
      require(Files.readString(directory.resolve(file)) == accepted)
      require(Files.readString(directory.resolve(kind + ".scala")) == acceptedFacet)
    println(
      s"PASS: 3 compiled copied-mini namespace migrations; old API compile, current guard refusal, " +
        s"exact accepted restoration, convergence and atomic ambiguity; evidence $archive"
    )

  private def parsed(text: String): Source = dialects.Scala3(Input.String(text)).parse[Source] match
    case Parsed.Success(tree) => tree
    case error: Parsed.Error  => throw IllegalArgumentException(error.message)

  private def machine(tree: Source, name: String): Defn.Object =
    val found = tree.collect { case o: Defn.Object if o.name.value == name => o }
    require(found.size == 1, s"one machine $name is required")
    found.head

  private def facet(tree: Source, name: String): Defn.Trait =
    val found = tree.collect { case t: Defn.Trait if t.name.value == name => t }
    require(found.size == 1, s"one facet $name is required")
    found.head

  private def without(source: String, ranges: List[(Int, Int)]): String =
    ranges.sortBy(-_._1).foldLeft(source) { case (text, (start, end)) =>
      text.take(start) + text.drop(end)
    }

  def writeCompanion(
      directory: Path,
      file: String,
      legacy: String,
      fixed: String,
      companion: String,
      acceptedMachine: String,
      acceptedFacet: String
  ): Unit =
    val owner = (file, companion) match
      case ("AnswerDispatch.scala", "Answering.scala") => "AnswerDispatch" -> "Answering"
      case ("Protocol.scala", "Completion.scala")      => "ActivityProtocol" -> "Completion"
      case ("Product.scala", "Completion.scala")       => "ActivityProduct" -> "Completion"
      case _                                           =>
        throw IllegalArgumentException("the inventoried matching companion path is required")
    val owned = Path.of(".flow/tmp").toRealPath()
    val real = directory.toRealPath()
    require(
      real.startsWith(owned) && real != owned,
      "only an isolated copied-source directory is writable"
    )
    require(
      directory.toAbsolutePath.normalize == real,
      "symlinked directory ownership is ambiguous"
    )
    val source = real.resolve(file)
    val target = real.resolve(companion)
    require(
      !Files.isSymbolicLink(source) && Files.isRegularFile(source),
      "machine must be an owned regular file"
    )
    require(
      !Files.exists(target, LinkOption.NOFOLLOW_LINKS),
      "companion destination already exists"
    )
    require(Files.readString(source) == legacy, "machine preimage changed before extraction")
    val oldTree = parsed(legacy)
    val fixedTree = parsed(fixed)
    val originalTree = parsed(acceptedMachine)
    val originalFacetTree = parsed(acceptedFacet)
    def packages(tree: Source) = tree.collect { case p: Pkg => p.ref.syntax }.mkString(".")
    val expectedPackage = if file == "Product.scala" then "temporal.features.activity.mini.product"
    else "temporal.features.activity.mini.protocol"
    require(
      List(oldTree, fixedTree, originalTree, originalFacetTree).forall(
        packages(_) == expectedPackage
      ),
      "all declarations must have the independently inventoried owner"
    )
    val oldMachine = machine(oldTree, owner._1)
    val newMachine = machine(fixedTree, owner._1)
    val original = machine(originalTree, owner._1)
    val extracted = facet(fixedTree, owner._2)
    val originalFacet = facet(originalFacetTree, owner._2)
    def headerImports(tree: Source, declaration: Tree) = tree.collect {
      case i: Import if i.pos.end <= declaration.pos.start => i
    }
    val originalImportTrees = headerImports(originalTree, original)
    val originalImports = originalImportTrees.map(_.syntax)
    val addedImports =
      additionalImports(originalImportTrees, headerImports(originalFacetTree, originalFacet))
    val legacyImports = headerImports(oldTree, oldMachine)
    require(
      legacyImports.map(_.syntax) == originalImports ++ addedImports,
      "only independently captured seed imports may be added"
    )
    val seedImports = legacyImports.filterNot(i => originalImports.contains(i.syntax))
    val seedImportRanges = seedImports.map { i =>
      val start = legacy.lastIndexOf('\n', i.pos.start - 1) + 1
      require(
        start == i.pos.start && legacy.lift(i.pos.end).contains('\n'),
        "seed imports must occupy their own complete lines"
      )
      start -> (i.pos.end + 1)
    }
    require(
      without(legacy, (oldMachine.pos.start -> oldMachine.pos.end) :: seedImportRanges) ==
        without(acceptedMachine, List(original.pos.start -> original.pos.end)),
      "unaccounted source outside the machine cannot be erased during layout restoration"
    )
    require(
      newMachine.structure == original.structure,
      "extracted machine meaning differs from the source pin"
    )
    require(
      extracted.structure == originalFacet.structure,
      "extracted facet meaning differs from the source pin"
    )
    require(
      fixed.substring(extracted.pos.start - 2, extracted.pos.start) == "\n\n",
      "the compiled rule's exact companion insertion is required"
    )
    val oldOutside = without(legacy, List(oldMachine.pos.start -> oldMachine.pos.end))
    val fixedOutside = without(
      fixed,
      List(
        newMachine.pos.start -> newMachine.pos.end,
        (extracted.pos.start - 2) -> extracted.pos.end
      )
    )
    require(
      oldOutside == fixedOutside,
      "compiled rewrite changed source outside the extracted declarations"
    )
    val oldParents = original.templ.inits
    val newParents = newMachine.templ.inits
    require(
      oldParents.nonEmpty && newParents.map(_.structure) == oldParents.map(_.structure),
      "typed parent and dependency inventory changed"
    )
    val restoredParentLayout =
      acceptedMachine.substring(oldParents.head.pos.start, oldParents.last.pos.end)
    val actualMachine = fixed.substring(newMachine.pos.start, newMachine.pos.end)
    val parentStart = newParents.head.pos.start - newMachine.pos.start
    val parentEnd = newParents.last.pos.end - newMachine.pos.start
    val laidOut =
      actualMachine.take(parentStart) + restoredParentLayout + actualMachine.drop(parentEnd)
    require(
      laidOut == acceptedMachine.substring(original.pos.start, original.pos.end),
      "only the captured parent-clause layout may differ"
    )
    require(
      extracted.pos.text == originalFacet.pos.text,
      "facet body and dependency bytes must be preserved"
    )
    val machineResult =
      acceptedMachine.take(original.pos.start) + laidOut + acceptedMachine.drop(original.pos.end)
    val facetResult =
      acceptedFacet.take(originalFacet.pos.start) + extracted.pos.text + acceptedFacet.drop(
        originalFacet.pos.end
      )
    val temporary = Files.createTempFile(real, ".facet-machine-", ".tmp")
    var created = false
    try
      Files.writeString(temporary, machineResult): Unit
      Files.writeString(
        target,
        facetResult,
        StandardOpenOption.CREATE_NEW,
        StandardOpenOption.WRITE
      ): Unit
      created = true
      require(
        Files.readString(source) == legacy && !Files.isSymbolicLink(source),
        "machine preimage changed before publication"
      )
      Files.move(
        temporary,
        source,
        StandardCopyOption.ATOMIC_MOVE,
        StandardCopyOption.REPLACE_EXISTING
      ): Unit
    catch
      case NonFatal(error) =>
        if created then Files.delete(target)
        throw error
    finally Files.deleteIfExists(temporary): Unit
