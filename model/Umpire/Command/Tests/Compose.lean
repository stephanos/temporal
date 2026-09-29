import Umpire.Command

/-!
# The `compose` command

A job and the agent that serves it, composed into one Model. The job replies once while pending,
may be poked first, expires on a timer, and stutters on a halt; the agent halts, resumes, and serves
while running. The composition synchronizes the two halts and the reply with the agent's `serve`, so
a reply is a step only while the agent runs. Both members' outcomes are `accepted`, and the
generated union keeps them apart.

The same composition is declared three times: as written, with its `members:` and `sync:` lines
reordered, and with its state structure's fields swapped. Each declares one `verify` Query under
the same Definition IDs -- each namespace hangs its IDs off the same root -- so their Behavior
Fingerprints compare the Models and nothing else. The other fixtures are the located errors the
command reports, and one composition that never enables a synchronized action.
-/

namespace Umpire.Command.Tests.Compose

open Umpire Umpire.Command

model_conventions root "umpire" under Umpire.Command.Tests.Compose

/-! ### Resolving a member-qualified reference against `sync:` groups

`resolveReference` is pure data over a composition's recorded `sync:` groups: the ambiguous case
needs a pair that is two groups' participant, which no shipped composition writes, so it is pinned
directly here rather than through a compiled `compose` command. -/

#guard Compose.resolveReference #[("reply", #[("job", "reply"), ("agent", "serve")])]
    ["job", "reply"] ["ok"] == .key "reply-ok"
#guard Compose.resolveReference #[("reply", #[("job", "reply"), ("agent", "serve")])]
    ["job", "poke"] ["ok"] == .key "job_poke-ok"
#guard Compose.resolveReference
    #[("first", #[("job", "halt")]), ("second", #[("job", "halt")])] ["job", "halt"] [] ==
  .ambiguous ["first", "second"]

/-! ### The members -/

namespace Job

entity job
  key: jobId

enum Reply
  | ok
  | error

enum JobPhase
  | pending
  | done
  | failed

/-- The flag comes first, so a job's phase is not its key's first segment: a Scenario's
`job.pending` is matched against every field its state holds. -/
structure JobState where
  poked : Bool
  phase : JobPhase
  deriving BEq, DecidableEq, Repr, Finite

enum JobOutcome
  | accepted

enum JobFact
  | replied
  | expired

action reply
  party: agent
  on: job
  input:
    result: Reply

action halt
  party: agent

action poke
  party: agent
  on: job
  input:
    result: Reply

/-- A pending job replies with the result it is given. -/
def replyStep (state : JobState) (result : Reply) : List (Step JobState JobOutcome JobFact) :=
  if state.phase != .pending then [] else
  match result with
  | .ok => [{ outcome := .accepted, state := { state with phase := .done }, facts := [.replied] }]
  | .error =>
      [{ outcome := .accepted, state := { state with phase := .failed }, facts := [.replied] }]

/-- A halt the job does not feel. -/
def haltStep (state : JobState) : List (Step JobState JobOutcome JobFact) :=
  [{ outcome := .accepted, state, facts := [] }]

/-- A pending job is poked, whatever the result. -/
def pokeStep (state : JobState) (_ : Reply) : List (Step JobState JobOutcome JobFact) :=
  if state.phase != .pending then [] else
  [{ outcome := .accepted, state := { state with poked := true }, facts := [] }]

/-- A pending job expires. -/
def expireStep (state : JobState) : List (Step JobState JobOutcome JobFact) :=
  if state.phase != .pending then [] else
  [{ outcome := .accepted, state := { state with phase := .failed }, facts := [.expired] }]

machine jobMachine
  for: job
  state: JobState
  starts: [pending]
  ends: [done, failed]
  timers: [expire]
  unobservable: [expire]
  steps:
    reply: replyStep
    halt: haltStep
    poke: pokeStep
    expire: expireStep

end Job

namespace Agent

entity agent
  key: queue

enum AgentPhase
  | running
  | halted

structure AgentState where
  phase : AgentPhase
  deriving BEq, DecidableEq, Repr, Finite

enum AgentOutcome
  | accepted

