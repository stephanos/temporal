# Agentworkflow Configuration and CLI Design

## Goal

Make Agentworkflow own a single YAML configuration at `.agentworkflow/config.yml`, support a
Codex and Claude model choice for every agent-backed workflow stage, use Cobra for the CLI, and
hide implementation packages that are not intended for external consumers.

This is a breaking configuration change. The implementation will not detect, migrate, or document
the earlier JSON configuration. JSON remains in use for runtime protocols, retained run state, and
machine-readable CLI output.

## Configuration layout

`agentworkflow init --project <root>` creates `<root>/.agentworkflow/config.yml`. The existing
`--config` override remains available and accepts a YAML file inside the project root. Initialization
continues to publish the file atomically without overwriting an existing configuration.

The entire `.agentworkflow` directory is copied into isolated workspaces so agents can read declared
instructions and task files. It is always protected from candidate mutation and cannot be excluded
through `source.exclude`. The resolved configuration also protects the selected configuration file
and every declared instruction file.

Documentation examples use this layout:

```text
.agentworkflow/
├── config.yml
├── instructions/
│   └── architecture.md
└── tasks/
    └── issue-123.md
```

## Stage model selection

Every agent-backed workflow stage accepts an optional `models` mapping:

```yaml
workflow:
  stages:
    - kind: implement
      enabled: true
      models:
        codex: gpt-5.3-codex
        claude: opus
      prompt: |-
        Implement the accepted plan in the candidate workspace.
```

The mapping has exactly two optional keys, `codex` and `claude`. Values must be non-empty after
trimming. `models` is valid on stages that invoke an agent: `discover`, `plan`, `implement`,
`review`, and `repair`. It is rejected on `check` and `apply`, which do not invoke a backend.

The selected CLI backend chooses the corresponding stage model. An omitted backend entry delegates
model selection to that provider's normal default. The existing `--model` flag remains a whole-run
override and takes precedence over every configured stage model.

The logical workflow stage supplies the model to every invocation it owns. Plan review and plan
revision use the `plan` model. Parallel review lenses use the `review` model. Repair attempts use the
`repair` model. Resume uses the admitted workflow stored in the checkpoint, so later configuration
edits cannot alter model selection for an existing run.

## Engine and backend boundary

`WorkflowStage` carries the resolved Codex and Claude model names. The workflow selects one using
the admitted backend identity and places it on `Invocation.Model`. Codex and Claude prefer their
construction-time model when `--model` supplied one; otherwise they add `Invocation.Model` to the
provider command.

The admitted request already persists the workflow, including its models. The backend configuration
digest continues to capture command-line overrides and qualification settings. This preserves the
existing resume identity check while making stage-specific choices part of durable request state.

No package outside `tools/agentworkflow` imports its root Go package. The engine, public data types,
workflow implementation, tests, examples, and backend test helper will therefore move under
`internal/agentworkflow` and `internal/backendtest`. Provider implementations remain under
`internal/backend`. This makes the module's supported product surface the executable rather than an
accidental Go library API.

## Cobra CLI

`cmd/agentworkflow/main.go` becomes a thin entry point that delegates to an internal CLI package.
The CLI package builds a Cobra root command with the existing subcommands:

- `init`
- `doctor`
- `run`
- `resume`
- `inspect`
- `report`
- `diff`
- `apply`
- `config explain`

Existing flags, defaults, output formats, and exit-code categories remain stable. Cobra supplies
argument validation and help rendering. Operational errors retain the `agentworkflow:` prefix;
usage errors return the existing usage exit code. Commands keep dependency construction behind
small helpers so configuration loading, backend selection, and engine opening remain independently
testable.

## Legacy configuration removal

The loader no longer probes `.agentworkflow/project.json` or emits migration instructions. Tests and
documentation no longer name the previous JSON configuration, its flags, or its migration path.
The YAML decoder remains strict about unknown fields, duplicate keys, aliases, tags, multiple
documents, and file containment. A non-YAML `--config` path fails through the normal YAML path and
content validation rather than a legacy-format-specific branch.

