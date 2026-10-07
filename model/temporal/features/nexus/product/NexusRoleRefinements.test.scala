package umpire

import temporal.features.nexus.{product, standalone, workflow}
import workflow.system.given

// The Nexus refinements whose System and Product phases both take roles preserve closedness: a
// System state is Closed exactly when its Product image is, over every Finite System state. A
// role-carrying refinement added later is added here by name.
class NexusRoleRefinements extends munit.FunSuite:
  // The System states whose Closed role differs from their Product image's, each with its image.
  def unclosed[S: Finite](toProduct: S => product.State)(closed: S => Boolean) =
    Finite[S].values
      .map(s => s -> toProduct(s))
      .filter((s, image) => closed(s) != image.phase.in[Closed])

  test("the Nexus workflow System's refinement keeps closedness"):
    import workflow.system.NexusSystem.refinement
    assertEquals(unclosed(refinement.toProduct)(_.phase.in[Closed]), IndexedSeq.empty)

  test("the Nexus standalone System's refinement keeps closedness"):
    import standalone.system.NexusSystem.refinement
    assertEquals(unclosed(refinement.toProduct)(_.phase.in[Closed]), IndexedSeq.empty)
