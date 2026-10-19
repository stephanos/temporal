---- MODULE Activity ----
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

VARIABLES phase, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts
vars == <<phase, __umpire_edge_tick, __umpire_pending, __umpire_steps, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>
ModelVars == <<phase>>

Init == (phase = "Phase::Scheduled") /\ (__umpire_edge_tick = FALSE) /\ (__umpire_pending = 0) /\ (__umpire_steps = 0) /\ (__umpire_action = "") /\ (__umpire_choice = 0) /\ (__umpire_outcome = <<>>) /\ (__umpire_facts = <<>>)

Client_Cancel == ((((((phase) = ("Phase::Scheduled"))) \/ (((phase) = ("Phase::Started"))))) /\ (("Phase::Canceled") = ("Phase::Canceled")) /\ (("Phase::Canceled" # phase)) /\ (phase' = "Phase::Canceled") /\ (__umpire_action' = "Client::Cancel") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((((phase) = ("Phase::Scheduled"))) \/ (((phase) = ("Phase::Started")))))) /\ (TRUE))) /\ ((phase # phase)) /\ (phase' = phase) /\ (__umpire_action' = "Client::Cancel") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Worker_Poll == ((((phase) = ("Phase::Scheduled"))) /\ (("Phase::Started") = ("Phase::Started")) /\ (("Phase::Started" # phase)) /\ (phase' = "Phase::Started") /\ (__umpire_action' = "Worker::Poll") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((phase) = ("Phase::Scheduled")))) /\ (TRUE))) /\ ((phase # phase)) /\ (phase' = phase) /\ (__umpire_action' = "Worker::Poll") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Worker_RespondCompleted == ((((phase) = ("Phase::Started"))) /\ (("Phase::Completed") = ("Phase::Completed")) /\ (("Phase::Completed" # phase)) /\ (phase' = "Phase::Completed") /\ (__umpire_action' = "Worker::RespondCompleted") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((phase) = ("Phase::Started")))) /\ (TRUE))) /\ ((phase # phase)) /\ (phase' = phase) /\ (__umpire_action' = "Worker::RespondCompleted") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Worker_RespondFailed == ((((phase) = ("Phase::Started"))) /\ (("Phase::Failed") = ("Phase::Failed")) /\ (("Phase::Failed" # phase)) /\ (phase' = "Phase::Failed") /\ (__umpire_action' = "Worker::RespondFailed") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((phase) = ("Phase::Started")))) /\ (TRUE))) /\ ((phase # phase)) /\ (phase' = phase) /\ (__umpire_action' = "Worker::RespondFailed") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Deadline_ScheduleToStart == ((((phase) = ("Phase::Scheduled"))) /\ (("Phase::TimedOut") = ("Phase::TimedOut")) /\ (("Phase::TimedOut" # phase)) /\ (phase' = "Phase::TimedOut") /\ (__umpire_action' = "Deadline::ScheduleToStart") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((phase) = ("Phase::Scheduled")))) /\ (TRUE))) /\ ((phase # phase)) /\ (phase' = phase) /\ (__umpire_action' = "Deadline::ScheduleToStart") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
Deadline_StartToClose == ((((phase) = ("Phase::Started"))) /\ (("Phase::TimedOut") = ("Phase::TimedOut")) /\ (("Phase::TimedOut" # phase)) /\ (phase' = "Phase::TimedOut") /\ (__umpire_action' = "Deadline::StartToClose") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>)) \/ ((((~(((phase) = ("Phase::Started")))) /\ (TRUE))) /\ ((phase # phase)) /\ (phase' = phase) /\ (__umpire_action' = "Deadline::StartToClose") /\ (__umpire_choice' = 0) /\ (__umpire_outcome' = <<>>) /\ (__umpire_facts' = <<>>))
ModelNext == (Client_Cancel) \/ (Worker_Poll) \/ (Worker_RespondCompleted) \/ (Worker_RespondFailed) \/ (Deadline_ScheduleToStart) \/ (Deadline_StartToClose)
ModelEdge == __umpire_edge_tick' # __umpire_edge_tick
Edge == ModelNext /\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\ UNCHANGED <<__umpire_pending, __umpire_steps>>
Complete == ~ENABLED ModelNext /\ UNCHANGED vars
Next == Edge \/ Complete
Spec == Init /\ [][Next]_vars

Post_terminal_remains == [][~(ModelEdge) \/ (((~(((((((((phase) = ("Phase::Completed"))) \/ (((phase) = ("Phase::Failed"))))) \/ (((phase) = ("Phase::Canceled"))))) \/ (((phase) = ("Phase::TimedOut")))))) \/ (((phase') = (phase)))))]_vars
====
