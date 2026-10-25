---- MODULE Fallback ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
VARIABLE \* @type: Seq(Int);
values
VARIABLE \* @type: Int;
ceiling
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Bool;
__umpire_arithmetic_error
VARIABLE \* @type: Int;
__umpire_before_ceiling
VARIABLE \* @type: Seq(Int);
__umpire_before_values
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
\* @type: <<Seq(Int), Int, Str, Bool, Int, Seq(Int), Int, Seq(Seq(Int)), Bool, Int, Bool, Seq(Int), Bool>>;
vars == <<values, ceiling, __umpire_action, __umpire_arithmetic_error, __umpire_before_ceiling, __umpire_before_values, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Seq(Int);
__umpire_s2 == IF __umpire_s3 THEN (<<>>) ELSE <<>> IN LET \* @type: Bool;
__umpire_s4 == (TRUE) /\ (__umpire_s3) IN LET \* @type: Bool;
__umpire_s6 == (__umpire_s4) /\ ((TRUE) /\ (TRUE)) IN (__umpire_before_values = <<>>) /\ (__umpire_before_ceiling = 0) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ ((IF __umpire_s4 THEN (values = __umpire_s2) ELSE values = <<>>) /\ (IF __umpire_s6 THEN (ceiling \in (255..(255))) ELSE ceiling = 0)) /\ (__umpire_arithmetic_error = ~(__umpire_s6 /\ (LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Int;
__umpire_c2 == IF __umpire_c3 THEN (Len(values)) ELSE 0 IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == __umpire_c5 IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN (~(__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == __umpire_c7 IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN (~(__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == ((TRUE) /\ (TRUE)) /\ (((ceiling) + (1)) >= 0 /\ ((ceiling) + (1)) <= 255) IN LET \* @type: Int;
__umpire_c10 == IF __umpire_c11 THEN ((ceiling) + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_c13 == (TRUE) /\ (TRUE) /\ (__umpire_c11) IN LET \* @type: Int;
__umpire_c12 == IF __umpire_c13 THEN (IF 0 >= 0 /\ 0 < Len(values) THEN values[0 + 1] ELSE __umpire_c10) ELSE 0 IN LET \* @type: Bool;
__umpire_c15 == (__umpire_c13) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c14 == IF __umpire_c15 THEN ((__umpire_c12) = (7)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c17 == (__umpire_c15) IN LET \* @type: Bool;
__umpire_c16 == IF __umpire_c17 THEN ((__umpire_c14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c19 == (__umpire_c9) /\ (IF __umpire_c8 THEN TRUE ELSE __umpire_c17) IN LET \* @type: Bool;
__umpire_c18 == IF __umpire_c19 THEN ((__umpire_c8) \/ (__umpire_c16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c21 == (__umpire_c19) IN LET \* @type: Bool;
__umpire_c20 == IF __umpire_c21 THEN ((__umpire_c18)) ELSE FALSE IN (__umpire_c21))))
\* @type: Bool;
User_Push == (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Seq(Int);
__umpire_s2 == IF __umpire_s3 THEN (Append(values, 7)) ELSE <<>> IN LET \* @type: Bool;
__umpire_s5 == TRUE IN LET \* @type: Int;
__umpire_s4 == IF __umpire_s5 THEN (Len(values)) ELSE 0 IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (__umpire_s3) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((__umpire_s2 # values) \/ (ceiling # ceiling))) /\ (values' = IF __umpire_s8 THEN __umpire_s2 ELSE values) /\ (__umpire_before_values' = values) /\ (ceiling' = IF __umpire_s8 THEN ceiling ELSE ceiling) /\ (__umpire_before_ceiling' = ceiling) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Push") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Int;
__umpire_c2 == IF __umpire_c3 THEN (Len(values')) ELSE 0 IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == __umpire_c5 IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN (~(__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == __umpire_c7 IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN (~(__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == ((TRUE) /\ (TRUE)) /\ (((ceiling') + (1)) >= 0 /\ ((ceiling') + (1)) <= 255) IN LET \* @type: Int;
__umpire_c10 == IF __umpire_c11 THEN ((ceiling') + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_c13 == (TRUE) /\ (TRUE) /\ (__umpire_c11) IN LET \* @type: Int;
__umpire_c12 == IF __umpire_c13 THEN (IF 0 >= 0 /\ 0 < Len(values') THEN values'[0 + 1] ELSE __umpire_c10) ELSE 0 IN LET \* @type: Bool;
__umpire_c15 == (__umpire_c13) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c14 == IF __umpire_c15 THEN ((__umpire_c12) = (7)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c17 == (__umpire_c15) IN LET \* @type: Bool;
__umpire_c16 == IF __umpire_c17 THEN ((__umpire_c14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c19 == (__umpire_c9) /\ (IF __umpire_c8 THEN TRUE ELSE __umpire_c17) IN LET \* @type: Bool;
__umpire_c18 == IF __umpire_c19 THEN ((__umpire_c8) \/ (__umpire_c16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c21 == (__umpire_c19) IN LET \* @type: Bool;
__umpire_c20 == IF __umpire_c21 THEN ((__umpire_c18)) ELSE FALSE IN (__umpire_c21)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Int;
__umpire_s2 == IF __umpire_s3 THEN (Len(values)) ELSE 0 IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == __umpire_s5 IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN (~(__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) /\ (IF __umpire_s6 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s10 == (__umpire_s9) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s9) \/ (__umpire_s8)) /\ (~(__umpire_s10) \/ ((values # values) \/ (ceiling # ceiling))) /\ (values' = IF __umpire_s10 THEN values ELSE values) /\ (__umpire_before_values' = values) /\ (ceiling' = IF __umpire_s10 THEN ceiling ELSE ceiling) /\ (__umpire_before_ceiling' = ceiling) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Push") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s10 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s10) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Int;
__umpire_c2 == IF __umpire_c3 THEN (Len(values')) ELSE 0 IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == __umpire_c5 IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN (~(__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == __umpire_c7 IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN (~(__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == ((TRUE) /\ (TRUE)) /\ (((ceiling') + (1)) >= 0 /\ ((ceiling') + (1)) <= 255) IN LET \* @type: Int;
__umpire_c10 == IF __umpire_c11 THEN ((ceiling') + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_c13 == (TRUE) /\ (TRUE) /\ (__umpire_c11) IN LET \* @type: Int;
__umpire_c12 == IF __umpire_c13 THEN (IF 0 >= 0 /\ 0 < Len(values') THEN values'[0 + 1] ELSE __umpire_c10) ELSE 0 IN LET \* @type: Bool;
__umpire_c15 == (__umpire_c13) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c14 == IF __umpire_c15 THEN ((__umpire_c12) = (7)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c17 == (__umpire_c15) IN LET \* @type: Bool;
__umpire_c16 == IF __umpire_c17 THEN ((__umpire_c14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c19 == (__umpire_c9) /\ (IF __umpire_c8 THEN TRUE ELSE __umpire_c17) IN LET \* @type: Bool;
__umpire_c18 == IF __umpire_c19 THEN ((__umpire_c8) \/ (__umpire_c16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c21 == (__umpire_c19) IN LET \* @type: Bool;
__umpire_c20 == IF __umpire_c21 THEN ((__umpire_c18)) ELSE FALSE IN (__umpire_c21)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
ModelNext == (User_Push)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<values, ceiling, __umpire_action, __umpire_arithmetic_error, __umpire_before_ceiling, __umpire_before_values, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
Invariant_eager_default == LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Int;
__umpire_s2 == IF __umpire_s3 THEN (Len(values)) ELSE 0 IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == __umpire_s5 IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN (~(__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == __umpire_s7 IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN (~(__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == ((TRUE) /\ (TRUE)) /\ (((ceiling) + (1)) >= 0 /\ ((ceiling) + (1)) <= 255) IN LET \* @type: Int;
__umpire_s10 == IF __umpire_s11 THEN ((ceiling) + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_s13 == (TRUE) /\ (TRUE) /\ (__umpire_s11) IN LET \* @type: Int;
__umpire_s12 == IF __umpire_s13 THEN (IF 0 >= 0 /\ 0 < Len(values) THEN values[0 + 1] ELSE __umpire_s10) ELSE 0 IN LET \* @type: Bool;
__umpire_s15 == (__umpire_s13) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s14 == IF __umpire_s15 THEN ((__umpire_s12) = (7)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s17 == (__umpire_s15) IN LET \* @type: Bool;
__umpire_s16 == IF __umpire_s17 THEN ((__umpire_s14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s19 == (__umpire_s9) /\ (IF __umpire_s8 THEN TRUE ELSE __umpire_s17) IN LET \* @type: Bool;
__umpire_s18 == IF __umpire_s19 THEN ((__umpire_s8) \/ (__umpire_s16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s21 == (__umpire_s19) IN LET \* @type: Bool;
__umpire_s20 == IF __umpire_s21 THEN ((__umpire_s18)) ELSE FALSE IN __umpire_arithmetic_error \/ __umpire_s20
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
