---- MODULE Kettle ----
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

BOILING_TEMPERATURE == 100
COLDEST_TEMPERATURE == -40
VARIABLES temperature, heating_on, beeping, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts
vars == <<temperature, heating_on, beeping, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
ModelVars == <<temperature, heating_on, beeping>>

Init == (temperature \in (-40..40)) /\ (heating_on = FALSE) /\ (beeping = FALSE) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

User_PressButton == ((~(heating_on)) /\ ((TRUE) = (TRUE)) /\ ((FALSE) = (FALSE)) /\ ((temperature # temperature) \/ (TRUE # heating_on) \/ (FALSE # beeping)) /\ (temperature' = temperature) /\ (heating_on' = TRUE) /\ (beeping' = FALSE) /\ (__umpire_action' = "User::PressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(~(heating_on))) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "User::PressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
User_DepressButton == ((heating_on) /\ ((FALSE) = (FALSE)) /\ ((FALSE) = (FALSE)) /\ ((temperature # temperature) \/ (FALSE # heating_on) \/ (FALSE # beeping)) /\ (temperature' = temperature) /\ (heating_on' = FALSE) /\ (beeping' = FALSE) /\ (__umpire_action' = "User::DepressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(heating_on)) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "User::DepressButton") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Heater_Heat == ((((heating_on) /\ (((temperature) < (100))))) /\ (((IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1)))) = ((IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1))))) /\ (((IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1))) # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = (IF Assert(((temperature) + (1)) \in ((-2147483647 - 1)..2147483647), "checked Rust integer overflow") THEN ((temperature) + (1)) ELSE ((temperature) + (1)))) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "Heater::Heat") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((heating_on) /\ (((temperature) < (100)))))) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "Heater::Heat") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Heater_Failover == ((((heating_on) /\ (((temperature) >= (100))))) /\ ((FALSE) = (FALSE)) /\ ((TRUE) = (TRUE)) /\ ((temperature # temperature) \/ (FALSE # heating_on) \/ (TRUE # beeping)) /\ (temperature' = temperature) /\ (heating_on' = FALSE) /\ (beeping' = TRUE) /\ (__umpire_action' = "Heater::Failover") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((heating_on) /\ (((temperature) >= (100)))))) /\ (TRUE))) /\ ((temperature # temperature) \/ (heating_on # heating_on) \/ (beeping # beeping)) /\ (temperature' = temperature) /\ (heating_on' = heating_on) /\ (beeping' = beeping) /\ (__umpire_action' = "Heater::Failover") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
ModelNext == (User_PressButton) \/ (User_DepressButton) \/ (Heater_Heat) \/ (Heater_Failover)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Invariant_beeps_only_when_off == (<<((~(beeping)) \/ ((<<~(heating_on)>> = <<TRUE>>)))>> = <<TRUE>>)
Post_warms_only_when_on == [][~(ModelEdge) \/ ((<<((~(~(heating_on))) \/ ((<<((temperature') <= (temperature))>> = <<TRUE>>)))>> = <<TRUE>>))]_vars
Post_fails_over_at_boiling == [][~(ModelEdge /\ (__umpire_action' = "Heater::Failover")) \/ ((<<((temperature') = (100)), beeping'>> = <<TRUE, TRUE>>))]_vars
Negative_never_beeps == ~(beeping)
Assumption_never_switched_off == (User_PressButton) \/ (Heater_Heat) \/ (Heater_Failover)
Scoped_boils == Assumption_never_switched_off
Activate_boils == (__umpire_pending = 0) /\ (heating_on) /\ ~(beeping) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
Monitor_boils == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF beeping' THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
ActiveEdge_boils == (__umpire_pending = 1) /\ Scoped_boils /\ Monitor_boils
PrefixEdge_boils == (__umpire_pending = 0) /\ Edge
Complete_boils == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_boils) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_boils == PrefixEdge_boils \/ Activate_boils \/ ActiveEdge_boils \/ Complete_boils
ProgressSpec_boils == Init /\ [][ProgressNext_boils]_vars
Maximal_boils == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_boils))
Progress_boils == (Maximal_boils => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
====
