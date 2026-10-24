---- MODULE Starts ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
\* @type: Int;
LIMIT == 127
VARIABLE \* @type: Int;
value
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Bool;
__umpire_arithmetic_error
VARIABLE \* @type: Int;
__umpire_before_value
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
\* @type: <<Int, Str, Bool, Int, Int, Seq(Seq(Int)), Bool, Int, Bool, Seq(Int), Bool>>;
vars == <<value, __umpire_action, __umpire_arithmetic_error, __umpire_before_value, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == LET \* @type: Bool;
__umpire_s3 == ((TRUE) /\ (TRUE)) /\ (((127) + (1)) >= -128 /\ ((127) + (1)) <= 127) IN LET \* @type: Int;
__umpire_s2 == IF __umpire_s3 THEN ((127) + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_s4 == (TRUE) /\ ((TRUE) /\ (__umpire_s3)) IN (__umpire_before_value = 0) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ ((IF __umpire_s4 THEN (value \in (1..(__umpire_s2))) ELSE value = 0)) /\ (__umpire_arithmetic_error = ~(__umpire_s4 /\ (TRUE)))
\* @type: Bool;
User_Idle == (LET \* @type: Bool;
__umpire_s2 == (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(TRUE) \/ (TRUE)) /\ (~(__umpire_s2) \/ ((value # value))) /\ (value' = IF __umpire_s2 THEN value ELSE value) /\ (__umpire_before_value' = value) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Idle") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s2 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s2) \/ ~(TRUE))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
ModelNext == (User_Idle)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<value, __umpire_action, __umpire_arithmetic_error, __umpire_before_value, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
