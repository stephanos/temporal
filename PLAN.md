# CATCH: **C**haos **A**nd **T**emporal **C**orrectness **H**arness

Property-Based Acceptance Testing for Temporal

## Vision

Transform Temporal testing from manual, incomplete functional tests to automated property-based acceptance testing that observes system behavior and validates correctness through invariants.

## Current State

| System | Cadence | Environment | Purpose | Problems |
|--------|---------|-------------|---------|----------|
| **Canary** | Every 2 min | Production | Regression & SLA | (deferred) |
| **Functional Tests** | Per PR | GitHub CI | Regression | Manual, tedious, incomplete, no fault injection, don't use real SDKs, don't find bugs |
| **Omes** | Nightly | CICD | Regression & Load | (to be leveraged) |

**Core insight**: We have excellent observability (OTEL spans) but lack systematic property verification across fault scenarios.

## Module Organization

All CATCH components are organized under `tools/catch/`:

```
tools/catch/
├── pitcher/          # Fault injection engine
│   ├── pitcher.go    # Core pitcher interface and implementation
│   └── interceptor.go # gRPC interceptors for automatic injection
├── play/             # Test scenario composition
│   ├── play.go       # Play and Pitch definitions
│   ├── library.go    # Standard play library
│   ├── executor.go   # Play execution engine
│   └── gameplan.go   # Coverage-driven test planning
├── skipper/          # Intelligent test orchestration
│   ├── skipper.go    # Play selection strategies
│   └── coverage.go   # Coverage tracking
└── umpire/           # Property validation (passive observation)
    ├── umpire.go     # Core umpire framework
    ├── entity/       # Domain entities (Workflow, TaskQueue, etc.)
    ├── event/        # OTEL event parsing
    ├── model/        # Property models (LostTask, StuckWorkflow, etc.)
    └── exporter.go   # OTEL span exporter integration
```

**Import Paths**:
- `go.temporal.io/server/tools/catch/pitcher`
- `go.temporal.io/server/tools/catch/play`
- `go.temporal.io/server/tools/catch/skipper`
- `go.temporal.io/server/tools/catch/umpire`

## Architecture Overview

### Deployment Models

CATCH supports two fundamentally different deployment models with different capabilities:

| Aspect | **Internal Deployment** | **External Deployment** |
|--------|------------------------|------------------------|
| **Context** | Inside server process (tests) | Outside server (canary) |
| **Access** | Server internals + SDK | SDK client/worker only |
| **Pitcher Intercepts** | In-code + gRPC (all services) | Client-side gRPC only |
| **Umpire Observation** | Direct OTEL span capture | Requires server OTEL export |
| **Property Scope** | All properties | External-observable only |
| **Example** | Functional tests | tools/canary |

**Critical Distinction**:
- **Internal**: Can inject faults into `matchingservice.AddWorkflowTask`, observe internal task state transitions
- **External**: Can only inject faults into SDK operations (e.g., delay activity execution), observe workflow completion/failure

### Component Compatibility Matrix

This table shows which CATCH components work in each deployment model:

| Component | Internal | External | Notes |
|-----------|----------|----------|-------|
| **Pitcher** | ✓ Full | ✓ SDK-only | Internal: all pitch types; External: SDK interceptors only |
| **Umpire** | ✓ Full | ✓ Limited | Internal: all properties; External: external-observable properties only |
| **Scorebook** | ✓ | ✓ | Works in both, but external has limited entity visibility |
| **Scout** | ✓ Direct | ✓ Via export | Internal: direct span capture; External: requires OTEL export |
| **Play System** | ✓ | ✓ | Both support composed scenarios, but external has limited pitch types |
| **Skipper** | ✓ | ✓ | Works in both contexts |

**Implementation Strategy**:
- **Phase 1-3**: Focus on internal deployment (functional tests)
  - Build server-side Pitcher with full intercept capabilities
  - Implement all properties (internal + external-observable)
  - Validate with functional test suite
- **Phase 4**: Add external deployment (canary)
  - Build SDK-level Pitcher (separate implementation)
  - Use external-observable properties only
  - Deploy as standalone service

### Two-System Design

1. **Passive System** (Phase 1 - MVP focus)
   - **Scout**: Observes OTEL telemetry from system behavior
   - **Scorebook**: In-memory SQLite storage of events and entities
   - **Rulebook**: Property definitions that specify correct behavior
   - **Umpire**: Validates observed behavior against Rulebook properties

2. **Active System** (Phase 2)
   - **Skipper**: Intelligent test orchestrator (decides what to test)
   - **Playbook**: Collection of all possible test scenarios (Plays)
   - **Game Plan**: Required coverage goals for a test run
   - **Pitcher**: Fault injection engine (executes chaos actions)
   - **Play**: Composed scenario (sequence of Pitches)
   - **Pitch**: Atomic behavior injection (latency, error, cancellation)

### Entity Hierarchy

```
Cluster
  └─ Namespace (many per cluster)
      └─ TaskQueue (many per namespace)
          └─ Workflow (many per taskqueue)
              ├─ WorkflowExecution (many per workflow)
              │   ├─ Activity (many per execution)
              │   └─ ChildWorkflow (many per execution)
              └─ WorkflowTask (many per workflow)
```

### Baseball Metaphor Glossary

| Term | Baseball Meaning | System Meaning |
|------|------------------|----------------|
| **Batter** | Faces pitches from pitcher | System Under Test (SUT) - Temporal service |
| **Pitcher** | Throws pitches; adds pressure | Fault injection engine; throws curveballs |
| **Pitch** | Single thrown ball | Atomic action (inject error, add latency, etc.) |
| **Play** | Coordinated sequence (e.g., double play) | Test scenario; sequence of Pitches |
| **Playbook** | Team's full collection of plays | All supported test scenarios |
| **Game Plan** | Plays selected for this game | Coverage requirements for this run |
| **Game** | Single baseball match | One test execution session |
| **Skipper** | Team manager; calls plays | Test orchestrator; selects scenarios |
| **Scout** | Observes and records | OTEL telemetry observer |
| **Scorebook** | Official game logbook | Event storage (SQLite) |
| **Rulebook** | Rules of baseball | Behavioral properties/invariants |
| **Umpire** | Enforces rules; judges correctness | Property validator |
| **Roster** | Team member with a role | Domain entity instance (e.g., a Namespace) |
| **Almanac** | Historical stats reference | Distinct behavior combinations storage |

---

## Phase 1: Passive Validation System (MVP)

**Goal**: Augment all existing functional tests with automated property verification - finding bugs we currently miss without changing test code.

### 1.1 Current State (✓ Already Complete)

**Status**: ✓ These components exist in `tools/umpire/`

- **Entity System**: Workflow, TaskQueue, WorkflowTask entities with FSM state tracking
- **Event System**: Parse OTLP spans into domain events
- **Scorebook**: In-memory SQLite (buntdb) for entity/event storage
- **Registry**: Routes events to entities, manages entity lifecycle
- **Umpire Framework**: Core validation engine with model registry
- **Test Integration**: FunctionalTestBase initializes Umpire, hooks OTEL exporter
- **Existing Models**:
  - `StuckWorkflowModel`: Detects workflows that start but never complete
  - `LostTaskModel`: Detects tasks stored but never successfully polled

**Evidence**: See `tools/umpire/`, `tests/testcore/functional_test_base.go:275-293`, `tests/lost_task_test.go`

### 1.2 Pitcher: Fault Injection Foundation

**Goal**: Build fault injection system that works in both local tests and future proxy deployment.

#### Architecture Decisions

**Pitcher Intercept Points**:
1. **In-Code Intercepts** (Phase 1a): Hooks directly in server codebase
   - Similar to existing test hooks pattern
   - Allows control beyond RPC/persistence boundaries
   - Example: hold requests between synchronization points

2. **Proxy Intercepts** (Phase 1b): gRPC interceptors in test proxy
   - Client & server interceptors for RPC-level control
   - Can modify deadlines, inject errors, add latency
   - Works with existing functional test local network

**Pitch Types** (Atomic Actions):

