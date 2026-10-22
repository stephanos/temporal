---- MODULE Counter ----
EXTENDS Integers, TLC, Sequences
CONSTANTS __umpire_world_value, __umpire_world_limit
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

VARIABLES value, origin, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts
vars == <<value, origin, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
ModelVars == <<value, origin>>

Init == (((value = 1) /\ (origin = 1)) \/ ((value = 2) /\ (origin = 2))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

User_Advance == (LET __umpire_l1 == (IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1))) IN (((value) < (__umpire_world_limit))) /\ (((IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1)))) = ((IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1))))) /\ ((__umpire_l1 # value) \/ (origin # origin)) /\ (value' = __umpire_l1) /\ (origin' = origin) /\ (__umpire_action' = "User::Advance") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((value) < (__umpire_world_limit)))) /\ (TRUE))) /\ ((value # value) \/ (origin # origin)) /\ (value' = value) /\ (origin' = origin) /\ (__umpire_action' = "User::Advance") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
ModelNext == (User_Advance)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Invariant_bounded == (<<((value) <= (__umpire_world_limit))>> = <<TRUE>>)
Post_advances == [][~(ModelEdge /\ (__umpire_action' = "User::Advance")) \/ ((<<((value') = ((IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1))))), ((origin') = (origin)), ((value') <= (__umpire_world_limit))>> = <<TRUE, TRUE, TRUE>>))]_vars
Negative_never_three == ((value) < (3))
Assumption_fair_user == (User_Advance)
Scoped_finishes == Assumption_fair_user
Activate_finishes == (__umpire_pending = 0) /\ (((value) = (__umpire_world_value))) /\ ~(((value) = (__umpire_world_limit))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
Monitor_finishes == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((value') = (__umpire_world_limit)) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
ActiveEdge_finishes == (__umpire_pending = 1) /\ Scoped_finishes /\ Monitor_finishes
PrefixEdge_finishes == (__umpire_pending = 0) /\ Edge
Complete_finishes == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_finishes) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_finishes == PrefixEdge_finishes \/ Activate_finishes \/ ActiveEdge_finishes \/ Complete_finishes
Fair_finishes_User_Advance == (__umpire_pending = 1) /\ User_Advance /\ Monitor_finishes
ProgressSpec_finishes == Init /\ [][ProgressNext_finishes]_vars /\ WF_vars(Fair_finishes_User_Advance)
Maximal_finishes == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_finishes))
Progress_finishes == (Maximal_finishes => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
====
