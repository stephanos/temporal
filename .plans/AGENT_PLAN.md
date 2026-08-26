# Agent Workflow: project-agnostic, evidence-qualified agent execution

**Plan date:** 2026-08-23

**Status:** first production slice implemented; extended qualification remains

**Implementation baseline:** `tools/agentworkflow`

### Implemented on 2026-08-23

The first usable system now exists in `tools/agentworkflow`:

- the root Go package exposes high-level run, resume, inspect, diff, and apply operations plus the
  provider-neutral backend seam;
- process, store, workspace, project, quality, and workflow policy are owned by deep internal
  packages;
- private Codex and Claude adapters implement their documented non-interactive structured-output
  and explicit-session contracts behind the public backend seam;
- `.spec/agentworkflow.yaml` is the only human configuration format and supports arbitrary
  command-based checks, environment allowlists, protected paths, assurance policy, named monorepo
  targets, and the guarded seven-stage workflow recipe;
- source is copied into isolated base/candidate/reviewer workspaces; read-only phases are hashed
  before and after invocation; qualified promotion is explicit, transactional, per-path
  source-drift checked, and rollback-safe after post-write failures;
- the canonical discovery, planning, optional high-assurance plan review/revision,
  implementation, checks, parallel reviews, bounded repairs, and final-check workflow is durable;
- immutable attempts retain bounded provider evidence and structured output with integrity digests;
  interrupted attempts are finalized during recovery, completed crash-window output is consumed
  without repeating mutation, and prototype v1 stage records remain inspectable read-only;
- backend resume identity binds the resolved executable, version, model, arguments, qualified mode,
  and capabilities; provider processes receive explicit minimal runtime/credential environments;
- public results expose run identities and content digests rather than mutable store/workspace paths,
  and retain an immutable confirmed/repaired/rejected/unresolved finding ledger;
- the CLI implements `init`, `doctor`, `run`, `resume`, `inspect`, `report`, `diff`, `apply`, and
  `config explain`;
- deterministic tests cover the public engine, both real adapters with fake executables, the full
  CLI journey, process limits/cancellation/environment isolation, transactional workspace
  promotion/rollback and drift races, checks, strict YAML, every stage control, custom prompts,
  v1/v2 store integrity, locking, crash recovery, resume, source drift, corruption, finding
  dispositions, and review concurrency;
  and
- the suite is race-clean under `go test -race` and remains credential-free by default.

The implementation deliberately reports unsupported or incomplete qualification rather than
pretending the following later milestones are done: optimized Git worktree/dirty-overlay source
strategies, exhaustive filesystem fault injection and subprocess kill-point matrices, automated
retention/pruning, a versioned hidden-task quality corpus, live-provider qualification jobs, and
Windows process/job-object support. Directory-copy mode already supports Git and non-Git projects
without language-specific core behavior; the additional source modes are performance and fidelity
extensions, not a requirement for the current portable command contract.

## 1. Executive decision

Evolve `tools/agentworkflow` from a Codex-specific stage wrapper into a project-agnostic workflow
engine with interchangeable agent backends, isolated workspaces, direct project verification,
independent review, crash-consistent evidence, and a small public Go API.

The central product rule is:

> An agent's output is a proposal. A workflow succeeds only when independently executed project
> checks and required review gates accept the resulting candidate under an explicit evidence and
> resource envelope.

The engine should support source code, documentation, configuration, data, generated artifacts, and
mixed-language repositories without embedding Go-, Node-, Python-, Java-, Rust-, Bazel-, or
Temporal-specific behavior in its core. A project supplies commands and capabilities through a
versioned project contract. Automatic discovery may suggest that contract, but uncertainty remains
visible and never silently grants execution authority.

Codex CLI and Claude Code CLI should be the first two agent backends. The workflow engine owns
workspace isolation, stage ordering, persistence, verification, review policy, budgets, and final
qualification. Each backend owns only translation between the common invocation contract and its
provider's command line, sessions, events, and structured-output mechanism.

The implementation should follow the same durable principles established elsewhere in this tree:

- evidence and claims remain separate;
- identities and bounds are explicit;
- unsupported behavior fails closed;
- completed evidence is immutable;
- incomplete work is inspectable and recoverable;
- concurrency never weakens isolation; and
- public packages are deep, with small APIs and substantial private implementations.

## 2. Goals

### 2.1 Project portability

The engine must be able to operate on any project that can declare:

- a readable source workspace;
- a supported snapshot/isolation strategy;
- zero or more direct verification commands;
- required environment and network capabilities;
- writable and forbidden paths;
- expected output artifacts; and
- explicit resource bounds.

Analysis-only work should remain possible when a project has no build or test system. Mutation work
may proceed without verification only when the caller explicitly selects a non-qualifying policy;
the result must then remain `inconclusive`, never `succeeded`.

### 2.2 Agent portability

The same task and project contract should run through any backend that satisfies the workflow's
required capabilities. Switching from Codex to Claude must not change:

- source snapshot identity;
- task success criteria;
- project checks;
- workspace isolation;
- artifact retention;
- review requirements; or
- the meaning of the final result.

Provider-specific sessions, event types, model names, configuration, and raw output remain backend
details and are retained as evidence without becoming workflow semantics.

### 2.3 Result quality

For a normal change workflow, the engine should:

1. discover and summarize the project without mutation;
2. produce a requirement-linked plan;
3. implement in an isolated candidate workspace;
4. run project checks directly outside the agent;
5. obtain risk-appropriate independent reviews from fresh sessions;
6. adjudicate concrete findings;
7. perform bounded repairs;
8. rerun all affected checks from a clean verification phase; and
9. publish a complete result and candidate patch only after every required gate terminates.

### 2.4 Operational quality

Every run must be bounded, cancellable, inspectable, and recoverable. A crash, timeout, malformed
agent stream, output overflow, cleanup failure, or partial filesystem write must not produce a
successful result or erase the last valid state.

### 2.5 Test quality

The workflow engine itself must have race-clean tests, systematic state-transition coverage,
filesystem fault injection, process-boundary tests, fuzzing, multi-project fixtures, backend
conformance tests, cross-platform CI, and a separate live-agent evaluation corpus.

## 3. Non-goals

- Treating an agent message, self-review, confidence score, or provider exit code as proof that a
  project change is correct.
- Automatically executing arbitrary setup commands discovered in an untrusted repository.
- Treating `workspace-write`, a child process, or a Git worktree as a security sandbox for hostile
  code.
- Guaranteeing equivalent patches, prose, schedules, token use, or raw events across providers.
- Providing stable agent-session replay across provider versions, models, or configuration changes.
- Supporting unbounded stages, reviews, repair loops, output, artifacts, or process trees.
- Building a distributed scheduler, remote artifact service, or plugin marketplace in the first
  implementation.
- Introducing language-specific logic into the workflow engine merely to increase the number of
  automatically recognized manifests.
- Creating interfaces for every internal component. Interfaces are reserved for actual independent
  implementations or required fault-injection seams.
- Adding third-party dependencies without explicit approval and a demonstrated need.

## 4. Starting baseline (superseded)

The prototype that preceded the implemented slice provided this initial kernel:

- bounded subprocess execution with context timeout;
- separate bounded stdout and stderr capture;
- Codex JSONL event parsing;
- optional Codex structured-output schema forwarding;
- stage-level exclusive creation;
- atomic file publication with file and directory synchronization;
- domain-separated stdout and stderr digests;
- parallel review invocation with declaration-order results; and
- discovery of completed stage records.

These were the prototype gaps that drove this plan; the implemented-slice summary above records
which ones are now closed:

