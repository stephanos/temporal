---- MODULE Counter ----
EXTENDS Integers, TLC, Sequences
CONSTANT \* @type: Int;
__umpire_world_value
CONSTANT \* @type: Int;
__umpire_world_limit
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

VARIABLE \* @type: Int;
value
VARIABLE \* @type: Int;
origin
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
\* @type: <<Int, Int, Bool, Int, Int, Str, Int, Seq(Int), Seq(Seq(Int))>>;
vars == <<value, origin, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: <<Int, Int>>;
ModelVars == <<value, origin>>

\* @type: Bool;
Init == (((value = 1) /\ (origin = 1)) \/ ((value = 2) /\ (origin = 2))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

\* @type: Bool;
User_Advance == (LET (* @type: Int; *) __umpire_l1 == (IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1))) IN (((value) < (__umpire_world_limit))) /\ (((IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1)))) = ((IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1))))) /\ ((__umpire_l1 # value) \/ (origin # origin)) /\ (value' = __umpire_l1) /\ (origin' = origin) /\ (__umpire_action' = "User::Advance") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((value) < (__umpire_world_limit)))) /\ (TRUE))) /\ ((value # value) \/ (origin # origin)) /\ (value' = value) /\ (origin' = origin) /\ (__umpire_action' = "User::Advance") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
ModelNext == (User_Advance)
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
Invariant_bounded == (<<((value) <= (__umpire_world_limit))>> = <<TRUE>>)
\* @type: Bool;
Post_advances == [][~(ModelEdge /\ (__umpire_action' = "User::Advance")) \/ ((<<((value') = ((IF Assert(((value) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((value) + (1)) ELSE ((value) + (1))))), ((origin') = (origin)), ((value') <= (__umpire_world_limit))>> = <<TRUE, TRUE, TRUE>>))]_vars
\* @type: Bool;
Negative_never_three == ((value) < (3))
\* @type: Bool;
Assumption_fair_user == (User_Advance)
\* @type: Bool;
Scoped_finishes == Assumption_fair_user
\* @type: Bool;
Activate_finishes == (__umpire_pending = 0) /\ (((value) = (__umpire_world_value))) /\ ~(((value) = (__umpire_world_limit))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: Bool;
Monitor_finishes == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((value') = (__umpire_world_limit)) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
\* @type: Bool;
ActiveEdge_finishes == (__umpire_pending = 1) /\ Scoped_finishes /\ Monitor_finishes
\* @type: Bool;
PrefixEdge_finishes == (__umpire_pending = 0) /\ Edge
\* @type: Bool;
Complete_finishes == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_finishes) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
\* @type: Bool;
ProgressNext_finishes == PrefixEdge_finishes \/ Activate_finishes \/ ActiveEdge_finishes \/ Complete_finishes
\* @type: Bool;
Fair_finishes_User_Advance == (__umpire_pending = 1) /\ User_Advance /\ Monitor_finishes
\* @type: Bool;
ProgressSpec_finishes == Init /\ [][ProgressNext_finishes]_vars /\ WF_vars(Fair_finishes_User_Advance)
\* @type: Bool;
Maximal_finishes == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_finishes))
\* @type: Bool;
Progress_finishes == (Maximal_finishes => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
====
