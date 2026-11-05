//! Finite worker inputs and operational roles for the independent activity graph.

use super::{Action, Realization, activity};
use activity::{Phase, Worker};
use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{AnyMessage as _, Evidence, prost_types::Any, protocol::Refusal};

/// An executor-visible key before a canonical policy name is stamped.
#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct TaskKey {
    /// Actual task attempt, reset on each delivery.
    pub attempt: u8,
    /// Successful correlated heartbeat replies for this owned task, in order.
    pub heartbeat_history: Vec<bool>,
}

/// Five finite typed input values, shared by raw and SDK executors.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Choice {
    /// The existing binary/plain completion payload.
    Complete,
    /// The existing retryable application failure.
    Retry,
    /// The existing permanent application failure.
    Fail,
    /// Heartbeat with absent details, preserving SDK round-trip semantics.
    Heartbeat,
    /// Cancellation acknowledgment with absent details.
    Canceled,
}

impl Choice {
    /// Exact typed Model request action for this task.
    pub fn action(self, attempt: u8) -> Worker {
        match self {
            Self::Complete => Worker::RespondCompleted,
            Self::Retry => Worker::RespondFailed { retryable: true },
            Self::Fail => Worker::RespondFailed { retryable: false },
            Self::Heartbeat => Worker::Heartbeat,
            Self::Canceled => Worker::RespondCanceled(attempt),
        }
    }

    /// Materializes through the registry's existing typed Fill capability.
    ///
    /// # Errors
    /// Preserves identity, input and World admission refusals from the registry.
    pub fn fill(
        self,
        world: &activity::World,
        start: &umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest,
        key: &TaskKey,
    ) -> Result<Any, Refusal> {
        if self == Self::Canceled && !key.heartbeat_history.contains(&true) {
            return Err(Refusal::unsupported(
                "SDK canceled leaf requires an observed true heartbeat",
            ));
        }
        let binding = Realization::default();
        let identity = umpire_runtime::capability::Identity::identify(&binding, &Any::pack(start))?;
        let action = self.action(key.attempt);
        let request = umpire_runtime::capability::Fill::fill(
            &binding,
            &Action::from(action),
            world,
            &Realization::program_inputs(action, key.attempt)?,
            &identity,
        )?;
        Ok(umpire_runtime::capability::Request::pack_request(&request))
    }
}

/// Operational meaning of an actually emitted edge, independent of response rows.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum OperationalRole {
    /// Accepted matching Poll delivers the executor's actual token.
    Delivery,
    /// A stale Poll reports `FailedPrecondition`; it is no programmable response.
    StalePoll,
    /// A typed controllable request and its finite filler value.
    Response(Choice),
    /// An abstract stale acknowledgment cannot be sent with the owned actual token.
    StaleAcknowledgment,
    /// Client cancellation, outside worker control.
    Client,
    /// Abstract deadline, outside worker control.
    Environment,
}

/// Classifies every typed action without inventing edges for disabled Rust arms.
pub fn operational_role(edge: &Evidence<activity::World>) -> OperationalRole {
    match &edge.action {
        Action::Client(_) => OperationalRole::Client,
        Action::Deadline(_) => OperationalRole::Environment,
        Action::Worker(worker) => match *worker {
            Worker::Poll(_) if edge.outcome == Outcome::Accepted => OperationalRole::Delivery,
            Worker::Poll(_) => OperationalRole::StalePoll,
            Worker::RespondCompleted => OperationalRole::Response(Choice::Complete),
            Worker::RespondFailed { retryable: true } => OperationalRole::Response(Choice::Retry),
            Worker::RespondFailed { retryable: false } => OperationalRole::Response(Choice::Fail),
            Worker::Heartbeat => OperationalRole::Response(Choice::Heartbeat),
            Worker::RespondCanceled(expected) if expected != edge.source.attempt => {
                OperationalRole::StaleAcknowledgment
            }
            Worker::RespondCanceled(_) => OperationalRole::Response(Choice::Canceled),
        },
    }
}

/// Current actual task ownership; hidden Model fields never enter this history.
#[derive(Clone, Debug, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct History {
    /// Absent before delivery or after the owned task ends.
    pub task: Option<TaskKey>,
}