- command construction is hard-coded to `codex exec --json --sandbox workspace-write`;
- every stage, including reviews, receives write access;
- concurrent reviews use the same workspace;
- the parser accepts output without requiring a successful terminal turn;
- provider errors, `turn.failed`, and error events are not represented in the result vocabulary;
- partial events and stderr are discarded on subprocess error, timeout, or output overflow;
- `Resume` lists completed stages but does not resume agent sessions or recover attempts;
- a process crash can leave `running.lock` permanently and prevent retry;
- stage records are trusted without revalidating their referenced artifact digests;
- prompts, resolved command/configuration, source identity, diffs, and verification evidence are not
  retained;
- process-tree termination and cleanup are not established;
- review concurrency is unbounded;
- `StageStatus` has only `completed`;
- only four tests cover the package; and
- the current concurrent fake runner is not race-safe. As of this plan, `go test -race -cover`
  reports a race in the test double and 61.1% statement coverage.

The existing schema is a prototype format. Compatibility with
`agentworkflow.stage-result/v1` is desirable for inspection, but it must not constrain the new run,
attempt, artifact, or outcome model.

## 5. Organizing principles

### 5.1 Agent output is untrusted evidence

The engine records what an agent proposed and did. It does not let the agent set the final workflow
status. Direct checks, workspace policy, review disposition, evidence completeness, and cleanup
determine qualification.

### 5.2 One semantic workflow, multiple backends

Workflow stages use common roles and contracts. Provider adapters translate those contracts; they do
not define their own meanings for planning, implementation, review, or success.

### 5.3 Deep modules

Each package owns a meaningful invariant end to end. For example, the store owns layout, journaling,
publication, recovery, validation, and streaming reads. The engine does not coordinate individual
`Create`, `Rename`, and `Sync` calls itself.

### 5.4 Validate before allocation

Validate task, project configuration, backend capabilities, workflow bounds, source strategy, check
commands, and permissions before creating a candidate workspace or invoking an agent.

### 5.5 Minimal privilege by phase

Discovery, planning, review, adjudication, and final reporting are read-only. Only implementation
and repair receive write access, and only inside a private candidate workspace. Network, additional
writable directories, environment inheritance, and dangerous sandbox modes require explicit policy.

### 5.6 Isolation before concurrency

Parallel work is allowed only after workspace and artifact isolation are established. Read-only
reviews consume immutable candidate snapshots. Multiple writers never share a workspace.

### 5.7 Evidence survives failure

Every admitted attempt publishes bounded events, stderr, terminal classification, and cleanup state,
even when the provider fails. A final success manifest is published last.

### 5.8 Honest portability

“Any project” means the core is project-neutral and can execute an explicit project contract. It
does not mean every project is automatically discoverable, every command is safe, or every host
platform is qualified. Unsupported capabilities remain explicit.

## 6. Architecture

```text
Task + project contract + source snapshot
                    |
                    v
             public Engine API
                    |
          validate and compile request
                    |
                    v
       internal run state machine and scheduler
          |             |              |
          v             v              v
    workspace       run store       quality gates
      manager       + journal       + check runner
          |             |              |
          +-------------+--------------+
                        |
                        v
                 AgentBackend seam
                   /           \
                  v             v
             Codex adapter   Claude adapter
                  |             |
            provider CLI, raw events, opaque sessions
```

Canonical change workflow:

```text
validate
  -> snapshot source
  -> discover (read-only)
  -> plan (read-only)
  -> optional plan review (read-only)
  -> implement (private writable candidate)
  -> direct checks
  -> independent reviews (parallel read-only snapshots)
  -> adjudicate
  -> bounded repair (private writable candidate)
  -> direct checks
  -> fresh final verification
  -> publish qualified result and candidate patch
  -> bounded cleanup
```

The internal representation may be a DAG, but the first public API should expose a high-level
request rather than a generic workflow-construction language. Publishing an arbitrary stage DAG too
early would expose orchestration mechanics and make later invariants difficult to strengthen.

## 7. Public Go API

Names are provisional, but the public surface should remain approximately this small:

```go
type Engine struct { /* private */ }

type Config struct {
    Root    string
    Backend Backend
    Limits  Limits
}

type Request struct {
    Task     Task
    Project  Project
    Policy   Policy
    Workflow Workflow
}

func Open(Config) (*Engine, error)
func (engine *Engine) Run(context.Context, Request) (Result, error)
func (engine *Engine) Resume(context.Context, RunID) (Result, error)
func (engine *Engine) Inspect(context.Context, RunID) (Status, error)
func (engine *Engine) Diff(context.Context, RunID) ([]Change, error)
func (engine *Engine) Apply(context.Context, RunID) error
```

`Run` returns a structured result for every admitted run. A non-nil Go error means the request could
not be admitted or the workflow infrastructure failed in a way that prevents a complete semantic
result. When execution began, the engine returns the fullest available partial `Result` and joins
primary and cleanup errors where necessary.

The public request types contain policy and the guarded recipe, not scheduler or graph machinery:

```go
type Task struct {
    Objective       string
    SuccessCriteria []string
    Constraints     []string
    NonGoals        []string
}

type Project struct {
    Root        string
    Source      SourcePolicy
    Checks      []Check
    Environment EnvironmentPolicy
}

type Check struct {
    Name       string
    Command    []string
    Directory  string
    Timeout    time.Duration
    Required   bool
}

type Workflow struct {
    Stages []WorkflowStage
}
```

`DefaultWorkflow` returns the canonical seven-stage recipe. Callers may edit prompts and enable
controls, but admission rejects missing, invented, duplicated, or reordered stages and any implicit
apply mode.

Command values are argument arrays, not shell strings. A caller that genuinely needs a shell names
the shell and its arguments explicitly. Relative directories resolve inside the candidate snapshot
and cannot escape it.

Avoid a large functional-option surface. Required policy belongs in explicit values that can be
validated, hashed, encoded, and retained. Add an option only when it does not affect run identity or
when the resolved value is recorded explicitly.

### 7.1 Backend extension seam

The only intentionally general public implementation interface is the agent backend:

```go
type Backend interface {
    Describe(context.Context) (BackendInfo, error)
    Execute(context.Context, Invocation, EventSink) (InvocationResult, error)
}
```

`Invocation` contains an optional opaque session reference. `Execute` therefore covers fresh and
resumed execution without separate start/resume methods. `BackendInfo` declares capabilities and
resolved identity. `EventSink` applies backpressure and returns an error when the engine's event or
byte budget is exhausted.

The interface must remain provider-neutral. It must not mention Codex threads, Claude sessions,
JSONL field names, CLI flags, or model-specific reasoning controls.

### 7.2 Built-in backend packages

Provider packages expose constructors and configuration only:

```go
backend, err := codex.New(codex.Config{Command: []string{"codex"}})
backend, err := claude.New(claude.Config{Command: []string{"claude"}})
```

They return `agentworkflow.Backend`. Their event decoders, process protocols, capability probes,
argument construction, session handling, and raw artifact interpretation remain private.

### 7.3 API-surface controls

- The root package exposes no filesystem layout, lock, journal, JSONL, process, worktree, or raw
  provider-event types.
- Public results expose semantic summaries and artifact references, not mutable stores.
- No exported interface is added for a single implementation solely to simplify mocking.
- No exported field is added unless callers must author or inspect it.
- `go doc` output and an exported-symbol inventory are retained as API drift fixtures.
- Every public addition requires a usage example and a statement of why an existing deep operation
  cannot own the behavior.

## 8. Package layout and ownership