inductive AgentFact
  deriving BEq, DecidableEq, Repr, Finite

action halt
  party: agent

action resume
  party: agent

action serve
  party: agent
  on: agent

def haltStep (state : AgentState) : List (Step AgentState AgentOutcome AgentFact) :=
  if state.phase != .running then [] else
  [{ outcome := .accepted, state := { phase := .halted }, facts := [] }]

def resumeStep (state : AgentState) : List (Step AgentState AgentOutcome AgentFact) :=
  if state.phase != .halted then [] else
  [{ outcome := .accepted, state := { phase := .running }, facts := [] }]

def serveStep (state : AgentState) : List (Step AgentState AgentOutcome AgentFact) :=
  if state.phase != .running then [] else
  [{ outcome := .accepted, state, facts := [] }]

machine agentMachine
  for: agent
  state: AgentState
  starts: [running]
  ends: [running, halted]
  steps:
    halt: haltStep
    resume: resumeStep
    serve: serveStep

end Agent

/-! ### The composition, and a Query over it of each form -/

namespace Forward

model_conventions root "umpire" under Umpire.Command.Tests.Compose.Forward

structure PipelineState where
  job : Job.JobState
  agent : Agent.AgentState
  deriving BEq, DecidableEq, Repr

compose pipeline
  for: [Job.job, Agent.agent]
  state: PipelineState
  members:
    job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    halt: job.halt ∥ agent.halt
    reply: job.reply ∥ agent.serve
  starts: [job.pending, agent.running]
  ends: [job.done, job.failed]

limits three
  steps: 3
  actions: 3
  search: 64

/- An ok reply to an unpoked job leaves it done with the agent running: one whole composed state,
named field by field. -/
property settled
  machine: pipeline
  when: reply ok
  holds: fun step =>
    step.state.job.phase == .done && !step.state.job.poked && step.state.agent.phase == .running

scenario replied
  model: pipeline
  starts: job.pending
  actions: [reply (ok)]

query settles
  verify: settled
  in: replied
  limits: three

/- A poked job that expires while the agent is halted. -/
property expiredWhileHalted
  machine: pipeline
  when: job.expire
  holds: fun step =>
    step.state.job.phase == .failed && step.state.job.poked && step.state.agent.phase == .halted

scenario outage
  model: pipeline
  starts: job.pending
  actions: [job.poke (ok), halt, job.expire]

query expires
  find: expiredWhileHalted
  in: outage
  limits: three

/- A classed action named bare is a claim about each of its classes: every reply, whichever
result, records that the job replied. -/
property replies
  machine: pipeline
  when: reply
  holds: fun step => step.facts.contains (.job .replied)

/- A member's own classed action, named bare. -/
property pokes
  machine: pipeline
  when: job.poke
  holds: fun step => step.outcome == .job .accepted

#guard replies.names.groups.map (fun group => (group.trigger, group.requirements)) ==
  [(.action "reply-error", [.factClause "reply-error-fact-job_replied" "job_replied"]),
    (.action "reply-ok", [.factClause "reply-ok-fact-job_replied" "job_replied"])]

#guard pokes.names.groups.map (·.trigger) == [.action "job_poke-error", .action "job_poke-ok"]

/- A reply never fires while the agent is halted: every reply, whichever result, leaves the agent
running while the job is done or failed, poked or not. No whole state is fixed, so the claim fixes
the agent's one field. -/
property repliesWhileRunning
  machine: pipeline
  when: reply
  holds: fun step => step.state.agent.phase == .running

#guard repliesWhileRunning.names.groups.map (fun group => (group.trigger, group.requirements)) ==
  [(.action "reply-error", [.stateFieldClause "reply-error-field-agent-running" "agent" "running"]),
    (.action "reply-ok", [.stateFieldClause "reply-ok-field-agent-running" "agent" "running"])]

scenario pokedReply
  model: pipeline
  starts: job.pending
  actions: [job.poke (ok), reply (error)]

query repliesRunning
  verify: repliesWhileRunning
  in: pokedReply
  limits: three

