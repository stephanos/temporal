// The Models the lifter's tests lift and compare with expected/, and the declarations they expect the
// lifter to refuse. They build against the framework and Temporal Models the gate packaged.
//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns
// Model/capability parameters are intentionally retained to pin the lifter's declared signatures.
//> using options "-Wconf:id=E198&msg=unused explicit parameter:s"
// These expressions deliberately reach the lifter's bare-expression, unheaded-rule and wrong-field
// refusals; making them compiler refusals would stop those diagnostic controls from running.
//> using options "-Wconf:src=/BlockRejects[.]scala$&id=E176&msg=unused value of type Boolean:s"
//> using options "-Wconf:src=/Rejects[.]scala$&id=E176&msg=unused value of type framework.StepBinding:s"
//> using options "-Wconf:src=/ScriptRejects[.]scala$&id=E175&msg=framework.Assigned:s"
// RequestScope is a Unit collector boundary. Its two core TypedAssignment expressions are lifted
// as assignments; adding a Unit ascription makes that supported core syntax a lifter refusal.
//> using options "-Wconf:src=/Scripts[.]scala$&id=E175&msg=framework.realize.TypedAssignment:s"
//> using options "-Wconf:src=/Scripts[.]scala$&id=E176&msg=framework.realize.TypedAssignment:s"
//> using jar ../../../build/model-scala.jar
//> using jar ../../../build/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
