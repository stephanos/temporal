import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { read } from './capture.mjs';

export function statusTotals(rows) {
  return rows.reduce((counts, row) => {
    counts[row.actual_current_status] = (counts[row.actual_current_status] ?? 0) + 1;
    return counts;
  }, {});
}

export function verifySummary(report) {
  assert.equal(report.rows.length, 282);
  assert.deepEqual(report.counts_by_actual_status, statusTotals(report.rows), 'status summary must equal actual final row aggregation');
  assert.equal(report.counts_by_actual_status['NOT REACHED; ancestor FAIL'], 104);
  assert.equal(Object.values(report.counts_by_actual_status).reduce((n, count) => n + count, 0), report.rows.length);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const report = JSON.parse(read(process.argv[2]));
  verifySummary(report);
  console.log(JSON.stringify({ valid: true, rows: report.rows.length, counts: report.counts_by_actual_status }));
}