/- An expiry fixes the job's phase alone: the job's flag and the agent both vary. -/
property expiryFails
  machine: pipeline
  when: job.expire
  holds: fun step => step.state.job.phase == .failed

#guard expiryFails.names.groups.map (·.requirements) ==
  [[.stateFieldClause "field-job_phase-failed" "job_phase" "failed"]]

/- A claim that reads the job's flag or the agent's phase fixes neither, and the step it cannot tell
apart is named. -/
/--
error: the predicate is not a conjunction of one state, one outcome and facts at `job_expire`: the clauses it fixes cannot tell the step to false-failed_halted with outcome job_accepted and facts [job_expired] apart from the steps it accepts; a Property is one such conjunction, so split it or restate it
-/
#guard_msgs in
property pokedOrRunning
  machine: pipeline
  when: job.expire
  holds: fun step => step.state.job.poked || step.state.agent.phase == .running

/- The documented dotted spelling for a synchronized action, `job.reply (ok)`, resolves through the
`reply:` `sync:` line to the same key the bare spelling does, in both a Property `when:` and a
Scenario `actions:` -- the completion review's finding (fn-92 R3). -/
property dottedReply
  machine: pipeline
  when: job.reply (ok)
  holds: fun step =>
    step.state.job.phase == .done && !step.state.job.poked && step.state.agent.phase == .running

#guard dottedReply.names.groups.map (·.trigger) == settled.names.groups.map (·.trigger)

scenario dottedReplied
  model: pipeline
  starts: job.pending
  actions: [job.reply (ok)]

#guard dottedReplied.names.occurrences == replied.names.occurrences

/- No `query` runs `dottedReply` or `dottedReplied`: the R14 differential sweep enumerates every
declared Query, and its expected block is required to stay byte-identical (Approach, fn-92.9); the
`.names` pins above are the regression. -/

/- An unknown member-qualified reference is a located error naming the mechanical key it falls back
to, the same message shape as any other unknown action. -/
/--
error: unknown Model action 'job_bogus'; declared: agent_resume, halt, job_expire, «job_poke-error», «job_poke-ok», «reply-error», «reply-ok»
-/
#guard_msgs in
property unknownMemberReference
  machine: pipeline
  when: job.bogus
  holds: fun step => step.outcome == .job .accepted

end Forward

namespace Reordered

model_conventions root "umpire" under Umpire.Command.Tests.Compose.Reordered

structure PipelineState where
  job : Job.JobState
  agent : Agent.AgentState
  deriving BEq, DecidableEq, Repr

compose pipeline
  for: [Agent.agent, Job.job]
  state: PipelineState
  members:
    agent: Agent.agentMachine
    job: Job.jobMachine
  sync:
    reply: job.reply ∥ agent.serve
    halt: job.halt ∥ agent.halt
  starts: [agent.running, job.pending]
  ends: [job.failed, job.done]

limits three
  steps: 3
  actions: 3
  search: 64

property settled
  machine: pipeline
  when: reply ok
  holds: fun step =>
    step.state.job.phase == .done && !step.state.job.poked && step.state.agent.phase == .running

scenario replied
  model: pipeline
  starts: job.pending
  actions: [reply (ok)]

query settles
  verify: settled
  in: replied
  limits: three

end Reordered

namespace Swapped

model_conventions root "umpire" under Umpire.Command.Tests.Compose.Swapped

structure PipelineState where
  agent : Agent.AgentState
  job : Job.JobState
  deriving BEq, DecidableEq, Repr

compose pipeline
  for: [Job.job, Agent.agent]
  state: PipelineState
  members:
    job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    halt: job.halt ∥ agent.halt
    reply: job.reply ∥ agent.serve
  starts: [job.pending, agent.running]
  ends: [job.done, job.failed]

limits three
  steps: 3
  actions: 3
  search: 64

property settled
  machine: pipeline
  when: reply ok
  holds: fun step =>
    step.state.job.phase == .done && !step.state.job.poked && step.state.agent.phase == .running

scenario replied
  model: pipeline
  starts: job.pending
  actions: [reply (ok)]

query settles
  verify: settled
  in: replied
  limits: three

end Swapped

