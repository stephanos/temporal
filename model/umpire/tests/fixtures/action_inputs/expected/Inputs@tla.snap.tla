---- MODULE Inputs ----
EXTENDS Integers, TLC, Sequences
CONSTANTS __umpire_world_attempts, __umpire_world_spares
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

VARIABLES attempt, done, failed, skipped, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input
vars == <<attempt, done, failed, skipped, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
ModelVars == <<attempt, done, failed, skipped>>

Init == (((attempt = 1) /\ (done = FALSE) /\ (failed = FALSE) /\ (skipped = FALSE))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>) /\ (__umpire_input = 0)

Worker_Poll_Inputs == <<<<0, 1>>, <<1, 2>>>>
Worker_Poll_At(__umpire_i) == LET __umpire_input_0 == Worker_Poll_Inputs[__umpire_i][2] IN ((((((((__umpire_input_0) = (attempt))) /\ (~(done)))) /\ (~(failed)))) /\ ((TRUE) = (TRUE)) /\ ((attempt # attempt) \/ (TRUE # done) \/ (failed # failed) \/ (skipped # skipped)) /\ (attempt' = attempt) /\ (done' = TRUE) /\ (failed' = failed) /\ (skipped' = skipped) /\ (__umpire_action' = "Worker::Poll") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Poll_Inputs[__umpire_i][1])) \/ ((((~(((((((__umpire_input_0) = (attempt))) /\ (~(done)))) /\ (~(failed))))) /\ (TRUE))) /\ (attempt' = attempt) /\ (done' = done) /\ (failed' = failed) /\ (skipped' = skipped) /\ (__umpire_action' = "Worker::Poll") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Poll_Inputs[__umpire_i][1])) \/ ((((~(((((((((__umpire_input_0) = (attempt))) /\ (~(done)))) /\ (~(failed)))) \/ (TRUE)))) /\ (TRUE))) /\ ((attempt # attempt) \/ (done # done) \/ (failed # failed) \/ (skipped # skipped)) /\ (attempt' = attempt) /\ (done' = done) /\ (failed' = failed) /\ (skipped' = skipped) /\ (__umpire_action' = "Worker::Poll") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Poll_Inputs[__umpire_i][1]))
Worker_Poll == \E __umpire_i \in DOMAIN Worker_Poll_Inputs : Worker_Poll_At(__umpire_i)
Worker_Fail_Inputs == <<<<2, FALSE>>, <<3, TRUE>>>>
Worker_Fail_At(__umpire_i) == LET __umpire_input_retryable == Worker_Fail_Inputs[__umpire_i][2] IN (LET __umpire_l1 == ((__umpire_input_retryable) /\ (((attempt) < (__umpire_world_attempts)))) IN LET __umpire_l2 == (IF Assert(((attempt) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((attempt) + (1)) ELSE ((attempt) + (1))) IN LET __umpire_l3 == (IF __umpire_l1 THEN __umpire_l2 ELSE attempt) IN LET __umpire_l4 == (IF __umpire_l1 THEN done ELSE done) IN LET __umpire_l5 == (IF __umpire_l1 THEN failed ELSE TRUE) IN LET __umpire_l6 == (IF __umpire_l1 THEN skipped ELSE skipped) IN (((~(done)) /\ (~(failed)))) /\ (((IF __umpire_l1 THEN (IF Assert(((attempt) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((attempt) + (1)) ELSE ((attempt) + (1))) ELSE 0)) = ((IF __umpire_l1 THEN (IF Assert(((attempt) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((attempt) + (1)) ELSE ((attempt) + (1))) ELSE 0))) /\ (((IF __umpire_l1 THEN FALSE ELSE TRUE)) = ((IF __umpire_l1 THEN FALSE ELSE TRUE))) /\ ((__umpire_l3 # attempt) \/ (__umpire_l4 # done) \/ (__umpire_l5 # failed) \/ (__umpire_l6 # skipped)) /\ (attempt' = __umpire_l3) /\ (done' = __umpire_l4) /\ (failed' = __umpire_l5) /\ (skipped' = __umpire_l6) /\ (__umpire_action' = "Worker::Fail") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Fail_Inputs[__umpire_i][1])) \/ ((((~(((~(done)) /\ (~(failed))))) /\ (TRUE))) /\ ((attempt # attempt) \/ (done # done) \/ (failed # failed) \/ (skipped # skipped)) /\ (attempt' = attempt) /\ (done' = done) /\ (failed' = failed) /\ (skipped' = skipped) /\ (__umpire_action' = "Worker::Fail") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Fail_Inputs[__umpire_i][1]))
Worker_Fail == \E __umpire_i \in DOMAIN Worker_Fail_Inputs : Worker_Fail_At(__umpire_i)
Worker_Skip_Inputs == <<>>
Worker_Skip_At(__umpire_i) == LET __umpire_input_0 == Worker_Skip_Inputs[__umpire_i][2] IN ((~(skipped)) /\ ((TRUE) = (TRUE)) /\ ((attempt # attempt) \/ (done # done) \/ (failed # failed) \/ (TRUE # skipped)) /\ (attempt' = attempt) /\ (done' = done) /\ (failed' = failed) /\ (skipped' = TRUE) /\ (__umpire_action' = "Worker::Skip") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Skip_Inputs[__umpire_i][1])) \/ ((((~(~(skipped))) /\ (TRUE))) /\ ((attempt # attempt) \/ (done # done) \/ (failed # failed) \/ (skipped # skipped)) /\ (attempt' = attempt) /\ (done' = done) /\ (failed' = failed) /\ (skipped' = skipped) /\ (__umpire_action' = "Worker::Skip") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Skip_Inputs[__umpire_i][1]))
Worker_Skip == \E __umpire_i \in DOMAIN Worker_Skip_Inputs : Worker_Skip_At(__umpire_i)
ModelNext == (Worker_Poll) \/ (Worker_Fail) \/ (Worker_Skip)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Invariant_attempts_bounded == (<<((((attempt) >= (1))) /\ (((attempt) <= (__umpire_world_attempts))))>> = <<TRUE>>)
Post_attempts_never_decrease == [][~(ModelEdge) \/ ((<<((attempt') >= (attempt)), ((~(done)) \/ ((<<done'>> = <<TRUE>>)))>> = <<TRUE, TRUE>>))]_vars
Negative_never_retried == ((attempt) = (1))
Assumption_current_polls == (Worker_Poll) \/ (Worker_Fail) \/ (Worker_Skip)
Assumption_stale_poll == (Worker_Poll) \/ (Worker_Fail) \/ (Worker_Skip)
Assumption_first_excluded == (\E __umpire_i \in {2} : Worker_Poll_At(__umpire_i)) \/ (Worker_Fail) \/ (Worker_Skip)
Scoped_current_poll_settles == Assumption_current_polls
Activate_current_poll_settles == (__umpire_pending = 0) /\ (((~(done)) /\ (~(failed)))) /\ ~(((done) \/ (failed))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
Monitor_current_poll_settles == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((done') \/ (failed')) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
ActiveEdge_current_poll_settles == (__umpire_pending = 1) /\ Scoped_current_poll_settles /\ Monitor_current_poll_settles
PrefixEdge_current_poll_settles == (__umpire_pending = 0) /\ Edge
Complete_current_poll_settles == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_current_poll_settles) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_current_poll_settles == PrefixEdge_current_poll_settles \/ Activate_current_poll_settles \/ ActiveEdge_current_poll_settles \/ Complete_current_poll_settles
Fair_current_poll_settles_Worker_Poll_1 == (__umpire_pending = 1) /\ Worker_Poll_At(1) /\ Monitor_current_poll_settles
Fair_current_poll_settles_Worker_Poll_2 == (__umpire_pending = 1) /\ Worker_Poll_At(2) /\ Monitor_current_poll_settles
ProgressSpec_current_poll_settles == Init /\ [][ProgressNext_current_poll_settles]_vars /\ WF_vars(Fair_current_poll_settles_Worker_Poll_1) /\ WF_vars(Fair_current_poll_settles_Worker_Poll_2)
Maximal_current_poll_settles == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_current_poll_settles))
Progress_current_poll_settles == (Maximal_current_poll_settles => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
Scoped_stale_poll_settles == Assumption_stale_poll
Activate_stale_poll_settles == (__umpire_pending = 0) /\ (((~(done)) /\ (~(failed)))) /\ ~(((done) \/ (failed))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
Monitor_stale_poll_settles == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((done') \/ (failed')) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
ActiveEdge_stale_poll_settles == (__umpire_pending = 1) /\ Scoped_stale_poll_settles /\ Monitor_stale_poll_settles
PrefixEdge_stale_poll_settles == (__umpire_pending = 0) /\ Edge
Complete_stale_poll_settles == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_stale_poll_settles) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_stale_poll_settles == PrefixEdge_stale_poll_settles \/ Activate_stale_poll_settles \/ ActiveEdge_stale_poll_settles \/ Complete_stale_poll_settles
Fair_stale_poll_settles_Worker_Poll_2 == (__umpire_pending = 1) /\ Worker_Poll_At(2) /\ Monitor_stale_poll_settles
ProgressSpec_stale_poll_settles == Init /\ [][ProgressNext_stale_poll_settles]_vars /\ WF_vars(Fair_stale_poll_settles_Worker_Poll_2)
Maximal_stale_poll_settles == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_stale_poll_settles))
Progress_stale_poll_settles == (Maximal_stale_poll_settles => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
Scoped_excluded_poll_settles == Assumption_first_excluded
Activate_excluded_poll_settles == (__umpire_pending = 0) /\ (((~(done)) /\ (~(failed)))) /\ ~(((done) \/ (failed))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
Monitor_excluded_poll_settles == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((done') \/ (failed')) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
ActiveEdge_excluded_poll_settles == (__umpire_pending = 1) /\ Scoped_excluded_poll_settles /\ Monitor_excluded_poll_settles
PrefixEdge_excluded_poll_settles == (__umpire_pending = 0) /\ Edge
Complete_excluded_poll_settles == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_excluded_poll_settles) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_excluded_poll_settles == PrefixEdge_excluded_poll_settles \/ Activate_excluded_poll_settles \/ ActiveEdge_excluded_poll_settles \/ Complete_excluded_poll_settles
Fair_excluded_poll_settles_Worker_Poll_2 == (__umpire_pending = 1) /\ Worker_Poll_At(2) /\ Monitor_excluded_poll_settles
ProgressSpec_excluded_poll_settles == Init /\ [][ProgressNext_excluded_poll_settles]_vars /\ WF_vars(Fair_excluded_poll_settles_Worker_Poll_2)
Maximal_excluded_poll_settles == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_excluded_poll_settles))
Progress_excluded_poll_settles == (Maximal_excluded_poll_settles => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
Scoped_settles_within_four == Assumption_current_polls
Activate_settles_within_four == (__umpire_pending = 0) /\ (((~(done)) /\ (~(failed)))) /\ ~(((done) \/ (failed))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
Monitor_settles_within_four == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((done') \/ (failed')) THEN 2 ELSE 1)) /\ (__umpire_steps' = (IF __umpire_steps < 5 THEN __umpire_steps + 1 ELSE __umpire_steps))
ActiveEdge_settles_within_four == (__umpire_pending = 1) /\ Scoped_settles_within_four /\ Monitor_settles_within_four
PrefixEdge_settles_within_four == (__umpire_pending = 0) /\ Edge
Complete_settles_within_four == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_settles_within_four) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
ProgressNext_settles_within_four == PrefixEdge_settles_within_four \/ Activate_settles_within_four \/ ActiveEdge_settles_within_four \/ Complete_settles_within_four
ProgressSpec_settles_within_four == Init /\ [][ProgressNext_settles_within_four]_vars
Maximal_settles_within_four == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_settles_within_four))
Progress_settles_within_four == (Maximal_settles_within_four => ((__umpire_pending = 1) ~> (__umpire_pending = 2))) /\ [](__umpire_steps <= 4)
====
