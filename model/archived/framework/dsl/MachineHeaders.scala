package framework.dsl

import io.temporal.server.api.umpire.v1 as ir

private[framework] object MachineHeaders:
  final case class Header(
      family: String,
      name: String,
      stateType: String,
      outcomeType: String,
      factType: String
  ) derives CanEqual
  final case class Contribution(header: Header, types: Vector[ir.Type]) derives CanEqual

  final class Metadata:
    private var declaration = Option.empty[(String, String)]
    private var selected = Vector.fill[Option[String]](3)(None)
    def identity: Option[(String, String)] = declaration
    def keys: Vector[Option[String]] = selected
    private def context: String =
      declaration.fold("Machine header")((family, name) => s"Machine $family.$name")
    def retainIdentity(family: String, name: String): Unit = synchronized {
      require(name.nonEmpty, s"$context: no declaration name")
      require(
        declaration.forall(_ == (family -> name)),
        s"$context: conflicting declaration identity $family.$name"
      )
      declaration = Some(family -> name)
    }
    def retainSchema(
        witnesses: Vector[Data[?]],
        index: Int,
        key: String,
        root: Seq[String],
        closure: Seq[String],
        inboxes: Seq[String]
    ): Unit = synchronized {
      require(0 <= index && index < 3, s"$context: invalid schema selection $index")
      val at = s"$context ${Vector("state", "outcome", "fact")(index)} type $key"
      require(key.nonEmpty, s"$at: no schema key")
      require(selected(index).forall(_ == key), s"$at: conflicting schema selection")
      val schema = FiniteSchema.decode(key, root, closure, inboxes)
      require(!schema.root.ref.isEmpty, s"$at: no retained type reference")
      val canonical = schema.root.ref match
        case ir.TypeRef.Ref.Named(name)      => Some(name)
        case ir.TypeRef.Ref.Bool(_)          => Some("scala.Boolean")
        case ir.TypeRef.Ref.Int(_)           => Some("scala.Int")
        case ir.TypeRef.Ref.Channel(message) => Some(s"framework.dsl.Inbox[$message]")
        case _                               => None
      require(
        canonical.forall(_ == key),
        s"$at: selected root requires key ${canonical.getOrElse(key)}"
      )
      val witness = witnesses(index)
      require(witness ne null, s"$at: witness is not initialized")
      try witness.retainSchema(key, root, closure, inboxes): Unit
      catch
        case error: IllegalArgumentException =>
          throw IllegalArgumentException(s"$at: ${error.getMessage}", error)
      selected = selected.updated(index, Some(key))
    }

  def collect(
      machine: MachineDeclaration[?, ?, ?],
      nested: Seq[(Data[?], String)] = Nil
  ): Contribution =
    require(machine ne null, "Machine header: declaration is not initialized")
    val (family, name) = machine.headerIdentity.getOrElse(
      throw IllegalArgumentException("Machine header: no retained declaration identity")
    )
    val context = s"Machine $family.$name"
    val witnesses = Vector(machine.fs, machine.fo, machine.ff)
    val schemas =
      witnesses.zip(machine.headerSchemaKeys).zipWithIndex.map { case ((witness, key), index) =>
        val at = s"$context ${Vector("state", "outcome", "fact")(index)} type"
        require(witness ne null, s"$at: witness is not initialized")
        key
          .flatMap(witness.schemaFor)
          .getOrElse(throw IllegalArgumentException(s"$at: no retained typed schema selection"))
      }
    val selected =
      witnesses.zip(machine.headerSchemaKeys) ++ nested.map((witness, key) => witness -> Some(key))
    selected.foreach { (witness, key) =>
      val schema = key
        .flatMap(witness.schemaFor)
        .getOrElse(
          throw IllegalArgumentException(
            s"$context type ${key.getOrElse("<missing>")}: no retained typed schema"
          )
        )
      require(
        !schema.root.ref.isEmpty,
        s"$context type ${schema.identity}: no retained type reference"
      )
      schema.types.foreach { definition =>
        val fields = definition.getRecord.fields.map("" -> _) ++
          definition.getEnum.cases.flatMap(c => c.fields.map(c.name -> _))
        fields.foreach { (caseName, field) =>
          val coordinate = InboxOwnership.Field(definition.name, caseName, field.name)
          require(
            field.`type`.nonEmpty && !field.getType.ref.isEmpty,
            s"$context type ${schema.identity} field ${coordinate.context}: no retained type reference"
          )
        }
      }
    }
    val types = Signature.stateTypes(selected, context)
    Contribution(
      Header(
        family,
        name,
        schemas(0).root.getNamed,
        schemas(1).root.getNamed,
        if schemas(2).identity == "scala.Nothing" then "" else schemas(2).root.getNamed
      ),
      types
    )