| Pitch Type | Internal | External | Description |
|------------|----------|----------|-------------|
| **RPC Pitches** | ✓ | ✓ (client-side) | Fail, delay, timeout server RPCs |
| **SDK Pitches** | ✓ | ✓ | Activity failure, workflow delay, task retry |
| **Timing Pitches** | ✓ | ✗ | Server-side: trigger deadline early, fire timer |
| **Persistence Pitches** | ✓ | ✗ | Delay write, fail transaction |
| **Coordination Pitches** | ✓ | ✗ | Hold request at synchronization point |

**Internal-Only Pitches** (requires server access):
- Fail `matchingservice.AddWorkflowTask` (server-side RPC)
- Delay persistence layer writes
- Hold workflow tasks at synchronization points
- Manipulate server-side timers

**External-Compatible Pitches** (work from SDK):
- Fail activity execution
- Delay activity/workflow task completion
- Timeout SDK operations (StartWorkflow, SignalWorkflow)
- Cancel workflows/activities
- Inject delays in worker task processing

#### 1.2.1 Milestone: Pitcher Core Framework ✓ COMPLETE

**Status**: ✓ Implemented with typed proto references and one-time matching

**Deployment Model**: Internal (functional tests) via gRPC interceptor

**Architecture**:
- Global state accessible during tests only (follows `testhooks.TestHooks` pattern)
- Configured via test cluster initialization
- **Interceptor-based**: gRPC unary/stream interceptors automatically inject faults
- No manual `Throw()` calls in service handlers needed

**Current Implementation**:
```go
// tools/catch/pitcher/pitcher.go
type Pitcher interface {
    // Throw executes a pitch (atomic fault injection action)
    // targetType is a proto message type (e.g., &matchingservice.AddWorkflowTaskRequest{})
    // If a pitch matches, it is executed once and deleted
    Throw(ctx context.Context, targetType any, request any) error

    // Configure sets pitch configuration for a target RPC method
    // targetType should be a proto message type
    // Multiple pitches for same target are matched in order
    Configure(targetType any, config PitchConfig)

    // Reset clears all configurations
    Reset()
}

// Each pitch matches ONCE then is deleted
// To match multiple times, configure multiple pitches
type PitchConfig struct {
    Action string            // "fail", "delay", "timeout"
    Match  *MatchCriteria    // Conditional matching (nil = match any)
    Params map[string]any    // Action-specific parameters
}

// Match criteria based on umpire entity fields
type MatchCriteria struct {
    WorkflowID  string  // from entity.Workflow
    NamespaceID string  // from entity.Workflow
    TaskQueue   string  // from entity.TaskQueue
    RunID       string  // from entity.WorkflowTask
    Custom      map[string]any  // for arbitrary matching
}
```

**gRPC Interceptor Pattern**:
```go
// tools/catch/pitcher/interceptor.go
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        if p := Get(); p != nil {
            if err := p.Throw(ctx, req, req); err != nil {
                return nil, err  // Fault injected
            }
        }
        return handler(ctx, req)  // Normal execution
    }
}
```

**Test Configuration** (typed proto references):
```go
// In test setup
func (s *MyTestSuite) SetupTest() {
    s.FunctionalTestBase.SetupTest()

    // Configure pitcher with typed proto message
    s.ConfigurePitcher(&matchingservice.AddWorkflowTaskRequest{}, pitcher.PitchConfig{
        Action: "fail",
        Match: nil,  // Match first request
        Params: map[string]any{"error": "RESOURCE_EXHAUSTED"},
    })

    // Multiple pitches for same target - matched in order
    s.ConfigurePitcher(&matchingservice.AddWorkflowTaskRequest{}, pitcher.PitchConfig{
        Action: "delay",
        Match: &pitcher.MatchCriteria{WorkflowID: "specific-wf"},
        Params: map[string]any{"duration": 500 * time.Millisecond},
    })
}
```

**Completed Tasks**:
- [x] Define Pitcher interface with typed proto references
- [x] Implement one-time matching (pitch deleted after use)
- [x] Add MatchCriteria based on entity fields
- [x] Implement gRPC unary/stream interceptors
- [x] Add to TemporalImpl in `tests/testcore/onebox.go`
- [x] Add ConfigurePitcher() helper to FunctionalTestBase
- [x] Remove Probability and Count fields (deterministic matching)

**Remaining Tasks**:
- [ ] Inject pitcher interceptor into test cluster automatically
- [ ] Implement request field extraction in matchesCriteria()
- [ ] Update all standard plays to use typed proto references
- [ ] Write comprehensive tests for pitcher framework
- [ ] Document pitch configuration patterns

**Success Criteria**: ✓ Can inject RPC failures via gRPC interceptor with type-safe configuration.

#### 1.2.2 Milestone: Intercept Points - In-Code ⊘ OBSOLETE

**Status**: ⊘ Replaced by gRPC interceptor approach (see 1.2.3)

**Rationale**: Manual intercept points in service handlers would require modifying many service files and maintaining consistency across codebases. The gRPC interceptor approach (1.2.3) provides automatic fault injection for all RPC methods without code changes, making it superior for both maintenance and functionality.

**Decision**: Use gRPC interceptors exclusively. In-code intercepts only needed for non-RPC operations (persistence, timers) in future phases.

#### 1.2.3 Milestone: Intercept Points - gRPC Interceptors ✓ COMPLETE

**Status**: ✓ Implemented as primary fault injection mechanism

**Rationale**: gRPC interceptors provide:
- Automatic fault injection for all RPC methods (no manual code changes)
- Type-safe proto message identification
- Cleaner separation between production and test code
- Foundation for future proxy-based deployment

**Implementation**: See `tools/catch/pitcher/interceptor.go`

**Integration with Test Cluster**:
```go
// Automatically added to all services in test cluster
// No manual configuration needed - pitcher interceptor is always active when pitcher is initialized
```

**Completed Tasks**:
- [x] Implement gRPC unary server interceptor with pitcher
- [x] Implement gRPC stream server interceptor with pitcher
- [x] Use request type for automatic target identification
- [x] Support all pitch types (fail, delay, timeout)

**Remaining Tasks**:
- [ ] Auto-inject interceptor into test cluster setup (`onebox.go`)
- [ ] Add debug logging for pitch execution
- [ ] Add metrics/counters for pitch activation
- [ ] Test interceptors work across all services
- [ ] Document pitch configuration patterns

**Success Criteria**: ✓ Can inject RPC-level failures automatically via gRPC interceptor.

### 1.3 Rulebook: MVP Property

**Goal**: Define and implement the first property that validates the MVP scenario.

#### 1.3.1 Milestone: Retry Resilience Property

**Property Definition**:
```
PROPERTY: WorkflowCompletesAfterTransientMatchingFailure

Given:
  - A workflow is started
  - The first AddWorkflowTask call to matching fails with RESOURCE_EXHAUSTED

Then:
  - The workflow task is retried
  - The workflow eventually completes successfully
  - No violations of task delivery guarantees occur

Verification:
  - Workflow entity transitions: created → started → completed
  - WorkflowTask entity shows: added → failed → added → polled → completed
  - Total completion time within reasonable bounds (< 30s)
```

**Implementation**:
```go
// tools/umpire/model/retry_resilience.go
type RetryResilienceModel struct {
    Logger    log.Logger
    Registry  EntityRegistry
    Threshold time.Duration // Max time for workflow to complete after start
}

func (m *RetryResilienceModel) Check(ctx context.Context) []Violation {
    // Query workflows that were started
    workflows := m.Registry.QueryEntities(entity.NewWorkflow())

    var violations []Violation
    for _, e := range workflows {
        wf := e.(*entity.Workflow)

        // Check if workflow completed within threshold
        if wf.FSM.Current() == "started" {
            age := time.Since(wf.StartedAt)
            if age > m.Threshold {
                violations = append(violations, Violation{
                    Model: "retry-resilience",
                    Message: "workflow did not complete after transient failure",
                    Tags: map[string]string{
                        "workflowID": wf.WorkflowID,
                        "age": age.String(),
                    },
                })
            }
        }
    }

    return violations
}
```