Proposed end-state layout:

```text
tools/agentworkflow/
  engine.go                 narrow public operations
  request.go                public task/project/policy values
  result.go                 public outcomes and artifact references
  backend.go                small backend extension contract

  codex/
    codex.go                constructor and provider configuration
    ...                     private Codex implementation

  claude/
    claude.go               constructor and provider configuration
    ...                     private Claude implementation

  backendtest/
    backendtest.go          one reusable adapter conformance entry point

  internal/
    run/                    compiled workflow, scheduler, attempts, cancellation
    workspace/              snapshots, candidates, promotion, cleanup
    store/                  journal, blobs, manifests, recovery, inspection
    project/                discovery and project-contract validation
    quality/                checks, reviews, findings, adjudication, repair
    process/                bounded process trees and portable termination

  cmd/agentworkflow/
    main.go                 thin CLI projection over the public Engine API
```

This layout is a destination, not a mandate to create empty packages. A package is introduced only
when it can own the complete responsibility described below.

### 8.1 `internal/run`

Owns request compilation, phase transitions, dependency scheduling, bounded concurrency, retries,
repair-loop policy, cancellation propagation, and deterministic result ordering. It consumes deep
operations from workspace, store, and quality; it does not manipulate their internal records.

### 8.2 `internal/workspace`

Owns source inspection, snapshot identity, Git worktrees, directory copies, dirty-state capture,
candidate creation, read-only review snapshots, diff production, promotion, and cleanup. It is the
only module allowed to mutate source-layout state.

### 8.3 `internal/store`

Owns the complete durable lifecycle: layout, permissions, attempts, streaming event blobs, hashes,
journal transitions, atomic publication, corruption detection, recovery, schema readers, retention,
and inspection projections. The engine submits typed transitions; it does not call atomic-write
helpers directly.

### 8.4 `internal/project`

Owns strict YAML configuration decoding, the guarded workflow recipe, manifest-only discovery,
check normalization, path resolution, capability requirements, and unsupported explanations. It
always protects `.spec`, the selected configuration, and declared instruction files. It never
executes build or test commands; quality owns execution.

### 8.5 `internal/quality`

Owns direct check execution, result classification, review requests, finding schemas, duplicate
identification, adjudication requirements, repair inputs, and final qualification. It never lets an
agent set a check result.

### 8.6 `internal/process`

Owns process launch, environment construction, stdout/stderr streaming, process-group or job-object
containment, cancellation escalation, exit classification, and descendant cleanup. Backends and
direct checks use this one process boundary instead of implementing subtly different supervision.

## 9. Canonical identities and schemas

All persistent formats are versioned. Identity uses domain-separated SHA-256 over canonical encoded
semantic values, not paths, timestamps, map iteration, process IDs, or provider-generated prose.

Required identities include:

- workflow implementation/schema version;
- normalized request;
- task contract;
- project contract;
- source snapshot;
- backend executable/version/capabilities;
- resolved provider/model and project configuration;
- prompt template and rendered prompt;
- output contract;
- stage and attempt;
- candidate source snapshot and patch;
- verification command and environment policy;
- review request and finding set; and
- final result manifest.

Do not promise that equal task identities produce equal patches. Identities establish exactly what
was requested and checked; they do not assert deterministic model generation.

### 9.1 Run lifecycle

```text
declared
  -> validated
  -> prepared
  -> running
  -> reviewing
  -> verifying
  -> committing
  -> completed

Any admitted phase may instead produce:
  cancelled | capacity-exhausted | unsupported | inconclusive |
  failed | infrastructure-failed | recoverable-interruption
```

### 9.2 Attempt lifecycle

```text
planned -> admitted -> running -> finalizing -> terminal
```

Every retry creates a new immutable attempt. A retry never overwrites a failed or interrupted
attempt. Attempt identity includes the reason and parent attempt.

### 9.3 Outcome vocabulary

Keep these conditions distinct:

- `succeeded`: every required check and review gate passed, evidence is complete, and cleanup met
  policy;
- `needs-changes`: concrete unresolved findings remain;
- `project-failed`: a direct project check failed;
- `agent-failed`: the backend terminated unsuccessfully or returned invalid output;
- `unsupported`: a required capability is unavailable;
- `inconclusive`: execution completed but available evidence cannot establish the requested claim;
- `cancelled`: caller cancellation won;
- `timed-out`: a declared phase deadline expired;
- `capacity-exhausted`: an explicit byte, event, attempt, process, or work budget stopped the run;
- `infrastructure-failed`: workspace, storage, process, or cleanup infrastructure failed;
- `recoverable-interruption`: durable state exists and a validated resume is possible; and
- `corrupt`: retained state fails integrity validation and cannot be used for a success claim.

There is no generic `verified` Boolean.

## 10. Agent backend contract

### 10.1 Capabilities

Backends report versioned capabilities such as:

- read-only execution;
- workspace-write execution;
- explicit network policy;
- structured final output;
- machine-readable events;
- resumable sessions;
- ephemeral sessions;
- image input;
- additional writable directories;
- isolated or ignored user configuration;
- token/usage reporting; and
- reliable cancellation.

Each stage declares required capabilities. The engine validates them before workspace allocation.
Optional capabilities may enrich evidence but cannot silently strengthen the result.

### 10.2 Normalized events

The common event vocabulary should be deliberately small:

- invocation started;
- session identified;
- progress item started/completed;
- command execution;
- file change;
- tool invocation;
- agent message;
- usage observation;
- invocation completed;
- invocation failed; and
- backend diagnostic.

Normalized events contain only portable fields required by workflow policy. Every backend also
retains its bounded raw stream. Unknown provider events are preserved and classified; they are not
silently discarded if they may affect terminal interpretation.

### 10.3 Terminal validation

A backend invocation succeeds only when:

- the process exits successfully;
- exactly one valid terminal outcome is observed or the provider's documented terminal mechanism
  establishes completion;
- required session identity is present;
- the final output artifact exists;
- the final output decodes strictly into the stage's typed schema;
- all required event and output streams terminate within bounds; and
- cancellation or timeout did not win concurrently.

The engine should use a provider's dedicated final-output file mechanism when available instead of
inferring the final answer from the last message event. The raw event stream remains necessary for
progress, auditing, and failure diagnosis.

### 10.4 Sessions

Session references are opaque backend values paired with backend identity. Use fresh sessions for
discovery, plan review, independent reviews, adjudication, and final verification. Resume the
implementation session only for repair when:

- the backend supports resume;
- the source candidate and workflow identities match;
- the previous attempt ended in a resumable state; and
- policy permits inherited context.

If any condition fails, start a fresh repair session with a canonical repair brief. Never use
provider “last session” lookup in a durable workflow; always bind an explicit session identity.

### 10.5 Configuration isolation

Offer two explicit modes:

- `developer`: honor documented user/project configuration and record the resolved backend identity;
- `qualified`: ignore uncontrolled user configuration, use strict explicit settings, and fail when
  required configuration cannot be isolated.

Authentication remains separate from behavioral configuration. Credentials are never written into
prompts, command arguments, event artifacts, or manifests.

### 10.6 Provider adapter conformance

`backendtest.Run(t, factory)` should run every backend through the same contract:

- capability discovery and stable identity;
- fresh execution;
- structured output;
- read-only and writable behavior;
- session resume when declared;
- malformed and unknown events;
- missing, duplicate, failed, and truncated terminal events;
- stdout/stderr and event overflow;
- timeout and cancellation;
- process crash and descendant cleanup;
- environment isolation;
- paths containing spaces and non-ASCII characters; and
- raw evidence retention.

