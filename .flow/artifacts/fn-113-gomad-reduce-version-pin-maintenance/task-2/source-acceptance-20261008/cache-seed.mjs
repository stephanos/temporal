import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';

const out = path.join(process.cwd(), '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008');
const observations = [];
function run(command, args) {
  const result = spawnSync(command, args, {encoding: 'utf8', maxBuffer: 1048576});
  observations.push({command: [command, ...args], exit_code: result.status, stdout: result.stdout, stderr: result.stderr});
  if (result.status !== 0) throw Error(command + ' failed: ' + result.stderr);
  return result.stdout.trim();
}
const origin = '/home/agent/go/pkg/mod';
const destination = run('mktemp', ['-d', '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-XXXXXXXX']);
const started = new Date().toISOString();
try {
  run('df', ['-h', origin, destination]);
  run('cp', ['-a', origin + '/.', destination]);
  run('du', ['-sb', origin, destination]);
  run('df', ['-h', origin, destination]);
  fs.writeFileSync(path.join(out, 'private-module-cache.json'), JSON.stringify({origin, destination, started, ended: new Date().toISOString(), copy_completed: true, observations}, null, 2) + '\n');
  console.log(destination);
} catch (error) {
  fs.writeFileSync(path.join(out, 'private-module-cache.json'), JSON.stringify({origin, destination, started, ended: new Date().toISOString(), copy_completed: false, observations, error: error.message}, null, 2) + '\n');
  throw error;
}