/-! A member field may spell a namespace the generated terms name: the job held in a field named
`Umpire` composes, and its agreement theorem is declared, because the view's state binders are the
command's own rather than the author's field names. -/
namespace Shadowing

model_conventions root "umpire" under Umpire.Command.Tests.Compose.Shadowing

structure PipelineState where
  Umpire : Job.JobState
  agent : Agent.AgentState
  deriving BEq, DecidableEq, Repr

compose pipeline
  for: [Job.job, Agent.agent]
  state: PipelineState
  members:
    Umpire: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    halt: Umpire.halt ∥ agent.halt
    reply: Umpire.reply ∥ agent.serve
  starts: [Umpire.pending, agent.running]
  ends: [Umpire.done, Umpire.failed]

/-- info: 'Umpire.Command.Tests.Compose.Shadowing.pipeline.agrees' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms pipeline.agrees

end Shadowing

/-! ### The composed table

Every state is keyed by the job's key and the agent's joined with `_`, and the catalog is sorted by
key rather than by the order the walk found them. The synchronized actions are keyed by their
`sync:` names, the job's own by `job_`, and the agent's resume by `agent_`. -/

open Forward in
#guard pipeline.table.states.map (·.key) == ["false-done_halted", "false-done_running",
  "false-failed_halted", "false-failed_running", "false-pending_halted", "false-pending_running",
  "true-done_halted", "true-done_running", "true-failed_halted", "true-failed_running",
  "true-pending_halted", "true-pending_running"]

open Forward in
#guard pipeline.table.actions.map (·.key) ==
  ["agent_resume", "halt", "job_expire", "job_poke-error", "job_poke-ok", "reply-error", "reply-ok"]

/- A reply is a row only where the agent runs, and a halt only where it has not halted. -/
open Forward in
#guard (pipeline.table.transitions.filter fun row =>
    row.key.endsWith "reply-ok" || row.key.endsWith "-halt").map (·.key) ==
  ["false-done_running-halt", "false-failed_running-halt", "false-pending_running-halt",
    "false-pending_running-reply-ok", "true-done_running-halt", "true-failed_running-halt",
    "true-pending_running-halt", "true-pending_running-reply-ok"]

/- Both members' outcomes are `accepted`; the catalog holds each under its member. -/
open Forward in
#guard pipeline.table.outcomes.map (·.key) == ["job_accepted", "agent_accepted"]

open Forward in
#guard pipeline.table.facts.map (·.key) == ["job_replied", "job_expired"]

/- Each member's state fields are the composed state's own: the job's two under `job_`, the agent's
one field as `agent`. -/
open Forward in
#guard pipeline.stateFieldIds.map (·.1) == ["job_poked", "job_phase", "agent"]

/- One start state, and the eight in which the job is done or failed end it. -/
open Forward in
#guard pipeline.initial.map pipeline.stateKeyFor == ["false-pending_running"] &&
  pipeline.terminal.length == 8

open Forward in
theorem pipeline_catalogs_valid :
    pipeline.table.states.Valid ∧ pipeline.table.actions.Valid ∧
      pipeline.table.outcomes.Valid ∧ pipeline.table.facts.Valid :=
  ⟨⟨by decide +kernel, by decide +kernel, by decide +kernel⟩,
    ⟨by decide +kernel, by decide +kernel, by decide +kernel⟩,
    ⟨by decide +kernel, by decide +kernel, by decide +kernel⟩,
    ⟨by decide +kernel, by decide +kernel, by decide +kernel⟩⟩

/-! ### The Behavior Fingerprint

Reordering the `members:` and `sync:` lines leaves the Model as it was; swapping the state
structure's fields reorders every composed key, as reordering a machine's fields does. -/

private def fingerprintOf {Setup State Action Outcome Fact : Type} [BEq Setup] [BEq State]
    [BEq Action] [BEq Outcome] [BEq Fact] {m : DeclaredModel Setup State Action Outcome Fact}
    (checked : Except AdmissionError (CheckedModel m)) : Option String :=
  checked.toOption.map (·.query.behaviorFingerprint.render)

