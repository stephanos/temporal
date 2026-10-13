package temporal.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn161Task4FacetWrapping extends SyntacticRule("Fn161Task4FacetWrapping"):
  override def fix(using doc: SyntacticDocument): Patch =
    val owner = doc.tree.collect { case p: Pkg => p.ref.syntax }.mkString(".")
    val targets = owner match
      case "temporal.features.activity.mini.protocol" =>
        Map(
          "AnswerDispatch" -> (
            "Answering",
            "answering",
            "AnswerState",
            "AnswerFact",
            "AnswerPhase"
          ),
          "ActivityProtocol" -> ("Completion", "completion", "State", "Fact", "Phase")
        )
      case "temporal.features.activity.mini.product" =>
        Map("ActivityProduct" -> ("Completion", "completion", "State", "Fact", "Phase"))
      case _ => Map.empty[String, (String, String, String, String, String)]
    val machines = doc.tree.collect {
      case o: Defn.Object if targets.contains(o.name.value) => o
    }
    val imports = doc.tree.collect { case i: Importer => i }
    val reserved = Set("Machine", "Phased", "Facet", "Facets", "FacetCapabilities", "Rules")
    // The DSL's package, as the real features still spell it and as fn-169.1 spells it.
    def dsl(ref: String) = ref == "framework" || ref == "framework.dsl"
    val framework = imports.exists(i =>
      dsl(i.ref.syntax) && i.importees.exists(_.isInstanceOf[Importee.Wildcard])
    )
    val competing = imports.exists(i =>
      !dsl(i.ref.syntax) && i.importees.exists {
        case n: Importee.Name     => reserved(n.name.value)
        case n: Importee.Rename   => reserved(n.rename.value)
        case _: Importee.Wildcard => true
        case _                    => false
      }
    )
    val source = doc.input.text
    val patches = scala.collection.mutable.ListBuffer.empty[Patch]
    val refused = scala.collection.mutable.ListBuffer.empty[Tree]
    for machine <- machines do
      val (facet, namespace, state, fact, phase) = targets(machine.name.value)
      val namespaces = machine.templ.body.stats.collect {
        case n: Defn.Object if n.name.value == namespace => n
      }
      if namespaces.nonEmpty then
        val own = namespaces.head
        val parents = machine.templ.inits
        val kinds = parents.map(_.tpe.syntax)
        val dependency = if facet == "Completion" then Some("Dispatching") else None
        val aggregation = machine.templ.body.stats.collect {
          case o: Defn.Object if o.name.value == "rules" => o
        }
        val expectedRules = dependency.fold(s"Facets($namespace.rules)")(_ =>
          s"Facets(dispatching.rules, $namespace.rules)"
        )
        val destinations = doc.tree.collect {
          case t: Defn.Trait if t.name.value == facet  => t
          case t: Defn.Class if t.name.value == facet  => t
          case t: Defn.Object if t.name.value == facet => t
        }
        val sections = Set("states", "effects", "rules", "properties", "capabilities")
        val badBody = own.templ.inits.nonEmpty || own.templ.body.stats.exists {
          case o: Defn.Object => !sections(o.name.value)
          case _              => true
        } || own.collect {
          case _: Defn.Var                                                      => true
          case a: Term.Apply if Set("action", "framework.action", "framework.dsl.action")(
                a.fun.syntax
              ) =>
            true
        }.nonEmpty
        val namespaceNames = own.templ.body.stats.collect { case o: Defn.Object =>
          o.name.value
        }.toSet
        val topLevel = machine.parent.exists {
          case _: Source | _: Pkg | _: Pkg.Body => true
          case _                                => false
        }
        val machineSections = Set(
          namespace,
          "states",
          "rules",
          "properties",
          "capabilities",
          "queries",
          "monitors",
          "refinement",
          "projection"
        )
        val leftover = machine.templ.body.stats.exists {
          case o: Defn.Object => !machineSections(o.name.value)
          case _              => false
        }
        val localMachineNames = machine.templ.body.stats
          .filterNot(_ == own)
          .flatMap {
            case v: Defn.Val    => v.pats.collect { case p: Pat.Var => p.name.value }
            case d: Defn.Def    => List(d.name.value)
            case o: Defn.Object => List(o.name.value)
            case _              => Nil
          }
          .toSet -- namespaceNames
        val escaped = own.collect {
          case n: Term.Name if localMachineNames(n.value) && !n.parent.exists {
                case s: Term.Select => s.name.pos == n.pos
                case _              => false
              } =>
            n
        }.nonEmpty
        val lineStart = source.lastIndexOf('\n', own.pos.start - 1) + 1
        val exactIndent = source.substring(lineStart, own.pos.start) == "  "
        val valid =
          topLevel && !leftover && namespaces.size == 1 && framework && !competing && destinations.isEmpty &&
            kinds.contains(s"Machine[$state, Outcome, $fact]") &&
            kinds.contains(s"Phased[$state, $phase]") && !kinds.contains(facet) &&
            dependency.forall(kinds.contains) && aggregation.size == 1 &&
            aggregation.head.templ.inits.map(_.syntax) == List(expectedRules) &&
            aggregation.head.templ.body.stats.isEmpty && !badBody && !escaped && exactIndent
        if !valid then refused += own
        else
          val end = (own.pos.end + (if source.lift(own.pos.end).contains('\n') then 1 else 0))
            .min(machine.pos.end)
          val body = source.substring(lineStart, own.pos.end)
          val before = source.substring(machine.pos.start, lineStart)
          val after = source.substring(end, machine.pos.end)
          val insertion = parents.last.pos.end - machine.pos.start
          val assembled = before.substring(0, insertion) + s", $facet" +
            before.substring(insertion) + after
          val self = dependency.fold("\n")(d => s"  this: $d =>\n")
          val declaration = s"trait $facet extends Facet[$state, Outcome, $fact], " +
            s"Phased[$state, $phase]:\n$self$body"
          patches += Patch.replaceTree(machine, assembled.stripSuffix("\n") + "\n\n" + declaration)
    if refused.nonEmpty then
      Patch.lint(
        Diagnostic(
          "unsupported-facet-wrapping",
          "unsupported or ambiguous facet ownership: manually inventory the namespace, its " +
            "action owners, imports, dependencies and matching companion-file destination; " +
            "preserve machine-wide declarations and compile/lift before accepting the extraction",
          refused.head.pos
        )
      )
    else patches.toList.asPatch.atomic
