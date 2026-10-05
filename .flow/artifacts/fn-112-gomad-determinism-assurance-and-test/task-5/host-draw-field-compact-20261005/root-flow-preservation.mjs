import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';

const flow = '/home/agent/.codex/scripts/flowctl';
const base = '1147416b2e6465de7b631e1e4b695eba33ebf000';
const tasks = ['fn-112.5', 'fn-110.2', 'fn-110.4'];
const results = [];
for (const id of tasks) {
  const current = execFileSync(flow, ['cat', id], { encoding: 'utf8' }).slice(0, -1);
  const metadata = JSON.parse(execFileSync(flow, ['show', id, '--json'], { encoding: 'utf8' }));
  const old = execFileSync('git', ['show', `${base}:${metadata.spec_path}`], { encoding: 'utf8' });
  const section = (text, start, end) => text.slice(text.indexOf(start), end ? text.indexOf(end) : undefined);
  assert.equal(section(current, '## Acceptance\n', '## Done summary\n'),
    section(old, '## Acceptance\n', '## Done summary\n'), `${id}: Acceptance changed`);
  const addedBlock = '\nBlocked:\n' + metadata.blocked_reason.trim() + '\n';
  const currentSummary = section(current, '## Done summary\n', '## Evidence\n');
  assert(currentSummary.endsWith(addedBlock), `${id}: expected Flow blocker append absent`);
  const priorSummary = currentSummary.slice(0, -addedBlock.length);
  assert.equal(priorSummary, section(old, '## Done summary\n', '## Evidence\n'), `${id}: historical summary changed`);
  assert.equal(section(current, '## Evidence\n'), section(old, '## Evidence\n'), `${id}: historical evidence changed`);
  assert.equal(metadata.status, 'blocked');
  results.push({ id: metadata.id, acceptance_byte_equal: true, historical_summary_byte_equal: true,
    historical_evidence_byte_equal: true, status: metadata.status, depends_on: metadata.depends_on });
}
console.log(JSON.stringify({ base, results }, null, 2));
