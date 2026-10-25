---- MODULE Inputs ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
CONSTANT \* @type: Int;
__umpire_world_limit
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
VARIABLE \* @type: Str;
__umpire_outcome
VARIABLE \* @type: Bool;
__umpire_stutter
\* @type: <<Int, Str, Bool, Int, Int, Seq(Seq(Int)), Bool, Int, Bool, Str, Bool>>;
vars == <<value, __umpire_action, __umpire_arithmetic_error, __umpire_before_value, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == (__umpire_before_value = 0) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = "Outcome::Accepted") /\ (__umpire_facts = <<>>) /\ (((value = 0)) \/ ((value = 1))) /\ (__umpire_arithmetic_error = ~(TRUE /\ (LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((value) <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((value) # (2)) ELSE FALSE IN (__umpire_c5) /\ (__umpire_c7))))
\* @type: Seq(<<Int, Int>>);
User_Set_Inputs == <<<<0, 1>>, <<0, 1>>, <<2, 2>>>>
\* @type: Int => Bool;
User_Set_At(__umpire_i) == LET \* @type: Int;
__umpire_input_0 == User_Set_Inputs[__umpire_i][2] IN (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_input_0) = (1)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s4 == (__umpire_s3) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s3) \/ (__umpire_s2)) /\ (~(__umpire_s4) \/ (TRUE)) /\ (value' = IF __umpire_s4 THEN value ELSE value) /\ (__umpire_before_value' = value) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Set") /\ (__umpire_input' = User_Set_Inputs[__umpire_i][1]) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s4 THEN "Outcome::Rejected(NotFound)" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s4) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((value') <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((value') # (2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((value') = (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == __umpire_p7 IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN (~(__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((value') # (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p9) /\ (IF __umpire_p8 THEN TRUE ELSE __umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p8) \/ (__umpire_p12)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14)) ELSE FALSE IN (__umpire_c5) /\ (__umpire_c7) /\ (IF __umpire_action' = "User::Reset" THEN __umpire_p5 ELSE TRUE) /\ (__umpire_p17)))) /\ (__umpire_invalid_step' = (__umpire_s4 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((value' = value))))) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_input_0) = (1)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_input_0) <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s5) /\ (IF __umpire_s4 THEN __umpire_s7 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s4) /\ (__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s10 == (__umpire_s9) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s9) \/ (__umpire_s8)) /\ (~(__umpire_s10) \/ ((__umpire_input_0 # value) \/ ("Outcome::Accepted" # "Outcome::Accepted"))) /\ (value' = IF __umpire_s10 THEN __umpire_input_0 ELSE value) /\ (__umpire_before_value' = value) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Set") /\ (__umpire_input' = User_Set_Inputs[__umpire_i][1]) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s10 THEN "Outcome::Accepted" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s10) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((value') <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((value') # (2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((value') = (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == __umpire_p7 IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN (~(__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((value') # (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p9) /\ (IF __umpire_p8 THEN TRUE ELSE __umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p8) \/ (__umpire_p12)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14)) ELSE FALSE IN (__umpire_c5) /\ (__umpire_c7) /\ (IF __umpire_action' = "User::Reset" THEN __umpire_p5 ELSE TRUE) /\ (__umpire_p17)))) /\ (__umpire_invalid_step' = (__umpire_s10 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((value' = value))))) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_input_0) = (1)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_input_0) <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s3) /\ (IF __umpire_s2 THEN TRUE ELSE __umpire_s5) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s2) \/ (__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == __umpire_s7 IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN (~(__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s9) /\ (IF __umpire_s8 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s8) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s12 == (__umpire_s11) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s11) \/ (__umpire_s10)) /\ (~(__umpire_s12) \/ ((value # value) \/ ("Outcome::Accepted" # "Outcome::Accepted"))) /\ (value' = IF __umpire_s12 THEN value ELSE value) /\ (__umpire_before_value' = value) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Set") /\ (__umpire_input' = User_Set_Inputs[__umpire_i][1]) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s12 THEN "Outcome::Accepted" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s12) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((value') <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((value') # (2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((value') = (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == __umpire_p7 IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN (~(__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((value') # (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p9) /\ (IF __umpire_p8 THEN TRUE ELSE __umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p8) \/ (__umpire_p12)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14)) ELSE FALSE IN (__umpire_c5) /\ (__umpire_c7) /\ (IF __umpire_action' = "User::Reset" THEN __umpire_p5 ELSE TRUE) /\ (__umpire_p17)))) /\ (__umpire_invalid_step' = (__umpire_s12 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((value' = value)))))
\* @type: Bool;
User_Set == \E __umpire_i \in DOMAIN User_Set_Inputs : User_Set_At(__umpire_i)
\* @type: Bool;
User_Reset == (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((value) > (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s4 == (__umpire_s3) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s3) \/ (__umpire_s2)) /\ (~(__umpire_s4) \/ ((0 # value) \/ ("Outcome::Accepted" # "Outcome::Accepted"))) /\ (value' = IF __umpire_s4 THEN 0 ELSE value) /\ (__umpire_before_value' = value) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Reset") /\ (__umpire_input' = 3) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s4 THEN "Outcome::Accepted" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s4) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((value') <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((value') # (2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((value') = (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == __umpire_p7 IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN (~(__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((value') # (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p9) /\ (IF __umpire_p8 THEN TRUE ELSE __umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p8) \/ (__umpire_p12)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14)) ELSE FALSE IN (__umpire_c5) /\ (__umpire_c7) /\ (IF __umpire_action' = "User::Reset" THEN __umpire_p5 ELSE TRUE) /\ (__umpire_p17)))) /\ (__umpire_invalid_step' = (__umpire_s4 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((value' = value))))) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((value) > (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((value # value) \/ ("Outcome::Accepted" # "Outcome::Accepted"))) /\ (value' = IF __umpire_s8 THEN value ELSE value) /\ (__umpire_before_value' = value) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Reset") /\ (__umpire_input' = 3) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN "Outcome::Accepted" ELSE "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((value') <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == (__umpire_c3) IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN ((__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((value') # (2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((value') = (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == (__umpire_p3) IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN ((__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((__umpire_outcome') = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == __umpire_p7 IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN (~(__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((value') # (__umpire_before_value')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p9) /\ (IF __umpire_p8 THEN TRUE ELSE __umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p8) \/ (__umpire_p12)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14)) ELSE FALSE IN (__umpire_c5) /\ (__umpire_c7) /\ (IF __umpire_action' = "User::Reset" THEN __umpire_p5 ELSE TRUE) /\ (__umpire_p17)))) /\ (__umpire_invalid_step' = (__umpire_s8 /\ __umpire_outcome' # "Outcome::Accepted" /\ ~((value' = value)))))
\* @type: Bool;
ModelNext == (User_Set) \/ (User_Reset)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<value, __umpire_action, __umpire_arithmetic_error, __umpire_before_value, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
Invariant_bounded == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((value) <= (__umpire_world_limit)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2)) ELSE FALSE IN __umpire_arithmetic_error \/ __umpire_s4
\* @type: Bool;
Negative_never_two == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((value) # (2)) ELSE FALSE IN __umpire_arithmetic_error \/ __umpire_s2
\* @type: Bool;
Post_no_reset == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((value) = (__umpire_before_value)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step /\ __umpire_action = "User::Reset") \/ __umpire_s4
\* @type: Bool;
Post_no_rejection == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_outcome) = ("Outcome::Rejected(NotFound)")) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((value) # (__umpire_before_value)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE __umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s4) \/ (__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == (__umpire_s11) IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN ((__umpire_s10)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step) \/ __umpire_s12
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