An adapter cannot be called supported until it passes the conformance suite against a deterministic
fake executable. Live-provider smoke tests are a separate qualification layer.

## 11. Project contract

### 11.1 Source strategies

Support these source strategies behind one workspace interface:

1. `git-worktree`: preferred for a clean Git commit and candidate mutation;
2. `git-with-dirty-overlay`: preserve the exact base plus a captured dirty patch without resetting
   or modifying the original tree;
3. `directory-copy`: bounded, filtered snapshot for non-Git projects;
4. `read-only-in-place`: analysis only; and
5. `external`: caller supplies an already isolated source and cleanup policy.

Every strategy returns a canonical source inventory and explicit omissions. Symlinks, submodules,
ignored files, large files, special files, nested repositories, and filesystem boundaries require
declared handling. A path escaping the admitted source root fails before copying or execution.

### 11.2 Project configuration

The portable human contract is the versioned YAML document `.spec/agentworkflow.yaml` and its
equivalent resolved Go value. YAML is the only accepted human configuration format. Durable
checkpoints, provider events, and structured results remain JSON or JSONL machine artifacts.

The configuration contains:

- source strategy and inclusion policy;
- instruction files that must remain visible to the agent;
- direct checks and their phase/time/resource limits;
- required tools and minimum versions when known;
- environment allowlist and secret names;
- network policy;
- writable and forbidden paths;
- generated-file policy;
- expected artifacts;
- cleanup command only when it is safe and explicitly required; and
- promotion policy; and
- the exact ordered `discover`, `plan`, `implement`, `check`, `review`, `repair`, and `apply` recipe,
  with editable prompts and explicit enable controls.

Unknown fields, duplicate keys, aliases, anchors, merge keys, custom tags, non-string keys, nulls,
multiple documents, JSON syntax, invalid durations, and reordered or invented stages fail strict
decoding. The normalized configuration and exact prompts are retained in the run identity.

`.spec` is human-owned input: it stays in the source snapshot for agent reads and is always
protected from candidate writes. The selected YAML file and every declared instruction file are
also protected. Generated run output always remains in the external run store.

### 11.3 Discovery

Discovery is read-only and has two parts:

1. deterministic manifest discovery inventories files such as instruction documents, VCS state,
   toolchain manifests, lockfiles, build entry points, and existing CI definitions without running
   project code;
2. an agent produces a structured project brief from that inventory and selected source files.

Discovery may recommend check commands with evidence and confidence, but recommendations do not
become executable checks without caller policy or an explicitly selected built-in profile. Unknown
projects remain usable through explicit command arrays.

### 11.4 Direct checks

Checks run through the common bounded process supervisor, not through the agent. Each result records:

- normalized command and working directory;
- admitted environment names and secret-retention policy;
- start/finish and terminal classification;
- bounded stdout/stderr and full-stream digests;
- exit status;
- timeout, cancellation, and cleanup state;
- parsed report artifacts when supported; and
- candidate source identity before and after the check.

A check that mutates source unexpectedly fails source-integrity policy unless explicitly declared as
a generator/formatter check. Generated changes become part of the candidate and force affected
checks to rerun.

## 12. Workspace isolation and promotion

### 12.1 Phase ownership

| Phase | Workspace | Permissions |
| --- | --- | --- |
| Discovery | immutable source snapshot | read-only |
| Planning | immutable source snapshot | read-only |
| Plan review | immutable source snapshot + plan | read-only |
| Implementation | private candidate | workspace-write |
| Direct checks | candidate or check-specific clone | declared by check |
| Reviews | immutable candidate snapshots | read-only |
| Adjudication | immutable candidate + findings | read-only |
| Repair | private candidate | workspace-write |
| Final verification | immutable candidate or verification clone | read-only except declared generators |
| Promotion | original target | engine-controlled explicit mutation |

### 12.2 Promotion

Default local behavior should return a patch/artifact rather than mutate the caller's original
workspace. An explicit promotion policy may apply the qualified candidate only when:

- source identity still matches the admitted base;
- the original dirty state has not changed;
- every required gate passed;
- the candidate diff is within path and byte budgets;
- no cleanup uncertainty remains; and
- patch application can be checked before commit.

Promotion never commits, pushes, opens a pull request, or sends external messages unless a separate
caller explicitly authorizes that action.

### 12.3 Cleanup

Cleanup runs under a fresh bounded context on every post-preparation exit path. It must attempt all
owned resources and retain both primary and cleanup failures. Failure to delete a temporary
workspace does not erase a valid candidate, but it prevents an unqualified “clean completion” claim
and retains recovery instructions.

## 13. Quality workflow

### 13.1 Task contract

Every run starts from an immutable task contract:

- objective;
- success criteria;
- constraints;
- non-goals;
- allowed side effects;
- required evidence; and
- stop/escalation conditions.

The planner maps each success criterion to implementation work and at least one verification route.
An unmapped criterion blocks implementation unless policy explicitly permits an exploratory run.

### 13.2 Planning

Planning output is structured and must include:

- project understanding and cited source locations;
- assumptions and unresolved questions;
- proposed file/package changes;
- verification commands;
- failure modes;
- performance, scalability, complexity, and security tradeoffs where relevant;
- migration and compatibility impact; and
- an ordered implementation plan.

For high-risk changes, a fresh plan reviewer checks requirement coverage, architecture fit, and
verification adequacy before mutation. Low-risk workflows may omit plan review under recorded
policy.

### 13.3 Implementation

The implementer receives the accepted task contract, project brief, plan, exact candidate workspace,
check contract, and stop rules. It does not receive reviewers' hidden criteria or authority to
change the task contract.

After implementation, the engine records the candidate patch and runs direct checks. An agent may
inspect check failures during repair, but cannot rewrite or suppress their classification.

### 13.4 Review

Reviews are risk-scaled and independent:

- small changes: one general correctness review;
- ordinary changes: correctness and test-quality reviews;
- high-risk changes: correctness, failure/concurrency, security, and maintainability/compatibility
  reviews as applicable.

Each reviewer receives an immutable candidate snapshot, task contract, accepted plan, project brief,
and direct-check evidence. Reviewers start from fresh sessions and cannot mutate source.

Canonical finding fields:

```text
id
lens
severity
confidence
requirement
location
claim
evidence
reproduction
impact
proposed_fix
```

Findings without a concrete claim and evidence are advisory notes, not blocking defects. File and
line references must resolve against the candidate snapshot.

### 13.5 Adjudication

Adjudication normalizes findings into:

- confirmed;
- duplicate of another finding;
- rejected with counter-evidence;
- unsupported because required evidence is unavailable; or
- deferred under an explicit non-goal.

A high-severity finding cannot be silently dropped. Rejection requires direct evidence, an accepted
requirement interpretation, or a fresh adjudicator. The artifact retains both the original finding
and disposition.

### 13.6 Repair loop

Confirmed findings become a canonical repair brief ordered by severity and dependency. Repair is
bounded by attempts, wall time, agent invocations, tokens when observable, and candidate diff size.
Every repair invalidates affected prior checks and reviews according to explicit dependency rules.

Stop when:

- all required gates pass;
- the repair budget is exhausted;
- the same normalized finding recurs without material candidate change;
- source or tool identity drifts;
- evidence becomes incomplete;
- cancellation occurs; or
- cleanup/isolation becomes uncertain.

### 13.7 Final verification

Final verification runs from a fresh phase after the last mutation. It must not reuse an agent's
assertion that tests passed. Required direct commands rerun against the final candidate. A fresh
read-only agent may summarize the evidence, but that summary is not itself a gate.

