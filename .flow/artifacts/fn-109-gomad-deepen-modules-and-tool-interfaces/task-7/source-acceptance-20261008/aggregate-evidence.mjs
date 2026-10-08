import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const directory = path.join(root, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const sha = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = name => JSON.parse(fs.readFileSync(path.join(directory, name)));
const frozen = 'c8bfba0aed3ba328aae4ffb7ad317b3cd051e379a66b54dabc6701676ac335e9';
const explanations = {
  'baseline-owner-runner': ['baseline-red', 'Actual ordinary pre-edit owner/Runner selection. Eleven host-profile refusals remain red; five named outcomes passed.'],
  'baseline-boundaries': ['baseline-green', 'Actual pre-edit architecture/exact-edge source assertions.'],
  'composition-red': ['compile-red', 'Missing private per-call composition symbols before implementation. Zero named outcomes; package build failure.'],
  'source-owner-first': ['intermediate-source-green', 'First source-profile owner assertions before later tests and lint fixes. Final owner execution supersedes this source state.'],
  'source-owner-final': ['intermediate-source-green', 'Expanded owner assertions before final test-only lint fixes. Final owner execution supersedes this source state.'],
  'source-runner-portable': ['intermediate-source-green', 'Focused Runner and portable assertions before final test-only lint fixes. Final source-runner-portable-final reran the same selection.'],
  'source-runner-portable-final': ['final-source-green', 'Current failure selector, mutation, portable plans, mounts and document validation on the frozen graph.'],
  'source-existing-owner': ['final-source-green', 'Unchanged existing preparation/portable-root assertion bodies plus new source controls. Source profile and test-only pinned-driver inputs are separately bound.'],
  'adapter-attachment-mutation-red': ['compile-red', 'Test-only fault overlay left selectedAdapters unused. No assertion executed. Retained before corrected fault input.'],
  'adapter-attachment-mutation-assertion-red': ['assertion-red', 'Corrected test-only missing-attachment fault failed TestPrepareCompositionOrderAndCleanupIsolation and TestPrepareCustomPreparerSkipsAdaptersAndValidates in ./internal/preparation. Production source unchanged. Independent portable-plan coverage is recorded in source-existing-owner, current-callers-owned-cache and source-runner-portable-final, without this mutation.'],
  'adapter-attachment-final-green': ['final-source-green', 'TestPrepareCompositionOrderAndCleanupIsolation and TestPrepareCustomPreparerSkipsAdaptersAndValidates in ./internal/preparation passed with frozen production source and no fault overlay.'],
  'original-callers-first': ['compile-red', 'Comparison harness imported an installation API absent from coherent original dependencies. Zero named outcomes.'],
  'original-callers-corrected': ['compile-red', 'Corrected identical harness exposed two original syscall.Dup2 references unavailable on ARM64. Zero named outcomes.'],
  'original-callers-compatible': ['fixture-environment-red', 'Original caller compiled with approved fail-closed standard-library-only overlays. Adapter cases failed because sanitized subprocesses used full HOME source cache. Two named passes and three named failures retained.'],
  'original-callers-owned-cache': ['original-caller-source-green', 'Matched original actual Explore/portable-plan calls after approved identical driver selected owned source-cache.'],
  'current-callers-owned-cache': ['current-caller-source-green', 'Matched current actual Explore/portable-plan calls with identical fixed inputs and initial cache state. Eight snapshots equal.'],
  'original-portable-missing-root': ['known-original-caller-assertion-red', 'Actual original portable caller fails its adapter-root assertion. Already-integrated task7 bundle-root migration passes the unchanged current assertion. No old pass or blanket preservation waiver.'],
  'lint-code-fast': ['introduced-lint-red', 'Private test switch triggered staticcheck; fixed before final source freeze.'],
  'lint-code-fast-final': ['introduced-lint-red', 'Private test switch omitted enumerated stage cases; fixed before final source freeze.'],
  'lint-code-fast-clean': ['final-changed-lines-lint-green', 'Mandatory fixes-disabled make lint-code-fast on final source. This does not supersede unfiltered affected-package findings.'],
  'lint-affected-unfiltered': ['retained-lint-red', 'Forty exact unchanged admission findings, zero new task7 findings. Every path, line, rule, origin and disposition is in lint-attribution.json.'],
  'source-list-linux': ['wrong-build-surface-red', 'Unrestricted ./... included runtime overlay/compiler fixtures with unavailable standard-library internal imports. Zero assertions; no static pass.'],
  'source-vet-linux': ['wrong-build-surface-red', 'Same unrestricted build-surface failure. Superseded only for legitimate static scope by unchanged architecture.Discover inventories.'],
  'final-architecture-host-vet': ['final-source-static-green', 'Architecture/public signature/external consumer assertions and unchanged HostPackageVet on supported darwin/arm64, linux/amd64 and actual linux/arm64 inventories. Cross-built sources were not executed.'],
  'source-list-inventory-darwin': ['static-list-green', 'Exact full nonempty authoritative darwin/arm64 inventory. List is static source coverage.'],
  'source-list-inventory-linux': ['static-list-green', 'Exact full nonempty authoritative linux/amd64 inventory. List is static source coverage.'],
  'host-errortype': ['static-errortype-green', 'Affected-package host source errortype. No named-test parser applies.'],
  'generated-check': ['check-only-validation-green', 'Nine repository validate commands ran check-only. Text named-outcome parser unavailable; compatibility host-pack wrapper internally skips unsupported actual ARM64. No native host-pack pass.'],
  'qualification-ordinary': ['qualification-red', '213 named passes and 11 named failures. Ten genuine analysis/inspection assertion gaps belong to task8; one portable prune assertion is resolved only by the separately admitted exact runtime command.'],
  'link-count-diagnostic': ['filesystem-diagnostic', 'Owned workspace os.Root.Lstat and os.Lstat observed Nlink 3/3/3 while removed paths were absent. No test pass claimed.'],
  'link-count-diagnostic-tmpfs': ['filesystem-diagnostic', 'Privately owned tmpfs capability probe observed Nlink 3/2/1 through both APIs and absent deleted paths. Admission remains one prune selector.'],
  'qualified-prune-tmpfs-exact': ['assertion-red', 'TMPDIR-only changed input did not move t.TempDir because this SDK testing uses GOTMPDIR. Same prune assertion still failed.'],
  'prune-test-binary-compile': ['compile-green', 'Compiled once with workspace compiler GOCACHE/GOTMPDIR. Compilation alone executes zero assertions.'],
  'qualified-prune-runtime-tmpfs-exact': ['converter-setup-red', 'Exit2 SIGBUS in telemetry mapped-file counter before any test. Zero named outcomes is not green.'],
  'qualified-prune-runtime-telemetry-exact': ['converter-setup-red', 'Exit127 missing guessed test2json path; runtime input binder also failed ENOENT. Zero named outcomes.'],
  'qualified-prune-runtime-resolved-exact': ['one-exact-portable-assertion-green', 'Resolved hash-bound converter with converter-only owned TEST_TELEMETRY_DIR; unchanged compiled child receives only admitted TMPDIR/GOTMPDIR filesystem changes and telemetry unset. One named pass.'],
  'ordinary-public-guard-runner': ['ordinary-source-green', 'Unchanged Default public refusal guards remain first on actual linux/arm64. Runner context failure assertions also pass. Guard passes do not prove downstream success.'],
  'tool-environment-final': ['environment-read', 'Revalidated actual platform, UID, tools, source cache locations, free space, mounts and absence of patched .toolchain/bin/go.'],
  'preservation-verify': ['preservation-verifier-green', 'Original bytes, historical snapshots, actual caller equality, runtime inputs, frozen dependency graph, protected user files and git diff --check verified.'],
  'task-status-read': ['lifecycle-read', 'Read-only flowctl show confirms in_progress. Worker performed no lifecycle mutation.'],
};
const receipts = fs.readdirSync(directory).filter(name => name.endsWith('-receipt.json')).map(name => ({ name, value: read(name) })).sort((a, b) => a.value.started.localeCompare(b.value.started));
const commands = receipts.map(({name, value: r}) => {
  const disposition = explanations[r.name];
  if (!disposition) throw new Error(`Missing command disposition: ${r.name}`);
  const bytes = fs.readFileSync(path.join(directory, r.log));
  if (sha(bytes) !== r.log_sha256) throw new Error(`Raw log changed: ${r.log}`);
  if (!r.source_unchanged || !r.tools_unchanged || r.controls_unchanged === false) throw new Error(`Inputs changed: ${r.name}`);
  const text = bytes.toString();
  const events = text.split('\n').flatMap(line => { try { const e = JSON.parse(line); return e.Action && e.Package ? [e] : []; } catch { return []; } });
  const jsonCommand = /(?:\bgo test[^\n]* -json|\bgo test -[^\n]*\b-json|test2json)/.test(r.command);
  const named = events.filter(e => e.Test && ['pass', 'fail', 'skip'].includes(e.Action));
  const packageOutcomes = events.filter(e => !e.Test && ['pass', 'fail', 'skip', 'build-fail'].includes(e.Action));
  const jsonCounts = Object.fromEntries(['pass','fail','skip'].map(a => [a, named.filter(e => e.Action === a).length]));
  if (JSON.stringify(jsonCounts) !== JSON.stringify(r.counts)) throw new Error(`Outcome count mismatch: ${r.name}`);
  return {
    name: r.name, command: r.command, exit_code: r.exit_code, signal: r.signal, error: r.error,
    started: r.started, ended: r.ended, elapsed_seconds: r.elapsed_seconds,
    classification: disposition[0], disposition: disposition[1], receipt: name,
    log: r.log, log_sha256: r.log_sha256, source_manifest: r.source_manifest,
    source_tree_sha256: r.source_tree_sha256, frozen_candidate: r.source_tree_sha256 === frozen,
    control_manifest: r.control_manifest ?? null,
    named_outcome_counts: jsonCommand ? jsonCounts : null,
    named_outcome_parser: jsonCommand ? 'Go JSON events; zero outcomes on setup/build failure remains zero' : 'unavailable/not applicable; receipt zero counters do not count assertions',
    package_outcomes: packageOutcomes.map(e => ({package: e.Package, action: e.Action, elapsed: e.Elapsed ?? null})),
    build_failure_diagnostics: text.split('\n').filter(line => /\[build failed\]|undefined:|no required module provides package|use of internal package .*not allowed|no such file or directory|SIGBUS/.test(line)).slice(0, 16),
  };
});
const receipt = name => { const r = receipts.find(x => x.value.name === name); if (!r) throw new Error(name); return r.value; };
const assertionCoverage = [
  ['single sequence and complete identity before validation', 'source-existing-owner', 'TestPrepareCompositionOrderAndCleanupIsolation'],
  ['adapter/target/validation/cleanup error identity, cleanup isolation and precedence', 'source-existing-owner', 'TestPrepareCompositionFailuresRetainIdentityAndCleanup'],
  ['real target kind/source/arguments/toolchain/platform validation', 'source-existing-owner', 'TestPrepareCustomTargetIdentityValidation'],
  ['custom success bypasses selection, empty nonnil adapters and real validation', 'source-existing-owner', 'TestPrepareCustomPreparerSkipsAdaptersAndValidates'],
  ['target error stage and errors.Is identity', 'source-existing-owner', 'TestPrepareKeepsTargetFailureStageAndIdentity'],
  ['external module/local replacement', 'source-existing-owner', 'TestPrepareExternalModuleWithLocalReplacement'],
  ['independent adapter identities and unique workspace cleanup', 'source-existing-owner', 'TestPrepareIndependentAdapterTargetsHaveStableIdentityAndCleanWorkspaces'],
  ['durable-root sharing, caller sibling, initial absent cache, counted build/hit and lock release', 'source-existing-owner', 'TestPrepareSourceFreshCacheAndSharedDurableRoot'],
  ['real missing/invalid sums, replacement conflict, adapter-error cleanup, original module files', 'source-existing-owner', 'TestPrepareSourceAdapterErrorsCleanWorkspace'],
  ['cleanup success/failure and primary error precedence', 'source-existing-owner', 'TestPrepareReportsCleanupFailureAndKeepsPrimaryFailure'],
  ['portable bundle root exists before actual adapter selection', 'source-existing-owner', 'TestCreateCampaignPlanPreparesAdapterAfterBundleRootExists'],
  ['actual public current Explore and CreateCampaignPlan fresh/cache plus full Prepared/RecordTarget', 'current-callers-owned-cache', 'TestPreparationSourceCallerPreservation'],
  ['matched original actual caller preservation and stop before target execution', 'original-callers-owned-cache', 'TestPreparationSourceCallerPreservation'],
  ['journal failure, cancelled and overall-timeout classifications', 'source-runner-portable-final', 'TestRunPreparationFailureLeavesClassifiedPartial'],
  ['changed prepared binary rejected before failure publication', 'source-runner-portable-final', 'TestRunRejectsPreparedTargetMutationBeforeFailurePublication'],
  ['canonical prepared portable bundle', 'source-runner-portable-final', 'TestCreateCampaignPlanPublishesCanonicalPreparedTargetBundle'],
  ['dynamic/early-stop refusal, portable mounts, invalid document and changed mount refusal', 'source-runner-portable-final', 'TestCreateCampaignPlanRejectsDynamicallyDiscoveredOrEarlyStopWork|TestCreateCampaignPlanReadOnlyMountsArePortableAndDetachedFromTheirSource|TestOpenCampaignPlanRejectsNoncanonicalAndInvalidDocuments|TestRunCampaignShardRejectsChangedReadOnlyMountBeforeExecution'],
  ['architecture owner/exact edges/public signatures/private seams/external consumer', 'final-architecture-host-vet', 'TestPackageArchitecture|TestExactModuleEdges|TestPublicPackagesDoNotExportTypeAliases|TestPublicPackagesDoNotExportForwardingAliases|TestArchitecturePublicSignatureFixtures|TestRunnerRequestsCompileInExternalModule|TestRunnerExecutionInjectionIsPrivate'],
  ['unchanged production public unsupported-host guard first', 'ordinary-public-guard-runner', 'TestPortableProfilePublicGuardsRemainFirst'],
  ['exact portable prune assertion on admitted filesystem', 'qualified-prune-runtime-resolved-exact', 'TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact'],
].map(([requirement, command, names]) => {
  const r = receipt(command);
  const outcomes = r.observations.filter(e => names.split('|').some(name => e.Test === name || e.Test.startsWith(`${name}/`)));
  if (r.exit_code !== 0 || !outcomes.length || outcomes.some(e => e.Action !== 'pass') || r.source_tree_sha256 !== frozen) throw new Error(`Incomplete final assertion coverage: ${requirement}`);
  return {requirement, command, receipt: `${command}-receipt.json`, named_assertions: outcomes.map(e => e.Test), source_tree_sha256: frozen};
});
const lint = read('lint-attribution.json');
const lintPaths = Object.values(lint.findings.reduce((out, item) => {
  const entry = out[item.path] ??= {path: item.path, findings: 0, origin_commits: [], source_sha256: item.source_sha256, disposition: item.disposition};
  entry.findings++;
  if (!entry.origin_commits.includes(item.origin_commit)) entry.origin_commits.push(item.origin_commit);
  return out;
}, {}));
const qualificationGaps = receipt('qualification-ordinary').observations.filter(e => e.Action === 'fail' && e.Package.endsWith('/qualification/analysis')).map(e => ({test: e.Test, owner: 'fn-109-gomad-deepen-modules-and-tool-interfaces.8 / retained analysis contract', status: 'source assertion unproved; ordinary host refusal retained', accepted_predecessor_pass_reuse: false}));
if (qualificationGaps.length !== 10) throw new Error('Expected exact ten analysis gaps');
const inventory = read('static-inventories.json').map(i => ({platform: i.platform, package_count: i.package_count, inventory_sha256: i.inventory_sha256, list_command: i.list_command, list_execution: i.platform.os === 'linux' && i.platform.arch === 'arm64' ? 'inventory discovery inside HostPackageVet; no separate expanded list command claimed' : 'new explicit enumerated list command', vet: i.vet}));
const head = execFileSync('git', ['rev-parse', 'HEAD'], {cwd: root, encoding: 'utf8'}).trim();
if (!['b4602685b3184387cf2d713178095247f0c11d8f', '34b398ecdadc9d79c64dec8d30dd9999b29ed66f'].includes(head)) throw new Error('Unexpected root history change');
if (read('task-status-read.log').status !== 'in_progress') throw new Error('Unexpected task state');
const evidence = {
  task: 'fn-109-gomad-deepen-modules-and-tool-interfaces.7', status: 'in_progress', tier: 'session (jev-unavailable(no_key))',
  workspace: root, branch: 'gomad', base_commit: 'b4602685b3184387cf2d713178095247f0c11d8f', root_review_base: 'f73e83c9d3a33931e3a28d0b1ba5f39a5af07370', commits: [], prs: [],
  tests: commands.map(c => c.command),
  ownership: 'Root alone commits, conducts independent formal review and mutates Flow/milestones. Worker source is coherent and uncommitted; no acceptance or review verdict issued.',
  frozen_candidate: {source_tree_sha256: frozen, manifest: `source-${frozen}.json`, preservation: 'preservation.json', tools_environment: 'tool-environment-final-receipt.json'},
  baseline: {state: 'red', before_edits: ['baseline-owner-runner-receipt.json', 'baseline-boundaries-receipt.json'], ordinary_failure: '11 host-profile refusals; exact named outcomes retained', canonical_quick_toolchain: 'Patched .toolchain/bin/go unavailable. Stock pinned ordinary/source controls do not claim the canonical native full-host gates.'},
  execution: {receipt_commands: commands.length, all_receipt_commands_terminal: true, commands,
    helper_failures_without_raw_receipt: [
      {helper: 'aggregate-evidence.mjs initial coverage-name lookup', status: 'failed', exit_code: 1, reason: 'Coverage map misspelled actual caller test name. Fail-closed lookup rejected missing outcomes; corrected only map spelling to TestPreparationSourceCallerPreservation without rerunning tests.', receipt: null, raw: 'worker tool transcript only; not a test outcome'},
      {helper: 'preservation.mjs initial parser', status: 'failed', reason: 'Used Prepared instead of the actual prepared_from_validated_caller_plan JSON tag; corrected parser input access without rerunning any tests.', receipt: null, raw: 'worker tool transcript only; no test outcomes or fabricated exit receipt'},
      {helper: 'runtime-inputs.mjs guessed SDK test2json path', status: 'failed', reason: 'ENOENT before qualified-prune-runtime-telemetry-exact exit127; captured in that raw log. Actual path subsequently resolved with go tool -n test2json.', receipt: 'qualified-prune-runtime-telemetry-exact-receipt.json'},
    ],
    retry_groups: {
      actual_original_caller: ['original-callers-first','original-callers-corrected','original-callers-compatible','original-callers-owned-cache'],
      adapter_attachment_fault: ['adapter-attachment-mutation-red','adapter-attachment-mutation-assertion-red','adapter-attachment-final-green'],
      fast_lint: ['lint-code-fast','lint-code-fast-final','lint-code-fast-clean'],
      prune: ['qualification-ordinary','link-count-diagnostic','link-count-diagnostic-tmpfs','qualified-prune-tmpfs-exact','prune-test-binary-compile','qualified-prune-runtime-tmpfs-exact','qualified-prune-runtime-telemetry-exact','qualified-prune-runtime-resolved-exact'],
    }},
  assertion_coverage: assertionCoverage,
  preservation: {manifest: 'preservation.json', actual_caller_snapshot_rows: 8, actual_callers: ['Explore','CreateCampaignPlan'], fixed_inputs: 'caller-fixed-inputs.json', original_dependency_graph: 'original-caller-source-final.json', original_anchor: 'a3b9f80efab9356c0be2080779133337e2471ac0', harness: 'tools/gomad3/runner/preparation_source_test.go', normalization: 'Prepared.Path only; full Prepared and RecordTarget otherwise byte-equal', initial_prepared_and_source_cache: 'absent on both sides after exclusively owned reset', build_count: '0 initial, 1 after fresh, still 1 after hit; released cache lock asserted', actual_cache_states: ['original-caller-cache-state.json','current-caller-cache-state.json'], later_source_cache_warmth: '313 files per fixture/caller on both matched sides; snapshots captured before later owner controls warmed current inputs', original_portable_empty_root_delta: 'original-portable-missing-root exit1; already-integrated bundle-root owner migration current pass in source-existing-owner. Successful matched adapter controls supply the same nonempty durable root on both sides.', controls: ['source-controls.json','caller-controls.json','owner-controls.json'], targets: 'linux/amd64 source-profile cross-built targets, never executed; actual Explore stops at ProgressRunning with Attempted=0', legacy: 'Original four preimages, original probe bodies, task-only.patch and both historical protocol snapshot pairs retained byte-exact. Old protocol probe is distinct from actual callers; stale legacy overlay retained separately. Original whole-host make exit unknown; watchdog first failure transcript-only.'},
  static_source_coverage: {inventories: 'static-inventories.json', platforms: inventory, authority: 'Unchanged architecture.Discover and TestHostPackageVet. No exclusions or source edits introduced.', unrestricted_failures: ['source-list-linux-receipt.json','source-vet-linux-receipt.json'], scope: 'List/vet only; supported-source cross-compilation does not establish native execution.'},
  generated: {receipt: 'generated-check-receipt.json', commands: 9, mode: 'check-only', input_changes: false, regenerated: false, text_named_assertions: null, unsupported_actual_host_pack_binding: 'existing skip, no native pass claimed'},
  lint: {mandatory_fast: 'lint-code-fast-clean-receipt.json', unfiltered: 'lint-affected-unfiltered-receipt.json', unfiltered_exit: 1, findings: 40, new_task7_findings: 0, exact_attribution: 'lint-attribution.json', paths: lintPaths, disposition: 'All 40 remain retained admission findings; conductor retains exact owners. No waiver or global lint-green claim.'},
  qualification: {ordinary_receipt: 'qualification-ordinary-receipt.json', ordinary_status: 'red', ordinary_named_counts: {pass:213, fail:11, skip:0}, resolved_prune: 'qualified-prune-runtime-resolved-exact-receipt.json', resolved_scope: 'Exactly one unchanged assertion, separately admitted child filesystem and converter telemetry inputs; not a qualification rerun or broad filesystem exception', runtime_inputs: 'runtime-inputs.json', filesystem_admission: 'filesystem-admission.json', remaining_source_gaps: qualificationGaps, ownership_bindings: 'analysis-ownership-research.md', source_gaps_are_native_waived: false},
  predecessor_reuse: {passing_test_receipts_reused: [], note: 'No fn109.1-6 receipt substituted for current task7 assertions or the ten analysis gaps. Static original-function/preimage comparison and fn113.2 pure-helper evidence have their narrower documented scope.'},
  native: {darwin: 'fn149 deferred/unverified', linux_amd64: 'fn128 deferred/unverified', full_host: 'not executed or passed by this worker; native scope transfer remains authoritative'},
  environment_limits: ['root overlay zero available, owned workspace source caches used only in admitted fixtures','workspace hard-link Nlink stale despite paths removed; exact tmpfs capability proved for one prune test','pure Go directory-sync EACCES separately UNKNOWN; not attributed to hard-link observations','inherited BASH_ENV removed only from command children; physical cwd asserted','proxy inputs preserved; no global HOME/cache/telemetry mutation or dependency/toolchain download'],
  stages: ['stage: impl-review - skipped(policy: conductor owns formal review and lifecycle)'],
  children: [{task: '/root/complete_fn1097_source/original_caller_recipe', role: 'read-only research', turns: 2, terminal: true, outputs: ['original-caller-research.md','analysis-ownership-research.md']}],
  terminal: {commands_running: [], children_running: [], root_actions_remaining: ['independent formal review','stage/commit if approved','task7 acceptance and Flow lifecycle','ordered task8 admission retaining ten genuine source gaps'], review_verdict: null},
};
fs.writeFileSync(path.join(directory, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n');
console.log(JSON.stringify({commands: commands.length, final_coverage_groups: assertionCoverage.length, task8_source_gaps: qualificationGaps.length, frozen_source_tree: frozen}));