impl History {
    /// Projects a checked edge into actual executor memory.
    ///
    /// # Errors
    /// Refuses a successful heartbeat without the matching owned actual task.
    pub fn advance(&self, edge: &Evidence<activity::World>) -> Result<Self, Refusal> {
        let mut next = self.clone();
        if edge.outcome != Outcome::Accepted {
            return Ok(next);
        }
        match operational_role(edge) {
            OperationalRole::Delivery => {
                next.task =
                    Some(TaskKey { attempt: edge.target.attempt, heartbeat_history: vec![] });
            }
            OperationalRole::Response(Choice::Heartbeat) => {
                let task = next
                    .task
                    .as_mut()
                    .filter(|task| task.attempt == edge.source.attempt)
                    .ok_or_else(|| Refusal::unsupported("heartbeat has no matching owned task"))?;
                task.heartbeat_history
                    .try_reserve(1)
                    .map_err(|_| Refusal::unsupported("policy history allocation failed"))?;
                task.heartbeat_history.push(edge.target.cancel_requested);
            }
            _ => {}
        }
        if edge.target.phase != Phase::Started {
            next.task = None;
        }
        Ok(next)
    }
}

/// Declared finite policy class; graph search bounds are supplied separately.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Domain {
    /// Maximum successful heartbeats for each actual delivered task.
    pub heartbeats_per_task: usize,
}

impl Domain {
    /// Choices executable at this actual key, independent of hidden cancellation state.
    pub fn choices(self, key: &TaskKey) -> Vec<Choice> {
        let mut choices = vec![Choice::Complete, Choice::Retry, Choice::Fail];
        if key.heartbeat_history.len() < self.heartbeats_per_task {
            choices.push(Choice::Heartbeat);
        }
        if key.heartbeat_history.contains(&true) {
            choices.push(Choice::Canceled);
        }
        choices
    }
}

impl umpire_runtime::policy::Adapter<activity::World> for Domain {
    type History = History;
    type Key = TaskKey;
    type Choice = Choice;

    fn admit(&self) -> Result<(), umpire_runtime::policy::Error> {
        self.heartbeats_per_task
            .checked_add(1)
            .ok_or(umpire_runtime::policy::Error::CombinatoricsOverflow)?;
        Ok(())
    }

    fn required(&self, edge: &Evidence<activity::World>) -> bool {
        matches!(operational_role(edge), OperationalRole::Response(_))
    }

    fn initial(&self, _: &activity::Model) -> Result<History, umpire_runtime::policy::Error> {
        Ok(History::default())
    }

    fn advance(
        &self,
        history: &History,
        edge: &Evidence<activity::World>,
    ) -> Result<History, umpire_runtime::policy::Error> {
        history.advance(edge).map_err(|error| umpire_runtime::policy::Error::Adapter(error.message))
    }

    fn key(&self, history: &History) -> Option<TaskKey> {
        history.task.clone()
    }

    fn role(
        &self,
        history: &History,
        edge: &Evidence<activity::World>,
    ) -> Result<umpire_runtime::policy::Role<Choice>, umpire_runtime::policy::Error> {
        use umpire_runtime::policy::Role;
        let choice = match operational_role(edge) {
            OperationalRole::Delivery
            | OperationalRole::StalePoll
            | OperationalRole::Client
            | OperationalRole::Environment => return Ok(Role::Autonomous),
            OperationalRole::StaleAcknowledgment => {
                return Ok(Role::Unrepresentable {
                    choices: vec![],
                    reason: format!(
                        "abstract {} cannot be sent with the actual owned token for attempt {}",
                        edge.action, edge.source.attempt
                    ),
                });
            }
            OperationalRole::Response(choice) => choice,
        };
        let Some(key) = &history.task else {
            return Ok(Role::Unrepresentable {
                choices: vec![choice],
                reason: "worker response has no owned actual task".into(),
            });
        };
        if choice == Choice::Heartbeat && key.heartbeat_history.len() >= self.heartbeats_per_task {
            return Ok(Role::OutsideHorizon(vec![Choice::Heartbeat]));
        }
        if choice == Choice::Canceled && !key.heartbeat_history.contains(&true) {
            return Ok(Role::Unrepresentable {
                choices: vec![Choice::Canceled],
                reason: "SDK canceled leaf requires an earlier actual true heartbeat; Model pending cancellation is hidden".into(),
            });
        }
        Ok(Role::Required(vec![choice]))
    }
}

