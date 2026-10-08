import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { worker, read, sha, sources } from './capture.mjs';
import { git } from '../source-acceptance-20261008/capture.mjs';
const json = p => JSON.parse(read(p));
const proof = json(resolve(worker, 'final-source-proof.json'));
const baseline = json(resolve(worker, 'sources-current.json'));
const current = sources();
const changed = [...new Set([...Object.keys(baseline), ...Object.keys(current)])].filter(p => baseline[p] !== current[p]);
assert(changed.every(p => p === 'MILESTONES.md'), changed.join(','));
for (const f of json(resolve(worker, 'terminal-seal.json')).files) {
  const bytes = read(resolve(worker, f.name));
  assert.equal(bytes.length, f.bytes, f.name);
  assert.equal(sha(bytes), f.sha256, f.name);
}
for (const f of proof.user_files.concat(proof.imported_helpers)) assert.equal(sha(read(f.path)), f.sha256, f.path);
for (const f of proof.owners) assert.equal(sha(read(f.path)), f.current_sha256, f.path);
for (const path of ['AGENTS.md', '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md']) assert.equal(sha(read(path)), proof.board[path]);
const accepted = 'c96fb60e654a5f0413b4da630d6a8fe8daf501e7';
const milestones = git(['show', accepted + ':MILESTONES.md']).toString();
const allowedMilestones = [milestones, milestones.replace('| ⬜ Todo | D3: move public executor injection behind private dependencies |', '| ✅ Done | D3: move public executor injection behind private dependencies |')];
allowedMilestones.push(allowedMilestones[1].replace('| 🔄 In progress | Move public executor injection behind private dependencies (D3); current source acceptance. |', '| ✅ Done | Move public executor injection behind private dependencies (D3); current source acceptance. |'));
assert(allowedMilestones.includes(read('MILESTONES.md').toString()), 'Only two exact lifecycle milestone rows may change');
for (const id of ['fn-109-gomad-deepen-modules-and-tool-interfaces.6', 'fn-105-gomad-follow-ups-deferred-scope.3']) {
  const path = '.flow/tasks/' + id + '.md';
  const before = git(['show', accepted + ':' + path]).toString(), after = read(path).toString();
  assert.equal(after.split('## Done summary')[0], before.split('## Done summary')[0], id + ' criteria must remain exact');
}
console.log(JSON.stringify({ verified: true, head: git(['rev-parse', 'HEAD']).toString().trim(), sealed_worker_files: 150, source_changes: changed, source_identity_sha256: sha(JSON.stringify(current)), original_source_identity_sha256: proof.source_identity_sha256, lifecycle_only_rebinding: true, native_pass: false, full_host_pass: false }));
