# Production canary

`tools/canary` runs the committed activity Model against a dedicated canary namespace in
production through the Runner (`umpire.local/model/runner`) and the Temporal Driver
(`common/testing/testpilot/temporal/act`). Each invocation makes at most two serial Runs under one
lease. Each Run starts one fresh activity on the canary task queue, whose in-process worker the
Model answers, and is judged by the Model: Explained, Unexplained or Refused. Each Run is published
with `runner.Publish`; the Run is the only artifact a Run writes.

**Nothing here authorizes a release.** A canary Run says that one Run of the activity Model, against
one dedicated namespace, was explained, unexplained or refused. It is not release evidence, not a
rollout gate and not a statement about customer traffic. This runbook gives no schedule, and the
canary never reruns itself. Every invocation is an operator's manual dispatch.

## Contents

- [What the canary may touch](#what-the-canary-may-touch)
- [Operator preconditions](#operator-preconditions)
- [Configuring the policy](#configuring-the-policy)
- [Invoking the canary](#invoking-the-canary)
- [What preflight checks](#what-preflight-checks)
- [The Limits](#the-limits)
- [Reading the outcome](#reading-the-outcome)
- [The lease, lost iterations and reconciliation](#the-lease-lost-iterations-and-reconciliation)
- [Clearing an uncertain scope](#clearing-an-uncertain-scope)
- [Published Runs and the retained artifact](#published-runs-and-the-retained-artifact)
- [Where the code lives](#where-the-code-lives)

## What the canary may touch

The canary holds one credential. It is a namespace writer on the canary namespace and has no other
permission. It arrives as a TLS client certificate and key (`UMPIRE_CANARY_TLS_CERT`,
`UMPIRE_CANARY_TLS_KEY`), an API key (`UMPIRE_CANARY_API_KEY`), or both. Every call the canary
makes is scoped to the namespace, except the SDK client's `GetSystemInfo` when it dials:

- the Model's activity RPCs (start, describe, cancel) and the worker's poll and responses;
- the lease workflow's start, signal and termination;
- history reads of the lease, and describe and terminate of the activities it fenced;
- one `DescribeNamespace` in preflight.

It changes no deployment, configuration or routing, injects no fault and sends no customer
traffic.

The target's three coordinates come from the environment. `UMPIRE_CANARY_GRPC` is the whole
`host:port` value. The other two are `UMPIRE_CANARY_NAMESPACE` and `UMPIRE_CANARY_TASK_QUEUE`. The
Redactor removes the credential and every coordinate from progress, summaries and logs. A
published Run keeps its World, the activity's Start request, which names the namespace and task
queue the Model needs to replay it; it never holds the credential or the gRPC target. The
coordinates are not secrets; the credential is. The policy stores the coordinates as unsalted SHA-256 digests, so
a digest confirms a guessed name but does not hide one.

## Operator preconditions

The repository cannot check these. An operator sets each one up before the first dispatch and keeps
it true afterwards.

1. **The `production-canary` GitHub environment is protected.** Its deployment branches are
   restricted to `main`, and it requires reviewers. A `workflow_dispatch` runs the workflow file
   and code of the branch it is dispatched on, so this protection alone decides who receives the
   credential. The workflow's `if: github.ref == 'refs/heads/main'` and preflight's ref checks are
   defense in depth.
2. **The environment holds the credential and the coordinates** as secrets with the six names
   listed above. Set a TLS pair, an API key, or both.
3. **The credential is a namespace writer on the canary namespace and nothing else.**
4. **The canary namespace's retention is at least 30 days.** The lease's history, which says which
   activities it fenced, ages out with the namespace's retention. Retention must be longer than the
   longest gap you allow before reconciling.
5. **The coordinate digests are committed to the policy** in a reviewed pull request to `main`.
   Until then preflight refuses every dispatch as `policy-unconfigured`.

## Configuring the policy

`policy/production-canary.json` is the canary's one policy. It is embedded in the binary and
decoded strictly. It pins the Model guest's hash (`modelHash`, the SHA-256 of the committed
`model/temporal/hosts/go/umpire_temporal_guest.wasm`), the authority class (`protected-workflow`), the lease, the repository, the trusted ref, the workflow path and the Limits.
It is committed with every coordinate set to the literal `unconfigured`.

To configure it, digest each coordinate's exact value as the environment holds it, with no
trailing newline:

```bash
printf '%s' "$VALUE" | shasum -a 256 | cut -d' ' -f1
```

Replace each `unconfigured` under `coordinates` (`grpc`, `namespace`, `taskQueue`) with its
lower-case hex digest. Open a pull request to `main` and have it reviewed.
Follow the same procedure for any later coordinate change. A policy with some coordinates configured
and others not is still `policy-unconfigured`. A digest that does not match the environment's value
is `coordinate-mismatch`. No flag can override the policy, and the untagged build has no other
policy. A change to the committed guest (`make umpire-guest`) changes its hash: update `modelHash`
in the same reviewed change, or preflight refuses every dispatch as `model-mismatch`.

## Invoking the canary

**The canary cannot be dispatched at present.** The policy's `workflowPath` names
`.github/workflows/umpire-production-canary.yml` as the one workflow with production authority, and
that file is not in the repository. Preflight's `workflow-context` check accepts no other workflow,
so every invocation is refused until the workflow file is restored or the policy names another one
in a reviewed pull request. This section describes the workflow the policy expects.

An operator dispatches that workflow (**Umpire production canary**) on `main` from the Actions tab.
An environment reviewer then approves the deployment. The workflow:

- runs on `workflow_dispatch` only, with `contents: read` and no other permission;
- runs in one concurrency group, `umpire-production-canary`, which never cancels a job in progress,
  so no two jobs overlap;
- has a 30-minute job timeout and uses a fresh GitHub-hosted runner, so nothing survives from one
  dispatch to the next;
- builds `umpire-canary` (`make canary-build`), then runs `umpire-canary run`;
- always runs `umpire-canary reconcile` and always uploads `canary-output/`, whatever `run` did.

Each mode takes only `--output <dir>` and `--recovery <file>`. Neither mode accepts a Model, target,
Driver, seed, retry, executable, endpoint, credential or release option. `run` writes
`run-summary.json` (stdout) and `run-progress.log` (stderr), and `reconcile` writes
`reconcile-summary.json` and `reconcile-progress.log`, all in the uploaded directory. The recovery
file lives in the runner's temporary directory with mode 0600. It is never uploaded and disappears
with the runner.

## What preflight checks

Preflight runs before any mutation, in this order. The first failure is a named refusal: `run`
exits 3, and no lease, Run or recovery file is created.

| Status | Check |
| --- | --- |
| `authority-unavailable` | a coordinate is missing, the TLS pair is half set or unreadable, or no credential is set |
| `workflow-context` | `GITHUB_EVENT_NAME` is `workflow_dispatch`, `GITHUB_REPOSITORY` and `GITHUB_REF` are the policy's, `GITHUB_WORKFLOW_REF` is `<repository>/<workflow path>@refs/heads/main`, and `GITHUB_RUN_ID` and `GITHUB_RUN_ATTEMPT` are positive numbers |
| `policy-unconfigured` | every coordinate digest is committed |
| `coordinate-mismatch` | each coordinate's digest equals the policy's |
| `model-mismatch` | the build's committed Model guest hashes to the policy's `modelHash` |
| `namespace-missing`, `namespace-unavailable` | `DescribeNamespace` finds the canary namespace registered |

A `policy-unavailable` status means the embedded policy did not decode, which is a build defect.

## The Limits

The policy's Limits cannot be raised by a flag:

| Limit | Value |
| --- | --- |
| Iterations per invocation | 2 |
| Invocation time | 10 minutes, counted from `run`'s start |
| Cleanup reserve | 2 minutes |
| Lease run timeout | 24 hours, a backstop only |
| Progress | 64 KiB, beyond which progress is dropped |

A Run's own budgets are the Runner's: 32 proposals, 1024 events, 30 seconds per wait and a
2-minute run timeout the canary sets. An iteration starts only when enough of the invocation limit
remains for one iteration's worst case (the run timeout, the Driver's close and its release). The
first Run that is not explained ends the invocation, so nothing more runs against production after
an unexplained or refused Run.

## Reading the outcome

### `run`

`run-summary.json` holds `status`, `detail`, `invocation` (`<run id>-<run attempt>`), one entry
per iteration (`activityId`, `status`, and `run`, the published Run's key under the output
directory, when there is one), and `cleanup` (`outcome`, `fenced`, `unverified`). The exit code is
the highest one the invocation reached, and `status` names the outcome that produced it:

| Exit | Status | Meaning | What to do |
| --- | --- | --- | --- |
| 0 | `explained` | every Run is explained, cleanup released the lease and every Run was published | nothing |
| 1 | `unexplained` | the Model could not explain an observation; the Run names the event | replay the Run (`umpire replay --model <guest> --run <path>`) and read the judgment |
| 1 | `refused` | the Model refused a call, or the Driver faulted (for example an unreachable endpoint or a read timeout) | read the Run's last event and the progress log |
| 2 | `lease-unreconciled` | the lease's latest run is open, or closed some way other than a canary termination; nothing ran | see [the lease](#the-lease-lost-iterations-and-reconciliation) |
| 2 | `cleanup-uncertain` | cleanup could not verify every fenced activity closed, so the lease stays held; Runs were still published | read `cleanup.unverified` and the reconcile report |
| 3 | a preflight status | refused before any mutation | fix the precondition |
| 3 | `failed` | a Run could not be made: its fence failed, or the loop could not run; nothing is published for it | read `detail` and the progress log |
| 3 | `no-iteration` | the lease was taken but no Run was made | read `detail` |
| 3 | `interrupted`, `tooling-failure` | the job was cancelled, a Driver did not release, or the recovery record could not be written | read `detail`; reconcile has run |
| 3 | `publication-conflict` | a Run's key already holds other bytes; nothing is overwritten | investigate the output directory; nothing reruns |
| 3 | `publication-unreported` | a Run was published but its publication could not be recorded | the Run stands; do not republish |
| 3 | `publication-failed` | publication failed for another reason | read `detail` |

### Each iteration

Each iteration that made a Run has it in `canary-output/` at
`<model hash>/<seed>/<sha256 of the canonical Run>.binpb`: the `umpire.run.v1.Run` with the model
hash, protocol version, World, seed, every Reset, Propose, Judge and Answer with its exact request
and response, and the verdict. A published Run replays offline with `umpire replay`. Its task
tokens name activities cleanup has already closed, and only the canary credential can use them. A
failed iteration has no Run.

## The lease, lost iterations and reconciliation

The lease is a workflow with a fixed ID and type (`umpire-canary-lease`) on a task queue no worker
polls. Its run ID is the fence. Before each Run starts, the canary signals the Run's fresh activity
ID to the exact lease run (`run-opened`). The lease's history is therefore the server's own list of
every activity the invocation may touch. Each Run's Driver is fenced too: it refuses any start but
that activity's. Cleanup and reconcile act on exactly those IDs, which all begin with
`umpire-canary-` and a UUID, and on nothing else.

- **Released.** Cleanup verifies every fenced activity closed and then terminates the lease with
  the reason `umpire-canary: released`.
- **Held.** If cleanup cannot verify a fenced activity, the lease stays open. The next dispatch
  refuses it as `lease-unreconciled` until reconcile closes the scope. A lease that reached its
  24-hour timeout, or that was closed any way other than a canary termination, is also
  unreconciled.
- **Lost.** If the process dies after an activity was fenced and before its Run was published, that
  iteration is lost. There is no Run, and none is fabricated. The lease stays open.

`reconcile` runs after every `run`. It acts only on the lease run that its own job's recovery file
names, never on whatever lease happens to be live. It never runs the Model or publishes, and it
writes only its own report (`reconcile-summary.json`).

| Exit | Status | Meaning |
| --- | --- | --- |
| 0 | `nothing-to-reconcile` | the job wrote no recovery file because preflight refused before any lease; or the record names no lease because the process died while taking one, in which case a lease may be held and the next dispatch finds it |
| 0 | `reconciled` | every fenced activity is verified closed, or terminated with `umpire-canary: reconciled`, and the lease is closed. After a clean run this closes nothing new |
| 2 | `lease-in-use` | the job found another invocation's lease still open and younger than 12 minutes (the invocation limit plus the reserve); it is left alone |
| 2 | `uncertain` | some fenced activity could not be verified closed, or the lease could not be closed; the lease stays held |
| 3 | `recovery-unreadable`, `tooling-failure`, and the policy or authority statuses | reconcile could not act |

The report lists `fenced`, `closed`, `terminated` and `unverified` activity IDs. It also reports
what happened to the iterations:

- `lost` lists this job's own activities that its recovery record shows fenced and whose Runs it
  does not record as published. After `run` ends `publication-unreported` or `publication-failed`,
  it can name an activity whose Run is in the artifact. Whether a Run was published is decided by
  the files in `canary-output/`, not by `lost`.
- `publicationUnknown` lists, when the job found an earlier invocation's lease, the activities that
  lease fenced. Whether they were published is recorded in the earlier invocation's artifact, which
  `foundArtifact` names (`umpire-production-canary-<run id>-<attempt>`) when the lease's start
  records it.

An activity the server does not find on two reads, taken one RPC timeout apart, never started and
counts as closed. A lease ID the server does not find at all is a clean scope: the first run, or a
lease whose history has aged out.

When `run` dies and the job survives, the same job's `reconcile` step closes the scope at once,
since the lease is the one its own job took, and reports the lost iterations. When the runner itself
is lost, dispatch again. The new job's `run` finds the old lease and refuses it as
`lease-unreconciled`. Its `reconcile` then closes the fenced activities and records the scope
reconciled, as long as the old lease run is older than 12 minutes; otherwise it reports
`lease-in-use`, and you dispatch again later. Once the scope is reconciled, dispatch once more to
run the canary. Reconciliation never dispatches a Run.

## Clearing an uncertain scope

When `reconcile` reports `uncertain`:

1. Take the activity IDs under `unverified` in `reconcile-summary.json`, or under
   `cleanup.unverified` in `run-summary.json`. Both summaries also list every fenced ID, so they
   remain available after the lease's history ages out.
2. Terminate each listed standalone activity by hand in the canary namespace
   (`TerminateActivityExecution`). Close only the listed IDs.
3. Dispatch the workflow again. Its `run` refuses the held lease as `lease-unreconciled`, and its
   `reconcile` verifies that the activities are closed and records the scope reconciled. If it
   reports `lease-in-use`, the lease run is younger than 12 minutes; dispatch again later.
4. Dispatch again to run the canary.

Do not terminate the lease by hand. A lease closed with any other reason is unreconciled. The next
reconcile still repairs it, by verifying the fenced activities and then starting and at once
terminating a fresh lease run with the reconciled reason.

## Published Runs and the retained artifact

**Runs are not self-authenticating.** A Run's bytes, and the key that names them, are canonical
and inspectable, but nothing in them proves that the protected workflow produced them. Whether a
canary Run is authentic depends on the channel it came through: the protected environment, the
`main`-only workflow and the Actions artifact.

The artifact is named `umpire-production-canary-<run id>-<run attempt>`. It holds only the
published Runs, both summaries and both progress logs. It carries no credential and no gRPC
target. Download it with `gh run download <run id> -n umpire-production-canary-<run id>-<run attempt>`.
Actions deletes artifacts after the repository's retention period, so copy any artifact you need to
keep to durable storage unchanged, under each Run's own key, since the key is the Run's identity.

## Where the code lives

Everything canary-specific lives here. Umpire never imports `tools/canary`.

| Package | Owns |
| --- | --- |
| `policy` | the embedded policy and its strict decoder |
| `authority` | the credential, the transport and the Redactor |
| `preflight` | the checks above |
| `recovery` | the mode-0600 recovery record |
| `controller` | the lease, fencing, the serial Runner loop, cleanup, publication, `Invoke` and `Reconcile` |
| `cmd/umpire-canary` | the binary |
| `testharness` | the `canary_harness` build's test policy, transport and crash hook |

The `canary_harness` build tag compiles a separate harness binary with a test policy, a plaintext
transport and the `harness` authority class. The live tests in `tests/testpilot_canary_test.go`
(`TestTestpilotCanary*`) run it against the test cluster. `build_test.go` pins that the untagged
build has no override path. The `canary` job in `.github/workflows/umpire.yml` runs the canary's
unit tests and the harness build's tests.