## 14. Prompt and context design

Prompts are versioned implementation assets with semantic digests. Keep stable policy first and
dynamic task/project context last. State the goal, success criteria, constraints, evidence rules,
output contract, and stop rules once.

Do not chain complete natural-language outputs blindly between stages. Build each stage context from
canonical artifacts:

- task contract;
- project brief;
- accepted plan;
- candidate diff or selected files;
- direct-check results;
- normalized findings; and
- unresolved assumptions.

Each stage receives only relevant evidence. This reduces correlated errors, stale context, token
use, and prompt-injection surface.

Provider output schemas are derived from versioned stage-specific typed records. The engine strictly
decodes the returned JSON, rejects unknown fields, validates identifiers and bounds, and retains the
raw final message separately. Do not introduce a general public JSON-schema execution API in the
first version.

## 15. Persistence and recovery

### 15.1 Store contract

The store uses private directories and immutable attempt records. Conceptually:

```text
runs/<run-id>/
  request
  journal
  attempts/<attempt-id>/
  artifacts/<content-id>/
  result
```

This layout is private and may change by schema version. Public callers use `Inspect` and artifact
references.

### 15.2 Publication rules

- Write and sync content before publishing its reference.
- Publish attempt terminal records after all required attempt artifacts.
- Publish the final run manifest last.
- Never delete resume-critical state before the final manifest is durable.
- A terminal record is immutable.
- A retry publishes a new attempt rather than editing the prior attempt.
- Readers validate size before allocation and hashes before interpretation.
- A corrupt artifact is never skipped to manufacture a complete result.

### 15.3 Journaling

Use a bounded append-only journal or immutable segments with explicit count and byte limits. Records
contain monotonic run-local ordinals and prior-record identity. Recovery accepts only a contiguous,
valid prefix.

### 15.4 Locking and interrupted attempts

Replace permanent `running.lock` semantics with a store-owned admission mechanism and attempt
records. A crash marker is evidence of interruption, not permanent ownership. Recovery must
determine one of:

- the attempt is still owned by a live validated executor;
- the attempt ended before provider execution;
- bounded partial evidence can be finalized as interrupted;
- the backend session can be resumed under exact identity; or
- recovery is unsafe and the run is blocked with a precise reason.

PID presence alone is insufficient because PIDs can be reused. Platform-specific locking remains
inside the store/process boundary and receives cross-process tests.

### 15.5 Resume

`Resume` means continue a validated incomplete run, not list its completed stages. Inspection is a
separate operation. Resume:

1. validates run and artifact identities;
2. reconstructs the last committed phase;
3. classifies incomplete attempts;
4. revalidates source and backend capabilities;
5. resumes an explicit backend session only when allowed;
6. otherwise starts the next safe attempt from canonical artifacts; and
7. never rediscovers or reruns committed work unless policy declares it stale.

## 16. Security and data handling

### 16.1 Trust model

The default product is for trusted projects and trusted agent backends. Project build/test commands
execute project code and can be hostile. Supporting untrusted repositories requires an externally
isolated container or VM profile; ordinary agent sandbox settings are not advertised as sufficient
containment.

### 16.2 Permissions

- Default agent phases to read-only.
- Grant workspace-write only to implementation and repair candidates.
- Deny dangerous/full-host modes unless the caller supplies an externally isolated profile.
- Deny network by default where the backend can enforce it.
- Prefer additional narrow writable directories over full-host access.
- Validate every writable path against the candidate root.

### 16.3 Environment and credentials

- Build subprocess environments from an allowlist rather than inheriting the entire host environment.
- Mark secret names explicitly and never retain their values.
- Supply provider credentials only to the provider invocation that needs them.
- Do not expose provider credentials to project checks or repository-controlled setup commands.
- Retain environment names, value digests where appropriate, and whether replay requires external
  input.

### 16.4 Artifact policy

Every run declares limits and retention for prompts, source excerpts, stdout/stderr, raw provider
events, patches, generated files, and check reports. Human summaries redact secret values and
potentially sensitive payloads. Full artifacts remain private by default.

### 16.5 Provider hooks and configuration

Qualified mode must disable or explicitly inventory provider hooks, plugins, MCP servers, rules, and
user configuration that could execute code or alter behavior. A required external tool that fails
to initialize fails the invocation instead of disappearing silently.

## 17. Resource bounds and scalability

`Limits` should distinguish ownership rather than use one ambiguous maximum:

- run wall time;
- stage and check wall time;
- cleanup time;
- maximum stage attempts;
- maximum repair iterations;
- maximum concurrent read-only agents;
- maximum concurrent project checks;
- per-stream and aggregate output bytes;
- raw and normalized event count/bytes;
- prompt and final-output bytes;
- candidate file, diff, and total snapshot bytes;
- artifact and journal bytes;
- process and descendant count;
- retained findings and evidence bytes; and
- provider usage/token budget when reliably observable.

Limits are checked before allocation where possible. Streaming paths apply backpressure. Hitting a
limit produces `capacity-exhausted` with retained partial evidence; it never truncates and reports
success.

At 10x stages, reviewers, events, and source size:

- the scheduler admits a bounded worker set rather than one goroutine per requested operation;
- immutable candidate snapshots are shared or copy-on-write where safe;
- events stream to bounded artifacts rather than accumulating entirely in memory;
- results preserve declaration order independent of completion order;
- review deduplication operates on bounded structured findings;
- journal readers stream rather than decode the entire run; and
- cancellation reaches queued work, active agents, checks, and process descendants.

## 18. CLI

Add a thin CLI only after the public engine operations stabilize:

```text
agentworkflow init --project <path> [--config <file.yaml>]
agentworkflow config explain --project <path> [--config <file.yaml>]
agentworkflow doctor --project <path> --backend codex
agentworkflow run --project <path> --task-file <file> [--config <file.yaml>] --backend codex
agentworkflow resume <run-id>
agentworkflow inspect <run-id> [--json]
agentworkflow report <run-id> [--json]
agentworkflow diff <run-id>
agentworkflow apply <run-id>
```

The CLI does not implement workflow policy, parse provider streams, inspect storage layout, or infer
success from strings. Human and JSON output are projections of the same public result values.

Use stable exit categories for admitted success, needs changes/project failure, unsupported,
cancelled/timeout/capacity, and infrastructure/corruption. Exact numeric codes are chosen and tested
when the CLI is introduced.

## 19. Error and failure semantics

### 19.1 Preflight failures

Invalid task, YAML configuration, paths, bounds, backend capabilities, source state, or check commands fail
before candidate allocation. Errors identify the exact field and required capability.

### 19.2 Agent failures

Provider authentication, unavailable model, malformed stream, missing terminal event, invalid
structured output, provider refusal, timeout, cancellation, output overflow, and process crash have
distinct stable classifications. All bounded partial evidence is retained.

Retry only failures explicitly classified transient and only under a bounded policy. Never retry a
mutation attempt blindly against the same workspace after an unknown provider termination.

### 19.3 Project failures

A nonzero check exit is a project result, not workflow infrastructure failure. A failure to launch
the check, enforce its bound, or clean its process tree is infrastructure failure.

### 19.4 Review failures

One unavailable required review makes the review gate incomplete. Optional review failure is
recorded as an omission. Review output that does not satisfy the finding schema is invalid agent
output, not an empty finding set.

### 19.5 Source drift

Unexpected change to the admitted base, project configuration, backend identity, or required tool identity
stops promotion and resume. The engine may offer a new run; it must not silently rebase the old one.

### 19.6 Cleanup failures