**Tasks**:
- [ ] Implement RetryResilienceModel
- [ ] Add to model registry
- [ ] Extend WorkflowTask entity to track retry attempts
- [ ] Add events for task retry scenarios (including pitcher-injected failures)
- [ ] Update event importer to capture failure injection events
- [ ] Add synchronization between test completion and umpire validation
- [ ] Write unit tests for model logic
- [ ] Test model with hand-crafted entity states

**Success Criteria**: Model detects violations when workflows fail to complete, passes when they succeed.

### 1.4 MVP Integration Test

**Goal**: End-to-end test demonstrating the complete passive system.

#### 1.4.1 Milestone: MVP Test Case

**Test Implementation**:
```go
// tests/pitcher_mvp_test.go
type PitcherMVPSuite struct {
    testcore.FunctionalTestBase
}

func TestPitcherMVP(t *testing.T) {
    suite.Run(t, new(PitcherMVPSuite))
}

func (s *PitcherMVPSuite) TestWorkflowCompletesAfterMatchingFailure() {
    ctx := context.Background()

    // Configure pitcher: fail first AddWorkflowTask
    s.ConfigurePitcher(pitcher.Config{
        Pitches: []pitcher.PitchConfig{
            {
                Target: "matchingservice.AddWorkflowTask",
                Action: "fail",
                Count:  1,
                Params: map[string]any{"error": "RESOURCE_EXHAUSTED"},
            },
        },
    })

    // Register and start worker
    w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
    w.RegisterWorkflow(SimpleWorkflow)
    s.NoError(w.Start())
    defer w.Stop()

    // Start a simple workflow
    workflowID := "pitcher-mvp-test"
    we, err := s.SdkClient().ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
        ID:        workflowID,
        TaskQueue: s.TaskQueue(),
    }, SimpleWorkflow)
    s.NoError(err)

    // Wait for workflow to complete
    err = we.Get(ctx, nil)
    s.NoError(err)

    // Allow time for all events to be processed
    time.Sleep(100 * time.Millisecond)

    // Verify with Umpire - should find NO violations
    violations := s.GetUmpire().Check(ctx)
    s.Empty(violations, "Expected no violations, but found: %v", violations)
}

func SimpleWorkflow(ctx workflow.Context) error {
    return nil // Just complete successfully
}
```

**Tasks**:
- [ ] Create test file and suite
- [ ] Implement simple workflow for testing
- [ ] Add pitcher configuration in test setup
- [ ] Verify workflow completes despite injected failure
- [ ] Verify umpire reports no violations
- [ ] Add assertions on entity states for deeper validation
- [ ] Document MVP test as template for future tests

**Success Criteria**:
- Test passes when fault injection + retry succeeds
- Test fails (Umpire reports violation) when retry logic is broken
- Demonstrates passive system catching bugs that functional tests miss

### 1.5 Rollout to All Functional Tests

**Goal**: Enable umpire validation on all existing functional tests without modifying test code.

#### 1.5.1 Milestone: Universal Umpire Integration

**Current State**: Umpire is already initialized in `FunctionalTestBase.SetupSuiteWithCluster` ✓

**Remaining Tasks**:
- [ ] Add `TearDownTest()` hook to check violations after each test (hard fail)
- [ ] Document how to interpret umpire violations in test failures
- [ ] Add structured violation output for CI parsing
- [ ] Create quick reference guide for common violations

**Implementation**:
```go
// tests/testcore/functional_test_base.go
func (s *FunctionalTestBase) TearDownTest() {
    // Check for umpire violations after test
    if s.umpire != nil {
        ctx := context.Background()
        violations := s.umpire.Check(ctx)

        // Report and fail on violations
        if len(violations) > 0 {
            for _, v := range violations {
                s.Logger.Error("Umpire violation detected",
                    tag.NewStringTag("model", v.Model),
                    tag.NewStringTag("message", v.Message))
            }

            // Hard fail on violations
            s.Fail(fmt.Sprintf("Umpire detected %d violation(s)", len(violations)))
        }
    }
}
```

**Success Criteria**: All functional tests run with passive monitoring, violations logged and optionally fail tests.

### 1.6 Expand Property Coverage

**Goal**: Add more properties to catch additional bug classes.

#### Properties to Implement (Priority Order)

| Property | Internal | External | Description |
|----------|----------|----------|-------------|
| **Task Delivery Guarantees** | ✓ | ✗ | Requires observing matching service internals |
| **Workflow Lifecycle Invariants** | ✓ | ✓ (partial) | External: workflow completion consistency |
| **Activity Execution** | ✓ | ✓ | Observable from SDK activity lifecycle |
| **Task Queue Consistency** | ✓ | ✗ | Requires observing task routing internals |
| **Timeout Accuracy** | ✓ | ✓ | Observable from SDK timeout behavior |
| **Retry Resilience** | ✓ | ✓ | Observable from workflow/activity completion |
| **Cancellation Propagation** | ✓ | ✓ | Observable from SDK cancellation behavior |

**1. Task Delivery Guarantees** (Internal-only)
   - Every task added to matching is eventually polled or explicitly cancelled
   - No task is polled multiple times without being re-added
   - *Requires*: Observing `AddWorkflowTask` and `PollWorkflowTaskQueue` spans

**2. Workflow Lifecycle Invariants** (Partial external)
   - Workflow cannot transition to completed without task completion (internal)
   - Started workflows must eventually complete or timeout (external-observable)
   - Workflow history is consistent with final state (external-observable)

**3. Activity Execution** (External-compatible)
   - Activity tasks follow schedule → started → complete/fail lifecycle
   - Activity timeouts trigger within expected bounds (+/- tolerance)
   - Activity retry policies are respected

**4. Task Queue Consistency** (Internal-only)
   - Task queue pollers receive tasks for their registered queue
   - Task routing respects task queue versions/builds
   - *Requires*: Observing matching service task routing

**5. Timeout Accuracy** (External-compatible)
   - Workflow execution timeout triggers within bounds
   - Activity StartToClose timeout triggers within bounds
   - ScheduleToClose timeout triggers within bounds

**6. Retry Resilience** (External-compatible)
   - Workflows complete despite transient failures
   - Activities retry according to policy
   - Backoff intervals are respected

**7. Cancellation Propagation** (External-compatible)
   - Cancelled workflows stop execution
   - Child workflows are cancelled when parent is cancelled
   - Activities are cancelled when workflow is cancelled

**8. Cross-Cluster Failover** (Internal-only, deferred to Phase 2)
   - Namespace failover transitions are monotonic
   - No dual-write scenarios during failover

**Milestone per Property**: Define, implement, test, document

**Note**: External-compatible properties will be used by both functional tests and tools/canary. Internal-only properties are available only to functional tests.

---

## Phase 2: Active Behavior Generation

**Goal**: Move from passive observation to active test scenario generation.

### 2.1 Kitchensink Workload Integration

**Leverage**: https://github.com/temporalio/omes/tree/main/workers/go/kitchensink

**Why Kitchensink**:
- Comprehensive workflow patterns (timers, activities, children, signals, updates, etc.)
- Real SDK usage (Go SDK initially)
- Battle-tested in Omes nightly runs
- Parameterizable chaos scenarios

**Integration Plan**:
- Import kitchensink as Go module dependency
- Create test harness that runs kitchensink workflows
- Configure pitcher to inject faults during execution
- Umpire validates all workflows complete correctly

### 2.2 Play Definition System

**Play**: A composed scenario = sequence of Pitches + expected outcome

**Example Play**:
```yaml
play:
  name: "MatchingFailureWithRetry"
  description: "Transient matching failure should not prevent workflow completion"
  pitches:
    - target: "matchingservice.AddWorkflowTask"
      action: "fail"
      count: 1
      error: "RESOURCE_EXHAUSTED"
  workload:
    type: "kitchensink"
    workflow: "simple"
  expected_outcome:
    workflow_state: "completed"
    max_duration: "30s"
    violations: []
```