/// One independent full-state checked-step inventory, before policy filtering.
pub struct Inventory {
    /// Every reached full state in canonical breadth-first order.
    pub states: Vec<activity::Model>,
    /// Every emitted action/outcome/fact/choice, including accepted stays and rejections.
    pub edges: Vec<Evidence<activity::World>>,
}

impl Inventory {
    /// Exhausts the Model's emitted graph under explicit traversal bounds.
    ///
    /// # Errors
    /// Refuses catalog/step errors, allocation or incomplete depth/work search.
    pub fn derive(world: &activity::World, depth: usize, work: usize) -> Result<Self, Refusal> {
        Realization::heartbeat_world(world, 1)?;
        let catalog = world
            .checked_catalog(umpire_dsl::CATALOG_LIMIT)
            .map_err(|error| Refusal::unsupported(format!("policy inventory catalog: {error}")))?;
        let mut search = umpire_checker::search::Traversal::<_, ()>::new(depth, work);
        let mut states = Vec::new();
        let mut edges = Vec::new();
        for state in world.starts() {
            if !states.contains(&state) {
                states.try_reserve(1).map_err(|_| allocation())?;
                search.root(state.clone()).map_err(search_limit)?;
                states.push(state);
            }
        }
        while let Some(parent) = search.next() {
            let source = search.label(parent).position.clone();
            for action in catalog.actions() {
                search.charge().map_err(search_limit)?;
                let steps = world.checked_step(&source, action.clone()).map_err(|error| {
                    Refusal::unsupported(format!("policy inventory step: {error}"))
                })?;
                for (choice, (outcome, facts, target)) in steps.into_iter().enumerate() {
                    if !states.contains(&target) {
                        states.try_reserve(1).map_err(|_| allocation())?;
                        search.push(parent, target.clone(), Some(())).map_err(search_limit)?;
                        states.push(target.clone());
                    }
                    edges.try_reserve(1).map_err(|_| allocation())?;
                    edges.push(Evidence {
                        source: source.clone(),
                        action: action.clone(),
                        choice,
                        outcome,
                        facts,
                        target,
                    });
                }
            }
        }
        Ok(Self { states, edges })
    }
}

fn allocation() -> Refusal {
    Refusal::unsupported("policy inventory allocation failed")
}

fn search_limit(error: umpire_checker::search::SearchLimit) -> Refusal {
    Refusal::unsupported(format!("policy inventory {error}"))
}

/// A checked canonical policy and its independently replayed heartbeat domains.
pub struct NamedPolicy {
    name: String,
    policy: umpire_runtime::policy::Policy<TaskKey, Choice>,
    successors: std::collections::BTreeMap<TaskKey, Vec<bool>>,
}

impl NamedPolicy {
    /// Stable structural encoding of reached observable keys and typed finite inputs.
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Exact reachable executable rows.
    pub fn rows(&self) -> &[(TaskKey, Choice)] {
        &self.policy.rows
    }

    /// Policy-conditioned successful heartbeat bits, derived before row validation.
    pub fn successors(&self) -> &std::collections::BTreeMap<TaskKey, Vec<bool>> {
        &self.successors
    }
}

/// Prepared issuance source; production switches only after consumer qualification.
pub struct Manifest {
    world: activity::World,
    enumeration: umpire_runtime::policy::Enumeration<activity::World, TaskKey, Choice>,
    policies: Vec<NamedPolicy>,
}

impl Manifest {
    /// Derives the complete production class from the explicit retry cap, without deadlines.
    pub(crate) fn production(
        start: &umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest,
    ) -> Result<Self, Refusal> {
        let maximum = start.retry_policy.as_ref().map_or(0, |policy| policy.maximum_attempts);
        let maximum = u8::try_from(maximum)
            .ok()
            .filter(|maximum| (1..=3).contains(maximum))
            .ok_or_else(|| {
                Refusal::unsupported("retry_policy.maximum_attempts must be explicitly in 1..=3")
            })?;
        Self::derive(
            &activity::World::new(maximum, None, None, None, None),
            Domain { heartbeats_per_task: 1 },
            umpire_runtime::policy::Limits {
                depth: 32,
                work: 300_000_000,
                assignments: 150_000,
                catalog: umpire_dsl::CATALOG_LIMIT,
            },
        )
    }