Cleanup always runs independently. Primary and cleanup failures are both preserved. A semantic
candidate can remain inspectable when cleanup fails, but the run is not fully successful under a
policy requiring clean resource release.

## 20. Testing strategy

Testing is part of each milestone, not a final hardening phase. Deterministic fake backends and
helper subprocesses run on every pull request. Live provider tests are isolated, budgeted
qualification jobs and never the only proof of engine correctness.

### 20.1 Unit and table tests

Cover every finite contract and boundary:

- empty, relative, root, symlinked, nonexistent, file, and valid workspace roots;
- run, stage, attempt, backend, finding, and artifact identifiers;
- zero, negative, exact-boundary, overflow, and conflicting limits;
- strict YAML configuration plus request/result decoding and unknown fields;
- canonical encoding and domain-separated digest goldens;
- every valid and invalid run/attempt state transition;
- capability satisfaction and precise unsupported reasons;
- dependency ordering, cycles, skipped dependents, retry policy, and deterministic output order;
- event streams with blank lines, large records, malformed JSON, unknown types, duplicate starts,
  missing sessions, multiple messages, successful completion, terminal failure, error events,
  truncated terminal records, and trailing events;
- output at limit-1, limit, and limit+1 for every stream;
- strict structured-output decode, missing fields, unknown fields, wrong types, and oversized values;
- finding normalization, invalid locations, duplicate findings, and adjudication rules;
- check classification for every exit, signal, timeout, cancellation, and capacity outcome; and
- artifact validation for wrong schema, identity, digest, size, path, and ownership.

Prefer testing public behavior and deep internal operations. Do not expose implementation details
solely to make a shallow unit test possible.

### 20.2 Process contract tests

Build a Go helper executable from the test binary or a small internal fixture. It should emulate
provider and project processes without shell dependencies and support scripted modes:

- valid event/final-output sequences;
- stdout/stderr interleaving;
- partial writes and long records;
- nonzero exit before and after output;
- close without terminal event;
- ignore graceful cancellation;
- spawn descendants;
- write after cancellation;
- exceed each byte/event limit;
- block forever for watchdog testing; and
- deliberately mutate forbidden paths.

These tests exercise the real process supervisor and filesystem while remaining deterministic and
credential-free.

### 20.3 Concurrency tests

Every pull request runs `go test -race`. Required cases include:

- review groups at concurrency 1, 2, 10, and at the configured maximum;
- bounded scheduling of more work than available workers;
- simultaneous same-stage admission from two engines;
- independent runs sharing one store;
- cancellation during queueing, provider execution, check execution, finalization, and cleanup;
- concurrent event and stderr streaming;
- reviewer result ordering independent of completion order;
- no mutation shared between review snapshots;
- no goroutine or process leaks; and
- repeated execution under randomized host scheduling.

The test fakes themselves must pass the race detector. Add synchronization or channel-driven fakes
rather than accepting races in test-only code.

### 20.4 Filesystem fault and crash-consistency tests

Place a narrow private filesystem mutation seam inside the store and workspace modules. Inject a
failure at every create, chmod, write, sync, close, rename, directory-sync, link, copy, and removal
boundary.

For every injected failure, assert that reopening yields exactly one of:

- the prior committed state;
- the new committed state; or
- a precisely classified recoverable interruption.

It must never yield a published success with missing evidence, a stage borrowed from another run,
or a permanently blocked lock with no recovery explanation.

Add subprocess kill tests at each durable lifecycle transition. Resume and recover must be
idempotent.

### 20.5 Fuzz and property tests

Continuously fuzz:

- provider event decoders;
- strict JSON schemas and manifests;
- journal recovery;
- artifact inventories;
- path normalization and containment;
- snapshot inventories and symlink handling;
- finding and review decoding; and
- output-limit accounting.

Properties include:

- decode never allocates beyond declared limits;
- open never accepts a digest mismatch;
- no admitted path escapes its root;
- no successful result exists without all required terminal evidence;
- encode/decode preserves semantic identity;
- scheduler results are independent of worker completion order;
- committed attempts are never mutated;
- recovery is idempotent; and
- raising concurrency does not change the semantic result ordering.

Seed the fuzz corpus with every historical malformed or crash artifact.

### 20.6 Backend conformance tests

Run `backendtest.Run` against:

- deterministic fake Codex CLI;
- deterministic fake Claude CLI;
- the real Codex adapter with the provider executable replaced by the fake;
- the real Claude adapter with the provider executable replaced by the fake; and
- any external backend submitted later.

Golden raw-event fixtures from each supported CLI schema verify backward reading. Capability probes
must reject unsupported or unknown versions unless an explicitly tested compatibility path exists.

### 20.7 Multi-project fixtures

Maintain small, hermetic repositories for:

- Go module;
- Node package;
- Python package;
- Rust crate;
- Java or Gradle project;
- documentation-only project;
- mixed-language monorepo;
- non-Git directory;
- Git repository with staged, unstaged, untracked, ignored, and submodule state;
- filenames containing spaces and Unicode;
- bounded symlink trees and rejected escaping symlinks; and
- generated files that legitimately change during verification.

The default PR suite uses deterministic fake agents to propose known patches. It proves that project
commands, workspaces, diffs, reviews, repair, promotion policy, and artifacts behave identically
across project types.

### 20.8 Quality evaluation corpus

Engine correctness and agent quality are separate test products. Build a versioned evaluation
corpus containing representative tasks with hidden acceptance checks and seeded defects:

- localized bug fix;
- cross-package API change;
- concurrency race or cancellation bug;
- error-handling omission;
- missing negative test;
- security-sensitive path validation;
- generated-file drift;
- documentation-only correction;
- refactor with behavioral invariance; and
- intentionally impossible or underspecified task.

Measure:

- task success rate under hidden checks;
- introduced regression rate;
- required-check pass rate;
- review precision and recall against seeded findings;
- severity calibration;
- repair convergence and recurrence;
- unrelated-diff rate;
- unsupported/inconclusive correctness;
- wall time, provider invocations, tokens where available, and artifact bytes; and
- cleanup and recovery success.

Compare workflow, prompt, model, and backend changes on the same corpus. Lower latency or token use is
an improvement only when required quality does not regress. A provider is swappable at the contract
level even when quality metrics differ; reports must expose those differences.

### 20.9 Live-provider qualification

Live tests are tagged and opt-in. They use small immutable fixtures, strict spend/invocation bounds,
no secrets in project environments, and no external side effects.

For each supported provider/version profile, qualify:

- read-only discovery;
- one structured planning result;
- one isolated patch;
- one resumed repair when supported;
- one independent review;
- timeout/cancellation behavior; and
- final-output/event compatibility.

Record the provider executable, version, resolved model/profile, source fixture, prompt schema,
limits, and observed result. A live green run supplements but does not replace deterministic adapter
tests.

### 20.10 Cross-platform testing

Qualify at least Linux and macOS before claiming general local support. Add Windows only after the
process, filesystem-sync, path, worktree, cancellation, and job-object contracts have direct tests.

Platform reports distinguish supported, unsupported, behavior failure, and infrastructure failure.
Do not infer Windows support from Go compilation alone.

### 20.11 Coverage and mutation expectations

- Require race-clean tests for every change.
- Target at least 90% statement coverage for the root contract and deep state/store/process modules.
- Require 100% enumerated transition and outcome coverage for finite state machines.
- Retain coverage by package so a shallow adapter cannot hide weak core coverage.
- Add mutation tests for terminal-event validation, digest checks, path containment, review gating,
  check enforcement, and recovery publication order.
