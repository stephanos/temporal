---- MODULE Arithmetic ----
EXTENDS Integers, Sequences
\* @type: (Int, Int) => Int;
__umpire_SafeDiv(__umpire_a, __umpire_b) == IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)
CONSTANT \* @type: Int;
__umpire_world_mode
CONSTANT \* @type: Int;
__umpire_world_start
CONSTANT \* @type: Int;
__umpire_world_divisor
VARIABLE \* @type: Int;
value
VARIABLE \* @type: Int;
divisor
VARIABLE \* @type: Bool;
done
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Bool;
__umpire_arithmetic_error
VARIABLE \* @type: Int;
__umpire_before_divisor
VARIABLE \* @type: Bool;
__umpire_before_done
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
\* @type: <<Int, Int, Bool, Str, Bool, Int, Bool, Int, Int, Seq(Seq(Int)), Bool, Int, Bool, Seq(Int), Bool>>;
vars == <<value, divisor, done, __umpire_action, __umpire_arithmetic_error, __umpire_before_divisor, __umpire_before_done, __umpire_before_value, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome, __umpire_stutter>>
\* @type: Bool;
Init == (__umpire_before_value = 0) /\ (__umpire_before_divisor = 0) /\ (__umpire_before_done = FALSE) /\ (__umpire_has_step = FALSE) /\ (__umpire_stutter = TRUE) /\ (__umpire_invalid_step = FALSE) /\ (__umpire_action = "") /\ (__umpire_input = 0) /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ (((value = -128) /\ (divisor = -1) /\ (done = FALSE))) /\ (__umpire_arithmetic_error = ~(TRUE /\ (LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((__umpire_world_mode) = (8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == __umpire_c3 IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (((TRUE) /\ (TRUE)) /\ (divisor # 0) /\ (~(1 = -128 /\ divisor = -1))) /\ ((__umpire_SafeDiv(1, divisor)) >= -128 /\ (__umpire_SafeDiv(1, divisor)) <= 127) IN LET \* @type: Int;
__umpire_c6 == IF __umpire_c7 THEN (__umpire_SafeDiv(1, divisor)) ELSE 0 IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (TRUE) /\ (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((FALSE) /\ (__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c13 == (__umpire_c5) /\ (IF __umpire_c4 THEN TRUE ELSE __umpire_c11) IN LET \* @type: Bool;
__umpire_c12 == IF __umpire_c13 THEN ((__umpire_c4) \/ (__umpire_c10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c14 == IF __umpire_c15 THEN ((__umpire_world_mode) = (10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c17 == __umpire_c15 IN LET \* @type: Bool;
__umpire_c16 == IF __umpire_c17 THEN (~(__umpire_c14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c19 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c18 == IF __umpire_c19 THEN ((divisor) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c21 == (TRUE) /\ (IF done THEN TRUE ELSE __umpire_c19) IN LET \* @type: Bool;
__umpire_c20 == IF __umpire_c21 THEN ((done) \/ (__umpire_c18)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c23 == (__umpire_c21) /\ (IF __umpire_c20 THEN TRUE ELSE __umpire_c9) IN LET \* @type: Bool;
__umpire_c22 == IF __umpire_c23 THEN ((__umpire_c20) \/ (__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c25 == (__umpire_c23) IN LET \* @type: Bool;
__umpire_c24 == IF __umpire_c25 THEN ((__umpire_c22)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c27 == (__umpire_c17) /\ (IF __umpire_c16 THEN TRUE ELSE __umpire_c25) IN LET \* @type: Bool;
__umpire_c26 == IF __umpire_c27 THEN ((__umpire_c16) \/ (__umpire_c24)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c29 == (__umpire_c13) /\ (__umpire_c27) IN LET \* @type: Bool;
__umpire_c28 == IF __umpire_c29 THEN ((__umpire_c12) /\ (__umpire_c26)) ELSE FALSE IN (__umpire_c29))))
\* @type: Bool;
User_Calculate == (LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_world_mode) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == ((TRUE) /\ (TRUE)) /\ (((value) + (1)) >= -128 /\ ((value) + (1)) <= 127) IN LET \* @type: Int;
__umpire_s4 == IF __umpire_s5 THEN ((value) + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s3) /\ (IF __umpire_s2 THEN TRUE ELSE TRUE) IN LET \* @type: Int;
__umpire_s6 == IF __umpire_s7 THEN (IF __umpire_s2 THEN divisor ELSE divisor) ELSE 0 IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s3) /\ (IF __umpire_s2 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN (IF __umpire_s2 THEN done ELSE done) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s3) /\ (IF __umpire_s2 THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Int;
__umpire_s10 == IF __umpire_s11 THEN (IF __umpire_s2 THEN __umpire_s4 ELSE value) ELSE 0 IN LET \* @type: Bool;
__umpire_s13 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN ((__umpire_world_mode) = (1)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s15 == ((__umpire_s11) /\ (TRUE)) /\ (((__umpire_s10) + (1)) >= -128 /\ ((__umpire_s10) + (1)) <= 127) IN LET \* @type: Int;
__umpire_s14 == IF __umpire_s15 THEN ((__umpire_s10) + (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_s17 == ((__umpire_s15) /\ (TRUE)) /\ (((__umpire_s14) - (1)) >= -128 /\ ((__umpire_s14) - (1)) <= 127) IN LET \* @type: Int;
__umpire_s16 == IF __umpire_s17 THEN ((__umpire_s14) - (1)) ELSE 0 IN LET \* @type: Bool;
__umpire_s19 == (__umpire_s13) /\ (IF __umpire_s12 THEN __umpire_s7 ELSE __umpire_s7) IN LET \* @type: Int;
__umpire_s18 == IF __umpire_s19 THEN (IF __umpire_s12 THEN __umpire_s6 ELSE __umpire_s6) ELSE 0 IN LET \* @type: Bool;
__umpire_s21 == (__umpire_s13) /\ (IF __umpire_s12 THEN __umpire_s9 ELSE __umpire_s9) IN LET \* @type: Bool;
__umpire_s20 == IF __umpire_s21 THEN (IF __umpire_s12 THEN __umpire_s8 ELSE __umpire_s8) ELSE FALSE IN LET \* @type: Bool;
__umpire_s23 == (__umpire_s13) /\ (IF __umpire_s12 THEN __umpire_s17 ELSE __umpire_s11) IN LET \* @type: Int;
__umpire_s22 == IF __umpire_s23 THEN (IF __umpire_s12 THEN __umpire_s16 ELSE __umpire_s10) ELSE 0 IN LET \* @type: Bool;
__umpire_s25 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s24 == IF __umpire_s25 THEN ((__umpire_world_mode) = (2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s27 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s26 == IF __umpire_s27 THEN ((__umpire_world_mode) = (4)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s29 == (__umpire_s25) /\ (IF __umpire_s24 THEN TRUE ELSE __umpire_s27) IN LET \* @type: Bool;
__umpire_s28 == IF __umpire_s29 THEN ((__umpire_s24) \/ (__umpire_s26)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s31 == (((__umpire_s23) /\ (__umpire_s19)) /\ (__umpire_s18 # 0) /\ (~(__umpire_s22 = -128 /\ __umpire_s18 = -1))) /\ ((__umpire_SafeDiv(__umpire_s22, __umpire_s18)) >= -128 /\ (__umpire_SafeDiv(__umpire_s22, __umpire_s18)) <= 127) IN LET \* @type: Int;
__umpire_s30 == IF __umpire_s31 THEN (__umpire_SafeDiv(__umpire_s22, __umpire_s18)) ELSE 0 IN LET \* @type: Bool;
__umpire_s33 == (__umpire_s29) /\ (IF __umpire_s28 THEN __umpire_s19 ELSE __umpire_s19) IN LET \* @type: Int;
__umpire_s32 == IF __umpire_s33 THEN (IF __umpire_s28 THEN __umpire_s18 ELSE __umpire_s18) ELSE 0 IN LET \* @type: Bool;
__umpire_s35 == (__umpire_s29) /\ (IF __umpire_s28 THEN __umpire_s21 ELSE __umpire_s21) IN LET \* @type: Bool;
__umpire_s34 == IF __umpire_s35 THEN (IF __umpire_s28 THEN __umpire_s20 ELSE __umpire_s20) ELSE FALSE IN LET \* @type: Bool;
__umpire_s37 == (__umpire_s29) /\ (IF __umpire_s28 THEN __umpire_s31 ELSE __umpire_s23) IN LET \* @type: Int;
__umpire_s36 == IF __umpire_s37 THEN (IF __umpire_s28 THEN __umpire_s30 ELSE __umpire_s22) ELSE 0 IN LET \* @type: Bool;
__umpire_s39 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s38 == IF __umpire_s39 THEN ((__umpire_world_mode) = (3)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s41 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s40 == IF __umpire_s41 THEN ((__umpire_world_mode) = (5)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s43 == (__umpire_s39) /\ (IF __umpire_s38 THEN TRUE ELSE __umpire_s41) IN LET \* @type: Bool;
__umpire_s42 == IF __umpire_s43 THEN ((__umpire_s38) \/ (__umpire_s40)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s45 == (((__umpire_s37) /\ (__umpire_s33)) /\ (__umpire_s32 # 0) /\ (~(__umpire_s36 = -128 /\ __umpire_s32 = -1))) /\ ((__umpire_s36 - (__umpire_SafeDiv(__umpire_s36, __umpire_s32)) * __umpire_s32) >= -128 /\ (__umpire_s36 - (__umpire_SafeDiv(__umpire_s36, __umpire_s32)) * __umpire_s32) <= 127) IN LET \* @type: Int;
__umpire_s44 == IF __umpire_s45 THEN (__umpire_s36 - (__umpire_SafeDiv(__umpire_s36, __umpire_s32)) * __umpire_s32) ELSE 0 IN LET \* @type: Bool;
__umpire_s47 == (__umpire_s43) /\ (IF __umpire_s42 THEN __umpire_s33 ELSE __umpire_s33) IN LET \* @type: Int;
__umpire_s46 == IF __umpire_s47 THEN (IF __umpire_s42 THEN __umpire_s32 ELSE __umpire_s32) ELSE 0 IN LET \* @type: Bool;
__umpire_s49 == (__umpire_s43) /\ (IF __umpire_s42 THEN __umpire_s35 ELSE __umpire_s35) IN LET \* @type: Bool;
__umpire_s48 == IF __umpire_s49 THEN (IF __umpire_s42 THEN __umpire_s34 ELSE __umpire_s34) ELSE FALSE IN LET \* @type: Bool;
__umpire_s51 == (__umpire_s43) /\ (IF __umpire_s42 THEN __umpire_s45 ELSE __umpire_s37) IN LET \* @type: Int;
__umpire_s50 == IF __umpire_s51 THEN (IF __umpire_s42 THEN __umpire_s44 ELSE __umpire_s36) ELSE 0 IN LET \* @type: Bool;
__umpire_s53 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s52 == IF __umpire_s53 THEN ((__umpire_world_mode) = (6)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s55 == (((__umpire_s51) /\ (__umpire_s47)) /\ (__umpire_s46 # 0) /\ (~(__umpire_s50 = -128 /\ __umpire_s46 = -1))) /\ ((__umpire_SafeDiv(__umpire_s50, __umpire_s46)) >= -128 /\ (__umpire_SafeDiv(__umpire_s50, __umpire_s46)) <= 127) IN LET \* @type: Int;
__umpire_s54 == IF __umpire_s55 THEN (__umpire_SafeDiv(__umpire_s50, __umpire_s46)) ELSE 0 IN LET \* @type: Bool;
__umpire_s57 == (__umpire_s49) /\ (IF __umpire_s48 THEN __umpire_s55 ELSE __umpire_s51) IN LET \* @type: Int;
__umpire_s56 == IF __umpire_s57 THEN (IF __umpire_s48 THEN __umpire_s54 ELSE __umpire_s50) ELSE 0 IN LET \* @type: Bool;
__umpire_s59 == (__umpire_s53) /\ (IF __umpire_s52 THEN __umpire_s47 ELSE __umpire_s47) IN LET \* @type: Int;
__umpire_s58 == IF __umpire_s59 THEN (IF __umpire_s52 THEN __umpire_s46 ELSE __umpire_s46) ELSE 0 IN LET \* @type: Bool;
__umpire_s61 == (__umpire_s53) /\ (IF __umpire_s52 THEN __umpire_s49 ELSE __umpire_s49) IN LET \* @type: Bool;
__umpire_s60 == IF __umpire_s61 THEN (IF __umpire_s52 THEN __umpire_s48 ELSE __umpire_s48) ELSE FALSE IN LET \* @type: Bool;
__umpire_s63 == (__umpire_s53) /\ (IF __umpire_s52 THEN __umpire_s57 ELSE __umpire_s51) IN LET \* @type: Int;
__umpire_s62 == IF __umpire_s63 THEN (IF __umpire_s52 THEN __umpire_s56 ELSE __umpire_s50) ELSE 0 IN LET \* @type: Bool;
__umpire_s65 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s64 == IF __umpire_s65 THEN ((__umpire_world_mode) = (7)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s67 == (__umpire_s65) /\ (IF __umpire_s64 THEN __umpire_s61 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s66 == IF __umpire_s67 THEN ((__umpire_s64) /\ (__umpire_s60)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s69 == (((__umpire_s63) /\ (__umpire_s59)) /\ (__umpire_s58 # 0) /\ (~(__umpire_s62 = -128 /\ __umpire_s58 = -1))) /\ ((__umpire_SafeDiv(__umpire_s62, __umpire_s58)) >= -128 /\ (__umpire_SafeDiv(__umpire_s62, __umpire_s58)) <= 127) IN LET \* @type: Int;
__umpire_s68 == IF __umpire_s69 THEN (__umpire_SafeDiv(__umpire_s62, __umpire_s58)) ELSE 0 IN LET \* @type: Bool;
__umpire_s71 == (__umpire_s69) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s70 == IF __umpire_s71 THEN ((__umpire_s68) > (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s73 == (__umpire_s67) /\ (IF __umpire_s66 THEN __umpire_s71 ELSE TRUE) IN LET \* @type: Bool;
__umpire_s72 == IF __umpire_s73 THEN ((__umpire_s66) /\ (__umpire_s70)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s75 == (__umpire_s73) /\ (IF __umpire_s72 THEN __umpire_s59 ELSE __umpire_s59) IN LET \* @type: Int;
__umpire_s74 == IF __umpire_s75 THEN (IF __umpire_s72 THEN __umpire_s58 ELSE __umpire_s58) ELSE 0 IN LET \* @type: Bool;
__umpire_s77 == (__umpire_s73) /\ (IF __umpire_s72 THEN __umpire_s61 ELSE __umpire_s61) IN LET \* @type: Bool;
__umpire_s76 == IF __umpire_s77 THEN (IF __umpire_s72 THEN __umpire_s60 ELSE __umpire_s60) ELSE FALSE IN LET \* @type: Bool;
__umpire_s79 == (__umpire_s73) /\ (IF __umpire_s72 THEN TRUE ELSE __umpire_s63) IN LET \* @type: Int;
__umpire_s78 == IF __umpire_s79 THEN (IF __umpire_s72 THEN 0 ELSE __umpire_s62) ELSE 0 IN LET \* @type: Bool;
__umpire_s81 == TRUE IN LET \* @type: Bool;
__umpire_s80 == IF __umpire_s81 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s83 == (__umpire_s3) /\ (IF __umpire_s2 THEN __umpire_s5 ELSE TRUE) IN LET \* @type: Int;
__umpire_s82 == IF __umpire_s83 THEN (IF __umpire_s2 THEN __umpire_s4 ELSE 0) ELSE 0 IN LET \* @type: Bool;
__umpire_s85 == (__umpire_s13) /\ (IF __umpire_s12 THEN __umpire_s17 ELSE TRUE) IN LET \* @type: Int;
__umpire_s84 == IF __umpire_s85 THEN (IF __umpire_s12 THEN __umpire_s16 ELSE 0) ELSE 0 IN LET \* @type: Bool;
__umpire_s87 == (__umpire_s29) /\ (IF __umpire_s28 THEN __umpire_s31 ELSE TRUE) IN LET \* @type: Int;
__umpire_s86 == IF __umpire_s87 THEN (IF __umpire_s28 THEN __umpire_s30 ELSE 0) ELSE 0 IN LET \* @type: Bool;
__umpire_s89 == (__umpire_s43) /\ (IF __umpire_s42 THEN __umpire_s45 ELSE TRUE) IN LET \* @type: Int;
__umpire_s88 == IF __umpire_s89 THEN (IF __umpire_s42 THEN __umpire_s44 ELSE 0) ELSE 0 IN LET \* @type: Bool;
__umpire_s91 == (__umpire_s53) /\ (IF __umpire_s52 THEN __umpire_s57 ELSE TRUE) IN LET \* @type: Int;
__umpire_s90 == IF __umpire_s91 THEN (IF __umpire_s52 THEN __umpire_s56 ELSE 0) ELSE 0 IN LET \* @type: Bool;
__umpire_s93 == (__umpire_s73) /\ (IF __umpire_s72 THEN TRUE ELSE TRUE) IN LET \* @type: Int;
__umpire_s92 == IF __umpire_s93 THEN (IF __umpire_s72 THEN 0 ELSE 0) ELSE 0 IN LET \* @type: Bool;
__umpire_s94 == (__umpire_s81) /\ (__umpire_s3) /\ (__umpire_s83) /\ (__umpire_s13) /\ (__umpire_s85) /\ (__umpire_s29) /\ (__umpire_s87) /\ (__umpire_s43) /\ (__umpire_s89) /\ (__umpire_s53) /\ (__umpire_s91) /\ (__umpire_s73) /\ (__umpire_s93) /\ (TRUE) /\ (__umpire_s79) /\ (__umpire_s75) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s81) \/ (__umpire_s80)) /\ (~(__umpire_s94) \/ ((__umpire_s78 # value) \/ (__umpire_s74 # divisor) \/ (TRUE # done))) /\ (value' = IF __umpire_s94 THEN __umpire_s78 ELSE value) /\ (__umpire_before_value' = value) /\ (divisor' = IF __umpire_s94 THEN __umpire_s74 ELSE divisor) /\ (__umpire_before_divisor' = divisor) /\ (done' = IF __umpire_s94 THEN TRUE ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Calculate") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s94 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s94) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((__umpire_world_mode) = (8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == __umpire_c3 IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (((TRUE) /\ (TRUE)) /\ (divisor' # 0) /\ (~(1 = -128 /\ divisor' = -1))) /\ ((__umpire_SafeDiv(1, divisor')) >= -128 /\ (__umpire_SafeDiv(1, divisor')) <= 127) IN LET \* @type: Int;
__umpire_c6 == IF __umpire_c7 THEN (__umpire_SafeDiv(1, divisor')) ELSE 0 IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (TRUE) /\ (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((FALSE) /\ (__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c13 == (__umpire_c5) /\ (IF __umpire_c4 THEN TRUE ELSE __umpire_c11) IN LET \* @type: Bool;
__umpire_c12 == IF __umpire_c13 THEN ((__umpire_c4) \/ (__umpire_c10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c14 == IF __umpire_c15 THEN ((__umpire_world_mode) = (10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c17 == __umpire_c15 IN LET \* @type: Bool;
__umpire_c16 == IF __umpire_c17 THEN (~(__umpire_c14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c19 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c18 == IF __umpire_c19 THEN ((divisor') = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c21 == (TRUE) /\ (IF done' THEN TRUE ELSE __umpire_c19) IN LET \* @type: Bool;
__umpire_c20 == IF __umpire_c21 THEN ((done') \/ (__umpire_c18)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c23 == (__umpire_c21) /\ (IF __umpire_c20 THEN TRUE ELSE __umpire_c9) IN LET \* @type: Bool;
__umpire_c22 == IF __umpire_c23 THEN ((__umpire_c20) \/ (__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c25 == (__umpire_c23) IN LET \* @type: Bool;
__umpire_c24 == IF __umpire_c25 THEN ((__umpire_c22)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c27 == (__umpire_c17) /\ (IF __umpire_c16 THEN TRUE ELSE __umpire_c25) IN LET \* @type: Bool;
__umpire_c26 == IF __umpire_c27 THEN ((__umpire_c16) \/ (__umpire_c24)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c29 == (__umpire_c13) /\ (__umpire_c27) IN LET \* @type: Bool;
__umpire_c28 == IF __umpire_c29 THEN ((__umpire_c12) /\ (__umpire_c26)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((__umpire_world_mode) = (9)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (((TRUE) /\ (TRUE)) /\ (divisor' # 0) /\ (~(value' = -128 /\ divisor' = -1))) /\ ((__umpire_SafeDiv(value', divisor')) >= -128 /\ (__umpire_SafeDiv(value', divisor')) <= 127) IN LET \* @type: Int;
__umpire_p6 == IF __umpire_p7 THEN (__umpire_SafeDiv(value', divisor')) ELSE 0 IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p4) \/ (__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p12)) ELSE FALSE IN (__umpire_c29) /\ (IF __umpire_action' = "User::Calculate" THEN __umpire_p15 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE)) \/ (LET \* @type: Bool;
__umpire_s3 == TRUE IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN (~(done)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE TRUE) IN LET \* @type: Bool;
__umpire_s6 == IF __umpire_s7 THEN ((__umpire_s4) /\ (TRUE)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s8 == (__umpire_s7) /\ (TRUE) /\ (TRUE) /\ (TRUE) /\ (TRUE) IN (~__umpire_arithmetic_error /\ ~__umpire_invalid_step) /\ (~(__umpire_s7) \/ (__umpire_s6)) /\ (~(__umpire_s8) \/ ((value # value) \/ (divisor # divisor) \/ (done # done))) /\ (value' = IF __umpire_s8 THEN value ELSE value) /\ (__umpire_before_value' = value) /\ (divisor' = IF __umpire_s8 THEN divisor ELSE divisor) /\ (__umpire_before_divisor' = divisor) /\ (done' = IF __umpire_s8 THEN done ELSE done) /\ (__umpire_before_done' = done) /\ (__umpire_has_step' = TRUE) /\ (__umpire_stutter' = FALSE) /\ (__umpire_action' = "User::Calculate") /\ (__umpire_input' = 0) /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = IF __umpire_s8 THEN <<>> ELSE <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_arithmetic_error' = (~(__umpire_s8) \/ ~(LET \* @type: Bool;
__umpire_c3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c2 == IF __umpire_c3 THEN ((__umpire_world_mode) = (8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c5 == __umpire_c3 IN LET \* @type: Bool;
__umpire_c4 == IF __umpire_c5 THEN (~(__umpire_c2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c7 == (((TRUE) /\ (TRUE)) /\ (divisor' # 0) /\ (~(1 = -128 /\ divisor' = -1))) /\ ((__umpire_SafeDiv(1, divisor')) >= -128 /\ (__umpire_SafeDiv(1, divisor')) <= 127) IN LET \* @type: Int;
__umpire_c6 == IF __umpire_c7 THEN (__umpire_SafeDiv(1, divisor')) ELSE 0 IN LET \* @type: Bool;
__umpire_c9 == (__umpire_c7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c8 == IF __umpire_c9 THEN ((__umpire_c6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c11 == (TRUE) /\ (__umpire_c9) IN LET \* @type: Bool;
__umpire_c10 == IF __umpire_c11 THEN ((FALSE) /\ (__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c13 == (__umpire_c5) /\ (IF __umpire_c4 THEN TRUE ELSE __umpire_c11) IN LET \* @type: Bool;
__umpire_c12 == IF __umpire_c13 THEN ((__umpire_c4) \/ (__umpire_c10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c14 == IF __umpire_c15 THEN ((__umpire_world_mode) = (10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c17 == __umpire_c15 IN LET \* @type: Bool;
__umpire_c16 == IF __umpire_c17 THEN (~(__umpire_c14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c19 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_c18 == IF __umpire_c19 THEN ((divisor') = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c21 == (TRUE) /\ (IF done' THEN TRUE ELSE __umpire_c19) IN LET \* @type: Bool;
__umpire_c20 == IF __umpire_c21 THEN ((done') \/ (__umpire_c18)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c23 == (__umpire_c21) /\ (IF __umpire_c20 THEN TRUE ELSE __umpire_c9) IN LET \* @type: Bool;
__umpire_c22 == IF __umpire_c23 THEN ((__umpire_c20) \/ (__umpire_c8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c25 == (__umpire_c23) IN LET \* @type: Bool;
__umpire_c24 == IF __umpire_c25 THEN ((__umpire_c22)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c27 == (__umpire_c17) /\ (IF __umpire_c16 THEN TRUE ELSE __umpire_c25) IN LET \* @type: Bool;
__umpire_c26 == IF __umpire_c27 THEN ((__umpire_c16) \/ (__umpire_c24)) ELSE FALSE IN LET \* @type: Bool;
__umpire_c29 == (__umpire_c13) /\ (__umpire_c27) IN LET \* @type: Bool;
__umpire_c28 == IF __umpire_c29 THEN ((__umpire_c12) /\ (__umpire_c26)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p2 == IF __umpire_p3 THEN ((__umpire_world_mode) = (9)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p5 == __umpire_p3 IN LET \* @type: Bool;
__umpire_p4 == IF __umpire_p5 THEN (~(__umpire_p2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p7 == (((TRUE) /\ (TRUE)) /\ (divisor' # 0) /\ (~(value' = -128 /\ divisor' = -1))) /\ ((__umpire_SafeDiv(value', divisor')) >= -128 /\ (__umpire_SafeDiv(value', divisor')) <= 127) IN LET \* @type: Int;
__umpire_p6 == IF __umpire_p7 THEN (__umpire_SafeDiv(value', divisor')) ELSE 0 IN LET \* @type: Bool;
__umpire_p9 == (__umpire_p7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_p8 == IF __umpire_p9 THEN ((__umpire_p6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p11 == (__umpire_p9) IN LET \* @type: Bool;
__umpire_p10 == IF __umpire_p11 THEN ((__umpire_p8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p13 == (__umpire_p5) /\ (IF __umpire_p4 THEN TRUE ELSE __umpire_p11) IN LET \* @type: Bool;
__umpire_p12 == IF __umpire_p13 THEN ((__umpire_p4) \/ (__umpire_p10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_p15 == (__umpire_p13) IN LET \* @type: Bool;
__umpire_p14 == IF __umpire_p15 THEN ((__umpire_p12)) ELSE FALSE IN (__umpire_c29) /\ (IF __umpire_action' = "User::Calculate" THEN __umpire_p15 ELSE TRUE)))) /\ (__umpire_invalid_step' = FALSE))
\* @type: Bool;
ModelNext == (User_Calculate)
\* @type: Bool;
Stutter == __umpire_stutter' = TRUE /\ UNCHANGED <<value, divisor, done, __umpire_action, __umpire_arithmetic_error, __umpire_before_divisor, __umpire_before_done, __umpire_before_value, __umpire_choice, __umpire_facts, __umpire_has_step, __umpire_input, __umpire_invalid_step, __umpire_outcome>>
\* @type: Bool;
Next == ModelNext \/ Stutter
\* @type: Bool;
Invariant_claims == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_world_mode) = (8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (((TRUE) /\ (TRUE)) /\ (divisor # 0) /\ (~(1 = -128 /\ divisor = -1))) /\ ((__umpire_SafeDiv(1, divisor)) >= -128 /\ (__umpire_SafeDiv(1, divisor)) <= 127) IN LET \* @type: Int;
__umpire_s6 == IF __umpire_s7 THEN (__umpire_SafeDiv(1, divisor)) ELSE 0 IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (TRUE) /\ (__umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((FALSE) /\ (__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE __umpire_s11) IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN ((__umpire_s4) \/ (__umpire_s10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s15 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s14 == IF __umpire_s15 THEN ((__umpire_world_mode) = (10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s17 == __umpire_s15 IN LET \* @type: Bool;
__umpire_s16 == IF __umpire_s17 THEN (~(__umpire_s14)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s19 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s18 == IF __umpire_s19 THEN ((divisor) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s21 == (TRUE) /\ (IF done THEN TRUE ELSE __umpire_s19) IN LET \* @type: Bool;
__umpire_s20 == IF __umpire_s21 THEN ((done) \/ (__umpire_s18)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s23 == (__umpire_s21) /\ (IF __umpire_s20 THEN TRUE ELSE __umpire_s9) IN LET \* @type: Bool;
__umpire_s22 == IF __umpire_s23 THEN ((__umpire_s20) \/ (__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s25 == (__umpire_s23) IN LET \* @type: Bool;
__umpire_s24 == IF __umpire_s25 THEN ((__umpire_s22)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s27 == (__umpire_s17) /\ (IF __umpire_s16 THEN TRUE ELSE __umpire_s25) IN LET \* @type: Bool;
__umpire_s26 == IF __umpire_s27 THEN ((__umpire_s16) \/ (__umpire_s24)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s29 == (__umpire_s13) /\ (__umpire_s27) IN LET \* @type: Bool;
__umpire_s28 == IF __umpire_s29 THEN ((__umpire_s12) /\ (__umpire_s26)) ELSE FALSE IN __umpire_arithmetic_error \/ __umpire_s28
\* @type: Bool;
Post_post == LET \* @type: Bool;
__umpire_s3 == (TRUE) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s2 == IF __umpire_s3 THEN ((__umpire_world_mode) = (9)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s5 == __umpire_s3 IN LET \* @type: Bool;
__umpire_s4 == IF __umpire_s5 THEN (~(__umpire_s2)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s7 == (((TRUE) /\ (TRUE)) /\ (divisor # 0) /\ (~(value = -128 /\ divisor = -1))) /\ ((__umpire_SafeDiv(value, divisor)) >= -128 /\ (__umpire_SafeDiv(value, divisor)) <= 127) IN LET \* @type: Int;
__umpire_s6 == IF __umpire_s7 THEN (__umpire_SafeDiv(value, divisor)) ELSE 0 IN LET \* @type: Bool;
__umpire_s9 == (__umpire_s7) /\ (TRUE) IN LET \* @type: Bool;
__umpire_s8 == IF __umpire_s9 THEN ((__umpire_s6) = (0)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s11 == (__umpire_s9) IN LET \* @type: Bool;
__umpire_s10 == IF __umpire_s11 THEN ((__umpire_s8)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s13 == (__umpire_s5) /\ (IF __umpire_s4 THEN TRUE ELSE __umpire_s11) IN LET \* @type: Bool;
__umpire_s12 == IF __umpire_s13 THEN ((__umpire_s4) \/ (__umpire_s10)) ELSE FALSE IN LET \* @type: Bool;
__umpire_s15 == (__umpire_s13) IN LET \* @type: Bool;
__umpire_s14 == IF __umpire_s15 THEN ((__umpire_s12)) ELSE FALSE IN __umpire_arithmetic_error \/ ~(__umpire_has_step /\ __umpire_action = "User::Calculate") \/ __umpire_s14
\* @type: Bool;
ArithmeticSafe == ~__umpire_arithmetic_error
\* @type: Bool;
ValidStep == ~__umpire_invalid_step
====
