package framework.dsl

import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

class FiniteSchemaTest extends munit.FunSuite:
  private def encoded(ref: ir.TypeRef): Seq[String] =
    Seq(Base64.getEncoder.encodeToString(ref.toByteArray))
  private val closure = Seq(Base64.getEncoder.encodeToString(ir.Model().toByteArray))

  test("metadata attachment keeps the witness and catalog, is idempotent, and refuses conflicts"):
    val witness = new Finite[Boolean]:
      def values = throw new AssertionError("catalog evaluated by schema attachment")
    assertEquals(witness.schema, None)
    val root = encoded(ir.TypeRef(ir.TypeRef.Ref.Bool(com.google.protobuf.empty.Empty())))
    val same = witness.retainSchema("scala.Boolean", root, closure)
    assert(same eq witness)
    assert(witness.retainSchema("scala.Boolean", root, closure) eq witness)
    assert(witness.schema.get.root.ref.isBool)
    val other = encoded(ir.TypeRef(ir.TypeRef.Ref.IntRange(ir.IntRange(0, 2))))
    val refused =
      intercept[IllegalArgumentException](witness.retainSchema("scala.Boolean", other, closure))
    assert(refused.getMessage.contains("conflicting definitions"))
    assert(witness.schema.get.root.ref.isBool)
    assert(witness.retainSchema("other.Boolean", root, closure) eq witness)
    assertEquals(witness.schema, None)
    assertEquals(witness.schemaFor("scala.Boolean").get.identity, "scala.Boolean")
    assertEquals(witness.schemaFor("other.Boolean").get.identity, "other.Boolean")
