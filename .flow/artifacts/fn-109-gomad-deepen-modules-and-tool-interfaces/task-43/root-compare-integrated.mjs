import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';

const artifacts = '/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces';
function readDiagnostics(path) {
  const raw = fs.readFileSync(path, 'utf8');
  const lines = raw.split('\n');
  const starts = [];
  for (let i = 0; i < lines.length; i++) {
    if (/^\S[^:]*\.go:\d+:\d+: .+ \([^)]+\)$/.test(lines[i])) starts.push(i);
  }
  const blocks = starts.map((start, index) => {
    let end = index + 1 < starts.length ? starts[index + 1] : lines.findIndex((line, i) => i > start && /^\d+ issues:/.test(line));
    if (end < 0) end = lines.length;
    return { header: lines[start], raw: lines.slice(start, end).join('\n') + '\n' };
  });
  return { path, sha256: crypto.createHash('sha256').update(raw).digest('hex'), blocks };
}
const baseline = readDiagnostics(`${artifacts}/task-42/root-integrated-final.log`);
const candidate = readDiagnostics(`${artifacts}/task-43/root-integrated.log`);
assert.equal(baseline.sha256, '51d765a947eb3a52300ed85195a1d04e09c7009bdf5a7f829e8fc0d7aee3064f');
assert.equal(baseline.blocks.length, 323);
assert.equal(candidate.blocks.length, 323);
const shifted = [];
const normalizedHeaders = [];
for (const block of candidate.blocks) {
  const match = block.header.match(/^(.*\/compatibilitypack\/schema\.go):(\d+):(\d+):/);
  const header = match ? block.header.replace(`${match[1]}:${match[2]}:`, `${match[1]}:${Number(match[2]) - 1}:`) : block.header;
  normalizedHeaders.push(header);
  const original = baseline.blocks.find(entry => entry.header === header);
  assert.ok(original, `new or unexpectedly moved diagnostic: ${block.header}`);
  assert.equal(block.raw.replace(block.header, header), original.raw, `changed diagnostic body: ${block.header}`);
  if (match) shifted.push({ baseline: header, candidate: block.header });
}
assert.equal(shifted.length, 4);
assert.deepEqual(normalizedHeaders.sort(), baseline.blocks.map(block => block.header).sort());
const counts = {};
for (const block of candidate.blocks) {
  const analyzer = block.header.match(/\(([^)]+)\)$/)[1];
  counts[analyzer] = (counts[analyzer] ?? 0) + 1;
}
assert.deepEqual(counts, { errcheck: 252, exhaustive: 3, forbidigo: 11, gci: 1, staticcheck: 56 });
process.stdout.write(JSON.stringify({
  baseline: { path: baseline.path, sha256: baseline.sha256, count: baseline.blocks.length },
  candidate: { path: candidate.path, sha256: candidate.sha256, count: candidate.blocks.length },
  counts, removed: [], added: [], exact_unchanged_blocks: 319,
  comment_only_line_shifts: shifted, shifted_blocks_otherwise_byte_identical: true,
}, null, 2) + '\n');