    /// The exact World retained at derivation.
    pub(crate) fn world(&self) -> &activity::World {
        &self.world
    }

    /// Selects over the entire canonical manifest in structural order.
    ///
    /// # Errors
    /// Refuses an empty manifest or an index outside the supported integer range.
    pub fn selected(&self, seed: u64) -> Result<&NamedPolicy, Refusal> {
        let count =
            u64::try_from(self.policies.len()).ok().filter(|count| *count != 0).ok_or_else(
                || Refusal::unsupported("canonical policy manifest is empty or oversized"),
            )?;
        let index = usize::try_from(seed % count)
            .map_err(|_| Refusal::unsupported("canonical policy index is out of range"))?;
        Ok(&self.policies[index])
    }

    /// Admits an actual immutable selected header before installing its receipt.
    pub(crate) fn admit_name(&self, name: &str) -> Result<(), Refusal> {
        if self.policies.iter().any(|policy| policy.name == name) {
            Ok(())
        } else {
            Err(Refusal::unsupported("unknown canonical worker policy"))
        }
    }

    /// Issues all canonical policies and checks the complete existing Reset ABI envelope.
    pub(crate) fn issued(
        &self,
        start: &umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest,
    ) -> Result<Any, Refusal> {
        use prost::Message as _;
        let (expected_policies, rows, continuations) =
            production_counts(self.world.maximum_attempts)?;
        if self.policies.len() != expected_policies {
            return Err(Refusal::unsupported(
                "production manifest differs from independent finite-class count",
            ));
        }
        let activity_type_bytes = start.activity_type.as_ref().map_or(0, |kind| kind.name.len());
        let repeated_type_bytes = rows
            .checked_add(continuations)
            .and_then(|keys| keys.checked_mul(activity_type_bytes))
            .ok_or_else(|| Refusal::unsupported("production encoded-size lower bound overflow"))?;
        if repeated_type_bytes > 1 << 20 {
            return Err(Refusal::unsupported(format!(
                "complete policy Reset exceeds ABI before wire assembly: repeated_activity_type_bytes={repeated_type_bytes} rows={rows} domains={continuations} limit={}",
                1 << 20
            )));
        }
        let program = self.wire_program(start, AssemblyLimits { rows, continuations })?;
        if program.rules.len() != rows || program.continuation_domains.len() != continuations {
            return Err(Refusal::unsupported(
                "production rows/domains differ from independent finite-class count",
            ));
        }
        let packed = Any::pack(&program);
        let envelope = umpire_runtime::protocol::Response {
            outcome: Some(umpire_runtime::protocol::response::Outcome::Reset(
                umpire_runtime::protocol::ResetResponse::current(Some(packed.clone())),
            )),
        };
        let bytes = envelope.encoded_len();
        if bytes > 1 << 20 {
            return Err(Refusal::unsupported(format!(
                "complete policy Reset exceeds ABI: policies={} rows={} domains={} program_bytes={} envelope_bytes={bytes} limit={}",
                self.policies.len(),
                program.rules.len(),
                program.continuation_domains.len(),
                program.encoded_len(),
                1 << 20
            )));
        }
        Ok(packed)
    }

