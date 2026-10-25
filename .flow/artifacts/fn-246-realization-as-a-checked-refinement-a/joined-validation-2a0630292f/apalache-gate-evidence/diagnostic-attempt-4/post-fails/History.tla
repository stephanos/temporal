---- MODULE History ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
VARIABLE \* @type: Bool;
done
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Bool;
__umpire_arithmetic_error
VARIABLE \* @type: Bool;
__umpire_before_done
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
\* @type: <<Bool, Str, Bool, Bool, Int, Seq(Seq(Int)), Bool, Int, Bool, Seq(Int), Bool>>;
vars == <<done, __umpire_action, __umpire_arithmetic_error, __umpire_before_done, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == LET \* @type: Bool;
__umpire_s2 == (TRUE) /\ (TRUE) IN (__umpire_before_done = FALSE) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ ((IF __umpire_s2 THEN (done = FALSE) ELSE done = FALSE)) /\ (__umpire_arithmetic_error = ~(__umpire_s2 /\ (TRUE)))
\* @type: Bool;
User_A == (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s4 == (__umpire_s3) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s3) \/ (__umpire_s2)) /\ (~(__umpire_s4) \/ ((TRUE # done))) /\ (done' = IF __umpire_s4 THEN TRUE ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::A") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s4 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s4) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN (IF __umpire_action' = "User::A" THEN __umpire_p5 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((done # done))) /\ (done' = IF __umpire_s8 THEN done ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::A") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN (IF __umpire_action' = "User::A" THEN __umpire_p5 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
User_B == (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s4 == (__umpire_s3) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s3) \/ (__umpire_s2)) /\ (~(__umpire_s4) \/ ((TRUE # done))) /\ (done' = IF __umpire_s4 THEN TRUE ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::B") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s4 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s4) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN (IF __umpire_action' = "User::A" THEN __umpire_p5 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((done # done))) /\ (done' = IF __umpire_s8 THEN done ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::B") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN (IF __umpire_action' = "User::A" THEN __umpire_p5 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
ModelNext == (User_A) \/ (User_B)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<done, __umpire_action, __umpire_arithmetic_error, __umpire_before_done, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
Post_fails == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((done) = (__umpire_before_done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step /\ __umpire_action = "User::A") \/ __umpire_s4
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
