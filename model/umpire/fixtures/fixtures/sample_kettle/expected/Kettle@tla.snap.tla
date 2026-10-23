---- MODULE Kettle ----
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
\* @type: <<Int, Bool, Bool, Bool, Int, Int, Str, Int, Seq(Int), Seq(Seq(Int))>>;
vars == <<temperature, heating_on, beeping, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: <<Int, Bool, Bool>>;
ModelVars == <<temperature, heating_on, beeping>>

\* @type: Bool;
Init == (temperature \in (-40..40)) /\ (heating_on = FALSE) /\ (beeping = FALSE) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

\* @type: Bool;
User_PressButton == ((~(heating_on)) /\ ((TRUE) = (TRUE)) /\ ((FALSE) = (FALSE)) /\ ((temperature # temperature) \/ (TRUE # heating_on) \/ (FALSE # beeping)) /\ (temperature' = temperature) /\ (heating_on' = TRUE) /\ (beeping' = FALSE) /\ (__umpire_action' = "User::PressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(heating_on))) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "User::PressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
User_DepressButton == ((heating_on) /\ ((FALSE) = (FALSE)) /\ ((FALSE) = (FALSE)) /\ ((temperature # temperature) \/ (FALSE # heating_on) \/ (FALSE # beeping)) /\ (temperature' = temperature) /\ (heating_on' = FALSE) /\ (beeping' = FALSE) /\ (__umpire_action' = "User::DepressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(heating_on)) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "User::DepressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
Heater_Heat == (LET (* @type: Int; *) __umpire_l1 == (IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1))) IN (((heating_on) /\ (((temperature) < (100))))) /\ (((IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1)))) = ((IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1))))) /\ ((__umpire_l1 # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = __umpire_l1) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "Heater::Heat") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((heating_on) /\ (((temperature) < (100)))))) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "Heater::Heat") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
Heater_Failover == ((((heating_on) /\ (((temperature) >= (100))))) /\ ((FALSE) = (FALSE)) /\ ((TRUE) = (TRUE)) /\ ((temperature # temperature) \/ (FALSE # heating_on) \/ (TRUE # beeping)) /\ (temperature' = temperature) /\ (heating_on' = FALSE) /\ (beeping' = TRUE) /\ (__umpire_action' = "Heater::Failover") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((heating_on) /\ (((temperature) >= (100)))))) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "Heater::Failover") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
\* @type: Bool;
ModelNext == (User_PressButton) \/ (User_DepressButton) \/ (Heater_Heat) \/ (Heater_Failover)
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
Invariant_beeps_only_when_off == (<<((~(beeping)) \/ ((<<~(heating_on)>> = <<TRUE>>)))>> = <<TRUE>>)
\* @type: Bool;
Post_warms_only_when_on == [][~(ModelEdge) \/ ((<<((~(~(heating_on))) \/ ((<<((temperature') <= (temperature))>> = <<TRUE>>)))>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_fails_over_at_boiling == [][~(ModelEdge /\ (__umpire_action' = "Heater::Failover")) \/ ((<<((temperature') = (100)), beeping'>> = <<TRUE, TRUE>>))]_vars
\* @type: Bool;
Negative_never_beeps == ~(beeping)
\* @type: Bool;
Assumption_never_switched_off == (User_PressButton) \/ (Heater_Heat) \/ (Heater_Failover)
\* @type: Bool;
Scoped_boils == Assumption_never_switched_off
\* @type: Bool;
Activate_boils == (__umpire_pending = 0) /\ (heating_on) /\ ~(beeping) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
\* @type: Bool;
Monitor_boils == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF beeping' THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
\* @type: Bool;
ActiveEdge_boils == (__umpire_pending = 1) /\ Scoped_boils /\ Monitor_boils
\* @type: Bool;
PrefixEdge_boils == (__umpire_pending = 0) /\ Edge
\* @type: Bool;
Complete_boils == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_boils) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
\* @type: Bool;
ProgressNext_boils == PrefixEdge_boils \/ Activate_boils \/ ActiveEdge_boils \/ Complete_boils
\* @type: Bool;
ProgressSpec_boils == Init /\ [][ProgressNext_boils]_vars
\* @type: Bool;
Maximal_boils == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_boils))
\* @type: Bool;
Progress_boils == (Maximal_boils => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
====