**Tasks**:
- Define Play schema (YAML or Go structs)
- Implement Play executor
- Create Play library with common scenarios
- Integrate with Pitcher and Umpire

### 2.3 Game Plan: Coverage Tracking

**Game Plan**: Set of Plays that must execute in a test run

**Coverage Metrics**:
- All Plays in Game Plan executed at least once
- All Pitch types exercised
- All entity types observed
- Statistical coverage (% of Play variations)

**Implementation**:
- Track Play execution in Almanac (distinct combinations)
- Report coverage at end of test run
- Fail if coverage goals not met

### 2.4 Skipper: Intelligent Orchestration

**Skipper Responsibilities**:
- Select which Plays to run based on coverage goals
- Adapt based on observed system behavior (feedback loop)
- Prioritize Plays that find bugs
- Generate new Play combinations

**Implementation Approaches**:
- Random selection with coverage tracking
- Reinforcement learning (reward = bugs found)
- Mutation-based fuzzing of existing Plays
- TigerBeetle VOPR-style predicate-based scenarios

### 2.5 Almanac: Behavior History

**Purpose**: Track unique behavior combinations to avoid redundant testing

**Storage**:
- Graph or tree structure of observed behaviors
- Hashing for deduplication
- Query interface for Skipper to check coverage

---

## Phase 3: Advanced Capabilities

### 3.1 Standalone Proxy Deployment

- Extract Pitcher proxy from test integration
- Deploy as sidecar or inline proxy
- Support production canary testing
- Remote configuration and monitoring

### 3.2 Multi-Language SDK Support

- Kitchensink in Java, Python, TypeScript
- Unified Play definitions across SDKs
- SDK-specific property validation

### 3.3 Cross-Cluster Scenarios

- Namespace failover chaos testing
- Multi-region consistency validation
- Replication lag property checking

### 3.4 Performance Properties

- Latency bounds under chaos
- Throughput degradation limits
- Resource usage invariants

### 3.5 Production Canary Integration

- Lightweight property checking in prod
- Anomaly detection vs hard failures
- Incremental rollout validation

---

## Milestones Summary

### Phase 1: Passive Validation (Weeks 1-8)

| Milestone | Duration | Deliverable | Status |
|-----------|----------|-------------|--------|
| **1.2.1** Pitcher Core | 1 week | Fault injection framework | Required |
| **1.2.2** In-Code Intercepts | 1 week | Service intercept points | Required |
| **1.2.3** gRPC Interceptors | 1 week | Interceptor-based injection | Optional - defer if not needed |
| **1.3.1** MVP Property | 1 week | RetryResilienceModel | Required |
| **1.4.1** MVP Test | 1 week | End-to-end demo test | Required |
| **1.5.1** Rollout to All Tests | 2 weeks | Universal umpire checking | Required |
| **1.6** Property Expansion | 2 weeks | 4-5 additional properties | Required |

**Note**: Phase 1 can be completed in 7 weeks if milestone 1.2.3 is deferred.

**Phase 1 Exit Criteria**:
- All functional tests run with umpire validation
- At least 5 properties detecting real bug classes
- At least 1 previously undetected bug found
- Documentation for adding new properties

### Phase 2: Active Generation (Weeks 9-16)

| Milestone | Duration | Deliverable |
|-----------|----------|-------------|
| **2.1** Kitchensink Integration | 2 weeks | Omes workload in tests |
| **2.2** Play System | 2 weeks | Play definition & execution |
| **2.3** Game Plan Coverage | 2 weeks | Coverage tracking |
| **2.4** Skipper (Basic) | 2 weeks | Random play selection |

**Phase 2 Exit Criteria**:
- Can replace functional tests with Play-based tests
- Coverage-driven test execution
- Clear path to deprecating manual tests

### Phase 3: Advanced (Weeks 17+)

| Milestone | Duration | Deliverable |
|-----------|----------|-------------|
| **3.1** Standalone Proxy | 3 weeks | Production-ready proxy |
| **3.2** Multi-SDK | 4 weeks | Java/Python/TS support |
| **3.3** Cross-Cluster | 3 weeks | Failover testing |
| **3.4** Performance | 2 weeks | SLA properties |
| **3.5** Production Canary | 4 weeks | Prod deployment |

---

## Success Metrics

### Phase 1
- **Bug Detection**: Find ≥1 bug missed by functional tests
- **Coverage**: All functional tests monitored by umpire
- **False Positives**: <5% violation reports are false positives
- **Performance**: <10% test execution overhead

### Phase 2
- **Test Migration**: 50% of functional tests replaced by Plays
- **Execution Time**: Play-based tests run in ≤ functional test time
- **Coverage**: ≥80% of Playbook exercised per run

### Phase 3
- **Production**: Canary tests run in prod with property validation
- **SDKs**: All 4 major SDKs supported
- **Consolidation**: Single framework for Canary/PR/Nightly

---

## Design Decisions

1. **Pitcher Configuration**: Test-only configuration via onebox pattern (see `tests/testcore/onebox.go`)
2. **Property Language**: Simple Go functions - no DSL investment yet
3. **Violation Handling**: Hard fail - violations fail the test immediately
4. **Proxy Extraction**: Defer until explicitly needed - stay integrated for now
5. **Skipper Intelligence**: Random play selection initially - optimize later based on data

### Deferred Decisions

- **Garbage Collection**: Address when scalability issues arise with entity accumulation
- **Almanac Storage**: Choose structure when implementing Skipper (Phase 2)

---

## References

- **Existing Umpire**: `tools/umpire/` - Current passive validation infrastructure
- **Functional Tests**: `tests/` - Integration with FunctionalTestBase
- **Omes Kitchensink**: https://github.com/temporalio/omes/tree/main/workers/go
- **TigerBeetle VOPR**: https://tigerbeetle.com/blog/2023-03-28-random-fuzzy-thoughts/
- **Workflow Update Properties**: https://chatgpt.com/c/691d1e45-71c0-8006-bc42-dedab68279a2
- **Stamp Branch**: https://github.com/temporalio/temporal/compare/main...stamp

---

## Critical Implementation Details

### Test Isolation
- **Pitcher State**: Must reset between tests via `pitcher.Reset()` in `TearDownTest()`
- **Umpire State**: Entity registry should be cleared or scoped per test
- **Parallel Tests**: Pitcher must be thread-safe (mutex protection on global state)

### Event Flow & Timing
- **Pitcher Events**: Injected failures need to generate observable events for umpire
- **Event Processing**: Need synchronization between test completion and umpire validation
- **Race Conditions**: Consider adding `s.Eventually()` wrapper for umpire checks

### Error Injection Specifics
- **gRPC Errors**: Map to specific status codes (RESOURCE_EXHAUSTED, DEADLINE_EXCEEDED, etc.)
- **Delay Implementation**: Use `time.Sleep()` for simplicity, consider async later
- **Probability**: Use `rand.Float64() < config.Probability` for stochastic injection

### Production Safety
- **Build Tags**: Consider using `//go:build test` to ensure pitcher code doesn't compile in prod
- **Runtime Checks**: Pitcher should panic if activated outside test environment
- **Configuration**: Never read pitcher config from production config files

---

## Risks & Mitigations

### Technical Risks

1. **Risk**: Pitcher intercepts slow down production code paths
   - **Mitigation**: Use build tags, ensure zero-cost abstraction when disabled
   - **Validation**: Benchmark production builds to verify no regression

2. **Risk**: Umpire state grows unbounded during long-running tests
   - **Mitigation**: Implement entity garbage collection after configurable TTL
   - **Validation**: Monitor memory usage during test suite execution

3. **Risk**: Race conditions between pitcher injection and umpire observation
   - **Mitigation**: Add explicit synchronization points, use Eventually() patterns
   - **Validation**: Run tests with race detector enabled

4. **Risk**: False positives from umpire properties
   - **Mitigation**: Start with conservative properties, add escape hatches
   - **Validation**: Track false positive rate, require <5% threshold

### Organizational Risks

