import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { out, worker, root, stock, read } from './capture.mjs';

const before = JSON.parse(read(worker + '/sealed-root.json'));
assert(before.environment.PATH.startsWith(stock + ':'));
const environment = { ...process.env, ...before.environment, GOCACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r' };
environment.PATH = before.environment.PATH.slice(stock.length + 1);
const result = spawnSync('node', [out + '/capture.mjs', 'independent-controlled-cache-fixtures', stock + '/go', '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-run', '^TestArchitecture(PublicSignature|Effect)Fixtures$', '.'], { cwd: root, env: environment, stdio: 'inherit', timeout: 610000 });
assert.equal(result.signal, null);
assert.notEqual(result.status, null);
process.exitCode = result.status;
