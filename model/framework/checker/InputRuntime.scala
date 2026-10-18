package framework.checker

import framework.dsl.*

private[framework] object InputRuntime extends InputEvaluation:
  private def validFields[I](shape: ActionInputs[I], value: I): Boolean = shape match
    case record: ActionInputs.Record[?] =>
      val product = value.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
      product.productArity == record.fields.size && record.fields
        .zip(product.productIterator.toList)
        .forall((domain, field) => domain.values.exists(_.equals(field)))
    case _ => true

  private def contains[I](shape: ActionInputs[I], value: I): Boolean = shape match
    case record: ActionInputs.Record[?] if record.fieldSelection => true
    case _ => shape.domain.values.exists(_.equals(value))

  def values[I](shape: ActionInputs[I], value: I): List[Any] = shape match
    case _: ActionInputs.Scalar[?] => List(value)
    case _: ActionInputs.Record[?] =>
      value.asInstanceOf[Product].productIterator.toList // scalafix:ok DisableSyntax.asInstanceOf

  def fromValues[I](shape: ActionInputs[I], values: List[Any]): I = shape match
    case _: ActionInputs.Scalar[?] =>
      require(values.size == 1, "a scalar input has exactly one value")
      values.head.asInstanceOf[I] // scalafix:ok DisableSyntax.asInstanceOf
    case record: ActionInputs.Record[?] =>
      record.mirror
        .fromProduct(Tuple.fromArray(values.toArray))
        .asInstanceOf[I] // scalafix:ok DisableSyntax.asInstanceOf

  def defaultAt[I](shape: ActionInputs[I], index: Int): Option[Any] = shape match
    case scalar: ActionInputs.Scalar[?] => if index == 0 then scalar.default else None
    case record: ActionInputs.Record[?] => record.defaults.lift(index).flatMap(_())

  def omitted[I](shape: ActionInputs[I], decl: ActionDecl): Class =
    val value = shape match
      case scalar: ActionInputs.Scalar[?] =>
        scalar.default.getOrElse(
          throw IllegalArgumentException(
            s"input ${decl.inputs.head} of ${decl.name} has no default: supply it"
          )
        )
      case record: ActionInputs.Record[?] =>
        val fields = record.defaults.zipWithIndex.map((default, index) =>
          default().getOrElse(
            throw IllegalArgumentException(s"input ${decl.inputs(index)} has no default: supply it")
          )
        )
        record.mirror.fromProduct(Tuple.fromArray(fields.toArray))
    classOf(shape, decl, value.asInstanceOf[I]) // scalafix:ok DisableSyntax.asInstanceOf

  def classes[I](shape: ActionInputs[I]): List[List[Any]] = shape.domain.values.toList.map: value =>
    require(validFields(shape, value), "input value is outside its declared field domains")
    values(shape, value)

  def inputClasses[I](shape: ActionInputs[I], decl: ActionDecl): List[Class] =
    shape.domain.values.toList.map(classOf(shape, decl, _))

  def input[I](shape: ActionInputs[I], cls: Class): I =
    require(cls.decl.nativeInputs.exists(_ eq shape), "input class belongs to another action shape")
    val value = cls.supplied.fold(fromValues(shape, cls.values))(
      _.value.asInstanceOf[I] // scalafix:ok DisableSyntax.asInstanceOf
    )
    require(
      validFields(shape, value) && contains(shape, value),
      "input value is outside its declared domain"
    )
    if cls.supplied.isEmpty then cls.supplied = Some(SuppliedInput(value))
    value

  def retained[I](shape: ActionInputs[I], decl: ActionDecl, value: I): Class =
    val result = Class(decl, values(shape, value))
    result.supplied = Some(SuppliedInput(value))
    result

  def classOf[I](shape: ActionInputs[I], decl: ActionDecl, value: I): Class =
    require(
      validFields(shape, value) && contains(shape, value),
      s"input value of ${decl.name} is outside its declared domain"
    )
    retained(shape, decl, value)
