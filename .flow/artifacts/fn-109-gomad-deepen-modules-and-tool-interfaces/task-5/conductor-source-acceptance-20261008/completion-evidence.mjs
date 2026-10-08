import assert from 'node:assert/strict';
import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, worker, sha } from './capture.mjs';
const relative = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-5/conductor-source-acceptance-20261008';
const receiptPath = '/tmp/impl-review-receipt-657da2bc4466-fn-109-gomad-deepen-modules-and-tool-interfaces.5.json';
const receiptBytes = readFileSync(receiptPath), receipt = JSON.parse(receiptBytes);
assert.equal(receipt.verdict, 'SHIP');
assert.equal(receipt.introduced_count, 0);
assert.equal(receipt.findings.headSha, '15d4be1e3fdbb3047d4aef57ed4198be84a6ea92');
const retained = JSON.parse(readFileSync(resolve(out, 'source-review.json')));
for (const record of retained.records) {
  const bytes = Buffer.from(record.data, 'base64');
  assert.equal(bytes.length, record.bytes);
  assert.equal(sha(bytes), record.sha256);
  assert.deepEqual(bytes, readFileSync(record.path));
}
const receipts = readdirSync(out).filter(p => p.endsWith('.json')).sort().flatMap(file => {
  const path = resolve(out, file), bytes = readFileSync(path), value = JSON.parse(bytes);
  if (!Array.isArray(value.argv)) return [];
  assert.equal(value.signal, null);
  assert.equal(value.error, null);
  assert.deepEqual(value.source_changes, []);
  assert.equal(value.stdout_sha256, sha(readFileSync(path.replace(/\.json$/, '.stdout'))));
  assert.equal(value.stderr_sha256, sha(readFileSync(path.replace(/\.json$/, '.stderr'))));
  return [{ path: relative + '/' + file, sha256: sha(bytes), argv: value.argv, exit: value.exit, signal: value.signal, error: value.error, started: value.started, ended: value.ended }];
});
assert.equal(receipts.find(r => r.path.endsWith('/postreview-independent-audit.json'))?.exit, 0);
const audit = JSON.parse(readFileSync(resolve(out, 'audit.json')));
const evidence = { commits: ['15d4be1e3fdbb3047d4aef57ed4198be84a6ea92'], tests: receipts.map(r => r.argv.join(' ') + '; exit=' + r.exit), prs: [], scope: 'Task5 retained shared parsing and Runner normalization source acceptance; task-level R6', native_qualification: false, full_host_pass: false, global_lint_green: false, portable_unique_top_level_tests: 139, negative_control_top_fail: 1, negative_control_subcase_fail: 2, historical_pre_restoration_full_cli: { pass: 89, fail: 3, current_candidate_pass: false }, worker_evidence_sha256: sha(readFileSync(resolve(worker, 'evidence.json'))), worker_freeze_sha256: sha(readFileSync(resolve(worker, 'freeze.json'))), source_identity_sha256: audit.source_identity_sha256, review_rid: receipt.rid, review_receipt_sha256: sha(receiptBytes), root_receipts: receipts };
writeFileSync(resolve(out, 'completion-evidence.json'), JSON.stringify(evidence, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ receipts: receipts.length, portable_unique: 139, review: receipt.verdict, native: false, global_lint_green: false }));
