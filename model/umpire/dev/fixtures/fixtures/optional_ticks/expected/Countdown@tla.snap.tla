---- MODULE Countdown ----
EXTENDS Integers, TLC, Sequences
CONSTANT \* @type: Int;
__umpire_world_timeout
CONSTANT \* @type: Int;
__umpire_world_remaining
CONSTANT \* @type: Bool;
__umpire_world_empty
CONSTANT \* @type: Bool;
__umpire_world_duplicates
CONSTANT \* @type: Bool;
__umpire_world_invalid_update
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

\* @type: Int;
__umpire_source_REMAINING == (IF Assert(((0) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((0) + 1) ELSE ((0) + 1))
VARIABLE \* @type: Int;
remaining
VARIABLE \* @type: Bool;
started
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
\* @type: <<Int, Bool, Bool, Int, Int, Str, Int, Seq(Int), Seq(Seq(Int))>>;
vars == <<remaining, started, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: <<Int, Bool>>;
ModelVars == <<remaining, started>>

\* @type: Bool;
Init == (((remaining = 0) /\ (started = FALSE))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

\* @type: Bool;
Clock_Start == (LET (* @type: Bool; *) __umpire_l1 == ~(((__umpire_world_timeout) = (0))) IN LET (* @type: Int; *) __umpire_l2 == (IF Assert((((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1) ELSE (((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1)) IN LET (* @type: Int; *) __umpire_l3 == (IF __umpire_l1 THEN __umpire_l2 ELSE 0) IN LET (* @type: Bool; *) __umpire_l4 == (IF __umpire_l1 THEN TRUE ELSE TRUE) IN (~(started)) /\ ((TRUE) = (TRUE)) /\ ((~(((__umpire_world_timeout) = (0)))) = (~(((__umpire_world_timeout) = (0))))) /\ (((IF __umpire_l1 THEN (IF Assert((((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1) ELSE (((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1)) ELSE 0)) = ((IF __umpire_l1 THEN (IF Assert((((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1) ELSE (((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) + 1)) ELSE 0))) /\ (((IF __umpire_l1 THEN 0 ELSE 0)) = ((IF __umpire_l1 THEN 0 ELSE 0))) /\ ((__umpire_l3 # remaining) \/ (__umpire_l4 # started)) /\ (remaining' = __umpire_l3) /\ (started' = __umpire_l4) /\ (__umpire_action' = "Clock::Start") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(started))) /\ (TRUE))) /\ ((remaining # remaining) \/ (started # started)) /\ (remaining' = remaining) /\ (started' = started) /\ (__umpire_action' = "Clock::Start") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
Clock_Tick == (LET (* @type: Int; *) __umpire_l1 == (IF Assert(((4) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((4) + 1) ELSE ((4) + 1)) IN LET (* @type: Int; *) __umpire_l2 == (IF Assert((((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1) ELSE (((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1)) IN LET (* @type: Int; *) __umpire_l3 == (IF __umpire_world_invalid_update THEN __umpire_l1 ELSE __umpire_l2) IN (((((started) /\ (~(((remaining) = (0)))))) /\ ((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) > (0))))) /\ ((__umpire_world_invalid_update) = (__umpire_world_invalid_update)) /\ (((IF __umpire_world_invalid_update THEN (IF Assert(((4) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((4) + 1) ELSE ((4) + 1)) ELSE 0)) = ((IF __umpire_world_invalid_update THEN (IF Assert(((4) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((4) + 1) ELSE ((4) + 1)) ELSE 0))) /\ (((IF __umpire_world_invalid_update THEN 0 ELSE (IF Assert((((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1) ELSE (((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1)))) = ((IF __umpire_world_invalid_update THEN 0 ELSE (IF Assert((((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1) ELSE (((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) - (1)))) + 1))))) /\ ((__umpire_l3 # remaining) \/ (started # started)) /\ (remaining' = __umpire_l3) /\ (started' = started) /\ (__umpire_action' = "Clock::Tick") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((((started) /\ (~(((remaining) = (0)))))) /\ ((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) > (0)))))) /\ (TRUE))) /\ ((remaining # remaining) \/ (started # started)) /\ (remaining' = remaining) /\ (started' = started) /\ (__umpire_action' = "Clock::Tick") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
ModelNext == (Clock_Start) \/ (Clock_Tick)
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
Invariant_bounded == (<<((((remaining) = (0))) \/ (((((0) <= ((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))))) /\ ((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) <= (3)))))), ((((__umpire_world_timeout) = (0))) \/ (((((1) <= ((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))))) /\ ((((IF (1) = (1) THEN (IF (__umpire_world_timeout) # 0 THEN (__umpire_world_timeout) - 1 ELSE (1)) ELSE (1))) <= (3))))))>> = <<TRUE, TRUE>>)
\* @type: Bool;
Invariant_never_exhausted == (<<((~(started)) \/ (((remaining) # ((IF Assert(((0) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((0) + 1) ELSE ((0) + 1))))))>> = <<TRUE>>)
\* @type: Bool;
Post_decrements == [][~(ModelEdge /\ (__umpire_action' = "Clock::Tick")) \/ ((<<(((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) = ((IF Assert((((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) + (1)) \in (0..255), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) + (1)) ELSE (((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) + (1))))), ~(((__umpire_world_timeout) = (0)))>> = <<TRUE, TRUE>>))]_vars
\* @type: Bool;
Post_starts_selected == [][~(ModelEdge /\ (__umpire_action' = "Clock::Start")) \/ ((<<~(started), started', ((~(((remaining') = (0)))) = (~(((__umpire_world_timeout) = (0)))))>> = <<TRUE, TRUE, TRUE>>))]_vars
\* @type: Bool;
Post_never_decrements == [][~(ModelEdge /\ (__umpire_action' = "Clock::Tick")) \/ ((<<(((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) = ((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))))>> = <<TRUE>>))]_vars
\* @type: Bool;
Negative_never_started == ~(started)
\* @type: Bool;
Scoped_finishes == (Clock_Start) \/ (Clock_Tick)
\* @type: Bool;
Activate_finishes == (__umpire_pending = 0) /\ (started) /\ ~(((((remaining) = (0))) \/ ((((IF Assert((((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) + 1) ELSE (((IF (0) = (0) THEN (IF (remaining) # 0 THEN (remaining) - 1 ELSE (0)) ELSE (0))) + 1))) = ((IF Assert(((0) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((0) + 1) ELSE ((0) + 1))))))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: Bool;
Monitor_finishes == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((((remaining') = (0))) \/ ((((IF Assert((((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN (((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) + 1) ELSE (((IF (0) = (0) THEN (IF (remaining') # 0 THEN (remaining') - 1 ELSE (0)) ELSE (0))) + 1))) = ((IF Assert(((0) + 1) \in (1..2147483647), "checked Rust integer overflow") THEN ((0) + 1) ELSE ((0) + 1)))))) THEN 2 ELSE 1)) /\ (__umpire_steps' = (IF __umpire_steps < 4 THEN __umpire_steps + 1 ELSE __umpire_steps))
\* @type: Bool;
ActiveEdge_finishes == (__umpire_pending = 1) /\ Scoped_finishes /\ Monitor_finishes
\* @type: Bool;
PrefixEdge_finishes == (__umpire_pending = 0) /\ Edge
\* @type: Bool;
Complete_finishes == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_finishes) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
\* @type: Bool;
ProgressNext_finishes == PrefixEdge_finishes \/ Activate_finishes \/ ActiveEdge_finishes \/ Complete_finishes
\* @type: Bool;
ProgressSpec_finishes == Init /\ [][ProgressNext_finishes]_vars
\* @type: Bool;
Maximal_finishes == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_finishes))
\* @type: Bool;
Progress_finishes == (Maximal_finishes => ((__umpire_pending = 1) ~> (__umpire_pending = 2))) /\ [](__umpire_steps <= 3)
====
