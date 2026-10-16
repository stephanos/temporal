package framework.dsl

import scala.annotation.publicInBinary

sealed abstract class FieldDurability[S] private[framework] (
    private[framework] val owner: Machine[S, ?, ?],
    private[framework] val field: StateField[S, ?]
):
  private[framework] def check(): Unit
  private[framework] def reset(state: S): S
  private[framework] def replacement(state: S): Option[(List[String], Any)]
  private[framework] def isDurable(path: List[String]): Boolean = false
  final private[framework] def context: String =
    s"durability field ${field.path.mkString(".")} at ${field.location}"

@publicInBinary
object FieldDurability:
  @publicInBinary private[framework] def durable[S, A](
      machine: Machine[S, ?, ?],
      selected: StateField[S, A]
  ): FieldDurability[S] =
    new FieldDurability[S](machine, selected):
      def check(): Unit = ()
      def reset(state: S): S = state
      def replacement(state: S): Option[(List[String], Any)] = None
      override def isDurable(path: List[String]): Boolean = path.startsWith(selected.path)

  private def checkValue[S, A](field: StateField[S, A], value: A): Unit =
    val context = s"durability field ${field.path.mkString(".")} at ${field.location}"
    require(
      field
        .valid(value)
        .getOrElse(
          throw IllegalArgumentException(s"$context: unsupported reset validity evidence")
        ),
      s"$context: reset value outside the field domain"
    )

  @publicInBinary private[framework] def ephemeral[S, A](
      machine: Machine[S, ?, ?],
      selected: StateField[S, A],
      resetTo: A
  ): FieldDurability[S] =
    new FieldDurability[S](machine, selected):
      def check(): Unit = checkValue(selected, resetTo)
      def reset(state: S): S = selected.write(state, resetTo)
      def replacement(state: S): Option[(List[String], Any)] = Some(selected.path -> resetTo)

  @publicInBinary private[framework] def inMemory[S, A](
      machine: Machine[S, ?, ?],
      selected: StateField[S, A],
      entries: Vector[(A, A)]
  )(using data: Data.Structural[A]): FieldDurability[S] =
    new FieldDurability[S](machine, selected):
      def check(): Unit =
        entries.zipWithIndex.foreach: (entry, index) =>
          val (key, fallback) = entry
          checkValue(selected, key)
          checkValue(selected, fallback)
          require(
            !entries.take(index).exists(other => data.equivalent(key, other._1)),
            s"$context: duplicate in-memory key"
          )
          require(
            !entries.exists(other => data.equivalent(fallback, other._1)),
            s"$context: in-memory fallback must be durable"
          )
      def reset(state: S): S =
        entries.find(entry => data.equivalent(selected.get(state), entry._1)) match
          case Some((_, fallback)) => selected.write(state, fallback)
          case None                => state
      def replacement(state: S): Option[(List[String], Any)] =
        entries
          .find(entry => data.equivalent(selected.get(state), entry._1))
          .map((_, fallback) => selected.path -> fallback)

  private[framework] def validate[S](
      owner: Machine[S, ?, ?],
      descriptors: Vector[FieldDurability[S]],
      budgetPaths: Vector[List[String]]
  ): Unit =
    validate(owner, descriptors, budgetPaths, candidate => candidate eq owner)

  private[framework] def validate[S](
      owner: Machine[S, ?, ?],
      descriptors: Vector[FieldDurability[S]],
      budgetPaths: Vector[List[String]],
      ownerAllowed: Machine[?, ?, ?] => Boolean
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