    /// Enumerates the finite declared class and requires independent complete coverage.
    ///
    /// # Errors
    /// Refuses finite search/enumeration exhaustion, missing required decisions or malformed
    /// policy-conditioned continuations; no positive partial manifest is returned.
    pub fn derive(
        world: &activity::World,
        domain: Domain,
        limits: umpire_runtime::policy::Limits,
    ) -> Result<Self, Refusal> {
        Realization::heartbeat_world(world, 1)?;
        let enumeration =
            umpire_runtime::policy::enumerate(world, &domain, limits).map_err(engine_error)?;
        let coverage =
            enumeration.coverage(enumeration.policies(), limits.work).map_err(engine_error)?;
        if !coverage.complete() {
            let mut details = Vec::new();
            details.try_reserve(coverage.missing.len()).map_err(|_| allocation())?;
            for id in &coverage.missing {
                let edge = &enumeration.inventory()[*id].edge;
                let limitations: Vec<_> = enumeration
                    .unrepresentable()
                    .iter()
                    .filter(|finding| finding.edge == *id)
                    .map(|finding| format!("key={:?}: {}", finding.key, finding.reason))
                    .collect();
                details.push(format!(
                    "edge={id} source={} action={} choice={} outcome={:?} facts={:?} target={} limitations={limitations:?}",
                    edge.source, edge.action, edge.choice, edge.outcome, edge.facts, edge.target));
            }
            return Err(Refusal::unsupported(format!(
                "policy coverage missing full emitted edges: {}",
                details.join("; "),
            )));
        }
        let mut policies = Vec::new();
        policies.try_reserve(enumeration.policies().len()).map_err(|_| allocation())?;
        for policy in enumeration.policies() {
            let mut successors = std::collections::BTreeMap::<TaskKey, Vec<bool>>::new();
            for transition in enumeration.execution(policy, limits.work).map_err(engine_error)? {
                if transition.choice != Some(Choice::Heartbeat) {
                    continue;
                }
                let source = transition
                    .key
                    .ok_or_else(|| Refusal::unsupported("heartbeat source key missing"))?;
                let next = transition
                    .next_key
                    .ok_or_else(|| Refusal::unsupported("heartbeat successor key missing"))?;
                let edge = &enumeration.inventory()[transition.edge].edge;
                let bit = edge.target.cancel_requested;
                let mut expected = source.clone();
                expected.heartbeat_history.try_reserve(1).map_err(|_| allocation())?;
                expected.heartbeat_history.push(bit);
                if expected != next {
                    return Err(Refusal::unsupported(
                        "heartbeat successor differs from actual observable bit",
                    ));
                }
                let bits = successors.entry(source).or_default();
                if !bits.contains(&bit) {
                    bits.try_reserve(1).map_err(|_| allocation())?;
                    bits.push(bit);
                    bits.sort_unstable();
                }
            }
            for (key, choice) in &policy.rows {
                if *choice != Choice::Heartbeat {
                    continue;
                }
                let bits =
                    successors.get(key).filter(|bits| !bits.is_empty()).ok_or_else(|| {
                        Refusal::unsupported("heartbeat has no independently replayed successor")
                    })?;
                for bit in bits {
                    let mut child = key.clone();
                    child.heartbeat_history.try_reserve(1).map_err(|_| allocation())?;
                    child.heartbeat_history.push(*bit);
                    if policy.rows.binary_search_by(|(key, _)| key.cmp(&child)).is_err() {
                        return Err(Refusal::unsupported(
                            "actual heartbeat successor has no response row",
                        ));
                    }
                }
            }
            policies.push(NamedPolicy {
                name: canonical_name(policy)?,
                policy: policy.clone(),
                successors,
            });
        }
        Ok(Self { world: world.clone(), enumeration, policies })
    }

    /// Independently derived inventory, counts and exact capability/conflict reports.
    pub fn enumeration(
        &self,
    ) -> &umpire_runtime::policy::Enumeration<activity::World, TaskKey, Choice> {
        &self.enumeration
    }

    /// Canonical semantic identities, in engine structural order.
    pub fn policies(&self) -> &[NamedPolicy] {
        &self.policies
    }

    /// One shared exact resolver for future issuance/bootstrap/judgment integration.
    pub fn resolve(&self, name: &str, key: &TaskKey) -> Option<Choice> {
        let policy = &self.policies.iter().find(|policy| policy.name == name)?.policy;
        policy.rows.binary_search_by(|(row, _)| row.cmp(key)).ok().map(|index| policy.rows[index].1)
    }

