---- MODULE ChangedState ----
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

VARIABLES value, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts
vars == <<value, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
ModelVars == <<value>>

Init == (value = FALSE) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

User_NoChange == ((TRUE) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::NoChange") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
User_SameValue == ((~(value)) /\ ((FALSE) = (FALSE)) /\ ((FALSE # value)) /\ (value' = FALSE) /\ (__umpire_action' = "User::SameValue") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(value))) /\ (TRUE))) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::SameValue") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
User_MutateRestore == ((TRUE) /\ ((value) = (value)) /\ ((~(value)) = (~(value))) /\ ((value) = (value)) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::MutateRestore") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((FALSE) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::MutateRestore") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
User_ActualChange == ((~(value)) /\ ((TRUE) = (TRUE)) /\ ((TRUE # value)) /\ (value' = TRUE) /\ (__umpire_action' = "User::ActualChange") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(value))) /\ (TRUE))) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::ActualChange") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
ModelNext == (User_NoChange) \/ (User_SameValue) \/ (User_MutateRestore) \/ (User_ActualChange)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Post_no_change_checked == [][~(ModelEdge /\ (__umpire_action' = "User::NoChange")) \/ (((value') # (value)))]_vars
Post_same_value_checked == [][~(ModelEdge /\ (__umpire_action' = "User::SameValue")) \/ (((value') # (value)))]_vars
Post_mutate_restore_checked == [][~(ModelEdge /\ (__umpire_action' = "User::MutateRestore")) \/ (((value') # (value)))]_vars
Post_actual_change_checked == [][~(ModelEdge /\ (__umpire_action' = "User::ActualChange")) \/ (((value') = (value)))]_vars
====
