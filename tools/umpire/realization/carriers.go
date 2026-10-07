package realization

import (
	"fmt"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// The carrier of an action class is the protobuf message the system carries when a realization
// performs the class (fn-133 R17). It is derived from the realization's binding of the class to the
// instruction that performs it, never declared beside the action: a call carries its method's
// request, a workflow command the command attributes it sets, a Nexus handler's answer or completion
// the message it answers with, an activity attempt's answer the request the worker responds with,
// and the start of an activity script the task the worker polls. A fault, a hold or a release, a
// wait and a read carry nothing the action owns, so the classes they perform have no carrier; nor
// does a class no binding performs.

// Carrier is what one instruction of a realization carries for one class it performs.
type Carrier struct {
	// The instruction kind it is derived from: rpc, workflow-command, nexus-reply, nexus-completion,
	// activity-answer or activity-delivery.
	Kind string
	// The full method name of a call, "/package.Service/Method"; empty for another kind.
	Method string
	// The carried message's full name.
	Message string
}

// Mapping is the carriers of one class of a realization, in the order its scripts bind them: one
// per script that performs the class.
type Mapping struct {
	Class    *umpirespb.ActionClass
	Carriers []Carrier
}

// Carriers is each class the realization performs and what performing it carries, in binding order.
// A class bound to an instruction that carries nothing is listed with no carrier. Two different
// carriers of one class in one script are ambiguous and refused, as is a call of a method the
// registry does not hold.
func Carriers(r *umpirespb.Realization, files *protoregistry.Files) ([]Mapping, error) {
	var out []Mapping
	index := map[string]int{}
	// The carriers each script binds each class to.
	perScript := map[string][]Carrier{}
	add := func(script string, class *umpirespb.ActionClass, c *Carrier) error {
		key := classKey(class)
		i, ok := index[key]
		if !ok {
			i = len(out)
			index[key] = i
			out = append(out, Mapping{Class: class})
		}
		if c == nil {
			return nil
		}
		for _, existing := range out[i].Carriers {
			if existing == *c {
				return nil
			}
		}
		for _, other := range perScript[script+"\x00"+key] {
			if other != *c {
				return fmt.Errorf("realization %s: script %s performs %s with two carriers, %s and %s: the class's carrier is ambiguous",
					r.GetName(), script, key, other.Message, c.Message)
			}
		}
		perScript[script+"\x00"+key] = append(perScript[script+"\x00"+key], *c)
		out[i].Carriers = append(out[i].Carriers, *c)
		return nil
	}
	for _, s := range r.GetScripts() {
		if a := s.GetActivity(); a != nil {
			for _, start := range a.GetStarts() {
				if err := add(s.GetId(), start, &Carrier{Kind: "activity-delivery", Message: pollActivityTask}); err != nil {
					return nil, err
				}
			}
		}
		for _, item := range s.GetItems() {
			for _, p := range item.GetPerforms() {
				c, err := carrierOf(s, p.GetCommand(), files)
				if err != nil {
					return nil, fmt.Errorf("realization %s: %w", r.GetName(), err)
				}
				if err := add(s.GetId(), p.GetStep(), c); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

const (
	pollActivityTask      = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"
	respondCompleted      = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest"
	respondFailed         = "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest"
	respondCanceled       = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest"
	commandAttributesTail = "_command_attributes"
)

// carrierOf is what the command carries, or nil for an instruction that carries nothing.
func carrierOf(s *umpirespb.Script, c *umpirespb.Command, files *protoregistry.Files) (*Carrier, error) {
	switch {
	case c.GetRpc() != nil:
		method := c.GetRpc().GetMethod()
		request, err := requestOf(method, files)
		if err != nil {
			return nil, fmt.Errorf("command %s: %w", c.GetId(), err)
		}
		return &Carrier{Kind: "rpc", Method: method, Message: request}, nil
	case c.GetWorkflowCommand() != nil:
		command := c.GetWorkflowCommand().GetCommand()
		for _, f := range command.GetFields() {
			if strings.HasSuffix(f.GetName(), commandAttributesTail) && f.GetValue().GetMessage() != nil {
				return &Carrier{Kind: "workflow-command", Message: f.GetValue().GetMessage().GetMessage()}, nil
			}
		}
		return nil, fmt.Errorf("command %s: the workflow command %s sets no command attributes to carry", c.GetId(), command.GetMessage())
	case c.GetNexusReply() != nil:
		return &Carrier{Kind: "nexus-reply", Message: c.GetNexusReply().GetReply().GetMessage()}, nil
	case c.GetNexusCompletion() != nil:
		return &Carrier{Kind: "nexus-completion", Message: c.GetNexusCompletion().GetResult().GetMessage()}, nil
	case s.GetActivity() != nil && c.GetFinish() != nil:
		return &Carrier{Kind: "activity-answer", Message: respondCompleted}, nil
	case s.GetActivity() != nil && c.GetAttemptFailure() != nil:
		return &Carrier{Kind: "activity-answer", Message: respondFailed}, nil
	case s.GetActivity() != nil && c.GetAttemptCanceled() != nil:
		return &Carrier{Kind: "activity-answer", Message: respondCanceled}, nil
	default:
		return nil, nil
	}
}

// requestOf is the full name of the request message of a unary method, "/package.Service/Method".
func requestOf(method string, files *protoregistry.Files) (string, error) {
	service, name, ok := strings.Cut(strings.TrimPrefix(method, "/"), "/")
	if !ok {
		return "", fmt.Errorf("%s is no method name", method)
	}
	d, err := files.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		return "", fmt.Errorf("%s names a service the registry does not hold: %w", method, err)
	}
	sd, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return "", fmt.Errorf("%s is no service", service)
	}
	md := sd.Methods().ByName(protoreflect.Name(name))
	if md == nil {
		return "", fmt.Errorf("%s has no method %s", service, name)
	}
	return string(md.Input().FullName()), nil
}

// classKey is a class as its action and each input's encoding, unique per class.
func classKey(c *umpirespb.ActionClass) string {
	var b strings.Builder
	b.WriteString(c.GetAction())
	for _, in := range c.GetInputs() {
		raw, _ := proto.MarshalOptions{Deterministic: true}.Marshal(in)
		fmt.Fprintf(&b, "|%x", raw)
	}
	return b.String()
}
