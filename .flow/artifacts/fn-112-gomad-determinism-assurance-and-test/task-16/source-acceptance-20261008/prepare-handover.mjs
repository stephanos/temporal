import assert from 'node:assert/strict';
import {existsSync, readFileSync, readdirSync, writeFileSync} from 'node:fs';
import {root, out, sha, git} from './capture.mjs';

const refresh = process.argv[2] === '--refresh-worker-snapshot';
assert(process.argv.length === (refresh ? 3 : 2));
for (const file of ['evidence.json', 'bundle-manifest.json']) if (!refresh) assert(!existsSync(out + '/' + file), 'existing final evidence ' + file);
const parse = file => JSON.parse(readFileSync(out + '/' + file));
const binding = file => ({path: file, bytes: readFileSync(out + '/' + file).length, sha256: sha(readFileSync(out + '/' + file))});
const proof = parse('input-proof.json');
if (refresh) {
  assert.equal(parse('evidence.json').task_id, 'fn-112-gomad-determinism-assurance-and-test.16');
  assert.equal(parse('evidence.json').source_map_sha256, proof.sources_sha256);
}
const standards = parse('standards-attribution.json');
const quote = value => /^[a-zA-Z0-9_./=:-]+$/.test(value) ? value : JSON.stringify(value);
const conductorReceipt = file => /^conductor-.*\.(json|stdout|stderr)$/.test(file);
const receipts = readdirSync(out).filter(file => file.endsWith('.json') && file !== 'failed-binding-observation.json' && !conductorReceipt(file)).map(file => [file, parse(file)]).filter(([, receipt]) => Array.isArray(receipt.argv));
const observations = receipts.map(([file, receipt]) => ({
  receipt: binding(file),
  argv: receipt.argv,
  cwd: receipt.cwd,
  environment: receipt.environment,
  unset_environment: receipt.unset_environment ?? null,
  exit: receipt.exit,
  signal: receipt.signal ?? null,
  error: receipt.error ?? null,
  started: receipt.started ?? null,
  ended: receipt.ended ?? null,
  elapsed_seconds: receipt.elapsed_seconds ?? null,
  tool_wall_seconds: receipt.tool_wall_seconds ?? null,
  package_elapsed_seconds: receipt.package_elapsed_seconds ?? null,
  test_counts: receipt.test_counts ?? null,
  package_results: receipt.package_results ?? null,
  source_count: receipt.source_count ?? null,
  source_before_sha256: receipt.source_before_sha256 ?? null,
  source_after_sha256: receipt.source_after_sha256 ?? null,
  source_changes: receipt.source_changes ?? null,
  source_binding_note: receipt.source_binding ?? null,
  head_before: receipt.head_before ?? null,
  head_after: receipt.head_after ?? null,
  stdout: binding(file.replace(/\.json$/, '.stdout')),
  stderr: binding(file.replace(/\.json$/, '.stderr')),
}));
const evidence = {
  task_id: 'fn-112-gomad-determinism-assurance-and-test.16',
  task_status: 'in_progress',
  source_ready: true,
  source_ready_meaning: 'Implementation handover only; conductor independent reruns, source review and acceptance adjudication remain required.',
  workspace: root,
  branch: git(['branch', '--show-current']).toString().trim(),
  base_commit: proof.base_commit,
  head: git(['rev-parse', 'HEAD']).toString().trim(),
  commits: [],
  tests: observations.map(observation => observation.argv.map(quote).join(' ')),
  prs: [],
  tier: 'Tier: session (jev-unavailable(no_key)); explicit project implementer gpt-6.1-sol/high remains selected.',
  judge_timing: 'Conductor judge call occurred just after dispatch, not before.',
  execution_model_metadata: 'Unknown; no actual_model field inferred from selected implementer.',
  stage: ['impl-review - skipped(policy: host-deferred - conductor owns the gate)'],
  ownership: 'Conductor owns all Git writes, Flow/MILESTONES lifecycle, fresh review and completion. No worker commit or external mutation.',
  host: {os: 'Linux', architecture: 'aarch64', uid: 1000, patched_runtime_supported: false, stock_go: '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'},
  source_proof: binding('input-proof.json'),
  source_count: proof.source_count,
  source_map_sha256: proof.sources_sha256,
  product_changes: proof.product_changes,
  review_source_count: proof.review_sources.length,
  provenance: {native_fix: proof.native_fix, native_prechange: proof.native_prechange, experimental_fix: proof.experimental_fix, experimental_prechange: proof.experimental_prechange, integration: proof.integration, historical_receipts: 'SHA-bound lossless references in input-proof.json, not current native evidence'},
  runtime_reuse: proof.runtime_reuse,
  archive: proof.archive,
  observations,
  failed_binding_attempt: {
    observation: binding('failed-binding-observation.json'),
    stdout: binding('source-binding.stdout'),
    stderr: binding('source-binding.stderr'),
    full_command_metadata_lost: true,
    status: 'inconclusive; observed exit 1 / 0.739s / source_changes [] retained, missing timestamps/environment/source hashes not reconstructed',
    repair: 'Successful proof mechanically moved by apply_patch to input-proof.json; command receipt source-proof.json remains distinct. Helper output renamed; capture labels have overwrite guards.',
  },
  acceptance: {
    R9: {status: 'source-covered; current native assertions transferred', selectors: ['TestPublishKeepsSuccessesWithOneSignatureAsDistinctReplayArtifacts', 'TestPublishKeepsEachExecutionOfOneOutcomeSignature', 'TestOpenCampaignKeepsSameSignatureSuccessesDistinct'], portable_top_level_tests: 155, portable_fail: 0, portable_skip: 0, new_campaign_test_events: 9, policies: ['StoreKeyFailureSignature with StoreKeyRecord fallback', 'StoreKeyExecution with execution identity fallback'], checks: ['equal success signatures and distinct stored replay identities', 'preserved first signature directory', 'OpenCampaign acceptance and matching journal/disk/retained counts and bytes', 'negative seed/ordinal/reference controls', 'existing exact-repeat idempotence and unchanged failure dedup', 'original literal identities unchanged'], sensitivity: 'Two current-source branch mutants fail publication identity validation; not exact historical RED. Original actual native RED retained by reference.'},
    generated_validation: {receipt: 'final-generated-validation.json', exit: 0},
    portable_retention_controls: {receipt: 'runner-portable-retention.json', tests: 4, exit: 0},
    standards: {attribution: binding('standards-attribution.json'), aggregate_green: false, configured_fast_lint_exit: 2, configured_fast_lint_findings: 68, explicit_source_lint_exit: 2, explicit_source_lint_findings: 12, original_native_task_findings: standards.original_task_findings, worker_introduced_findings: standards.worker_introduced_findings, changed_package_lint_exit: 0, configured_vet_exit: 0, suppressions_added: false, external_ownership_meaning: 'Exact blame/source attribution only, no waiver or synthetic filtered green. Conductor adjudicates scoped acceptance; original aggregate owners remain.'},
    native: {qualified: false, owners: ['fn-149', 'fn-128'], runner_observation: 'exit 1 at unsupported linux/arm64 guard; assertion not reached', cli_observation: 'exit 1 before collecting tests: .toolchain/bin/go absent', historical_native_receipts: 'original provenance only; no current-candidate native credit', static_materialization: 'both supported source sets static only'},
  },
  user_preservation: proof.user_files,
  user_scope_exclusion: proof.excluded_user_scope,
  verification_helper: binding('conductor-verify.mjs'),
  handover: binding('handover.md'),
  command_capture_limits: ['Earliest baseline inherited environment and full command duration not snapshotted; unknown values remain null.', 'Earlier captures before helper scope extension do not bind the actual root lint config; final source proof and standards bindings do.', 'Read-only investigation commands are not gate receipts; no durations, environment snapshots or test credit are reconstructed for them.'],
  running_commands: [],
  lane_released: true,
};
assert.equal(evidence.head, evidence.base_commit);
writeFileSync(out + '/evidence.json', JSON.stringify(evidence, null, 2) + '\n');
const files = readdirSync(out).filter(file => file !== 'bundle-manifest.json' && !conductorReceipt(file)).sort().map(binding);
writeFileSync(out + '/bundle-manifest.json', JSON.stringify({kind: 'worker-owned task-local evidence byte snapshot; conductor command receipts excluded; no whole-checkout equality claim', excluded_pattern: '^conductor-.*\\.(json|stdout|stderr)$', files}, null, 2) + '\n');
console.log(JSON.stringify({evidence: out + '/evidence.json', manifest: out + '/bundle-manifest.json', command_receipts: observations.length, files: files.length, source_ready: true, task_status: 'in_progress', aggregate_lint_green: false, native_qualification: false}));
