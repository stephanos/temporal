---- MODULE EmptyCondition ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
CONSTANT \* @type: Int;
__umpire_world_divisor
CONSTANT \* @type: Bool;
__umpire_world_enabled
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Bool;
__umpire_arithmetic_error
VARIABLE \* @type: Int;
__umpire_choice
VARIABLE \* @type: Seq(Seq(Int));
__umpire_facts
VARIABLE \* @type: Bool;
__umpire_has_step
VARIABLE \* @type: Int;
__umpire_input
VARIABLE \* @type: Bool;
__umpire_invalid_step
VARIABLE \* @type: Seq(Int);
__umpire_outcome
VARIABLE \* @type: Bool;
__umpire_stutter
\* @type: <<Str, Bool, Int, Seq(Seq(Int)), Bool, Int, Bool, Seq(Int), Bool>>;
vars == <<__umpire_action, __umpire_arithmetic_error, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ ((TRUE)) /\ (__umpire_arithmetic_error = ~(TRUE /\ (TRUE)))
\* @type: Bool;
User_Check == (LET \* @type: Bool;
__umpire_s3 == (((TRUE) /\ (TRUE)) /\ (__umpire_world_divisor # 0)) /\ ((__umpire_SafeDiv(1, __umpire_world_divisor)) >= 0 /\ (__umpire_SafeDiv(1, __umpire_world_divisor)) <= 255) IN LET \* @type: Int;
__umpire_s2 == IF __umpire_s3 THEN (__umpire_SafeDiv(1, __umpire_world_divisor)) ELSE 0 IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2) > (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (TRUE) /\ (IF __umpire_world_enabled THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN (IF __umpire_world_enabled THEN __umpire_s4 ELSE FALSE) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (TRUE) /\ (TRUE) /\ (__umpire_s7) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(TRUE) \/ (TRUE)) /\ (~(__umpire_s8) \/ (TRUE)) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Check") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(TRUE))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
ModelNext == (User_Check)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<__umpire_action, __umpire_arithmetic_error, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
