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

## Architecture Overview

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
- **RPC Pitches**: Fail, delay, timeout, drop response
- **Timing Pitches**: Trigger deadline early, fire timer before due time, remove deadline
- **Persistence Pitches**: Delay write, fail transaction
- **Coordination Pitches**: Hold request at synchronization point

#### 1.2.1 Milestone: Pitcher Core Framework

**Architecture**: Follow the `testhooks.TestHooks` pattern (see `common/testing/testhooks/`)
- Global state accessible during tests only
- Configuration set via onebox/test cluster initialization
- Injected into services via fx.Decorate (similar to `testHooks` in `onebox.go:397`)

**Deliverables**:
```go
// common/testing/pitcher/pitcher.go
type Pitcher interface {
    // Throw executes a pitch (atomic fault injection action)
    // Returns error to inject (nil = no injection)
    Throw(ctx context.Context, target string) error

    // Configure sets pitch configuration for a target
    Configure(target string, config PitchConfig)

    // Reset clears all configurations
    Reset()
}

// Pitch configuration
type PitchConfig struct {
    Action      string            // "fail", "delay", "timeout", etc.
    Probability float64           // 0.0-1.0 (1.0 = always)
    Count       int               // Execute N times then stop (0 = unlimited)
    Params      map[string]any    // Action-specific parameters
}

// Common error codes to inject
const (
    ErrorResourceExhausted = "RESOURCE_EXHAUSTED"
    ErrorDeadlineExceeded  = "DEADLINE_EXCEEDED"
    ErrorUnavailable       = "UNAVAILABLE"
    ErrorInternal          = "INTERNAL"
)

// Common delay parameters
type DelayParams struct {
    Duration time.Duration // How long to delay
    Jitter   float64       // Random jitter (0.0-1.0)
}

// Global pitcher instance (only active in tests)
var (
    globalPitcher Pitcher
    pitcherMu     sync.RWMutex // Thread-safety for parallel tests
)

// Get returns the global pitcher (nil if not in test mode)
func Get() Pitcher {
    pitcherMu.RLock()
    defer pitcherMu.RUnlock()
    return globalPitcher
}

// Set configures the global pitcher (test setup only)
func Set(p Pitcher) {
    pitcherMu.Lock()
    defer pitcherMu.Unlock()
    globalPitcher = p
}
```

**Integration Pattern**:
```go
// Service code intercept point (similar to testhooks pattern)
func (h *historyHandler) StartWorkflowExecution(ctx context.Context, req *historyservice.StartWorkflowExecutionRequest) (*historyservice.StartWorkflowExecutionResponse, error) {
    // Pitcher intercept - only active during tests
    if p := pitcher.Get(); p != nil {
        if err := p.Throw(ctx, "historyservice.StartWorkflowExecution"); err != nil {
            return nil, err
        }
    }

    // Normal execution
    // ...
}
```

**Test Configuration** (via onebox pattern):
```go
// In TemporalImpl (onebox.go)
type TemporalImpl struct {
    // ... existing fields ...
    pitcher pitcher.Pitcher
}

// In test setup
func (s *MyTestSuite) SetupTest() {
    s.FunctionalTestBase.SetupTest()

    // Configure pitcher via test cluster
    s.testCluster.GetPitcher().Configure("matchingservice.AddWorkflowTask", pitcher.PitchConfig{
        Action: "fail",
        Count:  1,
        Params: map[string]any{"error": "RESOURCE_EXHAUSTED"},
    })
}
```

**Tasks**:
- [ ] Define Pitcher interface and core types (`common/testing/pitcher/pitcher.go`)
- [ ] Implement basic Pitcher implementation with in-memory state
- [ ] Add global pitcher Get/Set functions with thread-safety
- [ ] Build pitch action handlers (RPC fail, RPC delay)
- [ ] Implement per-test state isolation (Reset() in SetupTest/TearDownTest)
- [ ] Add to TemporalImpl in `tests/testcore/onebox.go`
- [ ] Expose via TestCluster API for test configuration
- [ ] Add ConfigurePitcher() helper to FunctionalTestBase
- [ ] Write unit tests for pitcher framework

