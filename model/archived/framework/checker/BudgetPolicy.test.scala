package framework.checker

class BudgetPolicyTest extends munit.FunSuite:
  test("an absent policy uses the baseline; explicit policies resolve each suite independently"):
    assertEquals(BudgetPolicy.parse(None).forSuite("example.DefaultSuite"), 1 << 20)
    val policy = BudgetPolicy.parse(
      Some("example.SecondSuite=19\ndefault=31\nexample.FirstSuite=7"),
      "gate configuration"
    )
    policy.validate(Set("example.FirstSuite", "example.SecondSuite", "example.DefaultSuite"))
    assertEquals(policy.forSuite("example.FirstSuite"), 7)
    assertEquals(policy.forSuite("example.SecondSuite"), 19)
    assertEquals(policy.forSuite("example.DefaultSuite"), 31)
    assertEquals(
      policy.normalized,
      "default=31\nexample.FirstSuite=7\nexample.SecondSuite=19"
    )
    assertEquals(BudgetPolicy.parse(Some(policy.normalized)).normalized, policy.normalized)

  test("invalid budgets and malformed or duplicate entries name their configuration source"):
    val invalid = Vector(
      "default=0",
      "default=-1",
      "default=no",
      "default=2147483648",
      "default=1\ndefault=2",
      "example.Suite=1\nexample.Suite=2",
      "broken..Suite=1",
      "default=1=2",
      "=1",
      "default",
      ""
    )
    invalid.foreach: value =>
      val error = intercept[IllegalArgumentException]:
        BudgetPolicy.parse(Some(value), "gate configuration")
      assert(error.getMessage.contains("gate configuration"), value)

  test("unknown overrides name the suite and source while known unselected suites stay valid"):
    val policy = BudgetPolicy.parse(Some("example.HeldSuite=9"), "gate configuration")
    policy.validate(Set("example.HeldSuite", "example.SelectedSuite"))
    val error = intercept[IllegalArgumentException]:
      policy.validate(Set("example.SelectedSuite"))
    assert(error.getMessage.contains("example.HeldSuite"))
    assert(error.getMessage.contains("gate configuration"))

  test("a source-discovered suite in the default package resolves by its exact identity"):
    val policy = BudgetPolicy.parse(Some("SimpleSuite=7"), "gate configuration")
    policy.validate(Set("SimpleSuite"))
    assertEquals(policy.forSuite("SimpleSuite"), 7)