#guard (fingerprintOf Forward.settles).isSome
#guard fingerprintOf Forward.settles == fingerprintOf Reordered.settles
#guard fingerprintOf Forward.settles != fingerprintOf Swapped.settles

/-! ### What the command refuses -/

namespace Dial

entity dial

/-- Spelled as the job's `Reply` is, and a different domain. -/
enum Setting
  | ok
  | error

structure DialState where
  setting : Setting
  deriving BEq, DecidableEq, Repr, Finite

enum DialOutcome
  | accepted

inductive DialFact
  deriving BEq, DecidableEq, Repr, Finite

action turn
  party: agent
  on: dial
  input:
    setting: Setting

def turnStep (_ : DialState) (setting : Setting) : List (Step DialState DialOutcome DialFact) :=
  [{ outcome := .accepted, state := { setting }, facts := [] }]

machine dialMachine
  for: dial
  state: DialState
  starts: [ok]
  ends: [ok, error]
  steps:
    turn: turnStep

end Dial

namespace Spread

entity spread

enum SpreadPhase
  | spreading

structure SpreadState where
  phase : SpreadPhase
  level : Fin 5
  deriving BEq, DecidableEq, Repr, Finite

enum SpreadOutcome
  | accepted

inductive SpreadFact
  deriving BEq, DecidableEq, Repr, Finite

action scatter
  party: agent
  on: spread

/-- Any level scatters to every level. -/
def scatterStep (state : SpreadState) : List (Step SpreadState SpreadOutcome SpreadFact) :=
  (members (α := Fin 5)).map fun level =>
    { outcome := .accepted, state := { state with level }, facts := [] }

machine spreadMachine
  for: spread
  state: SpreadState
  starts: [spreading]
  ends: [spreading]
  steps:
    scatter: scatterStep

end Spread

namespace Counter

entity counter

enum CounterPhase
  | counting

structure CounterState where
  phase : CounterPhase
  count : Fin 128
  deriving BEq, DecidableEq, Repr, Finite

enum CounterOutcome
  | accepted

inductive CounterFact
  deriving BEq, DecidableEq, Repr, Finite

def tickStep (state : CounterState) : List (Step CounterState CounterOutcome CounterFact) :=
  [{ outcome := .accepted, state := { state with count := saturatingSucc state.count }, facts := [] }]

machine counterMachine
  for: counter
  state: CounterState
  starts: [counting]
  ends: [counting]
  timers: [tick]
  unobservable: [tick]
  steps:
    tick: tickStep

end Counter

structure DialedState where
  job : Job.JobState
  dial : Dial.DialState
  deriving BEq, DecidableEq, Repr

structure UnderscoredState where
  my_job : Job.JobState
  agent : Agent.AgentState
  deriving BEq, DecidableEq, Repr

structure SpreadPair where
  front : Spread.SpreadState
  back : Spread.SpreadState
  deriving BEq, DecidableEq, Repr

structure CounterPair where
  left : Counter.CounterState
  right : Counter.CounterState
  deriving BEq, DecidableEq, Repr

/--
error: members job, agent each own an action named 'halt' and no `sync:` line names every one of them; two members' actions of one name are synchronized explicitly
-/
#guard_msgs in
compose unsynced
  for: [Job.job, Agent.agent]
  state: Forward.PipelineState
  members:
    job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    reply: job.reply ∥ agent.serve
  starts: [job.pending]
  ends: [job.done]

/--
error: 'job.reply' and 'dial.turn' range over different inputs; a synchronized action takes one class per step, so its classed participants take the same input domains
-/
#guard_msgs in
compose mismatched
  for: [Job.job, Dial.dial]
  state: DialedState
  members:
    job: Job.jobMachine
    dial: Dial.dialMachine
  sync:
    reply: job.reply ∥ dial.turn
  starts: [job.pending]
  ends: [job.done]

/--
error: 'job.expire' is a timer; a timer is `system` behaviour one member fires on its own, so no `sync:` line names it
-/
#guard_msgs in
compose timed
  for: [Job.job, Agent.agent]
  state: Forward.PipelineState
  members:
    job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    halt: job.halt ∥ agent.halt
    expire: job.expire ∥ agent.serve
  starts: [job.pending]
  ends: [job.done]

