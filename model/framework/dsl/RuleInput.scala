package framework.dsl

sealed trait NativeInput[I]

sealed trait RuleInput[I, A]

object RuleInput:
  given native[I]: RuleInput[NativeInput[I], I] = new RuleInput[NativeInput[I], I] {}
