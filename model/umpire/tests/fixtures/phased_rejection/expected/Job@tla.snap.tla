---- MODULE Job ----
EXTENDS Integers, TLC, Sequences
CONSTANT \* @type: Int;
__umpire_world_attempts
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

VARIABLE \* @type: Str;
phase
VARIABLE \* @type: Int;
attempt
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
VARIABLE \* @type: Str;
__umpire_outcome
VARIABLE \* @type: Seq(Seq(Int));
__umpire_facts
VARIABLE \* @type: Int;
__umpire_input
\* @type: <<Str, Int, Bool, Int, Int, Str, Int, Str, Seq(Seq(Int)), Int>>;
vars == <<phase, attempt, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
\* @type: <<Str, Int>>;
ModelVars == <<phase, attempt>>

\* @type: Bool;
Init == (((phase = "Phase::Queued") /\ (attempt = 1))) /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = "Outcome::Accepted") /\ (__umpire_facts = <<>>) /\ (__umpire_input = 0)

\* @type: Seq(<<Int, Int>>);
Worker_Claim_Inputs == <<<<0, 1>>, <<1, 2>>>>
\* @type: Int => Bool;
Worker_Claim_At(__umpire_i) == LET (* @type: Int; *) __umpire_input_0 == Worker_Claim_Inputs[__umpire_i][2] IN ((((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped"))))) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Claim") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Rejected(NotFound)") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Claim_Inputs[__umpire_i][1])) \/ ((((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) /\ (((__umpire_input_0) # (attempt))))) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Claim") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Rejected(FailedPrecondition)") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Claim_Inputs[__umpire_i][1])) \/ ((((~(((((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped"))))) \/ (((__umpire_input_0) # (attempt)))))) /\ (((phase) = ("Phase::Queued"))))) /\ (("Phase::Running") = ("Phase::Running")) /\ (("Phase::Running" # phase) \/ (attempt # attempt)) /\ (phase' = "Phase::Running") /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Claim") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Claim_Inputs[__umpire_i][1])) \/ ((((~(((((((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped"))))) \/ (((__umpire_input_0) # (attempt))))) \/ (((phase) = ("Phase::Queued")))))) /\ (TRUE))) /\ ((phase # phase) \/ (attempt # attempt)) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Claim") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = Worker_Claim_Inputs[__umpire_i][1]))
\* @type: Bool;
Worker_Claim == \E __umpire_i \in DOMAIN Worker_Claim_Inputs : Worker_Claim_At(__umpire_i)
\* @type: Bool;
Worker_Finish == ((((phase) = ("Phase::Running"))) /\ (("Phase::Done") = ("Phase::Done")) /\ (("Phase::Done" # phase) \/ (attempt # attempt)) /\ (phase' = "Phase::Done") /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Finish") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 2)) \/ ((((~(((phase) = ("Phase::Running")))) /\ (TRUE))) /\ ((phase # phase) \/ (attempt # attempt)) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Finish") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 2))
\* @type: Bool;
Worker_Reopen == (LET (* @type: Int; *) __umpire_l1 == (IF Assert(((attempt) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((attempt) + (1)) ELSE ((attempt) + (1))) IN (((((phase) = ("Phase::Dropped"))) /\ (((attempt) < (__umpire_world_attempts))))) /\ (("Phase::Queued") = ("Phase::Queued")) /\ (((IF Assert(((attempt) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((attempt) + (1)) ELSE ((attempt) + (1)))) = ((IF Assert(((attempt) + (1)) \in (0..255), "checked Rust integer overflow") THEN ((attempt) + (1)) ELSE ((attempt) + (1))))) /\ (("Phase::Queued" # phase) \/ (__umpire_l1 # attempt)) /\ (phase' = "Phase::Queued") /\ (attempt' = __umpire_l1) /\ (__umpire_action' = "Worker::Reopen") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 3)) \/ ((((~(((((phase) = ("Phase::Dropped"))) /\ (((attempt) < (__umpire_world_attempts)))))) /\ (TRUE))) /\ ((phase # phase) \/ (attempt # attempt)) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Worker::Reopen") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 3))
\* @type: Bool;
Client_Drop == (LET (* @type: Bool; *) __umpire_l1 == ((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))) IN LET (* @type: Int; *) __umpire_l2 == (IF __umpire_l1 THEN attempt ELSE attempt) IN LET (* @type: Str; *) __umpire_l3 == (IF __umpire_l1 THEN phase ELSE phase) IN LET (* @type: Bool; *) __umpire_l4 == ~(__umpire_l1) IN LET (* @type: Bool; *) __umpire_l5 == ((((__umpire_l3) = ("Phase::Done"))) \/ (((__umpire_l3) = ("Phase::Dropped")))) IN LET (* @type: Int; *) __umpire_l6 == (IF __umpire_l5 THEN __umpire_l2 ELSE __umpire_l2) IN LET (* @type: Str; *) __umpire_l7 == (IF __umpire_l5 THEN __umpire_l3 ELSE __umpire_l3) IN LET (* @type: Bool; *) __umpire_l8 == ~(__umpire_l5) IN LET (* @type: Str; *) __umpire_l9 == (IF __umpire_l8 THEN "Phase::Dropped" ELSE __umpire_l7) IN (TRUE) /\ ((((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped"))))) = (((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) /\ ((((((__umpire_l3) = ("Phase::Done"))) \/ (((__umpire_l3) = ("Phase::Dropped"))))) = (((((__umpire_l3) = ("Phase::Done"))) \/ (((__umpire_l3) = ("Phase::Dropped")))))) /\ (((IF __umpire_l8 THEN "Phase::Dropped" ELSE "Phase::Queued")) = ((IF __umpire_l8 THEN "Phase::Dropped" ELSE "Phase::Queued"))) /\ (Assert(((IF __umpire_l1 THEN "Outcome::Rejected(NotFound)" ELSE (IF __umpire_l5 THEN "Outcome::Rejected(FailedPrecondition)" ELSE "Outcome::Accepted")) = "Outcome::Accepted") \/ ((__umpire_l9 = phase) /\ (__umpire_l6 = attempt)), "rejecting step changed state")) /\ ((__umpire_l9 # phase) \/ (__umpire_l6 # attempt) \/ ((IF __umpire_l1 THEN "Outcome::Rejected(NotFound)" ELSE (IF __umpire_l5 THEN "Outcome::Rejected(FailedPrecondition)" ELSE "Outcome::Accepted")) # "Outcome::Accepted")) /\ (phase' = __umpire_l9) /\ (attempt' = __umpire_l6) /\ (__umpire_action' = "Client::Drop") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = (IF __umpire_l1 THEN "Outcome::Rejected(NotFound)" ELSE (IF __umpire_l5 THEN "Outcome::Rejected(FailedPrecondition)" ELSE "Outcome::Accepted"))) /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 4)) \/ ((FALSE) /\ ((phase # phase) \/ (attempt # attempt)) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Client::Drop") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 4))
\* @type: Bool;
Client_Touch == ((((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped"))))) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Client::Touch") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 5)) \/ ((((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) /\ (TRUE))) /\ ((phase # phase) \/ (attempt # attempt)) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Client::Touch") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 5))
\* @type: Bool;
Client_Recheck == ((((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped"))))) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Client::Recheck") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Rejected(Unavailable)") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 6)) \/ ((((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) /\ (TRUE))) /\ ((phase # phase) \/ (attempt # attempt)) /\ (phase' = phase) /\ (attempt' = attempt) /\ (__umpire_action' = "Client::Recheck") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = "Outcome::Accepted") /\ (__umpire_facts' = <<>>) /\ (__umpire_input' = 6))
\* @type: Bool;
ModelNext == (Worker_Claim) \/ (Worker_Finish) \/ (Worker_Reopen) \/ (Client_Drop) \/ (Client_Touch) \/ (Client_Recheck)
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
Post_terminal_remains == [][~(ModelEdge) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((phase') = (phase))>> = <<TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_closed_rejects == [][~(ModelEdge) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((((phase') = (phase))) /\ (((attempt') = (attempt)))), ((__umpire_outcome') = ("Outcome::Rejected(NotFound)"))>> = <<TRUE, TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_drop_rejects == [][~(ModelEdge /\ (__umpire_action' = "Client::Drop")) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((((phase') = (phase))) /\ (((attempt') = (attempt)))), ((__umpire_outcome') = ("Outcome::Rejected(NotFound)"))>> = <<TRUE, TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_reopen_remains == [][~(ModelEdge /\ (__umpire_action' = "Worker::Reopen")) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((phase') = (phase))>> = <<TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_touch_remains == [][~(ModelEdge /\ (__umpire_action' = "Client::Touch")) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((phase') = (phase))>> = <<TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_touch_rejects == [][~(ModelEdge /\ (__umpire_action' = "Client::Touch")) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((((phase') = (phase))) /\ (((attempt') = (attempt)))), ((__umpire_outcome') = ("Outcome::Rejected(NotFound)"))>> = <<TRUE, TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_recheck_rejects == [][~(ModelEdge /\ (__umpire_action' = "Client::Recheck")) \/ ((<<(<<((~(((((phase) = ("Phase::Done"))) \/ (((phase) = ("Phase::Dropped")))))) \/ ((<<((((phase') = (phase))) /\ (((attempt') = (attempt)))), ((__umpire_outcome') = ("Outcome::Rejected(NotFound)"))>> = <<TRUE, TRUE>>)))>> = <<TRUE>>)>> = <<TRUE>>))]_vars
\* @type: Bool;
Post_steps_accept == [][~(ModelEdge) \/ ((<<((((__umpire_outcome') = ("Outcome::Accepted"))) \/ (((((phase') = ("Phase::Done"))) \/ (((phase') = ("Phase::Dropped"))))))>> = <<TRUE>>))]_vars
\* @type: Bool;
Assumption_current_claims == (Worker_Claim) \/ (Worker_Finish) \/ (Worker_Reopen) \/ (Client_Drop) \/ (Client_Touch) \/ (Client_Recheck)
\* @type: Bool;
Scoped_queued_job_runs == Assumption_current_claims
\* @type: Bool;
Activate_queued_job_runs == (__umpire_pending = 0) /\ (((phase) = ("Phase::Queued"))) /\ ~(((phase) # ("Phase::Queued"))) /\ (__umpire_pending' = 1) /\ (__umpire_steps' = 0) /\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts, __umpire_input>>
\* @type: Bool;
Monitor_queued_job_runs == (__umpire_edge_tick' = ~__umpire_edge_tick) /\ (__umpire_pending' = (IF ((phase') # ("Phase::Queued")) THEN 2 ELSE 1)) /\ (__umpire_steps' = 0)
\* @type: Bool;
ActiveEdge_queued_job_runs == (__umpire_pending = 1) /\ Scoped_queued_job_runs /\ Monitor_queued_job_runs
\* @type: Bool;
PrefixEdge_queued_job_runs == (__umpire_pending = 0) /\ Edge
\* @type: Bool;
Complete_queued_job_runs == ((__umpire_pending = 2) \/ ((__umpire_pending = 1) /\ ~ENABLED Scoped_queued_job_runs) \/ ((__umpire_pending = 0) /\ ~ENABLED ModelNext)) /\ UNCHANGED vars
\* @type: Bool;
ProgressNext_queued_job_runs == PrefixEdge_queued_job_runs \/ Activate_queued_job_runs \/ ActiveEdge_queued_job_runs \/ Complete_queued_job_runs
\* @type: Bool;
Fair_queued_job_runs_Worker_Claim_1 == (__umpire_pending = 1) /\ Worker_Claim_At(1) /\ Monitor_queued_job_runs
\* @type: Bool;
Fair_queued_job_runs_Worker_Claim_2 == (__umpire_pending = 1) /\ Worker_Claim_At(2) /\ Monitor_queued_job_runs
\* @type: Bool;
ProgressSpec_queued_job_runs == Init /\ [][ProgressNext_queued_job_runs]_vars /\ WF_vars(Fair_queued_job_runs_Worker_Claim_1) /\ WF_vars(Fair_queued_job_runs_Worker_Claim_2)
\* @type: Bool;
Maximal_queued_job_runs == (([]<>__umpire_edge_tick) /\ ([]<>(~__umpire_edge_tick))) \/ (<>[]((__umpire_pending = 1) /\ ~ENABLED Scoped_queued_job_runs))
\* @type: Bool;
Progress_queued_job_runs == (Maximal_queued_job_runs => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))
====
