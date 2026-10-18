package framework.dsl

import scala.collection.mutable

// What a machine's section of `on` blocks registers, whichever words its blocks are written in: the
// rules of each action, or class of it, in declaration order, the actions no state enables, the
// generated crashes, and the facet the section sits in. It refuses a block in a block, a bare
// target, a target its `from` does not declare, and an action both disabled and fired. The two
// sections that write blocks extend it and add their words; a block is opened here and its rules
// bound here.
abstract class RuleTable[S, O, F] private[framework] (using
    private[framework] val tableOwner: MachineDeclaration[S, O, F],
    private[framework] val tableSource: FacetSource
) extends RuleBook[S, O, F]:
  private var origin = Option.empty[FacetBindings.Origin]
  final override private[framework] def facetOrigin: Option[FacetBindings.Origin] = origin
  @scala.annotation.publicInBinary
  final private[framework] def retainFacetOrigin(
      assembly: MachineDeclaration[?, ?, ?],
      namespace: AnyRef,
      traitClass: java.lang.Class[?],
      traitName: String,
      namespaceName: String,
      rulesName: String
  ): Unit = synchronized {
    require(assembly.eq(tableOwner), "Machine Facets: foreign RuleBook assembly source")
    require(
      tableSource.facet.nonEmpty && traitName.nonEmpty && namespaceName.nonEmpty &&
        rulesName.nonEmpty,
      "Machine Facets: missing facet definition identity"
    )
    require(
      (traitClass ne null) && traitClass.isInterface && traitClass.isInstance(assembly),
      "Machine Facets: facet trait token does not own this assembly"
    )
    require(
      assembly.ownsFacetNamespace(namespace, traitName, namespaceName, traitClass),
      "Machine Facets: foreign or unregistered facet namespace receiver"
    )
    val identity =
      new FacetSource.Identity(assembly, namespace, traitClass, traitName, namespaceName, rulesName)
    origin match
      case Some(previous) =>
        require(
          previous.source.identity.exists(old =>
            old.source.eq(assembly) && old.namespace.eq(namespace) && old.traitClass.eq(
              traitClass
            ) &&
              old.traitName == traitName && old.namespaceName == namespaceName && old.rulesName == rulesName
          ),
          "Machine Facets: conflicting RuleBook origin"
        )
      case None =>
        val provenance = new FacetSource(tableSource.facet, tableSource.section, Some(identity))
        origin = Some(new FacetBindings.Origin(this, provenance))
  }
  override private[framework] def facetSource: Option[String] = tableSource.facet
  override private[framework] def sectionName: String = tableSource.section
  override private[framework] def actionName(decl: ActionDecl): String = names(decl)
  private val names = mutable.Map.empty[ActionDecl, String]
  private val written = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  private val never = mutable.ArrayBuffer.empty[ActionDecl]
  private[framework] val capturedDisabledRows = mutable.Map.empty[ActionDecl, Bound[S, O, F]]
  private val order = mutable.LinkedHashSet.empty[ActionDecl]
  private val generatedCrashes = mutable.Set.empty[ActionDecl]
  private var open: Option[String] = None // scalafix:ok DisableSyntax.var
  // The `from` the blocks are in, if any: its declarer's name and the actions it declares.
  private var within: Option[(String, Set[ActionDecl])] =
    None // scalafix:ok DisableSyntax.var

  // A generated crash: its reset is the one rule of its action, which holds in every state.
  private[framework] def registerCrash(binding: StepBinding[S, O, F], spelling: String): Unit =
    require(
      !order.contains(binding.decl),
      s"crashes($spelling) at ${binding.decl.nativeFaultLocation}: duplicate crash binding"
    )
    val reset =
      binding.function
        .asInstanceOf[S => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
    openBlock(binding.decl, spelling):
      bindRule(binding.decl, None, "always", _ => true, Programs.OfState(reset), spelling, None)
    generatedCrashes += binding.decl: Unit

  // Runs one block, refusing a block in a block, a bare target, and in a `from` an action its
  // declarer does not declare.
  private[framework] def openBlock(decl: ActionDecl, code: String)(body: => Unit): Unit =
    val action = writtenAction(code)
    require(
      !generatedCrashes.contains(decl),
      s"on($action) at ${decl.nativeFaultLocation}: duplicate crash binding"
    )
    names(decl) = action
    require(
      writtenPath(code).contains('.'),
      s"on($action) names its action bare: name it through the object that declares it, " +
        s"`on(${within.fold("<declarer>")(_._1)}.$action)`"
    )
    require(
      open.isEmpty,
      s"on($action) sits in on(${open.get}): a block holds cases alone"
    )
    require(!never.contains(decl), s"$action is disabled and fired by a rule")
    for (declarer, declared) <- within do
      require(
        declared.contains(decl),
        s"on($action) sits in from($declarer), and $declarer declares no $action: a from holds " +
          "the blocks of the actions its declarer declares"
      )
    open = Some(action)
    try body
    finally open = None

  // Registers a rule in declaration order.
  private[framework] def bindRule(
      decl: ActionDecl,
      values: Option[List[Any]],
      heading: String,
      guard: S => Boolean,
      effect: (S, List[Any]) => List[Step[S, O, F]],
      code: String,
      captured: Option[RuleBindings.Registration]
  ): Unit =
    val rule = Rule(
      written.size + 1,
      heading,
      writtenAction(code),
      decl,
      values,
      guard,
      effect
    )
    rule.publishCapture(captured)
    written += rule
    order += decl

  // The targets of one block that names several, each with its class values and spelling,
  // refusing a target named twice.
  private[framework] def blockTargets(
      named: List[(Action[?] | Class, String)]
  ): List[(ActionDecl, Option[List[Any]], String)] =
    val resolved = named.map:
      case (a: Action[?], code) => (a.decl, None, code)
      case (c: Class, code)     => (c.decl, Some(c.values), code)
    for ((decl, values, code), i) <- resolved.zipWithIndex do
      require(
        !resolved
          .take(i)
          .exists((d, v, _) => d == decl && java.util.Objects.equals(v, values)),
        s"${writtenAction(code)} is named twice in one on: name each action, or class of it, once"
      )
    resolved

  private[framework] def groupName: String = within.fold("<declarer>")(_._1)

  // Runs the blocks of the actions `declarer` declares, refusing an import of its members and a
  // group in a block or in another group.
  private[framework] def grouped(declarer: AnyRef, imported: String)(body: => Unit): Unit =
    val name = declarer match
      case a: Actor => a.name
      case other    => objectName(other)
    require(
      imported.isEmpty,
      s"from($name) imports members at $imported: a from only groups, so drop the import and " +
        s"name each action through its declarer, `on($name.<action>)`"
    )
    require(
      open.isEmpty,
      s"from($name) sits in on(${open.get}): a block holds cases alone"
    )
    require(
      within.isEmpty,
      s"from($name) sits in from(${within.get._1}): a from holds on blocks alone"
    )
    within = Some(name -> declaredBy(declarer))
    try body
    finally within = None

  // The actions an object declares: the `Action` values among its public vals. An action records
  // no object that holds it, so the object's members are read.
  private def declaredBy(declarer: AnyRef): Set[ActionDecl] =
    declarer.getClass.getMethods.toSet
      .filter(m =>
        m.getParameterCount == 0 && classOf[Action[?]]
          .isAssignableFrom(m.getReturnType)
      )
      .map(_.invoke(declarer))
      .collect { case a: Action[?] => a.decl }

  private[framework] def disable(labels: Vector[String], actions: Action[?]*): Unit =
    for (a, label) <- actions.zip(labels) do
      require(
        !never.contains(a.decl) && !written.exists(_.decl == a.decl),
        "an action is disabled twice, or disabled and bound by a rule"
      )
      never += a.decl
      order += a.decl
      names(a.decl) = label

  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    order.toVector.map: decl =>
      decl -> (if never.contains(decl) then
                 capturedDisabledRows.getOrElse(decl, Bound.Disabled[S, O, F]())
               else Bound.Ruled(written.filter(_.decl == decl).toVector))