5. **Risk**: Developer resistance to test failures from new properties
   - **Mitigation**: Gradual rollout, clear documentation, quick fix turnaround
   - **Validation**: Survey developers, track adoption metrics

6. **Risk**: Maintenance burden of properties as system evolves
   - **Mitigation**: Property versioning, ownership model, deprecation process
   - **Validation**: Track time spent on property maintenance

---

## Open Questions

### Implementation Questions
1. **Intercept Point Discovery**: How do we systematically find all RPC handler methods?
   - *Proposed*: Script that greps for interface implementations

2. **Negative Tests**: How to handle tests that legitimately cause violations?
   - *Proposed*: Add `ExpectedViolations` field to test annotations

3. **Flaky Tests**: How to distinguish between flaky tests and actual violations?
   - *Proposed*: Run property checks multiple times, require consistent failures

4. **Debug Support**: How to debug when a property fails?
   - *Proposed*: Enhanced logging mode that captures full entity state transitions

### Operational Questions
5. **Rollback Plan**: What if umpire causes too many false positives?
   - *Proposed*: Feature flag to disable umpire checking globally

6. **Property Versioning**: How to evolve properties as system behavior changes?
   - *Proposed*: Version properties, allow tests to specify min/max versions

7. **CI Integration**: How to surface violations in CI/CD pipeline?
   - *Proposed*: Structured JSON output for violations, integrate with test reporting

8. **Performance Impact**: What's acceptable overhead for property checking?
   - *Proposed*: Measure baseline, alert if >10% regression

### Phase 2 Questions
9. **Workflow Registration**: How does kitchensink register workflows with test worker?
   - *Proposed*: Auto-registration via reflection or explicit registry

10. **Coverage Definition**: What constitutes test coverage for properties?
    - *Proposed*: State space coverage (all FSM transitions observed)

---

## Phase 4: CATCH-Enabled Canary (`tools/canary`)

**Deployment Model**: External (SDK client/worker only)

**Goal**: Build a continuous testing tool that combines canary-go's comprehensive workflow coverage with CATCH's property-based validation and fault injection capabilities, operating purely from external SDK perspective.

### Vision

Transform canary testing from simple "workflows complete successfully" checks to sophisticated property-based validation under chaos conditions. The new `tools/canary` will serve as both a regression detection system and a living demonstration of CATCH capabilities.

**Key Differences from canary-go**:

| Aspect | canary-go | tools/canary |
|--------|-----------|--------------|
| **Validation** | Workflow completion + basic assertions | Property-based validation via Umpire |
| **Fault Injection** | Relies on natural failures | Proactive SDK-level chaos via Pitcher |
| **Coverage** | Feature functionality | Feature functionality + external-observable properties |
| **Deployment** | Production continuous testing | Development/staging continuous testing |
| **Purpose** | SLA monitoring & regression detection | Bug discovery + property validation |

**Critical Constraints** (External Deployment):
- ✗ Cannot inject faults into server internals (matching, history, persistence)
- ✗ Cannot observe internal service spans (AddWorkflowTask, StoreWorkflowTask)
- ✗ Cannot validate internal-only properties (task delivery guarantees)
- ✓ Can inject faults into SDK operations (activity failures, delays)
- ✓ Can observe workflow/activity lifecycle from SDK perspective
- ✓ Can validate external-observable properties (retry resilience, timeouts, cancellation)
- ✓ Can observe server spans IF server exports OTEL to external collector

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    tools/canary                              │
│                                                              │
│  ┌────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Canary Runner  │  │  Pitcher Core   │  │   Umpire     │ │
│  │                │  │                 │  │              │ │
│  │ - Schedules    │  │ - Inject faults │  │ - Observe    │ │
│  │ - Orchestrates │  │ - RPC failures  │  │ - Validate   │ │
│  │ - Reports      │  │ - Delays        │  │ - Report     │ │
│  └────────┬───────┘  └────────┬────────┘  └──────┬───────┘ │
│           │                   │                   │         │
│           └───────────────────┴───────────────────┘         │
│                              │                               │
└──────────────────────────────┼───────────────────────────────┘
                               │
                    ┌──────────▼───────────┐
                    │  Canary Workflows    │
                    │  (Based on canary-go)│
                    │                      │
                    │ - Sanity            │
                    │ - Signal/Query      │
                    │ - Timeout/Retry     │
                    │ - Cancellation      │
                    │ - Update            │
                    │ - Schedule/Cron     │
                    │ - Versioning        │
                    │ - Nexus             │
                    │ - Visibility        │
                    │ - Archival          │
                    └──────────────────────┘
```

### Workflow Portfolio (Imported from canary-go)

Phase 1 workflows to implement (priority order):

1. **Sanity** - Basic workflow execution
2. **Signal** - Signal handling and workflow interaction
3. **Query** - Query handling and consistency
4. **Timeout** - Timeout behavior and recovery
5. **Retry** - Retry policies and backoff
6. **Cancellation** - Cancellation propagation
7. **Update** - Workflow update correctness
8. **LocalActivity** - Local activity execution
9. **Schedule** - Schedule workflows
10. **Cron** - Cron workflow behavior

Phase 2 workflows (deferred):
- **Versioning** - Worker versioning and compatibility
- **Nexus** - Nexus endpoint integration
- **AdvancedVisibility** - Search attribute behavior
- **Batch** - Batch operations
- **Archival** - History/visibility archival
- **Reset** - Workflow reset behavior

### Implementation Plan

#### 4.1 Milestone: Canary Core Framework

**Goal**: Build the basic canary runner that can execute workflows on a schedule.

**SDK-Level Pitcher Implementation**:

Since canary runs externally, it needs a separate Pitcher implementation using Go SDK interceptors:

```go
// tools/canary/pitcher/sdk_pitcher.go
package pitcher

import (
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

// SDKPitcher implements fault injection via SDK interceptors
type SDKPitcher struct {
	config PitchConfig
}

// NewSDKPitcher creates a pitcher that works via SDK interceptors
func NewSDKPitcher(config PitchConfig) *SDKPitcher {
	return &SDKPitcher{config: config}
}

// WorkflowInterceptor returns an interceptor for workflow operations
func (p *SDKPitcher) WorkflowInterceptor() interceptor.WorkflowInterceptor {
	return &workflowPitchInterceptor{pitcher: p}
}

// ActivityInterceptor returns an interceptor for activity operations
func (p *SDKPitcher) ActivityInterceptor() interceptor.ActivityInterceptor {
	return &activityPitchInterceptor{pitcher: p}
}

// activityPitchInterceptor injects faults into activity execution
type activityPitchInterceptor struct {
	interceptor.ActivityInboundInterceptorBase
	pitcher *SDKPitcher
}

func (a *activityPitchInterceptor) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (interface{}, error) {
	// Check if we should inject a fault
	if a.pitcher.shouldInjectFault("activity.*", in.ActivityType.Name) {
		// Inject delay or error based on config
		if a.pitcher.shouldDelay() {
			time.Sleep(a.pitcher.config.GetRandomDelay())
		}
		if a.pitcher.shouldFail() {
			return nil, errors.New("PITCHER_INJECTED_FAILURE")
		}
	}

	// Execute normally
	return a.Next.ExecuteActivity(ctx, in)
}
```

**Integration with Worker**:

```go
// In canary runner
func (r *Runner) createWorker() worker.Worker {
	options := worker.Options{}

	// Add SDK pitcher interceptor if enabled
	if r.config.EnablePitcher {
		sdkPitcher := pitcher.NewSDKPitcher(r.config.PitchConfig)
		options.Interceptors = []interceptor.WorkerInterceptor{
			sdkPitcher,
		}
	}

	return worker.New(r.client, "canary-task-queue", options)
}
```

**Deliverables**:

```go
// tools/canary/canary.go
package canary

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/testing/pitcher"
	"go.temporal.io/server/tools/umpire"
)

