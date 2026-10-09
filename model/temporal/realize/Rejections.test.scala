package temporal.realize

import framework.outcomes.Rejection

class RejectionsTest extends munit.FunSuite:
  test("the Temporal rejection-code table is exhaustive and unique"):
    assertEquals(
      rejectionCodes.map(c => c.rejection -> c.grpcCode),
      Vector(
        Rejection.notFound -> "NOT_FOUND",
        Rejection.alreadyExists -> "ALREADY_EXISTS",
        Rejection.failedPrecondition -> "FAILED_PRECONDITION",
        Rejection.invalidArgument -> "INVALID_ARGUMENT"
      )
    )
    assertEquals(rejectionCodes.map(_.rejection).distinct, Rejection.values.toVector)
    assertEquals(
      rejectionCodes.map(c => c.rejection -> c.grpcCode),
      Rejection.values.toVector.map(rejection => rejection -> grpcCode(rejection))
    )