## Error handling

- Missing default configuration errors point to `.agentworkflow/config.yml` and `agentworkflow init`.
- Invalid stage model keys, empty model values, or models on non-agent stages fail before workspace
  creation or backend startup.
- Cobra parse and argument errors are classified as usage failures.
- Backend, filesystem, capacity, cancellation, and workflow outcomes retain their existing exit-code
  classification.
- Output write failures remain process failures.

## Testing

The repository's pinned golangci-lint v2.12.2 predates Go 1.27 generic-method support and panics in
its embedded `buildir` analyzer before Agentworkflow changes can be evaluated. As an implementation
prerequisite, the pin advances to v2.13.1, which includes Go 1.27 support and a stable Staticcheck
v0.8.0. The lint configuration and enabled analyzers remain unchanged. Once the analyzer can run,
it exposes Nexus Operation compile failures from an inherited merge that dropped previously
committed `TaskInvocation` signatures and handler embedding while retaining their interfaces and
tests. The prerequisite restores those exact hunks before repository lint verification.
Go 1.27 vet also requires the pointer-constructed `UnprocessableTaskError` to implement `error`
through a pointer receiver so `%w` preserves its `errors.Is` identity.
The resulting full lint pass also removes a stale Umpire2 call to a deliberately retired server
helper. Umpire2's existing payload domain instead enforces its documented bound using the complete
encoded payload size, including metadata.
The same inherited merge dropped the workflow resetter's CHASM/HSM reapply implementation and its
current workflow-context call shape while leaving updated callers and tests. The prerequisite
restores the latest ancestor implementation, including subsequent logger and routing behavior.
Two retained test-cluster contracts likewise require their lost call-site hunks: the current client
constructor arity and worker-service request plumbing are restored from their ancestor commits.
Matching tests also drop two assignments to the already-removed migration flag, completing the
ancestor change without restoring obsolete configuration.
The branch's retained in-memory SQLite functional-test caller regains its lost testcore option,
persistence propagation, and focused require-style test against the current cluster-pool shape.
The same testcore repair preserves lazy router initialization by routing all retained callers through
the existing getter.
XDC test call sites drop four options whose corresponding cluster API is absent on this branch,
completing the other side of that partial merge without inventing unsupported plumbing.

Because the current branch is 1,384 commits ahead of its six-month-old local `main` lint baseline,
the canonical comparison reports 1,811 inherited findings across thousands of unrelated files.
Task completion therefore requires the unchanged full analyzer configuration to report zero issues
against the task's committed baseline; the inherited branch-wide cleanup remains separate debt.

Implementation follows red-green-refactor cycles for:

1. default initialization and loading from `.agentworkflow/config.yml`;
2. protection and exclusion validation for `.agentworkflow`;
3. strict model mapping validation and resolved YAML output;
4. configured model propagation to Codex and Claude commands;
5. `--model` precedence over configured stage models;
6. Cobra command help, argument validation, and stable exit classifications;
7. the existing run, inspect, diff, apply, resume, and backend integration behavior; and
8. removal of previous JSON configuration references from code, tests, and documentation.

Focused tests run with `-tags test_dep` inside the nested module. Final verification includes the
complete nested-module test suite, build, formatting/import checks, and the repository lint command
required by the project instructions.

## Trade-offs and failure modes

Full internalization creates a larger mechanical diff, but there are no in-repository consumers and
the module was introduced as a standalone tool. It prevents an uncommitted library contract from
hardening while the implementation is still new.

Stage model lookup is constant work per invocation and has no material performance or scalability
cost. Parallel reviewers share immutable workflow configuration. A crash cannot change the chosen
model because the admitted request is checkpointed before workflow execution. Invalid configuration
fails before expensive workspace copies or agent calls. Provider or check load increasing tenfold
continues to be bounded by the engine's existing time, output, event, source, and reviewer limits.

No new security authority is introduced. Provider choice still comes from the CLI, model strings are
passed as single command arguments, and existing qualified-backend restrictions remain in force.