// Config holds configuration for the canary runner.
type Config struct {
	// Temporal server configuration
	ServerAddress string
	Namespace     string

	// Canary behavior
	Interval      time.Duration // How often to run canaries (default: 30s)
	RunOnce       bool          // Run once and exit (for testing)

	// CATCH integration
	EnablePitcher bool          // Enable fault injection
	EnableUmpire  bool          // Enable property validation
	PitchConfig   *PitchConfig  // Fault injection configuration

	// Workflow selection
	Workflows     []string      // Which workflows to run (empty = all)
	Excludes      []string      // Workflow name patterns to exclude

	// Logging and metrics
	Logger        log.Logger
}

// PitchConfig defines fault injection configuration for canary runs.
// NOTE: External deployment - only SDK-level faults are supported
type PitchConfig struct {
	// Probability of injecting faults (0.0-1.0)
	FailureProbability float64
	DelayProbability   float64

	// Target SDK operations for injection (external deployment)
	// Examples: "activity.*", "workflow.signal", "workflow.query"
	// NOT SUPPORTED: "matchingservice.*", "historyservice.*" (requires internal access)
	Targets []string

	// Fault types
	Errors  []string       // Error types to inject in activities
	Delays  []time.Duration // Delay durations for activity/workflow execution
}

// Runner orchestrates continuous canary testing with CATCH validation.
type Runner struct {
	config        *Config
	client        client.Client
	worker        worker.Worker
	pitcher       pitcher.Pitcher
	umpire        *umpire.Umpire
	logger        log.Logger

	// Workflow registry
	workflows     map[string]WorkflowDefinition
}

// WorkflowDefinition describes a canary workflow.
type WorkflowDefinition struct {
	Name           string
	WorkflowFunc   interface{}
	ActivityFuncs  []interface{}
	Description    string

	// Expected properties this workflow should validate
	// NOTE: Only external-observable properties for canary
	Properties     []string // e.g., ["retry-resilience", "timeout-accuracy"]

	// Pitch configuration specific to this workflow
	// NOTE: Only SDK-level pitches for external deployment
	Pitches        []pitcher.PitchConfig
}

// NewRunner creates a new canary runner.
func NewRunner(cfg *Config) (*Runner, error) {
	// Validate configuration
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.ServerAddress == "" {
		cfg.ServerAddress = client.DefaultHostPort
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "canary"
	}
	if cfg.Interval == 0 {
		cfg.Interval = 30 * time.Second
	}

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  cfg.ServerAddress,
		Namespace: cfg.Namespace,
		Logger:    cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize CATCH components
	var pitcherInstance pitcher.Pitcher
	if cfg.EnablePitcher {
		pitcherInstance = pitcher.NewPitcher(cfg.Logger)
		configurePitcher(pitcherInstance, cfg.PitchConfig)
	}

	var umpireInstance *umpire.Umpire
	if cfg.EnableUmpire {
		umpireInstance, err = umpire.New(umpire.Config{
			Logger: cfg.Logger,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create umpire: %w", err)
		}
	}

	runner := &Runner{
		config:    cfg,
		client:    temporalClient,
		pitcher:   pitcherInstance,
		umpire:    umpireInstance,
		logger:    cfg.Logger,
		workflows: make(map[string]WorkflowDefinition),
	}

	// Register workflows
	runner.registerWorkflows()

	return runner, nil
}

// Run starts the canary runner.
func (r *Runner) Run(ctx context.Context) error {
	// Create namespace if it doesn't exist
	if err := r.ensureNamespace(ctx); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Start worker
	r.worker = worker.New(r.client, "canary-task-queue", worker.Options{
		// Configure OTEL exporter to send to Umpire
		// (similar to FunctionalTestBase integration)
	})

	// Register all workflows and activities
	r.registerWorkflowsAndActivities(r.worker)

	if err := r.worker.Start(); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}
	defer r.worker.Stop()

	// Start cron workflow or run once
	if r.config.RunOnce {
		return r.runOnce(ctx)
	}
	return r.runContinuous(ctx)
}

// runContinuous runs canaries on a schedule until context is cancelled.
func (r *Runner) runContinuous(ctx context.Context) error {
	// Start cron workflow that triggers canaries every interval
	wfID := "temporal.canary.cron"
	wfOptions := client.StartWorkflowOptions{
		ID:           wfID,
		TaskQueue:    "canary-task-queue",
		CronSchedule: fmt.Sprintf("@every %ds", int(r.config.Interval.Seconds())),
	}

	_, err := r.client.ExecuteWorkflow(ctx, wfOptions, r.cronWorkflow)
	if err != nil {
		return fmt.Errorf("failed to start cron workflow: %w", err)
	}

	r.logger.Info("Canary runner started", "interval", r.config.Interval)

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

// runOnce runs all canaries once and reports results.
func (r *Runner) runOnce(ctx context.Context) error {
	results := make(map[string]*CanaryResult)

	for name, wf := range r.workflows {
		if r.shouldRunWorkflow(name) {
			r.logger.Info("Running canary workflow", "workflow", name)
			result := r.runCanaryWorkflow(ctx, wf)
			results[name] = result
		}
	}

	// Report results
	return r.reportResults(ctx, results)
}

// registerWorkflows registers all canary workflows.
func (r *Runner) registerWorkflows() {
	// Register workflows imported from canary-go
	// NOTE: Only external-observable properties and SDK-level pitches

	r.workflows["sanity"] = WorkflowDefinition{
		Name:         "workflow.sanity",
		WorkflowFunc: SanityWorkflow,
		ActivityFuncs: []interface{}{SanityActivity},
		Description:  "Basic workflow execution test",
		Properties:   []string{"workflow-lifecycle", "activity-execution"},
	}

	r.workflows["signal"] = WorkflowDefinition{
		Name:         "workflow.signal",
		WorkflowFunc: SignalWorkflow,
		ActivityFuncs: []interface{}{},
		Description:  "Signal handling test",
		Properties:   []string{"workflow-lifecycle"},
	}

	r.workflows["timeout"] = WorkflowDefinition{
		Name:         "workflow.timeout",
		WorkflowFunc: TimeoutWorkflow,
		ActivityFuncs: []interface{}{TimeoutActivity},
		Description:  "Timeout behavior test",
		Properties:   []string{"timeout-accuracy", "retry-resilience"},
		Pitches: []pitcher.PitchConfig{
			{
				Target: "activity.*",  // SDK-level: inject delay in activity execution
				Action: "delay",
				Params: map[string]any{"duration": "5s"},
			},
		},
	}

	r.workflows["retry"] = WorkflowDefinition{
		Name:         "workflow.retry",
		WorkflowFunc: RetryWorkflow,
		ActivityFuncs: []interface{}{RetryActivity},
		Description:  "Retry policy test",
		Properties:   []string{"retry-resilience", "activity-execution"},
		Pitches: []pitcher.PitchConfig{
			{
				Target: "activity.*",  // SDK-level: fail activity execution
				Action: "fail",
				Count:  2,
				Params: map[string]any{"error": "ACTIVITY_EXECUTION_ERROR"},
			},
		},
	}

	r.workflows["cancellation"] = WorkflowDefinition{
		Name:         "workflow.cancellation",
		WorkflowFunc: CancellationWorkflow,
		ActivityFuncs: []interface{}{CancellableActivity},
		Description:  "Cancellation propagation test",
		Properties:   []string{"cancellation-propagation"},
	}

	// Additional workflows registered here...
}

// CanaryResult holds the result of a canary run.
type CanaryResult struct {
	WorkflowName  string
	Success       bool
	Duration      time.Duration
	Violations    []umpire.Violation
	Error         error
}
```

**Configuration File**:

```yaml
# tools/canary/config/config.yaml
server:
  address: "localhost:7233"
  namespace: "canary"

canary:
  interval: 30s
  run_once: false

  # Enable CATCH components
  enable_pitcher: true
  enable_umpire: true

  # Workflow selection
  workflows:
    - sanity
    - signal
    - query
    - timeout
    - retry
    - cancellation
    - update
    - localactivity

  # Fault injection configuration (SDK-level only for external deployment)
  pitcher:
    failure_probability: 0.1  # 10% of operations fail
    delay_probability: 0.05    # 5% of operations delayed

    # SDK-level targets (external deployment)
    # NOTE: Cannot target server internals like "matchingservice.*"
    targets:
      - "activity.*"           # All activity executions
      - "workflow.signal"      # Signal operations
      - "workflow.query"       # Query operations

    # Activity-level errors
    errors:
      - "ACTIVITY_EXECUTION_ERROR"
      - "ACTIVITY_TIMEOUT"
      - "ACTIVITY_CANCELLED"

    # Execution delays
    delays:
      - "100ms"
      - "500ms"
      - "1s"

logging:
  level: "info"
  format: "json"

metrics:
  enabled: true
  port: 9090
```

**Tasks**:
- [ ] Create `tools/canary/` directory structure
- [ ] Implement SDK-level Pitcher (`tools/canary/pitcher/`)
  - [ ] Activity interceptor with fault injection
  - [ ] Workflow interceptor (optional for Phase 1)
  - [ ] Configuration and probability-based injection
- [ ] Implement canary runner core (config, scheduling, orchestration)
- [ ] Add configuration file loading (YAML)
- [ ] Implement cron workflow for continuous execution
- [ ] Add logging and metrics infrastructure
- [ ] Create CLI entry point (`tools/canary/main.go`)
- [ ] Write unit tests for runner logic
- [ ] Write unit tests for SDK pitcher

**Success Criteria**:
- Can start canary runner, it creates cron workflow, and orchestrates workflow execution on schedule
- SDK pitcher can inject activity failures and delays via interceptors

#### 4.2 Milestone: Workflow Implementation (Phase 1)

**Goal**: Implement core canary workflows based on canary-go patterns.

**Workflow Implementation Strategy**:
- Reference canary-go implementations
- Adapt to use Go SDK best practices
- Add OTEL instrumentation for Umpire observation
- Keep workflows simple and focused

**Example: Sanity Workflow**:

```go
// tools/canary/workflows/sanity.go
package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// SanityWorkflow is the simplest canary workflow - just completes successfully.
func SanityWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, SanityActivity, "hello").Get(ctx, &result)
	if err != nil {
		return err
	}

	workflow.GetLogger(ctx).Info("Sanity workflow completed", "result", result)
	return nil
}