**Success Criteria**: Can inject RPC failures via test code with targeted configuration.

#### 1.2.2 Milestone: Intercept Points - In-Code

**Deliverables**:
- Pitcher intercept macros/helpers for common patterns
- Intercepts in key locations:
  - `service/matching/`: AddWorkflowTask, PollWorkflowTaskQueue
  - `service/history/`: StartWorkflowExecution, RespondWorkflowTaskCompleted
  - `service/frontend/`: All public API methods
- Test utilities to configure pitches per test

**Integration with Tests**:
```go
// In test setup
func (s *MyTestSuite) SetupTest() {
    s.FunctionalTestBase.SetupTest()

    // Configure pitcher for this test
    s.ConfigurePitcher(pitcher.Config{
        Pitches: []pitcher.PitchConfig{
            {
                Target: "matchingservice.AddWorkflowTask",
                Action: "fail",
                Count:  1, // Fail first call only
                Params: map[string]any{
                    "error": "RESOURCE_EXHAUSTED",
                },
            },
        },
    })
}
```

**Tasks**:
- [ ] Add pitcher intercept points to matching service
- [ ] Add pitcher intercept points to history service
- [ ] Add pitcher intercept points to frontend service
- [ ] Create intercept point discovery script (grep for RPC handlers)
- [ ] Create test configuration helpers
- [ ] Document intercept point placement guidelines
- [ ] Add debug logging for pitch execution
- [ ] Add metrics/counters for pitch activation

**Success Criteria**: Can fail first matching call in functional test, verify via logs.

#### 1.2.3 Milestone: Intercept Points - gRPC Interceptors (Optional for MVP)

**Goal**: Add gRPC-level fault injection via interceptors in test cluster (can be extracted to proxy later).

**Note**: This milestone can be deferred if in-code intercepts (1.2.2) are sufficient for MVP.

**Deliverables**:
```go
// common/testing/pitcher/interceptor.go
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        // Execute pitcher logic before RPC
        if p := Get(); p != nil {
            if err := p.Throw(ctx, method); err != nil {
                return err
            }
        }

        // Invoke actual RPC
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
    // Similar for server side
}
```

**Integration with Test Cluster**:
- Add interceptors to gRPC clients/servers in onebox initialization
- Enable RPC-boundary fault injection without code changes
- Supports scenarios like connection drops, network delays

**Tasks**:
- [ ] Implement gRPC client interceptor with pitcher
- [ ] Implement gRPC server interceptor with pitcher
- [ ] Integrate interceptors into test cluster setup (`onebox.go`)
- [ ] Add interceptor-specific pitch types (connection drop, network delay)
- [ ] Test interceptors work with functional tests
- [ ] Document when to use interceptors vs in-code intercepts

**Success Criteria**: Can inject network-level failures via gRPC interceptor in functional tests.

**Decision Point**: Evaluate if this milestone is needed for MVP after completing 1.2.2. May defer to Phase 2 if in-code intercepts are sufficient.

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

1. **Task Delivery Guarantees**
   - Every task added to matching is eventually polled or explicitly cancelled
   - No task is polled multiple times without being re-added

2. **Workflow Lifecycle Invariants**
   - Workflow cannot transition to completed without task completion
   - Started workflows must have matching task queue activity

3. **Activity Execution**
   - Activity tasks follow add → poll → complete/fail lifecycle
   - Activity timeouts trigger within expected bounds

4. **Task Queue Consistency**
   - Task queue pollers receive tasks for their registered queue
   - Task routing respects task queue versions/builds

5. **Cross-Cluster Failover** (deferred to Phase 2)
   - Namespace failover transitions are monotonic
   - No dual-write scenarios during failover

**Milestone per Property**: Define, implement, test, document

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

## Next Steps

1. **Review & Align**: Discuss open questions, adjust priorities
2. **Milestone 1.2.1**: Start implementing Pitcher core framework
3. **Parallel Track**: Expand existing properties (1.6) while building Pitcher
4. **Documentation**: Create developer guide for adding intercept points and properties
