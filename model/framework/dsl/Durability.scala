package framework.dsl

import scala.annotation.publicInBinary

sealed abstract class FieldDurability[S] private[framework] (
    private[framework] val owner: RecordMachine[S, ?, ?],
    private[framework] val field: StateField[S, ?],
    private[framework] val durable: Boolean,
    private[framework] val fallback: Option[Any] = None,
    private[framework] val entries: Vector[(Any, Any)] = Vector.empty,
    private[framework] val data: Option[Data[Any]] = None
):
  final private[framework] def check(): Unit = NativeRuntime.installed.checkDurability(this)
  final private[framework] def reset(state: S): S =
    NativeRuntime.installed.resetDurability(this, state)
  final private[framework] def replacement(state: S): Option[(List[String], Any)] =
    NativeRuntime.installed.durabilityReplacement(this, state)
  final private[framework] def isDurable(path: List[String]): Boolean =
    durable && path.startsWith(field.path)
  final private[framework] def context: String =
    s"durability field ${field.path.mkString(".")} at ${field.location}"

@publicInBinary
object FieldDurability:
  @publicInBinary private[framework] def durable[S, A](
      machine: RecordMachine[S, ?, ?],
      selected: StateField[S, A]
  ): FieldDurability[S] = new FieldDurability[S](machine, selected, true) {}

  @publicInBinary private[framework] def ephemeral[S, A](
      machine: RecordMachine[S, ?, ?],
      selected: StateField[S, A],
      resetTo: A
  ): FieldDurability[S] = new FieldDurability[S](machine, selected, false, Some(resetTo)) {}

  @publicInBinary private[framework] def inMemory[S, A](
      machine: RecordMachine[S, ?, ?],
      selected: StateField[S, A],
      entries: Vector[(A, A)]
  )(using data: Data.Structural[A]): FieldDurability[S] =
    new FieldDurability[S](
      machine,
      selected,
      false,
      entries = entries,
      data = Some(data.asInstanceOf[Data[Any]])
    ) {} // scalafix:ok DisableSyntax.asInstanceOf

  private[framework] def validate[S](
      owner: RecordMachine[S, ?, ?],
      descriptors: Vector[FieldDurability[S]],
      budgetPaths: Vector[List[String]]
  ): Unit =
    validate(owner, descriptors, budgetPaths, candidate => candidate eq owner)

  private[framework] def validate[S](
      owner: RecordMachine[S, ?, ?],
      descriptors: Vector[FieldDurability[S]],
      budgetPaths: Vector[List[String]],
      ownerAllowed: RecordMachine[?, ?, ?] => Boolean
  ): Unit =
    descriptors.zipWithIndex.foreach: (descriptor, index) =>
      require(
        ownerAllowed(descriptor.owner),
        s"${descriptor.context}: descriptor belongs to another machine ${owner.name}"
      )
      require(
        !descriptors
          .take(index)
          .exists(previous =>
            descriptor.field.path.startsWith(previous.field.path) || previous.field.path
              .startsWith(descriptor.field.path)
          ),
        s"${descriptor.context}: duplicate or overlapping classification"
      )
      descriptor.check()
    descriptors.headOption.foreach: first =>
      first.field.rootFields.foreach: path =>
        require(
          descriptors.exists(d => path.startsWith(d.field.path)),
          s"durability field ${path.mkString(".")} at ${first.field.location}: unclassified state field"
        )
    budgetPaths.foreach: path =>
      val location = descriptors
        .find(d => path.startsWith(d.field.path))
        .orElse(descriptors.headOption)
        .map(_.field.location)
        .getOrElse(owner.name)
      require(
        descriptors.exists(_.isDurable(path)),
        s"durability budget field ${path.mkString(".")} at $location: budget must be durable"
      )

inline def durable[S, A, O, F](using
    owner: Owner[S, O, F]
)(inline selector: S => A): FieldDurability[S] =
  FieldDurability.durable(owner.machine, StateField.capture(selector))

inline def ephemeral[S, A, O, F](using
    owner: Owner[S, O, F]
)(inline selector: S => A, resetTo: A): FieldDurability[S] =
  FieldDurability.ephemeral(owner.machine, StateField.capture(selector), resetTo)

inline def inMemory[S, A, O, F](using
    owner: Owner[S, O, F]
)(inline selector: S => A)(entries: (A, A)*)(using data: Data.Structural[A]): FieldDurability[S] =
  FieldDurability.inMemory(owner.machine, StateField.capture(selector), entries.toVector)