// SanityActivity is a simple activity.
func SanityActivity(ctx context.Context, input string) (string, error) {
	return input + " world", nil
}
```

**Example: Retry Workflow with Pitcher**:

```go
// tools/canary/workflows/retry.go
package workflows

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// RetryWorkflow tests retry behavior under transient failures.
func RetryWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result int
	err := workflow.ExecuteActivity(ctx, RetryActivity).Get(ctx, &result)
	if err != nil {
		return err
	}

	if result < 1 {
		return fmt.Errorf("expected at least 1 retry, got %d", result)
	}

	return nil
}

// RetryActivity fails the first N times, then succeeds.
func RetryActivity(ctx context.Context) (int, error) {
	info := activity.GetInfo(ctx)
	attempt := info.Attempt

	// Fail first 2 attempts (Pitcher may inject additional failures)
	if attempt <= 2 {
		return 0, fmt.Errorf("intentional failure on attempt %d", attempt)
	}

	return int(attempt), nil
}
```

**Tasks**:
- [ ] Implement Sanity workflow + activity
- [ ] Implement Signal workflow (signal send/receive patterns)
- [ ] Implement Query workflow (query consistency checks)
- [ ] Implement Timeout workflow (timeout boundaries)
- [ ] Implement Retry workflow (retry policies)
- [ ] Implement Cancellation workflow (cancellation propagation)
- [ ] Implement Update workflow (update correctness)
- [ ] Implement LocalActivity workflow
- [ ] Add workflow registration to runner
- [ ] Write integration tests for each workflow

**Success Criteria**: All Phase 1 workflows can be executed by the runner and complete successfully without faults.

#### 4.3 Milestone: CATCH Integration

**Goal**: Integrate Pitcher fault injection and Umpire property validation into canary runs.

**Pitcher Integration**:

```go
// In canary runner
func (r *Runner) runCanaryWorkflow(ctx context.Context, wf WorkflowDefinition) *CanaryResult {
	result := &CanaryResult{
		WorkflowName: wf.Name,
	}

	// Configure pitcher for this workflow
	if r.pitcher != nil && len(wf.Pitches) > 0 {
		for _, pitch := range wf.Pitches {
			r.pitcher.Configure(pitch.Target, pitch)
		}
		defer r.pitcher.Reset() // Clean up after workflow
	}

	start := time.Now()

	// Execute workflow
	wfOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s-%d", wf.Name, time.Now().Unix()),
		TaskQueue: "canary-task-queue",
	}

	wfRun, err := r.client.ExecuteWorkflow(ctx, wfOptions, wf.WorkflowFunc)
	if err != nil {
		result.Error = err
		result.Success = false
		return result
	}

	// Wait for workflow to complete
	err = wfRun.Get(ctx, nil)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err
		result.Success = false
		return result
	}

	// Check umpire violations
	if r.umpire != nil {
		time.Sleep(100 * time.Millisecond) // Allow event processing
		violations := r.umpire.Check(ctx)
		result.Violations = violations
		result.Success = len(violations) == 0
	} else {
		result.Success = true
	}

	return result
}
```

**Umpire Integration**:

```go
// In canary runner initialization
func (r *Runner) setupOTELExporter() error {
	// EXTERNAL DEPLOYMENT LIMITATION:
	// Canary cannot directly capture server OTEL spans like functional tests do.
	// Instead, it has two options:

	// Option 1: Observe SDK-side spans only (limited visibility)
	// - Can see workflow start/complete, activity schedule/complete
	// - Cannot see internal server operations (AddWorkflowTask, etc.)
	sdkExporter := umpire.NewSDKSpanExporter(r.umpire)

	// Option 2: Connect to server's OTEL collector (requires server config)
	// - Server must export spans to external OTEL collector
	// - Canary subscribes to collector to receive spans
	// - Provides visibility into server operations (if server exports them)
	// collectorClient := otel.NewCollectorClient(r.config.OTELCollectorEndpoint)
	// collectorExporter := umpire.NewCollectorSpanExporter(r.umpire, collectorClient)

	return nil
}
```

**Property Validation** (External-Compatible Only):

The canary uses external-observable Umpire models:
- ✓ `StuckWorkflowModel`: Detects workflows that don't complete (observable from SDK)
- ✗ `LostTaskModel`: Detects tasks that aren't polled (requires internal spans - NOT available)
- ✓ `RetryResilienceModel`: Validates retry behavior under failures (observable from SDK)
- ✗ `TaskDeliveryGuaranteesModel`: Validates task delivery invariants (requires internal spans - NOT available)
- ✓ `TimeoutAccuracyModel`: Validates timeout behavior (observable from SDK)
- ✓ `CancellationPropagationModel`: Validates cancellation (observable from SDK)
- ✓ `ActivityExecutionModel`: Validates activity lifecycle (observable from SDK)

**Tasks**:
- [ ] Implement SDK-level Pitcher (separate from server-side pitcher in Phase 1)
  - [ ] Activity execution interceptor (fail/delay activities)
  - [ ] Workflow interceptor (delay workflow tasks)
  - [ ] Signal/Query interceptor (timeout/fail operations)
- [ ] Integrate Pitcher into runner workflow execution
- [ ] Configure Pitcher with workflow-specific pitches
- [ ] Set up SDK OTEL span capture for Umpire observation
- [ ] Add violation checking after workflow completion
- [ ] Implement result aggregation and reporting
- [ ] Add metrics for violations detected
- [ ] Write integration tests with fault injection enabled

**Success Criteria**:
- SDK-level Pitcher successfully injects faults (activity failures, delays)
- Umpire detects violations in external-observable properties
- Canary reports detailed results including violations
- Clear distinction: internal pitcher for tests, SDK pitcher for canary

#### 4.4 Milestone: Continuous Execution & Monitoring

**Goal**: Deploy canary as a continuously running service with monitoring and alerting.

**Deployment Options**:

1. **Standalone Binary**:
```bash
# Run locally
./temporal-canary --config config.yaml

