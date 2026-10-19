---- MODULE State ----
EXTENDS Integers, TLC, Sequences
__umpire_Div(__umpire_a, __umpire_b) ==
    IF Assert(__umpire_b # 0, "Rust integer division by zero") /\ Assert(~(__umpire_a = (-2147483647 - 1) /\ __umpire_b = -1), "Rust integer division overflow")
    THEN IF __umpire_b = (-2147483647 - 1)
         THEN IF __umpire_a = (-2147483647 - 1) THEN 1 ELSE 0
         ELSE IF __umpire_a < 0
              THEN LET __umpire_d == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
                       __umpire_n == -(__umpire_a + 1)
                       __umpire_q == __umpire_n \div __umpire_d
                       __umpire_r == __umpire_n - __umpire_q * __umpire_d
                       __umpire_negative == -__umpire_q - (IF __umpire_r = __umpire_d - 1 THEN 1 ELSE 0)
                   IN IF __umpire_b < 0 THEN -__umpire_negative ELSE __umpire_negative
              ELSE IF __umpire_b < 0 THEN -(__umpire_a \div (-__umpire_b)) ELSE __umpire_a \div __umpire_b
    ELSE 0

VARIABLES field_0, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts
vars == <<field_0, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
ModelVars == <<field_0>>

Init == (((field_0 = 0))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

Action_Enter == FALSE
Action_Left == FALSE
Action_Right == FALSE
Action_Escape == FALSE
ModelNext == (Action_Enter) \/ (Action_Left) \/ (Action_Right) \/ (Action_Escape)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Scoped_reaches_target == (Action_Enter) \/ (Action_Left) \/ (Action_Right) \/ (Action_Escape)
Activate_reaches_target == (__umpire_pending = 0) /\ (((field_0) = (0))) /\ ~(((((field_0) = (9))) \/ ((("Mode::Fulfilled") = ("Mode::Fulfilled"))))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
Monitor_reaches_target == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((((field_0') = (9))) \/ ((("Mode::Fulfilled") = ("Mode::Fulfilled")))) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
ActiveEdge_reaches_target == (__umpire_pending = 1) /\ Scoped_reaches_target /\ Monitor_reaches_target
PrefixEdge_reaches_target == (__umpire_pending = 0) /\ Edge
Complete_reaches_target == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_reaches_target) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_reaches_target == PrefixEdge_reaches_target \/ Activate_reaches_target \/ ActiveEdge_reaches_target \/ Complete_reaches_target
ProgressSpec_reaches_target == Init /\ [][ProgressNext_reaches_target]_vars
Maximal_reaches_target == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_reaches_target))
Progress_reaches_target == (Maximal_reaches_target => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
====
