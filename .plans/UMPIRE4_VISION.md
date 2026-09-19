# Umpire vision

- define a single model for software behavior
- use model to specify and verify known regression tests
- known regression tests execution (plan) is deterministic
- can use the same model and regression tests for verifying locally, on CI, CICD (cloud deployment) and canary (production)
- ability to work in white box (CI, single procees) and black box (only gRPC access, no internal traces) modes
- very developer-friendly API to define models and tests
- exploration mode to find unknown bugs
- faults as first class citizens
- works with non-linear scenario steps: eg full ID of entity (eg runID) not known at every step
- supports Omes kitchensink (or sth like it) where SDK workers can be defined to respond certain ways ("pre-programmed")
- guided exploration mode where certain parameters or templates can be provided and state space is explored automatically
- works in distributed processes where clock skew can happen
- guided automatic fuzzing where new behavior is explored and prioritized

---

## Acceptance Criteria

- two consumers: functional tests and canary
  - example showing how particular tests are not executable with canary since they require white box
- example showing Go SDK being driven
- example showing injecting one fault
- example of generating functional test inputs as file artifacts that can be checked-in and are deterministic
- example of explorartive test to find issues itself
- example showing how to translate a subset of existing functional tests to model
- developer-facing syntax is friendly enough for Lean-newbies
