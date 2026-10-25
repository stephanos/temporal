---- MODULE Inputs ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
CONSTANT \* @type: Int;
__umpire_world_limit
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
VARIABLE \* @type: Str;
__umpire_outcome
VARIABLE \* @type: Bool;
__umpire_stutter
\* @type: <<Bool, Str, Bool, Bool, Int, Seq(Seq(Int)), Bool, Int, Bool, Str, Bool>>;
vars == <<done, __umpire_action, __umpire_arithmetic_error, __umpire_before_done, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == (__umpire_before_done = FALSE) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = "Outcome::Accepted") /\ (__umpire_facts = <<>>) /\ (((done = FALSE))) /\ (__umpire_arithmetic_error = ~(TRUE /\ (TRUE)))
\* @type: Seq(<<Int, Bool>>);
User_Set_Inputs == <<<<0, FALSE>>, <<1, TRUE>>>>
\* @type: Int => Bool;
User_Set_At(__umpire_i) == LET \* @type: Bool;
__umpire_input_0 == User_Set_Inputs[__umpire_i][2] IN (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_input_0) = (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == TRUE IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s3) /\ (IF __umpire_s2 THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s2) /\ (__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((TRUE # done) \/ ("Outcome::Accepted" # "Outcome::Accepted"))) /\ (done' = IF __umpire_s8 THEN TRUE ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Set") /\ (__umpire_input' = User_Set_Inputs[__umpire_i][1]) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN "Outcome::Accepted" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == __umpire_p5 IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN (~(__umpire_p4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p13) /\ (IF __umpire_p12 THEN TRUE ELSE __umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p12) \/ (__umpire_p14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p19 == (__umpire_p11) /\ (IF __umpire_p10 THEN __umpire_p17 ELSE TRUE) IN LET \* @type: Bool;
__umpire_p18 == IF __umpire_p19 THEN ((__umpire_p10) /\ (__umpire_p16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p21 == (__umpire_p19) IN LET \* @type: Bool;
__umpire_p20 == IF __umpire_p21 THEN ((__umpire_p18)) ELSE FALSE IN (__umpire_p21)))) /\ (__umpire_invalid_step' = (__umpire_s8 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((done' = done))))) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_input_0) = (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == TRUE IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s3) /\ (IF __umpire_s2 THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s2) /\ (__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == __umpire_s7 IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN (~(__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_input_0) = (FALSE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == (__umpire_s9) /\ (IF __umpire_s8 THEN __umpire_s11 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN ((__umpire_s8) /\ (__umpire_s10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s14 == (__umpire_s13) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s13) \/ (__umpire_s12)) /\ (~(__umpire_s14) \/ (TRUE)) /\ (done' = IF __umpire_s14 THEN done ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Set") /\ (__umpire_input' = User_Set_Inputs[__umpire_i][1]) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s14 THEN "Outcome::Rejected(NotFound)" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s14) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == __umpire_p5 IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN (~(__umpire_p4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p13) /\ (IF __umpire_p12 THEN TRUE ELSE __umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p12) \/ (__umpire_p14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p19 == (__umpire_p11) /\ (IF __umpire_p10 THEN __umpire_p17 ELSE TRUE) IN LET \* @type: Bool;
__umpire_p18 == IF __umpire_p19 THEN ((__umpire_p10) /\ (__umpire_p16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p21 == (__umpire_p19) IN LET \* @type: Bool;
__umpire_p20 == IF __umpire_p21 THEN ((__umpire_p18)) ELSE FALSE IN (__umpire_p21)))) /\ (__umpire_invalid_step' = (__umpire_s14 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((done' = done))))) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_input_0) = (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == TRUE IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s3) /\ (IF __umpire_s2 THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s2) /\ (__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_input_0) = (FALSE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s7) /\ (IF __umpire_s6 THEN TRUE ELSE __umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s6) \/ (__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == __umpire_s11 IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN (~(__umpire_s10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s15 == (__umpire_s13) /\ (IF __umpire_s12 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s14 == IF __umpire_s15 THEN ((__umpire_s12) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s16 == (__umpire_s15) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s15) \/ (__umpire_s14)) /\ (~(__umpire_s16) \/ ((done # done) \/ ("Outcome::Accepted" # "Outcome::Accepted"))) /\ (done' = IF __umpire_s16 THEN done ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Set") /\ (__umpire_input' = User_Set_Inputs[__umpire_i][1]) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s16 THEN "Outcome::Accepted" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s16) \/ ~(LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((done') = (__umpire_before_done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == __umpire_p5 IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN (~(__umpire_p4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((done')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p13) /\ (IF __umpire_p12 THEN TRUE ELSE __umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p12) \/ (__umpire_p14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p19 == (__umpire_p11) /\ (IF __umpire_p10 THEN __umpire_p17 ELSE TRUE) IN LET \* @type: Bool;
__umpire_p18 == IF __umpire_p19 THEN ((__umpire_p10) /\ (__umpire_p16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p21 == (__umpire_p19) IN LET \* @type: Bool;
__umpire_p20 == IF __umpire_p21 THEN ((__umpire_p18)) ELSE FALSE IN (__umpire_p21)))) /\ (__umpire_invalid_step' = (__umpire_s16 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((done' = done)))))
\* @type: Bool;
User_Set == \E __umpire_i \in DOMAIN User_Set_Inputs : User_Set_At(__umpire_i)
\* @type: Bool;
ModelNext == (User_Set)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<done, __umpire_action, __umpire_arithmetic_error, __umpire_before_done, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
Post_answer == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_outcome) = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((done) = (__umpire_before_done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE __umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s4) \/ (__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == __umpire_s5 IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN (~(__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s15 == (TRUE) IN LET \* @type: Bool;
__umpire_s14 == IF __umpire_s15 THEN ((done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s17 == (__umpire_s13) /\ (IF __umpire_s12 THEN TRUE ELSE __umpire_s15) IN LET \* @type: Bool;
__umpire_s16 == IF __umpire_s17 THEN ((__umpire_s12) \/ (__umpire_s14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s19 == (__umpire_s11) /\ (IF __umpire_s10 THEN __umpire_s17 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s18 == IF __umpire_s19 THEN ((__umpire_s10) /\ (__umpire_s16)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s21 == (__umpire_s19) IN LET \* @type: Bool;
__umpire_s20 == IF __umpire_s21 THEN ((__umpire_s18)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step) \/ __umpire_s20
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
