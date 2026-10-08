import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import {root, out, sha} from './capture.mjs';

const protectedFiles = ['input-proof.json', 'source-proof.json', 'source-binding.stdout', 'source-binding.stderr'];
const hashes = () => protectedFiles.map(file => sha(readFileSync(out + '/' + file)));
const before = hashes();
const result = spawnSync(process.execPath, [out + '/bind-source.mjs'], {cwd: root, env: process.env, timeout: 10000, maxBuffer: 1 << 20});
assert.equal(result.status, 1);
assert(result.stderr.toString().includes('existing input proof; refusing overwrite'));
assert.deepEqual(hashes(), before);
console.log(JSON.stringify({binder_existing_proof_refused: true, binder_exit: result.status, protected_files_unchanged: protectedFiles, native_credit: false}));
