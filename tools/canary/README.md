# Production canary

`tools/canary` runs one fixed, bounded, no-fault Case against a dedicated canary namespace in
production and assesses each Run with Umpire's Claim Assessment. The Case is the Caller Model's
`nexusCallerCanary.syncCompletion`, pinned in `casebinding/testdata/`. Each invocation makes at
most two serial Runs under one lease. Each Run that reaches a decision produces an fn-26 receipt
and a canary provenance document beside it.

**Nothing here authorizes a release.** A canary receipt says that one Run of one Case, against one
dedicated namespace, was accepted, rejected or incomplete under the `production-canary` Evaluation
Profile. It is not release evidence, not a rollout gate and not a statement about customer
traffic. Every provenance carries `releaseEligibility: false`, the only value its decoder accepts.
This runbook gives no schedule, and the canary never reruns itself. Every invocation is an
operator's manual dispatch.

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
- [Receipts, provenance and the retained artifact](#receipts-provenance-and-the-retained-artifact)
- [Where the code lives](#where-the-code-lives)

## What the canary may touch

The canary holds one credential. It is a namespace writer on the canary namespace and has no other
permission. It arrives as a TLS client certificate and key (`UMPIRE_CANARY_TLS_CERT`,
`UMPIRE_CANARY_TLS_KEY`), an API key (`UMPIRE_CANARY_API_KEY`), or both. Every call the canary
makes is scoped to the namespace, except the SDK client's `GetSystemInfo` when it dials:

- the Case's own RPCs, worker and Nexus handler;
- the lease workflow's start, signal and termination;
- history reads of the lease and of the workflows it fenced;
- one `DescribeNamespace` in preflight.

The canary never lists or reads Nexus endpoints, since that needs cluster admin. It changes no
deployment, configuration or routing, injects no fault and sends no customer traffic.

The target's five coordinates come from the environment. `UMPIRE_CANARY_GRPC` is the whole
`host:port` value. The other four are `UMPIRE_CANARY_NAMESPACE`, `UMPIRE_CANARY_TASK_QUEUE`,
`UMPIRE_CANARY_HANDLER_QUEUE` and `UMPIRE_CANARY_ENDPOINT` (the Nexus endpoint's name). The Redactor
removes the credential and every coordinate from progress, summaries and logs. The coordinates are
not secrets; the credential is. The policy stores the coordinates as unsalted SHA-256 digests, so
a digest confirms a guessed name but does not hide one.

## Operator preconditions

The repository cannot check these. An operator sets each one up before the first dispatch and keeps
it true afterwards.

1. **The `production-canary` GitHub environment is protected.** Its deployment branches are
   restricted to `main`, and it requires reviewers. A `workflow_dispatch` runs the workflow file
   and code of the branch it is dispatched on, so this protection alone decides who receives the
   credential. The workflow's `if: github.ref == 'refs/heads/main'` and preflight's ref checks are
   defense in depth.
2. **The environment holds the credential and the coordinates** as secrets with the eight names
   listed above. Set a TLS pair, an API key, or both.
3. **The credential is a namespace writer on the canary namespace and nothing else.**
4. **The Nexus endpoint targets the canary namespace and the handler queue.** Where the deployment
   has an allowed-caller-namespaces setting on the endpoint, it allows the canary namespace. The
   canary cannot read the endpoint. A repointed endpoint shows up only after the fact: the Run does
   not observe the handler's reply and ends incomplete, with a receipt (see
   [Reading the outcome](#reading-the-outcome)).
5. **The canary namespace's retention is at least 30 days.** The lease's history, which says which
   workflows it fenced, ages out with the namespace's retention. The Case's workflows set no
   timeout, so only reconcile closes an orphan. Retention must be longer than the longest gap you
   allow before reconciling.
6. **The coordinate digests are committed to the policy** in a reviewed pull request to `main`.
   Until then preflight refuses every dispatch as `policy-unconfigured`.

## Configuring the policy

`policy/production-canary.json` is the canary's one policy. It is embedded in the binary and
decoded strictly. It pins the Case's identity, the Profile names, the authority class
(`protected-workflow`), the lease, the repository, the trusted ref, the workflow path and the Limits.
It is committed with every coordinate set to the literal `unconfigured`.

To configure it, digest each coordinate's exact value as the environment holds it, with no
trailing newline:

```bash
printf '%s' "$VALUE" | shasum -a 256 | cut -d' ' -f1
```

Replace each `unconfigured` under `coordinates` (`grpc`, `namespace`, `taskQueue`, `handlerQueue`,
`nexusEndpoint`) with its lower-case hex digest. Open a pull request to `main` and have it reviewed.
Follow the same procedure for any later coordinate change. A policy with some coordinates configured
and others not is still `policy-unconfigured`. A digest that does not match the environment's value
is `coordinate-mismatch`. No flag can override the policy, and the untagged build has no other
policy.

## Invoking the canary

Dispatch `.github/workflows/umpire-production-canary.yml` (**Umpire production canary**) on `main`
from the Actions tab, or with `gh workflow run umpire-production-canary.yml --ref main`. An
environment reviewer then approves the deployment. The workflow:

- runs on `workflow_dispatch` only, with `contents: read` and no other permission;
- runs in one concurrency group, `umpire-production-canary`, which never cancels a job in progress,
  so no two jobs overlap;
- has a 30-minute job timeout and uses a fresh GitHub-hosted runner, so nothing survives from one
  dispatch to the next;
- builds `umpire-canary` (`make canary-build`), then runs `umpire-canary run`;
- always runs `umpire-canary reconcile` and always uploads `canary-output/`, whatever `run` did.

Each mode takes only `--output <dir>` and `--recovery <file>`. Neither mode accepts a Case, target,
Driver, checker, retry, executable, endpoint, credential or release option. `run` writes
`run-summary.json` (stdout) and `run-progress.log` (stderr), and `reconcile` writes
`reconcile-summary.json` and `reconcile-progress.log`, all in the uploaded directory. The recovery
file lives in the runner's temporary directory with mode 0600. It is never uploaded and disappears
with the runner.

## What preflight checks

Preflight runs before any mutation, in this order. The first failure is a named refusal: `run`
exits 3, and no lease, Run, receipt or recovery file is created.

| Status | Check |
| --- | --- |
| `authority-unavailable` | a coordinate is missing, the TLS pair is half set or unreadable, or no credential is set |
| `workflow-context` | `GITHUB_EVENT_NAME` is `workflow_dispatch`, `GITHUB_REPOSITORY` and `GITHUB_REF` are the policy's, `GITHUB_WORKFLOW_REF` is `<repository>/<workflow path>@refs/heads/main`, and `GITHUB_RUN_ID` and `GITHUB_RUN_ATTEMPT` are positive numbers |
| `policy-unconfigured` | every coordinate digest is committed |
| `coordinate-mismatch` | each coordinate's digest equals the policy's |
| `case-mismatch` | the pinned Case has the policy's identity and prepares under the canary's hand-authored Driver Profile with the tree's catalog |
| `namespace-missing`, `namespace-unavailable` | `DescribeNamespace` finds the canary namespace registered |

A `policy-unavailable` status means the embedded policy or Evaluation Profile did not decode, which
is a build defect.

## The Limits

The policy's Limits cannot be raised by a flag:

| Limit | Value |
| --- | --- |
| Iterations per invocation | 2 |
| Invocation time | 10 minutes, counted from `run`'s start |
| Cleanup reserve | 2 minutes |
| Lease run timeout | 24 hours, a backstop only |
| Progress | 64 KiB, beyond which progress is dropped |

A Run's own duration, RPC, worker and event ceilings come from the Temporal Profile (30 seconds of
Run and 20 of cleanup). The evidence and receipt caps are fn-26's and are recorded in each receipt.
An iteration starts only when enough of the invocation limit remains for one iteration's worst
case. The first iteration that is not accepted ends the invocation, so nothing more runs against
production after a rejected or incomplete Run.

## Reading the outcome

### `run`

`run-summary.json` holds `status`, `detail`, `invocation` (`<run id>-<run attempt>`), one entry
per iteration (`runId`, `status`, and the `receipt` and `provenance` identities when there are
any), and `cleanup` (`outcome`, `fenced`, `unverified`). The exit code is the highest one the
invocation reached, and `status` names the outcome that produced it:

| Exit | Status | Meaning | What to do |
| --- | --- | --- | --- |
| 0 | `accepted` | every iteration's receipt is accepted, cleanup released the lease and everything was published | nothing |
| 1 | `rejected` | a Run's Verdict or its assessment rejects the claim; a proved violation stays rejected | read the receipt's `decision`, `reasons` and `verdict` |
| 1 | `incomplete` | the Run was inconclusive, or its evidence did not support the claim, for example a repointed endpoint whose handler reply was never observed | read the receipt's `reasons`; check preconditions 4 and 5 |
| 2 | `lease-unreconciled` | the lease's latest run is open, or closed some way other than a canary termination; nothing ran | see [the lease](#the-lease-lost-iterations-and-reconciliation) |
| 2 | `cleanup-uncertain` | cleanup could not verify every fenced workflow closed, so the lease stays held; receipts were still published | read `cleanup.unverified` and the reconcile report |
| 3 | a preflight status | refused before any mutation | fix the precondition |
| 3 | `unconstructible` | a Run errored, or fn-26 did not admit its record; there is no receipt | read `detail` and the progress log |
| 3 | `no-iteration` | the lease was taken but no Run was made | read `detail` |
| 3 | `interrupted`, `tooling-failure` | the job was cancelled, a Driver did not release, or the recovery record could not be written | read `detail`; reconcile has run |
| 3 | `publication-conflict` | a receipt's or provenance's name already holds other bytes; nothing is overwritten | investigate the output directory; nothing reruns |
| 3 | `publication-unreported` | a document was published but its publication could not be recorded | the documents stand; do not republish |
| 3 | `publication-failed` | publication failed for another reason | read `detail` |

A satisfied Verdict can still end rejected or incomplete, because the assessment also weighs
trust, Known Gaps and unsupported rules. Under `production-canary` every Known Gap blocks, and an
unsupported rule rejects.

### Each iteration

Each iteration that reached a decision has a receipt and a provenance in `canary-output/`:

- `<receipt identity>.json` is the fn-26 receipt, byte for byte. It holds the Evaluation Profile,
  the Case, the Run (its ID, Driver identity, disposition and cleanup), the Verdict with each
  rule's status and supporting event sequence numbers, the `decision`, its ordered `reasons`,
  unsupported rules, Known Gaps and caps.
- `<provenance identity>.provenance.json` is the canary provenance. It holds the receipt identity,
  the Evaluation Profile identity, the authority class, the workflow ref and run, the coordinate
  digests, the lease ID's digest and fence, the invocation, iteration and Run, the Limits, the
  iteration's cleanup and the invocation's cleanup outcome (`released` or `uncertain`), the
  isolation statement, every workflow ID the lease fenced, and `releaseEligibility: false`.

The two documents name each other and the same Run. Publication checks that before it writes
either one. An unconstructible iteration has neither.

## The lease, lost iterations and reconciliation

The lease is a workflow with a fixed ID and type (`umpire-canary-lease`) on a task queue no worker
polls. Its run ID is the fence. Before each Run opens, the canary signals the Run's workflow ID to
the exact lease run (`run-opened`). The lease's history is therefore the server's own list of every
workflow the invocation may touch. Cleanup and reconcile act on exactly those IDs, which all begin
with `testpilot.run.`, and on nothing else.

- **Released.** Cleanup verifies every fenced workflow closed and then terminates the lease with
  the reason `umpire-canary: released`.
- **Held.** If cleanup cannot verify a fenced workflow, the lease stays open. The next dispatch
  refuses it as `lease-unreconciled` until reconcile closes the scope. A lease that reached its
  24-hour timeout, or that was closed any way other than a canary termination, is also
  unreconciled.
- **Lost.** If the process dies after a Run was created and before the Run was published, that
  iteration is lost. There is no receipt, and none is fabricated. The lease stays open.

`reconcile` runs after every `run`. It acts only on the lease run that its own job's recovery file
names, never on whatever lease happens to be live. It never prepares, runs, assesses or publishes,
and it writes only its own report (`reconcile-summary.json`).

| Exit | Status | Meaning |
| --- | --- | --- |
| 0 | `nothing-to-reconcile` | the job wrote no recovery file because preflight refused before any lease; or the record names no lease because the process died while taking one, in which case a lease may be held and the next dispatch finds it |
| 0 | `reconciled` | every fenced workflow is verified closed, or terminated with `umpire-canary: reconciled`, and the lease is closed. After a clean run this closes nothing new |
| 2 | `lease-in-use` | the job found another invocation's lease still open and younger than 12 minutes (the invocation limit plus the reserve); it is left alone |
| 2 | `uncertain` | some fenced workflow could not be verified closed, or the lease could not be closed; the lease stays held |
| 3 | `recovery-unreadable`, `tooling-failure`, and the policy or authority statuses | reconcile could not act |

The report lists `fenced`, `closed`, `terminated` and `unverified` workflow IDs. It also reports
what happened to the iterations:

- `lost` lists this job's own Runs that its recovery record shows opened and not recorded as
  published. After `run` ends `publication-unreported` or `publication-failed`, it can name a Run
  whose receipt is in the artifact. Whether a Run was published is decided by the files in
  `canary-output/`, not by `lost`, and not by `run-summary.json`, which lists each decided
  iteration's identities before publication runs.
- `publicationUnknown` lists, when the job found an earlier invocation's lease, the Runs that lease
  fenced. Whether they were published is recorded in the earlier invocation's artifact, which
  `foundArtifact` names (`umpire-production-canary-<run id>-<attempt>`) when the lease's start
  records it.

A workflow the server does not find on two reads, taken one RPC timeout apart, never started and
counts as closed. A lease ID the server does not find at all is a clean scope: the first run, or a
lease whose history has aged out.

When `run` dies and the job survives, the same job's `reconcile` step closes the scope at once,
since the lease is the one its own job took, and reports the lost iterations. When the runner itself
is lost, dispatch again. The new job's `run` finds the old lease and refuses it as
`lease-unreconciled`. Its `reconcile` then closes the fenced workflows and records the scope
reconciled, as long as the old lease run is older than 12 minutes; otherwise it reports
`lease-in-use`, and you dispatch again later. Once the scope is reconciled, dispatch once more to
run the canary. Reconciliation never dispatches a Run.

## Clearing an uncertain scope

When `reconcile` reports `uncertain`:

1. Take the workflow IDs under `unverified` in `reconcile-summary.json`, or under
   `cleanup.unverified` in `run-summary.json`. The provenance and the report also list every
   fenced ID, so they remain available after the lease's history ages out.
2. Close each listed workflow by hand in the canary namespace, for example with
   `temporal workflow terminate --namespace <canary namespace> --workflow-id <id>`. Close only the listed IDs.
3. Dispatch the workflow again. Its `run` refuses the held lease as `lease-unreconciled`, and its
   `reconcile` verifies that the workflows are closed and records the scope reconciled. If it
   reports `lease-in-use`, the lease run is younger than 12 minutes; dispatch again later.
4. Dispatch again to run the canary.

Do not terminate the lease by hand. A lease closed with any other reason is unreconciled. The next
reconcile still repairs it, by verifying the fenced workflows and then starting and at once
terminating a fresh lease run with the reconciled reason.

## Receipts, provenance and the retained artifact

**Receipts are not self-authenticating.** A receipt's bytes, and the identity that names them, are
inspectable and canonical, but nothing in them proves that the protected workflow produced them.
Whether a canary receipt is authentic depends on the channel it came through: the protected
environment, the `main`-only workflow and the Actions artifact. Every canary receipt is published
beside a provenance whose `releaseEligibility` is always `false`, and no canary document can claim
otherwise.

The artifact is named `umpire-production-canary-<run id>-<run attempt>`. It holds only the
receipts, the provenance documents, both summaries and both progress logs. It carries no
credential, raw coordinate, payload or recorded Run. Recorded Runs hold whole history events, so
they stay in memory and are never written. Download the artifact with
`gh run download <run id> -n umpire-production-canary-<run id>-<run attempt>`. Actions deletes
artifacts after the repository's retention period, so copy any artifact you need to keep to
durable storage unchanged. Keep the receipt and provenance files together and under their own
names, since each name is the document's identity.

## Where the code lives

Everything canary-specific lives here. Umpire never imports `tools/canary`.

| Package | Owns |
| --- | --- |
| `policy` | the embedded policy and its strict decoder |
| `casebinding` | the pinned Case and the hand-authored Driver Profile |
| `authority` | the credential, the transport and the Redactor |
| `preflight` | the checks above |
| `recovery` | the mode-0600 recovery record |
| `controller` | the lease, fencing, the serial loop, cleanup, `Invoke` and `Reconcile` |
| `assessment` | fn-26 admission in memory, the canary Evaluation Profile and the provenance |
| `publication` | exclusive publication of each receipt and its provenance |
| `cmd/umpire-canary` | the binary |
| `testharness` | the `canary_harness` build's test policy, transport and crash hook |

The `canary_harness` build tag compiles a separate harness binary with a test policy, the
`canary-harness` Evaluation Profile and the `harness` authority class. The live tests in
`tests/testpilot_canary_test.go` and `tests/testpilot_canary_lifecycle_test.go`
(`TestTestpilotCanary*`) run it against the test cluster.
`build_test.go` pins that the untagged build has no override path, so a harness receipt is never a
production receipt. `make umpire-check-regression` and the `canary` job in
`.github/workflows/umpire.yml` run the canary's tests. `make canary-gen-case` and
`make canary-check-case` regenerate and check the pinned Case.
