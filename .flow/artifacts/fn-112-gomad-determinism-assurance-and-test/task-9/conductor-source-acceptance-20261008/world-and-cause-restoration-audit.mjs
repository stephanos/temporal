import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, worker, read, sha } from './capture.mjs';

const inherited = resolve(out, 'three-file-restoration-audit.mjs');
let source = read(inherited).toString();
const oldCheck = "assert.equal(oldCompletion.replace(beforeCompletion, '<completion-function>'), completion.replace(afterCompletion, '<completion-function>'));";
assert.equal(source.split(oldCheck).length, 2);
source = source.replace(oldCheck, String.raw`
const worldName = 'TestAssessWorldValidatesTheRecordAgainstItsSeed';
const beforeWorld = body(oldCompletion, worldName), afterWorld = body(completion, worldName);
const originalWorld = body(git(['show', '59ca3d17395be5501906586006e2539d33bea28c:' + completionPath]), worldName);
const seedRows = originalWorld.split('\\n').filter(line => line.includes('name: "seed mismatch",'));
assert.equal(seedRows.length, 1);
assert.equal(afterWorld.split(seedRows[0] + '\\n').length, 2);
assert.equal(afterWorld.replace(seedRows[0] + '\\n', ''), beforeWorld);
assert.equal(oldCompletion.replace(beforeCompletion, '<completion-function>').replace(beforeWorld, '<world-function>'), completion.replace(afterCompletion, '<completion-function>').replace(afterWorld, '<world-function>'));
`);
source = source.replace('three-file-restoration-proof.json', 'world-and-cause-restoration-proof.json');
source = source.replace('all_off_function_completion_bytes_exact: true,', 'all_off_function_completion_bytes_exact: true, private_world_delta_exactly_one_original_seed_mismatch_row: true, all_other_world_rows_and_assertions_byte_exact: true,');
source = source.replace('"three-file-restoration-audit.mjs"', '"world-and-cause-restoration-audit.mjs"');
source = source.replace(/^import .*;\n/gm, '');
new Function('assert', 'spawnSync', 'writeFileSync', 'resolve', 'out', 'worker', 'read', 'sha', source)(assert, spawnSync, writeFileSync, resolve, out, worker, read, sha);
