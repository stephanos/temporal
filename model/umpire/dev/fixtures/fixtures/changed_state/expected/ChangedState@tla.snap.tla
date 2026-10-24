---- MODULE ChangedState ----
EXTENDS Integers, TLC, Sequences
\* @type: (Int, Int) => Int;
__umpire_Div(__umpire_a, __umpire_b) ==
    IF Assert(__umpire_b # 0, "Rust integer division by zero") /\ Assert(~(__umpire_a = (-2147483647 - 1) /\ __umpire_b = -1), "Rust integer division overflow")
    THEN IF __umpire_b = (-2147483647 - 1)
         THEN IF __umpire_a = (-2147483647 - 1) THEN 1 ELSE 0
         ELSE IF __umpire_a < 0
              THEN LET (* @type: Int; *) __umpire_d == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
                       (* @type: Int; *) __umpire_n == -(__umpire_a + 1)
                       (* @type: Int; *) __umpire_q == __umpire_n \div __umpire_d
                       (* @type: Int; *) __umpire_r == __umpire_n - __umpire_q * __umpire_d
                       (* @type: Int; *) __umpire_negative == -__umpire_q - (IF __umpire_r = __umpire_d - 1 THEN 1 ELSE 0)
                   IN IF __umpire_b < 0 THEN -__umpire_negative ELSE __umpire_negative
              ELSE IF __umpire_b < 0 THEN -(__umpire_a \div (-__umpire_b)) ELSE __umpire_a \div __umpire_b
    ELSE 0

VARIABLE \* @type: Bool;
value
VARIABLE \* @type: Bool;
__umpire_edge_tick
VARIABLE \* @type: Int;
__umpire_pending
VARIABLE \* @type: Int;
__umpire_steps
VARIABLE \* @type: Str;
__umpire_action
VARIABLE \* @type: Int;
__umpire_choice
VARIABLE \* @type: Seq(Int);
__umpire_outcome
VARIABLE \* @type: Seq(Seq(Int));
__umpire_facts
\* @type: <<Bool, Bool, Int, Int, Str, Int, Seq(Int), Seq(Seq(Int))>>;
vars == <<value, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: <<Bool>>;
ModelVars == <<value>>

\* @type: Bool;
Init == (value = FALSE) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

\* @type: Bool;
User_NoChange == ((TRUE) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::NoChange") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
User_SameValue == ((~(value)) /\ ((FALSE) = (FALSE)) /\ ((FALSE # value)) /\ (value' = FALSE) /\ (__umpire_action' = "User::SameValue") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(value))) /\ (TRUE))) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::SameValue") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
User_MutateRestore == (LET (* @type: Bool; *) __umpire_l1 == ~(value) IN (TRUE) /\ ((value) = (value)) /\ ((~(value)) = (~(value))) /\ ((value) = (value)) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::MutateRestore") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((FALSE) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::MutateRestore") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
User_ActualChange == ((~(value)) /\ ((TRUE) = (TRUE)) /\ ((TRUE # value)) /\ (value' = TRUE) /\ (__umpire_action' = "User::ActualChange") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(value))) /\ (TRUE))) /\ ((value # value)) /\ (value' = value) /\ (__umpire_action' = "User::ActualChange") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
ModelNext == (User_NoChange) \/ (User_SameValue) \/ (User_MutateRestore) \/ (User_ActualChange)
\* @type: Bool;
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
\* @type: Bool;
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
\* @type: Bool;
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
\* @type: Bool;
Next == Edge \/ Complete
\* @type: Bool;
Spec == Init /\ [][Next]_vars

\* @type: Bool;
Post_no_change_checked == [][~(ModelEdge /\ (__umpire_action' = "User::NoChange")) \/ ((<<((value') # (value))>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_same_value_checked == [][~(ModelEdge /\ (__umpire_action' = "User::SameValue")) \/ ((<<((value') # (value))>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_mutate_restore_checked == [][~(ModelEdge /\ (__umpire_action' = "User::MutateRestore")) \/ ((<<((value') # (value))>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_actual_change_checked == [][~(ModelEdge /\ (__umpire_action' = "User::ActualChange")) \/ ((<<((value') = (value))>> = <<TRUE>>))]_vars
====
