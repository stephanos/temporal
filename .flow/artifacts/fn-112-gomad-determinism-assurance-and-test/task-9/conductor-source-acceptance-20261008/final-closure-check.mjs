import assert from 'node:assert/strict';
import { readdirSync, statSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { out, worker, root, read, sha, sources } from './capture.mjs';
import { sources as moduleSources } from '../source-acceptance-20261008/capture.mjs';
import { verifySummary } from './mapping-summary-test.mjs';

const json = p => JSON.parse(read(p));
const git = args => {
  const result = spawnSync('git', args, { cwd: root, maxBuffer: 64 << 20 });
  assert.equal(result.status, 0, result.stderr?.toString());
  return result.stdout;
};
const head = git(['rev-parse', 'HEAD']).toString().trim();
assert.equal(head, 'da10d3bdad0c10b844832fcc9ebb318ebdd936a8');
const proof = json(worker + '/sealed-source-proof.json');
assert.deepEqual(moduleSources(), proof.current_sources);
for (const original of proof.original_commits) {
  assert.equal(git(['rev-parse', original.commit + '^']).toString().trim(), original.parent);
}
assert.equal(proof.mapping_rows.length, 282);
assert.equal(proof.mapping_rows.filter(r => r.replacement === '-').length, 11);
assert.equal(proof.unaffected_function_bindings.length, 165);
assert.equal(proof.current_shared_assertion_bodies.length, 22);
assert.equal(proof.accepted_owner_proofs.task6.named_mapping_count, 403);
assert.equal(proof.accepted_owner_proofs.task6.current_exact_body_bindings, 383);
assert.equal(proof.accepted_owner_proofs.task3.current_functions.length, 4);
assert.equal(proof.accepted_owner_proofs.task3.all_four_exact, true);
assert.equal(proof.static.runtime_input_count, 87);
assert.equal(proof.static.first_baseline.all_original_bytes_rechecked, true);

const manifest = json(worker + '/manifest.json');
const seal = json(worker + '/terminal-seal.json');
assert.equal(manifest.files.length, 448);
assert.equal(sha(read(worker + '/manifest.json')), '25358815c9030c72348f560c6e17c646a8173bbb2e2c490c571ba3cab5accbe7');
assert.equal(sha(read(worker + '/terminal-seal.json')), 'a8fb30b9892076ac10fa9e45f976387c53cf25f2d8b8f4bb44c20b88aefebf1f');
assert.equal(seal.manifest_sha256, sha(read(worker + '/manifest.json')));
assert.equal(seal.terminal_files.length, 6);
for (const file of [...manifest.files, ...seal.terminal_files]) {
  assert.equal(sha(read(worker + '/' + file.path)), file.sha256, file.path);
  assert.equal(statSync(worker + '/' + file.path).size, file.bytes, file.path);
}

const originalMapping = json(worker + '/final-current-mapping-analysis.json');
const corrected = json(out + '/current-mapping-corrected.json');
const correctionProof = json(out + '/mapping-summary-correction-proof.json');
verifySummary(corrected);
const withoutSummary = value => {
  const { counts_by_actual_status, ...rest } = value;
  return rest;
};
assert.deepEqual(withoutSummary(corrected), withoutSummary(originalMapping));
assert.equal(sha(read(worker + '/final-current-mapping-analysis.json')), correctionProof.source.sha256);
assert.equal(sha(read(out + '/current-mapping-corrected.json')), correctionProof.corrected.sha256);
for (const [name, mutate] of [
  ['stale_summary', report => report.counts_by_actual_status = originalMapping.counts_by_actual_status],
  ['changed_row_status', report => report.rows[0].actual_current_status = 'changed'],
  ['missing_row', report => report.rows.pop()],
]) {
  const report = structuredClone(corrected);
  mutate(report);
  assert.throws(() => verifySummary(report), name);
}

const first = json(out + '/first-source-review.json');
assert.equal(first.verdict, 'NEEDS_WORK');
for (const record of first.records) {
  const bytes = Buffer.from(record.data, 'base64');
  assert.equal(bytes.length, record.bytes);
  assert.equal(sha(bytes), record.sha256);
}
const pending = json(out + '/final-prepublication-review-journal.json');
const published = json(out + '/final-published-source-review.json');
assert.equal(sha(read(out + '/final-prepublication-review-journal.json')), 'ba25d87dec875f2f07c52204c7eae04506bbaac381c3717cf8ebd4b5a348f8db');
assert.equal(Buffer.byteLength(pending.response), 727);
assert.equal(sha(pending.response), '63812bcd292546a4464c1de210b9b2c934a485de1c037832897edebd4f428666');
assert.equal(published.review, pending.response);
assert.equal(published.verdict, 'SHIP');
assert.equal(published.base, proof.base_commit);
assert.equal(published.findings.headSha, head);
assert.equal(published.review_reservation_id, pending.reservation_id);
assert.deepEqual(published.unaddressed, []);
assert(published.findings.items.every(finding => finding.status === 'fixed'));
assert.equal(sha(read(out + '/final-published-source-review.json')), 'b5eeeb0f06be9c875c40312c13725ef1973d5bafce34dc4c0e94041925c62013');

const receipts = readdirSync(out).filter(p => p.endsWith('.json')).flatMap(path => {
  const receipt = json(out + '/' + path);
  return Array.isArray(receipt.argv) && Object.hasOwn(receipt, 'exit') ? [{ path, receipt }] : [];
});
for (const { path, receipt } of receipts) {
  assert(Number.isInteger(receipt.exit), path);
  assert.equal(receipt.signal, null, path);
  assert.equal(receipt.error, null, path);
  assert.deepEqual(receipt.source_changes, [], path);
  assert.equal(receipt.sources_before_sha256, receipt.sources_after_sha256, path);
  assert.equal(sha(read(out + '/' + path.replace(/\.json$/, '.stdout'))), receipt.stdout_sha256, path);
  assert.equal(sha(read(out + '/' + path.replace(/\.json$/, '.stderr'))), receipt.stderr_sha256, path);
}
const verification = json(out + '/final-closure-worker-verification.json');
const verifiedOutput = json(out + '/final-closure-worker-verification.stdout');
assert.equal(verification.exit, 0);
assert.equal(verifiedOutput.valid, true);
assert.equal(verifiedOutput.terminal_receipts, 117);
assert.equal(verifiedOutput.current_source_paths, 1070);
assert.equal(verifiedOutput.negative_receipt_controls.length, 4);
assert(verifiedOutput.negative_receipt_controls.every(control => control.rejected));

const current = sources();
const committedMilestones = git(['show', 'HEAD:MILESTONES.md']);
const currentMilestones = read('MILESTONES.md').toString();
const start = currentMilestones.indexOf('### Orchestration\n');
const end = currentMilestones.indexOf('## Constraints\n', start);
assert(start >= 0 && end > start);
assert.equal(currentMilestones.slice(0, start) + currentMilestones.slice(end), committedMilestones.toString());
const historicalScope = { ...current, 'MILESTONES.md': sha(committedMilestones) };
const audit = json(out + '/audit.json');
assert.equal(sha(JSON.stringify(historicalScope)), audit.source_identity_sha256);
assert.equal(json(out + '/plan-sync-config.stdout').value, false);
const memoryPath = '.flow/memory/bug/integration/recompute-mapping-summaries-after-final-2026-10-08.md';
assert(read(memoryPath).length > 0);
console.log(JSON.stringify({
  valid: true, head, module_source_paths_exact: Object.keys(proof.current_sources).length,
  worker_manifest_files: manifest.files.length, worker_terminal_files: seal.terminal_files.length,
  worker_manifest_sha256: sha(read(worker + '/manifest.json')),
  worker_seal_sha256: sha(read(worker + '/terminal-seal.json')),
  mapping_rows: 282, original_commits_and_parents_exact: 3, housekeeping_reasons: 11,
  unaffected_functions: 165, shared_bodies: 22, task6_mappings: 403, task6_bodies: 383,
  task3_bodies: 4, exact_runtime_inputs: 87, conductor_terminal_receipts: receipts.length,
  published_receipt_sha256: sha(read(out + '/final-published-source-review.json')),
  published_backend: published.spec, published_timestamp: published.timestamp,
  backend_response_sha256: sha(published.review), backend_response_bytes: Buffer.byteLength(published.review),
  audit_nonflow_aggregate: audit.source_identity_sha256,
  current_nonflow_aggregate: sha(JSON.stringify(current)),
  current_milestones_sha256: sha(currentMilestones), committed_milestones_sha256: sha(committedMilestones),
  only_authorized_orchestration_document_difference: true, memory_path: memoryPath,
  native: false, full_host_pass: false, global_lint_pass: false, whole_source_format_pass: false,
}));
