package framework.dsl

import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

private[framework] object MachineEndpoints:
  final case class Contribution(
      starts: Vector[ir.Expr],
      end: ir.Expr,
      functions: Vector[ir.Function],
      types: Vector[ir.Type],
      channels: Vector[String],
      holes: Vector[String]
  )
  final case class Retained(
      source: MachineDeclaration[?, ?, ?],
      owner: String,
      family: String,
      name: String,
      starts: Vector[ir.Expr],
      end: ir.Expr,
      model: ir.Model
  )

  final class Metadata:
    private var retained = Option.empty[Retained]
    def value: Option[Retained] = synchronized(retained)
    def retain(
        source: MachineDeclaration[?, ?, ?],
        owner: String,
        family: String,
        name: String,
        starts: Seq[String],
        end: Seq[String],
        closure: Seq[String]
    ): Unit = synchronized {
      val context = s"Machine $family.$name endpoints"
      val decoder = Base64.getDecoder
      val payload = try
        Retained(
          source,
          owner,
          family,
          name,
          ir.ListOf.parseFrom(decoder.decode(starts.mkString)).items.toVector,
          ir.Expr.parseFrom(decoder.decode(end.mkString)),
          ir.Model.parseFrom(decoder.decode(closure.mkString))
        )
      catch
        case error: IllegalArgumentException =>
          throw IllegalArgumentException(s"$context: malformed retained payload", error)
        case error: com.google.protobuf.InvalidProtocolBufferException =>
          throw IllegalArgumentException(s"$context: malformed retained payload", error)
      checked(source, payload): Unit
      require(
        retained.forall(_.equals(payload)),
        s"$context: conflicting endpoint provenance or payload"
      )
      retained = Some(payload)
    }

  private def checked(source: MachineDeclaration[?, ?, ?], value: Retained): Contribution =
    val context = s"Machine ${value.family}.${value.name} endpoints"
    require(value.source.eq(source), s"$context: stale endpoint source")
    require(value.owner.nonEmpty && value.name.nonEmpty, s"$context: missing endpoint provenance")
    require(
      source.getClass.getName == value.owner.replace("$.", "$"),
      s"$context: stale endpoint owner ${value.owner}"
    )
    require(
      source.headerIdentity.contains(value.family -> value.name),
      s"$context: missing or stale endpoint declaration registration"
    )
    require(value.starts.nonEmpty, s"$context: missing retained starts")
    require(
      value.end.kind.isLambda && value.end.getLambda.params.size == 1,
      s"$context: end is not a retained state Lambda"
    )
    require(
      value.model.copy(functions = Nil, types = Nil).equals(ir.Model()),
      s"$context: endpoint payload contains unrelated declarations"
    )
    val dependencies = CoreBindings.dependencies(
      value.starts :+ value.end,
      Nil,
      value.model.functions,
      value.model.types,
      context
    )
    Contribution(
      value.starts,
      value.end,
      dependencies.functions,
      dependencies.types,
      dependencies.channels,
      dependencies.holes
    )

  def collect(machine: MachineDeclaration[?, ?, ?]): Contribution =
    require(machine ne null, "Machine endpoints: declaration is not initialized")
    val source = machine.endpointSource
    require(source ne null, "Machine endpoints: source is not initialized")
    val (family, name) = source.headerIdentity.getOrElse(
      throw IllegalArgumentException(
        "Machine endpoints: declaration registration is pending or absent"
      )
    )
    val payload = source.endpointMetadata.getOrElse(
      throw IllegalArgumentException(
        s"Machine $family.$name endpoints: no retained endpoint metadata"
      )
    )
    checked(source, payload)