    /// Assembles the checked manifest through registry Fill, without publishing it.
    ///
    /// # Errors
    /// Refuses crossed retry policy, missing routing fields, row/storage exhaustion or any
    /// registry filler refusal. Counts are admitted before allocating the output tables.
    pub fn wire_program(
        &self,
        start: &umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest,
        limits: AssemblyLimits,
    ) -> Result<umpire_temporal_api::umpire::temporal::v1::WorkerProgram, Refusal> {
        use umpire_temporal_api::umpire::temporal::v1::{
            ActivityTaskKey, ActivityTaskSuccessors, Rule,
        };
        if start.retry_policy.as_ref().map(|policy| policy.maximum_attempts)
            != Some(i32::from(self.world.maximum_attempts))
        {
            return Err(Refusal::unsupported(
                "Start retry policy differs from derived policy World",
            ));
        }
        if start.namespace.is_empty() {
            return Err(Refusal::malformed("program namespace is required"));
        }
        let activity_type = start
            .activity_type
            .as_ref()
            .filter(|kind| !kind.name.is_empty())
            .ok_or_else(|| Refusal::malformed("program activity_type is required"))?;
        let task_queue = start
            .task_queue
            .as_ref()
            .filter(|queue| !queue.name.is_empty())
            .ok_or_else(|| Refusal::malformed("program task_queue is required"))?;
        let (rows, domains) =
            self.policies.iter().try_fold((0usize, 0usize), |(rows, domains), policy| {
                Ok::<_, Refusal>((
                    rows.checked_add(policy.policy.rows.len())
                        .ok_or_else(|| Refusal::unsupported("policy wire row count overflow"))?,
                    domains.checked_add(policy.successors.len()).ok_or_else(|| {
                        Refusal::unsupported("policy wire continuation count overflow")
                    })?,
                ))
            })?;
        if rows > limits.rows || domains > limits.continuations {
            return Err(Refusal::unsupported(format!(
                "policy wire storage requires {rows} rows and {domains} continuations; admitted {} and {}",
                limits.rows, limits.continuations,
            )));
        }
        let mut rules = Vec::new();
        let mut continuation_domains = Vec::new();
        rules.try_reserve(rows).map_err(|_| allocation())?;
        continuation_domains.try_reserve(domains).map_err(|_| allocation())?;
        for policy in &self.policies {
            let wire_key = |key: &TaskKey| ActivityTaskKey {
                scenario: policy.name.clone(),
                attempt: i32::from(key.attempt),
                activity_type: activity_type.name.clone(),
                heartbeat_history: key.heartbeat_history.clone(),
            };
            for (key, choice) in &policy.policy.rows {
                rules.push(Rule {
                    key: Some(Any::pack(&wire_key(key))),
                    respond: vec![choice.fill(&self.world, start, key)?],
                });
            }
            for (key, bits) in &policy.successors {
                continuation_domains.push(Any::pack(&ActivityTaskSuccessors {
                    key: Some(wire_key(key)),
                    cancel_requested: bits.clone(),
                }));
            }
        }
        Ok(crate::support::worker_program(
            start.namespace.clone(),
            task_queue.name.clone(),
            rules,
            continuation_domains,
        ))
    }
}

fn production_counts(maximum: u8) -> Result<(usize, usize, usize), Refusal> {
    let (mut policies, mut rows, mut domains) = (1usize, 0usize, 0usize);
    let overflow = || Refusal::unsupported("production finite-class count overflow");
    for _ in 0..maximum {
        let next_rows = policies
            .checked_mul(13)
            .and_then(|count| count.checked_add(26))
            .and_then(|count| rows.checked_mul(5).and_then(|rows| count.checked_add(rows)))
            .ok_or_else(overflow)?;
        let next_domains = policies
            .checked_mul(4)
            .and_then(|count| count.checked_add(8))
            .and_then(|count| domains.checked_mul(5).and_then(|domains| count.checked_add(domains)))
            .ok_or_else(overflow)?;
        policies =
            policies.checked_mul(5).and_then(|count| count.checked_add(10)).ok_or_else(overflow)?;
        rows = next_rows;
        domains = next_domains;
    }
    Ok((policies, rows, domains))
}

/// Explicit output table ceilings, independent of the policy horizon and search limits.
#[derive(Clone, Copy, Debug)]
pub struct AssemblyLimits {
    /// Maximum total response rows across the canonical manifest.
    pub rows: usize,
    /// Maximum total independent heartbeat continuation domains.
    pub continuations: usize,
}

/// A raw history policy collision caused by the executor's deliberate per-task reset.
pub struct TaskResetAlias {
    /// Complete typed checked edges witnessing both aliased task suffixes.
    pub paths: [Vec<Evidence<activity::World>>; 2],
    /// Different prior-task observations, before a successful retry and fresh delivery.
    pub prior_histories: [Vec<bool>; 2],
    /// The single actual next-task key shared by both histories.
    pub next_key: TaskKey,
    /// An example pair of incompatible raw-history assignments at that key.
    pub incompatible_choices: [Choice; 2],
}

