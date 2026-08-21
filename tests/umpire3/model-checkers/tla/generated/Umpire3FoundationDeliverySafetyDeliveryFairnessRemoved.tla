---- MODULE Umpire3FoundationDeliverySafetyDeliveryFairnessRemoved ----
EXTENDS TLC

\* target: foundation-delivery-safety
\* property: entity.progress
\* semantic-hash: sha256:042870198d991baf3dd7ea94241ef69206665aabaffc38aa4404aecd6fd40b93
\* canonical-model: Umpire3.Temporal.System.TaskDeliveryProgress.mutatedBehavior

VARIABLES
    \* @type: Str;
    phase

\* @type: <<Str>>;
vars == <<phase>>

States == {"completed", "ready", "unavailable"}

Init == phase = "unavailable"

ProgressEntity ==
    FALSE

RecoverOwner ==
    /\ phase = "unavailable"
       /\ phase' = "ready"

Next ==
    \/ RecoverOwner

TypeOK == phase \in States

ResponsiveRecoverOwner == [](phase \in {"unavailable"} => <> (phase \notin {"unavailable"}))

Spec == Init /\ [][Next]_vars /\ ResponsiveRecoverOwner

Progress == [](phase \in {"ready", "unavailable"} => <> (phase \in {"completed"}))

====