/--
error: member field 'my_job' contains '_'; a composed key joins a member's own key to its field with `_`, so a field name carries none
-/
#guard_msgs in
compose underscored
  for: [Job.job, Agent.agent]
  state: UnderscoredState
  members:
    my_job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    halt: my_job.halt ∥ agent.halt
  starts: [agent.running]
  ends: [agent.halted]

/--
error: 'scatter' multiplies its participants' results out to 25 steps; a synchronized step has at most 16 results
-/
#guard_msgs in
compose scattered
  for: [Spread.spread]
  state: SpreadPair
  members:
    front: Spread.spreadMachine
    back: Spread.spreadMachine
  sync:
    scatter: front.scatter ∥ back.scatter
  starts: [front.spreading]
  ends: [front.spreading]

/--
error: the composition reaches at least 8193 states over 2 action classes, 16386 evaluations, and the bound is 16384; compose smaller machines
-/
#guard_msgs in
compose counted
  for: [Counter.counter]
  state: CounterPair
  members:
    left: Counter.counterMachine
    right: Counter.counterMachine
  starts: [left.counting]
  ends: [left.counting]

/--
error: 'nobody.done' names no value of a member field; `starts:` and `ends:` name `field.value`, one of the member fields job, agent and a value its machine's state holds
-/
#guard_msgs in
compose unended
  for: [Job.job, Agent.agent]
  state: Forward.PipelineState
  members:
    job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    halt: job.halt ∥ agent.halt
    reply: job.reply ∥ agent.serve
  starts: [job.pending]
  ends: [nobody.done]

/--
error: 'Forward.expires' runs on the composition 'Umpire.Command.Tests.Compose.Forward.pipeline'; a composed Model is verified by `verify` Queries and no set runs over one in version one, because no realization performs several machines' actions as one Case
-/
#guard_msgs in
set pipelineTests
  purpose: functional
  bind:
    agent: driven
  queries: [Forward.expires]

/--
error: 'agent_resume' is already the key of another action of the composition; a `sync:` name keys its action, so it is spelled apart from every member's `<field>_<action>` key
-/
#guard_msgs in
compose collided
  for: [Job.job, Agent.agent]
  state: Forward.PipelineState
  members:
    job: Job.jobMachine
    agent: Agent.agentMachine
  sync:
    agent_resume: job.halt ∥ agent.halt
    reply: job.reply ∥ agent.serve
  starts: [job.pending]
  ends: [job.done]

namespace Lamp

entity lamp

enum LampPhase
  | lit_up
  | dark

structure LampState where
  phase : LampPhase
  deriving BEq, DecidableEq, Repr, Finite

enum LampOutcome
  | accepted

inductive LampFact
  deriving BEq, DecidableEq, Repr, Finite

action glow
  party: agent
  on: lamp

def glowStep (state : LampState) : List (Step LampState LampOutcome LampFact) :=
  [{ outcome := .accepted, state, facts := [] }]

machine lampMachine
  for: lamp
  state: LampState
  starts: [lit_up]
  ends: [lit_up, dark]
  steps:
    glow: glowStep

end Lamp

structure LampedState where
  job : Job.JobState
  lamp : Lamp.LampState
  deriving BEq, DecidableEq, Repr

/--
error: member 'lamp' has a state keyed 'lit_up', which contains '_'; a composed state joins its members' keys with `_`, so a member's state key carries none
-/
#guard_msgs in
compose lamped
  for: [Job.job, Lamp.lamp]
  state: LampedState
  members:
    job: Job.jobMachine
    lamp: Lamp.lampMachine
  starts: [job.pending]
  ends: [job.done]

/-! ### A member with one state

A beacon is always lit, so a claim that it is lit fixes nothing: its one field has no other value
to change to, as a machine with one outcome fixes no outcome. -/

namespace Beacon

entity beacon

enum BeaconPhase
  | lit

-- One field of one value makes `mk.injEq` a lemma simp proves on its own, which `simpNF` refuses.
set_option genInjectivity false in
structure BeaconState where
  phase : BeaconPhase
  deriving BEq, DecidableEq, Repr, Finite