- Coverage percentage is diagnostic; race, fault, fuzz, mutation, and fixture evidence are the
  qualification gates.

## 21. Verification commands

The nested module should gain a dedicated repository target. At minimum:

```text
cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./...
cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep -race ./...
cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./...
```

Add bounded fuzz, live-provider, cross-platform, and evaluation jobs separately so ordinary unit
tests remain deterministic and fast. Repository-level verification continues to include formatting,
`make lint-code`, and the relevant parent targets.

Tests must not depend on `time.Sleep`. Use channels, controlled clocks, explicit process signals, and
eventual assertions compatible with the repository's lint rules.

## 22. Delivery milestones

Milestones are ordered. Later work may prototype early, but it cannot be declared complete before
the earlier trust boundary is established.

### M0 — Correct the current stage runner

**Goal:** make the existing package an honest, race-clean bounded Codex stage executor before
generalization.

**Work**

- Make all test doubles concurrency-safe and add `-race` as a required target.
- Parse and require the documented Codex terminal lifecycle.
- Classify error events, failed turns, malformed output, timeout, cancellation, and capacity.
- Persist bounded raw events and stderr for every admitted attempt, including failures.
- Capture final output through the dedicated final-output path and decode it strictly.
- Replace one permanent stage lock with immutable attempt records and recoverable admission.
- Revalidate stored artifact sizes and digests during inspection.
- Bound review concurrency.
- Make reviews read-only and prevent concurrent mutation of one workspace.
- Supervise and clean complete process trees.
- Split inspection from true resume semantics.

**Tests**

- Complete current helper/unit boundary cases.
- Add scripted subprocess, race, cancellation, overflow, malformed-terminal, crash-marker, and digest
  corruption tests.
- Raise core coverage without excluding error paths.

**Exit gate**

All deterministic tests, `go vet`, and `go test -race` pass. No failed or interrupted invocation can
publish `completed`, and every admitted failure retains inspectable evidence.

### M1 — Establish versioned contracts and the deep store

**Goal:** define the durable run/attempt/result model before adding workflows or providers.

**Work**

- Add strict versioned request, attempt, event, artifact, journal, and result schemas.
- Introduce the store as one deep module owning publication, validation, inspection, and recovery.
- Add domain-separated canonical identities and bounded readers.
- Preserve inspection of v1 stage records through a compatibility reader or an explicit migration
  tool; new writers emit only the current schema.
- Define outcome and failure vocabularies.

**Tests**

- Golden schemas and canonical identities.
- Filesystem mutation matrix and subprocess crash points.
- Corruption, truncation, wrong-run, wrong-stage, oversized, and historical-reader fixtures.
- Fuzz journal and artifact open.

**Exit gate**

Every injected store failure leaves a prior valid, new valid, or precisely recoverable state. A
reader never trusts a terminal record without all bound artifacts.

### M2 — Extract the backend seam and migrate Codex

**Goal:** make provider behavior replaceable without weakening the workflow contract.

**Work**

- Add the minimal `Backend` interface, capabilities, invocation, normalized events, opaque sessions,
  and event sink.
- Move all Codex flags, version probing, raw decoding, final-output handling, and resume behavior
  behind `codex.New`.
- Support developer and qualified configuration modes.
- Add `backendtest` with a deterministic provider executable.
- Keep the root engine free of Codex names and event fields.

**Tests**

- Run the full backend conformance suite against the Codex adapter.
- Retain golden fixtures for supported Codex event versions.
- Reject unknown capability/version combinations before project mutation.

**Exit gate**

The root engine contains no Codex-specific command construction or event parsing. The Codex adapter
passes the full deterministic conformance suite and a bounded live smoke run.

### M3 — Add project contracts and workspace isolation

**Goal:** support explicit workflows for arbitrary projects without touching the caller's source.

**Work**

- Implement strict YAML project configuration, guarded stage recipes, and manifest-only discovery.
- Implement Git worktree, dirty overlay, bounded directory-copy, read-only in-place, and external
  source strategies in evidence-driven order.
- Add candidate diffs, path policies, immutable review snapshots, promotion preflight, and bounded
  cleanup.
- Add the common direct-check runner.
- Record source and candidate identities at every mutation boundary.

**Tests**

- Multi-project fixture matrix.
- Dirty-tree preservation and source-drift tests.
- Symlink, nested repository, submodule, ignored-file, large-file, Unicode, and forbidden-path tests.
- Check mutation and process-tree cleanup tests.

**Exit gate**

Go, Node, Python, Rust, documentation, mixed, dirty-Git, and non-Git fixtures run through the same
public API. No test mutates its original workspace before explicit promotion.

### M4 — Implement the evidence-qualified quality workflow

**Goal:** produce consistently reviewable results rather than merely completed agent turns.

**Work**

- Add structured task, project brief, plan, candidate, check, finding, adjudication, repair, and
  final-result artifacts.
- Compile the canonical change workflow internally.
- Add risk-scaled plan review and code review policies.
- Add read-only parallel reviews, finding validation/deduplication, and bounded adjudication.
- Add resumed or fresh repair with invalidation of affected evidence.
- Add fresh final verification and qualified result publication.

**Tests**

- Seed known good/bad candidates and review findings.
- Prove high-severity findings cannot disappear without disposition.
- Prove mutation invalidates affected checks/reviews.
- Prove repeated finding and repair budgets stop deterministically.
- Run deterministic end-to-end fake-agent workflows across the fixture projects.

**Exit gate**

No agent can set final success. A run succeeds only through direct required checks, resolved review
gates, final candidate identity, and complete cleanup evidence.

### M5 — Add the Claude backend and prove swapability

**Goal:** demonstrate that the backend seam is real rather than a renamed Codex wrapper.

**Work**

- Implement Claude capability/version probing, command construction, event normalization,
  structured output, sessions, cancellation, and raw evidence retention.
- Map unsupported features precisely rather than emulating them unsafely.
- Add backend selection to the thin CLI.
- Run the same project/workflow evaluation corpus through Codex and Claude profiles.

**Tests**

- Full deterministic backend conformance suite.
- Golden event/failure fixtures for supported Claude versions.
- Cross-backend workflow tests requiring identical gates and artifact semantics.
- Bounded live smoke and evaluation runs.

**Exit gate**

The same normalized request runs through Codex and Claude without engine changes. Differences appear
only in backend identity, raw evidence, agent proposal, resource use, and measured quality—not in
the meaning of workflow success.

### M6 — Complete resume, recovery, inspection, and CLI operation

**Goal:** make long-running workflows dependable for local and CI use.

**Work**

- Implement validated run resume and explicit provider-session resume.
- Add recover, report, and backend doctor operations.
- Add artifact retention/secret policy and bounded pruning design before automatic deletion.
- Add stable CLI JSON and human projections.
- Add CI examples that separate provider credentials from repository-controlled project checks.

**Tests**

- Crash and recovery at every phase and publication boundary.
- Resume after provider, check, review, repair, and cleanup interruption.
- CLI exit/status golden tests.
- Secret non-retention and environment separation tests.

**Exit gate**

Every interrupted run is inspectable and is either safely resumable, terminally classified, or
blocked with a precise integrity reason. Resume never repeats committed work silently.

### M7 — Qualify quality, scale, and platforms

**Goal:** establish that the tool improves real outcomes and remains reliable at expected scale.

**Work**

- Establish the versioned quality evaluation corpus and baselines.
- Add mutation testing for core trust boundaries.
- Run 10x event, review, check, artifact, source, and process tests.
- Qualify Linux and macOS; design Windows support only from direct evidence.
- Publish operational and backend-author documentation.
- Add API drift and compatibility policy.

