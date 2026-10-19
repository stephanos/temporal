---- MODULE Value ----
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

Init == (((field_0 = 0))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = FALSE) /\ (__umpire_facts = <<>>)

Action_Advance == ((((((field_0) = (0))) /\ (TRUE))) /\ (field_0' = 2) /\ (__umpire_action' = "Action::Advance") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"rejected">>)) \/ ((((((field_0) = (0))) /\ (TRUE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Advance") /\ (__umpire_choice' = 1) /\ (__umpire_outcome' = TRUE) /\ (__umpire_facts' = <<"accepted">>)) \/ ((((((field_0) = (0))) /\ (TRUE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Advance") /\ (__umpire_choice' = 2) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"same_target">>))
Action_Reset == ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 2) /\ (__umpire_action' = "Action::Reset") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"rejected">>)) \/ ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Reset") /\ (__umpire_choice' = 1) /\ (__umpire_outcome' = TRUE) /\ (__umpire_facts' = <<"accepted">>)) \/ ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Reset") /\ (__umpire_choice' = 2) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"same_target">>))
Action_Idle == ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 2) /\ (__umpire_action' = "Action::Idle") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"rejected">>)) \/ ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Idle") /\ (__umpire_choice' = 1) /\ (__umpire_outcome' = TRUE) /\ (__umpire_facts' = <<"accepted">>)) \/ ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Idle") /\ (__umpire_choice' = 2) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"same_target">>))
Action_Unused == ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 2) /\ (__umpire_action' = "Action::Unused") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"rejected">>)) \/ ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Unused") /\ (__umpire_choice' = 1) /\ (__umpire_outcome' = TRUE) /\ (__umpire_facts' = <<"accepted">>)) \/ ((((((field_0) = (0))) /\ (FALSE))) /\ (field_0' = 1) /\ (__umpire_action' = "Action::Unused") /\ (__umpire_choice' = 2) /\ (__umpire_outcome' = FALSE) /\ (__umpire_facts' = <<"same_target">>))
ModelNext == (Action_Advance) \/ (Action_Reset) \/ (Action_Idle) \/ (Action_Unused)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Invariant_safe == TRUE
Post_choice_facts == [][~(ModelEdge) \/ (((((((__umpire_outcome') /\ (((field_0') = (1))))) /\ (((__umpire_facts') = (<<"accepted">>))))) \/ (((~(__umpire_outcome')) /\ (((((((field_0') = (2))) /\ (((__umpire_facts') = (<<"rejected">>))))) \/ (((((field_0') = (1))) /\ (((__umpire_facts') = (<<"same_target">>)))))))))))]_vars
====
