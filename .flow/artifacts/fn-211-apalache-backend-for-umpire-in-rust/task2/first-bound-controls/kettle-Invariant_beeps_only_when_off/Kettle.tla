---- MODULE Kettle ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
\* @type: Int;
BOILING_TEMPERATURE == 100
\* @type: Int;
COLDEST_TEMPERATURE == -40
VARIABLE \* @type: Int;
temperature
VARIABLE \* @type: Bool;
heating_on
VARIABLE \* @type: Bool;
beeping
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Bool;
__umpire_arithmetic_error
VARIABLE \* @type: Bool;
__umpire_before_beeping
VARIABLE \* @type: Bool;
__umpire_before_heating_on
VARIABLE \* @type: Int;
__umpire_before_temperature
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
\* @type: <<Int, Bool, Bool, Str, Bool, Bool, Bool, Int, Int, Seq(Seq(Int)), Bool, Int, Bool, Seq(Int), Bool>>;
vars == <<temperature, heating_on, beeping, __umpire_action, __umpire_arithmetic_error, __umpire_before_beeping, __umpire_before_heating_on, __umpire_before_temperature, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == LET \* @type: Bool;
__umpire_s2 == (TRUE) /\ ((TRUE) /\ (TRUE)) IN LET \* @type: Bool;
__umpire_s4 == (__umpire_s2) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s6 == (__umpire_s4) /\ (TRUE) IN (__umpire_before_temperature = 0) /\ (__umpire_before_heating_on = FALSE) /\ (__umpire_before_beeping = FALSE) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ ((IF __umpire_s2 THEN (temperature \in (-40..(40))) ELSE temperature = 0) /\ (IF __umpire_s4 THEN (heating_on = FALSE) ELSE heating_on = FALSE) /\ (IF __umpire_s6 THEN (beeping = FALSE) ELSE beeping = FALSE)) /\ (__umpire_arithmetic_error = ~(__umpire_s6 /\ (LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN (__umpire_c11))))
\* @type: Bool;
User_PressButton == (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(heating_on)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s4 == (__umpire_s3) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s3) \/ (__umpire_s2)) /\ (~(__umpire_s4) \/ ((temperature # temperature) \/ (TRUE # heating_on) \/ (FALSE # beeping))) /\ (temperature' = IF __umpire_s4 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s4 THEN TRUE ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s4 THEN FALSE ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::PressButton") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s4 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s4) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(heating_on)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping))) /\ (temperature' = IF __umpire_s8 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s8 THEN heating_on ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s8 THEN beeping ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::PressButton") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
User_DepressButton == (LET \* @type: Bool;
__umpire_s2 == (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(TRUE) \/ (heating_on)) /\ (~(__umpire_s2) \/ ((temperature # temperature) \/ (FALSE # heating_on) \/ (FALSE # beeping))) /\ (temperature' = IF __umpire_s2 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s2 THEN FALSE ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s2 THEN FALSE ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::DepressButton") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s2 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s2) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(heating_on)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) /\ (IF __umpire_s2 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s6 == (__umpire_s5) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s5) \/ (__umpire_s4)) /\ (~(__umpire_s6) \/ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping))) /\ (temperature' = IF __umpire_s6 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s6 THEN heating_on ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s6 THEN beeping ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::DepressButton") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s6 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s6) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
Heater_Heat == (LET \* @type: Bool;
__umpire_s3 == ((TRUE) /\ (TRUE)) /\ (((temperature) + (1)) >= (-2147483647 - 1) /\ ((temperature) + (1)) <= 2147483647) IN LET \* @type: Int;
__umpire_s2 == IF __umpire_s3 THEN ((temperature) + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_s5 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((temperature) < (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (TRUE) /\ (IF heating_on THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((heating_on) /\ (__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (__umpire_s3) /\ (__umpire_s3) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((__umpire_s2 # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping))) /\ (temperature' = IF __umpire_s8 THEN __umpire_s2 ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s8 THEN heating_on ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s8 THEN beeping ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "Heater::Heat") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((temperature) < (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (TRUE) /\ (IF heating_on THEN __umpire_s3 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((heating_on) /\ (__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == __umpire_s5 IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN (~(__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) /\ (IF __umpire_s6 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s10 == (__umpire_s9) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s9) \/ (__umpire_s8)) /\ (~(__umpire_s10) \/ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping))) /\ (temperature' = IF __umpire_s10 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s10 THEN heating_on ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s10 THEN beeping ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "Heater::Heat") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s10 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s10) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
Heater_Failover == (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((temperature) >= (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (TRUE) /\ (IF heating_on THEN __umpire_s3 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((heating_on) /\ (__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s6 == (__umpire_s5) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s5) \/ (__umpire_s4)) /\ (~(__umpire_s6) \/ ((temperature # temperature) \/ (FALSE # heating_on) \/ (TRUE # beeping))) /\ (temperature' = IF __umpire_s6 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s6 THEN FALSE ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s6 THEN TRUE ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "Heater::Failover") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s6 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s6) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((temperature) >= (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (TRUE) /\ (IF heating_on THEN __umpire_s3 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((heating_on) /\ (__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == __umpire_s5 IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN (~(__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) /\ (IF __umpire_s6 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s10 == (__umpire_s9) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s9) \/ (__umpire_s8)) /\ (~(__umpire_s10) \/ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping))) /\ (temperature' = IF __umpire_s10 THEN temperature ELSE temperature) /\ (__umpire_before_temperature' = temperature) /\ (heating_on' = IF __umpire_s10 THEN heating_on ELSE heating_on) /\ (__umpire_before_heating_on' = heating_on) /\ (beeping' = IF __umpire_s10 THEN beeping ELSE beeping) /\ (__umpire_before_beeping' = beeping) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "Heater::Failover") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s10 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s10) \/ ~(LET \* @type: Bool;
__umpire_c3 == TRUE IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN (~(beeping')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == TRUE IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (__umpire_c5) IN LET \* @type: Bool;
__umpire_c6 == IF __umpire_c7 THEN ((__umpire_c4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c3) /\ (IF __umpire_c2 THEN TRUE ELSE __umpire_c7) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c2) \/ (__umpire_c6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == TRUE IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN (~(__umpire_before_heating_on')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p6 == IF __umpire_p7 THEN ((temperature') <= (__umpire_before_temperature')) ELSE FALSE IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p4) \/ (__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((temperature') = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p17 == (__umpire_p15) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p16 == IF __umpire_p17 THEN ((__umpire_p14) /\ (beeping')) ELSE FALSE IN (__umpire_c11) /\ (__umpire_p13) /\ (IF __umpire_action' = "Heater::Failover" THEN __umpire_p17 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
ModelNext == (User_PressButton) \/ (User_DepressButton) \/ (Heater_Heat) \/ (Heater_Failover)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<temperature, heating_on, beeping, __umpire_action, __umpire_arithmetic_error, __umpire_before_beeping, __umpire_before_heating_on, __umpire_before_temperature, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
Invariant_beeps_only_when_off == LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(beeping)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == TRUE IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(heating_on)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s3) /\ (IF __umpire_s2 THEN TRUE ELSE __umpire_s7) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s2) \/ (__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s8)) ELSE FALSE IN __umpire_arithmetic_error \/ __umpire_s10
\* @type: Bool;
Post_warms_only_when_on == LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(__umpire_before_heating_on)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((temperature) <= (__umpire_before_temperature)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE __umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s4) \/ (__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == (__umpire_s11) IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN ((__umpire_s10)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step) \/ __umpire_s12
\* @type: Bool;
Post_fails_over_at_boiling == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((temperature) = (100)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == (__umpire_s3) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN ((__umpire_s2) /\ (beeping)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step /\ __umpire_action = "Heater::Failover") \/ __umpire_s4
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
