package fixture.typedProtoInvalid

import com.google.protobuf.ByteString
import io.temporal.api.command.v1.Command
import io.temporal.api.common.v1.Payload
import io.temporal.api.enums.v1.{ActivityExecutionStatus, CommandType}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import framework.realize.*

val wrongMessage = Proto[Payload](
  ProtoField.typed(Field[Failure, String](_.message), ProtoValue.text("bad"))
)
val wrongScalar = Proto[Failure](
  ProtoField.typed(Field[Failure, String](_.message), ProtoValue.flag(true))
)
val wrongEnum = Proto[Command](
  ProtoField.typed(
    Field[Command, CommandType](_.commandType),
    ProtoValue.enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED)
  )
)
val wrongNested = Proto[Failure](
  ProtoField.typed(
    Field[Failure, ApplicationFailureInfo](_.getApplicationFailureInfo),
    ProtoValue.message(Proto[Payload]())
  )
)
val wrongMapKey = ProtoEntry.typed(1, ProtoValue.utf8("bytes"))
val wrongMapValue = ProtoEntry.typed("encoding", ProtoValue.text("json/plain"))
val unknownEnum = CommandType.NO_SUCH_COMMAND_TYPE
val unrecognizedEnum = ProtoValue.enumValue(CommandType.Unrecognized(999))
val unrecognizedOperand = Operand.enumValue(CommandType.Unrecognized(999))
val unrecognizedField = Proto[Command](
  ProtoField.typed(
    Field[Command, CommandType](_.commandType),
    ProtoValue.enumValue(CommandType.Unrecognized(999))
  )
)
val wrongField = Field[Payload, ByteString](_.missing)
val forgedValue = new TypedProtoValue[CommandType](ProtoValue.text("forged"))
val forgedMessage = new TypedProto[Payload](Vector.empty)
