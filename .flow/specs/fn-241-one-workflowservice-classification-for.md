# One WorkflowService classification for all Temporal features (model, temporal)

## Conversation Evidence

> user (2026-10-23): "why is `fn classify(method: WorkflowServiceMethod) -> RpcRole {` in temporal/model/temporal/feature/activity/api.rs ? it's not activity specific"
> assistant: it does two jobs in one exhaustive match. Eight arms map activity RPCs to the activity Model's actions or mark them observed; about 40 out-of-scope groups classify the rest of WorkflowService by resource family, worded "outside this activity Model". The API build scans `model/temporal` for `fn classify` over `WorkflowServiceMethod` and checks the union against the descriptor, so a new upstream RPC fails the build. A second feature would have to copy the whole table. Recommendation: split it into one service table plus a per-feature claim list, after fn-228.4.
> user: "yes file as todo"

## Goal & Context

`model/temporal/feature/activity/api.rs` holds `classify`, an exhaustive match over all of `WorkflowServiceMethod`. Only eight arms are activity-specific: `PollActivityTaskQueue`, `RespondActivityTaskCompleted`, `RespondActivityTaskFailed`, `RespondActivityTaskCanceled`, `StartActivityExecution`, `DescribeActivityExecution`, `PollActivityExecution` and `RequestCancelActivityExecution`. The rest mark RPCs out of scope with reasons that describe the RPC's resource family (namespace administration, workflow execution, schedules, deployments, worker infrastructure), not the activity Model. The API build (`model/temporal/api/build.rs` through `umpire_catalog_build::coverage::check` and `umpire_catalog_syntax::services::classifications`) parses every such `classify` under `model/temporal` to prove every RPC is accounted for and to generate action-to-RPC bindings. With a second Temporal feature (workflow, Nexus, schedule) each would copy the full table, the out-of-scope coverage would be produced twice, and generated bindings keyed on each feature's action enum would meet an untested collision path.

## Architecture & Data Models

- **Service table.** One declaration in `model/temporal` beside the API crate (not under `feature/`) classifies every `WorkflowService` RPC once by resource family. It owns the exhaustiveness check: a new upstream RPC fails the build until it is assigned a family.
- **Feature claims.** Each feature declares only the RPCs it binds, with its own Model actions or an observed note. The activity feature claims the eight RPCs above.
- **Out of scope.** An RPC no feature claims is out of scope for that feature, with the reason derived from its family; reasons stop naming a particular Model.
- **Generation.** The catalog build reads the service table plus every feature's claims; two features claiming the same RPC is a build error unless the claim format later grows an explicit sharing rule (not in this spec).
- Nothing Temporal-specific moves into `model/umpire`; the generic catalog syntax change stays system-agnostic (service name and method enum remain parameters).

## API Contracts

- The generated `services.rs` dispatch and binding output for the activity feature is unchanged in meaning; byte changes are limited to out-of-scope reason text.
- `annotate` in `feature/activity/api_transport.rs` keeps its behaviour: out-of-scope RPCs return original bytes with a reason, claimed RPCs are validated.

## Edge Cases & Constraints

- Behavioral, wire and semantic goldens stay byte-identical; only out-of-scope reason strings and generated reason text may change, each listed in the handoff.
- The existing negative controls in `model/temporal/api/tests/services.rs` (a new RPC and a new Model action are refused) keep passing in their new form; add one for a duplicate claim and one for an unclassified RPC in the service table.
- Guest bytes may change; refresh the committed guest and rerun the slice/gaps proof if they do.
- Runs after fn-228.4 (which edits the same Temporal realization files) and before fn-228.5.

## Acceptance Criteria

- **R1:** One service table in `model/temporal` outside `feature/` classifies every `WorkflowService` RPC by resource family and owns the exhaustiveness check; the activity feature declares only its eight claimed RPCs. Errors: an unclassified RPC or a duplicate feature claim fails the build with a named diagnostic.
- **R2:** Generated dispatch and bindings, `annotate` behaviour and all goldens are unchanged except listed reason text; out-of-scope reasons no longer name the activity Model. Gates: default and `UMPIRE_BACKENDS=require make model-check`, plus slice/gaps proof if guest bytes change.

## Boundaries

- No new feature, Model or Model semantics change.
- No change to which RPCs the activity Model binds.
- No sharing rule for two features claiming one RPC.

## Decision Context

2026-10-23, owner asked why a service-wide table lives in the activity feature; the assistant recommended the split above and the owner asked to file it as todo, scheduled after fn-228.4.
