---- MODULE Umpire3FoundationDeliverySafetySound ----
EXTENDS TLC

\* target: foundation-delivery-safety
\* property: entity.progress
\* semantic-hash: sha256:042870198d991baf3dd7ea94241ef69206665aabaffc38aa4404aecd6fd40b93
\* canonical-model: Umpire3.Temporal.System.TaskDeliveryProgress.behavior

VARIABLES
    \* @type: Str;
    phase

\* @type: <<Str>>;
vars == <<phase>>

States == {"completed", "ready", "unavailable"}

Init == phase = "unavailable"

ProgressEntity ==
    /\ phase = "ready"
       /\ phase' = "completed"

RecoverOwner ==
    /\ phase = "unavailable"
       /\ phase' = "ready"

Next ==
    \/ RecoverOwner
    \/ ProgressEntity

TypeOK == phase \in States

ResponsiveRecoverOwner == [](phase \in {"unavailable"} => <> (phase \notin {"unavailable"}))

ResponsiveProgressEntity == [](phase \in {"ready"} => <> (phase \notin {"ready"}))

Spec == Init /\ [][Next]_vars /\ ResponsiveRecoverOwner /\ ResponsiveProgressEntity

Progress == [](phase \in {"ready", "unavailable"} => <> (phase \in {"completed"}))

====
