package framework.lint

import framework.inspect.{Index, LiftError}
import framework.check.lint.Literals
import java.nio.file.{Files, Path}
import scala.collection.mutable

final private[lint] class ModelDeclarations[I <: Index](val index: I):
  val quotes: index.quotes.type = index.quotes
  import quotes.reflect.*
  import index.given

  private val machine = Symbol.requiredClass("framework.dsl.Machine")
  private val declaration = Symbol.requiredClass("framework.dsl.MachineDeclaration")
  private val state = Symbol.requiredClass("framework.dsl.State")
  private val facet = Symbol.requiredClass("framework.dsl.Facet")
  private val actorClass = Symbol.requiredClass("framework.dsl.Actor")
  private val internalClass = Symbol.requiredClass("framework.dsl.Internal")

  def native(owner: Symbol): Boolean =
    !owner.isNoSymbol && owner.isClassDef && (
      owner.typeRef.derivesFrom(machine) || (owner.typeRef.baseType(declaration) match
        case AppliedType(_, first :: _) => first.dealias.typeSymbol == state
        case _                          => false)
    )

  def effects(owner: Symbol): Boolean =
    !owner.isNoSymbol && owner.flags.is(Flags.Module) && owner.name.stripSuffix(
      "$"
    ) == "effects" && (
      native(owner.maybeOwner) || (
        owner.maybeOwner.flags.is(Flags.Module) &&
          !owner.maybeOwner.maybeOwner.isNoSymbol &&
          owner.maybeOwner.maybeOwner.isClassDef &&
          owner.maybeOwner.maybeOwner.typeRef.derivesFrom(facet)
      )
    )

  def signature(owner: Symbol): Boolean =
    !owner.isNoSymbol && owner.isClassDef &&
      (owner.typeRef.derivesFrom(actorClass) || owner.typeRef.derivesFrom(internalClass))

  def defaultGetter(method: DefDef): Boolean =
    method.name.contains("$default$") && {
      val source = Files.readString(Path.of(file(method)))
      val code = Literals.code(source, Literals.literals(source))
      val span = code.substring(method.pos.start, method.pos.end)
      val name = java.util.regex.Pattern.quote(method.name)
      !s"\\bdef\\s+`?$name`?(?=[\\s\\[(=:])".r.findFirstIn(span).nonEmpty
    }

  def member(symbol: Symbol): Boolean =
    def enclosing(owner: Symbol): Boolean =
      native(owner) || (!owner.isNoSymbol && owner.isClassDef &&
        owner.flags.is(Flags.Module) && enclosing(owner.maybeOwner))
    enclosing(symbol.maybeOwner)

  def file(tree: Tree): String =
    val raw = tree.pos.sourceFile.path
    index.sourceRoots.getOrElse(raw, "") + raw

  def authored(file: String): Boolean =
    val path = Path.of(file).normalize.toString.replace('\\', '/')
    index.authorSource(path) && !path.endsWith(".test.scala") &&
    (path.contains("/testdata/") || !Seq("export", "check", "lint", "inspect", "reify", "migrate")
      .exists(tool =>
        path.startsWith(s"model/framework/$tool/") || path.contains(s"/model/framework/$tool/")
      ))

final private[framework] class DeclarationStyle(index: Index):
  private val model = ModelDeclarations[index.type](index)
  import model.quotes.reflect.*
  import index.given

  private val sources = mutable.Map.empty[String, String]
  private val code = mutable.Map.empty[String, String]
  private def textOf(tree: Tree): String =
    val file = model.file(tree)
    sources.getOrElseUpdate(file, Files.readString(Path.of(file)))

  private def first(tree: Tree): Tree = tree match
    case Block(statements, result)       => first(statements.headOption.getOrElse(result))
    case Inlined(Some(call), _, _)       => first(call)
    case Inlined(None, bindings, result) => first(bindings.headOption.getOrElse(result))
    case Typed(value, _)                 => first(value)
    case Match(scrutinee, _)             => first(scrutinee)
    case _                               => tree

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    index.defs.values.toSeq
      .flatMap { definition =>
        val sym = definition.symbol
        val pattern = definition match
          case value: ValDef => sym.flags.is(Flags.Synthetic) && value.name.matches("\\$[0-9]+\\$")
          case _             => false
        val excluded = List(
          Flags.FieldAccessor,
          Flags.Param,
          Flags.ParamAccessor,
          Flags.Module,
          Flags.Given,
          Flags.Mutable
        ).exists(sym.flags.is)
        val file = model.file(definition)
        if excluded || (sym.flags.is(Flags.Synthetic) && !pattern) || exempt(file) ||
          !model.authored(file) || !model.member(sym) || (definition match
            case method: DefDef => model.defaultGetter(method)
            case _              => false)
        then Nil
        else
          val body = definition match
            case value: ValDef                             => value.rhs
            case method: DefDef if method.name != "<init>" => method.rhs
            case _                                         => None
          body.toList.flatMap { rhs =>
            val text = textOf(definition)
            val begin = first(rhs).pos.start
            val masked = code.getOrElseUpdate(file, Literals.code(text, Literals.literals(text)))
            val header = masked.lastIndexOf('=', begin - 1)
            if header < definition.pos.start || index
                .sourceLine(file, begin) > index.sourceLine(file, header)
            then Nil
            else
              val line = index.sourceLine(file, definition.pos.start)
              val name = if pattern then "val pattern" else definition.name
              List(
                (
                  file,
                  line,
                  LiftError(
                    s"$file:$line",
                    s"R4.declaration-multiline: ${name} starts its body on the definition line: put the body on the next line"
                  )
                )
              )
          }
      }
      .groupBy((file, line, _) => (file, line))
      .values
      .map(_.head)
      .toSeq