**Exit gate**

The engine meets its correctness, race, crash, fuzz, mutation, fixture, performance, cleanup, and
platform gates. Codex and Claude each have current qualified profiles. Quality comparisons report
task success, regression, review, convergence, time, and resource evidence without collapsing them
into one marketing score.

## 23. Migration strategy

1. Preserve the current `ExecuteStage` and `ExecuteReviewGroup` temporarily as compatibility
   wrappers over the new attempt engine.
2. Mark their limitations explicitly: no project qualification and no generic backend guarantee.
3. Introduce new public `Run`, `Resume`, and `Inspect` operations beside them.
4. Add a reader for existing `agentworkflow.stage-result/v1` records.
5. Migrate the Gomad prototype or its successor to the new high-level API as the first real client.
6. Delete compatibility operations only after all callers migrate and retained v1 artifacts remain
   inspectable through the store.

Do not preserve behavior that publishes completion without terminal and verification evidence.
Schema readability is compatible; false success semantics are not.

## 24. Tradeoffs

### 24.1 Complexity

The design adds durable state, workspace management, provider adapters, and quality phases. That
complexity is justified only because each deep module removes a set of invariants from every caller.
Avoid a general plugin framework, arbitrary public DAG, or remote scheduler until the local deep
operations prove stable.

### 24.2 Portability

Explicit project commands provide broader portability than language-specific automation, but require
some configuration. Best-effort discovery improves ergonomics without becoming authority.

### 24.3 Provider abstraction

A lowest-common-denominator interface would hide useful features. Capability negotiation lets a
workflow require structured output, resume, or configuration isolation without embedding provider
types. Provider extensions may enrich evidence, but cannot alter qualification semantics.

### 24.4 Quality versus cost

Independent reviews and repair loops consume time and tokens. Risk-scaled policies keep small tasks
small. Evaluation data determines whether an additional reviewer or stronger model improves defects
found per unit cost.

### 24.5 Reproducibility

The engine can reproduce requests, source, checks, and evidence. It cannot guarantee that a model
generates the same patch again. Retaining the candidate patch and direct check results is therefore
more important than retaining only a seed or session ID.

### 24.6 Standard library versus dependencies

Staying on the standard library simplifies the nested module and supply chain. If cross-platform file
locking, JSON Schema, or process containment requires a dependency, first demonstrate that a small,
reviewed dependency is safer than an incomplete local implementation and obtain explicit approval.

## 25. Main failure modes

- **Correlated agent error:** planner, implementer, and reviewer repeat the same misconception.
  Mitigate with fresh review sessions, direct checks, hidden evaluation cases, and distinct review
  lenses.
- **Self-certified success:** the agent says tests passed when they did not run. Direct check
  execution and raw evidence own the claim.
- **Workspace interference:** concurrent agents overwrite changes. Only one writer owns a candidate;
  reviewers use immutable read-only snapshots.
- **Stale evidence:** a repair changes files after checks. Candidate identities invalidate affected
  evidence automatically.
- **Provider drift:** CLI events or flags change. Capability/version probes and golden conformance
  fixtures fail before mutation.
- **Project discovery error:** inferred commands are unsafe or wrong. Discovery recommends; explicit
  profile policy authorizes.
- **Crash-window false completion:** manifest exists without evidence. Store publishes terminal
  references last and validates every digest on open.
- **Leaked processes:** provider or check descendants survive cancellation. One process supervisor
  owns containment and final empty-tree checks.
- **Secret leakage:** credentials reach tests or artifacts. Environment separation and retention
  policy are tested invariants.
- **Repair loop churn:** agents oscillate without progress. Normalized finding recurrence, candidate
  identity, and hard budgets stop the loop.
- **Over-generalized API:** internal mechanics become permanent public contracts. API budget and
  high-level requests keep the DAG, storage, and event protocols private.
- **10x overload:** output, reviewers, or source copies exhaust memory/disk. Streaming, backpressure,
  bounded workers, copy-on-write strategies, and explicit capacity outcomes contain growth.

## 26. Definition of done

The agent workflow tool is ready for broad project use only when all of the following have current
evidence.

### Public design

- The root package exposes only high-level run, resume, and inspect operations plus one small backend
  interface.
- Storage, processes, workspaces, provider protocols, and workflow DAGs remain private.
- Codex and Claude adapters depend on the root contract; the root does not depend on either adapter.
- Public symbols have examples, compatibility tests, and a reviewed reason to exist.

### Project support

- Explicit YAML project configurations can represent arbitrary command-based checks without
  language-specific core changes.
- Git, dirty Git, non-Git, documentation, mixed-language, and generated-file fixtures pass.
- Unknown capabilities fail before mutation with useful explanations.
- The original workspace remains unchanged unless explicit qualified promotion succeeds.

### Result quality

- Task criteria map to plan and verification evidence.
- Direct checks, not agent assertions, determine project status.
- Reviews are independent, structured, immutable, and risk-scaled.
- Findings have retained dispositions and high-severity findings cannot disappear silently.
- Repairs invalidate stale evidence and stop under hard bounds.
- Final verification runs after the last mutation.

### Durability and safety

- Every admitted attempt retains bounded evidence, including failures.
- Final success is published last and is impossible with missing or corrupt artifacts.
- Crash recovery is idempotent across every durable transition.
- Resume is identity-bound and does not repeat committed work silently.
- Cancellation terminates provider/check process trees and cleanup runs independently.
- Secrets and uncontrolled configuration do not enter qualified artifacts.

### Testing

- All ordinary tests pass with `-count=1` and `-tags test_dep`.
- `go test -race` is clean.
- Core state/store/process packages meet the coverage target with complete enumerated transition
  coverage.
- Filesystem mutation, subprocess crash, fuzz, property, backend conformance, multi-project fixture,
  and mutation suites pass.
- Live Codex and Claude qualification profiles pass their bounded smoke contracts.
- The quality corpus reports stable baselines and catches seeded engine, adapter, reviewer, and
  project mutations.
- Linux and macOS have direct qualification evidence; every other platform is labeled accurately.

### Operational honesty

- Results distinguish succeeded, needs changes, project failure, agent failure, unsupported,
  inconclusive, cancellation, timeout, capacity, infrastructure failure, interruption, and
  corruption.
- Every result exposes backend, source, project, prompt, bounds, checks, reviews, omissions, cleanup,
  and artifact identities.
- No green result depends only on agent prose, cached output, an authored status Boolean, a missing
  review, or an unavailable check.

## 27. Source anchors

- Current engine and storage prototype: `tools/agentworkflow/agentworkflow.go`.
- Current tests: `tools/agentworkflow/agentworkflow_test.go`.
- Nested module: `tools/agentworkflow/go.mod`.
- Repository test hook and Gomad prototype target: `Makefile` (`gomad-test`).
- Evidence, trust, observation, and result vocabulary precedent: `.plans/UMPIRE3.md`.
- Developer-facing deep-facade precedent: `.plans/UMPIRE.md`.
- Durable store, bounded journal, recovery, and identity precedent:
  `.plans/GOMAD3_NEXT_PRODUCTIONIZATION.md`.
- Fail-closed compatibility and capability-reporting precedent:
  `.plans/GOMAD3_NEXT_COMPATIBILITY.md`.
- Bounded exploration, replay, and honest coverage precedent:
  `.plans/GOMAD3_NEXT_BUG_FINDING.md`.

These documents provide architectural precedent. The agent workflow remains a generic tool and must
not import Umpire or Gomad implementation packages merely to reuse their vocabulary.