impl Manifest {
    /// Reports exact cross-task memory collisions independently of stamped policy names.
    ///
    /// # Errors
    /// Refuses if the declared checked World cannot supply either concrete alias path.
    pub fn task_reset_aliases(&self) -> Result<Vec<TaskResetAlias>, Refusal> {
        let mut aliases = Vec::new();
        let mut state = self
            .world
            .starts()
            .into_iter()
            .next()
            .ok_or_else(|| Refusal::unsupported("policy World has no start"))?;
        for next_attempt in 2..=self.world.maximum_attempts {
            let delivery = alias_edge(&self.world, &state, Worker::Poll(next_attempt - 1))?;
            let owned = History::default().advance(&delivery)?;
            let heartbeat = alias_edge(&self.world, &delivery.target, Worker::Heartbeat)?;
            let observed = owned.advance(&heartbeat)?;
            let retry = Worker::RespondFailed { retryable: true };
            let direct_retry = alias_edge(&self.world, &delivery.target, retry)?;
            let heartbeat_retry = alias_edge(&self.world, &heartbeat.target, retry)?;
            let direct_delivery =
                alias_edge(&self.world, &direct_retry.target, Worker::Poll(next_attempt))?;
            let heartbeat_delivery =
                alias_edge(&self.world, &heartbeat_retry.target, Worker::Poll(next_attempt))?;
            let left = owned.advance(&direct_retry)?.advance(&direct_delivery)?.task;
            let right = observed.advance(&heartbeat_retry)?.advance(&heartbeat_delivery)?.task;
            if left != right {
                return Err(Refusal::unsupported(
                    "retry delivery did not share an exact observable key",
                ));
            }
            let next_key =
                left.ok_or_else(|| Refusal::unsupported("retry delivery has no owned task"))?;
            aliases.try_reserve(1).map_err(|_| allocation())?;
            state = direct_retry.target.clone();
            let second_delivery =
                alias_edge(&self.world, &delivery.source, Worker::Poll(next_attempt - 1))?;
            aliases.push(TaskResetAlias {
                paths: [
                    vec![delivery, direct_retry, direct_delivery],
                    vec![second_delivery, heartbeat, heartbeat_retry, heartbeat_delivery],
                ],
                prior_histories: [vec![], vec![false]],
                next_key,
                incompatible_choices: [Choice::Complete, Choice::Fail],
            });
        }
        Ok(aliases)
    }
}

fn alias_edge(
    world: &activity::World,
    source: &activity::Model,
    worker: Worker,
) -> Result<Evidence<activity::World>, Refusal> {
    let action = Action::from(worker);
    let (choice, (outcome, facts, target)) = world
        .checked_step(source, action.clone())
        .map_err(|error| Refusal::unsupported(format!("task-reset alias Model step: {error}")))?
        .into_iter()
        .enumerate()
        .find(|(_, (outcome, _, _))| *outcome == Outcome::Accepted)
        .ok_or_else(|| Refusal::unsupported("task-reset alias path is disabled"))?;
    Ok(Evidence { source: source.clone(), action, choice, outcome, facts, target })
}

fn canonical_name(
    policy: &umpire_runtime::policy::Policy<TaskKey, Choice>,
) -> Result<String, Refusal> {
    let mut name = String::from("activity-v1");
    for (key, choice) in &policy.rows {
        let count = key
            .heartbeat_history
            .len()
            .checked_add(7)
            .ok_or_else(|| Refusal::unsupported("policy name size overflow"))?;
        name.try_reserve(count).map_err(|_| allocation())?;
        name.push('_');
        name.push(char::from(b'0' + key.attempt));
        name.push('h');
        for bit in &key.heartbeat_history {
            name.push(if *bit { '1' } else { '0' });
        }
        name.push('=');
        name.push(match choice {
            Choice::Complete => 'c',
            Choice::Retry => 'r',
            Choice::Fail => 'f',
            Choice::Heartbeat => 'h',
            Choice::Canceled => 'a',
        });
    }
    Ok(name)
}

#[expect(clippy::needless_pass_by_value, reason = "map_err transfers the owned engine refusal")]
fn engine_error(error: umpire_runtime::policy::Error) -> Refusal {
    Refusal::unsupported(format!("activity policy {error}"))
}
