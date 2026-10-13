package framework.`export`

import io.temporal.server.api.umpire.v1 as ir
import scalapb.descriptors.{OneofDescriptor, PValue, ScalaType}

private[`export`] object Equality:
  given CanEqual[ir.Value, ir.Value] = CanEqual.derived
  given valueKind: CanEqual[ir.Value.Kind, ir.Value.Kind] = CanEqual.derived
  given exprKind: CanEqual[ir.Expr.Kind, ir.Expr.Kind] = CanEqual.derived
  given CanEqual[ir.TypeRef.Ref, ir.TypeRef.Ref] = CanEqual.derived
  given CanEqual[ir.Limits, ir.Limits] = CanEqual.derived
  given CanEqual[ir.Query, ir.Query] = CanEqual.derived
  given CanEqual[ir.Scenario, ir.Scenario] = CanEqual.derived
  given CanEqual[ir.Property, ir.Property] = CanEqual.derived
  given CanEqual[ir.Progress, ir.Progress] = CanEqual.derived
  given CanEqual[ir.RunExpectation.Outcome, ir.RunExpectation.Outcome] = CanEqual.derived
  given CanEqual[ir.Query.Form, ir.Query.Form] = CanEqual.derived
  given CanEqual[ir.TimeoutBasis, ir.TimeoutBasis] = CanEqual.derived
  given CanEqual[ir.ServerStep, ir.ServerStep] = CanEqual.derived
  given CanEqual[ir.ActionClass, ir.ActionClass] = CanEqual.derived
  given CanEqual[OneofDescriptor, OneofDescriptor] = CanEqual.derived
  given CanEqual[ScalaType, ScalaType] = CanEqual.derived
  given CanEqual[PValue, PValue] = CanEqual.derived
  given CanEqual[io.grpc.MethodDescriptor.MethodType, io.grpc.MethodDescriptor.MethodType] =
    CanEqual.derived