# Run in Kubernetes
kubectl apply -f k8s/canary-deployment.yaml
```

2. **Docker Container**:
```dockerfile
# Dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN make build-canary

FROM alpine:latest
COPY --from=builder /app/temporal-canary /usr/local/bin/
ENTRYPOINT ["temporal-canary"]
```

**Monitoring & Metrics**:

```go
// Metrics exported by canary
type Metrics struct {
	WorkflowsRun       counter    // Total workflows run
	WorkflowsSucceeded counter    // Workflows that completed successfully
	WorkflowsFailed    counter    // Workflows that failed
	ViolationsDetected counter    // Property violations detected
	FaultsInjected     counter    // Faults injected by Pitcher

	WorkflowDuration   histogram  // Workflow execution time
	E2ELatency         histogram  // End-to-end canary latency
}
```

**Alerting Configuration**:

```yaml
# Prometheus alert rules
groups:
  - name: temporal-canary
    rules:
      - alert: CanaryWorkflowFailureRate
        expr: |
          rate(canary_workflows_failed_total[5m]) /
          rate(canary_workflows_run_total[5m]) > 0.1
        for: 5m
        annotations:
          summary: "Canary workflow failure rate above 10%"

      - alert: CanaryViolationsDetected
        expr: canary_violations_detected_total > 0
        for: 1m
        annotations:
          summary: "Property violations detected by Umpire"
```

**Tasks**:
- [ ] Add Prometheus metrics exposition
- [ ] Create Grafana dashboard for canary monitoring
- [ ] Write Kubernetes deployment manifests
- [ ] Create Docker image build pipeline
- [ ] Document deployment procedures
- [ ] Set up alerting rules
- [ ] Add health check endpoint
- [ ] Implement graceful shutdown

**Success Criteria**:
- Canary runs continuously in Kubernetes
- Metrics are exported and visualized
- Alerts fire when violations are detected
- Can be deployed to staging/production environments

#### 4.5 Milestone: Advanced Features

**Goal**: Add sophisticated capabilities that leverage CATCH fully.

**Play-Based Execution** (Phase 2 integration):

```go
// Execute canaries as "Plays" with composed fault scenarios
type CanaryPlay struct {
	Name        string
	Description string
	Workflow    string
	Pitches     []pitcher.PitchConfig
	Properties  []string
	Frequency   time.Duration // How often to run this play
}

// Example Play: "Matching Failure with Retry"
var MatchingFailurePlay = CanaryPlay{
	Name:        "matching-failure-retry",
	Description: "Tests workflow completion after matching service failures",
	Workflow:    "sanity",
	Pitches: []pitcher.PitchConfig{
		{
			Target:      "matchingservice.AddWorkflowTask",
			Action:      "fail",
			Count:       3,
			Probability: 1.0,
			Params:      map[string]any{"error": "RESOURCE_EXHAUSTED"},
		},
	},
	Properties: []string{"retry-resilience", "task-delivery"},
	Frequency:  5 * time.Minute,
}
```

**Adaptive Fault Injection**:

```go
// Dynamically adjust fault injection based on observed behavior
type AdaptivePitcher struct {
	pitcher.Pitcher
	history []PitchOutcome
}

func (ap *AdaptivePitcher) Learn(outcome PitchOutcome) {
	// Increase fault probability if system is too stable
	// Decrease if too many failures
	// Focus on areas that find bugs
}
```

**Cross-Cluster Testing**:

```go
// Run canaries across multiple clusters/namespaces
type MultiClusterCanary struct {
	Clusters []ClusterConfig
	Runner   *Runner
}

// Tests namespace failover, replication, etc.
```

**Tasks** (Deferred to Phase 2):
- [ ] Implement Play-based canary execution
- [ ] Add adaptive fault injection
- [ ] Implement cross-cluster canary scenarios
- [ ] Add coverage tracking (Almanac integration)
- [ ] Implement intelligent play selection (Skipper integration)

**Success Criteria**: Canary system can execute complex composed scenarios and adapt based on results.

### Integration with Existing Canary (canary-go)

**Relationship**:
- `canary-go`: Production continuous testing (Cloud cells)
- `tools/canary`: Development/staging continuous testing (CATCH-enabled)

**Migration Path**:
1. Phase 1: Run `tools/canary` in parallel with `canary-go` in staging
2. Phase 2: Add CATCH validation to `canary-go` workflows (via Umpire library)
3. Phase 3: Consider consolidation once CATCH is mature

**Shared Components**:
- Workflow implementations (import from canary-go where possible)
- Configuration patterns
- Metrics and monitoring approaches

### Success Metrics

**Phase 1 (Milestones 4.1-4.4)**:
- [ ] 10 core workflows implemented and running
- [ ] Pitcher successfully injecting faults in 10% of operations
- [ ] Umpire detecting ≥1 violation class per workflow type
- [ ] Canary runs continuously with <5% false positive rate
- [ ] Deployed to staging environment

**Phase 2 (Milestone 4.5)**:
- [ ] Play-based execution implemented
- [ ] Adaptive fault injection operational
- [ ] Coverage tracking integrated with Almanac
- [ ] Finding bugs missed by functional tests and canary-go

### Timeline

| Milestone | Duration | Dependencies |
|-----------|----------|--------------|
| **4.1** Canary Core | 2 weeks | Phase 1 (Pitcher + Umpire complete) |
| **4.2** Workflows (Phase 1) | 3 weeks | 4.1 complete |
| **4.3** CATCH Integration | 2 weeks | 4.2 complete, Phase 1 complete |
| **4.4** Continuous Execution | 1 week | 4.3 complete |
| **4.5** Advanced Features | 4 weeks | Phase 2 complete |

**Total**: 12 weeks (8 weeks for MVP, 4 weeks for advanced features)

**Dependencies**:
- Requires Phase 1 (Pitcher + Umpire) to be complete for core functionality
- Advanced features require Phase 2 (Play system, Skipper) to be complete

### Open Questions

1. **Workflow Reuse**: Should we import canary-go as a Go module or reimplement workflows?
   - *Proposed*: Start with reimplementation for better control, consider import later

2. **Property Coverage**: Which properties should each workflow validate?
   - *Proposed*: Start with external-observable properties only (stuck workflows, retry resilience, timeout accuracy)
   - *Note*: Internal-only properties (task delivery, lost tasks) cannot be validated by canary

3. **Fault Injection Intensity**: What's the right balance of fault injection probability?
   - *Proposed*: Start conservative (10% for activities), increase based on data
   - *Note*: Limited to SDK-level faults (activity failures, delays)

4. **Deployment Target**: Where should tools/canary run initially?
   - *Proposed*: Local dev first, then staging environments, eventually production

5. **Failure Handling**: Should canary continue running if violations are detected?
   - *Proposed*: Log violations but continue (similar to canary-go) - alerts are external

6. **Multi-SDK Support**: Should tools/canary support Java/Python/TypeScript SDKs?
   - *Proposed*: Defer to Phase 2 - start with Go SDK only

7. **SDK Pitcher Implementation**: Where should SDK-level Pitcher live?
   - *Proposed*: `tools/canary/pitcher/` (separate from server-side pitcher in `common/testing/pitcher/`)
   - *Rationale*: Different intercept points, different deployment model

8. **OTEL Span Collection**: Should canary require server to export OTEL spans?
   - *Proposed*: Phase 1 uses SDK-side spans only (limited visibility)
   - *Proposed*: Phase 2 adds optional OTEL collector integration for server span visibility

### References

- **canary-go Repository**: https://github.com/temporalio/canary-go
- **Umpire Implementation**: `tools/umpire/`
- **Pitcher Plan**: Section 1.2 of this document
- **Phase 2 Play System**: Section 2.2 of this document

---

## Next Steps

1. **Review & Align**: Discuss open questions, adjust priorities
2. **Milestone 1.2.1**: Start implementing Pitcher core framework
3. **Parallel Track**: Expand existing properties (1.6) while building Pitcher
4. **Documentation**: Create developer guide for adding intercept points and properties
