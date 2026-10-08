import assert from 'node:assert/strict';
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, worker, read, sha } from './capture.mjs';
import { statusTotals, verifySummary } from './mapping-summary-test.mjs';

const sourcePath = worker + '/final-current-mapping-analysis.json';
const original = JSON.parse(read(sourcePath));
assert.throws(() => verifySummary(original), /status summary must equal actual final row aggregation/);
const corrected = { ...original, counts_by_actual_status: statusTotals(original.rows) };
verifySummary(corrected);
assert.deepEqual(corrected.rows, original.rows);
for (const key of Object.keys(original).filter(key => key !== 'counts_by_actual_status')) assert.deepEqual(corrected[key], original[key], key);
const controls = [];
for (const [name, alter] of [
  ['stale_summary', report => { report.counts_by_actual_status = original.counts_by_actual_status; }],
  ['changed_row_status', report => { report.rows[0].actual_current_status = 'forged pass'; }],
  ['missing_row', report => { report.rows.pop(); }],
]) {
  const forged = structuredClone(corrected);
  alter(forged);
  assert.throws(() => verifySummary(forged));
  controls.push({ name, rejected: true });
}
writeFileSync(resolve(out, 'current-mapping-corrected.json'), JSON.stringify(corrected, null, 2) + '\n', { flag: 'wx' });
const result = {
  source: { path: sourcePath, sha256: sha(read(sourcePath)) },
  corrected: { path: resolve(out, 'current-mapping-corrected.json'), sha256: sha(read(out + '/current-mapping-corrected.json')) },
  original_report_retained_unchanged: true,
  every_row_and_other_field_exact: true,
  earlier_summary: original.counts_by_actual_status,
  current_summary: corrected.counts_by_actual_status,
  controls,
  native: false,
  review_verdict: null,
};
writeFileSync(resolve(out, 'mapping-summary-correction-proof.json'), JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(result));
