package framework.reify

import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Flags.{Accessor, Module, Synthetic}
import dotty.tools.dotc.core.NameKinds.DefaultGetterName
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.plugins.PluginPhase
import dotty.tools.dotc.report
import dotty.tools.dotc.transform.{Pickler, PostTyper}

final private class EffectsPhase extends PluginPhase:
  val phaseName = "reifyEffects"
  override val runsAfter = Set(PostTyper.name)
  override val runsBefore = Set("reifyStore", Pickler.name)

  private def native(owner: Symbol)(using Context): Boolean =
    val machine = getClassIfDefined("framework.dsl.Machine")
    val record = getClassIfDefined("framework.dsl.RecordMachine")
    val state = getClassIfDefined("framework.dsl.State")
    owner.exists && owner.isClass && (
      (machine.exists && owner.asClass.derivesFrom(machine)) ||
        (record.exists && state.exists && owner.typeRef
          .baseType(record)
          .argInfos
          .headOption
          .exists(_.widenDealias.typeSymbol == state))
    )

  private def effects(owner: Symbol)(using Context): Boolean =
    owner.exists && owner.is(Module) && owner.name.toString.stripSuffix("$") == "effects" && (
      native(owner.owner) ||
        FacetDeclaration.namespace(owner.owner)
    )

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    val method = tree.symbol
    if !method.is(Synthetic) && !method.is(
        Accessor
      ) && method.name.toString != "<init>" && !method.name.is(DefaultGetterName) &&
      effects(method.owner) && !(method.info.finalResultType.widenDealias =:= defn.UnitType)
    then report.error(s"${method.name}: effect definitions must return Unit", tree.srcPos)
    tree
