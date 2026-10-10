package conformance

import (
	"context"
	"fmt"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/predicate"
	"go.temporal.io/server/tools/umpire/internal/runtimecel"
	"go.temporal.io/server/tools/umpire/realization"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// GuardError is a Run Event source's guard that cannot be evaluated on a Run Event.
type GuardError struct {
	Event   int64
	Message string
}

func (e *GuardError) Error() string {
	return fmt.Sprintf("run event %d: the guard %s", e.Event, e.Message)
}

var eventKinds = map[umpirespb.RunEventSource_Kind]testpilotspb.RunEventKind{
	umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
	umpirespb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT,
	umpirespb.RunEventSource_KIND_DIAGNOSTIC:            testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
}

func admits(source *umpirespb.RunEventSource, event *testpilotspb.RunEvent) (bool, error) {
	return admitsContext(context.Background(), source, event)
}

func admitsContext(ctx context.Context, source *umpirespb.RunEventSource, event *testpilotspb.RunEvent) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	refused := func(says string) (bool, error) { return false, &GuardError{Event: event.GetSequence(), Message: says} }
	kind, known := eventKinds[source.GetKind()]
	if !known {
		return refused("is of a Run Event source of no known kind")
	}
	at := event.GetCoordinates()
	if event.GetKind() != kind || at.GetEntrypointId() != source.GetScript() || at.GetInstructionId() != source.GetCommand() || event.GetOutcome() == nil {
		return false, nil
	}
	if of := source.GetAttempt(); of != nil && int64(event.GetOutcome().GetActivityAttempt().GetSdkAttempt()) != of.GetNumber() {
		return false, nil
	}
	if source.GetGuard() == nil {
		return true, nil
	}
	payload := event.GetOutcome()
	if err := realization.GuardProblem(source.GetGuard(), payload.ProtoReflect().Descriptor()); err != nil {
		return refused(err.Error())
	}
	expression, err := runtimecel.Lower(source.GetGuard(), payload.ProtoReflect().Descriptor(), nil)
	if err != nil {
		return refused(err.Error())
	}
	packed, err := anypb.New(payload)
	if err != nil {
		return refused(err.Error())
	}
	typ := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: string(payload.ProtoReflect().Descriptor().FullName())}}}}}
	value, _, err := predicate.EvaluateProjected(ctx, expression, &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: packed}}, typ, guardDescriptors(payload.ProtoReflect().Descriptor().ParentFile()), int64(maxWork))
	if err != nil {
		return refused(err.Error())
	}
	flag, ok := value.GetKind().(*celpb.Value_BoolValue)
	if !ok {
		return refused("is an absent value, and a guard is a condition")
	}
	return flag.BoolValue, nil
}

func guardDescriptors(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen, out := map[string]bool{}, &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		for i := 0; i < file.Imports().Len(); i++ {
			add(file.Imports().Get(i))
		}
		out.File = append(out.File, protodesc.ToFileDescriptorProto(file))
	}
	add(root)
	return out
}
