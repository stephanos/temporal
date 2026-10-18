package framework.dsl

sealed trait NativeInput[I]

sealed trait RuleInput[I, A]:
  private[framework] def native: I =:= NativeInput[A]

object RuleInput:
  given native[I]: RuleInput[NativeInput[I], I] = new RuleInput[NativeInput[I], I]:
    private[framework] def native: NativeInput[I] =:= NativeInput[I] = summon