enum BeaconOutcome
  | accepted

inductive BeaconFact
  deriving BEq, DecidableEq, Repr, Finite

action blink
  party: agent
  on: beacon

def blinkStep (state : BeaconState) : List (Step BeaconState BeaconOutcome BeaconFact) :=
  [{ outcome := .accepted, state, facts := [] }]

machine beaconMachine
  for: beacon
  state: BeaconState
  starts: [lit]
  ends: [lit]
  steps:
    blink: blinkStep

end Beacon

structure BeaconedState where
  job : Job.JobState
  beacon : Beacon.BeaconState
  deriving BEq, DecidableEq, Repr

compose beaconed
  for: [Job.job, Beacon.beacon]
  state: BeaconedState
  members:
    job: Job.jobMachine
    beacon: Beacon.beaconMachine
  starts: [job.pending]
  ends: [job.done, job.failed]

/--
error: the predicate holds on every step of this machine at `job_expire` and fixes no state, outcome or fact, so it claims nothing
-/
#guard_msgs in
property alwaysLit
  machine: beaconed
  when: job.expire
  holds: fun step => step.state.beacon.phase == .lit

/-! ### A synchronized action no reachable state enables

Two gates lift together and pass together; `early` would pass the front while the back is still
closed, which no reachable state is. The catalog leaves it out and says so. -/

namespace Gate

entity gate

enum GatePhase
  | closed
  | opened

structure GateState where
  phase : GatePhase
  deriving BEq, DecidableEq, Repr, Finite

enum GateOutcome
  | accepted

inductive GateFact
  deriving BEq, DecidableEq, Repr, Finite

action lift
  party: agent
  on: gate

action pass
  party: agent
  on: gate

def liftStep (state : GateState) : List (Step GateState GateOutcome GateFact) :=
  if state.phase != .closed then [] else
  [{ outcome := .accepted, state := { phase := .opened }, facts := [] }]

def passStep (state : GateState) : List (Step GateState GateOutcome GateFact) :=
  if state.phase != .opened then [] else
  [{ outcome := .accepted, state, facts := [] }]

machine gateMachine
  for: gate
  state: GateState
  starts: [closed]
  ends: [opened]
  steps:
    lift: liftStep
    pass: passStep

end Gate

structure GatesState where
  front : Gate.GateState
  back : Gate.GateState
  deriving BEq, DecidableEq, Repr

/--
warning: 'early' is never enabled: no composed state the composition reaches has a row for it, so its catalog leaves it out
-/
#guard_msgs in
compose gates
  for: [Gate.gate]
  state: GatesState
  members:
    front: Gate.gateMachine
    back: Gate.gateMachine
  sync:
    lift: front.lift ∥ back.lift
    pass: front.pass ∥ back.pass
    early: front.pass ∥ back.lift
  starts: [front.closed]
  ends: [front.opened]

#guard gates.table.actions.map (·.key) == ["lift", "pass"]
#guard gates.table.states.map (·.key) == ["closed_closed", "opened_opened"]

/-! ### A field claim where the members move together

The gates lift together while the agent moves on its own, so no state the composition reaches has
one gate open and the other closed. A claim that a lift opens the front gate still fixes the front
gate's field: the predicate is read with that field alone changed, in a state the composition never
reaches. -/

structure GatedAgentState where
  front : Gate.GateState
  back : Gate.GateState
  agent : Agent.AgentState
  deriving BEq, DecidableEq, Repr

compose gatedAgent
  for: [Gate.gate, Agent.agent]
  state: GatedAgentState
  members:
    front: Gate.gateMachine
    back: Gate.gateMachine
    agent: Agent.agentMachine
  sync:
    lift: front.lift ∥ back.lift
    pass: front.pass ∥ back.pass
  starts: [front.closed]
  ends: [front.opened]

property frontOpens
  machine: gatedAgent
  when: lift
  holds: fun step => step.state.front.phase == .opened

#guard frontOpens.names.groups.map (·.requirements) ==
  [[.stateFieldClause "field-front-opened" "front" "opened"]]

end Umpire.Command.Tests.Compose
