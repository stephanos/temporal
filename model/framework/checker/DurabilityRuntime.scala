package framework.checker

import framework.dsl.*

private[framework] object DurabilityRuntime:
  private def writable[S](descriptor: FieldDurability[S]): StateField[S, Any] =
    descriptor.field.asInstanceOf[StateField[S, Any]] // scalafix:ok DisableSyntax.asInstanceOf

  private def checkValue[S](descriptor: FieldDurability[S], value: Any): Unit =
    require(
      writable(descriptor)
        .valid(value)
        .getOrElse(
          throw IllegalArgumentException(
            s"${descriptor.context}: unsupported reset validity evidence"
          )
        ),
      s"${descriptor.context}: reset value outside the field domain"
    )

  def validate[S](descriptor: FieldDurability[S]): Unit =
    descriptor.fallback.foreach(checkValue(descriptor, _))
    descriptor.data.foreach: data =>
      descriptor.entries.zipWithIndex.foreach: (entry, index) =>
        val (key, fallback) = entry
        checkValue(descriptor, key)
        checkValue(descriptor, fallback)
        require(
          !descriptor.entries.take(index).exists(other => data.equivalent(key, other._1)),
          s"${descriptor.context}: duplicate in-memory key"
        )
        require(
          !descriptor.entries.exists(other => data.equivalent(fallback, other._1)),
          s"${descriptor.context}: in-memory fallback must be durable"
        )

  def replacement[S](descriptor: FieldDurability[S], state: S): Option[(List[String], Any)] =
    descriptor.fallback
      .map(descriptor.field.path -> _)
      .orElse(
        descriptor.data.flatMap(data =>
          descriptor.entries
            .find(entry => data.equivalent(writable(descriptor).get(state), entry._1))
            .map((_, fallback) => descriptor.field.path -> fallback)
        )
      )

  def reset[S](descriptor: FieldDurability[S], state: S): S =
    replacement(descriptor, state).fold(state)((_, value) =>
      writable(descriptor).write(state, value)
    )
