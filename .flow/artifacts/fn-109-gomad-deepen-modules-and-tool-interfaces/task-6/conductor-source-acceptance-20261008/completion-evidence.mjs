import assert from 'node:assert/strict';
import { readdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, worker, read, sha } from './capture.mjs';
const receiptPath = '/tmp/impl-review-receipt-657da2bc4466-fn-109-gomad-deepen-modules-and-tool-interfaces.6.json';
const receipt = JSON.parse(read(receiptPath));
assert.equal(receipt.verdict, 'SHIP');
assert.equal(receipt.introduced_count, 0);
assert.equal(receipt.findings.headSha, 'c96fb60e654a5f0413b4da630d6a8fe8daf501e7');
const retained = JSON.parse(read(resolve(out, 'source-review.json')));
for (const r of retained.records) {
  const bytes = Buffer.from(r.data, 'base64');
  assert.equal(bytes.length, r.bytes);
  assert.equal(sha(bytes), r.sha256);
  assert.deepEqual(bytes, read(r.path));
}
const rootReceipts = readdirSync(out).filter(p => p.endsWith('.json')).sort().flatMap(file => {
  const path = resolve(out, file), value = JSON.parse(read(path));
  if (!Array.isArray(value.argv)) return [];
  assert.equal(value.signal, null);
  assert.equal(value.error, null);
  assert.deepEqual(value.source_changes, []);
  assert.equal(sha(read(path.replace(/\.json$/, '.stdout'))), value.stdout_sha256);
  assert.equal(sha(read(path.replace(/\.json$/, '.stderr'))), value.stderr_sha256);
  return [{ path, sha256: sha(read(path)), argv: value.argv, exit: value.exit, started: value.started, ended: value.ended }];
});
assert.equal(rootReceipts.find(r => r.path.endsWith('/independent-source-audit.json'))?.exit, 0);
assert.equal(rootReceipts.find(r => r.path.endsWith('/final-postreview-source-binding.json'))?.exit, 0);
const reference = process.argv.includes('--reference');
const evidence = {
  commits: ['c96fb60e654a5f0413b4da630d6a8fe8daf501e7'], tests: rootReceipts.map(r => r.argv.join(' ') + '; exit=' + r.exit), prs: [],
  scope: reference ? 'fn105.3 D3 closes only by reference to the accepted fn109.6 R5 owner; no separate implementation/tests/review' : 'fn109.6 R5 retained source acceptance and fn105.3 D3 reference closure',
  native_qualification: false, full_host_pass: false, global_lint_green: false, whole_source_format_pass: false,
  portable_unique_top_level_tests: 81, failed_top_level_tests_retained: 45, failed_terminal_children_retained: 12,
  fast_lint_findings: 66, scoped_lint_findings: 91, new_waivers: [], product_source_changes: [],
  worker_evidence_sha256: sha(read(resolve(worker, 'evidence.json'))), worker_freeze_sha256: sha(read(resolve(worker, 'freeze.json'))),
  worker_terminal_seal_sha256: sha(read(resolve(worker, 'terminal-seal.json'))), original_test_function_mappings: 403,
  review_rid: receipt.rid, review_timestamp: receipt.timestamp, review_receipt_sha256: sha(read(receiptPath)),
  shared_review: reference, root_receipts: rootReceipts,
};
writeFileSync(resolve(out, reference ? 'd3-reference-evidence.json' : 'completion-evidence.json'), JSON.stringify(evidence, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ reference, receipts: rootReceipts.length, portable_unique: 81, verdict: receipt.verdict, native_pass: false, lint_green: false }));
